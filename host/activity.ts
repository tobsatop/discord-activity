// Turns the song from the extension into a Discord activity payload.
// Everything coming from the browser is treated as untrusted and validated here.

/** Same shape as `Track` in extension/types.d.ts. */
export interface Track {
  videoId: string;
  title: string;
  artist: string;
  album: string;
  artworkUrl: string;
  startMs: number;
  endMs: number;
  playing: boolean;
}

/** The subset of Discord's activity object this app sends. */
export interface Activity {
  type: number;
  status_display_type?: number;
  details: string;
  details_url?: string;
  state?: string;
  timestamps?: { start: number; end?: number };
  assets?: { large_image: string; large_text?: string };
  buttons?: Array<{ label: string; url: string }>;
}

const ACTIVITY_LISTENING = 2;
const STATUS_DISPLAY_DETAILS = 2; // member list shows "Listening to <song>" instead of the app name
const MAX_TEXT = 128; // Discord's limit for details/state/large_text
const MAX_IMAGE_URL = 256; // Discord's limit for an asset key or URL
const DAY_MS = 24 * 60 * 60 * 1000;
const VIDEO_ID = /^[\w-]{11}$/;

const str = (value: unknown, max = 1024): string => (typeof value === 'string' ? value.slice(0, max) : '');
const num = (value: unknown): number => (typeof value === 'number' && Number.isFinite(value) ? value : 0);

export function parseTrack(input: unknown): Track | null {
  if (!input || typeof input !== 'object') return null;
  const o = input as Record<string, unknown>;
  const title = str(o.title).trim();
  if (!title) return null;
  const videoId = str(o.videoId);
  return {
    videoId: VIDEO_ID.test(videoId) ? videoId : '',
    title,
    artist: str(o.artist).trim(),
    album: str(o.album).trim(),
    artworkUrl: str(o.artworkUrl, 4096),
    startMs: num(o.startMs),
    endMs: num(o.endMs),
    playing: o.playing === true,
  };
}

/** Collapse whitespace and fit Discord's 2..128 character rule without splitting emoji. */
export function fit(text: string, max = MAX_TEXT): string {
  const clean = text.replace(/\s+/g, ' ').trim();
  let out = clean;
  if (clean.length > max) {
    out = '';
    for (const ch of clean) {
      if (out.length + ch.length > max - 1) break;
      out += ch;
    }
    out += '…';
  }
  return out.padEnd(2, '⠀'); // braille blank: Discord rejects 1-character strings
}

function isHttpsUrl(value: string): boolean {
  try {
    return new URL(value).protocol === 'https:';
  } catch {
    return false;
  }
}

function imageFor(track: Track): string | undefined {
  if (track.artworkUrl.length <= MAX_IMAGE_URL && isHttpsUrl(track.artworkUrl)) return track.artworkUrl;
  if (track.videoId) return `https://i.ytimg.com/vi/${track.videoId}/hqdefault.jpg`;
  return undefined;
}

function timestampsFor(track: Track): Activity['timestamps'] {
  if (track.startMs <= 0) return undefined;
  const start = Math.round(track.startMs);
  const length = track.endMs - track.startMs;
  return length > 0 && length < DAY_MS ? { start, end: Math.round(track.endMs) } : { start };
}

/**
 * `basic` leaves out the newer fields (clickable song title, song name in the
 * member list), for Discord versions that reject them.
 */
export function toActivity(track: Track | null, options: { basic?: boolean } = {}): Activity | null {
  if (!track?.playing) return null;
  const songUrl = track.videoId ? `https://music.youtube.com/watch?v=${track.videoId}` : '';

  const activity: Activity = { type: ACTIVITY_LISTENING, details: fit(track.title) };
  if (!options.basic) {
    activity.status_display_type = STATUS_DISPLAY_DETAILS;
    if (songUrl) activity.details_url = songUrl;
  }
  if (track.artist) activity.state = fit(track.artist);

  const timestamps = timestampsFor(track);
  if (timestamps) activity.timestamps = timestamps;

  const image = imageFor(track);
  if (image) activity.assets = track.album ? { large_image: image, large_text: fit(track.album) } : { large_image: image };

  if (songUrl) activity.buttons = [{ label: 'Listen on YouTube Music', url: songUrl }];
  return activity;
}
