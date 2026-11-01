(()=>{
 'use strict';
 const $=id=>document.getElementById(id);let data={notes:'',sessions:[]};let minutes=25,remaining=25*60,deadline=0,running=false,startedAt=0,elapsed=0,saveTail=Promise.resolve(),connected=false;
 const esc=value=>String(value??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
 function render(){const seconds=running?Math.max(0,Math.ceil((deadline-Date.now())/1000)):remaining;$('timer').textContent=`${String(Math.floor(seconds/60)).padStart(2,'0')}:${String(seconds%60).padStart(2,'0')}`;$('timer-start').textContent=running?'Pause':startedAt?'Resume':'Start focus';$('timer-state').textContent=running?'Focusing':startedAt?'Paused':'Ready';$('timer-finish').disabled=!startedAt;document.querySelectorAll('[data-minutes]').forEach(b=>{b.disabled=Boolean(startedAt);b.classList.toggle('selected',Number(b.dataset.minutes)===minutes)});}
 function renderHistory(){const sessions=data.sessions.slice(-10).reverse();$('session-total').textContent=`${data.sessions.length} sessions`;$('history').innerHTML=sessions.length?sessions.map(s=>`<div class="history-row"><div><strong>${esc(s.objective)}</strong><small>${esc(new Date(s.finished_at).toLocaleString())}</small></div><span>${Math.round(s.seconds/60)} min</span></div>`).join(''):'<p class="muted">Your first focused step starts here.</p>';}
 function save(){if(!connected)return Promise.reject(new Error('App storage is not connected'));const snapshot=JSON.parse(JSON.stringify(data));$('save-state').textContent='Saving…';saveTail=saveTail.catch(()=>{}).then(()=>ADNSF.storage.set(snapshot));return saveTail.then(()=>{$('save-state').textContent='Saved locally'},error=>{$('save-state').textContent=error.message;throw error});}
 function pause(){if(!running)return;remaining=Math.max(0,Math.ceil((deadline-Date.now())/1000));elapsed+=Math.max(0,Date.now()-startedAt);running=false;}
 async function finish(){
  const worked=elapsed+(running?Math.max(0,Date.now()-startedAt):0);pause();
  data.sessions.push({objective:$('objective').value.trim()||'Focused work',seconds:Math.round(worked/1000),finished_at:new Date().toISOString()});data.sessions=data.sessions.slice(-100);data.notes=$('notes').value;startedAt=0;elapsed=0;remaining=minutes*60;render();renderHistory();try{await save();}catch{}
 }
 $('timer-start').addEventListener('click',()=>{if(running)pause();else{if(!$('objective').value.trim()){$('objective').focus();return;}startedAt=Date.now();deadline=Date.now()+remaining*1000;running=true;}render();});
 $('timer-reset').addEventListener('click',()=>{if(startedAt&&!confirm('Reset this session without saving it?'))return;running=false;startedAt=0;elapsed=0;remaining=minutes*60;render();});
 $('timer-finish').addEventListener('click',finish);
 document.querySelectorAll('[data-minutes]').forEach(b=>b.addEventListener('click',()=>{if(startedAt)return;minutes=Number(b.dataset.minutes);remaining=minutes*60;render();}));
 $('save-notes').addEventListener('click',async()=>{data.notes=$('notes').value;try{await save();}catch{}});
 setInterval(()=>{if(running&&Date.now()>=deadline){finish();return;}render();},250);
 ADNSF.ready.then(async()=>{try{const saved=await ADNSF.storage.get();if(saved&&typeof saved==='object'&&!Array.isArray(saved)){data={notes:typeof saved.notes==='string'?saved.notes:'',sessions:Array.isArray(saved.sessions)?saved.sessions.filter(s=>s&&typeof s.objective==='string'&&typeof s.seconds==='number'&&typeof s.finished_at==='string').slice(-100):[]};}connected=true;$('notes').value=data.notes;$('save-state').textContent='Local storage ready';$('save-notes').disabled=false;renderHistory();}catch(error){$('save-state').textContent=error.message;}});
 render();
})();
