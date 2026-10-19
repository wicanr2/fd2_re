#!/usr/bin/env python3
"""依61核准範圍保存母稿並登錄靜態頭像；預設為序章三人。Docker內執行。"""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import PIL
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
TARGETS = ((48, 'king'), (66, 'queen'), (96, 'bandit'))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--sources', type=Path, required=True)
    parser.add_argument('--receipt', type=Path, required=True)
    parser.add_argument('--catalog', type=Path,
                        default=ROOT / 'remake/assets/themes/modern/catalog.json')
    parser.add_argument('--target', action='append', metavar='SPEAKER:SLUG',
                        help='沿既有靜態頭像契約補圖；省略時保留原序章三人')
    parser.add_argument('--hd', action='store_true', help='同批登錄480與80兩角色')
    args = parser.parse_args()
    assert Path('/.dockerenv').exists()
    assert not args.receipt.exists(), '禁止覆寫驗收收據'
    targets = TARGETS
    if args.target:
        targets = tuple((int(t.split(':', 1)[0]), t.split(':', 1)[1]) for t in args.target)
        assert len({speaker for speaker, _ in targets}) == len(targets)
        assert all(0 <= speaker <= 255 and slug and
                   all(ch in 'abcdefghijklmnopqrstuvwxyz0123456789-' for ch in slug)
                   for speaker, slug in targets), '無效角色或檔名'
    catalog_path = args.catalog.resolve()
    catalog = json.loads(catalog_path.read_text())
    private_relative = Path(catalog['private_root'])
    assert not private_relative.is_absolute() and '..' not in private_relative.parts
    private = ROOT / catalog['private_root']
    assert private.is_dir() and private.stat().st_uid == catalog_path.stat().st_uid == os.getuid()
    entries, outputs = [], []
    # Validate and prepare every member before publishing the catalog group.
    for speaker, slug in targets:
        source = args.sources / (slug + '.png')
        raw = source.read_bytes()
        image = Image.open(io.BytesIO(raw))
        assert image.width == image.height and image.width >= 480
        assert image.convert('RGBA').getchannel('A').getextrema() == (255, 255)
        master = f'{slug}-portrait-style-a-v1-master.png'
        runtime = f'{slug}-portrait-style-a-v1-80.png'
        stream = io.BytesIO()
        image.convert('RGBA').resize((80, 80), Image.Resampling.LANCZOS).save(
            stream, format='PNG', compress_level=9)
        candidate = stream.getvalue()
        entry = {'asset_id': f'modern.{slug}.portrait.style_a.frame0',
                 'role': 'story_portrait_frame', 'status': 'runtime_candidate',
                 'file': runtime, 'master_file': master, 'width': 80, 'height': 80,
                 'sha256': hashlib.sha256(candidate).hexdigest(),
                 'source_refs': [f'asset:DATO_{speaker:03d}_m0',
                                 ('docs/knowledge-base/61-modern-theme-runtime-spec.md'
                                  if args.target else
                                  'docs/data/fd2-modern-intro-coverage-20261018.json')],
                 'consumer_contract': 'native_story_dialogue_rgba_overlay_v1',
                 'speaker_id': speaker, 'frame': 0, 'mouth_state': 'closed'}
        existing = [a for a in catalog['assets'] if a.get('speaker_id') == speaker
                    and a['role'] == 'story_portrait_frame']
        assert not existing or existing == [entry], f'conflicting speaker {speaker}'
        for name, data in ((master, raw), (runtime, candidate)):
            target = private / name
            assert not target.exists() or target.read_bytes() == data, name
            outputs.append((target, data))
        if not existing:
            catalog['assets'].append(entry)
        if args.hd:
            hd_name = f'{slug}-portrait-style-a-v1-480.png'
            hd_stream = io.BytesIO()
            image.convert('RGBA').resize((480, 480), Image.Resampling.LANCZOS).save(
                hd_stream, format='PNG', compress_level=9)
            hd_raw = hd_stream.getvalue()
            hd_entry = dict(entry, asset_id=entry['asset_id'].removesuffix('.frame0')+'.hd6.frame0',
                            role='story_portrait_hd_frame', width=480, height=480,
                            file=hd_name, sha256=hashlib.sha256(hd_raw).hexdigest(),
                            consumer_contract='native_story_dialogue_rgba_overlay_hd6_v1',
                            source_refs=list(dict.fromkeys(entry['source_refs']+[
                                'docs/knowledge-base/61-modern-theme-runtime-spec.md'])))
            existing_hd = [a for a in catalog['assets'] if a.get('speaker_id') == speaker
                           and a['role'] == 'story_portrait_hd_frame']
            assert not existing_hd or existing_hd == [hd_entry], f'conflicting HD speaker {speaker}'
            hd_target = private / hd_name
            assert not hd_target.exists() or hd_target.read_bytes() == hd_raw, hd_name
            outputs.append((hd_target, hd_raw))
            if not existing_hd:
                catalog['assets'].append(hd_entry)
        entries.append({'speaker_id': speaker, 'asset_id': entry['asset_id'],
                        'master_file': master, 'master_size': list(image.size),
                        'master_sha256': hashlib.sha256(raw).hexdigest(),
                        'runtime_file': runtime, 'runtime_sha256': entry['sha256']})
        if args.hd:
            entries[-1].update(hd_file=hd_name, hd_sha256=hd_entry['sha256'])
    for target, data in outputs:
        if not target.exists():
            target.write_bytes(data)
    catalog_path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + '\n')
    report = {'schema_version': 1, 'status': 'DATA-READY',
              'scope': '61已核准的靜態頭像補圖；正式呈現另須正常玩家驗證。',
              'hd': args.hd,
              'user_decision': '2026-10-18採用父王原型手繪風格，先完成序章頭像，再接高解析呈現；忠實模式保留。',
              'pillow_version': PIL.__version__,
              'resize': ('80x80 and 480x480 Lanczos; RGBA opaque' if args.hd
                         else '80x80 Lanczos; RGBA opaque'),
              'catalog_sha256': hashlib.sha256(catalog_path.read_bytes()).hexdigest(),
              'assets': entries, 'normal_start_passed': False,
              'private_remote_synced': False, 'public_art_distribution': False}
    args.receipt.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
