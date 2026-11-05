#!/usr/bin/env python3
"""依本人原版頭像補齊或明示替換伙伴80／480配對候選。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image
from build_fd2_gaia_battle_hd import names
from validate_modern_theme_catalog import ROOT, validate


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ['recipe', 'catalog', 'original', 'out']:
        parser.add_argument('--' + key, type=Path, required=True)
    parser.add_argument('--replace-existing', default='', help='明示允許替換的speaker ID，以逗號分隔')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    args.out = args.out.resolve()
    assert args.out.is_relative_to(ROOT)
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    base = validate(False, args.catalog)
    recipe = json.loads(args.recipe.read_text())
    assert recipe['schema_version'] == 1
    requested = {r['speaker_id'] for r in recipe['portraits']}
    assert requested and len(requested) == len(recipe['portraits'])
    assert all(type(i) is int and 0 <= i < 32 for i in requested)
    replacements = {int(i) for i in args.replace_existing.split(',') if i}
    occupied = {a['speaker_id'] for a in base['assets'] if a['role'] in ['story_portrait_frame', 'story_portrait_hd_frame']}
    assert replacements == requested.intersection(occupied), '已有頭像，須逐人明示替代範圍'
    args.out.mkdir()
    pack = args.out / 'pack'
    pack.mkdir()
    old = ROOT / base['private_root']
    for name in sorted(names(base)):
        assert (old / name).is_file(), name
        (pack / name).symlink_to(os.path.relpath(old / name, pack))
    candidate = copy.deepcopy(base)
    candidate['private_root'] = str(pack.relative_to(ROOT))
    candidate['assets'] = [a for a in candidate['assets'] if not (
        a['role'] in ['story_portrait_frame', 'story_portrait_hd_frame'] and a['speaker_id'] in replacements)]
    records = []
    for row in recipe['portraits']:
        speaker = row['speaker_id']
        master = ROOT / row['master']
        assert sha(master) == row['sha256'] and master.stat().st_uid == os.getuid()
        original = args.original / 'portraits' / f'DATO_{speaker:03}_m0.png'
        assert sha(original) == row['original_sha256']
        with Image.open(master) as source:
            assert source.width == source.height and source.width >= 480
            image = source.convert('RGBA')
        assert image.getchannel('A').getextrema() == (255, 255)
        revision = row.get('revision', 1)
        assert type(revision) is int and 1 <= revision <= 999
        stem = f'party-{speaker:03}-portrait-style-a-v{revision}'
        master_name = stem + '-master.png'
        assert not (pack / master_name).exists()
        shutil.copyfile(master, pack / master_name)
        for size in [80, 480]:
            filename = stem + f'-{size}.png'
            image.resize((size, size), Image.Resampling.LANCZOS).save(pack / filename, compress_level=9)
            entry = dict(asset_id=f'modern.party{speaker:03}.portrait.style_a.' + ('hd6.' if size == 480 else '') + 'frame0',
                role='story_portrait_hd_frame' if size == 480 else 'story_portrait_frame',
                status='runtime_candidate', file=filename, master_file=master_name,
                width=size, height=size, sha256=sha(pack / filename),
                source_refs=[f'asset:DATO_{speaker:03}_m0'],
                consumer_contract='native_story_dialogue_rgba_overlay_hd6_v1' if size == 480 else 'native_story_dialogue_rgba_overlay_v1',
                speaker_id=speaker, frame=0, mouth_state='closed')
            candidate['assets'].append(entry)
        records.append(dict(row, master_file=master_name, outputs=[stem + '-80.png', stem + '-480.png']))
    output = args.out / 'catalog.json'
    output.write_text(json.dumps(candidate, ensure_ascii=False, indent=2) + '\n')
    validate(True, output)
    hd = {a['speaker_id'] for a in candidate['assets'] if a['role'] == 'story_portrait_hd_frame'}
    assert set(range(32)).issubset(hd)
    report = dict(status='PASS-PAIRED-PORTRAIT-CANDIDATE', runtime_registered=False,
        base_catalog_sha256=sha(args.catalog), catalog_sha256=sha(output),
        producer_sha256=sha(__file__), recipe_sha256=sha(args.recipe),
        all_32_companion_hd_portraits=True, replaced_speakers=sorted(replacements), portraits=records)
    (args.out / 'portrait-receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'status': report['status'], 'all_32_companion_hd_portraits': True}))


if __name__ == '__main__':
    main()
