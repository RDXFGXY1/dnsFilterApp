(function () {
  if (location.hostname === '127.0.0.1' || location.hostname.endsWith('.localhost')) return;

  let lastCheckedURL = '';
  let lastHardURL = '';
  let checking = false;

  function youtubeChannels() {
    if (!/(^|\.)youtube\.com$/i.test(location.hostname) && location.hostname !== 'youtu.be') return [];
    const values = new Set();
    // Only inspect the video-owner area. Looking at every channel link on the
    // page can accidentally lock the capsule to a recommended video's channel.
    document.querySelectorAll('ytd-watch-metadata ytd-channel-name a, ytd-video-owner-renderer ytd-channel-name a, #owner ytd-channel-name a')
      .forEach(link => {
        const href = link.getAttribute('href') || '';
        if (/^\/(channel\/|@)/.test(href)) values.add(href.replace(/^\//, '').split(/[?#]/)[0]);
        if (link.textContent?.trim()) values.add(link.textContent.trim());
      });
    const channelID = document.querySelector('meta[itemprop="channelId"]')?.content?.trim();
    if (channelID) values.add('channel/' + channelID);
    return [...values].slice(0, 20);
  }

  function personality(style, capsule) {
    const name = capsule || 'Focus Capsule';
    if (style === 'roast') return ['Nice try 😏', `You opened a page that ${name} did not invite.`];
    if (style === 'meme') return ['Mission drift detected', 'Your focus capsule has respectfully declined this side quest.'];
    if (style === 'strict') return ['Navigation blocked', `This page is outside ${name}. Finish the mission or end the session first.`];
    return ['Stay with the mission', `You chose ${name} for a reason. Return to the work that matters.`];
  }

  function renderBlock(decision) {
    if (document.documentElement.dataset.dnsFocusBlocked === '1') return;
    document.documentElement.dataset.dnsFocusBlocked = '1';
    const [heading, message] = personality(decision.block_style, decision.capsule);
    document.documentElement.textContent = '';
    const head = document.createElement('head');
    const style = document.createElement('style');
    style.textContent = `:root{color-scheme:dark}*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;background:radial-gradient(circle at 50% 0,#17213b 0,#08090c 42%,#030303 100%);color:#f7f8fb;font-family:Inter,Segoe UI,Arial,sans-serif}main{width:min(720px,100%);padding:42px;border:1px solid #303646;border-radius:24px;background:rgba(15,17,23,.94);box-shadow:0 30px 100px #000}.mark{display:inline-grid;place-items:center;width:52px;height:52px;border-radius:16px;background:#365fc4;color:white;font-size:24px;font-weight:900}h1{margin:24px 0 10px;font-size:clamp(30px,5vw,48px)}p{color:#b9c0d0;line-height:1.65}.reason{margin-top:22px;padding:16px;border-left:3px solid #597fe6;background:#151a27}.actions{display:flex;gap:10px;margin-top:28px;flex-wrap:wrap}button{padding:12px 18px;border:1px solid #485168;border-radius:12px;background:#202633;color:white;font-weight:800;cursor:pointer}button.primary{background:#4169d8;border-color:#6287ed}small{display:block;margin-top:24px;color:#747d91}`;
    head.append(style);
    const body = document.createElement('body');
    const main = document.createElement('main');
    const mark = document.createElement('div'); mark.className = 'mark'; mark.textContent = '✓';
    const title = document.createElement('h1'); title.textContent = heading;
    const copy = document.createElement('p'); copy.textContent = message;
    const reason = document.createElement('p'); reason.className = 'reason'; reason.textContent = decision.reason || 'This destination is outside the active capsule.';
    const actions = document.createElement('div'); actions.className = 'actions';
    const back = document.createElement('button'); back.className = 'primary'; back.textContent = 'Return to focus'; back.onclick = () => history.back();
    const once = document.createElement('button'); once.textContent = 'Allow this page once'; once.onclick = () => { sessionStorage.setItem(decision.hard_system ? 'dns-filter-hard-system-allow' : 'dns-focus-allow:' + location.href, '1'); location.reload(); };
    const note = document.createElement('small'); note.textContent = decision.hard_system
      ? 'Controlled locally by DNS Filter · Hard System'
      : 'Controlled locally by DNS Filter · Discipline Focus Capsule';
    actions.append(back, once); main.append(mark, title, copy, reason, actions, note); body.append(main); document.documentElement.append(head, body);
  }

  async function hardSystemCheck(settings) {
    if (!settings.hardSystemEnabled || sessionStorage.getItem('dns-filter-hard-system-allow') === '1') return;
    const description = document.querySelector('meta[name="description"], meta[property="og:description"]')?.content || '';
    const text = (document.body?.innerText || '').slice(0, 120000);
    if (!text.trim()) return;
    const response = await chrome.runtime.sendMessage({ type: 'hard-system-analyze', page: { url: location.href, title: document.title, description, text } }).catch(() => null);
    if (!response?.ok) return;
    const result = response.result;
    const threshold = Math.max(40, Math.min(100, Number(settings.hardSystemThreshold) || 70));
    if (result.score >= threshold) renderBlock({ capsule: 'Hard System', block_style: 'strict', hard_system: true, reason: (result.reasons || []).slice(0, 3).join(' · ') });
  }

  async function checkPage(force = false) {
    if (checking || document.documentElement.dataset.dnsFocusBlocked === '1') return;
    if (!force && lastCheckedURL === location.href) return;
    checking = true;
    try {
      if (/youtube\.com|youtu\.be/i.test(location.hostname)) await new Promise(resolve => setTimeout(resolve, 800));
      const response = await chrome.runtime.sendMessage({ type: 'focus-capsule-check', page: { url: location.href, title: document.title, channels: youtubeChannels() } }).catch(() => null);
      lastCheckedURL = location.href;
      if (response?.ok && response.result?.active && !response.result.allowed && sessionStorage.getItem('dns-focus-allow:' + location.href) !== '1') {
        renderBlock(response.result);
        return;
      }
      if (lastHardURL !== location.href) {
        const settings = await chrome.storage.local.get({ hardSystemEnabled: true, hardSystemThreshold: 70 });
        await hardSystemCheck(settings);
        lastHardURL = location.href;
      }
    } finally {
      checking = false;
    }
  }

  checkPage(true);
  document.addEventListener('yt-navigate-finish', () => checkPage(true));
  window.addEventListener('popstate', () => checkPage(true));
  setInterval(() => checkPage(true), 3000);
})();
