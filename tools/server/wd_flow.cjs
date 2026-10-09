// wd_flow.cjs — reproduce "World Dashboard: use buttons then close -> main menu glitched/unclickable".
const fs=require('fs'),os=require('os'),path=require('path'),http=require('http'),{spawn}=require('child_process');
function chrome(){const c=[process.env.CHROME_BIN,'C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe',process.env.LOCALAPPDATA+'\\Google\\Chrome\\Application\\chrome.exe'].filter(Boolean);for(const x of c)if(fs.existsSync(x))return x;return null;}
function get(u){return new Promise((res,rej)=>{const r=http.get(u,x=>{let b='';x.on('data',d=>b+=d);x.on('end',()=>res({status:x.statusCode,body:b}));});r.on('error',rej);r.setTimeout(2000,()=>r.destroy(new Error('to')));});}
function wait(u,t){const s=Date.now();return new Promise((res,rej)=>{const tick=async()=>{try{const r=await get(u);if(r.status===200){let j;try{j=JSON.parse(r.body);}catch(_){return rej(new Error('bad json '+u));}return res(j);}}catch(_){}if(Date.now()-s>t)return rej(new Error('to '+u));setTimeout(tick,300);};tick();});}
class CDP{constructor(u){this.ws=new WebSocket(u);this.id=0;this.pending=new Map();this.h={};this.ws.onmessage=e=>{const m=JSON.parse(e.data);if(m.id!=null){const p=this.pending.get(m.id);if(p){this.pending.delete(m.id);m.error?p.reject(new Error(JSON.stringify(m.error))):p.resolve(m.result);}}else if(m.method&&this.h[m.method])this.h[m.method](m.params);};}open(){return new Promise((r,j)=>{this.ws.onopen=r;this.ws.onerror=j;});}on(m,c){this.h[m]=c;}send(m,p={}){const id=++this.id;return new Promise((r,j)=>{this.pending.set(id,{resolve:r,reject:j});this.ws.send(JSON.stringify({id,method:m,params:p}));});}}
async function main(){
  const CDP_PORT=9343,APP='http://localhost:8090/';
  const c=chrome();if(!c){console.log('NO CHROME');process.exit(0);}
  const ud=fs.mkdtempSync(path.join(os.tmpdir(),'vbt-wd-'));
  const proc=spawn(c,['--headless=new','--disable-gpu','--no-first-run','--no-default-browser-check','--remote-debugging-port='+CDP_PORT,'--remote-allow-origins=*','--user-data-dir='+ud,'about:blank'],{stdio:'ignore'});
  const clean=()=>{try{proc.kill('SIGKILL');}catch(e){}try{fs.rmSync(ud,{recursive:true,force:true});}catch(e){}};
  process.on('exit',clean);
  try{
    const v=await wait('http://127.0.0.1:'+CDP_PORT+'/json/version',15000);
    let tu=v.webSocketDebuggerUrl;
    try{const l=await wait('http://127.0.0.1:'+CDP_PORT+'/json/list',5000);const pt=Array.isArray(l)?l.find(t=>t.type==='page'):null;if(pt&&pt.webSocketDebuggerUrl)tu=pt.webSocketDebuggerUrl;}catch(e){}
    const cd=new CDP(tu);await cd.open();
    await cd.send('Runtime.enable');await cd.send('Page.enable');
    const loaded=new Promise(r=>{cd.on('Page.loadEventFired',()=>r());setTimeout(r,12000);});
    await cd.send('Page.navigate',{url:APP});await loaded;
    await new Promise(r=>setTimeout(r,8000));

    const dump=async(label)=>{
      const ex=`(async()=>{const cx=innerWidth/2,cy=innerHeight/2;const top=document.elementFromPoint(cx,cy);
        const bs=Array.from(document.querySelectorAll('*')).filter(e=>{const s=getComputedStyle(e);const fx=(s.position==='fixed'||s.position==='absolute')&&parseFloat(s.width||'0')>=Math.max(innerWidth,300)&&parseFloat(s.height||'0')>=Math.max(innerHeight,300);return fx&&s.display!=='none';}).map(e=>({id:e.id,cls:e.className.toString().slice(0,46),d:getComputedStyle(e).display,v:getComputedStyle(e).visibility,pe:getComputedStyle(e).pointerEvents,op:getComputedStyle(e).opacity,z:getComputedStyle(e).zIndex}));
        return JSON.stringify({body:document.body.className,pm:(window.PanelManager&&window.PanelManager.getActive)?window.PanelManager.getActive().id:'NO_PM',center:top?(top.id||top.className.toString().slice(0,36)):'null',wd:(()=>{const w=document.getElementById('world-dashboard-overlay');return w?getComputedStyle(w).display:'no';})(),bs});})()`;
      const r=await cd.send('Runtime.evaluate',{expression:ex,returnByValue:true,awaitPromise:true});
      console.log('=== '+label+' ===');
      console.log(r.result&&r.result.value?r.result.value:('ERR '+JSON.stringify(r.exceptionDetails||r)));
    };
    const ftest=async(label)=>{
      const ex=`(async()=>{
        const bs=Array.from(document.querySelectorAll('#constellation-lobby .menu-btn'));
        const wdBtn=bs.find(b=>/World Dashboard/.test(b.textContent));
        if(!wdBtn)return JSON.stringify({ok:false,r:'no btn'});
        wdBtn.click();
        await new Promise(r=>setTimeout(r,400));
        const w=document.getElementById('world-dashboard-overlay');
        const d=w?getComputedStyle(w).display:'no';
        const pe=w?getComputedStyle(w).pointerEvents:'no';
        // Try to CLICK INSIDE the reopened WD: switch to the Career tab
        let tabSwitched='n/a';
        try{const t=document.querySelector('#world-dashboard-overlay .wd-tab[data-tab="career"]');if(t){t.click();await new Promise(r=>setTimeout(r,300));const p=document.getElementById('wd-panel-career');tabSwitched=p?(getComputedStyle(p).display!=='none'?'yes':'no'):'no-panel';}}catch(e){tabSwitched='err:'+e.message;}
        const cb=document.getElementById('btn-close-wd');if(cb)cb.click();
        return JSON.stringify({ok:true,wdReopened:d,wdPointerEvents:pe,careerTabSwitched:tabSwitched});
      })()`;
      const r=await cd.send('Runtime.evaluate',{expression:ex,returnByValue:true,awaitPromise:true});
      console.log('--- FT '+label+' ---');console.log(r.result&&r.result.value?r.result.value:('ERR '+JSON.stringify(r.exceptionDetails||r)));
    };
    await dump('LOAD');
    const flow=`(async()=>{try{if(window.openWorldDashboard)window.openWorldDashboard();}catch(e){}await new Promise(r=>setTimeout(r,400));try{const t=document.querySelector('.wd-tab[data-tab="career"]');if(t)t.click();}catch(e){}await new Promise(r=>setTimeout(r,300));try{const t=document.querySelector('.wd-tab[data-tab="identity"]');if(t)t.click();}catch(e){}await new Promise(r=>setTimeout(r,700));try{const t=document.querySelector('.wd-tab[data-tab="leaderboard"]');if(t)t.click();}catch(e){}await new Promise(r=>setTimeout(r,700));return 'flow-done';})()`;
    const rf=await cd.send('Runtime.evaluate',{expression:flow,returnByValue:true,awaitPromise:true});
    console.log('flow:',rf.result&&rf.result.value);
    await dump('AFTER SPA ROUTE');
    const close=`(async()=>{let n=0;for(let i=0;i<6;i++){const cs=Array.from(document.querySelectorAll('button[id^="btn-close"],button.close-overlay-btn'));const vis=cs.find(b=>{const o=b.closest('.overlay,.vbt-overlay,.pp-hub');return o&&getComputedStyle(o).display!=='none';});if(vis){vis.click();n++;await new Promise(r=>setTimeout(r,400));}else break;}try{if(window.hideAllOverlays)window.hideAllOverlays();}catch(e){}const w=document.getElementById('world-dashboard-overlay');if(w)w.style.display='none';await new Promise(r=>setTimeout(r,500));return 'closed:'+n;})()`;
    const rc=await cd.send('Runtime.evaluate',{expression:close,returnByValue:true,awaitPromise:true});
    console.log('close:',rc.result&&rc.result.value);
    await dump('AFTER CLOSE');
    await ftest('reopen');
    clean();process.exit(0);
  }catch(e){console.log('ERR',e.message);clean();process.exit(1);}
}
main();
