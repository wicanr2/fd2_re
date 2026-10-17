#!/usr/bin/env python3
"""依61的READY序章範圍保存原圖並登錄三個靜態頭像。Docker內執行。"""
import argparse
import hashlib
import io
import json
from pathlib import Path
import PIL
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
TARGETS = ((48, 'king'), (66, 'queen'), (96, 'bandit'))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--sources', type=Path, required=True)
    parser.add_argument('--receipt', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists()
    catalog_path = ROOT / 'remake/assets/themes/modern/catalog.json'
    catalog = json.loads(catalog_path.read_text())
    private = ROOT / catalog['private_root']
    entries, outputs = [], []
    # Validate and prepare every member before publishing the catalog group.
    for speaker, slug in TARGETS:
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
                                 'docs/data/fd2-modern-intro-coverage-20261018.json'],
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
        entries.append({'speaker_id': speaker, 'asset_id': entry['asset_id'],
                        'master_file': master, 'master_size': list(image.size),
                        'master_sha256': hashlib.sha256(raw).hexdigest(),
                        'runtime_file': runtime, 'runtime_sha256': entry['sha256']})
    for target, data in outputs:
        if not target.exists():
            target.write_bytes(data)
    catalog_path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + '\n')
    report = {'schema_version': 1, 'status': 'DATA-READY',
              'scope': '61已核准的序章靜態頭像缺件；未宣稱高解析正式呈現。',
              'user_decision': '2026-10-18採用父王原型手繪風格，先完成序章頭像，再接高解析呈現；忠實模式保留。',
              'pillow_version': PIL.__version__, 'resize': '80x80 Lanczos; RGBA opaque',
              'catalog_sha256': hashlib.sha256(catalog_path.read_bytes()).hexdigest(),
              'assets': entries, 'normal_start_passed': False,
              'private_remote_synced': False, 'public_art_distribution': False}
    args.receipt.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
