# Kickoff prompt for the productization sprint

Paste this into a fresh Claude Code session opened in `C:\captured-ventures\upmark`.

---

I'm continuing work on **upmark**, a native markdown reader I just finished a 10-phase build sprint on. The codebase is feature-complete enough to plant a flag and turn into a real repo.

**Start by reading [HANDOFF.md](HANDOFF.md) end-to-end.** It explains what's built, the architecture, where everything lives, and the non-obvious gotchas from the build (Svelte 4 specificity, WebView2 scrollIntoView quirks, etc.). Don't skip it — the gotchas list will save you a lot of debugging.

This sprint is about making the project shippable: getting it into git, writing docs, setting up CI, and producing distributable installers. **Not** about adding features — feature work resumes after this is done.

## Concrete deliverables, in order

1. **`git init` + first commit.** Do not push anywhere. Use a sensible initial commit message like `Initial commit: upmark v0.7`. Verify `.gitignore` excludes `build/bin/`, `frontend/node_modules/`, `frontend/dist/`, `frontend/wailsjs/` (these are generated), and OS-specific cruft. Wails scaffolded a `.gitignore` — augment, don't replace.

2. **License.** Ask me which license to use (MIT vs. Apache 2.0 are the candidates). Once chosen, write the `LICENSE` file and add the matching badge to the README.

3. **README.md** with this rough structure:
   - Hero: italic "upmark" wordmark + one-line tagline ("a reading tool for markdown")
   - Status badge row (CI + license + version)
   - 2-3 paragraph "what this is, who it's for"
   - Screenshot(s) — start with one of `sample.md` rendered in the editorial theme; we can take more later via `wails build -debug` + manual screenshot
   - **Features** (with checkmarks): the 10 shipped phases summarized in user-facing language. Pull from HANDOFF.md's status section.
   - **Themes** gallery (just the 13 names + one-line blurbs from `App.svelte`'s `themeBlurbs` const)
   - **MCP server** section: describe what it does, link to `cmd/mcp-test`, show the Claude Desktop config snippet
   - **Install** — initially "build from source" only; once releases work, add "download the latest installer"
   - **Build from source** — exact commands (`wails build`, `wails dev`)
   - **Tech stack** — Wails v2 / Go / Svelte 4 / CodeMirror 6 / goldmark / mcp-go (one line each)
   - **License**

4. **CHANGELOG.md** in Keep-a-Changelog format. Seed it with a `## [0.7.0] - <today's date>` entry listing every phase as a bullet under `### Added`. Use the HANDOFF.md status section as source material. Keep entries terse.

5. **CONTRIBUTING.md** — short. Cover: prerequisites (Go 1.23+, Node 18+, Wails CLI), how to run dev mode, how to run the MCP test, the coding conventions (lowercase command labels, hairline-stroke SVG icons, themes use `!important` on Svelte component-overridden props — link this last one to the HANDOFF gotchas section).

6. **`.github/workflows/ci.yml`** — runs on `pull_request` and `push` to `main`. Steps: gofmt check (`gofmt -l .` should produce no output); `go vet ./...`; `go build ./...`; `cd frontend && npm ci && npm run check && npm run build`; smoke a `wails build` on Linux only (fast). Use the `dAppServer/wails-build-action@v2.2` or current equivalent — check Marketplace.

7. **`.github/workflows/release.yml`** — runs on tag `v*`. Matrix-build for `windows-latest` (amd64 + NSIS installer), `macos-latest` (universal), `ubuntu-latest` (amd64). Use the official `wails-build-action`. Attach all artifacts to a GitHub Release using `softprops/action-gh-release`. Filename pattern: `upmark-<version>-<platform>-<arch>.<ext>`.

8. **Issue templates** — minimal: `.github/ISSUE_TEMPLATE/bug_report.md` and `feature_request.md`. Standard fields.

9. **Verify everything still builds.** `wails build -debug` should still produce a working `upmark.exe`. Don't break what works.

## Conventions to keep

- **Lowercase UI strings** throughout. Command palette commands are all lowercase ("open file", "narrower reading width"). UI labels follow the same convention. README headings can use normal title-case.
- **Hairline icons**: stroke-width 1.3–1.4, hand-rolled inline SVG, no icon library. If anyone proposes Lucide, push back unless they're swapping all ~30 existing icons together.
- **Themes are CSS-var override blocks**, not separate stylesheets. Anything new added to `themes.css` must respect the `[data-theme=...]` pattern.
- **`!important` is the documented escape hatch** for theme overrides on properties Svelte components also declare. Use it; don't try to out-specificity Svelte 4's double-hash.

## What I want from you in this sprint

- **Status updates as you go** — one sentence per concrete deliverable when it lands. Don't summarize until you're done.
- **Pause and ask** before pushing anywhere (`git remote add`, `git push`), before publishing a release, or before any destructive operation.
- **Don't add features.** If you see something tempting, drop it into a follow-up issue once the repo is initialized and the issue templates exist.

When you're done with all 9 deliverables, give me:
1. A summary of what shipped
2. The exact commands to push to a new GitHub remote and cut the v0.7.0 release
3. Anything in the HANDOFF.md "deferred / known limitations" section that you think should be promoted to a real issue
