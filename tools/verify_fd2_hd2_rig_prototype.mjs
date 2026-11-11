// 僅驗證可丟棄骨架示意；不宣稱 Blender、遊戲執行期或正式角色動畫完成。
import fs from 'node:fs';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {createHash} from 'node:crypto';
const [input,output]=process.argv.slice(2);
const check=(value,message)=>{if(!value)throw Error(message);};
check(fs.existsSync('/.dockerenv')&&process.getuid()!==0,'Non-root Docker required');
check(input&&fs.statSync(input).isFile()&&output&&!fs.existsSync(output),'Existing input and new output required');
check(fs.statSync(path.dirname(output)).uid===process.getuid(),'Output parent ownership');
fs.mkdirSync(output);
const sha=file=>createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const binary=['/ms-playwright/chromium-1194/chrome-linux/chrome','/ms-playwright/chromium-1194/chrome-linux64/chrome'].find(fs.existsSync);
check(binary,'Pinned Chromium required');
const browser=spawn(binary,['--headless','--no-sandbox','--disable-gpu','--disable-dev-shm-usage','--remote-debugging-port=9222','--remote-allow-origins=*','--user-data-dir=/tmp/fd2-hd2-rig','about:blank'],{stdio:['ignore','ignore','pipe']});
let socket,serial=0,log='',report={status:'FAIL',scope:'DISPOSABLE-JOINT-PROJECTION-PROTOTYPE',input_sha256:sha(input),cases:[]};
const pending=new Map(),errors=[];browser.stderr.on('data',b=>{if(log.length<16000)log+=b;});
const delay=ms=>new Promise(r=>setTimeout(r,ms));
try{
 let endpoint;
 for(let i=0;i<100;i++){try{endpoint=(await(await fetch('http://127.0.0.1:9222/json')).json()).find(p=>p.type==='page')?.webSocketDebuggerUrl;}catch{}if(endpoint)break;await delay(100);}
 check(endpoint,'Browser startup timeout');socket=new WebSocket(endpoint);await new Promise((resolve,reject)=>{socket.onopen=resolve;socket.onerror=reject;});
 socket.onmessage=e=>{const d=JSON.parse(e.data);if(d.method==='Runtime.exceptionThrown')errors.push(d.params);const p=pending.get(d.id);if(!p)return;pending.delete(d.id);clearTimeout(p.timer);d.error?p.reject(Error(JSON.stringify(d.error))):p.resolve(d.result);};
 const call=(method,params={})=>new Promise((resolve,reject)=>{const id=++serial,timer=setTimeout(()=>{pending.delete(id);reject(Error(method+' timeout'));},10000);pending.set(id,{resolve,reject,timer});socket.send(JSON.stringify({id,method,params}));});
 const evaluate=async expression=>{const r=await call('Runtime.evaluate',{expression,returnByValue:true,awaitPromise:true});check(!r.exceptionDetails,JSON.stringify(r.exceptionDetails));return r.result.value;};
 await call('Runtime.enable');await call('Page.enable');
 await call('Emulation.setDeviceMetricsOverride',{width:1280,height:980,deviceScaleFactor:1,mobile:false});
 await call('Page.navigate',{url:'file://'+path.resolve(input)});
 for(let i=0;i<100;i++){if(await evaluate('!!window.hd2RigPrototype'))break;await delay(50);}
 check(await evaluate('!!window.hd2RigPrototype'),'Prototype unavailable');
 await evaluate('hd2RigPrototype.update({playing:false})');
 const targetLengths={'hipL:kneeL':.39,'kneeL:ankleL':.38,'hipR:kneeR':.39,'kneeR:ankleR':.38,'shoulderL:elbowL':.31,'elbowL:handL':.30,'shoulderR:elbowR':.31,'elbowR:handR':.30};
 for(const action of ['idle','walk','attack','cast'])for(const direction of [0,45,90,135,180,225,270,315])for(const phase of [0,.28,.45,.7,.95]){
  const r=await evaluate(`(()=>{const p=hd2RigPrototype.update({action:${JSON.stringify(action)},direction:${direction},time:0,playing:false});return hd2RigPrototype.update({time:p.duration*${phase}})})()`);
  check(r.identity===null&&r.gameWrites===0&&!r.blenderRendered,'Scope drift');
  check(r.footAnchor[0]===180&&r.footAnchor[1]===300,'Frame anchor changed');
  for(const row of r.lengths)if(targetLengths[row.bone])check(Math.abs(row.length-targetLengths[row.bone])<1e-7,'Bone length drift: '+row.bone);
  for(const v of Object.values(r.joints))check(v.every(Number.isFinite),'Invalid joint');
  report.cases.push({action,direction,phase,boneLengths:true,anchor:true});
 }
 for(const action of ['idle','walk','attack','cast']){
  await evaluate(`hd2RigPrototype.update({action:${JSON.stringify(action)},direction:45,time:.3,playing:false})`);
  const result=await call('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});
  fs.writeFileSync(path.join(output,action+'-1280.png'),Buffer.from(result.data,'base64'));
 }
 await evaluate('hd2RigPrototype.update({action:"attack",direction:45,time:.45,playing:false,bounds:false})');
 const first=await evaluate('document.querySelector("#sprite").toDataURL()');
 await evaluate('hd2RigPrototype.update({action:"cast",time:.6});hd2RigPrototype.update({action:"attack",time:.45})');
 const repeated=await evaluate('document.querySelector("#sprite").toDataURL()');check(first===repeated,'Deterministic repeat differs');
 // The exact same joint function is sampled for this inspectable frame sequence.
 const frames=[];for(let i=0;i<24;i++){await evaluate(`hd2RigPrototype.update({action:"attack",direction:45,time:${i/24*1.2},playing:false,bounds:false})`);frames.push(await evaluate('document.querySelector("#sprite").toDataURL()'));}
 for(let i=0;i<frames.length;i++)fs.writeFileSync(path.join(output,`attack-${String(i).padStart(2,'0')}.png`),Buffer.from(frames[i].split(',')[1],'base64'));
 for(const width of [720,390]){
  await call('Emulation.setDeviceMetricsOverride',{width,height:1000,deviceScaleFactor:1,mobile:false});
  check(await evaluate('document.documentElement.scrollWidth<=innerWidth'),'Horizontal layout overflow');
  const result=await call('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});fs.writeFileSync(path.join(output,'narrow-'+width+'.png'),Buffer.from(result.data,'base64'));
 }
 check(errors.length===0,'Browser exceptions');
 report={...report,status:'PASS-160-POSES-8-DIRECTIONS',deterministic_repeat:true,sequence_frames:24,responsive_widths:[1280,720,390],game_runtime_verified:false,blender_verified:false,character_art_approved:false};
}finally{
 socket?.close();browser.kill('SIGTERM');await Promise.race([new Promise(r=>browser.on('exit',r)),delay(3000)]);if(browser.exitCode===null)browser.kill('SIGKILL');
 fs.writeFileSync(path.join(output,'receipt.json'),JSON.stringify({...report,files:fs.readdirSync(output).filter(n=>n.endsWith('.png')).map(file=>({file,sha256:sha(path.join(output,file))}))},null,2)+'\n');
 fs.writeFileSync(path.join(output,'browser.log'),log);
}
console.log(JSON.stringify({status:report.status,cases:report.cases.length,output}));
