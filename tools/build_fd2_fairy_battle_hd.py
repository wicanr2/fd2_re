#!/usr/bin/env python3
"""按原始分層與遮罩，以四母稿編製風妖精14幀隔離候選。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageDraw, ImageFilter
from build_fd2_seraph_battle_hd import sha
from build_fd2_gaia_battle_hd import names
from validate_modern_theme_catalog import ROOT, validate
from build_modern_hd_map_sprites import master_components


def fit(picture, box):
    """只裁透明外圍，等比例對齊來源可見框中心，不改coverage。"""
    picture = picture.crop(picture.getchannel('A').getbbox())
    scale = min((box[2]-box[0])/picture.width, (box[3]-box[1])/picture.height)
    picture = picture.resize((max(1, round(picture.width*scale)), max(1, round(picture.height*scale))), Image.Resampling.LANCZOS)
    return picture, (box[0]+(box[2]-box[0]-picture.width)//2, box[1]+(box[3]-box[1]-picture.height)//2)


def parts(mask):
    """以最大空白欄間距，分開原始手臂與遠端小手。"""
    core = mask.point(lambda value: 255 if value >= 128 else 0)
    columns = [x for x in range(core.width) if core.crop((x, 0, x+1, core.height)).getbbox()]
    if len(columns) < 2:
        raise ValueError('缺少兩個手勢區域')
    left, right = max(zip(columns, columns[1:]), key=lambda pair: pair[1]-pair[0])
    # 原生mask在320世界內，不能把兩側空白算進手勢的跨度。
    if right-left < (columns[-1]-columns[0]+1)*.15:
        raise ValueError('手臂與小手之間沒有足夠間隔')
    split = (left+right)//2
    boxes = []
    for a, b in [(0, split), (split, mask.width)]:
        box = core.crop((a, 0, b, core.height)).getbbox()
        if box is None:
            raise ValueError('手勢區域為空')
        boxes.append((box[0]+a, box[1], box[2]+a, box[3]))
    return boxes


def source_parts(picture):
    """沿既有核心128／柔邊3契約，排除淡像素橋接而保留選中輪廓的coverage。"""
    boxes = parts(picture.getchannel('A'))
    results = []
    for box in boxes:
        expanded = (max(0, box[0]-3), max(0, box[1]-3), min(picture.width, box[2]+3), min(picture.height, box[3]+3))
        fragment, tight = master_components(picture.crop(expanded), 1)[0]
        results.append((fragment, (expanded[0]+tight[0], expanded[1]+tight[1], expanded[0]+tight[2], expanded[1]+tight[3])))
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['pack', 'inventory', 'body', 'raised', 'open', 'wind', 'out']:
        parser.add_argument('--'+name, required=True, type=Path)
    parser.add_argument('--base-catalog', type=Path, default=ROOT/'remake/assets/themes/modern/catalog.json')
    args = parser.parse_args()
    out = args.out.resolve()
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用儲存庫內本人擁有的新候選目錄')
    inventory = json.loads(args.inventory.read_text())
    if inventory.get('status') != 'PASS-COMPOUND-SOURCE' or sha(args.pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('來源盤點未通過或manifest不同')
    row = next(x for x in inventory['resources'] if x['resource'] == 66)
    folder = args.pack/'animations/FDOTHER_066'
    if sha(folder/'animation.json') != row['metadata_sha256']:
        raise SystemExit('來源metadata不同')
    native = json.loads((folder/'animation.json').read_text())
    if len(native['frames']) != 14 or len(row['frames']) != 14:
        raise SystemExit('原始14幀契約不同')
    pixels, masks, originals = [], [], []
    for frame, proof in zip(native['frames'], row['frames']):
        for key, label in [('path', 'pixels_png_sha256'), ('mask_path', 'mask_png_sha256')]:
            if sha(folder/frame[key]) != proof[label]:
                raise SystemExit('原始PNG不同')
        with Image.open(folder/frame['path']) as image, Image.open(folder/frame['mask_path']) as alpha:
            stage = Image.new('L', (320, 200))
            stage.paste(Image.frombytes('L', image.size, image.tobytes()), (frame['x'], frame['y']))
            mask = Image.new('L', (320, 200))
            mask.paste(alpha, (frame['x'], frame['y']))
            picture = image.convert('RGBA'); picture.putalpha(alpha)
        pixels.append(stage.tobytes()); masks.append(mask.tobytes())
        original = Image.new('RGBA', (320, 200)); original.alpha_composite(picture, (frame['x'], frame['y']))
        originals.append(original)
    if any(pixels[n] != pixels[1] or masks[n] != masks[1] for n in [2, 3]):
        raise SystemExit('前三個手勢不再相同')
    proofs = []
    for index in range(6, 14):
        visible = [i for i, (value, alpha) in enumerate(zip(pixels[index], masks[index])) if alpha and value != 200]
        if any(not masks[5][i] or pixels[5][i] != pixels[index][i] for i in visible):
            raise SystemExit('後段存在非風球的新手勢，停止編製')
        # 每個基底手勢像素須保持或由原始風球覆蓋，不能靜默漏掉。
        if any(not masks[index][i] or pixels[index][i] not in (value, 200) for i, value in enumerate(pixels[5]) if masks[5][i]):
            raise SystemExit('後段手勢消失且不是風球覆蓋')
        proofs.append({'frame': index, 'visible_arm_pixels': len(visible), 'wind_pixels': sum(bool(a) and v == 200 for v, a in zip(pixels[index], masks[index]))})
    art = {}
    for label in ['body', 'raised', 'open', 'wind']:
        with Image.open(getattr(args, label)) as image:
            picture = image.convert('RGBA')
            want = (255, 255) if label == 'wind' else (0, 255)
            if picture.getchannel('A').getextrema() != want:
                raise SystemExit('母稿alpha不符：'+label)
            art[label] = picture.copy()
    first = native['frames'][0]
    body_box = (first['x']*6, first['y']*6, (first['x']+first['width'])*6, (first['y']+first['height'])*6)
    body, body_origin = fit(art['body'], body_box)
    body_stage = Image.new('RGBA', (1920, 1200)); body_stage.alpha_composite(body, body_origin)
    pose_stages, pose_records = {}, {}
    for index, label in [(1, 'raised'), (4, 'open'), (5, 'open')]:
        native_boxes = parts(Image.frombytes('L', (320, 200), masks[index]))
        fragments = source_parts(art[label])
        stage = Image.new('RGBA', (1920, 1200)); details = []
        for (source_fragment, source_box), box in zip(fragments, native_boxes):
            fragment, origin = fit(source_fragment, tuple(v*6 for v in box))
            stage.alpha_composite(fragment, origin)
            details.append({'source_box': list(source_box), 'native_box': list(box), 'hd_origin': list(origin), 'hd_size': list(fragment.size)})
        pose_stages[index], pose_records[index] = stage, details
    texture = art['wind'].resize((1920, 1200), Image.Resampling.LANCZOS)
    frames, records = [body_stage], [{'frame': 0, 'source': 'body', 'hd_origin': list(body_origin), 'hd_size': list(body.size)}]
    for index in range(1, 14):
        pose = 1 if index <= 3 else 4 if index == 4 else 5
        stage = pose_stages[pose].copy()
        wind = Image.frombytes('L', (320, 200), bytes(255 if a and v == 200 else 0 for v, a in zip(pixels[index], masks[index])))
        # 小風球及超出畫布的大風球沿同一原始世界mask，不按每幀外框另縮材質。
        alpha = wind.filter(ImageFilter.GaussianBlur(.65)).resize((1920, 1200), Image.Resampling.LANCZOS)
        effect = texture.copy(); effect.putalpha(alpha); stage.alpha_composite(effect)
        frames.append(stage)
        records.append({'frame': index, 'pose': pose, 'source': 'raised' if pose == 1 else 'open', 'parts': pose_records[pose], 'wind_native_pixels': sum(v == 200 and bool(a) for v, a in zip(pixels[index], masks[index]))})
    catalog = copy.deepcopy(validate(False, args.base_catalog))
    catalog['assets'] = [x for x in catalog['assets'] if not (x['role'] == 'battle_effect_hd_set' and x['source_resource'] == 66)]
    private = ROOT/catalog['private_root']; old_names = names(catalog)
    if any(Path(name).name != name or not (private/name).is_file() for name in old_names):
        raise SystemExit('既有清冊缺件')
    out.mkdir(); pack = out/'pack'; pack.mkdir()
    for name in sorted(old_names): (pack/name).symlink_to(os.path.relpath(private/name, pack))
    source_files = [f'fairy-{label}-hd6-master-v1.png' for label in ['body', 'raised', 'open', 'wind']]
    for label, name in zip(['body', 'raised', 'open', 'wind'], source_files): shutil.copyfile(getattr(args, label), pack/name)
    files = []
    sheet = Image.new('RGB', (2240, 800), (30, 35, 42)); draw = ImageDraw.Draw(sheet)
    for index, (stage, frame) in enumerate(zip(frames, native['frames'])):
        name = f'fdother-066-hd6-v1-f{index:02d}.png'
        stage.crop((frame['x']*6, frame['y']*6, (frame['x']+frame['width'])*6, (frame['y']+frame['height'])*6)).save(pack/name)
        files.append(name); records[index].update(file=name, sha256=sha(pack/name))
        modern = stage.copy()
        original = originals[index].copy()
        if index:
            modern = body_stage.copy(); modern.alpha_composite(stage)
            original = originals[0].copy(); original.alpha_composite(originals[index])
        x, y = (index%7)*320, (index//7)*400
        thumb = modern.resize((320, 200), Image.Resampling.LANCZOS)
        sheet.paste(original, (x, y), original); sheet.paste(thumb, (x, y+200), thumb)
        draw.text((x+4, y+4), f'native source 0+{index}', fill='white'); draw.text((x+4, y+204), f'HD candidate 0+{index}', fill='white')
    sheet.save(out/'review.png')
    catalog['private_root'] = str(pack.relative_to(ROOT))
    catalog['assets'].append({'asset_id': 'modern.fdother.066.hd6.v1', 'role': 'battle_effect_hd_set', 'status': 'runtime_candidate',
        'files': files, 'width': 1920, 'height': 1200, 'frame_count': 14, 'frame_sha256': [sha(pack/n) for n in files],
        'consumer_contract': 'native_fdother_compound_rgba_hd6_v1', 'alpha_contract': 'coverage',
        'source_refs': ['asset:FDOTHER_066', 'docs/data/fd2-reference-files.json'], 'source_archive': 'FDOTHER.DAT', 'source_resource': 66,
        'source_metadata_sha256': row['metadata_sha256'], 'native_header': [row['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']],
        'native_frames': [{k: f[k] for k in ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']} for f in row['frames']],
        'cycle_policy': 'native_compound_frames', 'layout_policy': 'native_descriptor_hd6', 'source_files': source_files, 'source_sha256': [sha(pack/n) for n in source_files]})
    path = out/'catalog.json'; path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n'); validate(True, path)
    report = {'status': 'PASS-COMPILED-FAIRY-CANDIDATE', 'runtime_registered': False, 'scope': '四母稿重定向14個原始圖層；伸手4／5共用母稿，非14張獨立畫作，非GUI或原版執行收據。',
        'base_catalog_sha256': sha(args.base_catalog), 'catalog_sha256': sha(path), 'inventory_sha256': sha(args.inventory), 'compiler_sha256': sha(Path(__file__)),
        'source_sha256': dict(zip(source_files, [sha(pack/n) for n in source_files])), 'arm_source_proof': proofs, 'frames': records, 'review_sha256': sha(out/'review.png'), 'fx_edge_softness_native': .65,
        'instance_core_alpha': 128, 'coverage_edge_radius': 3, 'source_alpha_preserved_within_selected_contour': True}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'frames': len(files), 'catalog_assets': len(catalog['assets'])}))


if __name__ == '__main__':
    main()
