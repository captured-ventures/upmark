<script lang="ts">
  import { tick, onDestroy, createEventDispatcher } from 'svelte'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
  import { OpenPath, ResolveWikilink, SetScrollPos, GetScrollPos } from '../../wailsjs/go/main/App'
  import {
    enhanceCallouts,
    enhanceMath,
    enhanceMermaid,
    enhanceHeadingAnchors,
    enhanceCodeCopy,
    enhanceFootnotes,
    enableTaskList,
    countWords,
    readingMinutes,
    findInPage,
    highlightMatches,
    clearHighlights,
    scrollToInViewer,
    type FindMatch,
  } from './enhance'
  import ImageLightbox from './ImageLightbox.svelte'
  import type { TocItem } from './types'

  export let html: string
  export let baseDir: string
  export let docPath: string
  export let docName: string
  export let findOpen: boolean = false
  export let mcpId: string = ''   // non-empty → MCP-presented doc, task list becomes interactive

  const dispatch = createEventDispatcher<{
    toc: TocItem[]
    activeHeading: { id: string; text: string }
    openPath: string
    mcpTaskToggle: { docId: string; taskId: number; checked: boolean }
  }>()

  let container: HTMLDivElement | undefined
  let findInput: HTMLInputElement | undefined
  let query = ''
  let matches: FindMatch[] = []
  let activeIdx = 0
  let dragOver = false
  let readPct = 0
  let wordCount = 0
  let readMin = 0

  // Lightbox state
  let lbOpen = false
  let lbSrc = ''
  let lbAlt = ''

  // Scroll-save debounce
  let scrollSaveTimer: number | undefined

  $: if (html) {
    queueMicrotask(() => renderPipeline())
  }

  $: if (findOpen) {
    queueMicrotask(() => findInput?.focus())
  } else {
    if (container) clearHighlights(container)
    query = ''
    matches = []
  }

  async function renderPipeline() {
    if (!container) return
    await tick()
    const body = container.querySelector<HTMLElement>('.markdown-body')
    if (!body) return

    // Run enhancements that don't depend on async work first.
    enhanceCallouts(body)
    enhanceMath(body)
    enhanceHeadingAnchors(body, docName)
    enhanceCodeCopy(body)
    enhanceFootnotes(body)

    // For MCP-presented docs, make task-list checkboxes interactive and
    // wire each to the parent via an event.
    if (mcpId) {
      body.dataset.mcp = '1'
      enableTaskList(body, mcpId, (docId, taskId, checked) => {
        dispatch('mcpTaskToggle', { docId, taskId, checked })
      })
    }

    // Word count from the rendered, callout-stripped text.
    const text = body.innerText ?? ''
    wordCount = countWords(text)
    readMin = readingMinutes(wordCount)

    extractToc(body)

    // Reset scroll, then await mermaid (which can change height) before
    // restoring saved scroll. That way the restore lands on the right spot.
    container.scrollTop = 0
    await enhanceMermaid(body).catch((e) => console.error('mermaid:', e))
    await restoreScroll()
    updateActiveHeading()
    updateReadingProgress()
  }

  function extractToc(body: HTMLElement) {
    const hs = body.querySelectorAll<HTMLElement>('h2, h3, h4')
    const items: TocItem[] = []
    hs.forEach((h) => {
      if (!h.id) return
      // strip out the anchor "#" we just appended
      const cloned = h.cloneNode(true) as HTMLElement
      cloned.querySelectorAll('.hd-anchor').forEach((a) => a.remove())
      items.push({
        id: h.id,
        text: cloned.textContent?.trim() ?? '',
        level: parseInt(h.tagName.slice(1)),
      })
    })
    dispatch('toc', items)
  }

  // Active-heading update — throttled to ~20fps so dispatching to App and
  // re-rendering the sidebar TOC during smooth scroll doesn't churn class
  // attributes 60 times per second.
  let lastActiveId = ''
  let activeHeadingPending = false
  let activeHeadingNextAllowed = 0

  function updateActiveHeading() {
    const now = performance.now()
    if (now < activeHeadingNextAllowed) {
      if (!activeHeadingPending) {
        activeHeadingPending = true
        const wait = Math.max(0, activeHeadingNextAllowed - now)
        setTimeout(() => {
          activeHeadingPending = false
          updateActiveHeadingNow()
        }, wait)
      }
      return
    }
    activeHeadingNextAllowed = now + 50
    updateActiveHeadingNow()
  }

  function updateActiveHeadingNow() {
    if (!container) return
    const body = container.querySelector<HTMLElement>('.markdown-body')
    if (!body) return
    const hs = Array.from(body.querySelectorAll<HTMLElement>('h1, h2, h3, h4'))
    if (hs.length === 0) {
      if (lastActiveId !== '') {
        lastActiveId = ''
        dispatch('activeHeading', { id: '', text: '' })
      }
      return
    }
    const containerTop = container.getBoundingClientRect().top
    const triggerLine = 64
    let active: HTMLElement = hs[0]
    for (const h of hs) {
      const relTop = h.getBoundingClientRect().top - containerTop
      if (relTop <= triggerLine) active = h
      else break
    }
    if (active.id && active.id !== lastActiveId) {
      lastActiveId = active.id
      const cloned = active.cloneNode(true) as HTMLElement
      cloned.querySelectorAll('.hd-anchor').forEach((a) => a.remove())
      dispatch('activeHeading', { id: active.id, text: cloned.textContent?.trim() ?? '' })
    }
  }

  function updateReadingProgress() {
    if (!container) return
    const sh = container.scrollHeight - container.clientHeight
    readPct = sh > 0 ? Math.min(100, Math.max(0, (container.scrollTop / sh) * 100)) : 0
  }

  async function restoreScroll() {
    if (!container || !docPath) return
    try {
      const pct = await GetScrollPos(docPath)
      if (pct > 0 && pct < 1) {
        const sh = container.scrollHeight - container.clientHeight
        container.scrollTop = Math.round(pct * sh)
      }
    } catch (e) { console.error('restore scroll:', e) }
  }

  function saveScrollDebounced() {
    if (!container || !docPath) return
    if (scrollSaveTimer) clearTimeout(scrollSaveTimer)
    scrollSaveTimer = window.setTimeout(() => {
      if (!container) return
      const sh = container.scrollHeight - container.clientHeight
      const pct = sh > 0 ? container.scrollTop / sh : 0
      SetScrollPos(docPath, pct).catch(console.error)
    }, 350)
  }

  function onScroll() {
    updateActiveHeading()
    updateReadingProgress()
    saveScrollDebounced()
  }

  async function handleClick(ev: MouseEvent) {
    let target = ev.target as HTMLElement | null
    while (target && target !== container) {
      // Image click → lightbox (external URLs and local both)
      if (target.tagName === 'IMG') {
        const img = target as HTMLImageElement
        const src = img.getAttribute('src') ?? ''
        if (src) {
          ev.preventDefault()
          lbSrc = src
          lbAlt = img.getAttribute('alt') ?? ''
          lbOpen = true
        }
        return
      }

      if (target.tagName === 'A') {
        const a = target as HTMLAnchorElement
        const href = a.getAttribute('href') ?? ''
        if (!href) return

        // Heading anchor links (the # icons): let the dedicated handler in
        // enhance.ts process them — it already preventDefault'd, so by the
        // time we get here we just bail.
        if (a.classList.contains('hd-anchor')) return

        ev.preventDefault()

        if (href.startsWith('#')) {
          scrollToInViewer(href.slice(1))
          return
        }

        if (href.startsWith('upmark:wikilink/')) {
          const targetName = decodeURIComponent(href.slice('upmark:wikilink/'.length))
          try {
            const resolved = await ResolveWikilink(targetName)
            if (resolved) dispatch('openPath', resolved)
            else a.classList.add('broken')
          } catch (e) { console.error(e) }
          return
        }

        if (/^(https?:|mailto:)/i.test(href)) {
          BrowserOpenURL(href)
          return
        }

        if (href.startsWith('/local-asset/')) {
          const rest = href.slice('/local-asset/'.length)
          const slash = rest.indexOf('/')
          if (slash < 0) return
          const base = decodeURIComponent(rest.slice(0, slash))
          const rel = decodeURIComponent(rest.slice(slash + 1)).split('#')[0]
          const targetPath = joinPath(base, rel)
          if (/\.(md|markdown|mdown|mkd|mdx)$/i.test(targetPath)) {
            dispatch('openPath', targetPath)
          } else {
            BrowserOpenURL('file:///' + targetPath.replace(/\\/g, '/'))
          }
          return
        }
        return
      }
      target = target.parentElement
    }
  }

  function joinPath(base: string, rel: string): string {
    const sep = base.includes('\\') ? '\\' : '/'
    const parts = (base + sep + rel).split(/[\\/]+/)
    const stack: string[] = []
    for (const p of parts) {
      if (p === '..') stack.pop()
      else if (p !== '.' && p !== '') stack.push(p)
    }
    if (sep === '\\') return stack.join('\\')
    return '/' + stack.join('/')
  }

  function runFind() {
    if (!container) return
    clearHighlights(container)
    matches = findInPage(container, query)
    activeIdx = 0
    if (matches.length > 0) {
      highlightMatches(matches, 0)
      scrollToActive()
    }
  }

  function step(dir: 1 | -1) {
    if (matches.length === 0) return
    activeIdx = (activeIdx + dir + matches.length) % matches.length
    if (!container) return
    clearHighlights(container)
    matches = findInPage(container, query)
    if (matches.length === 0) return
    if (activeIdx >= matches.length) activeIdx = 0
    highlightMatches(matches, activeIdx)
    scrollToActive()
  }

  function scrollToActive() {
    requestAnimationFrame(() => {
      if (!container) return
      const active = container.querySelector('mark.find-hit.active') as HTMLElement | null
      if (active) scrollToInViewer(active, { block: 'center' })
    })
  }

  function onFindKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (matches.length === 0) runFind()
      else step(e.shiftKey ? -1 : 1)
    } else if (e.key === 'Escape') {
      findOpen = false
    }
  }

  function onDragEnter(e: DragEvent) {
    if (e.dataTransfer?.types?.includes('Files')) dragOver = true
  }
  function onDragLeave(e: DragEvent) {
    if (!container?.contains(e.relatedTarget as Node)) dragOver = false
  }
  function onDrop() { dragOver = false }

  onDestroy(() => {
    if (container) clearHighlights(container)
    if (scrollSaveTimer) clearTimeout(scrollSaveTimer)
  })
