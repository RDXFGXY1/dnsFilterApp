// ── Goals ────────────────────────────────────────────────────────────────────
let _allProjects = [];
async function loadGoals() {
    const [goals, projects] = await Promise.all([api('GET', '/goals'), api('GET', '/projects')]);
    _allProjects = projects || [];
    const all = goals || [], activeGoals = all.filter(g => g.status !== 'done');
    const linked = activeGoals.filter(g => (g.project_ids || []).length).length;
    renderViewKpis('goal-kpis', [
        ['Active outcomes', activeGoals.length, activeGoals.length > 5 ? 'consider narrowing focus' : 'portfolio size', activeGoals.length > 5 ? 'warn' : ''],
        ['Average progress', activeGoals.length ? Math.round(activeGoals.reduce((n,g)=>n+g.progress,0)/activeGoals.length)+'%' : '0%', 'across active goals'],
        ['Execution linked', linked, `${activeGoals.length ? Math.round(linked*100/activeGoals.length) : 0}% have projects`, linked < activeGoals.length ? 'warn' : 'good'],
        ['Completed', all.filter(g=>g.status==='done').length, 'outcomes achieved', 'good'],
    ]);
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

let _projectPortfolio = [];
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
    _projectPortfolio = data;
    const stalled = projects.filter(p => p.total > 0 && p.progress === 0).length;
    const complete = projects.filter(p => p.total > 0 && p.progress === 100).length;
    const doing = projects.reduce((n,p)=>n+(p.in_progress||0),0);
    renderViewKpis('project-kpis', [['Portfolio',projects.length,'total projects'],['Active work',doing,'tasks in progress',doing>7?'warn':''],['Stalled',stalled,'projects without movement',stalled?'danger':'good'],['Completed',complete,'fully delivered','good']]);
    renderProjectPortfolio();
}
function renderProjectPortfolio() {
    const grid = document.getElementById('projects-grid');
    const filter = document.getElementById('project-filter')?.value || 'all';
    const data = _projectPortfolio.filter(({p}) => filter === 'all' || (filter === 'active' && p.progress < 100 && (p.progress > 0 || p.in_progress > 0)) || (filter === 'stalled' && p.total > 0 && p.progress === 0) || (filter === 'complete' && p.total > 0 && p.progress === 100));
    grid.innerHTML = data.length ? data.map(({ p, stats, hm }) => {
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
    }).join('') : '<span class="no-data">No projects match this portfolio filter.</span>';
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
