package main

import (
	"bytes"
	"context"
	"encoding/json" 
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/disintegration/imaging"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Config - структура для хранения настроек из JSON
type Config struct {
	R2AccountId  string `json:"r2_account_id"`
	R2AccessKey  string `json:"r2_access_key"`
	R2SecretKey  string `json:"r2_secret_key"`
	BucketName   string `json:"bucket_name"`
	PublicDomain string `json:"public_domain"`
}

type App struct {
	ctx            context.Context
	telegraphToken string
	config         Config // Храним загруженный конфиг здесь
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 1. Загружаем конфиг
	if err := a.loadConfig(); err != nil {
		// Показываем нативный диалог ошибки, если конфиг не найден
		wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
			Type:    wailsRuntime.ErrorDialog,
			Title:   "Ошибка конфигурации",
			Message: "Не удалось загрузить config.json: " + err.Error(),
		})
		return
	}

	// 2. Инициализируем Telegraph
	token, err := createTelegraphAccount("MangaUploader")
	if err == nil {
		a.telegraphToken = token
	} else {
		fmt.Println("Ошибка создания Telegraph аккаунта:", err)
	}
}

// Метод для чтения config.json
func (a *App) loadConfig() error {
	// Пытаемся открыть файл в текущей папке
	file, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	bytesValue, _ := io.ReadAll(file)
	
	err = json.Unmarshal(bytesValue, &a.config)
	if err != nil {
		return err
	}

	// Простая валидация
	if a.config.R2AccountId == "" || a.config.R2AccessKey == "" {
		return fmt.Errorf("поля конфигурации пусты")
	}

	return nil
}

// === СТРУКТУРЫ ===

type UploadResult struct {
	Success bool     `json:"success"`
	Links   []string `json:"links"`
	Error   string   `json:"error"`
}

type ChapterResponse struct {
	Path       string   `json:"path"`
	Title      string   `json:"title"`
	Images     []string `json:"images"`
	ImageCount int      `json:"imageCount"`
}

type TelegraphResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		Url string `json:"url"`
	} `json:"result"`
	Error string `json:"error"`
}

type TelegraphNode struct {
	Tag   string            `json:"tag"`
	Attrs map[string]string `json:"attrs"`
}

// === МЕТОДЫ ===

func (a *App) UploadChapter(filePaths []string) UploadResult {
	// Используем настройки из загруженного конфига (a.config)
	endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", a.config.R2AccountId)
	
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(a.config.R2AccessKey, a.config.R2SecretKey, ""),
		Secure: true,
	})

	if err != nil {
		return UploadResult{Success: false, Error: "Ошибка R2 Client: " + err.Error()}
	}

	maxWorkers := runtime.NumCPU() * 2
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	uploadedLinks := make([]string, len(filePaths))
	var uploadErrors []string

	for i, path := range filePaths {
		wg.Add(1)
		go func(index int, srcPath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Обработка
			img, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("Open err (%s): %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			if img.Bounds().Dx() > 1600 {
				img = imaging.Resize(img, 1600, 0, imaging.Lanczos)
			}

			buf := new(bytes.Buffer)
			err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(80))
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("Encode err: %v", err))
				mu.Unlock()
				return
			}

			// Загрузка
			fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(srcPath))
			
			// Используем имя бакета из конфига
			_, err = client.PutObject(context.Background(), a.config.BucketName, fileName, buf, int64(buf.Len()), minio.PutObjectOptions{
				ContentType: "image/jpeg",
			})
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("Upload err: %v", err))
				mu.Unlock()
				return
			}

			// Используем домен из конфига
			// Убираем слэш в конце домена, если он есть, чтобы не было двойного //
			domain := strings.TrimRight(a.config.PublicDomain, "/")
			finalUrl := fmt.Sprintf("%s/%s", domain, fileName)
			
			mu.Lock()
			uploadedLinks[index] = finalUrl
			mu.Unlock()
		}(i, path)
	}

	wg.Wait()

	if len(uploadErrors) > 0 {
		return UploadResult{Success: false, Error: fmt.Sprintf("Ошибки (%d): %v", len(uploadErrors), uploadErrors[0])}
	}

	return UploadResult{Success: true, Links: uploadedLinks}
}

// 2. Создание статьи в Telegraph
func (a *App) CreateTelegraphPage(title string, imageUrls []string) string {
	if a.telegraphToken == "" {
		t, err := createTelegraphAccount("MangaUploader")
		if err != nil {
			return "Ошибка: Не удалось авторизоваться в Telegraph"
		}
		a.telegraphToken = t
	}

	var content []TelegraphNode
	for _, link := range imageUrls {
		node := TelegraphNode{
			Tag: "img",
			Attrs: map[string]string{
				"src": link,
			},
		}
		content = append(content, node)
	}

	contentJson, err := json.Marshal(content)
	if err != nil {
		return "JSON Error: " + err.Error()
	}

	apiURL := "https://api.telegra.ph/createPage"
	data := url.Values{}
	data.Set("access_token", a.telegraphToken)
	data.Set("title", title)
	data.Set("content", string(contentJson))
	data.Set("return_content", "false")

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return "Net Error: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var tgResp TelegraphResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return "Telegraph API Error: " + string(body)
	}

	if !tgResp.Ok {
		return "Telegraph Error: " + tgResp.Error
	}

	return tgResp.Result.Url
}

func createTelegraphAccount(shortName string) (string, error) {
	apiURL := "https://api.telegra.ph/createAccount"
	data := url.Values{}
	data.Set("short_name", shortName)
	data.Set("author_name", "Manga Bot")

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	type AccountResp struct {
		Ok     bool `json:"ok"`
		Result struct {
			AccessToken string `json:"access_token"`
		} `json:"result"`
	}
	var acc AccountResp
	json.Unmarshal(body, &acc)
	
	if !acc.Ok {
		return "", fmt.Errorf("failed")
	}
	return acc.Result.AccessToken, nil
}

// 3. Диалог
func (a *App) OpenFolderDialog() (ChapterResponse, error) {
	selection, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите папку с главой",
	})
	if err != nil || selection == "" {
		return ChapterResponse{}, err
	}

	images, err := getImagesInDir(selection)
	if err != nil {
		return ChapterResponse{}, err
	}

	title := filepath.Base(selection)

	return ChapterResponse{
		Path:       selection,
		Title:      title,
		Images:     images,
		ImageCount: len(images),
	}, nil
}

func getImagesInDir(dirPath string) ([]string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".png") ||
		   strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".webp") {
			images = append(images, filepath.Join(dirPath, entry.Name()))
		}
	}
	sort.Strings(images)
	return images, nil
}
