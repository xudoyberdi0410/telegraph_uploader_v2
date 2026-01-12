package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Settings struct {
	ID               uint `gorm:"primaryKey"`
	Resize           bool
	ResizeTo         int
	WebpQuality      int
	LastChannelID    int64
	LastChannelHash  int64
	LastChannelTitle string
}

type Title struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Name      string          `gorm:"unique" json:"name"`
	Folders   []TitleFolder   `gorm:"foreignKey:TitleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"folders"`
	Variables []TitleVariable `gorm:"foreignKey:TitleID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"variables"`
}

type TitleFolder struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	TitleID uint   `json:"title_id"`
	Path    string `gorm:"index" json:"path"`
}

type TitleVariable struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	TitleID uint   `json:"title_id"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

type Template struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"unique" json:"name"`
	Content string `json:"content"`
}

type UploadedFile struct {
	Hash      string    `gorm:"primaryKey" json:"hash"` // Unique hash (SHA-256)
	URL       string    `json:"url"`                    // URL in R2
	CreatedAt time.Time `json:"created_at"`
}

// HistoryEntry maps to database table "history_items"
type HistoryEntry struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Title     string
	Url       string
	ImgCount  int
	TgphToken string
	TitleID   *uint
}

func (HistoryEntry) TableName() string {
	return "history_items"
}

// HistoryItem is the DTO for Frontend
type HistoryItem struct {
	ID        uint   `json:"id"`
	Date      string `json:"date"`
	Title     string `json:"title"`
	Url       string `json:"url"`
	ImgCount  int    `json:"img_count"`
	TgphToken string `json:"tgph_token"`
	TitleID   *uint  `json:"title_id"`
}

// Init инициализирует БД и создает файл database.db
func Init() (*gorm.DB, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(filepath.Dir(ex), "database.db")
	return InitWithFile(dbPath)
}

// InitWithFile initializes DB at specific path
func InitWithFile(dbPath string) (*gorm.DB, error) {
	// Настройка логгера
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	// Подключение (Pure Go Driver)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, err
	}

	// Автоматическая миграция
	err = db.AutoMigrate(&Settings{}, &HistoryEntry{}, &Title{}, &TitleFolder{}, &TitleVariable{}, &Template{}, &UploadedFile{})
	if err != nil {
		return nil, err
	}

	// Создаем дефолтные настройки, если их нет
	var count int64
	db.Model(&Settings{}).Count(&count)
	if count == 0 {
		db.Create(&Settings{
			Resize:      false,
			ResizeTo:    1600,
			WebpQuality: 80,
		})
	}

	return db, nil
}