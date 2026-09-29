// ── Command Center: planning, triage, focus, and review ───────────────────────
let commandCache = { tasks: [], projects: [], goals: [], stats: {} };
let focusSeconds = 25 * 60;
let focusSessionMinutes = 25;
let focusTimer = null;

function daysFromToday(ds) {
    if (!ds) return 999;
    const a = new Date(todayStr() + 'T00:00:00');
    const b = new Date(ds + 'T00:00:00');
    return Math.round((b - a) / 86400000);
}

function actionScore(t) {
    let score = t.priority === 'high' ? 42 : t.priority === 'medium' ? 24 : 10;
    const days = daysFromToday(t.due_date);
    if (days < 0) score += 55 + Math.min(20, Math.abs(days) * 3);
    else if (days === 0) score += 45;
    else if (days <= 3) score += 28 - days * 5;
    if (t.status === 'in_progress') score += 18;
    if (t.project_id) score += 6;
    if (t.parent_id) score -= 3;
    return score;
}

function dueLanguage(t) {
    const d = daysFromToday(t.due_date);
    if (!t.due_date) return 'No deadline';
    if (d < 0) return `${Math.abs(d)}d overdue`;
    if (d === 0) return 'Due today';
    if (d === 1) return 'Due tomorrow';
    return `Due in ${d}d`;
}

async function loadCommand() {
    const [tasksRaw, projectsRaw, goalsRaw, statsRaw, weekRaw] = await Promise.all([
        api('GET', '/tasks?all=1'), api('GET', '/projects'), api('GET', '/goals'),
        api('GET', '/stats'), api('GET', '/heatmap?days=7'),
    ]);
    const tasks = tasksRaw || [], projects = projectsRaw || [], goals = goalsRaw || [];
    const open = tasks.filter(t => !t.done), overdue = open.filter(t => daysFromToday(t.due_date) < 0);
    const today = open.filter(t => daysFromToday(t.due_date) === 0);
    const blocked = open.filter(t => (t.labels || '').toLowerCase().split(',').some(x => x.trim() === 'blocked'));
    const doing = open.filter(t => t.status === 'in_progress');
    commandCache = { tasks, projects, goals, stats: statsRaw || {} };

    const dueSoon = open.filter(t => daysFromToday(t.due_date) >= 0 && daysFromToday(t.due_date) <= 7).length;
    const completion = tasks.length ? Math.round(tasks.filter(t => t.done).length * 100 / tasks.length) : 100;
    const pressure = overdue.length * 9 + dueSoon * 3 + doing.length * 2;
    const score = Math.max(0, Math.min(100, Math.round(completion * .35 + Math.min(30, (statsRaw?.current_streak || 0) * 4) + 45 - Math.min(45, pressure))));
    document.getElementById('command-score').textContent = score;
    document.getElementById('command-greeting').textContent = `${greeting()}, here is your execution brief.`;
    document.getElementById('command-brief').textContent = overdue.length
        ? `${overdue.length} overdue item${overdue.length === 1 ? '' : 's'} need a decision. Protect focus before adding more work.`
        : today.length ? `${today.length} mission${today.length === 1 ? '' : 's'} due today. Your system is under control.`
        : 'No urgent deadlines. Use this space to create strategic momentum.';

    document.getElementById('command-metrics').innerHTML = [
        ['Open loop', open.length, 'All unfinished work'],
        ['Due today', today.length, today.length ? 'Execution zone' : 'Clear runway'],
        ['Overdue', overdue.length, overdue.length ? 'Needs triage' : 'Nothing behind'],
        ['In progress', doing.length, doing.length > 5 ? 'WIP is too high' : 'Within focus limit'],
        ['Blocked', blocked.length, blocked.length ? 'Resolve dependencies' : 'Flow is clear'],
    ].map(([l,v,s]) => `<div class="command-metric"><span>${esc(l)}</span><strong>${v}</strong><small>${esc(s)}</small></div>`).join('');

    const projectNames = Object.fromEntries(projects.map(p => [p.id, p.name]));
    const ranked = [...open].sort((a,b) => actionScore(b) - actionScore(a)).slice(0, 7);
    document.getElementById('command-next').innerHTML = ranked.length ? ranked.map((t,i) => `
        <article class="next-action ${daysFromToday(t.due_date) < 0 ? 'at-risk' : ''}">
            <span class="action-rank">${i + 1}</span>
            <button class="task-check" onclick="toggleDone(${t.id}, true)" title="Complete"></button>
            <div class="action-copy"><strong>${esc(t.title)}</strong><span>${esc(projectNames[t.project_id] || 'Inbox')} · ${esc(dueLanguage(t))}</span></div>
            <span class="priority-pill p-${esc(t.priority)}">${esc(t.priority)}</span>
            <button class="action-focus" onclick="focusThis(${t.id})">Focus</button>
        </article>`).join('') : '<div class="command-empty">Everything is complete. Choose a meaningful new outcome instead of inventing busywork.</div>';

    const projectSelect = document.getElementById('command-project');
    projectSelect.innerHTML = '<option value="0">Inbox / no project</option>' + projects.map(p => `<option value="${p.id}">${esc(p.name)}</option>`).join('');
    const focusSelect = document.getElementById('focus-task');
    focusSelect.innerHTML = '<option value="">Choose a task to focus on</option>' + ranked.map(t => `<option value="${t.id}">${esc(t.title)}</option>`).join('');

    renderCommandRisks({ open, overdue, doing, blocked, projects });
    renderCommandGoals(goals, projects);
    renderCommandReview(tasks, weekRaw || [], statsRaw || {});
    renderFocusLog();
}

