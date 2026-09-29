// ── Home ─────────────────────────────────────────────────────────────────────
async function loadHome() {
    const [stats, projects, tasks] = await Promise.all([
        api('GET', '/stats'), api('GET', '/projects'), api('GET', '/tasks?all=1'),
    ]);
    if (stats) {
        document.getElementById('h-done').textContent = (stats.tasks_completed || 0).toLocaleString();
        document.getElementById('h-streak').textContent = stats.current_streak || 0;
        document.getElementById('h-active').textContent = stats.active_days || 0;
    }
    document.getElementById('h-projects').textContent = projects ? projects.length : 0;

    const list = document.getElementById('today-tasks');
    const active = (tasks || []).filter(t => !t.done);
    const overdue = active.filter(t => daysFromToday(t.due_date) < 0).length;
    const stalled = (projects || []).filter(p => p.total > 0 && p.progress === 0).length;
    const healthScore = Math.max(0, 100 - overdue * 10 - stalled * 8 - Math.max(0, active.filter(t => t.status === 'in_progress').length - 5) * 5);
    document.getElementById('home-health').innerHTML = `<div class="health-score"><strong>${healthScore}</strong><span>System health</span></div><div class="health-copy"><b>${healthScore >= 80 ? 'Your system is healthy' : healthScore >= 55 ? 'Your system needs attention' : 'Your system is overloaded'}</b><p>${overdue} overdue · ${stalled} stalled project${stalled === 1 ? '' : 's'} · ${active.length} open loops</p></div><div class="health-bar"><span style="width:${healthScore}%"></span></div>`;
    list.innerHTML = active.length ? active.slice(0, 12).map(t => taskRow(t, false)).join('')
        : '<li class="no-data">No open tasks. Add one!</li>';

    const pl = document.getElementById('home-projects');
    pl.innerHTML = (projects && projects.length) ? projects.map(p => `
        <li onclick="openProject(${p.id})" style="cursor:pointer;">
            <span>${esc(p.name)}</span>
            <span style="display:flex;align-items:center;gap:10px;flex-shrink:0;">
                <span class="mini-bar"><span style="width:${p.progress}%"></span></span>
                <span class="count">${p.progress}%</span>
            </span>
        </li>`).join('') : '<li class="no-data">No projects yet</li>';

    const hm = await api('GET', '/heatmap');
    renderHeatmap('home-heatmap', hm, 'h-contrib', null);

    // Small general analytics on Home
    const all = tasks || [];
    if (document.getElementById('home-status')) {
        renderHBars('home-status', taskStatus(all));
        renderHBars('home-priority', taskPriority(all));
        renderCompletion('home-completion', all);
    }
}
async function quickAddTask() {
    const inp = document.getElementById('quick-task');
    const title = inp.value.trim();
    if (!title) return;
    const r = await api('POST', '/tasks', { title, priority: 'medium', project_id: 0 });
    if (r && r.success) { inp.value = ''; notify('Task added'); loadHome(); } else notify('Failed', true);
}

