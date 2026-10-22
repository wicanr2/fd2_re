#!/usr/bin/env python3
"""核對FIGANI受擊閃色33沿同份來源alpha，未知填色保持拒收。"""
import argparse
import copy
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_compound_caster_gpu import make_native_occlusion_test


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['catalog', 'source', 'original-pack', 'inventory', 'report']:
        p.add_argument('--'+name, required=True, type=Path)
    a = p.parse_args()
    if a.report.exists():
        raise SystemExit('不覆寫舊收據')
    catalog = validate(False, a.catalog)
    entries = {v['source_resource']: v for v in catalog['assets'] if v['role'] == 'battle_animation_hd_set'}
    inventory = json.loads(a.inventory.read_text())
    asset = entries[0]
    frame = asset['native_frames'][0]
    layer = dict(archive='FIGANI.DAT', resource=0, frame=0, fill=33,
                 x=frame['x'], y=frame['y'], width=frame['width'], height=frame['height'], clip=None)
    checks = []

    def verify_mask(entry_map, mask, scale, fill, name):
        values = list(mask.getdata())
        opaque = next(i for i, value in enumerate(values) if value)
        clear = next(i for i, value in enumerate(values) if not value)
        test = make_native_occlusion_test(entry_map, a.source, a.original_pack, inventory)
        probe = dict(layer, fill=fill)
        for index, want in [(opaque, True), (clear, False)]:
            x = frame['x']*6 + (index % mask.width)*scale
            y = frame['y']*6 + (index // mask.width)*scale
            assert test(x, y, [probe]) is want, name
        checks.append(dict(name=name, opaque_and_clear_checked=True))

    with Image.open(a.source/asset['files'][0]) as im:
        hd_mask = im.getchannel('A').copy()
    verify_mask(entries, hd_mask, 1, -1, 'HD來源原色')
    verify_mask(entries, hd_mask, 1, 33, 'HD來源閃色33沿原alpha')
    metadata = a.original_pack/'animations/FIGANI_000/animation.json'
    native_frame = json.loads(metadata.read_text())['frames'][0]
    with Image.open(metadata.parent/native_frame['mask_path']) as im:
        native_mask = im.convert('L')
    verify_mask({}, native_mask, 6, 33, '原生fallback閃色33沿固定Mask')

    def reject(entry_map, proof, probe, name):
        test = make_native_occlusion_test(entry_map, a.source, a.original_pack, proof)
        try:
            test(probe['x']*6, probe['y']*6, [probe])
        except SystemExit:
            checks.append(dict(name=name, rejected=True))
        else:
            raise AssertionError('應拒收：'+name)

    for fill in [32, 34]:
        reject(entries, inventory, dict(layer, fill=fill), '未知填色'+str(fill))
    reject(entries, inventory, dict(layer, archive='FDOTHER.DAT'), '特效不可冒用人物閃色')
    bad = copy.deepcopy(entries)
    bad[0]['frame_sha256'][0] = '0'*64
    reject(bad, inventory, layer, 'HD來源SHA不符')
    bad = copy.deepcopy(inventory)
    bad['resources'][0]['metadata_sha256'] = '0'*64
    reject({}, bad, layer, '原生metadataSHA不符')
    report = dict(status='PASS-FIGANI-FILL33-OCCLUSION-GUARDS', checks=checks,
                  catalog_sha256=digest(a.catalog), inventory_sha256=digest(a.inventory),
                  original_manifest_sha256=digest(a.original_pack/'manifest.json'),
                  helper_sha256=digest(Path(__file__).with_name('verify_fd2_compound_caster_gpu.py')),
                  scope='只核固定來源alpha／Mask，填色33沿既有GPU RGB分支。未知填色及錯SHA拒收，不改遊戲或核心門檻。')
    a.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=report['status'], checks=len(checks))))


if __name__ == '__main__':
    main()
