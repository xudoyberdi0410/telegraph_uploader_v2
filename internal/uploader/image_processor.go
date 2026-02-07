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

// processImage берет данные и имя файла, обрабатывает картинку и возвращает буфер + имя
func processImage(data []byte, filename string, resizeSettings ResizeSettings) (*ProcessedImage, error) {
	// 1. Открытие
	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	// 2. Ресайз (бизнес-логика: ширина > 1200)
	if img.Bounds().Dx() > resizeSettings.ResizeTo && resizeSettings.Resize {
		img = imaging.Resize(img, resizeSettings.ResizeTo, 0, imaging.MitchellNetravali)
	}

	// 3. Кодирование в WebP
	buf := new(bytes.Buffer)
	err = webp.Encode(buf, img, &webp.Options{
		Lossless: false,
		Quality:  float32(resizeSettings.WebpQuality),
	})
	if err != nil {
		return nil, fmt.Errorf("encode error: %w", err)
	}

	// 4. Генерация имени
	originalName := filename
	ext := filepath.Ext(originalName)
	nameWithoutExt := strings.TrimSuffix(originalName, ext)
	fileName := fmt.Sprintf("%d_%s.webp", time.Now().UnixNano(), nameWithoutExt)

	return &ProcessedImage{
		Content:  buf,
		FileName: fileName,
		Size:     int64(buf.Len()),
	}, nil
}
