<script lang="ts">
  import { tick } from "svelte";
  import { fly } from "svelte/transition";
  import Icon from "./Icon.svelte";
  export let id: string;
  export let label: string;
  export let compact = false;
  export let value: string | number;
  export let options: { value: string | number; label: string; description?: string; sampleSize?: number }[];
  export let onchange: (value: string | number) => void;
  let open = false;
  let active = 0;
  let above = false;
  let menuHeight = 260;
  let root: HTMLDivElement;
  let trigger: HTMLButtonElement;
  let list: HTMLDivElement;
  $: selected = options.find(option => option.value === value);
  function reduced() { return window.matchMedia("(prefers-reduced-motion: reduce)").matches; }
  async function show() {
    const box = trigger.getBoundingClientRect();
    const dialog = trigger.closest("dialog")?.getBoundingClientRect();
    const bottom = Math.min(innerHeight - 12, dialog?.bottom ?? innerHeight - 12);
    const top = Math.max(12, dialog?.top ?? 12);
    const below = bottom - box.bottom - 10;
    const before = box.top - top - 10;
    above = below < 220 && before > below;
    menuHeight = Math.max(60, Math.min(300, above ? before : below));
    active = Math.max(0, options.findIndex(option => option.value === value));
    open = true;
    await tick();
    list?.focus({ preventScroll: true });
    scrollActive();
  }
  function close(focus = false) {
    open = false;
    if (focus) trigger?.focus({ preventScroll: true });
  }
  function choose(index: number) {
    const option = options[index];
    if (!option) return;
    onchange(option.value);
    close(true);
  }
  function scrollActive() {
    void tick().then(() => list?.querySelector(`[data-index="${active}"]`)?.scrollIntoView({ block: "nearest" }));
  }
  function keys(event: KeyboardEvent) {
    if (["ArrowDown", "ArrowUp", "Home", "End", "Enter", " ", "Escape"].includes(event.key)) {
      event.preventDefault(); event.stopPropagation();
    }
    if (event.key === "ArrowDown") active = (active + 1) % options.length;
    if (event.key === "ArrowUp") active = (active - 1 + options.length) % options.length;
    if (event.key === "Home") active = 0;
    if (event.key === "End") active = options.length - 1;
    if (event.key === "Enter" || event.key === " ") choose(active);
    if (event.key === "Escape") close(true);
    if (event.key === "Tab") close();
    if (open) scrollActive();
  }
</script>

<svelte:window on:pointerdown={(event) => { if (open && !root.contains(event.target as Node)) close(); }} on:resize={() => close()} />
<div class="choice" class:compact bind:this={root}>
  <button {id} class="choice-trigger" class:expanded={open} bind:this={trigger}
    type="button" aria-label={label} aria-haspopup="listbox" aria-expanded={open} aria-controls={open ? id + "-options" : undefined}
    on:click={() => open ? close(true) : show()}
    on:keydown={(event) => { if (event.key === "ArrowDown" || event.key === "ArrowUp") { event.preventDefault(); void show(); } }}>
    <span>{selected?.label || value}</span><span class="chevron"><Icon name="down" size={16} /></span>
  </button>
  {#if open}<div class="choice-menu" class:above bind:this={list} id={id + "-options"}
      role="listbox" tabindex="-1" aria-label={label} aria-activedescendant={id + "-option-" + active}
      style:max-height={menuHeight + "px"} transition:fly={{ y: above ? 5 : -5, duration: reduced() ? 0 : 140 }} on:keydown={keys}
      on:focusout={(event) => { if (!root.contains(event.relatedTarget as Node)) close(); }}>
      {#each options as option, index}<button type="button" role="option" aria-selected={option.value === value}
          id={id + "-option-" + index} data-index={index} tabindex="-1"
          class:highlighted={index === active} class:selected={option.value === value}
          on:click={() => choose(index)}>
          {#if option.sampleSize}<span class="type-sample" style:font-size={Math.min(24, option.sampleSize) + "px"}>Aa</span>{/if}
          <span class="option-copy"><strong>{option.label}</strong>{#if option.description}<small>{option.description}</small>{/if}</span>
          <span class="selected-icon">{#if option.value === value}<Icon name="check" size={16} />{/if}</span>
        </button>{/each}
    </div>{/if}
</div>

<style>
  .choice { position: relative; flex-shrink: 0; width: 160px; }
  .choice-trigger { width: 100%; min-height: 42px; display: flex; justify-content: space-between; gap: 12px; padding: 10px 12px; border: 1px solid var(--border); border-radius: 12px; background: var(--raised); color: var(--text); font-size: 12px; }
  .choice.compact { width: 200px; }
  .compact .choice-trigger { min-height: 28px; padding: 4px 8px; border-color: transparent; background: transparent; font-size: 11px; }
  .choice-trigger.expanded { border-color: var(--accent); background: var(--active); }
  .chevron { display: flex; color: var(--muted); transition: transform 160ms ease; }
  .expanded .chevron { transform: rotate(180deg); }
  .choice-menu { position: absolute; z-index: 30; top: calc(100% + 7px); right: 0; width: 250px; max-width: calc(100vw - 48px); overflow-y: auto; overscroll-behavior: contain; padding: 6px; border: 1px solid var(--border); border-radius: 16px; color: var(--text); background: var(--raised); box-shadow: 0 14px 38px #0005; }
  .choice-menu.above { top: auto; bottom: calc(100% + 7px); }
  .choice-menu:focus { outline: none; }
  .choice-menu button { width: 100%; min-height: 44px; padding: 10px; gap: 12px; justify-content: flex-start; text-align: left; border-radius: 10px; }
  .choice-menu button.highlighted, .choice-menu button:hover { background: var(--active); }
  .option-copy { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 3px; }
  .option-copy strong { font-size: 12px; font-weight: 500; }
  .option-copy small { font-size: 10px; line-height: 1.4; color: var(--muted); }
  .selected-icon { width: 18px; color: var(--accent); }
  .type-sample { width: 35px; font-family: Literata, Georgia, serif; color: var(--muted); }
  @media (max-width: 650px) { .choice { width: 136px; } }
  @media (prefers-reduced-motion: reduce) { .chevron { transition: none; } }
</style>
