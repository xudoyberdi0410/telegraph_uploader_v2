package config

import (
	"os"
	"testing"
)

func TestLoad_NoConfig(t *testing.T) {
	// Ensure no config.json exists in the test directory
	os.Remove("config.json")

	_, err := Load()
	if err == nil {
		t.Error("expected error when config file is missing, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	// Create invalid config.json
	err := os.WriteFile("config.json", []byte("{invalid json"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("config.json")

	_, err = Load()
	if err == nil {
		t.Error("expected error when config file contains invalid JSON, got nil")
	}
}

func TestLoad_ValidJSON_MissingFields(t *testing.T) {
	// Create config with missing fields
	jsonContent := `{"bucket_name": "test"}`
	err := os.WriteFile("config.json", []byte(jsonContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("config.json")

	_, err = Load()
	if err == nil {
		t.Error("expected error when config file is missing required fields, got nil")
	}
}

func TestLoad_Success(t *testing.T) {
	// Create valid config
	jsonContent := `{
		"r2_account_id": "acc1",
		"r2_access_key": "key1",
		"r2_secret_key": "sec1",
		"bucket_name": "buck1",
		"public_domain": "dom1",
		"telegraph_token": "tok1"
	}`
	err := os.WriteFile("config.json", []byte(jsonContent), 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove("config.json")

	cfg, err := Load()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if cfg.R2AccountId != "acc1" {
		t.Errorf("expected acc1, got %s", cfg.R2AccountId)
	}
}

// Helper to mock executable path if needed, but Load() uses os.Executable()
// which in `go test` returns the path to the test binary in a temporary folder.
// The fallback logic in Load() checks CWD if not found near Executable.
// Our tests write to CWD, so the fallback logic in Load() will pick it up.
