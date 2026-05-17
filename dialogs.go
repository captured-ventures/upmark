package main

import (
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// openDirectoryDialog wraps Wails' native directory picker so folder.go has a
// stable function signature even if the runtime API churns.
func openDirectoryDialog(a *App) (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Open Folder",
	})
}
