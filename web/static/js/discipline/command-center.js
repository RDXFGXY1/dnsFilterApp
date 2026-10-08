// ── Command Center: planning, triage, focus, and review ───────────────────────
let commandCache = { tasks: [], projects: [], goals: [], stats: {} };
let focusSeconds = 25 * 60;
let focusSessionMinutes = 25;
let focusTimer = null;
let focusSessionID = 0;
let focusActivityTimer = null;
let focusRestoreAttempted = false;

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
    renderFocusHierarchy();

    renderCommandRisks({ open, overdue, doing, blocked, projects });
    renderCommandGoals(goals, projects);
    renderCommandReview(tasks, weekRaw || [], statsRaw || {});
    renderFocusLog();
    loadFocusHistory();
    restoreFocusSession();
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

function focusEligibleTasks() {
    return commandCache.tasks.filter(t => !t.done && !t.archived && !t.backlog);
}

function renderFocusHierarchy(preferredTaskID = 0) {
    const goalSelect = document.getElementById('focus-goal');
    const projectSelect = document.getElementById('focus-project');
    const taskSelect = document.getElementById('focus-task');
    if (!goalSelect || !projectSelect || !taskSelect) return;

    const previousGoal = goalSelect.value;
    const previousProject = projectSelect.value;
    const activeGoals = commandCache.goals.filter(g => g.status !== 'done' && g.status !== 'paused');
    goalSelect.innerHTML = '<option value="">Choose a goal</option>' + activeGoals.map(g => `<option value="${g.id}">${esc(g.name)}</option>`).join('');
    if ([...goalSelect.options].some(o => o.value === previousGoal)) goalSelect.value = previousGoal;
    renderFocusProjects(previousProject, preferredTaskID);
}

function renderFocusProjects(preferredProject = '', preferredTaskID = 0) {
    const goalID = Number(document.getElementById('focus-goal').value || 0);
    const projectSelect = document.getElementById('focus-project');
    let projects = commandCache.projects.filter(p => p.status !== 'done' && p.status !== 'archived');
    if (goalID) {
        const goal = commandCache.goals.find(g => g.id === goalID);
        const linked = new Set((goal?.project_ids || []).map(Number));
        projects = projects.filter(p => linked.has(Number(p.id)));
    }
    projectSelect.innerHTML = `<option value="">${goalID ? 'Focus directly on this goal' : 'Choose a project'}</option>` + projects.map(p => `<option value="${p.id}">${esc(p.name)}</option>`).join('');
    if ([...projectSelect.options].some(o => o.value === String(preferredProject))) projectSelect.value = String(preferredProject);
    renderFocusTasks(preferredTaskID);
}

function renderFocusTasks(preferredTaskID = 0) {
    const projectID = Number(document.getElementById('focus-project').value || 0);
    const taskSelect = document.getElementById('focus-task');
    const note = document.getElementById('focus-selection-note');
    if (!projectID) {
        taskSelect.disabled = true;
        const goalID = Number(document.getElementById('focus-goal').value || 0);
        const goal = commandCache.goals.find(g => Number(g.id) === goalID);
        taskSelect.innerHTML = `<option value="">${goal ? 'Focus directly on this goal' : 'Choose a goal or project first'}</option>`;
        note.textContent = goal ? `Ready to focus on goal · ${goal.name}` : 'Choose a goal, or choose a project and a specific task.';
        return;
    }

    const tasks = focusEligibleTasks().filter(t => Number(t.project_id) === projectID);
    const byParent = new Map();
    tasks.forEach(t => {
        const parent = Number(t.parent_id || 0);
        if (!byParent.has(parent)) byParent.set(parent, []);
        byParent.get(parent).push(t);
    });
    const sortTasks = list => list.sort((a,b) => actionScore(b) - actionScore(a) || a.title.localeCompare(b.title));
    const options = [];
    const append = (task, depth, seen) => {
        if (seen.has(task.id)) return;
        seen.add(task.id);
        const prefix = depth ? `${'\u00a0\u00a0'.repeat(depth)}↳ ` : '';
        options.push(`<option value="${task.id}" data-title="${esc(task.title)}">${prefix}${esc(task.title)}</option>`);
        sortTasks(byParent.get(Number(task.id)) || []).forEach(child => append(child, depth + 1, seen));
    };
    const seen = new Set();
    sortTasks(byParent.get(0) || []).forEach(t => append(t, 0, seen));
    sortTasks(tasks.filter(t => !seen.has(t.id))).forEach(t => append(t, t.parent_id ? 1 : 0, seen));

    taskSelect.disabled = options.length === 0;
    taskSelect.innerHTML = `<option value="">${options.length ? 'Choose a task or sub-task' : 'No unfinished work in this project'}</option>${options.join('')}`;
    if (preferredTaskID && [...taskSelect.options].some(o => o.value === String(preferredTaskID))) taskSelect.value = String(preferredTaskID);
    const project = commandCache.projects.find(p => Number(p.id) === projectID);
    note.textContent = options.length ? `${project?.name || 'Project'} · ${options.length} available focus item${options.length === 1 ? '' : 's'}.` : 'This project has no unfinished tasks or sub-tasks.';
}