function renderCommandRisks({ open, overdue, doing, blocked, projects }) {
    const orphaned = open.filter(t => !t.project_id).length;
    const staleProjects = projects.filter(p => p.total > 0 && p.progress === 0).length;
    const risks = [
        ['Deadline debt', overdue.length, overdue.length ? 'Reschedule, delegate, or finish overdue work.' : 'Deadlines are healthy.'],
        ['Work in progress', doing.length, doing.length > 5 ? 'Finish before starting. Keep active work below five.' : 'Your active workload is focused.'],
        ['Unaligned tasks', orphaned, orphaned ? 'Move inbox tasks into projects or deliberately keep them standalone.' : 'Every task has context.'],
        ['Stalled projects', staleProjects, staleProjects ? 'Projects with no progress need a next action.' : 'All projects show movement.'],
        ['Dependencies', blocked.length, blocked.length ? 'Tasks labeled “blocked” need an unblock action.' : 'No explicit blockers detected.'],
    ];
    document.getElementById('command-risks').innerHTML = risks.map(([name,count,tip]) => `
        <div class="risk-row"><span class="risk-dot ${count ? 'hot' : 'clear'}"></span><div><strong>${esc(name)}</strong><small>${esc(tip)}</small></div><b>${count}</b></div>`).join('');
}

function renderCommandGoals(goals, projects) {
    const active = goals.filter(g => g.status !== 'done').sort((a,b) => (a.target_date || '9999').localeCompare(b.target_date || '9999'));
    document.getElementById('command-goals').innerHTML = active.length ? active.slice(0,5).map(g => {
        const linked = projects.filter(p => (g.project_ids || []).includes(p.id));
        return `<div class="alignment-row"><div><strong>${esc(g.name)}</strong><small>${linked.length ? linked.map(p => esc(p.name)).join(' · ') : 'No linked project — add an execution path'}</small></div><div class="alignment-progress"><span style="width:${g.progress || 0}%"></span></div><b>${g.progress || 0}%</b></div>`;
    }).join('') : '<div class="command-empty">Create a goal to connect daily execution with long-term direction.</div>';
}

