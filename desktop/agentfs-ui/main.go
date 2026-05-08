package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Fixed window dimensions for consistent UI
	const windowWidth = 480
	const windowHeight = 640

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "AgentFS",
		Width:         windowWidth,
		Height:        windowHeight,
		MinWidth:      windowWidth,
		MinHeight:     windowHeight,
		MaxWidth:      windowWidth,
		MaxHeight:     windowHeight,
		DisableResize: true,
		Frameless:     false,
		StartHidden:   false,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 26, G: 26, B: 46, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
		Linux: &linux.Options{
			ProgramName: "AgentFS",
			Icon:        nil, // Will use default icon
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
