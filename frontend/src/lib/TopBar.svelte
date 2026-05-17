<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import {
    MinimizeWindow,
    ToggleMaximizeWindow,
    CloseWindow,
  } from '../../wailsjs/go/main/App'

  export let docName: string = ''
  export let sidebarOpen: boolean = true
  export let editing: boolean = false
  export let readOnly: boolean = false
  export let isMCP: boolean = false
  export let mcpClient: string = ''
  export let mcpRunning: boolean = false
  export let focusActive: boolean = false

  const dispatch = createEventDispatcher<{
    toggleSidebar: void
    open: void
    find: void
    palette: void
    copy: void
    print: void
    edit: void
    settings: void
    toggleFocus: void
  }>()

  $: docDisplay = docName.replace(/\.(md|markdown|mdown|mkd|mdx)$/i, '')

  let copied = false
  async function copyHtml() {
    const body = document.querySelector('.markdown-body') as HTMLElement | null
    if (!body) return
    try {
      const html = body.innerHTML
      const text = body.innerText
      if ('ClipboardItem' in window) {
        await navigator.clipboard.write([
          new ClipboardItem({
            'text/html': new Blob([html], { type: 'text/html' }),
            'text/plain': new Blob([text], { type: 'text/plain' }),
          }),
        ])
      } else {
        await navigator.clipboard.writeText(html)
      }
      copied = true
      setTimeout(() => (copied = false), 1200)
    } catch (e) {
      console.error(e)
    }
  }
</script>

