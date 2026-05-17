package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Document is what the frontend gets back after opening a file.
type Document struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	HTML     string `json:"html"`
	Source   string `json:"source"` // raw markdown text, for the editor
	BaseDir  string `json:"baseDir"`
	Modified int64  `json:"modified"`
	// ReadOnly hides editor/save controls in the frontend. Set for the
	// embedded welcome doc and (eventually) MCP-presented documents.
	ReadOnly bool `json:"readOnly,omitempty"`
}

// App is the bound struct exposed to JS.
type App struct {
	ctx context.Context

	mu          sync.Mutex
	currentPath string
	currentDir  string
	watcher     *fileWatcher
	prefs       *Prefs
	renderer    *renderer

	// File path supplied via CLI arg / OS file-association double-click.
	// Read on startup and consumed once.
	startupFile string

	// Suppress fsnotify events for a brief window after we write to a file
	// ourselves, so the editor's auto-save doesn't trigger a re-render loop.
	selfWrites map[string]time.Time

	mcp *MCPManager
}

// SetStartupFile is called from main.go after parsing os.Args.
func (a *App) SetStartupFile(p string) {
	a.startupFile = p
}

// ConsumeStartupFile returns the startup file (if any) and clears it so the
// frontend only acts on it once.
func (a *App) ConsumeStartupFile() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	p := a.startupFile
	a.startupFile = ""
	return p
}

func NewApp() *App {
	r, err := newRenderer()
	if err != nil {
		panic(fmt.Errorf("renderer init: %w", err))
	}
	a := &App{
		prefs:      loadPrefs(),
		renderer:   r,
		selfWrites: make(map[string]time.Time),
	}
	a.mcp = newMCPManager(a)
	return a
}

// markSelfWrite records that we wrote to this path so the file watcher can
// ignore the resulting fsnotify event.
func (a *App) markSelfWrite(path string) {
	a.mu.Lock()
	a.selfWrites[path] = time.Now()
	a.mu.Unlock()
}

// isRecentSelfWrite is consulted by the watcher before re-rendering.
func (a *App) isRecentSelfWrite(path string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	t, ok := a.selfWrites[path]
	if !ok {
		return false
	}
	if time.Since(t) > 750*time.Millisecond {
		delete(a.selfWrites, path)
		return false
	}
	return true
}

// SaveDocument writes the given content to the given path. The fsnotify
// watcher will ignore the resulting event so we don't re-render from disk
// (the caller already has the buffer).
func (a *App) SaveDocument(path string, content string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	a.markSelfWrite(path)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return nil
}

// RenderMarkdown renders an in-memory string (no disk read). Used for live
// preview while the user types in the editor. baseDir is the directory of
// the document being edited, used to resolve relative images.
func (a *App) RenderMarkdown(content string, baseDir string) (string, error) {
	if baseDir == "" {
		baseDir = a.currentDir
	}
	return a.renderer.render([]byte(content), baseDir)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Restore window size/position.
	if a.prefs.WindowWidth > 0 && a.prefs.WindowHeight > 0 {
		wailsruntime.WindowSetSize(ctx, a.prefs.WindowWidth, a.prefs.WindowHeight)
	}
	if a.prefs.WindowX != 0 || a.prefs.WindowY != 0 {
		wailsruntime.WindowSetPosition(ctx, a.prefs.WindowX, a.prefs.WindowY)
	}

	// Start the MCP server if enabled.
	if a.prefs.MCPEnabled {
		port := a.prefs.MCPPort
		if port <= 0 {
			port = defaultMCPPort
		}
		if err := a.mcp.Start(port); err != nil {
			fmt.Println("mcp start:", err)
		}
	}
}

func (a *App) domReady(ctx context.Context) {
	// Frontend pulls the startup-or-last file via GetStartupOrLastFile() and
	// opens it through its normal flow, so the doc state actually surfaces in
	// the UI. No work here.
}

// GetStartupOrLastFile returns the path the frontend should open on launch.
// Priority: CLI-arg / file-association startup file > previously open file.
// The startup file is consumed (cleared) on read.
func (a *App) GetStartupOrLastFile() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.startupFile != "" {
		p := a.startupFile
		a.startupFile = ""
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if a.prefs.LastFile != "" {
		if _, err := os.Stat(a.prefs.LastFile); err == nil {
			return a.prefs.LastFile
		}
	}
	return ""
}

