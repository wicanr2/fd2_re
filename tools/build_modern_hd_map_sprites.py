#!/usr/bin/env python3
"""從既有手繪母稿編製 HD FDICON 幀；不放大 24×24 成品，不改 catalog。"""
import argparse
import hashlib
import json
import sys
from pathlib import Path

from PIL import Image
from process_modern_map_sprite_sheet import transparent_background


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def master_components(image, count):
    """以完整輪廓找來源格，避免非等距母稿被固定3×4切掉頭或武器。"""
    if 'A' in image.getbands() and image.getchannel('A').getextrema()[0] == 0:
        rgba = image.convert('RGBA')
    else:
        rgba = transparent_background(image)
    width, height = rgba.size
    pixels = rgba.load()
    seen = bytearray(width * height)
    components = []
    for y in range(height):
        for x in range(width):
            index = y * width + x
            if seen[index] or pixels[x, y][3] == 0:
                continue
            seen[index] = 1
            queue = [index]
            members = []
            left = right = x
            top = bottom = y
            while queue:
                current = queue.pop()
                py, px = divmod(current, width)
                members.append(current)
                left, right = min(left, px), max(right, px)
                top, bottom = min(top, py), max(bottom, py)
                for nx, ny in ((px-1, py), (px+1, py), (px, py-1), (px, py+1)):
                    if not (0 <= nx < width and 0 <= ny < height):
                        continue
                    neighbor = ny * width + nx
                    if not seen[neighbor] and pixels[nx, ny][3] != 0:
                        seen[neighbor] = 1
                        queue.append(neighbor)
            components.append((len(members), (left, top, right+1, bottom+1), members))
    components.sort(key=lambda c:c[0], reverse=True)
    selected = components[:count]
    if len(selected) != count or selected[-1][0] < selected[0][0] / 10:
        raise ValueError(f'cannot identify {count} complete figures; components={[c[0] for c in components[:count+1]]}')
    if len(components) > count and components[count][0] > selected[-1][0] / 2:
        raise ValueError('ambiguous extra figure/component in master; explicit source review required')
    selected.sort(key=lambda c:(c[1][1]+c[1][3])/2)
    ordered = []
    for start in range(0, count, 3):
        row = sorted(selected[start:start+3], key=lambda c:(c[1][0]+c[1][2])/2)
        if len(row) != 3:
            raise ValueError('master does not have three figures per row')
        if max(c[1][1] for c in row) >= min(c[1][3] for c in row):
            raise ValueError('master row ordering is ambiguous')
        ordered += row
    results = []
    for _, box, members in ordered:
        result = Image.new('RGBA', (box[2]-box[0], box[3]-box[1]))
        target = result.load()
        for index in members:
            y, x = divmod(index, width)
            target[x-box[0], y-box[1]] = pixels[x, y]
        results.append((result, box))
    return results


def compile_group(asset, source, output, master_name):
    group, count = asset['source_group'], asset['frame_count']
    assert count in (3, 12)
    master = source / master_name
    image = Image.open(master)
    components = master_components(image, count)
    records = []
    for frame in range(count):
        sprite, bounds = components[frame]
        crop = (0, 0, sprite.width, sprite.height)
        # Keep the already reviewed native ink rectangle and feet, while taking
        # every color/detail pixel from the master rather than the 24px image.
        native = Image.open(source / asset['files'][frame]).convert('RGBA')
        ink = native.getbbox()
        if ink is None or native.size != (24, 24):
            raise ValueError(f'group{group} frame{frame}: invalid registered native frame')
        size = ((ink[2] - ink[0]) * 6, (ink[3] - ink[1]) * 6)
        sprite = sprite.crop(crop).resize(size, Image.Resampling.LANCZOS)
        result = Image.new('RGBA', (144, 144))
        result.alpha_composite(sprite, (ink[0] * 6, ink[1] * 6))
        name = f'fdicon-{group:03d}-style-a-hd6-f{frame:02d}.png'
        target = output / name
        result.save(target, optimize=False)
        records.append({'file':name, 'sha256':digest(target), 'source_frame':frame,
                        'source_cell':list(bounds), 'source_ink':list(crop),
                        'native_ink':list(ink), 'hd_ink':list(result.getbbox())})
    if asset.get('cycle_policy') == 'source_exact_repeats':
        for i in range(count):
            for j in range(i):
                if asset['frame_sha256'][i] == asset['frame_sha256'][j]:
                    (output / records[i]['file']).write_bytes((output / records[j]['file']).read_bytes())
                    records[i]['sha256'] = records[j]['sha256']
                    records[i]['repeat_of'] = j
                    records[i]['source_cell'] = records[j]['source_cell']
                    records[i]['source_ink'] = records[j]['source_ink']
                    records[i]['hd_ink'] = records[j]['hd_ink']
                    break
    return {'source_group':group, 'master_file':master_name, 'master_sha256':digest(master),
            'extraction':'whole_master_connected_components', 'source_alpha_preserved':True,
            'master_size':list(image.size), 'master_mode':image.mode,
            'frame_count':count, 'cycle_policy':asset['cycle_policy'], 'frames':records}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--catalog', type=Path, required=True)
    p.add_argument('--source', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--master-overrides', type=Path, help='明示尚未登錄的母稿；只供候選，輸出記錄來源')
    p.add_argument('--group', type=int, action='append', help='只編製指定的可丟棄樣本')
    p.add_argument('--audit-all', action='store_true', help='列完全部來源拒收；有缺件仍以非零退出，不供runtime')
    a = p.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not a.output.exists(), '不得覆寫既有候選'
    assert a.output.parent.is_dir() and a.output.parent.stat().st_uid == 1000
    catalog = json.loads(a.catalog.read_text())
    overrides = json.loads(a.master_overrides.read_text()) if a.master_overrides else {}
    assets = [x for x in catalog['assets'] if x.get('role') == 'map_sprite_set']
    if a.group:
        assets = [x for x in assets if x['source_group'] in a.group]
        assert {x['source_group'] for x in assets} == set(a.group)
    # Validate all sources before creating a partial output.
    masters = {}
    for asset in assets:
        group = asset['source_group']
        name = overrides.get(str(group), asset.get('master_file'))
        if not name or Path(name).name != name or not (a.source / name).is_file():
            raise ValueError(f'group{group}: missing/unsafe master reference')
        masters[group] = name
        for name, expected in zip(asset['files'], asset['frame_sha256']):
            assert Path(name).name == name and digest(a.source/name) == expected
    a.output.mkdir()
    records = []
    errors = []
    for asset in assets:
        try:
            record = compile_group(asset, a.source, a.output, masters[asset['source_group']])
        except ValueError as error:
            if not a.audit_all:
                raise
            errors.append({'source_group':asset['source_group'], 'error':str(error)})
            print('REFUSED', asset['source_group'], str(error), flush=True)
            continue
        records.append(record)
        print('compiled', record['source_group'], record['frame_count'], flush=True)
    report = {'status':'CANDIDATE-INCOMPLETE-REVIEW-REQUIRED' if errors else 'CANDIDATE-REVIEW-REQUIRED', 'scale':6, 'frame_size':[144,144],
              'alpha_contract':'coverage', 'catalog_sha256':digest(a.catalog),
              'compiler_sha256':digest(Path(__file__)), 'master_overrides':overrides,
              'not_registered_or_runtime':True, 'groups':records, 'errors':errors}
    (a.output/'hd-map-sprites.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    if errors:
        sys.exit(1)


if __name__ == '__main__':
    main()
