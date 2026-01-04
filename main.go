package main

import (
	"embed"
	"log"
	"os"
	"io"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"gopkg.in/natefinch/lumberjack.v2"

	// Импорты ваших новых пакетов
	"telegraph_uploader_v2/internal/server"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	logFile := &lumberjack.Logger{
		Filename: "app.log",
		MaxSize: 10,
		MaxBackups: 5,
		MaxAge: 30,
		Compress: true,
	}

	multiWriter := io.MultiWriter(os.Stdout, logFile)

	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	log.Println("Приложения запущено")
	
	app := NewApp()

	// Создаем и запускаем окно
	err := wails.Run(&options.App{
		Title:  "Telegraph Uploader v2",
		Width:  1024,
		Height: 768,

		// Подключаем наш новый FileLoader для картинок
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: server.NewFileLoader(),
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
        // Настройки для Windows (опционально)
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