function focusGoalChanged() { renderFocusProjects(); }
function focusProjectChanged() { renderFocusTasks(); }

function focusThis(id) {
    const task = commandCache.tasks.find(t => Number(t.id) === Number(id));
    if (!task || !task.project_id) return notify('Move this task into a project before starting focused work', true);
    const goal = commandCache.goals.find(g => (g.project_ids || []).map(Number).includes(Number(task.project_id)) && g.status !== 'done' && g.status !== 'paused');
    document.getElementById('focus-goal').value = goal ? String(goal.id) : '';
    renderFocusProjects(task.project_id, task.id);
    document.querySelector('.focus-panel').scrollIntoView({ behavior: 'smooth', block: 'center' });
}

function focusKey() { return 'discipline-focus-' + todayStr(); }
function focusSecondsKey() { return focusKey() + '-seconds-v2'; }
function storedFocusSeconds() {
    const precise = localStorage.getItem(focusSecondsKey());
    if (precise !== null) return Math.max(0, Number(precise) || 0);
    const migrated = Math.max(0, Number(localStorage.getItem(focusKey()) || 0) * 60);
    localStorage.setItem(focusSecondsKey(), String(migrated));
    return migrated;
}
function addStoredFocusSeconds(seconds) {
    localStorage.setItem(focusSecondsKey(), String(storedFocusSeconds() + Math.max(0, Number(seconds) || 0)));
}
function focusStateKey() { return 'discipline-focus-active-v2'; }
function readFocusState() { try { return JSON.parse(localStorage.getItem(focusStateKey()) || 'null'); } catch { return null; } }
function saveFocusState(running) {
    if (!focusSessionID) return;
    const previous = readFocusState() || {};
    localStorage.setItem(focusStateKey(), JSON.stringify({
        sessionID:focusSessionID, minutes:focusSessionMinutes, seconds:focusSeconds,
        running:Boolean(running), updatedAt:Date.now(), activityLogging:Boolean(previous.activityLogging),
    }));
}
function clearFocusState() { localStorage.removeItem(focusStateKey()); }
function renderFocusLog() {
    const completedSeconds = storedFocusSeconds();
    const liveSeconds = focusSessionID ? Math.max(0, focusSessionMinutes * 60 - focusSeconds) : 0;
    const total = completedSeconds + liveSeconds;
    const minutes = Math.floor(total / 60), seconds = total % 60;
    document.getElementById('focus-log').textContent = `${minutes}m ${String(seconds).padStart(2,'0')}s focused today.`;
}
function setFocusRunningUI(active, activityLogging = false) {
    document.querySelector('.focus-panel')?.classList.toggle('is-running', active);
    document.getElementById('focus-setup').hidden = active;
    document.getElementById('focus-reset').hidden = active;
    document.getElementById('focus-restart').hidden = !active;
    document.getElementById('focus-end').hidden = !active;
    document.getElementById('focus-live-activity').hidden = !active;
    document.getElementById('focus-distraction-tools').hidden = !active;
    if (active) {
        const state = readFocusState() || {};
        state.activityLogging = Boolean(activityLogging);
        localStorage.setItem(focusStateKey(), JSON.stringify(state));
        startFocusActivityPreview(activityLogging);
    } else {
        clearInterval(focusActivityTimer); focusActivityTimer = null;
    }
}
async function refreshFocusActivityPreview(activityLogging) {
    const status = document.getElementById('focus-live-status');
    if (!activityLogging) {
        status.textContent = 'Logging is off for this session.';
        document.getElementById('focus-live-apps').textContent = '0';
        document.getElementById('focus-live-domains').textContent = '0';
        document.getElementById('focus-live-observations').textContent = '0';
        return;
    }
    const session = await api('GET', `/focus-sessions/${focusSessionID}`);
    if (!session || session.error) { status.textContent = 'Waiting for the local recorder…'; return; }
    const activity = session.activity || [], apps = activity.filter(x => x.kind === 'app'), domains = activity.filter(x => x.kind === 'domain');
    const observations = activity.reduce((n,x) => n + Number(x.observations || 0), 0);
    document.getElementById('focus-live-apps').textContent = apps.length;
    document.getElementById('focus-live-domains').textContent = domains.length;
    document.getElementById('focus-live-observations').textContent = observations;
    const latest = activity[0]?.name;
    status.textContent = latest ? `Latest · ${latest}` : 'Recording locally — first sample can take 5 seconds.';
}
function startFocusActivityPreview(activityLogging) {
    clearInterval(focusActivityTimer);
    refreshFocusActivityPreview(activityLogging);
    if (activityLogging) focusActivityTimer = setInterval(() => refreshFocusActivityPreview(true), 5000);
}
function focusDuration(seconds) {
    seconds = Math.max(0, Number(seconds || 0));
    const hours = Math.floor(seconds / 3600), minutes = Math.floor((seconds % 3600) / 60), secs = seconds % 60;
    if (hours) return `${hours}h ${minutes}m`;
    if (minutes) return `${minutes}m ${secs}s`;
    return `${secs}s`;
}
function focusWhen(value) {
    if (!value) return 'Unknown time';
    const normalized = value.includes('T') ? value : value.replace(' ', 'T') + 'Z';
    const date = new Date(normalized);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString([], {month:'short',day:'numeric',hour:'2-digit',minute:'2-digit'});
}
async function loadFocusHistory(keepSelection = false) {
    const list = document.getElementById('focus-session-list');
    if (!list) return;
    const selected = keepSelection ? Number(list.querySelector('.focus-session-item.active')?.dataset.id || 0) : 0;
    const sessions = await api('GET', '/focus-sessions');
    if (!Array.isArray(sessions)) {
        list.innerHTML = '<div class="focus-history-empty"><strong>Could not load focus history</strong><p>Try refreshing the logs.</p></div>';
        return;
    }
    if (!sessions.length) {
        list.innerHTML = '<div class="focus-history-empty"><strong>No focus sessions yet</strong><p>Enable Local activity log and start a session to build your private timeline.</p></div>';
        return;
    }
    list.innerHTML = sessions.map(s => {
        const activityCount = (s.activity || []).reduce((n,a) => n + Number(a.observations || 0), 0);
        return `<button class="focus-session-item ${selected === Number(s.id) ? 'active' : ''}" data-id="${s.id}" onclick="openFocusSession(${s.id},this)">
            <span class="focus-session-status ${esc(s.status || 'active')}">${esc(s.status || 'active')}</span>
            <strong>${esc(s.task_title || 'Focused work')}</strong>
            <small>${focusWhen(s.started_at)} · ${focusDuration(s.actual_seconds)} · ${activityCount} observations</small>
        </button>`;
    }).join('');
    if (selected && sessions.some(s => Number(s.id) === selected)) openFocusSession(selected, list.querySelector(`[data-id="${selected}"]`));
}
async function openFocusSession(id, button) {
    document.querySelectorAll('.focus-session-item').forEach(x => x.classList.toggle('active', x === button));
    const detail = document.getElementById('focus-session-detail');
    detail.innerHTML = '<span class="no-data">Loading session details…</span>';
    const s = await api('GET', `/focus-sessions/${id}`);
    if (!s || s.error) {
        detail.innerHTML = '<div class="focus-history-empty"><strong>Session unavailable</strong><p>The local record could not be loaded.</p></div>';
        return;
    }
    const activity = s.activity || [], apps = activity.filter(a => a.kind === 'app'), domains = activity.filter(a => a.kind === 'domain'), distractions = s.distractions || [];
    const activityRows = items => items.length ? items.map(a => `<div class="focus-activity-row"><div><strong>${esc(a.name)}</strong><small>${a.observations} observation${Number(a.observations) === 1 ? '' : 's'}</small></div><b>${focusDuration(a.seconds)}</b></div>`).join('') : '<p class="focus-history-muted">No records captured.</p>';
    detail.innerHTML = `<header class="focus-detail-head"><div><span class="command-kicker">Session ${s.id}</span><h3>${esc(s.task_title || 'Focused work')}</h3><p>${focusWhen(s.started_at)} · ${esc(s.status || 'active')}</p></div><div class="focus-detail-time"><strong>${focusDuration(s.actual_seconds)}</strong><span>${s.planned_minutes || 0}m planned</span></div></header>
        ${s.activity_logging ? '' : '<div class="focus-history-notice">Local activity logging was disabled for this session.</div>'}
        <div class="focus-detail-kpis"><span><strong>${apps.length}</strong> apps</span><span><strong>${domains.length}</strong> domains</span><span><strong>${activity.reduce((n,a)=>n+Number(a.observations||0),0)}</strong> observations</span><span><strong>${distractions.length}</strong> distractions</span></div>
        <div class="focus-activity-columns"><section><h4>Applications</h4>${activityRows(apps)}</section><section><h4>DNS-visible domains</h4>${activityRows(domains)}</section></div>
        <section class="focus-distraction-log"><h4>Distractions</h4>${distractions.length ? distractions.map(d => `<span>${esc(d.label)}${d.note ? ` · ${esc(d.note)}` : ''}</span>`).join('') : '<p class="focus-history-muted">No distractions logged.</p>'}</section>
        ${s.reflection ? `<section class="focus-reflection"><h4>Reflection</h4><p>${esc(s.reflection)}</p>${s.focus_score ? `<b>Focus score ${s.focus_score}/10</b>` : ''}</section>` : ''}`;
}
function renderFocusClock() {
    document.getElementById('focus-clock').textContent = `${String(Math.floor(focusSeconds/60)).padStart(2,'0')}:${String(focusSeconds%60).padStart(2,'0')}`;
}
function setFocusMinutes(m) { if (focusTimer) return; focusSessionMinutes = m; focusSeconds = m * 60; renderFocusClock(); }
function startFocusCountdown() {
    clearInterval(focusTimer);
    focusTimer = setInterval(async () => {
        focusSeconds = Math.max(0, focusSeconds - 1);
        renderFocusClock(); renderFocusLog(); saveFocusState(true);
        if (focusSeconds <= 0) {
            clearInterval(focusTimer); focusTimer = null;
            addStoredFocusSeconds(focusSessionMinutes * 60);
            await finishFocusSession('completed', true);
            focusSessionID = 0; clearFocusState(); setFocusRunningUI(false); renderFocusLog();
            document.getElementById('focus-start').textContent = 'Start another';
            document.getElementById('focus-state').textContent = 'Complete';
            notify('Focus session complete — journal and activity summary saved');
        }
    }, 1000);
}
function restartFocus() {
    if (!focusSessionID) return;
    focusSeconds = focusSessionMinutes * 60;
    renderFocusClock(); renderFocusLog(); saveFocusState(Boolean(focusTimer));
    notify('Focus timer restarted');
}
async function endFocusSession() {
    if (!focusSessionID) return;
    const activityLogging = Boolean(readFocusState()?.activityLogging);
    clearInterval(focusTimer); focusTimer = null;
    const elapsedSeconds = Math.max(0, focusSessionMinutes * 60 - focusSeconds);
    const result = await finishFocusSession('completed', true);
    if (result?.error) {
        notify(result.error || 'Could not end the focus session', true);
        saveFocusState(false); setFocusRunningUI(true, activityLogging);
        document.getElementById('focus-start').textContent = 'Resume';
        document.getElementById('focus-state').textContent = 'Paused';
        return;
    }
    if (elapsedSeconds > 0) addStoredFocusSeconds(elapsedSeconds);
    focusSessionID = 0; clearFocusState(); setFocusRunningUI(false);
    focusSeconds = focusSessionMinutes * 60;
    renderFocusClock(); renderFocusLog();
    document.getElementById('focus-start').textContent = 'Start another';
    document.getElementById('focus-state').textContent = 'Complete';
    notify('Focus session ended — time, reflection, and activity saved');
}
async function resetFocus() {
    clearInterval(focusTimer); focusTimer = null;
    if(focusSessionID){await finishFocusSession('cancelled',false);focusSessionID=0}
    clearFocusState(); setFocusRunningUI(false); focusSessionMinutes = 25; focusSeconds = 25 * 60;
    renderFocusClock(); renderFocusLog(); document.getElementById('focus-start').textContent = 'Start focus'; document.getElementById('focus-state').textContent = 'Ready';
}
async function toggleFocus() {
    if (focusTimer) {
        clearInterval(focusTimer); focusTimer = null; saveFocusState(false);
        document.getElementById('focus-start').textContent = 'Resume'; document.getElementById('focus-state').textContent = 'Paused';
        return;
    }
    if (focusSessionID) {
        document.getElementById('focus-start').textContent = 'Pause'; document.getElementById('focus-state').textContent = 'Focusing';
        saveFocusState(true); startFocusCountdown(); return;
    }
    const goalID=Number(document.getElementById('focus-goal').value||0),projectID=Number(document.getElementById('focus-project').value||0),taskID=Number(document.getElementById('focus-task').value||0);
    if (!goalID && !projectID && !taskID) return notify('Choose a goal, project, or task for this focus session', true);
    const taskOption=document.getElementById('focus-task').selectedOptions[0],projectOption=document.getElementById('focus-project').selectedOptions[0],goalOption=document.getElementById('focus-goal').selectedOptions[0];
    const focusTitle=taskID?(taskOption?.dataset.title||taskOption?.textContent.trim()):projectID?`Project · ${projectOption?.textContent.trim()}`:`Goal · ${goalOption?.textContent.trim()}`;
    const activityLogging=document.getElementById('focus-activity-log').checked;
    const started=await api('POST','/focus-sessions/start',{task_id:taskID,task_title:focusTitle||'Focused work',minutes:focusSessionMinutes,activity_logging:activityLogging});
    if(!started?.success)return notify(started?.error||'Could not start the focus session',true);
    focusSessionID=started.id;
    localStorage.setItem(focusStateKey(),JSON.stringify({sessionID:focusSessionID,minutes:focusSessionMinutes,seconds:focusSeconds,running:true,updatedAt:Date.now(),activityLogging}));
    setFocusRunningUI(true, activityLogging); loadFocusHistory();
    document.getElementById('focus-start').textContent = 'Pause'; document.getElementById('focus-state').textContent = 'Focusing';
    startFocusCountdown(); renderFocusLog();
}

