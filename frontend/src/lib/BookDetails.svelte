<script lang="ts">
  import BookCover from "./BookCover.svelte";
  import Icon from "./Icon.svelte";
  import { formatSize, canRead, type Book } from "./ui";
  export let book: Book;
  export let shelves: string[] = [];
  export let onread: () => void;
  export let onsave: (shelves: string[]) => Promise<void>;
  export let onexport: () => Promise<void>;
  export let ondelete: () => Promise<void>;
  let chosen = [...book.shelves];
  let deleting = false;
  let busy = false;
  let error = "";
  let saved = false;
  function toggle(shelf: string) {
    chosen = chosen.includes(shelf)
      ? chosen.filter((s) => s !== shelf)
      : [...chosen, shelf];
    saved = false;
  }
  async function run(action: () => Promise<void>) {
    busy = true;
    error = "";
    try {
      await action();
    } catch {
      error =
        "Не удалось выполнить действие. Проверьте доступ к файлам и попробуйте снова.";
    } finally {
      busy = false;
    }
  }
</script>

<div class="details-layout">
  <div class="detail-cover"><BookCover {book} /></div>
  <div class="detail-info">
    <span class="eyebrow">{book.format} · {formatSize(book.size)}</span>
    <h1>{book.title}</h1>
    <p class="muted">{book.author}</p>
    <small class="muted">{book.originalName}</small
    >{#if canRead(book.format)}<button
        class="primary"
        disabled={busy}
        on:click={onread}><Icon name="book" size={16} /> Читать</button
      >{:else}<div class="notice">
        <strong>Нет встроенного чтения</strong>
        <p>
          Оригинал сохранён в архиве. Экспортируйте его для подходящей
          программы.
        </p>
      </div>{/if}
    <div class="field">
      <span>Полки</span>{#each shelves as shelf}<label class="check"
          ><input
            type="checkbox"
            checked={chosen.includes(shelf)}
            on:change={() => toggle(shelf)}
          />
          {shelf}</label
        >{/each}{#if !shelves.length}<small class="muted"
          >Сначала создайте полку в библиотеке.</small
        >{:else}<button
          class="secondary"
          disabled={busy || saved}
          on:click={() =>
            run(async () => {
              await onsave(chosen);
              saved = true;
            })}>{saved ? "Сохранено" : "Сохранить полки"}</button
        >{/if}
    </div>
    <button class="secondary" disabled={busy} on:click={() => run(onexport)}
      ><Icon name="upload" size={16} /> Экспортировать оригинал</button
    ><small class="muted"
      >В выбранной папке появится незашифрованный файл. Существующие файлы не
      перезаписываются.</small
    >{#if deleting}<div class="notice">
        <strong>Убрать книгу из библиотеки?</strong>
        <p>
          Запись и позиции исчезнут из текущего каталога. Исходный файл и
          прежние резервные копии сохранятся.
        </p>
        <div class="row">
          <button class="secondary" on:click={() => (deleting = false)}
            >Отмена</button
          ><button
            class="primary"
            disabled={busy}
            on:click={() => run(ondelete)}>Убрать книгу</button
          >
        </div>
      </div>{:else}<button
        class="text-button"
        on:click={() => (deleting = true)}>Убрать из библиотеки</button
      >{/if}{#if error}<p class="error" role="alert">{error}</p>{/if}
  </div>
</div>

<style>
  .details-layout {
    display: flex;
    gap: 24px;
    align-items: flex-start;
  }
  .detail-cover {
    width: 160px;
    flex-shrink: 0;
  }
  .detail-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 14px;
  }
  .detail-info h1 {
    font-size: 25px;
  }
  .detail-info small {
    font-size: 10px;
    overflow-wrap: anywhere;
  }
  .detail-info .notice {
    padding: 14px;
  }
  .detail-info .notice p {
    font-size: 12px;
  }
  @media (max-width: 550px) {
    .details-layout {
      flex-direction: column;
    }
    .detail-cover {
      width: 120px;
      align-self: center;
    }
    .detail-cover :global(.book-cover) {
      height: 160px;
    }
    .detail-cover :global(.cover-title) {
      font-size: 16px;
    }
  }
</style>
