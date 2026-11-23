// @ts-check
// Service worker. Collects the song from every open YouTube Music tab, picks
// the one to show, and hands it to the native helper (host/main.ts), which
// Chrome starts on demand and which talks to the Discord app.
const HOST_NAME = 'local.ytm_discord_presence';
const RETRY_MS = 30_000;

/** Latest song per open YouTube Music tab, keyed by that tab's port. */
/** @type {Map<chrome.runtime.Port, { track: Track | null, at: number }>} */
const ytmTabs = new Map();

/** @type {chrome.runtime.Port | null} */
let hostPort = null;
let hostState = /** @type {HostStatus['state'] | 'missing'} */ ('connecting');
let hostDetail = '';
let sentKey = '';
let retryTimer = 0;

chrome.runtime.onConnect.addListener((port) => {
  if (port.name !== 'ytm-tab') return;
  ytmTabs.set(port, { track: null, at: Date.now() });
  port.onMessage.addListener((msg) => {
    if (msg?.type !== 'track') return;
    ytmTabs.set(port, { track: msg.track ?? null, at: Date.now() });
    sync();
  });
  port.onDisconnect.addListener(() => {
    ytmTabs.delete(port);
    sync();
  });
});

/** The playing song from the tab that changed most recently, or null. */
function currentTrack() {
  /** @type {{ track: Track | null, at: number } | null} */
  let best = null;
  for (const entry of ytmTabs.values()) {
    if (entry.track?.playing && (!best || entry.at > best.at)) best = entry;
  }
  return best?.track ?? null;
}

function sync() {
  if (ytmTabs.size === 0) {
    closeHost(); // no YouTube Music tab left: the helper exits and Discord clears
    return;
  }
  const port = openHost();
  if (port) {
    const track = currentTrack();
    const key = JSON.stringify(track);
    if (key !== sentKey) {
      sentKey = key;
      port.postMessage({ type: 'update', track });
    }
  }
  refreshBadge();
}

/** @returns {chrome.runtime.Port | null} */
function openHost() {
  if (hostPort) return hostPort;
  if (retryTimer) return null; // the helper failed recently; wait before retrying

  const port = chrome.runtime.connectNative(HOST_NAME);
  hostPort = port;
  hostState = 'connecting';
  hostDetail = '';
  sentKey = '';

  port.onMessage.addListener((/** @type {HostStatus} */ msg) => {
    if (msg?.type !== 'status') return;
    hostState = msg.state;
    hostDetail = msg.detail ?? '';
    refreshBadge();
  });

  port.onDisconnect.addListener(() => {
    const error = chrome.runtime.lastError?.message ?? '';
    if (hostPort !== port) return;
    hostPort = null;
    sentKey = '';
    if (ytmTabs.size === 0) return;
    hostState = 'missing';
    hostDetail = error || 'the helper stopped';
    retryTimer = setTimeout(() => {
      retryTimer = 0;
      sync();
    }, RETRY_MS);
    refreshBadge();
  });

  return port;
}

function closeHost() {
  clearTimeout(retryTimer);
  retryTimer = 0;
  if (hostPort) {
    const port = hostPort;
    hostPort = null;
    port.disconnect();
  }
  sentKey = '';
  hostState = 'connecting';
  hostDetail = '';
  refreshBadge();
}

/** Which toolbar icon (extension/icons/<name>-16|32.png) is showing. */
let shownIcon = '';

/**
 * Toolbar icon: red = showing in Discord, grey = nothing showing,
 * grey with an amber/red dot = something to fix (hover for the reason).
 */
function refreshBadge() {
  const track = currentTrack();
  let icon = 'idle';
  let title = 'YouTube Music → Discord';

  if (ytmTabs.size === 0) {
    title += '\nเปิด music.youtube.com แล้วเล่นเพลง';
  } else if (hostState === 'missing') {
    icon = 'error';
    title += `\nเปิดโปรแกรมเสริมไม่ได้ — รันตัวติดตั้งก่อน (README ขั้นที่ 3)\n(${hostDetail})`;
  } else if (hostState === 'unavailable') {
    icon = 'warn';
    title += `\nยังไม่เจอแอป Discord — เปิดแอป Discord บนเครื่องก่อน\n(${hostDetail})`;
  } else if (hostState === 'error') {
    icon = 'error';
    title += `\n${hostDetail}`;
  } else if (hostState === 'connected' && track) {
    icon = 'icon';
    title += `\nกำลังแสดง: ${track.title}${track.artist ? ` — ${track.artist}` : ''}`;
  } else if (hostState === 'connected') {
    title += '\nเชื่อม Discord แล้ว — รอเพลงเล่น';
  } else {
    title += '\nกำลังเชื่อม Discord…';
  }

  if (icon !== shownIcon) {
    shownIcon = icon;
    void chrome.action.setIcon({ path: { 16: `icons/${icon}-16.png`, 32: `icons/${icon}-32.png` } });
  }
  void chrome.action.setBadgeText({ text: '' });
  void chrome.action.setTitle({ title });
}
