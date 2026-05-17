import katex from 'katex'
import 'katex/contrib/mhchem'

let mermaidPromise: Promise<any> | null = null
let mermaidThemeIsDark: boolean | null = null

// Decide whether mermaid should use its dark or light palette by reading the
// currently-applied --bg CSS variable. This tracks upmark's actual rendered
// surface (different themes can resolve to light or dark, some auto-track the
// OS preference), so swapping themes mid-session can re-derive cleanly.
function isCurrentThemeDark(): boolean {
  const bg = getComputedStyle(document.documentElement).getPropertyValue('--bg').trim()
  const m = bg.match(/^#([0-9a-f]{3,8})$/i) ?? bg.match(/^rgba?\(([^)]+)\)$/i)
  if (m) {
    let r = 0, g = 0, b = 0
    if (bg.startsWith('#')) {
      const hex = m[1].length === 3 ? m[1].split('').map((c) => c + c).join('') : m[1]
      r = parseInt(hex.slice(0, 2), 16)
      g = parseInt(hex.slice(2, 4), 16)
      b = parseInt(hex.slice(4, 6), 16)
    } else {
      const parts = m[1].split(',').map((s) => parseFloat(s.trim()))
      r = parts[0]; g = parts[1]; b = parts[2]
    }
    return (r * 0.299 + g * 0.587 + b * 0.114) / 255 < 0.5
  }
  return matchMedia('(prefers-color-scheme: dark)').matches
}

async function getMermaid() {
  if (!mermaidPromise) {
    mermaidPromise = import('mermaid').then((m) => {
      const mermaid = m.default
      mermaidThemeIsDark = isCurrentThemeDark()
      mermaid.initialize({
        startOnLoad: false,
        theme: mermaidThemeIsDark ? 'dark' : 'default',
        securityLevel: 'strict',
      })
      return mermaid
    })
  }
  return mermaidPromise
}

// Called from refreshMermaid: re-initialize mermaid if the active theme's
// light/dark resolution has flipped since the last init.
function ensureMermaidThemeMatches(mermaid: any): boolean {
  const dark = isCurrentThemeDark()
  if (mermaidThemeIsDark === dark) return false
  mermaidThemeIsDark = dark
  mermaid.initialize({
    startOnLoad: false,
    theme: dark ? 'dark' : 'default',
    securityLevel: 'strict',
  })
  return true
}

const CALLOUT_KINDS = ['note', 'tip', 'important', 'warning', 'caution'] as const
const CALLOUT_RE = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]\s*/i

export function enhanceCallouts(root: HTMLElement) {
  const quotes = root.querySelectorAll('blockquote')
  quotes.forEach((bq) => {
    const firstP = bq.querySelector('p')
    if (!firstP) return
    const text = firstP.textContent ?? ''
    const m = text.match(CALLOUT_RE)
    if (!m) return
    const kind = m[1].toLowerCase()
    if (!(CALLOUT_KINDS as readonly string[]).includes(kind)) return

    // Strip the [!KIND] marker from the first paragraph.
    // First child node may be a text node or contain it.
    let removed = false
    for (const node of Array.from(firstP.childNodes)) {
      if (node.nodeType === Node.TEXT_NODE && node.textContent) {
        const replaced = node.textContent.replace(CALLOUT_RE, '')
        if (replaced !== node.textContent) {
          node.textContent = replaced
          removed = true
          break
        }
      }
    }
    if (!removed) firstP.innerHTML = firstP.innerHTML.replace(CALLOUT_RE, '')

    // Wrap.
    const wrap = document.createElement('div')
    wrap.className = `callout callout-${kind}`
    const title = document.createElement('div')
    title.className = 'callout-title'
    title.innerHTML = `${calloutIcon(kind)} <span>${kind[0].toUpperCase() + kind.slice(1)}</span>`
    wrap.appendChild(title)
    // Move children.
    while (bq.firstChild) wrap.appendChild(bq.firstChild)
    bq.replaceWith(wrap)

    // If the first p is now empty, drop it.
    const fp = wrap.querySelector('p')
    if (fp && !fp.textContent?.trim() && !fp.querySelector('img,a,code')) fp.remove()
  })
}

