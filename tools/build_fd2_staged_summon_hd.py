#!/usr/bin/env python3
"""以原始遮罩與兩母稿編製FDOTHER67或68的九幀隔離候選。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageDraw, ImageFilter
from build_fd2_fairy_battle_hd import fit
from build_fd2_seraph_battle_hd import sha
from build_fd2_gaia_battle_hd import names
from validate_modern_theme_catalog import ROOT, validate


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--resource', required=True, type=int, choices=[67, 68])
    for name in ['pack', 'inventory', 'body', 'energy', 'out']:
        parser.add_argument('--'+name, required=True, type=Path)
    parser.add_argument('--base-catalog', type=Path, default=ROOT/'remake/assets/themes/modern/catalog.json')
    args = parser.parse_args(); out = args.out.resolve(); resource = args.resource
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid != os.getuid():
        raise SystemExit('使用本人擁有的儲存庫新候選目錄')
    inventory = json.loads(args.inventory.read_text())
    if inventory.get('status') != 'PASS-COMPOUND-SOURCE' or sha(args.pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('未通過來源盤點或manifest不同')
    row = next(x for x in inventory['resources'] if x['resource'] == resource)
    folder = args.pack/f'animations/FDOTHER_{resource:03d}'
    if sha(folder/'animation.json') != row['metadata_sha256']:
        raise SystemExit('原始metadata不同')
    native = json.loads((folder/'animation.json').read_text())
    if len(native['frames']) != 9 or len(row['frames']) != 9:
        raise SystemExit('原始九幀契約不同')
    pixels, masks, originals = [], [], []
    for frame, proof in zip(native['frames'], row['frames']):
        for key, label in [('path', 'pixels_png_sha256'), ('mask_path', 'mask_png_sha256')]:
            if sha(folder/frame[key]) != proof[label]: raise SystemExit('原始PNG不同')
        with Image.open(folder/frame['path']) as picture, Image.open(folder/frame['mask_path']) as alpha:
            stage = Image.new('L', (320, 200)); stage.paste(Image.frombytes('L', picture.size, picture.tobytes()), (frame['x'], frame['y']))
            mask = Image.new('L', (320, 200)); mask.paste(alpha, (frame['x'], frame['y']))
            original = picture.convert('RGBA'); original.putalpha(alpha)
        pixels.append(stage.tobytes()); masks.append(mask.tobytes())
        world = Image.new('RGBA', (320, 200)); world.alpha_composite(original, (frame['x'], frame['y'])); originals.append(world)
    body_points = [i for i, value in enumerate(pixels[0]) if value and masks[0][i]]
    shifts = [0]*8+[1] if resource == 68 else [0]*9
    proofs = []
    for index in range(9):
        if resource == 67 and index:
            if any(value != 32 for value, alpha in zip(pixels[index], masks[index]) if alpha):
                raise SystemExit('球體有未知色索引，停止編製')
            proofs.append({'frame': index, 'effect_index': 32, 'effect_pixels': sum(bool(a) for a in masks[index])})
        elif resource == 68:
            dy = shifts[index]; visible = [i for i in body_points if i//320+dy < 200]
            if any(pixels[index][i+dy*320] != pixels[0][i] or not masks[index][i+dy*320] for i in visible):
                raise SystemExit('原始身體重用／位移不符')
            proofs.append({'frame': index, 'dy_native': dy, 'matched_body_pixels': len(visible), 'clipped_body_pixels': len(body_points)-len(visible)})
    with Image.open(args.body) as source:
        body = source.convert('RGBA')
        if body.getchannel('A').getextrema() != (0, 255): raise SystemExit('身體母稿alpha不符')
    with Image.open(args.energy) as source:
        if source.convert('RGBA').getchannel('A').getextrema() != (255, 255): raise SystemExit('FX材質須不透明')
        texture = source.convert('RGBA').resize((1920, 1200), Image.Resampling.LANCZOS)
    first = native['frames'][0]
    body, origin = fit(body, (first['x']*6, first['y']*6, (first['x']+first['width'])*6, (first['y']+first['height'])*6))
    frames, records = [], []
    for index in range(9):
        dy = shifts[index]
        body_mask = {i+dy*320 for i in body_points if i//320+dy < 200}
        fx = [bool(alpha) and (value == 32 if resource == 67 else value != 0 and i not in body_mask)
              for i, (value, alpha) in enumerate(zip(pixels[index], masks[index]))]
        if resource == 67 and index == 0: fx = [False]*64000
        alpha = Image.frombytes('L', (320, 200), bytes(255 if active else 0 for active in fx))
        # 不透明全畫面差分保持255，不由世界邊界外的透明值污染。
        # 球體直接抗鋸齒取樣，額外Gaussian會把11×11首幀的全透明角落抹成alpha1。
        if resource == 68:
            alpha = alpha.filter(ImageFilter.GaussianBlur(.65))
        alpha = alpha.resize((1920, 1200), Image.Resampling.LANCZOS)
        stage = texture.copy(); stage.putalpha(alpha)
        if resource == 68 or index == 0: stage.alpha_composite(body, (origin[0], origin[1]+dy*6))
        frames.append(stage)
        records.append({'frame': index, 'dy_native': dy, 'body_origin_hd': [origin[0], origin[1]+dy*6] if resource == 68 or index == 0 else None, 'fx_native_pixels': sum(fx)})
    catalog = copy.deepcopy(validate(False, args.base_catalog))
    catalog['assets'] = [x for x in catalog['assets'] if not (x['role'] == 'battle_effect_hd_set' and x['source_resource'] == resource)]
    private = ROOT/catalog['private_root']; old_names = names(catalog)
    if any(Path(n).name != n or not (private/n).is_file() for n in old_names): raise SystemExit('既有清冊缺件')
    out.mkdir(); pack = out/'pack'; pack.mkdir()
    for name in sorted(old_names): (pack/name).symlink_to(os.path.relpath(private/name, pack))
    label = 'destroyer' if resource == 67 else 'demon'
    sources = [f'{label}-body-hd6-master-v1.png', f'{label}-energy-hd6-master-v1.png']
    for source, name in zip([args.body, args.energy], sources): shutil.copyfile(source, pack/name)
    files = []; sheet = Image.new('RGB', (2880, 400), (30, 35, 42)); draw = ImageDraw.Draw(sheet)
    for index, (stage, frame) in enumerate(zip(frames, native['frames'])):
        name = f'fdother-{resource:03d}-hd6-v1-f{index:02d}.png'
        stage.crop((frame['x']*6, frame['y']*6, (frame['x']+frame['width'])*6, (frame['y']+frame['height'])*6)).save(pack/name)
        files.append(name); records[index].update(file=name, sha256=sha(pack/name))
        modern, original = stage, originals[index]
        if resource == 67 and index:
            modern = frames[0].copy(); modern.alpha_composite(stage)
            original = originals[0].copy(); original.alpha_composite(originals[index])
        thumb = modern.resize((320, 200), Image.Resampling.LANCZOS)
        sheet.paste(original, (index*320, 0), original); sheet.paste(thumb, (index*320, 200), thumb)
        draw.text((index*320+4, 4), f'native source {index}', fill='white'); draw.text((index*320+4, 204), f'HD candidate {index}', fill='white')
    sheet.save(out/'review.png')
    catalog['private_root'] = str(pack.relative_to(ROOT))
    catalog['assets'].append({'asset_id': f'modern.fdother.{resource:03d}.hd6.v1', 'role': 'battle_effect_hd_set', 'status': 'runtime_candidate',
        'files': files, 'width': 1920, 'height': 1200, 'frame_count': 9, 'frame_sha256': [sha(pack/n) for n in files],
        'consumer_contract': 'native_fdother_compound_rgba_hd6_v1', 'alpha_contract': 'coverage', 'source_refs': [f'asset:FDOTHER_{resource:03d}', 'docs/data/fd2-reference-files.json'],
        'source_archive': 'FDOTHER.DAT', 'source_resource': resource, 'source_metadata_sha256': row['metadata_sha256'],
        'native_header': [row['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']],
        'native_frames': [{k: f[k] for k in ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']} for f in row['frames']],
        'cycle_policy': 'native_compound_frames', 'layout_policy': 'native_descriptor_hd6', 'source_files': sources, 'source_sha256': [sha(pack/n) for n in sources]})
    path = out/'catalog.json'; path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n'); validate(True, path)
    report = {'status': 'PASS-COMPILED-STAGED-SUMMON', 'runtime_registered': False, 'resource': resource, 'scope': '兩母稿與原始遮罩的九幀重定向，非九張獨立繪畫或GUI。',
        'base_catalog_sha256': sha(args.base_catalog), 'catalog_sha256': sha(path), 'inventory_sha256': sha(args.inventory), 'compiler_sha256': sha(Path(__file__)),
        'source_sha256': dict(zip(sources, [sha(pack/n) for n in sources])), 'source_proof': proofs, 'body_size_hd': list(body.size), 'frames': records,
        'review_sha256': sha(out/'review.png'), 'fx_edge_softness_native': .65 if resource == 68 else 0,
        'fx_sampling': 'native_world_mask_lanczos6'}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'resource': resource, 'frames': 9, 'catalog_assets': len(catalog['assets'])}))


if __name__ == '__main__':
    main()
