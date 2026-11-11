[Русский](README.md) · **English**

<div align="center">
  <img src="design/logo-peek.svg" width="96" alt="DeepMole">
  <h1>DeepMole</h1>
  <p>Finds a working DPI bypass strategy on top of zapret by itself, remembers it per network and keeps it alive.</p>
</div>

DeepMole removes the main pain of zapret: no more trying `general.bat`, `ALT`, `FAKE TLS` one by one and checking by eye what worked. The program splits strategies into blocks (YouTube, Discord, the general list), verifies each block with real downloads and assembles a configuration from the variants that passed.

## Features

- Strategy search takes 10–40 seconds on a new network; a known strategy starts instantly.
- 28 starting presets from the Flowseal bundle plus an in-house generator built on zapret parameters: desync methods, fooling, split-pos, seqovl patterns, fake TLS with modifiers, repeats, TTL and autottl, wssize. Together that is more than two hundred thousand unique recipes, explored one parameter at a time from the best partially working variant.
- Independent search for YouTube and Discord: if one service needs one method and the other needs another, both are combined into a single winws run.
- DNS hijack protection: when the ISP's DNS returns no address for a site, probes resolve it through DoH or public DNS, and the program switches Windows to whichever resolver answers on this network (Google, Cloudflare, Quad9 or AdGuard, encrypted where possible). The previous DNS settings are restored on exit.
- Per-network memory: the fingerprint is built from the gateway, its MAC and the SSID, so the strategy is saved and brought up immediately on the next start.
- Watchdog: checks key sites every minute and re-searches when the strategy breaks or the network changes.
- Status window with a search journal, tray icon, buttons "Re-search", "Disable", "Autostart".
- winws is guaranteed to stop on exit, even if DeepMole is killed.

## Installation

Download `DeepMole-<version>-setup.exe` from the [releases page](../../releases/latest) and run it. The installer ships DeepMole together with winws, WinDivert and the lists, adds a Start menu shortcut, optionally enables start at sign-in and launches the program right away. No manual steps are needed.

Uninstalling through "Apps & features" stops the bypass and removes the autostart task and the WinDivert driver.

Portable builds are available on the same page:

- `DeepMole-<version>-full-windows-x64.zip` — with winws, WinDivert and the lists; unpack and run `deepmole.exe`;
- `DeepMole-<version>-windows-x64.zip` — DeepMole only, if you already have zapret.

The program asks for administrator rights at start: the WinDivert driver needs them. If you took the archive without zapret, put the `bin` and `lists` folders from [zapret-win-bundle](https://github.com/bol-van/zapret-win-bundle) or the [Flowseal/zapret-discord-youtube](https://github.com/Flowseal/zapret-discord-youtube) bundle into a `zapret` folder next to `deepmole.exe`, or set the path in `deepmole.json`:

```json
{
  "zapret_dir": "C:/zapret",
  "game_filter": "off",
  "probe_timeout_ms": 3000,
  "watchdog_interval_s": 60,
  "max_candidates": 30,
  "secure_dns": "auto",
  "language": ""
}
```

`game_filter` accepts `off`, `tcp`, `udp`, `all` and enables the bypass for ports 1024–65535, like Game Filter in the Flowseal bundle. `max_candidates` limits the variations tried after the presets: all 28 presets are always checked, then at most that many neighbouring recipes. `secure_dns` accepts `auto` and `off`; the same switch is in the window and the tray menu, and the choice made there is remembered. `language` accepts `ru` or `en`; when empty, the Windows display language is used.

Stop other zapret instances, GoodbyeDPI and similar tools before running, otherwise the checks will give wrong results.

## How it works

1. On start, the network fingerprint is computed.
2. If a saved strategy exists for this network, it is started immediately and verified in the background.
3. If there is no saved strategy or it stopped working, a search begins: first the proven presets, then variations of the best one, one parameter at a time (TTL and autottl, fooling, fake TLS, split-pos, desync method, repeats, pattern, wssize). The search stops at the first variant that passes all probes for the block.
4. The YouTube and Discord blocks are searched independently and merged into a single winws command line.
5. Every candidate is verified by downloading real pages and images rather than a single HEAD request, to catch connections that stall after the first kilobytes. Control sites that are not blocked tell "internet is down" apart from "the strategy broke".
6. When the system DNS returns no address for a probed site, the probe resolves it by itself: first through DoH (Google, Cloudflare, AdGuard), then through plain DNS to public servers. The search therefore scores only the DPI bypass, not DNS failures. On the first such hijack the program switches the active network adapters to the resolver that answered and enables DoH in Windows when it is reachable on this network; the settings are restored on exit, when the bypass is disabled or when the switch is turned off.

## Window and tray

A status window opens on start: bypass state, current network, strategies for YouTube and Discord, a DNS line (system, secure or hijacked by the ISP) and a search journal showing how many probes each candidate passed. Closing the window minimizes the program to the tray; clicking the tray icon opens it again.

- **Re-search**: start the search for the current network again.
- **Disable / Enable**: stop or bring up the bypass without closing the program.
- **Secure DNS**: allow the program to switch the Windows DNS when the ISP hijacks it. Turning the switch off restores the previous settings right away.
- **Autostart**: a scheduled task that runs as administrator at sign-in.
- **Log**: `deepmole.log` next to the program, with every probe and candidate.
- **Exit** in the tray menu stops winws and closes the program.

Saved strategies and the DNS switch live in `deepmole-state.json`; a backup of the DNS settings is kept in `deepmole-dns.json` while they are changed. If the program is killed before restoring them, it restores them on the next start.

## Building from source

Go 1.22+ is required.

```powershell
.\build.ps1 -Version 1.0.2
```

The script generates the icon, the exe resources (icon and a manifest requesting administrator rights) and builds `dist/deepmole.exe` (no console) and `dist/deepmole-cli.exe` (console output). `release.ps1` packs the zip archives, and with `-ZapretDir` also the full package and the installer (Inno Setup 6 required). GitHub releases are built automatically on `v*` tags.

Command line flags:

```
deepmole-cli.exe -cli            run in the console with the log
deepmole-cli.exe -dry            print winws arguments and run probes without starting the bypass
deepmole-cli.exe -ui             show the window with sample data, without starting the bypass
deepmole-cli.exe -zapret <dir>   path to the folder with bin and lists
deepmole-cli.exe -game all       enable Game Filter
```

Tests:

```powershell
go test ./...
```

## Antivirus

WinDivert may be flagged by antivirus software as a potentially unwanted program. It is a traffic interception driver; the bypass does not work without it. Add the folder to the exclusions.

## License and credits

DeepMole is distributed under the MIT license.

- [bol-van/zapret](https://github.com/bol-van/zapret) — the bypass tool itself and winws.
- [Flowseal/zapret-discord-youtube](https://github.com/Flowseal/zapret-discord-youtube) — the strategies and lists the presets are built from.
