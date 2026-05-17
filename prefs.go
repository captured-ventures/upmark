package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

const recentMax = 10

type RecentEntry struct {
	Path     string `json:"path"`
	Name     string `json:"name"`
	OpenedAt int64  `json:"openedAt"`
}

type Prefs struct {
	WindowWidth  int                `json:"windowWidth,omitempty"`
	WindowHeight int                `json:"windowHeight,omitempty"`
	WindowX      int                `json:"windowX,omitempty"`
	WindowY      int                `json:"windowY,omitempty"`
	LastFile     string             `json:"lastFile,omitempty"`
	LastFolder   string             `json:"lastFolder,omitempty"`
	Recent       []RecentEntry      `json:"recent,omitempty"`
	SidebarOpen  *bool              `json:"sidebarOpen,omitempty"`
	ScrollByFile map[string]float64 `json:"scrollByFile,omitempty"`

	// Read polish settings
	ReadingWidth string `json:"readingWidth,omitempty"` // "narrow" | "normal" | "wide"
	FontSize     int    `json:"fontSize,omitempty"`     // px, 14-22
	Theme        string `json:"theme,omitempty"`        // "editorial" | "broadsheet" | "terminal"

	// MCP server settings — off by default for privacy
	MCPEnabled bool `json:"mcpEnabled,omitempty"`
	MCPPort    int  `json:"mcpPort,omitempty"`

	// WelcomeSeen flips to true the first time the welcome doc opens, so
	// subsequent launches don't shove it in front of the user again.
	WelcomeSeen bool `json:"welcomeSeen,omitempty"`

	mu   sync.Mutex
	path string
}

func prefsPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		// fallback to working dir
		dir = "."
	}
	return filepath.Join(dir, "upmark", "prefs.json")
}

func loadPrefs() *Prefs {
	p := &Prefs{path: prefsPath()}
	data, err := os.ReadFile(p.path)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(data, p)
	return p
}

func (p *Prefs) save() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.path, data, 0o644)
}

func (p *Prefs) addRecent(absPath string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	name := filepath.Base(absPath)
	now := nowMs()

	// Remove any existing entry for this path.
	filtered := p.Recent[:0]
	for _, r := range p.Recent {
		if r.Path != absPath {
			filtered = append(filtered, r)
		}
	}
	// Prepend.
	entry := RecentEntry{Path: absPath, Name: name, OpenedAt: now}
	p.Recent = append([]RecentEntry{entry}, filtered...)
	if len(p.Recent) > recentMax {
		p.Recent = p.Recent[:recentMax]
	}
}

func nowMs() int64 {
	return timeNowMs()
}
