<script lang="ts">
  import { onMount, onDestroy, tick } from "svelte";
  import Icon from "./Icon.svelte";
  import PdfPage from "./PdfPage.svelte";
  import { chapters, readingText } from "./demo";
  import type { Book } from "./ui";
  import { OpenBook, SavePosition, SaveBookmarks } from "../wailsjs/go/app/App";
  import type { readers } from "../wailsjs/go/models";
  export let book: Book;
  export let preview = false;
  export let fontSize = 20;
  export let lineHeight = 1.5;
  export let onback: () => void;
  export let onsettings: () => void;
  export let onsecurity: () => void;
  export let onchange: (key: string, value: string | number) => void;
  export let onprogress: (
    id: string,
    progress: number,
    locator: string,
  ) => void = () => {};
  export let onbookmarks: (id: string, marks: string[]) => void = () => {};
  let bookmarkBusy = false;
  type TextPage = {
    title: string;
    chapter: number;
    start: number;
    blocks: readers.Block[];
  };
  let document: readers.Document | null = null;
  let pages: TextPage[] = [];
  let outline: { title: string; page: number }[] = [];
  let page = 1;
  let total = 1;
  let tab = "contents";
  let sidebarOpen = false;
  let pageTheme = "paper";
  let appearance = false;
  let zoom = 100;
  let loading = !preview;
  let error = "";
  let disposed = false;
  let marks = [...book.bookmarks];
  let saveQueue = Promise.resolve();
  let readingArea: HTMLElement;
  let pageInput = 1;
  let bookmarkMessage = "";
  let bookmarkError = "";
  $: pageInput = page;
  let orderedMarks: string[] = [];
  $: if (pages && total) orderedMarks = [...marks].sort((a, b) => (markPage(a) || Infinity) - (markPage(b) || Infinity));
  $: isPdf = book.format === "PDF";
  $: progress = Math.round((page / total) * 100);
  $: locator = isPdf
    ? "v1:pdf:" + page
    : preview
      ? "v1:demo:" + page
      : "v1:text:" +
        (pages[page - 1]?.chapter || 0) +
        ":" +
        (pages[page - 1]?.start || 0);
  $: bookmarked = marks.includes(locator);
  onMount(() => {
    if (preview) {
      total = chapters.length;
      page = 4;
      outline = chapters.map((title, i) => ({ title, page: i + 1 }));
      return;
    }
    void OpenBook(book.id)
      .then((doc) => {
        if (disposed) return;
        document = doc;
        if (!isPdf) {
          for (let c = 0; c < doc.chapters.length; c++) {
            const chapter = doc.chapters[c]!;
            outline = [
              ...outline,
              { title: chapter.title, page: pages.length + 1 },
            ];
            let blocks: readers.Block[] = [];
            let chars = 0;
            let start = 0;
            for (let b = 0; b < chapter.blocks.length; b++) {
              const block = chapter.blocks[b]!;
              if (
                blocks.length &&
                (chars + block.text.length > 7000 || blocks.length >= 60)
              ) {
                pages.push({ title: chapter.title, chapter: c, start, blocks });
                blocks = [];
                chars = 0;
                start = b;
              }
              blocks.push(block);
              chars += block.text.length;
            }
            if (blocks.length)
              pages.push({ title: chapter.title, chapter: c, start, blocks });
          }
          total = Math.max(1, pages.length);
          const parts = book.locator?.split(":");
          if (parts?.[1] === "text") {
            const found = pages.findIndex(
              (p) =>
                p.chapter === Number(parts[2]) && p.start === Number(parts[3]),
            );
            page = found >= 0 ? found + 1 : 1;
          }
        } else {
          const parts = book.locator?.split(":");
          if (parts?.[1] === "pdf") page = Math.max(1, Number(parts[2]) || 1);
        }
        loading = false;
        if (!isPdf) persist();
      })
      .catch(() => {
        if (!disposed) {
          loading = false;
          error =
            "Не удалось открыть книгу. Проверьте файл или экспортируйте оригинал для внешней программы.";
        }
      });
  });
  onDestroy(() => {
    disposed = true;
    document = null;
    pages = [];
    marks = [];
  });
  function persist() {
    if (preview || disposed || !document) return;
    const value = isPdf
      ? "v1:pdf:" + page
      : "v1:text:" +
        (pages[page - 1]?.chapter || 0) +
        ":" +
        (pages[page - 1]?.start || 0);
    const pct = Math.round((page / total) * 100);
    saveQueue = saveQueue
      .then(async () => {
        await SavePosition(book.id, pct, value);
        if (!disposed) onprogress(book.id, pct, value);
      })
      .catch(() => {
        if (!disposed)
          error = "Не удалось сохранить позицию. Проверьте доступ к диску.";
      });
  }
  function go(next: number) {
    if (!Number.isFinite(next)) { pageInput = page; return; }
    page = Math.max(1, Math.min(total, Math.trunc(next)));
    pageInput = page;
    sidebarOpen = false;
    bookmarkMessage = "";
    persist();
    // Text pages share DOM nodes; reset scrolling after Svelte updates them.
    void tick().then(() => { if (!disposed) readingArea?.scrollTo({ top: 0, left: 0 }); });
  }
  function navigate(delta: number) {
    go(page + delta);
  }
  function pdfReady(count: number, toc: { title: string; page: number }[]) {
    total = count;
    page = Math.min(page, total);
    outline = toc.length
      ? toc
      : Array.from({ length: Math.min(count, 1000) }, (_, i) => ({
          title: "Страница " + (i + 1),
          page: i + 1,
        }));
    persist();
  }
  async function leave() {
    await saveQueue;
    if (!disposed) onback();
  }
  async function updateMarks(next: string[]) {
    if (bookmarkBusy) return;
    bookmarkBusy = true;
    bookmarkError = "";
    const added = next.find(mark => !marks.includes(mark));
    try {
      if (!preview) await SaveBookmarks(book.id, next);
      if (!disposed) {
        marks = next;
        onbookmarks(book.id, next);
        bookmarkMessage = added ? "Закладка добавлена" : "Закладка удалена";
        if (added) { tab = "marks"; sidebarOpen = true; }
      }
    } catch {
      if (!disposed) bookmarkError = "Не удалось сохранить закладку. Попробуйте ещё раз.";
    } finally { bookmarkBusy = false; }
  }
  async function bookmark() {
    const value = locator;
    const next = marks.includes(value)
      ? marks.filter((m) => m !== value)
      : [...marks, value];
    await updateMarks(next);
  }
  function markPage(mark: string): number | null {
    const parts = mark.split(":");
    if (parts[0] !== "v1") return null;
    if ((isPdf && parts[1] === "pdf") || (preview && parts[1] === "demo")) {
      const value = Number(parts[2]);
      return Number.isInteger(value) && value >= 1 && value <= total ? value : null;
    }
    if (parts[1] !== "text" || parts.length !== 4) return null;
    const found = pages.findIndex(item => item.chapter === Number(parts[2]) && item.start === Number(parts[3]));
    return found >= 0 ? found + 1 : null;
  }
  function markTitle(mark: string) {
    const index = markPage(mark);
    return index ? (isPdf ? "Страница " : "Часть ") + index : "Позиция недоступна";
  }
  function markDescription(mark: string) {
    const index = markPage(mark);
    if (!index) return "Закладка из другой версии книги";
    if (isPdf) return "PDF · нажмите, чтобы перейти";
    return preview ? chapters[index - 1] || "" : pages[index - 1]?.title || "";
  }
  function openBookmarks() {
    tab = "marks";
    sidebarOpen = true;
  }
  function shortcut(e: KeyboardEvent) {
    if (
      window.document.querySelector("dialog[open]") ||
      e.target instanceof HTMLInputElement ||
      e.target instanceof HTMLSelectElement ||
      e.target instanceof HTMLTextAreaElement ||
      (e.target instanceof HTMLElement && e.target.isContentEditable) ||
      e.altKey || e.ctrlKey || e.metaKey || loading || Boolean(error)
    )
      return;
    if (e.key === "ArrowRight") { e.preventDefault(); navigate(1); }
    if (e.key === "ArrowLeft") { e.preventDefault(); navigate(-1); }
    if (e.key === "Escape") {
      if (appearance) appearance = false;
      else if (sidebarOpen) sidebarOpen = false;
      else void leave();
    }
  }
