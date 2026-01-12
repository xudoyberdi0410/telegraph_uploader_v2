package main

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"telegraph_uploader_v2/internal/config"
	"telegraph_uploader_v2/internal/database"
	"telegraph_uploader_v2/internal/repository"
	"telegraph_uploader_v2/internal/service"
	"telegraph_uploader_v2/internal/telegram"
	"telegraph_uploader_v2/internal/telegraph"
	"telegraph_uploader_v2/internal/uploader"

	"github.com/glebarez/sqlite"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// MockDialogProvider
type MockDialogProvider struct {
	DirSelection  string
	FileSelection []string
	Err           error
}

func (m *MockDialogProvider) OpenDirectory(ctx context.Context, options wailsRuntime.OpenDialogOptions) (string, error) {
	return m.DirSelection, m.Err
}

func (m *MockDialogProvider) OpenMultipleFiles(ctx context.Context, options wailsRuntime.OpenDialogOptions) ([]string, error) {
	return m.FileSelection, m.Err
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Migrate
	err = db.AutoMigrate(&database.Settings{}, &database.HistoryEntry{}, &database.Title{}, &database.TitleFolder{}, &database.TitleVariable{}, &database.Template{})
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	// Create default settings if not exists
	var count int64
	db.Model(&database.Settings{}).Count(&count)
	if count == 0 {
		db.Create(&database.Settings{
			Resize:      false,
			ResizeTo:    1600,
			WebpQuality: 80,
		})
	}

	return db
}

