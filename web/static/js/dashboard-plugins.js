(function () {
  'use strict';

  let installed = [];
  let activePlugin = null;
  let activePluginView = 'native';

  const esc = value => typeof escapeHtml === 'function'
    ? escapeHtml(String(value ?? ''))
    : String(value ?? '').replace(/[&<>"']/g, char => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[char]));

  async function loadPluginCenter() {
    await Promise.all([loadPlugins(), loadPluginEvents()]);
  }

  async function loadPlugins() {
    const result = await api('GET', '/plugin-registry');
    const list = document.getElementById('plugin-list');
    if (!list) return;
    if (!result || result.error) {
      list.innerHTML = `<div class="plugin-empty">${esc(result?.error || 'Plugin Center is unavailable.')}</div>`;
      return;
    }
    installed = Array.isArray(result.plugins) ? result.plugins : [];
    document.getElementById('plugin-installed-count').textContent = installed.length;
    document.getElementById('plugin-online-summary').textContent = installed.length
      ? `${installed.filter(item => item.enabled).length} enabled · local only`
      : 'Ready for your first plugin';
    if (!installed.length) {
      list.innerHTML = '<div class="plugin-empty"><strong>No plugins installed</strong><br>Load a trusted plugin.json manifest from disk to begin.</div>';
      return;
    }
    list.innerHTML = '';
    installed.forEach(item => list.appendChild(renderPluginCard(item)));
  }

  function renderPluginCard(item) {
    const manifest = item.manifest || {};
    const card = document.createElement('article');
    card.className = 'plugin-card';
    const permissions = (manifest.permissions || []).map(p => `<span>${esc(p)}</span>`).join('');
    card.innerHTML = `
      <div class="plugin-card-top">
        <div class="plugin-identity"><span class="plugin-icon">${manifest.icon === 'printer' ? '▣' : '◇'}</span><div><h3>${esc(manifest.name)}</h3><small>${esc(manifest.publisher || 'Unknown publisher')} · v${esc(manifest.version)}</small></div></div>
        <span class="plugin-state ${item.enabled ? 'enabled' : ''}">${item.enabled ? 'Enabled' : 'Disabled'}</span>
      </div>
      <p>${esc(manifest.description || 'Local API plugin')}</p>
      <div class="plugin-permissions">${permissions || '<span>No permissions</span>'}</div>
      <div class="plugin-runtime-summary"><span>Runtime</span><strong>${esc((manifest.server?.start_mode || 'external').replace('_', ' '))}</strong></div>
      <div class="plugin-runtime-card-controls"><span class="no-data">Checking runtime…</span></div>
      <div class="plugin-card-actions"></div>`;
    const actions = card.querySelector('.plugin-card-actions');
    const toggle = button(item.enabled ? 'Disable' : 'Enable', item.enabled ? 'btn btn-ghost' : 'btn');
    toggle.addEventListener('click', () => setPluginEnabled(manifest.id, !item.enabled));
    actions.appendChild(toggle);
    if (item.enabled) {
      const open = button('Open', 'btn btn-ghost');
      open.addEventListener('click', () => openPlugin(item));
      const test = button('Test connection', 'btn btn-ghost');
      test.addEventListener('click', () => checkPluginHealth(item, true));
      actions.append(open, test);
    }
    const remove = button('Remove', 'btn btn-danger');
    remove.addEventListener('click', () => removePlugin(manifest.id, manifest.name));
    actions.appendChild(remove);
    setTimeout(() => hydrateRuntimeCard(item, card), 0);
    return card;
  }

  async function hydrateRuntimeCard(item, card) {
    if (!card?.isConnected) return;
    const manifest = item.manifest || {};
    const target = card.querySelector('.plugin-runtime-card-controls');
    const status = await api('GET', `/plugins/${encodeURIComponent(manifest.id)}/runtime`);
    if (!target || status?.error) {
      if (target) target.innerHTML = `<span class="no-data">${esc(status?.error || 'Runtime unavailable')}</span>`;
      return;
    }
    target.innerHTML = '';
    const mode = document.createElement('select');
    mode.className = 'plugin-runtime-mode';
    const modes = manifest.server?.runtime ? ['manual', 'on_demand', 'automatic', 'external'] : ['external'];
    modes.forEach(value => {
      const option = document.createElement('option');
      option.value = value;
      option.textContent = ({manual:'Manual', on_demand:'On demand', automatic:'Start with DNS Filter', external:'External service'})[value];
      option.selected = value === status.mode;
      mode.appendChild(option);
    });
    mode.addEventListener('change', () => setRuntimeMode(manifest.id, mode.value));
    target.appendChild(mode);
    const state = document.createElement('span');
    state.className = `plugin-runtime-state ${status.state}`;
    state.textContent = status.managed ? status.state : 'external';
    target.appendChild(state);
    if (item.enabled && status.managed) {
      const main = button(status.state === 'running' ? 'Restart' : 'Start', 'btn btn-ghost');
      main.addEventListener('click', () => runtimeOperation(manifest.id, status.state === 'running' ? 'restart' : 'start'));
      target.appendChild(main);
      if (status.state === 'running') {
        const stop = button('Stop', 'btn btn-danger');
        stop.addEventListener('click', () => runtimeOperation(manifest.id, 'stop'));
        target.appendChild(stop);
      }
    }
    if (status.last_error) {
      const error = document.createElement('small'); error.className = 'plugin-runtime-error'; error.textContent = status.last_error;
      target.appendChild(error);
    }
  }

  async function setRuntimeMode(pluginID, mode) {
    const response = await api('POST', `/plugins/${encodeURIComponent(pluginID)}/runtime-mode`, {mode});
    if (!response?.success) return showNotification(response?.error || 'Could not change runtime mode', true);
    showNotification(`Runtime mode changed to ${mode.replace('_', ' ')}`);
    await loadPlugins();
  }

  function button(label, className) {
    const element = document.createElement('button');
    element.type = 'button';
    element.className = className;
    element.textContent = label;
    return element;
  }

  async function installSelectedManifest() {
    const input = document.getElementById('plugin-manifest-file');
    const file = input?.files?.[0];
    if (!file) return showNotification('Choose a plugin.json file first', true);
    if (file.size > 256 * 1024) return showNotification('Plugin manifest is too large', true);
    let manifest;
    try { manifest = JSON.parse(await file.text()); }
    catch (_) { return showNotification('This file is not valid JSON', true); }
    const permissions = Array.isArray(manifest.permissions) ? manifest.permissions.join('\n• ') : 'None declared';
    const interfaceLabel = ({native:'Native controls', web:'Embedded web app', both:'Web app + native controls'})[manifest.ui?.mode] || 'Native controls';
    const runtime = manifest.server?.runtime;
    const command = runtime?.windows || runtime?.linux || runtime?.darwin;
    const runtimeNotice = command ? `\nRuntime: ${command.executable} ${(command.args || []).join(' ')}\nMode: ${manifest.server?.start_mode || 'external'}\n` : '\nRuntime: externally managed\n';
    const accepted = confirm(`Install ${manifest.name || 'this plugin'}?\n\nPublisher: ${manifest.publisher || 'Unknown'}\nServer: ${manifest.server?.base_url || 'Missing'}\nInterface: ${interfaceLabel}${runtimeNotice}\nPermissions:\n• ${permissions}\n\nThe plugin will remain disabled until you enable it.`);
    if (!accepted) return;
    const result = await api('POST', '/plugin-registry/install', manifest);
    if (result?.code === 'vault_locked' && window.requestVaultUnlock) {
      return requestVaultUnlock('Unlock the vault to install this plugin.', installSelectedManifest);
    }
    if (!result?.success) return showNotification(result?.error || 'Plugin installation failed', true);
    input.value = '';
    showNotification(`${manifest.name || 'Plugin'} installed disabled`);
    await loadPluginCenter();
  }

  async function setPluginEnabled(id, enabled) {
    const result = await api('POST', `/plugin-registry/${encodeURIComponent(id)}/enabled`, { enabled });
    if (result?.code === 'vault_locked' && window.requestVaultUnlock) {
      return requestVaultUnlock(`Unlock the vault to ${enabled ? 'enable' : 'disable'} this plugin.`, () => setPluginEnabled(id, enabled));
    }
    if (!result?.success) return showNotification(result?.error || 'Could not change plugin state', true);
    showNotification(`Plugin ${enabled ? 'enabled' : 'disabled'}`);
    if (!enabled && activePlugin?.manifest?.id === id) closeWorkspace();
    await loadPluginCenter();
  }

  async function removePlugin(id, name) {
    if (!confirm(`Remove ${name || id} from Plugin Center?\n\nThis does not delete or stop the separate application.`)) return;
    const result = await api('DELETE', `/plugin-registry/${encodeURIComponent(id)}`);
    if (result?.code === 'vault_locked' && window.requestVaultUnlock) {
      return requestVaultUnlock('Unlock the vault to remove this plugin.', () => removePlugin(id, name));
    }
    if (!result?.success) return showNotification(result?.error || 'Could not remove plugin', true);
    if (activePlugin?.manifest?.id === id) closeWorkspace();
    showNotification('Plugin removed from DNS Filter');
    await loadPluginCenter();
  }

  async function openPlugin(item) {
    activePlugin = item;
    const manifest = item.manifest;
    const workspace = document.getElementById('plugin-workspace');
    workspace.hidden = false;
    document.body.classList.add('plugin-workspace-open');
    document.getElementById('plugin-workspace-title').textContent = manifest.name;
    document.getElementById('plugin-workspace-description').textContent = manifest.description || '';
    renderPanels(manifest);
    setupPluginViews(manifest);
    await renderRuntimeControls(item);
    const online = await checkPluginHealth(item, false);
    if (online && activePluginView === 'web') loadPluginWebApp(manifest);
    document.getElementById('plugin-close-workspace')?.focus();
  }

  function pluginUI(manifest) {
    const ui = manifest?.ui || {};
    const mode = ['native', 'web', 'both'].includes(ui.mode) ? ui.mode : 'native';
    const fallback = mode === 'web' ? 'web' : 'native';
    const preferred = ui.default_view === 'web' || ui.default_view === 'native' ? ui.default_view : fallback;
    return {mode, preferred, webPath: ui.web_path || '/'};
  }

  function setupPluginViews(manifest) {
    const ui = pluginUI(manifest);
    const tabs = document.getElementById('plugin-view-tabs');
    const webTab = document.getElementById('plugin-tab-web');
    const nativeTab = document.getElementById('plugin-tab-native');
    tabs.hidden = ui.mode !== 'both';
    webTab.hidden = ui.mode === 'native';
    nativeTab.hidden = ui.mode === 'web';
    setPluginView(ui.preferred, false);
  }

  function setPluginView(view, loadWeb = true) {
    if (!activePlugin) return;
    const ui = pluginUI(activePlugin.manifest);
    if (view === 'web' && ui.mode === 'native') view = 'native';
    if (view === 'native' && ui.mode === 'web') view = 'web';
    activePluginView = view;
    document.getElementById('plugin-native-view').hidden = view !== 'native';
    document.getElementById('plugin-web-view').hidden = view !== 'web';
    document.querySelectorAll('[data-plugin-view]').forEach(tab => {
      const selected = tab.dataset.pluginView === view;
      tab.classList.toggle('active', selected);
      tab.setAttribute('aria-selected', String(selected));
    });
    document.querySelector('.plugin-workspace-dialog')?.classList.toggle('web-active', view === 'web');
    if (view === 'web' && loadWeb) loadPluginWebApp(activePlugin.manifest);
  }

  function pluginWebURL(manifest) {
    try {
      const base = new URL(manifest.server.base_url);
      const target = new URL(pluginUI(manifest).webPath, `${base.origin}/`);
      return target.origin === base.origin ? target.href : '';
    } catch (_) { return ''; }
  }

  function loadPluginWebApp(manifest) {
    if (!activePlugin || activePlugin.manifest.id !== manifest.id) return;
    const frame = document.getElementById('plugin-web-frame');
    const loading = document.getElementById('plugin-web-loading');
    const url = pluginWebURL(manifest);
    if (!url) {
      loading.hidden = false;
      loading.textContent = 'This plugin has an invalid local web address.';
      return;
    }
    loading.hidden = false;
    loading.textContent = 'Loading the local plugin application…';
    frame.onload = () => { loading.hidden = true; };
    if (frame.src !== url) frame.src = url;
  }

  function clearPluginWebApp(message) {
    const frame = document.getElementById('plugin-web-frame');
    const loading = document.getElementById('plugin-web-loading');
    frame.removeAttribute('src');
    frame.onload = null;
    loading.hidden = false;
    loading.textContent = message || 'Start the local service to open its web application.';
  }

  async function renderRuntimeControls(item) {
    const manifest = item.manifest || {};
    const target = document.getElementById('plugin-health-card');
    const status = await api('GET', `/plugins/${encodeURIComponent(manifest.id)}/runtime`);
    if (!target || activePlugin?.manifest?.id !== manifest.id || status?.error) return;
    const controls = document.createElement('div');
    controls.className = 'plugin-runtime-controls';
    controls.innerHTML = `<span><b>${esc(status.managed ? status.state : 'External')}</b><small>${esc(status.mode || 'external')}${status.pid ? ` · PID ${status.pid}` : ''}</small></span>`;
    if (status.managed) {
      const start = button(status.state === 'running' ? 'Restart' : 'Start service', 'btn btn-ghost');
      start.addEventListener('click', () => runtimeOperation(manifest.id, status.state === 'running' ? 'restart' : 'start'));
      controls.appendChild(start);
      if (status.state === 'running') {
        const stop = button('Stop service', 'btn btn-danger');
        stop.addEventListener('click', () => runtimeOperation(manifest.id, 'stop'));
        controls.appendChild(stop);
      }
    }
    target.replaceChildren(controls);
  }

  async function runtimeOperation(pluginID, operation) {
    const response = await api('POST', `/plugins/${encodeURIComponent(pluginID)}/runtime/${operation}`, {});
    if (!response?.success) return showNotification(response?.error || `Could not ${operation} plugin service`, true);
    showNotification(`Plugin service ${operation === 'stop' ? 'stopped' : 'started'}`);
    if (activePlugin) {
      if (operation === 'stop') {
        clearPluginWebApp('The plugin service is stopped. Start it to reopen the web application.');
        const target = document.getElementById('plugin-health-card');
        if (target) {
          target.className = 'plugin-health-card offline';
          target.innerHTML = `<div><strong>● Service stopped</strong><small>${esc(activePlugin.manifest?.server?.base_url || '')}</small><small>Start it when you need this plugin.</small></div>`;
          const controls = document.createElement('div'); controls.className = 'plugin-runtime-buttons';
          const start = button('Start service', 'btn');
          start.addEventListener('click', () => runtimeOperation(pluginID, 'start'));
          controls.appendChild(start); target.appendChild(controls);
        }
      } else {
        const online = await checkPluginHealth(activePlugin, false);
        if (online && activePluginView === 'web') loadPluginWebApp(activePlugin.manifest);
      }
    }
    await loadPluginEvents();
    await loadPlugins();
  }

  function closeWorkspace() {
    clearPluginWebApp();
    activePlugin = null;
    const workspace = document.getElementById('plugin-workspace');
    if (workspace) workspace.hidden = true;
    document.body.classList.remove('plugin-workspace-open');
  }

  async function checkPluginHealth(item, notify) {
    const manifest = item.manifest || {};
    const target = document.getElementById('plugin-health-card');
    if (target && activePlugin?.manifest?.id === manifest.id) {
      target.className = 'plugin-health-card';
      target.innerHTML = '<strong>Checking local service…</strong>';
    }
    const result = await api('GET', `/plugins/${encodeURIComponent(manifest.id)}/health`);
    if (target && activePlugin?.manifest?.id === manifest.id) {
      const online = !!result?.online;
      target.className = `plugin-health-card ${online ? 'online' : 'offline'}`;
      const runtime = await api('GET', `/plugins/${encodeURIComponent(manifest.id)}/runtime`);
      target.innerHTML = `<div><strong>${online ? '● Service connected' : '● Service unavailable'}</strong><small>${esc(manifest.server?.base_url || '')}</small><small>${esc(runtime?.managed ? `${runtime.state} · ${runtime.mode}${runtime.pid ? ` · PID ${runtime.pid}` : ''}` : 'External service')}</small></div><pre>${esc(JSON.stringify(result?.data || {error: result?.error || 'No response'}, null, 2))}</pre>`;
      if (runtime?.managed) {
        const controls = document.createElement('div'); controls.className = 'plugin-runtime-buttons';
        const primary = button(runtime.state === 'running' ? 'Restart' : 'Start service', 'btn btn-ghost');
        primary.addEventListener('click', () => runtimeOperation(manifest.id, runtime.state === 'running' ? 'restart' : 'start'));
        controls.appendChild(primary);
        if (runtime.state === 'running') { const stop = button('Stop', 'btn btn-danger'); stop.addEventListener('click', () => runtimeOperation(manifest.id, 'stop')); controls.appendChild(stop); }
        target.appendChild(controls);
      }
    }
    if (notify) showNotification(result?.online ? `${manifest.name} is online` : (result?.error || `${manifest.name} is offline`), !result?.online);
    if (!result?.online && activePluginView === 'web') clearPluginWebApp('The local web application is unavailable. Start the plugin service and try again.');
    return !!result?.online;
  }

  function renderPanels(manifest) {
    const container = document.getElementById('plugin-panels');
    container.innerHTML = '';
    const actionMap = new Map((manifest.actions || []).map(action => [action.id, action]));
    (manifest.panels || []).forEach(panel => {
      const section = document.createElement('section');
      section.className = 'plugin-panel';
      section.innerHTML = `<h3>${esc(panel.title)}</h3><p>${esc(panel.description || '')}</p>`;
      (panel.actions || []).forEach(id => {
        const action = actionMap.get(id);
        if (action) section.appendChild(renderAction(manifest.id, action));
      });
      container.appendChild(section);
    });
  }

  function renderAction(pluginID, action) {
    const wrapper = document.createElement('div');
    wrapper.className = 'plugin-action';
    wrapper.innerHTML = `<div class="plugin-action-head"><div><strong>${esc(action.label)}</strong><small>${esc(action.description || action.permission || '')}</small></div></div>`;
    const fields = document.createElement('div');
    fields.className = 'plugin-fields';
    const controls = new Map();
    (action.fields || []).forEach(field => {
      const label = document.createElement('label');
      label.className = `plugin-field ${field.type === 'checkbox' ? 'plugin-checkbox' : ''}`;
      const caption = document.createElement('span'); caption.textContent = field.label;
      let control;
      if (field.type === 'textarea') control = document.createElement('textarea');
      else if (field.type === 'select') {
        control = document.createElement('select');
        (field.options || []).forEach(option => {
          const element = document.createElement('option');
          element.textContent = option.label; element.value = JSON.stringify(option.value);
          if (JSON.stringify(option.value) === JSON.stringify(field.default)) element.selected = true;
          control.appendChild(element);
        });
      } else {
        control = document.createElement('input');
        control.type = field.type === 'checkbox' ? 'checkbox' : field.type;
      }
      control.dataset.field = field.id;
      control.dataset.kind = field.type;
      control.dataset.transform = field.transform || '';
      if (field.placeholder) control.placeholder = field.placeholder;
      if (field.accept && field.type === 'file') control.accept = field.accept;
      if (field.required) control.required = true;
      if (field.default !== undefined && field.type !== 'select') {
        if (field.type === 'checkbox') control.checked = !!field.default;
        else control.value = field.default;
      }
      label.append(field.type === 'checkbox' ? control : caption, field.type === 'checkbox' ? caption : control);
      fields.appendChild(label); controls.set(field.id, control);
    });
    wrapper.appendChild(fields);
    const run = button(action.label, 'btn');
    const result = document.createElement('pre'); result.className = 'plugin-action-result'; result.hidden = true;
    run.addEventListener('click', async () => {
      if (action.confirm && !confirm(action.confirm)) return;
      const input = {};
      let uploadFile = null;
      for (const [id, control] of controls) {
        if (control.dataset.kind === 'file') {
          uploadFile = control.files?.[0] || null;
          if (control.required && !uploadFile) return showNotification('Choose a file first', true);
          continue;
        }
        if (control.required && !String(control.value || '').trim() && control.type !== 'checkbox') return showNotification(`${control.closest('label').innerText.trim()} is required`, true);
        if (control.dataset.transform === 'cards_lines') {
          input[id] = String(control.value || '').split(/\r?\n/).map(line => line.trim()).filter(Boolean).map(line => {
            const parts = line.split('|'); return {question:(parts.shift() || '').trim(), answer:parts.join('|').trim()};
          });
        } else if (control.dataset.kind === 'checkbox') input[id] = control.checked;
        else if (control.dataset.kind === 'number') input[id] = Number(control.value);
        else if (control.dataset.kind === 'select') { try { input[id] = JSON.parse(control.value); } catch (_) { input[id] = control.value; } }
        else input[id] = control.value;
      }
      run.disabled = true; run.textContent = 'Working…'; result.hidden = false; result.textContent = 'Contacting plugin…';
      let response;
      let binaryResponse = null;
      try {
        if (action.body_type === 'binary') {
          const form = new FormData(); form.append('file', uploadFile); form.append('input', JSON.stringify(input));
          const raw = await fetch(`/api/plugins/${encodeURIComponent(pluginID)}/actions/${encodeURIComponent(action.id)}/file`, {method:'POST', credentials:'same-origin', body:form});
          response = await raw.json();
        } else if (action.response_type === 'image' || action.response_type === 'download') {
          const raw = await fetch(`/api/plugins/${encodeURIComponent(pluginID)}/actions/${encodeURIComponent(action.id)}/content`, {method:'POST', credentials:'same-origin', headers:{'Content-Type':'application/json'}, body:JSON.stringify({input})});
          if (!raw.ok) { response = await raw.json(); }
          else { binaryResponse = await raw.blob(); response = {success:true}; }
        } else {
          response = await api('POST', `/plugins/${encodeURIComponent(pluginID)}/actions/${encodeURIComponent(action.id)}`, {input});
        }
      } catch (error) { response = {error:error.message || 'Plugin request failed'}; }
      run.disabled = false; run.textContent = action.label;
      wrapper.querySelector('.plugin-content-result')?.remove();
      if (binaryResponse && action.response_type === 'image') {
        result.hidden = true;
        const image = document.createElement('img'); image.className = 'plugin-content-result'; image.alt = `${action.label} result`; image.src = URL.createObjectURL(binaryResponse);
        image.addEventListener('load', () => URL.revokeObjectURL(image.src), {once:true}); wrapper.appendChild(image);
      } else if (binaryResponse && action.response_type === 'download') {
        result.hidden = true;
        const extension = binaryResponse.type.includes('pdf') ? 'pdf' : 'png';
        const link = document.createElement('a'); link.className = 'btn plugin-content-result'; link.download = `tinyprinter-${action.id}.${extension}`; link.href = URL.createObjectURL(binaryResponse); link.textContent = 'Save download'; wrapper.appendChild(link); link.click();
      } else {
        result.hidden = false; result.textContent = JSON.stringify(response?.data || {error: response?.error || 'No response'}, null, 2);
      }
      showNotification(response?.success ? `${action.label} completed` : (response?.error || `${action.label} failed`), !response?.success);
      await loadPluginEvents();
      if (activePlugin) checkPluginHealth(activePlugin, false);
    });
    wrapper.append(run, result);
    return wrapper;
  }

  async function loadPluginEvents() {
    const result = await api('GET', '/plugins/events/recent?limit=60');
    const target = document.getElementById('plugin-events');
    if (!target) return;
    const events = result?.events || [];
    if (!events.length) { target.innerHTML = '<span class="no-data">No plugin activity yet.</span>'; return; }
    target.innerHTML = events.map(event => `<article class="plugin-event ${event.success ? 'ok' : 'fail'}"><strong>${esc(event.action)} · ${esc(event.plugin_id)}</strong><span>${esc(event.message || (event.success ? 'Completed' : 'Failed'))}${event.status ? ` · HTTP ${event.status}` : ''}</span><small>${esc(new Date(event.created_at).toLocaleString())}</small></article>`).join('');
  }

  document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('plugin-install-button')?.addEventListener('click', installSelectedManifest);
    document.getElementById('plugin-refresh-button')?.addEventListener('click', loadPluginCenter);
    document.getElementById('plugin-close-workspace')?.addEventListener('click', closeWorkspace);
    document.getElementById('plugin-view-tabs')?.addEventListener('click', event => {
      const tab = event.target.closest('[data-plugin-view]');
      if (tab) setPluginView(tab.dataset.pluginView);
    });
    document.getElementById('plugin-workspace')?.addEventListener('click', event => {
      if (event.target === event.currentTarget) closeWorkspace();
    });
    document.addEventListener('keydown', event => {
      if (event.key === 'Escape' && !document.getElementById('plugin-workspace')?.hidden) closeWorkspace();
    });
  });

  window.loadPluginCenter = loadPluginCenter;
})();
