<script lang="ts">
  import Brand from "./Brand.svelte";
  import Icon from "./Icon.svelte";
  import { formatSize, type Book } from "./ui";
  import PathField from "./PathField.svelte";
  export let restore = false;
  export let preview = false;
  export let initial = "choose";
  export let books: Book[] = [];
  export let shelves: string[] = [];
  export let onback: () => void;
  export let onrestore: () => void;
  export let backupPath: string;
  export let restorePath: string;
  export let defaultBackupPath: string;
  export let defaultRestorePath: string;
  export let pickerAvailable = false;
  export let onpathchange: (key: "backup" | "restore", value: string) => void;
  export let onbrowse: (title: string, currentPath: string) => Promise<string>;
  let state = initial;
  let folder = backupPath;
  let destination = restorePath;
  let secret = "";
  let recovery = false;
  let visible = false;
  $: totalSize = books.reduce((sum, b) => sum + b.size, 0);
  function advance() {
    secret = "";
    state = state === "choose" ? "verify" : "done";
  }
</script>

<main class="flow-page">
  <header class="page-header">
    <Brand /><span
      >{restore ? "Восстановление библиотеки" : "Резервные копии"}</span
    ><span class="local-label">Личная библиотека</span>
  </header>
  <div class="flow-content">
    <section class="flow-card">
      {#if state === "choose"}<span class="eyebrow"
          >{restore
            ? "Возвращение к книгам"
            : "Копия отдельно от устройства"}</span
        >
        <h1 class="flow-title">
          {restore
            ? "Восстановите библиотеку из копии"
            : "Сохраните ещё одну копию библиотеки"}
        </h1>
        <p>
          {restore
            ? "Выберите копию и новую папку для библиотеки."
            : "Копия сохранит книги, полки и позиции чтения, если вы потеряете это устройство."}
        </p>
        <form class="stack" on:submit|preventDefault={advance}>
          {#if !restore}<div class="library-summary">
              <div>
                <h3>Ваша библиотека</h3>
                <p class="muted">
                  {books.length} книг · {shelves.length} полок · позиции чтения и
                  закладки
                </p>
              </div>
              <span>{formatSize(totalSize)}</span>
            </div>{/if}
          <PathField
            label={restore ? "Папка резервной копии" : "Куда сохранить копию"}
            bind:value={folder}
            defaultPath={defaultBackupPath}
            {pickerAvailable}
            onchange={(value) => onpathchange("backup", value)}
            {onbrowse}
          />
          {#if restore}<div class="row">
              <button
                class={recovery ? "secondary" : "primary"}
                type="button"
                on:click={() => {
                  recovery = false;
                  secret = "";
                }}>Пароль копии</button
              ><button
                class={recovery ? "primary" : "secondary"}
                type="button"
                on:click={() => {
                  recovery = true;
                  secret = "";
                }}>Код восстановления</button
              >
            </div>
            <div class="field">
              <label for="restore-secret"
                >{recovery
                  ? "Код восстановления"
                  : "Пароль, действовавший при создании копии"}</label
              >
              <div class="input-wrap">
                <input
                  id="restore-secret"
                  bind:value={secret}
                  type={visible ? "text" : "password"}
                  required
                  autocomplete="off"
                /><button type="button" on:click={() => (visible = !visible)}
                  >{visible ? "Скрыть" : "Показать"}</button
                >
              </div>
            </div>
            <PathField
              label="Новая папка библиотеки"
              bind:value={destination}
              defaultPath={defaultRestorePath}
              {pickerAvailable}
              onchange={(value) => onpathchange("restore", value)}
              {onbrowse}
            />{/if}
          <div class="notice">
            <strong
              >{restore
                ? "Исходная библиотека и копия сохранятся"
                : "Копия останется зашифрованной"}</strong
            >
            <p>
              {restore
                ? "Выберите пустую папку. Перед восстановлением файлы должны пройти проверку целостности."
                : "Чтобы открыть её на другом компьютере, понадобятся пароль или код восстановления. Лучше выбрать другой носитель."}
            </p>
          </div>
          {#if !restore}<div>
              <span class="eyebrow">После копирования</span>
              <p class="backup-note">
                mut должен повторно прочитать файлы копии и сверить контрольные
                суммы.
              </p>
              <p class="muted">
                Носитель можно отключить после завершения проверки.
              </p>
            </div>{/if}
          {#if !preview}<p class="availability muted">
              {restore
                ? "Восстановление пока недоступно в приложении."
                : "Создание копии пока недоступно в приложении."}
            </p>{/if}
          <div class="footer-row">
            <div class="row">
              <button type="button" class="secondary" on:click={onback}
                ><Icon name="back" size={16} /> Назад</button
              >{#if !restore}<button
                  type="button"
                  class="text-button"
                  on:click={onrestore}>Восстановить копию</button
                >{/if}
            </div>
            <button
              class="primary"
              disabled={!preview ||
                !folder.trim() ||
                (restore && (!secret || !destination.trim()))}
              >{preview
                ? "Посмотреть проверку"
                : restore
                  ? "Проверить и восстановить"
                  : "Создать и проверить"}
              <Icon name="arrow" size={16} /></button
            >
          </div>
        </form>
      {:else if state === "verify"}<span class="eyebrow"
          >{restore ? "Проверка перед восстановлением" : "Проверка копии"}</span
        >
        <h1 class="flow-title">Убеждаемся, что каждая книга на месте</h1>
        <p>Проверка целостности файлов и структуры архива.</p>
        <div class="stack">
          <div class="verification">
            <div class="row spread">
              <span>Проверено 14 из 20 файлов</span><span>70%</span>
            </div>
            <div class="progress-track"><span style:width="70%"></span></div>
            <p class="muted">
              Имена книг остаются скрыты до открытия библиотеки.
            </p>
          </div>
          <div class="status-row">
            <Icon name="check" /><span>Структура архива</span><span
              class="muted">Проверена</span
            >
          </div>
          <div class="status-row">
            <Icon name="check" /><span>Контрольные суммы</span><span
              class="muted">14 из 20</span
            >
          </div>
          <div class="notice">
            <strong>Не отключайте носитель</strong>
            <p>
              Дождитесь окончания проверки. Она подтверждает, что все файлы
              скопированы целиком.
            </p>
          </div>
          <div class="footer-row">
            <button class="secondary" on:click={onback}>Назад</button><button
              class="primary"
              on:click={advance}
              >Посмотреть результат <Icon name="arrow" size={16} /></button
            >
          </div>
        </div>
      {:else}<div class="verified">
          <div class="verified-icon"><Icon name="shield" size={36} /></div>
          <span class="eyebrow"
            >{restore ? "Библиотека готова" : "Копия проверена"}</span
          >
          <h1>
            {restore
              ? "Все ваши книги снова с вами"
              : "Ещё одно надёжное место для книг"}
          </h1>
          <p class="muted">
            {restore
              ? "Книги, полки, закладки и позиции чтения восстановлены."
              : "Файлы копии прочитаны повторно. Контрольные суммы совпадают."}
          </p>
          <div class="verified-report">
            <div><span>Файлы</span><strong>20 из 20</strong></div>
            <div><span>Целостность</span><strong>Проверена</strong></div>
            <div><span>Ошибки</span><strong>0</strong></div>
          </div>
          <div class="notice">
            <strong
              >{restore
                ? "Можно продолжать читать"
                : "Носитель можно отключить"}</strong
            >
            <p>
              {restore
                ? "Откройте библиотеку с паролем или кодом восстановления."
                : "Храните копию отдельно от основного устройства. Периодически проверяйте её."}
            </p>
          </div>
          <button class="primary" on:click={onback}
            >К библиотеке <Icon name="arrow" size={16} /></button
          >
        </div>{/if}
      {#if preview}<p class="preview-caption">
          Демонстрация интерфейса. Файлы не копируются и не проверяются.
        </p>{/if}
    </section>
  </div>
</main>

<style>
  .flow-title {
    margin-top: 12px;
  }
  .flow-card > h1 + p {
    margin-bottom: 28px;
  }
  .library-summary {
    border-top: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    padding: 22px 0;
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 20px;
  }
  .library-summary p {
    margin-top: 5px;
  }
  .library-summary > span {
    font-size: 24px;
    white-space: nowrap;
  }
  .backup-note {
    margin: 12px 0;
  }
  .availability {
    font-size: 12px;
  }
  .verification {
    padding: 24px 0;
  }
  .verification .progress-track {
    height: 5px;
    margin: 18px 0;
  }
  .status-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 0;
    border-bottom: 1px solid var(--line);
  }
  .status-row > span:last-child {
    margin-left: auto;
  }
  .verified {
    text-align: center;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 18px;
    padding: 20px 0;
  }
  .verified-icon {
    width: 72px;
    height: 72px;
    border-radius: 22px;
    display: grid;
    place-items: center;
    color: var(--success);
    background: var(--raised);
    border: 1px solid var(--border);
    margin-bottom: 8px;
  }
  .verified-report {
    display: flex;
    width: 100%;
    justify-content: space-around;
    border-top: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
    padding: 24px 0;
    margin: 10px 0;
  }
  .verified-report > div {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .verified-report span {
    color: var(--muted);
    font-size: 12px;
  }
  .verified-report strong {
    font-size: 18px;
    font-weight: 500;
  }
  .verified .notice {
    text-align: left;
    width: 100%;
  }
  .preview-caption {
    font-size: 10px;
    color: var(--quiet);
    margin-top: 24px;
    text-align: center;
  }
  @media (max-width: 650px) {
    .footer-row {
      flex-wrap: wrap;
      gap: 20px;
    }
    .library-summary > span {
      font-size: 20px;
    }
    .library-summary p {
      font-size: 11px;
    }
  }
</style>
