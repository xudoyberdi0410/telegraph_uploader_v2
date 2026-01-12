package repository

import (
	"path/filepath"
	"telegraph_uploader_v2/internal/database"

	"gorm.io/gorm"
)

type TitleRepository interface {
	Create(name string, rootFolder string) error
	GetAll() ([]database.Title, error)
	GetByID(id uint) (database.Title, error)
	Update(t database.Title) error
	Delete(id uint) error
	AddVariable(titleID uint, key, value string) error
	FindByPath(path string) (database.Title, error)
}

type titleRepo struct {
	db *gorm.DB
}

func NewTitleRepository(db *gorm.DB) TitleRepository {
	return &titleRepo{db: db}
}

func (r *titleRepo) Create(name string, rootFolder string) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	title := database.Title{Name: name}
	if err := tx.Create(&title).Error; err != nil {
		tx.Rollback()
		return err
	}

	if rootFolder != "" {
		if err := tx.Create(&database.TitleFolder{TitleID: title.ID, Path: rootFolder}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (r *titleRepo) GetAll() ([]database.Title, error) {
	var titles []database.Title
	err := r.db.Preload("Folders").Preload("Variables").Find(&titles).Error
	return titles, err
}

func (r *titleRepo) GetByID(id uint) (database.Title, error) {
	var t database.Title
	err := r.db.Preload("Variables").Preload("Folders").First(&t, id).Error
	return t, err
}

func (r *titleRepo) Update(t database.Title) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(&t).Error
}

func (r *titleRepo) Delete(id uint) error {
	return r.db.Delete(&database.Title{}, id).Error
}

func (r *titleRepo) AddVariable(titleID uint, key, value string) error {
	variable := database.TitleVariable{
		TitleID: titleID,
		Key:     key,
		Value:   value,
	}
	return r.db.Create(&variable).Error
}

func (r *titleRepo) FindByPath(path string) (database.Title, error) {
	path = filepath.Clean(path)
	var folder database.TitleFolder

	err := r.db.Where("LOWER(?) LIKE LOWER(path || '%')", path).
		Order("length(path) DESC").
		First(&folder).Error

	if err != nil {
		return database.Title{}, err
	}

	return r.GetByID(folder.TitleID)
}
