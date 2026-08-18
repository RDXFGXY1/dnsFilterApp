// ── Discipline page: tasks, projects, event-driven activity ──────────────────

const VIEWS = {
    today:     ['Today',     'Your daily missions'],
    home:      ['Home',      'Your tasks, projects, and momentum'],
    goals:     ['Goals',     'What you are ultimately working toward'],
    habits:    ['Habits',    'Things you do repeatedly — build the streak'],
    calendar:  ['Calendar',  'Your tasks and deadlines over time'],
    reminders: ['Reminders', "Don't forget the important stuff"],
    analytics: ['Analytics', 'Everything, across all projects'],
    projects:  ['Projects',  'Everything you are building'],
    workspace: ['Project',   ''],
    activity:  ['Activity',  'Your contribution history'],
};
let currentProject = 0;
let wsView = 'table';

function esc(s) {
    return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function notify(msg, isErr) {
    const el = document.getElementById('notification');
    el.textContent = (isErr ? '⚠  ' : '✓  ') + msg;
    el.classList.toggle('error', !!isErr);
    el.classList.add('show');
    clearTimeout(el._t);
    el._t = setTimeout(() => el.classList.remove('show'), 2500);
}

async function api(method, path, body) {
    const opts = { method, headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin' };
    if (body) opts.body = JSON.stringify(body);
    const r = await fetch('/api/discipline' + path, opts);
    if (r.status === 401) { window.location.href = '/login'; return null; }
    return r.json();
}

function switchView(name) {
    document.querySelectorAll('.view').forEach(v => v.classList.toggle('active', v.dataset.view === name));
    document.querySelectorAll('.sidenav a[data-view]').forEach(a => a.classList.toggle('active', a.dataset.view === name));
    const meta = VIEWS[name] || VIEWS.home;
    document.getElementById('page-title').textContent = meta[0];
    document.getElementById('page-sub').textContent = meta[1];
    window.scrollTo(0, 0);
    if (name === 'today') loadToday();
    if (name === 'home') loadHome();
    if (name === 'goals') loadGoals();
    if (name === 'habits') loadHabits();
    if (name === 'calendar') loadCalendar();
    if (name === 'reminders') loadReminders();
    if (name === 'analytics') loadGlobalAnalytics();
    if (name === 'projects') loadProjects();
    if (name === 'activity') loadActivity();
}
document.querySelectorAll('.sidenav a[data-view]').forEach(a =>
    a.addEventListener('click', e => { e.preventDefault(); switchView(a.dataset.view); }));

const PRIORITY = { high: '🔴', medium: '🟡', low: '🟢' };
const NEXT_STATUS = { todo: 'in_progress', in_progress: 'done', done: 'todo' };
const STATUS_LABEL = { todo: 'TODO', in_progress: 'DOING', done: 'DONE' };

function taskRow(t, onProject) {
    const due = t.due_date ? `<small> · ${esc(t.due_date)}</small>` : '';
    const proj = onProject ? '' : (t.project_id ? '' : '');
    return `<li class="task-row ${t.done ? 'is-done' : ''}">
        <span class="task-main">
            <button class="task-check ${t.done ? 'on' : ''}" title="Toggle done" onclick="toggleDone(${t.id}, ${!t.done})">${t.done ? '✓' : ''}</button>
            <span>${PRIORITY[t.priority] || '🟡'} ${esc(t.title)}${due}</span>
        </span>
        <span style="display:flex;gap:6px;flex-shrink:0;align-items:center;">
            <button class="btn-toggle ${t.status==='done'?'on':'off'}" title="Cycle status" onclick="cycleStatus(${t.id}, '${t.status}')">${STATUS_LABEL[t.status]||'TODO'}</button>
            <button class="btn-remove" onclick="deleteTask(${t.id})">DEL</button>
        </span>
    </li>`;
}

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

// ── Habits ───────────────────────────────────────────────────────────────────
const DOW = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];
async function loadHabits() {
    const habits = await api('GET', '/habits');
    const wrap = document.getElementById('habits-list');
    if (!habits || !habits.length) { wrap.innerHTML = '<span class="no-data">No habits yet. Add one above — things you want to do repeatedly.</span>'; return; }
    const today = todayStr();
    wrap.innerHTML = habits.map(h => {
        const set = {}; (h.days || []).forEach(d => { if (d.count > 0) set[d.day] = true; });
        // Mon–Sun of current week
        const now = new Date(); const wd = (now.getDay() + 6) % 7;
        const monday = new Date(now); monday.setDate(now.getDate() - wd);
        let strip = '';
        for (let i = 0; i < 7; i++) {
            const d = new Date(monday); d.setDate(monday.getDate() + i);
            const ds = fmtDate(d); const on = set[ds]; const future = ds > today;
            strip += `<span class="wday ${on ? 'on' : ''} ${ds === today ? 'is-today' : ''} ${future ? 'future' : ''}" ${future ? '' : `onclick="checkinDay(${h.id},'${ds}')"`}>${DOW[i]}</span>`;
        }
        return `<div class="habit-card">
            <div class="habit-head">
                <span class="habit-name">${esc(h.name)}</span>
                <span class="habit-freq">${esc(h.frequency)}</span>
                <span class="flex-spacer"></span>
                <span class="habit-streak">🔥 ${h.current_streak}</span>
            </div>
            <div class="week-strip">${strip}</div>
            ${miniHeatmap(h.days)}
            <div class="habit-foot">
                <button class="btn ${h.done_today ? 'btn-ghost' : ''}" onclick="checkinDay(${h.id},'${today}')">${h.done_today ? '✓ Done today' : 'Check today'}</button>
                <span class="habit-meta">Best ${h.longest_streak}d · ${h.week_done}/7 this week</span>
                <span class="flex-spacer"></span>
                <button class="btn-remove" onclick="deleteHabit(${h.id})">DEL</button>
            </div>
        </div>`;
    }).join('');
}
async function createHabit() {
    const name = document.getElementById('habit-name').value.trim();
    if (!name) { notify('Habit name required', true); return; }
    const r = await api('POST', '/habits', { name, frequency: document.getElementById('habit-freq').value });
    if (r && r.success) { document.getElementById('habit-name').value = ''; notify('Habit added'); loadHabits(); }
}
async function checkinDay(id, day) {
    const r = await api('POST', '/habits/' + id + '/checkin', { day });
    if (r && r.success) { if (r.done) notify('✓ Checked in'); loadHabits(); if (document.querySelector('.view[data-view="today"]').classList.contains('active')) loadToday(); }
}
async function deleteHabit(id) {
    if (!confirm('Delete this habit and its history?')) return;
    const r = await api('DELETE', '/habits/' + id);
    if (r && r.success) { notify('Habit deleted'); loadHabits(); }
}

// ── Calendar ─────────────────────────────────────────────────────────────────
let calState = { y: new Date().getFullYear(), m: new Date().getMonth() };
const MONTHS = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December'];
function calMonth(delta) { calState.m += delta; if (calState.m < 0) { calState.m = 11; calState.y--; } if (calState.m > 11) { calState.m = 0; calState.y++; } loadCalendar(); }
function calToday() { const n = new Date(); calState = { y: n.getFullYear(), m: n.getMonth() }; loadCalendar(); }
let _calTasks = [];
async function loadCalendar() {
    _calTasks = (await api('GET', '/tasks?all=1')) || [];
    document.getElementById('cal-title').textContent = MONTHS[calState.m] + ' ' + calState.y;
    document.getElementById('cal-dow').innerHTML = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'].map(d => `<span>${d}</span>`).join('');

    // tasks by due day
    const byDay = {};
    _calTasks.forEach(t => { if (t.due_date) (byDay[t.due_date] = byDay[t.due_date] || []).push(t); });

    const first = new Date(calState.y, calState.m, 1);
    const startPad = (first.getDay() + 6) % 7; // Mon=0
    const daysInMonth = new Date(calState.y, calState.m + 1, 0).getDate();
    const today = todayStr();
    let cells = '';
    for (let i = 0; i < startPad; i++) cells += '<div class="cal-cell empty"></div>';
    for (let d = 1; d <= daysInMonth; d++) {
        const ds = `${calState.y}-${String(calState.m + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
        const items = byDay[ds] || [];
        const dots = items.slice(0, 4).map(t => `<span class="cal-dot p-${esc(t.priority || 'medium')} ${t.done ? 'done' : ''}"></span>`).join('');
        cells += `<div class="cal-cell ${ds === today ? 'is-today' : ''}" onclick="calShowDay('${ds}')">
            <span class="cal-num">${d}</span>
            <span class="cal-dots">${dots}${items.length > 4 ? `<span class="cal-more">+${items.length - 4}</span>` : ''}</span>
        </div>`;
    }
    document.getElementById('cal-grid').innerHTML = cells;
    document.getElementById('cal-day').innerHTML = '<span class="no-data">Click a day to see its tasks.</span>';
}
function calShowDay(ds) {
    const items = _calTasks.filter(t => t.due_date === ds);
    const box = document.getElementById('cal-day');
    box.innerHTML = `<h3 class="an-title">${ds}</h3>` + (items.length
        ? '<ul class="data-list">' + items.map(t => `<li><span>${t.done ? '✓' : PRIORITY[t.priority] || '🟡'} ${esc(t.title)}</span>${t.project_id ? '' : '<span class="count">personal</span>'}</li>`).join('') + '</ul>'
        : '<p class="no-data">No tasks due.</p>');
}

// ── Reminders ────────────────────────────────────────────────────────────────
async function loadReminders() {
    const rem = await api('GET', '/reminders');
    const ul = document.getElementById('rem-list');
    if (!rem || !rem.length) { ul.innerHTML = '<li class="no-data">No reminders. Add one above.</li>'; return; }
    const now = new Date();
    ul.innerHTML = rem.map(r => {
        const due = !r.done && new Date(r.remind_at.replace(' ', 'T')) <= now;
        return `<li class="rem-row ${r.done ? 'is-done' : ''} ${due ? 'due' : ''}">
            <span class="task-main">
                <button class="task-check ${r.done ? 'on' : ''}" onclick="toggleReminder(${r.id}, ${!r.done})">${r.done ? '✓' : ''}</button>
                <span>🔔 ${esc(r.text)}</span>
            </span>
            <span style="display:flex;align-items:center;gap:10px;flex-shrink:0;">
                <span class="count">${esc(r.remind_at)}</span>
                <button class="btn-remove" onclick="deleteReminder(${r.id})">DEL</button>
            </span></li>`;
    }).join('');
}
async function createReminder() {
    const text = document.getElementById('rem-text').value.trim();
    const date = document.getElementById('rem-date').value;
    const time = document.getElementById('rem-time').value || '09:00';
    if (!text || !date) { notify('Text and date required', true); return; }
    const r = await api('POST', '/reminders', { text, remind_at: date + ' ' + time });
    if (r && r.success) {
        document.getElementById('rem-text').value = '';
        if ('Notification' in window && Notification.permission === 'default') Notification.requestPermission();
        notify('Reminder set'); loadReminders();
    }
}
async function toggleReminder(id, done) { const r = await api('PATCH', '/reminders/' + id, { done }); if (r && r.success) loadReminders(); }
async function deleteReminder(id) { const r = await api('DELETE', '/reminders/' + id); if (r && r.success) loadReminders(); }

// Fire reminders when their time arrives (while the page is open).
const _remNotified = new Set();
async function checkReminders() {
    const rem = await api('GET', '/reminders');
    if (!rem) return;
    const now = new Date();
    rem.forEach(r => {
        if (r.done || _remNotified.has(r.id)) return;
        if (new Date(r.remind_at.replace(' ', 'T')) <= now) {
            _remNotified.add(r.id);
            notify('🔔 ' + r.text);
            if ('Notification' in window && Notification.permission === 'granted') new Notification('Reminder', { body: r.text });
        }
    });
}

// ── Goals ────────────────────────────────────────────────────────────────────
let _allProjects = [];
async function loadGoals() {
    const [goals, projects] = await Promise.all([api('GET', '/goals'), api('GET', '/projects')]);
    _allProjects = projects || [];
    const grid = document.getElementById('goals-grid');
    if (!goals || !goals.length) { grid.innerHTML = '<span class="no-data">No goals yet. Create one above — big objectives your projects roll up into.</span>'; return; }
    const projName = id => (_allProjects.find(p => p.id === id) || {}).name || ('#' + id);
    grid.innerHTML = goals.map(g => {
        const isCount = (!g.project_ids || !g.project_ids.length) && g.target > 0;
        const linked = (g.project_ids || []).map(id => `<span class="glink">${esc(projName(id))}<button class="glink-x" onclick="unlinkGoal(${g.id},${id})">✕</button></span>`).join('');
        const catBadge = g.category ? `<span class="gcat">${esc(g.category)}</span>` : '';
        const done = g.status === 'done';
        const avail = _allProjects.filter(p => !(g.project_ids || []).includes(p.id));
        return `<div class="gcard ${done ? 'is-done' : ''}">
            <div class="gcard-head">
                <span class="gcard-name">${esc(g.name)}</span>${catBadge}
                <span class="flex-spacer"></span>
                <span class="gcard-pct">${g.progress}%</span>
            </div>
            ${g.description ? `<div class="gcard-desc">${esc(g.description)}</div>` : ''}
            <span class="progress-bar"><span class="progress-fill" style="width:${g.progress}%"></span></span>
            <div class="gcard-meta">
                ${g.target_date ? `<span>🎯 ${esc(g.target_date)}</span>` : ''}
                ${isCount ? `<span class="gcount"><button onclick="stepGoal(${g.id},${g.current},-1,${g.target})">−</button>${g.current}/${g.target}<button onclick="stepGoal(${g.id},${g.current},1,${g.target})">+</button></span>` : ''}
            </div>
            <div class="gcard-links">${linked || '<span class="lc-empty">No projects linked</span>'}</div>
            <div class="gcard-foot">
                <select class="glink-add" onchange="if(this.value){linkGoal(${g.id},this.value)}">
                    <option value="">+ Link project…</option>
                    ${avail.map(p => `<option value="${p.id}">${esc(p.name)}</option>`).join('')}
                </select>
                <span class="flex-spacer"></span>
                <button class="btn-toggle ${done ? 'on' : 'off'}" onclick="toggleGoalDone(${g.id}, ${!done})">${done ? 'DONE' : 'MARK DONE'}</button>
                <button class="btn-remove" onclick="deleteGoal(${g.id})">DEL</button>
            </div>
        </div>`;
    }).join('');
}
async function createGoal() {
    const name = document.getElementById('goal-name').value.trim();
    if (!name) { notify('Goal name required', true); return; }
    const target = parseInt(document.getElementById('goal-target').value, 10) || 0;
    const r = await api('POST', '/goals', {
        name, category: document.getElementById('goal-cat').value.trim(),
        target_date: document.getElementById('goal-date').value, target,
    });
    if (r && r.success) {
        ['goal-name', 'goal-cat', 'goal-date', 'goal-target'].forEach(id => document.getElementById(id).value = '');
        notify('Goal created'); loadGoals();
    } else notify('Failed', true);
}
async function linkGoal(goalId, projectId) {
    const r = await api('POST', '/goals/' + goalId + '/link', { project_id: parseInt(projectId, 10), link: true });
    if (r && r.success) { notify('Project linked'); loadGoals(); }
}
async function unlinkGoal(goalId, projectId) {
    const r = await api('POST', '/goals/' + goalId + '/link', { project_id: projectId, link: false });
    if (r && r.success) loadGoals();
}
async function stepGoal(goalId, current, delta, target) {
    const next = Math.max(0, Math.min(target, current + delta));
    const r = await api('PATCH', '/goals/' + goalId, { current: next });
    if (r && r.success) loadGoals();
}
async function toggleGoalDone(goalId, done) {
    const r = await api('PATCH', '/goals/' + goalId, { status: done ? 'done' : 'active' });
    if (r && r.success) { notify(done ? 'Goal completed! 🎯' : 'Goal reopened'); loadGoals(); }
}
async function deleteGoal(goalId) {
    if (!confirm('Delete this goal?')) return;
    const r = await api('DELETE', '/goals/' + goalId);
    if (r && r.success) { notify('Goal deleted'); loadGoals(); }
}

// ── Projects ─────────────────────────────────────────────────────────────────
function miniHeatmap(days) {
    if (!days || !days.length) return '';
    let cells = '';
    const first = new Date(days[0].day + 'T00:00:00');
    for (let i = 0; i < first.getDay(); i++) cells += '<span class="mhc empty"></span>';
    days.forEach(d => { cells += `<span class="mhc l${heatLevel(d.count)}" title="${d.day}: ${d.count}"></span>`; });
    return `<div class="minigrid">${cells}</div>`;
}

async function loadProjects() {
    const projects = await api('GET', '/projects');
    const grid = document.getElementById('projects-grid');
    if (!projects || !projects.length) { grid.innerHTML = '<span class="no-data">No projects yet. Create one above.</span>'; return; }

    // Summary header
    const sum = document.getElementById('proj-summary');
    if (sum) {
        const totTasks = projects.reduce((a, p) => a + p.total, 0);
        const totDone = projects.reduce((a, p) => a + p.done, 0);
        const overall = totTasks ? Math.round(totDone * 100 / totTasks) : 0;
        sum.innerHTML = `<span><b>${projects.length}</b> projects</span><span><b>${totTasks}</b> tasks</span><span><b>${totDone}</b> done</span><span><b>${overall}%</b> overall</span>`;
    }

    // Fetch per-project stats + a compact heatmap for each card.
    const data = await Promise.all(projects.map(async p => {
        const [stats, hm] = await Promise.all([
            api('GET', '/stats?project=' + p.id),
            api('GET', '/heatmap?project=' + p.id + '&days=98'),
        ]);
        return { p, stats: stats || {}, hm: hm || [] };
    }));

    grid.innerHTML = data.map(({ p, stats, hm }) => {
        const todo = Math.max(0, p.total - p.done - (p.in_progress || 0));
        return `<button class="pcard" onclick="openProject(${p.id})">
            <div class="pcard-top">
                <span class="pcard-name">${esc(p.name)}</span>
                <span class="pcard-pct">${p.progress}%</span>
            </div>
            <div class="pcard-desc">${esc(p.description || 'No description')}</div>
            <span class="progress-bar"><span class="progress-fill" style="width:${p.progress}%"></span></span>
            <div class="pcard-chips">
                <span class="pcard-chip c-done">${p.done} done</span>
                <span class="pcard-chip c-prog">${p.in_progress || 0} doing</span>
                <span class="pcard-chip c-todo">${todo} todo</span>
            </div>
            ${miniHeatmap(hm)}
            <div class="pcard-foot">
                <span>🔥 ${stats.current_streak || 0}d streak</span>
                <span>${(stats.total || 0)} events</span>
            </div>
        </button>`;
    }).join('');
}
async function createProject() {
    const name = document.getElementById('proj-name').value.trim();
    const description = document.getElementById('proj-desc').value.trim();
    if (!name) { notify('Name required', true); return; }
    const r = await api('POST', '/projects', { name, description });
    if (r && r.success) {
        document.getElementById('proj-name').value = '';
        document.getElementById('proj-desc').value = '';
        notify('Project created'); loadProjects();
    } else notify('Failed', true);
}

// ── Project workspace ────────────────────────────────────────────────────────
async function openProject(id) {
    currentProject = id;
    switchView('workspace');
    const projects = await api('GET', '/projects');
    const p = (projects || []).find(x => x.id === id) || { name: 'Project', progress: 0 };
    document.getElementById('ws-name').textContent = p.name;
    document.getElementById('ws-progress').style.width = (p.progress || 0) + '%';
    loadWsTasks();
}
function setWsView(v) {
    wsView = v;
    document.querySelectorAll('.ws-tab[data-ws]').forEach(b => b.classList.toggle('active', b.dataset.ws === v));
    document.getElementById('ws-toolbar').style.display = v === 'table' ? 'flex' : 'none';
    document.getElementById('ws-table').style.display = v === 'table' ? 'block' : 'none';
    document.getElementById('ws-list').style.display = v === 'list' ? 'block' : 'none';
    document.getElementById('ws-board').style.display = v === 'board' ? 'grid' : 'none';
    document.getElementById('ws-insights').style.display = v === 'insights' ? 'block' : 'none';
    if (v === 'insights') loadProjectActivity(); else loadWsTasks();
}

// ── Per-project analytics (stats · task breakdowns · charts · feed) ──────────
function hbar(label, value, max, color) {
    const pct = max ? Math.round(value * 100 / max) : 0;
    return `<div class="hbar"><span class="hbar-l">${esc(label)}</span>
        <span class="hbar-track"><span class="hbar-fill" style="width:${pct}%;background:${color}"></span></span>
        <span class="hbar-v">${value}</span></div>`;
}
function renderHBars(elId, rows) {
    const max = Math.max(1, ...rows.map(r => r[1]));
    const el = document.getElementById(elId);
    el.innerHTML = rows.length ? rows.map(r => hbar(r[0], r[1], max, r[2])).join('') : '<div class="no-data">No data</div>';
}
function renderBarchart(elId, series) {
    const el = document.getElementById(elId);
    if (!series || !series.length) { el.innerHTML = '<div class="no-data">No data</div>'; return; }
    const max = Math.max(1, ...series.map(d => d.count));
    el.innerHTML = series.map(d => {
        const h = Math.round(d.count / max * 100);
        const day = d.day.slice(8); // DD
        return `<span class="bc-col" title="${d.day}: ${d.count}"><span class="bc-bar" style="height:${Math.max(h, d.count ? 6 : 2)}%"></span><span class="bc-x">${day}</span></span>`;
    }).join('');
}

// Shared task-analytics helpers (used by project, home, and global analytics).
function taskPriority(tasks) {
    const p = { high: 0, medium: 0, low: 0 };
    tasks.forEach(t => { p[t.priority] = (p[t.priority] || 0) + 1; });
    return [['High', p.high, '#d1553b'], ['Medium', p.medium, '#d97706'], ['Low', p.low, '#2f8a55']];
}
function taskStatus(tasks) {
    const s = { todo: 0, in_progress: 0, done: 0 };
    tasks.forEach(t => { s[t.status] = (s[t.status] || 0) + 1; });
    return [['Todo', s.todo, '#9a9384'], ['In progress', s.in_progress, '#d97706'], ['Done', s.done, '#2f8a55']];
}
function taskLabels(tasks) {
    const c = {};
    tasks.forEach(t => (t.labels || '').split(',').map(x => x.trim()).filter(Boolean).forEach(l => c[l] = (c[l] || 0) + 1));
    return Object.entries(c).sort((a, b) => b[1] - a[1]).slice(0, 8).map(([l, v]) => [l, v, '#2f6fb0']);
}
function renderCompletion(elId, tasks) {
    const total = tasks.length, done = tasks.filter(t => t.done).length;
    const rate = total ? (done * 100 / total).toFixed(1) : '0.0';
    document.getElementById(elId).innerHTML = `
        <div class="cstat"><span>Created</span><b>${total}</b></div>
        <div class="cstat"><span>Completed</span><b>${done}</b></div>
        <div class="cstat"><span>Remaining</span><b>${total - done}</b></div>
        <div class="cstat cstat-big"><span>Completion</span><b>${rate}%</b></div>
        <span class="progress-bar"><span class="progress-fill" style="width:${rate}%"></span></span>`;
}

// ── Global analytics (all projects) ──────────────────────────────────────────
async function loadGlobalAnalytics() {
    const [stats, tasksRaw, perday, projects] = await Promise.all([
        api('GET', '/stats'),
        api('GET', '/tasks?all=1'),
        api('GET', '/series?type=task_completed&days=30'),
        api('GET', '/projects'),
    ]);
    const s = stats || {};
    const tasks = tasksRaw || [];
    const tiles = [
        ['Total events', s.total || 0], ['Active days', s.active_days || 0],
        ['Current streak', s.current_streak || 0], ['Longest streak', s.longest_streak || 0],
        ['Tasks completed', s.tasks_completed || 0], ['Bugs fixed', s.bugs_fixed || 0],
    ];
    document.getElementById('ga-tiles').innerHTML = tiles.map(([l, v]) =>
        `<div class="stat-card"><div class="stat-top"><span class="stat-label">${l}</span></div><div class="stat-value">${v}</div></div>`).join('');
    renderHBars('ga-priority', taskPriority(tasks));
    renderHBars('ga-status', taskStatus(tasks));
    renderHBars('ga-label', taskLabels(tasks));
    renderCompletion('ga-completion', tasks);
    renderBarchart('ga-perday', perday || []);
    const pl = document.getElementById('ga-projects');
    pl.innerHTML = (projects && projects.length) ? projects.map(p => `
        <li onclick="openProject(${p.id})" style="cursor:pointer;">
            <span>${esc(p.name)}</span>
            <span style="display:flex;align-items:center;gap:10px;flex-shrink:0;">
                <span class="mini-bar"><span style="width:${p.progress}%"></span></span>
                <span class="count">${p.done}/${p.total}</span>
            </span></li>`).join('') : '<li class="no-data">No projects</li>';
}

async function loadProjectActivity() {
    const [stats, hm, tasksRaw, perday, recent] = await Promise.all([
        api('GET', '/stats?project=' + currentProject),
        api('GET', '/heatmap?project=' + currentProject),
        api('GET', '/tasks?project=' + currentProject),
        api('GET', '/series?project=' + currentProject + '&type=task_completed&days=21'),
        api('GET', '/recent?project=' + currentProject + '&limit=15'),
    ]);
    const s = stats || {};
    const tasks = tasksRaw || [];

    // Project statistics tiles
    const tiles = [
        ['Total', s.total || 0], ['Active days', s.active_days || 0],
        ['Current streak', s.current_streak || 0], ['Longest streak', s.longest_streak || 0],
        ['Tasks completed', s.tasks_completed || 0], ['Subtasks done', s.subtasks_completed || 0],
        ['Bugs fixed', s.bugs_fixed || 0],
    ];
    document.getElementById('proj-stats').innerHTML = tiles.map(([l, v]) =>
        `<div class="pstat"><span class="pstat-num">${v}</span><span class="pstat-label">${l}</span></div>`).join('');

    // Task analytics (shared helpers)
    renderHBars('an-priority', taskPriority(tasks));
    renderHBars('an-status', taskStatus(tasks));
    renderHBars('an-label', taskLabels(tasks));
    renderCompletion('an-completion', tasks);

    // Progress chart + heatmap + recent feed
    renderBarchart('an-perday', perday || []);
    renderHeatmap('proj-heatmap', hm, 'proj-contrib', showProjDay);
    document.getElementById('proj-day-detail').innerHTML = '';
    renderRecent('an-recent', recent || []);
}

function renderRecent(elId, events) {
    const ul = document.getElementById(elId);
    if (!events || !events.length) { ul.innerHTML = '<li class="no-data">No activity yet</li>'; return; }
    ul.innerHTML = events.map(e => `<li>
        <span><span class="ev-dot ev-${esc(e.type)}"></span>${esc(EVENT_LABEL[e.type] || e.type)}${e.title ? ': ' + esc(e.title) : ''}</span>
        <small>${esc((e.created_at || '').replace('T', ' ').replace('Z', ''))}</small></li>`).join('');
}
async function showProjDay(date) {
    const events = await api('GET', '/day?date=' + date + '&project=' + currentProject);
    const box = document.getElementById('proj-day-detail');
    if (!events || !events.length) { box.innerHTML = `<p class="no-data">${date}: no activity</p>`; return; }
    box.innerHTML = `<h3 class="subhead">${date} · ${events.length} events</h3>` +
        events.map(e => `<div class="day-event"><span class="ev-dot ev-${esc(e.type)}"></span>${esc(EVENT_LABEL[e.type] || e.type)} <small>${esc(e.created_at)}</small></div>`).join('');
}
async function loadWsTasks() {
    const tasks = (await api('GET', '/tasks?project=' + currentProject)) || [];
    if (wsView === 'table') renderCards(tasks);
    else if (wsView === 'board') renderBoard(tasks);
    else {
        const ul = document.getElementById('ws-tasks');
        ul.innerHTML = tasks.length ? tasks.map(t => taskRow(t, true)).join('') : '<li class="no-data">No tasks yet</li>';
    }
}

// ── GitHub Projects-style tree table (parent tasks + subtasks) ───────────────
let groupBy = 'none';
let _focusSubAdd = 0;              // parent id whose add-subtask input to refocus after render
const collapsed = new Set();       // collapsed parent tasks
const collapsedGroups = new Set(); // collapsed group sections
const STATUS_ORDER = ['todo', 'in_progress', 'done'];
const STATUS_META = {
    todo:        { label: 'Todo',        icon: '○', cls: 's-todo' },
    in_progress: { label: 'In progress', icon: '◑', cls: 's-prog' },
    done:        { label: 'Done',        icon: '✓', cls: 's-done' },
};
const SIZES = ['', 'XS', 'S', 'M', 'L', 'XL'];

// Grouping definitions: order of groups, header dot class + label per value.
const GROUP_DEFS = {
    status: {
        field: t => t.status || 'todo',
        order: ['todo', 'in_progress', 'done'],
        meta: { todo: ['Todo', 'g-todo'], in_progress: ['In progress', 'g-prog'], done: ['Done', 'g-done'] },
    },
    priority: {
        field: t => t.priority || 'medium',
        order: ['high', 'medium', 'low'],
        meta: { high: ['🔴 High', 'g-high'], medium: ['🟡 Medium', 'g-med'], low: ['🟢 Low', 'g-low'] },
    },
    size: {
        field: t => t.size || '',
        order: ['XL', 'L', 'M', 'S', 'XS', ''],
        meta: { XL: ['XL', 'g-sz'], L: ['L', 'g-sz'], M: ['M', 'g-sz'], S: ['S', 'g-sz'], XS: ['XS', 'g-sz'], '': ['No size', 'g-none'] },
    },
};

function setGroupBy(v) { groupBy = v; loadWsTasks(); }

function sel(field, id, value, options) {
    const opts = options.map(o =>
        `<option value="${o.v}" ${o.v === value ? 'selected' : ''}>${esc(o.t)}</option>`).join('');
    const pill = (field === 'status' || field === 'priority') ? ' pill-sel' : '';
    return `<select class="cell-sel${pill} ${field}-${esc(value||'none')}" onchange="patchField(${id}, '${field}', this.value)">${opts}</select>`;
}

// Size shown as a compact colored badge (XS–XL), still a dropdown.
function sizeBadge(t) {
    const opts = SIZES.map(s => `<option value="${s}" ${s === t.size ? 'selected' : ''}>${s || '–'}</option>`).join('');
    return `<select class="cell-sel size-sel size-${t.size || 'none'}" onchange="patchField(${t.id}, 'size', this.value)">${opts}</select>`;
}

// Labels rendered as colored chips; click the cell to edit the comma list.
const LABEL_COLORS = [['#fee2e2','#a12327'],['#dde8f2','#14417f'],['#d9efdf','#12622f'],['#f3e6cb','#8a5a06'],['#ece7ff','#5b4ab0'],['#f7e2dc','#a1341d']];
function labelColor(name) { let h = 0; for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0; return LABEL_COLORS[h % LABEL_COLORS.length]; }
function labelsCell(t) {
    const labels = (t.labels || '').split(',').map(s => s.trim()).filter(Boolean);
    const chips = labels.map(l => { const [bg, fg] = labelColor(l); return `<span class="lchip" style="background:${bg};color:${fg}">${esc(l)}</span>`; }).join('');
    return `<span class="labels-cell" onclick="editLabels(this, ${t.id})" data-labels="${esc(t.labels || '')}">${chips || '<span class="lc-empty">＋ label</span>'}</span>`;
}
function editLabels(el, id) {
    const cur = el.getAttribute('data-labels') || '';
    el.innerHTML = `<input class="cell-labels-inp" value="${cur.replace(/"/g, '&quot;')}" placeholder="comma,separated">`;
    const inp = el.querySelector('input');
    inp.focus();
    inp.onblur = async () => { await api('PATCH', '/tasks/' + id, { labels: inp.value }); loadWsTasks(); };
    inp.onkeydown = e => { if (e.key === 'Enter') inp.blur(); };
}

