<script lang="ts">
  import { onMount, tick } from 'svelte'
  import type { Command } from './types'

  export let open: boolean = false
  export let commands: Command[] = []

  let query = ''
  let activeIdx = 0
  let inputEl: HTMLInputElement | undefined
  let listEl: HTMLDivElement | undefined

  type Scored = Command & { score: number }

  $: filtered = filterAndSort(commands, query)
  $: groups = groupBy(filtered)
  $: flat = groups.flatMap((g) => g.items)
  $: if (open) queueMicrotask(() => inputEl?.focus())
  $: if (!open) {
    query = ''
    activeIdx = 0
  }

  function score(label: string, q: string): number {
    if (!q) return 0
    const lower = label.toLowerCase()
    const idx = lower.indexOf(q)
    if (idx < 0) {
      // letter-by-letter subsequence?
      let li = 0, qi = 0
      while (li < lower.length && qi < q.length) {
        if (lower[li] === q[qi]) qi++
        li++
      }
      return qi === q.length ? 200 : -1
    }
    // Lower index = better; start-of-string strongly preferred.
    return idx === 0 ? 0 : 50 + idx
  }

  function filterAndSort(cmds: Command[], q: string): Scored[] {
    const lq = q.trim().toLowerCase()
    const scored: Scored[] = []
    for (const c of cmds) {
      const s = score((c.matchText ?? c.label).toLowerCase(), lq)
      if (s >= 0) scored.push({ ...c, score: s })
    }
    if (lq) scored.sort((a, b) => a.score - b.score)
    return scored
  }

  function groupBy(items: Scored[]) {
    const order: Command['group'][] = ['action', 'heading', 'folder', 'recent']
    const map: Record<string, Scored[]> = {}
    for (const it of items) {
      ;(map[it.group] ??= []).push(it)
    }
    return order
      .filter((g) => map[g]?.length)
      .map((g) => ({ group: g, items: map[g] }))
  }

  async function run(idx: number) {
    const c = flat[idx]
    if (!c) return
    open = false
    await tick()
    await c.run()
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault()
      open = false
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      activeIdx = (activeIdx + 1) % Math.max(flat.length, 1)
      scrollActiveIntoView()
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      activeIdx = (activeIdx - 1 + flat.length) % Math.max(flat.length, 1)
      scrollActiveIntoView()
    } else if (e.key === 'Enter') {
      e.preventDefault()
      run(activeIdx)
    }
  }

  function scrollActiveIntoView() {
    queueMicrotask(() => {
      const el = listEl?.querySelector('.cmd-row.active') as HTMLElement | null
      el?.scrollIntoView({ block: 'nearest' })
    })
  }

  $: if (query !== undefined) activeIdx = 0

  const groupLabel: Record<string, string> = {
    action: 'actions',
    heading: 'in document',
    folder: 'in folder',
    recent: 'recent',
  }

  // Compute the running index for each row so click handlers can hit it directly.
  function rowIdx(groupIdx: number, itemIdxInGroup: number): number {
    let i = 0
    for (let g = 0; g < groupIdx; g++) i += groups[g].items.length
    return i + itemIdxInGroup
  }
</script>

