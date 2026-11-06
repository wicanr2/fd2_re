#!/usr/bin/env python3
"""以原始遮罩、未改PNG與實際GPU反例驗證115%龍人清冊。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import sys

from PIL import Image
from build_fd2_character_battle_hd import load_native

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['base-catalog', 'catalog', 'modern', 'original-pack', 'lifecycle-root', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    args = p.parse_args()
    assert Path('/.dockerenv').exists() and not args.out.exists()
    base = json.loads(args.base_catalog.read_text())
    catalog = json.loads(args.catalog.read_text())
    old = {a['asset_id']: a for a in base['assets']}
    assert len(old) == len(catalog['assets'])
    maps, empty, visible, delta = [], [], 0, []
    inventory = json.loads((ROOT/'docs/data/fd2-battle-hd-source-inventory-20261106.json').read_text())
    for asset in catalog['assets']:
        previous = old[asset['asset_id']]
        if asset.get('visual_scale_percent') != 115:
            assert asset == previous
            continue
        delta.append(asset['asset_id'])
        for field in ['files', 'frame_sha256']:
            assert asset[field] == previous[field]
        for name, digest in zip(asset['files'], asset['frame_sha256']):
            assert sha(args.modern/name) == digest
        if asset['role'] == 'map_sprite_hd_set':
            assert asset['source_group'] in [16, 17, 26, 27]
            parent = next(a for a in catalog['assets'] if a['role'] == 'map_sprite_set' and a['source_group'] == asset['source_group'])
            for i, name in enumerate(asset['files']):
                with Image.open(args.modern/parent['files'][i]) as im:
                    foot = im.getchannel('A').getbbox()[3]*6
                with Image.open(args.modern/name) as im:
                    assert im.size == (144, 144)
                top = round(foot-foot*166/144)
                assert abs(top+foot*166/144-foot) <= .5
                maps.append(dict(group=asset['source_group'], frame=i, foot=foot, top=top, size=166))
        else:
            resource = asset['source_resource']
            assert resource in [48, 49, 51, 52, 78, 79, 81, 82]
            assert asset['native_frames'] == previous['native_frames']
            _, originals = load_native(args.original_pack, inventory, resource)
            for i, (name, native) in enumerate(zip(asset['files'], originals)):
                with Image.open(args.modern/name) as im:
                    transparent = im.getchannel('A').getbbox() is None
                assert transparent == (native.getchannel('A').getbbox() is None)
                if transparent:
                    empty.append([resource, i])
                else:
                    visible += 1
    assert len(delta) == 12 and len(maps) == 48 and visible == 63
    assert empty == [[49, 6], [79, 5], [82, 5], [82, 11]]
    args.out.mkdir()
    receipt_path = args.lifecycle_root/'lifecycle-17-r2/receipt.json'
    original_receipt = json.loads(receipt_path.read_text())
    guards = []
    for kind in ['wrong-scale', 'wrong-caller-position']:
        case = args.out/kind; case.mkdir()
        receipt = copy.deepcopy(original_receipt)
        for shot in receipt['source_shots'].values():
            name = shot['file']; assert Path(name).name == name
            target = case/name
            if not target.exists():
                target.symlink_to((receipt_path.parent/name).resolve())
            if kind == 'wrong-caller-position':
                for state in [shot['before'], shot['after']]:
                    for layer in state['battle_source_layers']:
                        if layer['resource'] in (51, 52):
                            layer['x'] += 1
        check_catalog = args.catalog
        if kind == 'wrong-scale':
            altered = copy.deepcopy(catalog)
            for asset in altered['assets']:
                if asset['role'] == 'battle_animation_hd_set' and asset['source_resource'] in (51, 52):
                    asset['visual_scale_percent'] = 100
            check_catalog = case/'catalog.json'
            check_catalog.write_text(json.dumps(altered, ensure_ascii=False, indent=2)+'\n')
            receipt['catalog_sha256'] = sha(check_catalog)
        path = case/'receipt.json'; path.write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
        report = case/'report.json'
        command = [sys.executable, str(ROOT/'tools/verify_fd2_character_battle_gpu.py'), '--figure', '17',
                   '--core-profile', 'opaque', '--sample-group', 'fixture', '--receipt', str(path),
                   '--catalog', str(check_catalog), '--source', str(args.modern), '--report', str(report)]
        result = subprocess.run(command, text=True, capture_output=True, timeout=90)
        reason = (result.stdout+result.stderr).strip()
        assert result.returncode != 0 and not report.exists() and '未核對到實際人物GPU圖源' in reason, (kind, reason)
        guards.append(dict(case=kind, status='PASS-REJECTED-BY-PIXELS', reason=reason, command=command))
    proof = dict(status='PASS-DRAGON-SCALE-SOURCES-AND-GPU-GUARDS', catalog_sha256=sha(args.catalog),
                 tool_sha256=sha(Path(__file__)), base_catalog_sha256=sha(args.base_catalog),
                 changed_assets=delta, map_frames=maps, visible_battle_frames=visible,
                 original_transparent_frames=empty, guards=guards)
    (args.out/'proof.json').write_text(json.dumps(proof, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=proof['status'], map_frames=len(maps), visible_battle_frames=visible, transparent_frames=len(empty), pixel_guards=len(guards))))


if __name__ == '__main__':
    main()