</script>

<svelte:window on:keydown={shortcut} />
<div class="reader-layout">
  <header class="reader-header">
    <button class="back-library panel" on:click={leave}
      ><Icon name="back" /><span>Все книги</span></button
    >
    <div class="reader-title">
      <strong>{book.title}</strong><small>{book.author}</small>
    </div>
    <div class="reader-actions">
      <button
        class="icon-button mobile-toc"
        aria-label="Показать оглавление"
        on:click={() => (sidebarOpen = !sidebarOpen)}
        ><Icon name="list" /></button
      ><button
        class="bookmark-toggle"
        class:marked={bookmarked}
        aria-pressed={bookmarked}
        aria-label={bookmarked ? "Удалить закладку с текущей страницы" : "Добавить закладку на текущую страницу"}
        title={bookmarked ? "Удалить закладку с текущей страницы" : "Сохранить текущую страницу в закладках"}
        disabled={loading || Boolean(error) || bookmarkBusy}
        on:click={bookmark}><Icon name={bookmarked ? "check" : "bookmark"} /><span>{bookmarkBusy ? "Сохраняем…" : bookmarked ? "В закладках" : "Добавить закладку"}</span></button
      ><button class="bookmarks-toggle" aria-label="Открыть список закладок" on:click={openBookmarks}
        ><Icon name="list" /><span>Закладки</span><span class="bookmark-count">{marks.length}</span></button
      ><button
        class="icon-button"
        aria-label={isPdf ? "Уменьшить масштаб" : "Уменьшить текст"}
        on:click={() =>
          isPdf
            ? (zoom = Math.max(50, zoom - 10))
            : onchange("fontSize", Math.max(16, fontSize - 2))}
        ><Icon name="minus" /></button
      ><button
        class="icon-button"
        aria-label={isPdf ? "Увеличить масштаб" : "Увеличить текст"}
        on:click={() =>
          isPdf
            ? (zoom = Math.min(200, zoom + 10))
            : onchange("fontSize", Math.min(28, fontSize + 2))}
        ><Icon name="plus" /></button
      ><button
        class="icon-button"
        aria-label="Внешний вид страницы"
        aria-expanded={appearance}
        on:click={() => (appearance = !appearance)}><Icon name="more" /></button
      >
    </div>
  </header>
  <div class="reader-body">
    <aside class="reader-sidebar" class:open={sidebarOpen}>
      <div class="panel contents">
        <div class="toc-tabs">
          <button
            class:active={tab === "contents"}
            aria-pressed={tab === "contents"}
            on:click={() => (tab = "contents")}>Оглавление</button
          ><button
            class:active={tab === "marks"}
            aria-pressed={tab === "marks"}
            on:click={() => (tab = "marks")}>Закладки ({marks.length})</button
          ><button class="sidebar-close icon-button" aria-label="Закрыть оглавление и закладки" on:click={() => (sidebarOpen = false)}><Icon name="close" size={16} /></button>
        </div>
        <div class="hairlines" aria-hidden="true"><i></i><i></i><i></i></div>
        <div class="toc-scroll">
          {#if tab === "contents"}{#each outline as chapter}<button
                class="chapter"
                class:active={chapter.page === page}
                on:click={() => go(chapter.page)}>{chapter.title}</button
              >{/each}{:else}<p class="bookmark-help">Сохраняйте текущую {isPdf ? "страницу" : "часть"} кнопкой «Добавить закладку». Нажмите на закладку, чтобы вернуться.</p>
            {#each orderedMarks as mark}<div class="bookmark-row" class:current={mark === locator}>
                <button class="bookmark-link" aria-label={"Открыть закладку: " + markTitle(mark)} disabled={markPage(mark) === null}
                  on:click={() => go(markPage(mark)!)}>
                  <strong>{markTitle(mark)}{#if mark === locator}<span class="current-mark">Сейчас</span>{/if}</strong>
                  <small>{markDescription(mark)}</small>
                </button>
                <button class="icon-button remove-bookmark" title="Удалить закладку" aria-label={"Удалить закладку: " + markTitle(mark)}
                  disabled={bookmarkBusy} on:click={() => updateMarks(marks.filter(m => m !== mark))}><Icon name="close" size={16} /></button>
              </div>{/each}
            {#if !marks.length}<p class="toc-empty">Здесь появятся сохранённые страницы. Позиция чтения запоминается автоматически, даже без закладки.</p>{/if}
          {/if}
        </div>
      </div>
      <div class="panel reading-progress">
        <div class="row spread">
          <span>{progress}%</span><span class="muted">{page} из {total}</span>
        </div>
        <div class="progress-track">
          <span style:width={progress + "%"}></span>
        </div>
      </div>
      <button class="text-button reader-lock" on:click={onsecurity}
        ><Icon name="lock" size={14} /> Защита архива</button
      >
    </aside>
    <div class="reader-content">
    <main class="reading-area" bind:this={readingArea}>
      {#if loading}<p class="muted" role="status">
          Открываем книгу…
        </p>{:else if error}<div class="reader-placeholder">
          <p class="error" role="alert">{error}</p>
          <button class="secondary" on:click={leave}>К библиотеке</button>
        </div>{:else}<article
          class="reading-page"
          class:night={pageTheme === "night"}
          class:white={pageTheme === "white"}
          class:pdf-page={isPdf && !preview}
          style:font-size={fontSize + "px"}
          style:line-height={lineHeight}
        >
          <div class="page-eyebrow">
            <span
              >{book.format} · {isPdf
                ? "СТРАНИЦА " + page
                : "ЧАСТЬ " + page}</span
            ><span>{book.title}</span>
          </div>
          {#if isPdf && document && !preview}<PdfPage
              data={document.pdf || ""}
              {page}
              {zoom}
              onloaded={pdfReady}
            />{:else}<h1>
              {preview
                ? page === 4
                  ? "Тишина, у которой есть орбита"
                  : chapters[page - 1]
                : pages[page - 1]?.title}
            </h1>
            <div class="page-lines" aria-hidden="true">
              <i></i><i></i><i></i>
            </div>
            <div class="prose">
              {#if preview}{#each readingText as paragraph}<p>
                    {paragraph}
                  </p>{/each}{:else}{#each pages[page - 1]?.blocks || [] as block}{#if block.kind === "image" && block.image}<img
                      class="book-image"
                      src={block.image}
                      alt="Иллюстрация"
                    />{:else if block.kind === "h2" || block.kind === "h1"}<h2>
                      {block.text}
                    </h2>{:else if block.kind === "quote"}<blockquote>
                      {block.text}
                    </blockquote>{:else if block.kind === "pre"}<pre>{block.text}</pre>{:else}<p
                    >
                      {block.text}
                    </p>{/if}{/each}{/if}
            </div>{/if}
        </article>{/if}{#if preview}<small class="reader-preview"
          >Демонстрационный текст · {book.format}</small
        >{/if}
    </main>
    <div class="reader-feedback" role="status" aria-live="polite">{bookmarkMessage}</div>
    {#if bookmarkError}<p class="error bookmark-error" role="alert">{bookmarkError}</p>{/if}
    <nav class="page-navigation panel" aria-label="Переход между страницами">
      <button class="page-turn" aria-label="Предыдущая страница" title="Предыдущая страница · ←" disabled={loading || Boolean(error) || page <= 1} on:click={() => navigate(-1)}><Icon name="back" /><span>Назад</span></button>
      <form class="page-jump" on:submit|preventDefault={() => go(pageInput)}>
        <label for="reader-page">{isPdf ? "Страница" : "Часть"}</label>
        <input id="reader-page" type="number" min="1" max={total} bind:value={pageInput} aria-label="Перейти к странице" disabled={loading || Boolean(error)} />
        <span>из {total}</span>
        <button class="jump-submit" type="submit" disabled={loading || Boolean(error)} title="Перейти к указанной странице">Перейти</button>
      </form>
      <button class="page-turn" aria-label="Следующая страница" title="Следующая страница · →" disabled={loading || Boolean(error) || page >= total} on:click={() => navigate(1)}><span>Далее</span><Icon name="arrow" /></button>
    </nav>
    </div>
  </div>
  {#if appearance}<div class="appearance panel">
      <div class="row spread">
        <strong>Страница</strong><button
          class="icon-button"
          on:click={() => (appearance = false)}
          aria-label="Закрыть настройки страницы"
          ><Icon name="close" size={14} /></button
        >
      </div>
      <div class="row theme-swatches" role="group" aria-label="Цвет страницы">
        {#each [{ id: "paper", name: "Бумага" }, { id: "white", name: "Светлая" }, { id: "night", name: "Ночная" }] as item}<button
            class:active={pageTheme === item.id}
            class={item.id}
            on:click={() => (pageTheme = item.id)}
            aria-pressed={pageTheme === item.id}>{item.name}</button
          >{/each}
      </div>
      <button
        class="text-button"
        on:click={() => {
          appearance = false;
          onsettings();
        }}>Настройки чтения <Icon name="arrow" size={14} /></button
      >
    </div>{/if}
</div>

<style>
  .toc-scroll {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .chapter {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .book-image {
    max-width: 100%;
    max-height: 600px;
    display: block;
    margin: 24px auto;
  }
  .prose h2 {
    font-family: Literata, serif;
    font-size: 1.2em;
    margin: 20px 0;
  }
  .prose blockquote {
    border-left: 2px solid #64726466;
    margin: 20px 0;
    padding-left: 20px;
  }
  .prose pre {
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    font-size: 0.8em;
  }
  .page-jump {
    display: flex;
    align-items: center;
    gap: 8px;
    font: 12px Onest, sans-serif;
    color: var(--muted);
  }
  .page-jump input {
    width: 55px;
    padding: 7px 8px;
    border-radius: 4px;
    border: 1px solid #64726433;
    color: inherit;
    background: none;
    font: inherit;
  }
  .pdf-page {
    padding: 24px;
    width: min(950px, 100%);
  }

  .reader-layout {
    height: 100%;
    display: flex;
    flex-direction: column;
    padding: 16px;
    gap: 10px;
  }
  .reader-header {
    min-height: 44px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: 20px;
  }
  .back-library {
    width: 230px;
    flex-shrink: 0;
    height: 40px;
    justify-content: flex-start;
    padding: 0 12px;
    font-size: 12px;
    background: var(--raised);
  }
  .reader-title {
    flex: 1;
    min-width: 0;
    text-align: center;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .reader-title strong {
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .reader-title small {
    font-size: 10px;
    color: var(--muted);
  }
  .reader-actions {
    display: flex;
    gap: 6px;
  }
  .reader-actions .icon-button {
    height: 40px;
    width: 40px;
    background: var(--raised);
  }
  .reader-actions .marked {
    color: var(--text);
    background: var(--active);
  }
  .reader-body {
    flex: 1;
    min-height: 0;
    display: flex;
    gap: 20px;
  }
  .reader-sidebar {
    min-height: 0;
    width: 230px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .contents {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 12px;
  }
  .toc-tabs {
    display: flex;
    gap: 10px;
  }
  .toc-tabs button {
    font-size: 9px;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    color: var(--quiet);
    padding: 0;
  }
  .toc-tabs button.active {
    color: var(--text);
  }
  .chapter {
    height: 35px;
    width: 100%;
    justify-content: flex-start;
    text-align: left;
    padding: 0 8px;
    border-radius: 18px;
    color: var(--muted);
    font-size: 12px;
    position: relative;
  }
  .chapter.active {
    background: var(--active);
    color: var(--text);
    font-weight: 500;
    padding-left: 20px;
  }
  .chapter.active::before {
    content: "";
    position: absolute;
    left: 8px;
    width: 2px;
    height: 20px;
    background: var(--accent);
    border-radius: 2px;
  }
  .toc-empty {
    color: var(--muted);
    font-size: 12px;
    padding: 12px 6px;
    line-height: 1.7;
  }
  .reading-progress {
    margin-top: auto;
    padding: 12px;
    font-size: 10px;
  }
  .reading-progress .progress-track {
    margin-top: 10px;
  }
  .reader-lock {
    font-size: 10px;
    margin: 0 auto;
  }
  .reading-area {
    flex: 1;
    min-width: 0;
    min-height: 0;
    overflow: auto;
    overscroll-behavior: contain;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 12px 16px 10px;
  }
  .reading-page {
    background: var(--paper);
    color: var(--ink);
    border-radius: 3px;
    width: min(870px, 100%);
    min-height: 100%;
    flex: none;
    padding: 46px 84px 28px;
    display: flex;
    flex-direction: column;
    font-family: Literata, Georgia, serif;
    box-shadow: 0 6px 28px #0003;
    user-select: text;
  }
  .reading-page > :global(*) { flex-shrink: 0; }
  .page-eyebrow {
    font-size: 9px;
    display: flex;
    justify-content: space-between;
    gap: 20px;
    letter-spacing: 1.6px;
    text-transform: uppercase;
    color: #68736a;
    line-height: 1.5;
  }
  .page-eyebrow > span:last-child {
    max-width: 50%;
    text-align: right;
  }
  .reading-page h1 {
    font-family: Literata, Georgia, serif;
    line-height: 1.35;
    font-size: 1.5em;
    letter-spacing: -0.5px;
    margin: 20px 0;
  }
  .page-lines {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0 0 26px;
  }
  .page-lines i {
    height: 2px;
    width: 120px;
    background: #64726433;
  }
  .page-lines i:nth-child(2) {
    width: 76px;
    opacity: 0.65;
  }
  .page-lines i:nth-child(3) {
    width: 100px;
    opacity: 0.35;
  }
  .prose p {
    margin-bottom: 18px;
  }
  .reader-content { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; gap: 8px; }
  .page-navigation { flex-shrink: 0; display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 12px; background: var(--raised); }
  .page-turn { min-height: 42px; padding: 0 16px; border: 1px solid var(--border); border-radius: 12px; font: 500 13px Onest, sans-serif; background: var(--panel); }
  .page-turn:hover:not(:disabled) { background: var(--active); }
  .jump-submit { min-height: 36px; padding: 0 10px; border-radius: 8px; font-size: 12px; color: var(--text); background: var(--active); }
  .reader-feedback { min-height: 0; color: var(--muted); font: 12px Onest, sans-serif; }
  .reader-feedback:empty { display: none; }
  .bookmark-error { margin: 0; font: 12px Onest, sans-serif; }
  .bookmark-toggle, .bookmarks-toggle { height: 40px; flex-shrink: 0; padding: 0 12px; border-radius: 12px; background: var(--raised); font-size: 12px; }
  .bookmark-count { min-width: 18px; padding: 1px 4px; border-radius: 5px; background: var(--active); font-size: 11px; }
  .bookmark-help { font-size: 11px; line-height: 1.6; color: var(--muted); margin-bottom: 14px; }
  .bookmark-row { display: flex; align-items: center; border-radius: 10px; margin-bottom: 6px; }
  .bookmark-row.current { background: var(--active); }
  .bookmark-link { flex: 1; min-width: 0; padding: 10px; display: flex; flex-direction: column; align-items: flex-start; gap: 4px; text-align: left; border-radius: 10px; }
  .bookmark-link strong { display: flex; align-items: center; gap: 8px; font-size: 12px; font-weight: 500; }
  .bookmark-link small { max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-size: 11px; color: var(--muted); }
  .current-mark { font-size: 9px; color: var(--muted); }
  .remove-bookmark { width: 30px; height: 30px; margin-right: 4px; flex-shrink: 0; }
  .night {
    background: #262d2b;
    color: #d9dfd6;
  }
  .night .page-eyebrow {
    color: #a6b5a4;
  }
  .white {
    background: #f9faf5;
  }
  .reader-placeholder {
    margin: auto;
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 20px;
  }

  .reader-placeholder p {
    font-size: 16px;
  }
  .reader-placeholder button {
    font: 13px Onest;
    background: #d6dad0;
    color: #323b31;
    border-color: #64726433;
  }
  .appearance {
    position: absolute;
    right: 16px;
    top: 72px;
    width: 290px;
    padding: 18px;
    box-shadow: 0 12px 40px #0005;
    z-index: 20;
  }
  .appearance strong {
    font-weight: 500;
  }
  .appearance .icon-button {
    width: 28px;
    height: 28px;
  }
  .theme-swatches {
    margin: 18px 0;
  }
  .theme-swatches button {
    padding: 12px 8px;
    border-radius: 10px;
    font-size: 11px;
    border: 2px solid transparent;
  }
  .theme-swatches .paper {
    background: #e7e8e0;
    color: #323535;
  }
  .theme-swatches .white {
    background: #f9faf5;
    color: #323535;
  }
  .theme-swatches .night {
    background: #262d2b;
    color: #d9dfd6;
  }
  .theme-swatches .active {
    border-color: var(--accent);
  }
  .appearance > .text-button {
    font-size: 12px;
  }
  .reader-preview {
    padding-top: 10px;
    color: var(--quiet);
    font: 9px Onest;
  }
  .mobile-toc, .sidebar-close {
    display: none !important;
  }
  @media (max-width: 1100px) {
    .reading-page {
      padding: 40px 48px 28px;
    }
    .reader-sidebar,
    .back-library {
      width: 200px;
    }
    .reader-title {
      text-align: left;
    }
    .reading-area {
      padding: 12px 0 10px;
    }
  }
  @media (max-width: 780px) {
    .reader-layout {
      padding: 12px;
    }
    .reader-sidebar {
      display: none;
      position: absolute;
      top: 70px;
      bottom: 86px;
      overflow: auto;
      left: 12px;
      z-index: 10;
      background: var(--bg);
      width: 230px;
      padding: 4px;
      box-shadow: 12px 0 25px #0005;
    }
    .reader-sidebar.open {
      display: flex;
    }
    .reader-body {
      gap: 0;
    }
    .back-library {
      width: 40px;
      padding: 0;
      justify-content: center;
    }
    .back-library span {
      display: none;
    }
    .mobile-toc, .sidebar-close {
      display: inline-flex !important;
    }
    .sidebar-close { margin-left: auto; width: 24px; height: 24px; }
    .reader-actions .icon-button {
      width: 32px;
      height: 32px;
    }
    .reader-actions {
      gap: 4px;
    }
    .reader-header {
      gap: 10px;
    }
    .reading-page {
      padding: 30px 26px 22px;
      min-height: 100%;
    }
    .page-eyebrow {
      font-size: 7px;
      letter-spacing: 1px;
    }
    .page-navigation { gap: 8px; padding: 8px; }
    .page-turn { padding: 0 10px; font-size: 12px; }
    .page-jump { gap: 5px; font-size: 11px; }
    .page-jump input { width: 48px; }
    .bookmark-toggle, .bookmarks-toggle { height: 32px; padding: 0 8px; }
    .bookmarks-toggle > span:not(.bookmark-count) { display: none; }
    .bookmark-toggle { font-size: 11px; }
    .reader-header { gap: 8px; }
    .reader-title small {
      font-size: 9px;
    }
  }
  @media (max-width: 520px) {
    .reader-title small { display: none; }
    .bookmark-toggle > span { display: none; }
    .page-jump label { display: none; }
    .jump-submit { padding: 0 6px; font-size: 11px; }
    .page-turn { padding: 0 8px; gap: 4px; }
  }
  @media (max-width: 420px) {
    .reader-title strong {
      font-size: 10px;
    }
    .reader-actions .icon-button {
      width: 28px;
      height: 28px;
    }
    .reading-page {
      padding: 28px 20px 20px;
    }
    .reading-page h1 {
      font-size: 1.3em;
    }
  }
</style>
