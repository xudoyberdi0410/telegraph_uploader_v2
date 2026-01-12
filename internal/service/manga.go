package service

import (
	"context"
	"telegraph_uploader_v2/internal/uploader"
)

type MangaService struct {
	uploader *uploader.R2Uploader
}

func NewMangaService(upl *uploader.R2Uploader) *MangaService {
	return &MangaService{uploader: upl}
}

func (s *MangaService) UploadChapter(ctx context.Context, filePaths []string, settings uploader.ResizeSettings) uploader.UploadResult {
	if s.uploader == nil {
		return uploader.UploadResult{Success: false, Error: "Загрузчик не инициализирован"}
	}
	
	// Вызов R2
	return s.uploader.UploadChapter(ctx, filePaths, settings)
}
