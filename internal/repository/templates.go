package repository

import (
	"telegraph_uploader_v2/internal/database"

	"gorm.io/gorm"
)

type TemplateRepository interface {
	Create(name, content string) error
	GetAll() ([]database.Template, error)
	GetByID(id uint) (database.Template, error)
	Update(t database.Template) error
	Delete(id uint) error
}

type templateRepo struct {
	db *gorm.DB
}

func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepo{db: db}
}

func (r *templateRepo) Create(name, content string) error {
	return r.db.Create(&database.Template{Name: name, Content: content}).Error
}

func (r *templateRepo) GetAll() ([]database.Template, error) {
	var t []database.Template
	err := r.db.Find(&t).Error
	return t, err
}

func (r *templateRepo) GetByID(id uint) (database.Template, error) {
	var t database.Template
	err := r.db.First(&t, id).Error
	return t, err
}

func (r *templateRepo) Update(t database.Template) error {
	return r.db.Save(&t).Error
}

func (r *templateRepo) Delete(id uint) error {
	return r.db.Delete(&database.Template{}, id).Error
}
