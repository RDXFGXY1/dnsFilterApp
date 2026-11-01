(function () {
  let activeID = '';
  let pollTimer = 0;
  let latest = null;

  function mount() {
    if (document.querySelector('section.view[data-view="networktest"]')) return;
    VIEW_META.networktest = ['Network Test', 'Live ping, jitter, packet loss, download and upload speed'];
    const nav = document.querySelector('.sidenav');
    const connections = nav?.querySelector('[data-view="connections"]');
    let link = nav?.querySelector('a[data-view="networktest"]');
    if (!link) {
      link = document.createElement('a');
      link.href = '#networktest';
      link.dataset.view = 'networktest';
      link.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 16l4-4 4 3 7-8"/><path d="M15 7h4v4"/><path d="M4 20h16"/></svg>Network Test';
      if (connections) connections.insertAdjacentElement('beforebegin', link); else nav?.appendChild(link);
    }
    link.onclick = event => { event.preventDefault(); switchView('networktest'); };

    const view = document.createElement('section');
    view.className = 'view';
    view.dataset.view = 'networktest';
    view.innerHTML = `<div class="network-lab">
      <section class="network-hero">
        <div class="network-hero-copy"><span class="network-eyebrow"><i></i> CONNECTION DIAGNOSTICS</span><h2>Know what your connection can do.</h2><p>Measure response time, stability, and real transfer speed. Run a test when you need it; nothing runs in the background.</p><div class="network-hero-meta"><span>◉ Cloudflare test server</span><span>◷ On demand only</span></div></div>
        <div class="network-verdict" id="network-test-verdict"><span class="network-verdict-label">CONNECTION VERDICT</span><strong id="network-test-grade">Ready to test</strong><p id="network-test-summary">Start a test to see a clear reading of your connection.</p><span class="network-verdict-stamp" id="network-test-stamp">NO RECENT TEST</span></div>
      </section>
      <section class="network-control panel"><div class="network-control-copy"><span class="network-section-label">RUN A DIAGNOSTIC</span><h3>Choose your test</h3><p>Every test checks ping, jitter, packet loss, download, and upload.</p></div><div class="network-test-actions"><button class="btn" id="network-test-quick">Quick · 7 MB <span>→</span></button><button class="btn btn-ghost" id="network-test-full">Full · 30 MB <span>→</span></button><button class="btn btn-danger" id="network-test-cancel" hidden>Cancel test</button></div></section>
      <section class="network-custom panel"><div><span class="network-section-label">CUSTOM ALL-IN-ONE TEST</span><h3>Set your own limit</h3><p>Choose a total data budget or run at full speed for a fixed time. Download and upload share the budget; latency checks run first.</p></div><div class="network-custom-controls"><select id="network-test-limit-mode" aria-label="Test limit type"><option value="data">Data limit</option><option value="time">Time limit</option></select><label><input id="network-test-limit-value" type="number" min="5" max="500" value="100" inputmode="numeric"><span id="network-test-limit-unit">MB total</span></label><button class="btn" id="network-test-custom">Run custom test →</button></div><p id="network-test-limit-help">5–500 MB total; 80% download and 20% upload. 100 MB is about 80 MB down + 20 MB up.</p></section>
      <section class="network-progress-panel panel"><div class="network-progress-top"><div><span class="network-section-label">TEST PROGRESS</span><strong id="network-test-phase">READY</strong></div><span id="network-test-percent">0%</span></div><div class="network-test-progress"><i id="network-test-progress"></i></div><div class="network-steps"><span data-phase="latency"><b>01</b> Latency</span><span data-phase="download"><b>02</b> Download</span><span data-phase="upload"><b>03</b> Upload</span><span data-phase="complete"><b>04</b> Results</span></div></section>
      <section class="network-results"><div class="network-results-head"><div><span class="network-section-label">YOUR RESULTS</span><h3>Connection at a glance</h3></div><span>Measured from this computer</span></div><div class="network-test-metrics">
        <article class="latency"><span>Ping</span><div><strong id="network-test-ping">—</strong><small>ms</small></div><p>Response time</p></article>
        <article class="jitter"><span>Jitter</span><div><strong id="network-test-jitter">—</strong><small>ms</small></div><p>Timing variation</p></article>
        <article class="loss"><span>Packet loss</span><div><strong id="network-test-loss">—</strong><small>%</small></div><p>Missed responses</p></article>
        <article class="download"><span>Download</span><div><strong id="network-test-download">—</strong><small>Mbps</small></div><p>Receiving data</p></article>
        <article class="upload"><span>Upload</span><div><strong id="network-test-upload">—</strong><small>Mbps</small></div><p>Sending data</p></article>
      </div></section>
      <section class="network-chart-panel panel"><div class="network-chart-head"><div><span class="network-section-label">SPEED COMPARISON</span><h3>Download versus upload</h3><p>These bars compare the two speeds from this test. Longer means faster; the exact Mbps values are shown on the right.</p></div></div><div class="network-speed-bars"><div class="network-speed-row"><span>Download</span><div class="network-speed-track"><i class="down" id="network-speed-down-bar"></i></div><strong id="network-speed-down-value">— Mbps</strong></div><div class="network-speed-row"><span>Upload</span><div class="network-speed-track"><i class="up" id="network-speed-up-bar"></i></div><strong id="network-speed-up-value">— Mbps</strong></div></div><p class="network-speed-note" id="network-test-mode-info">Run a test to compare speeds and see how much data it used.</p></section>
      <section class="network-history panel"><div class="network-chart-head"><div><span class="network-section-label">SAVED RESULTS</span><h3>Network Test history</h3><p>Last 30 tests stay on this computer, including after a restart.</p></div><button class="btn btn-ghost" id="network-test-history-refresh">Refresh history</button></div><p class="network-history-comparison" id="network-history-comparison">Run another test in the same mode to compare results.</p><div class="network-history-list" id="network-test-history"><div class="no-data">Loading previous tests…</div></div></section>
      <div class="network-test-message" id="network-test-message" role="status">Choose Quick or Full to begin.</div>
    </div>`;
    document.querySelector('.main').appendChild(view);
    document.getElementById('network-test-quick').onclick = () => start('quick');
    document.getElementById('network-test-full').onclick = () => start('full');
    document.getElementById('network-test-custom').onclick = startCustom;
    document.getElementById('network-test-limit-mode').onchange = updateLimitMode;
    document.getElementById('network-test-cancel').onclick = cancel;
    document.getElementById('network-test-history-refresh').onclick = loadHistory;
    loadHistory();
    if (location.hash === '#networktest') switchView('networktest');
  }

  function updateLimitMode() {
    const timed = document.getElementById('network-test-limit-mode').value === 'time';
    const input = document.getElementById('network-test-limit-value');
    input.min = '5'; input.max = timed ? '60' : '500'; input.value = timed ? '15' : '100';
    document.getElementById('network-test-limit-unit').textContent = timed ? 'seconds' : 'MB total';
    document.getElementById('network-test-limit-help').textContent = timed ? '5–60 seconds at full speed, split equally between download and upload. Data use depends on your connection.' : '5–500 MB total; 80% download and 20% upload. 100 MB is about 80 MB down + 20 MB up.';
  }

  function startCustom() {
    const timed = document.getElementById('network-test-limit-mode').value === 'time';
    const value = Number(document.getElementById('network-test-limit-value').value);
    if (!Number.isInteger(value) || value < 5 || value > (timed ? 60 : 500)) return message(timed ? 'Choose 5–60 seconds.' : 'Choose 5–500 MB.', true);
    start(timed ? 'custom_time' : 'custom_data', timed ? { duration_seconds: value } : { limit_mb: value });
  }

  async function start(mode, options = {}) {
    stopPolling();
    reset();
    setRunning(true);
    try {
      const result = await api('POST', '/network-test/start', { mode, ...options });
      if (!result?.id) {
        setRunning(false);
        return message(result?.error || 'The Network Test API is unavailable. Rebuild and restart the development server.', true);
      }
      activeID = result.id;
      latest = result;
      render(result);
      pollTimer = window.setInterval(poll, 400);
      poll();
    } catch (error) {
      setRunning(false);
      activeID = '';
      message('The Network Test API is unavailable. Rebuild and restart the development server, then refresh this page.', true);
    }
  }

  async function poll() {
    if (!activeID) return;
    try {
      const result = await api('GET', `/network-test/${encodeURIComponent(activeID)}`);
      if (!result || result.error) {
        stopPolling(); setRunning(false); activeID = ''; message(result?.error || 'The live test stopped responding', true); return;
      }
      latest = result;
      render(result);
      if (result.status !== 'running') {
        stopPolling(); setRunning(false); activeID = '';
        if (result.status === 'complete') message(`Complete · ${formatBytes(result.bytes_downloaded)} downloaded and ${formatBytes(result.bytes_uploaded)} uploaded.`);
        else if (result.status === 'cancelled') message('Test cancelled.');
        else message(result.error || 'Network test failed.', true);
        window.setTimeout(loadHistory, 500);
      }
    } catch (error) {
      stopPolling();
      setRunning(false);
      activeID = '';
      message('Lost contact with the local service. Restart the development server and try again.', true);
    }
  }

  async function cancel() {
    if (!activeID) return;
    try {
      await api('DELETE', `/network-test/${encodeURIComponent(activeID)}`);
      message('Stopping test…');
    } catch (error) {
      stopPolling();
      setRunning(false);
      activeID = '';
      message('Could not cancel the test because the local service is unavailable.', true);
    }
  }

  function render(result) {
    const hasLatency = (result.samples || []).some(sample => Number(sample.ping_ms) > 0);
    value('network-test-ping', result.ping_ms, hasLatency);
    value('network-test-jitter', result.jitter_ms, hasLatency);
    value('network-test-loss', result.packet_loss, Number(result.progress) > 0);
    value('network-test-download', result.download_mbps, Number(result.bytes_downloaded) > 0);
    value('network-test-upload', result.upload_mbps, Number(result.bytes_uploaded) > 0);
    const progress = Math.max(0, Math.min(100, Number(result.progress || 0)));
    document.getElementById('network-test-progress').style.width = `${progress}%`;
    document.getElementById('network-test-percent').textContent = `${Math.round(progress)}%`;
    document.getElementById('network-test-phase').textContent = phaseLabel(result.phase);
    document.querySelectorAll('.network-steps [data-phase]').forEach(step => {
      const order = ['latency', 'download', 'upload', 'complete'];
      const current = result.status === 'complete' ? 3 : order.indexOf(result.phase);
      const position = order.indexOf(step.dataset.phase);
      step.classList.toggle('done', position < current || result.status === 'complete');
      step.classList.toggle('current', position === current && result.status === 'running');
    });
    renderVerdict(result);
    draw(result);
  }

  function renderVerdict(result) {
    const box = document.getElementById('network-test-verdict');
    const grade = document.getElementById('network-test-grade');
    const summary = document.getElementById('network-test-summary');
    const stamp = document.getElementById('network-test-stamp');
    box.dataset.grade = result.status === 'complete' ? 'good' : 'running';
    if (result.status === 'running') {
      grade.textContent = 'Measuring…';
      summary.textContent = `Checking ${result.phase || 'connection'} performance. Your results will appear below.`;
      stamp.textContent = `${testLabel(result).toUpperCase()} IN PROGRESS`;
      return;
    }
    if (result.status !== 'complete') {
      box.dataset.grade = 'muted';
      grade.textContent = result.status === 'cancelled' ? 'Test cancelled' : 'Could not finish';
      summary.textContent = result.error || 'Start another test when your connection is ready.';
      stamp.textContent = (result.status || 'stopped').toUpperCase();
      return;
    }
    const loss = Number(result.packet_loss || 0), ping = Number(result.ping_ms || 0), down = Number(result.download_mbps || 0);
    if (loss > 2) {
      box.dataset.grade = 'poor'; grade.textContent = 'Unstable connection';
      summary.textContent = 'Packet loss is high. Check Wi-Fi signal or your network link before trusting the speed result.';
    } else if (ping > 120 || down < 10) {
      box.dataset.grade = 'moderate'; grade.textContent = 'Needs attention';
      summary.textContent = ping > 120 ? 'Response time is high. Pages and calls may feel delayed.' : 'Download speed is limited for larger transfers.';
    } else if (ping > 60) {
      box.dataset.grade = 'moderate'; grade.textContent = 'Good speed, slower response';
      summary.textContent = 'Transfer speed is healthy, while the response time could feel slower in interactive apps.';
    } else {
      grade.textContent = 'Strong connection';
      summary.textContent = 'Low packet loss, responsive latency, and solid download speed in this test.';
    }
    stamp.textContent = `${testLabel(result).toUpperCase()} · ${new Date(result.finished_at || Date.now()).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`;
  }

  function testLabel(result) {
    if (result.mode === 'custom_data') return `Custom · ${result.limit_mb} MB`;
    if (result.mode === 'custom_time') return `Custom · ${result.duration_seconds} sec`;
    return `${result.mode || 'quick'} test`;
  }

  function draw(result) {
    const down = Math.max(0, Number(result.download_mbps || 0));
    const up = Math.max(0, Number(result.upload_mbps || 0));
    const max = Math.max(down, up, 1);
    document.getElementById('network-speed-down-bar').style.width = `${down / max * 100}%`;
    document.getElementById('network-speed-up-bar').style.width = `${up / max * 100}%`;
    document.getElementById('network-speed-down-value').textContent = down ? `${down.toFixed(1)} Mbps` : '— Mbps';
    document.getElementById('network-speed-up-value').textContent = up ? `${up.toFixed(1)} Mbps` : '— Mbps';
    const mode = testLabel(result);
    document.getElementById('network-test-mode-info').textContent = `${mode} · ${formatBytes(result.bytes_downloaded)} downloaded · ${formatBytes(result.bytes_uploaded)} uploaded`;
  }

  function reset() {
    latest = null;
    ['network-test-ping','network-test-jitter','network-test-loss','network-test-download','network-test-upload'].forEach(id => document.getElementById(id).textContent = '—');
    document.getElementById('network-test-progress').style.width = '0%';
    document.getElementById('network-test-percent').textContent = '0%';
    document.getElementById('network-test-phase').textContent = 'PREPARING TEST';
    document.getElementById('network-speed-down-bar').style.width = '0%';
    document.getElementById('network-speed-up-bar').style.width = '0%';
    document.getElementById('network-speed-down-value').textContent = '— Mbps';
    document.getElementById('network-speed-up-value').textContent = '— Mbps';
    document.getElementById('network-test-mode-info').textContent = 'Measuring…';
    document.querySelectorAll('.network-steps [data-phase]').forEach(step => step.classList.remove('done', 'current'));
    document.getElementById('network-test-verdict').dataset.grade = 'running';
    document.getElementById('network-test-grade').textContent = 'Preparing test…';
    document.getElementById('network-test-summary').textContent = 'Checking your connection now.';
    document.getElementById('network-test-stamp').textContent = 'STARTING';
    message('Preparing test…');
  }

  function setRunning(running) {
    document.getElementById('network-test-quick').disabled = running;
    document.getElementById('network-test-full').disabled = running;
    document.getElementById('network-test-custom').disabled = running;
    document.getElementById('network-test-limit-mode').disabled = running;
    document.getElementById('network-test-limit-value').disabled = running;
    document.getElementById('network-test-cancel').hidden = !running;
  }

  function stopPolling() { if (pollTimer) clearInterval(pollTimer); pollTimer = 0; }
  function message(text, error) { const el=document.getElementById('network-test-message'); el.textContent=text; el.classList.toggle('error',!!error); }
  function value(id, number, visible) { document.getElementById(id).textContent = visible ? Number(number || 0).toFixed(1) : '—'; }
  function phaseLabel(phase) { return ({latency:'MEASURING LIVE PING',download:'TESTING DOWNLOAD',upload:'TESTING UPLOAD',complete:'TEST COMPLETE',cancelled:'CANCELLED',failed:'FAILED'})[phase] || 'READY'; }
  function formatBytes(bytes) { return `${(Number(bytes || 0) / 1024 / 1024).toFixed(1)} MB`; }

  async function loadHistory() {
    const host = document.getElementById('network-test-history');
    if (!host) return;
    const records = await api('GET', '/network-test/history');
    if (!Array.isArray(records)) { host.textContent = records?.error || 'Could not load test history.'; return; }
    const completed = records.filter(x => x.status === 'complete');
    const latest = completed[0], previous = latest && completed.slice(1).find(x => x.mode === latest.mode);
    const compare = document.getElementById('network-history-comparison');
    if (latest && previous) {
      const down = Number(latest.download_mbps || 0) - Number(previous.download_mbps || 0);
      const ping = Number(latest.ping_ms || 0) - Number(previous.ping_ms || 0);
      compare.textContent = `Compared with the previous ${testLabel(latest)}: download ${down >= 0 ? '+' : ''}${down.toFixed(1)} Mbps · ping ${ping >= 0 ? '+' : ''}${ping.toFixed(1)} ms. Lower ping is better.`;
    } else compare.textContent = 'Run another test in the same mode to compare results.';
    if (!records.length) { host.innerHTML = '<div class="no-data">No saved tests yet. Run Quick, Full, or Custom to start your history.</div>'; return; }
    host.innerHTML = records.slice(0, 10).map(x => {
      const status = ['complete','failed','cancelled'].includes(x.status) ? x.status : 'unknown';
      const date = new Date(x.finished_at || x.started_at).toLocaleString();
      return `<article class="network-history-row"><div><strong>${escapeHistory(testLabel(x))}</strong><small>${escapeHistory(date)}</small></div><span class="network-history-status ${status}">${status}</span><div><b>${Number(x.download_mbps || 0).toFixed(1)}</b><small>down Mbps</small></div><div><b>${Number(x.upload_mbps || 0).toFixed(1)}</b><small>up Mbps</small></div><div><b>${Number(x.ping_ms || 0).toFixed(1)}</b><small>ping ms</small></div></article>`;
    }).join('');
  }
  function escapeHistory(value) { return String(value ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch])); }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount); else mount();
})();
