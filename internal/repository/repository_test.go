package repository

import (
	"testing"
	"time"

	"telegraph_uploader_v2/internal/database"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t testing.TB) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&database.Settings{}, &database.HistoryEntry{}, &database.Title{}, &database.TitleFolder{}, &database.TitleVariable{}, &database.Template{})
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

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

func TestSettingsRepo(t *testing.T) {
	db := setupTestDB(t)
	repo := NewSettingsRepository(db)

	s, err := repo.Get()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if s.ResizeTo != 1600 {
		t.Errorf("expected default ResizeTo 1600, got %d", s.ResizeTo)
	}

	s.ResizeTo = 2000
	s.Resize = true
	err = repo.Update(s)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	s2, _ := repo.Get()
	if s2.ResizeTo != 2000 {
		t.Errorf("expected updated ResizeTo 2000, got %d", s2.ResizeTo)
	}
}

func TestHistoryRepo(t *testing.T) {
	db := setupTestDB(t)
	repo := NewHistoryRepository(db)

	// Add
	_, err := repo.Add("T1", "u1", 1, "tok", nil)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	_, err = repo.Add("T2", "u2", 2, "tok", nil)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Get
	items, err := repo.Get(10, 0)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if items[0].Title != "T2" {
		t.Errorf("expected newest first, got %s", items[0].Title)
	}

	// Clear
	err = repo.Clear()
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	items, _ = repo.Get(10, 0)
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestTitleRepo(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTitleRepository(db)

	err := repo.Create("Naruto", "C:\\Manga\\Naruto")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	titles, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(titles) != 1 {
		t.Fatalf("expected 1 title")
	}
	if titles[0].Name != "Naruto" {
		t.Errorf("expected Naruto")
	}

	// Update
	t1 := titles[0]
	t1.Name = "Boruto"
	err = repo.Update(t1)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	t2, err := repo.GetByID(t1.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if t2.Name != "Boruto" {
		t.Errorf("expected Boruto")
	}

	// FindByPath
	found, err := repo.FindByPath("C:\\Manga\\Naruto\\Chapter1")
	if err != nil {
		t.Fatalf("FindByPath failed: %v", err)
	}
	if found.Name != "Boruto" { // Renamed above
		t.Errorf("expected Boruto, got %s", found.Name)
	}

	// Delete
	err = repo.Delete(t1.ID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	titles, _ = repo.GetAll()
	if len(titles) != 0 {
		t.Errorf("expected 0 titles")
	}
}

func TestTemplateRepo(t *testing.T) {
	db := setupTestDB(t)
	repo := NewTemplateRepository(db)

	err := repo.Create("Tpl", "Content")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	tpls, _ := repo.GetAll()
	if len(tpls) != 1 {
		t.Fatalf("expected 1")
	}

	t1 := tpls[0]
	t1.Content = "New"
	repo.Update(t1)

	t2, _ := repo.GetByID(t1.ID)
	if t2.Content != "New" {
		t.Errorf("expected New")
	}

	repo.Delete(t1.ID)
	tpls, _ = repo.GetAll()
	if len(tpls) != 0 {
		t.Errorf("expected 0")
	}
}
