<script lang="ts">
  import Brand from "./Brand.svelte";
  import Icon from "./Icon.svelte";
  import ChoiceMenu from "./ChoiceMenu.svelte";
  import BookCollection from "./BookCollection.svelte";
  import { formatSize, type Book } from "./ui";
  export let books: Book[] = [];
  export let shelves: string[] = [];
  export let density = "comfortable";
  export let view: "grid" | "list" = "grid";
  export let onview: (view: "grid" | "list") => void;
  export let onlock: () => void;
  export let onimport: () => void;
  export let onshelf: () => void;
  export let onsettings: () => void;
  export let onbackups: () => void;
  export let onsecurity: () => void;
  export let onopen: (book: Book) => void;
  export let ondetails: (book: Book) => void;
  export let onmanageshelf: (name: string) => void;
  let active = "all";
  let shelf = "";
  let query = "";
  let sort = "recent";
  let sidebarOpen = false;
  let searchInput: HTMLInputElement;
  const tabs = [
    { id: "all", label: "Все книги", icon: "library" },
    { id: "reading", label: "Читаю", icon: "book" },
    { id: "unread", label: "Не прочитано", icon: "bookmark" },
    { id: "finished", label: "Прочитано", icon: "check" },
  ];
  $: reading = books.filter((b) => b.progress > 0 && b.progress < 100).length;
  $: unread = books.filter((b) => b.progress === 0).length;
  $: finished = books.filter((b) => b.progress >= 100).length;
  $: totalSize = books.reduce((sum, b) => sum + b.size, 0);
  $: filtered = books
    .filter(
      (b) =>
        (!query ||
          [b.title, b.author].some((v) =>
            v.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
          )) &&
        (active === "all" ||
          (active === "reading" && b.progress > 0 && b.progress < 100) ||
          (active === "unread" && b.progress === 0) ||
          (active === "finished" && b.progress >= 100) ||
          (active === "unshelved" && !b.shelves.length) ||
          (active === "shelf" && b.shelves.includes(shelf))),
    )
    .sort((a, b) =>
      sort === "title"
        ? a.title.localeCompare(b.title, "ru")
        : sort === "author"
          ? a.author.localeCompare(b.author, "ru")
          : b.updatedAt.localeCompare(a.updatedAt),
    );
  $: title =
    active === "shelf"
      ? shelf
      : active === "unshelved"
        ? "Без полки"
        : tabs.find((t) => t.id === active)?.label || "Все книги";
  function select(id: string, name = "") {
    active = id;
    shelf = name;
    sidebarOpen = false;
  }
  function shortcut(event: KeyboardEvent) {
    if (document.querySelector("dialog[open]")) return;
    if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      searchInput?.focus();
    }
    if (event.key === "Escape") {
      query = "";
      sidebarOpen = false;
    }
  }
</script>

