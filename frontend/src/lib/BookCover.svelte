<script lang="ts">
  import { tones, type Book } from "./ui";
  import { onMount } from "svelte";
  import { GetCover } from "../wailsjs/go/app/App";
  export let book: Book;
  export let index = 0;
  export let compact = false;
  let cover = "";
  onMount(() => {
    let alive = true;
    if (book.coverId)
      void GetCover(book.id)
        .then((value) => {
          if (alive) cover = value;
        })
        .catch(() => {});
    return () => {
      alive = false;
      cover = "";
    };
  });
  $: tone = tones[book.tone % tones.length];
  $: darkText = [1, 3, 4, 6, 9, 10].includes(book.tone % 12);
</script>

<div
  class="book-cover"
  class:compact
  style:background={tone}
  style:color={darkText ? "#262e2c" : "#eef0ee"}
  aria-hidden="true"
>
  {#if cover}<img class="cover-image" src={cover} alt="" />{:else}
    <div class="cover-meta">
      <span>K—{String(index + 1).padStart(2, "0")}</span><span
        >{book.format}</span
      >
    </div>
    <div class="cover-title">{book.title}</div>
    <span class="cover-author">{book.author}</span>{/if}
</div>

<style>
  .cover-image {
    width: 100%;
    height: 100%;
    object-fit: cover;
    border-radius: 12px;
  }
  .book-cover {
    width: 100%;
    height: 238px;
    border-radius: 18px;
    border-left: 4px solid #ffffff22;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    text-align: left;
    padding: 12px 10px;
    overflow: hidden;
  }
  .cover-meta {
    display: flex;
    justify-content: space-between;
    gap: 6px;
    font-size: 8px;
  }
  .cover-meta > span:first-child {
    letter-spacing: 1.5px;
  }
  .cover-title {
    font-size: 21px;
    line-height: 1.08;
    font-weight: 500;
    overflow-wrap: anywhere;
    display: -webkit-box;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 5;
    line-clamp: 5;
    overflow: hidden;
  }
  .cover-author {
    font-size: 8px;
    letter-spacing: 1.2px;
    text-transform: uppercase;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .compact {
    height: 44px;
    width: 34px;
    flex-shrink: 0;
    border-radius: 5px;
    padding: 3px;
    border-left-width: 2px;
  }
  .compact .cover-meta,
  .compact .cover-author {
    display: none;
  }
  .compact .cover-title {
    font-size: 5px;
    margin: auto 0;
  }
</style>