func (a *App) shutdown(ctx context.Context) {
	width, height := wailsruntime.WindowGetSize(ctx)
	x, y := wailsruntime.WindowGetPosition(ctx)
	a.prefs.WindowWidth = width
	a.prefs.WindowHeight = height
	a.prefs.WindowX = x
	a.prefs.WindowY = y
	_ = a.prefs.save()

	a.mu.Lock()
	if a.watcher != nil {
		a.watcher.Stop()
		a.watcher = nil
	}
	a.mu.Unlock()

	if a.mcp != nil {
		_ = a.mcp.Stop()
	}
}

// OpenDialog shows the native file picker and opens whatever the user selects.
func (a *App) OpenDialog() (*Document, error) {
	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Open Markdown File",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Markdown", Pattern: "*.md;*.markdown;*.mdown;*.mkd;*.mdx"},
			{DisplayName: "All Files", Pattern: "*.*"},
		},
	})
	if err != nil || path == "" {
		return nil, err
	}
	return a.OpenPath(path)
}

// OpenPath loads the given file, renders it, starts watching, and returns the doc.
func (a *App) OpenPath(path string) (*Document, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(abs)
	html, err := a.renderer.render(data, baseDir)
	if err != nil {
		return nil, err
	}

	doc := &Document{
		Path:     abs,
		Name:     filepath.Base(abs),
		HTML:     html,
		Source:   string(data),
		BaseDir:  baseDir,
		Modified: info.ModTime().UnixMilli(),
	}

	a.mu.Lock()
	a.currentPath = abs
	a.currentDir = baseDir
	if a.watcher != nil {
		a.watcher.Stop()
	}
	a.watcher = newFileWatcher(abs, func() {
		a.reloadCurrent()
	})
	if err := a.watcher.Start(); err != nil {
		// non-fatal: just no live reload
		fmt.Println("watcher start:", err)
	}
	a.mu.Unlock()

	a.prefs.LastFile = abs
	a.prefs.addRecent(abs)
	_ = a.prefs.save()

	return doc, nil
}

func (a *App) reloadCurrent() {
	a.mu.Lock()
	path := a.currentPath
	a.mu.Unlock()
	if path == "" {
		return
	}
	// Skip the reload if this fsnotify event came from our own save.
	if a.isRecentSelfWrite(path) {
		return
	}
	doc, err := a.renderPath(path)
	if err != nil {
		wailsruntime.EventsEmit(a.ctx, "file-error", err.Error())
		return
	}
	wailsruntime.EventsEmit(a.ctx, "file-changed", doc)
}

func (a *App) renderPath(path string) (*Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(path)
	html, err := a.renderer.render(data, baseDir)
	if err != nil {
		return nil, err
	}
	return &Document{
		Path:     path,
		Name:     filepath.Base(path),
		HTML:     html,
		Source:   string(data),
		BaseDir:  baseDir,
		Modified: info.ModTime().UnixMilli(),
	}, nil
}

// CloseDocument stops watching and clears state.
func (a *App) CloseDocument() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.watcher != nil {
		a.watcher.Stop()
		a.watcher = nil
	}
	a.currentPath = ""
	a.currentDir = ""
	a.prefs.LastFile = ""
	_ = a.prefs.save()
}

// RecentFiles returns up to N most recently opened files that still exist.
func (a *App) RecentFiles() []RecentEntry {
	out := make([]RecentEntry, 0, len(a.prefs.Recent))
	for _, r := range a.prefs.Recent {
		if _, err := os.Stat(r.Path); err == nil {
			out = append(out, r)
		}
	}
	return out
}

// ClearRecent wipes the recent-files list.
func (a *App) ClearRecent() {
	a.prefs.Recent = nil
	_ = a.prefs.save()
}

// ChromaCSS returns syntax-highlighting stylesheets for both color schemes.
// The frontend injects them once at startup.
type ChromaCSS struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

func (a *App) GetChromaCSS() ChromaCSS {
	light, dark := a.renderer.chromaCSS()
	return ChromaCSS{Light: light, Dark: dark}
}

// UIPrefs is the read-side bundle for the frontend.
type UIPrefs struct {
	ReadingWidth string `json:"readingWidth"`
	FontSize     int    `json:"fontSize"`
	Theme        string `json:"theme"`
}

func (a *App) GetUIPrefs() UIPrefs {
	rw := a.prefs.ReadingWidth
	if rw == "" {
		rw = "normal"
	}
	fs := a.prefs.FontSize
	if fs < 12 || fs > 26 {
		fs = 17
	}
	th := a.prefs.Theme
	if th == "" {
		th = "editorial"
	}
	return UIPrefs{ReadingWidth: rw, FontSize: fs, Theme: th}
}

