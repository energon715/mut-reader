<script lang="ts">
  export let name = "book";
  export let size = 18;
  const paths: Record<string, string> = {
    search: "m21 21-4.5-4.5 M19 10.5a8.5 8.5 0 1 1-17 0 8.5 8.5 0 0 1 17 0",
    plus: "M12 5v14 M5 12h14",
    minus: "M5 12h14",
    close: "m6 6 12 12 M6 18 18 6",
    arrow: "M5 12h14 m-6-6 6 6-6 6",
    back: "M19 12H5 m6-6-6 6 6 6",
    chevron: "m9 5 7 7-7 7",
    down: "m6 9 6 6 6-6",
    book: "M12 5v16 M12 5C8 2 4 3 2 4v15c4-2 7-1 10 2 3-3 6-4 10-2V4c-2-1-6-2-10 1",
    library: "M4 4v16 M9 6v14 M14 4v16 m4-16 4 16",
    bookmark: "M6 3h12v18l-6-4-6 4V3",
    check: "m5 12 4 4L19 6",
    lock: "M7 10V7a5 5 0 0 1 10 0v3 M5 10h14v11H5z M12 14v3",
    grid: "M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z",
    list: "M8 5h13 M8 12h13 M8 19h13 M3 5h.01 M3 12h.01 M3 19h.01",
    settings:
      "M12 8a4 4 0 1 1 0 8 4 4 0 0 1 0-8 M12 2v2 M12 20v2 M2 12h2 M20 12h2 M5 5l1.5 1.5 M17.5 17.5 1.5 1.5 M5 19l1.5-1.5 M17.5 6.5 1.5-1.5",
    disk: "M4 4h13l3 3v13H4z M8 4v6h8V4 M8 20v-7h8v7",
    shield: "M12 2 3 6v6c0 5 9 10 9 10s9-5 9-10V6l-9-4 m-4 10 3 3 5-6",
    folder: "M3 6h7l2 3h9v11H3V6",
    upload: "M12 16V3 m-5 5 5-5 5 5 M4 16v5h16v-5",
    more: "M5 12h.01 M12 12h.01 M19 12h.01",
    eye: "M2 12s4-7 10-7 10 7 10 7-4 7-10 7-10-7-10-7 M12 9a3 3 0 1 1 0 6 3 3 0 0 1 0-6",
    info: "M12 11v6 M12 7h.01 M22 12a10 10 0 1 1-20 0 10 10 0 0 1 20 0",
    copy: "M8 8h13v13H8z M16 8V3H3v13h5",
    restore: "M3 11a9 9 0 1 1 2 7 M3 4v7h7 M12 7v5l3 2",
    file: "M5 2h9l5 5v15H5z M14 2v6h5",
    menu: "M4 6h16 M4 12h16 M4 18h16",
  };
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="1.5"
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"><path d={paths[name] || paths.book} /></svg
>
