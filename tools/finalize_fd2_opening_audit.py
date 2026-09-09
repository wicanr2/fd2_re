#!/usr/bin/env python3
"""整理 2026-09-07 本機 FD2 操作收據；原版圖像不加入公開庫。"""
import hashlib
import json
import pathlib
import shutil

assert pathlib.Path('/.dockerenv').exists(), '只能在 Docker 內執行'
src, shots = pathlib.Path('/src'), pathlib.Path('/shots')
version = (src / 'VERSION').read_text().strip()
assert version == 'v.1.0.9-20260907'
full = src / 'dist-all' / version / 'full'
assert full.stat().st_uid == 1000
out = full / 'input-audit'
out.mkdir(exist_ok=True)
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
names = [
    'original-check-range.png', 'original-check-turn2-info.png',
    'oracle-turn3-info.png', 'oracle-after-end3.png', 'oracle-end2-prompt.png',
    'oracle-neutral-cycle1.png', 'oracle-neutral-cycle2.png', 'oracle-neutral-cycle3.png',
    'oracle-neutral-status.png', 'oracle-neutral-commands.png',
    'oracle-neutral-acted-status.png', 'oracle-neutral-acted-sheet.png', 'oracle-neutral-inputs.txt',
    'v107-first-control.png', 'v107-initial-range.png', 'v107-cancel-range.png',
    'v107-cycle1.png', 'v107-cycle2.png', 'v107-cycle3.png', 'v107-cycle-skips-acted.png',
    'v107-yuni-status.png', 'v107-yuni-commands.png', 'v107-yuni-closed.png',
    'v107-movement-ring.png', 'v107-waited.png', 'v107-acted-status.png',
    'v107-end2-prompt.png', 'v107-after-end1.png', 'v107-turn3-info.png',
    'v109-title.png', 'v109-loaded.png', 'v109-chest-question.png',
    'v109-chest-no-selected.png', 'v109-chest-no-closed.png',
    'v109-chest-question-again.png', 'v109-chest-found-herb.png',
    'v109-chest-yes-closed.png', 'v109-chest-inventory.png', 'v109-turn3-info.png',
    'v109-end1-prompt.png', 'v109-after-end1.png', 'v109-end2-prompt.png', 'v109-after-end2.png',
    'v107-opening-inputs.json', 'v107-driver-inputs.jsonl', 'v107-state.jsonl', 'v107-app.log',
    'v109-driver-inputs.jsonl', 'v109-state.jsonl', 'v109-app.log', 'v109-running-binary.txt',
    'v105-rules.log', 'v105-ui-tests.log', 'initial-gold-tests.log',
    'v107-neutral-tests.log', 'v107-regression.log', 'v107-full-opening-test.log',
    'v108-treasure-tests.log', 'v109-wait-tests.log', 'v109-build.log', 'regression.log', 'baseline-failures.log',
    'replay.py', 'v107_driver.py', 'v107_phase.py', 'v107_neutral.py',
    'v109_driver.py', 'v109_phase.py', 'v109_chest.py',
]
for name in names:
    shutil.copyfile(shots / name, out / name)
