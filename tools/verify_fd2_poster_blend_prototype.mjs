// Browser-only design validation. This does not run the FD2 game engine.
import fs from 'node:fs';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {createServer} from 'node:http';
import {createHash} from 'node:crypto';
const [html,prefix,figures]=process.argv.slice(2);
if(!fs.existsSync('/.dockerenv')||!html||!prefix||!figures||!fs.existsSync(html)||!fs.statSync(path.dirname(prefix)).isDirectory()||!fs.statSync(figures).isDirectory())throw Error('Docker, HTML and existing output directories required');
if(fs.existsSync(prefix+'.json'))throw Error('Existing receipt');
const binary=['/ms-playwright/chromium-1194/chrome-linux/chrome','/ms-playwright/chromium-1194/chrome-linux64/chrome'].find(fs.existsSync);
if(!binary)throw Error('Pinned Chromium absent');
const browser=spawn(binary,['--headless','--no-sandbox','--disable-gpu','--disable-dev-shm-usage','--remote-debugging-port=9222','--remote-allow-origins=*','--user-data-dir=/tmp/fd2-poster-blend','--force-color-profile=srgb','about:blank'],{stdio:['ignore','ignore','pipe']});
let errors='',socket,serial=0,webServer;const pending=new Map();browser.stderr.on('data',b=>{if(errors.length<32768)errors+=b});
const delay=ms=>new Promise(r=>setTimeout(r,ms));const requireValue=(ok,message)=>{if(!ok)throw Error(message)};
const sha=p=>createHash('sha256').update(fs.readFileSync(p)).digest('hex');
try{
 let endpoint;
 for(let i=0;i<100;i++){try{const pages=await(await fetch('http://127.0.0.1:9222/json')).json();endpoint=pages.find(p=>p.type==='page')?.webSocketDebuggerUrl;if(endpoint)break}catch{}await delay(100)}
 requireValue(endpoint,'Browser endpoint timeout');socket=new WebSocket(endpoint);await new Promise((resolve,reject)=>{socket.onopen=resolve;socket.onerror=reject});
 socket.onmessage=e=>{const d=JSON.parse(e.data);if(pending.has(d.id)){const p=pending.get(d.id);pending.delete(d.id);clearTimeout(p.timer);d.error?p.reject(Error(JSON.stringify(d.error))):p.resolve(d.result)}};
 const call=(method,params={})=>new Promise((resolve,reject)=>{const id=++serial,timer=setTimeout(()=>{pending.delete(id);reject(Error('CDP timeout '+method))},10000);pending.set(id,{resolve,reject,timer});socket.send(JSON.stringify({id,method,params}))});
 const evaluate=async expression=>{const d=await call('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(d.exceptionDetails)throw Error(JSON.stringify(d.exceptionDetails));return d.result.value};
 const settle=()=>evaluate('new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)))');
 const move=(x,y)=>call('Input.dispatchMouseEvent',{type:'mouseMoved',x,y});
 const reveal=async()=>{const r=await evaluate('document.getElementById("edge").getBoundingClientRect().toJSON()');await move(r.x+r.width/2,r.y+2);await settle()};
 const click=async selector=>{if(await evaluate(`document.querySelector(${JSON.stringify(selector)}).closest('.toolbar.hidden')!==null`))await reveal();const r=await evaluate(`document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect().toJSON()`);await move(r.x+r.width/2,r.y+r.height/2);await call('Input.dispatchMouseEvent',{type:'mousePressed',x:r.x+r.width/2,y:r.y+r.height/2,button:'left',clickCount:1});await call('Input.dispatchMouseEvent',{type:'mouseReleased',x:r.x+r.width/2,y:r.y+r.height/2,button:'left',clickCount:1});await settle()};
 const key=async(key,code,virtual)=>{await call('Input.dispatchKeyEvent',{type:'keyDown',key,code,windowsVirtualKeyCode:virtual});await call('Input.dispatchKeyEvent',{type:'keyUp',key,code,windowsVirtualKeyCode:virtual});await settle()};
 const choose=async(selector,index)=>{await click(selector);await key('Home','Home',36);for(let i=0;i<index;i++)await key('ArrowDown','ArrowDown',40);await key('Enter','Enter',13)};
 // file:// assigns opaque origins to images. Serve only these three files on container loopback.
 const allowed=new Set([path.basename(html),'fd2-hd2-poster-set1-source.png','fd2-hd2-poster-set2-source.png']);
 webServer=createServer((req,res)=>{const name=new URL(req.url,'http://localhost').pathname.slice(1);if(!allowed.has(name)){res.writeHead(404);res.end();return}const p=path.join(path.dirname(html),name);res.setHeader('Content-Type',name.endsWith('.png')?'image/png':'text/html; charset=utf-8');res.end(fs.readFileSync(p))});
 await new Promise(resolve=>webServer.listen(8123,'127.0.0.1',resolve));
 await call('Page.enable');await call('Runtime.enable');await call('Emulation.setDeviceMetricsOverride',{width:1800,height:1200,deviceScaleFactor:1,mobile:false});
 await call('Page.navigate',{url:'http://127.0.0.1:8123/'+path.basename(html)});
 for(let i=0;i<100;i++){if(await evaluate('!!window.fd2PosterPrototype && document.readyState==="complete"'))break;await delay(50)}
 await evaluate('fd2PosterPrototype.ready');await evaluate('document.fonts.ready');await delay(1300);
 const coreCheck=()=>evaluate(`(()=>{const p=fd2PosterPrototype,[x,y,w,h]=p.coreRect,src=p.sourceImages[p.state.poster==='2'?1:0],a=document.createElement('canvas');a.width=1672;a.height=941;const c=a.getContext('2d');c.drawImage(src,0,0);const base=c.getImageData(x,y,w,h).data,now=document.getElementById('scene').getContext('2d').getImageData(x,y,w,h).data;for(let i=0;i<base.length;i++)if(base[i]!==now[i])return {equal:false,byte:i};return {equal:true,bytes:base.length}})()`);
 const bandHash=()=>evaluate(`(()=>{const c=document.getElementById('scene').getContext('2d'),b=c.getImageData(248,90,48,745).data;let h=2166136261;for(const v of b){h^=v;h=Math.imul(h,16777619)}return h>>>0})()`);
 const cases=[],captures=[];
 const capture=async file=>{requireValue(!fs.existsSync(file),'Existing screenshot');await reveal();const r=await evaluate('document.getElementById("stage").getBoundingClientRect().toJSON()');const d=await call('Page.captureScreenshot',{format:'png',captureBeyondViewport:true,clip:{x:r.x,y:r.y,width:r.width,height:r.height,scale:1}});fs.writeFileSync(file,Buffer.from(d.data,'base64'));captures.push({file,sha256:sha(file)})};
 for(const poster of [1,2]){
  await choose('#posters',poster-1);await delay(1300);requireValue(await evaluate('fd2PosterPrototype.state.poster')===String(poster),'Poster real select failed');
  await click('#blend');requireValue(!await evaluate('document.getElementById("blend-settings").hidden'),'Blend popover did not open');
  await click('#width');await key('Home','Home',36);requireValue(await evaluate('fd2PosterPrototype.state.width')===0,'Gradient off failed');const hard=await bandHash();requireValue((await coreCheck()).equal,'Core differs with hard border');
  await key('End','End',35);requireValue(await evaluate('fd2PosterPrototype.state.width')===96,'Wide gradient failed');const wide=await bandHash();requireValue(hard!==wide,'Gradient control changed no pixels');requireValue((await coreCheck()).equal,'Core differs with wide gradient');
  await key('Home','Home',36);for(let i=0;i<48;i++)await key('ArrowRight','ArrowRight',39);requireValue(await evaluate('fd2PosterPrototype.state.width')===48,'48px gradient failed');const soft=await bandHash();requireValue(soft!==hard&&soft!==wide,'Middle gradient not distinct');
  await choose('#duration',3);requireValue(await evaluate('fd2PosterPrototype.state.duration')===1200,'Fade duration failed');await key('Escape','Escape',27);
  await click('#replay');const early=await evaluate('fd2PosterPrototype.state.progress');requireValue(early<1,'Fade never started');requireValue((await coreCheck()).equal,'Core faded with poster');const earlyHash=await bandHash();await delay(1350);requireValue(await evaluate('fd2PosterPrototype.state.progress')===1,'Fade not complete');requireValue(earlyHash!==await bandHash(),'Temporal fade changed no poster pixels');
  await reveal();await click('#blend');await choose('#duration',2);await key('Escape','Escape',27);await delay(700);
  const core=await coreCheck();requireValue(core.equal,'Default core differs');
  const clean=await evaluate(`(()=>{const p=fd2PosterPrototype,im=p.sourceImages[p.state.poster==='2'?1:0],s=document.createElement('canvas');s.width=1672;s.height=941;const c=s.getContext('2d');c.drawImage(im,0,0);const source=c.getImageData(0,0,1672,941).data;p.render(0);const now=document.getElementById('scene').getContext('2d').getImageData(0,0,1672,941).data;let pixels=0;for(const side of [0,1])for(let y=600;y<835;y++)for(let d=0;d<48;d++){const x=side===0?295-d:1376+d,sx=side===0?296+d:1375-d,sy=1199-y;for(let channel=0;channel<4;channel++)if(now[(y*1672+x)*4+channel]!==source[(sy*1672+sx)*4+channel])return {equal:false,x,y,channel};pixels++}p.render(1);return {equal:true,pixels}})()`);
  requireValue(clean.equal,'HUD leaked into poster terrain extension '+JSON.stringify(clean));
  cases.push({poster,hard,wide,soft,early_progress:early,core,clean_terrain_extension:clean});
  await capture(path.join(figures,'fd2-hd2-poster-blend-set'+poster+'.png'));
 }
 await choose('#posters',2);await settle();requireValue(await evaluate('fd2PosterPrototype.state.poster')==='off','Poster off failed');requireValue((await coreCheck()).equal,'Poster off changed core');
 await choose('#posters',0);await delay(700);
 const layouts=[];
 for(const width of [1800,720]){
  await call('Emulation.setDeviceMetricsOverride',{width,height:1200,deviceScaleFactor:1,mobile:false});await reveal();
  const l=await evaluate('fd2PosterPrototype.layout()');const ys=l.controls.map(c=>c.rect.y+c.rect.height/2);requireValue(Math.max(...ys)-Math.min(...ys)<2,'Toolbar wrapped to multiple rows');layouts.push({width,layout:l});
 }
 await call('Emulation.setDeviceMetricsOverride',{width:1800,height:1200,deviceScaleFactor:1,mobile:false});await reveal();
 const l=await evaluate('fd2PosterPrototype.layout()');await move(l.stage.x+l.stage.width/2,l.stage.y+l.stage.height/2);await settle();requireValue(await evaluate('document.getElementById("toolbar").classList.contains("hidden")'),'Auto-hide failed');await reveal();requireValue(!await evaluate('document.getElementById("toolbar").classList.contains("hidden")'),'Hot edge reveal failed');
 const events=await evaluate('fd2PosterPrototype.state.events');
 fs.writeFileSync(prefix+'.json',JSON.stringify({status:'PASS-POSTER-BLEND-PROTOTYPE',html:{file:html,sha256:sha(html)},verifier:{file:'tools/verify_fd2_poster_blend_prototype.mjs',sha256:sha(new URL(import.meta.url))},cases,layouts,captures,events,poster_off:true,core_region:[296,90,1080,745],core_bytes_per_sample:1080*745*4,game_connected:false,scope:'real browser pointer and key events; spatial and temporal poster pixels and invariant central concept checked; not FD2 gameplay acceptance'},null,2)+'\n');
 process.stdout.write(JSON.stringify({status:'PASS-POSTER-BLEND-PROTOTYPE',posters:cases.length,layouts:layouts.length,captures:captures.length,core_bytes:1080*745*4})+'\n');
}finally{
 fs.writeFileSync(prefix+'.chromium.log',errors);for(const p of pending.values())clearTimeout(p.timer);if(socket)socket.close();if(webServer)webServer.close();browser.kill('SIGTERM');await Promise.race([new Promise(r=>browser.once('close',r)),delay(3000)]);if(browser.exitCode===null)browser.kill('SIGKILL');
}
