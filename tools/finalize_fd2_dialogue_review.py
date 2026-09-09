"""整理本機對話修正版收據；原版圖片只留 dist-all/full 私有交付。"""
import argparse
import hashlib
import json
import os
import pathlib
import shutil

p = argparse.ArgumentParser()
p.add_argument("repo", type=pathlib.Path)
p.add_argument("oracle", type=pathlib.Path)
a = p.parse_args()
w = a.repo / "work/dialogue-cause-20260908"
version = (a.repo / "VERSION").read_text().strip()
full = a.repo / "dist-all" / version / "full"
assert full.stat().st_uid == os.getuid()
manifest = json.loads((full / "manifest.json").read_text())
app = full / f"FD2-{version}-linux-x86_64-full.AppImage"
sha = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
assert sha(app) == manifest["sha256"]
out = full / "dialogue-review"
out.mkdir(exist_ok=True)
assert out.stat().st_uid == os.getuid()
copies = {
    "dosgolem-first.png": a.oracle / "gap-122.png",
    "dosgolem-second.png": a.oracle / "gap-124.png",
    "old-remake-second.png": w / "remake-v4/remake-second-09.png",
    "remake-first.png": w / "remake-v110/remake-first.png",
    "remake-second.png": w / "remake-v110/remake-second.png",
    "remake-speaking.png": w / "remake-v110/first-flow-03.png",
    "inputs.json": w / "remake-v110/remake-inputs.json",
    "state.jsonl": w / "remake-v110/remake-state.jsonl",
    "app.log": w / "remake-v110/remake-app.log",
    "tests.log": w / "remake-tests-v3.log",
}
for name, source in copies.items():
    shutil.copyfile(source, out / name)
receipt = {
    "version": version, "appimage_sha256": sha(app), "normal_start": True,
    "scope": "王宮前兩句正常 START；完整開場至第一關另由快速回歸驗證",
    "full_machine_same_state": False, "audio_verified": False,
    "first": json.loads((w / "v110-first-diff.json").read_text()),
    "second": json.loads((w / "v110-second-diff.json").read_text()),
    "mouth_samples": json.loads((w / "v110-mouth-frames.json").read_text()),
    "test_pass_count": (w / "remake-tests-v3.log").read_text().count("--- PASS:"),
    "files": {name: {"source": str(source), "sha256": sha(source)} for name, source in copies.items()},
    "limitations": "第一句框外482像素未鎖動畫相位；不宣稱全段、時鐘或音訊一致",
}
assert receipt["second"]["different_pixels"] == 0
assert receipt["test_pass_count"] == 11
(out / "receipt.json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
canonical = a.repo / "docs/data/ui-traces/dialogue-remake-fix-20260908.json"
assert canonical.parent.stat().st_uid == os.getuid()
canonical.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
counter = a.repo / "docs/data/ida/fd2_dialogue_counter_20260908.json"
assert counter.parent.stat().st_uid == os.getuid()
shutil.copyfile(w / "ida-dialogue-counter-initial.json", counter)
(out / "index.html").write_text("""<!doctype html><meta charset="utf-8">
<title>FD2 對話修正驗證</title><style>body{background:#161b22;color:#eee;font:18px sans-serif;margin:24px}section{display:flex;gap:12px;flex-wrap:wrap}figure{margin:8px}img{width:480px;max-width:90vw;image-rendering:pixelated}a{color:#8cd}</style>
<h1>王宮對話修正驗證</h1><p>正常 START 路徑。第二句完整 64,000 像素相符；不是完整機器同狀態或全戰役一致聲明。</p>
<section><figure><figcaption>dosgolem 自行執行原版</figcaption><img src="dosgolem-second.png"></figure>
<figure><figcaption>修正前 v.1.0.9：漏了聚焦</figcaption><img src="old-remake-second.png"></figure>
<figure><figcaption>修正後：第二句零差異</figcaption><img src="remake-second.png"></figure></section>
<h2>逐字頭像</h2><p>本圖文字尚未寫完，完整頭像區與 DATO 第 1 格零差異。</p><img src="remake-speaking.png">
<h2>第一句限制</h2><p>框內相符，框外仍有 482 像素差異，未鎖人物動畫相位。</p>
<section><img src="dosgolem-first.png"><img src="remake-first.png"></section>
<p><a href="receipt.json">完整像素與輸入收據</a> · <a href="tests.log">11 項回歸紀錄</a></p>
""", encoding="utf-8")
manifest["status"] = "王宮前兩句正常 START 已驗收；第二句完整畫面零差異，非全段穩定版"
manifest["validation_receipt"] = "dialogue-review/receipt.json"
manifest["validation_receipt_sha256"] = sha(out / "receipt.json")
(full / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
print(out)
