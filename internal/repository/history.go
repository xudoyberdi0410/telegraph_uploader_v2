package repository

import (
	"telegraph_uploader_v2/internal/database"
	"time"

	"gorm.io/gorm"
)

type HistoryRepository interface {
	Add(title, url string, imgCount int, tgphToken string, titleID *uint) (uint, error)
	Get(limit, offset int) ([]database.HistoryItem, error)
	GetByID(id uint) (database.HistoryItem, error)
	Clear() error
}

type historyRepo struct {
	db *gorm.DB
}

func NewHistoryRepository(db *gorm.DB) HistoryRepository {
	return &historyRepo{db: db}
}

func (r *historyRepo) Add(title, url string, imgCount int, tgphToken string, titleID *uint) (uint, error) {
	item := database.HistoryEntry{
		Title:     title,
		Url:       url,
		ImgCount:  imgCount,
		TgphToken: tgphToken,
		TitleID:   titleID,
		CreatedAt: time.Now(), // Gorm usually handles this, but explicit is fine
	}
	err := r.db.Create(&item).Error
	return item.ID, err
}

func (r *historyRepo) Get(limit, offset int) ([]database.HistoryItem, error) {
	var dbItems []database.HistoryEntry
	err := r.db.Order("created_at desc").Limit(limit).Offset(offset).Find(&dbItems).Error
	if err != nil {
		return nil, err
	}

	result := make([]database.HistoryItem, len(dbItems))
	for i, item := range dbItems {
		result[i] = database.HistoryItem{
			ID:        item.ID,
			Date:      item.CreatedAt.Format("2006-01-02 15:04:05"),
			Title:     item.Title,
			Url:       item.Url,
			ImgCount:  item.ImgCount,
			TgphToken: item.TgphToken,
			TitleID:   item.TitleID,
		}
	}
	return result, nil
}

func (r *historyRepo) GetByID(id uint) (database.HistoryItem, error) {
	var item database.HistoryEntry
	err := r.db.First(&item, id).Error
	if err != nil {
		return database.HistoryItem{}, err
	}
	return database.HistoryItem{
		ID:        item.ID,
		Date:      item.CreatedAt.Format("2006-01-02 15:04:05"),
		Title:     item.Title,
		Url:       item.Url,
		ImgCount:  item.ImgCount,
		TgphToken: item.TgphToken,
		TitleID:   item.TitleID,
	}, nil
}

func (r *historyRepo) Clear() error {
	return r.db.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&database.HistoryEntry{}).Error
}