func setupTestApp(t *testing.T) (*App, *httptest.Server, *httptest.Server) {
	tsTelegraph := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/createPage" {
			w.Write([]byte(`{"ok": true, "result": {"url": "http://telegra.ph/test"}}`))
			return
		}
		if r.URL.Path == "/editPage" {
			w.Write([]byte(`{"ok": true, "result": {"url": "http://telegra.ph/edited"}}`))
			return
		}
		if r.URL.Path == "/getPage/slug" {
			w.Write([]byte(`{"ok": true, "result": {"title": "T", "content": []}}`))
			return
		}
		w.Write([]byte(`{"ok": false, "error": "unknown"}`))
	}))

	tsS3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		if r.Method == "PUT" {
			w.Header().Set("ETag", "\"mock-etag\"")
		}
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>buck</Name><Prefix></Prefix><Marker></Marker><MaxKeys>1000</MaxKeys><IsTruncated>false</IsTruncated><Contents></Contents></ListBucketResult>`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	cfg := &config.Config{
		TelegraphToken: "test_token",
		R2AccountId:    "acc",
		R2AccessKey:    "key",
		R2SecretKey:    "secret",
		BucketName:     "buck",
		PublicDomain:   "http://dom.com",
	}

	db := setupTestDB(t)
	
	// Repos
	settingsRepo := repository.NewSettingsRepository(db)
	historyRepo := repository.NewHistoryRepository(db)
	titleRepo := repository.NewTitleRepository(db)
	templateRepo := repository.NewTemplateRepository(db)

	// Clients
	tgClient := telegraph.New(cfg)
	tgClient.BaseURL = tsTelegraph.URL

	minioClient, _ := minio.New(tsS3.Listener.Addr().String(), &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Secure: false,
	})
	upl := uploader.NewWithClient(minioClient, cfg)
	
	// Services
	mangaService := service.NewMangaService(upl)
	
	// Mock Telegram Client (nil for now as it's hard to mock without interface, but we can pass nil if methods check it)
	// Or create a real one if needed. PublishPost needs it.
	// For now we pass nil to PublicationService and hope tests that need it are skipped or mocked differently.
	// But PublishPost uses it.
	// The original tests didn't test PublishPost with real telegram client, they mocked `telegram` package?
	// No, the original test file calls `app.PublishPost` which used `app.telegram`.
	// `app.telegram` was `*telegram.Client`.
	// In `setupTestApp` original, `telegram` field was NOT set.
	// So `PublishPost` tests relied on `app.telegram` being nil?
	// `PublishPost` checks `if a.telegram == nil`.
	// Wait, original `PublishPost` in `app.go`:
	// `func (a *App) PublishPost(...)`
	// It did NOT check if `telegram` is nil before parsing args.
	// But `a.telegram.ScheduleMessageByID` would panic if `telegram` is nil.
	// The original `app_test.go` has `TestApp_PublishPost_Errors`. It tests validation errors (channel ID, hash, history, date). It does NOT test success or schedule error.
	// So we don't need a real telegram client for those tests.
	
	// But `PublicationService.PublishPost` MIGHT crash if telegram client is nil?
	// `s.telegram.ScheduleMessageByID`.
	// We need to pass a telegram client that doesn't crash, or ensure we only test failure paths that return before calling it.
	// The tests 1-4 in `TestApp_PublishPost_Errors` fail BEFORE calling `telegram`.
	// So passing nil is fine for now.
	var tgApp *telegram.Client // nil

	pubService := service.NewPublicationService(tgClient, tgApp, historyRepo, titleRepo)

	app := &App{
		ctx:          context.Background(),
		config:       cfg,
		mangaService: mangaService,
		pubService:   pubService,
		settingsRepo: settingsRepo,
		historyRepo:  historyRepo,
		titleRepo:    titleRepo,
		templateRepo: templateRepo,
		dialogs:      &MockDialogProvider{},
		authPasswordChan: make(chan string),
	}

	return app, tsTelegraph, tsS3
}

func TestApp_OpenFolderDialog(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	// Setup temp dir with images
	tmpDir := t.TempDir()
	f1, _ := os.Create(filepath.Join(tmpDir, "img1.jpg"))
	f1.Close()

	// Configure mock
	app.dialogs = &MockDialogProvider{DirSelection: tmpDir}

	resp, err := app.OpenFolderDialog()
	if err != nil {
		t.Fatalf("OpenFolderDialog failed: %v", err)
	}

	if resp.Path != tmpDir {
		t.Errorf("expected path %s, got %s", tmpDir, resp.Path)
	}
	if resp.ImageCount != 1 {
		t.Errorf("expected 1 image, got %d", resp.ImageCount)
	}
}

func TestApp_OpenFolderDialog_Cancel(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	app.dialogs = &MockDialogProvider{DirSelection: ""} // User canceled

	resp, err := app.OpenFolderDialog()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Path != "" {
		t.Error("expected empty result on cancel")
	}
}

func TestApp_OpenFolderDialog_Errors(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	// Dialog Error
	app.dialogs = &MockDialogProvider{Err: fmt.Errorf("dialog error")}
	_, err := app.OpenFolderDialog()
	if err == nil {
		t.Error("expected error from dialog")
	}

	// Dir Scan Error (non-existent dir)
	app.dialogs = &MockDialogProvider{DirSelection: "nonexistent_dir_12345"}
	_, err = app.OpenFolderDialog()
	if err == nil {
		t.Error("expected error scanning bad dir")
	}
}

func TestApp_OpenFilesDialog_Errors(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	app.dialogs = &MockDialogProvider{Err: fmt.Errorf("dialog error")}
	_, err := app.OpenFilesDialog()
	if err == nil {
		t.Error("expected error from dialog")
	}

	app.dialogs = &MockDialogProvider{FileSelection: nil}
	_, err = app.OpenFilesDialog()
	if err != nil {
		t.Errorf("unexpected error for cancel/empty: %v", err)
	}
}

func TestApp_UploadChapter(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	f, _ := os.Create(tmpFile.Name())
	png.Encode(f, img)
	f.Close()

	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	res := app.UploadChapter([]string{tmpFile.Name()}, uploader.ResizeSettings{})
	if !res.Success {
		t.Errorf("expected success, got %v", res.Error)
	}

	// Test nil uploader - Create App with nil MangaService or nil uploader inside it?
	// MangaService checks if its uploader is nil.
	ms := service.NewMangaService(nil)
	appNil := &App{mangaService: ms}
	res = appNil.UploadChapter([]string{"f"}, uploader.ResizeSettings{})
	if res.Success || res.Error != "Загрузчик не инициализирован" {
		t.Error("expected error for nil uploader")
	}
}

func TestApp_UploadChapter_Error(t *testing.T) {
	// Setup failing S3
	tsS3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer tsS3.Close()

	cfg := &config.Config{
		R2AccountId:    "acc",
		R2AccessKey:    "key",
		R2SecretKey:    "secret",
		BucketName:     "buck",
		PublicDomain:   "http://dom.com",
	}

	minioClient, _ := minio.New(tsS3.Listener.Addr().String(), &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Secure: false,
	})
	upl := uploader.NewWithClient(minioClient, cfg)
	
	// Manually wire app
	ms := service.NewMangaService(upl)
	app := &App{mangaService: ms, ctx: context.Background()}

	tmpFile, err := os.CreateTemp("", "test*.png")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	
	// Create actual image content
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	f, _ := os.Create(tmpFile.Name())
	png.Encode(f, img)
	f.Close()

	res := app.UploadChapter([]string{tmpFile.Name()}, uploader.ResizeSettings{})
	if res.Success {
		t.Error("expected failure when S3 errors")
	}
}

func TestApp_CreateTelegraphPage(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	resp := app.CreateTelegraphPage("Title", []string{"http://img.jpg"}, 0)
	if !resp.Success {
		t.Errorf("expected success, got error %s", resp.Error)
	}
	if resp.Url != "http://telegra.ph/test" {
		t.Errorf("expected url http://telegra.ph/test, got %s", resp.Url)
	}

	// Test failure
	tsFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok": false, "error": "FAIL"}`))
	}))
	defer tsFail.Close()

	// Recreate pubService with failing client
	cfg := &config.Config{TelegraphToken: "t"}
	tgClient := telegraph.New(cfg)
	tgClient.BaseURL = tsFail.URL
	
	app.pubService = service.NewPublicationService(tgClient, nil, app.historyRepo, app.titleRepo)

	resp = app.CreateTelegraphPage("Title", nil, 0)
	// It should log failure and return error string
	if resp.Success {
		t.Error("expected failure")
	}
}

