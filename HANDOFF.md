# upmark — handoff

A native, lightweight markdown reader built with Wails v2 (Go + Svelte 4 + WebView2). Currently at v0.7 — feature-complete enough to call "worthy of a repo and CI."

This document is the handoff from the initial build sprint to the productization/CI sprint. Read it end-to-end before touching the code; everything you need to ramp quickly is here.

---

## Status at a glance

**Shipped (10 phases):**

1. **UX refresh + navigation + palette** — editorial typography (Newsreader + IBM Plex Sans + JetBrains Mono), frameless window with custom top bar, collapsible sidebar (contents / folder tree / recent), command palette (`Ctrl K`).
2. **Read polish** — heading anchor icons, code block copy buttons, reading width + font size controls (persisted), per-file scroll memory, image lightbox, footnote tooltips, word count.
3. **Editor mode** — CodeMirror 6 split pane (`Ctrl E`), `Ctrl S` save, 800ms-idle autosave, live preview on 200ms debounce; fsnotify swallows self-write events for 750ms.
4. **OS integration** — CLI argument (`upmark file.md`), single-instance lock with arg forwarding via `open-from-second-instance` event, NSIS installer registers `.md` / `.markdown` / `.mdown` / `.mkd` file associations, "reveal in Explorer" + "open containing folder" via palette.
5. **Theme system spike** — 13 themes via `[data-theme=...]` CSS vars. Includes flex/microinteraction themes (arcade, vapor, gameboy) with scan-line overlays, glow text-shadow, pixel rendering filters. Persisted; auto light/dark for most.
6. ~~Phase 6~~ moved to bottom — see Phase 11 below.
7. **Front-matter (YAML/TOML)** — parsed via `gopkg.in/yaml.v3` + `BurntSushi/toml`, rendered as a styled byline (title / subtitle / by-author · date / tags / custom-fields). Smart dedup: if `frontmatter.title` matches first H1 (case-insensitive), the byline drops the title and injects the remaining metadata *under* the H1 as a standfirst.
8. **Open from URL** — `Ctrl L` opens a modal URL prompt. 30s timeout, 8MB body cap, github.com/user/repo/blob/branch/path URLs canonicalized to raw.githubusercontent.com. Relative image refs resolved against the source URL via `url.ResolveReference`.
9. **MCP spike** — local MCP server (off by default) on `127.0.0.1:11451`. Spec-compliant via `github.com/mark3labs/mcp-go`. Tools: `present_document` / `update_document` / `get_document_status` / `close_document` / `list_presented`. MCP-presented docs get interactive task lists (frontend strips `disabled`, wires changes back via `MCPSetTaskChecked` → Go state → `get_document_status` returns it).
10. **Settings pane** — `Ctrl ,` opens a modal with three sections (Appearance / MCP server / About). Theme grid uses live mini-previews with the real theme colors and fonts. Topbar sliders icon is the discoverable entry point.

**Pending:**

- **Phase 11 — "Size to fit" button** (TBD; the original `Phase 6` slot, moved). User indicated they wanted to walk through the design. Options floated last time: snap window to natural reading width; reset to default size; shrink to content height; or "something else." Hasn't been decided.
- **Productization** (the new sprint this handoff exists for): scaffold the repo, write README/LICENSE/CHANGELOG, set up CI for lint+build, add a release workflow that builds Windows/macOS/Linux artifacts + NSIS installer on tag, write contributor docs.

---

## Architecture

### Stack
- **Backend**: Go 1.25 + Wails v2.12.0
- **Frontend**: Svelte 4 + Vite + TypeScript
- **Webview**: Microsoft Edge WebView2 (Windows); WKWebView on macOS; WebKit2GTK on Linux
- **Renderer**: goldmark with extensions — GFM, Footnote, Typographer, DefinitionList, emoji, goldmark-highlighting/v2 + chroma
- **Editor**: CodeMirror 6 with `@codemirror/lang-markdown`
- **MCP**: `github.com/mark3labs/mcp-go` v0.54.0 over SSE transport