// One task row (parent or subtask). `kids` = subtask list (parents); `isLast`
// marks the final subtask so the tree draws └ instead of ├.
function taskRowHTML(t, num, kids, isLast) {
    const sm = STATUS_META[t.status] || STATUS_META.todo;
    const isSub = t.parent_id > 0;
    let lead = '<span class="tw-spacer"></span>';
    let badge = '';
    if (isSub) {
        lead = '';                       // connector is drawn in the # gutter
    } else if (kids && kids.length) {
        const done = kids.filter(k => k.done).length;
        const open = !collapsed.has(t.id);
        lead = `<button class="tw-toggle" onclick="toggleTask(${t.id})">${open ? '▾' : '▸'}</button>`;
        badge = `<span class="sub-badge">${done}/${kids.length}</span>`;
    }
    const subAdd = isSub ? '' : `<button class="tw-addsub" title="Add subtask" onclick="addSubtask(${t.id})">+ sub</button>`;
    const rowCls = isSub ? 'is-sub' + (isLast ? ' last' : '') : '';
    return `<div class="ght-row p-${esc(t.priority || 'medium')} ${rowCls}">
        <span class="c-num">${isSub ? '' : num}</span>
        <span class="c-title">
            ${lead}
            <span class="tstat ${sm.cls}">${sm.icon}</span>
            <input class="cell-title" value="${esc(t.title)}" onchange="patchField(${t.id}, 'title', this.value)">
            <span class="task-id">#${t.id}</span>${badge}${subAdd}
        </span>
        ${sel('status', t.id, t.status, STATUS_ORDER.map(s => ({v:s, t:STATUS_META[s].label})))}
        ${sel('priority', t.id, t.priority, [{v:'high',t:'High'},{v:'medium',t:'Medium'},{v:'low',t:'Low'}])}
        ${sizeBadge(t)}
        <input class="cell-date" type="date" value="${esc(t.due_date)}" onchange="patchField(${t.id}, 'due_date', this.value)">
        ${labelsCell(t)}
        <button class="btn-remove" onclick="deleteTask(${t.id})">DEL</button>
    </div>`;
}

