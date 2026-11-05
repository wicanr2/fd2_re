#!/usr/bin/env python3
"""只產生HD1貼圖縮圖對照；不修改清冊、母稿或正式執行期。"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def build_gui_overview(args):
    receipt_path = args.gui/'receipt.json'
    receipt = json.loads(receipt_path.read_text())
    assert receipt['status'] == 'PASS'
    assert not args.out.exists()
    args.out.mkdir()
    canvas = Image.new('RGB', (1560, 800), '#101e32')
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.truetype(str(args.font), 28, index=3)
    small = ImageFont.truetype(str(args.font), 18, index=3)
    sources = []
    for col, (quality, label) in enumerate([('full', '完整'), ('balanced', '均衡，預設'), ('light', '輕量')]):
        name = f'quality-menu-{quality}.png'
        source = args.gui/name
        raw = source.read_bytes()
        digest = hashlib.sha256(raw).hexdigest()
        evidence = receipt['screenshots'][name]
        assert digest == evidence['sha256']
        assert evidence['state']['hd_texture_quality'] == quality
        with Image.open(source) as im:
            picture = im.convert('RGB')
        assert picture.size == (1920, 1272)
        x = 20 + col*520
        draw.text((x, 16), label, font=font, fill='#f2d995')
        canvas.paste(picture.resize((500, 331), Image.Resampling.LANCZOS), (x, 64))
        detail = picture.crop((1060, 70, 1920, 610))
        canvas.paste(detail.resize((500, 314), Image.Resampling.LANCZOS), (x, 420))
        sources.append({'quality': quality, 'file': name, 'sha256': digest})
    draw.text((20, 760), '正常LOAD後實際點選；下排放大影像選單。不同時間畫面，不作原版逐像素對拍。', font=small, fill='#b6c7dc')
    output = args.out/'quality-overview.png'
    canvas.save(output)
    result = {'status': 'PASS-SOURCE-CHECKED-GUI-OVERVIEW', 'sources': sources,
              'gui_receipt_sha256': hashlib.sha256(receipt_path.read_bytes()).hexdigest(),
              'binary_sha256': receipt['binary_sha256'],
              'font_sha256': hashlib.sha256(args.font.read_bytes()).hexdigest(),
              'builder_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              'output_sha256': hashlib.sha256(output.read_bytes()).hexdigest()}
    (args.out/'overview.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(result, ensure_ascii=False))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--modern', type=Path, required=True)
    parser.add_argument('--font', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--gui', type=Path, help='已通過的正常GUI收據目錄，改產生實際三品質工具列總覽')
    args = parser.parse_args()
    if args.gui:
        build_gui_overview(args)
        return
    assert not args.out.exists()
    catalog = json.loads(args.catalog.read_text())
    choices = [
        ('地圖人物，144像素', 'map_sprite_hd_set', 'source_group', 0),
        ('地形，144像素', 'native_terrain_hd_set', 'map_id', 0),
        ('戰鬥人物，原尺寸局部', 'battle_animation_hd_set', 'source_resource', 0),
    ]
    samples = []
    for label, role, key, value in choices:
        asset = next(a for a in catalog['assets'] if a['role'] == role and a.get(key) == value)
        path = args.modern / asset['files'][0]
        raw = path.read_bytes()
        digest = hashlib.sha256(raw).hexdigest()
        assert digest == asset['frame_sha256'][0], path
        with Image.open(path) as source:
            picture = source.convert('RGBA')
        if role == 'battle_animation_hd_set':
            box = picture.getbbox()
            assert box
            x = max(0, min(picture.width - 300, (box[0] + box[2]) // 2 - 150))
            crop = (x, box[1], min(x + 300, picture.width), min(box[1] + 300, picture.height))
        else:
            crop = (0, 0, min(300, picture.width), min(300, picture.height))
        samples.append((label, path, digest, picture, crop))
    args.out.mkdir()
    font = ImageFont.truetype(str(args.font), 24)
    small = ImageFont.truetype(str(args.font), 18)
    canvas = Image.new('RGB', (1020, 1000), '#101e32')
    draw = ImageDraw.Draw(canvas)
    variants = [('完整HD1', 1), ('均衡候選：寬高2/3', 2/3), ('輕量候選：寬高1/2', 1/2)]
    for col, (title, scale) in enumerate(variants):
        x = 20 + col * 340
        draw.text((x, 16), title, font=font, fill='#f2d995')
        draw.text((x, 52), f'貼圖畫素量 {scale*scale:.0%}', font=small, fill='#b6c7dc')
        y = 100
        for label, path, digest, picture, crop in samples:
            draw.text((x, y), label, font=small, fill='white')
            y += 32
            if scale == 1:
                result = picture
            else:
                # Premultiplied-alpha resampling avoids dark transparent edges.
                reduced = picture.convert('RGBa').resize(
                    (round(picture.width*scale), round(picture.height*scale)), Image.Resampling.LANCZOS)
                # This preview proposes smooth final presentation; it does not
                # claim that a lower-resolution production renderer exists.
                result = reduced.resize(picture.size, Image.Resampling.BILINEAR).convert('RGBA')
            result = result.crop(crop)
            canvas.paste(result, (x, y), result)
            y += (180 if '144' in label else 330)
    draw.text((20, 960), '縮圖呈現原型。字型、介面與母稿保持；尚未接入遊戲，也不代表60fps驗收。', font=small, fill='#b6c7dc')
    output = args.out/'texture-comparison.png'
    canvas.save(output)
    receipt = {
        'status': 'DRAFT-VISUAL-ONLY', 'catalog_sha256': hashlib.sha256(args.catalog.read_bytes()).hexdigest(),
        'variants': [{'name': name, 'width_height_scale': s, 'pixel_fraction': s*s} for name, s in variants],
        'sources': [{'label': label, 'file': path.name, 'sha256': digest, 'source_size': list(picture.size), 'crop': list(crop)}
                    for label, path, digest, picture, crop in samples],
        'output': output.name, 'output_sha256': hashlib.sha256(output.read_bytes()).hexdigest(),
        'font_sha256': hashlib.sha256(args.font.read_bytes()).hexdigest(),
        'limit': '只比較同來源縮圖觀感；畫素量比例不等於整個遊戲記憶體或幀率比例。',
    }
    (args.out/'preview.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(receipt, ensure_ascii=False))


if __name__ == '__main__':
    main()
