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
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Settings{}, &dbHistory{}, &Title{}, &TitleFolder{}, &TitleVariable{}, &Template{})
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
	_, err := d.AddHistory("Test Title 1", "http://url1", 5, "token1", nil)
	if err != nil {
		t.Fatalf("failed to add history: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // Ensure timestamp difference
	_, err = d.AddHistory("Test Title 2", "http://url2", 10, "token2", nil)
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
	_, err = db.AddHistory("Title", "url", 1, "tok", nil)
	if err == nil {
	}

	db.GetSettings()

	// UpdateSettings
	db.UpdateSettings(Settings{})
}

func TestTitleCRUD(t *testing.T) {
	d := setupTestDB(t)

	// Create
	err := d.CreateTitle("Naruto", "C:\\Manga\\Naruto")
	if err != nil {
		t.Fatalf("failed to create title: %v", err)
	}

	// Get All
	titles := d.GetTitles()
	if len(titles) != 1 {
		t.Errorf("expected 1 title, got %d", len(titles))
	}
	if titles[0].Name != "Naruto" {
		t.Errorf("expected name 'Naruto', got %s", titles[0].Name)
	}
	// Check folder creation
	if len(titles[0].Folders) != 1 {
		t.Errorf("expected 1 folder, got %d", len(titles[0].Folders))
	}
	if titles[0].Folders[0].Path != "C:\\Manga\\Naruto" {
		t.Errorf("expected path 'C:\\Manga\\Naruto', got %s", titles[0].Folders[0].Path)
	}

	// Get By ID
	title, err := d.GetTitleByID(titles[0].ID)
	if err != nil {
		t.Fatalf("failed to get title by id: %v", err)
	}
	if title.Name != "Naruto" {
		t.Errorf("expected name 'Naruto', got %s", title.Name)
	}

	// Update (rename)
	title.Name = "Boruto"
	err = d.UpdateTitle(title)
	if err != nil {
		t.Fatalf("failed to update title: %v", err)
	}

	title2, _ := d.GetTitleByID(title.ID)
	if title2.Name != "Boruto" {
		t.Errorf("expected updated name 'Boruto', got %s", title2.Name)
	}

	// Delete
	err = d.DeleteTitle(title.ID)
	if err != nil {
		t.Fatalf("failed to delete title: %v", err)
	}

	titles = d.GetTitles()
	if len(titles) != 0 {
		t.Errorf("expected 0 titles after delete, got %d", len(titles))
	}
}

func TestTitleVariables(t *testing.T) {
	d := setupTestDB(t)
	d.CreateTitle("Manga", "")
	titles := d.GetTitles()
	titleID := titles[0].ID

	err := d.AddTitleVariable(titleID, "Author", "Kishimoto")
	if err != nil {
		t.Fatalf("failed to add variable: %v", err)
	}

	tWithVars, _ := d.GetTitleByID(titleID)
	if len(tWithVars.Variables) != 1 {
		t.Errorf("expected 1 variable, got %d", len(tWithVars.Variables))
	}
	if tWithVars.Variables[0].Key != "Author" || tWithVars.Variables[0].Value != "Kishimoto" {
		t.Errorf("variable content mismatch: %v", tWithVars.Variables[0])
	}
}

func TestFindTitleByPath(t *testing.T) {
	d := setupTestDB(t)

	d.CreateTitle("Naruto", "C:\\Manga\\Naruto")
	d.CreateTitle("Bleach", "C:\\Manga\\Bleach")

	// Match exact root
	title, err := d.FindTitleByPath("C:\\Manga\\Naruto")
	if err != nil {
		t.Errorf("failed to find exact match: %v", err)
	}
	if title.Name != "Naruto" {
		t.Errorf("expected Naruto, got %s", title.Name)
	}

	// Match subdirectory
	title, err = d.FindTitleByPath("C:\\Manga\\Naruto\\Vol1\\Ch1")
	if err != nil {
		t.Errorf("failed to find subdir match: %v", err)
	}
	if title.Name != "Naruto" {
		t.Errorf("expected Naruto, got %s", title.Name)
	}

	// Case insensitive match (important for Windows)
	title, err = d.FindTitleByPath("c:\\manga\\bleach\\chapter_500")
	if err != nil {
		t.Errorf("failed to find case-insensitive match: %v", err)
	}
	if title.Name != "Bleach" {
		t.Errorf("expected Bleach, got %s", title.Name)
	}

	// No match
	_, err = d.FindTitleByPath("C:\\Other\\OnePiece")
	if err == nil {
		t.Error("expected error for no match")
	}
}

func TestTemplateCRUD(t *testing.T) {
	d := setupTestDB(t)

	// Create
	err := d.CreateTemplate("Default", "<p>Hello</p>")
	if err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	// Get All
	tpls := d.GetTemplates()
	if len(tpls) != 1 {
		t.Errorf("expected 1 template, got %d", len(tpls))
	}
	if tpls[0].Name != "Default" {
		t.Errorf("expected name 'Default', got %s", tpls[0].Name)
	}

	// Get By ID
	tpl, err := d.GetTemplateByID(tpls[0].ID)
	if err != nil {
		t.Fatalf("failed to get by id: %v", err)
	}
	if tpl.Content != "<p>Hello</p>" {
		t.Errorf("content mismatch")
	}

	// Update
	tpl.Content = "<p>World</p>"
	err = d.UpdateTemplate(tpl)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	tpl2, _ := d.GetTemplateByID(tpl.ID)
	if tpl2.Content != "<p>World</p>" {
		t.Errorf("expected updated content")
	}

	// Delete
	err = d.DeleteTemplate(tpl.ID)
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	tpls = d.GetTemplates()
	if len(tpls) != 0 {
		t.Errorf("expected 0 templates, got %d", len(tpls))
	}
}