{#if open}
  <div class="palette-backdrop" on:click={() => (open = false)}>
    <div class="palette" on:click|stopPropagation>
      <div class="palette-inner">
        <div class="cmd-input-row">
          <span class="cmd-prompt">⌘</span>
          <input
            bind:this={inputEl}
            bind:value={query}
            on:keydown={onKey}
            placeholder="type to search · ↑↓ to navigate · ↵ to run"
            type="text"
            autocomplete="off"
            spellcheck="false"
          />
          <span class="cmd-esc">esc</span>
        </div>
        <div class="cmd-list" bind:this={listEl}>
          {#if flat.length === 0}
            <div class="cmd-empty">no matches</div>
          {/if}
          {#each groups as g, gi (g.group)}
            <div class="cmd-group-label">{groupLabel[g.group]}</div>
            {#each g.items as item, ii (item.id)}
              {@const idx = rowIdx(gi, ii)}
              <button
                class="cmd-row"
                class:active={idx === activeIdx}
                on:mouseenter={() => (activeIdx = idx)}
                on:click={() => run(idx)}
              >
                <span class="cmd-label">{item.label}</span>
                {#if item.hint}
                  <span class="cmd-hint">{item.hint}</span>
                {/if}
              </button>
            {/each}
          {/each}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .palette-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.18);
    z-index: 1000;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 14vh;
    animation: bdfade 140ms ease;
  }
  @media (prefers-color-scheme: dark) {
    .palette-backdrop { background: rgba(0, 0, 0, 0.42); }
  }

  .palette {
    width: 560px;
    max-width: 92vw;
    max-height: 72vh;
    background: var(--bg);
    border: 1px solid var(--rule-strong);
    border-radius: 8px;
    box-shadow: 0 24px 60px -20px rgba(0, 0, 0, 0.35),
                0 2px 8px rgba(0, 0, 0, 0.12);
    overflow: hidden;
    display: flex;
    flex-direction: column;
    animation: pal-in 200ms cubic-bezier(0.16, 1, 0.3, 1);
  }

  .palette-inner {
    display: flex;
    flex-direction: column;
    min-height: 0;
    flex: 1;
  }

  .cmd-input-row {
    display: flex;
    align-items: center;
    padding: 0 16px;
    height: 48px;
    flex-shrink: 0;       /* never let a long list compress the input row */
    border-bottom: 1px solid var(--rule);
    gap: 12px;
  }
  .cmd-prompt {
    font-family: var(--font-mono);
    color: var(--accent);
    font-size: 14px;
    font-weight: 500;
  }
  .cmd-input-row input {
    flex: 1;
    /* Without min-width: 0 the input refuses to shrink below the intrinsic
       width of its placeholder text, which stretches the palette wider than
       560px (especially obvious under mono themes). */
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--fg);
    font-family: var(--font-sans);
    font-size: 14px;
    padding: 0;
  }
  .cmd-input-row input::placeholder {
    color: var(--fg-subtle);
    font-style: italic;
    font-family: var(--font-serif);
    font-size: 13px;
    font-variation-settings: "opsz" 14;
  }
  .cmd-esc {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--fg-subtle);
    padding: 2px 6px;
    border: 1px solid var(--rule);
    border-radius: 3px;
  }

  .cmd-list {
    flex: 1;
    min-height: 0;          /* allow flex item to shrink and become scrollable */
    overflow-y: auto;
    padding: 6px 0 8px;
  }

  .cmd-empty {
    padding: 24px;
    text-align: center;
    color: var(--fg-subtle);
    font-style: italic;
    font-family: var(--font-serif);
    font-size: 14px;
  }

  .cmd-group-label {
    padding: 10px 18px 4px;
    font-family: var(--font-sans);
    font-size: 10px;
    font-weight: 500;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--fg-subtle);
  }

  .cmd-row {
    display: flex;
    align-items: center;
    width: 100%;
    text-align: left;
    padding: 7px 18px;
    gap: 12px;
    font-family: var(--font-sans);
    color: var(--fg-muted);
    border-left: 2px solid transparent;
    transition: background 80ms ease, color 80ms ease, border-color 80ms ease;
  }

  .cmd-row:hover, .cmd-row.active {
    background: var(--hover);
    color: var(--fg);
    border-left-color: var(--accent);
  }

  .cmd-label {
    flex: 1;
    font-size: 13.5px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .cmd-hint {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg-subtle);
    letter-spacing: 0.02em;
  }

  @keyframes bdfade {
    from { opacity: 0; }
    to { opacity: 1; }
  }
  @keyframes pal-in {
    from { opacity: 0; transform: translateY(-8px) scale(0.99); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }
</style>
