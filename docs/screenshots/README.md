# Screenshot sources

The root README uses these images with relative paths so the gallery is part of `main` and remains available without fetching another branch.

## Desktop

Captured on 16 October 2026 from the native Wails/WebKit Linux app built from the desktop implementation on `docs/v1.0.0-plan` (application source at `d203590`). The app ran with its own temporary vault and cache under `/tmp/mut-readme-desktop`, using eight generated FB2/EPUB demonstration books and embedded covers. The books use fictional titles/authors and original demonstration text; no personal library was opened.

All four captures are 1360 × 900 PNGs taken from the WebKit viewport using `Page.snapshotRect`. Screens show the actual library, FB2 reader with a bookmark, font-size menu, and light theme. The pictures have not been composited with mock interface elements.

## Mobile

Copied without alteration from the local `app/mobile` branch at `fc0a89c`, from the Russian Flutter UI gallery under `pictures/rustore/`:

| README image | Original file |
| --- | --- |
| `mobile/library.png` | `pictures/rustore/01-library-ru.png` |
| `mobile/book-details.png` | `pictures/rustore/02-book-details-ru.png` |
| `mobile/reader.png` | `pictures/rustore/03-reader-ru.png` |
| `mobile/appearance.png` | `pictures/rustore/04-appearance-ru.png` |

These are 1080 × 1920 PNG captures of Flutter widgets in a test environment, using a temporary encrypted demonstration library. They are not device screenshots. The original generator is `mobile/tool/capture_screenshots.dart` on the mobile branch, with `MUT_RUSTORE_SCREENSHOTS=true`.
