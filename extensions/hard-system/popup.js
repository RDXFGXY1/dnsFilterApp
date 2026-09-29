const enabled = document.getElementById('enabled');
const threshold = document.getElementById('threshold');

async function load() {
  const state = await chrome.storage.local.get({ hardSystemEnabled: true, hardSystemThreshold: 70, hardSystemLast: null, hardSystemConnected: false });
  enabled.checked = state.hardSystemEnabled;
  threshold.value = state.hardSystemThreshold;
  document.getElementById('threshold-value').textContent = state.hardSystemThreshold;
  document.getElementById('protection-copy').textContent = state.hardSystemEnabled ? 'Analyzing visible page content' : 'Protection is paused';
  document.getElementById('connection').classList.toggle('offline', !state.hardSystemConnected);
  document.querySelector('#connection b').textContent = state.hardSystemConnected ? 'Local' : 'Offline';
  if (state.hardSystemLast) {
    const score = Math.max(0, Math.min(100, Number(state.hardSystemLast.score) || 0));
    const action = ['allow', 'warn', 'block'].includes(state.hardSystemLast.action) ? state.hardSystemLast.action : 'neutral';
    document.getElementById('score').textContent = score;
    document.getElementById('score-ring').style.setProperty('--score', score);
    document.getElementById('risk-card').className = `risk-card ${action}`;
    document.getElementById('domain').textContent = state.hardSystemLast.domain || '';
    document.getElementById('status').textContent = action === 'block' ? 'High risk · blocked' : action === 'warn' ? 'Review recommended' : 'Looks safe';
  } else {
    document.getElementById('status').textContent = state.hardSystemConnected ? 'Ready to analyze' : 'Service unavailable';
  }
}
enabled.addEventListener('change', async () => {
  await chrome.storage.local.set({ hardSystemEnabled: enabled.checked });
  document.getElementById('protection-copy').textContent = enabled.checked ? 'Analyzing visible page content' : 'Protection is paused';
});
threshold.addEventListener('input', () => {
  threshold.value = Math.max(40, Math.min(100, Number(threshold.value) || 70));
  document.getElementById('threshold-value').textContent = threshold.value;
});
threshold.addEventListener('change', () => {
  chrome.storage.local.set({ hardSystemThreshold: Number(threshold.value) });
});
document.getElementById('dashboard').onclick = () => chrome.tabs.create({ url: 'http://dnsfilter.localhost:8080/#rules' });
load();
