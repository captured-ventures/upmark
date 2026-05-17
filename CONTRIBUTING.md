# Contributing to upmark

Thanks for your interest. upmark is a small project with a strong point of view — contributions are welcome, but please open an issue to discuss feature work before sending a large PR.

## Prerequisites

- **Go 1.23 or newer** (1.25 in the current build)
- **Node.js 18 or newer**
- **Wails CLI** — `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **WebView2 runtime** on Windows (preinstalled on Windows 11)

## Running in dev

```sh
wails dev
```

This boots Vite with hot-reload for frontend changes. Backend changes require a restart.

For a production build:

```sh
wails build              # → build/bin/upmark.exe
wails build -debug       # adds Edge DevTools (right-click → Inspect)
wails build -nsis        # also produces the Windows installer
```

## Running the MCP test client

With upmark running and the MCP server enabled (Settings → MCP server → enable):

```sh
go run ./cmd/mcp-test
```

The test client pushes a sample design-review document with a 5-item task list and polls for user selections every 5 seconds.

## Before opening a PR

```sh
gofmt -l .           # should produce no output
go vet ./...
go build ./...
cd frontend
npm run check        # svelte-check
npm run build
```

CI runs all of the above plus a Linux `wails build` smoke test.

## Coding conventions

These come from the original build sprint. Please follow them — they keep the project's voice consistent.

### Lowercase UI strings

Every user-facing string in the chrome is lowercase. The command palette commands ("open file", "narrower reading width", "toggle editor"), tooltips, button labels, settings section headers — all lowercase. Title-case is for markdown headings and this README.

### Hairline SVG icons

All icons are hand-rolled inline SVG with stroke-width 1.3–1.4. No icon library. There are roughly 30 icons in `TopBar.svelte` / `Sidebar.svelte` / `Settings.svelte`; if you need a new one, match the existing style. Don't introduce Lucide or similar unless we swap all of them in one PR.

### Themes are CSS-var override blocks

Anything new in `frontend/src/styles/themes.css` must respect the `[data-theme=...]` pattern — one selector per theme, no per-theme stylesheets, no JavaScript theme switching beyond `data-theme` on `<html>`.

### `!important` is the documented escape hatch for theme overrides

Svelte 4's component-scoped CSS doubles the hash class on the selector, producing specificity `(0,3,0)`. Global theme rules that target the same class tie on specificity and lose by source order. For any property a Svelte component declares, theme overrides must use `!important`. Don't try to out-specificity Svelte.

### One window, one document, one job

upmark is intentionally narrow. New features should make the reading or note-traversal experience better; they should not turn it into a notes manager, sync client, or general-purpose IDE. If you're not sure whether a feature fits, open an issue first.

## Repo layout

The top-level Go files (`app.go`, `renderer.go`, `frontmatter.go`, `folder.go`, `prefs.go`, `watcher.go`, `mcp.go`, `urlopen.go`, etc.) each own one concern. The Svelte frontend lives in `frontend/src/` — see [`frontend/README.md`](frontend/README.md) for the layout.

## License

By contributing you agree your work will be licensed under the [MIT License](LICENSE).
