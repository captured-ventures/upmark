package main

import (
	"strings"
	"testing"
)

// parseFrontmatter handles both YAML and TOML, returns (matter, rest).
// When no frontmatter is present, rest equals the input and matter is nil.
func TestParseFrontmatter_YAML(t *testing.T) {
	src := []byte("---\ntitle: hello\ntags:\n  - one\n  - two\n---\n# heading\nbody\n")
	matter, rest := parseFrontmatter(src)
	if matter == nil {
		t.Fatalf("expected matter, got nil")
	}
	if got := matter["title"]; got != "hello" {
		t.Errorf("title = %v, want hello", got)
	}
	if !strings.HasPrefix(string(rest), "# heading") {
		t.Errorf("rest should start with '# heading', got %q", string(rest)[:20])
	}
}

func TestParseFrontmatter_TOML(t *testing.T) {
	src := []byte("+++\ntitle = \"hello\"\nauthor = \"Brad\"\n+++\ncontent\n")
	matter, rest := parseFrontmatter(src)
	if matter == nil {
		t.Fatalf("expected matter, got nil")
	}
	if got := matter["title"]; got != "hello" {
		t.Errorf("title = %v, want hello", got)
	}
	if got := matter["author"]; got != "Brad" {
		t.Errorf("author = %v, want Brad", got)
	}
	if string(rest) != "content\n" {
		t.Errorf("rest = %q, want 'content\\n'", string(rest))
	}
}

func TestParseFrontmatter_None(t *testing.T) {
	src := []byte("# just a heading\nbody\n")
	matter, rest := parseFrontmatter(src)
	if matter != nil {
		t.Errorf("expected nil matter, got %v", matter)
	}
	if string(rest) != string(src) {
		t.Errorf("rest should equal input when no frontmatter present")
	}
}

func TestParseFrontmatter_EmptyBody(t *testing.T) {
	// An empty YAML body should yield nil matter (parsed map is empty).
	src := []byte("---\n\n---\nbody\n")
	matter, rest := parseFrontmatter(src)
	if matter != nil {
		t.Errorf("empty frontmatter should produce nil matter, got %v", matter)
	}
	// Even with empty matter, the block should be stripped from rest.
	if !strings.Contains(string(rest), "body") {
		t.Errorf("body should still be in rest")
	}
}

func TestParseFrontmatter_CRLFLineEndings(t *testing.T) {
	src := []byte("---\r\ntitle: windows\r\n---\r\nbody\r\n")
	matter, rest := parseFrontmatter(src)
	if matter == nil {
		t.Fatalf("CRLF frontmatter should parse")
	}
	if got := matter["title"]; got != "windows" {
		t.Errorf("title = %v, want windows", got)
	}
	if !strings.Contains(string(rest), "body") {
		t.Errorf("body should be in rest")
	}
}

// findFirstH1 finds the first ATX-style # heading outside of fenced code.
func TestFindFirstH1(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"basic", "# Hello world\nbody", "Hello world"},
		{"first wins", "# First\n# Second", "First"},
		{"trailing whitespace", "#   Trim me   \n", "Trim me"},
		{"setext not matched", "Heading\n=======\n", ""},
		{"h2 not matched", "## not h1\n# real h1", "real h1"},
		{"inside fence ignored", "```\n# in code\n```\n# real\n", "real"},
		{"inside tilde fence", "~~~\n# in code\n~~~\n# real\n", "real"},
		{"no heading", "just text\n", ""},
		{"crlf", "# Win line\r\nbody", "Win line"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := findFirstH1([]byte(c.src))
			if got != c.want {
				t.Errorf("findFirstH1(%q) = %q, want %q", c.src, got, c.want)
			}
		})
	}
}

