// ADNF v1 guest SDK. The host only exposes explicitly granted, app-scoped methods.
(()=>{
 'use strict';
 let port=null,sequence=0;const pending=new Map();let readyResolve;
 const ready=new Promise(resolve=>{readyResolve=resolve});
 window.addEventListener('message',function connect(event){
  if(event.source!==window.parent||event.data?.type!=='adnf:connect'||port||event.ports.length!==1)return;
  port=event.ports[0];port.onmessage=event=>{const response=event.data,job=pending.get(response?.id);if(!job)return;pending.delete(response.id);clearTimeout(job.timer);response.error?job.reject(new Error(response.error)):job.resolve(response.value)};port.start();readyResolve();
 });
 async function call(method,value){await ready;return new Promise((resolve,reject)=>{const id=++sequence;const timer=setTimeout(()=>{pending.delete(id);reject(new Error('DNS Filter did not respond'))},15000);pending.set(id,{resolve,reject,timer});port.postMessage({id,method,value})})}
 Object.defineProperty(window,'ADNSF',{value:Object.freeze({ready,storage:Object.freeze({get:()=>call('storage.get'),set:value=>call('storage.set',value)})}),writable:false,configurable:false});
 if(window.parent!==window)window.parent.postMessage({type:'adnf:ready'},'*');
})();
