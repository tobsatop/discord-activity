// Minimal client for the Discord desktop app's local IPC, the channel that
// Rich Presence libraries use. Each frame is an int32 LE opcode, an int32 LE
// byte length, then UTF-8 JSON. Reconnects on its own and respects Discord's
// SET_ACTIVITY rate limit (5 updates per 20 seconds).
import net from 'node:net';
import { randomUUID } from 'node:crypto';
import type { Activity } from './activity.ts';

export const OP = { HANDSHAKE: 0, FRAME: 1, CLOSE: 2, PING: 3, PONG: 4 } as const;

const PIPE_COUNT = 10; // Discord listens on the first free of discord-ipc-0..9
const MAX_FRAME = 16 * 1024 * 1024;

export type DiscordStatus =
  | { state: 'connecting' }
  | { state: 'connected'; detail: string }
  | { state: 'unavailable'; detail: string }
  | { state: 'error'; detail: string };

export interface DiscordIpcOptions {
  clientId: string;
  log: (...parts: unknown[]) => void;
  onStatus: (status: DiscordStatus) => void;
  /** Discord answered SET_ACTIVITY with an error (usually a field it doesn't accept). */
  onActivityError: (message: string) => void;
  retryMs?: number;
  rateWindowMs?: number;
  rateMax?: number;
}

export interface Frame {
  op: number;
  data: any;
}

export function ipcPath(index: number, platform: string = process.platform, env: NodeJS.ProcessEnv = process.env): string {
  if (platform === 'win32') return `\\\\?\\pipe\\discord-ipc-${index}`;
  const base = (env.XDG_RUNTIME_DIR || env.TMPDIR || env.TMP || env.TEMP || '/tmp').replace(/\/+$/, '');
  return `${base}/discord-ipc-${index}`;
}

export function encodeFrame(op: number, payload: unknown): Buffer {
  const body = Buffer.from(JSON.stringify(payload), 'utf8');
  const header = Buffer.alloc(8);
  header.writeInt32LE(op, 0);
  header.writeInt32LE(body.length, 4);
  return Buffer.concat([header, body]);
}

export class FrameDecoder {
  #buffer: Buffer = Buffer.alloc(0);

  push(chunk: Buffer): Frame[] {
    this.#buffer = Buffer.concat([this.#buffer, chunk]);
    const frames: Frame[] = [];
    while (this.#buffer.length >= 8) {
      const op = this.#buffer.readInt32LE(0);
      const length = this.#buffer.readInt32LE(4);
      if (length < 0 || length > MAX_FRAME) {
        this.#buffer = Buffer.alloc(0); // corrupt stream; drop it
        break;
      }
      if (this.#buffer.length < 8 + length) break;
      const body = this.#buffer.subarray(8, 8 + length).toString('utf8');
      this.#buffer = this.#buffer.subarray(8 + length);
      try {
        frames.push({ op, data: JSON.parse(body) });
      } catch {
        // skip a malformed frame
      }
    }
    return frames;
  }
}

function connectTo(path: string): Promise<net.Socket> {
  return new Promise((resolve, reject) => {
    const socket = net.createConnection(path);
    const onError = (error: Error) => {
      socket.destroy();
      reject(error);
    };
    socket.once('error', onError);
    socket.once('connect', () => {
      socket.off('error', onError);
      resolve(socket);
    });
  });
}

export class DiscordIpc {
  readonly #options: Required<DiscordIpcOptions>;
  #socket: net.Socket | null = null;
  #ready = false;
  #connecting = false;
  #stopped = false;
  #closeReason = '';
  #lastStatus = '';
  #retryTimer: NodeJS.Timeout | undefined;
  #flushTimer: NodeJS.Timeout | undefined;
  #desired: Activity | null = null;
  #desiredKey = 'null';
  #appliedKey = 'null'; // what Discord shows from this connection
  #sendTimes: number[] = [];

  constructor(options: DiscordIpcOptions) {
    this.#options = { retryMs: 15_000, rateWindowMs: 20_000, rateMax: 5, ...options };
  }

  start(): void {
    void this.#connect();
  }

  /** Show this activity (null clears it). Only the latest value is sent. */
  setActivity(activity: Activity | null): void {
    this.#desired = activity;
    this.#desiredKey = JSON.stringify(activity);
    this.#scheduleFlush();
  }

