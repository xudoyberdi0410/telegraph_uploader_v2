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

	"telegraph_uploader_v2/internal/config"
	"telegraph_uploader_v2/internal/database"
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

// Helper setup reused
// ... (omitted if separate file, but here we append to app_test.go or create new one?
// I should append to app_test.go or overwrite it with MORE tests.
// I'll overwrite app_test.go with comprehensive set.

func setupTestDB(t *testing.T) *database.Database {
	// Use unshared memory DB for isolation per test or handle cleanup
	// "file::memory:?cache=shared" shares DB across connections, good for multiple opens but needs cleanup.
	// We can use unique name per test? Or just :memory: without shared cache if we reuse 'gorm.Open' result.
	// database.New takes *gorm.DB.
	// But InitWithFile does migrations.
	// Let's use database.New which does migrations too.
	// But we need to ensure defaults are created.
	// The problem in TestApp_Settings failure: "record not found".
	// database.New runs AutoMigrate but does NOT create default settings.
	// We should duplicate default creation logic or refactor db.New to do it.

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	d := database.New(db)

	// Manually create defaults if needed, OR refactor db.New.
	// Let's manually create default settings here to match Init behavior.
	var count int64
	db.Model(&database.Settings{}).Count(&count)
	if count == 0 {
		db.Create(&database.Settings{
			Resize:      false,
			ResizeTo:    1600,
			WebpQuality: 80,
		})
	}

	return d
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
	tgClient := telegraph.New(cfg)
	tgClient.BaseURL = tsTelegraph.URL

	minioClient, _ := minio.New(tsS3.Listener.Addr().String(), &minio.Options{
		Creds:  credentials.NewStaticV4("key", "secret", ""),
		Secure: false,
	})
	upl := uploader.NewWithClient(minioClient, cfg)

	app := &App{
		ctx:        context.Background(),
		config:     cfg,
		uploader:   upl,
		tgphClient: tgClient,
		db:         db,
		dialogs:    &MockDialogProvider{}, // Default mock
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

	// Test nil uploader
	appNil := &App{uploader: nil}
	res = appNil.UploadChapter([]string{"f"}, uploader.ResizeSettings{})
	if res.Success || res.Error != "Загрузчик не инициализирован (проверьте config.json)" {
		t.Error("expected error for nil uploader")
	}
}

func TestApp_CreateTelegraphPage(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	url := app.CreateTelegraphPage("Title", []string{"http://img.jpg"}, 0)
	if url != "http://telegra.ph/test" {
		t.Errorf("expected url http://telegra.ph/test, got %s", url)
	}

	// Test failure from client
	// We need client to fail.
	// We can mock the client or change BaseURL to invalid?
	// But `app.tgClient` is concrete struct.
	// We can change `BaseURL` on the client instance app holds.
	// But `CreatePage` in client handles errors by returning error string.
	// We want `url[:4] == "http"` check to fail.

	// Let's replace BaseURL with bad one that returns error
	tsFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok": false, "error": "FAIL"}`))
	}))
	defer tsFail.Close()

	app.tgphClient.BaseURL = tsFail.URL
	url = app.CreateTelegraphPage("Title", nil, 0)
	// It should log failure and return error string
	if url[:4] == "http" {
		t.Error("expected error string, got http url")
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
	app.tgphClient.BaseURL = tsFail.URL

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

	// Invalid URL (no slug?)
	// app.GetTelegraphPage logic: split by '/', len(parts)==0 -> invalid.
	// "" -> split -> [""] -> len=1. path=""
	// "http://t.ph/" -> ...
	// To trigger `len(parts) == 0`, string must be empty?
	// split "" -> [""] (len 1).
	// usage of strings.Split will almost always return len >= 1.
	// Wait, code says:
	// parts := strings.Split(pageUrl, "/")
	// if len(parts) == 0 { ... }
	// This branch might be unreachable with logic `Split(s, "/")`.
	// Only if s is somehow resulting in empty slice? strings.Split docs say: "If s does not contain sep and sep is not empty, Split returns a slice of length 1 whose only element is s."
	// So `len(parts) == 0` is dead code?
	// Yes. But let's verify logic in `app.go`.

	// GetPage error
	tsFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer tsFail.Close()
	app.tgphClient.BaseURL = tsFail.URL

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

func TestNewApp(t *testing.T) {
	// NewApp uses os.Executable to find config/db path.
	// In test, it might use the temp dir where test binary is, or fallback to current dir.
	// We want to ensure it doesn't panic.
	// It touches real filesystem (database.db, config.json).

	// Create a dummy config in current dir to avoid "could not load config" warning impacting flow?
	// Or just let it use defaults.

	// Issue: NewApp calls database.Init() which locks database.db.
	// We must ensure we clean it up or NewApp might fail if locked by other tests (TestInit in db package runs in different process usually, but here checking file lock).

	// We should cleanup before calling NewApp just in case.
	ex, _ := os.Executable()
	dbPath := filepath.Join(filepath.Dir(ex), "database.db")
	os.Remove(dbPath)
	defer os.Remove(dbPath) // Cleanup after

	app := NewApp()
	if app == nil {
		t.Fatal("NewApp returned nil")
	}

	// Check if defaults loaded
	if app.config == nil {
		t.Error("config is nil")
	}
	if app.db == nil {
		t.Error("db is nil")
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

	app.SaveSettings(uploader.ResizeSettings{ResizeTo: 2000})
	s = app.GetSettings()
	if s.ResizeTo != 2000 {
		t.Errorf("expected 2000, got %d", s.ResizeTo)
	}
}

func TestApp_History(t *testing.T) {
	app, ts1, ts2 := setupTestApp(t)
	defer ts1.Close()
	defer ts2.Close()

	app.db.AddHistory("T1", "u1", 1, "tok", nil)

	// CreateTelegraphPage from previous test might have added history if tests run in parallel or DB shared?
	// DB is shared "file::memory:?cache=shared".
	// We should clear history before asserting count or use empty DB.
	// Better: setupTestDB should return fresh DB.
	// If we use "file::memory:?cache=shared", it persists as long as one connection is open?
	// Actually, with unique names per test is safer.
	// Or just ClearHistory first.
	app.ClearHistory()

	app.db.AddHistory("T1", "u1", 1, "tok", nil)
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
	// Need to expose this or extract it. `getImagesInDir` is unexported in main package.
	// But since we are in `package main` (test package `main`), we can access it!
	// Yes, `app_test.go` is package main.

	tmpDir := t.TempDir()
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)

	// Windows file locking issue: getImagesInDir opens files? No, it uses os.ReadDir.
	// Maybe "ignore.txt" creation holds lock?
	// os.Create returns *File, we must close it!
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
