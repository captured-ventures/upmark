//go:build !windows

package main

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

func revealInExplorer(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Run()
	default:
		// Linux + others: best-effort, open the parent dir.
		return openFolder(filepath.Dir(path))
	}
}

func openFolder(dir string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", dir).Run()
	default:
		return exec.Command("xdg-open", dir).Run()
	}
}
