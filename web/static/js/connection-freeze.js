(function () {
    let events = [], busy = false, loading = false;
    const el = id => document.getElementById(id);
    const escape = value => String(value || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
    window.renderFreezeEvents = function () {
        const search = el('freeze-search').value.toLowerCase();
        const rows = events.filter(e => [e.app,e.remote,e.action,e.reason].join(' ').toLowerCase().includes(search));
        el('freeze-event-count').textContent = `${rows.length} events`;
        el('freeze-events').innerHTML = rows.map(e => `<tr><td>${escape(new Date(e.time).toLocaleString())}</td><td>${escape(e.action)}</td><td style="max-width:320px;overflow-wrap:anywhere">${escape(e.app || 'Session')}</td><td>${escape(e.remote || '—')}</td><td>${escape(e.protocol || '—')}</td><td>${escape(e.reason)}</td></tr>`).join('') || `<tr><td colspan="6" class="freeze-empty"><strong>${search ? 'No matching events' : 'No connection events yet'}</strong>${search ? 'Try another application or IP address.' : 'Start a freeze to see blocked attempts and session changes here.'}</td></tr>`;
    };
    window.loadConnectionFreeze = async function () {
        if (loading) return;
        loading = true;
        try {
            const r = await api('GET', '/connection-freeze');
            if (!r || r.error) throw new Error(r?.error || 'Cannot read Connection Freeze status');
            const s = r.status;
            el('freeze-status').textContent = s.active ? (s.manual ? 'Active — until you stop it manually' : `Active — ends at ${new Date(s.until).toLocaleTimeString()}`) : (s.supported ? 'Off — new connections are allowed' : 'Available on Windows only');
            document.querySelector('.freeze-page').dataset.active = String(s.active);
            el('freeze-badge').textContent = s.active ? (s.manual ? 'Manual freeze' : 'Timed freeze') : 'Not active';
            el('freeze-error').textContent = s.error || '';
            el('freeze-error').hidden = !s.error;
            el('freeze-duration').disabled = busy || s.active;
            el('freeze-start').disabled = busy || s.active || !s.supported;
            el('freeze-stop').disabled = busy || !s.active;
            events = r.events || [];
            renderFreezeEvents();
        } catch (e) {
            el('freeze-status').textContent = 'Status unavailable';
            el('freeze-error').textContent = e.message;
            el('freeze-error').hidden = false;
            el('freeze-badge').textContent = 'Unavailable';
            el('freeze-start').disabled = true;
            // Keep recovery available if a status poll failed.
            el('freeze-stop').disabled = busy;
        } finally { loading = false; }
    };
    async function change(action) {
        if (busy) return;
        busy = true;
        el('freeze-start').disabled = el('freeze-stop').disabled = true;
        let error = '';
        try {
            const r = await api('POST', `/connection-freeze/${action}`, action === 'start' ? {minutes:Number(el('freeze-duration').value)} : {});
            if (!r?.success) throw new Error(r?.error || 'Connection Freeze could not be changed');
            showNotification(action === 'start' ? 'New connections are now frozen' : 'Connection Freeze stopped');
        } catch (e) { error=e.message; showNotification(error,true); }
        finally { busy=false; await loadConnectionFreeze(); if (error) {el('freeze-error').textContent=error;el('freeze-error').hidden=false;} }
    }
    window.startConnectionFreeze = () => change('start');
    window.stopConnectionFreeze = () => change('stop');
    setInterval(() => {if (document.querySelector('.view[data-view="freeze"]')?.classList.contains('active')) loadConnectionFreeze();}, 3000);
    if (location.hash === '#freeze') {
        const show = () => {switchView('freeze');loadConnectionFreeze();};
        if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', show);
        else show();
    }
})();
