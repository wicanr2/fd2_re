#!/usr/bin/env python3
"""以原始描述子及逐幀母稿編製亞雷斯19幀隔離候選，不登錄正式清冊。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageDraw
from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate


DESCRIPTOR = ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']


def blue_anchor(picture):
    """本批已審藍衣美術的錨點，非原版欄位語意。"""
    # 冷色金屬反光與孤立的彩色邊點不能延長衣帽錨點。
    pending = {i for i, (r, g, b, a) in enumerate(picture.getdata())
               if a >= 192 and b >= 70 and b*100 >= r*175 and b*100 >= g*145}
    points = []
    while pending:
        stack = [pending.pop()]; component = []
        while stack:
            i = stack.pop(); x, y = i % picture.width, i // picture.width
            component.append((x, y))
            neighbors = ([] if x == 0 else [i-1]) + ([] if x+1 == picture.width else [i+1])
            neighbors += ([] if y == 0 else [i-picture.width]) + ([] if y+1 == picture.height else [i+picture.width])
            for neighbor in neighbors:
                if neighbor in pending:
                    pending.remove(neighbor); stack.append(neighbor)
        if len(component) >= 1000:
            points.extend(component)
    if len(points) < 1000:
        raise ValueError('藍衣錨點不足，停止並檢查母稿')
    xs, ys = zip(*points)
    return [min(xs), min(ys), max(xs)+1, max(ys)+1]


def native_frames(pack, inventory, resource):
    row = next(x for x in inventory['resources'] if x['resource'] == resource)
    folder = pack/'animations'/f'FIGANI_{resource:03d}'
    if row['status'] != 'decoded' or digest(folder/'animation.json') != row['metadata_sha256']:
        raise ValueError('原始metadata與盤點不符')
    doc = json.loads((folder/'animation.json').read_text())
    if len(doc['frames']) != len(row['frames']):
        raise ValueError('原始幀數不符')
    result = []
    for expected, frame in zip(row['frames'], doc['frames']):
        if any(frame[k] != expected[k] for k in DESCRIPTOR):
            raise ValueError('原始描述子不同')
        if digest(folder/frame['path']) != expected['pixels_sha256'] or digest(folder/frame['mask_path']) != expected['mask_sha256']:
            raise ValueError('原始PNG檔案雜湊不同')
        with Image.open(folder/frame['path']) as source, Image.open(folder/frame['mask_path']) as mask:
            if source.mode != 'P' or source.size != (frame['width'], frame['height']) or mask.size != source.size:
                raise ValueError('原始像素形狀不符')
            points = [(i % source.width, i // source.width)
                      for i, (v, a) in enumerate(zip(source.tobytes(), mask.convert('L').tobytes()))
                      if a and 72 <= v <= 79]
            if not points:
                raise ValueError('原始藍衣參照範圍消失')
            xs, ys = zip(*points)
            anchor = [min(xs), min(ys), max(xs)+1, max(ys)+1]
            original = source.convert('RGBA'); original.putalpha(mask.convert('L'))
        result.append((expected, anchor, original))
    return row, result


def fit(picture, anchor, target, size, phase_scale=1):
    scale = (target[3]-target[1])*6/(anchor[3]-anchor[1])*phase_scale
    resized = picture.resize((max(1, round(picture.width*scale)), max(1, round(picture.height*scale))), Image.Resampling.LANCZOS)
    sx, sy = resized.width/picture.width, resized.height/picture.height
    origin = [round((target[0]+target[2])*3-(anchor[0]+anchor[2])*sx/2), round(target[3]*6-anchor[3]*sy)]
    actual = [origin[0]+anchor[0]*sx, origin[1]+anchor[1]*sy,
              origin[0]+anchor[2]*sx, origin[1]+anchor[3]*sy]
    if actual[0] < -1 or actual[1] < -1 or actual[2] > size[0]+1 or actual[3] > size[1]+1:
        raise ValueError('藍衣錨點超出原始描述子，需審查母稿')
    canvas = Image.new('RGBA', size)
    canvas.alpha_composite(resized, tuple(origin))
    if canvas.getchannel('A').getextrema() != (0, 255):
        raise ValueError('取樣後缺少透明背景或不透明核心')
    visible = resized.getchannel('A').crop((-origin[0], -origin[1], size[0]-origin[0], size[1]-origin[1]))
    clipped = sum(v >= 192 for v in resized.getchannel('A').getdata())-sum(v >= 192 for v in visible.getdata())
    return canvas, {'fit_size': list(resized.size), 'origin': origin, 'scale_xy': [sx, sy], 'phase_scale': phase_scale,
                    'fitted_blue_anchor': actual, 'clipped_high_coverage_pixels': clipped}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['pack', 'inventory', 'sources', 'out']:
        p.add_argument('--'+name, required=True, type=Path)
    p.add_argument('--base-catalog', type=Path, default=ROOT/'remake/assets/themes/modern/catalog.json')
    args = p.parse_args(); out = args.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用本人擁有的儲存庫新候選目錄')
    inventory = json.loads(args.inventory.read_text())
    if inventory.get('status') != 'PASS-SOURCE-INVENTORY' or not inventory['archive_reconstruction']['matches_pinned_original'] or digest(args.pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('未通過固定來源盤點或分離包不同')
    assignments = {12: ['idle']*4, 13: ['idle']+[f'f{i:02d}' for i in range(1,15)]}
    sources = json.loads(args.sources.read_text())
    if set(sources) != set(assignments[13]):
        raise SystemExit('需要idle與f01..f14共15張母稿')
    native = {r: native_frames(args.pack, inventory, r) for r in assignments}
    if [len(native[r][1]) for r in assignments] != [4, 15]:
        raise SystemExit('原始4／15幀範圍改變')
    if native[12][0]['frames'][0] != native[13][0]['frames'][0]:
        raise SystemExit('待機及攻擊首幀的原始來源不同')
    base_path = args.base_catalog
    base = validate(False, base_path)
    if any(a['role'] == 'battle_animation_hd_set' and a['source_resource'] in assignments for a in base['assets']):
        raise SystemExit('資源已登錄，不覆寫')
    art, fingerprints = {}, {}
    for key, value in sources.items():
        source = Path(value).resolve()
        if not source.is_relative_to(ROOT) or source.stat().st_uid != os.getuid():
            raise SystemExit('母稿須位於本人擁有的工作區')
        with Image.open(source) as im:
            if im.mode != 'RGBA' or im.getchannel('A').getextrema()[0] != 0 or im.getchannel('A').getextrema()[1] < 250:
                raise SystemExit('母稿需透明背景與高覆蓋核心')
            picture = im.copy()
        anchor = blue_anchor(picture); art[key] = (picture, anchor)
        fingerprints[key] = {'file': str(source.relative_to(ROOT)), 'sha256': digest(source), 'blue_anchor': anchor}
    old_names = names(base); private = ROOT/base['private_root']
    if any(Path(n).name != n or not (private/n).is_file() for n in old_names):
        raise SystemExit('基底清冊缺件')
    # 所有新幀先在記憶體編製通過，避免留下部分可用候選。
    follow_scale = 1
    for index, (frame, target, _) in enumerate(native[13][1]):
        if index < 11:
            continue
        _, anchor = art[f'f{index:02d}']
        height_scale = (target[3]-target[1])*6/(anchor[3]-anchor[1])
        center = (target[0]+target[2])*3
        available_width = min(center, frame['width']*6-center)*2-2
        follow_scale = min(follow_scale, available_width/((anchor[2]-anchor[0])*height_scale))
    prepared = []
    for resource, keys in assignments.items():
        for index, (key, (frame, target, original)) in enumerate(zip(keys, native[resource][1])):
            picture, anchor = art[key]
            canvas, recipe = fit(picture, anchor, target, (frame['width']*6, frame['height']*6), follow_scale if resource == 13 and index >= 11 else 1)
            prepared.append((resource, index, key, frame, target, canvas, original, recipe))
    out.mkdir(); pack = out/'pack'; pack.mkdir()
    shutil.copyfile(base_path, out/'base-catalog.json')
    for name in sorted(old_names): (pack/name).symlink_to(os.path.relpath(private/name, pack))
    source_names = {}
    for key, value in sources.items():
        name = f'ares-battle-{key}-hd6-master-v1.png'; shutil.copyfile(value, pack/name); source_names[key] = name
    candidate = copy.deepcopy(base); candidate['private_root'] = str(pack.relative_to(ROOT))
    records = []; sheet = Image.new('RGB', (1920, 2000), (35, 40, 48)); draw = ImageDraw.Draw(sheet)
    for i, (resource, index, key, frame, target, canvas, original, recipe) in enumerate(prepared):
        name = f'figani-{resource:03d}-hd6-v1-f{index:02d}.png'; canvas.save(pack/name)
        records.append({'resource': resource, 'frame': index, 'pose': key, 'file': name, 'sha256': digest(pack/name),
                        'native_blue_anchor': target, 'native_descriptor': {k: frame[k] for k in DESCRIPTOR}, **recipe})
        x, y = i % 4*480, i//4*400
        for dy, picture, scale, label in [(0, original, 1, 'native source'), (200, canvas, 6, 'HD candidate')]:
            stage = Image.new('RGBA', (320*scale, 200*scale)); stage.alpha_composite(picture, (frame['x']*scale, frame['y']*scale))
            thumb = stage.resize((320, 200), Image.Resampling.LANCZOS)
            sheet.paste(thumb, (x, y+dy), thumb); draw.text((x+4, y+dy+4), f'{resource:03d}/{index:02d} {label}', fill='white')
    for resource, keys in assignments.items():
        row = native[resource][0]; selected = [r for r in records if r['resource'] == resource]
        used = sorted(set(keys))
        candidate['assets'].append({'asset_id': f'modern.figani.{resource:03d}.hd6.v1', 'role': 'battle_animation_hd_set',
            'status': 'runtime_candidate', 'files': [r['file'] for r in selected], 'width': 1920, 'height': 1200,
            'frame_count': len(selected), 'frame_sha256': [r['sha256'] for r in selected],
            'consumer_contract': 'native_figani_rgba_hd6_v1', 'alpha_contract': 'coverage',
            'source_refs': [f'asset:FIGANI_{resource:03d}', 'docs/data/fd2-reference-files.json'],
            'source_archive': 'FIGANI.DAT', 'source_resource': resource, 'source_metadata_sha256': row['metadata_sha256'],
            'native_header': [row['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']],
            'native_frames': [r['native_descriptor'] for r in selected], 'cycle_policy': 'native_descriptor_key_poses',
            'layout_policy': 'contain_native_descriptor_hd6_candidate', 'source_files': [source_names[k] for k in used],
            'source_sha256': [digest(pack/source_names[k]) for k in used]})
    path = out/'catalog.json'; path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2)+'\n'); validate(True, path)
    sheet.save(out/'review.png')
    runtime = out/'runtime'; (runtime/'assets/themes/modern').mkdir(parents=True)
    for source in (ROOT/'remake/assets').iterdir():
        if source.name != 'themes': (runtime/'assets'/source.name).symlink_to(os.path.relpath(source, runtime/'assets'))
    (runtime/'assets/themes/modern/catalog.json').symlink_to(os.path.relpath(path, runtime/'assets/themes/modern'))
    report = {'schema_version': 1, 'status': 'PASS-COMPILED-ARES-CANDIDATE', 'runtime_registered': False,
        'scope': '15母稿重定向完整4／15幀；未驗GUI，非原版逐幀視覺對拍。',
        'base_catalog_sha256': digest(base_path), 'catalog_sha256': digest(path), 'inventory_sha256': digest(args.inventory),
        'pack_manifest_sha256': inventory['pack_manifest_sha256'], 'compiler_sha256': digest(Path(__file__)),
        'blue_anchor_predicate': 'alpha>=192; B>=70; B*100>=R*175; B*100>=G*145; four-connected components>=1000 source pixels',
        'sampling': 'LANCZOS; uniform height fit; blue anchor center and bottom; shared contain scale for follow frames11..14; native descriptor clip',
        'follow_phase_scale': follow_scale,
        'sources': fingerprints, 'frames': records, 'review_sha256': digest(out/'review.png')}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'frames': len(records), 'catalog_assets': len(candidate['assets'])}))


if __name__ == '__main__':
    main()
