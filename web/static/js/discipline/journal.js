// ── Daily Journal ────────────────────────────────────────────────────────────
let journalTypes = [];
let journalEntries = [];
let journalSearchTimer = 0;
let journalPromptIndex = 0;
const JOURNAL_PROMPTS = [
    'What did you understand today that you did not understand yesterday?',
    'What moved forward today, even if only a little?',
    'What distracted you, and how will you protect your attention tomorrow?',
    'What was difficult, and what did it teach you?',
    'What is the single most useful next action?',
    'What are you grateful for today?',
];

function journalToday() { return new Date().toISOString().slice(0, 10); }
function journalMinutes(n) {
    n = Number(n || 0);
    if (n < 60) return n + 'm';
    const h = Math.floor(n / 60), m = n % 60;
    return h + 'h' + (m ? ' ' + m + 'm' : '');
}
function journalScore(n) { return Number(n || 0) ? Number(n).toFixed(1) : '—'; }
function journalDateLabel(day) {
    const d = new Date(day + 'T12:00:00');
    const today = journalToday();
    const yesterday = new Date(Date.now() - 86400000).toISOString().slice(0,10);
    if (day === today) return 'Today';
    if (day === yesterday) return 'Yesterday';
    return d.toLocaleDateString(undefined, {weekday:'long', month:'short', day:'numeric'});
}

async function loadJournal() {
    const day = document.getElementById('journal-day');
    if (day && !day.value) day.value = journalToday();
    await Promise.all([loadJournalTypes(), loadJournalStats(), loadJournalProjects()]);
    await loadJournalEntries();
}

async function loadJournalProjects() {
    const data=await api('GET','/projects'); if(!Array.isArray(data))return;
    document.getElementById('journal-project').innerHTML='<option value="0">No project</option>'+data.map(p=>`<option value="${p.id}">${esc(p.name)}</option>`).join('');
}

async function loadJournalTypes() {
    const data = await api('GET', '/journal/types');
    if (!Array.isArray(data)) return notify(data?.error || 'Could not load log types', true);
    journalTypes = data;
    const options = '<option value="0">General</option>' + data.map(t => `<option value="${t.id}">${esc(t.name)}</option>`).join('');
    document.getElementById('journal-type').innerHTML = options;
    document.getElementById('journal-filter-type').innerHTML = '<option value="0">All log types</option>' + data.map(t => `<option value="${t.id}">${esc(t.name)}</option>`).join('');
    const list = document.getElementById('journal-types-list');
    list.innerHTML = `<button class="journal-type-chip general" onclick="filterJournalType(0)"><i></i><span>General</span></button>` + data.map(t => `<button class="journal-type-chip" onclick="filterJournalType(${t.id})"><i style="background:${esc(t.color)}"></i><span>${esc(t.name)}</span><b>${t.entry_count}</b><em onclick="event.stopPropagation();deleteJournalType(${t.id})" title="Delete category">×</em></button>`).join('');
}

async function createJournalType() {
    const name = document.getElementById('journal-type-name').value.trim();
    const color = document.getElementById('journal-type-color').value;
    if (!name) return notify('Enter a name for the custom log', true);
    const data = await api('POST','/journal/types',{name,color});
    if (!data?.success) return notify(data?.error || 'Could not create log type', true);
    document.getElementById('journal-type-name').value = '';
    await loadJournalTypes();
    document.getElementById('journal-type').value = String(data.id);
    notify('Custom log created');
}

async function deleteJournalType(id) {
    if (!confirm('Delete this category? Its entries will remain under General.')) return;
    const data = await api('DELETE','/journal/types/'+id);
    if (!data?.success) return notify(data?.error || 'Could not delete log type', true);
    await Promise.all([loadJournalTypes(),loadJournalEntries()]);
    notify('Category removed; entries were preserved');
}

