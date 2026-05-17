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

// hasFlag returns true if `--name` (or `-name`) appears anywhere in os.Args.
// We don't use the flag package because it would consume positional args we
// reserve for file paths.
func hasFlag(name string) bool {
	for _, a := range os.Args[1:] {
		if a == "--"+name || a == "-"+name {
			return true
		}
	}
	return false
}

func main() {
	serverMode := hasFlag("mcp-server")

	app := NewApp()
	app.SetStartupFile(parseStartupFile())
	app.SetServerMode(serverMode)

	// In server mode (--mcp-server, invoked by the MCPB bridge), refuse to
	// start a second instance if a live MCP server is already listening.
	// The bridge will connect to the existing one.
	if serverMode {
		if lf := liveMCPLock(); lf != nil {
			// Existing server is alive — exit clean so the bridge knows it
			// can proceed against the already-running endpoint.
			os.Exit(0)
		}
	}

	opts := &options.App{
		Title:         "upmark",
		Width:         900,
		Height:        720,
		MinWidth:      480,
		MinHeight:     320,
		Frameless:     true,
		DisableResize: false,
		// Hidden launch for --mcp-server: the bridge invoked us as a
		// background process; show the window only when an LLM presents a
		// doc (controlled by the MCPWindowOnPresent pref).
		StartHidden: serverMode,
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
	}

	// Single-instance lock only applies to UI mode. Server-mode invocations
	// are coordinated via the MCP lockfile instead — different concern, different
	// mechanism. Letting both share the same instance lock would forward an
	// --mcp-server launch into an existing UI window, which isn't what the
	// bridge expects.
	if !serverMode {
		opts.SingleInstanceLock = &options.SingleInstanceLock{
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
		}
	}

	if err := wails.Run(opts); err != nil {
		println("Error:", err.Error())
	}
}
