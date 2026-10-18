#!/usr/bin/env python3
"""在Docker建立完整HD人物候選包；通過實檔世界候選後才登錄正式清冊。"""
import argparse
from concurrent.futures import ProcessPoolExecutor
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image

from build_modern_hd_map_sprites import master_components
from validate_modern_theme_catalog import ROOT, VISIBLE_FDICON_GROUPS, validate
from verify_modern_hd_map_sprites import verify


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def write_new(path, value):
    assert not path.exists() and path.parent.stat().st_uid == 1000
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2)+'\n')


def safe_name(name):
    assert isinstance(name, str) and Path(name).name == name and Path(name).suffix == '.png'
    return name


def compile_parent(job):
    group, master, source, output, canonical = job
    with Image.open(source/master) as image:
        components = master_components(image, 12)
    files, hashes = [], []
    for index, (sprite, _) in enumerate(components):
        name = f'fdicon-{group:03d}-style-a-f{index:02d}.png'
        path = output/name
        if canonical[index] != index:
            shutil.copyfile(output/files[canonical[index]], path)
        else:
            scale = min(22/sprite.width, 22/sprite.height)
            size = max(1, round(sprite.width*scale)), max(1, round(sprite.height*scale))
            sprite = sprite.resize(size, Image.Resampling.NEAREST)
            sprite.putalpha(sprite.getchannel('A').point(lambda value: 255 if value >= 128 else 0))
            frame = Image.new('RGBA', (24, 24))
            frame.alpha_composite(sprite, ((24-size[0])//2, 23-size[1]))
            assert frame.getbbox() and frame.getchannel('A').getextrema() == (0, 255)
            assert not path.exists()
            frame.save(path, optimize=False)
        files.append(name)
        hashes.append(digest(path))
    for i in range(12):
        for j in range(i):
            assert (hashes[i] == hashes[j]) == (canonical[i] == canonical[j]), f'group{group}: 24px phase collapse'
    return {'asset_id': f'modern.fdicon.group_{group:03d}.style_a', 'role': 'map_sprite_set',
            'status': 'runtime_candidate', 'master_file': master, 'files': files, 'width': 24, 'height': 24,
            'frame_count': 12, 'frame_sha256': hashes, 'source_group': group,
            'consumer_contract': 'fdicon_map_sprite_12x24_v1', 'alpha_contract': 'binary',
            'cycle_policy': 'source_exact_repeats' if len(set(canonical)) != 12 else 'three_distinct_cycles',
            'source_refs': [f'asset:FDICON.B24.group_{group:03d}', 'docs/data/fd2-modern-hd-battlefield-20261019.json']}


def prepare(args):
    base = validate(True, args.catalog)
    prior, audit, new_masters = read(args.prior_manifest), read(args.native_audit), read(args.new_masters)
    assert prior['catalog_sha256'] == digest(args.catalog)
    assert audit['original']['sha256'] == '7efb4448d05f19c1e17ebd53f3e3afead235f5c008d5167548d834c3686b1e44'
    groups = {g['selector']: g for g in audit['groups']}
    assert {group for group, record in groups.items() if record['visible_frames']} == VISIBLE_FDICON_GROUPS
    parents = {a['source_group']: a for a in base['assets'] if a['role'] == 'map_sprite_set'}
    assert {int(g) for g in new_masters} == VISIBLE_FDICON_GROUPS-set(parents)
    assert not args.output.exists() and args.output.parent.stat().st_uid == 1000
    # Preflight every registered and selected source before writing the pack.
    needed = {}
    for asset in base['assets']:
        for name in ([asset['file']] if 'file' in asset else asset.get('files', [])):
            needed[safe_name(name)] = None
        if asset.get('master_file'):
            needed[safe_name(asset['master_file'])] = None
    masters, overrides = {}, {}
    for group in prior['groups']:
        key = str(group['source_group'])
        masters[key] = group['master_file']
        needed[safe_name(group['master_file'])] = group['master_sha256']
        for index, frame in enumerate(group['frames']):
            needed[safe_name(frame['source_file'])] = frame['source_sha256']
            if frame['source_file'] != group['master_file'] and 'repeat_of' not in frame:
                overrides.setdefault(key, {})[str(index)] = frame['source_file']
    for key, master in new_masters.items():
        needed[safe_name(master)] = None
        masters[key] = master
    for name, expected in needed.items():
        path = args.source/name
        assert path.is_file() and (expected is None or digest(path) == expected), name
    args.output.mkdir()
    for name in needed:
        # Relative links work on both the host and the /repo Docker mount.
        (args.output/name).symlink_to(os.path.relpath(args.source/name, args.output))
    jobs = []
    for key, master in new_masters.items():
        group = int(key)
        native = groups[group]
        assert native['visible_frames'] == list(range(12))
        hashes = [frame['decoded_sha256'] for frame in native['frames']]
        jobs.append((group, master, args.source, args.output, [hashes.index(h) for h in hashes]))
    with ProcessPoolExecutor(max_workers=args.workers) as executor:
        new_assets = list(executor.map(compile_parent, jobs))
    base['assets'].extend(sorted(new_assets, key=lambda a:a['source_group']))
    base['private_root'] = str(args.output.relative_to(ROOT))
    write_new(args.output/'catalog-parent.json', base)
    write_new(args.output/'master-overrides.json', masters)
    write_new(args.output/'frame-overrides.json', overrides)
    write_new(args.output/'preparation.json', {'base_catalog_sha256': digest(args.catalog),
              'native_audit_sha256': digest(args.native_audit), 'prior_manifest_sha256': digest(args.prior_manifest),
              'tool_sha256': digest(Path(__file__)), 'new_groups': sorted(int(g) for g in new_masters)})
    validate(True, args.output/'catalog-parent.json')
    print(f'prepared complete native parents: {len(parents)+len(new_assets)} groups')


def register(args):
    parent = validate(True, args.parent_catalog)
    verification = verify(args.manifest, args.parent_catalog, args.source)
    assert verification['native_bank_coverage_verified']
    receipt = read(args.world_receipt)
    assert receipt['status'] == 'PASS' and receipt['manifest_sha256'] == digest(args.manifest)
    assert receipt['groups'] == len(VISIBLE_FDICON_GROUPS) and receipt['frames'] == 1455
    assert receipt['original_save_unchanged'] is True
    preparation = read(args.parent_catalog.parent/'preparation.json')
    art = read(args.art_review)
    assert art['status'] == 'PASS-LIMITED-ART-REVIEW'
    assert art['manifest_sha256'] == digest(args.manifest)
    assert art['groups'] == len(preparation['new_groups'])
    assert digest(args.catalog) == preparation['base_catalog_sha256'], 'formal catalog changed during review'
    assert args.catalog.stat().st_uid == 1000 and args.catalog.parent.stat().st_uid == 1000
    assert args.destination.is_dir() and args.destination.stat().st_uid == 1000
    base = read(args.catalog)
    manifest = read(args.manifest)
    complete = copy.deepcopy(parent)
    complete['private_root'] = str(args.destination.relative_to(ROOT))
    old_parents = {a['source_group']: a for a in parent['assets'] if a['role'] == 'map_sprite_set'}
    copies = {}
    for asset in parent['assets']:
        for name in ([asset['file']] if 'file' in asset else asset.get('files', [])):
            copies[safe_name(name)] = args.source/name
        if asset.get('master_file'):
            copies[safe_name(asset['master_file'])] = args.source/asset['master_file']
    for group in manifest['groups']:
        source_group = group['source_group']
        frames = group['frames']
        overrides = {str(i): f['source_file'] for i, f in enumerate(frames)
                     if f['source_file'] != group['master_file'] and 'repeat_of' not in f}
        sources = sorted({group['master_file'], *overrides.values()})
        contract = 'fdicon_hd_static_sprite_3x144_v1' if group['frame_count'] == 3 else 'fdicon_hd_map_sprite_12x144_v1'
        asset = {'asset_id': old_parents[source_group]['asset_id']+'.hd6', 'role': 'map_sprite_hd_set',
                 'status': 'runtime_candidate', 'files': [f['file'] for f in frames], 'width': 144, 'height': 144,
                 'frame_count': group['frame_count'], 'frame_sha256': [f['sha256'] for f in frames],
                 'source_group': source_group, 'consumer_contract': contract, 'alpha_contract': 'coverage',
                 'cycle_policy': group['cycle_policy'], 'source_refs': old_parents[source_group]['source_refs'],
                 'layout_policy': manifest['layout_policy'], 'master_file': group['master_file'],
                 'frame_overrides': overrides, 'source_files': sources,
                 'source_sha256': [digest(args.source/name) for name in sources]}
        complete['assets'].append(asset)
        for frame in frames:
            copies[safe_name(frame['file'])] = args.manifest.parent/frame['file']
        for name in sources:
            copies[safe_name(name)] = args.source/name
    for name, source in copies.items():
        target = args.destination/name
        if target.exists():
            assert digest(target) == digest(source), f'refusing to overwrite a different asset: {name}'
    assert [a for a in complete['assets'] if a in base['assets']] == base['assets'], 'existing asset metadata changed'
    for name, source in copies.items():
        target = args.destination/name
        if not target.exists():
            shutil.copyfile(source, target)
    temporary = args.catalog.with_suffix('.hd-candidate.json')
    write_new(temporary, complete)
    try:
        validate(True, temporary)
        os.replace(temporary, args.catalog)
    finally:
        temporary.unlink(missing_ok=True)
    print(f'registered complete native HD sources: {len(manifest["groups"])} groups')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    prep = sub.add_parser('prepare')
    for name in ('catalog', 'prior-manifest', 'native-audit', 'new-masters', 'source', 'output'):
        prep.add_argument('--'+name, type=Path, required=True)
    prep.add_argument('--workers', type=int, choices=range(1, 5), default=1)
    reg = sub.add_parser('register')
    for name in ('catalog', 'parent-catalog', 'manifest', 'world-receipt', 'art-review', 'source', 'destination'):
        reg.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    (prepare if args.mode == 'prepare' else register)(args)


if __name__ == '__main__':
    main()
