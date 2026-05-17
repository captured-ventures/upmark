package main

import (
	"bytes"
	"html"
	"net/url"
	"regexp"
	"strings"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

type renderer struct {
	md goldmark.Markdown

	cssOnce  sync.Once
	cssLight string
	cssDark  string
}

func newRenderer() (*renderer, error) {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Footnote,
			extension.Typographer,
			extension.DefinitionList,
			emoji.Emoji,
			highlighting.NewHighlighting(
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
					chromahtml.TabWidth(4),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
		),
	)
	return &renderer{md: md}, nil
}

// render produces HTML for the given markdown source. The output is a single
// HTML tree wrapped in <div class="markdown-body">. Theme switching is handled
// purely on the frontend (CSS variables + class-based chroma stylesheets).
func (r *renderer) render(src []byte, baseDir string) (string, error) {
	matter, src := parseFrontmatter(src)
	firstH1 := findFirstH1(src)

	src = preprocessMermaid(src)
	src = preprocessMath(src)
	src = preprocessWikilinks(src)

	var buf bytes.Buffer
	if err := r.md.Convert(src, &buf); err != nil {
		return "", err
	}

	body := rewriteAssetURLs(buf.String(), baseDir)
	br := renderByline(matter, firstH1)

	// If the frontmatter title matched the doc's first H1, place the rest of
	// the byline (subtitle / meta / tags / fields) immediately after that H1
	// so it reads as a standfirst, not a competing headline.
	if br.TitleDeduped && br.HTML != "" {
		if idx := strings.Index(body, "</h1>"); idx >= 0 {
			body = body[:idx+5] + br.HTML + body[idx+5:]
			return `<div class="markdown-body">` + body + `</div>`, nil
		}
	}
	return `<div class="markdown-body">` + br.HTML + body + `</div>`, nil
}

// Wikilinks: [[target]] or [[target|label]]. Rendered as anchor with a special
// scheme the frontend intercepts. We don't resolve here — the frontend asks
// the backend for the real path on click, so links update if the folder
// changes.
var wikilinkRE = regexp.MustCompile(`\[\[([^\]\n]{1,200})\]\]`)

func preprocessWikilinks(src []byte) []byte {
	type token struct {
		key  string
		body []byte
	}
	var masked []token
	masker := func(re *regexp.Regexp, prefix string) {
		src = re.ReplaceAllFunc(src, func(b []byte) []byte {
			k := []byte("\x00WL_MASK_" + prefix + "_" + itoa(len(masked)) + "\x00")
			masked = append(masked, token{string(k), b})
			return k
		})
	}
	masker(codeFenceRE, "F")
	masker(codeInlineRE, "I")

	src = wikilinkRE.ReplaceAllFunc(src, func(b []byte) []byte {
		m := wikilinkRE.FindSubmatch(b)
		if len(m) < 2 {
			return b
		}
		inner := strings.TrimSpace(string(m[1]))
		target := inner
		label := inner
		if pipe := strings.Index(inner, "|"); pipe >= 0 {
			target = strings.TrimSpace(inner[:pipe])
			label = strings.TrimSpace(inner[pipe+1:])
		}
		return []byte(`<a class="wikilink" data-wikilink="` + html.EscapeString(target) + `" href="upmark:wikilink/` + url.PathEscape(target) + `">` + html.EscapeString(label) + `</a>`)
	})

	for _, t := range masked {
		src = bytes.ReplaceAll(src, []byte(t.key), t.body)
	}
	return src
}

// chromaCSS returns two stylesheets — one for light, one for dark — built from
// chroma's class-based output. Cached after first call.
func (r *renderer) chromaCSS() (light string, dark string) {
	r.cssOnce.Do(func() {
		r.cssLight = chromaStylesheet("github")
		r.cssDark = chromaStylesheet("github-dark")
	})
	return r.cssLight, r.cssDark
}

func chromaStylesheet(name string) string {
	style := styles.Get(name)
	if style == nil {
		style = styles.Fallback
	}
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	var buf bytes.Buffer
	if err := formatter.WriteCSS(&buf, style); err != nil {
		return ""
	}
	return buf.String()
}

// Frontmatter is parsed (and stripped) by parseFrontmatter in frontmatter.go.

// Mermaid fenced blocks become raw HTML the frontend can render.
var mermaidRE = regexp.MustCompile("(?m)^[ \\t]*```[ \\t]*mermaid[ \\t]*\\r?\\n([\\s\\S]*?)\\r?\\n[ \\t]*```[ \\t]*$")

