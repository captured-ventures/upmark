<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { ClearRecent, OpenFolderDialog, CloseFolder } from '../../wailsjs/go/main/App'
  import FolderTree from './FolderTree.svelte'
  import { scrollToInViewer } from './enhance'
  import type { TocItem, RecentEntry, Folder } from './types'

  export let open: boolean = true
  export let toc: TocItem[] = []
  export let activeTocId: string = ''
  export let folder: Folder | null = null
  export let recent: RecentEntry[] = []
  export let currentPath: string = ''

  const dispatch = createEventDispatcher<{
    openPath: string
    setFolder: Folder
  }>()

  let showContents = true
  let showFolder = true
  let showRecent = true

  async function pickFolder() {
    try {
      const f = await OpenFolderDialog()
      if (f && f.root) dispatch('setFolder', f as Folder)
    } catch (e) {
      console.error('open folder', e)
    }
  }

  async function closeFolderClicked() {
    await CloseFolder()
    dispatch('setFolder', null as any)
  }

  async function clearRecentClicked() {
    await ClearRecent()
    recent = []
  }

  function tocClass(item: TocItem) {
    let c = `toc-item toc-h${item.level}`
    if (item.id === activeTocId) c += ' active'
    return c
  }

  function stripExt(name: string) {
    return name.replace(/\.(md|markdown|mdown|mkd|mdx)$/i, '')
  }

  // Prevent default focus-on-mousedown for buttons inside the sidebar so the
  // browser doesn't auto-scroll .sb-top to "bring the focused button into
  // view" — which shifts content up by a few px on first click.
  function suppressButtonFocus(e: MouseEvent) {
    const t = e.target as HTMLElement
    if (t.closest('button')) e.preventDefault()
  }
</script>