  /** Clear the activity and disconnect. */
  async stop(): Promise<void> {
    this.#stopped = true;
    clearTimeout(this.#retryTimer);
    clearTimeout(this.#flushTimer);
    const socket = this.#socket;
    if (!socket) return;
    await new Promise<void>((resolve) => {
      const timer = setTimeout(() => {
        socket.destroy();
        resolve();
      }, 1000);
      socket.once('close', () => {
        clearTimeout(timer);
        resolve();
      });
      if (this.#ready && this.#appliedKey !== 'null') {
        socket.write(encodeFrame(OP.FRAME, { cmd: 'SET_ACTIVITY', args: { pid: process.pid }, nonce: randomUUID() }));
      }
      socket.end();
    });
  }

  #status(status: DiscordStatus): void {
    const key = JSON.stringify(status);
    if (key === this.#lastStatus) return;
    this.#lastStatus = key;
    this.#options.log(`discord: ${status.state}`, 'detail' in status ? status.detail : '');
    this.#options.onStatus(status);
  }

  async #connect(): Promise<void> {
    if (this.#stopped || this.#socket || this.#connecting) return;
    this.#connecting = true;
    if (!this.#lastStatus) this.#status({ state: 'connecting' });

    let socket: net.Socket | null = null;
    for (let i = 0; i < PIPE_COUNT && !socket; i++) {
      socket = await connectTo(ipcPath(i)).catch(() => null);
    }
    this.#connecting = false;

    if (this.#stopped) {
      socket?.destroy();
      return;
    }
    if (!socket) {
      this.#status({ state: 'unavailable', detail: 'Discord app is not running' });
      this.#retry(this.#options.retryMs);
      return;
    }
    this.#attach(socket);
  }

  #attach(socket: net.Socket): void {
    this.#socket = socket;
    this.#ready = false;
    this.#appliedKey = 'null';
    this.#closeReason = '';
    const decoder = new FrameDecoder();

    socket.on('data', (chunk: Buffer) => {
      for (const frame of decoder.push(chunk)) this.#handle(socket, frame);
    });
    socket.on('error', (error) => this.#options.log('discord socket error:', error.message));
    socket.on('close', () => {
      if (this.#socket !== socket) return;
      this.#socket = null;
      this.#ready = false;
      this.#appliedKey = 'null';
      clearTimeout(this.#flushTimer);
      this.#flushTimer = undefined;
      if (this.#stopped) return;
      if (this.#closeReason) {
        this.#status({ state: 'error', detail: this.#closeReason });
        this.#retry(this.#options.retryMs * 4);
      } else {
        this.#status({ state: 'unavailable', detail: 'Discord app closed the connection' });
        this.#retry(this.#options.retryMs);
      }
    });

    socket.write(encodeFrame(OP.HANDSHAKE, { v: 1, client_id: this.#options.clientId }));
  }

  #handle(socket: net.Socket, frame: Frame): void {
    const data = frame.data;
    if (frame.op === OP.PING) {
      socket.write(encodeFrame(OP.PONG, data));
    } else if (frame.op === OP.CLOSE) {
      this.#closeReason = `Discord ไม่รับการเชื่อมต่อ: ${data?.message ?? 'unknown'} — ตรวจ Application ID (npm run setup)`;
      socket.destroy();
    } else if (frame.op === OP.FRAME) {
      if (data?.cmd === 'DISPATCH' && data.evt === 'READY') {
        this.#ready = true;
        const user = data.data?.user;
        this.#status({ state: 'connected', detail: String(user?.global_name || user?.username || '') });
        this.#scheduleFlush();
      } else if (data?.cmd === 'SET_ACTIVITY' && data.evt === 'ERROR') {
        this.#options.onActivityError(String(data.data?.message ?? 'unknown error'));
      }
    }
  }

  #retry(ms: number): void {
    clearTimeout(this.#retryTimer);
    this.#retryTimer = setTimeout(() => void this.#connect(), ms);
  }

  #scheduleFlush(): void {
    if (!this.#ready || this.#flushTimer || this.#desiredKey === this.#appliedKey) return;
    const { rateWindowMs, rateMax } = this.#options;
    const now = Date.now();
    this.#sendTimes = this.#sendTimes.filter((t) => now - t < rateWindowMs);
    const oldest = this.#sendTimes[0] ?? now;
    const wait = this.#sendTimes.length < rateMax ? 0 : rateWindowMs - (now - oldest) + 50;
    this.#flushTimer = setTimeout(() => {
      this.#flushTimer = undefined;
      this.#flush();
    }, wait);
  }

  #flush(): void {
    const socket = this.#socket;
    if (!this.#ready || !socket || this.#desiredKey === this.#appliedKey) return;
    this.#sendTimes.push(Date.now());
    this.#appliedKey = this.#desiredKey;
    const args = this.#desired ? { pid: process.pid, activity: this.#desired } : { pid: process.pid };
    socket.write(encodeFrame(OP.FRAME, { cmd: 'SET_ACTIVITY', args, nonce: randomUUID() }));
  }
}