start = [json.loads(s) for s in (shots/'v107-state.jsonl').read_text().splitlines()]
final = [json.loads(s) for s in (shots/'v109-state.jsonl').read_text().splitlines()]
assert any(s.get('turn') == 3 and s.get('system_info_phase') == 'steady' for s in start)
assert any(s.get('turn') == 3 and s.get('system_info_phase') == 'steady' for s in final)
assert any(s.get('treasure_ack') for s in final)
assert not any(s.get('error') for s in start + final)
app = full / f'FD2-{version}-linux-x86_64-full.AppImage'
images = [(name, sha(out/name)) for name in names if name.endswith('.png')]
receipt = dict(
    schema_version=1, kind='fd2_local_input_audit', date='2026-09-07', version=version,
    appimage_sha256=sha(app), running_binary_sha256=(shots/'v109-running-binary.txt').read_text().split()[0],
    original_exe_sha256='222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f',
    original_continue_save_sha256='6d14f2c22562cabca83725084f1a9d6539a1d4066da5c1debcdadb446812691f',
    original_runner='DOSBox 輔助基準，非 dosgolem 自行執行',
    rng_synchronized=False, audio_verified=False, complete_parity=False, public_distribution=False,
    normal_start_path=dict(version='v.1.0.7-20260907', entry='START', end='battle_ch01 TURN 003，父子登場前',
        actions=['完整序幕', '選人與取消', 'Escape 循環', 'F2 狀態兩頁', '移動待機', '已行動狀態', '兩次 END 與回合聚焦'],
        direct_entry=False, save_injection=False),
    final_package_check=dict(version=version, entry='標題 → 合法原版 CONTINUE → 第一關 TURN 003',
        actions=['普通移動待機', '寶箱 NO 不領取', 'END 後再開箱 YES', '藥草名稱回覆與取得後確認', '狀態物品欄', '兩次 END', '零金錢戰況面板'],
        save='未修改的合法原版存檔，唯讀掛載', direct_entry=False,
        startup_regression='完整 97 句開場與來源角色回歸通過；此最終版本未重跑完整序幕 GUI'),
    comparison_class='相近狀態／排版比較；未宣稱整幀逐像素或同亂數一致',
    remaining=[
        'dosgolem 自行重生的正式 FD2 原版收據尚缺；精確按鍵重複速率、動畫相位與音訊未閉合',
        '滿物品欄交換、金錢與特殊事件寶藏分支未以本次窄對話修正提升',
        'F5/F9 為既有 JSON 節點重啟，不是原版目前戰況續存；真正 CONTINUE 另行驗證',
        '整個 cmd/fd2 套件未全綠；限定回歸通過，五項既有失敗另有 HEAD 重現',
    ], images=images,
)
(out/'receipt.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
rows = [
    ('選人與移動範圍', 'original-check-range.png', 'v107-initial-range.png', '相近狀態；未鎖定角色動畫相位。'),
    ('Escape 第二次聚焦悠妮', 'oracle-neutral-cycle2.png', 'v107-cycle2.png', '原版 CONTINUE、重製 START；相近狀態，鏡頭初值不同。'),
    ('F2 角色狀態', 'oracle-neutral-status.png', 'v107-yuni-status.png', '核對頭像、框線、字型、物品與數值排版。'),
    ('狀態的命令頁', 'oracle-neutral-commands.png', 'v107-yuni-commands.png', '同一確認／返回順序，不宣稱整幀像素一致。'),
    ('寶箱取得藥草的對話', 'oracle-neutral-acted-status.png', 'v109-chest-found-herb.png', '只比較對話；原版第一回合、重製第二回合，背景戰況不同。原版檔名保留最初預想操作，實際圖為寶箱回覆。'),
    ('戰況資訊', 'original-check-turn2-info.png', 'v109-turn3-info.png', '只比較面板、字型與零金錢；回合和戰況不同。'),
    ('END 確認', 'oracle-end2-prompt.png', 'v107-end2-prompt.png', '原版 CONTINUE 與重製 START；只比較確認框。'),
]
html = '''<!doctype html><html lang="zh-Hant"><meta charset="utf-8"><title>FD2 操作比較</title>
<style>body{max-width:1320px;margin:32px auto;padding:0 20px;background:#f4f1e8;color:#222;font:17px/1.7 sans-serif}h1{font-size:28px}h2{font-size:22px}.pair{display:grid;grid-template-columns:1fr 1fr;gap:12px}img{width:100%;image-rendering:pixelated;border:1px solid #555}small{display:block}</style>
<h1>FD2 原版主題操作比較</h1>
<p>本機候選 v.1.0.9-20260907。已修開場地形與人物、初始視圖、移動資訊框與路徑、指令圖示、取消游標、滿血誤判敗退、戰況面板與金錢；並補上 Escape 換人、回合聚焦、角色狀態入口與普通物品寶箱問答。</p>
<p>原版實測 TURN 003 結束後出現父子對話，故以登場前為比較終點。v.1.0.7 完整 START 與 v.1.0.9 合法原版 CONTINUE 均實際到 TURN 003；兩條路徑分開記錄。DOSBox 為輔助基準；未鎖亂數、未驗音訊，也不宣稱逐像素完全一致。<a href="receipt.json">詳細收據與限制</a></p>'''
for title, a, b, caption in rows:
    html += f'<h2>{title}</h2><div class="pair"><div>原版（DOSBox）<img src="{a}"></div><div>重製實際 AppImage<img src="{b}"></div></div><small>{caption}</small>'
html += '<h2>寶箱取消與再開啟</h2><div class="pair"><img src="v109-chest-no-selected.png"><img src="v109-chest-question-again.png"></div><p>NO 不領取，下回合仍可開啟。圖像含原版衍生內容，只留本機，不加入公開發行。</p></html>'
(out/'index.html').write_text(html)
shutil.copyfile(pathlib.Path(__file__), out/'finalize_fd2_opening_audit.py')
(out/'SHA256SUMS').write_text(''.join(f'{sha(p)}  {p.name}\n' for p in sorted(out.iterdir()) if p.is_file() and p.name!='SHA256SUMS'))
manifest = json.loads((full/'manifest.json').read_text())
manifest.update(status='本機限定操作驗收候選；完整一致性與公開發布未宣告', input_audit='input-audit/receipt.json', complete_parity=False)
(full/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
public = {k:v for k,v in receipt.items() if k!='images'}
public['images'] = [dict(file=n,sha256=h,local_root=f'dist-all/{version}/full/input-audit') for n,h in images]
(src/'docs/data/ui-traces/opening-input-audit-20260907.json').write_text(json.dumps(public,ensure_ascii=False,indent=2)+'\n')
print(out/'index.html')
print('AppImage SHA-256',sha(app))
