# Changelog

All notable changes to upmark are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
