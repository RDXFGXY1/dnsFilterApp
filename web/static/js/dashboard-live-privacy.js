(function(){
function mount(){
 const panel=document.getElementById('live-privacy-panel');if(!panel)return;
 const retention=panel.querySelector('#live-retention'),allowed=panel.querySelector('#live-save-allowed'),hideIP=panel.querySelector('#live-hide-ip'),save=panel.querySelector('#live-privacy-save'),status=panel.querySelector('#live-privacy-status');
 async function load(){const data=await api('GET','/live/privacy');if(!data||data.error){status.textContent=data?.error||'Could not load privacy settings';return}retention.value=String(data.retention_hours||168);allowed.checked=!data.blocked_only;hideIP.checked=!!data.hide_client_ip;status.textContent='Settings loaded.'}
 async function submit(){save.disabled=true;status.textContent='Saving…';try{
  const data=await api('POST','/live/privacy',{retention_hours:Number(retention.value),hide_client_ip:hideIP.checked,blocked_only:!allowed.checked});
  if(data?.code==='vault_locked'&&window.requestVaultUnlock){status.textContent='Unlock the Vault to change Live history privacy.';requestVaultUnlock('Unlock the Configuration Vault to change Live history privacy.',submit);return}
  if(data?.error){status.textContent=data.error;return}
  status.textContent='Saved. Existing Live history was updated.';
  if(window.refreshLiveHistory)await window.refreshLiveHistory();
 }catch(error){status.textContent=error.message||'Could not save privacy settings'}finally{save.disabled=false}}
 save.addEventListener('click',submit);load();
}
if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',mount);else mount();
})();
