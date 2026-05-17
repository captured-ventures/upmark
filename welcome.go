package main

import (
	_ "embed"
)

//go:embed assets/welcome.md
var welcomeMarkdown string

// welcomePath is the synthetic path assigned to the embedded welcome doc.
// It's not a real file; the frontend uses the prefix to skip save / edit /
// reveal-in-explorer operations.
const welcomePath = "upmark://welcome"

// IsFirstLaunch reports whether the welcome doc should auto-open. It returns
// true exactly once per machine — the first call flips the persisted flag
// so subsequent launches stay quiet.
func (a *App) IsFirstLaunch() bool {
	if a.prefs.WelcomeSeen {
		return false
	}
	a.prefs.WelcomeSeen = true
	_ = a.prefs.save()
	return true
}

// OpenWelcome renders the embedded welcome.md and returns it as a Document.
// No disk path, no file watcher. The ReadOnly flag is set so the frontend
// hides editor/save controls.
func (a *App) OpenWelcome() (*Document, error) {
	html, err := a.renderer.render([]byte(welcomeMarkdown), "")
	if err != nil {
		return nil, err
	}
	return &Document{
		Path:     welcomePath,
		Name:     "welcome",
		HTML:     html,
		Source:   welcomeMarkdown,
		BaseDir:  "",
		Modified: 0,
		ReadOnly: true,
	}, nil
}