<aside class="sidebar" class:open on:mousedown={suppressButtonFocus}>
  <div class="sb-inner">
    <div class="sb-top">
      {#if toc.length > 0}
        <section class="sb-section">
          <header class="sb-head" on:click={() => (showContents = !showContents)}>
            <span class="sb-icon section-mark" aria-hidden="true">§</span>
            <span class="sb-label">contents</span>
            <span class="sb-chevron" class:flip={!showContents}>›</span>
          </header>
          {#if showContents}
            <ul>
              {#each toc as item (item.id)}
                <li>
                  <button
                    class={tocClass(item)}
                    on:click={() => scrollToInViewer(item.id)}
                    title={item.text}
                  >{item.text}</button>
                </li>
              {/each}
            </ul>
          {/if}
        </section>
      {/if}

      <section class="sb-section">
        <header class="sb-head" on:click={() => (showFolder = !showFolder)}>
          <span class="sb-icon" aria-hidden="true">
            <svg width="11" height="11" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.2">
              <path d="M1.5 4a1 1 0 0 1 1-1h3l1.5 1.5h4.5a1 1 0 0 1 1 1V11a1 1 0 0 1-1 1h-9a1 1 0 0 1-1-1V4z"/>
            </svg>
          </span>
          <span class="sb-label">{folder ? folder.name : 'folder'}</span>
          {#if folder}
            <button class="sb-action" on:click|stopPropagation={closeFolderClicked} title="Close folder">close</button>
          {/if}
          <span class="sb-chevron" class:flip={!showFolder}>›</span>
        </header>
        {#if showFolder}
          {#if folder}
            <FolderTree entries={folder.entries} {currentPath} on:openPath />
          {:else}
            <button class="sb-empty" on:click={pickFolder}>open a folder…</button>
          {/if}
        {/if}
      </section>
    </div>

    {#if recent.length > 0}
      <div class="sb-bottom">
        <section class="sb-section">
          <header class="sb-head" on:click={() => (showRecent = !showRecent)}>
            <span class="sb-icon" aria-hidden="true">
              <svg width="11" height="11" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.2">
                <circle cx="7" cy="7" r="5.4"/>
                <path d="M7 3.8V7l2.2 1.5"/>
              </svg>
            </span>
            <span class="sb-label">recent</span>
            <button class="sb-action" on:click|stopPropagation={clearRecentClicked} title="Clear">clear</button>
            <span class="sb-chevron" class:flip={!showRecent}>›</span>
          </header>
          {#if showRecent}
            <ul>
              {#each recent.slice(0, 8) as r (r.path)}
                <li>
                  <button
                    class="file-item"
                    class:active={r.path === currentPath}
                    on:click={() => dispatch('openPath', r.path)}
                    title={r.path}
                  >{stripExt(r.name)}</button>
                </li>
              {/each}
            </ul>
          {/if}
        </section>
      </div>
    {/if}
  </div>
</aside>

<style>
  .sidebar {
    width: 0;
    overflow: hidden;
    background: transparent;
    border-right: 1px solid transparent;
    transition: width 240ms cubic-bezier(0.32, 0.72, 0.16, 1),
                border-color 240ms ease;
    flex-shrink: 0;
    /* Paint-isolate the sidebar — nothing inside can paint above the topbar's
       bottom rule or below the window's bottom edge during reflows. */
    contain: layout paint;
  }
  .sidebar.open {
    width: 252px;
    border-right-color: var(--rule);
  }

  .sb-inner {
    width: 252px;
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .sb-top {
    flex: 1;
    overflow-y: auto;
    padding: 8px 0 12px;
    min-height: 0;
    /* Reserve the 10px scrollbar gutter always so showing/hiding the thumb on
       hover doesn't shift content horizontally and never triggers a reflow. */
    scrollbar-gutter: stable;
  }

  .sb-bottom {
    flex: 0 0 auto;
    max-height: 38%;
    overflow-y: auto;
    border-top: 1px solid var(--rule);
    padding: 4px 0 8px;
    background: var(--bg-elev);
    scrollbar-gutter: stable;
  }
  .sb-bottom .sb-section { margin-bottom: 0; }
  .sb-bottom .sb-section:last-child > ul { padding-bottom: 4px; }

  .sb-section { margin-bottom: 8px; }
  .sb-section ul { list-style: none; margin: 0; padding: 0; }

  .sb-head {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px 8px 18px;
    cursor: pointer;
    user-select: none;
  }

  .sb-icon {
    width: 14px;
    height: 14px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: var(--fg-subtle);
    flex-shrink: 0;
    transition: color 120ms ease;
  }
  .sb-head:hover .sb-icon { color: var(--fg-muted); }

  /* The § connects to the hanging section marks in the document */
  .sb-icon.section-mark {
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 14px;
    line-height: 1;
    color: var(--accent);
    font-variation-settings: "opsz" 14;
    margin-top: -1px;
  }

  .sb-label {
    flex: 1;
    font-family: var(--font-sans);
    font-size: 10px;
    font-weight: 500;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--fg-subtle);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .sb-action {
    font-family: var(--font-sans);
    font-size: 10px;
    color: var(--fg-subtle);
    padding: 2px 4px;
    border-radius: 2px;
    letter-spacing: 0.06em;
    text-transform: lowercase;
    transition: color 120ms ease;
  }
  .sb-action:hover { color: var(--fg); }

  .sb-chevron {
    font-size: 14px;
    line-height: 1;
    color: var(--fg-subtle);
    transform: rotate(90deg);
    transition: transform 200ms ease;
  }
  .sb-chevron.flip { transform: rotate(0deg); }

  .toc-item, .file-item {
    width: 100%;
    text-align: left;
    display: block;
    font-family: var(--font-sans);
    font-size: 13px;
    line-height: 1.45;
    color: var(--fg-muted);
    padding: 3px 16px 3px 32px;
    border-left: 2px solid transparent;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: color 120ms ease, border-color 120ms ease;
  }
  .toc-item:hover, .file-item:hover { color: var(--fg); }
  /* Color + border-color only — no font-weight change. Bolding shifts text
     width which causes a reflow flash every time the active section changes
     during a smooth scroll. */
  .toc-item.active, .file-item.active {
    color: var(--accent);
    border-left-color: var(--accent);
  }

  .toc-h1, .toc-h2 { padding-left: 32px; }
  .toc-h3 { padding-left: 46px; font-size: 12px; color: var(--fg-subtle); }
  .toc-h4 { padding-left: 60px; font-size: 11.5px; color: var(--fg-subtle); }
  .toc-h5, .toc-h6 { padding-left: 70px; font-size: 11px; color: var(--fg-subtle); }

  .sb-empty {
    margin: 4px 18px 0 32px;
    color: var(--fg-subtle);
    font-family: var(--font-sans);
    font-size: 12px;
    font-style: italic;
    padding: 2px 0;
    transition: color 120ms ease;
  }
  .sb-empty:hover { color: var(--accent); }
</style>
