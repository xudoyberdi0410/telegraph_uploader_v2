package main

import (
	"embed"
	"image/jpeg"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

// 1. Создаем структуру для нашего загрузчика файлов
type FileLoader struct {
	http.Handler
}

func (h *FileLoader) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 1. Парсим путь из URL
	// URL будет вида: /thumbnail/D%3A%5CManga%5CChapter1%5C01.jpg
	rawPath := strings.TrimPrefix(req.URL.Path, "/thumbnail/")
	
	// Декодируем (превращаем %20 в пробелы, %5C в слэши)
	decodedPath, err := url.QueryUnescape(rawPath)
	if err != nil {
		http.Error(res, "Bad path", http.StatusBadRequest)
		return
	}

	// 2. Открываем файл
	file, err := os.Open(decodedPath)
	if err != nil {
		http.NotFound(res, req)
		return
	}
	defer file.Close()

	// 3. Быстро декодируем конфиг (чтобы узнать размер)
	// Если картинка огромная, декодируем её с resize-on-load (если формат позволяет)
	// Но для универсальности используем imaging.Decode
	img, err := imaging.Decode(file, imaging.AutoOrientation(true))
	if err != nil {
		http.Error(res, "Image decode failed", http.StatusInternalServerError)
		return
	}

	// 4. ГЕНЕРАЦИЯ ПРЕВЬЮ
	// imaging.Box - самый быстрый алгоритм ресайза (быстрее Lanczos, но качество чуть хуже, для превью идеально)
	// Ширина 300px, высота авто (0)
	thumbnail := imaging.Resize(img, 300, 0, imaging.Box)

	// 5. Отдаем браузеру
	res.Header().Set("Content-Type", "image/jpeg")
	// Кэшируем в браузере на час, чтобы при скролле не пересчитывать
	res.Header().Set("Cache-Control", "public, max-age=3600") 

	// Качество 70 - достаточно для превью, файл будет весить 10-20кб
	jpeg.Encode(res, thumbnail, &jpeg.Options{Quality: 70})
}

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "Manga Uploader",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
			// 3. Подключаем наш обработчик
			Handler: &FileLoader{},
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