### Repo layout (top level)
```
upmark/
├── main.go                 # Wails entry, CLI arg parsing, single-instance lock
├── app.go                  # App struct, all Wails-bound methods, lifecycle
├── renderer.go             # goldmark setup, math/mermaid/wikilink preprocessing, asset URL rewriting
├── frontmatter.go          # YAML/TOML parsing, byline HTML generation, dedup logic
├── folder.go               # OpenFolder, FolderEntry tree, wikilink resolution
├── prefs.go                # JSON prefs persisted at %AppData%/upmark/prefs.json
├── watcher.go              # fsnotify-based file watcher, parent-dir mode, debounced
├── assethandler.go         # /local-asset/* HTTP handler for relative images
├── urlopen.go              # OpenURL: fetch + render remote markdown
├── reveal_windows.go       # Reveal-in-Explorer (Windows only)
├── reveal_other.go         # Cross-platform fallback (macOS / Linux)
├── mcp.go                  # MCP server, tool registration, document state
├── dialogs.go              # OpenDirectoryDialog wrapper
├── cmd/mcp-test/main.go    # Standalone test client for the MCP server
├── frontend/
│   ├── src/
│   │   ├── App.svelte              # Root: state orchestration, keyboard, palette command list
│   │   ├── main.ts                 # Vite entry, font imports, CSS imports, app mount
│   │   ├── styles/
│   │   │   ├── app.css             # Reset, layout, scrollbars, drag region, print
│   │   │   ├── themes.css          # 13 [data-theme=...] blocks
│   │   │   └── markdown.css        # .markdown-body styles (themed via vars)
│   │   └── lib/
│   │       ├── TopBar.svelte       # 40px top bar, drag region, all action icons
│   │       ├── Sidebar.svelte      # Collapsible sidebar (contents/folder/recent)
│   │       ├── FolderTree.svelte   # Recursive folder tree component
│   │       ├── EmptyState.svelte   # Editorial empty state
│   │       ├── Viewer.svelte       # Doc viewer, find-in-page, drag-over
│   │       ├── Editor.svelte       # CodeMirror 6 wrapper
│   │       ├── CommandPalette.svelte
│   │       ├── URLPrompt.svelte
│   │       ├── Settings.svelte
│   │       ├── ImageLightbox.svelte
│   │       ├── enhance.ts          # DOM post-processing: callouts, math (KaTeX), mermaid, code-copy, heading-anchors, footnote-tooltips, find, task-lists
│   │       └── types.ts            # Doc, RecentEntry, Folder, MCPDoc, MCPStatus
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
├── build/
│   ├── appicon.png
│   ├── windows/
│   │   ├── icon.ico
│   │   ├── fileIcon.ico            # for .md association
│   │   ├── info.json
│   │   └── installer/
│   │       ├── project.nsi         # NSIS installer config
│   │       └── wails_tools.nsh
│   └── darwin/Info.plist
├── sample.md                       # Full feature demo
├── notes.md                        # Wikilink target demo
├── hanging-marks.svg               # Image demo for sample
├── wails.json                      # Project config + fileAssociations + info
└── go.mod / go.sum
```

### Backend data flow
```
disk file / URL → renderer.render(src, baseRef)
                  ├── parseFrontmatter (returns matter + body)
                  ├── findFirstH1 (for dedup)
                  ├── preprocessMermaid  (replace ```mermaid blocks with <pre class="mermaid">)
                  ├── preprocessMath     (replace $$..$$ / $..$ with <span class="math-*">)
                  ├── preprocessWikilinks (replace [[name]] with <a class="wikilink">)
                  ├── goldmark.Convert
                  ├── rewriteAssetURLs   (relative paths → /local-asset/ or absolute URL)
                  └── renderByline       (inject byline; under H1 if title-deduped)
