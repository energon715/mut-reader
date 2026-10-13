<script lang="ts">
  import { onMount, tick } from "svelte";
  import Icon from "./Icon.svelte";
  import BookCover from "./BookCover.svelte";
  import { formatSize, type Book } from "./ui";
  export let books: Book[] = [];
  export let view: "grid" | "list" = "grid";
  export let density = "comfortable";
  export let onopen: (book: Book) => void;
  export let ondetails: (book: Book) => void;
  let viewport: HTMLDivElement;
  let width = 1100;
  let height = 650;
  let scrollTop = 0;
  $: columns =
    view === "list"
      ? 1
      : Math.max(
          1,
          Math.floor((width + 8) / (density === "compact" ? 152 : 182)),
        );
  $: rowHeight = view === "list" ? 76 : 302;
  $: rows = Math.ceil(books.length / columns);
  $: start = Math.max(0, Math.floor(scrollTop / rowHeight) - 2);
  $: end = Math.min(rows, Math.ceil((scrollTop + height) / rowHeight) + 2);
  $: visible = books.slice(start * columns, end * columns);
  $: if (books && view && density) resetScroll();
  async function resetScroll() {
    await tick();
    if (viewport) {
      viewport.scrollTop = 0;
      scrollTop = 0;
    }
  }
  onMount(() => {
    const observer = new ResizeObserver(([entry]) => {
      width = entry!.contentRect.width;
      height = entry!.contentRect.height;
    });
    observer.observe(viewport);
    return () => observer.disconnect();
  });
</script>

<div
  class="viewport"
  bind:this={viewport}
  on:scroll={() => (scrollTop = viewport.scrollTop)}
>
  <div style:height={start * rowHeight + "px"} aria-hidden="true"></div>
  <div
    class="collection"
    class:list={view === "list"}
    style:grid-template-columns={"repeat(" + columns + ",minmax(0,1fr))"}
  >
    {#each visible as book, i (book.id)}
      <div class="book-entry" class:list-entry={view === "list"}>
        <button
          class="book-item"
          class:list-item={view === "list"}
          on:click={() => onopen(book)}
          aria-label={"Открыть " + book.title + " · " + book.author}
        >
          <BookCover
            {book}
            index={start * columns + i}
            compact={view === "list"}
          />
          <div class="book-info">
            <strong title={book.title}>{book.title}</strong>
            <div class="book-byline">
              <span title={book.author}>{book.author}</span
              >{#if book.progress > 0}<span class="size"
                  >{Math.round(book.progress)}%</span
                >{/if}
            </div>
          </div>
          {#if view === "list"}<span class="list-format">{book.format}</span
            ><span class="list-size size">{formatSize(book.size)}</span>{/if}
        </button><button
          class="book-details icon-button"
          aria-label={"Карточка книги: " + book.title}
          on:click={() => ondetails(book)}
          ><Icon name="more" size={16} /></button
        >
      </div>
    {/each}
  </div>
  <div
    style:height={Math.max(0, rows - end) * rowHeight + "px"}
    aria-hidden="true"
  ></div>
</div>

<style>
  .book-entry {
    position: relative;
    min-width: 0;
  }
  .book-entry .book-item {
    width: 100%;
  }
  .book-details {
    position: absolute;
    right: 10px;
    bottom: 74px;
    width: 28px;
    height: 28px;
    opacity: 0;
    background: var(--panel);
  }
  .book-entry:hover .book-details,
  .book-details:focus-visible {
    opacity: 1;
  }
  .list-entry .book-details {
    bottom: 24px;
    right: 2px;
    opacity: 1;
  }
  .list-entry .book-item {
    padding-right: 40px;
  }

  .viewport {
    flex: 1;
    min-height: 0;
    overflow: auto;
    padding: 4px 6px 8px 4px;
    margin-left: -4px;
  }
  .collection {
    display: grid;
    gap: 0 8px;
  }
  .book-item {
    height: 302px;
    min-width: 0;
    display: flex;
    flex-direction: column;
    align-items: stretch;
    justify-content: flex-start;
    text-align: left;
    padding: 0 0 12px;
    border-radius: 18px;
  }
  .book-item:hover {
    background: none !important;
  }
  .book-item:hover :global(.book-cover) {
    box-shadow:
      0 0 0 1px var(--accent),
      0 6px 18px #0002;
    transform: translateY(-2px);
    transition: transform 0.15s;
  }
  .book-info {
    padding: 7px 2px 0;
    min-width: 0;
  }
  .book-info strong {
    display: block;
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .book-byline {
    display: flex;
    justify-content: space-between;
    gap: 10px;
    margin-top: 5px;
    font-size: 10px;
    color: var(--muted);
  }
  .book-byline > span:first-child {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .book-byline > span:last-child {
    flex-shrink: 0;
  }
  .list-item {
    height: 76px;
    flex-direction: row;
    align-items: center;
    padding: 12px 16px;
    gap: 16px;
    border-bottom: 1px solid var(--line);
    border-radius: 0;
  }
  .list-item:hover {
    background: var(--panel) !important;
  }
  .list-item .book-info {
    flex: 1;
    padding: 0;
  }
  .list-item .book-byline {
    max-width: 240px;
  }
  .list-format,
  .list-size {
    font-size: 11px;
    color: var(--muted);
    width: 72px;
    text-align: right;
  }
  .list-format {
    letter-spacing: 0.6px;
  }
  @media (max-width: 650px) {
    .list-size {
      display: none;
    }
    .list-item {
      padding: 12px 6px;
      gap: 10px;
    }
    .list-format {
      width: 38px;
    }
  }
</style>
