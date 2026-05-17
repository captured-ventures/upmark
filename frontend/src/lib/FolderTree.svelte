<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { FolderEntry } from './types'

  export let entries: FolderEntry[] = []
  export let currentPath: string = ''
  export let depth: number = 0

  const dispatch = createEventDispatcher<{ openPath: string }>()

  function stripExt(name: string) {
    return name.replace(/\.(md|markdown|mdown|mkd|mdx)$/i, '')
  }
</script>

<ul class="ft" class:nested={depth > 0}>
  {#each entries as e (e.path)}
    {#if e.isDir}
      <li class="ft-dir">
        <details open={depth < 1}>
          <summary class="ft-dirname">
            <span class="ft-caret" aria-hidden="true">›</span>
            <span class="ft-dirlabel">{e.name}</span>
          </summary>
          {#if e.children && e.children.length > 0}
            <svelte:self
              entries={e.children}
              {currentPath}
              depth={depth + 1}
              on:openPath
            />
          {/if}
        </details>
      </li>
    {:else}
      <li>
        <button
          class="ft-file"
          class:active={e.path === currentPath}
          on:click={() => dispatch('openPath', e.path)}
          title={e.path}
        >{stripExt(e.name)}</button>
      </li>
    {/if}
  {/each}
</ul>

<style>
  .ft {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  /* Root-level rows align with other section content (32px gutter) */
  .ft > li > .ft-file,
  .ft > li > details > .ft-dirname {
    padding: 3px 16px 3px 32px;
  }

  /* Nested levels are indented under their parent AND get a vertical
     guide rule — the editorial equivalent of tree connectors.
     margin-left compounds naturally on each recursive level. */
  .ft.nested {
    margin-left: 38px;
    border-left: 1px solid var(--rule);
    padding-left: 0;
    margin-top: 2px;
    margin-bottom: 4px;
  }
  .ft.nested > li > .ft-file,
  .ft.nested > li > details > .ft-dirname {
    padding: 3px 16px 3px 14px;
  }

  /* Folder name — italic serif so it reads as a *container heading*
     rather than another row. Connects to the rest of the editorial type. */
  .ft-dirname {
    list-style: none;
    cursor: pointer;
    display: flex;
    align-items: baseline;
    gap: 6px;
    font-family: var(--font-serif);
    font-style: italic;
    font-weight: 400;
    font-size: 13.5px;
    font-variation-settings: "opsz" 14;
    color: var(--fg-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: color 120ms ease;
  }
  .ft-dirname::-webkit-details-marker { display: none; }
  .ft-dirname:hover { color: var(--fg); }

  .ft-caret {
    font-family: var(--font-sans);
    font-style: normal;
    font-size: 11px;
    color: var(--fg-subtle);
    transform: rotate(0deg);
    transition: transform 160ms ease;
    flex-shrink: 0;
    display: inline-block;
    width: 8px;
    line-height: 1;
  }
  details[open] > .ft-dirname > .ft-caret {
    transform: rotate(90deg);
  }
  .ft-dirname:hover .ft-caret { color: var(--fg-muted); }

  .ft-dirlabel {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* File rows — sans, content voice */
  .ft-file {
    width: 100%;
    text-align: left;
    display: block;
    font-family: var(--font-sans);
    font-size: 13px;
    line-height: 1.5;
    color: var(--fg-muted);
    border-left: 2px solid transparent;
    margin-left: -2px;  /* so the active rule overlaps the guide rule cleanly */
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: color 120ms ease, border-color 120ms ease;
  }
  .ft-file:hover { color: var(--fg); }
  .ft-file.active {
    color: var(--accent);
    border-left-color: var(--accent);
  }
</style>
