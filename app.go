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
	"telegraph_uploader_v2/internal/telegram"
	"telegraph_uploader_v2/internal/telegraph"
	"telegraph_uploader_v2/internal/uploader"

	"github.com/gotd/td/tg"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// DialogProvider interface mocks Wails dialogs
type DialogProvider interface {
	OpenDirectory(ctx context.Context, options wailsRuntime.OpenDialogOptions) (string, error)
	OpenMultipleFiles(ctx context.Context, options wailsRuntime.OpenDialogOptions) ([]string, error)
}

// WailsDialogProvider implements DialogProvider using real Wails runtime
type WailsDialogProvider struct{}

func (w *WailsDialogProvider) OpenDirectory(ctx context.Context, options wailsRuntime.OpenDialogOptions) (string, error) {
	return wailsRuntime.OpenDirectoryDialog(ctx, options)
}

func (w *WailsDialogProvider) OpenMultipleFiles(ctx context.Context, options wailsRuntime.OpenDialogOptions) ([]string, error) {
	return wailsRuntime.OpenMultipleFilesDialog(ctx, options)
}

type App struct {
	ctx        context.Context
	config     *config.Config
	uploader   *uploader.R2Uploader
	tgphClient *telegraph.Client
	telegram   *telegram.Client
	db         *database.Database
	dialogs    DialogProvider

	authCodeChan     chan string
	authPasswordChan chan string // Added channel for password
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

	tgApp, err := telegram.New(cfg)
	if err != nil {
		log.Println("[App] Telegram client init error:", err)
	} else {
		log.Println("[App] Telegram client initialized")
	}

	return &App{
		config:           cfg,
		uploader:         upl,
		tgphClient:       tg,
		db:               dbService,
		dialogs:          &WailsDialogProvider{},
		telegram:         tgApp,
		authCodeChan:     make(chan string),
		authPasswordChan: make(chan string), // Initialize channel
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("[App] Application startup complete")

	if a.telegram != nil {
		a.telegram.Start(ctx)
	}
}

// === СТРУКТУРЫ ===
type ChapterResponse struct {
	Path            string   `json:"path"`
	Title           string   `json:"title"`
	Images          []string `json:"images"`
	ImageCount      int      `json:"imageCount"`
	DetectedTitleID *uint    `json:"detected_title_id"`
}

type CreatePageResponse struct {
	Success   bool   `json:"success"`
	Url       string `json:"url"`
	HistoryID uint   `json:"history_id"`
	Error     string `json:"error"`
}

type TelegramChannel struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AccessHash string `json:"access_hash"`
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

func (a *App) CreateTelegraphPage(title string, imageUrls []string, titleID int) CreatePageResponse {
	log.Printf("[App] CreateTelegraphPage called. Title: '%s', Images: %d, TitleID: %d", title, len(imageUrls), titleID)

	url := a.tgphClient.CreatePage(title, imageUrls)

	if len(url) > 4 && url[:4] == "http" {
		log.Printf("[App] Page created successfully: %s", url)
		var tID *uint
		if titleID > 0 {
			u := uint(titleID)
			tID = &u
		}
		id, err := a.db.AddHistory(title, url, len(imageUrls), a.config.TelegraphToken, tID)
		if err != nil {
			log.Printf("[App] Warning: Failed to save to history: %v", err)
			return CreatePageResponse{
				Success:   true,
				Url:       url,
				HistoryID: 0,
				Error:     "Saved to Telegraph but failed to save history: " + err.Error(),
			}
		} else {
			log.Println("[App] Saved to history")
			return CreatePageResponse{
				Success:   true,
				Url:       url,
				HistoryID: id,
			}
		}
	} else {
		log.Printf("[App] Failed to create page. Result URL/Error: %s", url)
		return CreatePageResponse{
			Success: false,
			Error:   url,
		}
	}
}

func (a *App) EditTelegraphPage(path string, title string, imageUrls []string, token string) string {
	log.Printf("[App] EditTelegraphPage called. Path: '%s', Title: '%s', Images: %d", path, title, len(imageUrls))

	url := a.tgphClient.EditPage(path, title, imageUrls, token)

	if len(url) > 4 && url[:4] == "http" {
		log.Printf("[App] Page edited successfully: %s", url)
	} else {
		log.Printf("[App] Failed to edit page. Result URL/Error: %s", url)
	}

	return url
}

func (a *App) GetTelegraphPage(pageUrl string) (ChapterResponse, error) {
	log.Printf("[App] GetTelegraphPage called. URL: %s", pageUrl)

	// Извлекаем slug из URL
	parts := strings.Split(pageUrl, "/")
	// Split always returns at least one element
	path := parts[len(parts)-1]

	title, images, err := a.tgphClient.GetPage(path)
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

	var detectedID *uint
	dbTitle, err := a.db.FindTitleByPath(selection)
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

type FrontendSettings struct {
	Resize           bool   `json:"resize"`
	ResizeTo         int    `json:"resize_to"`
	WebpQuality      int    `json:"webp_quality"`
	LastChannelID    string `json:"last_channel_id"`
	LastChannelHash  string `json:"last_channel_hash"`
	LastChannelTitle string `json:"last_channel_title"`
}

// GetSettings вызывается фронтендом при старте
func (a *App) GetSettings() FrontendSettings {
	log.Println("[App] GetSettings called")
	s := a.db.GetSettings()
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

	// Parse IDs
	cID, _ := strconv.ParseInt(s.LastChannelID, 10, 64)
	cHash, _ := strconv.ParseInt(s.LastChannelHash, 10, 64)

	a.db.UpdateSettings(database.Settings{
		Resize:           s.Resize,
		ResizeTo:         s.ResizeTo,
		WebpQuality:      s.WebpQuality,
		LastChannelID:    cID,
		LastChannelHash:  cHash,
		LastChannelTitle: s.LastChannelTitle,
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

func (a *App) TelegramLogin(phone string) string {
	log.Printf("[App] Starting Telegram login for %s", phone)

	go func() {
		// 'a' satisfies AuthFlowHandler because it implements GetCode and GetPassword
		err := a.telegram.Login(a.ctx, phone, a)
		if err != nil {
			log.Printf("[App] Login failed: %v", err)
			wailsRuntime.EventsEmit(a.ctx, "tg_auth_error", err.Error())
		} else {
			log.Println("[App] Login success!")
			wailsRuntime.EventsEmit(a.ctx, "tg_auth_success", true)
		}
	}()

	return "Process started"
}

func (a *App) TelegramLoginQR() string {
	log.Println("[App] Starting Telegram QR login...")

	go func() {
		// Callback to send QR code image to frontend
		displayQR := func(qrImage []byte) {
			// Convert to base64
			base64Image := base64.StdEncoding.EncodeToString(qrImage)
			log.Printf("[App] Emitting QR code, size: %d bytes", len(qrImage))
			wailsRuntime.EventsEmit(a.ctx, "tg_qr_code", base64Image)
		}

		// Pass 'a' as the AuthFlowHandler
		err := a.telegram.LoginQR(a.ctx, displayQR, a)
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

// TelegramSubmitCode вызывается из UI, когда пользователь ввел код
func (a *App) TelegramSubmitCode(code string) {
	log.Printf("[App] User submitted code: %s", code)
	if a.authCodeChan != nil {
		a.authCodeChan <- code
	}
}

func (a *App) AddTitleVariable(titleID uint, key string, value string) error {
	return a.db.AddTitleVariable(titleID, key, value)
}

// TelegramSubmitPassword вызывается из UI, когда пользователь ввел пароль (2FA)
func (a *App) TelegramSubmitPassword(password string) {
	log.Printf("[App] User submitted password") // Don't log the actual password
	a.authPasswordChan <- password
}

// GetCode is called by the Telegram Client
func (a *App) GetCode(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	// 1. Determine where the code was sent
	typeStr := "unknown"
	var length int

	switch t := sentCode.Type.(type) {
	case *tg.AuthSentCodeTypeApp:
		typeStr = "app"
		length = t.Length
	case *tg.AuthSentCodeTypeSMS:
		typeStr = "sms"
		length = t.Length
	case *tg.AuthSentCodeTypeCall:
		typeStr = "call"
		length = t.Length
	case *tg.AuthSentCodeTypeFlashCall:
		typeStr = "flash_call"
	}

	nextTypeStr := "none"
	if sentCode.NextType != nil {
		switch sentCode.NextType.(type) {
		case *tg.AuthCodeTypeSMS:
			nextTypeStr = "sms"
		case *tg.AuthCodeTypeCall:
			nextTypeStr = "call"
		case *tg.AuthCodeTypeFlashCall:
			nextTypeStr = "flash_call"
		}
	}

	log.Printf("[App] Requesting code from user. Code sent via: %s, length: %d, timeout: %d, next_type: %s", typeStr, length, sentCode.Timeout, nextTypeStr)

	// 2. Сообщаем UI, что нам нужен код, и передаем тип
	wailsRuntime.EventsEmit(a.ctx, "tg_request_code", map[string]interface{}{
		"type":      typeStr,
		"length":    length,
		"timeout":   sentCode.Timeout,
		"next_type": nextTypeStr,
	})

	// 3. Ждем ответа из канала (который заполнит метод TelegramSubmitCode)
	select {
	case code := <-a.authCodeChan:
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// GetPassword is called by the Telegram Client (if 2FA is enabled)
func (a *App) GetPassword(ctx context.Context) (string, error) {
	// 1. Сообщаем UI, что нам нужен пароль
	log.Println("[App] Requesting password from user via UI event...")
	wailsRuntime.EventsEmit(a.ctx, "tg_request_password", nil)

	// 2. Ждем ответа из канала
	select {
	case pwd := <-a.authPasswordChan:
		return pwd, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

// --- Title Management ---

func (a *App) GetTitles() []database.Title {
	return a.db.GetTitles()
}

func (a *App) CreateTitle(name string, rootFolder string) error {
	return a.db.CreateTitle(name, rootFolder)
}

func (a *App) UpdateTitle(t database.Title) error {
	return a.db.UpdateTitle(t)
}

func (a *App) DeleteTitle(id uint) error {
	return a.db.DeleteTitle(id)
}

func (a *App) GetTitleByID(id uint) (database.Title, error) {
	return a.db.GetTitleByID(id)
}

// --- Template Management ---

func (a *App) GetTemplates() []database.Template {
	return a.db.GetTemplates()
}

func (a *App) CreateTemplate(name, content string) error {
	return a.db.CreateTemplate(name, content)
}

func (a *App) UpdateTemplate(t database.Template) error {
	return a.db.UpdateTemplate(t)
}

func (a *App) DeleteTemplate(id uint) error {
	return a.db.DeleteTemplate(id)
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
	log.Printf("[App] PublishPost called. History: %d, Channel: %s, Content-Len: %d, Date: %s", historyID, channelIDStr, len(content), dateStr)

	// Parse IDs
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid channel id: %v", err)
	}
	accessHash, err := strconv.ParseInt(accessHashStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid access hash: %v", err)
	}

	// 1. Get History Item
	item, err := a.db.GetHistoryByID(historyID)
	if err != nil {
		return err
	}

	// 2. Prepare Content (content is now passed directly)
	content = strings.ReplaceAll(content, "{{Title}}", item.Title)
	content = strings.ReplaceAll(content, "{{Link}}", item.Url)

	// Custom Variables
	if item.TitleID != nil && *item.TitleID > 0 {
		title, err := a.db.GetTitleByID(*item.TitleID)
		if err == nil {
			for _, v := range title.Variables {
				if v.Key != "" {
					content = strings.ReplaceAll(content, "{{"+v.Key+"}}", v.Value)
				}
			}
		}
	}

	// 3. Parse Date (Assume RFC3339/ISO8601 from JS)
	// JS: 2024-01-11T12:00:00.000Z
	scheduledTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		// Try without milliseconds if helpful, or just return error
		log.Printf("[App] Date parse error: %v", err)
		return err
	}

	// 4. Schedule
	err = a.telegram.ScheduleMessageByID(a.ctx, channelID, accessHash, content, scheduledTime)
	if err != nil {
		log.Printf("[App] Schedule error: %v", err)
		return err
	}

	log.Println("[App] Post scheduled successfully")
	return nil
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
