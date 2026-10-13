<script lang="ts">
  import Icon from "./Icon.svelte";
  import { formatSize, type Book } from "./ui";
  export let theme = "dark";
  export let density = "comfortable";
  export let fontSize = 20;
  export let autoLockMinutes = 20;
  export let lineHeight = 1.5;
  export let books: Book[] = [];
  export let path = "";
  export let onchange: (key: string, value: string | number) => void;
  export let onbackup: () => void;
  export let onpasswordchange: (
    oldPassword: string,
    newPassword: string,
  ) => Promise<void>;
  let oldPassword = "";
  let newPassword = "";
  let confirmation = "";
  let passwordBusy = false;
  let passwordMessage = "";
  async function changePassword() {
    if (newPassword.length < 8 || newPassword !== confirmation) return;
    passwordBusy = true;
    passwordMessage = "";
    const old = oldPassword;
    const next = newPassword;
    oldPassword = "";
    newPassword = "";
    confirmation = "";
    try {
      await onpasswordchange(old, next);
      passwordMessage =
        "Пароль изменён. Код восстановления продолжает работать.";
    } catch {
      passwordMessage =
        "Не удалось изменить пароль. Проверьте текущий пароль и доступ к диску.";
    } finally {
      passwordBusy = false;
    }
  }
  let tab = "general";
  const tabs = [
    { id: "general", title: "Общие", icon: "settings" },
    { id: "reading", title: "Чтение", icon: "book" },
    { id: "storage", title: "Хранилище", icon: "disk" },
    { id: "about", title: "О приложении", icon: "info" },
  ];
</script>

