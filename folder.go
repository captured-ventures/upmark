package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FolderEntry represents one node in the folder tree shown in the sidebar.
type FolderEntry struct {
	Name     string        `json:"name"`
	Path     string        `json:"path"`
	IsDir    bool          `json:"isDir"`
	Children []FolderEntry `json:"children,omitempty"`
}

// Folder is the result of OpenFolder — a name + a tree of markdown files.
type Folder struct {
	Root    string        `json:"root"`
	Name    string        `json:"name"`
	Entries []FolderEntry `json:"entries"`
}

var markdownExts = map[string]bool{
	".md": true, ".markdown": true, ".mdown": true, ".mkd": true, ".mdx": true,
}

func isMarkdown(name string) bool {
	return markdownExts[strings.ToLower(filepath.Ext(name))]
}

// OpenFolderDialog shows a native directory picker.
func (a *App) OpenFolderDialog() (*Folder, error) {
	dir, err := openDirectoryDialog(a)
	if err != nil || dir == "" {
		return nil, err
	}
	return a.OpenFolder(dir)
}

// OpenFolder scans a directory recursively (max depth) and returns its
// markdown files as a tree. Hidden directories and node_modules are skipped.
func (a *App) OpenFolder(path string) (*Folder, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	entries, err := scanDir(abs, 0)
	if err != nil {
		return nil, err
	}
	a.prefs.LastFolder = abs
	_ = a.prefs.save()
	return &Folder{
		Root:    abs,
		Name:    filepath.Base(abs),
		Entries: entries,
	}, nil
}

// CloseFolder clears the persisted folder.
func (a *App) CloseFolder() {
	a.prefs.LastFolder = ""
	_ = a.prefs.save()
}

// LastFolder reopens the previously-open folder (if any). Called by the
// frontend on startup so the sidebar can be populated immediately.
func (a *App) LastFolder() *Folder {
	if a.prefs.LastFolder == "" {
		return nil
	}
	if _, err := os.Stat(a.prefs.LastFolder); err != nil {
		return nil
	}
	f, err := a.OpenFolder(a.prefs.LastFolder)
	if err != nil {
		return nil
	}
	return f
}

const maxFolderDepth = 5

func scanDir(path string, depth int) ([]FolderEntry, error) {
	if depth > maxFolderDepth {
		return nil, nil
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs, files []FolderEntry
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if item.IsDir() {
			if name == "node_modules" || name == "vendor" || name == "dist" || name == "build" {
				continue
			}
			full := filepath.Join(path, name)
			kids, err := scanDir(full, depth+1)
			if err != nil {
				continue
			}
			if len(kids) == 0 {
				continue
			}
			dirs = append(dirs, FolderEntry{
				Name:     name,
				Path:     full,
				IsDir:    true,
				Children: kids,
			})
			continue
		}
		if !isMarkdown(name) {
			continue
		}
		files = append(files, FolderEntry{
			Name:  name,
			Path:  filepath.Join(path, name),
			IsDir: false,
		})
	}
	sort.Slice(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name) })
	return append(dirs, files...), nil
}

// ResolveWikilink looks up a wikilink target by basename (case-insensitive).
// Searches the current folder first, then the document's directory.
func (a *App) ResolveWikilink(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	// Strip extension and any anchor.
	if i := strings.IndexAny(target, "#|"); i >= 0 {
		target = target[:i]
	}
	target = strings.TrimSuffix(target, filepath.Ext(target))
	wantLower := strings.ToLower(target)

	roots := []string{}
	if a.prefs.LastFolder != "" {
		roots = append(roots, a.prefs.LastFolder)
	}
	a.mu.Lock()
	if a.currentDir != "" && (len(roots) == 0 || a.currentDir != roots[0]) {
		roots = append(roots, a.currentDir)
	}
	a.mu.Unlock()

	for _, root := range roots {
		var found string
		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil || found != "" {
				return nil
			}
			if d.IsDir() {
				n := d.Name()
				if strings.HasPrefix(n, ".") || n == "node_modules" || n == "vendor" || n == "dist" || n == "build" {
					return filepath.SkipDir
				}
				return nil
			}
			if !isMarkdown(d.Name()) {
				return nil
			}
			base := strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))
			if strings.ToLower(base) == wantLower {
				found = p
			}
			return nil
		})
		if found != "" {
			return found
		}
	}
	return ""
}
