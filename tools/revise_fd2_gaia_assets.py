#!/usr/bin/env python3
"""依規格61編製／登錄已核准的蓋亞造型；只在Docker執行。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image
from build_modern_hd_map_sprites import compile_group
from prepare_modern_hd_map_catalog import compile_parent
from validate_modern_theme_catalog import ROOT, validate
from verify_modern_hd_map_sprites import verify

CATALOG = ROOT/'remake/assets/themes/modern/catalog.json'
APPROVED = 'f5b53ec12ecd73f104a79044985dc383a6bdc0a4c0df0df7daf1c38ef9900463'
CONCEPT_ID = 'modern.gaia.design.style_a.v2'


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def write(path, data):
    assert not path.exists(), f'refusing existing output: {path}'
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2)+'\n')


def names(catalog):
    result = set()
    for asset in catalog['assets']:
        result.update(asset.get('files', []))
        result.update(asset.get('source_files', []))
        for key in ('file', 'master_file'):
            if key in asset:
                result.add(asset[key])
    assert all(Path(name).name == name for name in result)
    return result


def selected(asset):
    return (asset.get('speaker_id') == 30 and asset['role'] in
            {'story_portrait_frame', 'story_portrait_hd_frame'}) or (
            asset.get('source_group') == 30 and asset['role'] in
            {'map_sprite_set', 'map_sprite_hd_set'})


def prepare(args):
    base = validate(True, args.base_catalog)
    assert not any(a['asset_id'] == CONCEPT_ID for a in base['assets']), 'revision already registered; reproduce from the recorded base commit'
    assert digest(args.design) == APPROVED, 'not the approved design'
    assert args.out.parent.stat().st_uid == 1000 and not args.out.exists()
    source = ROOT/base['private_root']
    args.out.mkdir()
    pack, hdout = args.out/'pack', args.out/'hd'
    pack.mkdir(); hdout.mkdir()
    write(args.out/'base-catalog.json', base)
    write(args.out/'before.json', {'catalog_sha256': digest(args.base_catalog),
          'files': {name: digest(source/name) for name in sorted(names(base))}})
    for name in sorted(names(base)):
        (pack/name).symlink_to(os.path.relpath(source/name, pack))
    design_name = 'gaia-original-based-hd-v2-concept.png'
    shutil.copyfile(args.design, pack/design_name)
    portrait_master = 'gaia-portrait-style-a-v2-master.png'
    with Image.open(args.design) as design:
        assert design.size == (1536, 1024)
        portrait = design.convert('RGBA').crop((890, 168, 1518, 796))
    assert portrait.getchannel('A').getextrema() == (255, 255)
    portrait.save(pack/portrait_master, compress_level=9)
    master = 'fdicon-030-style-a-v2-master.png'
    shutil.copyfile(args.walk_master, pack/master)
    parent_out = args.out/'parent'
    parent_out.mkdir()
    parent = compile_parent((30, master, pack, parent_out, list(range(12))))
    for i, old_name in enumerate(parent['files']):
        name = old_name.replace('style-a-f', 'style-a-v2-f')
        (parent_out/old_name).rename(pack/name)
        parent['files'][i] = name
    parent['source_refs'] += ['asset:'+CONCEPT_ID]
    group = compile_group(parent, pack, hdout, master, {})
    for frame in group['frames']:
        old_name = frame['file']
        name = old_name.replace('style-a-hd6', 'style-a-v2-hd6')
        (hdout/old_name).rename(hdout/name)
        frame['file'] = name
        (pack/name).symlink_to(os.path.relpath(hdout/name, pack))
    candidate = copy.deepcopy(base)
    for i, asset in enumerate(candidate['assets']):
        if not selected(asset):
            continue
        if asset['role'] in {'story_portrait_frame', 'story_portrait_hd_frame'}:
            size = asset['width']
            name = f'gaia-portrait-style-a-v2-{size}.png'
            portrait.resize((size, size), Image.Resampling.LANCZOS).save(pack/name, compress_level=9)
            asset.update(file=name, master_file=portrait_master, sha256=digest(pack/name))
            asset['source_refs'] = ['asset:DATO_030_m0', 'asset:'+CONCEPT_ID,
                                    'docs/knowledge-base/61-modern-theme-runtime-spec.md']
        elif asset['role'] == 'map_sprite_set':
            candidate['assets'][i] = parent
        else:
            asset.update(files=[f['file'] for f in group['frames']],
                         frame_sha256=[f['sha256'] for f in group['frames']],
                         master_file=master, frame_overrides={}, source_files=[master],
                         source_sha256=[digest(pack/master)])
            asset['source_refs'] = parent['source_refs']
    candidate['assets'].append({'asset_id': CONCEPT_ID, 'role': 'portrait_concept',
        'status': 'concept', 'file': design_name, 'width': 1536, 'height': 1024,
        'sha256': APPROVED, 'source_refs': ['asset:DATO_030_m0',
            'asset:FDICON.B24.group_030', 'docs/data/fd2-modern-hd-battlefield-20261019.json']})
    candidate['private_root'] = str(pack.relative_to(ROOT))
    write(pack/'catalog.json', candidate)
    validate(True, pack/'catalog.json')
    manifest = {'status': 'CANDIDATE-REVIEW-REQUIRED', 'scale': 6,
        'frame_size': [144, 144], 'alpha_contract': 'coverage',
        'layout_policy': 'contain_22px_cell_feet_v1',
        'catalog_sha256': digest(pack/'catalog.json'),
        'compiler_sha256': digest(ROOT/'tools/build_modern_hd_map_sprites.py'),
        'master_overrides': {'30': master}, 'frame_overrides': {},
        'not_registered_or_runtime': True, 'groups': [group], 'errors': []}
    write(hdout/'hd-map-sprites.json', manifest)
    write(args.out/'data-verification.json', verify(hdout/'hd-map-sprites.json', pack/'catalog.json', pack, True))
    assert [a for a in candidate['assets'] if not selected(a) and a['asset_id'] != CONCEPT_ID] == [a for a in base['assets'] if not selected(a)]
    print('DATA-READY: Gaia 80/480 portrait, 24/144 twelve frames; GUI pending', flush=True)


def register(args):
    candidate = read(args.candidate/'pack/catalog.json')
    before = read(args.candidate/'before.json')
    assert digest(CATALOG) == before['catalog_sha256'], 'formal catalog changed'
    base = validate(True)
    source = ROOT/base['private_root']
    assert source.stat().st_uid == CATALOG.stat().st_uid == 1000
    assert all(digest(source/n) == h for n, h in before['files'].items())
    intro, world = read(args.gui_intro), read(args.gui_world)
    assert intro['status'] in {'PLAYER-E2-LIMITED-GPU-REVIEW-REQUIRED', 'START-PASS-TAIL-INCOMPLETE'} and world['status'] == 'PASS'
    if intro['status'] == 'START-PASS-TAIL-INCOMPLETE':
        assert intro['start_completed'] and intro['tail_failure']
    assert '30' in intro['screenshots'] and intro['screenshots']['30']['mae_normalized'] < .0001
    assert intro['dialogue_beats'] == 97 and intro['final_state']['node'] == 'battle_ch01'
    assert intro['catalog_sha256'] == digest(args.candidate/'pack/catalog.json')
    assert world['catalog_sha256'] == digest(args.candidate/'pack/catalog.json')
    assert world['world_runtime_checked'] and world['new_process_title_load']
    assert world['source_save_bytes_equal'] and world['theme_save_bytes_equal'] and world['loaded_units_equal']
    gaia, gpu = read(args.gui_gaia), read(args.gpu_gaia)
    assert gaia['status'] == gpu['status'] == 'PASS' and gpu['source_group'] == 30
    assert gaia['catalog_sha256'] == gpu['catalog_sha256'] == digest(args.candidate/'pack/catalog.json')
    assert gpu['receipt_sha256'] == digest(args.gui_gaia)
    assert gaia['source_save_bytes_equal'] and gaia['new_process_title_load']
    assert intro['binary_sha256'] == world['binary_sha256'] == gaia['binary_sha256']
    assert read(args.candidate/'data-verification.json')['status'] == 'PASS'
    assert [a for a in candidate['assets'] if not selected(a) and a['asset_id'] != CONCEPT_ID] == [a for a in base['assets'] if not selected(a)]
    for name in sorted(names(candidate)):
        src, dst = args.candidate/'pack'/name, source/name
        if dst.exists():
            assert digest(src) == digest(dst), f'cannot overwrite {name}'
        else:
            shutil.copyfile(src, dst)
    candidate['private_root'] = base['private_root']
    temp = CATALOG.with_suffix('.gaia-candidate.json')
    write(temp, candidate)
    try:
        validate(True, temp)
        os.replace(temp, CATALOG)
    finally:
        temp.unlink(missing_ok=True)
    write(args.candidate/'registered.json', {'status': 'REGISTERED',
        'catalog_sha256': digest(CATALOG), 'gui_intro_sha256': digest(args.gui_intro),
        'gui_world_sha256': digest(args.gui_world), 'approved_design_sha256': APPROVED,
        'gui_gaia_sha256': digest(args.gui_gaia), 'gpu_gaia_sha256': digest(args.gpu_gaia),
        'old_files_unchanged': len(before['files']), 'changed_assets': 4,
        'added_concepts': 1, 'private_remote_synced': False})
    print('REGISTERED: Gaia only; prior files retained')


def main():
    assert Path('/.dockerenv').is_file()
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='mode', required=True)
    prep = sub.add_parser('prepare')
    prep.add_argument('--base-catalog', type=Path, default=CATALOG,
                      help='重生既有修訂時使用記錄的舊清冊；原私人圖源須可取得')
    for name in ('design', 'walk-master', 'out'):
        prep.add_argument('--'+name, type=Path, required=True)
    reg = sub.add_parser('register')
    for name in ('candidate', 'gui-intro', 'gui-world', 'gui-gaia', 'gpu-gaia'):
        reg.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    (prepare if args.mode == 'prepare' else register)(args)


if __name__ == '__main__':
    main()