// ── Today (daily mission tracker) ────────────────────────────────────────────
function fmtDate(d) { return d.getFullYear() + '-' + String(d.getMonth() + 1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0'); }
function todayStr() { return fmtDate(new Date()); }
function greeting() { const h = new Date().getHours(); return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening'; }
function pOrder(t) { return t.priority === 'high' ? 0 : t.priority === 'medium' ? 1 : 2; }
function progressRing(pct) {
    const r = 34, c = 2 * Math.PI * r, off = c * (1 - pct / 100);
    return `<svg viewBox="0 0 80 80" width="88" height="88">
        <circle cx="40" cy="40" r="${r}" fill="none" stroke="var(--surface-2)" stroke-width="8"/>
        <circle cx="40" cy="40" r="${r}" fill="none" stroke="var(--coral)" stroke-width="8" stroke-linecap="butt"
            stroke-dasharray="${c.toFixed(1)}" stroke-dashoffset="${off.toFixed(1)}" transform="rotate(-90 40 40)"/>
        <text x="40" y="46" text-anchor="middle" font-size="18" font-weight="700" fill="var(--ink)">${pct}%</text></svg>`;
}

async function loadToday() {
    const [tasksRaw, stats, goals, hm] = await Promise.all([
        api('GET', '/tasks?all=1'), api('GET', '/stats'), api('GET', '/goals'), api('GET', '/heatmap?days=1'),
    ]);
    const tasks = tasksRaw || [];
    const today = todayStr();
    document.getElementById('today-date').textContent = greeting();
    document.getElementById('today-sub').textContent = new Date().toLocaleDateString(undefined, { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' });

    // Today's missions = top-level tasks due today, or overdue & not done
    const todays = tasks.filter(t => !t.parent_id && (t.due_date === today || (t.due_date && t.due_date < today && !t.done)));
    todays.sort((a, b) => (a.done - b.done) || pOrder(a) - pOrder(b));
    const doneC = todays.filter(t => t.done).length;
    const pct = todays.length ? Math.round(doneC * 100 / todays.length) : 0;
    document.getElementById('today-ring').innerHTML = progressRing(pct);
    document.getElementById('today-count').textContent = doneC + '/' + todays.length + ' done';
    const overdueCount = todays.filter(t => !t.done && t.due_date < today).length;
    const highCount = todays.filter(t => !t.done && t.priority === 'high').length;
    renderViewKpis('today-kpis', [
        ['Committed', todays.length, 'missions in today’s plan'],
        ['Completed', doneC, pct + '% execution rate', doneC ? 'good' : ''],
        ['Overdue', overdueCount, overdueCount ? 'needs a decision' : 'clear', overdueCount ? 'danger' : 'good'],
        ['High priority', highCount, highCount > 3 ? 'reduce commitments' : 'focused load', highCount > 3 ? 'warn' : ''],
    ]);

    const list = document.getElementById('today-list');
    list.innerHTML = todays.length ? todays.map(t => {
        const overdue = t.due_date && t.due_date < today && !t.done;
        return `<li class="task-row ${t.done ? 'is-done' : ''}">
            <span class="task-main">
                <button class="task-check ${t.done ? 'on' : ''}" onclick="completeToday(${t.id}, ${!t.done})">${t.done ? '✓' : ''}</button>
                <span>${PRIORITY[t.priority] || '🟡'} ${esc(t.title)} ${overdue ? '<b class="overdue">overdue</b>' : ''}</span>
            </span>
            <button class="btn-remove" onclick="delToday(${t.id})">DEL</button></li>`;
    }).join('') : '<li class="no-data">Nothing due today. Add a mission, or enjoy the calm. ☕</li>';

    const todayContrib = (hm && hm.length) ? hm[hm.length - 1].count : 0;
    document.getElementById('today-momentum').innerHTML = `
        <div class="mo"><span class="mo-num">🔥 ${stats.current_streak || 0}</span><span class="mo-l">Day streak</span></div>
        <div class="mo"><span class="mo-num">${todayContrib}</span><span class="mo-l">Today's activity</span></div>
        <div class="mo"><span class="mo-num">${doneC}</span><span class="mo-l">Missions done</span></div>`;

    // Upcoming (next 7 days)
    const in7 = new Date(); in7.setDate(in7.getDate() + 7); const in7s = fmtDate(in7);
    const upcoming = tasks.filter(t => !t.done && t.due_date && t.due_date > today && t.due_date <= in7s).sort((a, b) => a.due_date.localeCompare(b.due_date));
    document.getElementById('today-upcoming').innerHTML = upcoming.length
        ? upcoming.slice(0, 8).map(t => `<li><span>${PRIORITY[t.priority] || '🟡'} ${esc(t.title)}</span><span class="count">${esc(t.due_date)}</span></li>`).join('')
        : '<li class="no-data">Nothing in the next 7 days</li>';

    // Active goals
    const active = (goals || []).filter(g => g.status !== 'done');
    document.getElementById('today-goals').innerHTML = active.length
        ? active.map(g => `<li onclick="switchView('goals')" style="cursor:pointer;"><span>${esc(g.name)}</span>
            <span style="display:flex;align-items:center;gap:10px;flex-shrink:0;"><span class="mini-bar"><span style="width:${g.progress}%"></span></span><span class="count">${g.progress}%</span></span></li>`).join('')
        : '<li class="no-data">No active goals</li>';
}
async function completeToday(id, done) {
    const r = await api('PATCH', '/tasks/' + id, { status: done ? 'done' : 'todo' });
    if (r && r.success) { if (r.xp) notify(`+${r.xp} XP — mission complete!`); loadToday(); }
}
async function delToday(id) { const r = await api('DELETE', '/tasks/' + id); if (r && r.success) loadToday(); }
async function addTodayTask() {
    const inp = document.getElementById('today-add');
    const title = inp.value.trim();
    if (!title) return;
    const r = await api('POST', '/tasks', { title, priority: 'medium', project_id: 0, due_date: todayStr() });
    if (r && r.success) { inp.value = ''; notify('Mission added for today'); loadToday(); }
}
