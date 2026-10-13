<script lang="ts">
  import { onMount } from "svelte";
  import Icon from "./Icon.svelte";
  export let title: string;
  export let subtitle = "";
  export let wide = false;
  export let anchored = false;
  export let onclose: () => void;
  let dialog: HTMLDialogElement;
  onMount(() => {
    dialog.showModal();
    return () => dialog.close();
  });
  function backdrop(event: MouseEvent) {
    if (event.target !== dialog) return;
    const box = dialog.getBoundingClientRect();
    if (
      event.clientX < box.left ||
      event.clientX > box.right ||
      event.clientY < box.top ||
      event.clientY > box.bottom
    )
      onclose();
  }
</script>

<dialog
  bind:this={dialog}
  class:wide
  class:anchored
  aria-label={title}
  on:cancel|preventDefault={onclose}
  on:click={backdrop}
  on:keydown={(event) => {
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      onclose();
    }
  }}
>
  <header>
    <div>
      <h2>{title}</h2>
      {#if subtitle}<p class="muted">{subtitle}</p>{/if}
    </div>
    <button class="icon-button" on:click={onclose} aria-label="Закрыть"
      ><Icon name="close" size={16} /></button
    >
  </header>
  <div class="dialog-content"><slot /></div>
</dialog>

<style>
  dialog {
    padding: 0;
    border: 1px solid var(--border);
    border-radius: 22px;
    background: var(--panel);
    color: var(--text);
    width: min(600px, calc(100vw - 40px));
    max-height: calc(100vh - 64px);
    box-shadow: 0 24px 80px #0005;
    overflow: auto;
  }
  dialog::backdrop {
    background: #0007;
    backdrop-filter: blur(5px);
  }
  dialog.wide {
    width: min(820px, calc(100vw - 40px));
  }
  dialog.anchored {
    margin: 72px auto auto 16px;
  }
  header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 24px;
    padding: 24px 28px;
    border-bottom: 1px solid var(--line);
  }
  header p {
    font-size: 12px;
    margin-top: 5px;
  }
  .dialog-content {
    padding: 28px;
  }
  .wide .dialog-content {
    padding: 0;
  }
  @media (max-width: 650px) {
    header {
      padding: 22px;
    }
    .dialog-content {
      padding: 22px;
    }
    dialog.anchored {
      margin: auto;
    }
    header h2 {
      font-size: 22px;
    }
  }
  @media (prefers-reduced-motion: no-preference) {
    dialog[open] {
      animation: appear 0.15s ease-out;
    }
    @keyframes appear {
      from {
        opacity: 0;
        transform: translateY(6px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
  }
</style>
