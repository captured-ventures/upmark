<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { MCPStatus } from './types'

  // Theme keys list mirrors what App.svelte tracks; kept in sync via prop.
  type ThemeKey =
    | 'editorial' | 'broadsheet' | 'newsprint' | 'terminal'
    | 'manuscript' | 'brutalist' | 'arcade'
    | 'pastoral' | 'architect' | 'vapor'
    | 'typewriter' | 'midnight' | 'gameboy'

  type WidthKey = 'narrow' | 'normal' | 'wide'

  export let open: boolean = false
  export let theme: ThemeKey
  export let readingWidth: WidthKey
  export let fontSize: number
  export let mcpStatus: MCPStatus

  const dispatch = createEventDispatcher<{
    setTheme: ThemeKey
    setWidth: WidthKey
    setFontSize: number
    toggleMCP: void
    setMCPPort: number
    copyMCPURL: void
  }>()

  let section: 'appearance' | 'mcp' | 'about' = 'appearance'
  const widthOptions: WidthKey[] = ['narrow', 'normal', 'wide']

  function onFontRange(e: Event) {
    const v = parseInt((e.target as HTMLInputElement).value, 10)
    if (!isNaN(v)) dispatch('setFontSize', v)
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      open = false
    }
  }

  // For the theme preview tiles. The tile renders inline with its own colors
  // instead of inheriting from data-theme, so the user sees how each theme
  // looks without applying it first.
  type TilePreview = {
    bg: string
    fg: string
    accent: string
    fontFamily: string
    fontStyle?: string
    fontWeight?: string
    textTransform?: string
    overlay?: string  // optional CSS background overlay (scan lines, etc.)
  }

  const themeOrder: ThemeKey[] = [
    'editorial', 'broadsheet', 'newsprint', 'terminal',
    'manuscript', 'brutalist', 'arcade',
    'pastoral', 'architect', 'vapor',
    'typewriter', 'midnight', 'gameboy',
  ]

  const previews: Record<ThemeKey, TilePreview> = {
    editorial:  { bg: '#faf8f3', fg: '#1a1a1c', accent: '#a8451c', fontFamily: 'Newsreader Variable, Newsreader, serif', fontStyle: 'italic' },
    broadsheet: { bg: '#fefcf7', fg: '#14110d', accent: '#6b3a16', fontFamily: 'Newsreader Variable, Newsreader, serif', fontWeight: '500' },
    newsprint:  { bg: '#e8e4dc', fg: '#0a0a0a', accent: '#c0312c', fontFamily: 'IBM Plex Sans, sans-serif', fontWeight: '700', textTransform: 'uppercase' },
    terminal:   { bg: '#0d1117', fg: '#cdd9e5', accent: '#f0883e', fontFamily: 'JetBrains Mono, monospace' },
    manuscript: { bg: '#f4e8d4', fg: '#2c1f12', accent: '#8b1e1e', fontFamily: 'EB Garamond, serif' },
    brutalist:  { bg: '#ffffff', fg: '#000000', accent: '#ff0033', fontFamily: 'IBM Plex Sans, sans-serif', fontWeight: '700', textTransform: 'uppercase' },
    arcade:     { bg: '#0a0118', fg: '#f8f8ff', accent: '#ff10f0', fontFamily: 'VT323, monospace',
                  overlay: 'repeating-linear-gradient(0deg, transparent 0, transparent 1px, rgba(0,0,0,0.4) 1px, rgba(0,0,0,0.4) 2px)' },
    pastoral:   { bg: '#faf6f0', fg: '#3a3530', accent: '#6b8e6f', fontFamily: 'Lora Variable, Lora, serif' },
    architect:  { bg: '#eef2f5', fg: '#1d3a52', accent: '#1d3a52', fontFamily: 'IBM Plex Mono, monospace',
                  overlay: 'linear-gradient(to right, rgba(29,58,82,0.10) 1px, transparent 1px), linear-gradient(to bottom, rgba(29,58,82,0.10) 1px, transparent 1px)' },
    vapor:      { bg: 'linear-gradient(135deg, #1a0b2e 0%, #38205a 100%)', fg: '#f6d8f5', accent: '#ff61d8', fontFamily: 'Caveat, cursive', fontWeight: '600' },
    typewriter: { bg: '#f4f0e6', fg: '#2a2620', accent: '#b71c1c', fontFamily: 'Courier Prime, monospace', fontWeight: '700' },
    midnight:   { bg: '#0d1b2a', fg: '#f0e6d2', accent: '#d4af37', fontFamily: 'Lora Variable, Lora, serif' },
    gameboy:    { bg: '#c4cfa1', fg: '#2a3a1c', accent: '#2a3a1c', fontFamily: 'VT323, monospace',
                  overlay: 'repeating-linear-gradient(0deg, transparent 0, transparent 1px, rgba(42,58,28,0.18) 1px, rgba(42,58,28,0.18) 2px)' },
  }

  function tileStyle(t: ThemeKey): string {
    const p = previews[t]
    const parts: string[] = []
    if (p.bg.startsWith('linear-gradient')) {
      parts.push(`background: ${p.bg}`)
    } else {
      parts.push(`background-color: ${p.bg}`)
    }
    if (p.overlay) {
      parts.push(`background-image: ${p.overlay}`)
      parts.push(`background-size: 100% 100%`)
    }
    parts.push(`color: ${p.fg}`)
    return parts.join('; ')
  }
  function previewTextStyle(t: ThemeKey): string {
    const p = previews[t]
    const s: string[] = [`font-family: ${p.fontFamily}`]
    if (p.fontStyle)     s.push(`font-style: ${p.fontStyle}`)
    if (p.fontWeight)    s.push(`font-weight: ${p.fontWeight}`)
    if (p.textTransform) s.push(`text-transform: ${p.textTransform}`)
    return s.join('; ')
  }
  function previewAccentStyle(t: ThemeKey): string {
    return `color: ${previews[t].accent}`
  }

  // ─── MCP port input ───
  let portDraft = String(mcpStatus.port)
  $: portDraft = String(mcpStatus.port)
  function applyPort() {
    const n = parseInt(portDraft, 10)
    if (n > 0 && n < 65536 && n !== mcpStatus.port) {
      dispatch('setMCPPort', n)
    }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <div class="settings-backdrop" on:click={() => (open = false)} role="dialog" aria-modal="true">
    <div class="settings-panel" on:click|stopPropagation>
      <header class="settings-header">
        <h2>Settings</h2>
        <button class="settings-close" on:click={() => (open = false)} title="Close (Esc)" aria-label="Close">
          <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M3 3l10 10M13 3L3 13"/></svg>
        </button>
      </header>

      <div class="settings-body">
        <nav class="settings-nav">
          <button class:active={section === 'appearance'} on:click={() => (section = 'appearance')}>appearance</button>
          <button class:active={section === 'mcp'}        on:click={() => (section = 'mcp')}>mcp server</button>
          <button class:active={section === 'about'}      on:click={() => (section = 'about')}>about</button>
        </nav>

        <div class="settings-section">
          {#if section === 'appearance'}
            <h3>theme</h3>
            <div class="theme-grid">
              {#each themeOrder as t (t)}
                <button
                  class="theme-tile"
                  class:active={theme === t}
                  on:click={() => dispatch('setTheme', t)}
                  title={t}
                >
                  <div class="tile-preview" style={tileStyle(t)}>
                    <div class="tile-text" style={previewTextStyle(t)}>Aa</div>
                    <div class="tile-bar" style={previewAccentStyle(t)}>●</div>
                  </div>
                  <span class="tile-name">{t}</span>
                </button>
              {/each}
            </div>

            <h3>reading width</h3>
            <div class="seg">
              {#each widthOptions as w (w)}
                <button
                  class="seg-btn"
                  class:active={readingWidth === w}
                  on:click={() => dispatch('setWidth', w)}
                >{w}</button>
              {/each}
            </div>

            <h3>font size</h3>
            <div class="font-row">
              <button class="step" on:click={() => dispatch('setFontSize', Math.max(12, fontSize - 1))}>−</button>
              <input
                type="range"
                min="12" max="26" step="1"
                value={fontSize}
                on:input={onFontRange}
              />
              <button class="step" on:click={() => dispatch('setFontSize', Math.min(26, fontSize + 1))}>+</button>
              <span class="font-val">{fontSize}px</span>
              <button class="reset" on:click={() => dispatch('setFontSize', 17)}>reset</button>
            </div>

          {:else if section === 'mcp'}
            <h3>local mcp server</h3>
            <p class="hint">
              Lets LLM clients (Claude Desktop, custom agents) push markdown documents
              into upmark for you to review. Off by default. Localhost-only.
            </p>

            <label class="row toggle-row">
              <span>enable server</span>
              <button
                class="toggle"
                class:on={mcpStatus.running}
                on:click={() => dispatch('toggleMCP')}
                aria-pressed={mcpStatus.running}
              >
                <span class="toggle-knob"></span>
              </button>
            </label>

            <label class="row">
              <span>port</span>
              <input
                type="number"
                bind:value={portDraft}
                on:blur={applyPort}
                on:keydown={(e) => e.key === 'Enter' && applyPort()}
                min="1" max="65535"
              />
            </label>

            {#if mcpStatus.running}
              <label class="row">
                <span>endpoint</span>
                <code class="url-display">{mcpStatus.url}</code>
                <button class="ghost-btn" on:click={() => dispatch('copyMCPURL')}>copy</button>
              </label>

              <details class="config-snippet">
                <summary>Claude Desktop configuration</summary>
                <pre><code>{`{
  "mcpServers": {
    "upmark": {
      "type": "sse",
      "url": "${mcpStatus.url}"
    }
  }
}`}</code></pre>
              </details>
            {/if}

          {:else if section === 'about'}
            <h3>upmark</h3>
            <p class="hint about-line">A reading tool for markdown · v0.7</p>
            <dl class="about-list">
              <dt>renderer</dt>
              <dd>goldmark (GFM, footnotes, typographer, emoji) + chroma highlighting</dd>
              <dt>editor</dt>
              <dd>CodeMirror 6 with markdown grammar</dd>
              <dt>fonts</dt>
              <dd>Newsreader · IBM Plex Sans · JetBrains Mono · EB Garamond · Lora · Caveat · Courier Prime · VT323</dd>
              <dt>protocol</dt>
              <dd>Model Context Protocol via SSE</dd>
              <dt>framework</dt>
              <dd>Wails v2 (Go + Svelte 4 + WebView2)</dd>
            </dl>
          {/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .settings-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.30);
    z-index: 1500;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 8vh;
    animation: fade 140ms ease;
  }
  @media (prefers-color-scheme: dark) {
    .settings-backdrop { background: rgba(0, 0, 0, 0.54); }
  }

  .settings-panel {
    width: 760px;
    max-width: 94vw;
    max-height: 84vh;
    background: var(--bg);
    border: 1px solid var(--rule-strong);
    border-radius: 10px;
    box-shadow: 0 24px 64px -20px rgba(0, 0, 0, 0.35), 0 4px 12px rgba(0, 0, 0, 0.14);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    animation: pop 220ms cubic-bezier(0.16, 1, 0.3, 1);
  }

  .settings-header {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px 12px;
    border-bottom: 1px solid var(--rule);
  }
  .settings-header h2 {
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 18px;
    margin: 0;
    color: var(--fg);
    font-variation-settings: "opsz" 20;
  }
  .settings-close {
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--fg-muted);
    border-radius: 4px;
    transition: background 120ms, color 120ms;
  }
  .settings-close:hover {
    background: var(--hover);
    color: var(--fg);
  }

  .settings-body {
    display: flex;
    flex: 1;
    min-height: 0;
  }

  .settings-nav {
    width: 168px;
    flex-shrink: 0;
    border-right: 1px solid var(--rule);
    padding: 16px 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .settings-nav button {
    width: 100%;
    text-align: left;
    padding: 7px 16px 7px 22px;
    font-family: var(--font-sans);
    font-size: 12.5px;
    color: var(--fg-muted);
    border-left: 2px solid transparent;
    transition: color 120ms, border-color 120ms;
  }
  .settings-nav button:hover { color: var(--fg); }
  .settings-nav button.active {
    color: var(--accent);
    border-left-color: var(--accent);
  }

  .settings-section {
    flex: 1;
    overflow-y: auto;
    padding: 18px 24px 28px;
    scrollbar-gutter: stable;
  }

  .settings-section h3 {
    font-family: var(--font-sans);
    font-size: 11px;
    font-weight: 500;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--fg-subtle);
    margin: 14px 0 10px;
  }
  .settings-section h3:first-child { margin-top: 0; }

  .hint {
    color: var(--fg-muted);
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 13px;
    line-height: 1.55;
    margin: 0 0 12px;
    font-variation-settings: "opsz" 14;
  }

  /* ─── theme grid ─── */
  .theme-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(108px, 1fr));
    gap: 10px;
    margin-bottom: 18px;
  }

  .theme-tile {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 0;
    background: transparent;
    border: 1px solid var(--rule);
    border-radius: 5px;
    overflow: hidden;
    transition: border-color 140ms, transform 140ms;
  }
  .theme-tile:hover { border-color: var(--rule-strong); }
  .theme-tile.active {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .tile-preview {
    aspect-ratio: 16 / 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 10px;
    overflow: hidden;
    position: relative;
  }
  .tile-text {
    font-size: 22px;
    line-height: 1;
    z-index: 1;
  }
  .tile-bar {
    font-size: 22px;
    line-height: 1;
    z-index: 1;
  }

  .tile-name {
    font-family: var(--font-sans);
    font-size: 11px;
    color: var(--fg-muted);
    padding: 4px 8px 6px;
    text-align: left;
    text-transform: lowercase;
    letter-spacing: 0.02em;
  }
  .theme-tile.active .tile-name { color: var(--accent); }

  /* ─── segmented control ─── */
  .seg {
    display: inline-flex;
    border: 1px solid var(--rule-strong);
    border-radius: 5px;
    overflow: hidden;
    margin-bottom: 18px;
  }
  .seg-btn {
    padding: 5px 16px;
    font-family: var(--font-sans);
    font-size: 12px;
    color: var(--fg-muted);
    border-right: 1px solid var(--rule-strong);
    transition: background 120ms, color 120ms;
    text-transform: lowercase;
  }
  .seg-btn:last-child { border-right: none; }
  .seg-btn:hover { color: var(--fg); }
  .seg-btn.active {
    background: var(--accent-soft);
    color: var(--accent);
  }

  /* ─── font row ─── */
  .font-row {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 18px;
  }
  .step {
    width: 26px;
    height: 26px;
    border: 1px solid var(--rule);
    border-radius: 4px;
    font-family: var(--font-sans);
    color: var(--fg-muted);
    transition: border-color 120ms, color 120ms;
  }
  .step:hover { border-color: var(--accent); color: var(--accent); }
  .font-row input[type="range"] {
    flex: 1;
    accent-color: var(--accent);
  }
  .font-val {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg-muted);
    font-variant-numeric: tabular-nums;
    min-width: 36px;
    text-align: right;
  }
  .reset {
    font-family: var(--font-sans);
    font-size: 11px;
    color: var(--fg-subtle);
    text-transform: lowercase;
    letter-spacing: 0.04em;
    padding: 2px 6px;
    transition: color 120ms;
  }
  .reset:hover { color: var(--accent); }

  /* ─── MCP section ─── */
  .row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 0;
    border-bottom: 1px solid var(--rule);
    font-family: var(--font-sans);
    font-size: 13px;
    color: var(--fg-muted);
  }
  .row:last-of-type { border-bottom: none; }
  .row > span { flex: 0 0 100px; color: var(--fg-muted); }
  .row input[type="number"] {
    width: 100px;
    padding: 4px 8px;
    background: var(--bg);
    color: var(--fg);
    border: 1px solid var(--rule);
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 12px;
  }
  .row input[type="number"]:focus {
    outline: none;
    border-color: var(--accent);
  }

  .toggle-row { gap: 16px; }
  .toggle {
    width: 36px;
    height: 20px;
    border-radius: 999px;
    background: var(--rule-strong);
    position: relative;
    transition: background 160ms ease;
    cursor: pointer;
  }
  .toggle-knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 16px;
    height: 16px;
    border-radius: 50%;
    background: var(--bg);
    transition: left 200ms cubic-bezier(0.32, 0.72, 0.16, 1);
  }
  .toggle.on { background: var(--accent); }
  .toggle.on .toggle-knob { left: 18px; }

  .url-display {
    flex: 1;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg);
    background: var(--bg-elev);
    border: 1px solid var(--rule);
    padding: 4px 8px;
    border-radius: 4px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ghost-btn {
    font-family: var(--font-sans);
    font-size: 11px;
    color: var(--accent);
    border: 1px solid var(--accent);
    border-radius: 4px;
    padding: 4px 10px;
    background: transparent;
    transition: background 120ms;
    text-transform: lowercase;
  }
  .ghost-btn:hover { background: var(--accent-soft); }

  .config-snippet {
    margin-top: 16px;
    border: 1px solid var(--rule);
    border-radius: 4px;
    overflow: hidden;
  }
  .config-snippet summary {
    padding: 8px 12px;
    cursor: pointer;
    font-family: var(--font-sans);
    font-size: 11px;
    color: var(--fg-muted);
    letter-spacing: 0.06em;
    text-transform: uppercase;
    list-style: none;
  }
  .config-snippet summary::-webkit-details-marker { display: none; }
  .config-snippet[open] summary { border-bottom: 1px solid var(--rule); }
  .config-snippet pre {
    margin: 0;
    padding: 10px 14px;
    background: var(--bg-elev);
    font-family: var(--font-mono);
    font-size: 11.5px;
    line-height: 1.5;
    color: var(--fg);
    overflow-x: auto;
  }

  /* ─── about ─── */
  .about-line {
    margin-bottom: 18px;
  }
  .about-list {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 8px 18px;
    font-family: var(--font-sans);
    font-size: 12.5px;
    margin: 0;
  }
  .about-list dt {
    color: var(--fg-subtle);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 10.5px;
    padding-top: 2px;
  }
  .about-list dd {
    margin: 0;
    color: var(--fg);
    line-height: 1.55;
  }

  @keyframes fade { from { opacity: 0; } to { opacity: 1; } }
  @keyframes pop {
    from { opacity: 0; transform: translateY(-10px) scale(0.98); }
    to   { opacity: 1; transform: translateY(0) scale(1); }
  }
</style>
