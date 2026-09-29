// ── Project workspace ────────────────────────────────────────────────────────
let currentProjectData = null;
async function openProject(id) {
    currentProject = id;
    switchView('workspace');
    const [projects, goals] = await Promise.all([api('GET', '/projects'), api('GET','/goals')]);
    window._workspaceGoals = goals || [];
    const p = (projects || []).find(x => x.id === id) || { name: 'Project', progress: 0 };
    currentProjectData = p;
    renderProjectCommand(p);
    if (typeof restoreSavedView === 'function') restoreSavedView();
    loadWsTasks();
}
function projectStatusLabel(status) {
    return ({ planning:'Planning', active:'Active', paused:'Paused', at_risk:'At risk', completed:'Completed' })[status] || 'Active';
}
function renderProjectCommand(p, tasks) {
    if (!p) return;
    currentProjectData = p;
    const status = p.status || 'active';
    document.getElementById('ws-name').textContent = p.name;
    document.getElementById('ws-objective').textContent = p.objective || p.description || 'Define what success looks like for this project.';
    const statusEl = document.getElementById('ws-status');
    statusEl.textContent = projectStatusLabel(status);
    statusEl.className = 'project-status status-' + status;
    document.getElementById('ws-progress').style.width = (p.progress || 0) + '%';
    document.getElementById('ws-progress-label').textContent = (p.progress || 0) + '%';
    document.getElementById('ws-timeline-signal').textContent = p.target_date ? `${p.start_date || 'Unscheduled'} → ${p.target_date}` : 'Not scheduled';
    const goal = (window._workspaceGoals || []).find(g => (g.project_ids || []).includes(p.id));
    document.getElementById('ws-goal').textContent = goal ? goal.name : 'Not linked';
    if (tasks) updateProjectHealth(tasks);
}
function updateProjectHealth(tasks) {
    const open = tasks.filter(t => !t.done);
    const overdue = open.filter(t => daysFromToday(t.due_date) < 0).length;
    const blocked = open.filter(t => (t.labels || '').toLowerCase().split(',').map(x=>x.trim()).includes('blocked')).length;
    const unscheduled = open.filter(t => !t.due_date).length;
    const lateProject = currentProjectData?.target_date && daysFromToday(currentProjectData.target_date) < 0 && (currentProjectData.progress || 0) < 100;
    const health = Math.max(0, 100 - overdue*12 - blocked*15 - Math.min(20,unscheduled*2) - (lateProject?25:0));
    document.getElementById('ws-health').textContent = health;
    document.getElementById('ws-open-work').textContent = `${open.length} task${open.length===1?'':'s'}`;
}
function toggleProjectSetup(force) {
    const panel = document.getElementById('project-setup');
    const show = typeof force === 'boolean' ? force : panel.style.display === 'none';
    panel.style.display = show ? 'block' : 'none';
    if (show && currentProjectData) {
        document.getElementById('ws-edit-name').value = currentProjectData.name || '';
        document.getElementById('ws-edit-status').value = currentProjectData.status || 'active';
        document.getElementById('ws-edit-objective').value = currentProjectData.objective || '';
        document.getElementById('ws-edit-description').value = currentProjectData.description || '';
        document.getElementById('ws-edit-notes').value = currentProjectData.notes || '';
        document.getElementById('ws-edit-links').value = currentProjectData.links || '';
        document.getElementById('ws-edit-start').value = currentProjectData.start_date || '';
        document.getElementById('ws-edit-target').value = currentProjectData.target_date || '';
        panel.scrollIntoView({behavior:'smooth',block:'start'});
    }
}
async function saveProjectSetup() {
    const start = document.getElementById('ws-edit-start').value;
    const target = document.getElementById('ws-edit-target').value;
    const warning = document.getElementById('ws-date-warning');
    if (start && target && target < start) { warning.textContent = 'Target date must be after the start date.'; return; }
    warning.textContent = '';
    const payload = { name:document.getElementById('ws-edit-name').value.trim(), status:document.getElementById('ws-edit-status').value, objective:document.getElementById('ws-edit-objective').value.trim(), description:document.getElementById('ws-edit-description').value.trim(), notes:document.getElementById('ws-edit-notes').value.trim(), links:document.getElementById('ws-edit-links').value.trim(), start_date:start, target_date:target };
    if (!payload.name) { warning.textContent = 'Project name is required.'; return; }
    const r = await api('PATCH','/projects/'+currentProject,payload);
    if (!r || !r.success) { warning.textContent = r?.error || 'Could not save project.'; return; }
    notify('Project definition updated'); toggleProjectSetup(false); await refreshWs();
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
    const days = Number(document.getElementById('analytics-window')?.value || 30);
    const [stats, tasksRaw, perday, projects] = await Promise.all([
        api('GET', '/stats'),
        api('GET', '/tasks?all=1'),
        api('GET', '/series?type=task_completed&days=' + days),
        api('GET', '/projects'),
    ]);
    const s = stats || {};
    const tasks = tasksRaw || [];
    const open = tasks.filter(t => !t.done), overdue = open.filter(t => daysFromToday(t.due_date) < 0);
    const done = tasks.filter(t => t.done).length;
    const rate = tasks.length ? Math.round(done*100/tasks.length) : 0;
    const insight = overdue.length ? `${overdue.length} overdue task${overdue.length===1?' is':'s are'} suppressing flow. Resolve deadline debt before increasing throughput.` : rate < 40 && tasks.length > 5 ? `Completion is ${rate}%. Reduce active work and finish smaller batches.` : `Completion is ${rate}% with no overdue debt. Your strongest lever is consistent delivery.`;
    document.getElementById('analytics-insight').innerHTML = `<span>System insight</span><strong>${esc(insight)}</strong>`;
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
    updateProjectHealth(tasks);
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
        ${taskStatusControl(t)}
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

function taskStatusControl(t) {
    const meta = STATUS_META[t.status] || STATUS_META.todo;
    return `<div class="task-status-control" data-status-task="${t.id}">
        <button class="task-status-button status-${esc(t.status)}" type="button" onclick="toggleTaskStatusMenu(event,${t.id})"><span>${meta.icon}</span>${esc(meta.label)}<b>▾</b></button>
        <div class="task-status-menu" role="menu">
            ${STATUS_ORDER.map(s => `<button type="button" class="${s===t.status?'active':''}" onclick="setTaskStatus(${t.id},'${s}')"><span>${STATUS_META[s].icon}</span><span><strong>${esc(STATUS_META[s].label)}</strong><small>${s==='todo'?'Ready to start':s==='in_progress'?'Currently being worked on':'Finished work'}</small></span></button>`).join('')}
        </div>
    </div>`;
}
function toggleTaskStatusMenu(event,id){event.stopPropagation();const root=document.querySelector(`[data-status-task="${id}"]`),was=root.classList.contains('open');document.querySelectorAll('.task-status-control.open').forEach(x=>x.classList.remove('open'));if(!was)root.classList.add('open')}
async function setTaskStatus(id,status){document.querySelectorAll('.task-status-control.open').forEach(x=>x.classList.remove('open'));await patchField(id,'status',status)}
document.addEventListener('click',()=>document.querySelectorAll('.task-status-control.open').forEach(x=>x.classList.remove('open')));

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
            ${taskStatusControl(t)}
            <button class="tw-addsub" onclick="openTaskDrawer(${t.id})">details</button>
            <button class="hierarchy-action" onclick="revealSubAdd(this, ${t.id})">+ Subtask</button>
            <button class="hierarchy-action" onclick="openChecklistComposer(${t.id})">+ Checklist</button>
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
                        ${taskStatusControl(k)}
                        <button class="tw-addsub" onclick="openTaskDrawer(${k.id})">details</button>
                        <button class="hierarchy-action" onclick="openChecklistComposer(${k.id})">+ Checklist</button>
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
        return `<div class="tcard ${hasKids ? 'has-kids' : ''} p-${esc(t.priority || 'medium')}" draggable="true" ondragstart="planDrag(event,${t.id})" ondragover="event.preventDefault()" ondrop="planDrop(event,${t.id})">${head}${body}</div>`;
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
    if (p) renderProjectCommand(p);
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
    const [projects, goals] = await Promise.all([api('GET', '/projects'), api('GET','/goals')]);
    window._workspaceGoals = goals || [];
    const p = (projects || []).find(x => x.id === currentProject);
    if (p) renderProjectCommand(p);
    loadWsTasks();
}

async function openChecklistComposer(id) {
    await openTaskDrawer(id);
    const input = document.getElementById('td-new-check');
    if (input) { input.scrollIntoView({block:'center'}); input.focus(); }
}