// Plain (non-pill) select for card meta — colored text only, no pill background.
function csel(field, id, value, options) {
    const opts = options.map(o => `<option value="${o.v}" ${o.v === value ? 'selected' : ''}>${esc(o.t)}</option>`).join('');
    return `<select class="csel ${field}-${esc(value || 'none')}" onchange="patchField(${id}, '${field}', this.value)">${opts}</select>`;
}
const STATUS_OPTS = STATUS_ORDER.map(s => ({ v: s, t: STATUS_META[s].label }));
const PRIORITY_OPTS = [{v:'high',t:'High'},{v:'medium',t:'Medium'},{v:'low',t:'Low'}];
const SIZE_OPTS = SIZES.map(s => ({ v: s, t: s || '–' }));

// ── Task cards: each top-level task is its own collapsible section ────────────
function renderCards(tasks) {
    const wrap = document.getElementById('ws-table');
    const childrenOf = {};
    const top = [];
    tasks.forEach(t => { if (t.parent_id) (childrenOf[t.parent_id] = childrenOf[t.parent_id] || []).push(t); else top.push(t); });

    const card = (t) => {
        const kids = childrenOf[t.id] || [];
        const hasKids = kids.length > 0;
        const open = !collapsed.has(t.id);
        const sm = STATUS_META[t.status] || STATUS_META.todo;
        const done = kids.filter(k => k.done).length;

        const head = `<div class="tcard-head">
            ${hasKids ? `<button class="tw-toggle" onclick="toggleTask(${t.id})">${open ? '▾' : '▸'}</button>` : '<span class="tw-spacer"></span>'}
            <span class="tstat ${sm.cls}">${sm.icon}</span>
            <input class="cell-title tcard-title" value="${esc(t.title)}" onchange="patchField(${t.id}, 'title', this.value)">
            <span class="task-id">#${t.id}</span>
            ${hasKids ? `<span class="sub-badge">${done}/${kids.length}</span>` : ''}
            <span class="flex-spacer"></span>
            ${csel('status', t.id, t.status, STATUS_OPTS)}
            <button class="tw-addsub" onclick="revealSubAdd(this, ${t.id})">+ sub</button>
            <button class="btn-remove" onclick="deleteTask(${t.id})">DEL</button>
        </div>`;

        const showBody = hasKids ? open : true;
        let body = '';
        if (showBody) {
            let subs = '';
            if (hasKids) {
                subs = '<div class="tcard-subs">' + kids.map((k, i) => {
                    const km = STATUS_META[k.status] || STATUS_META.todo;
                    return `<div class="tcard-sub">
                        <span class="tree-branch">${i === kids.length - 1 ? '└─' : '├─'}</span>
                        <span class="tstat ${km.cls}">${km.icon}</span>
                        <input class="cell-title" value="${esc(k.title)}" onchange="patchField(${k.id}, 'title', this.value)">
                        <span class="task-id">#${k.id}</span>
                        <span class="flex-spacer"></span>
                        ${csel('status', k.id, k.status, STATUS_OPTS)}
                        <button class="btn-remove" onclick="deleteTask(${k.id})">DEL</button>
                    </div>`;
                }).join('') + `<div class="sub-add"><span class="tree-branch">└─</span><input class="sub-add-inp" data-parent="${t.id}" placeholder="Add a subtask…" onkeydown="subAddKey(event, ${t.id})"></div></div>`;
            }
            const meta = `<div class="tcard-meta">
                <span><label>Priority</label>${csel('priority', t.id, t.priority, PRIORITY_OPTS)}</span>
                <span><label>Size</label>${csel('size', t.id, t.size, SIZE_OPTS)}</span>
                <span><label>Due</label><input class="cell-date" type="date" value="${esc(t.due_date)}" onchange="patchField(${t.id}, 'due_date', this.value)"></span>
                <span class="meta-labels"><label>Labels</label>${labelsCell(t)}</span>
            </div>`;
            body = subs + meta;
        }
        return `<div class="tcard ${hasKids ? 'has-kids' : ''} p-${esc(t.priority || 'medium')}">${head}${body}</div>`;
    };

    let html = '<div class="tcards">';
    if (groupBy === 'none') {
        html += top.map(card).join('');
        html += `<div class="tcard-add" onclick="addRow('')">＋ Add task</div>`;
    } else {
        const def = GROUP_DEFS[groupBy];
        const buckets = {}; def.order.forEach(k => buckets[k] = []);
        top.forEach(t => { const k = def.field(t); (buckets[k] = buckets[k] || []).push(t); });
        const keys = Object.keys(buckets).sort((a, b) => (def.order.indexOf(a) < 0 ? 99 : def.order.indexOf(a)) - (def.order.indexOf(b) < 0 ? 99 : def.order.indexOf(b)));
        for (const key of keys) {
            const items = buckets[key]; if (!items.length && def.order.indexOf(key) < 0) continue;
            const [label, dotCls] = (def.meta[key] || [key || '—', 'g-none']);
            html += `<div class="tcards-group"><span class="ght-dot ${dotCls}"></span>${esc(label)}<span class="grp-count">${items.length}</span></div>`;
            html += items.map(card).join('');
            html += `<div class="tcard-add" onclick="addRow('${esc(key)}')">＋ Add task</div>`;
        }
    }
    html += '</div>';
    wrap.innerHTML = html;

    if (_focusSubAdd) {
        const el = wrap.querySelector('.sub-add-inp[data-parent="' + _focusSubAdd + '"]');
        _focusSubAdd = 0;
        if (el) el.focus();
    }
}

