<script lang="ts">
  import { onMount } from "svelte";
  import LoginScreen from "./lib/LoginScreen.svelte";
  import CreateLibrary from "./lib/CreateLibrary.svelte";
  import LibraryView from "./lib/LibraryView.svelte";
  import ReaderView from "./lib/ReaderView.svelte";
  import BackupView from "./lib/BackupView.svelte";
  import Dialog from "./lib/Dialog.svelte";
  import SettingsView from "./lib/SettingsView.svelte";
  import ImportView from "./lib/ImportView.svelte";
  import BookDetails from "./lib/BookDetails.svelte";
  import { EventsOn } from "./wailsjs/runtime/runtime";
  import Icon from "./lib/Icon.svelte";
  import {
    IsVaultConfigurated,
    IsUnlocked,
    CreateVault,
    Unlock,
    UnlockWithRecovery,
    Lock,
    ListBooks,
    ListShelves,
    ImportBook,
    ListImportFiles,
    ChangePassword,
    ImportEdition,
    CreateShelf,
    RenameShelf,
    DeleteShelf,
    SetBookShelves,
    DeleteBook,
    ChooseExportPath,
    ExportOriginal,
    SetAutoLockMinutes,
    Touch,
  } from "./wailsjs/go/app/App";
  import {
    fromCatalog,
    formatSize,
    canRead,
    type Book,
    type Screen,
    type Overlay,
  } from "./lib/ui";
  import { demoBooks, demoShelves, demoRecovery } from "./lib/demo";
  import {
    initialDefaults,
    loadSavedPaths,
    getNativeDefaults,
    hasDirectoryPicker,
    hasFilePicker,
    chooseDirectory,
    chooseFiles,
    type DirectoryPaths,
  } from "./lib/paths";
  const runtime = Boolean(
    (window as unknown as { go?: { app?: { App?: unknown } } }).go?.app?.App,
  );
  const requestedPreview = new URLSearchParams(window.location.search).get(
    "preview",
  );
  // Preview data never reaches the real vault, including inside Wails.
  const preview = !runtime || Boolean(requestedPreview);
  let screen: Screen = preview ? "library" : "login";
  let overlay: Overlay = null;
  let books: Book[] = preview ? structuredClone(demoBooks) : [];
  let shelves: string[] = preview ? [...demoShelves] : [];
  let selectedBook: Book | null = null;
  let recoveryKey = "";
  let defaults = { ...initialDefaults };
  const pathStorageKey = preview ? "mut.preview.paths" : "mut.paths";
  let paths = loadSavedPaths(defaults, pathStorageKey);
  let path = paths.library;
  const directoryPickerAvailable = !preview && hasDirectoryPicker();
  const filePickerAvailable = !preview && hasFilePicker();
  let flowReturn: Screen = "library";
  let busy = false;
  let errorMessage = "";
  let ready = preview;
  let previewName = requestedPreview || "library";
  let importInitial = "choose";
  let backupInitial = "choose";
  let shelfName = "";
  let editingShelf = "";
  let confirmShelfDelete = false;
  let autoLockMinutes = 20;
  let lastTouch = 0;
  let theme = "dark";
  let density = "comfortable";
  let view: "grid" | "list" = "grid";
  let fontSize = 20;
  let lineHeight = 1.5;
  let toast = "";
  let toastTimer: ReturnType<typeof setTimeout>;
  let session = 0;
  const previewOptions = [
    ["library", "Библиотека"],
    ["empty", "Пустая библиотека"],
    ["login", "Вход"],
    ["create", "Создание"],
    ["recovery", "Код восстановления"],
    ["reader", "Чтение EPUB"],
    ["pdf", "Чтение PDF"],
    ["fb2", "Чтение FB2"],
    ["unsupported", "Другой формат"],
    ["settings", "Настройки"],
    ["import", "Импорт"],
    ["duplicate", "Дубликат"],
    ["issues", "Ошибки импорта"],
    ["backup", "Резервная копия"],
    ["verify", "Проверка копии"],
    ["verified", "Копия готова"],
    ["restore", "Восстановление"],
    ["restored", "Восстановлено"],
  ];
  function notify(message: string) {
    toast = message;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => (toast = ""), 4000);
  }
  function preference(key: string, value: string | number) {
    if (key === "theme" && (value === "dark" || value === "light"))
      theme = value;
    if (key === "density" && (value === "compact" || value === "comfortable"))
      density = value;
    if (key === "fontSize")
      fontSize = Math.max(16, Math.min(28, Number(value)));
    if (key === "lineHeight")
      lineHeight = Math.max(1.4, Math.min(1.8, Number(value)));
    if (key === "autoLockMinutes") {
      autoLockMinutes = Number(value);
      if (!preview)
        void SetAutoLockMinutes(autoLockMinutes).catch(() =>
          notify("Не удалось изменить автоблокировку."),
        );
    }
    try {
      localStorage.setItem(
        "mut.appearance",
        JSON.stringify({
          theme,
          density,
          view,
          fontSize,
          lineHeight,
          autoLockMinutes,
        }),
      );
    } catch {}
  }
  function setView(value: "grid" | "list") {
    view = value;
    preference("view", value);
  }
  $: document.documentElement.dataset.theme = theme;
  onMount(() => {
    try {
      const settings = JSON.parse(
        localStorage.getItem("mut.appearance") || "{}",
      );
      theme = settings.theme === "light" ? "light" : "dark";
      density = settings.density === "compact" ? "compact" : "comfortable";
      autoLockMinutes = [0, 5, 10, 20, 60, 120].includes(
        settings.autoLockMinutes,
      )
        ? settings.autoLockMinutes
        : 20;
      view = settings.view === "list" ? "list" : "grid";
      fontSize = [16, 18, 20, 22, 24, 26, 28].includes(settings.fontSize)
        ? settings.fontSize
        : 20;
      lineHeight = [1.4, 1.5, 1.8].includes(settings.lineHeight)
        ? settings.lineHeight
        : 1.5;
    } catch {}
    if (preview) showPreview(previewName);
    else void initialize();
    const cancel =
      !preview && (window as unknown as { runtime?: unknown }).runtime
        ? EventsOn("vault:locked", clearSession)
        : () => {};
    return () => {
      clearTimeout(toastTimer);
      cancel();
    };
  });
  async function initialize() {
    try {
      try {
        const native = await getNativeDefaults();
        if (native) {
          for (const key of Object.keys(defaults) as (keyof DirectoryPaths)[]) {
            if (typeof native[key] === "string" && native[key].startsWith("/"))
              defaults[key] = native[key];
          }
        }
      } catch {
        notify(
          "Не удалось получить стандартные папки. Используются пути этой сборки.",
        );
      }
      await SetAutoLockMinutes(autoLockMinutes);
      paths = loadSavedPaths(defaults, pathStorageKey);
      path = paths.library;
      if (await IsUnlocked()) {
        await loadLibrary();
        screen = "library";
      } else {
        screen =
          (await IsVaultConfigurated()) || path !== defaults.library
            ? "login"
            : "create";
      }
    } catch {
      errorMessage = "Не удалось открыть библиотеку. Попробуйте ещё раз.";
    } finally {
      ready = true;
    }
  }
  async function loadLibrary() {
    const currentSession = session;
    const [bookData, shelfData] = await Promise.all([
      ListBooks(),
      ListShelves(),
    ]);
    if (currentSession !== session) return;
    books = (bookData || []).map(fromCatalog);
    shelves = shelfData || [];
  }
  async function unlockVault(
    folder: string,
    secret: string,
    recovery: boolean,
  ) {
    busy = true;
    errorMessage = "";
    path = folder;
    try {
      if (preview) {
        books = structuredClone(demoBooks);
        shelves = [...demoShelves];
      } else {
        await (recovery
          ? UnlockWithRecovery(folder, secret)
          : Unlock(folder, secret));
        await loadLibrary();
      }
      screen = "library";
      updatePath("library", folder);
    } catch {
      errorMessage =
        "Не удалось открыть архив. Проверьте папку и " +
        (recovery ? "код восстановления." : "пароль.");
    } finally {
      busy = false;
    }
  }
  async function createVault(folder: string, password: string) {
    busy = true;
    errorMessage = "";
    path = folder;
    try {
      recoveryKey = preview
        ? demoRecovery
        : await CreateVault(folder, password);
      screen = "recovery";
      updatePath("library", folder);
    } catch {
      errorMessage =
        "Не удалось создать библиотеку. Проверьте путь и доступ к папке.";
    } finally {
      busy = false;
    }
  }
  async function finishSetup() {
    busy = true;
    errorMessage = "";
    try {
      if (preview) {
        books = [];
        shelves = [];
      } else await loadLibrary();
      recoveryKey = "";
      screen = "library";
    } catch {
      errorMessage =
        "Библиотека создана, но не удалось загрузить каталог. Попробуйте ещё раз.";
    } finally {
      busy = false;
    }
  }
  async function lockVault() {
    try {
      if (!preview) await Lock();
      clearSession();
    } catch {
      notify("Не удалось заблокировать архив. Попробуйте ещё раз.");
    }
  }
  function clearSession() {
    session++;
    screen = "login";
    overlay = null;
    books = [];
    shelves = [];
    selectedBook = null;
    recoveryKey = "";
    errorMessage = "";
  }
  function activity() {
    if (preview) return;
    const now = Date.now();
    if (now - lastTouch > 15000) {
      lastTouch = now;
      void Touch();
    }
  }
  function readingProgress(id: string, progress: number, locator: string) {
    books = books.map((book) =>
      book.id === id
        ? { ...book, progress, locator, updatedAt: new Date().toISOString() }
        : book,
    );
  }
  async function refresh() {
    if (!preview) await loadLibrary();
  }
  async function importFile(filePath: string, keepDuplicate = false) {
    if (preview) {
      const filename = filePath.split("/").at(-1) || "Книга";
      const id =
        "demo-import-" + Date.now() + "-" + Math.random().toString(36).slice(2);
      books = [
        ...books,
        {
          id,
          title: filename.replace(/\.[^.]+$/, ""),
          author: "Автор не указан",
          format: filePath.split(".").at(-1)?.toUpperCase() || "FILE",
          size: 1024 * 1024,
          progress: 0,
          shelves: [],
          bookmarks: [],
          originalName: filename,
          updatedAt: new Date().toISOString(),
          tone: books.length % 12,
        },
      ];
    } else {
      await ImportEdition(filePath, keepDuplicate);
    }
  }
  function openBook(book: Book) {
    selectedBook = book;
    screen = canRead(book.format) ? "reader" : "library";
    overlay = canRead(book.format) ? null : "book";
  }
  function openImport() {
    importInitial = "choose";
    overlay = "import";
  }
  function openBackup() {
    if (screen !== "backup" && screen !== "restore") flowReturn = screen;
    overlay = null;
    backupInitial = "choose";
    screen = "backup";
  }
  function openRestore() {
    if (screen !== "backup" && screen !== "restore") flowReturn = screen;
    overlay = null;
    backupInitial = "choose";
    screen = "restore";
  }
  function updatePath(key: keyof DirectoryPaths, value: string) {
    value = value.trim();
    if (key === "library") path = value;
    if (!value.startsWith("/")) return;
    paths = { ...paths, [key]: value };
    try {
      localStorage.setItem(pathStorageKey, JSON.stringify(paths));
    } catch {
      notify(
        "Не удалось запомнить папку. При следующем запуске введите путь снова.",
      );
    }
  }
  async function selectImportFiles() {
    const selected = await chooseFiles(paths.imports);
    if (selected.length) {
      const directory =
        selected[0]!.slice(0, selected[0]!.lastIndexOf("/")) || "/";
      updatePath("imports", directory);
    }
    return selected;
  }
  function showPreview(name: string) {
    flowReturn = "library";
    previewName = name;
    errorMessage = "";
    overlay = null;
    selectedBook = null;
    recoveryKey = "";
    books = structuredClone(demoBooks);
    shelves = [...demoShelves];
    screen = "library";
    backupInitial = "choose";
    importInitial = "choose";
    if (["login", "create", "recovery"].includes(name)) {
      screen = name as Screen;
      if (name === "recovery") recoveryKey = demoRecovery;
    }
    if (name === "empty") {
      books = [];
      shelves = [];
    }
    if (["reader", "pdf", "fb2"].includes(name)) {
      selectedBook = books[name === "pdf" ? 1 : name === "fb2" ? 2 : 0]!;
      screen = "reader";
    }
    if (name === "unsupported") {
      selectedBook = {
        ...books[0]!,
        title: "Полевые заметки",
        format: "DOCX",
        originalName: "field-notes.docx",
      };
      overlay = "book";
    }
    if (name === "settings") overlay = "settings";
    if (["import", "duplicate", "issues"].includes(name)) {
      overlay = "import";
      importInitial = name === "import" ? "choose" : name;
    }
    if (["backup", "verify", "verified"].includes(name)) {
      screen = "backup";
      backupInitial =
        name === "backup" ? "choose" : name === "verify" ? "verify" : "done";
    }
    if (["restore", "restored"].includes(name)) {
      screen = "restore";
      backupInitial = name === "restored" ? "done" : "choose";
    }
  }
  async function createShelf() {
    const name = shelfName.trim();
    if (!name || busy) return;
    busy = true;
    errorMessage = "";
    try {
      if (preview) {
        if (!shelves.includes(name)) shelves = [...shelves, name];
      } else {
        await CreateShelf(name);
        await loadLibrary();
      }
      shelfName = "";
      overlay = null;
    } catch {
      errorMessage =
        "Не удалось создать полку. Проверьте название и доступ к диску.";
    } finally {
      busy = false;
    }
  }
  function manageShelf(name: string) {
    editingShelf = name;
    shelfName = name;
    confirmShelfDelete = false;
    errorMessage = "";
    overlay = "manageshelf";
  }
  async function changeShelf(remove = false) {
    busy = true;
    errorMessage = "";
    try {
      if (preview) {
        shelves = remove
          ? shelves.filter((s) => s !== editingShelf)
          : shelves.map((s) => (s === editingShelf ? shelfName : s));
        books = books.map((b) => ({
          ...b,
          shelves: remove
            ? b.shelves.filter((s) => s !== editingShelf)
            : b.shelves.map((s) => (s === editingShelf ? shelfName : s)),
        }));
      } else {
        if (remove) await DeleteShelf(editingShelf);
        else await RenameShelf(editingShelf, shelfName);
        await loadLibrary();
      }
      overlay = null;
    } catch {
      errorMessage = "Не удалось изменить полку.";
    } finally {
      busy = false;
    }
  }
  async function saveShelves(chosen: string[]) {
    if (!selectedBook) return;
    if (preview) {
      selectedBook.shelves = chosen;
      books = [...books];
    } else {
      await SetBookShelves(selectedBook.id, chosen);
      await refresh();
    }
  }
  async function exportBook() {
    if (!selectedBook) return;
    if (preview) {
      notify("Экспорт доступен в desktop-приложении.");
      return;
    }
    const destination = await ChooseExportPath(selectedBook.id);
    if (!destination) return;
    await ExportOriginal(selectedBook.id, destination);
    notify("Оригинал экспортирован в выбранную папку.");
  }
  async function removeBook() {
    if (!selectedBook) return;
    const id = selectedBook.id;
    if (preview) books = books.filter((b) => b.id !== id);
    else {
      await DeleteBook(id);
      await refresh();
    }
    overlay = null;
    selectedBook = null;
  }