</script>

<div
  class="viewer"
  class:drag-over={dragOver}
  tabindex="-1"
  bind:this={container}
  on:click={handleClick}
  on:scroll={onScroll}
  on:dragenter={onDragEnter}
  on:dragleave={onDragLeave}
  on:dragover|preventDefault
  on:drop={onDrop}
>
  {@html html}
  <div class="progress-rail" aria-hidden="true">
    <div class="progress-fill" style={`height: ${readPct}%`}></div>
  </div>
</div>

{#if wordCount > 0}
  <div class="doc-stats" aria-hidden="true">
    {wordCount.toLocaleString()}w · {readMin}m
  </div>
{/if}

<ImageLightbox bind:open={lbOpen} src={lbSrc} alt={lbAlt} />

{#if findOpen}
  <div class="find no-drag">
    <input
      bind:this={findInput}
      bind:value={query}
      on:input={runFind}
      on:keydown={onFindKey}
      placeholder="find"
      type="text"
    />
    <span class="find-count">
      {#if query && matches.length === 0}
        no matches
      {:else if matches.length > 0}
        {activeIdx + 1} / {matches.length}
      {/if}
    </span>
    <button on:click={() => step(-1)} title="Previous (Shift Enter)" aria-label="Previous">
      <svg width="11" height="11" viewBox="0 0 12 12"><path d="M3 7l3-3 3 3" stroke="currentColor" stroke-width="1.4" fill="none"/></svg>
    </button>
    <button on:click={() => step(1)} title="Next (Enter)" aria-label="Next">
      <svg width="11" height="11" viewBox="0 0 12 12"><path d="M3 5l3 3 3-3" stroke="currentColor" stroke-width="1.4" fill="none"/></svg>
    </button>
    <button on:click={() => (findOpen = false)} title="Close (Esc)" aria-label="Close">
      <svg width="11" height="11" viewBox="0 0 12 12"><path d="M2 2l8 8M10 2L2 10" stroke="currentColor" stroke-width="1.4" fill="none"/></svg>
    </button>
  </div>
{/if}

<style>
  .viewer {
    flex: 1;
    overflow: auto;
    background: var(--bg);
    position: relative;
    transition: background 200ms ease;
  }

  .viewer.drag-over { background: var(--paper-tint); }
  .viewer.drag-over::before {
    content: "drop to open";
    position: fixed;
    left: 50%;
    top: 50%;
    transform: translate(-50%, -50%);
    pointer-events: none;
    z-index: 5;
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 24px;
    color: var(--accent);
    background: var(--bg);
    padding: 14px 28px;
    border: 1px dashed var(--accent);
    border-radius: 4px;
    font-variation-settings: "opsz" 24;
  }

  .progress-rail {
    position: fixed;
    right: 6px;
    top: 48px;
    bottom: 28px;
    width: 1px;
    background: transparent;
    pointer-events: none;
  }
  .progress-fill {
    width: 1px;
    background: var(--accent);
    opacity: 0.55;
    transition: height 80ms linear;
  }

  .doc-stats {
    position: fixed;
    bottom: 10px;
    right: 18px;
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--fg-subtle);
    letter-spacing: 0.04em;
    pointer-events: none;
    font-variant-numeric: tabular-nums;
    z-index: 1;
  }

  .find {
    position: fixed;
    top: 50px;
    right: 18px;
    display: flex;
    align-items: center;
    gap: 4px;
    background: var(--bg);
    border: 1px solid var(--rule-strong);
    padding: 5px 6px 5px 10px;
    border-radius: 6px;
    box-shadow: 0 8px 24px -8px rgba(0, 0, 0, 0.25);
    z-index: 100;
  }
  .find input {
    border: none; outline: none; background: transparent;
    color: var(--fg);
    font-family: var(--font-sans);
    font-size: 13px;
    width: 180px;
  }
  .find input::placeholder {
    color: var(--fg-subtle);
    font-style: italic;
    font-family: var(--font-serif);
  }
  .find-count {
    color: var(--fg-subtle);
    font-family: var(--font-mono);
    font-size: 10.5px;
    min-width: 50px;
    text-align: right;
    margin-right: 2px;
  }
  .find button {
    color: var(--fg-muted);
    padding: 4px;
    border-radius: 3px;
    transition: background 100ms ease, color 100ms ease;
  }
  .find button:hover { background: var(--hover); color: var(--fg); }
</style>
