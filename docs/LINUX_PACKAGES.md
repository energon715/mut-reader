# Linux x86-64: пакеты для GitHub Releases

Версия **1.0.0-rc.1**, архитектура **x86-64 / amd64** (Intel/AMD).

## Что скачать

| Файл | Назначение |
| --- | --- |
| `mut-1.0.0-rc.1-linux-x86_64.AppImage` | Переносной запуск; GTK, WebKit 2.54.0 и их процессы внутри |
| `mut-1.0.0-rc.1-amd64.deb` | Установка через apt; GTK/WebKit берутся из системы |
| `mut-1.0.0-rc.1-linux-amd64.tar.gz` | Ручная установка бинарников с системными GTK/WebKit |
| `mut-1.0.0-rc.1-runtime-sources.tar.gz` | Соответствующие исходники библиотек AppImage; для установки приложения не нужны |
| `SHA256SUMS` | Контрольные суммы файлов релиза |
| `RELEASE-NOTES.md`, `BUILD-INFO.json` | Инструкция и сведения о сборке |

AppImage требует **glibc >= 2.39**, графическую Linux-сессию и системные
`bubblewrap`, `xdg-dbus-proxy`. Эти два помощника используются песочницей WebKit;
пакет не отключает её. Графические драйверы и libc предоставляет система.
Ubuntu 22.04 и Debian 12 не входят в целевой диапазон этой сборки.

DEB требует `libwebkit2gtk-4.1-0 >= 2.54.0` и зависимости, вычисленные
`dpkg-shlibdeps`. Проверена установка в Debian 13. В репозитории Ubuntu 24.04
на дату сборки ещё WebKit 2.52.6: для неё используйте AppImage. Не устанавливайте
чужие Debian-библиотеки поверх Ubuntu ради выполнения зависимости DEB.

## AppImage

```sh
chmod +x mut-1.0.0-rc.1-linux-x86_64.AppImage
./mut-1.0.0-rc.1-linux-x86_64.AppImage
```

Если помощники песочницы отсутствуют, установите их штатным пакетным менеджером:

```sh
# Debian / Ubuntu
sudo apt install bubblewrap xdg-dbus-proxy
# Arch
sudo pacman -S bubblewrap xdg-dbus-proxy
```

Если FUSE недоступен, runtime поддерживает запуск через извлечение:

```sh
./mut-1.0.0-rc.1-linux-x86_64.AppImage --appimage-extract-and-run
```

Для постоянного запуска без FUSE можно извлечь файл в отдельной папке и запустить
`squashfs-root/AppRun` после `--appimage-extract`. Эта папка содержит приложение,
а не пользовательский архив. Не извлекайте её поверх библиотеки книг.

Утилита восстановления также включена:

```sh
./mut-1.0.0-rc.1-linux-x86_64.AppImage --vaultctl --help
./mut-1.0.0-rc.1-linux-x86_64.AppImage --vaultctl verify /полный/путь/к/копии
```

## DEB

```sh
sudo apt install ./mut-1.0.0-rc.1-amd64.deb
mut
vaultctl --help
```

Пакет добавляет приложение в меню, иконку и утилиту `vaultctl`.
Удаление пакета: `sudo apt remove mut`. Пользовательская библиотека хранится
в выбранной папке и не относится к файлам установленного пакета.

## Проверка скачивания

Если скачаны все файлы, перечисленные в списке:

```sh
sha256sum -c SHA256SUMS
```

Для одного файла сравните `sha256sum имя-файла` с его строкой в `SHA256SUMS`.

## Сборка у автора

Нужны Python 3.12+, `bubblewrap`, `unshare`, `newuidmap`/`newgidmap` и диапазоны
пользователя в `/etc/subuid` и `/etc/subgid`. Сборка работает от обычного пользователя;
SDK и пакеты устанавливаются в игнорируемый `build/.release-debian-rootfs`.
Хостовая ОС не обновляется. В Ubuntu AppArmor может ограничивать user namespaces;
workflow меняет этот параметр только на временном GitHub runner.

```sh
python3 scripts/build-linux-release.py
```

Скрипт проверяет SHA-256 образа Debian, Go, Node и инструментов из
`packaging/tools.lock.json`, проверяет минимальную версию WebKit, запускает
frontend/Go проверки, audit, govulncheck, сборку, упаковку и проверку артефактов.
APT получает актуальные подписанные пакеты; точные версии сохраняются в AppImage
и `BUILD-INFO.json`. Это повторяемая процедура, а не обещание побайтово одинаковой
сборки при обновившихся репозиториях APT.

GTK/WebKit и модули динамически связаны; соответствующие исходники Debian
и лицензии поставляются с релизом. Собственный небольшой GLib spawn hook
перенаправляет абсолютные пути процессов WebKit внутрь AppImage, сохраняя
аргументы песочницы. Его исходник — `packaging/webkit-relocate.c`; он также лежит
в `usr/share/doc/mut/licenses/system` внутри образа. WebKit не патчится.

## GitHub Releases

Workflow **Linux x86-64 release packages** запускается вручную в Actions
или по тегу `v*`; он сохраняет файлы в Actions artifact и не публикует релиз.

1. Проверьте пакет и скопируйте все файлы из `dist/` в одну папку.
2. Создайте GitHub Release для `v1.0.0-rc.1` с целевой веткой `docs/v1.0.0-plan`.
3. Отметьте **Set as a pre-release**.
4. Вставьте текст из `dist/RELEASE-NOTES.md` и приложите файлы таблицы выше,
   включая архив соответствующих исходников runtime.
5. Скачайте опубликованный пакет и сверяйте его контрольную сумму перед проверкой.

Финальный тег `v1.0.0` — после критериев из `docs/RELEASE.md`.

## Выполненные проверки

- AppImage runtime: полный цикл PDF/EPUB/FB2, закладки, копия/восстановление,
  настройки, прокрутка и перезапуск в Arch x86-64.
- Тот же runtime: создание библиотеки, три читалки, закладки, резервирование
  и восстановление в файловой среде Ubuntu 24.04 с glibc 2.39. X11-сервер
  предоставлял хост; это не полноценная проверка отдельной машины или Wayland.
- Установка DEB через apt в Debian 13, метаданные amd64, работа CLI.
- Процесс страницы WebKit использует свой executable, отдельные пространства
  имён, `NoNewPrivs=1`, seccomp. В пакете не включён remote inspector.
- Проверка самого AppImage после извлечения: ELF64, build IDs production
  бинарников, процессы WebKit, launcher, лицензии, CLI и SHA-256.

Fedora, чистый Wayland, ARM64 и Windows здесь не объявляются проверенными.
