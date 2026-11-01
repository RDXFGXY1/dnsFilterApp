// A read-only overview of existing Discipline work; actions use its normal API.
const attentionState = { tasks: [], reminders: [], filter: 'all', plan: [], plannedMinutes: 45 };
const attentionDate = value => { const d = new Date(String(value || '').replace(' ', 'T')); return Number.isNaN(d.getTime()) ? null : d; };
const attentionDay = value => String(value || '').slice(0, 10);
const attentionOpen = task => !task.done && !task.archived && task.status !== 'done';
const attentionBlocked = task => Boolean(String(task.blocked_reason || '').trim() || String(task.waiting_for || '').trim());

async function loadAttention() {
    const [tasks, reminders] = await Promise.all([api('GET', '/tasks?all=1'), api('GET', '/reminders')]);
    if (!Array.isArray(tasks) || !Array.isArray(reminders)) return notify(tasks?.error || reminders?.error || 'Could not load attention items', true);
    attentionState.tasks = tasks.filter(attentionOpen);
    attentionState.reminders = reminders.filter(r => !r.done);
    renderAttention();
    if (attentionState.plan.length) buildRescuePlan(false);
}

function attentionItems() {
    const today = todayStr(), now = new Date(), week = new Date(today + 'T00:00:00');
    week.setDate(week.getDate() + 7);
    const taskItems = attentionState.tasks.map(t => {
        const due = attentionDay(t.due_date), reminder = attentionDate(t.reminder_at);
        let group = 'unplanned';
        if (attentionBlocked(t)) group = 'blocked';
        else if (due && due < today || reminder && reminder < now && attentionDay(t.reminder_at) < today) group = 'overdue';
        else if (due === today || reminder && attentionDay(t.reminder_at) === today) group = 'today';
        else if (due && due <= fmtDate(week) || reminder && reminder <= week) group = 'upcoming';
        else if (due || reminder) group = 'later';
        const sortDate = due || attentionDay(t.reminder_at) || '9999-12-31';
        return { kind: 'task', data: t, group, sortDate, reminder: Boolean(reminder) };
    });
    const reminderItems = attentionState.reminders.map(r => {
        const at = attentionDate(r.remind_at), day = attentionDay(r.remind_at);
        return { kind: 'reminder', data: r, group: !at ? 'upcoming' : at < now && day < today ? 'overdue' : day === today ? 'today' : at <= week ? 'upcoming' : 'later', sortDate: day || '9999-12-31', reminder: true };
    });
    return [...taskItems, ...reminderItems];
}

function renderAttention() {
    const items = attentionItems(), counts = Object.fromEntries(['overdue','today','upcoming','blocked'].map(k => [k, items.filter(i => i.group === k).length]));
    document.getElementById('attention-kpis').innerHTML = [['Overdue',counts.overdue],['Today',counts.today],['Next 7 days',counts.upcoming],['Blocked',counts.blocked]].map(([label,n]) => `<div><strong>${n}</strong><span>${label}</span></div>`).join('');
    const badge = document.getElementById('attention-nav-count');
    if (badge) { badge.textContent = counts.overdue + counts.today; badge.hidden = !Number(badge.textContent); }
    document.querySelectorAll('[data-attention-filter]').forEach(b => b.classList.toggle('active', b.dataset.attentionFilter === attentionState.filter));
    const filtered = items.filter(i => attentionState.filter === 'all' ? i.group !== 'later' : attentionState.filter === 'reminders' ? i.reminder && i.group !== 'later' : i.group === attentionState.filter)
        .sort((a,b) => ({overdue:0,today:1,blocked:2,upcoming:3,unplanned:4,later:5}[a.group] - {overdue:0,today:1,blocked:2,upcoming:3,unplanned:4,later:5}[b.group]) || a.sortDate.localeCompare(b.sortDate) || ({high:0,medium:1,low:2}[a.data.priority] ?? 1) - ({high:0,medium:1,low:2}[b.data.priority] ?? 1));
    document.getElementById('attention-list').innerHTML = filtered.length ? filtered.map(attentionCard).join('') : '<p class="no-data">Nothing needs attention in this filter.</p>';
}

function attentionCard(item) {
    const x = item.data, id = Number(x.id), task = item.kind === 'task', title = task ? x.title : x.text;
    const when = task ? (x.due_date ? `Due ${esc(x.due_date)}` : x.reminder_at ? `Reminder ${esc(x.reminder_at)}` : 'No date yet') : `Reminder ${esc(x.remind_at)}`;
    const why = task && attentionBlocked(x) ? `<div class="attention-reason"><span>Blocked:</span><span>${esc(x.blocked_reason || x.waiting_for)}</span><button onclick="attentionClearBlock(${id})">Clear</button></div>` : '';
    const actions = task ? `<button onclick="attentionDone(${id})">Complete</button>${x.reminder_at ? `<button onclick="attentionSnooze('task',${id})">Snooze 1h</button>` : ''}<button onclick="attentionReschedule(${id})">Reschedule</button><button onclick="attentionSetReason(${id})">${attentionBlocked(x) ? 'Change reason' : 'Why delayed?'}</button>${attentionBlocked(x) ? '' : `<button onclick="attentionFocus(${id})">Start Focus</button>`}` : `<button onclick="attentionReminderDone(${id})">Done</button><button onclick="attentionSnooze('reminder',${id})">Snooze 1h</button>`;
    return `<article class="attention-card ${item.group}"><div><span class="attention-kind">${task ? 'Task' : 'Reminder'} · ${esc(item.group)}</span><h3>${esc(title)}</h3><p>${when}${task && x.estimate_minutes ? ` · ${Number(x.estimate_minutes)} min` : ''}${task ? ` · ${esc(x.priority || 'medium')} priority` : ''}</p>${why}</div><div class="attention-actions">${actions}</div></article>`;
}

