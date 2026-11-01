(function(){
const esc=v=>String(v??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let allEvents=[];
function mount(){
 VIEW_META.routewatch=['DNS Health','DNS route, upstream connectivity, and service history'];
 const nav=document.querySelector('.sidenav'),activity=nav?.querySelector('[data-view="activity"]'),link=document.createElement('a');
 link.href='#routewatch';link.dataset.view='routewatch';link.innerHTML='<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 12h4l2-5 4 10 2-5h6"/><circle cx="12" cy="12" r="10"/></svg>DNS Health';
 link.onclick=e=>{e.preventDefault();switchView('routewatch');load()};activity?.insertAdjacentElement('beforebegin',link);
 const view=document.createElement('section');view.className='view';view.dataset.view='routewatch';
 view.innerHTML='<section class="dashboard-command"><div><span class="command-kicker">PROTECTION HEALTH</span><h2>DNS Health Timeline</h2><p>See when DNS routing changes, an upstream stops answering, and service recovers.</p></div></section><section class="route-watch-current" id="route-watch-current"><div class="no-data">Loading DNS route…</div></section><section class="panel"><div class="panel-head"><div><h2>Health timeline</h2><p>Recent route and upstream changes are saved even while this page is closed.</p></div><div class="action-buttons"><select id="dns-health-filter" aria-label="Health event filter"><option value="all">All events</option><option value="issues">Issues only</option></select><button class="btn btn-ghost" id="dns-health-check">Check route now</button></div></div><div id="route-watch-events" class="route-event-list"><div class="no-data">Loading…</div></div></section>';
 document.querySelector('.main').appendChild(view);
 view.querySelector('#dns-health-filter').addEventListener('change',renderEvents);
 view.querySelector('#dns-health-check').addEventListener('click',check);
 if(location.hash==='#routewatch'){switchView('routewatch');load()}
}
function renderCurrent(current){
 const head=document.getElementById('route-watch-current');if(!head)return;
 const state=current.error?'unknown':current.protected?'protected':'warning';
 head.innerHTML=`<div class="route-watch-icon ${state}">${state==='protected'?'✓':state==='warning'?'!':'?'}</div><div><span>Current DNS route</span><h2>${state==='protected'?'Protected by local DNS Filter':state==='warning'?'External DNS detected':'Route could not be verified'}</h2><p>${esc(current.interface||'No active adapter')} · ${esc(current.dns||current.error||'No DNS server reported')}</p></div><div class="route-watch-time"><span>Last change</span><strong>${current.changed_at?new Date(current.changed_at).toLocaleString():'—'}</strong></div>`;
}
function renderEvents(){
 const list=document.getElementById('route-watch-events');if(!list)return;
 const onlyIssues=document.getElementById('dns-health-filter')?.value==='issues';
 const items=onlyIssues?allEvents.filter(e=>e.status!=='ok'):allEvents;
 list.innerHTML=items.length?items.map(e=>`<article class="route-event ${e.status==='ok'?'protected':e.status==='warning'?'warning':'unknown'}"><i></i><div><small class="route-event-kind">${esc(e.kind||'DNS')}</small><strong>${esc(e.title||'DNS event')}</strong><span>${esc(e.detail||'')}</span></div><time>${new Date(e.at).toLocaleString()}</time></article>`).join(''):'<div class="no-data">No events in this view.</div>';
}
async function load(){
 const [route,timeline]=await Promise.all([api('GET','/dns-route-watch'),api('GET','/dns-health/timeline')]);
 if(route?.error){document.getElementById('route-watch-current').innerHTML=`<div class="no-data">${esc(route.error)}</div>`}else renderCurrent(route?.current||{});
 if(timeline?.error){document.getElementById('route-watch-events').innerHTML=`<div class="no-data">${esc(timeline.error)}</div>`;return}
 allEvents=timeline?.events||[];renderEvents();
}
async function check(){const r=await api('POST','/dns-route-watch/check',{});if(r?.error)return showNotification(r.error,true);showNotification(r.current?.protected?'DNS route is protected':'DNS route needs attention');load()}
window.loadDNSRouteWatch=load;window.checkDNSRouteNow=check;
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount);else mount();
})();
