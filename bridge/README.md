# upmark MCP bridge

A stdio MCP server that proxies to upmark's localhost SSE endpoint, so MCP clients that don't speak SSE (Claude Desktop, Claude Code, etc.) can still talk to upmark.

This folder is the working copy of what eventually gets packaged as `upmark.mcpb` and attached to GitHub Releases. For now it lives here so it can be developed and tested against `upmark --mcp-server`.

## Status

Closes [#9](https://github.com/captured-ventures/upmark/issues/9). All 5 phases shipped.

- [x] Phase 1: backend `--mcp-server` flag + lockfile + idle exit + window-show pref
- [x] Phase 2: stdio↔SSE proxy against a *running* upmark
- [x] Phase 3: bridge auto-launches `upmark --mcp-server` when not running
- [x] Phase 4: MCPB packaging (`manifest.json` + zip → `upmark.mcpb`)
- [x] Phase 5: release workflow attaches `.mcpb` to GitHub Releases

## Pack a .mcpb locally

```sh
cd bridge
npm install     # one-time, gets the MCP SDK
npm run pack    # produces upmark.mcpb in this directory
```

`upmark.mcpb` is a zip containing `manifest.json`, `bridge.js`, the icon, and `node_modules`. Install via Claude Desktop → Settings → Extensions → Install Extension → pick the file. (The current Extensions pane is file-picker only; no drag/drop.) Tools appear immediately; first tool call triggers the bridge → auto-launches upmark → proxies.

If you're testing against a dev build instead of an installed upmark, set the `upmark_bin` user-config in the extension's settings to the absolute path of your `upmark.exe` (or set the `UPMARK_BIN` env var). Otherwise the bridge probes only the NSIS / Applications / standard PATH locations.

## Run it

```sh
cd bridge
npm install      # one-time
node bridge.js   # speaks stdio MCP, proxies to upmark
```

If upmark is already running, the bridge connects to its SSE endpoint. If not, it locates the upmark binary, launches it in `--mcp-server` mode, polls until the endpoint is up (~1-2s typical cold start), then proxies. The launched upmark is detached — bridge exit doesn't take it down, the user may still be reading.

Useful for connecting a stdio-only MCP client (Claude Desktop, MCP Inspector, etc.).

## Endpoint discovery

For the SSE URL the bridge connects to, in priority order:

1. `UPMARK_MCP_URL` env var (full SSE URL — override for testing)
2. `mcp.lock` in upmark's config directory (matches the live server's port)
3. `UPMARK_MCP_PORT` env var (port-only shorthand)
4. Default port `11451`

## Binary discovery (Phase 3 auto-launch)

If no live server is found, the bridge needs to launch upmark. It looks for the binary in:

1. `UPMARK_BIN` env var (explicit path override)
2. Platform default install locations:
   - **Windows:** `%PROGRAMFILES%\Captured Ventures\upmark\upmark.exe`, `%PROGRAMFILES(X86)%\Captured Ventures\upmark\upmark.exe`, `%LOCALAPPDATA%\Programs\upmark\upmark.exe`
   - **macOS:** `/Applications/upmark.app/Contents/MacOS/upmark`, `~/Applications/upmark.app/Contents/MacOS/upmark`
   - **Linux:** `/usr/local/bin/upmark`, `/usr/bin/upmark`, `~/.local/bin/upmark`
3. `PATH` search (`upmark` / `upmark.exe`)

If none of these work, the bridge exits with a clear error directing the user to install upmark or set `UPMARK_BIN`.

## Smoke test

```sh
node test-bridge.js
```

Spawns `bridge.js` as a stdio subprocess, connects via the MCP SDK's `StdioClientTransport`, exercises `tools/list` and `present_document`. Same pattern Claude Desktop uses.

## Architecture

```
┌─────────────────┐        stdio MCP        ┌──────────────┐        SSE         ┌────────────┐
│ Claude Desktop  │ ◄──────(JSON-RPC)──────►│  bridge.js   │ ◄──(HTTP+SSE)────► │   upmark   │
│ Claude Code     │                          │  (this dir)  │                    │  (Wails)   │
│ MCP Inspector   │                          └──────────────┘                    └────────────┘
└─────────────────┘
```

Internally the bridge holds two MCP transports back-to-back:
- `StdioServerTransport` — talks to whatever spawned it
- `SSEClientTransport` — talks to upmark

Capabilities, tools, resources, prompts, and notifications are mirrored both ways. Anything the SDK supports is forwarded transparently — no per-tool wiring.
