"""Docker 內核對普通 START 對話前綴，保存本機對照與純雜湊收據。"""
import argparse
import hashlib
import html
import json
import os
from pathlib import Path
import shutil
import subprocess

p = argparse.ArgumentParser()
p.add_argument('repo', type=Path)
p.add_argument('version')
p.add_argument('capture', type=Path)
p.add_argument('oracle', type=Path)
p.add_argument('--count', type=int, default=38)
a = p.parse_args()
assert Path('/.dockerenv').exists()
root = a.repo
full = root / 'dist-all' / a.version / 'full'
assert full.stat().st_uid == os.getuid()
review = full / 'dialogue-review'
assert not review.exists(), '拒絕覆寫對照收據'
sha = lambda f: hashlib.sha256(f.read_bytes()).hexdigest()
manifest = json.loads((full / 'manifest.json').read_text())
app = full / f'FD2-{a.version}-linux-x86_64-full.AppImage'
assert sha(app) == manifest['sha256']
for name, expected in manifest['source_files']:
    assert sha(root / name) == expected, name
pages = json.loads((a.capture / 'pages.json').read_text())
oracle = json.loads((a.oracle / 'receipt.json').read_text())
assert len(pages) == a.count and len(oracle['dialogue_receipts']) >= a.count
assert all(not row['state'].get('error') for row in pages)
review.mkdir()
rows = []
for i in range(a.count):
    original = oracle['dialogue_receipts'][i]
    assert original['index'] == pages[i]['index'] == i
    name = f'dialogue-{i:03d}.png'
    source = a.oracle / 'frames' / name
    remake = a.capture / name
    assert sha(source) == original['sha256']
    result = json.loads(subprocess.check_output([
        str(root / 'work/dialogue-cause-20260908/pixel-probe'), str(source), str(remake),
        'upper' if original['portrait_anchor'] == 1832 else 'lower']))
    assert result['nonuniform_scaled_pixels'] == [0, 0]
    (a.capture / (name + '.diff.json')).write_text(json.dumps(result))
    for prefix, file in [('original', source), ('remake', remake)]:
        shutil.copyfile(file, review / f'{prefix}-{i:03d}.png')
    rows.append(dict(index=i, original_sha256=sha(source), remake_sha256=sha(remake),
                     original_step=original['step'], different_pixels=result['different_pixels'],
                     regions=result['regions'], bounds=result['bounds']))
receipt = dict(version=a.version, appimage_sha256=sha(app), pages=rows,
    input_method='實際 AppImage、全新設定、普通 START／Enter；原版 dosgolem 在等待邊界送 BIOS Enter',
    classification='對應對話頁；全畫面未遮罩；未鎖亂數、嘴型及人物動畫相位，不是全機器同狀態',
    oracle_receipt_sha256=sha(a.oracle / 'receipt.json'),
    input_log_sha256=sha(a.capture / 'remake-inputs.json'),
    state_log_sha256=sha(a.capture / 'remake-state.jsonl'),
    tests_log_sha256=sha(root / 'work/dialogue-cause-20260908/pagination-tests.log'),
    limitations=['尚未完成第一關父子登場前全段對拍', '音訊靜音，未驗收音畫同步',
                 '王宮第一頁尚有13個未解背景像素，人物動畫相位另計',
                 '廣泛抽測兩項失敗另存 pan-tests.log，未宣稱全套通過'])
(review / 'receipt.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
for name in ['pages.json', 'remake-inputs.json', 'remake-state.jsonl', 'remake-app.log']:
    shutil.copyfile(a.capture / name, review / name)
shutil.copyfile(root / 'work/dialogue-cause-20260908/pagination-tests.log', review / 'tests.log')
opts = ''.join(f'<option value="{r["index"]}">{r["index"]+1}：全圖差 {r["different_pixels"]} 像素</option>' for r in rows)
page = '''<!doctype html><html lang="zh-Hant"><meta charset="utf-8"><title>FD2 對話前綴對照</title>
<style>body{background:#171b25;color:#eef;font:18px sans-serif;margin:24px}a{color:#acd}section{display:flex;gap:20px;flex-wrap:wrap}figure{margin:12px 0}img{width:640px;max-width:95vw;image-rendering:pixelated}select,button{font:inherit}pre{white-space:pre-wrap}</style>
<h1>VERSION：普通 START 對話對照</h1><p>原版由 dosgolem 自行執行；重製為實際 AppImage。完整畫面未遮罩，但未鎖動畫相位，不宣稱全機器同狀態。</p>
<p>本包修正鏡頭終點交接、跨頁保留前文與翻頁箭頭。第一關父子登場前全段、音訊與剩餘差異仍待驗收。</p>
<button onclick="move(-1)">上一頁</button> <select id="page" onchange="show()">OPTIONS</select> <button onclick="move(1)">下一頁</button>
<section><figure><figcaption>原版 dosgolem</figcaption><img id="original"></figure><figure><figcaption>重製 AppImage</figcaption><img id="remake"></figure></section><pre id="detail"></pre>
<p><a href="receipt.json">完整雜湊與差異收據</a> · <a href="tests.log">針對性回歸</a> · <a href="remake-state.jsonl">正常輸入狀態日誌</a></p>
<script>const rows=ROWS;function show(){const i=+document.getElementById('page').value,n=String(i).padStart(3,'0');for(const k of ['original','remake'])document.getElementById(k).src=k+'-'+n+'.png';document.getElementById('detail').textContent=JSON.stringify(rows[i],null,2)}function move(d){const s=document.getElementById('page');s.selectedIndex=Math.max(0,Math.min(rows.length-1,s.selectedIndex+d));show()}show()</script></html>'''
page = page.replace('VERSION', html.escape(a.version)).replace('OPTIONS', opts).replace('ROWS', json.dumps(rows))
(review / 'index.html').write_text(page)
manifest['status'] = f'已通過實際 AppImage 普通 START 前 {a.count} 頁重播；有剩餘差異，非全段一致'
manifest['validation'] = dict(receipt='dialogue-review/receipt.json', sha256=sha(review / 'receipt.json'))
(full / 'manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
dest = root / 'docs/data/ui-traces/dialogue-prefix-fix-20260908.json'
assert dest.parent.stat().st_uid == os.getuid()
dest.write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
for file in review.iterdir():
    assert file.stat().st_uid == os.getuid()
print(json.dumps([dict(index=r['index'], pixels=r['different_pixels'], regions=r['regions']) for r in rows], ensure_ascii=False))
