#!/usr/bin/env python3
"""編製已定案115%龍人顯示清冊；原始PNG與原版描述子保持。"""
import argparse
import copy
import hashlib
import json
import os
import shutil
from pathlib import Path

from validate_modern_theme_catalog import validate

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--catalog', type=Path, default=ROOT/'remake/assets/themes/modern/catalog.json')
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    args.out = args.out.resolve()
    args.catalog = args.catalog.resolve()
    assert Path('/.dockerenv').exists()
    assert args.out.resolve().is_relative_to(ROOT) and not args.out.exists()
    assert args.out.parent.stat().st_uid == os.getuid()
    base = validate(False, args.catalog)
    characters = json.loads((ROOT/'docs/data/exe_tables/characters.json').read_text())
    dragons = [c for c in characters if c['race'] == 4]
    assert [c['index'] for c in dragons] == [16, 17, 26, 27]
    groups = {c['sprite_group'] for c in dragons}
    resources = {group*3+phase for group in groups for phase in (0, 1)}
    original = ROOT/base['private_root']
    review = ROOT/'work/hd1-feedback-20261106-r91/dragon-scale-review-r2'
    review_data = json.loads((review/'review.json').read_text())
    assert review_data['catalog_sha256'] == sha(args.catalog)
    assert review_data['comparison_sha256'] == sha(review/'dragon-scale-comparison.png')
    args.out.mkdir()
    shutil.copyfile(args.catalog, args.out/'base-catalog.json')
    pack = args.out/'pack'; pack.mkdir()
    names = set()
    for asset in base['assets']:
        names.update(asset.get('files', []))
        names.update(asset.get('source_files', []))
        names.update(asset[k] for k in ('file', 'master_file') if k in asset)
    for name in sorted(names):
        assert Path(name).name == name and (original/name).is_file()
        (pack/name).symlink_to(os.path.relpath(original/name, pack))
    approved = 'hd1-dragon-scale-115-approved-review-v1.png'
    assert approved not in names
    shutil.copyfile(review/'dragon-scale-comparison.png', pack/approved)
    candidate = copy.deepcopy(base)
    candidate['private_root'] = str(pack.relative_to(ROOT))
    selected = []
    for asset in candidate['assets']:
        is_map = asset['role'] == 'map_sprite_hd_set' and asset['source_group'] in groups
        is_battle = asset['role'] == 'battle_animation_hd_set' and asset['source_resource'] in resources
        if not (is_map or is_battle):
            continue
        assert 'visual_scale_percent' not in asset
        asset['visual_scale_percent'] = 115
        selected.append({'asset_id': asset['asset_id'], 'frames': asset['frame_count'],
                         'files': asset['files'], 'sha256': asset['frame_sha256']})
        if is_battle and asset['source_resource'] == 48:
            asset['source_files'].append(approved)
            asset['source_sha256'].append(sha(pack/approved))
    assert len(selected) == 12 and sum(a['frames'] for a in selected) == 115
    catalog = args.out/'catalog.json'
    catalog.write_text(json.dumps(candidate, ensure_ascii=False, indent=2)+'\n')
    validate(True, catalog)
    runtime = args.out/'runtime'
    (runtime/'assets/themes/modern').mkdir(parents=True)
    for source in (ROOT/'remake/assets').iterdir():
        if source.name != 'themes':
            (runtime/'assets'/source.name).symlink_to(os.path.relpath(source, runtime/'assets'))
    (runtime/'assets/themes/modern/catalog.json').symlink_to(os.path.relpath(catalog, runtime/'assets/themes/modern'))
    report = dict(status='PASS-COMPILED-DRAGON-SCALE-CANDIDATE', runtime_registered=False,
                  scale_percent=115, base_catalog_sha256=sha(args.catalog), catalog_sha256=sha(catalog),
                  character_table_sha256=sha(ROOT/'docs/data/exe_tables/characters.json'),
                  tool_sha256=sha(Path(__file__)), assets=selected, map_frames=48, battle_frames=67,
                  source_pixels_changed=False, approved_review_file=approved,
                  approved_review_sha256=sha(pack/approved))
    (args.out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k: report[k] for k in ('status', 'map_frames', 'battle_frames', 'source_pixels_changed')}))


if __name__ == '__main__':
    main()
