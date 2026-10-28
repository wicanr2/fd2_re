#!/usr/bin/env python3
"""以同版完整Game.Draw收據製作README代表地圖總覽，不重繪來源。"""
import argparse
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


MAPS = [0, 1, 9, 24, 25, 26]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def validate(receipt, folder, catalog, coverage):
    if (receipt.get('status') != 'PASS-HD-MAP-OVERVIEW-GPU' or
            receipt.get('catalog_sha256') != sha(catalog) or receipt.get('coverage_sha256') != sha(coverage)):
        raise ValueError('擷取清冊或覆蓋來源不同')
    cov = read(coverage)
    if cov['catalog_sha256'] != receipt['catalog_sha256']:
        raise ValueError('覆蓋資料不是目前清冊')
    if ([m['map'] for m in receipt['maps']] != MAPS or
            [c['map'] for c in receipt['cases']] != MAPS or len(receipt['cases']) != 6):
        raise ValueError('六張代表地圖缺項或順序不符')
    output = []
    for state, case in zip(receipt['maps'], receipt['cases']):
        if (state['chapter'] != state['map'] + 1 or state['painted_terrain_draws'] <= 0 or state['save_bytes'] <= 0 or
                not case.get('game_draw') or not case.get('f2') or case.get('rgba_tolerance') != 2):
            raise ValueError('缺完整Game.Draw、HD來源、F2或保存檢查')
        name = case['file']
        if Path(name).name != name or sha(folder / name) != case['sha256']:
            raise ValueError('完整PNG路徑或SHA不符')
        with Image.open(folder / name) as image:
            if image.size != (1920, 1200):
                raise ValueError('來源須為完整1920×1200遊戲畫面')
        coverage_map = cov['maps'][state['map']]
        if coverage_map['map'] != state['map']:
            raise ValueError('地圖覆蓋順序不符')
        output.append({**state, 'file': name, 'sha256': case['sha256'],
                       'hd_slots': sum(f['status'] == 'registered-hd' for f in coverage_map['required']),
                       'required_slots': len(coverage_map['required'])})
    return output


def bounded_text(canvas, text, rect, font, fill):
    x, y, width, height = rect
    draw = ImageDraw.Draw(canvas)
    original = text
    while text and draw.textlength(text, font=font) > width:
        if not original:
            raise ValueError('文字安全區不足以顯示省略符號')
        original = original[:-1]
        text = original + '…'
    box = draw.textbbox((0, 0), text, font=font)
    if box[2] - box[0] > width or box[3] - box[1] > height:
        raise ValueError('總覽標籤超出文字安全區')
    mask = Image.new('L', canvas.size)
    ImageDraw.Draw(mask).text((x - box[0], y - box[1]), text, font=font, fill=255)
    ink = mask.getbbox()
    if ink and not (x <= ink[0] <= ink[2] <= x + width and y <= ink[1] <= ink[3] <= y + height):
        raise ValueError('實際字形墨跡越界')
    canvas.paste(fill, mask=mask)
    return {'text': text, 'rect': rect, 'ink_bbox': ink}


def build(args):
    if not Path('/.dockerenv').exists() or args.out.exists() or args.report.exists():
        raise ValueError('只在Docker使用全新輸出，不覆寫現有證據')
    if any(not p.parent.is_dir() or p.parent.stat().st_uid != os.getuid() for p in (args.out, args.report)):
        raise ValueError('輸出父目錄必須已存在且由目前使用者擁有')
    receipt = read(args.receipt)
    rows = validate(receipt, args.receipt.parent, args.catalog, args.coverage)
    rejected = []
    if args.self_test:
        for fault in ('catalog', 'missing-map', 'png-sha', 'tolerance', 'missing-f2'):
            bad = deepcopy(receipt)
            if fault == 'catalog':
                bad['catalog_sha256'] = 'wrong'
            elif fault == 'missing-map':
                bad['maps'].pop()
            elif fault == 'png-sha':
                bad['cases'][0]['sha256'] = 'wrong'
            elif fault == 'tolerance':
                bad['cases'][0]['rgba_tolerance'] = 3
            else:
                bad['cases'][0]['f2'] = False
            try:
                validate(bad, args.receipt.parent, args.catalog, args.coverage)
            except ValueError:
                rejected.append(fault)
            else:
                raise ValueError('拒收反例未失敗：' + fault)
    fonts = {size: ImageFont.truetype(str(args.font), size, index=3) for size in (18, 24, 34)}
    if 'TC' not in fonts[24].getname()[0]:
        raise ValueError('總覽須沿既有Noto Sans CJK TC字型')
    canvas = Image.new('RGB', (1352, 1662), '#131b24')
    draw = ImageDraw.Draw(canvas)
    labels = [bounded_text(canvas, 'HD 地圖總覽', (24, 22, 1304, 48), fonts[34], '#f3d799'),
              bounded_text(canvas, '目前已接素材的六張代表地圖 · 同版本受控擷取', (24, 80, 1304, 26), fonts[18], '#cad4df')]
    for i, row in enumerate(rows):
        x, y = 24 + (i % 2) * 664, 126 + (i // 2) * 488
        draw.rectangle((x - 1, y - 1, x + 640, y + 480), outline='#455364', width=1)
        labels.append(bounded_text(canvas, f'第{row["chapter"]}章 {row["name"]}｜地圖{row["map"]:02d}',
                                   (x + 12, y + 8, 616, 32), fonts[24], '#f0dfb6'))
        with Image.open(args.receipt.parent / row['file']) as source:
            canvas.paste(source.convert('RGB').resize((640, 400), Image.Resampling.LANCZOS), (x, y + 48))
        labels.append(bounded_text(canvas, f'已登錄 HD 圖源 {row["hd_slots"]}／{row["required_slots"]} 槽',
                                   (x + 12, y + 452, 616, 28), fonts[18], '#c5d4df'))
    labels.append(bounded_text(canvas, '仍含原版素材；槽數含事件有效時的來源上界，不代表地圖完成比例。',
                               (24, 1612, 1304, 28), fonts[18], '#c5d4df'))
    canvas.save(args.out, optimize=True)
    result = {'schema_version': 1, 'status': 'PASS-HD-MAP-OVERVIEW', 'map_ids': MAPS,
              'size': list(canvas.size), 'rows': rows, 'labels': labels, 'rejected_mutants': rejected,
              'input_hashes': {'receipt': sha(args.receipt), 'catalog': sha(args.catalog), 'coverage': sha(args.coverage),
                               'font': sha(args.font), 'tool': sha(Path(__file__))},
              'output': {'file': args.out.name, 'sha256': sha(args.out)},
              'scope': '六張同版完整遊戲PNG等比例縮圖，保留像素區域；受控相機與程式F2，非普通玩家或原版parity。'}
    args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'status': result['status'], 'maps': MAPS, 'size': result['size'], 'rejected': rejected}, ensure_ascii=False))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'catalog', 'coverage', 'font', 'out', 'report'):
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--self-test', action='store_true')
    build(parser.parse_args())
