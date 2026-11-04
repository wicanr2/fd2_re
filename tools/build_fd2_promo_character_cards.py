#!/usr/bin/env python3
"""依正式姓名、原版角色資料與已登錄美術，編製推廣片雙人人物介紹卡。"""

import argparse
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def lines_for(draw, text, font, width):
    rows, line = [], ''
    for char in text:
        if line and draw.textlength(line+char, font=font) > width and char not in '，。！？；：、）》」』】':
            rows.append(line)
            line = char
        else:
            line += char
    if line:
        rows.append(line)
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--storyboard', type=Path, required=True)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--modern', type=Path, required=True)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--font', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and args.font.is_file()
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    args.out.mkdir()
    root = Path(__file__).resolve().parents[1]
    storyboard = json.loads(args.storyboard.read_text())
    characters = storyboard['characters']
    assert [row['identity'] for row in characters] == list(range(32))
    assets = json.loads(args.catalog.read_text())['assets']
    portraits = {a['speaker_id']: a for a in assets if a['role'] == 'story_portrait_hd_frame'}
    figures = {a['source_resource']: a for a in assets if a['role'] == 'battle_animation_hd_set'}
    entities = json.loads((root/'remake/assets/locales/zh-Hant/entities.json').read_text())
    constructors = {row['id']: row for row in json.loads((root/'remake/assets/data/native_join_constructor.json').read_text())['rows']}
    fonts = {size: ImageFont.truetype(str(args.font), size, index=3) for size in (24, 28, 30, 36, 46, 56)}
    records = []
    for page in range(16):
        canvas = Image.new('RGB', (1920, 1080))
        draw = ImageDraw.Draw(canvas)
        for y in range(1080):
            amount = y/1079
            draw.line((0, y, 1920, y), fill=(round(9+5*amount), round(17+9*amount), round(36+15*amount)))
        draw.text((60, 46), '再一次，並肩作戰。', font=fonts[46], fill='#f6dfaa')
        draw.text((1860, 61), f'伙伴群像  {page+1:02} / 16', font=fonts[28], fill='#b5c3d8', anchor='ra')
        page_rows = []
        for side in range(2):
            char = characters[page*2+side]
            identity = char['identity']
            assert entities['characters'][str(identity)]['name'] == char['name']
            # The source constructor gives initial DATO/BattleFig=identity.
            # Cards use that initial appearance, never a guessed promoted ID.
            job = bytes.fromhex(constructors[identity]['default_raw'])[1]
            class_name = entities['class_names'][str(job)]['name']
            left, top, width, height = 60+side*930, 150, 870, 790
            box = (left, top, left+width, top+height)
            draw.rectangle(box, fill='#142744', outline='#3c2b16', width=8)
            draw.rectangle((left+7, top+7, left+width-7, top+height-7), outline='#d6b56f', width=3)
            draw.line((left+11, top+11, left+width-12, top+11), fill='#f7e3aa', width=2)
            draw.line((left+11, top+height-12, left+width-12, top+height-12), fill='#675135', width=3)
            draw.text((left+38, top+29), char['name'], font=fonts[56], fill='#fff0c6')
            draw.text((left+41, top+105), class_name, font=fonts[28], fill='#a8bfd8')
            portrait_style = 'HD'
            if identity in portraits:
                asset = portraits[identity]
                path = args.modern/asset['file']
                assert sha(path) == asset['sha256']
            else:
                portrait_style = '原版頭像'
                path = args.original/f'portraits/DATO_{identity:03}_m0.png'
                assert path.is_file()
            portrait = Image.open(path).convert('RGBA')
            pbox = (left+40, top+184, left+260, top+404)
            draw.rectangle((pbox[0]-5, pbox[1]-5, pbox[2]+5, pbox[3]+5), fill='#071327', outline='#d6b56f', width=2)
            canvas.paste(portrait.resize((220, 220), Image.Resampling.LANCZOS if portrait_style == 'HD' else Image.Resampling.NEAREST), pbox[:2],
                         portrait.resize((220, 220), Image.Resampling.LANCZOS if portrait_style == 'HD' else Image.Resampling.NEAREST).getchannel('A'))
            if portrait_style != 'HD':
                draw.text((left+40, top+414), portrait_style, font=fonts[24], fill='#b5c3d8')
            resource = identity*3
            asset = figures[resource]
            body_path = args.modern/asset['files'][0]
            assert sha(body_path) == asset['frame_sha256'][0]
            body = Image.open(body_path).convert('RGBA')
            bounds = body.getchannel('A').getbbox()
            assert bounds
            body = body.crop(bounds)
            body.thumbnail((490, 600), Image.Resampling.LANCZOS)
            canvas.paste(body, (left+337+(490-body.width)//2, top+140+(600-body.height)), body)
            draw.text((left+40, top+478), char['headline'], font=fonts[30], fill='#f0cb79')
            text_rows = lines_for(draw, char['copy'], fonts[36], 272)
            assert len(text_rows) <= 5
            for row, text in enumerate(text_rows):
                draw.text((left+40, top+537+row*45), text, font=fonts[36], fill='#eef1f5')
            page_rows.append(dict(identity=identity, name=char['name'], initial_class=class_name,
                                  portrait_style=portrait_style, portrait_file=path.name, portrait_sha256=sha(path),
                                  figure_resource=resource, figure_file=body_path.name, figure_sha256=sha(body_path),
                                  source_string_id=char['source_string_id'], copy=char['copy']))
        draw.text((60, 986), '那些曾陪你遠行的人，重新站上戰場。', font=fonts[30], fill='#becbdd')
        draw.text((1860, 991), '人物介紹・依原版資料重編', font=fonts[24], fill='#8b9bb3', anchor='ra')
        filename = f'companions-{page+1:02}.png'
        canvas.save(args.out/filename)
        records.append(dict(file=filename, sha256=sha(args.out/filename), seconds=8, characters=page_rows))
    receipt = dict(schema_version=1, status='CHARACTER-CARDS-COMPILED-REVIEW-PENDING', size=[1920, 1080],
                   character_count=32, page_count=16, seconds=128, gameplay_capture=False,
                   source='原版FDTXT_031、正式姓名與JOIN初始造型，依README結局人物卡編輯成推廣片介紹。',
                   font_sha256=sha(args.font), storyboard_sha256=sha(args.storyboard),
                   catalog_sha256=sha(args.catalog), compiler_sha256=sha(Path(__file__)), pages=records)
    (args.out/'character-cards.json').write_text(json.dumps(receipt, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(pages=16, characters=32, original_portrait_fallback=[r['identity'] for p in records for r in p['characters'] if r['portrait_style'] != 'HD']), ensure_ascii=False))


if __name__ == '__main__':
    main()
