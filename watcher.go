package main

import (
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// fileWatcher watches a single file for changes (including atomic-save replace)
// by watching the parent directory and filtering by filename.
type fileWatcher struct {
	path    string
	onEvent func()

	w       *fsnotify.Watcher
	stop    chan struct{}
	mu      sync.Mutex
	stopped bool
}

func newFileWatcher(path string, onEvent func()) *fileWatcher {
	return &fileWatcher{
		path:    path,
		onEvent: onEvent,
		stop:    make(chan struct{}),
	}
}

func (f *fileWatcher) Start() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	f.w = w
	dir := filepath.Dir(f.path)
	if err := w.Add(dir); err != nil {
		w.Close()
		return err
	}
	go f.loop()
	return nil
}

func (f *fileWatcher) loop() {
	target := filepath.Clean(f.path)
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	pending := false
	for {
		select {
		case <-f.stop:
			return
		case ev, ok := <-f.w.Events:
			if !ok {
				return
			}
			if filepath.Clean(ev.Name) != target {
				continue
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}
			// Some editors do remove+create on save; if our watched name reappears,
			// fsnotify already tracks it because we watch the parent directory.
			pending = true
			debounce.Reset(120 * time.Millisecond)
		case <-debounce.C:
			if pending {
				pending = false
				f.onEvent()
			}
		case <-f.w.Errors:
			// swallow errors silently; live reload is best-effort
		}
	}
}

func (f *fileWatcher) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopped {
		return
	}
	f.stopped = true
	close(f.stop)
	if f.w != nil {
		_ = f.w.Close()
	}
}
