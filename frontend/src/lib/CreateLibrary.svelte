<script lang="ts">
  import Brand from "./Brand.svelte";
  import Icon from "./Icon.svelte";
  import PathField from "./PathField.svelte";
  export let recoveryKey = "";
  export let loading = false;
  export let errorMessage = "";
  export let path = "";
  export let defaultPath: string;
  export let pickerAvailable = false;
  export let onpathchange: (value: string) => void;
  export let onbrowse: (title: string, currentPath: string) => Promise<string>;
  export let step = 1;
  export let oncreate: (path: string, password: string) => void;
  export let onfinish: () => void;
  export let onlogin: () => void;
  export let onrestore: () => void;
  let password = "";
  let confirm = "";
  let visible = false;
  let saved = false;
  let verification = "";
  $: finalCharacters = recoveryKey.replace(/\s+/g, "").slice(-6);
  function finish() {
    if (
      !loading &&
      saved &&
      finalCharacters.length === 6 &&
      verification.replace(/\s+/g, "") === finalCharacters
    )
      onfinish();
  }
  function submit() {
    if (path.trim() && password && password === confirm) {
      const value = password;
      password = "";
      confirm = "";
      oncreate(path.trim(), value);
    }
  }
</script>

<main class="flow-page">
  <header class="page-header">
    <Brand /><span>Создание библиотеки · шаг {step} из 3</span><span
      class="local-label">Личная библиотека</span
    >
  </header>
  <div class="setup-layout">
    <aside class="setup-aside">
      <span class="eyebrow">{step === 1 ? "Начало" : "Доступ к архиву"}</span>
      <h1>
        {step === 1 ? "Ваша библиотека." : "Запасной ключ."}{" "}<br />{step ===
        1
          ? "На вашем устройстве."
          : "Только для вас."}
      </h1>
      <p class="muted">
        {step === 1
          ? "Храните книги в зашифрованном архиве и читайте без подключения к интернету."
          : "Этот код откроет библиотеку, если вы забудете пароль."}
      </p>
      {#if step === 1}<ol class="steps">
          <li class="current"><span>01</span> Папка и пароль</li>
          <li><span>02</span> Код восстановления</li>
          <li><span>03</span> Ваша первая книга</li>
        </ol>
        <div class="existing">
          <p class="muted">Уже есть библиотека?</p>
          <button class="text-button" on:click={onlogin}
            >Открыть библиотеку <Icon name="arrow" size={14} /></button
          ><button class="text-button" on:click={onrestore}
            >Восстановить копию <Icon name="arrow" size={14} /></button
          >
        </div>{:else}<div class="notice">
          <strong>Храните отдельно от библиотеки</strong>
          <p>
            Запишите код на бумаге и уберите в надёжное место. Он даёт доступ ко
            всем файлам архива.
          </p>
        </div>{/if}
    </aside>
    <section class="flow-card setup-card">
      {#if step === 1}<h1>Создайте библиотеку</h1>
        <p>Выберите папку и задайте пароль.</p>
        <form class="stack" on:submit|preventDefault={submit}>
          <PathField
            label="Папка библиотеки"
            bind:value={path}
            {defaultPath}
            {pickerAvailable}
            onchange={onpathchange}
            {onbrowse}
          />
          <div class="field">
            <label for="new-password">Пароль</label>
            <div class="input-wrap">
              <input
                id="new-password"
                bind:value={password}
                type={visible ? "text" : "password"}
                required
                autocomplete="off"
              /><button
                type="button"
                aria-pressed={visible}
                on:click={() => (visible = !visible)}
                >{visible ? "Скрыть" : "Показать"}</button
              >
            </div>
          </div>
          <label class="field"
            ><span>Повторите пароль</span><input
              bind:value={confirm}
              type={visible ? "text" : "password"}
              required
              autocomplete="off"
            />{#if confirm && password !== confirm}<small class="mismatch"
                >Пароли не совпадают</small
              >{/if}</label
          >
          <div class="notice">
            <strong>Пароль известен только вам</strong>
            <p>
              Без пароля или кода восстановления архив открыть нельзя. Сохраните
              код на следующем шаге.
            </p>
          </div>
          {#if errorMessage}<p class="error" role="alert">
              {errorMessage}
            </p>{/if}
          <div class="footer-row">
            <span class="muted">Шаг 1 из 3</span><button
              type="submit"
              class="primary"
              disabled={loading ||
                !path.trim() ||
                !password ||
                password !== confirm}
              >{loading ? "Создаём…" : "Продолжить"}
              <Icon name="arrow" size={16} /></button
            >
          </div>
        </form>
      {:else}<h1>Сохраните код восстановления</h1>
        <p>Создан на этом устройстве. Никуда не отправляется.</p>
        <form class="stack" on:submit|preventDefault={finish}>
          <div class="recovery-box">
            <span class="eyebrow">Ваш код восстановления</span><code
              >{recoveryKey}</code
            >
          </div>
          <label class="field"
            ><span>Для проверки введите последние 6 символов кода</span><input
              bind:value={verification}
              autocomplete="off"
              placeholder="Последние 6 символов"
              spellcheck={false}
              required
            /><small class="muted"
              >Это помогает убедиться, что вы записали код.</small
            ></label
          ><label class="check"
            ><input type="checkbox" bind:checked={saved} /> Я сохранил код отдельно
            от библиотеки</label
          >{#if errorMessage}<p class="error" role="alert">
              {errorMessage}
            </p>{/if}
          <div class="footer-row">
            <span class="muted">Шаг 2 из 3</span><button
              type="submit"
              class="primary"
              disabled={!saved ||
                verification.replace(/\s+/g, "") !== finalCharacters ||
                finalCharacters.length !== 6 ||
                loading}>К библиотеке <Icon name="arrow" size={16} /></button
            >
          </div>
        </form>{/if}
    </section>
  </div>
</main>

<style>
  .setup-layout {
    max-width: 1152px;
    display: grid;
    grid-template-columns: 280px minmax(0, 656px);
    gap: 80px;
    padding: 40px 24px 64px;
    margin: auto;
  }
  .setup-aside h1 {
    margin: 18px 0 22px;
    font-size: 30px;
  }
  .setup-aside > p {
    line-height: 1.65;
    font-size: 14px;
  }
  .steps {
    list-style: none;
    padding: 0;
    margin: 34px 0 28px;
    display: grid;
    gap: 18px;
    color: var(--muted);
  }
  .steps span {
    margin-right: 12px;
  }
  .steps .current {
    color: var(--text);
  }
  .existing {
    border-top: 1px solid var(--border);
    padding-top: 30px;
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  .setup-aside .notice {
    margin-top: 30px;
  }
  .setup-card {
    align-self: start;
  }
  .recovery-box {
    padding: 24px 22px;
    background: var(--input);
    border: 1px solid var(--border);
    border-radius: 16px;
  }
  .recovery-box code {
    display: block;
    font:
      23px/1.65 Onest,
      sans-serif;
    letter-spacing: 2px;
    overflow-wrap: anywhere;
    user-select: text;
    margin-top: 12px;
  }
  .mismatch {
    color: var(--error);
  }
  @media (max-width: 1100px) {
    .setup-layout {
      gap: 40px;
      grid-template-columns: 240px minmax(0, 1fr);
      padding-top: 24px;
    }
  }
  @media (max-width: 760px) {
    .setup-layout {
      grid-template-columns: 1fr;
      max-width: 650px;
      gap: 24px;
    }
    .setup-aside h1 {
      font-size: 27px;
    }
    .steps,
    .existing,
    .setup-aside .notice {
      display: none;
    }
    .setup-aside h1 br {
      display: none;
    }
    .setup-aside > p {
      margin-top: 8px;
    }
  }
</style>