function renderTable(tasks) {
    const wrap = document.getElementById('ws-table');

    // Split into top-level tasks and a children map.
    const childrenOf = {};
    const top = [];
    tasks.forEach(t => {
        if (t.parent_id) (childrenOf[t.parent_id] = childrenOf[t.parent_id] || []).push(t);
        else top.push(t);
    });

    const counter = { n: 0 };
    const renderParent = (t) => {
        counter.n++;
        const kids = childrenOf[t.id] || [];
        const hasKids = kids.length > 0;
        let h = taskRowHTML(t, counter.n, kids);
        if (hasKids && !collapsed.has(t.id)) {
            h += '<div class="subtree">' +
                kids.map((k, i) => { counter.n++; return taskRowHTML(k, counter.n, null, i === kids.length - 1); }).join('') +
                '</div>';
        }
        return `<div class="task-block${hasKids ? ' has-kids' : ''}">${h}</div>`;
    };

    let html = '<div class="ght">';
    html += `<div class="ght-row ght-header">
        <span class="c-num">#</span><span class="c-title">Title</span>
        <span>Status</span><span>Priority</span><span>Size</span><span>Due</span><span>Labels</span><span></span></div>`;

    if (groupBy === 'none') {
        html += top.map(renderParent).join('');
        html += `<div class="ght-add" onclick="addRow('')">+ Add task</div>`;
    } else {
        const def = GROUP_DEFS[groupBy];
        const buckets = {};
        def.order.forEach(k => buckets[k] = []);
        top.forEach(t => { const k = def.field(t); (buckets[k] = buckets[k] || []).push(t); });
        const keys = Object.keys(buckets).sort((a, b) => {
            const ia = def.order.indexOf(a), ib = def.order.indexOf(b);
            return (ia < 0 ? 99 : ia) - (ib < 0 ? 99 : ib);
        });
        for (const key of keys) {
            const items = buckets[key];
            if (!items.length && def.order.indexOf(key) < 0) continue;
            const [label, dotCls] = (def.meta[key] || [key || '—', 'g-none']);
            const done = items.filter(t => t.done).length;
            const pct = items.length ? Math.round(done * 100 / items.length) : 0;
            const gid = groupBy + ':' + key;
            const gopen = !collapsedGroups.has(gid);
            html += `<div class="ght-group ${gopen ? '' : 'closed'}" onclick="toggleGroup('${esc(gid)}')">
                <span class="grp-chevron">${gopen ? '▾' : '▸'}</span>
                <span class="ght-dot ${dotCls}"></span>
                <span class="grp-name">${esc(label)}</span>
                <span class="grp-count">${items.length}</span>
                <span class="ght-prog"><span style="width:${pct}%"></span></span><span class="ght-pct">${pct}%</span></div>`;
            if (gopen) {
                html += `<div class="ght-section">`;
                html += items.map(renderParent).join('');
                html += `<div class="ght-add" onclick="addRow('${esc(key)}')">+ Add task</div>`;
                html += `</div>`;
            }
        }
    }
    html += '</div>';
    wrap.innerHTML = html;
}

