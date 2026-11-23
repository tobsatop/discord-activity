# CLAUDE.md

Shows the song playing on YouTube Music (Chrome) as the user's Discord Rich Presence.
Chrome extension → native messaging host (Node) → Discord desktop app over local IPC.
User-facing docs: README.md (Thai, main) and README.en.md (English) — keep both in sync; CHANGELOG.md is Thai.
Most users are on Windows.

## Commands

```
npm run check    # type-check host + scripts (tsconfig.json) and the extension (extension/tsconfig.json)
npm test         # node:test; host/test/host.test.ts runs host/main.ts end to end against a fake Discord
npm run icons    # rebuild every icon PNG from extension/icons/icon.svg (needs ImageMagick)
npm run setup    # register the native host with Chrome (Windows PowerShell: node scripts\install.ts)
npm run update   # ZIP installs only: pull the latest files from GitHub (refuses inside a git clone)
```

CI (`.github/workflows/ci.yml`) runs check + test on Windows, macOS and Linux with Node 22 and 24,
plus an installer register/verify/uninstall smoke test on Windows. On Windows the fake Discord
listens on the real pipe name `\\?\pipe\discord-ipc-0`, so test files run one at a time and Discord
must be closed when running tests locally.

No build step: Node 22.18+ runs the `.ts` files directly (type stripping), so keep TypeScript
erasable (no enums, no parameter properties, `.ts` extensions in imports). The extension is
plain `.js` with `// @ts-check` and is loaded unpacked from `extension/`.

## Layout

- `extension/page-reader.js` (MAIN world: reads `#movie_player`, `<video>`, `navigator.mediaSession`)
  → `bridge.js` (isolated world, `window.postMessage` → runtime port)
  → `background.js` (picks the track, `connectNative`, toolbar icon state)
- `host/main.ts` (stdin/stdout native messaging) → `activity.ts` (validation, Discord payload)
  → `discord-ipc.ts` (IPC framing, reconnect, 5-per-20s rate limit)
- `scripts/install.ts` writes the launcher to the per-user app-data folder and registers the host
  manifest (file on macOS/Linux, HKCU registry on Windows). Extension ID is fixed by `key` in the manifest.

## Gotchas (all found the hard way)

- macOS: Chrome won't start a native host that lives in ~/Documents, ~/Desktop or ~/Downloads.
  That's why the launcher goes to `~/Library/Application Support/ytm-discord-presence/`.
- Windows PowerShell blocks `npm` (execution policy). Docs tell users to run `node scripts\...`.
- The host must never write to stdout except protocol frames; logs go to `host/host.log`
  (stderr only when it is a TTY, since nobody may drain it under Chrome).
- Content scripts only attach on page load: after reloading the extension, refresh the YouTube Music tab.
- ImageMagick's SVG renderer ignores `transform` on `<ellipse>`; icon.svg draws note heads as arc paths.

## Conventions

- Don't use the real YouTube Music logo (Google trademark; the repo is public). The icon is original.
- Windows instructions come before macOS in docs.
- Releasing: bump `version` in both `package.json` and `extension/manifest.json`, add a
  CHANGELOG.md entry, run `npm run check && npm test`.
- `host/config.json` (Discord Application ID) and `host/host.log` are per machine and gitignored.
- Never log the Discord user name or other account details: users paste host.log into public issues.
- License is MIT (copyright "tobsatop"). Keep the trademark disclaimer at the end of both READMEs.
