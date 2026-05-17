<script lang="ts">
  import { onMount, onDestroy } from 'svelte'
  import { EventsOn } from '../wailsjs/runtime/runtime'
  import * as wailsRuntime from '../wailsjs/runtime/runtime'
  import {
    OpenDialog,
    OpenPath,
    CloseDocument,
    RecentFiles,
    GetChromaCSS,
    OpenFolderDialog,
    OpenFolder,
    LastFolder,
    GetUIPrefs,
    SetReadingWidth,
    SetFontSize,
    SetTheme,
    SaveDocument,
    RenderMarkdown,
    GetStartupOrLastFile,
    RevealInExplorer,
    OpenContainingFolder,
    OpenURL,
    GetMCPStatus,
    SetMCPEnabled,
    SetMCPPort,
    MCPSetTaskChecked,
    MCPSetViewState,
    IsFirstLaunch,
    OpenWelcome,
    SetMCPWindowOnPresent,
    EnterFocusMode,
    ExitFocusMode,
  } from '../wailsjs/go/main/App'

  import TopBar from './lib/TopBar.svelte'
  import Sidebar from './lib/Sidebar.svelte'
  import EmptyState from './lib/EmptyState.svelte'
  import Viewer from './lib/Viewer.svelte'
  import Editor from './lib/Editor.svelte'
  import CommandPalette from './lib/CommandPalette.svelte'
  import URLPrompt from './lib/URLPrompt.svelte'
  import Settings from './lib/Settings.svelte'
  import { scrollToInViewer, refreshMermaid } from './lib/enhance'
  import type { Doc, RecentEntry, Folder, TocItem, Command, MCPDoc, MCPStatus } from './lib/types'

  const OnFileDrop = (wailsRuntime as any).OnFileDrop as (
    cb: (x: number, y: number, paths: string[]) => void,
    useDropTarget: boolean,
  ) => void
  const OnFileDropOff = (wailsRuntime as any).OnFileDropOff as () => void

  let doc: Doc | null = null
  let recent: RecentEntry[] = []
  let folder: Folder | null = null
  let toc: TocItem[] = []
  let activeHeading: { id: string; text: string } = { id: '', text: '' }
  let sidebarOpen = true
  let findOpen = false
  let paletteOpen = false
  let urlPromptOpen = false
  let settingsOpen = false
  let errorMsg = ''
  let busy = false

  // Phase 11 — focus mode: hide sidebar, snap window to the current reading
  // column at monitor full height. Press again, Esc, or palette to exit.
  // preFocusSidebarOpen captures the user's sidebar state so we can restore it.
  let focusModeActive = false
  let preFocusSidebarOpen = true

  // Edit mode
  let editing = false
  let editorContent = ''
  let savedContent = ''     // last value confirmed on disk
  let editorDirty = false
  let saveStatus: 'idle' | 'saving' | 'saved' = 'idle'
  let autosaveTimer: number | undefined
  let liveRenderTimer: number | undefined

  // Read polish: reading width + font size with CSS-var application
  type WidthKey = 'narrow' | 'normal' | 'wide'
  const widthPx: Record<WidthKey, string> = {
    narrow: '54ch',
    normal: '68ch',
    wide: '88ch',
  }
  const widthOrder: WidthKey[] = ['narrow', 'normal', 'wide']
  let readingWidth: WidthKey = 'normal'
  let fontSize = 17 // px, 12-26

  // Theme system
  type ThemeKey =
    | 'editorial' | 'broadsheet' | 'newsprint' | 'terminal'
    | 'manuscript' | 'brutalist' | 'arcade'
    | 'pastoral' | 'architect' | 'vapor'
    | 'typewriter' | 'midnight' | 'gameboy'
  const themeOrder: ThemeKey[] = [
    'editorial', 'broadsheet', 'newsprint', 'terminal',
    'manuscript', 'brutalist', 'arcade',
    'pastoral', 'architect', 'vapor',
    'typewriter', 'midnight', 'gameboy',
  ]
  // Short blurbs surface in the palette so each theme is discoverable.
  const themeBlurbs: Record<ThemeKey, string> = {
    editorial:  'warm paper, rust, serif body',
    broadsheet: 'all-serif newspaper',
    newsprint:  'halftone dots, slab headlines, ink red',
    terminal:   'mono, dark, amber',
    manuscript: 'parchment, drop caps, sepia',
    brutalist:  'oversized, black & white, hard edges',
    arcade:     'neon synthwave, scan lines, glow',
    pastoral:   'cream, sage, rounded',
    architect:  'blueprint grid, prussian blue',
    vapor:      'vaporwave gradient, handwriting',
    typewriter: 'courier prime, ribbon red',
    midnight:   'navy library, gold accent',
    gameboy:    'pixel 4-color green',
  }
  let theme: ThemeKey = 'editorial'

  // MCP server state
  let mcpStatus: MCPStatus = { enabled: false, running: false, port: 11451, url: '' }
  // present_document window behavior pref — synced from backend on mount.
  type WindowOnPresent = 'show-no-focus' | 'show-and-focus'
  let mcpWindowOnPresent: WindowOnPresent = 'show-no-focus'

  function applyTheme() {
    document.documentElement.setAttribute('data-theme', theme)
  }

  async function setThemeAndSave(t: ThemeKey) {
    theme = t
    applyTheme()
    // Mermaid is initialized once with a light/dark choice; re-render any
    // diagrams in the active doc so they pick up the new theme.
    requestAnimationFrame(() => {
      const body = document.querySelector('.markdown-body') as HTMLElement | null
      if (body) refreshMermaid(body).catch((e) => console.error('mermaid refresh:', e))
    })
    try { await SetTheme(t) } catch (e) { console.error(e) }
  }

  function applyReadingVars() {
    document.documentElement.style.setProperty('--reading-width', widthPx[readingWidth])
    document.documentElement.style.setProperty('--reading-size', `${fontSize}px`)
  }

  async function toggleFocusMode() {
    if (focusModeActive) {
      sidebarOpen = preFocusSidebarOpen
      try { await ExitFocusMode() } catch (e) { console.error(e) }
      focusModeActive = false
      return
    }
    // Measure the rendered doc width *before* collapsing the sidebar so the
    // column is captured at its current measure (collapsing first would let
    // the body briefly grow). Fall back to a sensible default when no doc
    // is open (empty state has no .markdown-body to measure).
    const docEl = document.querySelector('.markdown-body') as HTMLElement | null
    const contentW = docEl ? Math.ceil(docEl.getBoundingClientRect().width) : 720
    const chromeW = Math.max(0, window.outerWidth - window.innerWidth)
    preFocusSidebarOpen = sidebarOpen
    sidebarOpen = false
    try {
      await EnterFocusMode(contentW + chromeW)
      focusModeActive = true
    } catch (e) {
      console.error(e)
      sidebarOpen = preFocusSidebarOpen
    }
  }

  async function setWidth(w: WidthKey) {
    readingWidth = w
    applyReadingVars()
    try { await SetReadingWidth(w) } catch (e) { console.error(e) }
  }

  async function adjustWidth(dir: 1 | -1) {
    const idx = widthOrder.indexOf(readingWidth)
    const next = widthOrder[Math.min(widthOrder.length - 1, Math.max(0, idx + dir))]
    if (next !== readingWidth) await setWidth(next)
  }

  async function adjustFont(dir: 1 | -1 | 0) {
    const next = dir === 0 ? 17 : Math.min(26, Math.max(12, fontSize + dir))
    if (next !== fontSize) {
      fontSize = next
      applyReadingVars()
      try { await SetFontSize(next) } catch (e) { console.error(e) }
    }
  }

  // History stacks for back/forward navigation.
  let backStack: string[] = []
  let forwardStack: string[] = []
  $: canBack = backStack.length > 0
  $: canForward = forwardStack.length > 0

  // ───── data loading helpers ─────

  async function refreshRecent() {
    try { recent = (await RecentFiles()) ?? [] } catch { recent = [] }
  }

  async function injectChromaCSS() {
    try {
      const css = await GetChromaCSS()
      const style = document.createElement('style')
      style.id = 'chroma-css'
      style.textContent = `${css.light ?? ''}\n@media (prefers-color-scheme: dark) { ${css.dark ?? ''} }`
      document.head.appendChild(style)
    } catch (e) { console.error('chroma css:', e) }
  }

  // ───── file ops ─────

  async function openDialog() {
    if (busy) return
    busy = true; errorMsg = ''
    try {
      const d = await OpenDialog()
      if (d && d.path) await setDoc(d as Doc, { pushHistory: true })
    } catch (e: any) { errorMsg = String(e?.message ?? e) }
    finally { busy = false }
  }

  async function openPath(path: string, opts: { pushHistory?: boolean } = { pushHistory: true }) {
    if (busy) return
    busy = true; errorMsg = ''
    try {
      const d = await OpenPath(path)
      if (d && d.path) await setDoc(d as Doc, opts)
    } catch (e: any) { errorMsg = String(e?.message ?? e) }
    finally { busy = false }
  }

  async function openFromURL(rawURL: string) {
    if (busy) return
    busy = true; errorMsg = ''
    try {
      const d = await OpenURL(rawURL)
      if (d && d.path) {
        // Treat remote docs like a new doc in history (push the previous one).
        await setDoc(d as Doc, { pushHistory: true })
        urlPromptOpen = false
      }
    } catch (e: any) {
      errorMsg = String(e?.message ?? e)
      // Keep the prompt open so the user can correct the URL.
    } finally { busy = false }
  }

  async function openWelcome() {
    if (busy) return
    busy = true; errorMsg = ''
    try {
      const d = await OpenWelcome()
      if (d) await setDoc(d as Doc, { pushHistory: true })
    } catch (e: any) { errorMsg = String(e?.message ?? e) }
    finally { busy = false }
  }

  async function setDoc(d: Doc, opts: { pushHistory?: boolean }) {
    // Flush any pending save for the outgoing doc before switching.
    if (doc && editorDirty) {
      if (autosaveTimer) { clearTimeout(autosaveTimer); autosaveTimer = undefined }
      await saveNow()
    }
    // If we're leaving an MCP doc, mark it un-viewed.
    if (doc?.isMCP && doc.mcpId && doc.mcpId !== d.mcpId) {
      MCPSetViewState(doc.mcpId, false, false).catch(console.error)
    }
    if (opts.pushHistory && doc && doc.path !== d.path) {
      backStack = [...backStack, doc.path]
      forwardStack = []
    }
    doc = d
    // Sync editor state to the new document.
    editorContent = d.source ?? ''
    savedContent = editorContent
    editorDirty = false
    saveStatus = 'idle'
    // If we're entering an MCP doc, mark it viewed.
    if (d.isMCP && d.mcpId) {
      MCPSetViewState(d.mcpId, true, false).catch(console.error)
    }
    await refreshRecent()
  }

  // ─── editor flow ───

  async function toggleEdit() {
    if (!doc || doc.readOnly) return
    if (editing && editorDirty) {
      if (autosaveTimer) { clearTimeout(autosaveTimer); autosaveTimer = undefined }
      await saveNow()
    }
    editing = !editing
  }

  function onEditorChange(content: string) {
    if (!doc) return
    editorContent = content
    editorDirty = content !== savedContent

    // Live preview: re-render from the editor buffer (no disk read).
    if (liveRenderTimer) clearTimeout(liveRenderTimer)
    liveRenderTimer = window.setTimeout(async () => {
      if (!doc) return
      try {
        const html = await RenderMarkdown(editorContent, doc.baseDir)
        // Mutate just the html field so Viewer's `html` prop changes and
        // re-renders without disturbing the rest of the doc state.
        doc = { ...doc, html }
      } catch (e) {
        console.error('live render:', e)
      }
    }, 200)

    // Autosave on idle.
    if (autosaveTimer) clearTimeout(autosaveTimer)
    autosaveTimer = window.setTimeout(() => saveNow(), 800)
  }

  async function saveNow() {
    if (!doc || !editorDirty) return
    saveStatus = 'saving'
    try {
      await SaveDocument(doc.path, editorContent)
      savedContent = editorContent
      editorDirty = false
      saveStatus = 'saved'
      setTimeout(() => { if (saveStatus === 'saved') saveStatus = 'idle' }, 900)
    } catch (e: any) {
      saveStatus = 'idle'
      errorMsg = String(e?.message ?? e)
    }
  }

  async function closeDoc() {
    // If closing an MCP doc, mark it closed-by-user so the LLM can see.
    if (doc?.isMCP && doc.mcpId) {
      MCPSetViewState(doc.mcpId, false, true).catch(console.error)
    }
    await CloseDocument()
    doc = null
    toc = []
    activeHeading = { id: '', text: '' }
    backStack = []
    forwardStack = []
    await refreshRecent()
  }

  // ─── MCP server + presented-doc events ───

  function mcpDocToDoc(m: MCPDoc): Doc {
    return {
      path: `mcp:${m.id}`,
      name: m.title,
      html: m.rendered,
      source: m.source,
      baseDir: '',
      modified: Date.parse(m.updatedAt) || Date.now(),
      isMCP: true,
      mcpId: m.id,
      mcpClient: m.client,
    }
  }

  async function refreshMCPStatus() {
    try {
      mcpStatus = (await GetMCPStatus()) as MCPStatus
    } catch (e) { console.error('mcp status:', e) }
  }

  async function toggleMCP() {
    try {
      await SetMCPEnabled(!mcpStatus.enabled)
    } catch (e: any) {
      errorMsg = `MCP: ${e?.message ?? e}`
    }
    await refreshMCPStatus()
  }

  async function onMCPTaskToggle(docId: string, taskId: number, checked: boolean) {
    try { await MCPSetTaskChecked(docId, taskId, checked) } catch (e) { console.error(e) }
  }

  // Settings event handlers — multi-statement so they live here, not inline.
  function onSettingsFontSize(px: number) {
    fontSize = px
    applyReadingVars()
    SetFontSize(px).catch(console.error)
  }
  async function onSettingsMCPPort(port: number) {
    try { await SetMCPPort(port) } catch (e) { console.error(e) }
    await refreshMCPStatus()
  }
  async function onSettingsWindowOnPresent(behavior: WindowOnPresent) {
    mcpWindowOnPresent = behavior
    try { await SetMCPWindowOnPresent(behavior) } catch (e) { console.error(e) }
  }
  async function onCopyMCPURL() {
    try { await navigator.clipboard.writeText(mcpStatus.url) } catch (e) { console.error(e) }
  }

  async function pickFolder() {
    try {
      const f = await OpenFolderDialog()
      if (f && f.root) folder = f as Folder
    } catch (e) { console.error(e) }
  }

  function goBack() {
    if (backStack.length === 0 || !doc) return
    const prev = backStack[backStack.length - 1]
    backStack = backStack.slice(0, -1)
    forwardStack = [doc.path, ...forwardStack]
    openPath(prev, { pushHistory: false })
  }

  function goForward() {
    if (forwardStack.length === 0 || !doc) return
    const next = forwardStack[0]
    forwardStack = forwardStack.slice(1)
    backStack = [...backStack, doc.path]
    openPath(next, { pushHistory: false })
  }

  // ───── keyboard ─────

  function handleKey(e: KeyboardEvent) {
    const mod = e.ctrlKey || e.metaKey
    const k = e.key.toLowerCase()

    if (mod && e.shiftKey && k === 'o') { e.preventDefault(); pickFolder(); return }
    if (mod && k === 'o') { e.preventDefault(); openDialog(); return }
    if (mod && k === 'l') { e.preventDefault(); urlPromptOpen = !urlPromptOpen; return }
    if (mod && e.key === ',') { e.preventDefault(); settingsOpen = !settingsOpen; return }
    if (mod && k === 'k') { e.preventDefault(); paletteOpen = !paletteOpen; return }
    if (mod && k === 'b') { e.preventDefault(); sidebarOpen = !sidebarOpen; return }
    if (mod && k === 'w') { e.preventDefault(); if (doc) closeDoc(); return }
    if (mod && e.shiftKey && k === 'f') { e.preventDefault(); toggleFocusMode(); return }
    if (mod && k === 'f') { e.preventDefault(); if (doc) findOpen = !findOpen; return }
    if (mod && k === 'p') { e.preventDefault(); window.print(); return }
    if (mod && k === 'e') { e.preventDefault(); toggleEdit(); return }
    if (mod && k === 's') { e.preventDefault(); saveNow(); return }
    if (e.altKey && e.key === 'ArrowLeft') { e.preventDefault(); goBack(); return }
    if (e.altKey && e.key === 'ArrowRight') { e.preventDefault(); goForward(); return }
    if (e.key === 'Escape' && focusModeActive) { toggleFocusMode(); return }
    if (e.key === 'Escape' && findOpen) { findOpen = false; return }

    // Reading polish shortcuts. Use e.code for the bracket keys since Shift+[
    // produces "{" on US layouts, breaking an e.key === "[" check.
    if (mod && (e.key === '+' || e.key === '=')) { e.preventDefault(); adjustFont(1); return }
    if (mod && e.key === '-')                    { e.preventDefault(); adjustFont(-1); return }
    if (mod && e.key === '0')                    { e.preventDefault(); adjustFont(0); return }
    if (mod && e.shiftKey && e.code === 'BracketLeft')  { e.preventDefault(); adjustWidth(-1); return }
    if (mod && e.shiftKey && e.code === 'BracketRight') { e.preventDefault(); adjustWidth(1); return }
  }

  // ───── command palette ─────

  $: commands = buildCommands(doc, folder, recent, toc, editing, theme, mcpStatus, focusModeActive, sidebarOpen)

  function buildCommands(
    d: Doc | null,
    f: Folder | null,
    r: RecentEntry[],
    t: TocItem[],
    _editing: boolean,
    _theme: ThemeKey,
    _mcp: MCPStatus,
    _focusActive: boolean,
    _sidebarOpen: boolean,
  ): Command[] {
    const cmds: Command[] = []

    cmds.push({ id: 'open-file',   group: 'action', label: 'open file',   hint: '⌃ O',   run: openDialog })
    cmds.push({ id: 'open-folder', group: 'action', label: 'open folder', hint: '⌃ ⇧ O', run: pickFolder })
    cmds.push({ id: 'open-url',    group: 'action', label: 'open from url', hint: '⌃ L', matchText: 'open url remote github', run: () => { urlPromptOpen = true } })
    cmds.push({ id: 'welcome',     group: 'action', label: 'show welcome', matchText: 'show welcome help tour intro getting started', run: openWelcome })
    if (d) {
      if (!d.readOnly) {
        cmds.push({ id: 'edit',    group: 'action', label: editing ? 'exit edit mode' : 'edit document', hint: '⌃ E', run: toggleEdit })
        if (editing) {
          cmds.push({ id: 'save',  group: 'action', label: 'save', hint: '⌃ S', run: saveNow })
        }
      }
      cmds.push({ id: 'close',     group: 'action', label: 'close document',  hint: '⌃ W', run: closeDoc })
      cmds.push({ id: 'find',      group: 'action', label: 'find in document', hint: '⌃ F', run: () => { findOpen = true } })
      cmds.push({ id: 'print',     group: 'action', label: 'print / save pdf', hint: '⌃ P', run: () => window.print() })
      if (!d.readOnly && !d.isMCP) {
        cmds.push({ id: 'reveal',    group: 'action', label: 'reveal in explorer', run: () => RevealInExplorer(d.path).catch(console.error) })
        cmds.push({ id: 'open-dir',  group: 'action', label: 'open containing folder', run: () => OpenContainingFolder(d.path).catch(console.error) })
      }
    }
    cmds.push({ id: 'sidebar',     group: 'action', label: sidebarOpen ? 'hide sidebar' : 'show sidebar', hint: '⌃ B', run: () => { sidebarOpen = !sidebarOpen } })
    cmds.push({ id: 'focus',       group: 'action', label: focusModeActive ? 'exit focus mode' : 'focus mode', hint: '⌃ ⇧ F', matchText: 'focus distraction-free slab read', run: toggleFocusMode })
    cmds.push({ id: 'settings',    group: 'action', label: 'settings…', hint: '⌃ ,', matchText: 'settings preferences config', run: () => { settingsOpen = true } })

    // Reading polish — cycle actions get the shortcut hint, presets show ✓ on active.
    cmds.push({ id: 'width-narrower', group: 'action', label: 'narrower reading width', hint: '⌃ ⇧ [', run: () => adjustWidth(-1) })
    cmds.push({ id: 'width-wider',    group: 'action', label: 'wider reading width',    hint: '⌃ ⇧ ]', run: () => adjustWidth(1) })
    cmds.push({ id: 'width-narrow',   group: 'action', label: 'set width: narrow' + (readingWidth === 'narrow' ? '  ✓' : ''), matchText: 'set width narrow', run: () => setWidth('narrow') })
    cmds.push({ id: 'width-normal',   group: 'action', label: 'set width: normal' + (readingWidth === 'normal' ? '  ✓' : ''), matchText: 'set width normal', run: () => setWidth('normal') })
    cmds.push({ id: 'width-wide',     group: 'action', label: 'set width: wide'   + (readingWidth === 'wide'   ? '  ✓' : ''), matchText: 'set width wide',   run: () => setWidth('wide') })
    cmds.push({ id: 'font-up',        group: 'action', label: 'larger font',  hint: '⌃ +', run: () => adjustFont(1) })
    cmds.push({ id: 'font-down',      group: 'action', label: 'smaller font', hint: '⌃ −', run: () => adjustFont(-1) })
    cmds.push({ id: 'font-reset',     group: 'action', label: 'reset font',   hint: '⌃ 0', run: () => adjustFont(0) })

    // MCP server controls
    cmds.push({
      id: 'mcp-toggle',
      group: 'action',
      label: mcpStatus.running ? 'mcp server: turn off' : 'mcp server: turn on',
      hint: mcpStatus.running ? `:${mcpStatus.port}` : '',
      matchText: 'mcp server enable disable llm',
      run: toggleMCP,
    })
    if (mcpStatus.running) {
      cmds.push({
        id: 'mcp-copy-url',
        group: 'action',
        label: 'mcp: copy endpoint URL',
        hint: mcpStatus.url,
        matchText: 'mcp url endpoint copy',
        run: async () => {
          try { await navigator.clipboard.writeText(mcpStatus.url) } catch {}
        },
      })
    }

    // Themes — one command per theme; current theme gets a ✓.
    for (const t of themeOrder) {
      cmds.push({
        id: `th-${t}`,
        group: 'action',
        label: `theme: ${t}${theme === t ? '  ✓' : ''}`,
        hint: themeBlurbs[t],
        matchText: `theme ${t} ${themeBlurbs[t]}`,
        run: () => setThemeAndSave(t),
      })
    }

    for (const h of t) {
      cmds.push({
        id: `h-${h.id}`,
        group: 'heading',
        label: `${'›'.repeat(Math.max(h.level - 1, 0))} ${h.text}`.trim(),
        matchText: h.text,
        run: () => scrollToInViewer(h.id),
      })
    }

    if (f) {
      const walk = (entries: typeof f.entries, prefix = '') => {
        for (const e of entries) {
          if (e.isDir && e.children) {
            walk(e.children, prefix + e.name + '/')
          } else if (!e.isDir) {
            cmds.push({
              id: `f-${e.path}`,
              group: 'folder',
              label: prefix + e.name.replace(/\.(md|markdown|mdown|mkd|mdx)$/i, ''),
              matchText: prefix + e.name,
              run: () => openPath(e.path),
            })
          }
        }
      }
      walk(f.entries)
    }

    for (const re of r.slice(0, 12)) {
      cmds.push({
        id: `r-${re.path}`,
        group: 'recent',
        label: re.name.replace(/\.(md|markdown|mdown|mkd|mdx)$/i, ''),
        matchText: re.path,
        hint: shortPath(re.path),
        run: () => openPath(re.path),
      })
    }

    return cmds
  }

  function shortPath(p: string): string {
    const parts = p.split(/[\\/]/)
    if (parts.length <= 2) return p
    return '… / ' + parts.slice(-2, -1).join('/')
  }

  // ───── lifecycle ─────

  let unsubChanged: (() => void) | undefined
  let unsubError: (() => void) | undefined

  // Track maximize state so we can compensate for the ~8px Windows expands
  // frameless windows beyond the visible screen edges.
  //
  // Sync detection (no async race): when the window covers the entire available
  // screen area, we're maximized. Reading window.outerWidth / screen.availWidth
  // is instantaneous, so the class is applied in the same frame as the resize
  // event — no flash of clipped content.
  function syncMaximizeClass() {
    const max =
      window.outerWidth  >= screen.availWidth  - 2 &&
      window.outerHeight >= screen.availHeight - 2
    document.documentElement.classList.toggle('is-maximized', max)
  }

  onMount(async () => {
    await injectChromaCSS()

    // Load read-polish prefs and apply CSS vars before first render
    try {
      const p = await GetUIPrefs()
      if (p?.readingWidth && (widthOrder as string[]).includes(p.readingWidth)) {
        readingWidth = p.readingWidth as WidthKey
      }
      if (p?.fontSize && p.fontSize >= 12 && p.fontSize <= 26) {
        fontSize = p.fontSize
      }
      if (p?.theme && (themeOrder as string[]).includes(p.theme)) {
        theme = p.theme as ThemeKey
      }
      if (p?.mcpWindowOnPresent === 'show-no-focus' || p?.mcpWindowOnPresent === 'show-and-focus') {
        mcpWindowOnPresent = p.mcpWindowOnPresent
      }
    } catch (e) { console.error('GetUIPrefs:', e) }
    applyReadingVars()
    applyTheme()

    await refreshRecent()

    // Reopen last folder if any.
    try {
      const f = await LastFolder()
      if (f && f.root) folder = f as Folder
    } catch (e) { console.error('last folder:', e) }

    // Open startup file (CLI arg / file-association) or fall back to last-opened.
    // If neither is set and this is a first launch, open the embedded welcome doc.
    try {
      const p = await GetStartupOrLastFile()
      if (p) {
        openPath(p)
      } else if (await IsFirstLaunch()) {
        await openWelcome()
      }
    } catch (e) { console.error('startup file:', e) }

    unsubChanged = (EventsOn as any)('file-changed', (d: Doc) => {
      // External edit — replace the active doc without touching history.
      doc = d
    })
    unsubError = (EventsOn as any)('file-error', (msg: string) => { errorMsg = msg })

    // Second-instance launches (Explorer double-click while running, or
    // running upmark on the CLI again) forward their file arg here.
    ;(EventsOn as any)('open-from-second-instance', (p: string) => {
      if (p) openPath(p)
    })

    // MCP server events — when an LLM client pushes a doc into upmark.
    ;(EventsOn as any)('mcp-doc-presented', (m: MCPDoc) => {
      setDoc(mcpDocToDoc(m), { pushHistory: true })
    })
    ;(EventsOn as any)('mcp-doc-updated', (m: MCPDoc) => {
      // If we're viewing this MCP doc, refresh its content; otherwise ignore
      // (the next time the user navigates back, they'll see the new version).
      if (doc?.isMCP && doc.mcpId === m.id) {
        doc = mcpDocToDoc(m)
      }
    })
    ;(EventsOn as any)('mcp-doc-closed', (id: string) => {
      if (doc?.isMCP && doc.mcpId === id) {
        // LLM closed the doc — return to empty state without marking
        // closedByUser (it was the LLM's choice).
        doc = null
        toc = []
        activeHeading = { id: '', text: '' }
      }
    })

    await refreshMCPStatus()

    OnFileDrop((_x, _y, paths) => {
      if (!paths || paths.length === 0) return
      const md = paths.find(p => /\.(md|markdown|mdown|mkd|mdx)$/i.test(p)) ?? paths[0]
      openPath(md)
    }, true)

    window.addEventListener('keydown', handleKey)

    // Initial maximize check + on resize/focus.
    syncMaximizeClass()
    window.addEventListener('resize', syncMaximizeClass)
    window.addEventListener('focus', syncMaximizeClass)
  })

  onDestroy(() => {
    unsubChanged?.()
    unsubError?.()
    OnFileDropOff()
    window.removeEventListener('keydown', handleKey)
    window.removeEventListener('resize', syncMaximizeClass)
    window.removeEventListener('focus', syncMaximizeClass)
  })
