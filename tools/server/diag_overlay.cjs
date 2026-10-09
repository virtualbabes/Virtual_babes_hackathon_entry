#!/usr/bin/env node
// diag_overlay.cjs — reproduce the "return to menu blocks interaction" bug.
const fs = require('fs');
const os = require('os');
const path = require('path');
const http = require('http');
const { spawn } = require('child_process');

function resolveChrome() {
  const cands = [
    process.env.CHROME_BIN,
    'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',
    process.env.LOCALAPPDATA + '\\Google\\Chrome\\Application\\chrome.exe',
  ].filter(Boolean);
  for (const c of cands) if (fs.existsSync(c)) return c;
  return null;
}
function httpGet(url){return new Promise((res,rej)=>{const r=http.get(url,x=>{let b='';x.on('data',d=>b+=d);x.on('end',()=>res({status:x.statusCode,body:b}));});r.on('error',rej);r.setTimeout(2000,()=>r.destroy(new Error('timeout')));});}
function waitFor(url,t){const s=Date.now();return new Promise((res,rej)=>{const tick=async()=>{try{const r=await httpGet(url);if(r.status===200){let j;try{j=JSON.parse(r.body);}catch(_){return rej(new Error('bad json '+url));}return res(j);}}catch(_){}if(Date.now()-s>t)return rej(new Error('timeout '+url));setTimeout(tick,300);};tick();});}
class CDP{constructor(u){this.ws=new WebSocket(u);this.id=0;this.pending=new Map();this.handlers={};this.ws.onmessage=e=>{const m=JSON.parse(e.data);if(m.id!=null){const p=this.pending.get(m.id);if(p){this.pending.delete(m.id);m.error?p.reject(new Error(JSON.stringify(m.error))):p.resolve(m.result);}}else if(m.method&&this.handlers[m.method])this.handlers[m.method](m.params);};}
open(){return new Promise((r,j)=>{this.ws.onopen=r;this.ws.onerror=j;});}
on(m,c){this.handlers[m]=c;}
send(m,p={}){const id=++this.id;return new Promise((r,j)=>{this.pending.set(id,{resolve:r,reject:j});this.ws.send(JSON.stringify({id,method:m,params:p}));});}}

async function main(){
  const APP_URL='http://localhost:8090/';
  const CDP_PORT=9341;
  const chrome=resolveChrome();
  if(!chrome){console.log('NO CHROME');process.exit(0);}
  const userData=fs.mkdtempSync(path.join(os.tmpdir(),'vbt-diag-'));
  const proc=spawn(chrome,['--headless=new','--disable-gpu','--no-first-run','--no-default-browser-check','--remote-debugging-port='+CDP_PORT,'--remote-allow-origins=*','--user-data-dir='+userData,'about:blank'],{stdio:'ignore'});
  const cleanup=()=>{try{proc.kill('SIGKILL');}catch(_){}try{fs.rmSync(userData,{recursive:true,force:true});}catch(_){}};
  process.on('exit',cleanup);
  try{
    const ver=await waitFor('http://127.0.0.1:'+CDP_PORT+'/json/version',15000);
    let targetUrl=ver.webSocketDebuggerUrl;
    try{const list=await waitFor('http://127.0.0.1:'+CDP_PORT+'/json/list',5000);const pt=Array.isArray(list)?list.find(t=>t.type==='page'):null;if(pt&&pt.webSocketDebuggerUrl)targetUrl=pt.webSocketDebuggerUrl;}catch(_){}
    const cdp=new CDP(targetUrl);await cdp.open();
    await cdp.send('Runtime.enable');await cdp.send('Page.enable');
    const loaded=new Promise(r=>{cdp.on('Page.loadEventFired',()=>r());setTimeout(r,12000);});
    await cdp.send('Page.navigate',{url:APP_URL});await loaded;
    await new Promise(r=>setTimeout(r,8000)); // allow WS identity sync (5s watchdog)

    const postLoad = await cdp.send('Runtime.evaluate',{expression:`(async()=>{
      const cfg = (window.CONFIG||{});
      const wso = document.getElementById('wallet-selector-overlay');
      const wsoHidden = wso ? wso.classList.contains('hidden') : null;
      const wsoStyle = wso ? getComputedStyle(wso) : null;
      return JSON.stringify({
        vault: cfg.VAULT_ADDRESS, vbv: cfg.VBV_ASSET_ID,
        walletOverlayHidden: wsoHidden,
        walletOverlayVisible: wsoStyle ? (wsoStyle.display!=='none' && wsoStyle.visibility!=='hidden') : null,
        walletOverlayPE: wsoStyle ? wsoStyle.pointerEvents : null,
        userAddress: window.userAddress||null,
        hasIdentity: cfg.VAULT_ADDRESS!=null
      });
    })()`,returnByValue:true,awaitPromise:true});
    console.log('--- POST-LOAD STATE ---');
    console.log(postLoad.result&&postLoad.result.value?postLoad.result.value:('ERR '+JSON.stringify(postLoad.exceptionDetails||postLoad)));

    const diag=async (openExpr, closeExpr) => {
      const expr = `(async()=>{
        function visibleBlocking(){
          const out=[];
          const els=document.querySelectorAll('*');
          const cx=window.innerWidth/2, cy=window.innerHeight/2;
          const top=document.elementFromPoint(cx,cy);
          const all=Array.from(document.querySelectorAll('.overlay,.vbt-overlay,.pp-hub,[class*="overlay"]')).filter(e=>{
            const s=getComputedStyle(e);
            return s.display!=='none' && s.visibility!=='hidden' && s.position==='fixed' && parseFloat(s.opacity||'1')>0.01;
          }).map(e=>({cls:e.className.toString().slice(0,60),id:e.id,disp:getComputedStyle(e).display,vis:getComputedStyle(e).visibility,pe:getComputedStyle(e).pointerEvents,z:getComputedStyle(e).zIndex}));
          return {centerEl: top? (top.id||top.className.toString().slice(0,40)||top.tagName):'null', visibleOverlays: all};
        }
        const before=visibleBlocking();
        try{ ${openExpr} }catch(e){}
        await new Promise(r=>setTimeout(r,500));
        const afterOpen=visibleBlocking();
        try{ ${closeExpr} }catch(e){}
        await new Promise(r=>setTimeout(r,500));
        const afterClose=visibleBlocking();
        return JSON.stringify({before,afterOpen,afterClose});
      })()`;
      const res=await cdp.send('Runtime.evaluate',{expression:expr,returnByValue:true,awaitPromise:true});
      return res.result&&res.result.value?res.result.value:('ERR '+JSON.stringify(res.exceptionDetails||res));
    };

    // Test Profile (likely "identity")
    let r1=await diag('if(window.openPlayerProfile)window.openPlayerProfile();','if(window.closePlayerProfile)window.closePlayerProfile();');
    console.log('--- PROFILE PANEL ---'); console.log(r1);

    // Test World Dashboard
    let r2=await diag('if(window.openWorldDashboard)window.openWorldDashboard();','if(window.hideAllOverlays)window.hideAllOverlays();');
    console.log('--- WORLD DASHBOARD ---'); console.log(r2);

    cleanup();process.exit(0);
  }catch(e){console.log('ERR',e.message);cleanup();process.exit(1);}
}
main();
