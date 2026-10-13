<script lang="ts">
  import { onMount } from "svelte";
  import * as pdfjs from "pdfjs-dist/legacy/build/pdf.mjs";
  import workerUrl from "pdfjs-dist/legacy/build/pdf.worker.mjs?url";
  import type { PDFDocumentProxy, RenderTask } from "pdfjs-dist";
  export let data: string;
  export let page = 1;
  export let zoom = 100;
  export let onloaded: (
    total: number,
    outline: { title: string; page: number }[],
  ) => void;
  let canvas: HTMLCanvasElement;
  let wrapper: HTMLDivElement;
  let textContainer: HTMLDivElement;
  let pdf: PDFDocumentProxy | null = null;
  let renderTask: RenderTask | null = null;
  let textLayer: pdfjs.TextLayer | null = null;
  let loading = true;
  let error = "";
  let width = 750;
  let disposed = false;
  let renderVersion = 0;
  pdfjs.GlobalWorkerOptions.workerSrc = workerUrl;
  onMount(() => {
    const observer = new ResizeObserver(
      ([entry]) => (width = Math.max(200, entry!.contentRect.width)),
    );
    observer.observe(wrapper);
    const bytes = Uint8Array.from(atob(data), (c) => c.charCodeAt(0));
    const task = pdfjs.getDocument({
      data: bytes,
      enableXfa: false,
      isOffscreenCanvasSupported: false,
      isImageDecoderSupported: false,
      maxImageSize: 16000000,
      canvasMaxAreaInBytes: 32000000,
      cMapUrl: "/pdfjs/cmaps/",
      cMapPacked: true,
      standardFontDataUrl: "/pdfjs/standard_fonts/",
      wasmUrl: "/pdfjs/wasm/",
      iccUrl: "/pdfjs/iccs/",
      verbosity: 0,
    });
    void task.promise
      .then(async (value) => {
        if (disposed) {
          await task.destroy();
          return;
        }
        const outline: { title: string; page: number }[] = [];
        const items = await value.getOutline();
        for (const item of items || []) {
          try {
            const dest =
              typeof item.dest === "string"
                ? await value.getDestination(item.dest)
                : item.dest;
            if (dest) {
              const index =
                typeof dest[0] === "number"
                  ? dest[0]
                  : await value.getPageIndex(dest[0]);
              outline.push({ title: item.title, page: index + 1 });
            }
          } catch {}
        }
        if (disposed) return;
        pdf = value;
        loading = false;
        onloaded(value.numPages, outline);
      })
      .catch(() => {
        if (!disposed) {
          loading = false;
          error =
            "Не удалось прочитать PDF. Файл может быть повреждён или защищён паролем.";
        }
      });
    return () => {
      disposed = true;
      renderVersion++;
      observer.disconnect();
      renderTask?.cancel();
      textLayer?.cancel();
      void task.destroy();
    };
  });
  $: if (pdf && canvas && textContainer && width)
    void render(pdf, page, zoom, width);
  async function render(
    document: PDFDocumentProxy,
    index: number,
    scalePercent: number,
    availableWidth: number,
  ) {
    // Local references prevent canvas mutations from invalidating this effect.
    const targetCanvas = canvas;
    const targetLayer = textContainer;
    const version = ++renderVersion;
    renderTask?.cancel();
    textLayer?.cancel();
    try {
      if (renderTask) await renderTask.promise.catch(() => {});
      if (version !== renderVersion || disposed) return;
      const source = await document.getPage(
        Math.max(1, Math.min(document.numPages, index)),
      );
      if (version !== renderVersion || disposed) return;
      const initial = source.getViewport({ scale: 1 });
      let scale = ((availableWidth / initial.width) * scalePercent) / 100;
      const ratio = Math.min(window.devicePixelRatio || 1, 2);
      if (
        initial.width * initial.height * scale * scale * ratio * ratio >
        16000000
      )
        scale = Math.sqrt(
          16000000 / (initial.width * initial.height * ratio * ratio),
        );
      const viewport = source.getViewport({ scale });
      targetCanvas.width = Math.floor(viewport.width * ratio);
      targetCanvas.height = Math.floor(viewport.height * ratio);
      targetCanvas.style.width = viewport.width + "px";
      targetCanvas.style.height = viewport.height + "px";
      targetLayer.replaceChildren();
      targetLayer.style.width = viewport.width + "px";
      targetLayer.style.height = viewport.height + "px";
      targetLayer.style.setProperty("--scale-factor", String(scale));
      targetLayer.style.setProperty("--total-scale-factor", String(scale));
      renderTask = source.render({
        canvas: targetCanvas,
        viewport,
        transform: ratio === 1 ? undefined : [ratio, 0, 0, ratio, 0, 0],
      });
      await renderTask.promise;
      if (version !== renderVersion || disposed) return;
      const content = await source.getTextContent();
      if (version !== renderVersion || disposed) return;
      textLayer = new pdfjs.TextLayer({
        textContentSource: content,
        container: targetLayer,
        viewport,
      });
      await textLayer.render();
    } catch (err) {
      if (
        !disposed &&
        version === renderVersion &&
        String(err).indexOf("cancel") < 0
      )
        error = "Не удалось отобразить страницу PDF.";
    }
  }
</script>

<div class="pdf-wrapper" bind:this={wrapper}>
  {#if loading}<p class="muted" role="status">
      Открываем PDF…
    </p>{/if}{#if error}<p class="error" role="alert">{error}</p>{/if}
  <div class="pdf-sheet">
    <canvas bind:this={canvas} aria-label={"Страница " + page}></canvas>
    <div class="textLayer" bind:this={textContainer}></div>
  </div>
</div>

<style>
  .pdf-wrapper {
    width: 100%;
    min-height: 200px;
  }
  .pdf-sheet {
    position: relative;
    width: max-content;
    margin: auto;
  }
  canvas {
    display: block;
  }
  .textLayer {
    position: absolute;
    left: 0;
    top: 0;
    overflow: clip;
    --min-font-size: 1;
    --text-scale-factor: calc(var(--total-scale-factor) * var(--min-font-size));
    --min-font-size-inv: calc(1 / var(--min-font-size));
    line-height: 1;
    text-align: initial;
    transform-origin: 0 0;
    color: transparent;
    user-select: text;
    cursor: text;
  }
  .textLayer :global(span),
  .textLayer :global(br) {
    color: transparent;
    position: absolute;
    white-space: pre;
    transform-origin: 0% 0%;
    --font-height: 0;
    --scale-x: 1;
    --rotate: 0deg;
    font-size: calc(var(--text-scale-factor) * var(--font-height));
    transform: rotate(var(--rotate)) scaleX(var(--scale-x)) scale(var(--min-font-size-inv));
  }
  .textLayer :global(::selection) {
    background: #5680be66;
    color: transparent;
  }
  .textLayer :global(span.markedContent) {
    display: contents;
  }
</style>
