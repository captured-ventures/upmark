# Changelog

All notable changes to upmark are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.7.1] - 2026-05-17

### Added

- **MCPB bridge** ([#9](https://github.com/captured-ventures/upmark/issues/9)) — Claude Desktop, Claude Code, and Claude.ai can now talk to upmark through a stdio↔SSE bridge. Released as `upmark.mcpb` alongside the platform installers. The bridge auto-launches upmark on first tool call, mirrors all five MCP tools, and proxies transparently. Install via Claude Desktop → Settings → Extensions → Install Extension.
- **Phase 11 — focus mode** ([#6](https://github.com/captured-ventures/upmark/issues/6)) — top-bar toggle (also `Ctrl+Shift+F` or palette "focus mode") collapses the sidebar and resizes the window to fit the current reading column at monitor full height. Press again, Esc, or the palette command to restore prior dimensions.
- **MCP source banner** — when an MCP-presented doc is active, the topbar's anchor strip tints with the theme accent and a `[mcp]` chip + originating client name (e.g. "Claude Desktop", "Cursor") appears immediately right of the sidebar divider. Captures `clientInfo.name` from the MCP initialize handshake.
- **Settings → MCP server: Claude Desktop entry** — 6th client group in the picker that shows `.mcpb` install steps instead of a paste-in snippet.
- **Settings → MCP server: in-app client setup snippets** ([#8](https://github.com/captured-ventures/upmark/issues/8)) — 5-button picker collapsing 8 supported clients (Cursor / Cline / Warp / Gemini CLI, VS Code, Continue, Windsurf, Codex) into pattern groups with copy-to-clipboard.
- **Sidebar keyboard navigation** ([#3](https://github.com/captured-ventures/upmark/issues/3)) — ↑↓ Home End Enter Esc, plus ←/→ on folder rows.
- **Welcome doc** — embedded markdown auto-opens on first launch (gated by `prefs.WelcomeSeen`), always reachable via palette "show welcome".
- **Brand assets** — production wordmark + multi-size Windows icons + correct macOS bundle ID (`com.captured-ventures.upmark`).

### Fixed

- **Renderer rewrites `srcset` URLs in `<picture>` and responsive `<img>`** — README-style dark/light wordmarks now resolve through the local-asset handler. Previously only `<img src=...>` was rewritten; matching `<source srcset=...>` URLs stayed bare and failed to load.
- **Mermaid diagrams re-render on theme switch** ([#2](https://github.com/captured-ventures/upmark/issues/2)) — `refreshMermaid` reads `--bg` luminance and re-inits when the light/dark resolution flips.
- **Window-control buttons fill the full topbar height** ([#10](https://github.com/captured-ventures/upmark/issues/10)) — minimize/maximize/close hit area no longer cut off vertically.
- **Task-list checkboxes interactive in non-MCP docs** — was inadvertently restricted to MCP-presented docs after a refactor.

### Changed

- **Welcome doc + Settings → MCP server hint** now reflect the shipped Claude Desktop integration instead of promising it.
- **Bridge icon** upgraded from a hand-shrunk 256×256 to the canonical 512×512 to match Claude Desktop's display recommendation.

### Tests

- **First Go tests** ([#4](https://github.com/captured-ventures/upmark/issues/4)) — `renderer_test.go` + `frontmatter_test.go`, 29 tests covering preprocessors, frontmatter parsing, and the render pipeline. Mandatory in CI.

### Infrastructure

- **MCPB release artifact** — `release.yml` builds + attaches `upmark-<v>.mcpb` to GitHub Releases on tag push.
- **upmark.run release-time dispatch** — successful release fires `repository_dispatch` to `captured-ventures/upmark.run` so the sibling site rebuilds with fresh download URLs + sha256s.

## [0.7.0] - 2026-05-17

First public release. Ten phases of build sprint, productized into a real repo with CI and releases.

### Added

- **UX refresh + navigation + palette** — editorial typography (Newsreader, IBM Plex Sans, JetBrains Mono), frameless window with custom top bar, collapsible sidebar (contents / folder tree / recent), command palette (`Ctrl K`).
- **Read polish** — heading anchor icons, code block copy buttons, persisted reading width + font size, per-file scroll memory, image lightbox, footnote tooltips, word count.
- **Editor mode** — CodeMirror 6 split pane (`Ctrl E`), `Ctrl S` save, 800ms-idle autosave, 200ms live preview; fsnotify self-write suppression for 750ms.
- **OS integration** — CLI arg (`upmark file.md`), single-instance lock with arg forwarding, NSIS installer registers `.md` / `.markdown` / `.mdown` / `.mkd` associations, "reveal in Explorer" + "open containing folder".
- **Theme system** — 13 themes via `[data-theme=...]` CSS-var blocks, including flex/microinteraction themes (arcade, vapor, gameboy) with scan lines, glow, and pixel filters. Persisted across launches.
- **Front-matter (YAML / TOML)** — parsed and rendered as a styled byline; smart dedup folds the front-matter title into the first H1 when they match (case-insensitive) and injects remaining metadata as a standfirst.
- **Open from URL** — `Ctrl L` URL prompt with 30s timeout and 8MB body cap; `github.com/user/repo/blob/...` canonicalized to `raw.githubusercontent.com`; relative images resolved against source.
- **MCP spike** — local MCP server (off by default) on `127.0.0.1:11451`, spec-compliant via `mcp-go`. Tools: `present_document`, `update_document`, `get_document_status`, `close_document`, `list_presented`. Task lists are interactive — user checks flow back to the LLM.
- **Settings pane** — `Ctrl ,` opens a modal with Appearance / MCP server / About sections. Theme grid uses live mini-previews with each theme's real colors and fonts.

### Project

- Git initialized; MIT license; README, CHANGELOG, CONTRIBUTING.
- GitHub Actions CI (gofmt, go vet, go build, svelte-check, Linux wails build).
- GitHub Actions release workflow producing Windows (amd64 + NSIS installer), macOS (universal), and Linux (amd64) artifacts on `v*` tags.
- Issue templates for bugs and feature requests.

[Unreleased]: https://github.com/captured-ventures/upmark/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/captured-ventures/upmark/releases/tag/v0.7.0
