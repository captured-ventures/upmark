#!/usr/bin/env node
/**
 * upmark MCP bridge — stdio MCP server that proxies to upmark's SSE endpoint.
 *
 * Why this exists: Claude Desktop, Claude Code, and Claude.ai don't currently
 * accept localhost SSE MCP servers — only stdio (Claude Desktop/Code) or
 * remote HTTPS (Claude.ai). upmark's MCP server is local SSE. This bridge
 * lets stdio-only clients talk to it.
 *
 * Phase 2 scope (this file): connect to an already-running upmark, mirror
 * its tools, proxy stdio calls through. If upmark isn't running, exit with
 * a clear error — Phase 3 will add the auto-launch path.
 *
 * Layout:
 *   - Acts as an MCP *client* against upmark via SSEClientTransport
 *   - Acts as an MCP *server* against the host (Claude) via StdioServerTransport
 *   - Tools, capabilities, and notifications are mirrored both ways
 *
 * Configuration:
 *   UPMARK_MCP_URL    full SSE endpoint URL (default: http://127.0.0.1:11451/sse)
 *   UPMARK_MCP_PORT   shorthand for port-only override (default: 11451)
 */

import { readFileSync, existsSync } from 'node:fs'
import { homedir, platform } from 'node:os'
import { join } from 'node:path'
import process from 'node:process'

import { Client } from '@modelcontextprotocol/sdk/client/index.js'
import { SSEClientTransport } from '@modelcontextprotocol/sdk/client/sse.js'
import { Server } from '@modelcontextprotocol/sdk/server/index.js'
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js'
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ListResourcesRequestSchema,
  ReadResourceRequestSchema,
  ListPromptsRequestSchema,
  GetPromptRequestSchema,
} from '@modelcontextprotocol/sdk/types.js'

const DEFAULT_PORT = 11451

// ---------------------------------------------------------------------------
// Endpoint discovery
//
// Order of precedence:
//   1. UPMARK_MCP_URL env var (explicit override)
//   2. mcp.lock file in upmark's config dir (matches the live server)
//   3. UPMARK_MCP_PORT env var (port-only override)
//   4. Hardcoded default port
// ---------------------------------------------------------------------------

function upmarkConfigDir() {
  switch (platform()) {
    case 'win32':
      return join(process.env.APPDATA ?? join(homedir(), 'AppData', 'Roaming'), 'upmark')
    case 'darwin':
      return join(homedir(), 'Library', 'Application Support', 'upmark')
    default:
      return join(process.env.XDG_CONFIG_HOME ?? join(homedir(), '.config'), 'upmark')
  }
}

function readLockfile() {
  const lockPath = join(upmarkConfigDir(), 'mcp.lock')
  if (!existsSync(lockPath)) return null
  try {
    return JSON.parse(readFileSync(lockPath, 'utf8'))
  } catch {
    return null
  }
}

function resolveEndpoint() {
  if (process.env.UPMARK_MCP_URL) return process.env.UPMARK_MCP_URL
  const lock = readLockfile()
  if (lock?.url) return lock.url
  const port = parseInt(process.env.UPMARK_MCP_PORT ?? '', 10) || DEFAULT_PORT
  return `http://127.0.0.1:${port}/sse`
}

// ---------------------------------------------------------------------------
// Bridge wiring
// ---------------------------------------------------------------------------

function logErr(...parts) {
  // stderr only — stdout is reserved for the MCP protocol stream.
  process.stderr.write(`[upmark-bridge] ${parts.join(' ')}\n`)
}

async function main() {
  const endpoint = resolveEndpoint()
  logErr('connecting to', endpoint)

  // 1. Client side: connect to upmark over SSE.
  const upstream = new Client(
    { name: 'upmark-bridge-client', version: '0.7.0' },
    { capabilities: {} },
  )
  const sseTransport = new SSEClientTransport(new URL(endpoint))
  try {
    await upstream.connect(sseTransport)
  } catch (e) {
    logErr('failed to connect:', e?.message ?? e)
    logErr('is upmark running with the MCP server enabled?')
    process.exit(1)
  }

  const serverCapabilities = upstream.getServerCapabilities() ?? {}
  const serverVersion = upstream.getServerVersion()
  logErr(
    'connected to',
    serverVersion?.name ?? 'unknown',
    serverVersion?.version ?? '?',
  )

  // 2. Server side: speak MCP over stdio to whatever launched us (Claude).
  //
  // We advertise the same capabilities upmark exposed upstream, so the host
  // doesn't see a degraded interface compared to talking to upmark directly.
  const downstream = new Server(
    { name: 'upmark', version: serverVersion?.version ?? '0.7.0' },
    { capabilities: serverCapabilities },
  )

  // ---- tools ----
  if (serverCapabilities.tools) {
    downstream.setRequestHandler(ListToolsRequestSchema, async () => {
      const result = await upstream.listTools()
      return { tools: result.tools }
    })
    downstream.setRequestHandler(CallToolRequestSchema, async (req) => {
      return await upstream.callTool({
        name: req.params.name,
        arguments: req.params.arguments,
      })
    })
  }

  // ---- resources ----
  if (serverCapabilities.resources) {
    downstream.setRequestHandler(ListResourcesRequestSchema, async () => {
      return await upstream.listResources()
    })
    downstream.setRequestHandler(ReadResourceRequestSchema, async (req) => {
      return await upstream.readResource({ uri: req.params.uri })
    })
  }

  // ---- prompts ----
  if (serverCapabilities.prompts) {
    downstream.setRequestHandler(ListPromptsRequestSchema, async () => {
      return await upstream.listPrompts()
    })
    downstream.setRequestHandler(GetPromptRequestSchema, async (req) => {
      return await upstream.getPrompt({
        name: req.params.name,
        arguments: req.params.arguments,
      })
    })
  }

  // ---- forward upstream-originated notifications downstream ----
  //
  // upmark emits a few notifications (resources/list_changed, etc.) when its
  // state shifts. Forward them so the host stays in sync.
  upstream.fallbackNotificationHandler = async (notification) => {
    try {
      await downstream.notification(notification)
    } catch (e) {
      logErr('forward notification failed:', e?.message ?? e)
    }
  }

  // ---- lifecycle ----
  //
  // If upmark goes away (server-side stream closes), exit. The host will
  // typically re-launch us on the next tool call.
  sseTransport.onclose = () => {
    logErr('upmark connection closed; exiting')
    process.exit(0)
  }
  sseTransport.onerror = (err) => {
    logErr('upstream error:', err?.message ?? err)
  }

  const stdioTransport = new StdioServerTransport()
  await downstream.connect(stdioTransport)
  logErr('bridge online — proxying stdio <-> sse')
}

main().catch((e) => {
  logErr('fatal:', e?.message ?? e)
  process.exit(1)
})