func preprocessMermaid(src []byte) []byte {
	return mermaidRE.ReplaceAllFunc(src, func(match []byte) []byte {
		m := mermaidRE.FindSubmatch(match)
		if len(m) < 2 {
			return match
		}
		escaped := html.EscapeString(string(m[1]))
		return []byte("\n\n<pre class=\"mermaid\">" + escaped + "</pre>\n\n")
	})
}

// Math: $$...$$ and $...$. We mask code blocks first so we don't touch math-like
// content inside code.
var (
	codeFenceRE   = regexp.MustCompile("(?s)```[\\s\\S]*?```")
	codeInlineRE  = regexp.MustCompile("`[^`\\n]+`")
	mathDisplayRE = regexp.MustCompile(`(?s)\$\$(.+?)\$\$`)
	mathInlineRE  = regexp.MustCompile(`(?:^|[^\\$])\$([^\$\n]{1,200}?)\$`)
)

func preprocessMath(src []byte) []byte {
	type token struct {
		key  string
		body []byte
	}
	var masked []token
	masker := func(re *regexp.Regexp, prefix string) {
		src = re.ReplaceAllFunc(src, func(b []byte) []byte {
			k := []byte("\x00MD_MASK_" + prefix + "_" + itoa(len(masked)) + "\x00")
			masked = append(masked, token{string(k), b})
			return k
		})
	}
	masker(codeFenceRE, "F")
	masker(codeInlineRE, "I")

	src = mathDisplayRE.ReplaceAllFunc(src, func(b []byte) []byte {
		m := mathDisplayRE.FindSubmatch(b)
		if len(m) < 2 {
			return b
		}
		return []byte("\n\n<div class=\"math-display\">" + html.EscapeString(string(m[1])) + "</div>\n\n")
	})
	src = mathInlineRE.ReplaceAllFunc(src, func(b []byte) []byte {
		m := mathInlineRE.FindSubmatch(b)
		if len(m) < 2 {
			return b
		}
		lead := []byte{}
		if len(b) > 0 && b[0] != '$' {
			lead = []byte{b[0]}
		}
		return append(lead, []byte("<span class=\"math-inline\">"+html.EscapeString(string(m[1]))+"</span>")...)
	})

	for _, t := range masked {
		src = bytes.ReplaceAll(src, []byte(t.key), t.body)
	}
	return src
}

var (
	imgSrcRE = regexp.MustCompile(`(<img\b[^>]*\bsrc=")([^"]+)(")`)
	aHrefRE  = regexp.MustCompile(`(<a\b[^>]*\bhref=")([^"]+)(")`)
)

func rewriteAssetURLs(htmlStr, base string) string {
	if base == "" {
		return htmlStr
	}
	// If base looks like an http(s) URL, resolve relative refs against it
	// directly. Otherwise treat it as a local directory and route through
	// /local-asset/ so the asset handler can serve files from disk.
	lowerBase := strings.ToLower(base)
	isURL := strings.HasPrefix(lowerBase, "http://") || strings.HasPrefix(lowerBase, "https://")
	var baseURL *url.URL
	if isURL {
		baseURL, _ = url.Parse(base)
	}

	rewrite := func(orig string) string {
		if orig == "" {
			return orig
		}
		lower := strings.ToLower(orig)
		switch {
		case strings.HasPrefix(orig, "#"),
			strings.HasPrefix(lower, "http://"),
			strings.HasPrefix(lower, "https://"),
			strings.HasPrefix(lower, "data:"),
			strings.HasPrefix(lower, "mailto:"),
			strings.HasPrefix(lower, "upmark:"),
			strings.HasPrefix(lower, "/local-asset/"):
			return orig
		}
		if isURL && baseURL != nil {
			if ref, err := url.Parse(orig); err == nil {
				return baseURL.ResolveReference(ref).String()
			}
			return orig
		}
		return "/local-asset/" + url.PathEscape(base) + "/" + url.PathEscape(orig)
	}
	htmlStr = imgSrcRE.ReplaceAllStringFunc(htmlStr, func(m string) string {
		parts := imgSrcRE.FindStringSubmatch(m)
		return parts[1] + rewrite(parts[2]) + parts[3]
	})
	htmlStr = aHrefRE.ReplaceAllStringFunc(htmlStr, func(m string) string {
		parts := aHrefRE.FindStringSubmatch(m)
		href := parts[2]
		if strings.HasPrefix(href, "#") {
			return m
		}
		return parts[1] + rewrite(href) + parts[3]
	})
	return htmlStr
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var b [20]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}
