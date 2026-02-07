package uploader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"telegraph_uploader_v2/internal/config"
	"telegraph_uploader_v2/internal/repository"

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
	client    *minio.Client
	cfg       *config.Config
	cacheRepo repository.ImageCacheRepository
}

type RemoteFile struct {
	Name         string `json:"name"`
	LastModified int64  `json:"last_modified"` // Unix timestamp
	Size         int64  `json:"size"`
	Url          string `json:"url"`
}

type ResizeSettings struct {
	Resize      bool `json:"resize"`
	ResizeTo    int  `json:"resize_to"`
	WebpQuality int  `json:"webp_quality"`
	MockR2      bool `json:"mock_r2"`
}

// New создает новый экземпляр загрузчика. Вызывается 1 раз при старте.
func New(cfg *config.Config, cacheRepo repository.ImageCacheRepository) (*R2Uploader, error) {
	endpoint := fmt.Sprintf("%s.r2.cloudflarestorage.com", cfg.R2AccountId)

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.R2AccessKey, cfg.R2SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		return nil, err
	}

	return &R2Uploader{
		client:    client,
		cfg:       cfg,
		cacheRepo: cacheRepo,
	}, nil
}

// NewWithClient creates uploader with specific minio client (useful for tests)
func NewWithClient(client *minio.Client, cfg *config.Config, cacheRepo repository.ImageCacheRepository) *R2Uploader {
	return &R2Uploader{
		client:    client,
		cfg:       cfg,
		cacheRepo: cacheRepo,
	}
}

func (u *R2Uploader) normalizeDomain() string {
	domain := strings.TrimRight(u.cfg.PublicDomain, "/")
	if !strings.HasPrefix(domain, "http") {
		domain = "https://" + domain
	}
	return domain
}

// ListAllFiles returns all files from the bucket
func (u *R2Uploader) ListAllFiles(ctx context.Context) ([]RemoteFile, error) {
	var files []RemoteFile

	opts := minio.ListObjectsOptions{
		Recursive: true,
	}

	objectCh := u.client.ListObjects(ctx, u.cfg.BucketName, opts)

	domain := u.normalizeDomain()

	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}

		files = append(files, RemoteFile{
			Name:         object.Key,
			LastModified: object.LastModified.Unix(),
			Size:         object.Size,
			Url:          fmt.Sprintf("%s/%s", domain, object.Key),
		})
	}

	return files, nil
}

// DeleteFiles removes multiple files from the bucket
func (u *R2Uploader) DeleteFiles(ctx context.Context, filenames []string) error {
	objectsCh := make(chan minio.ObjectInfo)

	go func() {
		defer close(objectsCh)
		for _, name := range filenames {
			objectsCh <- minio.ObjectInfo{
				Key: name,
			}
		}
	}()

	opts := minio.RemoveObjectsOptions{
		// GovernanceBypass: true, // R2 does not support this
	}

	errorCh := u.client.RemoveObjects(ctx, u.cfg.BucketName, objectsCh, opts)

	// Collect errors
	var errs []string
	for err := range errorCh {
		errs = append(errs, fmt.Sprintf("failed to remove %s: %v", err.ObjectName, err.Err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors deleting files: %s", strings.Join(errs, "; "))
	}

	return nil
}

// UploadChapter теперь использует errgroup для параллельной загрузки
func (u *R2Uploader) UploadChapter(ctx context.Context, filePaths []string, resizeSettings ResizeSettings, onProgress func(int, int)) UploadResult {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.NumCPU())

	uploadedLinks := make([]string, len(filePaths))
	var uploadErrors []string
	var mu sync.Mutex

	var processedCount int32
	totalFiles := len(filePaths)

	for i, path := range filePaths {
		i, path := i, path // Capture vars
		g.Go(func() error {
			// Проверяем контекст
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if resizeSettings.MockR2 {
				// MOCK MODE: Just wait a bit and return fake link
				time.Sleep(time.Millisecond * 2000)
				uploadedLinks[i] = fmt.Sprintf("https://cxc-images.khudoberdi.uz/1767902262155077900_D1000.webp?id=%d", i)

				// Progress update
				newCount := atomic.AddInt32(&processedCount, 1)
				if onProgress != nil {
					onProgress(int(newCount), totalFiles)
				}
				return nil
			}

			// --- НОВАЯ ЛОГИКА: ХЭШИРОВАНИЕ ---

			// 0. Читаем файл в память (ОПТИМИЗАЦИЯ: одно чтение вместо двух)
			fileData, err := os.ReadFile(path)
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Read error: %v", filepath.Base(path), err))
				mu.Unlock()
				return nil
			}

			// 1. Считаем хэш
			fileHash := calculateHash(fileData)

			// 2. Проверяем в базе
			if u.cacheRepo != nil { // Check if repo is available
				if cachedURL, found := u.cacheRepo.GetURL(fileHash); found {
					// УРА! Файл уже был загружен.
					uploadedLinks[i] = cachedURL
					// Progress update
					newCount := atomic.AddInt32(&processedCount, 1)
					if onProgress != nil {
						onProgress(int(newCount), totalFiles)
					}
					return nil
				}
			}
			// ----------------------------------

			// ШАГ 1: Обработка изображения
			processed, err := processImage(fileData, filepath.Base(path), resizeSettings)
			if err != nil {
				mu.Lock()
				uploadErrors = append(uploadErrors, fmt.Sprintf("[%s] Processing failed: %v", filepath.Base(path), err))
				mu.Unlock()
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
			domain := u.normalizeDomain()
			finalUrl := fmt.Sprintf("%s/%s", domain, processed.FileName)

			// --- НОВАЯ ЛОГИКА: СОХРАНЕНИЕ В КЭШ ---
			if u.cacheRepo != nil {
				_ = u.cacheRepo.Save(fileHash, finalUrl)
			}
			// --------------------------------------

			// Индексы уникальны, мьютекс не нужен для uploadedLinks
			uploadedLinks[i] = finalUrl

			// Progress update
			newCount := atomic.AddInt32(&processedCount, 1)
			if onProgress != nil {
				onProgress(int(newCount), totalFiles)
			}

			return nil
		})
	}

	// Ждем завершения всех горутин
	if err := g.Wait(); err != nil {
		return UploadResult{Success: false, Error: "Upload cancelled or failed: " + err.Error()}
	}

	if len(uploadErrors) > 0 {
		return UploadResult{Success: false, Error: fmt.Sprintf("Ошибок: %d. Первая: %s", len(uploadErrors), uploadErrors[0])}
	}

	return UploadResult{Success: true, Links: uploadedLinks}
}

func calculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

