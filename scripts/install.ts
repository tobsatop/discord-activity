// One-time setup on each computer (macOS, Windows or Linux):
//   npm run setup                      ask for the Discord Application ID, register the helper
//   npm run setup -- <Application ID>  same, without asking
//   npm run setup -- --uninstall       remove the registration
// Re-run it after moving this folder or changing Node versions.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import crypto from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { createInterface } from 'node:readline/promises';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const HOST_DIR = path.join(ROOT, 'host');
const CONFIG_FILE = path.join(HOST_DIR, 'config.json');
const HOST_NAME = 'local.ytm_discord_presence'; // must match HOST_NAME in extension/background.js
const CLIENT_ID = /^\d{17,20}$/;
const IS_WINDOWS = process.platform === 'win32';

/** Chrome derives an unpacked extension's id from the "key" in its manifest. */
function extensionId(): string {
  const manifest = JSON.parse(fs.readFileSync(path.join(ROOT, 'extension', 'manifest.json'), 'utf8'));
  const hex = crypto.createHash('sha256').update(Buffer.from(manifest.key, 'base64')).digest('hex').slice(0, 32);
  return [...hex].map((c) => String.fromCharCode(97 + Number.parseInt(c, 16))).join('');
}

/** Where each Chromium browser looks for native messaging host manifests. */
function manifestTargets(): Array<{ browser: string; file?: string; registryKey?: string }> {
  const home = os.homedir();
  if (IS_WINDOWS) {
    return [
      { browser: 'Chrome', registryKey: `HKCU\\Software\\Google\\Chrome\\NativeMessagingHosts\\${HOST_NAME}` },
      { browser: 'Edge', registryKey: `HKCU\\Software\\Microsoft\\Edge\\NativeMessagingHosts\\${HOST_NAME}` },
    ];
  }
  const profiles =
    process.platform === 'darwin'
      ? {
          Chrome: 'Library/Application Support/Google/Chrome',
          Edge: 'Library/Application Support/Microsoft Edge',
          Brave: 'Library/Application Support/BraveSoftware/Brave-Browser',
          Chromium: 'Library/Application Support/Chromium',
        }
      : {
          Chrome: '.config/google-chrome',
          Edge: '.config/microsoft-edge',
          Brave: '.config/BraveSoftware/Brave-Browser',
          Chromium: '.config/chromium',
        };
  return Object.entries(profiles).map(([browser, dir]) => ({
    browser,
    file: path.join(home, dir, 'NativeMessagingHosts', `${HOST_NAME}.json`),
  }));
}

function checkNode(): void {
  const [major = 0, minor = 0] = process.versions.node.split('.').map(Number);
  if (major < 22 || (major === 22 && minor < 18)) {
    console.error(`ต้องใช้ Node.js 22.18 ขึ้นไป (แนะนำ 24 LTS) — ตอนนี้เป็น ${process.version}`);
    process.exit(1);
  }
}

