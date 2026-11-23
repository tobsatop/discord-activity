# YouTube Music → Discord

[![CI](https://github.com/tobsatop/discord-activity/actions/workflows/ci.yml/badge.svg)](https://github.com/tobsatop/discord-activity/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

[ภาษาไทย](README.md) | **English**

Show the song you're playing on YouTube Music (in Chrome) as your Discord status: **"Listening to …"**.
Works on **Windows** and **macOS**.

<p><img src="docs/discord-status.png" alt="Example Discord status: listening to YouTube Music with song, artist, album and a time bar"></p>

## Features

- Shows the song title, artist, album art, a time bar and a "Listen on YouTube Music" button
- The status clears by itself when you pause, during ads, or when you close the tab or Chrome
- Nothing to keep running: Chrome starts the helper by itself when YouTube Music is open
- Works with **music.youtube.com** only (not regular youtube.com)

## Requirements

| What | Notes |
|---|---|
| Google Chrome on your computer | Tested on Windows and macOS |
| Discord desktop app | Must be the app; Discord in a browser won't work |
| Node.js **22.18 or newer** | **24 LTS** recommended, from https://nodejs.org |
| A Discord Application ID | Free; you create it in step 1 |

> The installer, the updater and the toolbar tooltips show messages in Thai. The ones you need are translated below.

## Install (once per computer)

### Step 1 — Create a Discord application (once; reuse it on every computer)

1. Go to https://discord.com/developers/applications and log in with your Discord account
2. Click **New Application** → name it `YouTube Music` → accept the terms → **Create**
   The name appears on the status card, e.g. "Listening to YouTube Music"
3. On **General Information**, click **Copy** next to **Application ID** (a 17–20 digit number) and keep it

Nothing else to set up: no bot, no images.

### Step 2 — Download the code

Pick one:

- **With git:** `git clone https://github.com/tobsatop/discord-activity.git`
- **Without git:** on GitHub click **Code** → **Download ZIP** → **extract the ZIP first** (Windows: right-click → Extract All)

> Put the folder somewhere permanent before installing. If you move or rename it later, redo steps 3 and 4.

### Step 3 — Install the helper

<details open>
<summary><b>Windows</b></summary>

1. Install Node.js 24 LTS from https://nodejs.org (the `.msi`; click Next to the end, the defaults are fine)
2. Open the project folder in File Explorer → **right-click an empty area** → **Open in Terminal**
   (Windows 10: hold `Shift` and right-click → **Open PowerShell window here**)
3. Check Node:
   ```
   node -v
   ```
   It must be `v22.18` or newer. If it says `node is not recognized`, close and reopen the terminal (or restart the PC)
4. Run the installer:
   ```
   node scripts\install.ts
   ```
   **Don't use `npm run setup` in PowerShell.** PowerShell blocks npm (error: *running scripts is disabled on this system*).
   In Command Prompt (cmd), `npm run setup` works.
5. Paste your Application ID (right-click or `Ctrl + V`) → Enter
6. **"ลงทะเบียนโปรแกรมเสริมกับ Chrome, Edge แล้ว"** ("registered the helper with Chrome, Edge") means it worked

The installer keeps its launcher in `%LOCALAPPDATA%\ytm-discord-presence\` and registers it in your user's registry (HKCU). No administrator rights needed.

</details>

<details open>
<summary><b>macOS</b></summary>

1. Install Node.js 24 LTS from https://nodejs.org (the `.pkg`) or with `brew install node`
2. Open Chrome at least once first (the installer looks for Chrome's folder)
3. Open **Terminal** (press `Cmd + Space`, type Terminal)
4. Type `cd ` (with a space after it), **drag the project folder** into the Terminal window → Enter
5. Check Node:
   ```
   node -v
   ```
   It must be `v22.18` or newer
6. Run the installer:
   ```
   npm run setup
   ```
7. Paste your Application ID (`Cmd + V`) → Enter
8. **"ลงทะเบียนโปรแกรมเสริมกับ Chrome … แล้ว"** ("registered the helper with Chrome …") means it worked

The installer keeps its launcher in `~/Library/Application Support/ytm-discord-presence/`, so the project can live in Documents.
If you use nvm and remove or switch Node versions, run the installer again.

</details>

### Step 4 — Load the extension in Chrome

1. Type `chrome://extensions` in the address bar
2. Turn on **Developer mode** (top right)
3. Click **Load unpacked** → choose the **`extension`** folder inside the project
   (not the whole project folder, which gives *Manifest file is missing*)
4. Click the puzzle icon in Chrome's toolbar → pin 📌 **YouTube Music → Discord** so you can see the status icon
5. If a YouTube Music tab was already open, **refresh it once**

### Step 5 — Try it

1. Open the Discord app and log in (Windows: **don't run Discord as administrator**)
2. Discord → User Settings → **Activity Privacy** → turn on activity sharing
   (the option is called something like *Share my activity* or *Display current activity as a status message*)
3. Open https://music.youtube.com and play a song
4. Within a few seconds the extension icon turns **red** and your Discord profile shows "Listening to <song>"

## Daily use

**You don't run the installer every time you start your computer.** Steps 1–5 are once per computer.

After a restart, just:

1. Open the Discord app
2. Open Chrome and play music on music.youtube.com

Chrome starts the helper when a YouTube Music tab is open and stops it when you close the tab.

To make Discord start with your computer:
- Windows: Discord → User Settings → Windows Settings → turn on **Open Discord**
- macOS: System Settings → General → Login Items → add Discord

Run the installer again (step 3) only when you:
- move or rename the project folder
- remove or move Node, or update Node with nvm / Homebrew
  (updating with the installer from nodejs.org is fine)
- change the Application ID (see "Change the Application ID")

## What the icon means

Hover over the icon for details.

| Icon | Meaning |
|---|---|
| <img src="extension/icons/icon-32.png" width="20" alt="red"> Red | Showing in Discord |
| <img src="extension/icons/idle-32.png" width="20" alt="grey"> Grey | No YouTube Music tab, or nothing playing |
| <img src="extension/icons/warn-32.png" width="20" alt="amber dot"> Grey + amber dot | The Discord app isn't running |
| <img src="extension/icons/error-32.png" width="20" alt="red dot"> Grey + red dot | The helper can't start, or Discord refused the Application ID |

## Troubleshooting

| Symptom | Fix |
|---|---|
| Red dot, "เปิดโปรแกรมเสริมไม่ได้" (can't start the helper) | Step 3 not done, the folder moved, or Node was removed → redo step 3, then wait about 30 seconds |
| Red dot, "Discord ไม่รับการเชื่อมต่อ … Application ID" (Discord refused the connection) | Wrong Application ID → see "Change the Application ID" below |
| Amber dot, "ยังไม่เจอแอป Discord" (Discord app not found) | Open the Discord app and wait about 15 seconds. Still there? On Windows, close a Discord that runs as administrator and open it normally |
| Icon stays grey while music plays | Refresh the YouTube Music tab (needed every time after loading or reloading the extension) |
| Icon is red but friends don't see it | Turn on Activity Privacy (step 5) and don't set your status to Invisible |
| Can't click "Listen on YouTube Music" on your own profile | Normal for Discord; other people can click it |
| Status lags when you skip songs quickly | Discord accepts 5 updates per 20 seconds; it catches up |
| Installer says `Unknown file extension ".ts"` | Node is too old → install Node 24 LTS |
| Installer says "ไม่พบ Chrome บนเครื่องนี้" (Chrome not found, macOS) | Open Chrome once, then run the installer again |
| Chrome profile managed by work or school | The organisation may block Developer mode or native messaging → try a personal profile |

Still stuck? Look at the log in `host/host.log` in the project folder.

## Change the Application ID

The installer remembers the current ID. To change it, put the new ID after the command:

- Windows PowerShell: `node scripts\install.ts 123456789012345678`
- macOS: `npm run setup -- 123456789012345678`

Then close **every** YouTube Music tab and open one again, so the helper restarts with the new ID.

## Update to a new version

- Your version: `chrome://extensions` → the number next to the extension's name
- What's new: [CHANGELOG.md](CHANGELOG.md) (in Thai)
- Updating keeps your Application ID; no need to reinstall

**Downloaded as ZIP (no git)**, one command:

1. Open a terminal in the project folder (as in step 3) and run
   - Windows PowerShell: `node scripts\update.ts`
   - macOS: `npm run update`

   It downloads the latest version from GitHub, overwrites only the files that changed, and registers the helper again.
2. `chrome://extensions` → click the reload button ↻ on the extension's card
3. Refresh the YouTube Music tab

**With git:** `git pull`, then steps 2–3

<details>
<summary>On version 0.1.0 (no <code>scripts/update.ts</code> yet)? Update by hand once</summary>

1. On GitHub: **Code** → **Download ZIP** → right-click → **Extract All**
2. Open folders until you reach the one that contains `extension`, `host` and `scripts`
3. Select everything in it → Copy → paste into your existing folder → choose **Replace the files in the destination**
4. Do steps 2–3 above

Careful: if you extract into a new folder and don't copy it over, Chrome keeps using the old folder.
From then on, `node scripts\update.ts` works.

</details>

## Uninstall

1. Windows PowerShell: `node scripts\install.ts --uninstall` · macOS: `npm run setup -- --uninstall`
2. `chrome://extensions` → **Remove** the extension
3. Delete the project folder, and the application in the Discord Developer Portal if you don't need it any more

## Privacy

- The extension runs only on music.youtube.com and asks only for `nativeMessaging` (to talk to the helper on your computer)
- Song info goes only to the Discord app on your computer; there is no other server. Discord shows it to friends according to your settings
- `host/host.log` keeps recent song titles on your computer for troubleshooting (cleared once it passes 1 MB)

## How it works

```
music.youtube.com ─▶ Chrome extension ─▶ helper (started by Chrome) ─▶ Discord app
```

| File | What it does |
|---|---|
| `extension/page-reader.js` | Reads the song, artist, cover and play position from the YouTube Music page |
| `extension/bridge.js` | Passes that from the page to the background script |
| `extension/background.js` | Picks the song to show, sends it to the helper, updates the icon |
| `extension/icons/` | The icon source `icon.svg` and a PNG for each state |
| `host/main.ts` | The helper that Chrome starts (native messaging) |
| `host/discord-ipc.ts` | Talks to the Discord app over IPC, reconnects when needed, respects Discord's rate limit |
| `host/activity.ts` | Turns the song into a Discord status and validates what comes from the browser |
| `scripts/install.ts` | Registers the helper with Chrome (Windows / macOS / Linux) |
| `scripts/update.ts` | Updates a ZIP download to the latest version on GitHub |
| `scripts/icons.ts` | Builds every icon PNG from `icon.svg` (needs ImageMagick) |

Node runs the `.ts` files directly, so there is no build step. The extension is plain `.js`, type-checked by TypeScript.

## For developers

```
npm install      # only TypeScript and type definitions
npm run check    # type-check both sides
npm test         # test the helper against a fake Discord
```

- Changed `extension/` → click ↻ in `chrome://extensions`, then refresh the YouTube Music tab
- Changed `host/` → close every YouTube Music tab and open one again (the helper restarts)
- Icons: edit `extension/icons/icon.svg` and run `npm run icons` (needs ImageMagick) to build every state (`icon-*` red, `idle-*` grey, `warn-*` amber dot, `error-*` red dot), or drop your own PNGs over those files and click ↻
- Windows PowerShell: use `npm.cmd` instead of `npm`, e.g. `npm.cmd test` (close the Discord app before running tests on Windows: the fake Discord uses the same pipe name as the real one)
- On every push, GitHub Actions runs `npm run check` and `npm test` on Windows, macOS and Linux
- Releasing: set the same version in `package.json` and `extension/manifest.json`, add a `CHANGELOG.md` entry, and keep `README.md` (Thai) and `README.en.md` (English) in sync

## License

[MIT](LICENSE): use, change and share it, with credit.

YouTube Music is a trademark of Google LLC and Discord is a trademark of Discord Inc.
This project is not affiliated with or endorsed by either company.
