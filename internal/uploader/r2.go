package uploader

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"telegraph_uploader_v2/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"golang.org/x/sync/errgroup"
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
	Resize      bool `json:"resize"`
	ResizeTo    int  `json:"resize_to"`
	WebpQuality int  `json:"webp_quality"`
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

// UploadChapter теперь использует errgroup для параллельной загрузки
func (u *R2Uploader) UploadChapter(ctx context.Context, filePaths []string, resizeSettings ResizeSettings) UploadResult {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	uploadedLinks := make([]string, len(filePaths))
	var uploadErrors []string
	var mu sync.Mutex // Mutex для защиты записи ошибок и ссылок (хотя ссылки по индексу можно писать и без него, но для безопасности оставим, или уберем, т.к. индексы разные)
	// Ссылки по уникальным индексам можно писать без мьютекса.
	// Ошибки - нужен мьютекс.

	for i, path := range filePaths {
		i, path := i, path // Capture vars
		g.Go(func() error {
			// Проверяем контекст (хотя minio.PutObject тоже его проверяет)
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// ШАГ 1: Обработка изображения
			processed, err := processImage(path, resizeSettings)
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Processing failed: %v", filepath.Base(path), err))
				mu.Unlock()
				// Если хотим остановить всё при первой ошибке - return err.
				// Если хотим попробовать загрузить остальные - return nil (но логируем ошибку).
				// Текущая логика была: собирать ошибки. Оставим её, чтобы не прерывать весь батч из-за одной битой картинки.
				return nil
			}

			// ШАГ 2: Загрузка
			_, err = u.client.PutObject(ctx, u.cfg.BucketName, processed.FileName, processed.Content, processed.Size, minio.PutObjectOptions{
				ContentType: "image/webp",
			})
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Upload error: %v", filepath.Base(path), err))
				mu.Unlock()
				return nil
			}

			// ШАГ 3: Формирование ссылки
			domain := strings.TrimRight(u.cfg.PublicDomain, "/")
			finalUrl := fmt.Sprintf("%s/%s", domain, processed.FileName)

			// Индексы уникальны, мьютекс не нужен для uploadedLinks
			uploadedLinks[i] = finalUrl
			return nil
		})
	}

	// Ждем завершения всех горутин
	// Так как мы возвращаем nil из горутин (кроме ctx error), Wait вернет ошибку только если контекст отменили.
	if err := g.Wait(); err != nil {
		return UploadResult{Success: false, Error: "Upload cancelled or failed: " + err.Error()}
	}

	if len(uploadErrors) > 0 {
		return UploadResult{Success: false, Error: fmt.Sprintf("Ошибок: %d. Первая: %s", len(uploadErrors), uploadErrors[0])}
	}

	return UploadResult{Success: true, Links: uploadedLinks}
}