async function resolveClientId(arg: string | undefined): Promise<string> {
  if (arg) return arg.trim();
  try {
    const existing = String(JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8')).clientId ?? '');
    if (CLIENT_ID.test(existing)) {
      console.log(`ใช้ Application ID เดิม: ${existing}`);
      return existing;
    }
  } catch {
    // no config yet
  }
  const rl = createInterface({ input: process.stdin, output: process.stdout });
  const answer = await rl.question('วาง Application ID จาก Discord Developer Portal: ');
  rl.close();
  return answer.trim();
}

/**
 * Per-user app-data folder for the launcher and (on Windows) the host manifest.
 * Kept outside the project folder: macOS won't let Chrome start a program that
 * lives in Documents, Desktop or Downloads.
 */
function dataDir(): string {
  const home = os.homedir();
  if (IS_WINDOWS) return path.join(process.env.LOCALAPPDATA || path.join(home, 'AppData', 'Local'), 'ytm-discord-presence');
  if (process.platform === 'darwin') return path.join(home, 'Library', 'Application Support', 'ytm-discord-presence');
  return path.join(process.env.XDG_DATA_HOME || path.join(home, '.local', 'share'), 'ytm-discord-presence');
}

/** Files older versions of this script wrote into host/. */
function removeLegacyFiles(): void {
  for (const name of ['run-host.sh', 'run-host.bat', 'native-host.json']) {
    fs.rmSync(path.join(HOST_DIR, name), { force: true });
  }
}

/** Chrome runs this file; it starts the host with this exact Node. */
function writeLauncher(): string {
  const main = path.join(HOST_DIR, 'main.ts');
  const flags = '--disable-warning=ExperimentalWarning';
  fs.mkdirSync(dataDir(), { recursive: true });
  if (IS_WINDOWS) {
    const file = path.join(dataDir(), 'run-host.bat');
    fs.writeFileSync(file, `@echo off\r\n"${process.execPath}" ${flags} "${main}" %*\r\n`);
    return file;
  }
  const file = path.join(dataDir(), 'run-host.sh');
  fs.writeFileSync(file, `#!/bin/sh\nexec "${process.execPath}" ${flags} "${main}" "$@"\n`, { mode: 0o755 });
  fs.chmodSync(file, 0o755);
  return file;
}

function install(clientId: string, options: { update?: boolean } = {}): void {
  fs.writeFileSync(CONFIG_FILE, `${JSON.stringify({ clientId }, null, 2)}\n`);
  removeLegacyFiles();
  const id = extensionId();
  const manifest = JSON.stringify(
    {
      name: HOST_NAME,
      description: 'YouTube Music to Discord Rich Presence',
      path: writeLauncher(),
      type: 'stdio',
      allowed_origins: [`chrome-extension://${id}/`],
    },
    null,
    2,
  );

  const done: string[] = [];
  if (IS_WINDOWS) {
    const file = path.join(dataDir(), 'native-host.json');
    fs.writeFileSync(file, `${manifest}\n`);
    for (const target of manifestTargets()) {
      execFileSync('reg', ['add', target.registryKey!, '/ve', '/t', 'REG_SZ', '/d', file, '/f'], { stdio: 'ignore' });
      done.push(target.browser);
    }
  } else {
    for (const target of manifestTargets()) {
      const browserDir = path.dirname(path.dirname(target.file!));
      if (!fs.existsSync(browserDir)) continue; // browser not installed
      fs.mkdirSync(path.dirname(target.file!), { recursive: true });
      fs.writeFileSync(target.file!, `${manifest}\n`);
      done.push(target.browser);
    }
  }

  if (done.length === 0) {
    console.error('ไม่พบ Chrome บนเครื่องนี้ — ติดตั้ง Chrome ก่อนแล้วรันใหม่');
    process.exit(1);
  }
  console.log(`ลงทะเบียนโปรแกรมเสริมกับ ${done.join(', ')} แล้ว`);
  if (options.update) return; // scripts/update.ts prints its own next steps
  console.log(`ID ส่วนขยาย: ${id}`);
  console.log('');
  console.log('ขั้นต่อไป:');
  console.log('1. เปิด chrome://extensions แล้วเปิด Developer mode');
  console.log(`2. กด Load unpacked แล้วเลือกโฟลเดอร์ ${path.join(ROOT, 'extension')}`);
  console.log('3. เปิดแอป Discord แล้วเล่นเพลงที่ music.youtube.com');
}

function uninstall(): void {
  for (const target of manifestTargets()) {
    if (target.registryKey) {
      try {
        execFileSync('reg', ['delete', target.registryKey, '/f'], { stdio: 'ignore' });
      } catch {
        // was not registered
      }
    } else if (target.file) {
      fs.rmSync(target.file, { force: true });
    }
  }
  fs.rmSync(dataDir(), { recursive: true, force: true });
  removeLegacyFiles();
  console.log('ถอนการลงทะเบียนแล้ว (ลบส่วนขยายเองได้ที่ chrome://extensions)');
}

checkNode();
const args = process.argv.slice(2);
const positional = args.filter((arg) => !arg.startsWith('--'));
if (args.includes('--uninstall')) {
  uninstall();
} else {
  const clientId = await resolveClientId(positional[0]);
  if (!CLIENT_ID.test(clientId)) {
    console.error('Application ID ต้องเป็นตัวเลข 17–20 หลัก (Discord Developer Portal → General Information)');
    process.exit(1);
  }
  install(clientId, { update: args.includes('--update') });
}
