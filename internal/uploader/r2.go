package uploader

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	// Замените на название вашего модуля из go.mod
	"telegraph_uploader_v2/internal/config"

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

type ResizeSettings struct {
	Resize	bool	`json:"resize"`
	ResizeTo int	`json:"resize_to"`
	WebpQuality int	`json:"webp_quality"`
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

// NewWithClient creates uploader with specific minio client (useful for tests)
func NewWithClient(client *minio.Client, cfg *config.Config) *R2Uploader {
	return &R2Uploader{
		client: client,
		cfg:    cfg,
	}
}

// UploadChapter теперь метод структуры, а не просто функция
func (u *R2Uploader) UploadChapter(filePaths []string, resizeSettings ResizeSettings) UploadResult {
	maxWorkers := runtime.NumCPU()
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

			// ШАГ 1: Делегируем обработку изображения другой функции
			processed, err := processImage(srcPath, resizeSettings)
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Processing failed: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// ШАГ 2: Загрузка (сетевая операция)
			_, err = u.client.PutObject(context.Background(), u.cfg.BucketName, processed.FileName, processed.Content, processed.Size, minio.PutObjectOptions{
				ContentType: "image/webp",
			})
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Upload error: %v", filepath.Base(srcPath), err))
				mu.Unlock()
				return
			}

			// ШАГ 3: Формирование ссылки
			domain := strings.TrimRight(u.cfg.PublicDomain, "/")
			finalUrl := fmt.Sprintf("%s/%s", domain, processed.FileName)

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