```

### Frontend data flow
- All state lives in `App.svelte`. Children receive props and dispatch events upward.
- Doc state is type `Doc` with optional `isMCP` / `mcpId` for MCP-presented docs.
- Theme is applied via `document.documentElement.setAttribute('data-theme', name)`; all components react via CSS variables.
- Reading width / font size are CSS variables (`--reading-width`, `--reading-size`) on `document.documentElement`.
- Maximize state is detected synchronously via `window.outerWidth >= screen.availWidth - 2` (NOT via async `IsMaximized()` — that introduced a race; see "non-obvious things" below).

### Wails-bound methods (callable from JS)
Backend methods on `*App` that the frontend uses:
- File: `OpenDialog`, `OpenPath`, `CloseDocument`, `RecentFiles`, `ClearRecent`
- Folder: `OpenFolderDialog`, `OpenFolder`, `CloseFolder`, `LastFolder`, `ResolveWikilink`
- Edit: `SaveDocument`, `RenderMarkdown`
- URL: `OpenURL`
- Reveal: `RevealInExplorer`, `OpenContainingFolder`
- Window: `MinimizeWindow`, `ToggleMaximizeWindow`, `CloseWindow`, `IsMaximized`
- Prefs: `GetUIPrefs`, `SetReadingWidth`, `SetFontSize`, `SetTheme`, `GetScrollPos`, `SetScrollPos`
- MCP: `GetMCPStatus`, `SetMCPEnabled`, `SetMCPPort`, `MCPSetTaskChecked`, `MCPSetViewState`
- Misc: `GetChromaCSS`, `GetStartupOrLastFile`

Frontend events the backend emits:
- `file-changed` — fsnotify detected an external write to the open file
- `file-error` — render or watch error
- `open-from-second-instance` — single-instance lock forwarded a CLI arg
- `mcp-doc-presented` / `mcp-doc-updated` / `mcp-doc-closed` — MCP tool calls

---

## Build / run / dev

### Prerequisites
- Go 1.23+ (1.25 in current sum)
- Node.js 18+ (24 in dev env)
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- WebView2 runtime on Windows (preinstalled on Windows 11)

### Dev
```
wails dev
```
Hot-reloads the frontend on file save. Restart needed for backend changes.

### Production build
```
wails build              # → build/bin/upmark.exe
wails build -debug       # adds Edge DevTools (right-click → Inspect)
wails build -nsis        # also produces installer at build/bin/upmark-amd64-installer.exe
```

### MCP test
With upmark running and MCP enabled (Settings → MCP server → enable):
```
go run ./cmd/mcp-test
```
Pushes a sample design-review doc with a 5-item task list; polls user selections every 5s.

---

## Non-obvious things future-you needs to know

These are all things I learned the hard way during the sprint. Reading this list now saves you the same time.

### 1. Svelte 4 scoped styles double the hash class for specificity
Component-scoped CSS compiles `.foo` → `.foo.svelte-xxx.svelte-xxx` (specificity 0,3,0). Global theme overrides that target the same class tie on specificity and lose by source order (component CSS loads after static stylesheets). **For theme overrides on properties a Svelte component declares, you must use `!important`.** Properties the component doesn't declare can stay plain. Memory file at `~/.claude/projects/.../memory/feedback_svelte_theme_overrides.md` documents this.

### 2. WebView2's `scrollIntoView` doesn't honor `overflow: hidden` on programmatic scrolls
Calling `el.scrollIntoView()` walks up scrollable ancestors and scrolls them even when `body` has `overflow: hidden`. Symptom: clicking a TOC item upshifts the entire document by a few px (`topbar.top` goes from `0` to `-5.33` etc.). **Fix: use `scrollToInViewer()` from `enhance.ts` everywhere instead of `scrollIntoView()`.** That helper writes only to `viewer.scrollTop`, never touches ancestors.

### 3. Default focus on mousedown for sidebar buttons triggers ancestor auto-scroll
The browser scrolls the nearest scrollable ancestor of a focused element to keep it visible. In a long TOC, clicking an item focused the button which scrolled `.sb-top`, shifting all items up. **Fix: `suppressButtonFocus` handler on `.sidebar` calls `preventDefault()` on `mousedown` for any button target. Keyboard Tab focus still works.**

### 4. `yaml.v3` auto-parses dates as `time.Time`, not strings
Wrote `dateField()` to handle both. `strField()` only handles strings.

### 5. CommandPalette input needs `min-width: 0`
Flex item default `min-width: auto` means the input refuses to shrink below its placeholder's intrinsic content width. In mono themes (terminal, gameboy) this stretched the palette past its 560px target.

### 6. Maximize race condition on frameless Windows
Frameless windows extend ~8px beyond every screen edge when maximized; content gets clipped without compensation. Used to call `IsMaximized()` async on `resize` — race window of clipped content before the class arrived. **Switched to sync detection: `window.outerWidth >= screen.availWidth - 2`.**

### 7. Goldmark renders raw HTML when `WithUnsafe()` is enabled
We do enable it because mermaid/math preprocessing emits raw HTML blocks. Anything else inside the markdown is also passed through. Acceptable risk since these are user-opened local files.

### 8. Wails v2 file association is declarative
Lives in `wails.json` under `fileAssociations`. NSIS `wails.associateFiles` macro reads it; requires a matching `fileIcon.ico` in `build/windows/`. Only takes effect after an installer-based install.

### 9. The `.svelte-extra.d.ts` shim was removed; we cast `OnFileDrop` inline
Wails' bundled `runtime.d.ts` is out of date and missing `OnFileDrop` / `OnFileDropOff`. In `App.svelte` they're imported via `import * as wailsRuntime` and cast `as any`. Works because `runtime.js` does export them.

### 10. Inline TypeScript casts break Svelte's template parser
`{(e.target as HTMLInputElement).value}` in a template attribute is a `ParseError`. **Extract any expression that uses `as` to a function in the script block.** Hit this twice in `Settings.svelte`.

---

## Memory files

`~/.claude/projects/C--captured-ventures-upmark/memory/`
- `MEMORY.md` (index)
- `feedback_svelte_theme_overrides.md` — the doubled-hash specificity finding

---

## Things deferred / known limitations

- **Editor for MCP docs**: button doesn't disable; `SaveDocument` on `mcp:abc123` would fail at `os.WriteFile`. Should be hidden when `isMCP`.
- **URL docs in recents**: not tracked. Would need a separate "recent URLs" pref.
- **URL doc caching**: refetch every open.
- **MCP server auth**: localhost-only; no shared secret. Fine for local use, not for any public surface.
- **MCP update_document task preservation**: matches by line text. Fragile if the LLM rewords items. A stable ID scheme would fix it.
- **Mermaid theme**: initialized once on first render based on `prefers-color-scheme`. Doesn't re-init on theme switch — diagrams keep their original light/dark setting until the doc is reopened.
- **Wikilink disambiguation**: if multiple files have the same basename across a folder tree, the first match wins. No UI to choose.
- **No keyboard nav in sidebar**: Tab works but no arrow-key sequencing through TOC / folder.
- **Welcome doc**: first-launch empty state is sparse if no recent files. Could ship a built-in walkthrough.
- **No actual tests** in the Go code — verification has been manual through the running app and the `cmd/mcp-test` client.

---

## License

Not yet chosen. The user (project owner: Brad Webb / Captured Ventures, `brad@founder.media`) should decide between MIT / Apache-2 / something else before the repo goes public.

---

## What "ready to ship" looks like

The productization sprint that comes next should produce, at minimum:

1. `git init` + initial commit
2. **README.md** with: hero, feature list, screenshot(s), install instructions (download release + build from source), quickstart, MCP server section, theme gallery, tech stack, license badge
3. **LICENSE** (user-chosen)
4. **CHANGELOG.md** seeded with v0.7 entry (this sprint)
5. **CONTRIBUTING.md** — local dev setup, code style, PR conventions
6. **`.github/workflows/ci.yml`** — runs on PR: gofmt, go vet, go build, npm install, svelte-check, wails build smoke (Linux only is fine for CI speed)
7. **`.github/workflows/release.yml`** — runs on tag (`v*`): cross-platform Wails builds (Windows AMD64 + macOS universal + Linux AMD64), produces the NSIS installer on Windows, attaches all artifacts to a GitHub Release
8. **`.github/ISSUE_TEMPLATE/`** — bug + feature templates
9. **`.gitignore`** — verify what Wails scaffolded plus add `/build/bin/`, OS-specific patterns
10. Address the deferred items above as their own follow-up issues, not blocking the v0.7 release

`wails-build-action` from GitHub Marketplace can simplify the release workflow significantly.
