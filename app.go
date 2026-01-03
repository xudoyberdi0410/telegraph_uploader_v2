package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"telegraph_uploader_v2/internal/config"
	"telegraph_uploader_v2/internal/database"
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
	cfg, err := config.Load()
	if err != nil {
		// Если конфига нет, можно упасть или создать пустой.
		// Для GUI приложений лучше логировать, но не падать сразу,
		// чтобы можно было показать ошибку в окне (если реализуете).
		log.Println("Warning: could not load config:", err)
		cfg = &config.Config{} // Пустой конфиг, чтобы не было nil pointer
	}

	dbService, err := database.Init()
	if err != nil {
		log.Fatal("Database init error:", err)
	}

	upl, err := uploader.New(cfg)
	if err != nil {
		log.Println("Uploader init error:", err)
	}
	tg := telegraph.New(cfg)

	return &App{
		config:   cfg,
		uploader: upl,
		tgClient: tg,
		db:       dbService,
	}
}
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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
	if a.uploader == nil {
		return uploader.UploadResult{Success: false, Error: "Загрузчик не инициализирован (проверьте config.json)"}
	}
	// Просто вызываем метод нашего сервиса
	return a.uploader.UploadChapter(filePaths, resizeSettings)
}

func (a *App) CreateTelegraphPage(title string, imageUrls []string) string {
	url := a.tgClient.CreatePage(title, imageUrls)

	if len(url) > 4 && url[:4] == "http" {
		a.db.AddHistory(title, url, len(imageUrls), a.config.TelegraphToken)
	}
	return url
}

// Диалог выбора папки (без изменений)
func (a *App) OpenFolderDialog() (ChapterResponse, error) {
	selection, err := wailsRuntime.OpenDirectoryDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите папку с главой",
	})
	if err != nil || selection == "" {
		return ChapterResponse{}, err
	}

	images, err := getImagesInDir(selection)
	if err != nil {
		return ChapterResponse{}, err
	}

	title := filepath.Base(selection)

	return ChapterResponse{
		Path:       selection,
		Title:      title,
		Images:     images,
		ImageCount: len(images),
	}, nil
}

// НОВЫЙ МЕТОД: Выбор отдельных файлов
func (a *App) OpenFilesDialog() ([]string, error) {
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
		return nil, err
	}

	return selection, nil
}

func getImagesInDir(dirPath string) ([]string, error) {
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

// func (a *App) getSettingsPath() string {
// 	ex, err := os.Executable()
// 	if err != nil {
// 		return "settings.json"
// 	}
// 	return filepath.Join(filepath.Dir(ex), "settings.json")
// }

// GetSettings вызывается фронтендом при старте
func (a *App) GetSettings() uploader.ResizeSettings {

	s := a.db.GetSettings()

	return uploader.ResizeSettings{
		Resize:      s.Resize,
		ResizeTo:    s.ResizeTo,
		WebpQuality: s.WebpQuality,
	}
}

// SaveSettings вызывается фронтендом при любом изменении
func (a *App) SaveSettings(s uploader.ResizeSettings) {

	a.db.UpdateSettings(database.Settings{
		Resize:      s.Resize,
		ResizeTo:    s.ResizeTo,
		WebpQuality: s.WebpQuality,
	})
}

func (a *App) GetHistory(limit int, offset int) []database.HistoryItem {
	return a.db.GetHistory(limit, offset)
}

func (a *App) ClearHistory() {
	a.db.ClearHistory()
}
