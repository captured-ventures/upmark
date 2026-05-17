package main

import (
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// newAssetHandler returns an http.Handler that serves local files from the
// /local-asset/<urlencoded baseDir>/<urlencoded relPath> route. This lets the
// rendered HTML reference images that live next to the opened markdown file.
func newAssetHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if !strings.HasPrefix(p, "/local-asset/") {
			http.NotFound(w, r)
			return
		}
		rest := strings.TrimPrefix(p, "/local-asset/")
		slash := strings.Index(rest, "/")
		if slash < 0 {
			http.NotFound(w, r)
			return
		}
		baseEnc, relEnc := rest[:slash], rest[slash+1:]
		base, err := url.PathUnescape(baseEnc)
		if err != nil {
			http.Error(w, "bad base", http.StatusBadRequest)
			return
		}
		rel, err := url.PathUnescape(relEnc)
		if err != nil {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		// Strip any anchor fragment that snuck in.
		if i := strings.IndexAny(rel, "#?"); i >= 0 {
			rel = rel[:i]
		}

		// Normalize and confine: the resolved path must remain inside base.
		joined := filepath.Join(base, rel)
		absBase, err1 := filepath.Abs(base)
		absJoined, err2 := filepath.Abs(joined)
		if err1 != nil || err2 != nil {
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		if !strings.HasPrefix(absJoined+string(filepath.Separator), absBase+string(filepath.Separator)) && absJoined != absBase {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		f, err := os.Open(absJoined)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeContent(w, r, filepath.Base(absJoined), stat.ModTime(), f)
	})
}

func timeNowMs() int64 { return time.Now().UnixMilli() }
