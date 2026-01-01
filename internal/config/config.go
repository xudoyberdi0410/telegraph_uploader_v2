package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Config struct {
	R2AccountId    string `json:"r2_account_id"`
	R2AccessKey    string `json:"r2_access_key"`
	R2SecretKey    string `json:"r2_secret_key"`
	BucketName     string `json:"bucket_name"`
	PublicDomain   string `json:"public_domain"`
	TelegraphToken string `json:"telegraph_token"` // Опционально: токен Telegraph
}

// loadConfig ищет config.json рядом с исполняемым файлом
func Load() (*Config, error) {
	// Получаем путь к исполняемому файлу
	ex, err := os.Executable()
	if err != nil {
		return nil, err
	}
	exPath := filepath.Dir(ex)

	// Сначала пробуем найти конфиг рядом с .exe
	configPath := filepath.Join(exPath, "config.json")

	// Если не нашли (например, при wails dev), пробуем в текущей директории
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "config.json"
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytesValue, _ := io.ReadAll(file)
	
	var cfg Config // Создаем локальную переменную
	err = json.Unmarshal(bytesValue, &cfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	// Валидация
	if cfg.R2AccountId == "" || cfg.R2AccessKey == "" || cfg.R2SecretKey == "" {
		return nil, fmt.Errorf("в config.json не заполнены ключи R2")
	}

	return &cfg, nil // Возвращаем готовую структуру
}
