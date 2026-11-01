(()=>{
 'use strict';
 const state={apps:[],file:null,review:null,frame:null,port:null,active:null,revision:0,render:0};
 const requestedAppView=location.hash.slice(1);
 const $=id=>document.getElementById(id);
 const esc=value=>String(value??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
 async function request(path,options={}){
  const response=await fetch('/api/side-apps'+path,{credentials:'same-origin',cache:'no-store',...options});
  if(response.status===404)throw new Error('The running DNS Filter service does not include Side Apps. Restart it with the updated server build, then reload this page.');
  if(response.status===401||response.redirected)throw new Error('Your session has expired. Sign in again, then reopen Side Apps.');
  const result=await response.json().catch(()=>{throw new Error(`DNS Filter returned a non-JSON response (HTTP ${response.status}). Restart the updated local service and reload this page.`);});
  if(!response.ok)throw new Error(result.error||`Request failed (${response.status})`);return result;
 }
 function report(error){$('sideapps-status').textContent=error.message||String(error);}
 function stop(){state.render++;state.port?.close();state.port=null;state.frame?.remove();state.frame=null;state.active=null;$('sideapps-frame-slot').replaceChildren();}
 window.stopSideApp=stop;
 async function load(){
  try{state.apps=await request('');$('sideapps-status').textContent='';render();}catch(error){report(error);$('sideapps-nav-list').textContent='Could not load apps';}
 }
 window.loadSideApps=load;
 function render(){
  const nav=$('sideapps-nav-list');nav.replaceChildren();
  for(const app of state.apps.filter(app=>app.enabled)){
   const button=document.createElement('button');button.type='button';button.textContent=app.manifest.name;button.title=app.manifest.name;button.classList.toggle('active',state.active===app.manifest.id);button.addEventListener('click',()=>open(app.manifest.id));nav.append(button);
  }
  if(!nav.children.length){const text=document.createElement('small');text.textContent='No enabled apps installed';nav.append(text);}
  const list=$('sideapps-installed');list.replaceChildren();
  if(!state.apps.length){list.innerHTML='<div class="sideapps-empty"><h3>Your app shelf is ready</h3><p>Add an .adnf application to get started.</p><button class="btn" id="sideapps-empty-add">+ Add app</button></div>';$('sideapps-empty-add').addEventListener('click',add);return;}
  for(const app of state.apps){
   const m=app.manifest,card=document.createElement('article');card.className='sideapps-card';
   card.innerHTML=`<div class="sideapps-icon">${esc(m.name.slice(0,1))}</div><h3>${esc(m.name)}</h3><small>v${esc(m.version)} · ${esc(m.author||'Unknown author')} · ${app.enabled?'Enabled':'Disabled'}</small><p>${esc(m.description)}</p><small>${m.permissions.includes('storage')?'Own local storage':'No host permissions'}</small><div class="sideapps-card-actions"></div>`;
   const actions=card.querySelector('.sideapps-card-actions');
   const action=(label,fn,primary=false)=>{const b=document.createElement('button');b.type='button';b.className=primary?'btn':'btn btn-ghost';b.textContent=label;b.addEventListener('click',async()=>{b.disabled=true;try{await fn();}catch(e){report(e);}finally{b.disabled=false;}});actions.append(b);};
   if(app.enabled)action('Open app',()=>open(m.id),true);
   action(app.enabled?'Disable':'Enable',async()=>{await request('/'+m.id+'/enabled',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:!app.enabled})});if(state.active===m.id)stop();await load();});
   action('Uninstall',async()=>{if(!confirm(`Uninstall ${m.name} and delete its own local data?`))return;await request('/'+m.id,{method:'DELETE'});if(state.active===m.id)stop();await load();});
   list.append(card);
  }
 }
 function add(){state.revision++;state.file=null;state.review=null;$('sideapps-package-file').value='';$('sideapps-package-preview').innerHTML='<p>Choose an .adnf file to review it.</p>';$('sideapps-confirm-install').disabled=true;$('sideapps-replace').checked=false;$('sideapps-replace-label').hidden=true;$('sideapps-install-error').textContent='';$('sideapps-install-dialog').showModal();}
 $('sideapps-add').addEventListener('click',add);$('sideapps-nav-add').addEventListener('click',()=>{switchView('sideapps');add();});
 $('sideapps-nav-manage').addEventListener('click',()=>switchView('sideapps'));
 $('sideapps-cancel').addEventListener('click',()=>{$('sideapps-install-dialog').close();state.revision++;});
 $('sideapps-install-dialog').addEventListener('cancel',()=>{state.revision++;});
 $('sideapps-stop').addEventListener('click',()=>switchView('sideapps'));
 $('sideapps-package-file').addEventListener('change',async event=>{
  const revision=++state.revision;state.review=null;state.file=null;$('sideapps-confirm-install').disabled=true;$('sideapps-install-error').textContent='';$('sideapps-replace').checked=false;$('sideapps-replace-label').hidden=true;
  const file=event.target.files?.[0];if(!file)return;
  $('sideapps-package-preview').textContent='Checking package…';
  try{
   if(!file.name.toLowerCase().endsWith('.adnf')||file.size>5*1024*1024)throw new Error('Choose an .adnf package up to 5 MB');
   const form=new FormData();form.append('package',file);const preview=await request('/preview',{method:'POST',body:form});if(revision!==state.revision)return;
   const m=preview.manifest;state.file=file;state.review=preview;
   $('sideapps-package-preview').innerHTML=`<article class="sideapps-preview"><h3>${esc(m.name)}</h3><p>${esc(m.description)}</p><p>${esc(m.author||'Unknown author')} · v${esc(m.version)} · ${preview.file_count} files</p><p><strong>Requested permissions</strong></p><ul>${m.permissions.includes('storage')?'<li>Read and write only this app’s own local data (up to 1 MB).</li>':'<li>No host permissions.</li>'}</ul><small>Package ID: ${esc(m.id)}<br>SHA-256: ${esc(preview.sha256)}</small></article>`;
   $('sideapps-replace-label').hidden=!state.apps.some(a=>a.manifest.id===m.id);$('sideapps-confirm-install').disabled=false;
  }catch(error){if(revision!==state.revision)return;$('sideapps-package-preview').textContent=error.message;}
 });
 $('sideapps-confirm-install').addEventListener('click',async()=>{
  if(!state.file||!state.review)return;const exists=state.apps.some(a=>a.manifest.id===state.review.manifest.id);
  if(exists&&!$('sideapps-replace').checked){$('sideapps-install-error').textContent='Approve replacement before installing this version.';return;}
  const button=$('sideapps-confirm-install');button.disabled=true;
  try{const form=new FormData();form.append('package',state.file);form.append('sha256',state.review.sha256);form.append('replace',String(exists&&$('sideapps-replace').checked));await request('/install',{method:'POST',body:form});$('sideapps-install-dialog').close();state.review=null;state.file=null;switchView('sideapps');await load();$('sideapps-nav').open=true;}catch(error){$('sideapps-install-error').textContent=error.message;button.disabled=false;}
 });
 async function open(id){
  stop();switchView('sideapp');const generation=state.render;
  try{
   state.apps=await request('');if(generation!==state.render)return;
   const app=state.apps.find(a=>a.manifest.id===id&&a.enabled);if(!app)throw new Error('This app is unavailable or disabled');
   state.active=id;$('sideapps-running-name').textContent=app.manifest.name;$('sideapps-running-author').textContent=`v${app.manifest.version} · ${app.manifest.author||'Unknown author'} · Independent workspace`;
   const frame=document.createElement('iframe');frame.className='sideapps-frame';frame.title=app.manifest.name;frame.setAttribute('sandbox','allow-scripts');frame.setAttribute('referrerpolicy','no-referrer');frame.setAttribute('allow',"camera 'none'; microphone 'none'; geolocation 'none'; clipboard-read 'none'; clipboard-write 'none'");state.frame=frame;
   frame.src='/side-apps/run/'+encodeURIComponent(id);$('sideapps-frame-slot').replaceChildren(frame);location.hash='sideapp='+id;$('sideapps-nav').open=true;render();
  }catch(error){report(error);switchView('sideapps');}
 }
 window.addEventListener('message',event=>{
  if(event.source!==state.frame?.contentWindow||event.origin!=='null'||event.data?.type!=='adnf:ready'||state.port||!state.active)return;
  const channel=new MessageChannel(),frame=state.frame,appID=state.active;state.port=channel.port1;
  let busy=false;
  channel.port1.onmessage=async event=>{
   const msg=event.data;if(frame!==state.frame||appID!==state.active||!msg||!Number.isSafeInteger(msg.id))return;
   const reply=data=>{if(frame===state.frame)channel.port1.postMessage({id:msg.id,...data});};
   if(busy){reply({error:'Another storage request is in progress'});return;}
   if(!['storage.get','storage.set'].includes(msg.method)){reply({error:'Method is not available to side apps'});return;}
   busy=true;
   try{
    const path='/'+encodeURIComponent(appID)+'/storage';
    if(msg.method==='storage.get'){const result=await request(path);reply({value:result.value});}
    else{const body=JSON.stringify(msg.value);if(body===undefined||new TextEncoder().encode(body).length>1024*1024)throw new Error('App storage is limited to 1 MB');await request(path,{method:'PUT',headers:{'Content-Type':'application/json'},body});reply({value:true});}
   }catch(error){reply({error:error.message});}finally{busy=false;}
  };
  channel.port1.start();frame.contentWindow.postMessage({type:'adnf:connect'},'*',[channel.port2]);
 });
 document.addEventListener('visibilitychange',()=>{if(!document.hidden&&state.active){const active=state.active;request('').then(apps=>{if(state.active===active&&!apps.some(a=>a.manifest.id===active&&a.enabled)){stop();switchView('sideapps');}}).catch(()=>{stop();switchView('sideapps');});}});
 const initialize=()=>load().then(()=>{if(requestedAppView==='sideapps')switchView('sideapps');else if(requestedAppView.startsWith('sideapp='))open(requestedAppView.slice(8));});
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',initialize);else initialize();
})();
