(function(){
const el=id=>document.getElementById(id);
const esc=value=>String(value??'').replace(/[&<>"']/g,ch=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
const mark=(id,state,label)=>{const card=el(id);if(!card)return;card.classList.remove('healthy','warning','danger');card.classList.add(state);card.querySelector('.security-state').textContent=label};
const tags=(id,items)=>{el(id).innerHTML=items.map(x=>`<span>${esc(x)}</span>`).join('')};
let lastData=null;

window.loadSecurityCenter=window.refreshSecurityCenter=async function(){
 const alerts=el('security-alerts');alerts.innerHTML='<div class="security-loading">Checking all security layers…</div>';
 const [info,health,lockdown,packs,vault]=await Promise.all([
  api('GET','/system/info'),api('GET','/protection/health'),api('GET','/protection/lockdown'),api('GET','/security-packs'),api('GET','/vault/status')
 ]);
 if(!info||!health||!lockdown||!packs||!vault){alerts.innerHTML='<div class="security-alert danger"><strong>Security check could not be completed</strong><span>One or more local services did not respond.</span></div>';return}
 lastData={info,health,lockdown,packs,vault};render(lastData);
};

function render({info,health,lockdown,packs,vault}){
 const upstream=Array.isArray(info.upstream_dns)?info.upstream_dns:[];
 const encrypted=!!info.secure_dns&&upstream.length>0&&upstream.every(x=>String(x).startsWith('https://')||String(x).startsWith('tls://'));
 mark('security-feature-dns',encrypted?'healthy':'danger',encrypted?'Protected':'Action required');
 el('security-dns-state').textContent=encrypted?'Encrypted':'Not strict';
 el('security-dns-detail').textContent=encrypted?'Every configured upstream uses encrypted transport and plaintext fallback is disabled.':'Enable strict secure DNS and remove ordinary UDP/TCP upstream resolvers.';
 tags('security-dns-meta',upstream.length?upstream.map(x=>String(x).startsWith('https://')?'DoH':String(x).startsWith('tls://')?'DoT':'Plain DNS'):['No upstream configured']);

 const healthOK=Number(health.score||0)>=80;
 mark('security-feature-decisions',healthOK?'healthy':'warning',healthOK?'Ready':'Review');
 el('security-decision-detail').textContent=healthOK?'Decision explanations and protection health checks are available.':`Protection health is ${Number(health.score||0)}/100. Review failed checks in Decision Center.`;

 mark('security-feature-lockdown',lockdown.active?'danger':'healthy',lockdown.active?'Active':'Standby');
 el('security-lockdown-detail').textContent=lockdown.active?'Emergency Lockdown is currently blocking every domain outside its allowlist.':'Emergency controls are ready and normal DNS policy is active.';
 tags('security-lockdown-meta',lockdown.active?[`${(lockdown.allowed_domains||[]).length} exceptions`,'Automatic expiry enabled']:['Ready when needed','1 minute–24 hours']);

 const packCount=(packs.packs||[]).length,keyCount=Number(packs.trusted_key_count||0);
 mark('security-feature-packs',packCount?'healthy':'warning',packCount?`${packCount} installed`:'Not installed');
 el('security-packs-detail').textContent=packCount?'Installed packs passed signature verification and are active in filtering.':'No signed threat-intelligence pack is installed yet.';
 tags('security-packs-meta',[`${packCount} packs`,`${keyCount} trusted keys`]);

 const vaultReady=!!vault.configured,vaultOpen=!!vault.unlocked;
 mark('security-feature-vault',vaultReady?'healthy':'warning',vaultReady?(vaultOpen?'Unlocked':'Locked'):'Not enabled');
 el('security-vault-detail').textContent=vaultReady?(vaultOpen?'Configuration changes are currently unlocked until the automatic timeout.':'Configuration changes are protected; DNS continues to run while the vault is locked.'):'Enable the vault to protect configuration changes and encrypted backups.';
 tags('security-vault-meta',[vaultReady?`${Number(vault.audit_entries||0)} audit events`:'Recovery code not created',vaultOpen?'Changes allowed':'Changes protected']);

 const checks=[
  {ok:encrypted,title:'Encrypt every upstream DNS request',detail:encrypted?'Strict DoH/DoT mode is active.':'Secure DNS requires attention.',action:'securityOpenSettings()'},
  {ok:healthOK,title:'Keep protection health above 80',detail:`Current health score: ${Number(health.score||0)}/100.`,action:"switchView('decisions')"},
  {ok:packCount>0,title:'Install at least one signed security pack',detail:packCount?`${packCount} verified pack(s) installed.`:'No signed pack is active.',action:"switchView('blocklist')"},
  {ok:vaultReady,title:'Protect configuration changes with the vault',detail:vaultReady?'Vault protection is configured.':'The configuration vault is not enabled.',action:'securityOpenVault()'}
 ];
 el('security-checklist').innerHTML=checks.map(x=>`<article class="security-check ${x.ok?'done':'todo'}"><i>${x.ok?'✓':'!'}</i><div><strong>${esc(x.title)}</strong><span>${esc(x.detail)}</span></div><button class="btn btn-ghost" onclick="${x.action}">${x.ok?'Review':'Fix'}</button></article>`).join('');
 const passed=checks.filter(x=>x.ok).length,score=Math.round(passed/checks.length*100),scoreWrap=el('security-score-wrap');
 el('security-score').textContent=score;el('security-summary').textContent=`${passed} of ${checks.length} recommended protections ready`;scoreWrap.className='security-score '+(score>=75?'good':score>=50?'review':'risk');
 const problems=checks.filter(x=>!x.ok);el('security-alerts').innerHTML=lockdown.active?'<div class="security-alert danger"><strong>Emergency Lockdown is active</strong><span>Only explicitly allowed domains can resolve.</span></div>':problems.length?problems.map(x=>`<div class="security-alert warning"><strong>${esc(x.title)}</strong><span>${esc(x.detail)}</span></div>`).join(''):'<div class="security-alert success"><strong>All recommended security features are ready</strong><span>No action is currently required.</span></div>';
 const nav=el('security-nav-alert');nav.hidden=!problems.length&&!lockdown.active;
 el('security-last-checked').textContent='Checked '+new Date().toLocaleTimeString();
}

window.securityInspectDomain=function(){const domain=el('security-domain')?.value.trim();switchView('decisions');if(domain&&window.inspectDecisionDomain)setTimeout(()=>inspectDecisionDomain(domain),50)};
window.securityOpenLockdown=function(){switchView('decisions');setTimeout(()=>document.querySelector('.emergency-lockdown')?.scrollIntoView({behavior:'smooth',block:'center'}),80)};
window.securityOpenSettings=function(){switchView('settings');setTimeout(()=>document.getElementById('system-info')?.scrollIntoView({behavior:'smooth',block:'center'}),80)};
window.securityOpenVault=function(){switchView('settings');setTimeout(()=>{if(window.dashboardSettingsTab)dashboardSettingsTab('system',false);document.querySelector('.vault-panel')?.scrollIntoView({behavior:'smooth',block:'center'})},80)};
if(location.hash==='#security')document.addEventListener('DOMContentLoaded',loadSecurityCenter);
})();
