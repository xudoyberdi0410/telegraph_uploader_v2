package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	R2AccountId     string `json:"r2_account_id"`
	R2AccessKey     string `json:"r2_access_key"`
	R2SecretKey     string `json:"r2_secret_key"`
	BucketName      string `json:"bucket_name"`
	PublicDomain    string `json:"public_domain"`
	TelegraphToken  string `json:"telegraph_token"`
	TelegramAppId   int    `json:"telegram_app_id"`
	TelegramApiHash string `json:"telegram_app_hash"`
}

// loadConfig ищет config.json рядом с исполняемым файлом, а также поддерживает переменные окружения
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

	var cfg Config

	file, err := os.Open(configPath)
	if err == nil {
		defer file.Close()
		bytesValue, _ := io.ReadAll(file)
		if err := json.Unmarshal(bytesValue, &cfg); err != nil {
			return nil, fmt.Errorf("ошибка парсинга JSON: %v", err)
		}
	} else {
		// Если файла нет, не страшно, если все есть в ENV
		// Но пока оставим логику как есть - файл ожидается.
		// Хотя для улучшения можно сделать файл необязательным, если есть ENV.
		// Вернем ошибку только если и файла нет, и ENV пустые (проверим валидацию ниже).
	}

	// Переопределение через переменные окружения
	if val := os.Getenv("R2_ACCOUNT_ID"); val != "" {
		cfg.R2AccountId = val
	}
	if val := os.Getenv("R2_ACCESS_KEY"); val != "" {
		cfg.R2AccessKey = val
	}
	if val := os.Getenv("R2_SECRET_KEY"); val != "" {
		cfg.R2SecretKey = val
	}
	if val := os.Getenv("BUCKET_NAME"); val != "" {
		cfg.BucketName = val
	}
	if val := os.Getenv("PUBLIC_DOMAIN"); val != "" {
		cfg.PublicDomain = val
	}
	if val := os.Getenv("TELEGRAPH_TOKEN"); val != "" {
		cfg.TelegraphToken = val
	}
	if val := os.Getenv("TELEGRAM_APP_ID"); val != "" {
		// Parse int
		if id, err := strconv.Atoi(val); err == nil {
			cfg.TelegramAppId = id
		}
	}
	if val := os.Getenv("TELEGRAM_API_HASH"); val != "" {
		cfg.TelegramApiHash = val
	}

	// Валидация
	if cfg.R2AccountId == "" || cfg.R2AccessKey == "" || cfg.R2SecretKey == "" {
		return nil, fmt.Errorf("конфигурация неполная: проверьте config.json или переменные окружения (R2 keys)")
	}

	return &cfg, nil // Возвращаем готовую структуру
}
