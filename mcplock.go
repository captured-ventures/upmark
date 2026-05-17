package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// mcplock.go — tracks which process currently owns the MCP SSE port.
//
// The lockfile sits in upmark's per-user config dir (same place as prefs.json).
// Any upmark process that starts the MCP server writes the lock; whichever
// process stops it clears the lock. The MCPB bridge reads this file to find
// the live SSE endpoint without having to probe a port range.
//
// Liveness check is HTTP, not PID-based: we GET the recorded URL and trust
// the response. A stale lockfile (process died, file left behind) shows up as
// a connection error and gets overwritten on the next start.

type mcpLockFile struct {
	PID       int    `json:"pid"`
	Port      int    `json:"port"`
	URL       string `json:"url"`
	Mode      string `json:"mode"` // "ui" or "server"
	StartedAt string `json:"started_at"`
}

func mcpLockPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "upmark", "mcp.lock")
}

// readMCPLock returns the on-disk lockfile or nil if missing/unparseable.
func readMCPLock() *mcpLockFile {
	data, err := os.ReadFile(mcpLockPath())
	if err != nil {
		return nil
	}
	var lf mcpLockFile
	if err := json.Unmarshal(data, &lf); err != nil {
		return nil
	}
	return &lf
}

// writeMCPLock records the current process as owner of the MCP server.
func writeMCPLock(port int, mode string) error {
	lf := mcpLockFile{
		PID:       os.Getpid(),
		Port:      port,
		URL:       sseURL(port),
		Mode:      mode,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(&lf, "", "  ")
	if err != nil {
		return err
	}
	path := mcpLockPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// clearMCPLock removes the lockfile. Safe to call when the file is missing
// or owned by a different process — we only clear if it's ours.
func clearMCPLock() {
	lf := readMCPLock()
	if lf == nil {
		return
	}
	if lf.PID != os.Getpid() {
		// Not ours — leave it; the owning process will clean up.
		return
	}
	_ = os.Remove(mcpLockPath())
}

// liveMCPLock returns the lockfile only if the recorded URL actually answers
// an HTTP GET. Stale entries (process died, file left behind) return nil so
// the caller can claim the port. Quick timeout — this is a fast-path check.
func liveMCPLock() *mcpLockFile {
	lf := readMCPLock()
	if lf == nil || lf.URL == "" {
		return nil
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, err := http.NewRequest(http.MethodGet, lf.URL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	// Any response (200 SSE stream open, or 4xx/5xx) means a server is
	// answering on that port. Treat anything as "alive."
	return lf
}

func sseURL(port int) string {
	return "http://127.0.0.1:" + itoa(port) + "/sse"
}
