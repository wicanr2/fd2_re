#!/usr/bin/env python3
"""以原生光色像素定位112眼光，將真正手繪紋理編製至既有HD6 PNG。"""
import argparse
import copy
import json
import math
import os
from pathlib import Path
import shutil
import statistics

from PIL import Image, ImageDraw

from build_fd2_character_battle_hd import load_native
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT, validate


def component_boxes(points):
    remaining = set(points)
    boxes = []
    while remaining:
        first = remaining.pop()
        group, pending = [first], [first]
        while pending:
            x, y = pending.pop()
            for dx in [-1, 0, 1]:
                for dy in [-1, 0, 1]:
                    point = x+dx, y+dy
                    if point in remaining:
                        remaining.remove(point)
                        group.append(point)
                        pending.append(point)
        boxes.append([min(x for x, y in group), min(y for x, y in group),
                      max(x for x, y in group)+1, max(y for x, y in group)+1, len(group)])
    changed = True
    while changed:
        changed = False
        for i, a in enumerate(boxes):
            for j in range(i+1, len(boxes)):
                b = boxes[j]
                gx = max(a[0], b[0])-min(a[2], b[2])
                gy = max(a[1], b[1])-min(a[3], b[3])
                if (gx <= 2 and gy <= 10) or (gx <= 6 and gy <= 0):
                    boxes[i] = [min(a[0], b[0]), min(a[1], b[1]),
                                max(a[2], b[2]), max(a[3], b[3]), a[4]+b[4]]
                    boxes.pop(j)
                    changed = True
                    break
            if changed:
                break
    return sorted(boxes)


def native_geometry(frame, picture, index):
    warm, bright = [], []
    for y in range(picture.height):
        for x in range(picture.width):
            red, green, blue, alpha = picture.getpixel((x, y))
            if alpha and red >= 100 and green >= 40 and red >= green and red > blue+30 and green > blue+20:
                point = x+frame['x'], y+frame['y']
                warm.append(point)
                if red >= 190 and green >= 140:
                    bright.append(point)
    ray = None
    if 2 <= index <= 16:
        points = [(x, y) for x, y in bright if
                  (20 <= y <= 100 if index <= 15 else y >= 135 and x <= 274)]
        if len(points) < 10:
            raise ValueError('原生光束定位點不足')
        mx, my = statistics.mean(x for x, y in points), statistics.mean(y for x, y in points)
        denominator = sum((y-my)**2 for x, y in points)
        if denominator <= 0:
            raise ValueError('原生光束方向退化')
        slope = sum((y-my)*(x-mx) for x, y in points)/denominator
        intercept = mx-slope*my
        rms = math.sqrt(statistics.mean((x-slope*y-intercept)**2 for x, y in points))
        if rms > 1:
            raise ValueError('原生光色像素不支持單一直線')
        start_y = 6 if index <= 15 else min(y for x, y in points)
        ray = {'points': len(points), 'slope_dx_dy': slope, 'intercept': intercept,
               'rms_native_pixels': rms, 'start_world': [slope*start_y+intercept, start_y],
               'end_world': [slope*174+intercept, 174], 'attached_to_eye': index <= 15}
    def in_ray(point):
        x, y = point
        return (ray is not None and ray['start_world'][1] <= y <= 177 and
                abs(x-ray['slope_dx_dy']*y-ray['intercept']) <= 2)
    fire_points = [point for point in warm if point[1] >= 110 and not in_ray(point)]
    fires = []
    for box in component_boxes(fire_points):
        x0, y0, x1, y1, count = box
        # 小而稀疏的直列使用獨立手繪火星，保留末拍暗色點，不把它當空白。
        kind = 'embers' if count < 25 and x1-x0 <= 4 else 'flame'
        pad = 0 if kind == 'embers' else 1
        bounds = [max(frame['x'], x0-pad), max(frame['y'], y0-pad),
                  min(frame['x']+frame['width'], x1+pad),
                  min(frame['y']+frame['height'], y1+pad)]
        if bounds[0] >= bounds[2] or bounds[1] >= bounds[3]:
            raise ValueError('原生火焰區域越界')
        fires.append({'world_box': bounds, 'kind': kind, 'native_pixels': count})
    unassigned = [p for p in warm if not in_ray(p) and not any(
        f['world_box'][0] <= p[0] < f['world_box'][2] and
        f['world_box'][1] <= p[1] < f['world_box'][3] for f in fires)]
    if unassigned:
        raise ValueError('原生暖色光效仍有未分類像素')
    return {'ray': ray, 'fires': fires, 'warm_pixels': len(warm), 'unassigned_warm_pixels': 0}


