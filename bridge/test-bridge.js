#!/usr/bin/env node
/**
 * Bridge smoke test.
 *
 * Spawns bridge.js as a stdio subprocess (the same way Claude Desktop would)
 * and exercises the proxy:
 *   - initialize handshake
 *   - tools/list (should mirror upmark's tools)
 *   - present_document call (should return a document_id)
 *
 * Run with upmark.exe --mcp-server already running on the default port.
 *
 *   node test-bridge.js
 */

import { Client } from '@modelcontextprotocol/sdk/client/index.js'
import { StdioClientTransport } from '@modelcontextprotocol/sdk/client/stdio.js'

const EXPECTED_TOOLS = [
  'present_document',
  'update_document',
  'get_document_status',
  'close_document',
  'list_presented',
]

async function main() {
  // StdioClientTransport uses a minimal "default environment" for security
  // unless env is supplied explicitly. Forward our env so UPMARK_BIN /
  // UPMARK_MCP_URL / UPMARK_MCP_PORT overrides reach the bridge.
  const transport = new StdioClientTransport({
    command: process.execPath,
    args: ['./bridge.js'],
    env: Object.fromEntries(
      Object.entries(process.env).filter(([, v]) => v !== undefined),
    ),
  })

  const client = new Client(
    { name: 'bridge-smoke-test', version: '0.1.0' },
    { capabilities: {} },
  )
  await client.connect(transport)
  console.log('✓ connected to bridge')

  const { tools } = await client.listTools()
  console.log(`✓ tools/list returned ${tools.length} tools`)
  const names = tools.map((t) => t.name).sort()
  console.log('  →', names.join(', '))

  for (const expected of EXPECTED_TOOLS) {
    if (!names.includes(expected)) {
      throw new Error(`expected tool not mirrored: ${expected}`)
    }
  }
  console.log('✓ all upmark tools mirrored')

  const presentResult = await client.callTool({
    name: 'present_document',
    arguments: {
      content: '# Bridge smoke test\n\nIf you can see this in upmark, the bridge works.\n',
      title: 'Bridge smoke test',
    },
  })
  const text = presentResult?.content?.[0]?.text ?? ''
  console.log('✓ present_document returned')
  console.log('  →', text.slice(0, 200))

  await client.close()
  console.log('✓ disconnected cleanly')
}

main().catch((e) => {
  console.error('✗ failed:', e?.message ?? e)
  process.exit(1)
})
