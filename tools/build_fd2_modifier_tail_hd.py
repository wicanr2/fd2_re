#!/usr/bin/env python3
"""依READY契約編製191～229狀態光效候選，保留正式素材包。"""
import argparse
import hashlib
import json
import os
import shutil
from collections import deque
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter
from audit_fd2_player9_map_source import audit
from build_fd2_compound_tail_hd import extend_rgb
from validate_modern_theme_catalog import ROOT, validate

INDICES = list(range(191, 230))


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def slot(index):
    if index not in INDICES:
        raise ValueError('未審查的狀態光效索引')
    return 0 if index < 204 else 1 if index < 217 else 2


def components(native):
    width, height = native.size
    seen, result = set(), []
    for y in range(height):
        for x in range(width):
            if (x, y) in seen or native.getpixel((x, y))[3] == 0:
                continue
            queue = deque([(x, y)])
            seen.add((x, y))
            points = []
            while queue:
                px, py = queue.popleft()
                points.append((px, py))
                for dy in [-1, 0, 1]:
                    for dx in [-1, 0, 1]:
                        q = px+dx, py+dy
                        if q not in seen and 0 <= q[0] < width and 0 <= q[1] < height and native.getpixel(q)[3]:
                            seen.add(q)
                            queue.append(q)
            result.append(points)
    return sorted(result, key=len, reverse=True)


