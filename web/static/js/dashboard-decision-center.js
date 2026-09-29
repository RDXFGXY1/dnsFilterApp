(function () {
    const snapshotKey = 'dns-filter-protection-snapshots-v1';
    let timelineEvents = [];
    let lockdownTimer = null;

    const html = value => String(value ?? '').replace(/[&<>"']/g, ch => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
    const fmtDate = value => {
        if (!value) return 'Never';
        const date = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
        return Number.isNaN(date.getTime()) ? String(value) : date.toLocaleString();
    };
    const setText = (id, value) => { const el = document.getElementById(id); if (el) el.textContent = value; };

    window.openDomainExplanation = function (domain) {
        switchView('decisions');
        inspectDecisionDomain(domain);
        document.querySelector('.decision-inspector')?.scrollIntoView({behavior:'smooth', block:'start'});
    };

    window.inspectDecisionDomain = async function (raw) {
        const input = document.getElementById('dc-domain');
        if (raw) input.value = raw;
        const domain = input.value.trim();
        const out = document.getElementById('dc-inspection');
        if (!domain) return;
        out.innerHTML = '<span class="no-data">Evaluating every active decision layer…</span>';
        const r = await api('GET', '/decision/inspect?domain=' + encodeURIComponent(domain));
        if (!r || r.error) {
            out.innerHTML = `<div class="decision-error">${html(r?.error || 'Inspection failed')}</div>`;
            return;
        }
        const aiScore = Math.round(Math.max(0, Math.min(100, Number(r.ai?.confidence) || 0)));
        const conflicts = r.conflicts || [];
        const blockRules = r.block_matches || [];
        const allowRules = r.allow_matches || [];
        out.innerHTML = `
            <div class="decision-verdict ${r.blocked ? 'blocked' : 'allowed'}">
                <div><span>Current decision</span><strong>${r.blocked ? 'BLOCKED' : 'ALLOWED'}</strong></div>
                <div><span>Winning layer</span><strong>${html(r.layer)}</strong></div>
                <div><span>Exact reason</span><strong>${html(r.reason || 'No blocking policy matched')}</strong></div>
                <div><span>Checked</span><strong>${fmtDate(r.checked_at)}</strong></div>
            </div>
            <div class="website-profile">
                <div class="profile-title"><div><span>Website intelligence profile</span><h3>${html(r.domain)}</h3></div><div class="profile-risk ${html(r.risk || 'low')}"><strong>${html((r.risk || 'low').toUpperCase())}</strong><span>Risk level · ${aiScore}% AI</span></div></div>
                <div class="profile-grid">
                    <div><span>AI category</span><strong>${html(r.ai?.category || 'unknown')}</strong></div>
                    <div><span>Blocked history</span><strong>${Number(r.blocked_visits || 0).toLocaleString()} events</strong></div>
                    <div><span>Last blocked</span><strong>${fmtDate(r.last_blocked)}</strong></div>
                    <div><span>Conflicts</span><strong>${conflicts.length}</strong></div>
                </div>
                <p class="profile-explanation">${html(r.ai?.reason || 'The local model found no notable domain-pattern signal.')}</p>
                <p class="profile-recommendation"><strong>Recommendation</strong>${html(r.recommendation || 'Continue monitoring this domain if it is unfamiliar.')}</p>
                <div class="rule-evidence">
                    <div><span>Matching block rules</span>${blockRules.length ? blockRules.map(x=>`<b>${html(x)}</b>`).join('') : '<em>None</em>'}</div>
                    <div><span>Matching exceptions</span>${allowRules.length ? allowRules.map(x=>`<b>${html(x)}</b>`).join('') : '<em>None</em>'}</div>
                </div>
                ${conflicts.length ? `<div class="inline-conflicts">${conflicts.map(x=>`<p>⚠ ${html(x)}</p>`).join('')}</div>` : ''}
                <div class="investigation-actions">
                    <button class="btn btn-ghost" onclick="inspectDecisionDomain()">Retest now</button>
                    <button class="btn btn-danger" data-domain="${html(r.domain)}" onclick="prepareDecisionRule('block',this.dataset.domain)">Prepare block</button>
                    <button class="btn" data-domain="${html(r.domain)}" onclick="prepareDecisionRule('allow',this.dataset.domain)">Prepare exception</button>
                </div>
            </div>`;
    };

    window.prepareDecisionRule = function (action, domain) {
        document.getElementById('dc-rule-pattern').value = domain;
        document.getElementById('dc-rule-action').value = action;
        document.getElementById('dc-rule-samples').value = domain + '\nwww.' + domain;
        document.querySelector('.rule-simulator')?.scrollIntoView({behavior:'smooth', block:'start'});
    };

    window.simulateDecisionRule = async function () {
        const pattern = document.getElementById('dc-rule-pattern').value.trim();
        const action = document.getElementById('dc-rule-action').value;
        const samples = document.getElementById('dc-rule-samples').value.split(/[\n,]+/).map(x=>x.trim()).filter(Boolean);
        const out = document.getElementById('dc-simulation');
        if (!pattern || !samples.length) { showNotification('Add a rule and at least one sample website', true); return; }
        out.innerHTML = '<span class="no-data">Simulating…</span>';
        const r = await api('POST', '/decision/simulate', {pattern, action, samples});
        if (!r || r.error) { out.innerHTML = `<div class="decision-error">${html(r?.error || 'Simulation failed')}</div>`; return; }
        out.innerHTML = `<div class="simulation-summary"><strong>${r.changed}</strong><span>of ${r.results.length} decisions would change</span></div><div class="simulation-results">${r.results.map(item => `<div class="simulation-row ${item.matches ? 'matched' : ''}"><span>${html(item.domain)}</span><b>${item.before_blocked ? 'Blocked' : 'Allowed'} → ${item.after_blocked ? 'Blocked' : 'Allowed'}</b><small>${item.matches ? 'Rule matches' : 'Outside pattern'}</small></div>`).join('')}</div>`;
    };

    window.loadDecisionConflicts = async function () {
        const r = await api('GET', '/decision/conflicts');
        if (!r) return;
        setText('dc-conflict-count', r.count || 0);
        const chain = document.getElementById('dc-precedence');
        chain.innerHTML = (r.precedence || []).map((x,i)=>`<span><b>${i+1}</b>${html(x)}</span>`).join('');
        const out = document.getElementById('dc-conflicts');
        out.innerHTML = r.items?.length ? r.items.map(item=>`<article><div><span>Overlapping policies</span><strong>${html(item.blocked)}</strong><small>conflicts with exception ${html(item.allowed)}</small></div><b>Winner: ${html(item.winner)}</b></article>`).join('') : '<div class="decision-empty compact"><strong>No direct conflicts detected</strong><span>Blocklist and whitelist rules do not currently overlap.</span></div>';
    };

    window.loadProtectionHealth = async function () {
        const r = await api('GET', '/protection/health');
        if (!r) return;
        setText('dc-health-score', (r.score || 0) + '/100');
        setText('dc-engine-state', r.enabled ? 'Active' : 'Disabled');
        const route = (r.checks || []).find(x=>x.id==='route');
        setText('dc-route-state', route?.ok ? 'Local DNS' : 'Review');
        document.getElementById('dc-health-checks').innerHTML = (r.checks || []).map(check=>`<article class="${check.ok ? 'healthy' : 'unhealthy'}"><i>${check.ok ? '✓' : '!'}</i><div><strong>${html(check.label)}</strong><span>${html(check.detail)}</span></div></article>`).join('');
    };

    window.repairProtection = async function () {
        const out = document.getElementById('dc-repair-result');
        out.textContent = 'Repairing live protection state…';
        const r = await api('POST', '/protection/repair', {});
        if (!r) return;
        const steps = (r.steps || []).join(' · ');
        out.className = 'operation-note ' + (r.success ? 'success' : 'warning');
        out.textContent = r.success ? `Repair completed: ${steps}` : `Partial repair: ${steps}. ${r.hint || r.error || ''}`;
        showNotification(r.success ? 'Protection repaired' : 'Protection needs administrator permission', !r.success);
        loadProtectionHealth();
    };

    window.loadEmergencyLockdown = async function () {
        const r = await api('GET', '/protection/lockdown');
        if (!r || r.error) return;
        clearInterval(lockdownTimer);
        const state = document.getElementById('lockdown-state');
        const countdown = document.getElementById('lockdown-countdown');
        const start = document.getElementById('lockdown-start');
        const stop = document.getElementById('lockdown-stop');
        state.textContent = r.active ? 'LOCKDOWN ACTIVE' : 'Ready';
        state.classList.toggle('active', !!r.active);
        start.hidden = !!r.active;
        stop.hidden = !r.active;
        if (!r.active) { countdown.textContent = ''; return; }
        document.getElementById('lockdown-allowlist').value = (r.allowed_domains || []).join('\n');
        const tick = () => {
            const left = Math.max(0, Number(r.until || 0) - Math.floor(Date.now()/1000));
            const h = Math.floor(left/3600), m = Math.floor((left%3600)/60), s = left%60;
            countdown.textContent = `Ends in ${h ? h + 'h ' : ''}${m}m ${String(s).padStart(2,'0')}s`;
            if (left <= 0) { clearInterval(lockdownTimer); loadEmergencyLockdown(); }
        };
        tick(); lockdownTimer = setInterval(tick, 1000);
    };

    window.startEmergencyLockdown = async function () {
        const minutes = Number(document.getElementById('lockdown-minutes').value || 60);
        const allowed_domains = document.getElementById('lockdown-allowlist').value.split(/[\n,]+/).map(x=>x.trim()).filter(Boolean);
        const warning = allowed_domains.length
            ? `Block every domain except ${allowed_domains.length} allowed domain(s) for ${minutes} minutes?`
            : `Block ALL DNS access for ${minutes} minutes?`;
        if (!window.confirm(warning)) return;
        const r = await api('POST', '/protection/lockdown/start', {minutes, allowed_domains});
        if (!r || r.error) { showNotification(r?.error || 'Could not start lockdown', true); return; }
        showNotification('Emergency lockdown started');
        loadEmergencyLockdown();
    };

    window.stopEmergencyLockdown = async function () {
        if (!window.confirm('End emergency lockdown and restore normal DNS decisions?')) return;
        const r = await api('POST', '/protection/lockdown/stop', {});
        if (!r || r.error) { showNotification(r?.error || 'Could not stop lockdown', true); return; }
        showNotification('Emergency lockdown ended');
        loadEmergencyLockdown();
    };

    window.loadDecisionTimeline = async function () {
        timelineEvents = await api('GET', '/recent') || [];
        renderTimeline(timelineEvents);
    };
    function renderTimeline(events) {
        const root = document.getElementById('dc-timeline');
        if (!events.length) { root.innerHTML = '<div class="decision-empty compact"><strong>No blocked activity yet</strong><span>Future website decisions will appear here.</span></div>'; return; }
        root.innerHTML = events.map(item=>`<button class="timeline-event" data-domain="${html(item.domain)}" data-client="${html(item.client_ip || '')}" data-time="${item.timestamp || 0}" onclick="openDomainExplanation(this.dataset.domain)"><i></i><div><strong>${html(item.domain)}</strong><span>${html(item.client_ip || 'unknown device')}</span></div><time>${fmtDate(item.timestamp)}</time><b>Explain</b></button>`).join('');
    }
    window.filterDecisionTimeline = function () {
        const q = (document.getElementById('dc-history-search').value || '').toLowerCase();
        const win = document.getElementById('dc-history-window').value;
        const now = Date.now()/1000;
        const visible = timelineEvents.filter(x => {
            const text = `${x.domain} ${x.client_ip || ''}`.toLowerCase();
            const cutoff = win === 'hour' ? 3600 : win === 'day' ? 86400 : Infinity;
            return text.includes(q) && (cutoff === Infinity || now - Number(x.timestamp || 0) <= cutoff);
        });
        renderTimeline(visible);
    };

    function getSnapshots() { try { return JSON.parse(localStorage.getItem(snapshotKey) || '[]'); } catch { return []; } }
    function renderSnapshots() {
        const root = document.getElementById('dc-snapshots');
        const items = getSnapshots();
        root.innerHTML = items.length ? items.map((x,i)=>`<article><div><strong>${html(x.name)}</strong><span>${fmtDate(x.created_at)}</span></div><small>${x.whitelist.length} exceptions · ${x.blocklist.length} direct blocks</small><button class="btn-remove" onclick="deleteProtectionSnapshot(${i})">DEL</button></article>`).join('') : '<div class="decision-empty compact"><strong>No snapshots yet</strong><span>Create one before changing several policies.</span></div>';
    }
    window.createProtectionSnapshot = async function () {
        const [whitelist, blocklist, modules, settings] = await Promise.all([api('GET','/whitelist'), api('GET','/custom-blocklist'), api('GET','/filtering/modules'), api('GET','/system/info')]);
        const items = getSnapshots();
        items.unshift({name:'Protection checkpoint ' + new Date().toLocaleString(), created_at:new Date().toISOString(), whitelist:whitelist||[], blocklist:blocklist||[], modules:modules||{}, settings:settings||{}});
        localStorage.setItem(snapshotKey, JSON.stringify(items.slice(0,10)));
        renderSnapshots(); showNotification('Configuration snapshot created');
    };
    window.deleteProtectionSnapshot = function (index) { const items=getSnapshots(); items.splice(index,1); localStorage.setItem(snapshotKey,JSON.stringify(items)); renderSnapshots(); };
    window.downloadLatestSnapshot = function () {
        const item = getSnapshots()[0]; if (!item) { showNotification('Create a snapshot first', true); return; }
        const url = URL.createObjectURL(new Blob([JSON.stringify(item,null,2)],{type:'application/json'}));
        const a=document.createElement('a'); a.href=url; a.download='dns-filter-snapshot.json'; a.click(); URL.revokeObjectURL(url);
    };

    window.loadDecisionCenter = function () { loadProtectionHealth(); loadEmergencyLockdown(); loadDecisionConflicts(); loadDecisionTimeline(); renderSnapshots(); };
    window.dnsBypassFindings = function(r) {
        const route=(r.checks||[]).find(x=>x.id==='route');
        const known=!!route&&!r.route_error&&!!r.dns;
        return [
            {state:!known?'unknown':route.ok?'checked':'warning', title:'Selected adapter DNS', detail:!known?'Could not verify the selected adapter. Retry with an active network connection.':`${r.interface||'Selected adapter'}: ${r.dns}`, help:known&&!route.ok?'External DNS is configured. Use System DNS controls to route this adapter through the filter, then scan again.':'Local DNS configuration does not prove that browsers or other adapters use it.'},
            {state:r.enabled===true?'checked':r.enabled===false?'warning':'unknown',title:'Filtering engine',detail:r.enabled===true?'Filtering is enabled.':r.enabled===false?'Filtering is disabled or paused.':'Engine status unavailable.',help:'Enable filtering in Settings if you want DNS rules to apply.'},
            {state:'unknown',title:'Browser secure DNS / DoH',detail:'Not automatically verified.',help:'Check Secure DNS / DNS over HTTPS in your browser settings. A custom resolver can bypass the local filter. After changing it, restart the browser and inspect a fresh lookup in Live Monitor.'},
            {state:'unknown',title:'Other adapters, VPNs and applications',detail:'Not covered by this selected-adapter scan.',help:'VPNs, other active adapters and apps with built-in resolvers may use another DNS path. Cached pages, existing connections and direct IP connections are not proof of DNS bypass.'},
            {state:'unknown',title:'Traffic evidence',detail:`${Number(r.queries)||0} queries observed since server startup.`,help:'This is historical evidence only, not a browser-specific test or a guarantee that every request is filtered.'}
        ];
    };
    window.scanDNSBypass = async function(){
        const button=document.getElementById('dns-bypass-scan'),out=document.getElementById('dns-bypass-results');
        if(!button||button.disabled)return;
        button.disabled=true;out.textContent='Checking selected adapter and filtering state…';
        try{
            const r=await api('GET','/protection/health');
            if(!r||r.error)throw new Error(r?.error||'Scan unavailable. Sign in again or retry.');
            const findings=window.dnsBypassFindings(r);
            const warning=findings.some(x=>x.state==='warning');
            out.innerHTML=`<p class="operation-note">${warning?'Potential bypass or disabled protection detected.':'No confirmed bypass in the available checks.'} Browser and alternate routes remain unverified. Checked ${html(new Date().toLocaleTimeString())}.</p>`+findings.map(x=>`<article class="dns-bypass-finding" data-state="${x.state}"><strong>${html(x.title)} <span>${x.state==='checked'?'Checked':x.state==='warning'?'Needs attention':'Not verified'}</span></strong><p>${html(x.detail)}</p><small>${html(x.help)}</small></article>`).join('');
        }catch(e){out.textContent=e.message||'Scan failed. Retry; no network settings were changed.'}
        finally{button.disabled=false}
    };
    function mountDNSBypass(){
        const health=document.querySelector('.decision-health');
        if(!health||document.getElementById('dns-bypass-results'))return;
        const panel=document.createElement('section');panel.className='panel dns-bypass-panel';
        panel.innerHTML='<div class="panel-head"><div><span class="command-kicker">Read-only diagnostics</span><h2>DNS bypass assistant</h2></div><button class="btn" id="dns-bypass-scan" onclick="scanDNSBypass()">Run DNS check</button></div><p class="panel-desc">Find routing risks without changing your network settings. This is not a complete DNS leak test.</p><div id="dns-bypass-results" aria-live="polite">Run a check to inspect the current DNS route and see what still needs manual verification.</div>';
        health.insertAdjacentElement('afterend',panel);
    }
    function mount() {
        mountDNSBypass();
        document.getElementById('dc-domain')?.addEventListener('keydown', e=>{if(e.key==='Enter')inspectDecisionDomain()});
        renderSnapshots();
        if (location.hash === '#decisions') loadDecisionCenter();
    }
    if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', mount); else mount();
})();
