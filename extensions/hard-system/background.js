const API = 'http://127.0.0.1:8080/extension/v1/analyze';
const FOCUS_API = 'http://127.0.0.1:8080/extension/v1/focus/check';

chrome.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (!['hard-system-analyze', 'focus-capsule-check'].includes(message?.type)) return false;
  const focus = message.type === 'focus-capsule-check';
  fetch(focus ? FOCUS_API : API, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(message.page || {}),
    cache: 'no-store',
    credentials: 'omit'
  })
    .then(async response => {
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || `Local service returned ${response.status}`);
      if (focus) await chrome.storage.local.set({ focusCapsuleLast: data, hardSystemConnected: true });
      else await chrome.storage.local.set({ hardSystemLast: data, hardSystemConnected: true });
      sendResponse({ ok: true, result: data });
    })
    .catch(async error => {
      await chrome.storage.local.set({ hardSystemConnected: false, hardSystemError: error.message });
      sendResponse({ ok: false, error: error.message });
    });
  return true;
});
