(function () {
  const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));

  function mount() {
    VIEW_META.profiles = ['Network Profiles', 'Save and restore complete protection modes'];
    const nav = document.querySelector('.sidenav');
    const schedules = nav?.querySelector('[data-view="schedules"]');
    const link = document.createElement('a');
    link.href = '#'; link.dataset.view = 'profiles';
    link.innerHTML = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 6h16M7 12h10M10 18h4"/><circle cx="7" cy="6" r="2"/><circle cx="15" cy="12" r="2"/><circle cx="12" cy="18" r="2"/></svg>Network Profiles';
    link.onclick = event => { event.preventDefault(); switchView('profiles'); loadNetworkProfiles(); };
    schedules?.insertAdjacentElement('beforebegin', link);

    const view = document.createElement('section');
    view.className = 'view'; view.dataset.view = 'profiles';
    view.innerHTML = `
      <section class="dashboard-command"><div><span class="command-kicker">Save · Preview · Switch</span><h2>Network Profiles</h2><p>Capture the current filtering, app blocks, categories, and wildcard exceptions as one reusable protection mode.</p></div></section>
      <section class="panel"><div class="panel-head"><div><h2>Capture current protection</h2><p>Create profiles such as Gaming, Work, Privacy, or Family.</p></div></div><div class="profile-create"><input id="profile-name" placeholder="Profile name, e.g. Work"><input id="profile-description" placeholder="Short description (optional)"><button class="btn" onclick="createNetworkProfile()">Save current state</button></div></section>
      <section class="panel"><div class="panel-head"><div><h2>Saved profiles</h2><p>Preview every change before applying a profile.</p></div><button class="btn btn-ghost" onclick="loadNetworkProfiles()">Refresh</button></div><div id="network-profile-list" class="profile-list"><div class="no-data">Loading…</div></div></section>`;
    document.querySelector('.main').appendChild(view);
  }

  async function load() {
    const root = document.getElementById('network-profile-list'); if (!root) return;
    const items = await api('GET', '/network-profiles');
    if (items?.error) { root.innerHTML = `<div class="no-data">${esc(items.error)}</div>`; return; }
    root.innerHTML = items.length ? items.map(profile => {
      const apps = Object.values(profile.state.apps || {}).filter(Boolean).length;
      const categories = Object.values(profile.state.categories || {}).filter(Boolean).length;
      return `<article class="profile-card"><div class="profile-card-copy"><span class="profile-state ${profile.state.filtering?'on':'off'}">${profile.state.filtering?'FILTER ON':'FILTER OFF'}</span><h3>${esc(profile.name)}</h3><p>${esc(profile.description || 'Saved protection state')}</p><small>${apps} blocked apps · ${categories} categories · ${(profile.state.whitelist || []).length} exceptions</small></div><div class="profile-actions"><button class="btn btn-ghost" onclick="previewNetworkProfile(${profile.id})">Preview</button><button class="btn-remove" onclick="deleteNetworkProfile(${profile.id})">Delete</button></div><div class="profile-preview" id="profile-preview-${profile.id}" hidden></div></article>`;
    }).join('') : '<div class="no-data">No network profiles yet. Capture the current protection state above.</div>';
  }

  async function create() {
    const name = document.getElementById('profile-name').value.trim();
    const description = document.getElementById('profile-description').value.trim();
    if (!name) return showNotification('Enter a profile name', true);
    const result = await api('POST', '/network-profiles', {name, description});
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to save a network profile.', create);
    if (result?.error) return showNotification(result.error, true);
    document.getElementById('profile-name').value = ''; document.getElementById('profile-description').value = '';
    showNotification('Network profile saved'); load();
  }

  async function preview(id) {
    const box = document.getElementById(`profile-preview-${id}`); if (!box) return;
    if (!box.hidden) { box.hidden = true; return; }
    box.hidden = false; box.innerHTML = '<span class="no-data">Comparing with current protection…</span>';
    const result = await api('GET', `/network-profiles/${id}/preview`);
    if (result?.error) { box.innerHTML = `<span class="no-data">${esc(result.error)}</span>`; return; }
    const c = result.changes, p = result.profile;
    box.innerHTML = `<div class="profile-change-grid"><div><strong>${c.filtering_changes?'1':'0'}</strong><span>filter switch</span></div><div><strong>${c.app_changes}</strong><span>app changes</span></div><div><strong>${c.category_changes}</strong><span>category changes</span></div><div><strong>+${c.whitelist_added} / -${c.whitelist_removed}</strong><span>exceptions</span></div></div><p>Applying this profile restores its complete saved state, including removing exceptions that are not part of it.</p><button class="btn" onclick="applyNetworkProfile(${id},'${encodeURIComponent(p.name)}')">Apply ${esc(p.name)}</button>`;
  }

  async function applyProfile(id, encodedName) {
    const name = decodeURIComponent(encodedName);
    if (!confirm(`Apply ${name}? This will replace the current saved protection state.`)) return;
    const result = await api('POST', `/network-profiles/${id}/apply`, {});
    if (result?.code === 'vault_locked') return requestVaultUnlock(`Unlock the vault to apply ${name}.`, () => applyProfile(id, encodedName));
    if (result?.error) return showNotification(result.error, true);
    showNotification(`${name} profile applied`); load();
    if (window.loadWhitelist) loadWhitelist(); if (window.loadApps) loadApps(); if (window.loadCategories) loadCategories();
  }

  async function remove(id) {
    if (!confirm('Delete this network profile?')) return;
    const result = await api('DELETE', `/network-profiles/${id}`);
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to delete this profile.', () => remove(id));
    if (result?.error) return showNotification(result.error, true);
    showNotification('Network profile deleted'); load();
  }

  window.loadNetworkProfiles = load; window.createNetworkProfile = create; window.previewNetworkProfile = preview; window.applyNetworkProfile = applyProfile; window.deleteNetworkProfile = remove;
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount); else mount();
})();