// titlesMatch is the dedup rule for frontmatter title vs first H1.
func TestTitlesMatch(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"Hello", "hello", true},
		{"  Hello  ", "Hello", true},
		{"Hello", "Hello world", false},
		{"", "anything", false},
		{"anything", "", false},
		{"", "", false},
		{"UPMARK", "upmark", true},
	}
	for _, c := range cases {
		got := titlesMatch(c.a, c.b)
		if got != c.want {
			t.Errorf("titlesMatch(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// strField returns first non-empty string match from the alias list.
func TestStrField(t *testing.T) {
	m := map[string]any{
		"title":    "the title",
		"subtitle": "the sub",
		"empty":    "",
		"number":   42,
	}
	if got := strField(m, "missing", "title"); got != "the title" {
		t.Errorf("strField(missing, title) = %q, want 'the title'", got)
	}
	if got := strField(m, "empty", "subtitle"); got != "the sub" {
		t.Errorf("strField(empty, subtitle) = %q — should skip empty and pick subtitle", got)
	}
	if got := strField(m, "number"); got != "" {
		t.Errorf("strField(number) = %q — non-string should yield empty", got)
	}
	if got := strField(m, "nope"); got != "" {
		t.Errorf("strField(nope) = %q — missing key should yield empty", got)
	}
}

// listField accepts arrays, comma-separated strings, and []string.
func TestListField(t *testing.T) {
	cases := []struct {
		name string
		m    map[string]any
		keys []string
		want []string
	}{
		{
			"slice of any",
			map[string]any{"tags": []any{"a", "b", "c"}},
			[]string{"tags"},
			[]string{"a", "b", "c"},
		},
		{
			"slice of string",
			map[string]any{"tags": []string{"x", "y"}},
			[]string{"tags"},
			[]string{"x", "y"},
		},
		{
			"comma-separated string",
			map[string]any{"tags": "alpha, beta , gamma"},
			[]string{"tags"},
			[]string{"alpha", "beta", "gamma"},
		},
		{
			"empty string yields nil",
			map[string]any{"tags": ""},
			[]string{"tags"},
			nil,
		},
		{
			"missing key yields nil",
			map[string]any{},
			[]string{"tags"},
			nil,
		},
		{
			"falls through to next alias",
			map[string]any{"keywords": []any{"k1", "k2"}},
			[]string{"tags", "keywords"},
			[]string{"k1", "k2"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := listField(c.m, c.keys...)
			if !sliceEq(got, c.want) {
				t.Errorf("listField = %v, want %v", got, c.want)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"  a , b ,c  ", []string{"a", "b", "c"}},
		{"single", []string{"single"}},
		{"a,,b", []string{"a", "b"}},
		{"", []string{}},
	}
	for _, c := range cases {
		got := splitCSV(c.in)
		if !sliceEq(got, c.want) {
			t.Errorf("splitCSV(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFormatDate(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"2026-05-17", "May 17, 2026"},
		{"2026-05-17T12:34:56Z", "May 17, 2026"},
		{"", ""},
		{"unparseable garbage", "unparseable garbage"}, // fall-through to raw
	}
	for _, c := range cases {
		got := formatDate(c.in)
		if got != c.want {
			t.Errorf("formatDate(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// renderByline at a high level: dedup behavior + non-title metadata.
func TestRenderByline_TitleDedupedAgainstH1(t *testing.T) {
	matter := map[string]any{
		"title":  "hello",
		"author": "Brad",
	}
	br := renderByline(matter, "Hello") // case-insensitive match
	if !br.TitleDeduped {
		t.Errorf("expected TitleDeduped=true when title matches first H1")
	}
	if strings.Contains(br.HTML, "fm-title") {
		t.Errorf("byline should NOT include fm-title when deduped, got: %s", br.HTML)
	}
	if !strings.Contains(br.HTML, "Brad") {
		t.Errorf("byline should still include author, got: %s", br.HTML)
	}
}

func TestRenderByline_NoDedupKeepsTitle(t *testing.T) {
	matter := map[string]any{
		"title":  "frontmatter title",
		"author": "Brad",
	}
	br := renderByline(matter, "different H1")
	if br.TitleDeduped {
		t.Errorf("expected TitleDeduped=false when titles differ")
	}
	if !strings.Contains(br.HTML, "fm-title") {
		t.Errorf("byline should include fm-title when not deduped")
	}
}

func TestRenderByline_OnlyDedupedNoExtras_EmptyHTML(t *testing.T) {
	// Title matches and there's nothing else to surface — HTML should be empty
	// but TitleDeduped should still be reported.
	matter := map[string]any{"title": "x"}
	br := renderByline(matter, "x")
	if !br.TitleDeduped {
		t.Errorf("expected TitleDeduped=true")
	}
	if br.HTML != "" {
		t.Errorf("expected empty HTML when nothing remains after dedup, got: %s", br.HTML)
	}
}

func TestRenderByline_EmptyMatter(t *testing.T) {
	br := renderByline(map[string]any{}, "")
	if br.HTML != "" || br.TitleDeduped {
		t.Errorf("empty matter should yield empty result")
	}
}

func TestRenderByline_EscapesHTML(t *testing.T) {
	// Frontmatter values must not be allowed to inject HTML.
	matter := map[string]any{
		"title":  "<script>",
		"author": "<b>x</b>",
		"tags":   []any{"<i>"},
	}
	br := renderByline(matter, "different")
	if strings.Contains(br.HTML, "<script>") {
		t.Errorf("title HTML not escaped: %s", br.HTML)
	}
	if strings.Contains(br.HTML, "<b>x</b>") {
		t.Errorf("author HTML not escaped: %s", br.HTML)
	}
	if strings.Contains(br.HTML, "<i>") {
		t.Errorf("tag HTML not escaped: %s", br.HTML)
	}
}

// Helpers.

func sliceEq(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
