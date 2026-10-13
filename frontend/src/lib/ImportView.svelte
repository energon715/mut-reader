<script lang="ts">
  import Icon from "./Icon.svelte";
  import { onDestroy } from "svelte";
  import PathField from "./PathField.svelte";
  export let preview = false;
  export let initial = "choose";
  export let onimport: (path: string) => Promise<void>;
  export let onclose: () => void;
  export let importDirectory: string;
  export let defaultImportDirectory: string;
  export let pickerAvailable = false;
  export let directoryPickerAvailable = false;
  export let onpathchange: (value: string) => void;
  export let onbrowse: (title: string, currentPath: string) => Promise<string>;
  export let onselectfiles: () => Promise<string[]>;
  type Item = { name: string; format: string; state: string; error?: string };
  let paths = "";
  let items: Item[] = [];
  let state = initial;
  let busy = false;
  let cancelled = false;
  let processed = 0;
  let selecting = false;
  let pickerError = "";
  async function selectFiles() {
    selecting = true;
    pickerError = "";
    try {
      const selected = await onselectfiles();
      if (selected.length)
        paths = [
          ...new Set([...paths.split("\n").filter(Boolean), ...selected]),
        ].join("\n");
    } catch {
      pickerError =
        "Не удалось открыть выбор файлов. Можно ввести пути вручную.";
    } finally {
      selecting = false;
    }
  }
  onDestroy(() => {
    cancelled = true;
  });
  $: percent = items.length ? Math.round((processed / items.length) * 100) : 0;
  $: if (initial === "duplicate" && items.length === 0) {
    items = [
      { name: "Orbits of Silence", format: "EPUB", state: "Сохранено" },
      { name: "Coastal Atlas", format: "PDF", state: "В очереди" },
      { name: "Map of Winds", format: "PDF", state: "Дубликат" },
    ];
    processed = 1;
  }
  $: if (initial === "issues" && items.length === 0) {
    items = [
      { name: "Orbits of Silence", format: "EPUB", state: "Сохранено" },
      {
        name: "Coastal Atlas",
        format: "PDF",
        state: "Ошибка",
        error: "Не удалось прочитать файл. Проверьте доступ к нему.",
      },
      { name: "Map of Winds", format: "PDF", state: "Пропущено" },
    ];
    processed = 3;
    state = "done";
  }
  async function start() {
    const files = paths
      .split("\n")
      .map((p) => p.trim())
      .filter(Boolean);
    if (!files.length) return;
    items = files.map((path) => ({
      name: path.split("/").at(-1) || path,
      format: path.split(".").at(-1)?.toUpperCase() || "FILE",
      state: "В очереди",
    }));
    state = "progress";
    busy = true;
    cancelled = false;
    processed = 0;
    for (let i = 0; i < files.length; i++) {
      if (cancelled) break;
      items[i]!.state = "Сохраняем…";
      items = [...items];
      try {
        await onimport(files[i]!);
        items[i]!.state = "Сохранено";
      } catch (error) {
        const message = String(error);
        items[i]!.state = /duplicate|already exists|дубликат/i.test(message)
          ? "Пропущено"
          : "Ошибка";
        items[i]!.error =
          items[i]!.state === "Ошибка"
            ? "Файл не добавлен. Проверьте путь, доступ и целостность файла."
            : "Такой файл уже есть в библиотеке.";
      }
      processed = i + 1;
      items = [...items];
    }
    busy = false;
    state = "done";
  }
  function resolveDuplicate(edition = false) {
    items = items.map((item) =>
      item.state === "Дубликат"
        ? { ...item, state: edition ? "Сохранено" : "Пропущено" }
        : {
            ...item,
            state: item.state === "В очереди" ? "Сохранено" : item.state,
          },
    );
    processed = items.length;
    state = "done";
  }
</script>

