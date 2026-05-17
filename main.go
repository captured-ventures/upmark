package main

import (
	"embed"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

// parseStartupFile pulls the first non-flag argument from os.Args and returns
// its absolute path if it points at an existing file. Lets `upmark README.md`
// and OS file-association double-clicks both feed a path into the running app.
func parseStartupFile() string {
	for _, a := range os.Args[1:] {
		if a == "" || a[0] == '-' {
			continue
		}
		if abs, err := filepath.Abs(a); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return ""
}

func main() {
	app := NewApp()
	app.SetStartupFile(parseStartupFile())

	err := wails.Run(&options.App{
		Title:         "upmark",
		Width:         900,
		Height:        720,
		MinWidth:      480,
		MinHeight:     320,
		Frameless:     true,
		DisableResize: false,
		// Single-instance: a second `upmark file.md` invocation forwards its
		// args to the running instance and exits. Lets file-association double-
		// click reuse an existing window instead of spawning new ones.
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.captured-ventures.upmark",
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				path := ""
				for _, a := range data.Args[1:] {
					if a == "" || a[0] == '-' {
						continue
					}
					if abs, err := filepath.Abs(a); err == nil {
						if _, err := os.Stat(abs); err == nil {
							path = abs
							break
						}
					}
				}
				if path != "" {
					wailsruntime.EventsEmit(app.ctx, "open-from-second-instance", path)
				}
				wailsruntime.WindowUnminimise(app.ctx)
				wailsruntime.Show(app.ctx)
			},
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop:     true,
			DisableWebViewDrop: false,
			CSSDropProperty:    "--wails-drop-target",
			CSSDropValue:       "drop",
		},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: newAssetHandler(),
		},
		// Neutral dark fallback: shown only in the gap before the webview paints
		// (corners during maximize/restore, initial frame). White flashes in dark
		// mode were jarring; dark is acceptable in both schemes.
		BackgroundColour: &options.RGBA{R: 26, G: 24, B: 22, A: 1},
		OnStartup:        app.startup,
		OnDomReady:       app.domReady,
		OnShutdown:       app.shutdown,
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