function calloutIcon(kind: string): string {
  const map: Record<string, string> = {
    note: '<svg width="14" height="14" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6.5" stroke="currentColor" stroke-width="1.3" fill="none"/><path d="M8 4.5v4M8 11h0.01" stroke="currentColor" stroke-width="1.5" fill="none"/></svg>',
    tip: '<svg width="14" height="14" viewBox="0 0 16 16"><path d="M8 1.5a5 5 0 0 0-3 9v2h6v-2a5 5 0 0 0-3-9zM6 14h4" stroke="currentColor" stroke-width="1.3" fill="none"/></svg>',
    important: '<svg width="14" height="14" viewBox="0 0 16 16"><path d="M8 1L1 14h14L8 1z" stroke="currentColor" stroke-width="1.3" fill="none"/><path d="M8 6v4M8 12h0.01" stroke="currentColor" stroke-width="1.5"/></svg>',
    warning: '<svg width="14" height="14" viewBox="0 0 16 16"><path d="M8 1L1 14h14L8 1z" stroke="currentColor" stroke-width="1.3" fill="none"/><path d="M8 6v4M8 12h0.01" stroke="currentColor" stroke-width="1.5"/></svg>',
    caution: '<svg width="14" height="14" viewBox="0 0 16 16"><circle cx="8" cy="8" r="6.5" stroke="currentColor" stroke-width="1.3" fill="none"/><path d="M5 5l6 6M11 5L5 11" stroke="currentColor" stroke-width="1.5"/></svg>',
  }
  return map[kind] ?? ''
}

export function enhanceMath(root: HTMLElement) {
  const inline = root.querySelectorAll('.math-inline')
  inline.forEach((el) => {
    const tex = el.textContent ?? ''
    try {
      katex.render(tex, el as HTMLElement, { throwOnError: false, displayMode: false })
    } catch (e) {
      console.error('katex inline:', e)
    }
  })
  const display = root.querySelectorAll('.math-display')
  display.forEach((el) => {
    const tex = el.textContent ?? ''
    try {
      katex.render(tex, el as HTMLElement, { throwOnError: false, displayMode: true })
    } catch (e) {
      console.error('katex display:', e)
    }
  })
}

// Heading anchor icons — # appears on hover, click copies a markdown link.
export function enhanceHeadingAnchors(root: HTMLElement, fileName: string) {
  const headings = root.querySelectorAll<HTMLElement>('h1, h2, h3, h4, h5, h6')
  headings.forEach((h) => {
    if (!h.id || h.querySelector('.hd-anchor')) return
    const a = document.createElement('a')
    a.className = 'hd-anchor'
    a.href = `#${h.id}`
    a.title = 'Copy link'
    a.textContent = '#'
    a.addEventListener('click', async (e) => {
      e.preventDefault()
      e.stopPropagation()
      const text = (h.firstChild?.textContent ?? h.textContent ?? '').trim()
      const md = `[${text}](${fileName}#${h.id})`
      try {
        await navigator.clipboard.writeText(md)
        a.classList.add('copied')
        setTimeout(() => a.classList.remove('copied'), 1200)
      } catch (err) {
        console.error('copy heading link:', err)
      }
    })
    h.appendChild(a)
  })
}

// Code block copy buttons — fade in on hover over the <pre>.
export function enhanceCodeCopy(root: HTMLElement) {
  const blocks = root.querySelectorAll<HTMLElement>('pre')
  blocks.forEach((pre) => {
    if (pre.classList.contains('mermaid')) return
    if (pre.querySelector('.code-copy')) return
    pre.classList.add('has-copy')
    const btn = document.createElement('button')
    btn.className = 'code-copy'
    btn.type = 'button'
    btn.title = 'Copy'
    const iconCopy = '<svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="4.5" y="4.5" width="8.5" height="9" rx="1"/><path d="M3 11V3.5a1 1 0 0 1 1-1H10"/></svg>'
    const iconCheck = '<svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M3 8l3.5 3.5L13 5"/></svg>'
    btn.innerHTML = iconCopy
    btn.addEventListener('click', async (e) => {
      e.preventDefault()
      e.stopPropagation()
      const code = pre.querySelector('code')
      const text = code?.textContent ?? pre.textContent ?? ''
      try {
        await navigator.clipboard.writeText(text)
        btn.classList.add('copied')
        btn.innerHTML = iconCheck
        setTimeout(() => {
          btn.classList.remove('copied')
          btn.innerHTML = iconCopy
        }, 1400)
      } catch (err) {
        console.error('copy code:', err)
      }
    })
    pre.appendChild(btn)
  })
}

// Footnote tooltips on hover — reuses a single floating element.
let footnoteTooltip: HTMLElement | null = null
let footnoteHideTimer: number | null = null

function ensureFootnoteTooltip(): HTMLElement {
  if (!footnoteTooltip) {
    footnoteTooltip = document.createElement('div')
    footnoteTooltip.className = 'footnote-tooltip'
    footnoteTooltip.addEventListener('mouseenter', () => {
      if (footnoteHideTimer) {
        clearTimeout(footnoteHideTimer)
        footnoteHideTimer = null
      }
    })
    footnoteTooltip.addEventListener('mouseleave', hideFootnoteTooltip)
    document.body.appendChild(footnoteTooltip)
  }
  return footnoteTooltip
}