<div class="settings-layout">
  <nav aria-label="Разделы настроек">
    <span class="eyebrow">Разделы</span>{#each tabs as item}<button
        class:active={tab === item.id}
        on:click={() => (tab = item.id)}
        aria-current={tab === item.id ? "page" : undefined}
        ><Icon name={item.icon} size={17} />{item.title}</button
      >{/each}
    <div class="version">
      <span>mut для Linux</span><span>Личная библиотека</span>
    </div>
  </nav>
  <section>
    {#if tab === "general"}<h3>Общие</h3>
      <p class="muted section-desc">Внешний вид приложения</p>
      <div class="setting">
        <div>
          <label for="theme">Оформление</label><small
            >Комфортный свет для вашей библиотеки</small
          >
        </div>
        <select
          id="theme"
          value={theme}
          on:change={(e) => onchange("theme", e.currentTarget.value)}
          ><option value="dark">Тёмное</option><option value="light"
            >Светлое</option
          ></select
        >
      </div>
      <div class="setting">
        <div>
          <label for="density">Плотность обложек</label><small
            >Больше книг или больше пространства</small
          >
        </div>
        <select
          id="density"
          value={density}
          on:change={(e) => onchange("density", e.currentTarget.value)}
          ><option value="comfortable">Свободная</option><option value="compact"
            >Компактная</option
          ></select
        >
      </div>
      <div class="setting">
        <div>
          <span>Язык интерфейса</span><small
            >Названия книг сохраняются в оригинале</small
          >
        </div>
        <span class="muted">Русский</span>
      </div>
      <div class="setting">
        <div>
          <label for="auto-lock">Автоблокировка</label><small
            >После отсутствия действий в приложении</small
          >
        </div>
        <select
          id="auto-lock"
          value={autoLockMinutes}
          on:change={(e) =>
            onchange("autoLockMinutes", Number(e.currentTarget.value))}
          >{#each [0, 5, 10, 20, 60, 120] as minutes}<option value={minutes}
              >{minutes ? minutes + " мин" : "Выключена"}</option
            >{/each}</select
        >
      </div>
    {:else if tab === "reading"}<h3>Чтение</h3>
      <p class="muted section-desc">Настройки страницы для EPUB и FB2</p>
      <div class="setting">
        <div>
          <label for="font-size">Размер текста</label><small
            >От 16 до 28 пикселей</small
          >
        </div>
        <select
          id="font-size"
          value={fontSize}
          on:change={(e) => onchange("fontSize", Number(e.currentTarget.value))}
          >{#each [16, 18, 20, 22, 24, 26, 28] as size}<option value={size}
              >{size} px</option
            >{/each}</select
        >
      </div>
      <div class="setting">
        <div>
          <label for="line-height">Межстрочный интервал</label><small
            >Расстояние между строками</small
          >
        </div>
        <select
          id="line-height"
          value={lineHeight}
          on:change={(e) =>
            onchange("lineHeight", Number(e.currentTarget.value))}
          ><option value={1.4}>Плотный</option><option value={1.5}
            >Обычный</option
          ><option value={1.8}>Свободный</option></select
        >
      </div>
      <div
        class="reading-sample"
        style:font-size={fontSize + "px"}
        style:line-height={lineHeight}
      >
        Всё важное движется медленно.<br />У каждой книги — своё время.
      </div>
    {:else if tab === "storage"}<h3>Хранилище</h3>
      <p class="muted section-desc">Ваша коллекция на этом устройстве</p>
      <div class="setting">
        <span>Книг в архиве</span><span>{books.length}</span>
      </div>
      <div class="setting">
        <span>Размер файлов</span><span
          >{formatSize(books.reduce((s, b) => s + b.size, 0))}</span
        >
      </div>
      <div class="storage-path">
        <span class="eyebrow">Папка библиотеки</span>
        <p>{path || "Папка не выбрана"}</p>
      </div>
      <button class="secondary" on:click={onbackup}
        ><Icon name="disk" size={16} /> Резервные копии</button
      >
      <form
        class="stack password-form"
        on:submit|preventDefault={changePassword}
      >
        <h3>Изменить пароль</h3>
        <label class="field"
          ><span>Текущий пароль</span><input
            bind:value={oldPassword}
            type="password"
            autocomplete="current-password"
            required
          /></label
        ><label class="field"
          ><span>Новый пароль</span><input
            bind:value={newPassword}
            type="password"
            minlength="8"
            autocomplete="new-password"
            required
          /></label
        ><label class="field"
          ><span>Повторите новый пароль</span><input
            bind:value={confirmation}
            type="password"
            autocomplete="new-password"
            required
          /></label
        ><button
          class="primary"
          disabled={passwordBusy ||
            !oldPassword ||
            newPassword.length < 8 ||
            newPassword !== confirmation}>Изменить пароль</button
        >{#if passwordMessage}<p class="muted" role="status">
            {passwordMessage}
          </p>{/if}
      </form>
    {:else}<h3>Ваши книги остаются вашими</h3>
      <p class="about-text muted">
        mut — тихое место для чтения и личный архив. Без аккаунта, ленты и
        подписки. Для книг, к которым хочется возвращаться.
      </p>
      <div class="setting">
        <span>Приложение</span><span class="muted">mut · 1.0.0-rc.1</span>
      </div>
      <div class="setting">
        <span>Платформа</span><span class="muted">Linux desktop</span>
      </div>{/if}
    <div class="privacy notice">
      <div class="row">
        <Icon name="lock" size={17} /><strong
          >Всё остаётся на этом устройстве</strong
        >
      </div>
      <p>Книги и история чтения не отправляются в облако.</p>
    </div>
  </section>
</div>

<style>
  .password-form {
    border-top: 1px solid var(--line);
    margin-top: 24px;
    padding-top: 24px;
    gap: 16px;
  }

  .settings-layout {
    display: grid;
    grid-template-columns: 214px minmax(0, 1fr);
    min-height: 480px;
  }
  nav {
    padding: 24px 14px;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--line);
    gap: 6px;
  }
  nav > .eyebrow {
    padding: 0 12px 8px;
  }
  nav button {
    height: 42px;
    justify-content: flex-start;
    padding: 0 12px;
    border-radius: 12px;
    color: var(--muted);
    gap: 12px;
  }
  nav button.active {
    background: var(--active);
    color: var(--text);
  }
  .version {
    margin-top: auto;
    padding: 20px 12px 0;
    display: flex;
    flex-direction: column;
    font-size: 10px;
    color: var(--quiet);
  }
  section {
    padding: 26px 30px;
    min-width: 0;
  }
  section h3 {
    font-size: 20px;
    font-weight: 600;
    letter-spacing: -0.6px;
  }
  .section-desc {
    font-size: 12px;
    margin-top: 4px;
    margin-bottom: 12px;
  }
  .setting {
    min-height: 76px;
    border-top: 1px solid var(--line);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 18px;
  }
  .setting small {
    display: block;
    color: var(--quiet);
    margin-top: 4px;
  }
  .setting select {
    width: 155px;
    padding: 9px 11px;
    font-size: 12px;
    border-radius: 10px;
    background: var(--panel);
  }
  .privacy {
    margin-top: 24px;
    padding: 16px;
  }
  .privacy strong {
    font-size: 12px;
    font-weight: 500;
  }
  .privacy p {
    font-size: 11px;
  }
  .reading-sample {
    background: var(--paper);
    color: var(--ink);
    font-family: Literata, Georgia, serif;
    padding: 24px;
    border-radius: 4px;
    margin-top: 20px;
  }
  .storage-path {
    padding: 20px 0;
    border-top: 1px solid var(--line);
    margin-bottom: 12px;
  }
  .storage-path p {
    overflow-wrap: anywhere;
    color: var(--muted);
    margin-top: 8px;
    user-select: text;
  }
  .about-text {
    line-height: 1.7;
    margin: 16px 0 28px;
  }
  @media (max-width: 650px) {
    .settings-layout {
      grid-template-columns: 1fr;
    }
    nav {
      flex-direction: row;
      overflow: auto;
      padding: 12px;
      border-right: 0;
      border-bottom: 1px solid var(--line);
    }
    nav > .eyebrow,
    .version {
      display: none;
    }
    nav button {
      white-space: nowrap;
      height: 36px;
      font-size: 11px;
      gap: 6px;
      padding: 0 8px;
    }
    section {
      padding: 22px;
    }
    .setting select {
      width: 128px;
    }
    .setting small {
      max-width: 200px;
      font-size: 10px;
    }
  }
</style>
