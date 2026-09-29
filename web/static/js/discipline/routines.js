// ── Habits ───────────────────────────────────────────────────────────────────
const DOW = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];
async function loadHabits() {
    const habits = await api('GET', '/habits');
    const wrap = document.getElementById('habits-list');
    const all = habits || [];
    const doneToday = all.filter(h => h.done_today).length;
    const avgWeek = all.length ? Math.round(all.reduce((n,h) => n + h.week_done, 0) * 100 / (all.length * 7)) : 0;
    renderViewKpis('habit-kpis', [
        ['Active habits', all.length, 'routines being tracked'],
        ['Done today', doneToday, all.length ? `${Math.round(doneToday*100/all.length)}% daily completion` : 'start with one'],
        ['Weekly consistency', avgWeek + '%', 'across all routines', avgWeek >= 70 ? 'good' : avgWeek < 40 ? 'warn' : ''],
        ['Strongest streak', all.length ? Math.max(...all.map(h => h.current_streak)) : 0, 'consecutive days'],
    ]);
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
    const open = _calTasks.filter(t => !t.done);
    const unscheduled = open.filter(t => !t.due_date);
    const monthPrefix = `${calState.y}-${String(calState.m + 1).padStart(2, '0')}`;
    const monthDue = open.filter(t => t.due_date && t.due_date.startsWith(monthPrefix));
    const overdue = open.filter(t => daysFromToday(t.due_date) < 0);
    renderViewKpis('calendar-kpis', [
        ['This month', monthDue.length, 'open deadlines'], ['Unscheduled', unscheduled.length, 'tasks without a date', unscheduled.length > 5 ? 'warn' : ''],
        ['Overdue', overdue.length, 'past deadlines', overdue.length ? 'danger' : 'good'], ['Busiest day', Math.max(0, ...Object.values(byDay).map(x => x.filter(t => !t.done).length)), 'maximum open tasks'],
    ]);
    document.getElementById('cal-unscheduled-count').textContent = unscheduled.length;
    document.getElementById('cal-unscheduled').innerHTML = unscheduled.length ? unscheduled.slice(0,8).map(t => `<li><span>${PRIORITY[t.priority] || '🟡'} ${esc(t.title)}</span><button class="action-focus" onclick="switchView('projects')">Plan</button></li>`).join('') : '<li class="no-data">Every open task has a date.</li>';
    document.getElementById('cal-pressure').innerHTML = [
        ['Overdue debt', overdue.length, overdue.length ? 'Renegotiate these dates.' : 'No deadline debt.'],
        ['Next 7 days', open.filter(t => { const d=daysFromToday(t.due_date); return d>=0&&d<=7; }).length, 'Upcoming execution load.'],
        ['High priority', open.filter(t => t.priority==='high').length, 'Strategic commitments.'],
    ].map(([n,c,s]) => `<div class="risk-row"><span class="risk-dot ${c ? 'hot' : 'clear'}"></span><div><strong>${n}</strong><small>${s}</small></div><b>${c}</b></div>`).join('');
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
let _allReminders = [];
async function loadReminders() {
    const rem = await api('GET', '/reminders');
    _allReminders = rem || [];
    const now = new Date();
    const overdue = _allReminders.filter(r => !r.done && new Date(r.remind_at.replace(' ','T')) <= now).length;
    const upcoming = _allReminders.filter(r => !r.done && new Date(r.remind_at.replace(' ','T')) > now).length;
    renderViewKpis('reminder-kpis', [['Open',overdue+upcoming,'active reminders'],['Overdue',overdue,'needs attention',overdue?'danger':'good'],['Upcoming',upcoming,'scheduled ahead'],['Completed',_allReminders.filter(r=>r.done).length,'closed loops']]);
    renderReminderList();
}
function renderReminderList() {
    const rem = _allReminders;
    const ul = document.getElementById('rem-list');
    if (!rem || !rem.length) { ul.innerHTML = '<li class="no-data">No reminders. Add one above.</li>'; return; }
    const now = new Date();
    const filter = document.getElementById('rem-filter')?.value || 'open';
    const shown = rem.filter(r => filter === 'all' || (filter === 'open' && !r.done) || (filter === 'overdue' && !r.done && new Date(r.remind_at.replace(' ','T')) <= now));
    ul.innerHTML = shown.length ? shown.map(r => {
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
    }).join('') : '<li class="no-data">No reminders match this filter.</li>';
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
    const [rem, tasks] = await Promise.all([api('GET', '/reminders'), api('GET', '/tasks?all=1')]);
    if (!rem && !tasks) return;
    const now = new Date();
    (rem || []).forEach(r => {
        if (r.done || _remNotified.has(r.id)) return;
        if (new Date(r.remind_at.replace(' ', 'T')) <= now) {
            _remNotified.add(r.id);
            notify('🔔 ' + r.text);
            if ('Notification' in window && Notification.permission === 'granted') new Notification('Reminder', { body: r.text });
        }
    });
    (tasks || []).forEach(t => {
        const key = 'task-' + t.id;
        if (t.done || t.archived || !t.reminder_at || _remNotified.has(key)) return;
        if (new Date(t.reminder_at.replace(' ', 'T')) <= now) {
            _remNotified.add(key);
            notify('🔔 Task: ' + t.title);
            if ('Notification' in window && Notification.permission === 'granted') new Notification('Project task reminder', { body: t.title });
        }
    });
}