function showFootnoteTooltip(ref: HTMLElement, target: Element) {
  const tip = ensureFootnoteTooltip()
  const clone = target.cloneNode(true) as HTMLElement
  clone.querySelectorAll('.footnote-backref, a[href^="#fnref"]').forEach((b) => b.remove())
  tip.innerHTML = clone.innerHTML
  const rect = ref.getBoundingClientRect()
  // Position above the ref if there's not enough room below
  tip.style.left = `${Math.max(8, rect.left - 8)}px`
  // Render hidden first to measure
  tip.style.top = '0px'
  tip.classList.add('visible')
  const tipRect = tip.getBoundingClientRect()
  const spaceBelow = window.innerHeight - rect.bottom
  if (spaceBelow < tipRect.height + 16 && rect.top > tipRect.height + 16) {
    tip.style.top = `${rect.top - tipRect.height - 6}px`
  } else {
    tip.style.top = `${rect.bottom + 6}px`
  }
  // Clamp horizontally
  const left = parseFloat(tip.style.left)
  const maxLeft = window.innerWidth - tipRect.width - 8
  if (left > maxLeft) tip.style.left = `${Math.max(8, maxLeft)}px`
}

function hideFootnoteTooltip() {
  if (footnoteHideTimer) clearTimeout(footnoteHideTimer)
  footnoteHideTimer = window.setTimeout(() => {
    footnoteTooltip?.classList.remove('visible')
  }, 140)
}

export function enhanceFootnotes(root: HTMLElement) {
  const refs = root.querySelectorAll<HTMLElement>('a.footnote-ref, sup[id^="fnref"] > a')
  refs.forEach((ref) => {
    if (ref.dataset.fnEnhanced) return
    const href = ref.getAttribute('href') ?? ''
    if (!href.startsWith('#')) return
    const id = href.slice(1)
    const target = root.querySelector(`[id="${CSS.escape(id)}"]`)
    if (!target) return
    ref.dataset.fnEnhanced = '1'
    ref.addEventListener('mouseenter', () => {
      if (footnoteHideTimer) { clearTimeout(footnoteHideTimer); footnoteHideTimer = null }
      showFootnoteTooltip(ref, target)
    })
    ref.addEventListener('mouseleave', hideFootnoteTooltip)
  })
}

// Scroll a heading/element into view by scrolling ONLY the .viewer container.
// Using el.scrollIntoView() walks up the DOM and also scrolls body/html, even
// when they have overflow:hidden — Chromium ignores the overflow hint for
// programmatic scrolls. That causes a few px of unrecoverable upshift on the
// document.
export function scrollToInViewer(
  idOrEl: string | HTMLElement,
  opts: { block?: 'start' | 'center'; behavior?: ScrollBehavior; offset?: number } = {},
) {
  const viewer = document.querySelector('.viewer') as HTMLElement | null
  if (!viewer) return
  let el: HTMLElement | null
  if (typeof idOrEl === 'string') {
    el = viewer.querySelector(`[id="${CSS.escape(idOrEl)}"]`) as HTMLElement | null
  } else {
    el = idOrEl
  }
  if (!el) return
  const block = opts.block ?? 'start'
  const behavior = opts.behavior ?? 'smooth'
  const offset = opts.offset ?? 16
  const viewerRect = viewer.getBoundingClientRect()
  const elRect = el.getBoundingClientRect()
  let target: number
  if (block === 'center') {
    target =
      viewer.scrollTop + (elRect.top - viewerRect.top) - viewer.clientHeight / 2 + el.offsetHeight / 2
  } else {
    target = viewer.scrollTop + (elRect.top - viewerRect.top) - offset
  }
  viewer.scrollTo({ top: Math.max(0, target), behavior })
}

// Make GFM task list checkboxes interactive — goldmark renders them with
// `disabled`. For MCP-presented docs we re-enable them and wire each one to
// a per-doc task id so the calling LLM can poll user selections.
export function enableTaskList(
  root: HTMLElement,
  docId: string,
  onToggle: (docId: string, taskId: number, checked: boolean) => void,
) {
  const boxes = root.querySelectorAll<HTMLInputElement>('input[type="checkbox"]')
  let id = 0
  boxes.forEach((box) => {
    if (box.dataset.taskWired) return
    box.disabled = false
    box.removeAttribute('disabled')
    box.dataset.taskId = String(id)
    box.dataset.taskWired = '1'
    const myId = id
    box.addEventListener('change', () => {
      onToggle(docId, myId, box.checked)
    })
    // The wrapping <li> picks up a class so the row reads as interactive.
    const li = box.closest('li')
    if (li) li.classList.add('task-interactive')
    id++
  })
}

// Word count + reading time
export function countWords(text: string): number {
  const trimmed = text.trim()
  if (!trimmed) return 0
  return trimmed.split(/\s+/).length
}

export function readingMinutes(words: number): number {
  return Math.max(1, Math.round(words / 220))
}

