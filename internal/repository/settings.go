package repository

import (
	"sync"
	"telegraph_uploader_v2/internal/database"

	"gorm.io/gorm"
)

type SettingsRepository interface {
	Get() (database.Settings, error)
	Update(s database.Settings) error
}

type settingsRepo struct {
	db    *gorm.DB
	cache *database.Settings
	mu    sync.RWMutex
}

func NewSettingsRepository(db *gorm.DB) SettingsRepository {
	return &settingsRepo{db: db}
}

func (r *settingsRepo) Get() (database.Settings, error) {
	r.mu.RLock()
	if r.cache != nil {
		s := *r.cache
		r.mu.RUnlock()
		return s, nil
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-checked locking
	if r.cache != nil {
		return *r.cache, nil
	}

	var s database.Settings
	// Bereм первую запись (она там одна)
	err := r.db.First(&s).Error
	if err == nil {
		r.cache = &s
	}
	return s, err
}

func (r *settingsRepo) Update(s database.Settings) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Обновляем запись с ID=1
	s.ID = 1
	err := r.db.Save(&s).Error
	if err == nil {
		r.cache = &s
	}
	return err
}
