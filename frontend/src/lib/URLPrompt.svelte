<script lang="ts">
  import { tick, createEventDispatcher } from 'svelte'

  export let open: boolean = false

  const dispatch = createEventDispatcher<{ submit: string }>()

  let inputEl: HTMLInputElement | undefined
  let url = ''
  let busy = false
  let errorMsg = ''

  $: if (open) {
    queueMicrotask(() => {
      tick().then(() => inputEl?.focus())
    })
  } else {
    url = ''
    errorMsg = ''
    busy = false
  }

  async function submit() {
    const trimmed = url.trim()
    if (!trimmed) return
    if (!/^https?:\/\//i.test(trimmed)) {
      errorMsg = 'URL must start with http:// or https://'
      return
    }
    errorMsg = ''
    busy = true
    dispatch('submit', trimmed)
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      e.preventDefault()
      submit()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      open = false
    }
  }
</script>

{#if open}
  <div class="url-backdrop" on:click={() => (open = false)}>
    <div class="url-prompt" on:click|stopPropagation>
      <div class="url-row">
        <span class="url-glyph" aria-hidden="true">↗</span>
        <input
          bind:this={inputEl}
          bind:value={url}
          on:keydown={onKey}
          placeholder="paste a markdown URL · github.com link works too"
          type="url"
          autocomplete="off"
          spellcheck="false"
          disabled={busy}
        />
        <button on:click={submit} disabled={busy || !url.trim()}>
          {busy ? 'fetching…' : 'open'}
        </button>
        <span class="url-esc">esc</span>
      </div>
      {#if errorMsg}
        <div class="url-error">{errorMsg}</div>
      {/if}
      <div class="url-hint">
        <span class="url-hint-label">examples</span>
        <code>https://raw.githubusercontent.com/user/repo/main/README.md</code>
        <code>https://github.com/user/repo/blob/main/docs/spec.md</code>
      </div>
    </div>
  </div>
{/if}

<style>
  .url-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.20);
    z-index: 1100;
    display: flex;
    align-items: flex-start;
    justify-content: center;
    padding-top: 18vh;
    animation: bdfade 140ms ease;
  }
  @media (prefers-color-scheme: dark) {
    .url-backdrop { background: rgba(0, 0, 0, 0.46); }
  }

  .url-prompt {
    width: 620px;
    max-width: 92vw;
    background: var(--bg);
    border: 1px solid var(--rule-strong);
    border-radius: 8px;
    box-shadow: 0 24px 60px -20px rgba(0, 0, 0, 0.35),
                0 2px 8px rgba(0, 0, 0, 0.12);
    overflow: hidden;
    animation: prompt-in 200ms cubic-bezier(0.16, 1, 0.3, 1);
  }

  .url-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 0 16px;
    height: 52px;
    border-bottom: 1px solid var(--rule);
  }

  .url-glyph {
    font-family: var(--font-mono);
    color: var(--accent);
    font-size: 16px;
    font-weight: 500;
    line-height: 1;
  }

  .url-row input {
    flex: 1;
    min-width: 0;
    border: none;
    outline: none;
    background: transparent;
    color: var(--fg);
    font-family: var(--font-sans);
    font-size: 14px;
    padding: 0;
  }
  .url-row input::placeholder {
    color: var(--fg-subtle);
    font-style: italic;
    font-family: var(--font-serif);
    font-size: 13px;
  }
  .url-row input:disabled { color: var(--fg-muted); }

  .url-row button {
    font-family: var(--font-sans);
    font-size: 12px;
    color: var(--accent);
    background: var(--accent-soft);
    border: 1px solid transparent;
    padding: 5px 12px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 100ms ease, border-color 100ms ease;
  }
  .url-row button:hover:not(:disabled) {
    border-color: var(--accent);
  }
  .url-row button:disabled { opacity: 0.4; cursor: default; }

  .url-esc {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--fg-subtle);
    padding: 2px 6px;
    border: 1px solid var(--rule);
    border-radius: 3px;
  }

  .url-error {
    padding: 8px 16px;
    background: var(--accent-soft);
    color: var(--accent);
    font-family: var(--font-sans);
    font-size: 12px;
    border-bottom: 1px solid var(--rule);
  }

  .url-hint {
    padding: 10px 16px 14px;
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--fg-subtle);
  }
  .url-hint-label {
    font-family: var(--font-sans);
    font-size: 10px;
    font-weight: 500;
    letter-spacing: 0.14em;
    text-transform: uppercase;
    color: var(--fg-subtle);
    margin-bottom: 2px;
  }
  .url-hint code {
    background: transparent;
    border: none;
    padding: 0;
    font-size: 11px;
    color: var(--fg-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  @keyframes bdfade {
    from { opacity: 0; }
    to { opacity: 1; }
  }
  @keyframes prompt-in {
    from { opacity: 0; transform: translateY(-8px) scale(0.99); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }
</style>
