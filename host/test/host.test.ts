// End to end: run host/main.ts the way Chrome does (native messages on
// stdin/stdout) against a fake Discord app.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { spawn } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { encodeNativeMessage, readNativeMessages } from '../native-messaging.ts';
import { fakeDiscord, shortTempDir } from './fake-discord.ts';

const MAIN = path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'main.ts');
const TRACK = {
  videoId: 'dQw4w9WgXcQ',
  title: 'Song Title',
  artist: 'Some Artist',
  album: 'Some Album',
  artworkUrl: 'https://lh3.googleusercontent.com/abc=w544-h544',
  startMs: Date.now() - 30_000,
  endMs: Date.now() + 170_000,
  playing: true,
};

function startHost(dir: string, env: Record<string, string> = {}) {
  const child = spawn(process.execPath, ['--disable-warning=ExperimentalWarning', MAIN], {
    env: {
      ...process.env,
      TMPDIR: dir,
      XDG_RUNTIME_DIR: '',
      DISCORD_CLIENT_ID: '123456789012345678',
      YTM_DISCORD_CONFIG: path.join(dir, 'missing-config.json'),
      YTM_DISCORD_LOG: path.join(dir, 'host.log'),
      ...env,
    },
    stdio: ['pipe', 'pipe', 'ignore'],
  });
  const fromHost: Array<{ type: string; state?: string; detail?: string }> = [];
  readNativeMessages(child.stdout, (m) => fromHost.push(m as (typeof fromHost)[number]), (e) => assert.fail(e));
  const exited = new Promise<number | null>((resolve) => child.on('exit', resolve));
  const update = (track: unknown) => child.stdin.write(encodeNativeMessage({ type: 'update', track }));
  return { child, fromHost, exited, update };
}

test('shows the song, clears on pause, and clears + exits when Chrome disconnects', async (t) => {
  const dir = shortTempDir();
  const discord = await fakeDiscord(dir);
  t.after(() => discord.close());
  const host = startHost(dir);

  host.update(TRACK);
  await discord.waitFor((list) => list.length === 1);
  assert.equal(discord.activities[0]?.details, 'Song Title');
  assert.equal(discord.activities[0]?.state, 'Some Artist');
  assert.equal(discord.activities[0]?.type, 2);

  host.update({ ...TRACK, playing: false });
  await discord.waitFor((list) => list.length === 2);
  assert.equal(discord.activities[1], null);

  host.update(TRACK);
  await discord.waitFor((list) => list.length === 3);
  host.child.stdin.end(); // what Chrome does when the last YouTube Music tab closes
  await discord.waitFor((list) => list.length === 4);
  assert.equal(discord.activities[3], null);
  assert.equal(await host.exited, 0);
  assert.ok(host.fromHost.some((m) => m.type === 'status' && m.state === 'connected'));
});

test('falls back to basic fields if Discord rejects the newer ones', async (t) => {
  const dir = shortTempDir();
  const discord = await fakeDiscord(dir, { rejectDetailsUrl: true });
  t.after(() => discord.close());
  const host = startHost(dir);
  t.after(() => host.child.kill());

  host.update(TRACK);
  await discord.waitFor((list) => list.length === 2);
  assert.ok(discord.activities[0]?.details_url, 'first try has the link');
  assert.equal(discord.activities[1]?.details_url, undefined, 'retry drops it');
  assert.equal(discord.activities[1]?.details, 'Song Title');
});

test('reports Discord not running, and a missing Application ID', async (t) => {
  const dir = shortTempDir(); // no fake Discord here
  const noDiscord = startHost(dir);
  const noId = startHost(dir, { DISCORD_CLIENT_ID: '' });
  t.after(() => {
    noDiscord.child.kill();
    noId.child.kill();
  });
  const until = async (ok: () => boolean) => {
    for (let i = 0; i < 50 && !ok(); i++) await new Promise((r) => setTimeout(r, 100));
  };
  await until(() => noDiscord.fromHost.some((m) => m.state === 'unavailable'));
  await until(() => noId.fromHost.some((m) => m.state === 'error'));
  assert.ok(noDiscord.fromHost.some((m) => m.state === 'unavailable'));
  assert.match(noId.fromHost.find((m) => m.state === 'error')?.detail ?? '', /Application ID/);
});
