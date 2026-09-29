(function () {
    let selectedPack = null;
    const html = value => String(value ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
    const date = value => { const d = new Date(value); return Number.isNaN(d.getTime()) ? 'Unknown' : d.toLocaleString(); };

    window.loadSecurityPacks = async function () {
        const r = await api('GET', '/security-packs');
        if (!r || r.error) return;
        document.getElementById('pack-trust-count').textContent = `${r.trusted_key_count || 0} trusted key${r.trusted_key_count === 1 ? '' : 's'}`;
        const root = document.getElementById('security-pack-list');
        root.innerHTML = r.packs?.length ? r.packs.map(pack => `
            <article>
                <div><span>${html(pack.category || 'threat intelligence')}</span><strong>${html(pack.name)}</strong><small>${Number(pack.domain_count || 0).toLocaleString()} domains · v${Number(pack.version || 0)} · expires ${date(pack.expires_at)}</small></div>
                <button class="btn btn-ghost" data-id="${html(pack.id)}" onclick="removeSecurityPack(this.dataset.id)">Remove</button>
            </article>`).join('') : '<div class="decision-empty compact"><strong>No signed packs installed</strong><span>Add a trusted publisher key, then verify and install a pack.</span></div>';
    };

    window.securityPackSelected = function (input) {
        selectedPack = input.files?.[0] || null;
        document.getElementById('security-pack-filename').textContent = selectedPack?.name || 'No file selected';
        document.getElementById('security-pack-verify').disabled = !selectedPack;
        document.getElementById('security-pack-install').disabled = !selectedPack;
        document.getElementById('security-pack-result').textContent = selectedPack ? 'Ready to verify. Nothing has been installed yet.' : 'Select a JSON pack to inspect its signature before installation.';
    };

    async function uploadPack(action) {
        if (!selectedPack) return null;
        const form = new FormData(); form.append('pack', selectedPack);
        const response = await fetch(`/api/security-packs/${action}`, {method:'POST', body:form});
        let result; try { result = await response.json(); } catch { result = {error:'Invalid server response'}; }
        if (!response.ok && !result.error) result.error = `Request failed (${response.status})`;
        return result;
    }

    function renderResult(r, installed) {
        const out = document.getElementById('security-pack-result');
        if (!r || r.error || r.valid === false) {
            out.className = 'operation-note warning';
            out.textContent = `Rejected: ${r?.error || 'verification failed'}`;
            return false;
        }
        const pack = r.pack || {};
        out.className = 'operation-note success';
        out.textContent = `${installed ? 'Installed' : 'Verified'}: ${pack.name} v${pack.version} · ${Number(pack.domain_count || 0).toLocaleString()} domains · signed by ${pack.key_id}.`;
        return true;
    }

    window.verifySelectedSecurityPack = async function () { renderResult(await uploadPack('verify'), false); };
    window.installSelectedSecurityPack = async function () {
        const r = await uploadPack('install');
        if (renderResult(r, true)) { showNotification('Signed security pack installed'); loadSecurityPacks(); }
    };

    window.removeSecurityPack = async function (id) {
        if (!confirm(`Remove security pack ${id}? Its domains will stop being blocked by this pack.`)) return;
        const r = await api('DELETE', '/security-packs/' + encodeURIComponent(id));
        if (!r || r.error) { showNotification(r?.error || 'Could not remove pack', true); return; }
        showNotification('Security pack removed'); loadSecurityPacks();
    };

    window.trustSecurityPackKey = async function () {
        const id = document.getElementById('security-key-id').value.trim();
        const public_key = document.getElementById('security-public-key').value.trim();
        if (!id || !public_key) { showNotification('Key ID and public key are required', true); return; }
        if (!confirm(`Trust publisher key “${id}”? Only continue if you verified it through a separate trusted channel.`)) return;
        const r = await api('POST', '/security-packs/trusted-keys', {id, public_key});
        if (!r || r.error) { showNotification(r?.error || 'Could not trust key', true); return; }
        document.getElementById('security-key-id').value = '';
        document.getElementById('security-public-key').value = '';
        showNotification('Publisher key trusted'); loadSecurityPacks();
    };

    const mount = () => {
        if (location.hash === '#blocklist' || document.querySelector('.view[data-view="blocklist"]')?.classList.contains('active')) loadSecurityPacks();
    };
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount); else mount();
})();
