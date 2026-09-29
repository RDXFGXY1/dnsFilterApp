(function () {
  const esc = value => String(value ?? '').replace(/[&<>"']/g, char => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  })[char]);
  let destinationRows = [];

  function mount() {
    VIEW_META.destinations = ['New Connections', 'New destinations discovered after trusting an application baseline'];
    const nav = document.querySelector('.sidenav');
    const route = nav?.querySelector('[data-view="routewatch"]');
    const link = document.createElement('a');
    link.href = '#';
    link.dataset.view = 'destinations';
    link.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 3v18M3 12h18"/><circle cx="12" cy="12" r="9"/></svg>New Connections';
    link.onclick = event => {
      event.preventDefault();
      switchView('destinations');
      loadNewDestinationAlerts();
    };
    route?.insertAdjacentElement('beforebegin', link);

    const view = document.createElement('section');
    view.className = 'view';
    view.dataset.view = 'destinations';
    view.innerHTML = `<section class="dashboard-command"><div><span class="command-kicker">Application · Destination · Evidence</span><h2>New Connections</h2><p>Review, trust, or block destinations that were not part of an application's trusted baseline.</p></div></section><section class="panel"><div class="panel-head"><div><h2>Destination alerts <span class="tag" id="destination-alert-count">0 NEW</span></h2><p>Trust accepts only that endpoint. Block app stops the application's outbound internet access.</p></div><button class="btn btn-ghost" onclick="loadNewDestinationAlerts()">Refresh</button></div><div class="destination-filter-bar"><input id="destination-search" placeholder="Search app, domain, IP, port, or reason…" oninput="filterNewDestinations()"><select id="destination-risk" onchange="filterNewDestinations()"><option value="0">Any risk</option><option value="20">Risk 20+</option><option value="35">Risk 35+</option><option value="60">Risk 60+</option></select><select id="destination-time" onchange="filterNewDestinations()"><option value="0">Any time</option><option value="1">Last hour</option><option value="24">Last 24 hours</option><option value="168">Last 7 days</option></select><button class="btn btn-ghost" onclick="resetDestinationFilters()">Reset</button></div><div id="destination-alert-list" class="destination-alert-list"><div class="no-data">Loading…</div></div></section><section class="panel destination-notification-panel"><div class="panel-head"><div><span class="command-kicker">Windows alerts</span><h2>Notification settings</h2><p>Control which new destinations can create a desktop notification.</p></div><label class="destination-notification-toggle"><input type="checkbox" id="destination-notifications-enabled"> Enabled</label></div><div class="destination-notification-controls"><label>Minimum risk<input id="destination-notification-risk" type="number" min="0" max="100" value="20"></label><label>Applications (optional)<textarea id="destination-notification-apps" rows="3" placeholder="One executable name or full path per line&#10;Discord.exe&#10;C:\\Program Files\\Example\\app.exe"></textarea></label><button class="btn" onclick="saveDestinationNotifications()">Save notifications</button></div><p class="panel-desc">Leave applications empty to receive alerts from every app. Existing alerts are never replayed.</p></section><section class="panel destination-automation-panel"><div class="panel-head"><div><span class="command-kicker">Policy automation</span><h2>Automatic response</h2><p>Evaluate new, untrusted destinations every 20 seconds.</p></div><label class="destination-notification-toggle"><input type="checkbox" id="automatic-response-enabled"> Enabled</label></div><div class="destination-automation-controls"><label>Minimum risk<input id="automatic-response-risk" type="number" min="0" max="100" value="35"></label><label>Action<select id="automatic-response-action"><option value="review">Review only</option><option value="block">Block application</option></select></label><button class="btn" onclick="saveAutomaticResponse()">Save automation</button></div><div class="automation-warning">Block application creates a Windows Firewall rule for the whole app. Start with Review only and inspect the activity below.</div><div id="automatic-response-events" class="automatic-response-events"><div class="no-data">No automatic actions yet.</div></div></section><section class="panel destination-analytics-panel"><div class="panel-head"><div><span class="command-kicker">Security intelligence</span><h2>Connection analytics</h2><p>Understand which applications and destinations create the most network activity.</p></div><button class="btn btn-ghost" onclick="loadSecurityAnalytics()">Refresh analytics</button></div><div id="destination-analytics-summary" class="destination-analytics-summary"></div><div class="destination-analytics-grid"><div><h3>Top applications</h3><div id="destination-top-apps" class="analytics-bars"></div></div><div><h3>Top destinations</h3><div id="destination-top-targets" class="analytics-bars"></div></div></div></section><section class="panel destination-history-panel"><div class="panel-head"><div><span class="command-kicker">Evidence lifecycle</span><h2>Reports & history</h2><p id="destination-history-summary">Loading retained connection history…</p></div><div class="destination-export-actions"><button class="btn btn-ghost" onclick="exportConnectionReport('csv')">Export CSV</button><button class="btn btn-ghost" onclick="exportConnectionReport('json')">Export JSON</button></div></div><div class="destination-history-controls"><label>Keep connection history<select id="destination-retention"><option value="1">1 day</option><option value="7">7 days</option><option value="14">14 days</option><option value="30">30 days</option><option value="90">90 days</option></select></label><button class="btn" onclick="saveConnectionRetention()">Save retention</button><button class="btn btn-danger" onclick="clearConnectionHistory()">Clear history</button></div><p class="panel-desc">Clearing history removes observed connection events and learning observations. Trusted destination baselines stay protected.</p></section>`;
    document.querySelector('.main').appendChild(view);
  }

  async function trustDestination(button) {
    const body = {
      path: decodeURIComponent(button.dataset.trustPath),
      remote_ip: button.dataset.ip,
      remote_port: Number(button.dataset.port),
      protocol: button.dataset.protocol
    };
    const result = await api('POST', '/network-intelligence/destination-alerts/trust', body);
    if (result?.code === 'vault_locked') {
      return requestVaultUnlock('Unlock the vault to trust this destination.', () => trustDestination(button));
    }
    if (result?.error) return showNotification(result.error, true);
    showNotification(`Trusted ${body.remote_ip}:${body.remote_port} for this application`);
    load();
  }

  async function blockApplication(button) {
    const path = decodeURIComponent(button.dataset.blockPath);
    const name = path.split(/[\\/]/).pop();
    if (!confirm(`Block all outbound internet access for ${name}?`)) return;
    const result = await api('POST', '/app-network/block', { path });
    if (result?.code === 'vault_locked') {
      return requestVaultUnlock('Unlock the vault to block this application.', () => blockApplication(button));
    }
    if (result?.error) return showNotification(result.error, true);
    showNotification(`Blocked ${name}`);
    load();
  }

  async function load() {
    const root = document.getElementById('destination-alert-list');
    if (!root) return;
    const result = await api('GET', '/network-intelligence/destination-alerts');
    if (result?.error) {
      root.innerHTML = `<div class="no-data">${esc(result.error)}</div>`;
      return;
    }
    destinationRows = result.connections || [];
    document.getElementById('destination-alert-count').textContent = `${destinationRows.length} NEW`;
    renderRows();
    loadHistoryStatus();
    loadNotificationSettings();
    loadAutomaticResponse();
    loadSecurityAnalytics();
  }

  function renderRows() {
    const root = document.getElementById('destination-alert-list');
    if (!root) return;
    const query = (document.getElementById('destination-search')?.value || '').trim().toLowerCase();
    const minRisk = Number(document.getElementById('destination-risk')?.value || 0);
    const hours = Number(document.getElementById('destination-time')?.value || 0);
    const cutoff = hours ? Date.now() - hours * 3600000 : 0;
    const rows = destinationRows.filter(item => {
      const text = `${item.application} ${item.path} ${item.hostname || ''} ${item.remote_ip} ${item.remote_port} ${item.protocol} ${item.reasons}`.toLowerCase();
      return (!query || text.includes(query)) && item.risk >= minRisk && (!cutoff || new Date(item.last_seen).getTime() >= cutoff);
    });
    root.innerHTML = rows.length ? rows.map(item => `<article class="destination-alert-card">
      <div class="destination-alert-risk ${item.risk >= 35 ? 'high' : item.risk >= 20 ? 'review' : ''}">${item.risk}</div>
      <div class="destination-alert-app"><span>APPLICATION</span><strong>${esc(item.application)}</strong><small title="${esc(item.path)}">${esc(item.path)}</small></div>
      <div class="destination-alert-target"><span>NEW DESTINATION</span><strong>${item.hostname ? `${esc(item.hostname)} <small>(${esc(item.remote_ip)}:${item.remote_port})</small>` : `${esc(item.remote_ip)}:${item.remote_port}`}</strong><small>${esc(item.protocol)} · ${item.seen} observations · ${esc(item.reasons)}</small></div>
      <time>${new Date(item.last_seen).toLocaleString()}</time>
      <div class="destination-triage-actions">
        <button class="btn btn-ghost" data-app-path="${encodeURIComponent(item.path)}">Review</button>
        <button class="btn btn-ghost" data-trust-path="${encodeURIComponent(item.path)}" data-ip="${esc(item.remote_ip)}" data-port="${item.remote_port}" data-protocol="${esc(item.protocol)}">Trust</button>
        <button class="btn btn-danger" data-block-path="${encodeURIComponent(item.path)}">Block app</button>
      </div>
    </article>`).join('') : '<div class="no-data">No new connections. Trust a baseline from Privacy Center, then future destinations will appear here.</div>';

    root.querySelectorAll('[data-app-path]').forEach(button => {
      button.onclick = () => openConnectionDetails(decodeURIComponent(button.dataset.appPath));
    });
    root.querySelectorAll('[data-trust-path]').forEach(button => {
      button.onclick = () => trustDestination(button);
    });
    root.querySelectorAll('[data-block-path]').forEach(button => {
      button.onclick = () => blockApplication(button);
    });
    const count = document.getElementById('destination-alert-count');
    if (count) count.textContent = rows.length === destinationRows.length ? `${destinationRows.length} NEW` : `${rows.length} OF ${destinationRows.length}`;
  }

  window.filterNewDestinations = renderRows;
  window.resetDestinationFilters = function () {
    document.getElementById('destination-search').value = '';
    document.getElementById('destination-risk').value = '0';
    document.getElementById('destination-time').value = '0';
    renderRows();
  };

  async function loadNotificationSettings() {
    const result = await api('GET', '/network-intelligence/notifications');
    if (!result || result.error) return;
    document.getElementById('destination-notifications-enabled').checked = !!result.enabled;
    document.getElementById('destination-notification-risk').value = result.min_risk;
    document.getElementById('destination-notification-apps').value = (result.applications || []).join('\n');
  }

  window.saveDestinationNotifications = async function () {
    const body = {
      enabled: document.getElementById('destination-notifications-enabled').checked,
      min_risk: Number(document.getElementById('destination-notification-risk').value),
      applications: document.getElementById('destination-notification-apps').value.split(/\r?\n/).map(x => x.trim()).filter(Boolean)
    };
    const result = await api('POST', '/network-intelligence/notifications', body);
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to save notification settings.', window.saveDestinationNotifications);
    if (result?.error) return showNotification(result.error, true);
    showNotification('Desktop notification settings saved');
  };

  async function loadAutomaticResponse() {
    const result = await api('GET', '/network-intelligence/automatic-response');
    if (!result || result.error) return;
    document.getElementById('automatic-response-enabled').checked = !!result.settings.enabled;
    document.getElementById('automatic-response-risk').value = result.settings.min_risk;
    document.getElementById('automatic-response-action').value = result.settings.action;
    const events = document.getElementById('automatic-response-events');
    events.innerHTML = (result.events || []).length ? result.events.map(item => `<div class="automatic-response-row"><span class="tag">${esc(item.action).toUpperCase()}</span><strong>${esc(item.application || item.path.split(/[\\/]/).pop())}</strong><span>${esc(item.remote_ip)}:${item.remote_port}</span><span>Risk ${item.risk}</span><time>${new Date(item.created_at).toLocaleString()}</time></div>`).join('') : '<div class="no-data">No automatic actions yet.</div>';
  }

  window.saveAutomaticResponse = async function () {
    const body = {
      enabled: document.getElementById('automatic-response-enabled').checked,
      min_risk: Number(document.getElementById('automatic-response-risk').value),
      action: document.getElementById('automatic-response-action').value
    };
    if (body.enabled && body.action === 'block' && !confirm(`Automatically block an entire application when a new destination reaches risk ${body.min_risk}+?`)) return;
    const result = await api('POST', '/network-intelligence/automatic-response', body);
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to save automatic response settings.', window.saveAutomaticResponse);
    if (result?.error) return showNotification(result.error, true);
    showNotification(body.enabled ? `Automatic ${body.action} response enabled` : 'Automatic response disabled');
    loadAutomaticResponse();
  };

  window.loadSecurityAnalytics = async function () {
    const result = await api('GET', '/network-intelligence/analytics');
    if (!result || result.error) return;
    document.getElementById('destination-analytics-summary').innerHTML = `<div><strong>${result.applications}</strong><span>Applications</span></div><div><strong>${result.destinations}</strong><span>Destinations</span></div><div><strong>${result.observations}</strong><span>Observations</span></div><div><strong>${result.high_risk}</strong><span>High risk</span></div>`;
    const render = (id, rows, noun) => {
      const root = document.getElementById(id);
      const max = Math.max(1, ...rows.map(x => x.observations));
      root.innerHTML = rows.length ? rows.map(item => `<div class="analytics-bar-row"><div><strong>${esc(item.label)}</strong><small>${item.destinations} ${noun} · max risk ${item.max_risk}</small></div><div class="analytics-track"><i style="width:${Math.max(3, item.observations / max * 100)}%"></i></div><b>${item.observations}</b></div>`).join('') : '<div class="no-data">No connection evidence yet.</div>';
    };
    render('destination-top-apps', result.top_apps || [], 'destinations');
    render('destination-top-targets', result.top_targets || [], 'applications');
  };

  async function loadHistoryStatus() {
    const result = await api('GET', '/network-intelligence/history');
    if (!result || result.error) return;
    const select = document.getElementById('destination-retention');
    if (select) select.value = String(result.retention_days);
    const summary = document.getElementById('destination-history-summary');
    if (summary) summary.textContent = `${result.connections} connection records across ${result.applications} applications · kept for ${result.retention_days} days.`;
  }

  window.saveConnectionRetention = async function () {
    const days = Number(document.getElementById('destination-retention')?.value || 7);
    const result = await api('POST', '/network-intelligence/history', { days });
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to change history retention.', window.saveConnectionRetention);
    if (result?.error) return showNotification(result.error, true);
    showNotification(`Connection history will be kept for ${days} days`);
    loadHistoryStatus();
  };

  window.clearConnectionHistory = async function () {
    if (!confirm('Clear connection history and learning observations? Trusted baselines will remain.')) return;
    const result = await api('DELETE', '/network-intelligence/history');
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to clear connection history.', window.clearConnectionHistory);
    if (result?.error) return showNotification(result.error, true);
    showNotification('Connection history cleared');
    load();
  };

  window.exportConnectionReport = function (format) {
    window.location.href = `/api/network-intelligence/export?format=${encodeURIComponent(format)}`;
  };

  window.loadNewDestinationAlerts = load;
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount);
  else mount();
})();
