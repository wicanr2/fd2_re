// 獨立瀏覽器呈現原型驗證；不作為正式遊戲或原版對拍的驗收。
import fs from 'node:fs';
import path from 'node:path';
import {spawn} from 'node:child_process';
import {createServer} from 'node:http';
import {createHash} from 'node:crypto';

const [directory, output] = process.argv.slice(2);
const check = (ok, message) => { if (!ok) throw Error(message); };
check(fs.existsSync('/.dockerenv') && process.getuid() !== 0, 'Non-root Docker required');
check(directory && fs.statSync(directory).isDirectory(), 'Compiled prototype required');
check(output && !fs.existsSync(output) && fs.statSync(path.dirname(output)).uid === process.getuid(), 'New owned output required');
const sha = file => createHash('sha256').update(fs.readFileSync(file)).digest('hex');
const input = JSON.parse(fs.readFileSync(path.join(directory, 'PROTOTYPE.json')));
for (const row of input.files) check(sha(path.join(directory, row.file)) === row.sha256, 'Compiled file changed: ' + row.file);
fs.mkdirSync(output);
const binary = ['/ms-playwright/chromium-1194/chrome-linux/chrome', '/ms-playwright/chromium-1194/chrome-linux64/chrome'].find(fs.existsSync);
check(binary, 'Pinned Chromium absent');
const browser = spawn(binary, ['--headless', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--remote-debugging-port=9222', '--remote-allow-origins=*', '--user-data-dir=/tmp/fd2-hd2-first-map', '--force-color-profile=srgb', 'about:blank'], {stdio: ['ignore', 'ignore', 'pipe']});
let log = '', socket, server, serial = 0;
const pending = new Map(), errors = [], cases = [], captures = [];
const delay = ms => new Promise(r => setTimeout(r, ms));
browser.stderr.on('data', b => { if (log.length < 32768) log += b; });
try {
  let endpoint;
  for (let i = 0; i < 100; i++) {
    try { endpoint = (await (await fetch('http://127.0.0.1:9222/json')).json()).find(p => p.type === 'page')?.webSocketDebuggerUrl; } catch {}
    if (endpoint) break;
    await delay(100);
  }
  check(endpoint, 'Browser endpoint timeout');
  socket = new WebSocket(endpoint);
  await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
  socket.onmessage = e => {
    const d = JSON.parse(e.data);
    if (d.method === 'Runtime.exceptionThrown') errors.push(d.params);
    if (!pending.has(d.id)) return;
    const p = pending.get(d.id); pending.delete(d.id); clearTimeout(p.timer);
    d.error ? p.reject(Error(JSON.stringify(d.error))) : p.resolve(d.result);
  };
  const call = (method, params = {}) => new Promise((resolve, reject) => {
    const id = ++serial, timer = setTimeout(() => { pending.delete(id); reject(Error('CDP timeout: ' + method)); }, 10000);
    pending.set(id, {resolve, reject, timer}); socket.send(JSON.stringify({id, method, params}));
  });
  const evaluate = async expression => {
    const d = await call('Runtime.evaluate', {expression, awaitPromise: true, returnByValue: true});
    check(!d.exceptionDetails, JSON.stringify(d.exceptionDetails)); return d.result.value;
  };
  const settle = () => evaluate('new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)))');
  const state = () => evaluate('hd2PrototypeState()');
  const rect = selector => evaluate(`document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect().toJSON()`);
  const move = (x, y) => call('Input.dispatchMouseEvent', {type: 'mouseMoved', x, y});
  const reveal = async () => { const r = await rect('#world'); await move(r.x + r.width / 2, r.y + 2); await delay(180); };
  const click = async selector => {
    if (selector !== '#world') await reveal();
    const r = await rect(selector), x = r.x + r.width / 2, y = r.y + r.height / 2;
    await move(x, y);
    for (const type of ['mousePressed', 'mouseReleased']) await call('Input.dispatchMouseEvent', {type, x, y, button: 'left', clickCount: 1});
    await settle();
  };
  const key = async (key, code, virtual) => {
    await call('Input.dispatchKeyEvent', {type: 'keyDown', key, code, windowsVirtualKeyCode: virtual});
    await call('Input.dispatchKeyEvent', {type: 'keyUp', key, code, windowsVirtualKeyCode: virtual}); await settle();
  };
  const choose = async index => { await click('#height'); await key('Home', 'Home', 36); for (let i = 0; i < index; i++) await key('ArrowDown', 'ArrowDown', 40); await key('Enter', 'Enter', 13); };
  const invariant = async () => {
    const s = await state();
    check(s.actor.identity === 0 && s.actor.x === 7 && s.actor.y === 20, 'Standing prototype moved or changed actor');
    check(s.mapIdentity === s.hudIdentity && s.mapImage === s.hudImage, 'Map/HUD identity or image differs');
    check(!s.rulesExecuted && s.saveWrites === 0, 'Unexpected gameplay or save mutation');
    check(await evaluate('document.querySelector("#portrait").complete && document.querySelector("#portrait").naturalWidth>0 && document.querySelector("#icon").naturalWidth>0'), 'HUD image missing');
    return s;
  };
  const pixels = () => evaluate(`(()=>{const c=document.querySelector('#world').getContext('2d'),a=hd2PrototypeState().foot,b=c.getImageData(Math.max(0,a.x-140),Math.max(0,a.y-240),280,250).data;let h=2166136261;for(const v of b){h^=v;h=Math.imul(h,16777619)}return h>>>0})()`);
  const capture = async name => {
    await reveal(); await evaluate('document.fonts.ready'); await settle();
    const r = await rect('#stage'), file = path.join(output, name + '.png');
    const d = await call('Page.captureScreenshot', {format: 'png', captureBeyondViewport: false, clip: {x: r.x, y: r.y, width: r.width, height: r.height, scale: 1}});
    fs.writeFileSync(file, Buffer.from(d.data, 'base64')); captures.push({file, sha256: sha(file)});
  };
  const allowed = new Set(input.files.map(row => row.file));
  server = createServer((req, res) => {
    const name = new URL(req.url, 'http://localhost').pathname.slice(1);
    if (!allowed.has(name)) { res.writeHead(404); res.end(); return; }
    res.setHeader('Content-Type', name.endsWith('.png') ? 'image/png' : name.endsWith('.ttc') ? 'font/collection' : 'text/html; charset=utf-8');
    res.end(fs.readFileSync(path.join(directory, name)));
  });
  await new Promise(resolve => server.listen(8123, '127.0.0.1', resolve));
  await call('Page.enable'); await call('Runtime.enable');
  await call('Emulation.setDeviceMetricsOverride', {width: 1800, height: 1200, deviceScaleFactor: 1, mobile: false});
  await call('Page.navigate', {url: 'http://127.0.0.1:8123/index.html'});
  for (let i = 0; i < 100; i++) { if (await evaluate('!!window.hd2PrototypeState && hd2PrototypeState().ready && document.readyState==="complete"')) break; await delay(100); }
  check((await state()).ready, 'Prototype load timeout');
  await evaluate('document.fonts.ready'); await settle();
  const baseline = await invariant();
  check(baseline.foot.x === 1104 && baseline.foot.y === 744, 'Unexpected initial foot anchor');
  for (const [index, height] of [144, 168, 192].entries()) {
    await choose(index); const s = await invariant();
    check(s.height === height, 'Real size select failed');
    check(JSON.stringify(s.foot) === JSON.stringify(baseline.foot), 'Size changed foot anchor');
    cases.push({kind: 'height', height, foot: s.foot, pixel_hash: await pixels()});
    await capture('standing-' + height);
  }
  check(new Set(cases.map(c => c.pixel_hash)).size === 3, 'Size selector changed no actor pixels');
  await choose(1); await click('#world');
  await key('F2', 'F2', 113); let s = await invariant(); check(s.mode === 'original', 'F2 original toggle failed');
  check(JSON.stringify(s.foot) === JSON.stringify(baseline.foot), 'Original toggle moved foot');
  await capture('original-actor');
  await click('#mode'); check((await invariant()).mode === 'hd2', 'Real toolbar toggle failed');
  cases.push({kind: 'original-toggle', same_identity: true, same_anchor: true});
  const before = await pixels(); await click('#grid'); check(!(await state()).grid, 'Grid toggle failed');
  check(before !== await pixels(), 'Grid toggle changed no pixels'); await click('#grid');
  await click('#world');
  for (let i = 0; i < 30; i++) await key('ArrowLeft', 'ArrowLeft', 37);
  for (let i = 0; i < 30; i++) await key('ArrowUp', 'ArrowUp', 38);
  s = await invariant(); check(s.cursor.x === 0 && s.cursor.y === 0 && s.camera.x === 0 && s.camera.y === 0, 'Upper-left boundary failed');
  for (let i = 0; i < 30; i++) await key('ArrowRight', 'ArrowRight', 39);
  for (let i = 0; i < 30; i++) await key('ArrowDown', 'ArrowDown', 40);
  s = await invariant(); check(s.cursor.x === 23 && s.cursor.y === 23 && s.camera.x === 11 && s.camera.y === 16, 'Lower-right boundary failed');
  cases.push({kind: 'bounded-camera-and-cursor', minimum: [0, 0], maximum: [23, 23], maximum_camera: [11, 16]});
  await click('#reset'); check(JSON.stringify((await state()).camera) === JSON.stringify(baseline.camera), 'Deployment reset failed');
  for (const width of [1800, 720]) {
    await call('Emulation.setDeviceMetricsOverride', {width, height: 1200, deviceScaleFactor: 1, mobile: false});
    await click('#world');
    const r = await rect('#world');
    await move(r.x + (24 + 2.5 * 144) / 1920 * r.width, r.y + (24 + 2.5 * 144) / 1200 * r.height);
    s = await invariant(); check(s.cursor.x === 2 && s.cursor.y === 18, 'Scaled pointer mapping failed: ' + width);
    await delay(180); check(await evaluate('document.querySelector("#toolbar").getBoundingClientRect().bottom <= document.querySelector("#world").getBoundingClientRect().top+1'), 'Toolbar hide failed');
    const cursor = s.cursor;
    await move(r.x + 2, r.y + r.height / 2); await settle();
    check(JSON.stringify((await state()).cursor) === JSON.stringify(cursor), 'Blank scene margin accepted map input');
    await reveal(); check(await evaluate('document.querySelector("#toolbar").classList.contains("open")'), 'Toolbar reveal failed');
    check(JSON.stringify((await state()).cursor) === JSON.stringify(cursor), 'Toolbar hot edge accepted map input');
    const layout = await evaluate(`(()=>{const t=document.querySelector('#toolbar'),controls=[...t.querySelectorAll('button,label')].map(e=>e.getBoundingClientRect().toJSON()),info=document.querySelector('.info').getBoundingClientRect().toJSON(),copy=document.querySelector('.copy');return{controls,info,copy:{height:copy.clientHeight,scroll:copy.scrollHeight},bodyWidth:document.body.scrollWidth,viewport:innerWidth,toolbarWidth:t.clientWidth,toolbarScroll:t.scrollWidth}})()`);
    const centers = layout.controls.map(c => c.y + c.height / 2);
    check(Math.max(...centers) - Math.min(...centers) < 2, 'Toolbar wrapped');
    check(layout.bodyWidth <= width, 'Page horizontal overflow');
    check(layout.copy.scroll <= layout.copy.height + 1, 'HUD text clipped');
    cases.push({kind: 'scaled-pointer-and-layout', width, layout});
    await capture('layout-' + width);
  }
  check(errors.length === 0, 'Browser runtime exception');
  const report = {status: 'PASS-HD2-STANDING-PRESENTATION-PROTOTYPE', production_game_connected: false, original_parity_claimed: false,
    normal_player_path_claimed: false, rules_executed: false, save_writes: 0, identity: 0,
    input: {file: path.join(directory, 'PROTOTYPE.json'), sha256: sha(path.join(directory, 'PROTOTYPE.json'))},
    verifier_sha256: sha(new URL(import.meta.url)), chromium: binary, cases, captures, runtime_errors: errors,
    scope: 'Real browser pointer, keyboard, size, original-actor comparison, camera bounds and matching map/HUD identity; single standing draft over provisional HD1 terrain.'};
  fs.writeFileSync(path.join(output, 'receipt.json'), JSON.stringify(report, null, 2) + '\n');
  process.stdout.write(JSON.stringify({status: report.status, cases: cases.length, captures: captures.length}) + '\n');
} finally {
  fs.writeFileSync(path.join(output, 'chromium.log'), log);
  for (const p of pending.values()) clearTimeout(p.timer);
  if (socket) socket.close(); if (server) server.close(); browser.kill('SIGTERM');
  await Promise.race([new Promise(r => browser.once('close', r)), delay(3000)]);
  if (browser.exitCode === null) browser.kill('SIGKILL');
}
