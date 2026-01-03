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
	ID          uint `gorm:"primaryKey"`
	Resize      bool
	ResizeTo    int
	WebpQuality int
}

type dbHistory struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Title     string
	Url       string
	ImgCount  int
	TgphToken string
}

func (dbHistory) TableName() string {
	return "history_items"
}

type HistoryItem struct {
	ID        uint   `json:"id"`
	Date      string `json:"date"`
	Title     string `json:"title"`
	Url       string `json:"url"`
	ImgCount  int    `json:"img_count"`
	TgphToken string `json:"tgph_token"`
}

type Database struct {
	conn *gorm.DB
}

// Init инициализирует БД и создает файл database.db
func Init() (*Database, error) {
	// Определяем путь к файлу БД (рядом с exe)
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(filepath.Dir(ex), "database.db")

	// Настройка логгера (чтобы не спамил в консоль, если не нужно)
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Error, // Показывать только ошибки
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

	// Автоматическая миграция (создает таблицы, если их нет)
	err = db.AutoMigrate(&Settings{}, &dbHistory{})
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

	return &Database{conn: db}, nil
}

// --- Методы для Настроек ---

func (d *Database) GetSettings() Settings {
	var s Settings
	// Берем первую запись (она там одна)
	d.conn.First(&s)
	return s
}

func (d *Database) UpdateSettings(s Settings) {
	// Обновляем запись с ID=1
	s.ID = 1
	d.conn.Save(&s)
}

// --- Методы для Истории ---

func (d *Database) AddHistory(title, url string, imgCount int, tgphToken string) {
	d.conn.Create(&dbHistory{
		Title:     title,
		Url:       url,
		ImgCount:  imgCount,
		TgphToken: tgphToken,
	})
}

// GetHistory возвращает последние N записей
func (d *Database) GetHistory(limit int, offset int) []HistoryItem {
	var dbItems []dbHistory
	// Достаем из БД сырые данные
	d.conn.Order("created_at desc").Limit(limit).Offset(offset).Find(&dbItems)

	// Конвертируем в формат для фронтенда
	result := make([]HistoryItem, len(dbItems))
	for i, item := range dbItems {
		result[i] = HistoryItem{
			ID:       item.ID,
			Date:     item.CreatedAt.Format("2006-01-02 15:04:05"), // Форматируем дату здесь!
			Title:    item.Title,
			Url:      item.Url,
			ImgCount: item.ImgCount,
		}
	}
	return result

}

func (d *Database) ClearHistory() {
	// Удаляет все записи (мягкое удаление или полное - тут используем Unscoped для полного)
	d.conn.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbHistory{})
}
