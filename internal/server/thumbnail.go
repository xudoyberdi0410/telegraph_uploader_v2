package server

import (
	"image/jpeg"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/disintegration/imaging"
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
	// URL: /thumbnail/D%3A%5CManga%5CChapter1%5C01.jpg -> D%3A%5CManga%5CChapter1%5C01.jpg
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
		// Файл не найден или нет прав
		http.NotFound(res, req)
		return
	}
	defer file.Close()

	// 3. Декодируем картинку
	// AutoOrientation(true) важен для фото с телефонов
	img, err := imaging.Decode(file, imaging.AutoOrientation(true))
	if err != nil {
		http.Error(res, "Image decode failed", http.StatusInternalServerError)
		return
	}

	// 4. ГЕНЕРАЦИЯ ПРЕВЬЮ
	// imaging.Box — самый быстрый фильтр.
	// 300px ширина, высота авто.
	thumbnail := imaging.Resize(img, 300, 0, imaging.Box)

	// 5. Отдаем браузеру
	res.Header().Set("Content-Type", "image/jpeg")
	// Cache-Control важен! Чтобы браузер не запрашивал картинку снова при скролле
	res.Header().Set("Cache-Control", "public, max-age=3600")

	// Quality 70 — оптимально для превью
	if err := jpeg.Encode(res, thumbnail, &jpeg.Options{Quality: 70}); err != nil {
		http.Error(res, "Encode failed", http.StatusInternalServerError)
	}
}