async function restoreFocusSession() {
    if (focusRestoreAttempted) return;
    focusRestoreAttempted = true;
    const response = await api('GET', '/focus-sessions/active');
    if (!response?.active || !response.session) { clearFocusState(); return; }
    const session = response.session, saved = readFocusState();
    focusSessionID = Number(session.id); focusSessionMinutes = Number(session.planned_minutes || 25);
    let running = true, activityLogging = Boolean(session.activity_logging);
    if (saved && Number(saved.sessionID) === focusSessionID) {
        focusSeconds = Math.max(1, Number(saved.seconds || focusSessionMinutes * 60));
        running = Boolean(saved.running); activityLogging = Boolean(saved.activityLogging);
        if (running && saved.updatedAt) focusSeconds = Math.max(1, focusSeconds - Math.floor((Date.now() - Number(saved.updatedAt)) / 1000));
    } else {
        const started = new Date((session.started_at || '').replace(' ', 'T') + 'Z').getTime();
        const elapsed = Number.isNaN(started) ? 0 : Math.max(0, Math.floor((Date.now() - started) / 1000));
        focusSeconds = Math.max(1, focusSessionMinutes * 60 - elapsed);
    }
    localStorage.setItem(focusStateKey(),JSON.stringify({sessionID:focusSessionID,minutes:focusSessionMinutes,seconds:focusSeconds,running,updatedAt:Date.now(),activityLogging}));
    setFocusRunningUI(true, activityLogging); renderFocusClock(); renderFocusLog();
    document.getElementById('focus-start').textContent = running ? 'Pause' : 'Resume';
    document.getElementById('focus-state').textContent = running ? 'Focusing' : 'Paused';
    if (running) startFocusCountdown();
}

async function finishFocusSession(status,createJournal){
    if(!focusSessionID)return;
    let reflection='',score=0;
    if(status==='completed'){
        reflection=prompt('Focus reflection (optional): what did you finish or learn?','')||'';
        const raw=prompt('How focused were you from 1 to 10?','8');score=Math.max(0,Math.min(10,Number(raw)||0));
    }
    const result=await api('POST',`/focus-sessions/${focusSessionID}/end`,{status,reflection,focus_score:score,create_journal:createJournal});
    if (!result?.error) { clearFocusState(); loadFocusHistory(); }
    return result;
}

async function logFocusDistraction(label){if(!focusSessionID)return notify('Start a focus session first',true);const r=await api('POST',`/focus-sessions/${focusSessionID}/distraction`,{label,note:''});if(r?.success)notify(label+' distraction logged');else notify(r?.error||'Could not log distraction',true)}
function logCustomFocusDistraction(){const label=prompt('What distracted you?','');if(label)logFocusDistraction(label)}

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
