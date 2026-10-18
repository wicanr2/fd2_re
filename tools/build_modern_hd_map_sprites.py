#!/usr/bin/env python3
"""從既有手繪母稿編製 HD FDICON 幀；不放大 24×24 成品，不改 catalog。"""
import argparse
import hashlib
import json
import sys
from concurrent.futures import ProcessPoolExecutor
from pathlib import Path

from PIL import Image, ImageChops, ImageFilter
from process_modern_map_sprite_sheet import transparent_background


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def fit_hd_cell(source_size, registered_ink):
    """沿既有22px等比例方格，保留已登錄成品的腳底；不繼承舊稿寬高比。"""
    width, height = source_size
    bottom = registered_ink[3] * 6
    max_width, max_height = 132, min(132, bottom)
    if width <= 0 or height <= 0 or max_height <= 0:
        raise ValueError('invalid HD source size or registered feet')
    if width * max_height >= height * max_width:
        fitted_width = max_width
        fitted_height = max(1, (2*height*max_width+width)//(2*width))
    else:
        fitted_height = max_height
        fitted_width = max(1, (2*width*max_height+height)//(2*height))
    left = (144-fitted_width)//2
    return left, bottom-fitted_height, left+fitted_width, bottom


def master_components(image, count):
    """以完整輪廓找來源格，避免非等距母稿被固定3×4切掉頭或武器。"""
    if 'A' in image.getbands() and image.getchannel('A').getextrema()[0] == 0:
        rgba = image.convert('RGBA')
    else:
        rgba = transparent_background(image)
    width, height = rgba.size
    pixels = rgba.load()
    alpha = rgba.getchannel('A')
    partial_alpha = any(alpha.histogram()[1:255])
    # Opacity is used only to identify instances. Low-coverage antialias pixels
    # can connect neighboring figures even when there is a visible gap. Keep
    # their original coverage when rebuilding the selected contour below.
    core = alpha.point(lambda value: 255 if value >= 128 else 0).load()
    seen = bytearray(width * height)
    components = []
    for y in range(height):
        for x in range(width):
            index = y * width + x
            if seen[index] or core[x, y] == 0:
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
                    if not seen[neighbor] and core[nx, ny] != 0:
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
    columns = min(count, 3)
    for start in range(0, count, columns):
        row = sorted(selected[start:start+columns], key=lambda c:(c[1][0]+c[1][2])/2)
        if len(row) != columns:
            raise ValueError('master has an incomplete figure row')
        if max(c[1][1] for c in row) >= min(c[1][3] for c in row):
            raise ValueError('master row ordering is ambiguous')
        ordered += row
    results = []
    for _, box, members in ordered:
        if partial_alpha:
            left, top, right, bottom = box
            expanded = (max(0,left-3),max(0,top-3),min(width,right+3),min(height,bottom+3))
            mask = Image.new('L',(expanded[2]-expanded[0],expanded[3]-expanded[1]))
            target = mask.load()
            for index in members:
                y, x = divmod(index, width)
                target[x-expanded[0],y-expanded[1]] = 255
            result = rgba.crop(expanded)
            result.putalpha(ImageChops.multiply(result.getchannel('A'),mask.filter(ImageFilter.MaxFilter(7))))
            tight = result.getbbox()
            if tight is None:
                raise ValueError('selected opacity core has no original coverage')
            results.append((result.crop(tight),(expanded[0]+tight[0],expanded[1]+tight[1],expanded[0]+tight[2],expanded[1]+tight[3])))
            continue
        result = Image.new('RGBA', (box[2]-box[0], box[3]-box[1]))
        target = result.load()
        for index in members:
            y, x = divmod(index, width)
            target[x-box[0], y-box[1]] = pixels[x, y]
        results.append((result, box))
    return results


def compile_group(asset, source, output, master_name, frame_overrides, prepared=None):
    group, count = asset['source_group'], asset['frame_count']
    assert count in (3, 12)
    master = source / master_name
    image = Image.open(master)
    components = master_components(image, count) if prepared is None else prepared
    records = []
    for frame in range(count):
        sprite, bounds = components[frame]
        frame_source = master
        if str(frame) in frame_overrides:
            frame_source = source / frame_overrides[str(frame)]
            sprite, bounds = master_components(Image.open(frame_source), 1)[0]
        crop = (0, 0, sprite.width, sprite.height)
        # The 24px input is a registered modern preview, not an original FDICON
        # silhouette. Retain its feet, without stretching new art to its aspect.
        native = Image.open(source / asset['files'][frame]).convert('RGBA')
        ink = native.getbbox()
        if ink is None or native.size != (24, 24):
            raise ValueError(f'group{group} frame{frame}: invalid registered native frame')
        fit = fit_hd_cell(sprite.size, ink)
        size = (fit[2]-fit[0], fit[3]-fit[1])
        sprite = sprite.crop(crop).resize(size, Image.Resampling.LANCZOS)
        result = Image.new('RGBA', (144, 144))
        result.alpha_composite(sprite, fit[:2])
        name = f'fdicon-{group:03d}-style-a-hd6-f{frame:02d}.png'
        target = output / name
        result.save(target, optimize=False)
        records.append({'file':name, 'sha256':digest(target), 'source_frame':frame,
                        'source_file':frame_source.name, 'source_sha256':digest(frame_source),
                        'source_cell':list(bounds), 'source_ink':list(crop),
                        'registered_ink':list(ink), 'fit_rect':list(fit), 'hd_ink':list(result.getbbox())})
    if asset.get('cycle_policy') == 'source_exact_repeats':
        for i in range(count):
            for j in range(i):
                if asset['frame_sha256'][i] == asset['frame_sha256'][j]:
                    (output / records[i]['file']).write_bytes((output / records[j]['file']).read_bytes())
                    records[i]['sha256'] = records[j]['sha256']
                    records[i]['repeat_of'] = j
                    records[i]['source_file'] = records[j]['source_file']
                    records[i]['source_sha256'] = records[j]['source_sha256']
                    records[i]['source_cell'] = records[j]['source_cell']
                    records[i]['source_ink'] = records[j]['source_ink']
                    records[i]['hd_ink'] = records[j]['hd_ink']
                    records[i]['fit_rect'] = records[j]['fit_rect']
                    break
    return {'source_group':group, 'master_file':master_name, 'master_sha256':digest(master),
            'extraction':'whole_master_connected_components', 'source_alpha_preserved':True,
            'instance_core_alpha':128, 'coverage_edge_radius':3,
            'master_size':list(image.size), 'master_mode':image.mode,
            'frame_count':count, 'cycle_policy':asset['cycle_policy'], 'frames':records}


def compile_family(job):
    """同一母稿與槽數只辨識一次；各圖組仍依自己的24父項定位。"""
    assets, source, output, master_name, frame_overrides, audit_all = job
    records, errors = [], []
    try:
        with Image.open(source/master_name) as image:
            prepared = master_components(image, assets[0]['frame_count'])
    except ValueError as error:
        if not audit_all:
            raise
        return [], [{'source_group':a['source_group'], 'error':str(error)} for a in assets]
    for asset in assets:
        try:
            records.append(compile_group(asset, source, output, master_name,
                                         frame_overrides.get(str(asset['source_group']), {}), prepared))
        except ValueError as error:
            if not audit_all:
                raise
            errors.append({'source_group':asset['source_group'], 'error':str(error)})
    return records, errors


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--catalog', type=Path, required=True)
    p.add_argument('--source', type=Path, required=True)
    p.add_argument('--output', type=Path, required=True)
    p.add_argument('--master-overrides', type=Path, help='明示尚未登錄的母稿；只供候選，輸出記錄來源')
    p.add_argument('--frame-overrides', type=Path, help='明示group→frame→單幀來源檔名；只供候選')
    p.add_argument('--group', type=int, action='append', help='只編製指定的可丟棄樣本')
    p.add_argument('--audit-all', action='store_true', help='列完全部來源拒收；有缺件仍以非零退出，不供runtime')
    p.add_argument('--workers', type=int, choices=range(1, 5), default=1, help='有界工作程序數；同母稿只辨識一次')
    a = p.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not a.output.exists(), '不得覆寫既有候選'
    assert a.output.parent.is_dir() and a.output.parent.stat().st_uid == 1000
    catalog = json.loads(a.catalog.read_text())
    overrides = json.loads(a.master_overrides.read_text()) if a.master_overrides else {}
    frame_overrides = json.loads(a.frame_overrides.read_text()) if a.frame_overrides else {}
    assets = [x for x in catalog['assets'] if x.get('role') == 'map_sprite_set']
    assets_by_group = {str(x['source_group']):x for x in assets}
    for group, frames in frame_overrides.items():
        if group not in assets_by_group or not isinstance(frames, dict):
            raise ValueError('unknown group or invalid single-frame overrides')
        asset = assets_by_group[group]
        for frame, name in frames.items():
            if frame not in {str(i) for i in range(asset['frame_count'])}:
                raise ValueError(f'group{group}: invalid single-frame index')
            if not isinstance(name, str) or Path(name).name != name or not (a.source/name).is_file():
                raise ValueError(f'group{group} frame{frame}: missing/unsafe frame reference')
            if asset['cycle_policy'] == 'source_exact_repeats' and asset['frame_sha256'][int(frame)] in asset['frame_sha256'][:int(frame)]:
                raise ValueError('a repeated phase cannot override its canonical source')
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
    families = {}
    for asset in assets:
        key = (masters[asset['source_group']], asset['frame_count'])
        families.setdefault(key, []).append(asset)
    jobs = [(family, a.source, a.output, master, frame_overrides, a.audit_all)
            for (master, _), family in families.items()]
    executor = ProcessPoolExecutor(max_workers=a.workers) if a.workers > 1 else None
    try:
        results = executor.map(compile_family, jobs) if executor else map(compile_family, jobs)
        for family_records, family_errors in results:
            records.extend(family_records)
            errors.extend(family_errors)
            for record in family_records:
                print('compiled', record['source_group'], record['frame_count'], flush=True)
            for error in family_errors:
                print('REFUSED', error['source_group'], error['error'], flush=True)
    finally:
        if executor:
            executor.shutdown(wait=True, cancel_futures=True)
    records.sort(key=lambda item:item['source_group'])
    errors.sort(key=lambda item:item['source_group'])
    report = {'status':'CANDIDATE-INCOMPLETE-REVIEW-REQUIRED' if errors else 'CANDIDATE-REVIEW-REQUIRED', 'scale':6, 'frame_size':[144,144],
              'alpha_contract':'coverage', 'layout_policy':'contain_22px_cell_feet_v1', 'catalog_sha256':digest(a.catalog),
              'compiler_sha256':digest(Path(__file__)), 'master_overrides':overrides,
              'frame_overrides':frame_overrides,
              'not_registered_or_runtime':True, 'groups':records, 'errors':errors}
    (a.output/'hd-map-sprites.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    if errors:
        sys.exit(1)


if __name__ == '__main__':
    main()