func TestApp_EditTelegraphPage(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	url := app.EditTelegraphPage("path", "Title", []string{"http://img.jpg"}, "token")
	if url != "http://telegra.ph/edited" {
		t.Errorf("expected url, got %s", url)
	}

	// Test failure
	tsFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok": false, "error": "FAIL"}`))
	}))
	defer tsFail.Close()
	
	cfg := &config.Config{TelegraphToken: "t"}
	tgClient := telegraph.New(cfg)
	tgClient.BaseURL = tsFail.URL
	app.pubService = service.NewPublicationService(tgClient, nil, app.historyRepo, app.titleRepo)

	url = app.EditTelegraphPage("path", "Title", nil, "")
	if url[:4] == "http" {
		t.Error("expected error string")
	}
}

func TestApp_GetTelegraphPage(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	resp, err := app.GetTelegraphPage("http://telegra.ph/slug")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resp.Title != "T" {
		t.Errorf("expected title T, got %s", resp.Title)
	}

	// GetPage error
	tsFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tsFail.Close()
	
	cfg := &config.Config{TelegraphToken: "t"}
	tgClient := telegraph.New(cfg)
	tgClient.BaseURL = tsFail.URL
	app.pubService = service.NewPublicationService(tgClient, nil, app.historyRepo, app.titleRepo)

	_, err = app.GetTelegraphPage("http://t.ph/bad")
	if err == nil {
		t.Error("expected error for bad page")
	}
}

func TestApp_Startup(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	app.startup(context.Background())
	if app.ctx == nil {
		t.Error("startup did not set context")
	}
}

func TestApp_Settings(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	s := app.GetSettings()
	if s.ResizeTo != 1600 {
		t.Errorf("expected default 1600, got %d", s.ResizeTo)
	}

	// Use FrontendSettings
	newSettings := FrontendSettings{
		Resize:           true,
		ResizeTo:         2000,
		WebpQuality:      80,
		LastChannelID:    "1",
		LastChannelHash:  "2",
		LastChannelTitle: "C",
	}

	app.SaveSettings(newSettings)
	s = app.GetSettings()
	if s.ResizeTo != 2000 {
		t.Errorf("expected 2000, got %d", s.ResizeTo)
	}
	if s.LastChannelID != "1" {
		t.Errorf("expected 1, got %s", s.LastChannelID)
	}
}

