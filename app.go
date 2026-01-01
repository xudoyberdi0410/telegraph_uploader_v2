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

// Config - структура настроек
type Config struct {
	R2AccountId    string `json:"r2_account_id"`
	R2AccessKey    string `json:"r2_access_key"`
	R2SecretKey    string `json:"r2_secret_key"`
	BucketName     string `json:"bucket_name"`
	PublicDomain   string `json:"public_domain"`
	TelegraphToken string `json:"telegraph_token"` // Опционально: токен Telegraph
}

type App struct {
	ctx    context.Context
	config Config
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 1. Загружаем конфиг
	if err := a.loadConfig(); err != nil {
		// Критическая ошибка - показываем диалог
		wailsRuntime.MessageDialog(ctx, wailsRuntime.MessageDialogOptions{
			Type:          wailsRuntime.ErrorDialog,
			Title:         "Ошибка конфигурации",
			Message:       fmt.Sprintf("Не удалось загрузить config.json:\n%s\n\nУбедитесь, что файл находится рядом с программой.", err.Error()),
			Buttons:       []string{"OK"},
			DefaultButton: "OK",
		})
	}
}

// loadConfig ищет config.json рядом с исполняемым файлом
func (a *App) loadConfig() error {
	// Получаем путь к исполняемому файлу
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	exPath := filepath.Dir(ex)

	// Сначала пробуем найти конфиг рядом с .exe
	configPath := filepath.Join(exPath, "config.json")
	
	// Если не нашли (например, при wails dev), пробуем в текущей директории
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "config.json"
	}

	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	bytesValue, _ := io.ReadAll(file)
	err = json.Unmarshal(bytesValue, &a.config)
	if err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Валидация обязательных полей
	if a.config.R2AccountId == "" || a.config.R2AccessKey == "" || a.config.R2SecretKey == "" {
		return fmt.Errorf("в config.json не заполнены ключи R2")
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
	// Проверка перед стартом
	if a.config.R2AccountId == "" {
		return UploadResult{Success: false, Error: "Конфигурация не загружена. Проверьте config.json"}
	}

	endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", a.config.R2AccountId)
	
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(a.config.R2AccessKey, a.config.R2SecretKey, ""),
		Secure: true,
	})

	if err != nil {
		return UploadResult{Success: false, Error: "Ошибка клиента R2: " + err.Error()}
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

			// 1. Открытие
			img, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Open error: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// 2. Ресайз
			if img.Bounds().Dx() > 1600 {
				img = imaging.Resize(img, 1600, 0, imaging.Lanczos)
			}

			// 3. Кодирование
			buf := new(bytes.Buffer)
			err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(80))
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Encode error: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// 4. Загрузка
			fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(srcPath))
			
			_, err = client.PutObject(context.Background(), a.config.BucketName, fileName, buf, int64(buf.Len()), minio.PutObjectOptions{
				ContentType: "image/jpeg",
			})
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Upload error: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// 5. Формирование ссылки
			domain := strings.TrimRight(a.config.PublicDomain, "/")
			finalUrl := fmt.Sprintf("%s/%s", domain, fileName)
			
			mu.Lock()
			uploadedLinks[index] = finalUrl
			mu.Unlock()
		}(i, path)
	}

	wg.Wait()

	if len(uploadErrors) > 0 {
		return UploadResult{Success: false, Error: fmt.Sprintf("Ошибок: %d. Первая: %s", len(uploadErrors), uploadErrors[0])}
	}

	return UploadResult{Success: true, Links: uploadedLinks}
}

func (a *App) CreateTelegraphPage(title string, imageUrls []string) string {
	// Определяем токен: либо из конфига, либо создаем новый
	token := a.config.TelegraphToken
	if token == "" {
		var err error
		token, err = createTelegraphAccount("MangaUploader")
		if err != nil {
			return "Ошибка создания аккаунта Telegraph: " + err.Error()
		}
		// В идеале тут можно сохранить токен обратно в config.json, но пока не будем усложнять
		fmt.Println("ВНИМАНИЕ: Создан новый временный токен Telegraph:", token)
	}

	// Формируем контент
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
		return "Ошибка JSON: " + err.Error()
	}

	apiURL := "https://api.telegra.ph/createPage"
	data := url.Values{}
	data.Set("access_token", token)
	data.Set("title", title)
	data.Set("content", string(contentJson))
	data.Set("return_content", "false")

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return "Ошибка сети Telegraph: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	var tgResp TelegraphResponse
	if err := json.Unmarshal(body, &tgResp); err != nil {
		return "Ошибка ответа API: " + string(body)
	}

	if !tgResp.Ok {
		return "Telegraph API Error: " + tgResp.Error
	}

	return tgResp.Result.Url
}

func createTelegraphAccount(shortName string) (string, error) {
	apiURL := "https://api.telegra.ph/createAccount"
	data := url.Values{}
	data.Set("short_name", shortName)
	data.Set("author_name", "MangaBot")

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
		return "", fmt.Errorf("не удалось создать аккаунт")
	}
	return acc.Result.AccessToken, nil
}

// Диалог выбора папки (без изменений)
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

// НОВЫЙ МЕТОД: Выбор отдельных файлов
func (a *App) OpenFilesDialog() ([]string, error) {
	selection, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите изображения",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Images",
				Pattern:     "*.jpg;*.jpeg;*.png;*.webp",
			},
		},
	})
	
	if err != nil {
		return nil, err
	}

	return selection, nil
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
