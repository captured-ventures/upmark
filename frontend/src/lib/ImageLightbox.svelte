<script lang="ts">
  export let src: string = ''
  export let alt: string = ''
  export let open: boolean = false

  function close() {
    open = false
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
    }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <div class="lb-backdrop" on:click={close} role="dialog">
    <figure class="lb-frame" on:click|stopPropagation>
      <img {src} alt={alt} />
      {#if alt}
        <figcaption>{alt}</figcaption>
      {/if}
      <button class="lb-close" on:click={close} aria-label="Close" title="Close (Esc)">
        <svg width="14" height="14" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M3 3l10 10M13 3L3 13"/></svg>
      </button>
    </figure>
  </div>
{/if}

<style>
  .lb-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(20, 18, 16, 0.94);
    z-index: 2000;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: zoom-out;
    animation: lb-fade 160ms ease;
  }

  .lb-frame {
    margin: 0;
    max-width: 92vw;
    max-height: 92vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    position: relative;
    cursor: default;
  }

  .lb-frame img {
    max-width: 92vw;
    max-height: 84vh;
    object-fit: contain;
    border-radius: 3px;
    background: var(--bg-elev);
  }

  .lb-frame figcaption {
    font-family: var(--font-serif);
    font-style: italic;
    font-size: 13px;
    color: rgba(232, 226, 213, 0.7);
    text-align: center;
    max-width: 480px;
    font-variation-settings: "opsz" 14;
  }

  .lb-close {
    position: absolute;
    top: -36px;
    right: -4px;
    width: 28px;
    height: 28px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    color: rgba(232, 226, 213, 0.65);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: color 120ms ease, background 120ms ease;
  }
  .lb-close:hover {
    color: rgb(232, 226, 213);
    background: rgba(232, 226, 213, 0.08);
  }

  @keyframes lb-fade {
    from { opacity: 0; }
    to { opacity: 1; }
  }
</style>