def compile_frame(native, master_rgb, intensity):
    size = native.width*6, native.height*6
    colours = [[0, 0, 0, 0] for _ in range(size[0]*size[1])]
    boxes = []
    for points in components(native):
        xs, ys = zip(*points)
        box = min(xs), min(ys), max(xs)+1, max(ys)+1
        boxes.append(box)
        coverage = Image.new('L', native.size)
        for point in points:
            coverage.putpixel(point, 255)
        coverage = coverage.resize(size, Image.Resampling.LANCZOS).filter(ImageFilter.GaussianBlur(1.1))
        box6 = tuple(v*6 for v in box)
        art = master_rgb.resize((box6[2]-box6[0], box6[3]-box6[1]), Image.Resampling.LANCZOS)
        for y in range(size[1]):
            for x in range(size[0]):
                alpha = coverage.getpixel((x, y))
                if not alpha:
                    continue
                colour = art.getpixel((min(max(x-box6[0], 0), art.width-1), min(max(y-box6[1], 0), art.height-1)))
                acc = colours[y*size[0]+x]
                for c in range(3):
                    acc[c] += colour[c]*alpha
                acc[3] += alpha
    blended = Image.new('RGBA', size)
    blended.putdata([(*[round(v/acc[3]) for v in acc[:3]], 255) if acc[3] else (0, 0, 0, 0) for acc in colours])
    fringe_rgb = extend_rgb(blended)
    matte = native.getchannel('A').resize(size, Image.Resampling.LANCZOS).filter(ImageFilter.GaussianBlur(1.1))
    warm = Image.new('L', native.size)
    warm.putdata([255 if a and rr > g and g > rr*.2 and b < g*.6 else 0 for rr, g, b, a in native.getdata()])
    warm = warm.resize(size, Image.Resampling.LANCZOS)
    values, fringe = [], []
    for y in range(size[1]):
        for x in range(size[0]):
            edge = min(1, min(x, y, size[0]-1-x, size[1]-1-y)/2)
            alpha = round(matte.getpixel((x, y))*intensity*edge)
            if not alpha:
                values.append((0, 0, 0, 0))
                continue
            acc = colours[y*size[0]+x]
            if not acc[3]:
                fringe.append(alpha)
            colour = [round(v/acc[3]) for v in acc[:3]] if acc[3] else list(fringe_rgb[y*size[0]+x])
            brown = warm.getpixel((x, y))/255
            if brown:
                lum = max(colour)
                colour = [round(v*(1-brown)+lum*f*brown) for v, f in zip(colour, [.65, .43, .12])]
            values.append((*colour, alpha))
    output = Image.new('RGBA', size)
    output.putdata(values)
    if output.getchannel('A').getextrema()[0] != 0 or output.getchannel('A').getextrema()[1] == 0 or max(fringe, default=0) > 1:
        raise ValueError('缺少透明／可見素材，或量化邊緣超過已審查範圍')
    return output, boxes, len(fringe), max(fringe, default=0)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack', 'base-catalog', 'out']:
        parser.add_argument('--'+key, type=Path, required=True)
    parser.add_argument('--master-root', type=Path, help='以私人素材包或保存目錄內的同名未修改母稿重生')
    args = parser.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists() or args.out.parent.stat().st_uid != os.getuid():
        raise ValueError('Docker內使用本人擁有的全新候選目錄')
    state = json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['modifier_tail_art_followup_20261024']
    if state['status'] not in ['READY', 'CONFORMED-SCOPED / RUNTIME-E1'] or sha(args.base_catalog) != state['base_catalog']['sha256']:
        raise ValueError('未達READY或不是固定763項基底')
    masters, source_paths = [], []
    for source in state['masters']:
        path = args.master_root/Path(source['file']).name if args.master_root else ROOT/source['file']
        if sha(path) != source['sha256']:
            raise ValueError('未修改母稿SHA不同')
        with Image.open(path) as im:
            if im.mode != 'RGBA' or list(im.size) != source['size'] or list(im.getchannel('A').getextrema()) != source['alpha_extrema']:
                raise ValueError('原稿形狀或alpha不同')
            box = im.getchannel('A').point(lambda v: 255 if v >= 32 else 0).getbbox()
            if list(box) != source['crop32']:
                raise ValueError('材質裁切不同')
            cropped = im.crop(box)
            rgb = Image.new('RGB', cropped.size)
            rgb.putdata(extend_rgb(cropped))
            masters.append(rgb.filter(ImageFilter.GaussianBlur(3)))
            source_paths.append(path)
    report, pictures = audit(args.original_pack, {6: INDICES})
    pinned = json.loads((ROOT/state['fixed_sources']['file']).read_text())
    fixed = {f['index']: f for b in pinned['resources'] if b['resource'] == 6 for f in b['frames']}
    if report['pack_manifest_sha256'] != pinned['pack_manifest_sha256'] or any(json.loads(json.dumps(row)) != fixed[row['index']] for _, row, _ in pictures):
        raise ValueError('固定原版來源不同')
    catalog = validate(True, args.base_catalog)
    if len(catalog['assets']) != 763 or any(row['index'] in INDICES for asset in catalog['assets'] if asset['role'] == 'map_overlay_hd_set' and asset['source_resource'] == 6 for row in asset['native_lmi_frames']):
        raise ValueError('不是未擴充的763項基底')
    old_root = ROOT/catalog['private_root']
    old_names = set()
    for asset in catalog['assets']:
        old_names.update(asset.get('files', []))
        old_names.update(asset.get('source_files', []))
        old_names.update(asset[key] for key in ['file', 'master_file'] if key in asset)
    for name in old_names:
        if Path(name).name != name or not (old_root/name).is_file():
            raise ValueError('既有來源缺件')
    args.out.mkdir()
    for name in sorted(old_names):
        (args.out/name).symlink_to(os.path.relpath(old_root/name, args.out))
    for source, source_path in zip(state['masters'], source_paths):
        name = Path(source['file']).name
        if name not in old_names:
            shutil.copyfile(source_path, args.out/name)
    peaks = [0, 0, 0]
    for _, row, im in pictures:
        peaks[slot(row['index'])] = max(peaks[slot(row['index'])], max(max(rr, g, b) for rr, g, b, a in im.getdata() if a))
    rows, files, descriptions = [], [], []
    for _, row, native in pictures:
        index = row['index']
        intensity = max(max(rr, g, b) for rr, g, b, a in native.getdata() if a)/peaks[slot(index)]
        output, boxes, fringe_count, fringe_max = compile_frame(native, masters[slot(index)], intensity)
        name = f'modifier-tail-fdother006-{index:03d}-hd6-r1.png'
        output.save(args.out/name)
        files.append(name)
        descriptions.append({key: row[key] for key in ['index', 'width', 'height', 'pixels_sha256']})
        rows.append({'index': index, 'slot': slot(index), 'component_boxes': boxes, 'intensity': intensity, 'quantized_fringe_pixels': fringe_count, 'quantized_fringe_alpha_max': fringe_max, 'file': name, 'sha256': sha(args.out/name), 'alpha_extrema': output.getchannel('A').getextrema()})
    bank = report['resources'][0]
    asset = dict(asset_id='modifier-tail-fdother006-191-229-hd6', role='map_overlay_hd_set', status='runtime_candidate', files=files, width=1920, height=1200, frame_count=39,
                 frame_sha256=[sha(args.out/name) for name in files], consumer_contract='native_lmi1_rgba_hd6_v1', alpha_contract='coverage', cycle_policy='native_lmi1_entries', source_archive='FDOTHER.DAT', source_resource=6,
                 source_metadata_sha256=bank['metadata_sha256'], native_lmi_frames=descriptions, layout_policy='native_lmi1_rect_hd6', source_files=[Path(source['file']).name for source in state['masters']],
                 source_sha256=[source['sha256'] for source in state['masters']], source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md', 'tools/build_fd2_modifier_tail_hd.py'])
    catalog['assets'].append(asset)
    catalog['private_root'] = str(args.out.resolve().relative_to(ROOT))
    path = args.out/'catalog.json'
    path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    validate(True, path)
    recipe = {'status': 'PASS-MODIFIER-TAIL-ART-CANDIDATE', 'base_catalog_sha256': sha(args.base_catalog), 'catalog_sha256': sha(path), 'tool_sha256': sha(Path(__file__)), 'source': report, 'masters': state['masters'], 'records': rows, 'old_file_count': len(old_names), 'old_files_are_links': True, 'production_changed': False, 'policy': state['policy']}
    (args.out/'compiled.json').write_text(json.dumps(recipe, ensure_ascii=False, indent=2)+'\n')
    sheet = Image.new('RGB', (960, 7*168), '#151721')
    draw = ImageDraw.Draw(sheet)
    for i, row in enumerate(rows):
        im = Image.open(args.out/row['file'])
        im.thumbnail((148, 140), Image.Resampling.LANCZOS)
        x, y = i % 6*160, i//6*168
        sheet.paste(im, (x+(160-im.width)//2, y+24), im)
        draw.text((x+5, y+4), str(row['index']), fill='white')
    sheet.save(args.out/'overview.png')
    print(json.dumps({'status': recipe['status'], 'new_frames': 39, 'catalog_assets': len(catalog['assets']), 'old_files': len(old_names)}))


if __name__ == '__main__':
    main()