function renderCommandReview(tasks, week, stats) {
    const completed = tasks.filter(t => t.done && t.completed_at && daysFromToday(t.completed_at.slice(0,10)) >= -6).length;
    const created = (week || []).reduce((n,d) => n + d.count, 0);
    const highOpen = tasks.filter(t => !t.done && t.priority === 'high').length;
    const prompts = [
        ['Shipped', completed, completed ? 'Completed tasks in the last seven days.' : 'Nothing shipped yet—make the next action smaller.'],
        ['Momentum', created, `${stats.current_streak || 0}-day current streak.`],
        ['Strategic debt', highOpen, highOpen > 3 ? 'Too many high priorities means priority has lost meaning.' : 'High-priority load is believable.'],
        ['Reflection', '→', 'What should you stop, start, and continue next week?'],
    ];
    document.getElementById('command-review').innerHTML = prompts.map(([l,v,s]) => `<div class="review-card"><span>${l}</span><strong>${v}</strong><p>${esc(s)}</p></div>`).join('');
}

function parseCapture(raw) {
    let priority = /!high\b/i.test(raw) ? 'high' : /!low\b/i.test(raw) ? 'low' : 'medium';
    let due = '';
    if (/\btoday\b/i.test(raw)) due = todayStr();
    if (/\btomorrow\b/i.test(raw)) { const d = new Date(); d.setDate(d.getDate() + 1); due = fmtDate(d); }
    const labels = [...raw.matchAll(/#([\w-]+)/g)].map(m => m[1]).join(',');
    const title = raw.replace(/!(high|low|medium)\b/gi,'').replace(/#([\w-]+)/g,'').replace(/\b(today|tomorrow)\b/gi,'').replace(/\s+/g,' ').trim();
    return { title, priority, labels, due_date: due };
}

async function commandCapture() {
    const input = document.getElementById('command-capture');
    const parsed = parseCapture(input.value.trim());
    if (!parsed.title) return notify('Give the task a clear action', true);
    parsed.project_id = Number(document.getElementById('command-project').value || 0);
    const r = await api('POST', '/tasks', parsed);
    if (r && r.success) { input.value = ''; notify('Captured and added to your system'); loadCommand(); }
}

function focusThis(id) {
    document.getElementById('focus-task').value = String(id);
    document.querySelector('.focus-panel').scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function focusKey() { return 'discipline-focus-' + todayStr(); }
function renderFocusLog() { document.getElementById('focus-log').textContent = `${Number(localStorage.getItem(focusKey()) || 0)} focused minutes recorded today.`; }
function renderFocusClock() {
    document.getElementById('focus-clock').textContent = `${String(Math.floor(focusSeconds/60)).padStart(2,'0')}:${String(focusSeconds%60).padStart(2,'0')}`;
}
function setFocusMinutes(m) { if (focusTimer) return; focusSessionMinutes = m; focusSeconds = m * 60; renderFocusClock(); }
function resetFocus() { clearInterval(focusTimer); focusTimer = null; focusSessionMinutes = 25; focusSeconds = 25 * 60; renderFocusClock(); document.getElementById('focus-start').textContent = 'Start focus'; document.getElementById('focus-state').textContent = 'Ready'; }
function toggleFocus() {
    if (focusTimer) { clearInterval(focusTimer); focusTimer = null; document.getElementById('focus-start').textContent = 'Resume'; document.getElementById('focus-state').textContent = 'Paused'; return; }
    if (!document.getElementById('focus-task').value) return notify('Choose one task for this focus session', true);
    document.getElementById('focus-start').textContent = 'Pause'; document.getElementById('focus-state').textContent = 'Focusing';
    focusTimer = setInterval(() => {
        focusSeconds--; renderFocusClock();
        if (focusSeconds <= 0) {
            clearInterval(focusTimer); focusTimer = null;
            const minutes = Number(localStorage.getItem(focusKey()) || 0) + focusSessionMinutes;
            localStorage.setItem(focusKey(), minutes); renderFocusLog();
            document.getElementById('focus-start').textContent = 'Start another'; document.getElementById('focus-state').textContent = 'Complete';
            notify('Focus session complete — protect the momentum');
        }
    }, 1000);
}

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

function renderViewKpis(id, items) {
    const el = document.getElementById(id);
    if (!el) return;
    el.innerHTML = items.map(([label, value, note, tone]) => `<div class="view-kpi ${tone || ''}"><span>${esc(label)}</span><strong>${value}</strong><small>${esc(note)}</small></div>`).join('');
}
