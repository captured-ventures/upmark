<script lang="ts">
  import { onMount, onDestroy, createEventDispatcher } from 'svelte'
  import { EditorState, Compartment } from '@codemirror/state'
  import {
    EditorView,
    keymap,
    drawSelection,
    highlightActiveLine,
    lineNumbers,
    highlightActiveLineGutter,
  } from '@codemirror/view'
  import {
    defaultKeymap,
    history,
    historyKeymap,
    indentWithTab,
  } from '@codemirror/commands'
  import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
  import {
    bracketMatching,
    foldGutter,
    foldKeymap,
    syntaxHighlighting,
    HighlightStyle,
    indentOnInput,
  } from '@codemirror/language'
  import { searchKeymap } from '@codemirror/search'
  import { tags as t } from '@lezer/highlight'

  export let value: string = ''
  export let dirty: boolean = false

  const dispatch = createEventDispatcher<{
    change: string
    save: void
  }>()

  let editorEl: HTMLDivElement | undefined
  let view: EditorView | undefined

  // Editorial syntax highlighting — restrained, follows the rest of the app
  const editorHighlight = HighlightStyle.define([
    { tag: [t.heading1, t.heading2, t.heading3, t.heading4, t.heading5, t.heading6],
      color: 'var(--fg)', fontWeight: '500' },
    { tag: t.emphasis,    fontStyle: 'italic' },
    { tag: t.strong,      fontWeight: '600' },
    { tag: t.strikethrough, textDecoration: 'line-through' },
    { tag: t.link,        color: 'var(--accent)' },
    { tag: t.url,         color: 'var(--accent)' },
    { tag: t.monospace,   fontFamily: 'var(--font-mono)', color: 'var(--fg-muted)' },
    { tag: t.contentSeparator, color: 'var(--fg-subtle)' },
    { tag: t.quote,       color: 'var(--fg-muted)', fontStyle: 'italic' },
    { tag: t.list,        color: 'var(--fg-muted)' },
    { tag: t.comment,     color: 'var(--fg-subtle)' },
    { tag: t.processingInstruction, color: 'var(--accent)' },
  ])

  const editorTheme = EditorView.theme(
    {
      '&': {
        height: '100%',
        backgroundColor: 'var(--bg)',
        color: 'var(--fg)',
        fontFamily: 'var(--font-serif)',
        fontSize: '15.5px',
        lineHeight: '1.7',
        fontVariationSettings: '"opsz" 16',
      },
      '.cm-scroller': {
        fontFamily: 'inherit',
        overflow: 'auto',
        scrollbarGutter: 'stable',
      },
      '.cm-content': {
        caretColor: 'var(--accent)',
        padding: '40px 56px 120px',
        maxWidth: '68ch',
        margin: '0 auto',
      },
      '.cm-line': { padding: '0' },
      '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--accent)', borderLeftWidth: '2px' },
      '&.cm-focused .cm-selectionBackground, ::selection, .cm-selectionBackground': {
        backgroundColor: 'var(--accent-soft)',
      },
      '.cm-activeLine': { backgroundColor: 'transparent' },
      '.cm-activeLineGutter': { backgroundColor: 'transparent' },
      '.cm-gutters': {
        backgroundColor: 'transparent',
        color: 'var(--fg-subtle)',
        border: 'none',
        fontFamily: 'var(--font-mono)',
        fontSize: '11px',
      },
      '.cm-gutterElement': { padding: '0 8px' },
      '.cm-foldGutter .cm-gutterElement': { color: 'var(--fg-subtle)' },
      '.cm-fat-cursor': { backgroundColor: 'var(--accent)' },
      '&.cm-focused': { outline: 'none' },
      // markdown code blocks: switch to mono
      '.cm-line .ͼ8, .cm-line .tok-monospace, .cm-line span[class*="codeText"]': {
        fontFamily: 'var(--font-mono)',
        fontSize: '13px',
      },
    },
    { dark: matchMedia('(prefers-color-scheme: dark)').matches },
  )

  // Compartment for re-themable bits (in case prefs change later)
  const themeCompartment = new Compartment()

  function makeState(initial: string): EditorState {
    return EditorState.create({
      doc: initial,
      extensions: [
        lineNumbers(),
        highlightActiveLineGutter(),
        highlightActiveLine(),
        foldGutter(),
        drawSelection(),
        indentOnInput(),
        bracketMatching(),
        history(),
        markdown({ base: markdownLanguage }),
        syntaxHighlighting(editorHighlight),
        themeCompartment.of(editorTheme),
        EditorView.lineWrapping,
        EditorState.tabSize.of(2),
        keymap.of([
          ...defaultKeymap,
          ...historyKeymap,
          ...foldKeymap,
          ...searchKeymap,
          indentWithTab,
          {
            key: 'Mod-s',
            run: () => {
              dispatch('save')
              return true
            },
          },
        ]),
        EditorView.updateListener.of((u) => {
          if (u.docChanged) {
            const v = u.state.doc.toString()
            dispatch('change', v)
          }
        }),
      ],
    })
  }

  // External value updates (e.g., opening a different file) should reset the
  // doc. Track the last-set value so we don't blow away in-progress edits when
  // the same value comes back via the parent's reactivity.
  let lastExternalValue = value
  $: if (view && value !== lastExternalValue) {
    lastExternalValue = value
    const current = view.state.doc.toString()
    if (current !== value) {
      view.dispatch({
        changes: { from: 0, to: current.length, insert: value },
      })
    }
  }

  onMount(() => {
    if (!editorEl) return
    view = new EditorView({
      state: makeState(value),
      parent: editorEl,
    })
    setTimeout(() => view?.focus(), 0)
  })

  onDestroy(() => {
    view?.destroy()
    view = undefined
  })

  export function focus() {
    view?.focus()
  }
</script>

<div class="editor-wrap">
  <div class="editor" bind:this={editorEl}></div>
  {#if dirty}
    <div class="dirty-indicator" aria-hidden="true">●</div>
  {/if}
</div>

<style>
  .editor-wrap {
    flex: 1;
    min-width: 0;
    height: 100%;
    position: relative;
    background: var(--bg);
    overflow: hidden;
  }

  .editor {
    height: 100%;
  }

  /* Restrained line numbers — almost invisible */
  :global(.editor-wrap .cm-gutters) {
    border-right: 1px solid var(--rule) !important;
  }

  :global(.editor-wrap .cm-line) {
    color: var(--fg);
  }

  .dirty-indicator {
    position: absolute;
    top: 8px;
    right: 14px;
    color: var(--accent);
    font-size: 10px;
    pointer-events: none;
    line-height: 1;
  }
</style>
