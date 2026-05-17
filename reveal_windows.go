//go:build windows

package main

import (
	"fmt"
	"os/exec"
)

// revealInExplorer: `explorer /select,"path"` opens Explorer at the file's
// directory with the file pre-selected.
//
// explorer.exe returns exit code 1 on success — a Go-canonical "error" we
// have to swallow. We only treat a "executable not found" failure as real.
func revealInExplorer(path string) error {
	cmd := exec.Command("explorer.exe", "/select,"+path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("explorer.exe start: %w", err)
	}
	// Wait briefly so we can report immediate spawn failures, then detach.
	go func() { _ = cmd.Wait() }()
	return nil
}

func openFolder(dir string) error {
	cmd := exec.Command("explorer.exe", dir)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("explorer.exe start: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
