import { test } from 'node:test';
import assert from 'node:assert/strict';
import { DiscordIpc, FrameDecoder, encodeFrame, ipcPath } from '../discord-ipc.ts';
import { fakeDiscord, shortTempDir } from './fake-discord.ts';

test('ipc path per platform', () => {
  assert.equal(ipcPath(0, 'win32'), '\\\\?\\pipe\\discord-ipc-0');
  assert.equal(ipcPath(3, 'darwin', { TMPDIR: '/var/folders/x/T/' }), '/var/folders/x/T/discord-ipc-3');
  assert.equal(ipcPath(0, 'linux', { XDG_RUNTIME_DIR: '/run/user/1000', TMPDIR: '/tmp' }), '/run/user/1000/discord-ipc-0');
  assert.equal(ipcPath(0, 'linux', {}), '/tmp/discord-ipc-0');
});

test('frames survive being split and glued together', () => {
  const bytes = Buffer.concat([encodeFrame(1, { a: 'ไทย' }), encodeFrame(3, { b: 2 })]);
  const decoder = new FrameDecoder();
  const frames = [...decoder.push(bytes.subarray(0, 5)), ...decoder.push(bytes.subarray(5, 20)), ...decoder.push(bytes.subarray(20))];
  assert.deepEqual(frames, [
    { op: 1, data: { a: 'ไทย' } },
    { op: 3, data: { b: 2 } },
  ]);
});

test('updates over the rate limit wait, and only the latest is sent', async (t) => {
  const dir = shortTempDir();
  const discord = await fakeDiscord(dir);
  const prevTmp = process.env.TMPDIR;
  const prevXdg = process.env.XDG_RUNTIME_DIR;
  process.env.TMPDIR = dir;
  delete process.env.XDG_RUNTIME_DIR;
  const ipc = new DiscordIpc({
    clientId: '123456789012345678',
    log: () => {},
    onStatus: () => {},
    onActivityError: () => {},
    rateWindowMs: 400,
    rateMax: 2,
  });
  t.after(async () => {
    await ipc.stop();
    await discord.close();
    process.env.TMPDIR = prevTmp;
    if (prevXdg !== undefined) process.env.XDG_RUNTIME_DIR = prevXdg;
  });

  ipc.start();
  const act = (details: string) => ({ type: 2, details });
  ipc.setActivity(act('one'));
  await discord.waitFor((list) => list.length === 1);
  const firstAt = Date.now();
  ipc.setActivity(act('two'));
  await discord.waitFor((list) => list.length === 2);
  ipc.setActivity(act('three')); // over the limit: waits for the window
  ipc.setActivity(act('four')); // replaces "three" before it is sent
  await discord.waitFor((list) => list.length === 3);
  assert.ok(Date.now() - firstAt >= 300, 'third update waited for the rate window');
  assert.deepEqual(
    discord.activities.map((a) => a?.details),
    ['one', 'two', 'four'],
  );
});
