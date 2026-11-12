**Русский** · [English](#english)

# Политика конфиденциальности

DeepMole не собирает телеметрию, не ведёт учётных записей и не передаёт сведения о пользователе разработчикам или третьим лицам. Программа работает полностью на компьютере пользователя.

## Какие сетевые запросы делает программа

- **Проверочные запросы.** Чтобы подобрать и контролировать стратегию обхода, программа загружает небольшие страницы и файлы с сайтов проверяемых сервисов (YouTube, Discord, Telegram) и двух контрольных сайтов (ya.ru, vk.com). Это обычные HTTPS-запросы без идентификаторов пользователя. Без них программа не может выполнять свою функцию.
- **Запасные DNS-резолверы.** Если системный DNS не отдаёт адрес сайта, имя разрешается через DoH (Google, Cloudflare, AdGuard) или обычный DNS к публичным серверам (Google, Cloudflare, Quad9, AdGuard). Передаётся только имя проверяемого сайта. Переключение системного DNS можно запретить переключателем «Защищённый DNS» в окне или ключом `secure_dns: "off"`.
- **Проверка обновлений.** При запуске и раз в сутки программа запрашивает у GitHub сведения о последнем релизе проекта. GitHub получает стандартные данные HTTP-запроса и строку `DeepMole/<версия>`. Отключается ключом `auto_update: "off"`.

Ничего, кроме перечисленного, программа в сеть не отправляет.

## Что хранится на компьютере

Рядом с программой или в `%ProgramData%\DeepMole`: журнал `deepmole.log` с результатами проверок, `deepmole-state.json` с подобранными стратегиями и отпечатками сетей (шлюз, его MAC-адрес, имя Wi-Fi), `deepmole-dns.json` с резервной копией настроек DNS на время их изменения. Файлы удаляются вместе с программой.

## Изменения в системе

Программа запускает winws с драйвером WinDivert для обработки сетевых пакетов, при подмене DNS провайдером меняет DNS-серверы активных сетевых адаптеров и включает DoH, а по задаче автозапуска добавляет запись в планировщик Windows. Все изменения показываются в журнале окна и отменяются при выходе, при выключении обхода или при удалении программы.

---

<a name="english"></a>

# Privacy policy

DeepMole collects no telemetry, has no accounts and sends no information about the user to the developers or any third party. The program runs entirely on the user's computer.

## Network requests the program makes

- **Probe requests.** To find and maintain a bypass strategy the program downloads small pages and files from the probed services (YouTube, Discord, Telegram) and two control sites (ya.ru, vk.com). These are ordinary HTTPS requests without user identifiers. The program cannot do its job without them.
- **Fallback DNS resolvers.** When the system DNS returns no address for a site, the name is resolved through DoH (Google, Cloudflare, AdGuard) or plain DNS to public servers (Google, Cloudflare, Quad9, AdGuard). Only the probed site name is sent. Switching the system DNS can be prohibited with the "Secure DNS" switch in the window or `secure_dns: "off"`.
- **Update check.** On start and once a day the program asks GitHub for the project's latest release. GitHub receives the standard HTTP request data and the string `DeepMole/<version>`. Disabled with `auto_update: "off"`.

Nothing else is sent over the network.

## What is stored on the computer

Next to the program or in `%ProgramData%\DeepMole`: the log `deepmole.log` with probe results, `deepmole-state.json` with the found strategies and network fingerprints (gateway, its MAC address, Wi-Fi name), and `deepmole-dns.json` with a backup of the DNS settings while they are changed. The files are removed together with the program.

## System changes

The program runs winws with the WinDivert driver to process network packets, changes the DNS servers of the active network adapters and enables DoH when the ISP hijacks DNS, and adds a Windows Task Scheduler entry when autostart is enabled. All changes are shown in the window's journal and reverted on exit, when the bypass is disabled or when the program is uninstalled.
