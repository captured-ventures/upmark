package main

import (
	"strings"
	"testing"
)

// preprocessWikilinks: [[name]] and [[name|label]] -> <a class="wikilink">
// Inside fenced and inline code, wikilinks must not be touched.
func TestPreprocessWikilinks(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		contains []string
		excludes []string
	}{
		{
			"simple",
			"see [[notes]] for context",
			[]string{`class="wikilink"`, `data-wikilink="notes"`, `>notes</a>`},
			nil,
		},
		{
			"with label",
			"see [[notes|my notes]] for context",
			[]string{`data-wikilink="notes"`, `>my notes</a>`},
			nil,
		},
		{
			"inside fenced code untouched",
			"```\n[[notes]] here\n```",
			[]string{"[[notes]]"},
			[]string{"wikilink"},
		},
		{
			"inside inline code untouched",
			"see `[[notes]]` here",
			[]string{"[[notes]]"},
			[]string{"wikilink"},
		},
		{
			"escapes target",
			"[[evil<script>]]",
			[]string{"evil&lt;script&gt;"},
			[]string{"<script>"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := string(preprocessWikilinks([]byte(c.in)))
			for _, s := range c.contains {
				if !strings.Contains(out, s) {
					t.Errorf("output missing %q: %s", s, out)
				}
			}
			for _, s := range c.excludes {
				if strings.Contains(out, s) {
					t.Errorf("output should not contain %q: %s", s, out)
				}
			}
		})
	}
}

// preprocessMermaid: ```mermaid blocks become <pre class="mermaid">.
func TestPreprocessMermaid(t *testing.T) {
	in := "before\n```mermaid\nflowchart LR\nA-->B\n```\nafter"
	out := string(preprocessMermaid([]byte(in)))
	if !strings.Contains(out, `<pre class="mermaid">`) {
		t.Errorf("missing <pre class=\"mermaid\">: %s", out)
	}
	if !strings.Contains(out, "flowchart LR") {
		t.Errorf("mermaid source not preserved: %s", out)
	}
	if strings.Contains(out, "```mermaid") {
		t.Errorf("original fence should be replaced: %s", out)
	}
}