func TestApp_History(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	app.historyRepo.Add("T1", "u1", 1, "tok", nil)

	app.ClearHistory()

	app.historyRepo.Add("T1", "u1", 1, "tok", nil)
	h := app.GetHistory(10, 0)
	if len(h) != 1 {
		t.Errorf("expected 1 item, got %d", len(h))
	}

	app.ClearHistory()
	h = app.GetHistory(10, 0)
	if len(h) != 0 {
		t.Errorf("expected 0 items, got %d", len(h))
	}
}

func TestGetImagesInDir(t *testing.T) {
	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	f1, _ := os.Create(filepath.Join(tmpDir, "img1.jpg"))
	f1.Close()
	f2, _ := os.Create(filepath.Join(tmpDir, "img2.png"))
	f2.Close()
	f3, _ := os.Create(filepath.Join(tmpDir, "ignore.txt"))
	f3.Close()

	imgs, err := getImagesInDir(tmpDir)
	if err != nil {
		t.Fatalf("getImagesInDir failed: %v", err)
	}
	if len(imgs) != 2 {
		t.Errorf("expected 2 images, got %d", len(imgs))
	}
}

func TestApp_PublishPost_Errors(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	// 1. Invalid Channel ID
	err := app.PublishPost(1, "invalid", "123", "content", time.Now().Format(time.RFC3339))
	if err == nil {
		t.Error("expected error for invalid channel id")
	}

	// 2. Invalid Access Hash
	err = app.PublishPost(1, "123", "invalid", "content", time.Now().Format(time.RFC3339))
	if err == nil {
		t.Error("expected error for invalid access hash")
	}

	// 3. History Item Not Found
	err = app.PublishPost(999, "123", "456", "content", time.Now().Format(time.RFC3339))
	if err == nil {
		t.Error("expected error for missing history")
	}

	// 4. Date Parse Error
	app.historyRepo.Add("T", "u", 1, "tok", nil)
	
	hist := app.GetHistory(1, 0)
	if len(hist) == 0 {
		t.Fatal("failed to setup history")
	}
	id := hist[0].ID

	err = app.PublishPost(id, "123", "456", "content", "bad-date")
	if err == nil {
		t.Error("expected error for bad date")
	}
}

func TestApp_TitleCRUD(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	err := app.CreateTitle("MangaX", "C:/Manga/X")
	if err != nil {
		t.Fatalf("CreateTitle failed: %v", err)
	}

	titles := app.GetTitles()
	if len(titles) != 1 {
		t.Errorf("expected 1 title, got %d", len(titles))
	}

	t1 := titles[0]
	t1.Name = "MangaY"
	err = app.UpdateTitle(t1)
	if err != nil {
		t.Errorf("UpdateTitle failed: %v", err)
	}

	t2, _ := app.GetTitleByID(t1.ID)
	if t2.Name != "MangaY" {
		t.Errorf("expected updated name, got %s", t2.Name)
	}

	err = app.DeleteTitle(t1.ID)
	if err != nil {
		t.Errorf("DeleteTitle failed: %v", err)
	}

	titles = app.GetTitles()
	if len(titles) != 0 {
		t.Errorf("expected 0 titles, got %d", len(titles))
	}
}

func TestApp_TemplateCRUD(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	err := app.CreateTemplate("Tpl1", "content")
	if err != nil {
		t.Fatalf("CreateTemplate failed: %v", err)
	}

	tpls := app.GetTemplates()
	if len(tpls) != 1 {
		t.Errorf("expected 1 template, got %d", len(tpls))
	}

	t1 := tpls[0]
	t1.Content = "new content"
	err = app.UpdateTemplate(t1)
	if err != nil {
		t.Errorf("UpdateTemplate failed: %v", err)
	}

	err = app.DeleteTemplate(t1.ID)
	if err != nil {
		t.Errorf("DeleteTemplate failed: %v", err)
	}
}