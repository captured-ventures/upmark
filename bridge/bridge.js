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

import { spawn } from 'node:child_process'
import { readFileSync, existsSync } from 'node:fs'
import { homedir, platform } from 'node:os'
import { delimiter, join } from 'node:path'
import process from 'node:process'
import { setTimeout as sleep } from 'node:timers/promises'

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
const LAUNCH_TIMEOUT_MS = 10_000
const LAUNCH_POLL_INITIAL_MS = 100
const LAUNCH_POLL_MAX_MS = 500

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
// Probe a candidate SSE endpoint to see if upmark is already running there.
// AbortController gives us a hard timeout so a half-dead server doesn't hang
// the bridge on the first connection attempt.
// ---------------------------------------------------------------------------

async function probeEndpoint(url, timeoutMs = 750) {
  const ctrl = new AbortController()
  const t = setTimeout(() => ctrl.abort(), timeoutMs)
  try {
    const res = await fetch(url, {
      method: 'GET',
      headers: { Accept: 'text/event-stream' },
      signal: ctrl.signal,
    })
    // Any response (even an error code) means a server is answering on the
    // port. SSE servers stream forever, so cancel the body as soon as headers
    // arrive — we don't want to actually read the stream here.
    res.body?.cancel().catch(() => {})
    return true
  } catch {
    return false
  } finally {
    clearTimeout(t)
  }
}

// ---------------------------------------------------------------------------
// Binary discovery
//
// MCPB extensions ship the bridge but not upmark itself — the user installs
// upmark separately via the platform installer. To launch upmark when no
// server is running, the bridge has to find the binary.
//
// Order of precedence:
//   1. UPMARK_BIN env var (explicit override; used by tests and power users)
//   2. Platform default install location (NSIS / Applications / standard FHS)
//   3. PATH lookup
//
// Returns the first existing path, or null if nothing was found.
// ---------------------------------------------------------------------------

function isExecutable(path) {
  try {
    return existsSync(path)
  } catch {
    return false
  }
}

function platformDefaults() {
  switch (platform()) {
    case 'win32': {
      // NSIS installs to "$PROGRAMFILES64\Captured Ventures\upmark\upmark.exe"
      // per build/windows/installer/project.nsi. Also probe the 32-bit
      // location and LocalAppData as user-mode fallbacks.
      const candidates = []
      if (process.env.PROGRAMFILES) {
        candidates.push(join(process.env.PROGRAMFILES, 'Captured Ventures', 'upmark', 'upmark.exe'))
      }
      if (process.env['PROGRAMFILES(X86)']) {
        candidates.push(join(process.env['PROGRAMFILES(X86)'], 'Captured Ventures', 'upmark', 'upmark.exe'))
      }
      if (process.env.LOCALAPPDATA) {
        candidates.push(join(process.env.LOCALAPPDATA, 'Programs', 'upmark', 'upmark.exe'))
      }
      return candidates
    }
    case 'darwin':
      return [
        '/Applications/upmark.app/Contents/MacOS/upmark',
        join(homedir(), 'Applications', 'upmark.app', 'Contents', 'MacOS', 'upmark'),
      ]
    default:
      return [
        '/usr/local/bin/upmark',
        '/usr/bin/upmark',
        join(homedir(), '.local', 'bin', 'upmark'),
      ]
  }
}

function searchPath() {
  const PATH = process.env.PATH ?? ''
  const exeName = platform() === 'win32' ? 'upmark.exe' : 'upmark'
  const dirs = PATH.split(delimiter).filter(Boolean)
  for (const d of dirs) {
    const candidate = join(d, exeName)
    if (isExecutable(candidate)) return candidate
  }
  return null
}

function findUpmarkBinary() {
  if (process.env.UPMARK_BIN) {
    if (isExecutable(process.env.UPMARK_BIN)) return process.env.UPMARK_BIN
    logErr('UPMARK_BIN points to a missing file:', process.env.UPMARK_BIN)
  }
  for (const candidate of platformDefaults()) {
    if (isExecutable(candidate)) return candidate
  }
  return searchPath()
}

// ---------------------------------------------------------------------------
// Launch upmark in --mcp-server mode and wait for its SSE endpoint to come
// up. The launched process is fully detached + unref'd so the bridge can
// exit later without killing the user's reading session.
// ---------------------------------------------------------------------------

async function launchAndWait(binaryPath, endpoint) {
  logErr('launching', binaryPath, '--mcp-server')
  // detached + unref so killing the bridge doesn't take upmark with it.
  // ignore stdio: don't inherit any handles the parent (Claude) is using.
  const child = spawn(binaryPath, ['--mcp-server'], {
    detached: true,
    stdio: 'ignore',
    windowsHide: true,
  })
  child.on('error', (err) => {
    logErr('spawn error:', err?.message ?? err)
  })
  child.unref()

  // Poll the SSE endpoint with light exponential backoff until it responds
  // or the timeout elapses. upmark cold-starts in ~1-2s typically.
  const deadline = Date.now() + LAUNCH_TIMEOUT_MS
  let delay = LAUNCH_POLL_INITIAL_MS
  while (Date.now() < deadline) {
    if (await probeEndpoint(endpoint)) return true
    await sleep(delay)
    delay = Math.min(delay * 1.5, LAUNCH_POLL_MAX_MS)
  }
  return false
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
  logErr('endpoint:', endpoint)

  // 1. Auto-launch if upmark isn't already listening. The lockfile is a hint;
  // a quick HTTP probe is the source of truth (handles stale lockfiles from
  // force-killed processes).
  if (!(await probeEndpoint(endpoint))) {
    logErr('no live server — trying to launch')
    const binary = findUpmarkBinary()
    if (!binary) {
      logErr('could not find upmark binary')
      logErr('set UPMARK_BIN to override, or install upmark from the GitHub release')
      process.exit(1)
    }
    const launched = await launchAndWait(binary, endpoint)
    if (!launched) {
      logErr('upmark launched but SSE endpoint never came up within', LAUNCH_TIMEOUT_MS, 'ms')
      process.exit(1)
    }
    logErr('upmark is up')
  }

  // 2. Client side: connect to upmark over SSE.
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
