package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitWithFile(t *testing.T) {
	// Temp file
	tmpDB := filepath.Join(t.TempDir(), "test.db")
	db, err := InitWithFile(tmpDB)
	if err != nil {
		t.Fatalf("InitWithFile failed: %v", err)
	}

	// Check if settings created (to ensure migration ran)
	var count int64
	db.Model(&Settings{}).Count(&count)
	if count == 0 {
		t.Error("expected default settings to be created")
	}

	// Close to release file lock
	sqlDB, _ := db.DB()
	sqlDB.Close()
}

func TestInitWithFile_Error(t *testing.T) {
	// Use a directory as path, which should cause an error for sqlite open
	_, err := InitWithFile(t.TempDir())
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestTableNames(t *testing.T) {
	h := HistoryEntry{}
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
	sqlDB, _ := db.DB()
	sqlDB.Close()

	// Cleanup the created database.db file
	ex, _ := os.Executable()
	dbPath := filepath.Join(filepath.Dir(ex), "database.db")
	os.Remove(dbPath)
}