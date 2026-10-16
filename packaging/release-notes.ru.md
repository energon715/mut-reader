# mut 1.0.0-rc.1 — Linux x86-64

Кандидат выпуска локальной зашифрованной библиотеки: PDF, EPUB, FB2,
полки, поиск, позиции чтения, закладки, резервные копии и восстановление.

**Выбор пакета:** AppImage для Linux с glibc >= 2.39 (включая Ubuntu 24.04),
DEB для Debian 13 и систем с WebKitGTK >= 2.54.0. Архитектура x86-64 / amd64.

AppImage:

```sh
chmod +x mut-1.0.0-rc.1-linux-x86_64.AppImage
./mut-1.0.0-rc.1-linux-x86_64.AppImage
```

Нужны системные `bubblewrap` и `xdg-dbus-proxy` для песочницы WebKit.
В Debian/Ubuntu: `sudo apt install bubblewrap xdg-dbus-proxy`.
Без FUSE: добавьте `--appimage-extract-and-run` к команде запуска.
GTK и WebKit 2.54.0 включены; графические драйверы предоставляет система.

DEB:

```sh
sudo apt install ./mut-1.0.0-rc.1-amd64.deb
mut
```

В Ubuntu 24.04 системный WebKit пока ниже требуемого: выбирайте AppImage.
Для проверки скачанного файла сравните `sha256sum имя-файла` с `SHA256SUMS`.

AppImage также содержит CLI: `./mut-1.0.0-rc.1-linux-x86_64.AppImage --vaultctl --help`.
Архив `runtime-sources.tar.gz` содержит исходники включённых библиотек и не нужен
для установки. `BUILD-INFO.json` фиксирует версии компонентов.

Проверены полный цикл чтения и восстановления на Arch и в Ubuntu 24.04
(X11, изолированная файловая среда), установка DEB в Debian 13, тесты Go
с race detector и проверка фронтенда. Wayland и Fedora ещё требуют испытаний.

Версия остаётся предварительной: независимое ревью криптографии и испытания
аварийного сохранения входят в критерии финальной 1.0. При создании библиотеки
сохраните полный recovery-код отдельно от архива.
