// @ts-check
// Runs on YouTube Music in the extension's own (isolated) world. Passes the
// song from page-reader.js to background.js, and reconnects if the service
// worker restarts.
(() => {
  const SOURCE = 'ytm-discord-presence';

  /** @type {chrome.runtime.Port | null} */
  let port = null;
  /** @type {Track | null} */
  let track = null;

  function send() {
    try {
      port?.postMessage({ type: 'track', track });
    } catch {
      port = null; // disconnected; onDisconnect will reconnect
    }
  }

  function connect() {
    if (!chrome.runtime?.id) return; // extension was reloaded: this tab needs a refresh
    try {
      port = chrome.runtime.connect({ name: 'ytm-tab' });
    } catch {
      return;
    }
    port.onDisconnect.addListener(() => {
      void chrome.runtime.lastError;
      port = null;
      setTimeout(connect, 1000);
    });
    send();
  }

  window.addEventListener('message', (event) => {
    if (event.source !== window || event.origin !== location.origin) return;
    const data = event.data;
    if (!data || data.source !== SOURCE || data.kind !== 'track') return;
    track = data.track ?? null;
    send();
  });

  connect();
  window.postMessage({ source: SOURCE, kind: 'resend' }, location.origin);
})();
