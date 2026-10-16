<script lang="ts">
  import Icon from "./Icon.svelte";
  export let label: string;
  export let value: string;
  export let defaultPath: string;
  export let pickerAvailable = false;
  export let onchange: (value: string) => void;
  export let onbrowse: (title: string, currentPath: string) => Promise<string>;
  let choosing = false;
  let error = "";
  async function browse() {
    if (choosing || !pickerAvailable) return;
    choosing = true;
    error = "";
    try {
      const selected = await onbrowse(label, value || defaultPath);
      if (selected) {
        value = selected;
        onchange(selected);
      }
    } catch {
      error = "Не удалось открыть выбор папки. Можно ввести путь вручную.";
    } finally {
      choosing = false;
    }
  }
</script>

<div class="field">
  <div class="row spread">
    <span class="path-label">{label}</span><button
      class="text-button default-button"
      type="button"
      on:click={() => {
        value = defaultPath;
        onchange(value);
      }}>По умолчанию</button
    >
  </div>
  <div class="input-wrap">
    <input
      aria-label={label}
      bind:value
      placeholder={defaultPath}
      required
      autocomplete="off"
      on:input={(event) => onchange(event.currentTarget.value)}
      on:change={() => onchange(value.trim())}
    />
    <button
      type="button"
      class="browse-button"
      disabled={!pickerAvailable || choosing}
      aria-label={"Выбрать папку: " + label}
      title={pickerAvailable
        ? "Открыть системный выбор папки"
        : "Выбор папки пока не подключён"}
      on:click={browse}
      ><Icon name="folder" size={16} /><span
        >{choosing ? "Открываем…" : "Обзор…"}</span
      ></button
    >
  </div>
  <small class="default-path muted">По умолчанию: {defaultPath}</small>
  {#if !pickerAvailable}<small class="muted"
      >Пока можно ввести путь вручную.</small
    >{/if}
  {#if error}<small class="error" role="alert">{error}</small>{/if}
</div>

<style>
  .path-label {
    font-size: 13px;
  }
  .default-button {
    font-size: 10px;
    white-space: nowrap;
    flex-shrink: 0;
  }
  .browse-button {
    padding: 8px !important;
  }
  .default-path {
    overflow-wrap: anywhere;
    user-select: text;
    font-size: 10px;
  }
  @media (max-width: 480px) {
    .browse-button span {
      display: none;
    }
  }
</style>
