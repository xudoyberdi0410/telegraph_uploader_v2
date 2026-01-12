package repository

import (
	"telegraph_uploader_v2/internal/database"

	"gorm.io/gorm"
)

type SettingsRepository interface {
	Get() (database.Settings, error)
	Update(s database.Settings) error
}

type settingsRepo struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepo{db: db}
}

func (r *settingsRepo) Get() (database.Settings, error) {
	var s database.Settings
	// Bereм первую запись (она там одна)
	err := r.db.First(&s).Error
	return s, err
}

func (r *settingsRepo) Update(s database.Settings) error {
	// Обновляем запись с ID=1
	s.ID = 1
	return r.db.Save(&s).Error
}
