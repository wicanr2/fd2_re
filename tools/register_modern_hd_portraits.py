#!/usr/bin/env python3
"""依規格61將已登錄母稿衍生為480×480頭像；Docker內執行。"""
import argparse
import hashlib
import io
import json
from pathlib import Path

import PIL
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists()
    catalog_path = ROOT / 'remake/assets/themes/modern/catalog.json'
    catalog = json.loads(catalog_path.read_text())
    private = ROOT / catalog['private_root']
    assert private.is_dir() and private.stat().st_uid == catalog_path.stat().st_uid
    base = [a for a in catalog['assets'] if a['role'] == 'story_portrait_frame']
    assert base and len({a['speaker_id'] for a in base}) == len(base)
    hd_entries, outputs, evidence = [], [], []
    for asset in base:
        name = asset['master_file']
        assert Path(name).name == name
        raw = (private / name).read_bytes()
        image = Image.open(io.BytesIO(raw)).convert('RGBA')
        assert min(image.size) >= 480 and image.width == image.height, name
        assert image.getchannel('A').getextrema() == (255, 255), name
        assert asset['frame'] == 0 and asset['mouth_state'] == 'closed'
        runtime = Path(name).stem.removesuffix('-master') + '-480.png'
        stream = io.BytesIO()
        image.resize((480, 480), Image.Resampling.LANCZOS).save(
            stream, format='PNG', compress_level=9)
        output = stream.getvalue()
        entry = dict(asset, asset_id=asset['asset_id'].removesuffix('.frame0')+'.hd6.frame0',
                     role='story_portrait_hd_frame', width=480, height=480,
                     file=runtime, sha256=hashlib.sha256(output).hexdigest(),
                     consumer_contract='native_story_dialogue_rgba_overlay_hd6_v1',
                     source_refs=asset['source_refs']+['docs/knowledge-base/61-modern-theme-runtime-spec.md'])
        hd_entries.append(entry)
        target = private / runtime
        assert not target.exists() or target.read_bytes() == output, runtime
        outputs.append((target, output))
        evidence.append({'speaker_id': asset['speaker_id'], 'master_file': name,
                         'master_size': list(image.size),
                         'master_sha256': hashlib.sha256(raw).hexdigest(),
                         'runtime_file': runtime, 'runtime_sha256': entry['sha256']})
    old = [a for a in catalog['assets'] if a['role'] == 'story_portrait_hd_frame']
    assert not old or old == hd_entries, '已有不同的高解析清冊，禁止覆寫'
    for target, raw in outputs:
        if not target.exists():
            target.write_bytes(raw)
    if not old:
        catalog['assets'].extend(hd_entries)
        catalog_path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    report = {'schema_version': 1, 'status': 'DATA-READY',
              'user_decision': '2026-10-18撤回160方案，確認1920×1200、480×480頭像，維持版面比例。',
              'scope': '高解析故事頭像與最終呈現層；背景、字型及戰場尚未HD化。',
              'pillow_version': PIL.__version__, 'resize': '480x480 Lanczos from original masters',
              'canvas': [1920, 1200], 'game_coordinates': [640, 400],
              'catalog_sha256': hashlib.sha256(catalog_path.read_bytes()).hexdigest(),
              'assets': evidence, 'normal_start_passed': False,
              'private_remote_synced': False, 'public_art_distribution': False}
    assert not args.receipt.exists(), '禁止覆寫驗收收據'
    args.receipt.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(f"DATA-READY: {len(hd_entries)} HD portraits, 480x480; production validation pending")


if __name__ == '__main__':
    main()
