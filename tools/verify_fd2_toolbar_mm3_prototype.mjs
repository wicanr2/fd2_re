// Run only inside the existing Playwright Docker image. No external service.
import fs from 'node:fs';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {pathToFileURL} from 'node:url';
import {createHash} from 'node:crypto';
const [html, prefix] = process.argv.slice(2);
if (!fs.existsSync('/.dockerenv') || !html || !prefix || !fs.statSync(html).isFile() || !fs.statSync(path.dirname(prefix)).isDirectory()) throw Error('Docker, input HTML and existing output directory required');
if (fs.existsSync(prefix+'.browser.json')) throw Error('Do not overwrite a receipt');
const candidates=['/ms-playwright/chromium-1194/chrome-linux/chrome','/ms-playwright/chromium-1194/chrome-linux64/chrome'];
const binary=candidates.find(p=>fs.existsSync(p));if(!binary)throw Error('Pinned existing Chromium missing');
const browser=spawn(binary,['--headless','--no-sandbox','--disable-dev-shm-usage','--disable-gpu','--no-first-run','--no-default-browser-check','--remote-debugging-port=9222','--remote-allow-origins=*','--force-color-profile=srgb','--user-data-dir=/tmp/fd2-toolbar-chromium','about:blank'],{stdio:['ignore','ignore','pipe']});
let errors='';browser.stderr.on('data',b=>{if(errors.length<32768)errors+=b.toString()});
const delay=ms=>new Promise(r=>setTimeout(r,ms));let socket;const pending=new Map();let serial=0;
function requireValue(ok,message){if(!ok)throw Error(message)}
try{
 let endpoint;const deadline=Date.now()+20000;
 while(Date.now()<deadline){try{const pages=await(await fetch('http://127.0.0.1:9222/json')).json();endpoint=pages.find(p=>p.type==='page')?.webSocketDebuggerUrl;if(endpoint)break}catch{}await delay(100)}
 requireValue(endpoint,'Chromium endpoint timed out');socket=new WebSocket(endpoint);await new Promise((resolve,reject)=>{socket.addEventListener('open',resolve,{once:true});socket.addEventListener('error',reject,{once:true})});
 socket.addEventListener('message',e=>{const m=JSON.parse(e.data);if(m.id&&pending.has(m.id)){const {resolve,reject}=pending.get(m.id);pending.delete(m.id);m.error?reject(Error(JSON.stringify(m.error))):resolve(m.result)}});
 const call=(method,params={})=>new Promise((resolve,reject)=>{const id=++serial;pending.set(id,{resolve,reject});socket.send(JSON.stringify({id,method,params}))});
 const evaluate=async expression=>{const r=await call('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});if(r.exceptionDetails)throw Error(r.exceptionDetails.text);return r.result.value};
 await call('Page.enable');await call('Runtime.enable');await call('Emulation.setDeviceMetricsOverride',{width:1280,height:1120,deviceScaleFactor:1,mobile:false});
 await call('Page.navigate',{url:pathToFileURL(path.resolve(html)).href+'?mode=float&size=150&open=1'});
 for(let n=0;n<100;n++){if(await evaluate('document.readyState==="complete" && !!window.fd2Prototype && document.getElementById("scene").naturalWidth>0'))break;await delay(50)}
 requireValue(await evaluate('!!window.fd2Prototype'),'Prototype did not load');await evaluate('document.fonts.ready.then(()=>true)');
 const settle=()=>evaluate('new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(()=>r(true))))');
 const move=(x,y)=>call('Input.dispatchMouseEvent',{type:'mouseMoved',x,y});
 const click=async selector=>{if(await evaluate(`!!document.querySelector(${JSON.stringify(selector)}).closest('.toolbar.hidden')`)){const stage=await evaluate('document.getElementById("stage").getBoundingClientRect().toJSON()');await move(stage.x+20,stage.y+2);await settle()}const rect=await evaluate(`document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect().toJSON()`);const x=rect.x+rect.width/2,y=rect.y+rect.height/2;await move(x,y);await call('Input.dispatchMouseEvent',{type:'mousePressed',x,y,button:'left',clickCount:1});await call('Input.dispatchMouseEvent',{type:'mouseReleased',x,y,button:'left',clickCount:1});await settle()};
 const capture=async name=>{await settle();const metrics=await call('Page.getLayoutMetrics');const height=Math.ceil(metrics.cssContentSize.height);requireValue(height<=4096,'Unbounded prototype height');const r=await call('Page.captureScreenshot',{format:'png',fromSurface:true,captureBeyondViewport:true,clip:{x:0,y:0,width:1280,height,scale:1}});const p=prefix+'-'+name+'.png';requireValue(!fs.existsSync(p),'Screenshot exists');fs.writeFileSync(p,Buffer.from(r.data,'base64'));return p};
 const captures=[];captures.push(await capture('float-150'));
 const stage=await evaluate('document.getElementById("stage").getBoundingClientRect().toJSON()');await move(stage.x+stage.width/2,stage.y+500);await settle();requireValue(await evaluate('document.getElementById("toolbar").classList.contains("hidden")'),'Floating toolbar did not hide');await move(stage.x+20,stage.y+2);await settle();requireValue(!await evaluate('document.getElementById("toolbar").classList.contains("hidden")'),'Hot edge did not reveal toolbar');
 const layouts=[];
 for(const size of [100,125,150]){
  for(let attempt=0;await evaluate('fd2Prototype.state.size')!==size;attempt++){requireValue(attempt<3,'Text size control made no progress');await click('#size')}
  for(const locale of ['zh-Hant','zh-Hans','en','ja']){
   await click(`[data-locale="${locale}"]`);requireValue(await evaluate('fd2Prototype.state.locale')===locale,'Locale control failed');const l=await evaluate('fd2Prototype.layout()');requireValue(l.controls.every(c=>!c.overflow),'Toolbar label overflow '+size+'/'+locale+' '+l.controls.filter(c=>c.overflow).map(c=>c.id).join(','));requireValue(l.controls.every(c=>c.rect.x>=l.toolbar.x&&c.rect.right<=l.toolbar.right&&c.rect.y>=l.toolbar.y&&c.rect.bottom<=l.toolbar.bottom),'Toolbar hit box out of bounds');layouts.push({size,locale,layout:l});
  }
  await click('[data-locale="zh-Hant"]');captures.push(await capture('float-'+size+'-verified'));
 }
 const imageBefore=await evaluate('document.getElementById("scene").src');
 for(const speed of [1,2,4,8,16,32]){await click('#speed');await click(`[data-speed="${speed}"]`);requireValue(await evaluate('fd2Prototype.state.speed')===speed,'Speed selection failed');requireValue(!await evaluate('fd2Prototype.state.menu'),'Speed menu remained open')}
 await click('#speed');captures.push(await capture('float-150-speed'));
 const bar=await evaluate('document.getElementById("toolbar").getBoundingClientRect().toJSON()');
 const toolbarOnly=prefix+'-toolbar-only.png';requireValue(!fs.existsSync(toolbarOnly),'Toolbar preview exists');
 const barCapture=await call('Page.captureScreenshot',{format:'png',fromSurface:true,clip:{x:bar.x,y:bar.y,width:bar.width,height:bar.height,scale:1}});
 fs.writeFileSync(toolbarOnly,Buffer.from(barCapture.data,'base64'));captures.push(toolbarOnly);
 const displayCombinations=[];
 if(await evaluate('fd2Prototype.displayModes')){
  await click('#speed');
  for(const uiHD of [false,true])for(const hd of [false,true]){
   await click(uiHD?'#ui-hd':'#ui-original');const themeBefore=await evaluate('fd2Prototype.state.hd');
   requireValue(await evaluate('fd2Prototype.state.uiHD')===uiHD,'UI selector failed');
   await click(hd?'#hd':'#original');requireValue(await evaluate('fd2Prototype.state.uiHD')===uiHD,'Theme changed UI choice');
   requireValue(await evaluate('fd2Prototype.state.hd')===hd,'Theme selector failed');
   const card=await evaluate(`(()=>{const c=document.querySelector('[data-ui="${uiHD}"][data-theme="${hd}"]');const im=c.querySelector('.mode-portrait')||c.querySelector('svg image');return {selected:c.classList.contains('selected'),icon:im?.getAttribute('src')||im?.getAttribute('href'),ui:c.dataset.ui,theme:c.dataset.theme}})()`);
   requireValue(card.selected&&card.icon,'Missing selected combination or character icon');displayCombinations.push({...card,themeBefore});
  }
  requireValue(displayCombinations[0].icon===displayCombinations[2].icon&&displayCombinations[1].icon===displayCombinations[3].icon&&displayCombinations[0].icon!==displayCombinations[1].icon,'Character icon did not follow theme independently of UI');
  const key=async name=>{await call('Input.dispatchKeyEvent',{type:'keyDown',key:name,code:name,windowsVirtualKeyCode:name==='F11'?122:113});await call('Input.dispatchKeyEvent',{type:'keyUp',key:name,code:name,windowsVirtualKeyCode:name==='F11'?122:113});await settle()};
  await key('F11');requireValue(await evaluate('!fd2Prototype.state.uiHD && fd2Prototype.state.hd'),'F11 changed theme');
  await key('F2');requireValue(await evaluate('!fd2Prototype.state.uiHD && !fd2Prototype.state.hd'),'F2 changed UI');
  await key('F11');await key('F2');
  const panel=await evaluate('document.getElementById("display-modes").getBoundingClientRect().toJSON()');
  const p=prefix+'-four-combinations.png';requireValue(!fs.existsSync(p),'Combination preview exists');
  const img=await call('Page.captureScreenshot',{format:'png',fromSurface:true,captureBeyondViewport:true,clip:{x:panel.x,y:panel.y,width:panel.width,height:panel.height,scale:1}});fs.writeFileSync(p,Buffer.from(img.data,'base64'));captures.push(p);
  for(const row of displayCombinations)row.icon_sha256=createHash('sha256').update(row.icon).digest('hex');
  for(const row of displayCombinations)delete row.icon;
 }
 for(const key of ['F2','F2','F4','F4','F4','F4']){await call('Input.dispatchKeyEvent',{type:'keyDown',key,code:key,windowsVirtualKeyCode:key==='F2'?113:115});await call('Input.dispatchKeyEvent',{type:'keyUp',key,code:key,windowsVirtualKeyCode:key==='F2'?113:115});await settle()}
 requireValue(await evaluate('fd2Prototype.state.hd && fd2Prototype.state.locale==="zh-Hant"'),'F2/F4 did not return to initial selection');
 await click('#dock');captures.push(await capture('dock-150'));requireValue(await evaluate('fd2Prototype.state.mode')==='dock','Dock mode failed');requireValue(await evaluate('fd2Prototype.layout().canvas.y >= fd2Prototype.layout().toolbar.bottom'),'Docked toolbar covered scene');
 requireValue(await evaluate('document.getElementById("scene").src')===imageBefore,'Prototype unexpectedly modified the static scene');
 const events=await evaluate('fd2Prototype.state.events');
 const digest=p=>createHash('sha256').update(fs.readFileSync(p)).digest('hex');
 fs.writeFileSync(prefix+'.browser.json',JSON.stringify({status:'PASS-PROTOTYPE-CONTROLS',browser:binary,html:{file:html,sha256:digest(html)},verifier:{file:'tools/verify_fd2_toolbar_mm3_prototype.mjs',sha256:digest(new URL(import.meta.url))},layouts,captures,events,display_combinations:displayCombinations,game_connected:false,game_speed_implemented:false,scope:'CDP real pointer/key events into local HTML only; static Game.Draw screenshot, no FD2 engine or gameplay acceptance.'},null,2)+'\n');
 process.stdout.write(JSON.stringify({status:'PASS-PROTOTYPE-CONTROLS',layouts:layouts.length,captures:captures.length,events:events.length,game_speed_implemented:false})+'\n');
}finally{
 fs.writeFileSync(prefix+'.chromium.log',errors);if(socket)socket.close();browser.kill('SIGTERM');await Promise.race([new Promise(r=>browser.once('close',r)),delay(3000)]);if(browser.exitCode===null)browser.kill('SIGKILL');
}
