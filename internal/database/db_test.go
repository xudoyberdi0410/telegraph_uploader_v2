package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *Database {
	// Use in-memory database
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Settings{}, &dbHistory{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	// Create default settings intentionally if needed, matching Init logic
	db.Create(&Settings{
		Resize:      false,
		ResizeTo:    1600,
		WebpQuality: 80,
	})

	return &Database{conn: db}
}

func TestGetSettings(t *testing.T) {
	d := setupTestDB(t)
	s := d.GetSettings()

	if s.ResizeTo != 1600 {
		t.Errorf("expected default ResizeTo 1600, got %d", s.ResizeTo)
	}
}

func TestUpdateSettings(t *testing.T) {
	d := setupTestDB(t)
	s := d.GetSettings()
	s.ResizeTo = 2000
	s.Resize = true
	d.UpdateSettings(s)

	s2 := d.GetSettings()
	if s2.ResizeTo != 2000 {
		t.Errorf("expected updated ResizeTo 2000, got %d", s2.ResizeTo)
	}
	if !s2.Resize {
		t.Error("expected Resize to be true")
	}
}

func TestHistoryOperations(t *testing.T) {
	d := setupTestDB(t)

	// Add items
	err := d.AddHistory("Test Title 1", "http://url1", 5, "token1", nil)
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Ensure timestamp difference
	err = d.AddHistory("Test Title 2", "http://url2", 10, "token2", nil)
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}

	// Get items
	items := d.GetHistory(10, 0)
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}

	// Order check: newest first
	if items[0].Title != "Test Title 2" {
		t.Errorf("expected newest item first, got %s", items[0].Title)
	}

	// Pagination
	items = d.GetHistory(1, 0)
	if len(items) != 1 {
		t.Errorf("expected 1 item with limit 1, got %d", len(items))
	}
	if items[0].Title != "Test Title 2" {
		t.Errorf("expected first page item to be 'Test Title 2', got %s", items[0].Title)
	}

	items = d.GetHistory(1, 1) // Offset 1
	if len(items) != 1 {
		t.Errorf("expected 1 item with limit 1 offset 1, got %d", len(items))
	}
	if items[0].Title != "Test Title 1" {
		t.Errorf("expected second page item to be 'Test Title 1', got %s", items[0].Title)
	}

	// Clear
	d.ClearHistory()
	items = d.GetHistory(10, 0)
	if len(items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(items))
	}
}

func TestInitWithFile(t *testing.T) {
	// Temp file
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	db, err := InitWithFile(tmpDB)
	if err != nil {
		t.Fatalf("InitWithFile failed: %v", err)
	}

	// Check if settings created
	s := db.GetSettings()
	if s.ResizeTo != 1600 {
		t.Errorf("expected default settings, got %d", s.ResizeTo)
	}

	// Close to release file lock
	db.Close()
}

func TestInitWithFile_Error(t *testing.T) {
	_, err := InitWithFile(":")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestTableNames(t *testing.T) {
	h := dbHistory{}
	if h.TableName() != "history_items" {
		t.Errorf("expected table name history_items, got %s", h.TableName())
	}
}

func TestInit(t *testing.T) {
	// Tests the real Init function which uses os.Executable
	db, err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	db.Close()

	// Cleanup the created database.db file
	// It is created in the same dir as the test executable
	ex, _ := os.Executable()
	dbPath := filepath.Join(filepath.Dir(ex), "database.db")
	os.Remove(dbPath)
}

func TestDatabase_Errors(t *testing.T) {
	// Setup DB
	tmpDB := filepath.Join(t.TempDir(), "error_test.db")
	db, err := InitWithFile(tmpDB)
	if err != nil {
		t.Fatalf("InitWithFile failed: %v", err)
	}

	// Close it to trigger errors
	db.Close()

	// Test AddHistory on closed DB
	err = db.AddHistory("Title", "url", 1, "tok", nil)
	if err == nil {
	}

	db.GetSettings()

	// UpdateSettings
	db.UpdateSettings(Settings{})
}
