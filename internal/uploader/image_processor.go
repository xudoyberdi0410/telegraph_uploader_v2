package uploader

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ProcessedImage содержит готовые данные для отправки
type ProcessedImage struct {
	Content  *bytes.Buffer
	FileName string
	Size     int64
}

// processImage берет путь, обрабатывает картинку и возвращает буфер + имя
func processImage(srcPath string) (*ProcessedImage, error) {
	// 1. Открытие
	img, err := imaging.Open(srcPath, imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("open error: %w", err)
	}

	// 2. Ресайз (бизнес-логика: ширина > 1200)
	if img.Bounds().Dx() > 1200 {
		img = imaging.Resize(img, 1200, 0, imaging.Lanczos)
	}

	// 3. Кодирование в WebP
	buf := new(bytes.Buffer)
	err = webp.Encode(buf, img, &webp.Options{
		Lossless: false,
		Quality:  60,
	})
	if err != nil {
		return nil, fmt.Errorf("encode error: %w", err)
	}

	// 4. Генерация имени
	originalName := filepath.Base(srcPath)
	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)
	fileName := fmt.Sprintf("%d_%s.webp", time.Now().UnixNano(), nameWithoutExt)

	return &ProcessedImage{
		Content:  buf,
		FileName: fileName,
		Size:     int64(buf.Len()),
	}, nil
}
