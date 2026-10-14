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
    view.innerHTML = `<section class="dashboard-command"><div><span class="command-kicker">Live connection quality</span><h2>Network Test</h2><p>Measure latency and internet speed on demand. No test runs in the background.</p></div><span class="network-test-server">TEST SERVER · CLOUDFLARE</span></section>
      <section class="panel network-test-panel">
        <div class="network-test-toolbar"><div><h2>Connection quality</h2><p class="panel-desc">Quick uses about 7 MB. Full uses about 30 MB. Results may vary with Wi-Fi, VPNs, and server load.</p></div><div class="network-test-actions"><button class="btn" id="network-test-quick">Quick test</button><button class="btn btn-ghost" id="network-test-full">Full test</button><button class="btn btn-danger" id="network-test-cancel" hidden>Cancel</button></div></div>
        <div class="network-test-progress"><i id="network-test-progress"></i></div>
        <div class="network-test-phase" id="network-test-phase">READY</div>
        <div class="network-test-metrics">
          <article><span>Ping</span><strong id="network-test-ping">—</strong><small>ms</small></article>
          <article><span>Jitter</span><strong id="network-test-jitter">—</strong><small>ms</small></article>
          <article><span>Packet loss</span><strong id="network-test-loss">—</strong><small>%</small></article>
          <article class="download"><span>Download</span><strong id="network-test-download">—</strong><small>Mbps</small></article>
          <article class="upload"><span>Upload</span><strong id="network-test-upload">—</strong><small>Mbps</small></article>
        </div>
        <div class="network-test-chart"><svg id="network-test-chart" viewBox="0 0 900 230" preserveAspectRatio="none" aria-label="Live network test chart"></svg><div class="network-chart-legend"><span class="ping">Ping</span><span class="down">Download</span><span class="up">Upload</span></div></div>
        <div class="network-test-message" id="network-test-message">Choose Quick test or Full test to begin.</div>
      </section>`;
    document.querySelector('.main').appendChild(view);
    document.getElementById('network-test-quick').onclick = () => start('quick');
    document.getElementById('network-test-full').onclick = () => start('full');
    document.getElementById('network-test-cancel').onclick = cancel;
    if (location.hash === '#networktest') switchView('networktest');
  }

  async function start(mode) {
    stopPolling();
    reset();
    setRunning(true);
    try {
      const result = await api('POST', '/network-test/start', { mode });
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
    document.getElementById('network-test-progress').style.width = `${Math.max(0, Math.min(100, Number(result.progress || 0)))}%`;
    document.getElementById('network-test-phase').textContent = phaseLabel(result.phase);
    draw(result.samples || []);
  }

  function draw(samples) {
    const svg = document.getElementById('network-test-chart');
    if (!svg) return;
    const maxAt = Math.max(1, ...samples.map(x => Number(x.at_ms || 0)));
    const series = [
      ['ping_ms', 'network-line-ping'],
      ['download_mbps', 'network-line-down'],
      ['upload_mbps', 'network-line-up']
    ];
    const grid = [46, 92, 138, 184].map(y => `<path class="network-chart-grid" d="M0 ${y}H900"/>`).join('');
    const lines = series.map(([key, cls]) => {
      const points = samples.filter(x => Number(x[key]) > 0);
      if (!points.length) return '';
      const maxValue = Math.max(1, ...points.map(x => Number(x[key])));
      const path = points.map((x, index) => `${index ? 'L' : 'M'}${Number(x.at_ms || 0) / maxAt * 900} ${215 - Number(x[key]) / maxValue * 190}`).join(' ');
      return `<path class="${cls}" d="${path}"/>`;
    }).join('');
    svg.innerHTML = grid + lines;
  }

  function reset() {
    latest = null;
    ['network-test-ping','network-test-jitter','network-test-loss','network-test-download','network-test-upload'].forEach(id => document.getElementById(id).textContent = '—');
    document.getElementById('network-test-progress').style.width = '0%';
    document.getElementById('network-test-chart').innerHTML = '';
    message('Preparing test…');
  }

  function setRunning(running) {
    document.getElementById('network-test-quick').disabled = running;
    document.getElementById('network-test-full').disabled = running;
    document.getElementById('network-test-cancel').hidden = !running;
  }

  function stopPolling() { if (pollTimer) clearInterval(pollTimer); pollTimer = 0; }
  function message(text, error) { const el=document.getElementById('network-test-message'); el.textContent=text; el.classList.toggle('error',!!error); }
  function value(id, number, visible) { document.getElementById(id).textContent = visible ? Number(number || 0).toFixed(1) : '—'; }
  function phaseLabel(phase) { return ({latency:'MEASURING LIVE PING',download:'TESTING DOWNLOAD',upload:'TESTING UPLOAD',complete:'TEST COMPLETE',cancelled:'CANCELLED',failed:'FAILED'})[phase] || 'READY'; }
  function formatBytes(bytes) { return `${(Number(bytes || 0) / 1024 / 1024).toFixed(1)} MB`; }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount); else mount();
})();
