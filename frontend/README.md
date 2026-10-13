# Desktop UI

Svelte + TypeScript, Wails v2. `npm ci`, `npm run check`, `npm run build`.
`npm run dev` открывает дизайн с демонстрационными данными; `?preview=reader`
показывает макет читалки. Работа с архивом доступна в собранном Wails-приложении.

PDF.js и worker, CMaps, шрифты, WASM и ICC поставляются локально. EPUB/FB2
приходят из Go как ограниченная структура текста и изображений; HTML книг
не вставляется в DOM. Production CSP ограничивает ресурсы локальным origin.

[Контракт моста](BRIDGE.md), [сборка](../README.md),
[результаты и ограничения](../docs/RELEASE.md).
