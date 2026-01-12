package repository

import (
	"telegraph_uploader_v2/internal/database"

	"gorm.io/gorm"
)

type ImageCacheRepository interface {
	GetURL(hash string) (string, bool)
	Save(hash, url string) error
}

type imageCacheRepo struct {
	db *gorm.DB
}

func NewImageCacheRepository(db *gorm.DB) ImageCacheRepository {
	return &imageCacheRepo{db: db}
}

func (r *imageCacheRepo) GetURL(hash string) (string, bool) {
	var item database.UploadedFile
	err := r.db.First(&item, "hash = ?", hash).Error
	if err != nil {
		return "", false
	}
	return item.URL, true
}

func (r *imageCacheRepo) Save(hash, url string) error {
	item := database.UploadedFile{
		Hash: hash,
		URL:  url,
	}
	return r.db.Save(&item).Error
}
