#!/usr/bin/env python3
"""獨立核對地表刷新範圍，並以遺漏引用及改號候選確認拒收。"""
import argparse
import copy
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageFilter

from verify_fd2_terrain_sources import sha
from build_fd2_hd_terrain_reuse import native


def check(base, candidate, recipe, report, pack):
    assert len(base['assets']) == recipe['base_assets']
    assert len(candidate['assets']) == len(base['assets']) + 7
    expected = {a['asset_id']: a for a in base['assets']}
    got = {a['asset_id']: a for a in candidate['assets']}
    assert len(got) == len(candidate['assets'])
    assert set(got) - set(expected) == {s['asset_id'] for s in [recipe['style'], *recipe['alpha_sources']]}
    replacements = {f['old_file']: f for f in report['frames']}
    assert len(replacements) == len(report['frames']) == 165 + len(recipe['embedded_grass_groups']) == 202
    masters = {m['asset_id']: m for m in recipe['materials'].values()}
    targets = set()
    changed = []
    for identity, old in expected.items():
        new = got[identity]
        allowed = copy.deepcopy(old)
        if identity in masters:
            m = masters[identity]
            allowed.update(file=m['file'], sha256=m['sha256'], width=m['size'][0], height=m['size'][1])
        if old['role'] == 'native_terrain_hd_set':
            for index, filename in enumerate(old['files']):
                if filename in replacements:
                    row = replacements[filename]
                    targets.add((old['map_id'], old['frame_ids'][index]))
                    allowed['files'][index], allowed['frame_sha256'][index] = row['file'], row['sha256']
        if allowed != old:
            allowed['source_refs'] = list(dict.fromkeys([*old['source_refs'], 'remake/assets/themes/modern/hd1-pixel-terrain-refresh.recipe.json']))
            changed.append(identity)
        assert new == allowed, identity
    assert set(changed) == set(report['changed_assets'])
    assert targets == {(b['map'], b['frame']) for b in report['bindings']} and len(targets) == 911
    assert len({m for m, f in targets}) == 25
    for row in report['frames']:
        path = pack / row['file']
        assert sha(path) == row['sha256']
        with Image.open(path) as im:
            assert im.size == (144, 144) and im.convert('RGBA').getextrema()[3] == (255, 255)
    for m in [*recipe['materials'].values(), recipe['style'], *recipe['alpha_sources']]:
        assert sha(pack / m['file']) == m['sha256']
    return dict(changed_assets=len(changed), untouched_assets=len(base['assets']) - len(changed),
                runtime_pngs=len(replacements), maps=25, slots=len(targets))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('base', 'recipe', 'candidate', 'report', 'original-pack', 'sources', 'base-pack'):
        parser.add_argument('--' + name, type=Path, required=True)
    a = parser.parse_args()
    assert Path('/.dockerenv').exists()
    load = lambda p: json.loads(p.read_text())
    base, recipe, catalog, report = load(a.base), load(a.recipe), load(a.candidate / 'catalog.json'), load(a.candidate / 'manifest.json')
    result = check(base, catalog, recipe, report, a.candidate / 'pack')
    guards = []
    for label in ('missed-reference', 'renumbered-frame', 'unrelated-battle-art'):
        bad = copy.deepcopy(catalog)
        by_id = {asset['asset_id']: asset for asset in bad['assets']}
        old = next(asset for asset in base['assets'] if asset['asset_id'] in report['changed_assets'] and asset['role'] == 'native_terrain_hd_set')
        if label == 'missed-reference':
            by_id[old['asset_id']] = old
            bad['assets'] = [by_id[asset['asset_id']] for asset in bad['assets']]
        elif label == 'renumbered-frame':
            by_id[old['asset_id']]['frame_ids'][0] += 1
        else:
            next(asset for asset in bad['assets'] if asset['role'] == 'battle_animation_hd_set')['width'] += 1
        try:
            check(base, bad, recipe, report, a.candidate / 'pack')
        except AssertionError:
            guards.append(label)
        else:
            raise AssertionError('錯誤候選未拒收：' + label)
    # HD artwork defines the object silhouette; original palette indices only
    # define background shading. Check every fully covered painted pixel.
    source = load(a.sources / 'manifest.json')
    replacements = {f['old_file']: f['file'] for f in report['frames']}
    kept_pixels = 0
    changed_background = 0
    for group in recipe['embedded_grass_groups']:
        content = native(a.original_pack, source, *group['anchor'])
        assert len(content) == 1728 and all(content[576:1152]) and not any(content[1152:])
        mask_path = a.candidate / 'masks' / group['file']
        row = next(f for f in report['frames'] if f['old_file'] == group['file'])
        assert sha(mask_path) == row['painted_alpha_sha256']
        object_core = Image.open(mask_path).convert('L').point(lambda v: 255 if v == 255 else 0)
        old = Image.open(a.base_pack / group['file']).convert('RGBA')
        new = Image.open(a.candidate / 'pack' / replacements[group['file']]).convert('RGBA')
        delta = ImageChops.difference(old, new)
        red, green, blue, _ = delta.split()
        changed = ImageChops.lighter(ImageChops.lighter(red, green), blue).point(lambda v: 255 if v else 0)
        assert ImageChops.multiply(changed, object_core).getbbox() is None, group['file']
        kept_pixels += sum(v == 255 for v in object_core.getdata())
        changed_background += sum(v != 0 for v in changed.getdata())
    assert kept_pixels > 0 and changed_background > 0
    result.update(status='PASS', rejected=guards, catalog_sha256=sha(a.candidate / 'catalog.json'),
                  tool_sha256=sha(Path(__file__)), preserved_object_core_pixels=kept_pixels,
                  refreshed_background_pixels=changed_background)
    a.report.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