function journalFormData() {
    return {
        type_id:Number(document.getElementById('journal-type').value || 0),
        title:document.getElementById('journal-title').value.trim(),
        content:document.getElementById('journal-content').value.trim(),
        day:document.getElementById('journal-day').value,
        duration_minutes:Number(document.getElementById('journal-duration').value || 0),
        mood:Number(document.getElementById('journal-mood').value || 0),
        focus:Number(document.getElementById('journal-focus').value || 0),
        tags:document.getElementById('journal-tags').value.trim(),
        project_id:Number(document.getElementById('journal-project').value||0),
        pinned:document.getElementById('journal-pinned').checked,
    };
}

async function saveJournalEntry() {
    const id = Number(document.getElementById('journal-entry-id').value || 0);
    const body = journalFormData();
    if (!body.title) return notify('Give this journal entry a title', true);
    const button = document.getElementById('journal-save');
    button.disabled = true; button.textContent = id ? 'Updating…' : 'Saving…';
    try {
        const data = await api(id ? 'PUT' : 'POST', '/journal/entries' + (id ? '/'+id : ''), body);
        if (!data?.success) return notify(data?.error || 'Could not save the entry', true);
        resetJournalForm();
        await Promise.all([loadJournalTypes(),loadJournalStats(),loadJournalEntries()]);
        notify(id ? 'Journal entry updated' : 'Journal entry saved');
    } finally { button.disabled=false; button.textContent='Save entry'; }
}

function resetJournalForm() {
    document.getElementById('journal-entry-id').value='';
    document.getElementById('journal-title').value='';
    document.getElementById('journal-content').value='';
    document.getElementById('journal-day').value=journalToday();
    document.getElementById('journal-duration').value='';
    document.getElementById('journal-mood').value='0';
    document.getElementById('journal-focus').value='0';
    document.getElementById('journal-tags').value='';
    document.getElementById('journal-type').value='0';
    document.getElementById('journal-project').value='0';
    document.getElementById('journal-pinned').checked=false;
    document.getElementById('journal-form-heading').textContent='Write today’s entry';
    document.getElementById('journal-cancel-edit').hidden=true;
}

function editJournalEntry(id) {
    const e=journalEntries.find(x=>x.id===id); if(!e)return;
    document.getElementById('journal-entry-id').value=e.id;
    document.getElementById('journal-title').value=e.title;
    document.getElementById('journal-content').value=e.content;
    document.getElementById('journal-day').value=e.day;
    document.getElementById('journal-duration').value=e.duration_minutes||'';
    document.getElementById('journal-mood').value=String(e.mood||0);
    document.getElementById('journal-focus').value=String(e.focus||0);
    document.getElementById('journal-tags').value=e.tags;
    document.getElementById('journal-type').value=String(e.type_id||0);
    document.getElementById('journal-project').value=String(e.project_id||0);
    document.getElementById('journal-pinned').checked=!!e.pinned;
    document.getElementById('journal-form-heading').textContent='Edit journal entry';
    document.getElementById('journal-cancel-edit').hidden=false;
    document.querySelector('.journal-compose').scrollIntoView({behavior:'smooth',block:'start'});
}

async function deleteJournalEntry(id) {
    if(!confirm('Delete this journal entry?'))return;
    const data=await api('DELETE','/journal/entries/'+id);
    if(!data?.success)return notify(data?.error||'Could not delete entry',true);
    await Promise.all([loadJournalTypes(),loadJournalStats(),loadJournalEntries()]);notify('Entry deleted');
}

