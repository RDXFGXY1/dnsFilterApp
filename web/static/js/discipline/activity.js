// ── Task actions ─────────────────────────────────────────────────────────────
async function toggleDone(id, done) {
    const r = await api('PATCH', '/tasks/' + id, { status: done ? 'done' : 'todo' });
    if (r && r.success) {
        if (r.xp) notify(`+${r.xp} XP — task done!`);
        refreshCurrent();
    }
}
async function cycleStatus(id, status) {
    const next = NEXT_STATUS[status] || 'in_progress';
    const r = await api('PATCH', '/tasks/' + id, { status: next });
    if (r && r.success) { if (r.xp) notify(`+${r.xp} XP — task done!`); refreshCurrent(); }
}
async function deleteTask(id) {
    const r = await api('PATCH', '/tasks/' + id, { archived: true });
    if (r && r.success) { notify('Task archived — use Undo in the project workspace to restore it'); refreshCurrent(); }
}
function refreshCurrent() {
    const active = document.querySelector('.view.active').dataset.view;
    if (active === 'command') loadCommand();
    else if (active === 'home') loadHome();
    else if (active === 'workspace') refreshWs();
    else if (active === 'activity') loadActivity();
}

// ── Activity + heatmap ───────────────────────────────────────────────────────
async function loadActivity() {
    const [stats, hm] = await Promise.all([api('GET', '/stats'), api('GET', '/heatmap')]);
    if (stats) {
        document.getElementById('a-streak').textContent = stats.current_streak || 0;
        document.getElementById('a-longest').textContent = stats.longest_streak || 0;
        document.getElementById('a-active').textContent = stats.active_days || 0;
        document.getElementById('a-total').textContent = (stats.total || 0).toLocaleString();
    }
    renderHeatmap('activity-heatmap', hm, 'a-contrib', showDay);
    const recent = await api('GET', '/recent?limit=80');
    const filter = document.getElementById('activity-filter')?.value || 'all';
    const shown = (recent || []).filter(e => filter === 'all' || e.type === filter);
    renderRecent('activity-recent', shown.slice(0,40));
    const week = (hm || []).slice(-7), prior = (hm || []).slice(-14,-7);
    const weekTotal = week.reduce((n,d)=>n+d.count,0), priorTotal = prior.reduce((n,d)=>n+d.count,0);
    const delta = priorTotal ? Math.round((weekTotal-priorTotal)*100/priorTotal) : (weekTotal ? 100 : 0);
    const topType = Object.entries((stats && stats.by_type) || {}).sort((a,b)=>b[1]-a[1])[0];
    document.getElementById('activity-insight').innerHTML = `<span>Momentum narrative</span><strong>${weekTotal} contributions this week · ${delta>=0?'+':''}${delta}% vs previous week${topType ? ` · most common: ${esc(EVENT_LABEL[topType[0]] || topType[0])}` : ''}</strong>`;
}

function heatLevel(c) { return c === 0 ? 0 : c < 2 ? 1 : c < 4 ? 2 : c < 7 ? 3 : 4; }

function renderHeatmap(elId, days, contribId, onClick) {
    const el = document.getElementById(elId);
    if (!el) return;
    if (!days || !days.length) { el.innerHTML = '<span class="no-data">No activity yet</span>'; return; }
    let total = 0;
    // Pad so the first column starts on Sunday.
    const first = new Date(days[0].day + 'T00:00:00');
    const pad = first.getDay();
    let cells = '';
    for (let i = 0; i < pad; i++) cells += '<span class="hcell empty"></span>';
    days.forEach(d => {
        total += d.count;
        cells += `<span class="hcell l${heatLevel(d.count)}" data-day="${d.day}" title="${d.day}: ${d.count}"></span>`;
    });
    el.innerHTML = `<div class="heatgrid">${cells}</div>`;
    if (contribId) document.getElementById(contribId).textContent = total.toLocaleString() + ' contributions';
    if (onClick) el.querySelectorAll('.hcell[data-day]').forEach(c =>
        c.addEventListener('click', () => onClick(c.dataset.day)));
}

const EVENT_LABEL = {
    task_created: 'Created a task', task_completed: 'Completed a task',
    subtask_completed: 'Completed a subtask', task_updated: 'Updated a task',
    status_changed: 'Changed status', priority_changed: 'Changed priority',
    bug_fixed: 'Fixed a bug', project_created: 'Created a project',
};
async function showDay(date) {
    const events = await api('GET', '/day?date=' + date);
    const box = document.getElementById('day-detail');
    if (!events || !events.length) { box.innerHTML = `<p class="no-data">${date}: no activity</p>`; return; }
    box.innerHTML = `<h3 class="subhead">${date} · ${events.length} events</h3>` +
        events.map(e => `<div class="day-event">${esc(EVENT_LABEL[e.type] || e.type)} <small>${esc(e.created_at)}</small></div>`).join('');
}
