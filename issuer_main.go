//go:build issuer

package main

import (
	"context"
	"embed"

	"res-downloader/core"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var issuerAssets embed.FS

//go:embed build/appicon.png
var issuerIcon []byte

func main() {
	issuer := core.NewIssuer()
	_ = issuerIcon
	err := wails.Run(&options.App{
		Title:                    "签发授权码",
		Width:                    560,
		MinWidth:                 480,
		Height:                   680,
		MinHeight:                620,
		Frameless:                true,
		EnableDefaultContextMenu: true,
		AssetServer: &assetserver.Options{
			Assets: issuerAssets,
		},
		BackgroundColour: &options.RGBA{R: 255, G: 255, B: 255, A: 1},
		OnStartup: func(ctx context.Context) {
			issuer.Startup(ctx)
		},
		Bind: []interface{}{
			issuer,
		},
		Windows: &windows.Options{
			WebviewIsTransparent:              false,
			WindowIsTranslucent:               false,
			DisableFramelessWindowDecorations: false,
		},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