// SetTheme persists the chosen theme name.
var validThemes = map[string]bool{
	"editorial": true, "broadsheet": true, "terminal": true,
	"manuscript": true, "brutalist": true, "arcade": true,
	"pastoral": true, "architect": true, "vapor": true,
	"typewriter": true, "midnight": true, "gameboy": true,
	"newsprint": true,
}

func (a *App) SetTheme(name string) {
	if validThemes[name] {
		a.prefs.Theme = name
		_ = a.prefs.save()
	}
}

// SetReadingWidth persists the reading column width preset.
func (a *App) SetReadingWidth(width string) {
	switch width {
	case "narrow", "normal", "wide":
		a.prefs.ReadingWidth = width
		_ = a.prefs.save()
	}
}

// SetFontSize persists the reading body font size in px (12-26).
func (a *App) SetFontSize(px int) {
	if px < 12 {
		px = 12
	}
	if px > 26 {
		px = 26
	}
	a.prefs.FontSize = px
	_ = a.prefs.save()
}

// SetScrollPos remembers where the user was in a given file (0.0-1.0).
func (a *App) SetScrollPos(path string, pct float64) {
	if path == "" {
		return
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	a.prefs.mu.Lock()
	if a.prefs.ScrollByFile == nil {
		a.prefs.ScrollByFile = make(map[string]float64)
	}
	a.prefs.ScrollByFile[path] = pct
	// Cap the map at 100 entries so it doesn't grow unbounded.
	if len(a.prefs.ScrollByFile) > 100 {
		// Drop a random entry - not perfect LRU, but bounded.
		for k := range a.prefs.ScrollByFile {
			if k != path {
				delete(a.prefs.ScrollByFile, k)
				break
			}
		}
	}
	a.prefs.mu.Unlock()
	_ = a.prefs.save()
}

// GetScrollPos returns the saved scroll position for a file, or 0.
func (a *App) GetScrollPos(path string) float64 {
	a.prefs.mu.Lock()
	defer a.prefs.mu.Unlock()
	if a.prefs.ScrollByFile == nil {
		return 0
	}
	return a.prefs.ScrollByFile[path]
}

// Window controls for the frameless title bar.
func (a *App) MinimizeWindow()       { wailsruntime.WindowMinimise(a.ctx) }
func (a *App) ToggleMaximizeWindow() { wailsruntime.WindowToggleMaximise(a.ctx) }
func (a *App) CloseWindow()          { wailsruntime.Quit(a.ctx) }
func (a *App) IsMaximized() bool     { return wailsruntime.WindowIsMaximised(a.ctx) }

// ───── MCP server control & status (frontend-bound) ─────

func (a *App) GetMCPStatus() MCPStatus {
	port := a.prefs.MCPPort
	if port <= 0 {
		port = defaultMCPPort
	}
	a.mcp.mu.RLock()
	running := a.mcp.running
	a.mcp.mu.RUnlock()
	return MCPStatus{
		Enabled: a.prefs.MCPEnabled,
		Running: running,
		Port:    port,
		URL:     fmt.Sprintf("http://127.0.0.1:%d/sse", port),
	}
}

func (a *App) SetMCPEnabled(enabled bool) error {
	a.prefs.MCPEnabled = enabled
	_ = a.prefs.save()
	if enabled {
		port := a.prefs.MCPPort
		if port <= 0 {
			port = defaultMCPPort
		}
		return a.mcp.Start(port)
	}
	return a.mcp.Stop()
}

func (a *App) SetMCPPort(port int) error {
	if port <= 0 || port > 65535 {
		return fmt.Errorf("port out of range")
	}
	a.prefs.MCPPort = port
	_ = a.prefs.save()
	// If running, restart on the new port.
	a.mcp.mu.RLock()
	running := a.mcp.running
	a.mcp.mu.RUnlock()
	if running {
		_ = a.mcp.Stop()
		return a.mcp.Start(port)
	}
	return nil
}

// MCPSetTaskChecked is called from JS when the user toggles an interactive
// task checkbox in an MCP-presented doc.
func (a *App) MCPSetTaskChecked(docID string, taskID int, checked bool) {
	a.mcp.SetTaskChecked(docID, taskID, checked)
}

// MCPSetViewState reports whether the user has the doc in view / has closed it.
func (a *App) MCPSetViewState(docID string, viewed, closed bool) {
	a.mcp.SetViewState(docID, viewed, closed)
}

// RevealInExplorer opens the OS file manager with the given file selected.
// On Windows: `explorer /select,<path>`.
func (a *App) RevealInExplorer(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	return revealInExplorer(path)
}

// OpenContainingFolder opens the OS file manager on the directory of `path`.
func (a *App) OpenContainingFolder(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	return openFolder(filepath.Dir(path))
}
