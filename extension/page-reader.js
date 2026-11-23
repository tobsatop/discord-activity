// @ts-check
// Runs inside the YouTube Music page itself ("MAIN" world) so it can use the
// page's player API. It only reads. When the song, play state or position
// changes, it posts the current song to bridge.js.
(() => {
  const SOURCE = 'ytm-discord-presence';
  const CHECK_EVERY_MS = 1000;
  const SETTLE_MS = 400; // metadata lags the <video> a little on song change
  const SEEK_TOLERANCE_MS = 2000;

  /** @type {Track | null} */
  let lastSent = null;
  let settleTimer = 0;

  /** @param {readonly MediaImage[] | undefined} artwork */
  function pickArtwork(artwork) {
    let best = '';
    let bestSize = -1;
    for (const image of artwork ?? []) {
      const size = Number.parseInt(String(image.sizes ?? '').split('x')[0], 10) || 0;
      if (image.src && size >= bestSize) {
        best = image.src;
        bestSize = size;
      }
    }
    return best;
  }

  /** @returns {Track | null} */
  function readTrack() {
    const player = /** @type {YtPlayerElement | null} */ (document.getElementById('movie_player'));
    const video = player?.querySelector('video');
    if (!player || !video) return null;

    const data = typeof player.getVideoData === 'function' ? player.getVideoData() : undefined;
    const meta = navigator.mediaSession?.metadata;
    const title = meta?.title || data?.title || '';
    if (!title) return null;

    const now = Date.now();
    const rate = video.playbackRate || 1;
    const positionMs = (video.currentTime * 1000) / rate;
    const hasLength = Number.isFinite(video.duration) && video.duration > 0;
    const isAd = player.classList.contains('ad-showing') || player.classList.contains('ad-interrupting');

    return {
      videoId: data?.video_id || '',
      title,
      artist: meta?.artist || data?.author || '',
      album: meta?.album || '',
      artworkUrl: pickArtwork(meta?.artwork),
      startMs: Math.round(now - positionMs),
      endMs: hasLength ? Math.round(now - positionMs + (video.duration * 1000) / rate) : 0,
      playing: !video.paused && !video.ended && !isAd,
    };
  }

  /** @param {Track} t */
  const identity = (t) => [t.videoId, t.title, t.artist, t.album, t.artworkUrl, t.playing].join('\u0000');

  /** @param {Track | null} track */
  function differsFromLastSent(track) {
    if (!track || !lastSent) return track !== lastSent;
    if (identity(track) !== identity(lastSent)) return true;
    if (!track.playing) return false; // while paused the computed start time drifts; ignore it
    if ((track.endMs === 0) !== (lastSent.endMs === 0)) return true;
    return Math.abs(track.startMs - lastSent.startMs) > SEEK_TOLERANCE_MS;
  }

  /** @param {Track | null} track */
  function post(track) {
    lastSent = track;
    window.postMessage({ source: SOURCE, kind: 'track', track }, location.origin);
  }

  function check() {
    if (!differsFromLastSent(readTrack())) return;
    clearTimeout(settleTimer);
    settleTimer = window.setTimeout(() => {
      const track = readTrack();
      if (differsFromLastSent(track)) post(track);
    }, SETTLE_MS);
  }

  // bridge.js asks for the current song when it (re)starts.
  window.addEventListener('message', (event) => {
    if (event.source !== window || event.data?.source !== SOURCE || event.data?.kind !== 'resend') return;
    post(readTrack());
  });

  // Media events don't bubble, but they can be caught on the way down.
  for (const type of ['play', 'pause', 'seeked', 'ended', 'durationchange', 'ratechange', 'emptied']) {
    document.addEventListener(type, check, true);
  }
  setInterval(check, CHECK_EVERY_MS);
  check();
})();