</script>

<svelte:window on:pointerdown={activity} on:keydown={activity} />
<div class="app-root">
  {#if !ready}<main class="loading-screen">
      <Icon name="book" size={32} />
      <p class="muted">Открываем mut…</p>
    </main>
  {:else if screen === "login"}<LoginScreen
      loading={busy}
      {errorMessage}
      {path}
      defaultPath={defaults.library}
      pickerAvailable={directoryPickerAvailable}
      onpathchange={(value) => updatePath("library", value)}
      onbrowse={chooseDirectory}
      onunlock={unlockVault}
      oncreate={() => {
        errorMessage = "";
        screen = "create";
      }}
      onrestore={openRestore}
    />
  {:else if screen === "create" || screen === "recovery"}{#key screen}<CreateLibrary
        step={screen === "recovery" ? 2 : 1}
        {recoveryKey}
        loading={busy}
        {errorMessage}
        {path}
        defaultPath={defaults.library}
        pickerAvailable={directoryPickerAvailable}
        onpathchange={(value) => updatePath("library", value)}
        onbrowse={chooseDirectory}
        oncreate={createVault}
        onfinish={finishSetup}
        onlogin={() => {
          errorMessage = "";
          screen = "login";
        }}
        onrestore={openRestore}
      />{/key}
  {:else if screen === "reader" && selectedBook}{#key selectedBook.id}<ReaderView
        book={selectedBook}
        {preview}
        {fontSize}
        {lineHeight}
        onback={() => (screen = "library")}
        onsettings={() => (overlay = "settings")}
        onsecurity={() => (overlay = "security")}
        onchange={preference}
        onprogress={readingProgress}
        onbookmarks={(id, bookmarks) => { books = books.map(book => book.id === id ? {...book, bookmarks} : book); }}
      />{/key}
  {:else if screen === "backup" || screen === "restore"}{#key screen + backupInitial}<BackupView
        restore={screen === "restore"}
        {preview}
        initial={backupInitial}
        {books}
        {shelves}
        backupPath={paths.backup}
        restorePath={paths.restore}
        defaultBackupPath={defaults.backup}
        defaultRestorePath={defaults.restore}
        pickerAvailable={directoryPickerAvailable}
        onpathchange={updatePath}
        onbrowse={chooseDirectory}
        onback={() => (screen = flowReturn)}
        onrestore={openRestore}
        onrestored={(value) => {
          if (flowReturn === "login" || flowReturn === "create") {
            updatePath("library", value);
            flowReturn = "login";
          }
        }}
      />{/key}
  {:else}<LibraryView
      {books}
      {shelves}
      {density}
      {view}
      onview={setView}
      onlock={lockVault}
      onimport={openImport}
      onshelf={() => {
        shelfName = "";
        overlay = "shelf";
      }}
      onsettings={() => (overlay = "settings")}
      onbackups={openBackup}
      onsecurity={() => (overlay = "security")}
      onopen={openBook}
      ondetails={(book) => {
        selectedBook = book;
        overlay = "book";
      }}
      onmanageshelf={manageShelf}
    />{/if}
  {#if overlay === "settings"}<Dialog
      title="Настройки mut"
      animateClose
      subtitle="Ваша библиотека · оформление сохраняется автоматически"
      wide
      anchored
      onclose={() => (overlay = null)}
      ><SettingsView
        {theme}
        {density}
        {fontSize}
        {lineHeight}
        {autoLockMinutes}
        {books}
        {path}
        onchange={preference}
        onbackup={openBackup}
        onpasswordchange={async (oldPassword, newPassword) => {
          if (preview) {
            notify("Смена пароля доступна в приложении.");
            return;
          }
          await ChangePassword(oldPassword, newPassword);
        }}
      /></Dialog
    >
  {:else if overlay === "import"}<Dialog
      title={importInitial === "choose"
        ? "Добавить книги"
        : "Добавляем вашу коллекцию"}
      subtitle="Локальный зашифрованный архив"
      onclose={() => (overlay = null)}
      ><ImportView
        {preview}
        initial={importInitial}
        importDirectory={paths.imports}
        defaultImportDirectory={defaults.imports}
        pickerAvailable={filePickerAvailable}
        {directoryPickerAvailable}
        onpathchange={(value) => updatePath("imports", value)}
        onbrowse={chooseDirectory}
        onselectfiles={selectImportFiles}
        onselectfolder={() =>
          preview ? Promise.resolve([]) : ListImportFiles(paths.imports)}
        oncomplete={refresh}
        onimport={importFile}
        onclose={() => (overlay = null)}
      /></Dialog
    >
  {:else if overlay === "book" && selectedBook}<Dialog
      title="Карточка книги"
      onclose={() => {
        overlay = null;
        selectedBook = null;
      }}
      ><BookDetails
        book={selectedBook}
        {shelves}
        onread={() => {
          if (selectedBook) openBook(selectedBook);
        }}
        onsave={saveShelves}
        onexport={exportBook}
        ondelete={removeBook}
      /></Dialog
    >
  {:else if overlay === "shelf"}<Dialog
      title="Новая полка"
      subtitle="Соберите вместе книги, к которым хочется возвращаться"
      onclose={() => (overlay = null)}
      ><form class="stack" on:submit|preventDefault={createShelf}>
        <label class="field"
          ><span>Название полки</span><input
            bind:value={shelfName}
            placeholder="Например, перед сном"
            maxlength="80"
            required
          /></label
        >{#if errorMessage}<p class="error" role="alert">{errorMessage}</p>{/if}
        <div class="footer-row">
          <button
            type="button"
            class="secondary"
            on:click={() => (overlay = null)}>Отмена</button
          ><button class="primary" disabled={!shelfName.trim() || busy}
            >Создать полку <Icon name="plus" size={16} /></button
          >
        </div>
      </form></Dialog
    >
  {:else if overlay === "manageshelf"}<Dialog
      title="Изменить полку"
      onclose={() => (overlay = null)}
      ><form class="stack" on:submit|preventDefault={() => changeShelf()}>
        <label class="field"
          ><span>Название полки</span><input
            bind:value={shelfName}
            required
            maxlength="80"
          /></label
        >{#if errorMessage}<p class="error" role="alert">
            {errorMessage}
          </p>{/if}<button class="primary" disabled={busy || !shelfName.trim()}
          >Сохранить</button
        ><label class="check"
          ><input type="checkbox" bind:checked={confirmShelfDelete} /> Удалить полку,
          сохранив книги</label
        ><button
          class="secondary"
          type="button"
          disabled={!confirmShelfDelete || busy}
          on:click={() => changeShelf(true)}>Удалить полку</button
        >
      </form></Dialog
    >
  {:else if overlay === "security"}<Dialog
      title="Ваш личный архив"
      subtitle="Локально. Под вашим контролем."
      onclose={() => (overlay = null)}
      ><div class="stack">
        <div class="security-status">
          <Icon name="shield" size={32} />
          <div>
            <h3>Библиотека открыта</h3>
            <p class="muted">Книги доступны до блокировки архива.</p>
          </div>
        </div>
        <div class="notice">
          <strong>На диске файлы зашифрованы</strong>
          <p>
            При чтении страницы видны в открытом виде. Блокировка закрывает
            библиотеку и скрывает названия книг.
          </p>
        </div>
        <div class="footer-row">
          <button class="secondary" on:click={openBackup}
            ><Icon name="disk" size={16} /> Резервные копии</button
          ><button class="primary" on:click={lockVault}
            ><Icon name="lock" size={16} /> Заблокировать</button
          >
        </div>
      </div></Dialog
    >{/if}
  {#if preview}<div class="preview-control">
      <span>Дизайн</span><select
        aria-label="Просмотр экранов дизайна"
        value={previewName}
        on:change={(e) => showPreview(e.currentTarget.value)}
        >{#each previewOptions as [id, label]}<option value={id}>{label}</option
          >{/each}</select
      ><span class="preview-only">Демо-данные</span>
    </div>{/if}
  {#if toast}<div class="toast panel" role="status">{toast}</div>{/if}
</div>

<style>
  .app-root {
    height: 100%;
    width: 100%;
    position: relative;
    isolation: isolate;
    background: var(--bg);
  }
  .loading-screen {
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 20px;
  }
  .preview-control {
    position: fixed;
    left: 50%;
    bottom: 14px;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    gap: 10px;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: var(--panel);
    box-shadow: 0 4px 16px #0003;
    padding: 6px 12px;
    color: var(--muted);
    z-index: 30;
    font-size: 9px;
    white-space: nowrap;
  }
  .preview-control > span:first-child {
    letter-spacing: 0.8px;
    text-transform: uppercase;
  }
  .preview-control select {
    font-size: 10px;
    border: none;
    border-radius: 5px;
    padding: 2px 4px;
    width: auto;
    background: var(--panel);
    max-width: 155px;
  }
  .preview-only {
    color: var(--quiet);
    border-left: 1px solid var(--border);
    padding-left: 10px;
  }
  .toast {
    position: fixed;
    bottom: 70px;
    left: 50%;
    transform: translateX(-50%);
    padding: 14px 22px;
    box-shadow: 0 8px 32px #0004;
    z-index: 40;
    max-width: calc(100vw - 40px);
  }
  .security-status {
    display: flex;
    align-items: center;
    gap: 18px;
    color: var(--success);
  }
  .security-status h3 {
    color: var(--text);
  }
  .security-status p {
    margin-top: 6px;
    font-size: 12px;
  }
  @media (max-width: 550px) {
    .preview-control {
      bottom: 10px;
    }
    .preview-only {
      display: none;
    }
  }
</style>
