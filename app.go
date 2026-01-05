package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"telegraph_uploader_v2/internal/config"
	"telegraph_uploader_v2/internal/database"
	"telegraph_uploader_v2/internal/telegram"
	"telegraph_uploader_v2/internal/telegraph"
	"telegraph_uploader_v2/internal/uploader"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx      context.Context
	config   *config.Config
	uploader *uploader.R2Uploader
	tgClient *telegraph.Client
	db       *database.Database
}

func NewApp() *App {
	log.Println("[App] Starting initialization...")

	cfg, err := config.Load()
	if err != nil {
		// Если конфига нет, можно упасть или создать пустой.
		// Для GUI приложений лучше логировать, но не падать сразу.
		log.Println("[App] Warning: could not load config:", err)
		cfg = &config.Config{} // Пустой конфиг, чтобы не было nil pointer
	} else {
		log.Println("[App] Config loaded successfully")
	}

	dbService, err := database.Init()
	if err != nil {
		log.Fatal("[App] Database init error:", err)
	}
	log.Println("[App] Database initialized")

	upl, err := uploader.New(cfg)
	if err != nil {
		log.Println("[App] Uploader init error:", err)
	} else {
		log.Println("[App] Uploader initialized")
	}

	tg := telegraph.New(cfg)
	log.Println("[App] Telegraph client initialized")

	return &App{
		config:   cfg,
		uploader: upl,
		tgClient: tg,
		db:       dbService,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("[App] Application startup complete")
}

// === СТРУКТУРЫ ===
type ChapterResponse struct {
	Path       string   `json:"path"`
	Title      string   `json:"title"`
	Images     []string `json:"images"`
	ImageCount int      `json:"imageCount"`
}

// === МЕТОДЫ ===

func (a *App) UploadChapter(filePaths []string, resizeSettings uploader.ResizeSettings) uploader.UploadResult {
	log.Printf("[App] UploadChapter called. Files: %d, Settings: %+v", len(filePaths), resizeSettings)

	if a.uploader == nil {
		errMsg := "Загрузчик не инициализирован (проверьте config.json)"
		log.Println("[App] Error: " + errMsg)
		return uploader.UploadResult{Success: false, Error: errMsg}
	}

	// Просто вызываем метод нашего сервиса
	result := a.uploader.UploadChapter(filePaths, resizeSettings)

	if result.Success {
		log.Printf("[App] UploadChapter finished successfully. URLs generated: %d", len(result.Links))
	} else {
		log.Printf("[App] UploadChapter failed. Error: %s", result.Error)
	}

	return result
}

func (a *App) CreateTelegraphPage(title string, imageUrls []string) string {
	log.Printf("[App] CreateTelegraphPage called. Title: '%s', Images: %d", title, len(imageUrls))

	url := a.tgClient.CreatePage(title, imageUrls)

	if len(url) > 4 && url[:4] == "http" {
		log.Printf("[App] Page created successfully: %s", url)
		err := a.db.AddHistory(title, url, len(imageUrls), a.config.TelegraphToken)
		if err != nil {
			log.Printf("[App] Warning: Failed to save to history: %v", err)
		} else {
			log.Println("[App] Saved to history")
		}
	} else {
		log.Printf("[App] Failed to create page. Result URL/Error: %s", url)
	}

	return url
}

// Диалог выбора папки
func (a *App) OpenFolderDialog() (ChapterResponse, error) {
	log.Println("[App] OpenFolderDialog called")

	selection, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите папку с главой",
	})
	if err != nil {
		log.Printf("[App] OpenDirectoryDialog error: %v", err)
		return ChapterResponse{}, err
	}
	if selection == "" {
		log.Println("[App] OpenFolderDialog canceled by user")
		return ChapterResponse{}, nil // Или вернуть ошибку, если фронтенд ждет
	}

	log.Printf("[App] Folder selected: %s", selection)

	images, err := getImagesInDir(selection)
	if err != nil {
		log.Printf("[App] Error reading directory: %v", err)
		return ChapterResponse{}, err
	}

	title := filepath.Base(selection)
	log.Printf("[App] Found %d images in folder '%s'", len(images), title)

	return ChapterResponse{
		Path:       selection,
		Title:      title,
		Images:     images,
		ImageCount: len(images),
	}, nil
}

