<div align="center">

# *upmark*

**a reading tool for markdown**

[![CI](https://github.com/captured-ventures/upmark/actions/workflows/ci.yml/badge.svg)](https://github.com/captured-ventures/upmark/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/badge/version-0.7.0-rust.svg)](CHANGELOG.md)

</div>

---

upmark is a native, lightweight markdown reader for people who'd rather *read* their notes than fight a kitchen-sink editor. It opens a `.md` file in a calm, typographic frame, gets out of the way, and remembers where you left off.

It is built for the way notes actually live: scattered across folders, cross-linked with wikilinks, sometimes fetched from a URL, occasionally edited in a quick split-pane, mostly just read. The default look is editorial — Newsreader for body, IBM Plex Sans for chrome, JetBrains Mono for code — and there are thirteen other looks to swap to if that's not your taste.

It is intentionally small. No plugin marketplace, no syncing, no AI sidebar (though it can act as an MCP server, see below). One window, one document, one job.

## Screenshot

> _Screenshot pending — open `sample.md` in the editorial theme to see the rendered demo._

## Features

- ✅ **Editorial typography** — Newsreader serif, IBM Plex Sans UI, JetBrains Mono code; frameless window with custom top bar
- ✅ **Read polish** — heading anchors, copy buttons on code blocks, reading-width and font-size controls, per-file scroll memory, image lightbox, footnote tooltips, live word count
- ✅ **Editor mode** — CodeMirror 6 split pane (`Ctrl E`), `Ctrl S` save, 800ms-idle autosave, 200ms-debounced live preview
- ✅ **OS integration** — CLI arg (`upmark file.md`), single-instance lock, `.md` / `.markdown` / `.mdown` / `.mkd` file associations via NSIS installer, "reveal in Explorer" + "open containing folder"
- ✅ **Front-matter** — YAML and TOML parsed and rendered as a styled byline; if the front-matter title matches the first H1, it's deduped and metadata becomes a standfirst under the heading
- ✅ **Open from URL** — `Ctrl L` opens a remote markdown URL; `github.com/.../blob/...` URLs are canonicalized to `raw.githubusercontent.com`; relative images resolved against the source
- ✅ **MCP server** — a local Model Context Protocol server (off by default) so an LLM can present a document to you, request a task list, and watch what you check off
- ✅ **Settings pane** (`Ctrl ,`) — appearance, MCP server toggle, about; theme grid uses live mini-previews
- ✅ **Command palette** (`Ctrl K`) — every action is one keystroke away
- ✅ **Folder tree + wikilinks** — open a folder, click through `[[notes]]`-style links across files

## Themes

upmark ships thirteen themes. Each is a single `[data-theme=...]` CSS-var block — no separate stylesheets. Swap from the palette (`Ctrl K` → "theme") or the settings pane.

| Theme | Vibe |
|---|---|
| **editorial** | warm paper, rust, serif body |
| **broadsheet** | all-serif newspaper |
| **newsprint** | halftone dots, slab headlines, ink red |
| **terminal** | mono, dark, amber |
| **manuscript** | parchment, drop caps, sepia |
| **brutalist** | oversized, black & white, hard edges |
| **arcade** | neon synthwave, scan lines, glow |
| **pastoral** | cream, sage, rounded |
| **architect** | blueprint grid, prussian blue |
| **vapor** | vaporwave gradient, handwriting |
| **typewriter** | courier prime, ribbon red |
| **midnight** | navy library, gold accent |
| **gameboy** | pixel 4-color green |

## MCP server

upmark can run a local [Model Context Protocol](https://modelcontextprotocol.io) server on `127.0.0.1:11451`. When enabled (Settings → MCP server), an LLM client can:

- **`present_document`** — push a markdown document to the reader window
- **`update_document`** — update its content (task-list checks are preserved by line text)
- **`get_document_status`** — read back which tasks the user has checked
- **`list_presented`** — list all open MCP documents
- **`close_document`** — close one

This makes upmark useful as the "reading surface" for an agent: the model writes a design review, the human reads it in good typography, checks off action items, the model polls to see what was accepted.

A test client lives at [`cmd/mcp-test`](cmd/mcp-test/) — run it with upmark open and MCP enabled:

```sh
go run ./cmd/mcp-test
```

To wire it into Claude Desktop, add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "upmark": {
      "url": "http://127.0.0.1:11451/sse"
    }
  }
}
```

The server is localhost-only and has no auth — it is intended for a single user on their own machine.

## Install

**Build from source** is the only path today. Releases with prebuilt installers will land at the first tagged version.

## Build from source

Prereqs: Go 1.23+, Node.js 18+, and the Wails CLI.

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# dev mode (hot-reloads the frontend)
wails dev

# production build → build/bin/upmark.exe
wails build

# Windows installer with .md file association
wails build -nsis

# include Edge DevTools (right-click → Inspect)
wails build -debug
```

## Tech stack

- **[Wails v2](https://wails.io)** — Go ↔ webview bridge; frameless windows, file associations, single-instance
- **Go 1.23+** — backend, file I/O, watcher, MCP server
- **[Svelte 4](https://svelte.dev)** + **TypeScript** + **Vite** — frontend
- **[CodeMirror 6](https://codemirror.net)** — the editor pane
- **[goldmark](https://github.com/yuin/goldmark)** — markdown renderer, with GFM + footnote + typographer + emoji + chroma syntax highlighting
- **[mcp-go](https://github.com/mark3labs/mcp-go)** — MCP server over SSE

## License

[MIT](LICENSE) © 2026 Brad Webb / Captured Ventures
