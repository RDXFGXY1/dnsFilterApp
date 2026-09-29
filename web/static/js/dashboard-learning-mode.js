(function () {
  const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]));

  function mount() {
    VIEW_META.learning = ['Learning Mode', 'Build a local connection baseline for one Windows application'];

    const nav = document.querySelector('.sidenav');
    const anchor = nav?.querySelector('[data-view="decisions"]') || nav?.querySelector('[data-view="freeze"]');
    const link = document.createElement('a');
    link.href = '#';
    link.dataset.view = 'learning';
    link.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3a5 5 0 0 0-5 5v2a4 4 0 0 0-2 3.5V17h14v-3.5a4 4 0 0 0-2-3.5V8a5 5 0 0 0-5-5Z"/><path d="M9 21h6M9 8h6M12 8v5"/></svg>Learning Mode';
    link.onclick = event => {
      event.preventDefault();
      switchView('learning');
      loadLearningSessions();
    };
    anchor?.insertAdjacentElement('beforebegin', link);

    const view = document.createElement('section');
    view.className = 'view';
    view.dataset.view = 'learning';
    view.innerHTML = `
      <section class="dashboard-command">
        <div>
          <span class="command-kicker">Observe · Learn · Explain</span>
          <h2>Learning Mode</h2>
          <p>Watch one application's connection pattern and build a baseline before creating stricter network rules.</p>
        </div>
      </section>
      <section class="panel">
        <div class="panel-head"><div><h2>Start a learning session</h2><p>Only destination addresses, ports, and observation counts are stored. Traffic content is never captured.</p></div></div>
        <div class="input-row learning-controls">
          <input id="learning-path" placeholder="Full path to application.exe" autocomplete="off">
          <select id="learning-duration" aria-label="Learning duration">
            <option value="5">5 minutes</option>
            <option value="15" selected>15 minutes</option>
            <option value="30">30 minutes</option>
            <option value="60">1 hour</option>
          </select>
          <button class="btn" onclick="startLearningMode()">Start learning</button>
        </div>
        <p class="learning-hint">Start the application and use it normally during the session. Learning continues even when this dashboard is closed.</p>
      </section>
      <section class="panel">
        <div class="panel-head"><div><h2>Connection baselines</h2><p>Use these results to understand what the application normally contacts.</p></div><button class="btn btn-ghost" onclick="loadLearningSessions()">Refresh</button></div>
        <div id="learning-sessions" class="intelligence-cards"><div class="no-data">Loading…</div></div>
      </section>`;
    document.querySelector('.main').appendChild(view);
  }

  function formatTime(value) {
    if (!value) return '—';
    return new Date(value).toLocaleString();
  }

  async function start() {
    const path = document.getElementById('learning-path').value.trim();
    const minutes = Number(document.getElementById('learning-duration').value);
    if (!path) return showNotification('Enter the full path to an application.', true);
    const result = await api('POST', '/network-intelligence/learning/start', { path, minutes });
    if (result?.error) return showNotification(result.error, true);
    showNotification('Learning Mode started');
    load();
  }

  async function stop(id) {
    const result = await api('POST', `/network-intelligence/learning/${id}/stop`, {});
    if (result?.error) return showNotification(result.error, true);
    showNotification('Learning session stopped');
    load();
  }

  async function load() {
    const box = document.getElementById('learning-sessions');
    if (!box) return;
    const items = await api('GET', '/network-intelligence/learning');
    if (items?.error) {
      box.innerHTML = `<div class="no-data">${esc(items.error)}</div>`;
      return;
    }
    box.innerHTML = items.length ? items.map(item => `
      <article class="intelligence-card learning-session-card">
        <div class="intelligence-score ${item.active ? 'high' : ''}">${item.active ? 'LIVE' : 'DONE'}</div>
        <div class="learning-session-main">
          <strong>${esc(item.application)}</strong>
          <small title="${esc(item.path)}">${esc(item.path)}</small>
          <small>${formatTime(item.started_at)} → ${formatTime(item.stopped_at || item.ends_at)}</small>
        </div>
        <div class="learning-metrics">
          <span><b>${item.destinations}</b> destinations</span>
          <span><b>${item.ports}</b> ports</span>
          <span><b>${item.observations}</b> observations</span>
        </div>
        <div class="learning-actions"><button class="btn btn-ghost" onclick="loadIsolationPlan(${item.id},this)">Isolation plan</button>${item.active ? `<button class="btn btn-danger" onclick="stopLearningMode(${item.id})">Stop</button>` : ''}</div>
        <div class="learning-baseline" id="learning-baseline-${item.id}" hidden></div>
      </article>`).join('') : '<div class="no-data">No learning sessions yet.</div>';
  }

  async function isolationPlan(id, button) {
    const box = document.getElementById(`learning-baseline-${id}`);
    if (!box) return;
    if (!box.hidden) {
      box.hidden = true;
      button.textContent = 'Isolation plan';
      return;
    }
    box.hidden = false;
    button.textContent = 'Hide plan';
    if (box.dataset.loaded === '1') return;
    box.innerHTML = '<span class="no-data">Building plan from the learned baseline…</span>';
    const result = await api('GET', `/network-intelligence/learning/${id}/baseline`);
    if (result?.error) {
      box.innerHTML = `<span class="no-data">${esc(result.error)}</span>`;
      return;
    }
    const connections = result.connections || [];
    box.dataset.loaded = '1';
    box.dataset.plan = JSON.stringify(result, null, 2);
    box.innerHTML = `
      <div class="baseline-head"><div><strong>Review-only isolation plan</strong><small>${esc(result.note)}</small></div><button class="btn btn-ghost" onclick="copyIsolationPlan(${id})">Copy JSON</button></div>
      ${connections.length ? `<div class="baseline-table"><div class="baseline-row baseline-labels"><span>Destination</span><span>Port</span><span>Protocol</span><span>Seen</span><span>Risk</span></div>${connections.map(item => `<div class="baseline-row"><span>${esc(item.remote_ip)}</span><span>${item.remote_port}</span><span>${esc(item.protocol)}</span><span>${item.observations}</span><span>${item.risk}</span></div>`).join('')}</div>` : '<span class="no-data">No connections were captured during this learning session.</span>'}`;
  }

  async function copyPlan(id) {
    const value = document.getElementById(`learning-baseline-${id}`)?.dataset.plan;
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
      showNotification('Isolation plan copied as JSON');
    } catch (_) {
      showNotification('Clipboard access was denied', true);
    }
  }

  window.startLearningMode = start;
  window.stopLearningMode = stop;
  window.loadLearningSessions = load;
  window.loadIsolationPlan = isolationPlan;
  window.copyIsolationPlan = copyPlan;
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount);
  else mount();
})();
