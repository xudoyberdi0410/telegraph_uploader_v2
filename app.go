package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"telegraph_uploader_v2/internal/config"
	"telegraph_uploader_v2/internal/database"
	"telegraph_uploader_v2/internal/repository"
	"telegraph_uploader_v2/internal/service"
	"telegraph_uploader_v2/internal/telegram"
	"telegraph_uploader_v2/internal/telegraph"
	"telegraph_uploader_v2/internal/uploader"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	config       *config.Config
	
	// Services
	mangaService *service.MangaService
	pubService   *service.PublicationService
	
	// Infrastructure
	r2Uploader *uploader.R2Uploader

	// Repositories (Direct access for simple CRUD)
	settingsRepo repository.SettingsRepository
	historyRepo  repository.HistoryRepository
	titleRepo    repository.TitleRepository
	templateRepo repository.TemplateRepository
	
	// Infrastructure (Direct access where Service isn't needed/created yet)
	telegram     *telegram.Client
	
	dialogs      DialogProvider

	authPasswordChan chan string
	authHandler      *WailsAuthHandler
}

func NewApp() *App {
	log.Println("[App] Starting initialization...")

	cfg, err := config.Load()
	if err != nil {
		log.Println("[App] Warning: could not load config:", err)
		cfg = &config.Config{}
	} else {
		log.Println("[App] Config loaded successfully")
	}

	// 1. Init Database
	dbInstance, err := database.Init()
	if err != nil {
		log.Fatal("[App] Database init error:", err)
	}
	log.Println("[App] Database initialized")

	// 2. Init Repositories
	settingsRepo := repository.NewSettingsRepository(dbInstance)
	historyRepo := repository.NewHistoryRepository(dbInstance)
	titleRepo := repository.NewTitleRepository(dbInstance)
	templateRepo := repository.NewTemplateRepository(dbInstance)
	cacheRepo := repository.NewImageCacheRepository(dbInstance)

	// 3. Init Infrastructure Clients
	r2Uploader, err := uploader.New(cfg, cacheRepo)
	if err != nil {
		log.Println("[App] Uploader init error:", err)
	} else {
		log.Println("[App] Uploader initialized")
	}

	tgClient := telegraph.New(cfg)
	log.Println("[App] Telegraph client initialized")

	tgApp, err := telegram.New(cfg)
	if err != nil {
		log.Println("[App] Telegram client init error:", err)
	} else {
		log.Println("[App] Telegram client initialized")
	}

	// 4. Init Services
	mangaService := service.NewMangaService(r2Uploader)
	pubService := service.NewPublicationService(tgClient, tgApp, historyRepo, titleRepo)

	pwdChan := make(chan string)

	return &App{
		config:           cfg,
		mangaService:     mangaService,
		pubService:       pubService,
		r2Uploader:       r2Uploader,
		settingsRepo:     settingsRepo,
		historyRepo:      historyRepo,
		titleRepo:        titleRepo,
		templateRepo:     templateRepo,
		telegram:         tgApp,
		dialogs:          &WailsDialogProvider{},
		authPasswordChan: pwdChan,
		authHandler:      &WailsAuthHandler{passwordChan: pwdChan},
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.authHandler.SetContext(ctx)
	// Wire the channels: App writes to its channel, AuthHandler reads from it
	// Actually, let's share the channel instance or link them.
	// In NewApp I created a new channel for authHandler, but App uses authPasswordChan.
	// Let's fix NewApp to share the same channel.
	log.Println("[App] Application startup complete")

	if a.telegram != nil {
		a.telegram.Start(ctx)
	}
}

// === МЕТОДЫ ===

func (a *App) UploadChapter(filePaths []string, resizeSettings uploader.ResizeSettings) uploader.UploadResult {
	log.Printf("[App] UploadChapter called. Files: %d, Settings: %+v", len(filePaths), resizeSettings)
	
	onProgress := func(current, total int) {
		percentage := int(float64(current) / float64(total) * 100)
		wailsRuntime.EventsEmit(a.ctx, "upload_progress", map[string]int{
			"current":    current,
			"total":      total,
			"percentage": percentage,
		})
	}

	// Делегируем сервису
	result := a.mangaService.UploadChapter(a.ctx, filePaths, resizeSettings, onProgress)

	if result.Success {
		log.Printf("[App] UploadChapter finished successfully. URLs generated: %d", len(result.Links))
	} else {
		log.Printf("[App] UploadChapter failed. Error: %s", result.Error)
	}

	return result
}

func (a *App) ListFiles() ([]uploader.RemoteFile, error) {
	if a.r2Uploader == nil {
		return nil, fmt.Errorf("uploader service not available")
	}
	return a.r2Uploader.ListAllFiles(a.ctx)
}

func (a *App) DeleteFiles(filenames []string) error {
	if a.r2Uploader == nil {
		return fmt.Errorf("uploader service not available")
	}
	return a.r2Uploader.DeleteFiles(a.ctx, filenames)
}

func (a *App) CreateTelegraphPage(title string, imageUrls []string, titleID int) CreatePageResponse {
	log.Printf("[App] CreateTelegraphPage called. Title: '%s', Images: %d, TitleID: %d", title, len(imageUrls), titleID)

	res, err := a.pubService.CreatePage(title, imageUrls, titleID)
	
	if err != nil {
		log.Printf("[App] Failed to create page: %v", err)
		return CreatePageResponse{
			Success:   false,
			Error:     err.Error(),
		}
	}

	log.Printf("[App] Page created successfully: %s", res.URL)
	return CreatePageResponse{
		Success:   true,
		Url:       res.URL,
		HistoryID: res.HistoryID,
	}
}

func (a *App) EditTelegraphPage(path string, title string, imageUrls []string, token string) string {
	log.Printf("[App] EditTelegraphPage called. Path: '%s', Title: '%s', Images: %d", path, title, len(imageUrls))

	url := a.pubService.EditPage(path, title, imageUrls, token)

	if len(url) > 4 && url[:4] == "http" {
		log.Printf("[App] Page edited successfully: %s", url)
	} else {
		log.Printf("[App] Failed to edit page. Result URL/Error: %s", url)
	}

	return url
}

func (a *App) GetTelegraphPage(pageUrl string) (ChapterResponse, error) {
	log.Printf("[App] GetTelegraphPage called. URL: %s", pageUrl)

	title, images, err := a.pubService.GetPage(pageUrl)
	if err != nil {
		log.Printf("[App] Error getting page: %v", err)
		return ChapterResponse{}, err
	}

	log.Printf("[App] Got page '%s' with %d images", title, len(images))

	return ChapterResponse{
		Path:       pageUrl,
		Title:      title,
		Images:     images,
		ImageCount: len(images),
	}, nil
}

// Диалог выбора папки
func (a *App) OpenFolderDialog() (ChapterResponse, error) {
	log.Println("[App] OpenFolderDialog called")

	selection, err := a.dialogs.OpenDirectory(a.ctx, wailsRuntime.OpenDialogOptions{
		Title: "Выберите папку с главой",
	})
	if err != nil {
		log.Printf("[App] OpenDirectoryDialog error: %v", err)
		return ChapterResponse{}, err
	}
	if selection == "" {
		log.Println("[App] OpenFolderDialog canceled by user")
		return ChapterResponse{}, nil
	}

	log.Printf("[App] Folder selected: %s", selection)

	images, err := getImagesInDir(selection)
	if err != nil {
		log.Printf("[App] Error reading directory: %v", err)
		return ChapterResponse{}, err
	}

	title := filepath.Base(selection)
	log.Printf("[App] Found %d images in folder '%s'", len(images), title)

	var detectedID *uint
	// Используем репозиторий вместо прямого вызова БД
	dbTitle, err := a.titleRepo.FindByPath(selection)
	if err == nil {
		detectedID = &dbTitle.ID
		log.Printf("[App] Detected title: %s (ID: %d)", dbTitle.Name, dbTitle.ID)
	}

	return ChapterResponse{
		Path:            selection,
		Title:           title,
		Images:          images,
		ImageCount:      len(images),
		DetectedTitleID: detectedID,
	}, nil
}

// Выбор отдельных файлов
func (a *App) OpenFilesDialog() ([]string, error) {
	log.Println("[App] OpenFilesDialog called")

	selection, err := a.dialogs.OpenMultipleFiles(a.ctx, wailsRuntime.OpenDialogOptions{
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
func (a *App) GetSettings() FrontendSettings {
	log.Println("[App] GetSettings called")
	s, err := a.settingsRepo.Get()
	if err != nil {
		log.Printf("[App] Error getting settings: %v", err)
		return FrontendSettings{} // Or default
	}
	
	log.Printf("[App] Returning settings: %+v", s)

	return FrontendSettings{
		Resize:           s.Resize,
		ResizeTo:         s.ResizeTo,
		WebpQuality:      s.WebpQuality,
		LastChannelID:    strconv.FormatInt(s.LastChannelID, 10),
		LastChannelHash:  strconv.FormatInt(s.LastChannelHash, 10),
		LastChannelTitle: s.LastChannelTitle,
	}
}

// SaveSettings вызывается фронтендом при любом изменении
func (a *App) SaveSettings(s FrontendSettings) {
	log.Printf("[App] SaveSettings called: %+v", s)

	cID, _ := strconv.ParseInt(s.LastChannelID, 10, 64)
	cHash, _ := strconv.ParseInt(s.LastChannelHash, 10, 64)

	err := a.settingsRepo.Update(database.Settings{
		Resize:           s.Resize,
		ResizeTo:         s.ResizeTo,
		WebpQuality:      s.WebpQuality,
		LastChannelID:    cID,
		LastChannelHash:  cHash,
		LastChannelTitle: s.LastChannelTitle,
	})
	
	if err != nil {
		log.Printf("[App] Error saving settings: %v", err)
	} else {
		log.Println("[App] Settings saved")
	}
}

func (a *App) GetHistory(limit int, offset int) []database.HistoryItem {
	log.Printf("[App] GetHistory called (limit: %d, offset: %d)", limit, offset)
	items, err := a.historyRepo.Get(limit, offset)
	if err != nil {
		log.Printf("[App] Error getting history: %v", err)
		return []database.HistoryItem{}
	}
	log.Printf("[App] Returned %d history items", len(items))
	return items
}

func (a *App) ClearHistory() {
	log.Println("[App] ClearHistory called")
	a.historyRepo.Clear()
	log.Println("[App] History cleared")
}

func (a *App) TelegramLoginQR() string {
	log.Println("[App] Starting Telegram QR login...")

	go func() {
		displayQR := func(qrImage []byte) {
			base64Image := base64.StdEncoding.EncodeToString(qrImage)
			log.Printf("[App] Emitting QR code, size: %d bytes", len(qrImage))
			wailsRuntime.EventsEmit(a.ctx, "tg_qr_code", base64Image)
		}

		err := a.telegram.LoginQR(a.ctx, displayQR, a.authHandler)
		if err != nil {
			log.Printf("[App] QR Login failed: %v", err)
			wailsRuntime.EventsEmit(a.ctx, "tg_auth_error", err.Error())
		} else {
			log.Println("[App] QR Login success!")
			wailsRuntime.EventsEmit(a.ctx, "tg_auth_success", true)
		}
	}()

	return "QR Process started"
}

func (a *App) AddTitleVariable(titleID uint, key string, value string) error {
	return a.titleRepo.AddVariable(titleID, key, value)
}

func (a *App) TelegramSubmitPassword(password string) {
	log.Printf("[App] User submitted password")
	a.authPasswordChan <- password
}

// WailsAuthHandler handles authentication flow interactions via Wails
type WailsAuthHandler struct {
	ctx          context.Context
	passwordChan <-chan string
}

func (h *WailsAuthHandler) SetContext(ctx context.Context) {
	h.ctx = ctx
}

func (h *WailsAuthHandler) GetPassword(ctx context.Context) (string, error) {
	log.Println("[WailsAuthHandler] Requesting password from user via UI event...")
	
	if h.ctx == nil {
		return "", fmt.Errorf("context not set for auth handler")
	}

	wailsRuntime.EventsEmit(h.ctx, "tg_request_password", nil)

	select {
	case pwd := <-h.passwordChan:
		return pwd, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// --- Title Management ---

func (a *App) GetTitles() []database.Title {
	t, _ := a.titleRepo.GetAll()
	return t
}

func (a *App) CreateTitle(name string, rootFolder string) error {
	return a.titleRepo.Create(name, rootFolder)
}

func (a *App) UpdateTitle(t database.Title) error {
	return a.titleRepo.Update(t)
}

func (a *App) DeleteTitle(id uint) error {
	return a.titleRepo.Delete(id)
}

func (a *App) GetTitleByID(id uint) (database.Title, error) {
	return a.titleRepo.GetByID(id)
}

// --- Template Management ---

func (a *App) GetTemplates() []database.Template {
	t, _ := a.templateRepo.GetAll()
	return t
}

func (a *App) CreateTemplate(name, content string) error {
	return a.templateRepo.Create(name, content)
}

func (a *App) UpdateTemplate(t database.Template) error {
	return a.templateRepo.Update(t)
}

func (a *App) DeleteTemplate(id uint) error {
	return a.templateRepo.Delete(id)
}

// --- Telegram Feature ---

func (a *App) SearchChannels(query string) ([]TelegramChannel, error) {
	if a.telegram == nil {
		return nil, os.ErrInvalid
	}
	channels, err := a.telegram.SearchAdminChannels(a.ctx, query)
	if err != nil {
		return nil, err
	}

	var result []TelegramChannel
	for _, ch := range channels {
		result = append(result, TelegramChannel{
			ID:         strconv.FormatInt(ch.ID, 10),
			Title:      ch.Title,
			AccessHash: strconv.FormatInt(ch.AccessHash, 10),
		})
	}
	return result, nil
}

func (a *App) PublishPost(historyID uint, channelIDStr string, accessHashStr string, content string, dateStr string) error {
	log.Printf("[App] PublishPost called. History: %d, Channel: %s", historyID, channelIDStr)

	// Parse IDs
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid channel id: %v", err)
	}
	accessHash, err := strconv.ParseInt(accessHashStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid access hash: %v", err)
	}

	// Parse Date
	scheduledTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		log.Printf("[App] Date parse error: %v", err)
		return err
	}

	// Delegate to Service
	return a.pubService.PublishPost(a.ctx, historyID, channelID, accessHash, content, scheduledTime)
}

func (a *App) IsTelegramLoggedIn() bool {
	if a.telegram == nil {
		return false
	}
	auth, err := a.telegram.CheckAuth(a.ctx)
	if err != nil {
		log.Printf("[App] CheckAuth error: %v", err)
		return false
	}
	return auth
}

func (a *App) GetTelegramUser() (*telegram.TelegramUser, error) {
	if a.telegram == nil {
		return nil, fmt.Errorf("telegram client not initialized")
	}
	return a.telegram.GetMe(a.ctx)
}