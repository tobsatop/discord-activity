// Native messaging host. Chrome starts this (through the launcher that
// `npm run setup` writes) while a YouTube Music tab is open. It reads songs
// from the extension on stdin and shows them in the Discord app. When Chrome
// closes stdin, it clears the status and exits.
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { parseTrack, toActivity, type Track } from './activity.ts';
import { DiscordIpc } from './discord-ipc.ts';
import { encodeNativeMessage, readNativeMessages } from './native-messaging.ts';

const HOST_DIR = path.dirname(fileURLToPath(import.meta.url));
const CONFIG_FILE = process.env.YTM_DISCORD_CONFIG || path.join(HOST_DIR, 'config.json');
const LOG_FILE = process.env.YTM_DISCORD_LOG || path.join(HOST_DIR, 'host.log');
const MAX_LOG_BYTES = 1024 * 1024;
const CLIENT_ID = /^\d{17,20}$/;

// stdout belongs to Chrome; anything else printed there breaks the protocol.
console.log = console.error;

function log(...parts: unknown[]): void {
  const text = parts
    .map((p) => (p instanceof Error ? (p.stack ?? p.message) : typeof p === 'string' ? p : JSON.stringify(p)))
    .join(' ');
  const line = `${new Date().toISOString()} ${text}\n`;
  // Only echo to a terminal: under Chrome nobody may be reading stderr, and a
  // full pipe would block the host.
  if (process.stderr.isTTY) process.stderr.write(line);
  try {
    fs.appendFileSync(LOG_FILE, line);
  } catch {
    // logging must never take the host down
  }
}

function trimLog(): void {
  try {
    if (fs.statSync(LOG_FILE).size > MAX_LOG_BYTES) fs.writeFileSync(LOG_FILE, '');
  } catch {
    // no log yet
  }
}

function send(message: unknown): void {
  process.stdout.write(encodeNativeMessage(message));
}

function loadClientId(): string | undefined {
  let id = process.env.DISCORD_CLIENT_ID;
  if (!id) {
    try {
      id = String(JSON.parse(fs.readFileSync(CONFIG_FILE, 'utf8')).clientId ?? '');
    } catch {
      return undefined;
    }
  }
  return CLIENT_ID.test(id) ? id : undefined;
}

trimLog();
log(`host started (node ${process.version}, pid ${process.pid})`);
process.stdout.on('error', () => process.exit(0)); // Chrome went away mid-write
process.on('uncaughtException', (error) => {
  log('crash:', error);
  process.exit(1);
});

const clientId = loadClientId();
let discord: DiscordIpc | undefined;
let track: Track | null = null;
let basic = false;

if (clientId) {
  discord = new DiscordIpc({
    clientId,
    log,
    onStatus: (status) => send({ type: 'status', ...status }),
    onActivityError: (message) => {
      log('Discord rejected the activity:', message);
      if (!basic) {
        basic = true; // retry without the newer optional fields
        discord?.setActivity(toActivity(track, { basic }));
        return;
      }
      send({ type: 'status', state: 'error', detail: `Discord ไม่รับสถานะ: ${message}` });
    },
  });
  discord.start();
} else {
  log(`no valid Discord Application ID in ${CONFIG_FILE}`);
  send({ type: 'status', state: 'error', detail: 'ยังไม่ได้ใส่ Discord Application ID — รัน npm run setup' });
}

readNativeMessages(
  process.stdin,
  (message) => {
    const msg = message as { type?: unknown; track?: unknown } | null;
    if (msg?.type !== 'update') return;
    track = parseTrack(msg.track);
    log(track?.playing ? `playing: ${track.title} — ${track.artist}` : 'nothing playing');
    discord?.setActivity(toActivity(track, { basic }));
  },
  (error) => log('bad message from Chrome:', error.message),
);

process.stdin.on('end', () => {
  log('Chrome disconnected; clearing the status');
  void (discord?.stop() ?? Promise.resolve()).finally(() => process.exit(0));
});
