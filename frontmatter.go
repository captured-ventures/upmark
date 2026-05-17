package main

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// frontmatterFullRE matches both YAML (---) and TOML (+++) frontmatter blocks
// at the very start of the document, with the inner content captured.
var frontmatterFullRE = regexp.MustCompile(`(?s)\A(---|\+\+\+)\r?\n(.*?)\r?\n(---|\+\+\+)\r?\n`)

// parseFrontmatter pulls a leading YAML/TOML block off the source and returns
// (parsed map, remaining source). If no frontmatter is present or parsing
// fails, the map is nil and the source is returned unchanged.
func parseFrontmatter(src []byte) (map[string]any, []byte) {
	m := frontmatterFullRE.FindSubmatch(src)
	if m == nil {
		return nil, src
	}
	openDelim := m[1]
	inner := m[2]
	rest := src[len(m[0]):]

	matter := make(map[string]any)
	switch string(openDelim) {
	case "---":
		_ = yaml.Unmarshal(inner, &matter)
	case "+++":
		_ = toml.Unmarshal(inner, &matter)
	}
	if len(matter) == 0 {
		return nil, rest
	}
	return matter, rest
}

// Recognized field aliases — first non-empty wins.
var (
	titleKeys    = []string{"title"}
	subtitleKeys = []string{"subtitle", "description", "standfirst", "summary"}
	authorKeys   = []string{"author", "by", "byline", "authors"}
	dateKeys     = []string{"date", "published", "publishedAt", "created", "createdAt"}
	tagsKeys     = []string{"tags", "keywords", "categories"}
)

// known is the union of every alias above plus "draft", used to filter the
// "remaining custom fields" definition list.
var known = func() map[string]bool {
	m := map[string]bool{"draft": true}
	for _, ks := range [][]string{titleKeys, subtitleKeys, authorKeys, dateKeys, tagsKeys} {
		for _, k := range ks {
			m[k] = true
		}
	}
	return m
}()

func strField(matter map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := matter[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

// dateField is strField's date-aware cousin. yaml.v3 auto-parses ISO dates as
// time.Time values, so we need to handle both string and time inputs.
func dateField(matter map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := matter[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			if t != "" {
				return formatDate(t)
			}
		case time.Time:
			return t.Format("January 2, 2006")
		}
	}
	return ""
}

func listField(matter map[string]any, keys ...string) []string {
	for _, k := range keys {
		v, ok := matter[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case string:
			if t == "" {
				continue
			}
			return splitCSV(t)
		case []any:
			out := make([]string, 0, len(t))
			for _, item := range t {
				if s, ok := item.(string); ok && s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				return out
			}
		case []string:
			return t
		}
	}
	return nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// formatDate tries a few common date layouts; falls back to the raw string.
func formatDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02 15:04:05",
		time.RFC3339,
		time.RFC1123,
		"January 2, 2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.Format("January 2, 2006")
		}
	}
	return s
}

// formatScalar renders a YAML/TOML value as a plain string for the custom-
// fields definition list. Maps and arrays are JSON-ish for visibility.
func formatScalar(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64, float32, int, int64, int32:
		return fmt.Sprintf("%v", t)
	case time.Time:
		return t.Format("January 2, 2006")
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			parts[i] = formatScalar(x)
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// bylineResult is returned by renderByline so the caller can choose where to
// place the byline. When the frontmatter title was deduped against the doc's
// first H1, the byline should appear *after* the H1 (as a standfirst/byline
// block under the headline). Otherwise it goes above the body.
type bylineResult struct {
	HTML         string
	TitleDeduped bool
}

// renderByline builds the HTML byline block from frontmatter. If the
// frontmatter title matches firstH1 (case-insensitive trim), the title is
// dropped from the byline and TitleDeduped is set so the caller can position
// the remaining metadata under the doc's H1.
func renderByline(matter map[string]any, firstH1 string) bylineResult {
	if len(matter) == 0 {
		return bylineResult{}
	}
	var inner strings.Builder
	titleDeduped := false

	if t := strField(matter, titleKeys...); t != "" {
		if titlesMatch(t, firstH1) {
			titleDeduped = true
		} else {
			inner.WriteString(`<h1 class="fm-title">` + html.EscapeString(t) + `</h1>`)
		}
	}
	if s := strField(matter, subtitleKeys...); s != "" {
		inner.WriteString(`<p class="fm-subtitle">` + html.EscapeString(s) + `</p>`)
	}

	var meta []string
	if a := strField(matter, authorKeys...); a != "" {
		meta = append(meta, `<span class="fm-author">by `+html.EscapeString(a)+`</span>`)
	}
	if d := dateField(matter, dateKeys...); d != "" {
		meta = append(meta, `<time class="fm-date">`+html.EscapeString(d)+`</time>`)
	}
	if draft, ok := matter["draft"]; ok {
		if b2, ok := draft.(bool); ok && b2 {
			meta = append(meta, `<span class="fm-draft">draft</span>`)
		}
	}
	if len(meta) > 0 {
		inner.WriteString(`<div class="fm-meta">`)
		inner.WriteString(strings.Join(meta, ` <span class="fm-sep">·</span> `))
		inner.WriteString(`</div>`)
	}

	if tags := listField(matter, tagsKeys...); len(tags) > 0 {
		inner.WriteString(`<ul class="fm-tags">`)
		for _, t := range tags {
			inner.WriteString(`<li class="fm-tag">` + html.EscapeString(t) + `</li>`)
		}
		inner.WriteString(`</ul>`)
	}

	var extraKeys []string
	for k := range matter {
		if !known[k] {
			extraKeys = append(extraKeys, k)
		}
	}
	if len(extraKeys) > 0 {
		sortStrings(extraKeys)
		inner.WriteString(`<dl class="fm-fields">`)
		for _, k := range extraKeys {
			inner.WriteString(`<dt>` + html.EscapeString(k) + `</dt>`)
			inner.WriteString(`<dd>` + html.EscapeString(formatScalar(matter[k])) + `</dd>`)
		}
		inner.WriteString(`</dl>`)
	}

	if inner.Len() == 0 {
		// Title was deduped and there's no other metadata to surface — nothing
		// to render. Still report the dedup so callers know.
		return bylineResult{TitleDeduped: titleDeduped}
	}
	cls := "frontmatter"
	if titleDeduped {
		cls += " frontmatter-under"
	}
	return bylineResult{
		HTML:         `<header class="` + cls + `">` + inner.String() + `</header>`,
		TitleDeduped: titleDeduped,
	}
}

// titlesMatch is the dedup rule: case-insensitive equality after trimming.
// Either being empty disqualifies (no match).
func titlesMatch(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// findFirstH1 returns the text of the first ATX-style `# heading` outside any
// fenced code block. Returns "" if none found. Setext (underline) headings
// aren't considered — ATX is the convention for the dedup case anyway.
func findFirstH1(src []byte) string {
	inFence := false
	for _, line := range bytes.Split(src, []byte("\n")) {
		line = bytes.TrimRight(line, "\r")
		trimmed := bytes.TrimSpace(line)
		if bytes.HasPrefix(trimmed, []byte("```")) || bytes.HasPrefix(trimmed, []byte("~~~")) {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		if bytes.HasPrefix(line, []byte("# ")) {
			return strings.TrimSpace(string(line[2:]))
		}
	}
	return ""
}

// sort wrapper kept private to avoid pulling in sort in the import group
// twice (already used elsewhere). Linear bubble is fine for ~10 keys.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
