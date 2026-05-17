<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { OpenFolderDialog } from '../../wailsjs/go/main/App'
  import type { RecentEntry, Folder } from './types'

  export let recent: RecentEntry[] = []

  const dispatch = createEventDispatcher<{
    open: void
    openPath: string
    setFolder: Folder
  }>()

  function fmtDate(ms: number) {
    if (!ms) return ''
    const d = new Date(ms)
    const now = new Date()
    const sameDay = d.toDateString() === now.toDateString()
    if (sameDay) return 'today'
    const yest = new Date(now)
    yest.setDate(now.getDate() - 1)
    if (d.toDateString() === yest.toDateString()) return 'yesterday'
    const sameYear = d.getFullYear() === now.getFullYear()
    return d.toLocaleDateString(undefined, sameYear ? { month: 'short', day: 'numeric' } : { year: 'numeric', month: 'short', day: 'numeric' })
  }

  function stripExt(name: string) {
    return name.replace(/\.(md|markdown|mdown|mkd|mdx)$/i, '')
  }

  async function pickFolder() {
    try {
      const f = await OpenFolderDialog()
      if (f && f.root) dispatch('setFolder', f as Folder)
    } catch (e) {
      console.error(e)
    }
  }
</script>

<div class="empty">
  <div class="empty-page">
    <div class="masthead">
      <h1 class="logo">upmark</h1>
      <hr class="rule" />
      <p class="tagline"><em>a reading tool for markdown</em></p>
    </div>

    <nav class="actions">
      <button class="action" on:click={() => dispatch('open')}>
        <span class="action-label">open a file</span>
        <span class="action-hint">⌃ O</span>
      </button>
      <button class="action" on:click={pickFolder}>
        <span class="action-label">open a folder</span>
        <span class="action-hint">⌃ ⇧ O</span>
      </button>
      <button class="action" on:click={() => dispatch('open')}>
        <span class="action-label">or drop a file anywhere</span>
        <span class="action-hint">⇊</span>
      </button>
    </nav>

    {#if recent.length > 0}
      <section class="recent">
        <h2 class="section-label">recent</h2>
        <ol class="recent-list">
          {#each recent.slice(0, 8) as r, i (r.path)}
            <li>
              <button class="row" on:click={() => dispatch('openPath', r.path)} title={r.path}>
                <span class="idx">{String(i + 1).padStart(2, '0')}</span>
                <span class="name">{stripExt(r.name)}</span>
                <span class="dot">·</span>
                <span class="meta">{fmtDate(r.openedAt)}</span>
              </button>
            </li>
          {/each}
        </ol>
      </section>
    {/if}

    <footer class="foot">
      <span>press</span>
      <kbd>⌃ K</kbd>
      <span>for the command palette</span>
    </footer>
  </div>
</div>

<style>
  .empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 48px 32px 64px;
    overflow: auto;
  }

  .empty-page {
    width: 100%;
    max-width: 480px;
  }

  .masthead {
    text-align: center;
    margin-bottom: 56px;
  }

  .logo {
    font-family: var(--font-serif);
    font-weight: 400;
    font-style: italic;
    font-variation-settings: "opsz" 60;
    font-size: 64px;
    line-height: 1;
    letter-spacing: -0.025em;
    color: var(--fg);
    margin: 0;
  }

  .rule {
    width: 56px;
    border: none;
    border-top: 1px solid var(--rule-strong);
    margin: 16px auto 12px;
  }

  .tagline {
    font-family: var(--font-serif);
    font-size: 14px;
    color: var(--fg-muted);
    font-variation-settings: "opsz" 14;
    margin: 0;
  }

  .actions {
    display: flex;
    flex-direction: column;
    margin-bottom: 56px;
  }

  .action {
    display: flex;
    align-items: baseline;
    padding: 10px 0;
    border-bottom: 1px solid var(--rule);
    transition: padding-left 200ms cubic-bezier(0.16, 1, 0.3, 1);
  }
  .action:hover {
    padding-left: 8px;
  }
  .action:first-child { border-top: 1px solid var(--rule); }

  .action-label {
    flex: 1;
    text-align: left;
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 18px;
    font-variation-settings: "opsz" 18;
    color: var(--fg);
    transition: color 120ms ease;
  }
  .action:hover .action-label { color: var(--accent); }

  .action-hint {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg-subtle);
    letter-spacing: 0.02em;
  }

  .recent { margin-bottom: 56px; }

  .section-label {
    font-family: var(--font-sans);
    font-size: 10px;
    font-weight: 500;
    letter-spacing: 0.16em;
    text-transform: uppercase;
    color: var(--fg-subtle);
    text-align: center;
    margin: 0 0 16px;
  }

  .recent-list {
    list-style: none;
    margin: 0;
    padding: 0;
    font-family: var(--font-serif);
  }

  .row {
    display: flex;
    align-items: baseline;
    gap: 12px;
    width: 100%;
    padding: 6px 0;
    text-align: left;
    border-bottom: 1px solid var(--rule);
    font-size: 15px;
    color: var(--fg-muted);
    transition: color 120ms ease;
  }
  .row:hover { color: var(--fg); }
  .row:hover .idx { color: var(--accent); }

  .idx {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
    letter-spacing: 0;
    min-width: 22px;
  }
  .name {
    flex: 1;
    font-family: var(--font-serif);
    font-variation-settings: "opsz" 16;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .dot { color: var(--fg-subtle); font-size: 12px; }
  .meta {
    font-family: var(--font-sans);
    font-size: 11.5px;
    color: var(--fg-subtle);
    font-variant-numeric: tabular-nums;
  }

  .foot {
    text-align: center;
    margin-top: 32px;
    font-family: var(--font-sans);
    font-size: 11px;
    color: var(--fg-subtle);
    letter-spacing: 0.02em;
  }
  .foot kbd {
    font-family: var(--font-mono);
    font-size: 10px;
    padding: 2px 6px;
    border: 1px solid var(--rule);
    border-radius: 3px;
    color: var(--fg-muted);
    margin: 0 4px;
  }
</style>