<header class="topbar drag" on:dblclick={() => ToggleMaximizeWindow()}>
  <div class="anchor no-drag" class:sidebar-open={sidebarOpen} class:mcp={isMCP}>
    <button class="tb-action" on:click={() => dispatch('toggleSidebar')} title="Toggle sidebar (Ctrl B)" aria-label="Toggle sidebar">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
        <rect x="2" y="3" width="12" height="10" rx="1" />
        <line x1="6.5" y1="3" x2="6.5" y2="13" />
      </svg>
    </button>
    <span class="doc-name">
      {#if docDisplay}
        <span class="doc">{docDisplay}</span>
      {:else}
        <span class="wordmark">upmark</span>
      {/if}
    </span>
  </div>

  {#if isMCP}
    <div class="mcp-banner no-drag" title="Presented via MCP — task list is interactive">
      <span class="mcp-chip">mcp</span>
      {#if mcpClient}
        <span class="mcp-client">{mcpClient}</span>
      {/if}
    </div>
  {/if}

  <div class="drag-fill" aria-hidden="true"></div>

  <div class="cluster right no-drag">
    <button class="tb-action palette-btn" on:click={() => dispatch('palette')} title="Command palette (Ctrl K)" aria-label="Command palette">
      <span class="cmd-glyph">⌘</span>
    </button>
    <button class="tb-action" on:click={() => dispatch('open')} title="Open file (Ctrl O)" aria-label="Open">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
        <path d="M2 4.5a1 1 0 0 1 1-1h3l1.5 1.5h5a1 1 0 0 1 1 1v6a1 1 0 0 1-1 1H3a1 1 0 0 1-1-1z" />
      </svg>
    </button>
    {#if docName && !readOnly}
      <button
        class="tb-action"
        class:active={editing}
        on:click={() => dispatch('edit')}
        title={editing ? 'Exit edit mode (Ctrl E)' : 'Edit (Ctrl E)'}
        aria-label={editing ? 'Exit edit mode' : 'Edit'}
      >
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
          <path d="M11 2.5l2.5 2.5L5 13.5l-3 0.5L2.5 11l8.5-8.5z"/>
          <path d="M9.5 4l2.5 2.5"/>
        </svg>
      </button>
    {/if}
    {#if docName}
      <button class="tb-action" on:click={() => dispatch('find')} title="Find (Ctrl F)" aria-label="Find">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
          <circle cx="7" cy="7" r="4.2" /><path d="M10 10l3 3" />
        </svg>
      </button>
      <button class="tb-action" on:click={copyHtml} title={copied ? 'Copied' : 'Copy rendered'} aria-label="Copy">
        {#if copied}
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M3 8l3.5 3.5L13 5" /></svg>
        {:else}
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
            <rect x="4.5" y="4.5" width="8.5" height="9" rx="1" />
            <path d="M3 11V3.5a1 1 0 0 1 1-1H10" />
          </svg>
        {/if}
      </button>
      <button class="tb-action" on:click={() => window.print()} title="Print (Ctrl P)" aria-label="Print">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
          <path d="M4 6V2.5h8V6M4 11H2.5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h11a1 1 0 0 1 1 1v3a1 1 0 0 1-1 1H12M4 9.5h8v4H4z" />
        </svg>
      </button>
    {/if}
    <button class="tb-action settings-btn" on:click={() => dispatch('settings')} title="Settings (Ctrl ,)" aria-label="Settings">
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4">
        <line x1="2.5" y1="4" x2="13.5" y2="4"/>
        <line x1="2.5" y1="8" x2="13.5" y2="8"/>
        <line x1="2.5" y1="12" x2="13.5" y2="12"/>
        <circle cx="6"  cy="4"  r="1.6" fill="var(--bg)"/>
        <circle cx="10" cy="8"  r="1.6" fill="var(--bg)"/>
        <circle cx="5"  cy="12" r="1.6" fill="var(--bg)"/>
      </svg>
    </button>
    <button
      class="tb-action"
      class:active={focusActive}
      on:click={() => dispatch('toggleFocus')}
      title={focusActive ? 'Exit focus mode (Esc or Ctrl Shift F)' : 'Focus mode (Ctrl Shift F)'}
      aria-label={focusActive ? 'Exit focus mode' : 'Focus mode'}
    >
      <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round">
        <rect x="6" y="4" width="4" height="8" rx="0.5"/>
        <polyline points="6.5,2.5 8,1 9.5,2.5"/>
        <polyline points="6.5,13.5 8,15 9.5,13.5"/>
      </svg>
    </button>
    <div class="window-controls">
      <button class="win-ctrl" on:click={() => MinimizeWindow()} title="Minimize" aria-label="Minimize">
        <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 5h10" stroke="currentColor" stroke-width="1" fill="none"/></svg>
      </button>
      <button class="win-ctrl" on:click={() => ToggleMaximizeWindow()} title="Maximize" aria-label="Maximize">
        <svg width="10" height="10" viewBox="0 0 10 10"><rect x="0.5" y="0.5" width="9" height="9" stroke="currentColor" stroke-width="1" fill="none"/></svg>
      </button>
      <button class="win-ctrl close" on:click={() => CloseWindow()} title="Close" aria-label="Close">
        <svg width="10" height="10" viewBox="0 0 10 10"><path d="M0 0l10 10M10 0L0 10" stroke="currentColor" stroke-width="1" fill="none"/></svg>
      </button>
    </div>
  </div>
</header>

<style>
  .topbar {
    height: 40px;
    flex: 0 0 40px;
    display: flex;
    align-items: stretch;
    background: var(--bg);
    border-bottom: 1px solid var(--rule);
    color: var(--fg);
    z-index: 20;
  }

  /* The "anchor" cluster sits over the sidebar column. When the sidebar is open
     it sizes to 252px (sidebar width) and gets a right rule that lines up with
     the sidebar's right edge. When closed it just hugs its contents. */
  .anchor {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 0 12px 0 10px;
    flex-shrink: 0;
    width: auto;
    transition: width 240ms cubic-bezier(0.32, 0.72, 0.16, 1),
                border-color 240ms ease;
    border-right: 1px solid transparent;
    min-width: 0;
  }
  .anchor.sidebar-open {
    width: 252px;
    border-right-color: var(--rule);
  }

  .doc-name {
    overflow: hidden;
    min-width: 0;
  }
  .doc {
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: 14px;
    font-variation-settings: "opsz" 14;
    color: var(--fg);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: block;
  }
  .wordmark {
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 14px;
    color: var(--fg-muted);
    font-variation-settings: "opsz" 14;
  }
  .mcp-chip {
    display: inline-block;
    font-family: var(--font-mono);
    font-size: 9px;
    letter-spacing: 0.12em;
    text-transform: uppercase;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: 3px;
    padding: 1px 5px;
    line-height: 1;
    vertical-align: middle;
  }

  /* Banner sits immediately right of the sidebar divider when an LLM has
     pushed a doc into upmark. Pairs the chip with the originating client
     name (e.g. "Claude Desktop") so you can see *who* is driving. */
  .mcp-banner {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 14px;
    flex-shrink: 0;
  }
  .mcp-client {
    font-family: var(--font-sans);
    font-size: 12px;
    color: var(--fg-muted);
  }

  /* When the active doc is MCP-presented, tint the .anchor strip (sits over
     the sidebar column in the topbar) with the theme's accent — ambient
     signal that this window is being driven externally. Foreground colors
     invert to read against the tint. */
  .anchor.mcp {
    background: var(--accent);
  }
  .anchor.mcp .doc,
  .anchor.mcp .wordmark,
  .anchor.mcp .tb-action {
    color: var(--bg);
  }
  .anchor.mcp .tb-action:hover {
    color: var(--bg);
    background: rgba(0, 0, 0, 0.12);
  }

  /* The center band is pure drag region — eats all available space between
     the anchor and the action cluster so the user has somewhere to grab. */
  .drag-fill {
    flex: 1;
    min-width: 0;
    height: 100%;
  }

  .cluster.right {
    display: flex;
    align-items: center;
    gap: 2px;
    padding: 0;
    flex-shrink: 0;
  }

  .tb-action {
    width: 30px;
    height: 30px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--fg-muted);
    border-radius: 4px;
    transition: background 120ms ease, color 120ms ease;
    margin: 0 1px;
  }
  .tb-action:hover {
    background: var(--hover);
    color: var(--fg);
  }
  .tb-action.active {
    color: var(--accent);
    background: var(--accent-soft);
  }
  .tb-action.active:hover {
    background: var(--accent-soft);
    color: var(--accent);
  }
  .tb-action:disabled {
    color: var(--fg-subtle);
    opacity: 0.4;
    cursor: default;
  }

  /* Palette button — mirrors the ⌘ prompt glyph used inside the palette */
  .tb-action.palette-btn {
    margin-right: 4px;
  }
  .cmd-glyph {
    font-family: var(--font-mono);
    font-size: 14px;
    font-weight: 500;
    color: var(--accent);
    line-height: 1;
    transition: opacity 120ms ease;
  }
  .tb-action.palette-btn:hover .cmd-glyph {
    opacity: 0.75;
  }

  .settings-btn {
    margin-left: 6px;     /* small visual break between doc actions and app/window */
  }

  .window-controls {
    display: flex;
    align-items: stretch;
    align-self: stretch;     /* override .cluster.right's align-items: center
                                so we fill the topbar's 40px height */
    margin-left: 8px;
  }
  .win-ctrl {
    width: 44px;
    height: 100%;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--fg-muted);
    transition: background 120ms ease, color 120ms ease;
  }
  .win-ctrl:hover { background: var(--hover); color: var(--fg); }
  .win-ctrl.close:hover {
    background: var(--accent-soft);
    color: var(--accent);
  }
</style>