func TestPreprocessMermaid_EscapesHTML(t *testing.T) {
	// Mermaid source inside <pre> must be HTML-escaped so the frontend
	// doesn't pick up stray tags.
	in := "```mermaid\nflowchart\nA[\"<script>\"]\n```"
	out := string(preprocessMermaid([]byte(in)))
	if strings.Contains(out, "<script>") {
		t.Errorf("script tag not escaped in mermaid source: %s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Errorf("expected &lt;script&gt; escape: %s", out)
	}
}

// preprocessMath: $$..$$ -> <div class="math-display">, $..$ -> inline span,
// math inside code blocks must be untouched.
func TestPreprocessMath_Display(t *testing.T) {
	out := string(preprocessMath([]byte("text\n$$E = mc^2$$\nmore")))
	if !strings.Contains(out, `<div class="math-display">E = mc^2</div>`) {
		t.Errorf("expected math-display div: %s", out)
	}
}

func TestPreprocessMath_Inline(t *testing.T) {
	out := string(preprocessMath([]byte("inline $a + b$ math")))
	if !strings.Contains(out, `<span class="math-inline">a + b</span>`) {
		t.Errorf("expected math-inline span: %s", out)
	}
}

func TestPreprocessMath_CodeBlocksUntouched(t *testing.T) {
	in := "```\n$E=mc^2$\n```\nand $real$ math here"
	out := string(preprocessMath([]byte(in)))
	// Inside fenced code: dollars should remain literal.
	if !strings.Contains(out, "$E=mc^2$") {
		t.Errorf("math inside fenced code was processed: %s", out)
	}
	// Outside fenced code: should be processed.
	if !strings.Contains(out, `<span class="math-inline">real</span>`) {
		t.Errorf("math outside code was not processed: %s", out)
	}
}

func TestPreprocessMath_InlineCodeUntouched(t *testing.T) {
	in := "see `$x$` for syntax"
	out := string(preprocessMath([]byte(in)))
	if !strings.Contains(out, "`$x$`") {
		t.Errorf("math inside inline code was processed: %s", out)
	}
}

// rewriteAssetURLs: relative refs get routed through /local-asset/ when
// baseDir is a local path, or resolved against baseURL when it's http(s).
// Absolute / data: / mailto: / fragment refs must pass through untouched.
func TestRewriteAssetURLs_LocalDir(t *testing.T) {
	html := `<img src="diagram.png"><a href="other.md">link</a><a href="#anchor">a</a><a href="https://example.com">ext</a>`
	out := rewriteAssetURLs(html, "/tmp/notes")
	if !strings.Contains(out, "/local-asset/") {
		t.Errorf("local relative img should route through /local-asset/: %s", out)
	}
	if !strings.Contains(out, `href="#anchor"`) {
		t.Errorf("fragment link should be untouched: %s", out)
	}
	if !strings.Contains(out, `href="https://example.com"`) {
		t.Errorf("absolute http link should be untouched: %s", out)
	}
}

func TestRewriteAssetURLs_HTTPBase(t *testing.T) {
	// Relative refs against an http base resolve via url.ResolveReference.
	html := `<img src="diagram.png"><a href="../other.md">link</a>`
	out := rewriteAssetURLs(html, "https://example.com/repo/docs/index.md")
	if !strings.Contains(out, "https://example.com/repo/docs/diagram.png") {
		t.Errorf("relative img not resolved against base URL: %s", out)
	}
	if !strings.Contains(out, "https://example.com/repo/other.md") {
		t.Errorf("relative anchor not resolved against base URL: %s", out)
	}
}

func TestRewriteAssetURLs_NoBase(t *testing.T) {
	html := `<img src="diagram.png">`
	out := rewriteAssetURLs(html, "")
	if out != html {
		t.Errorf("empty base should pass through unchanged: %s", out)
	}
}

func TestRewriteAssetURLs_PreservesSpecialSchemes(t *testing.T) {
	html := `<a href="mailto:a@b.c">e</a><a href="upmark:wikilink/foo">w</a><a href="data:text/plain,hi">d</a>`
	out := rewriteAssetURLs(html, "/tmp/notes")
	for _, s := range []string{`href="mailto:a@b.c"`, `href="upmark:wikilink/foo"`, `href="data:text/plain,hi"`} {
		if !strings.Contains(out, s) {
			t.Errorf("special-scheme URL %q was rewritten: %s", s, out)
		}
	}
}

// Full render() pipeline integration: front-matter dedup + GFM + math + mermaid
// + wikilinks all in one document.
func TestRender_FullPipeline(t *testing.T) {
	r, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	src := []byte(`---
title: Pipeline test
author: Brad
---
# Pipeline test

A paragraph with **bold** and a [[wikilink]] and $E=mc^2$ and:

` + "```mermaid\nflowchart LR\nA-->B\n```\n" + `

` + "```go\nfunc main() {}\n```\n")

	out, err := r.render(src, "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	mustContain := []string{
		`class="markdown-body"`,
		`<h1 id="pipeline-test">Pipeline test</h1>`,
		`<strong>bold</strong>`,
		`class="wikilink"`,
		`class="math-inline"`,
		`class="mermaid"`,
		`class="frontmatter frontmatter-under"`, // title was deduped → under H1
		`by Brad`,                              // author byline still rendered
		`class="chroma"`,                       // chroma syntax highlighting applied
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("rendered output missing %q\n---\n%s", s, out)
		}
	}
}

func TestRender_NoFrontmatter(t *testing.T) {
	r, _ := newRenderer()
	out, err := r.render([]byte("# heading\nbody\n"), "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out, "frontmatter") {
		t.Errorf("output should not contain frontmatter block when none present: %s", out)
	}
	if !strings.Contains(out, `<h1 id="heading">heading</h1>`) {
		t.Errorf("expected h1: %s", out)
	}
}

func TestRender_TaskListRendered(t *testing.T) {
	r, _ := newRenderer()
	out, err := r.render([]byte("- [x] done\n- [ ] not done\n"), "")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(out, `type="checkbox"`) {
		t.Errorf("task list checkboxes missing: %s", out)
	}
	if !strings.Contains(out, `checked=""`) && !strings.Contains(out, `checked`) {
		t.Errorf("checked attribute missing for done task: %s", out)
	}
}
