package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

const (
	urlFetchMaxBytes = 8 * 1024 * 1024 // 8 MB ceiling; refuse anything bigger
	urlFetchTimeout  = 30 * time.Second
)

// urlOpenClient is reused so DNS / keep-alive isn't re-established on every fetch.
var urlOpenClient = &http.Client{Timeout: urlFetchTimeout}

// OpenURL fetches a remote markdown file and renders it as a Document. The
// document's Path is the URL itself (no on-disk file), so save/edit features
// are inert for these — the renderer treats it as read-only.
func (a *App) OpenURL(rawURL string) (*Document, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, errors.New("empty URL")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q (only http/https)", u.Scheme)
	}

	// Common convenience: rewrite a github.com/user/repo/blob/... link to its
	// raw.githubusercontent.com equivalent so users can paste the URL they see
	// in the browser address bar.
	rawURL = canonicalizeGithubURL(rawURL)
	u, _ = url.Parse(rawURL)

	req, err := http.NewRequestWithContext(a.ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "upmark/0.7 (markdown reader)")
	req.Header.Set("Accept", "text/markdown,text/plain,*/*;q=0.8")

	resp, err := urlOpenClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, urlFetchMaxBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	name := filepath.Base(u.Path)
	if name == "" || name == "/" || name == "." {
		name = u.Host
	}

	// baseRef = the URL itself so relative image/link refs resolve via
	// url.ResolveReference (this lives in renderer.rewriteAssetURLs).
	html, err := a.renderer.render(data, rawURL)
	if err != nil {
		return nil, err
	}

	return &Document{
		Path:     rawURL,
		Name:     name,
		HTML:     html,
		Source:   string(data),
		BaseDir:  rawURL,
		Modified: time.Now().UnixMilli(),
	}, nil
}

// canonicalizeGithubURL turns a github.com/user/repo/blob/branch/path link
// into its raw.githubusercontent.com equivalent, which is what we actually
// want to fetch.
func canonicalizeGithubURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host != "github.com" {
		return rawURL
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	// /user/repo/blob/branch/path/to/file.md  →  parts: [user repo blob branch path... file.md]
	if len(parts) < 5 || parts[2] != "blob" {
		return rawURL
	}
	u.Host = "raw.githubusercontent.com"
	u.Path = "/" + parts[0] + "/" + parts[1] + "/" + parts[3] + "/" + strings.Join(parts[4:], "/")
	u.RawQuery = ""
	return u.String()
}
