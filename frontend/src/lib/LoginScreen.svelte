<script lang="ts">
  import Brand from "./Brand.svelte";
  import Icon from "./Icon.svelte";
  import PathField from "./PathField.svelte";
  export let loading = false;
  export let errorMessage = "";
  export let path = "";
  export let defaultPath: string;
  export let pickerAvailable = false;
  export let onpathchange: (value: string) => void;
  export let onbrowse: (title: string, currentPath: string) => Promise<string>;
  export let onunlock: (
    path: string,
    secret: string,
    recovery: boolean,
  ) => void;
  export let oncreate: () => void;
  export let onrestore: () => void;
  let secret = "";
  let visible = false;
  let recovery = false;
  function submit() {
    if (secret.trim() && path.trim() && !loading) {
      const value = secret;
      secret = "";
      onunlock(path.trim(), value, recovery);
    }
  }
</script>

<main class="login-page">
  <section class="login-card panel">
    <div class="row spread">
      <Brand subtitle />
      <div class="status-icon"><Icon name="lock" /></div>
    </div>
    <div class="intro">
      <h1>Вернитесь к своим книгам.</h1>
      <p class="muted">
        Откройте личную библиотеку и продолжите с того места, где остановились.
      </p>
    </div>
    <div class="hairlines" aria-hidden="true"><i></i><i></i><i></i></div>
    <form on:submit|preventDefault={submit} class="stack">
      <PathField
        label="Папка библиотеки"
        bind:value={path}
        {defaultPath}
        {pickerAvailable}
        onchange={onpathchange}
        {onbrowse}
      />
      <div class="field">
        <div class="row spread">
          <label for="archive-key" class="eyebrow"
            >{recovery ? "Код восстановления" : "Пароль архива"}</label
          ><button
            class="text-button"
            type="button"
            on:click={() => {
              recovery = !recovery;
              secret = "";
              visible = false;
            }}>{recovery ? "Войти с паролем" : "Использовать код"}</button
          >
        </div>
        <div class="input-wrap">
          <span class="input-icon"><Icon name="lock" size={16} /></span><input
            id="archive-key"
            type={visible ? "text" : "password"}
            bind:value={secret}
            placeholder={recovery
              ? "Введите полный код восстановления"
              : "Введите пароль"}
            required
            autocomplete="off"
          /><button
            type="button"
            aria-label={visible ? "Скрыть пароль" : "Показать пароль"}
            aria-pressed={visible}
            on:click={() => (visible = !visible)}><Icon name="eye" /></button
          >
        </div>
      </div>
      {#if errorMessage}<p class="error" role="alert">{errorMessage}</p>{/if}
      <button
        class="primary enter"
        type="submit"
        disabled={loading || !secret.trim() || !path.trim()}
        ><span>{loading ? "Открываем архив…" : "Войти в mut"}</span><Icon
          name="arrow"
        /></button
      >
    </form>
    <div class="login-links">
      <button class="text-button" on:click={oncreate}
        >Создать библиотеку <Icon name="arrow" size={14} /></button
      ><button class="text-button" on:click={onrestore}
        >Восстановить копию</button
      >
    </div>
    <p class="principles">
      На устройстве <span>·</span> Зашифровано <span>·</span> Без интернета
    </p>
  </section>
</main>

<style>
  .login-page {
    height: 100%;
    overflow: auto;
    display: grid;
    place-items: center;
    padding: 40px 24px;
  }
  .login-card {
    width: min(520px, 100%);
    padding: 30px;
    border-radius: 24px;
    background: var(--bg);
  }
  .status-icon {
    height: 36px;
    width: 36px;
    border-radius: 50%;
    background: var(--panel);
    display: grid;
    place-items: center;
    color: var(--muted);
    border: 1px solid var(--border);
  }
  .intro {
    margin: 32px 0 24px;
  }
  .intro h1 {
    font-size: 27px;
  }
  .intro p {
    font-size: 12px;
    margin-top: 8px;
  }
  .hairlines {
    margin-bottom: 24px;
  }
  .stack {
    gap: 20px;
  }
  .enter {
    justify-content: space-between;
    height: 50px;
  }
  .input-icon {
    color: var(--muted);
    padding-left: 15px;
    display: flex;
  }
  .input-icon + input {
    padding-left: 0;
  }
  .login-links {
    display: flex;
    justify-content: space-between;
    gap: 8px;
    margin-top: 20px;
    font-size: 11px;
  }
  .principles {
    text-align: center;
    color: var(--muted);
    font-size: 10px;
    margin-top: 28px;
  }
  .principles span {
    margin: 0 9px;
  }
</style>