function journalEntryHTML(e) {
    const tags=String(e.tags||'').split(',').map(x=>x.trim()).filter(Boolean).map(x=>`<span>#${esc(x)}</span>`).join('');
    return `<article class="journal-entry ${e.pinned?'pinned':''}" style="--entry-color:${esc(e.type_color)}">
        <div class="journal-entry-rail"></div><div class="journal-entry-body">
            <header><div><span class="journal-entry-type"><i></i>${esc(e.type_name)}</span>${e.pinned?'<span class="journal-pinned-mark">PINNED</span>':''}<h3>${esc(e.title)}</h3></div><div class="journal-entry-actions"><button onclick="editJournalEntry(${e.id})">Edit</button><button onclick="deleteJournalEntry(${e.id})">Delete</button></div></header>
            ${e.content?`<p>${esc(e.content).replace(/\n/g,'<br>')}</p>`:''}${e.project_name?`<div class="journal-project-link">Project · ${esc(e.project_name)}</div>`:''}
            <footer><div class="journal-tags">${tags}</div><div class="journal-metrics">${e.duration_minutes?`<span>${journalMinutes(e.duration_minutes)}</span>`:''}${e.mood?`<span>Mood ${e.mood}/10</span>`:''}${e.focus?`<span>Focus ${e.focus}/10</span>`:''}</div></footer>
        </div></article>`;
}

async function loadJournalEntries() {
    const q=document.getElementById('journal-search').value.trim();
    const type=document.getElementById('journal-filter-type').value;
    const date=document.getElementById('journal-filter-date').value;
    const params=new URLSearchParams({limit:'300'}); if(q)params.set('q',q);if(type!=='0')params.set('type',type);if(date){params.set('from',date);params.set('to',date)}
    const data=await api('GET','/journal/entries?'+params);
    const timeline=document.getElementById('journal-timeline');
    if(!Array.isArray(data)){timeline.innerHTML='<div class="no-data">Could not load journal entries.</div>';return}
    journalEntries=data; document.getElementById('journal-result-count').textContent=data.length+' entr'+(data.length===1?'y':'ies');
    if(!data.length){timeline.innerHTML='<div class="journal-empty"><strong>Your journal is ready.</strong><p>Capture a lesson, a study session, or simply how your day went.</p></div>';return}
    const groups={};data.forEach(e=>(groups[e.day]??=[]).push(e));
    timeline.innerHTML=Object.entries(groups).map(([day,items])=>`<section class="journal-day-group"><div class="journal-day-label"><strong>${journalDateLabel(day)}</strong><span>${esc(day)} · ${items.length} entr${items.length===1?'y':'ies'}</span></div>${items.map(journalEntryHTML).join('')}</section>`).join('');
}

function queueJournalSearch(){clearTimeout(journalSearchTimer);journalSearchTimer=setTimeout(loadJournalEntries,250)}
function clearJournalFilters(){document.getElementById('journal-search').value='';document.getElementById('journal-filter-type').value='0';document.getElementById('journal-filter-date').value='';loadJournalEntries()}
function filterJournalType(id){document.getElementById('journal-filter-type').value=String(id);loadJournalEntries()}

async function loadJournalStats(){
    const s=await api('GET','/journal/stats');if(!s||s.error)return;
    document.getElementById('journal-streak').textContent=s.current_streak||0;
    document.getElementById('journal-kpis').innerHTML=`<article><span>Entries</span><strong>${s.total_entries}</strong><small>all time</small></article><article><span>Active days</span><strong>${s.active_days}</strong><small>longest streak ${s.longest_streak}</small></article><article><span>Logged time</span><strong>${journalMinutes(s.duration_minutes)}</strong><small>intentional activity</small></article><article><span>Focus</span><strong>${journalScore(s.average_focus)}</strong><small>mood ${journalScore(s.average_mood)} / 10</small></article>`;
    const cal=document.getElementById('journal-calendar');
    cal.innerHTML=(s.last_28_days||[]).map(d=>{const lvl=d.entries===0?0:d.entries===1?1:d.entries===2?2:d.entries<5?3:4;return `<button class="journal-calendar-day level-${lvl}" title="${esc(d.day)}: ${d.entries} entries, ${d.duration_minutes} minutes" onclick="document.getElementById('journal-filter-date').value='${esc(d.day)}';loadJournalEntries()"><span>${Number(d.day.slice(-2))}</span></button>`}).join('');
}

function nextJournalPrompt(){journalPromptIndex=(journalPromptIndex+1)%JOURNAL_PROMPTS.length;document.getElementById('journal-prompt').textContent=JOURNAL_PROMPTS[journalPromptIndex]}
