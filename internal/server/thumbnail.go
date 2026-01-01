package server

import (
    "io"
    "net/http"
    "net/url"
    "os"
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
    prefix := "/thumbnail/"
    if !strings.HasPrefix(req.URL.Path, prefix) {
        http.NotFound(res, req)
        return
    }

    // 1. Парсим путь из URL
    rawPath := strings.TrimPrefix(req.URL.Path, prefix)
    
    // Декодируем (превращаем %20 в пробелы, %5C в слэши)
    decodedPath, err := url.QueryUnescape(rawPath)
    if err != nil {
        http.Error(res, "Bad URL path", http.StatusBadRequest)
        return
    }

    // 2. Открываем файл
    file, err := os.Open(decodedPath)
    if err != nil {
        http.NotFound(res, req)
        return
    }
    defer file.Close()

    // 3. Отдаем файл как есть
    res.Header().Set("Content-Type", "image/jpeg")
    res.Header().Set("Cache-Control", "public, max-age=3600")
    
    // Копируем содержимое файла напрямую в response
    if _, err := io.Copy(res, file); err != nil {
        http.Error(res, "Failed to send file", http.StatusInternalServerError)
    }
}