// Выбор отдельных файлов
func (a *App) OpenFilesDialog() ([]string, error) {
	log.Println("[App] OpenFilesDialog called")

	selection, err := wailsRuntime.OpenMultipleFilesDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите изображения",
		Filters: []wailsRuntime.FileFilter{
			{
				DisplayName: "Images",
				Pattern:     "*.jpg;*.jpeg;*.png;*.webp",
			},
		},
	})

	if err != nil {
		log.Printf("[App] OpenMultipleFilesDialog error: %v", err)
		return nil, err
	}

	if len(selection) == 0 {
		log.Println("[App] OpenFilesDialog canceled or no files selected")
	} else {
		log.Printf("[App] Selected %d files", len(selection))
	}

	return selection, nil
}

func getImagesInDir(dirPath string) ([]string, error) {
	// log.Printf("[App] Scanning directory: %s", dirPath) // Можно раскомментировать для отладки
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		lower := strings.ToLower(entry.Name())
		if strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".png") ||
			strings.HasSuffix(lower, ".jpeg") || strings.HasSuffix(lower, ".webp") {
			images = append(images, filepath.Join(dirPath, entry.Name()))
		}
	}
	sort.Strings(images)
	return images, nil
}

// GetSettings вызывается фронтендом при старте
func (a *App) GetSettings() uploader.ResizeSettings {
	log.Println("[App] GetSettings called")
	s := a.db.GetSettings()
	log.Printf("[App] Returning settings: %+v", s)

	return uploader.ResizeSettings{
		Resize:      s.Resize,
		ResizeTo:    s.ResizeTo,
		WebpQuality: s.WebpQuality,
	}
}

// SaveSettings вызывается фронтендом при любом изменении
func (a *App) SaveSettings(s uploader.ResizeSettings) {
	log.Printf("[App] SaveSettings called: %+v", s)

	a.db.UpdateSettings(database.Settings{
		Resize:      s.Resize,
		ResizeTo:    s.ResizeTo,
		WebpQuality: s.WebpQuality,
	})
	log.Println("[App] Settings saved")
}

func (a *App) GetHistory(limit int, offset int) []database.HistoryItem {
	log.Printf("[App] GetHistory called (limit: %d, offset: %d)", limit, offset)
	items := a.db.GetHistory(limit, offset)
	log.Printf("[App] Returned %d history items", len(items))
	return items
}

func (a *App) ClearHistory() {
	log.Println("[App] ClearHistory called")
	a.db.ClearHistory()
	log.Println("[App] History cleared")
}

// В App добавим:
func (a *App) SendToTelegram(botToken, channelID, text string, scheduleUnix int64) string {
	var scheduleTime time.Time
	if scheduleUnix > 0 {
		scheduleTime = time.Unix(scheduleUnix, 0)
	}

	err := telegram.SendScheduledMessage(botToken, channelID, text, scheduleTime)
	if err != nil {
		return "Ошибка: " + err.Error()
	}
	return "Успешно отправлено!"
}

// Добавьте также методы Get/Save для ботов, каналов и шаблонов, вызывающие БД

func (a *App) GetTgBots() []database.TgBot {
	return a.db.GetTgBots()
}

func (a *App) SaveTgBot(bot database.TgBot) {
	a.db.SaveTgBot(bot)
}

func (a *App) GetTgChannels() []database.TgChannel {
	return a.db.GetTgChannels()
}

func (a *App) SaveTgChannel(channel database.TgChannel) {
	a.db.SaveTgChannel(channel)
}
func (a *App) GetTgTemplates() []database.TgTemplate {
	return a.db.GetTgTemplates()
}

func (a *App) SaveTgTemplate(template database.TgTemplate) {
	a.db.SaveTgTemplate(template)
}

