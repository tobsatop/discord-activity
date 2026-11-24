# CLAUDE.md

Shows the song playing on YouTube Music (Chrome) as the user's Discord Rich Presence.
Chrome extension → native messaging helper (one Go program) → Discord desktop app over local IPC.
User-facing docs: README.md (Thai, main) and README.en.md (English) — keep both in sync; CHANGELOG.md is Thai.
Most users are on Windows. Users download one file from GitHub Releases, double-click it, paste
their own Discord Application ID, then Load unpacked the folder it prints (no Web Store, no Node.js).

## Commands

```
go vet ./... && go test ./...   # helper + installer; end-to-end tests run the test binary the way Chrome does, against a fake Discord
GOOS=windows go vet ./...       # also checks the Windows-only files (registry, named pipes) from macOS/Linux
go build -o ytm-discord .       # the program users download: installer and helper in one binary
npm run check                   # type-check the extension (extension/tsconfig.json) and scripts/icons.ts
npm run icons                   # rebuild every icon PNG from extension/icons/icon.svg (needs ImageMagick)
```

CI (`.github/workflows/ci.yml`): go vet + test on Windows, macOS and Linux (`-race` on Linux), the
extension type-check, and the Windows installer smoke test `scripts/smoke-windows.ps1` (temp data dir +
throwaway HKCU key; also re-installs while the helper runs). release.yml runs the same smoke test on
windows-latest before it publishes anything.

go.mod: `go 1.24.0`, `toolchain go1.26.11` (an older Go switches automatically). Deps: go-winio
v0.6.2 (Windows pipes) and x/sys v0.41.0 — newer releases of both need a newer `go` line.
The extension is plain `.js` with `// @ts-check`; no build step.

## Layout

- `extension/page-reader.js` (MAIN world: reads `#movie_player`, `<video>`, `navigator.mediaSession`)
  → `bridge.js` (isolated world, `window.postMessage` → runtime port)
  → `background.js` (picks the track, `connectNative`, toolbar icon state)
- `main.go` embeds `extension/` (manifest.json, *.js, icons/*.png) and picks the mode from the args:
  `chrome-extension://…` or `--parent-window=` → helper; none (double-click) → installer that asks
  and waits for Enter; `<Application ID>` → installer without asking; `--uninstall`; `--version`.
- `internal/helper` (stdin/stdout, config, host.log) → `internal/activity` (validation, Discord payload)
  → `internal/discordipc` (IPC framing, reconnect, 5-per-20s rate limit; all state on one event-loop goroutine).
  `internal/nativemsg` is Chrome's framing. `internal/fakediscord` is the test-only fake Discord.
- `internal/installer` copies the running program to the data dir as `ytm-discord-host(.exe)`, writes
  the host manifest (files for Chrome/Edge/Brave/Chromium on macOS/Linux, HKCU keys for Chrome/Edge on
  Windows), unpacks the extension, removes Node-era `run-host.*` launchers and reuses the Application ID
  from the old project's `host/config.json`.
- Data dir: `%LOCALAPPDATA%\ytm-discord-presence\`, `~/Library/Application Support/ytm-discord-presence/`,
  `$XDG_DATA_HOME/ytm-discord-presence/`. Holds the helper, `config.json` (Application ID), `host.log`,
  `extension/` (what users Load unpacked) and on Windows `native-host.json`. The helper looks for
  config/log next to its own executable.
- Extension ID is fixed by `key` in the manifest (`bfcijjaglbmlpnnpnekjcamlmgdkdebb`). `HOST_NAME` in
  background.js must equal `installer.HostName` (a test checks).

## Test overrides — never run the installer against real folders on a dev machine

`YTM_DISCORD_DATA_DIR`, `YTM_DISCORD_BROWSER_HOME` (stands in for home when looking for browsers),
`YTM_DISCORD_REGISTRY_KEY` (replaces `HKCU\Software`), `YTM_DISCORD_IPC` (Discord socket/pipe path
without `-N`), `DISCORD_CLIENT_ID`. The root and discordipc `TestMain`s set them all to temp values.
The installer reveals the extension folder in Explorer/Finder only when double-clicked (no args), never in
tests. It does not open Chrome: Chrome ignores chrome:// URLs passed from outside, so users type chrome://extensions.

## Gotchas (all found the hard way)

- macOS: Chrome won't start a native host that lives in ~/Documents, ~/Desktop or ~/Downloads.
  That's why the helper is copied to `~/Library/Application Support/ytm-discord-presence/`.
- macOS: the downloaded file is quarantined, so Gatekeeper asks once (Privacy & Security → Open Anyway).
  The installed copy has the quarantine flag removed. The release lipo-merges arm64+amd64 and then
  re-signs ad hoc: Apple Silicon refuses unsigned code.
- Windows: open Discord's pipe with go-winio (overlapped I/O). A plain os.OpenFile pipe deadlocks when
  one goroutine reads while another writes.
- Windows: a running helper can't be overwritten, only renamed: the installer renames it to `*.old-*`
  and deletes those next time. Uninstall can't delete it while Chrome runs it (it says so).
- Windows: `explorer.exe /select,` needs a raw `SysProcAttr.CmdLine`; Go's quoting breaks paths with spaces.
- The helper must never write to stdout except protocol frames; logs go to host.log (stderr only when it
  is a terminal, since nobody may drain it under Chrome). Crashes go to host.log via debug.SetCrashOutput.
- Text lengths in internal/activity count UTF-16 units, like the JavaScript version and Discord.
- Content scripts only attach on page load: after reloading the extension, refresh the YouTube Music tab.
- ImageMagick's SVG renderer ignores `transform` on `<ellipse>`; icon.svg draws note heads as arc paths.

## Conventions

- Don't use the real YouTube Music logo (Google trademark; the repo is public). The icon is original.
- Windows instructions come before macOS in docs.
- Status texts the helper sends are shown by the extension and cite README sections ("README ขั้นที่ 3",
  "README หัวข้อเปลี่ยน Application ID"); keep them and the README in step.
- Never log the Discord user name or other account details: users paste host.log into public issues.
- License is MIT (copyright "tobsatop"). Keep the trademark disclaimer at the end of both READMEs.

## Releasing

1. Set the same version in `version.go`, `extension/manifest.json`, `package.json` (and the root entries of
   `package-lock.json`); `TestVersionsMatch` checks the first three.
2. Add a CHANGELOG.md entry (Thai, with the release date); update both READMEs if steps changed.
3. `go vet ./... && go test ./... && npm run check`, commit, then push a tag `vX.Y.Z`.
4. `.github/workflows/release.yml` checks the tag against the three versions, tests, builds
   `YouTube-Music-Discord-Setup.exe`, `YouTube-Music-Discord-Setup-ARM.exe` (icon + version details from
   `tools/genicons`), `YouTube-Music-Discord-Setup-Mac.zip` ("YouTube Music Discord Setup.app", universal,
   talks through dialogs; built by `scripts/build-release.sh`)
   and `SHA256SUMS.txt`, and publishes the GitHub Release with `gh release create`.
