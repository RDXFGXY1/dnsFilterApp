// ── Discipline page: tasks, projects, event-driven activity ──────────────────

const VIEWS = {
    command:   ['Command Center', 'Decide, focus, execute, and review from one place'],
    today:     ['Today',     'Your daily missions'],
    tasks:     ['Tasks',     'Inbox, My Day, upcoming work, and reusable workflows'],
    home:      ['Home',      'Your tasks, projects, and momentum'],
    goals:     ['Goals',     'What you are ultimately working toward'],
    habits:    ['Habits',    'Things you do repeatedly — build the streak'],
    calendar:  ['Calendar',  'Your tasks and deadlines over time'],
    reminders: ['Reminders', "Don't forget the important stuff"],
    journal:   ['Daily Journal', 'Remember the work, lessons, and moments that shaped your day'],
    inbox:     ['Intelligence Inbox', 'Private briefings, reminders, and decisions from your discipline system'],
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
    const opts = { method, headers: { 'Content-Type': 'application/json' }, credentials: 'same-origin', cache: 'no-store' };
    if (body) opts.body = JSON.stringify(body);
    try {
        const r = await fetch('/api/discipline' + path, opts);
        if (r.status === 401) { window.location.href = '/login'; return null; }
        const data = await r.json().catch(() => ({ error: `Request failed with HTTP ${r.status}` }));
        return data;
    } catch (error) {
        return { error: error?.message || 'Local service is unavailable' };
    }
}

function switchView(name) {
    document.querySelectorAll('.view').forEach(v => v.classList.toggle('active', v.dataset.view === name));
    document.querySelectorAll('.sidenav a[data-view]').forEach(a => a.classList.toggle('active', a.dataset.view === name));
    const meta = VIEWS[name] || VIEWS.home;
    document.getElementById('page-title').textContent = meta[0];
    document.getElementById('page-sub').textContent = meta[1];
    window.scrollTo(0, 0);
    if (name === 'today') loadToday();
    if (name === 'tasks') loadTodoHub();
    if (name === 'command') loadCommand();
    if (name === 'home') loadHome();
    if (name === 'goals') loadGoals();
    if (name === 'habits') loadHabits();
    if (name === 'calendar') loadCalendar();
    if (name === 'reminders') loadReminders();
    if (name === 'journal') loadJournal();
    if (name === 'inbox') loadDisciplineInbox();
    if (name === 'analytics') loadGlobalAnalytics();
    if (name === 'projects') loadProjects();
    if (name === 'activity') loadActivity();
}
document.querySelectorAll('.sidenav a[data-view]').forEach(a =>
    a.addEventListener('click', e => { e.preventDefault(); switchView(a.dataset.view); }));

const PRIORITY = { high: '🔴', medium: '🟡', low: '🟢' };
const NEXT_STATUS = { todo: 'in_progress', in_progress: 'done', done: 'todo' };
const STATUS_LABEL = { todo: 'TODO', in_progress: 'DOING', done: 'DONE' };
