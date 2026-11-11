#!/usr/bin/env python3
"""從原版字形與目前戰場來源製作可操作金色命令框試稿，不接正式遊戲。"""
import argparse
import base64
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def uri(path):
    return 'data:image/png;base64,' + base64.b64encode(path.read_bytes()).decode()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ['scene', 'original', 'modern', 'out']:
        parser.add_argument('--' + key, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    names = ['attack', 'spell', 'item', 'wait']
    native = [args.original / 'ui/action_cells' / f'cell_{i:03}.png' for i in [0, 3, 6, 9]]
    icons = [args.modern / f'battle-action-{name}-style-a-28x26.png' for name in names]
    paths = []
    for file in native:
        picture = Image.open(file).convert('RGB')
        assert picture.size == (24, 20)
        bits = []
        for y in range(2, 18):
            for x in range(2, 22):
                r, g, b = picture.getpixel((x, y))
                if r > 200 and g > 100 and b < 120:
                    bits.append(f'M{x-2} {y-2}h1v1h-1z')
        assert bits
        paths.append(''.join(bits))
    payload = dict(scene=uri(args.scene), native=[uri(f) for f in native],
                   icons=[uri(f) for f in icons], paths=paths)
    html = TEMPLATE.replace('__DATA__', json.dumps(payload, ensure_ascii=False))
    args.out.mkdir()
    (args.out/'prototype.html').write_text(html)
    inputs = [args.scene, *native, *icons, Path(__file__)]
    (args.out/'SOURCE.json').write_text(json.dumps(dict(status='PROTOTYPE-NOT-RUNTIME',
        inputs=[dict(file=str(p), sha256=sha(p)) for p in inputs],
        output_sha256=sha(args.out/'prototype.html'),
        scope='原版金字輪廓不補筆；金框與金屬圖示為程式式SVG試稿。尺寸與位置僅供視覺比較，正式開收與點擊框沿原生契約。',
        distribution='LOCAL-ONLY'), ensure_ascii=False, indent=2)+'\n')
    print(args.out/'prototype.html')


TEMPLATE = r'''<!doctype html><html lang="zh-Hant"><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>炎龍騎士團2｜金色命令框試稿</title>
<style>
@font-face{font-family:FDSerif;src:url(file:///fonts/opentype/noto/NotoSerifCJK-Regular.ttc)}
*{box-sizing:border-box}body{margin:0;background:#09131f;color:#eee4c8;font-family:FDSerif,"Noto Serif CJK TC",serif;padding:24px}
h1{font-size:28px;margin:0 0 6px;font-weight:600}p{margin:6px 0 14px;color:#b6c4d3;font-size:15px}
.toolbar{display:flex;align-items:center;gap:8px;background:#11233a;border:1px solid #ad8950;padding:10px 12px;position:relative;margin:14px 0 18px;min-height:58px}
button{font:inherit;background:linear-gradient(#223952,#112237);border:1px solid #7d7354;color:#e8dfc5;padding:6px 13px;cursor:pointer}
button.active,button[aria-pressed=true]{color:#fff1bd;border-color:#efd28a;background:linear-gradient(#715329,#392913)}
.speed-panel{display:none;position:absolute;top:56px;left:360px;width:450px;z-index:2;background:#12243c;border:3px double #d4b16b;padding:16px;box-shadow:3px 5px 0 #0008}
.speed-panel.open{display:block}.speed-panel label{display:block;font-size:17px;margin:0 0 8px}.speed-panel .row{display:flex;gap:6px;margin-bottom:14px}.speed-panel p{font-size:13px;margin-bottom:0}
.cards{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:18px}.card{border:1px solid #6e634c;background:#0d1e30;padding:14px}.card h2{font-size:21px;margin:0 0 5px}.card small{color:#b1c0cf;display:block;min-height:39px;line-height:1.6;font-size:14px}.view{position:relative;width:100%;aspect-ratio:1;overflow:hidden;background:#172c2f;margin:10px 0}.view canvas{position:absolute;width:100%;height:100%}
.note{font-size:13px;color:#a9bacb;line-height:1.7}.state{display:flex;gap:12px;align-items:center;margin:16px 0}.thumb{border:1px solid #a48a53;height:26px;max-width:560px;background:#0b1929;flex:1;overflow:hidden}.beat{height:100%;width:12px;background:#e6c16e;animation:walk 2s linear infinite}@keyframes walk{from{transform:translateX(0)}to{transform:translateX(540px)}}
@media(max-width:1050px){.cards{grid-template-columns:1fr}.card{max-width:600px;margin:auto;width:100%}.toolbar{flex-wrap:wrap}.speed-panel{left:0;max-width:100%;width:450px}}
</style><body>
<h1>炎龍騎士團 2・原版強化金色命令框</h1><p>設計試稿。三欄使用同一場景與尺寸；點選四方格可比較選取狀態，尚未接入遊戲。</p>
<div class="toolbar"><span>原版　HD　｜　介面：HD　｜　繁體中文　｜</span><button id="speedToggle" aria-expanded="false">速度 ▾</button><span>影像　　全螢幕　　能力</span><span id="speedSummary" style="margin-left:auto;color:#e8c880">遊戲 1×・戰鬥跟隨</span>
<div class="speed-panel" id="speedPanel"><label>遊戲速度</label><div class="row" id="globalChoices"></div><label>戰鬥動畫速度</label><div class="row" id="battleChoices"></div><p>預設跟隨遊戲，也可獨立指定 1～4 倍。<br>戰鬥結束後恢復遊戲速度，音樂與音效維持原速。</p></div></div>
<div class="cards">
<article class="card"><h2>目前 HD</h2><small>細亮黃線、彩色小圖示<br>作為既有配色與辨識方式參考</small><div class="view"><canvas data-style="current" width="560" height="560"></canvas></div></article>
<article class="card"><h2>A・原版金字浮雕</h2><small>原版字形輪廓、藍／紅漆面<br>加厚古金框與雕刻明暗，建議方向</small><div class="view"><canvas data-style="letters" width="560" height="560"></canvas></div></article>
<article class="card"><h2>B・原版框線＋金屬圖示</h2><small>四種圖示統一金色，深藍底<br>與 A 共用分層框線及朱紅選取</small><div class="view"><canvas data-style="symbols" width="560" height="560"></canvas></div></article></div>
<div class="state"><button id="disable">切換：全部可用／左側停用</button><button id="phase">示意：目前在地圖</button><span id="effective">有效速度 1×</span><div class="thumb"><div class="beat" id="beat"></div></div></div>
<div class="note">上、左、右、下仍對應攻擊、法術／特技、道具、待機。試稿只比較材質與選取層次；正式方向、原版開收、可用條件與輸入規則保持。<br>下方移動條只示意速度設定；不是遊戲動畫或幀率量測。</div>
<script>
const DATA=__DATA__;let selected=0,disabled=false,gameSpeed=1,battleSpeed=0,inBattle=false;
const labels=['攻擊','法術／特技','道具','待機'];
const positions=[[208,138],[64,278],[352,278],[208,418]];
const defs=`<defs><linearGradient id="rim" x2="0.7" y2="1"><stop stop-color="#fff0bc"/><stop offset=".22" stop-color="#d0a65a"/><stop offset=".55" stop-color="#7c562b"/><stop offset=".77" stop-color="#d0a65f"/><stop offset="1" stop-color="#402e1c"/></linearGradient><linearGradient id="blue" x2=".2" y2="1"><stop stop-color="#253c58"/><stop offset=".55" stop-color="#152b47"/><stop offset="1" stop-color="#0a1b31"/></linearGradient><linearGradient id="red" x2=".2" y2="1"><stop stop-color="#922c23"/><stop offset=".55" stop-color="#672118"/><stop offset="1" stop-color="#3d1414"/></linearGradient><linearGradient id="letter" x2=".15" y2="1"><stop stop-color="#fff6c4"/><stop offset=".35" stop-color="#e3b95e"/><stop offset=".67" stop-color="#a56b2d"/><stop offset="1" stop-color="#f2d487"/></linearGradient></defs>`;
const shapes=[`<path d="M95 16L113 16 113 34 62 83 50 70Z"/><path d="M40 64L69 93M54 82L28 109M21 103L35 117" fill="none" stroke="#e5c77b" stroke-width="8"/>`, `<path d="M34 105L83 47" fill="none" stroke="#e5c77b" stroke-width="9"/><path d="M89 15L95 36 117 43 95 49 89 71 81 50 60 43 81 36Z"/><circle cx="89" cy="43" r="8" fill="#fff0b4"/>`, `<path d="M59 21H86V48L104 74Q115 105 72 109Q29 105 40 74L59 48Z"/><path d="M47 76Q72 67 99 76V84Q72 97 45 84Z" fill="#775325"/><path d="M55 19H90" stroke="#fff0b4" stroke-width="7"/>`, `<path d="M43 24H101M43 106H101M49 29Q49 49 70 62Q49 77 49 101H95Q95 77 74 62Q95 49 95 29Z"/><path d="M61 37H83L72 52Z M58 95L72 77 86 95Z" fill="#6a4728"/>`];
function svgIcon(style,index,active,off){let interior=active?'red':'blue',glyph=style==='letters'?`<g transform="translate(23 23) scale(4.9 4.7)"><path d="${DATA.paths[index]}" transform="translate(.3 .35)" fill="#24170e"/><path d="${DATA.paths[index]}" fill="url(#letter)"/></g>`:`<g fill="url(#letter)" stroke="#51391f" stroke-width="2">${shapes[index]}</g>`;return `<svg xmlns="http://www.w3.org/2000/svg" width="144" height="120" viewBox="0 0 144 120">${defs}<path d="M8 2H136L142 8V112L136 118H8L2 112V8Z" fill="#101915" stroke="#080e0c" stroke-width="2"/><path d="M9 4H135L140 9V111L135 116H9L4 111V9Z" fill="url(#rim)"/><path d="M13 11H131L133 13V107L131 109H13L11 107V13Z" fill="url(#${interior})" stroke="#452e19" stroke-width="3"/><path d="M14 13H129M13 15V106" fill="none" stroke="#f8e2a3" stroke-opacity=".45"/><path d="M131 14V107H15" fill="none" stroke="#070e14" stroke-width="2"/>${glyph}<path d="M7 13V7H14M130 7H137V14M137 106V113H130M14 113H7V106" fill="none" stroke="${active?'#fff2bf':'#d6b46b'}" stroke-width="2"/>${off?'<path d="M11 12H133V109H11Z" fill="#0d1b29" fill-opacity=".63"/>':''}</svg>`}
const images={};const load=src=>new Promise((resolve,reject)=>{let im=new Image();im.onload=()=>resolve(im);im.onerror=reject;im.src=src});
async function render(){let generation=++window.renderGeneration;await document.fonts.ready;for(const canvas of document.querySelectorAll('canvas')){let ctx=canvas.getContext('2d');ctx.clearRect(0,0,560,560);ctx.drawImage(images.scene,970,580,560,560,0,0,560,560);let style=canvas.dataset.style;for(let i=0;i<4;i++){let [x,y]=positions[i],off=disabled&&i===1;if(style==='current'){ctx.imageSmoothingEnabled=false;ctx.drawImage(images.native[i],x,y,144,120);ctx.save();ctx.filter='brightness(1.6)';ctx.imageSmoothingEnabled=true;ctx.drawImage(images.icons[i],2,2,24,22,x+10,y+10,124,100);ctx.restore();if(selected===i&&!off){ctx.strokeStyle='#ffb12c';ctx.lineWidth=4;ctx.strokeRect(x+2,y+2,140,116)}}else{let img=await load('data:image/svg+xml;base64,'+btoa(svgIcon(style,i,selected===i&&!off,off)));if(generation!==window.renderGeneration)return;ctx.drawImage(img,x,y)}}}window.prototypeReady=true}
window.renderGeneration=0;Promise.all([load(DATA.scene),...DATA.native.map(load),...DATA.icons.map(load)]).then(loaded=>{images.scene=loaded[0];images.native=loaded.slice(1,5);images.icons=loaded.slice(5);render()});
for(const canvas of document.querySelectorAll('canvas'))canvas.addEventListener('click',e=>{const r=canvas.getBoundingClientRect(),x=(e.clientX-r.left)*560/r.width,y=(e.clientY-r.top)*560/r.height;positions.forEach(([a,b],i)=>{if(x>=a&&x<a+144&&y>=b&&y<b+120&&!(disabled&&i===1)){selected=i;render()}})});
function speedRender(){for(const [id,values,current]of [['globalChoices',[1,2,3,4],gameSpeed],['battleChoices',[0,1,2,3,4],battleSpeed]]){let el=document.getElementById(id);el.innerHTML='';for(const value of values){let b=document.createElement('button');b.textContent=value?value+'×':'跟隨';b.setAttribute('aria-pressed',String(current===value));b.onclick=()=>{if(id==='globalChoices')gameSpeed=value;else battleSpeed=value;speedRender()};el.append(b)}}let effective=inBattle&&battleSpeed?battleSpeed:gameSpeed;document.getElementById('speedSummary').textContent=`遊戲 ${gameSpeed}×・戰鬥${battleSpeed?battleSpeed+'×':'跟隨'}`;document.getElementById('effective').textContent=`有效速度 ${effective}×`;document.getElementById('beat').style.animationDuration=(2/effective)+'s';window.prototypeState={gameSpeed,battleSpeed,inBattle,effective,selected,disabled}}
document.getElementById('speedToggle').onclick=()=>{let open=document.getElementById('speedPanel').classList.toggle('open');document.getElementById('speedToggle').setAttribute('aria-expanded',String(open))};document.getElementById('disable').onclick=()=>{disabled=!disabled;if(disabled&&selected===1)selected=0;render();speedRender()};document.getElementById('phase').onclick=()=>{inBattle=!inBattle;document.getElementById('phase').textContent='示意：目前在'+(inBattle?'戰鬥':'地圖');speedRender()};speedRender();
</script></body></html>'''


if __name__ == '__main__':
    main()