function toggleTask(id) {
    if (collapsed.has(id)) collapsed.delete(id); else collapsed.add(id);
    loadWsTasks();
}
function toggleGroup(gid) {
    if (collapsedGroups.has(gid)) collapsedGroups.delete(gid); else collapsedGroups.add(gid);
    loadWsTasks();
}
async function addSubtask(parentId) {
    const title = prompt('New subtask:');
    if (!title || !title.trim()) return;
    const r = await api('POST', '/tasks', { title: title.trim(), priority: 'medium', project_id: currentProject, parent_id: parentId });
    if (r && r.success) { collapsed.delete(parentId); notify('Subtask added'); loadWsTasks(); refreshWsProgress(); }
}

// Inline subtask entry: Enter creates & keeps focus for rapid adding; Esc cancels.
async function subAddKey(e, parentId) {
    if (e.key === 'Escape') { e.target.value = ''; e.target.blur(); return; }
    if (e.key !== 'Enter') return;
    const v = e.target.value.trim();
    if (!v) return;
    const r = await api('POST', '/tasks', { title: v, priority: 'medium', project_id: currentProject, parent_id: parentId });
    if (r && r.success) {
        collapsed.delete(parentId);
        _focusSubAdd = parentId;            // refocus the input after re-render
        loadWsTasks();
        refreshWsProgress();
    }
}
// Reveal an inline subtask input on a card (used by the "+ sub" button).
function revealSubAdd(btn, id) {
    const card = btn.closest('.tcard');
    let inp = card.querySelector('.sub-add-inp');
    if (inp) { inp.focus(); return; }        // parent card already has one
    const box = document.createElement('div');
    box.className = 'tcard-subs';
    box.innerHTML = `<div class="sub-add"><span class="tree-branch">└─</span><input class="sub-add-inp" placeholder="Add a subtask…"></div>`;
    card.querySelector('.tcard-head').after(box);
    inp = box.querySelector('input');
    inp.onkeydown = ev => subAddKey(ev, id);
    inp.focus();
}

