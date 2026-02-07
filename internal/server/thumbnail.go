package server

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// FileLoader обрабатывает запросы на получение локальных изображений
type FileLoader struct{}

// NewFileLoader создает новый экземпляр хендлера
func NewFileLoader() *FileLoader {
    return &FileLoader{}
}

func (h *FileLoader) ServeHTTP(res http.ResponseWriter, req *http.Request) {
    // 0. Проверка безопасности: обрабатываем только пути, начинающиеся с /thumbnail/
    log.Printf("Received thumbnail request: %s", req.URL.Path)
	prefix := "/thumbnail/"
    if !strings.HasPrefix(req.URL.Path, prefix) {
        http.NotFound(res, req)
		log.Printf("Invalid thumbnail request path: %s", req.URL.Path)
        return
    }

    // 1. Парсим путь из URL
    rawPath := strings.TrimPrefix(req.URL.Path, prefix)
    
    // Декодируем (превращаем %20 в пробелы, %5C в слэши)
    decodedPath, err := url.QueryUnescape(rawPath)
    if err != nil {
        http.Error(res, "Bad URL path", http.StatusBadRequest)
		log.Printf("Error decoding thumbnail path %s: %v", rawPath, err)
        return
    }

    // 1.5. Санитизация и проверка безопасности
    // Нормализуем путь
    cleanedPath := filepath.Clean(decodedPath)

    // Проверка на path traversal (наличие ..)
    // Clean убирает .. если это возможно, но если путь начинается с .. они остаются.
    // Мы запрещаем использование .. как класс для безопасности.
    // Обратите внимание: мы разрешаем абсолютные пути, так как в Wails-приложении
    // фронтенд часто запрашивает локальные файлы по их полному пути.
    // Безопасность обеспечивается ограничением по расширению и запретом на ..
    if strings.Contains(cleanedPath, "..") || strings.Contains(decodedPath, "..") {
        http.Error(res, "Invalid path: traversal not allowed", http.StatusForbidden)
        log.Printf("Blocked potential path traversal attempt: %s", decodedPath)
        return
    }

    // Проверка расширения (только изображения)
    ext := strings.ToLower(filepath.Ext(cleanedPath))
    allowedExtensions := map[string]bool{
        ".jpg":  true,
        ".jpeg": true,
        ".png":  true,
        ".webp": true,
    }
    if !allowedExtensions[ext] {
        http.Error(res, "Invalid file type", http.StatusForbidden)
        log.Printf("Blocked attempt to access non-image file: %s", cleanedPath)
        return
    }

    // Проверка, что это файл, а не директория
    info, err := os.Stat(cleanedPath)
    if err != nil {
        if os.IsNotExist(err) {
            http.NotFound(res, req)
        } else {
            http.Error(res, "Error accessing file", http.StatusInternalServerError)
        }
        log.Printf("Error stating file %s: %v", cleanedPath, err)
        return
    }
    if info.IsDir() {
        http.Error(res, "Is a directory", http.StatusForbidden)
        log.Printf("Blocked attempt to access directory: %s", cleanedPath)
        return
    }

    // 2. Открываем файл
    file, err := os.Open(cleanedPath)
    if err != nil {
        http.NotFound(res, req)
		log.Printf("File not found: %s", cleanedPath)
        return
    }
    defer file.Close()
	log.Printf("Opened file for thumbnail: %s", cleanedPath)

    // 3. Отдаем файл
    contentType := "application/octet-stream"
    switch ext {
    case ".jpg", ".jpeg":
        contentType = "image/jpeg"
    case ".png":
        contentType = "image/png"
    case ".webp":
        contentType = "image/webp"
    }

    res.Header().Set("Content-Type", contentType)
    res.Header().Set("Cache-Control", "public, max-age=3600")
    log.Printf("Sending thumbnail file: %s", cleanedPath)
    // Копируем содержимое файла напрямую в response
    if _, err := io.Copy(res, file); err != nil {
		log.Printf("Error sending file %s: %v", cleanedPath, err)
        http.Error(res, "Failed to send file", http.StatusInternalServerError)
    }
	log.Printf("Served thumbnail: %s", cleanedPath)
}