<svelte:window on:keydown={shortcut} />
<div class="library-layout">
  <header class="archive-header">
    <button
      class="brand-menu panel"
      on:click={onsettings}
      aria-label="Открыть настройки mut"
      ><Brand /><Icon name="chevron" size={14} /></button
    ><button
      class="mobile-menu icon-button"
      on:click={() => (sidebarOpen = !sidebarOpen)}
      aria-label="Полки и навигация"
      aria-expanded={sidebarOpen}><Icon name="menu" /></button
    ><label class="search-box"
      ><Icon name="search" /><input
        bind:this={searchInput}
        bind:value={query}
        placeholder="Поиск в архиве"
        aria-label="Поиск по названию и автору"
      /><kbd>Ctrl K</kbd>{#if query}<button
          class="clear-search"
          on:click={() => (query = "")}
          aria-label="Очистить поиск"><Icon name="close" size={14} /></button
        >{/if}</label
    ><button class="primary add-button" on:click={onimport}
      ><Icon name="plus" size={17} /><span>Добавить</span></button
    >
  </header>
  <div class="library-body">
    <aside
      class="sidebar"
      class:open={sidebarOpen}
      aria-label="Библиотека и полки"
    >
      <nav class="panel nav-panel" aria-label="Мой архив">
        <span class="eyebrow">Мой архив</span>
        <div class="hairlines" aria-hidden="true"><i></i><i></i><i></i></div>
        {#each tabs as tab}<button
            class="nav-item"
            class:active={active === tab.id}
            aria-current={active === tab.id ? "page" : undefined}
            on:click={() => select(tab.id)}
            ><Icon name={tab.icon} size={16} /><span>{tab.label}</span><small
              >{tab.id === "all"
                ? books.length
                : tab.id === "reading"
                  ? reading
                  : tab.id === "unread"
                    ? unread
                    : finished}</small
            ></button
          >{/each}
      </nav>
      <nav class="panel nav-panel shelves" aria-label="Полки">
        <div class="row spread">
          <span class="eyebrow">Полки</span><button
            class="shelf-add"
            on:click={onshelf}
            aria-label="Создать полку"><Icon name="plus" size={14} /></button
          >
        </div>
        <div class="hairlines" aria-hidden="true"><i></i><i></i><i></i></div>
        {#each shelves as name, i}<button
            class="nav-item shelf-item"
            class:active={active === "shelf" && shelf === name}
            on:click={() => select("shelf", name)}
            ><i style:opacity={0.35 + (i % 3) * 0.25}></i><span>{name}</span
            ><small
              >{books.filter((b) => b.shelves.includes(name)).length}</small
            ></button
          >{/each}{#if !shelves.length}<p class="no-shelves">
            Здесь будут ваши полки
          </p>{/if}<button
          class="nav-item shelf-item unshelved"
          class:active={active === "unshelved"}
          on:click={() => select("unshelved")}
          ><i></i><span>Без полки</span><small
            >{books.filter((b) => !b.shelves.length).length}</small
          ></button
        >
      </nav>
      <div class="sidebar-bottom">
        <div class="panel storage">
          <div class="row spread">
            <span>На устройстве</span><span class="size"
              >{formatSize(totalSize)}</span
            >
          </div>
          <div class="storage-divider" aria-hidden="true"></div>
          <small>Размер файлов в библиотеке</small>
        </div>
        <div class="panel utility">
          <button on:click={onbackups}
            ><Icon name="disk" size={16} /> Резервные копии</button
          ><button on:click={onsettings}
            ><Icon name="settings" size={16} /> Настройки</button
          ><button on:click={onlock}
            ><Icon name="lock" size={16} /> Заблокировать</button
          >
        </div>
      </div>
    </aside>
    <main class="workspace">
      <div class="workspace-title">
        <div>
          <div class="row">
            <h1>{query ? "Результаты поиска" : title}</h1>
            {#if active === "shelf"}<button
                class="icon-button"
                aria-label="Изменить полку"
                on:click={() => onmanageshelf(shelf)}
                ><Icon name="more" size={16} /></button
              >{/if}
          </div>
          <div class="subtitle">
            <span
              >{filtered.length}
              {filtered.length === 1 ? "книга" : "книг"}</span
            ><span>·</span><ChoiceMenu id="book-sort" label="Сортировка книг" compact value={sort} options={[{value:"recent",label:"По последнему чтению"},{value:"title",label:"По названию"},{value:"author",label:"По автору"}]} onchange={value => (sort = String(value))} />
          </div>
        </div>
        <div class="view-switch" role="group" aria-label="Вид библиотеки">
          <button
            class="icon-button"
            class:chosen={view === "grid"}
            aria-pressed={view === "grid"}
            aria-label="Сетка"
            on:click={() => onview("grid")}
            ><Icon name="grid" size={16} /></button
          ><button
            class="icon-button"
            class:chosen={view === "list"}
            aria-pressed={view === "list"}
            aria-label="Список"
            on:click={() => onview("list")}
            ><Icon name="list" size={16} /></button
          >
        </div>
      </div>
      <div class="hairlines workspace-lines" aria-hidden="true">
        <i></i><i></i><i></i>
      </div>
      {#if !books.length}<div class="empty-state">
          <span class="eyebrow">Библиотека создана</span>
          <h1>Ваша коллекция начинается здесь</h1>
          <p class="muted">Добавьте первую книгу. Для каждой найдётся место.</p>
          <button class="dropzone" on:click={onimport}
            ><span class="drop-plus"><Icon name="plus" size={28} /></span
            ><strong>Добавить книги</strong><span class="muted"
              >PDF, EPUB, FB2 — для чтения. Любой файл — для хранения.</span
            ></button
          >
          <p class="empty-footnote">
            <Icon name="shield" size={15} /> Файлы хранятся локально. Оригиналы остаются
            у вас.
          </p>
        </div>
      {:else if !filtered.length}<div class="empty-state">
          <Icon name={query ? "search" : "book"} size={32} />
          <h2>{query ? "Ничего не найдено" : "Пока нет книг"}</h2>
          <p class="muted">
            {query
              ? "Попробуйте другое название или имя автора."
              : "Книги появятся здесь, когда вы начнёте читать или добавите их на полку."}
          </p>
          <button
            class="secondary"
            on:click={() => {
              query = "";
              select("all");
            }}>Все книги <Icon name="arrow" size={16} /></button
          >
        </div>
      {:else}<BookCollection
          books={filtered}
          {view}
          {density}
          {onopen}
          {ondetails}
        />{/if}
    </main>
  </div>
  <button
    class="vault-button panel"
    on:click={onsecurity}
    aria-label="Состояние защиты архива"><Icon name="lock" size={24} /></button
  >
</div>

<style>
  .library-layout {
    height: 100%;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
    background: var(--bg);
  }
  .archive-header {
    display: flex;
    align-items: center;
    gap: 10px;
    min-height: 48px;
  }
  .brand-menu {
    width: 226px;
    height: 44px;
    justify-content: space-between;
    padding: 0 12px 0 10px;
    flex-shrink: 0;
    background: var(--raised);
  }
  .brand-menu :global(.brand) {
    font-size: 13px;
  }
  .brand-menu :global(.brand img) {
    width: 38px;
    height: 38px;
  }
  .search-box {
    flex: 1;
    min-width: 0;
    height: 44px;
    background: var(--panel);
    border: 1px solid var(--border);
    border-radius: 10px;
    display: flex;
    gap: 10px;
    align-items: center;
    padding: 0 12px;
    color: var(--muted);
  }
  .search-box:focus-within {
    outline: 2px solid var(--accent);
    outline-offset: 2px;
  }
  .search-box input {
    flex: 1;
    border: none;
    padding: 0;
    background: none;
    border-radius: 0;
    outline: none;
  }
  .search-box kbd {
    font: 10px Onest;
    color: var(--quiet);
    white-space: nowrap;
    border: 1px solid var(--line);
    padding: 2px 5px;
    border-radius: 4px;
  }
  .clear-search {
    padding: 4px;
  }
  .add-button {
    height: 40px;
    min-height: 40px;
    padding: 0 14px;
    border-radius: 18px;
    font-size: 12px;
  }
  .library-body {
    display: flex;
    gap: 26px;
    flex: 1;
    min-height: 0;
  }
  .sidebar {
    width: 226px;
    flex-shrink: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
    overflow: auto;
  }
  .nav-panel {
    padding: 12px;
  }
  .nav-item {
    width: 100%;
    height: 42px;
    justify-content: flex-start;
    gap: 11px;
    padding: 0 8px;
    border-radius: 18px;
    color: var(--muted);
    font-size: 13px;
    min-width: 0;
  }
  .nav-item > span {
    flex: 1;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .nav-item small {
    font-size: 11px;
  }
  .nav-item.active {
    background: var(--active);
    color: var(--text);
  }
  .shelves .hairlines {
    margin-top: 4px;
  }
  .shelf-add {
    color: var(--muted);
    width: 22px;
    height: 22px;
  }
  .shelf-item {
    height: 38px;
    font-size: 12px;
  }
  .shelf-item i {
    width: 3px;
    height: 16px;
    background: var(--accent);
    border-radius: 2px;
  }
  .unshelved {
    margin-top: 7px;
    border-top: 1px solid var(--line);
    border-radius: 0 0 12px 12px;
  }
  .no-shelves {
    font-size: 11px;
    color: var(--quiet);
    padding: 12px 6px;
  }
  .sidebar-bottom {
    margin-top: auto;
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-top: 20px;
  }
  .storage {
    padding: 12px;
    font-size: 11px;
    color: var(--muted);
  }
  .storage-divider {
    margin-top: 10px;
    height: 2px;
    border-radius: 2px;
    background: var(--line);
  }
  .storage small {
    font-size: 9px;
    display: block;
    margin-top: 8px;
    color: var(--quiet);
  }
  .storage .size {
    color: var(--text);
  }
  .utility {
    padding: 8px;
  }
  .utility button {
    width: 100%;
    justify-content: flex-start;
    padding: 9px 8px;
    color: var(--muted);
    font-size: 11px;
    border-radius: 10px;
  }
  .workspace {
    min-width: 0;
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 12px 16px 48px 0;
  }
  .workspace-title {
    display: flex;
    justify-content: space-between;
    gap: 16px;
    align-items: center;
  }
  .workspace-title h1 {
    font-size: 29px;
  }
  .subtitle {
    display: flex;
    align-items: center;
    gap: 5px;
    margin-top: 6px;
    color: var(--muted);
    font-size: 11px;
  }
  .view-switch {
    display: flex;
    gap: 6px;
    align-self: flex-end;
  }
  .view-switch .icon-button {
    width: 36px;
    height: 36px;
    background: transparent;
  }
  .view-switch .chosen {
    background: var(--raised);
  }
  .workspace-lines {
    margin: 15px 0 12px;
    gap: 5px;
  }
  .workspace-lines i {
    height: 3px;
  }
  .empty-state {
    flex: 1;
    min-height: 300px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    gap: 14px;
    padding: 32px 10px;
  }
  .empty-state > h1 {
    font-size: 28px;
    margin-top: 2px;
  }
  .empty-state > p {
    font-size: 13px;
    max-width: 450px;
  }
  .dropzone {
    margin-top: 18px;
    width: min(640px, 100%);
    min-height: 210px;
    border: 1px dashed var(--quiet);
    border-radius: 22px;
    display: flex;
    flex-direction: column;
    gap: 14px;
    background: var(--panel);
    padding: 24px;
  }
  .dropzone strong {
    font-size: 16px;
    font-weight: 500;
  }
  .dropzone > span:last-child {
    font-size: 12px;
  }
  .drop-plus {
    display: grid;
    place-items: center;
    width: 50px;
    height: 50px;
    background: var(--raised);
    border-radius: 15px;
    color: var(--accent);
  }
  .empty-footnote {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 10px !important;
    color: var(--muted);
    margin-top: 10px;
  }
  .vault-button {
    position: absolute;
    right: 24px;
    bottom: 24px;
    width: 58px;
    height: 58px;
    background: var(--panel);
    box-shadow: 0 8px 20px #0004;
    color: var(--accent);
  }
  .mobile-menu {
    display: none;
  }
  @media (max-width: 1100px) {
    .brand-menu,
    .sidebar {
      width: 200px;
    }
    .library-body {
      gap: 22px;
    }
    .workspace {
      padding-right: 0;
    }
    .workspace-title h1 {
      font-size: 27px;
    }
  }
  @media (max-width: 720px) {
    .library-layout {
      padding: 12px;
    }
    .brand-menu {
      width: 56px;
      padding: 0 8px;
    }
    .brand-menu :global(.brand > div),
    .brand-menu > :global(svg) {
      display: none;
    }
    .mobile-menu {
      display: inline-flex;
    }
    .sidebar {
      display: none;
      position: absolute;
      top: 74px;
      bottom: 16px;
      left: 12px;
      z-index: 10;
      width: 226px;
      background: var(--bg);
      padding: 4px;
      box-shadow: 12px 0 25px #0005;
      border-radius: 18px;
    }
    .sidebar.open {
      display: flex;
    }
    .library-body {
      gap: 0;
    }
    .workspace {
      padding: 12px 0 48px;
    }
    .search-box kbd {
      display: none;
    }
    .add-button span {
      display: none;
    }
    .workspace-title h1 {
      font-size: 26px;
    }
    .empty-state > h1 {
      font-size: 24px;
    }
    .vault-button {
      right: 16px;
      bottom: 16px;
      width: 44px;
      height: 44px;
    }
    .search-box {
      padding: 0 8px;
      gap: 6px;
    }
    .subtitle {
      font-size: 10px;
    }
  }
</style>
