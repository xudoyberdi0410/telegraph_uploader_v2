package main

import (
	"embed"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"gopkg.in/natefinch/lumberjack.v2"

	"telegraph_uploader_v2/internal/server"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// --- Логирование (без изменений) ---
	logFile := &lumberjack.Logger{
		Filename:   "app.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("Приложение запущено")
	// -----------------------------------

	app := NewApp()

	// Инициализируем наш обработчик файлов
	thumbnailHandler := server.NewFileLoader()

	err := wails.Run(&options.App{
		Title:  "Telegraph Uploader v2",
		Width:  1024,
		Height: 768,

		AssetServer: &assetserver.Options{
			Assets: assets,
			// ИСПОЛЬЗУЕМ MIDDLEWARE ВМЕСТО HANDLER
			// Middleware выполняется первым, до того как запрос уйдет в Vite (в dev) или embed.FS (в prod)
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Проверяем, начинается ли путь с /thumbnail/
					if strings.HasPrefix(r.URL.Path, "/thumbnail/") {
						// Если да - обрабатываем нашим Go-кодом
						log.Printf("[Middleware] Serving thumbnail: %s", r.URL.Path)
						thumbnailHandler.ServeHTTP(w, r)
						return
					}
					
					// Если нет - передаем управление дальше (Vite или статика)
					next.ServeHTTP(w, r)
				})
			},
		},

		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			BackdropType:         windows.Mica,
		},
		LogLevel: logger.ERROR,

		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
	})

	if err != nil {
		log.Fatal("Error:", err)
	}
}
