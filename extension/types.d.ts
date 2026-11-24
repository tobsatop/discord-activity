// Shared shapes for the extension scripts (type-checking only; Chrome never loads this file).

/** The song as read from the YouTube Music page. Same shape as `Track` in internal/activity/activity.go. */
interface Track {
  /** YouTube video id (11 chars), or '' if unknown. */
  videoId: string;
  title: string;
  artist: string;
  album: string;
  artworkUrl: string;
  /** Wall-clock time (ms) the song would have started, given the current position. */
  startMs: number;
  /** Wall-clock time (ms) the song will end, or 0 if the length is unknown (e.g. live). */
  endMs: number;
  playing: boolean;
}

/** The parts of YouTube's player element API that page-reader.js uses. */
interface YtPlayerElement extends HTMLElement {
  getVideoData?: () => { video_id?: string; title?: string; author?: string } | undefined;
}

/** Messages from the helper (internal/helper). */
interface HostStatus {
  type: 'status';
  state: 'connecting' | 'connected' | 'unavailable' | 'error';
  detail?: string;
}
