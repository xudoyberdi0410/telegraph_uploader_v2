package uploader

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

    // Замените на название вашего модуля из go.mod
	"telegraph_uploader_v2/internal/config"

	"github.com/disintegration/imaging"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Структуры ответа для фронтенда
type UploadResult struct {
	Success bool     `json:"success"`
	Links   []string `json:"links"`
	Error   string   `json:"error"`
}

// R2Uploader хранит состояние: готовый клиент и конфиг
type R2Uploader struct {
	client *minio.Client
	cfg    *config.Config
}

// New создает новый экземпляр загрузчика. Вызывается 1 раз при старте.
func New(cfg *config.Config) (*R2Uploader, error) {
	endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", cfg.R2AccountId)
	
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.R2AccessKey, cfg.R2SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}

	return &R2Uploader{
		client: client,
		cfg:    cfg,
	}, nil
}

// UploadChapter теперь метод структуры, а не просто функция
func (u *R2Uploader) UploadChapter(filePaths []string) UploadResult {
	// Конфиг и клиент уже есть внутри 'u', не нужно их проверять/создавать заново

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

			// 2. Ресайз (чуть оптимизировал: если меньше 1600, не трогаем)
			if img.Bounds().Dx() > 1600 {
				img = imaging.Resize(img, 1600, 0, imaging.Lanczos)
			}

			// 3. Кодирование
			buf := new(bytes.Buffer)
			// Quality 80 - оптимальный баланс
			err = imaging.Encode(buf, img, imaging.JPEG, imaging.JPEGQuality(80))
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Encode error: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// 4. Загрузка
			// Используем u.cfg для бакета
			fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(srcPath))

			// Используем u.client
			_, err = u.client.PutObject(context.Background(), u.cfg.BucketName, fileName, buf, int64(buf.Len()), minio.PutObjectOptions{
				ContentType: "image/jpeg",
			})
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Upload error: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// 5. Формирование ссылки
			domain := strings.TrimRight(u.cfg.PublicDomain, "/")
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
