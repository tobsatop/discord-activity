// A stand-in for the Discord app's IPC socket, for tests.
import fs from 'node:fs';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import type { Activity } from '../activity.ts';
import { FrameDecoder, OP, encodeFrame } from '../discord-ipc.ts';

/** Unix socket paths are limited to ~104 bytes, so keep the folder short. */
export function shortTempDir(): string {
  const base = os.tmpdir().length < 60 ? os.tmpdir() : '/tmp';
  return fs.mkdtempSync(path.join(base, 'ytmd-'));
}

export interface FakeDiscord {
  /** Every SET_ACTIVITY received, in order (null = cleared). */
  activities: Array<Activity | null>;
  waitFor: (ready: (activities: Array<Activity | null>) => boolean, timeoutMs?: number) => Promise<void>;
  close: () => Promise<void>;
}

export async function fakeDiscord(dir: string, options: { rejectDetailsUrl?: boolean } = {}): Promise<FakeDiscord> {
  const activities: Array<Activity | null> = [];
  const listeners = new Set<() => void>();
  const sockets = new Set<net.Socket>();

  const server = net.createServer((socket) => {
    sockets.add(socket);
    socket.on('close', () => sockets.delete(socket));
    socket.on('error', () => {});
    const decoder = new FrameDecoder();
    socket.on('data', (chunk: Buffer) => {
      for (const { op, data } of decoder.push(chunk)) {
        if (op === OP.HANDSHAKE) {
          socket.write(encodeFrame(OP.FRAME, { cmd: 'DISPATCH', evt: 'READY', data: { user: { username: 'tester' } } }));
        } else if (op === OP.FRAME && data?.cmd === 'SET_ACTIVITY') {
          const activity: Activity | null = data.args?.activity ?? null;
          activities.push(activity);
          const rejected = options.rejectDetailsUrl && activity?.details_url;
          socket.write(
            encodeFrame(OP.FRAME, {
              cmd: 'SET_ACTIVITY',
              evt: rejected ? 'ERROR' : null,
              data: rejected ? { code: 4000, message: "child 'details_url' not allowed" } : activity,
              nonce: data.nonce,
            }),
          );
          for (const listener of listeners) listener();
        }
      }
    });
  });
  await new Promise<void>((resolve) => server.listen(path.join(dir, 'discord-ipc-0'), resolve));

  return {
    activities,
    waitFor: (ready, timeoutMs = 5000) =>
      new Promise<void>((resolve, reject) => {
        const check = () => {
          if (!ready(activities)) return;
          listeners.delete(check);
          clearTimeout(timer);
          resolve();
        };
        const timer = setTimeout(() => {
          listeners.delete(check);
          reject(new Error(`timed out; got ${JSON.stringify(activities)}`));
        }, timeoutMs);
        listeners.add(check);
        check();
      }),
    close: () =>
      new Promise<void>((resolve) => {
        for (const socket of sockets) socket.destroy();
        server.close(() => resolve());
      }),
  };
}
