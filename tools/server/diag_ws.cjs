#!/usr/bin/env node
// diag_ws.cjs — capture live WS frames the browser receives, to see if `identity` arrives.
const fs=require('fs'),os=require('os'),path=require('path'),http=require('http'),{spawn}=require('child_process');
function resolveChrome(){const cands=[process.env.CHROME_BIN,'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',process.env.LOCALAPPDATA+'\\Google\\Chrome\\Application\\chrome.exe'].filter(Boolean);for(const c of cands)if(fs.existsSync(c))return c;return null;}
function httpGet(u){return new Promise((res,rej)=>{const r=http.get(u,x=>{let b='';x.on('data',d=>b+=d);x.on('end',()=>res({status:x.statusCode,body:b}));});r.on('error',rej);r.setTimeout(2000,()=>r.destroy(new Error('timeout')));});}
function waitFor(u,t){const s=Date.now();return new Promise((res,rej)=>{const tick=async()=>{try{const r=await httpGet(u);if(r.status===200){try{return res(JSON.parse(r.body));}catch(_){return rej(new Error('badjson'));}}else{if(Date.now()-s>t)return rej(new Error('timeout '+u));setTimeout(tick,300);}}catch(_){if(Date.now()-s>t)return rej(new Error('timeout '+u));setTimeout(tick,300);}};tick();});}
class CDP{constructor(u){this.ws=new WebSocket(u);this.id=0;this.pending=new Map();this.handlers={};this.ws.onmessage=e=>{const m=JSON.parse(e.data);if(m.id!=null){const p=this.pending.get(m.id);if(p){this.pending.delete(m.id);m.error?p.reject(new Error(JSON.stringify(m.error))):p.resolve(m.result);}}else if(m.method&&this.handlers[m.method])this.handlers[m.method](m.params);};}
open(){return new Promise((r,j)=>{this.ws.onopen=r;this.ws.onerror=j;});}on(m,c){this.handlers[m]=c;}send(m,p={}){const id=++this.id;return new Promise((r,j)=>{this.pending.set(id,{resolve:r,reject:j});this.ws.send(JSON.stringify({id,method:m,params:p}));});}}
const chrome=resolveChrome();
async function main(){
  const APP_URL='http://localhost:8090/',CDP_PORT=9342;
  if(!chrome){console.log('NO CHROME');process.exit(0);}
  const ud=fs.mkdtempSync(path.join(os.tmpdir(),'vbt-ws-'));
  const proc=spawn(chrome,['--headless=new','--disable-gpu','--no-first-run','--no-default-browser-check','--remote-debugging-port='+CDP_PORT,'--remote-allow-origins=*','--user-data-dir='+ud,'about:blank'],{stdio:'ignore'});
  const cleanup=()=>{try{proc.kill('SIGKILL');}catch(_){}try{fs.rmSync(ud,{recursive:true,force:true});}catch(_){}};process.on('exit',cleanup);
  try{
    const ver=await waitFor('http://127.0.0.1:'+CDP_PORT+'/json/version',15000);let targetUrl=ver.webSocketDebuggerUrl;
    try{const list=await waitFor('http://127.0.0.1:'+CDP_PORT+'/json/list',5000);const pt=Array.isArray(list)?list.find(t=>t.type==='page'):null;if(pt&&pt.webSocketDebuggerUrl)targetUrl=pt.webSocketDebuggerUrl;}catch(_){}
    const cdp=new CDP(targetUrl);await cdp.open();
    const frames=[];
    cdp.on('Network.webSocketFrameReceived',p=>{try{const d=JSON.parse(p.response.payloadData);frames.push('RECV '+d.type);}catch(_){frames.push('RECV_RAW '+(p.response.payloadData||'').slice(0,80));}});
    cdp.on('Network.webSocketFrameSent',p=>{try{const d=JSON.parse(p.response.payloadData);frames.push('SENT '+d.type);}catch(_){frames.push('SENT_RAW');}});
    await cdp.send('Runtime.enable');await cdp.send('Page.enable');await cdp.send('Network.enable');
    const loaded=new Promise(r=>{cdp.on('Page.loadEventFired',()=>r());setTimeout(r,12000);});
    await cdp.send('Page.navigate',{url:APP_URL});await loaded;
    await new Promise(r=>setTimeout(r,8000));
    console.log('WS FRAMES (first 40):');console.log(frames.slice(0,40).join('\n'));
    console.log('TOTAL FRAMES:',frames.length);
    cleanup();process.exit(0);
  }catch(e){console.log('ERR',e.message);cleanup();process.exit(1);}
}
main();
