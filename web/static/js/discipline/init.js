// ── Init ─────────────────────────────────────────────────────────────────────
document.getElementById('quick-task').addEventListener('keydown', e => { if (e.key === 'Enter') quickAddTask(); });
document.getElementById('ws-task-title').addEventListener('keydown', e => { if (e.key === 'Enter') addWsTask(); });
document.getElementById('today-add').addEventListener('keydown', e => { if (e.key === 'Enter') addTodayTask(); });
document.getElementById('command-capture').addEventListener('keydown', e => { if (e.key === 'Enter') commandCapture(); });
switchView('today');
loadAttention();
checkReminders();
setInterval(checkReminders, 60000);