{#if state === "choose"}<div class="stack">
    <p class="muted">
      Добавьте одну книгу или несколько файлов. Оригиналы останутся на своих
      местах.
    </p>
    <div class="import-art">
      <Icon name="upload" size={32} /><strong
        >Каждой книге найдётся место</strong
      ><span>PDF · EPUB · FB2 · любой другой формат</span>
    </div>
    <form class="stack" on:submit|preventDefault={start}>
      <PathField
        label="Папка с книгами"
        bind:value={importDirectory}
        defaultPath={defaultImportDirectory}
        pickerAvailable={directoryPickerAvailable}
        onchange={onpathchange}
        {onbrowse}
      />
      <button
        class="secondary choose-files"
        type="button"
        disabled={!pickerAvailable || selecting}
        on:click={selectFiles}
        ><Icon name="folder" size={18} />{selecting
          ? "Открываем…"
          : "Выбрать файлы…"}</button
      >
      {#if !pickerAvailable}<small class="muted"
          >Системный выбор файлов пока не подключён. Укажите пути ниже.</small
        >{/if}
      {#if pickerError}<p class="error" role="alert">{pickerError}</p>{/if}
      <label class="field"
        ><span>Пути к файлам</span><textarea
          bind:value={paths}
          rows="3"
          placeholder={importDirectory +
            "/book.epub\n" +
            importDirectory +
            "/notes.pdf"}
          required></textarea><small class="muted"
          >Один полный путь на строку.</small
        ></label
      >
      <div class="notice">
        <strong>Хранение и чтение — разные вещи</strong>
        <p>
          Файл любого формата можно сохранить в архив. Встроенное чтение
          предусмотрено для PDF, EPUB и FB2.
        </p>
      </div>
      <div class="footer-row">
        <button type="button" class="secondary" on:click={onclose}
          >Отмена</button
        ><button class="primary" type="submit" disabled={!paths.trim()}
          >Добавить <Icon name="arrow" size={16} /></button
        >
      </div>
    </form>
  </div>
{:else}<div class="stack">
    <div>
      <div class="row spread">
        <span
          >{state === "done" ? "Обработано" : "Сохранено"}
          {processed} из {items.length}</span
        ><span>{percent}%</span>
      </div>
      <div class="progress-track import-progress">
        <span style:width={percent + "%"}></span>
      </div>
      <p class="muted import-caption">
        Файлы шифруются на этом устройстве. Оригиналы не удаляются.
      </p>
    </div>
    <div class="import-files">
      {#each items as item}<div class="import-file">
          <span class="file-tag">{item.format}</span>
          <div>
            <strong>{item.name}</strong>{#if item.error}<small
                >{item.error}</small
              >{/if}
          </div>
          <span class="file-state" class:failed={item.state === "Ошибка"}
            >{item.state}</span
          >
        </div>{/each}
    </div>
    {#if state === "duplicate"}<div class="duplicate notice">
        <div class="row spread">
          <strong>«Map of Winds» уже в библиотеке</strong><small>ДУБЛИКАТ</small
          >
        </div>
        <p>
          Содержимое файлов совпадает. Выберите, что сделать с этим экземпляром.
        </p>
        <div class="row">
          <button class="primary" on:click={() => resolveDuplicate()}
            >Пропустить</button
          ><button class="secondary" on:click={() => resolveDuplicate(true)}
            >Сохранить как редакцию</button
          >
        </div>
        <label class="check"
          ><input type="checkbox" /> Применить к оставшимся дубликатам</label
        >
      </div>{/if}{#if state === "done"}<div class="notice">
        <strong>{cancelled ? "Импорт остановлен" : "Импорт завершён"}</strong>
        <p>
          {items.filter((i) => i.state === "Сохранено").length} сохранено · {items.filter(
            (i) => i.state === "Пропущено",
          ).length} пропущено · {items.filter((i) => i.state === "Ошибка")
            .length} ошибок. Сохранённые книги уже в библиотеке.
        </p>
      </div>{/if}
    <div class="footer-row">
      <span class="muted">{preview ? "Демонстрация интерфейса" : ""}</span
      >{#if busy}<button
          class="secondary"
          on:click={() => (cancelled = true)}
          disabled={cancelled}
          >{cancelled ? "Останавливаем…" : "Остановить импорт"}</button
        >{:else}<button class="primary" on:click={onclose}
          >Вернуться к книгам <Icon name="arrow" size={16} /></button
        >{/if}
    </div>
  </div>{/if}

<style>
  .import-art {
    padding: 30px;
    border: 1px dashed var(--quiet);
    border-radius: 18px;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    color: var(--muted);
  }
  .import-art strong {
    color: var(--text);
    font-size: 16px;
    font-weight: 500;
  }
  .import-art span {
    font-size: 11px;
  }
  .import-progress {
    margin: 14px 0;
    height: 5px;
  }
  .import-caption {
    font-size: 12px;
  }
  .import-files {
    max-height: 280px;
    overflow: auto;
  }
  .import-file {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px 0;
    border-bottom: 1px solid var(--line);
  }
  .file-tag {
    display: grid;
    place-items: center;
    min-width: 46px;
    height: 54px;
    background: var(--active);
    border-radius: 8px;
    font-size: 9px;
  }
  .import-file > div {
    flex: 1;
    min-width: 0;
  }
  .import-file strong {
    font-weight: 400;
    overflow-wrap: anywhere;
  }
  .import-file small {
    display: block;
    color: var(--muted);
    margin-top: 5px;
  }
  .file-state {
    font-size: 11px;
    color: var(--muted);
    white-space: nowrap;
  }
  .failed {
    color: var(--error);
  }
  .duplicate {
    background: var(--input);
  }
  .duplicate .row {
    flex-wrap: wrap;
  }
  .duplicate .check {
    font-size: 11px;
    color: var(--muted);
    margin-top: 8px;
  }
  .duplicate small {
    color: var(--muted);
    font-size: 9px;
  }
</style>