function buildRescuePlan(announce = true) {
    const minutes = Number(document.getElementById('attention-minutes').value);
    attentionState.plannedMinutes = minutes;
    const today = todayStr(), rank = { high: 0, medium: 1, low: 2 };
    const candidates = attentionState.tasks.filter(t => !attentionBlocked(t) && (attentionDay(t.due_date) <= today && t.due_date || t.my_day || attentionDay(t.reminder_at) <= today && t.reminder_at))
        .sort((a,b) => Number(Boolean(b.due_date && attentionDay(b.due_date) < today)) - Number(Boolean(a.due_date && attentionDay(a.due_date) < today)) || (rank[a.priority] ?? 1) - (rank[b.priority] ?? 1) || (a.due_date || '9999').localeCompare(b.due_date || '9999'));
    const selected = [];
    let left = minutes;
    for (const t of candidates) { const estimate = Number(t.estimate_minutes) > 0 ? Number(t.estimate_minutes) : 30; if (selected.length === 3) break; if (estimate <= left) { selected.push(t); left -= estimate; } }
    attentionState.plan = selected;
    const box = document.getElementById('attention-plan');
    box.innerHTML = selected.length ? `<p class="attention-plan-note">${minutes - left} of ${minutes} minutes planned. Tasks without an estimate count as 30 minutes.</p>` + selected.map((t,i) => `<article class="attention-card"><div><span class="attention-kind">Step ${i+1} · ${Number(t.estimate_minutes) || 30} min</span><h3>${esc(t.title)}</h3><p>${t.due_date ? 'Due ' + esc(t.due_date) : 'My Day'} · ${esc(t.priority || 'medium')} priority</p></div><div class="attention-actions"><button onclick="attentionFocus(${Number(t.id)})">Start Focus</button><button onclick="attentionDone(${Number(t.id)})">Complete</button></div></article>`).join('') : '<p class="no-data">No actionable task fits this time. Give an overdue task an estimate, clear a block, or choose more time.</p>';
    if (announce) notify(selected.length ? `Plan ready: ${selected.length} task${selected.length === 1 ? '' : 's'}` : 'No task fits the available time');
}

async function attentionUpdate(id, patch) { const r = await api('PATCH', `/tasks/${id}`, patch); if (!r?.success) return notify(r?.error || 'Could not update task', true); await loadAttention(); }
async function attentionDone(id) { await attentionUpdate(id, { status: 'done' }); }
async function attentionReminderDone(id) { const r = await api('PATCH', `/reminders/${id}`, { done: true }); if (!r?.success) return notify(r?.error || 'Could not complete reminder', true); await loadAttention(); }
async function attentionSnooze(kind, id) { const d = new Date(Date.now() + 60 * 60 * 1000), when = `${fmtDate(d)} ${String(d.getHours()).padStart(2,'0')}:${String(d.getMinutes()).padStart(2,'0')}`; if (kind === 'task') return attentionUpdate(id, { reminder_at: when }); const r = await api('PATCH', `/reminders/${id}`, { remind_at: when }); if (!r?.success) return notify(r?.error || 'Could not snooze reminder', true); await loadAttention(); }
async function attentionReschedule(id) { const t = attentionState.tasks.find(x => Number(x.id) === id); const date = prompt('New due date (YYYY-MM-DD)', attentionDay(t?.due_date) || todayStr()); if (date === null) return; const parsed = new Date(date + 'T00:00:00'); if (!/^\d{4}-\d{2}-\d{2}$/.test(date) || Number.isNaN(parsed.getTime()) || `${parsed.getFullYear()}-${String(parsed.getMonth()+1).padStart(2,'0')}-${String(parsed.getDate()).padStart(2,'0')}` !== date) return notify('Enter a valid date in YYYY-MM-DD format', true); await attentionUpdate(id, { due_date: date }); }
async function attentionSetReason(id) { const t = attentionState.tasks.find(x => Number(x.id) === id); const reason = prompt('Why is this task delayed? Leave blank to clear the block.', t?.blocked_reason || ''); if (reason === null) return; await attentionUpdate(id, { blocked_reason: reason.trim() }); }
async function attentionClearBlock(id) { await attentionUpdate(id, { blocked_reason: '', waiting_for: '' }); }
async function attentionFocus(id) { await loadTodoHub(); await todoFocus(id); }
document.querySelectorAll('[data-attention-filter]').forEach(b => b.addEventListener('click', () => { attentionState.filter = b.dataset.attentionFilter; renderAttention(); }));