</script>

<TopBar
  docName={doc?.name ?? ''}
  sidebarOpen={sidebarOpen}
  editing={editing}
  readOnly={!!doc?.readOnly}
  isMCP={!!doc?.isMCP}
  mcpClient={doc?.mcpClient ?? ''}
  mcpRunning={mcpStatus.running}
  focusActive={focusModeActive}
  on:toggleSidebar={() => (sidebarOpen = !sidebarOpen)}
  on:open={openDialog}
  on:find={() => (findOpen = !findOpen)}
  on:palette={() => (paletteOpen = !paletteOpen)}
  on:edit={toggleEdit}
  on:settings={() => (settingsOpen = true)}
  on:toggleFocus={toggleFocusMode}
/>

<div class="shell">
  <Sidebar
    open={sidebarOpen}
    {toc}
    activeTocId={activeHeading.id}
    {folder}
    {recent}
    currentPath={doc?.path ?? ''}
    on:openPath={(e) => openPath(e.detail)}
    on:setFolder={(e) => (folder = e.detail)}
  />

  <div class="main-col" class:split={editing && doc}>
    {#if doc}
      {#if editing}
        <div class="edit-pane">
          <Editor
            value={doc.source}
            dirty={editorDirty}
            on:change={(e) => onEditorChange(e.detail)}
            on:save={saveNow}
          />
        </div>
        <div class="split-divider" aria-hidden="true"></div>
      {/if}
      <Viewer
        html={doc.html}
        baseDir={doc.baseDir}
        docPath={doc.path}
        docName={doc.name}
        mcpId={doc.mcpId ?? ''}
        bind:findOpen
        on:toc={(e) => (toc = e.detail)}
        on:activeHeading={(e) => (activeHeading = e.detail)}
        on:openPath={(e) => openPath(e.detail)}
        on:mcpTaskToggle={(e) => onMCPTaskToggle(e.detail.docId, e.detail.taskId, e.detail.checked)}
      />
    {:else}
      <EmptyState
        {recent}
        on:open={openDialog}
        on:openPath={(e) => openPath(e.detail)}
        on:setFolder={(e) => (folder = e.detail)}
      />
    {/if}
  </div>
</div>

<CommandPalette bind:open={paletteOpen} {commands} />
<URLPrompt bind:open={urlPromptOpen} on:submit={(e) => openFromURL(e.detail)} />
<Settings
  bind:open={settingsOpen}
  theme={theme}
  readingWidth={readingWidth}
  fontSize={fontSize}
  mcpStatus={mcpStatus}
  mcpWindowOnPresent={mcpWindowOnPresent}
  on:setTheme={(e) => setThemeAndSave(e.detail)}
  on:setWidth={(e) => setWidth(e.detail)}
  on:setFontSize={(e) => onSettingsFontSize(e.detail)}
  on:toggleMCP={toggleMCP}
  on:setMCPPort={(e) => onSettingsMCPPort(e.detail)}
  on:setMCPWindowOnPresent={(e) => onSettingsWindowOnPresent(e.detail)}
  on:copyMCPURL={onCopyMCPURL}
/>

{#if errorMsg}
  <div class="error-toast" on:click={() => (errorMsg = '')}>{errorMsg}</div>
{/if}

<style>
  .error-toast {
    position: fixed;
    bottom: 18px;
    right: 18px;
    background: var(--accent);
    color: var(--bg);
    padding: 8px 14px;
    border-radius: 4px;
    font-family: var(--font-sans);
    font-size: 12px;
    max-width: 360px;
    z-index: 2000;
    cursor: pointer;
    box-shadow: 0 8px 24px -8px rgba(0, 0, 0, 0.3);
  }

  .main-col.split {
    flex-direction: row;
  }
  .edit-pane {
    flex: 1 1 50%;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .main-col.split :global(.viewer) {
    flex: 1 1 50%;
    min-width: 0;
  }
  .split-divider {
    flex: 0 0 1px;
    background: var(--rule-strong);
  }
</style>
