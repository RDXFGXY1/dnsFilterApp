(function () {
  const esc = value => String(value ?? '').replace(/[&<>"']/g, c => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]));

  function mount() {
    const root = document.getElementById('wildcard-allowlist-feature');
    if (!root) return;
    root.innerHTML = `
      <section class="wildcard-tool">
        <div class="wildcard-tool-copy">
          <span class="command-kicker">Wildcard exception</span>
          <h3>Allow a service and all its subdomains</h3>
          <p>Enter a root domain such as <code>spotify.com</code>. The rule <code>*.spotify.com</code> safely covers the root and every nested subdomain.</p>
        </div>
        <div class="input-row">
          <input id="wildcard-root" placeholder="spotify.com" autocomplete="off">
          <button class="btn btn-ghost" onclick="discoverServiceSubdomains()">Discover observed</button>
          <button class="btn" onclick="allowServiceWildcard()">Allow all</button>
        </div>
        <div id="wildcard-discovery" class="wildcard-results"><span class="no-data">Discovery uses only this computer's local blocked DNS history.</span></div>
      </section>`;
  }

  function rootValue() {
    return document.getElementById('wildcard-root')?.value.trim().toLowerCase() || '';
  }

  async function discover() {
    const root = rootValue();
    if (!root) return showNotification('Enter a root domain such as spotify.com', true);
    const box = document.getElementById('wildcard-discovery');
    box.innerHTML = '<span class="no-data">Searching local DNS history…</span>';
    const result = await api('GET', `/whitelist/discover?domain=${encodeURIComponent(root)}`);
    if (result?.error) {
      box.innerHTML = `<span class="no-data">${esc(result.error)}</span>`;
      return;
    }
    const domains = result.domains || [];
    box.innerHTML = `
      <div class="wildcard-summary"><strong>${esc(result.wildcard)}</strong><span>${domains.length} observed domain${domains.length === 1 ? '' : 's'}</span></div>
      ${domains.length ? `<div class="wildcard-domain-list">${domains.map(item => `<span>${esc(item.domain)} <b>${item.count}</b></span>`).join('')}</div>` : '<span class="no-data">No matching blocked domains have been observed yet. The wildcard rule can still be added.</span>'}`;
  }

  async function allowAll() {
    const raw = rootValue();
    if (!raw) return showNotification('Enter a root domain such as spotify.com', true);
    const root = raw.replace(/^https?:\/\//, '').split('/')[0].replace(/^\*\./, '').replace(/^\.|\.$/g, '');
    const wildcard = `*.${root}`;
    const result = await api('POST', '/whitelist', { domain: wildcard });
    if (result?.code === 'vault_locked') return requestVaultUnlock('Unlock the vault to add this wildcard exception.', allowAll);
    if (result?.error || !result?.success) return showNotification(result?.error || 'Could not add wildcard exception', true);
    document.getElementById('whitelist-input').value = '';
    showNotification(`Allowed ${root} and all subdomains`);
    loadWhitelist();
    discover();
  }

  async function toggleRule(button, encodedRule) {
    const row = button.closest('.wildcard-list-item');
    const details = row?.querySelector('.wildcard-rule-details');
    if (!details) return;
    const opening = details.hidden;
    details.hidden = !opening;
    button.setAttribute('aria-expanded', String(opening));
    button.querySelector('small').textContent = opening ? 'Hide observed subdomains' : 'Show observed subdomains';
    if (!opening || details.dataset.loaded === '1') return;

    const rule = decodeURIComponent(encodedRule);
    const root = rule.replace(/^\*\./, '');
    details.innerHTML = '<span class="no-data">Loading observed subdomains…</span>';
    const result = await api('GET', `/whitelist/discover?domain=${encodeURIComponent(root)}`);
    if (result?.error) {
      details.innerHTML = `<span class="no-data">${esc(result.error)}</span>`;
      return;
    }
    const domains = result.domains || [];
    details.dataset.loaded = '1';
    details.innerHTML = domains.length
      ? `<div class="wildcard-rule-summary"><span>${domains.length} observed</span><span>${domains.reduce((sum, item) => sum + Number(item.count || 0), 0)} blocked requests</span></div><div class="wildcard-domain-list">${domains.map(item => `<span>${esc(item.domain)} <b>${item.count}</b></span>`).join('')}</div>`
      : '<span class="no-data">No matching subdomains are present in local blocked DNS history yet.</span>';
  }

  window.discoverServiceSubdomains = discover;
  window.allowServiceWildcard = allowAll;
  window.toggleWildcardRule = toggleRule;
  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount);
  else mount();
})();
