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

type dbHistory struct {
	ID        uint `gorm:"primaryKey"`
	CreatedAt time.Time
	Title     string
	Url       string
	ImgCount  int
	TgphToken string
	TitleID   *uint
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
	TitleID   *uint  `json:"title_id"`
}

type Database struct {
	conn *gorm.DB
}

// Close closes the underlying database connection
func (d *Database) Close() error {
	sqlDB, err := d.conn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Init инициализирует БД и создает файл database.db
func Init() (*Database, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(filepath.Dir(ex), "database.db")
	return InitWithFile(dbPath)
}

// InitWithFile initializes DB at specific path
func InitWithFile(dbPath string) (*Database, error) {
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
	err = db.AutoMigrate(&Settings{}, &dbHistory{}, &Title{}, &TitleFolder{}, &TitleVariable{}, &Template{})
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

// New allows creating Database with existing connection (useful for tests)
func New(db *gorm.DB) *Database {
	// AutoMigrate is idempotent, safe to call
	db.AutoMigrate(&Settings{}, &dbHistory{}, &Title{}, &TitleFolder{}, &TitleVariable{}, &Template{})
	return &Database{conn: db}
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

func (d *Database) AddHistory(title, url string, imgCount int, tgphToken string, titleID *uint) (uint, error) {
	item := dbHistory{
		Title:     title,
		Url:       url,
		ImgCount:  imgCount,
		TgphToken: tgphToken,
		TitleID:   titleID,
	}
	err := d.conn.Create(&item).Error
	return item.ID, err
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
			ID:        item.ID,
			Date:      item.CreatedAt.Format("2006-01-02 15:04:05"), // Форматируем дату здесь!
			Title:     item.Title,
			Url:       item.Url,
			ImgCount:  item.ImgCount,
			TgphToken: item.TgphToken, // <--- Добавили токен
			TitleID:   item.TitleID,
		}
	}
	return result

}

func (d *Database) ClearHistory() {
	// Удаляет все записи (мягкое удаление или полное - тут используем Unscoped для полного)
	d.conn.Unscoped().Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&dbHistory{})
}

// --- Методы для Настроек ---

func (d *Database) GetHistoryByID(id uint) (HistoryItem, error) {
	var item dbHistory
	err := d.conn.First(&item, id).Error
	if err != nil {
		return HistoryItem{}, err
	}
	return HistoryItem{
		ID:        item.ID,
		Date:      item.CreatedAt.Format("2006-01-02 15:04:05"),
		Title:     item.Title,
		Url:       item.Url,
		ImgCount:  item.ImgCount,
		TgphToken: item.TgphToken,
		TitleID:   item.TitleID,
	}, nil
}

// --- Методы для Тайтлов ---

func (d *Database) CreateTitle(name string, rootFolder string) error {
	tx := d.conn.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	title := Title{Name: name}
	if err := tx.Create(&title).Error; err != nil {
		tx.Rollback()
		return err
	}

	if rootFolder != "" {
		if err := tx.Create(&TitleFolder{TitleID: title.ID, Path: rootFolder}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

func (d *Database) GetTitles() []Title {
	var titles []Title
	d.conn.Preload("Folders").Preload("Variables").Find(&titles)
	return titles
}

func (d *Database) UpdateTitle(t Title) error {
	// Gorm full update with associations is tricky.
	// Easiest is to save basic info, then replace associations if needed.
	// For now, let's just save the Title struct.
	// But we need to handle Folders and Variables.
	// Simplest: Delete existing folders/vars and re-create? Or just Save.

	return d.conn.Session(&gorm.Session{FullSaveAssociations: true}).Save(&t).Error
}

func (d *Database) DeleteTitle(id uint) error {
	return d.conn.Delete(&Title{}, id).Error
}

func (d *Database) GetTitleByID(id uint) (Title, error) {
	var t Title
	err := d.conn.Preload("Variables").Preload("Folders").First(&t, id).Error
	return t, err
}

func (d *Database) AddTitleVariable(titleID uint, key, value string) error {
	variable := TitleVariable{
		TitleID: titleID,
		Key:     key,
		Value:   value,
	}
	return d.conn.Create(&variable).Error
}

// SearchTitleByPath tries to find a title that has a folder matching the path
// Logic: exact match or maybe parent match?
// User said: "program based on path to pages determines what manga it is"
// If we store "/foo/bar/manga_name", and images are in "/foo/bar/manga_name/chapter1",
// we should match if the folder path starts with the stored path.
func (d *Database) FindTitleByPath(path string) (Title, error) {
	path = filepath.Clean(path)
	var folder TitleFolder

	// Optimizing to use SQL instead of in-memory loop.
	// We want to find a folder where: input_path starts with folder.path
	// In SQL: input_path LIKE folder.path || '%'
	// We sort by length(path) DESC to find the most specific (longest) match.
	// Using LOWER for case-insensitivity on Windows.
	err := d.conn.Where("LOWER(?) LIKE LOWER(path || '%')", path).
		Order("length(path) DESC").
		First(&folder).Error

	if err != nil {
		return Title{}, err
	}

	return d.GetTitleByID(folder.TitleID)
}

// --- Методы для Шаблонов ---

func (d *Database) CreateTemplate(name, content string) error {
	return d.conn.Create(&Template{Name: name, Content: content}).Error
}

func (d *Database) GetTemplates() []Template {
	var t []Template
	d.conn.Find(&t)
	return t
}

func (d *Database) UpdateTemplate(t Template) error {
	return d.conn.Save(&t).Error
}

func (d *Database) GetTemplateByID(id uint) (Template, error) {
	var t Template
	err := d.conn.First(&t, id).Error
	return t, err
}

func (d *Database) DeleteTemplate(id uint) error {
	return d.conn.Delete(&Template{}, id).Error
}