async function patchField(id, field, value) {
    const body = {}; body[field] = value;
    const r = await api('PATCH', '/tasks/' + id, body);
    if (r && r.success) {
        if (r.xp) notify(`+${r.xp} XP — task done!`);
        if (field === groupBy || field === 'status') loadWsTasks(); // regroup
        refreshWsProgress();
    } else notify('Update failed', true);
}
async function addRow(groupValue) {
    const title = prompt('New task title:');
    if (!title || !title.trim()) return;
    const r = await api('POST', '/tasks', { title: title.trim(), priority: 'medium', project_id: currentProject });
    // Apply the grouped field so the new task lands in the group it was added to.
    if (r && r.success && groupValue !== '' && !(groupBy === 'status' && groupValue === 'todo')
        && !(groupBy === 'priority' && groupValue === 'medium')) {
        await api('PATCH', '/tasks/' + r.id, { [groupBy]: groupValue });
    }
    loadWsTasks(); refreshWsProgress();
}
async function refreshWsProgress() {
    const projects = await api('GET', '/projects');
    const p = (projects || []).find(x => x.id === currentProject);
    if (p) document.getElementById('ws-progress').style.width = p.progress + '%';
}
function renderBoard(tasks) {
    const cols = { todo: [], in_progress: [], done: [] };
    tasks.forEach(t => (cols[t.status] || cols.todo).push(t));
    const board = document.getElementById('ws-board');
    const colHtml = (key, label) => `
        <div class="kanban-col">
            <div class="kanban-head">${label} <span class="count">${cols[key].length}</span></div>
            ${cols[key].map(t => `
                <div class="kanban-card">
                    <div>${PRIORITY[t.priority]||'🟡'} ${esc(t.title)}</div>
                    <div class="kanban-actions">
                        <button class="btn-toggle off" onclick="cycleStatus(${t.id}, '${t.status}')">move →</button>
                        <button class="btn-remove" onclick="deleteTask(${t.id})">DEL</button>
                    </div>
                </div>`).join('') || '<div class="no-data">—</div>'}
        </div>`;
    board.innerHTML = colHtml('todo','TODO') + colHtml('in_progress','IN PROGRESS') + colHtml('done','DONE');
}
async function addWsTask() {
    const title = document.getElementById('ws-task-title').value.trim();
    const priority = document.getElementById('ws-task-priority').value;
    if (!title) return;
    const r = await api('POST', '/tasks', { title, priority, project_id: currentProject });
    if (r && r.success) { document.getElementById('ws-task-title').value = ''; notify('Task added'); refreshWs(); } else notify('Failed', true);
}
async function deleteProject() {
    if (!confirm('Delete this project and its tasks?')) return;
    const r = await api('DELETE', '/projects/' + currentProject);
    if (r && r.success) { notify('Project deleted'); switchView('projects'); }
}
async function refreshWs() {
    const projects = await api('GET', '/projects');
    const p = (projects || []).find(x => x.id === currentProject);
    if (p) document.getElementById('ws-progress').style.width = p.progress + '%';
    loadWsTasks();
}

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
    const r = await api('DELETE', '/tasks/' + id);
    if (r && r.success) { notify('Task deleted'); refreshCurrent(); }
}
function refreshCurrent() {
    const active = document.querySelector('.view.active').dataset.view;
    if (active === 'home') loadHome();
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
    const recent = await api('GET', '/recent?limit=40');
    renderRecent('activity-recent', recent || []);
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

// ── Init ─────────────────────────────────────────────────────────────────────
document.getElementById('quick-task').addEventListener('keydown', e => { if (e.key === 'Enter') quickAddTask(); });
document.getElementById('ws-task-title').addEventListener('keydown', e => { if (e.key === 'Enter') addWsTask(); });
document.getElementById('today-add').addEventListener('keydown', e => { if (e.key === 'Enter') addTodayTask(); });
switchView('today');
checkReminders();
setInterval(checkReminders, 60000);