def axis_matrix(axis, start, end, width_hd, source_width):
    (x0, y0), (x1, y1) = axis
    slen = math.hypot(x1-x0, y1-y0)
    dx, dy = end[0]-start[0], end[1]-start[1]
    length = math.hypot(dx, dy)
    if min(slen, length, width_hd, source_width) <= 0:
        raise ValueError('紋理軸向或寬度退化')
    ex, ey = (x1-x0)/slen, (y1-y0)/slen
    nx, ny = -ey, ex
    ux, uy = dx/length, dy/length
    px, py = -uy, ux
    along, across = length/slen, width_hd/source_width
    a = along*ux*ex+across*px*nx
    b = along*ux*ey+across*px*ny
    c = along*uy*ex+across*py*nx
    d = along*uy*ey+across*py*ny
    return [a, b, start[0]-a*x0-b*y0, c, d, start[1]-c*x0-d*y0]


def project_texture(picture, matrix, size):
    a, b, tx, c, d, ty = matrix
    determinant = a*d-b*c
    if not math.isfinite(determinant) or determinant <= 0:
        raise ValueError('紋理仿射退化或反射')
    corners = [(a*x+b*y+tx, c*x+d*y+ty)
               for x in [0, picture.width] for y in [0, picture.height]]
    ox = math.floor(min(x for x, y in corners))-2
    oy = math.floor(min(y for x, y in corners))-2
    width = math.ceil(max(x for x, y in corners))-ox+2
    height = math.ceil(max(y for x, y in corners))-oy+2
    if not 0 < width <= 5000 or not 0 < height <= 5000:
        raise ValueError('紋理投影範圍過大')
    local_tx, local_ty = tx-ox, ty-oy
    inverse = [d/determinant, -b/determinant, (b*local_ty-d*local_tx)/determinant,
               -c/determinant, a/determinant, (c*local_tx-a*local_ty)/determinant]
    fitted = picture.transform((width, height), Image.Transform.AFFINE, inverse,
                               Image.Resampling.BICUBIC)
    canvas = Image.new('RGBA', size)
    canvas.alpha_composite(fitted, (ox, oy))
    if canvas.getchannel('A').getextrema()[1] < 250:
        raise ValueError('光痕取樣失去高覆蓋核心')
    visible = fitted.getchannel('A').crop((-ox, -oy, size[0]-ox, size[1]-oy))
    clipped = sum(v >= 192 for v in fitted.getchannel('A').getdata())-sum(
        v >= 192 for v in visible.getdata())
    return canvas, clipped


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['parent', 'pack', 'inventory', 'ray', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--ray-sha256', required=True)
    p.add_argument('--axis', type=float, nargs=4, required=True)
    p.add_argument('--eye', type=float, nargs=2, required=True)
    p.add_argument('--body-pose', required=True)
    p.add_argument('--width-hd', type=float, required=True)
    args = p.parse_args()
    parent, ray_path, out = args.parent.resolve(), args.ray.resolve(), args.out.resolve()
    if (not parent.is_relative_to(ROOT) or not ray_path.is_relative_to(ROOT) or
            not out.is_relative_to(ROOT) or out.exists() or
            out.parent.stat().st_uid != os.getuid() or ray_path.stat().st_uid != os.getuid()):
        raise SystemExit('使用本人擁有的專案輸入與新候選目錄')
    if (not all(math.isfinite(x) for x in args.axis+args.eye+[args.width_hd]) or
            not 0 < args.width_hd <= 24 or digest(ray_path) != args.ray_sha256):
        raise SystemExit('光痕SHA、定位點或寬度不符')
    report_path, catalog_path = parent/'compiled.json', parent/'catalog.json'
    original_report = json.loads(report_path.read_text())
    catalog = validate(True, catalog_path)
    inventory = json.loads(args.inventory.read_text())
    if (original_report['status'] != 'PASS-COMPILED-CHARACTER-CANDIDATE' or
            original_report['figure'] != 112 or original_report.get('ray_projection') or
            original_report['catalog_sha256'] != digest(catalog_path) or
            original_report['inventory_sha256'] != digest(args.inventory)):
        raise SystemExit('只接受固定112父候選與相同盤點')
    idle, idle_pictures = load_native(args.pack, inventory, 336)
    attack, pictures = load_native(args.pack, inventory, 337)
    if (len(idle['frames']) != 1 or len(attack['frames']) != 20 or
            attack['native_header'] != {'byte_1': 0, 'byte_2': 20, 'byte_4': 4}):
        raise SystemExit('112固定圖庫形狀不符')
    axis = [args.axis[:2], args.axis[2:]]
    with Image.open(ray_path) as image:
        if image.mode != 'RGBA' or image.getchannel('A').getextrema()[0] != 0 or image.getchannel('A').getextrema()[1] < 250:
            raise SystemExit('手繪光痕缺透明背景或高覆蓋核心')
        texture = image.copy()
    for x, y in axis:
        if not 0 <= x < texture.width or not 0 <= y < texture.height or texture.getpixel((round(x), round(y)))[3] < 192:
            raise SystemExit('光痕軸端不是原稿高覆蓋點')
    widths = [sum(texture.getpixel((x, y))[3] >= 192 for x in range(texture.width))
              for y in range(50, min(1400, texture.height), 50)]
    source_width = statistics.median(widths)
    if source_width <= 0:
        raise SystemExit('光痕寬度不足')
    source = original_report['sources'][args.body_pose]
    if digest(ROOT/source['file']) != source['sha256']:
        raise SystemExit('父候選人物原稿SHA不同')
    with Image.open(ROOT/source['file']) as image:
        x, y = args.eye
        if not 0 <= x < image.width or not 0 <= y < image.height or image.getpixel((round(x), round(y)))[3] < 192:
            raise SystemExit('眼部定位點缺少原稿高覆蓋像素')
    frames = copy.deepcopy(original_report['frames'])
    if {(v['resource'], v['frame']) for v in frames} != {(336, 0)} | {(337, i) for i in range(20)}:
        raise SystemExit('父候選未涵蓋完整21幀')
    for v in frames:
        if digest(parent/'pack'/v['file']) != v['sha256']:
            raise SystemExit('父候選PNG SHA不同')
    geometries = [native_geometry(f, im, i) for i, (f, im) in enumerate(zip(attack['frames'], pictures))]
    out.mkdir()
    pack, layers = out/'pack', out/'ray-layers'
    pack.mkdir()
    layers.mkdir()
    for file in (parent/'pack').iterdir():
        (pack/file.name).symlink_to(os.path.relpath(file, pack))
    raw_name = 'figure-112-beam112vertical2-battle-hd6-master-v1.png'
    if (pack/raw_name).exists():
        raise SystemExit('父候選含同名光痕原稿')
    shutil.copyfile(ray_path, pack/raw_name)
    changed = []
    for record in frames:
        if record['resource'] != 337:
            continue
        i = record['frame']
        geometry = geometries[i]
        ray = geometry['ray']
        if ray is None:
            continue
        frame = attack['frames'][i]
        body = next(v for v in record.get('layers', [record]) if v['pose'] == args.body_pose)
        root = [frame['x']*6+body['origin'][0]+args.eye[0]*body['scale_xy'][0],
                frame['y']*6+body['origin'][1]+args.eye[1]*body['scale_xy'][1]]
        start = root if ray['attached_to_eye'] else [x*6 for x in ray['start_world']]
        start = [start[0]-frame['x']*6, start[1]-frame['y']*6]
        end = [ray['end_world'][0]*6-frame['x']*6, ray['end_world'][1]*6-frame['y']*6]
        matrix = axis_matrix(axis, start, end, args.width_hd, source_width)
        image_path = parent/'pack'/record['file']
        with Image.open(image_path) as image:
            image = image.copy()
        beam, clipped = project_texture(texture, matrix, image.size)
        layer_path = layers/f'ray-337-f{i:02d}.png'
        beam.save(layer_path)
        image.alpha_composite(beam)
        if image.getchannel('A').getextrema() != (0, 255):
            raise SystemExit('成品透明度契約不符')
        (pack/record['file']).unlink()
        image.save(pack/record['file'])
        record['parent_png_sha256'] = record['sha256']
        record['sha256'] = digest(pack/record['file'])
        record['clipped_high_coverage_pixels'] += clipped
        record['ray_projection'] = {'source_axis': axis, 'forward': matrix,
            'start_hd': start, 'end_hd': end, 'width_hd': args.width_hd,
            'attached_to_eye': ray['attached_to_eye'], 'native_geometry': geometry,
            'layer_file': str(layer_path.relative_to(ROOT)), 'layer_sha256': digest(layer_path),
            'clipped_high_coverage_pixels': clipped, 'alpha_extrema': list(beam.getchannel('A').getextrema())}
        changed.append(i)
    if changed != list(range(2, 17)):
        raise SystemExit('光痕影格集不符')
    candidate = copy.deepcopy(catalog)
    candidate['private_root'] = str(pack.relative_to(ROOT))
    asset = next(a for a in candidate['assets'] if a.get('source_archive') == 'FIGANI.DAT' and a['source_resource'] == 337)
    asset['frame_sha256'] = [v['sha256'] for v in frames if v['resource'] == 337]
    asset['source_files'].append(raw_name)
    asset['source_sha256'].append(args.ray_sha256)
    candidate_path = out/'catalog.json'
    candidate_path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2)+'\n')
    validate(True, candidate_path)
    shutil.copyfile(parent/'base-catalog.json', out/'base-catalog.json')
    runtime = out/'runtime'
    (runtime/'assets/themes/modern').mkdir(parents=True)
    for source_path in (ROOT/'remake/assets').iterdir():
        if source_path.name != 'themes':
            (runtime/'assets'/source_path.name).symlink_to(
                os.path.relpath(source_path, runtime/'assets'))
    (runtime/'assets/themes/modern/catalog.json').symlink_to(
        os.path.relpath(candidate_path, runtime/'assets/themes/modern'))
    sheet = Image.new('RGB', (1600, ((len(frames)+3)//4)*400), (35, 40, 48))
    draw = ImageDraw.Draw(sheet)
    native = {(336, 0): (idle['frames'][0], idle_pictures[0])}
    native.update({(337, i): pair for i, pair in enumerate(zip(attack['frames'], pictures))})
    for i, record in enumerate(frames):
        f, original = native[record['resource'], record['frame']]
        with Image.open(pack/record['file']) as image:
            for dy, picture, scale, label in [(0, original, 1, 'native source'), (200, image, 6, 'HD + authored ray')]:
                stage = Image.new('RGBA', (320*scale, 200*scale))
                stage.alpha_composite(picture, (f['x']*scale, f['y']*scale))
                thumb = stage.resize((320, 200), Image.Resampling.LANCZOS)
                x, y = i%4*400, i//4*400+dy
                sheet.paste(thumb, (x, y), thumb)
                draw.text((x+4, y+4), f"{record['resource']}/{record['frame']} {label}", fill='white')
    sheet.save(out/'review.png')
    report = copy.deepcopy(original_report)
    report.update(catalog_sha256=digest(candidate_path), frames=frames, review_sha256=digest(out/'review.png'),
        parent_compiled={'file': str(report_path.relative_to(ROOT)), 'sha256': digest(report_path)},
        parent_catalog={'file': str(catalog_path.relative_to(ROOT)), 'sha256': digest(catalog_path)},
        overlay_compiler_sha256=digest(Path(__file__)),
        ray_source={'file': str(ray_path.relative_to(ROOT)), 'sha256': args.ray_sha256,
                    'axis': axis, 'source_core_width': source_width, 'body_pose': args.body_pose, 'eye': args.eye},
        native_art_geometry=geometries, ray_projection=True,
        scope='父候選身體／火焰與原描述子保持；第二編製步驟投影真正手繪光痕，2～15連眼、16分離尾段，其他六PNG逐SHA保持。沿原生暖色像素量測的美術幾何近似，不宣稱原版逐RGB、正常玩家或新規則。')
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    if digest(ray_path) != args.ray_sha256 or digest(report_path) != report['parent_compiled']['sha256']:
        raise SystemExit('原稿或父候選被改寫')
    print(json.dumps({'status': report['status'], 'figure': 112, 'frames': len(frames),
                      'ray_frames': len(changed), 'attached': 14, 'detached': 1,
                      'catalog_assets': len(candidate['assets'])}))


if __name__ == '__main__':
    main()
