(function () {
    const KEY = 'dnsfilter.notification-center.v1';
    const LIMIT = 200;

    function read() {
        try {
            const items = JSON.parse(localStorage.getItem(KEY) || '[]');
            return Array.isArray(items) ? items : [];
        } catch (_) { return []; }
    }

    function write(items) {
        try { localStorage.setItem(KEY, JSON.stringify(items.slice(0, LIMIT))); } catch (_) {}
        render();
    }

    function save(message, isError, options) {
        const text = String(message || '').trim();
        if (!text) return;
        const opts = options || {};
        const items = read();
        items.unshift({
            id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
            message: text,
            level: opts.level || (isError ? 'error' : 'success'),
            source: opts.source || (location.pathname.startsWith('/discipline') ? 'Discipline' : 'DNS Filter'),
            created_at: new Date().toISOString(),
            read: false
        });
        write(items);
    }

    function relativeDate(value) {
        const date = new Date(value), seconds = Math.max(0, (Date.now() - date.getTime()) / 1000);
        if (!Number.isFinite(seconds)) return '';
        if (seconds < 60) return 'just now';
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
        return date.toLocaleString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
    }

    function escapeHTML(value) {
        return String(value).replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
    }

    function render() {
        const list = document.getElementById('notification-center-list');
        const badge = document.getElementById('notification-center-count');
        if (!list || !badge) return;
        const items = read(), unread = items.filter(item => !item.read).length;
        badge.textContent = unread > 99 ? '99+' : String(unread);
        badge.hidden = unread === 0;
        list.innerHTML = items.length ? items.map(item => `
            <article class="notification-center-item level-${escapeHTML(item.level)} ${item.read ? '' : 'unread'}">
                <span class="notification-center-dot"></span>
                <div><strong>${escapeHTML(item.source)}</strong><p>${escapeHTML(item.message)}</p><time>${escapeHTML(relativeDate(item.created_at))}</time></div>
                <button type="button" data-notification-remove="${escapeHTML(item.id)}" aria-label="Remove notification">×</button>
            </article>`).join('') : '<div class="notification-center-empty"><strong>All clear</strong><p>New alerts and completed actions will be saved here.</p></div>';
    }

    function markAllRead() {
        write(read().map(item => ({ ...item, read: true })));
    }

    function toggle(force) {
        const panel = document.getElementById('notification-center-panel');
        if (!panel) return;
        const open = typeof force === 'boolean' ? force : panel.hidden;
        panel.hidden = !open;
        if (open) markAllRead();
    }

    function mount() {
        if (document.getElementById('notification-center-button')) return;
        document.body.insertAdjacentHTML('beforeend', `
            <button class="notification-center-button" id="notification-center-button" type="button" aria-label="Open notification center" aria-controls="notification-center-panel">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9"/><path d="M10 21h4"/></svg>
                <span id="notification-center-count" hidden>0</span>
            </button>
            <aside class="notification-center-panel" id="notification-center-panel" hidden>
                <header><div><span>Activity archive</span><h2>Notifications</h2></div><button type="button" id="notification-center-close" aria-label="Close">×</button></header>
                <div class="notification-center-tools"><button type="button" id="notification-center-read">Mark all read</button><button type="button" id="notification-center-clear">Clear history</button></div>
                <div class="notification-center-list" id="notification-center-list"></div>
            </aside>`);
        const actions = document.querySelector('.page-actions');
        if (actions) actions.prepend(document.getElementById('notification-center-button'));
        document.getElementById('notification-center-button').onclick = () => toggle();
        document.getElementById('notification-center-close').onclick = () => toggle(false);
        document.getElementById('notification-center-read').onclick = markAllRead;
        document.getElementById('notification-center-clear').onclick = () => { if (confirm('Clear all saved notifications?')) write([]); };
        document.getElementById('notification-center-list').onclick = event => {
            const id = event.target.closest('[data-notification-remove]')?.dataset.notificationRemove;
            if (id) write(read().filter(item => item.id !== id));
        };
        document.addEventListener('keydown', event => { if (event.key === 'Escape') toggle(false); });
        render();
    }

    window.saveAppNotification = save;
    window.openNotificationCenter = () => toggle(true);
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount); else mount();
})();
