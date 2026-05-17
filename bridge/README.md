# upmark MCP bridge

A stdio MCP server that proxies to upmark's localhost SSE endpoint, so MCP clients that don't speak SSE (Claude Desktop, Claude Code, etc.) can still talk to upmark.

This folder is the working copy of what eventually gets packaged as `upmark.mcpb` and attached to GitHub Releases. For now it lives here so it can be developed and tested against `upmark --mcp-server`.

## Status

Phase 2 of [#9](https://github.com/captured-ventures/upmark/issues/9).

- [x] Phase 1: backend `--mcp-server` flag + lockfile + idle exit + window-show pref
- [x] Phase 2: stdio↔SSE proxy against a *running* upmark
- [ ] Phase 3: bridge auto-launches `upmark --mcp-server` when not running
- [ ] Phase 4: MCPB packaging (`manifest.json` + zip → `upmark.mcpb`)
- [ ] Phase 5: release workflow attaches `.mcpb` to GitHub Releases

## Run it

You need upmark already running with the MCP server enabled. The simplest way is:

```sh
# in one terminal — starts upmark headless and runs the MCP server
upmark.exe --mcp-server
```

Then in another terminal:

```sh
cd bridge
npm install      # one-time
node bridge.js   # speaks stdio MCP, proxies to the running upmark
```

Useful for connecting a stdio-only MCP client (Claude Desktop, MCP Inspector, etc.).

## Endpoint discovery

In priority order:

1. `UPMARK_MCP_URL` env var (full SSE URL — override for testing)
2. `mcp.lock` in upmark's config directory (matches the live server's port)
3. `UPMARK_MCP_PORT` env var (port-only shorthand)
4. Default port `11451`

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
