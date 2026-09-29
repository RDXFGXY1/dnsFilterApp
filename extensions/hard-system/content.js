(async function () {
  if (location.hostname === '127.0.0.1' || location.hostname.endsWith('.localhost')) return;
  if (sessionStorage.getItem('dns-filter-hard-system-allow') === '1') return;

  const settings = await chrome.storage.local.get({ hardSystemEnabled: true, hardSystemThreshold: 70 });
  if (!settings.hardSystemEnabled) return;

  const description = document.querySelector('meta[name="description"], meta[property="og:description"]')?.content || '';
  const text = (document.body?.innerText || '').slice(0, 120000);
  if (!text.trim()) return;

  const response = await chrome.runtime.sendMessage({
    type: 'hard-system-analyze',
    page: { url: location.href, title: document.title, description, text }
  }).catch(() => null);
  if (!response?.ok) return;

  const result = response.result;
  const threshold = Math.max(40, Math.min(100, Number(settings.hardSystemThreshold) || 70));
  await chrome.storage.local.set({ hardSystemLast: result });
  if (result.score < threshold) return;

  document.documentElement.textContent = '';
  const head = document.createElement('head');
  const style = document.createElement('style');
  style.textContent = `
    :root{color-scheme:dark}*{box-sizing:border-box}body{margin:0;background:#050505;color:#f5f5f5;font-family:Segoe UI,Arial,sans-serif;min-height:100vh;display:grid;place-items:center}
    main{width:min(720px,calc(100% - 32px));border:1px solid #343434;background:#111;padding:36px;border-radius:18px;box-shadow:0 24px 80px #000}
    .score{width:112px;height:112px;border:2px solid #ef5350;border-radius:50%;display:grid;place-items:center;font-size:38px;font-weight:800;color:#ff6b66}
    h1{font-size:30px;margin:24px 0 8px}p{line-height:1.6;color:#bdbdbd}.domain{color:#fff;font-weight:700}.reason{border-left:3px solid #ef5350;padding:12px 16px;background:#191313}
    .actions{display:flex;gap:10px;margin-top:24px;flex-wrap:wrap}button{border:1px solid #555;background:#222;color:#fff;padding:11px 18px;border-radius:10px;font-weight:700;cursor:pointer}button.primary{background:#365fc4;border-color:#557dea}
    small{display:block;margin-top:22px;color:#858585}
  `;
  head.append(style);
  const body = document.createElement('body');
  const main = document.createElement('main');
  const score = document.createElement('div'); score.className = 'score'; score.textContent = result.score;
  const title = document.createElement('h1'); title.textContent = 'Hard System blocked this page';
  const domain = document.createElement('p'); domain.className = 'domain'; domain.textContent = result.domain;
  const reason = document.createElement('p'); reason.className = 'reason'; reason.textContent = (result.reasons || []).slice(0, 3).join(' · ');
  const actions = document.createElement('div'); actions.className = 'actions';
  const back = document.createElement('button'); back.className = 'primary'; back.textContent = 'Go back'; back.onclick = () => history.back();
  const allow = document.createElement('button'); allow.textContent = 'Allow once'; allow.onclick = () => { sessionStorage.setItem('dns-filter-hard-system-allow', '1'); location.reload(); };
  const note = document.createElement('small'); note.textContent = 'Analyzed locally. Page text was not saved. You can review the score in DNS Filter → Rules & Intelligence.';
  actions.append(back, allow); main.append(score, title, domain, reason, actions, note); body.append(main);
  document.documentElement.append(head, body);
})();