export async function enhanceMermaid(root: HTMLElement) {
  const blocks = Array.from(root.querySelectorAll('pre.mermaid')) as HTMLElement[]
  if (blocks.length === 0) return
  const mermaid = await getMermaid()
  for (const block of blocks) {
    const code = block.textContent ?? ''
    // Stash the source so refreshMermaid can re-render on theme switch without
    // the markdown having to round-trip through the renderer again.
    block.setAttribute('data-mermaid-source', code)
    const id = `m-${Math.random().toString(36).slice(2, 10)}`
    try {
      const { svg, bindFunctions } = await mermaid.render(id, code)
      block.innerHTML = svg
      bindFunctions?.(block)
    } catch (e: any) {
      block.innerHTML = `<div style="color: var(--callout-caution); padding: 8px; font-family: monospace; font-size: 12px;">Mermaid error: ${(e?.message ?? e).toString().replace(/[<>]/g, (c: string) => (c === '<' ? '&lt;' : '&gt;'))}</div>`
    }
  }
}

// Re-render every mermaid block in `root` with the current theme. Called
// when the user swaps themes mid-session; reads the original source out of
// data-mermaid-source so we don't need the markdown text.
export async function refreshMermaid(root: HTMLElement) {
  if (!mermaidPromise) return // mermaid never loaded; nothing to refresh
  const blocks = Array.from(root.querySelectorAll('pre.mermaid')) as HTMLElement[]
  if (blocks.length === 0) return
  const mermaid = await mermaidPromise
  if (!ensureMermaidThemeMatches(mermaid)) return
  for (const block of blocks) {
    const code = block.getAttribute('data-mermaid-source') ?? ''
    if (!code) continue
    const id = `m-${Math.random().toString(36).slice(2, 10)}`
    try {
      const { svg, bindFunctions } = await mermaid.render(id, code)
      block.innerHTML = svg
      bindFunctions?.(block)
    } catch (e: any) {
      block.innerHTML = `<div style="color: var(--callout-caution); padding: 8px; font-family: monospace; font-size: 12px;">Mermaid error: ${(e?.message ?? e).toString().replace(/[<>]/g, (c: string) => (c === '<' ? '&lt;' : '&gt;'))}</div>`
    }
  }
}

// Find-in-page over a single .md-theme subtree.
export type FindMatch = { node: Text; start: number; end: number }

export function findInPage(root: HTMLElement, query: string): FindMatch[] {
  clearHighlights(root)
  if (!query) return []
  const q = query.toLowerCase()
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      if (!node.textContent || !node.textContent.trim()) return NodeFilter.FILTER_REJECT
      const parent = node.parentElement
      if (!parent) return NodeFilter.FILTER_REJECT
      // Skip script/style and the find highlight itself.
      const tag = parent.tagName
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'MARK') return NodeFilter.FILTER_REJECT
      return NodeFilter.FILTER_ACCEPT
    },
  })
  const matches: FindMatch[] = []
  while (walker.nextNode()) {
    const text = walker.currentNode as Text
    const lower = (text.textContent ?? '').toLowerCase()
    let i = 0
    while ((i = lower.indexOf(q, i)) !== -1) {
      matches.push({ node: text, start: i, end: i + q.length })
      i += q.length
    }
  }
  return matches
}

export function highlightMatches(matches: FindMatch[], activeIdx: number) {
  // Replace matches in reverse order (per node) to keep offsets valid.
  const byNode = new Map<Text, FindMatch[]>()
  for (const m of matches) {
    if (!byNode.has(m.node)) byNode.set(m.node, [])
    byNode.get(m.node)!.push(m)
  }
  let globalIdx = 0
  // Need to keep the global order, so iterate matches in original order.
  const flatOrder: Array<{ m: FindMatch; idx: number }> = matches.map((m, idx) => ({ m, idx }))
  // Process per node from end → start so offsets in source text remain stable.
  for (const [node, ms] of byNode.entries()) {
    const sortedDesc = [...ms].sort((a, b) => b.start - a.start)
    for (const m of sortedDesc) {
      const original = node
      const idx = flatOrder.find((f) => f.m === m)?.idx ?? -1
      const range = document.createRange()
      try {
        range.setStart(original, m.start)
        range.setEnd(original, m.end)
      } catch {
        continue
      }
      const mark = document.createElement('mark')
      mark.className = idx === activeIdx ? 'find-hit active' : 'find-hit'
      mark.dataset.findIdx = String(idx)
      range.surroundContents(mark)
    }
  }
  globalIdx++
}

export function clearHighlights(root: HTMLElement) {
  const marks = root.querySelectorAll('mark.find-hit')
  marks.forEach((m) => {
    const parent = m.parentNode
    if (!parent) return
    while (m.firstChild) parent.insertBefore(m.firstChild, m)
    parent.removeChild(m)
    parent.normalize?.()
  })
}
