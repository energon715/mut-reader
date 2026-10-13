package main

import (
	"embed"
	"log"
	"mut/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	appInstance := app.NewApp()
	err := wails.Run(&options.App{
		Title:      "mut",
		Width:      1024,
		Height:     768,
		MinWidth:   800,
		MinHeight:  600,
		OnStartup:  appInstance.Startup,
		OnShutdown: appInstance.Shutdown,
		Bind: []interface{}{
			appInstance,
		},
		Assets: assets,
	})
	if err != nil {
		log.Printf("wails run error: %v", err)
	}

}
