#!/usr/bin/env python3
"""以明示母稿／錨點配方編製完整FIGANI候選，不登錄正式清冊。"""
import argparse
import copy
import json
import os
from pathlib import Path
import re
import shutil

from PIL import Image, ImageDraw
from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate


DESCRIPTOR = ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']
KEY = re.compile(r'[a-z][a-z0-9_-]{0,39}')


def rectangle(value, size):
    if (not isinstance(value, list) or len(value) != 4 or any(type(v) is not int for v in value)
        or not 0 <= value[0] < value[2] <= size[0] or not 0 <= value[1] < value[3] <= size[1]):
        raise ValueError('錨點須是畫布內的非空整數矩形')
    return value


def load_native(pack, inventory, resource):
    row = next(r for r in inventory['resources'] if r['resource'] == resource)
    folder = pack/'animations'/f'FIGANI_{resource:03d}'
    if row['status'] != 'decoded' or digest(folder/'animation.json') != row['metadata_sha256']:
        raise ValueError('原始動畫未解碼或metadata不同')
    doc = json.loads((folder/'animation.json').read_text())
    if len(doc['frames']) != len(row['frames']): raise ValueError('原始幀數不同')
    pictures = []
    for frame, proof in zip(doc['frames'], row['frames']):
        if any(frame[k] != proof[k] for k in DESCRIPTOR): raise ValueError('原始描述子不同')
        if digest(folder/frame['path']) != proof['pixels_sha256'] or digest(folder/frame['mask_path']) != proof['mask_sha256']:
            raise ValueError('原始PNG檔案雜湊不同')
        with Image.open(folder/frame['path']) as picture, Image.open(folder/frame['mask_path']) as mask:
            if picture.mode != 'P' or picture.size != (frame['width'], frame['height']) or mask.size != picture.size:
                raise ValueError('原始像素形狀不同')
            original = picture.convert('RGBA'); original.putalpha(mask.convert('L')); pictures.append(original)
    return row, pictures


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['pack', 'inventory', 'recipe', 'out']:
        p.add_argument('--'+name, required=True, type=Path)
    p.add_argument('--base-catalog', type=Path, default=ROOT/'remake/assets/themes/modern/catalog.json')
    args = p.parse_args(); out = args.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用本人擁有的工作區新候選目錄')
    inventory = json.loads(args.inventory.read_text()); recipe = json.loads(args.recipe.read_text())
    if recipe.get('schema_version') != 1 or set(recipe) != {'schema_version', 'figure', 'inventory_sha256', 'sources', 'resources'}:
        raise SystemExit('未知配方版本或欄位')
    if (inventory.get('status') != 'PASS-SOURCE-INVENTORY' or not inventory['archive_reconstruction']['matches_pinned_original']
        or digest(args.pack/'manifest.json') != inventory['pack_manifest_sha256']
        or recipe.get('inventory_sha256') != digest(args.inventory)):
        raise SystemExit('盤點、配方或原版包指紋不符')
    figure = recipe['figure']
    if type(figure) is not int or not 0 <= figure <= 135: raise SystemExit('人物資源範圍不符')
    resources = [figure*3, figure*3+1]
    if set(recipe['resources']) != {str(r) for r in resources}: raise SystemExit('配方須涵蓋完整待機及攻擊資源')
    base = validate(False, args.base_catalog)
    if any(a['role'] == 'battle_animation_hd_set' and a['source_resource'] in resources for a in base['assets']):
        raise SystemExit('所選圖源已登錄，不覆寫')
    art, source_records = {}, {}
    for key, value in recipe['sources'].items():
        if not KEY.fullmatch(key) or set(value) != {'file', 'sha256', 'anchor'}: raise SystemExit('母稿名稱或欄位不符')
        path = Path(value['file']); path = (ROOT/path).resolve() if not path.is_absolute() else path.resolve()
        if not path.is_relative_to(ROOT) or path.stat().st_uid != os.getuid() or digest(path) != value['sha256']:
            raise SystemExit('母稿路徑、擁有權或SHA不符')
        with Image.open(path) as im:
            if im.mode != 'RGBA' or im.getchannel('A').getextrema()[0] != 0 or im.getchannel('A').getextrema()[1] < 250:
                raise SystemExit('母稿缺少透明背景或高覆蓋核心')
            picture = im.copy()
        anchor = rectangle(value['anchor'], picture.size); art[key] = (picture, anchor)
        source_records[key] = {'file': str(path.relative_to(ROOT)), 'sha256': digest(path), 'anchor': anchor}
    natives, pending, used = {}, [], set()
    for resource in resources:
        row, originals = load_native(args.pack, inventory, resource); natives[resource] = row
        assignments = recipe['resources'][str(resource)]
        if len(assignments) != len(row['frames']): raise SystemExit('配方影格不完整')
        for index, (frame, assignment, original) in enumerate(zip(row['frames'], assignments, originals)):
            if set(assignment) != {'pose', 'anchor', 'scale_group'} or assignment['pose'] not in art or not KEY.fullmatch(assignment['scale_group']):
                raise SystemExit('未知母稿、群組或配方欄位')
            key = assignment['pose']; used.add(key); picture, anchor = art[key]
            target = rectangle(assignment['anchor'], (frame['width'], frame['height']))
            scale = (target[3]-target[1])*6/(anchor[3]-anchor[1])
            center = (target[0]+target[2])*3
            available = min(center, frame['width']*6-center)*2-2
            limit = min(1, available/((anchor[2]-anchor[0])*scale))
            if limit <= 0: raise SystemExit('無法在原矩形容納錨點')
            pending.append((resource, index, key, frame, original, picture, anchor, target, scale, assignment['scale_group'], limit))
    if used != set(art): raise SystemExit('配方含未使用母稿')
    groups = {}
    for item in pending: groups[item[9]] = min(groups.get(item[9], 1), item[10])
    prepared = []
    for resource, index, key, frame, original, picture, anchor, target, scale, group, _ in pending:
        scale *= groups[group]
        fitted = picture.resize((max(1, round(picture.width*scale)), max(1, round(picture.height*scale))), Image.Resampling.LANCZOS)
        sx, sy = fitted.width/picture.width, fitted.height/picture.height
        origin = [round((target[0]+target[2])*3-(anchor[0]+anchor[2])*sx/2), round(target[3]*6-anchor[3]*sy)]
        actual = [origin[0]+anchor[0]*sx, origin[1]+anchor[1]*sy, origin[0]+anchor[2]*sx, origin[1]+anchor[3]*sy]
        size = (frame['width']*6, frame['height']*6)
        if actual[0] < -1 or actual[1] < -1 or actual[2] > size[0]+1 or actual[3] > size[1]+1:
            raise SystemExit('取樣後錨點越界，停止審查')
        canvas = Image.new('RGBA', size); canvas.alpha_composite(fitted, tuple(origin))
        native_empty = original.getchannel('A').getbbox() is None
        if native_empty:
            # 原始Mask已由load_native核SHA；空白present仍保留完整PNG與排程。
            canvas = Image.new('RGBA', size)
        elif canvas.getchannel('A').getextrema() != (0, 255):
            raise SystemExit('成品缺少透明背景或不透明核心')
        visible = fitted.getchannel('A').crop((-origin[0], -origin[1], size[0]-origin[0], size[1]-origin[1]))
        clipped = 0 if native_empty else sum(a >= 192 for a in fitted.getchannel('A').getdata())-sum(a >= 192 for a in visible.getdata())
        record = {'resource': resource, 'frame': index, 'pose': key, 'native_anchor': target, 'source_anchor': anchor,
            'fit_size': list(fitted.size), 'origin': origin, 'scale_xy': [sx, sy], 'scale_group': group,
            'phase_scale': groups[group], 'fitted_anchor': actual, 'clipped_high_coverage_pixels': clipped,
            'native_empty_source': native_empty, 'pose_used': not native_empty,
            'native_descriptor': {k: frame[k] for k in DESCRIPTOR}}
        prepared.append((record, canvas, original))
    private = ROOT/base['private_root']; old_names = names(base)
    if any(Path(n).name != n or not (private/n).is_file() for n in old_names): raise SystemExit('基底清冊缺件')
    out.mkdir(); pack = out/'pack'; pack.mkdir(); shutil.copyfile(args.base_catalog, out/'base-catalog.json')
    for name in sorted(old_names): (pack/name).symlink_to(os.path.relpath(private/name, pack))
    source_names = {}
    for key, value in source_records.items():
        name = f'figure-{figure:03d}-{key}-battle-hd6-master-v1.png'; shutil.copyfile(ROOT/value['file'], pack/name); source_names[key] = name
    sheet = Image.new('RGB', (1600, ((len(prepared)+3)//4)*400), (35, 40, 48)); draw = ImageDraw.Draw(sheet)
    records = []
    for i, (record, canvas, original) in enumerate(prepared):
        resource, index = record['resource'], record['frame']; frame = record['native_descriptor']
        name = f'figani-{resource:03d}-hd6-v1-f{index:02d}.png'; canvas.save(pack/name)
        record.update(file=name, sha256=digest(pack/name)); records.append(record)
        x, y = i%4*400, i//4*400
        for dy, picture, scale, label in [(0, original, 1, 'native source'), (200, canvas, 6, 'HD candidate')]:
            stage = Image.new('RGBA', (320*scale, 200*scale)); stage.alpha_composite(picture, (frame['x']*scale, frame['y']*scale))
            thumb = stage.resize((320, 200), Image.Resampling.LANCZOS); sheet.paste(thumb, (x, y+dy), thumb)
            draw.text((x+4, y+dy+4), f'{resource:03d}/{index:02d} {label}', fill='white')
    candidate = copy.deepcopy(base); candidate['private_root'] = str(pack.relative_to(ROOT))
    for resource in resources:
        row = natives[resource]; selected = [r for r in records if r['resource'] == resource]; keys = sorted({r['pose'] for r in selected})
        candidate['assets'].append({'asset_id': f'modern.figani.{resource:03d}.hd6.v1', 'role': 'battle_animation_hd_set',
            'status': 'runtime_candidate', 'files': [r['file'] for r in selected], 'width': 1920, 'height': 1200,
            'frame_count': len(selected), 'frame_sha256': [r['sha256'] for r in selected],
            'consumer_contract': 'native_figani_rgba_hd6_v1', 'alpha_contract': 'coverage',
            'source_refs': [f'asset:FIGANI_{resource:03d}', 'docs/data/fd2-reference-files.json'],
            'source_archive': 'FIGANI.DAT', 'source_resource': resource, 'source_metadata_sha256': row['metadata_sha256'],
            'native_header': [row['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']],
            'native_frames': [r['native_descriptor'] for r in selected], 'cycle_policy': 'native_descriptor_key_poses',
            'layout_policy': 'contain_native_descriptor_hd6_candidate', 'source_files': [source_names[k] for k in keys],
            'source_sha256': [digest(pack/source_names[k]) for k in keys]})
    path = out/'catalog.json'; path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2)+'\n'); validate(True, path)
    sheet.save(out/'review.png'); runtime = out/'runtime'; (runtime/'assets/themes/modern').mkdir(parents=True)
    for source in (ROOT/'remake/assets').iterdir():
        if source.name != 'themes': (runtime/'assets'/source.name).symlink_to(os.path.relpath(source, runtime/'assets'))
    (runtime/'assets/themes/modern/catalog.json').symlink_to(os.path.relpath(path, runtime/'assets/themes/modern'))
    report = {'status': 'PASS-COMPILED-CHARACTER-CANDIDATE', 'runtime_registered': False, 'figure': figure,
        'base_catalog_sha256': digest(args.base_catalog), 'catalog_sha256': digest(path), 'compiler_sha256': digest(Path(__file__)),
        'inventory_sha256': digest(args.inventory), 'recipe_sha256': digest(args.recipe), 'sources': source_records,
        'frames': records, 'scale_groups': groups, 'review_sha256': digest(out/'review.png'),
        'scope': '明示錨點與姿勢重定向的完整候選，不代表正常GUI或原版逐幀視覺相同。'}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'figure': figure, 'frames': len(records), 'catalog_assets': len(candidate['assets'])}))


if __name__ == '__main__': main()
