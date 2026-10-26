#!/usr/bin/env python3
"""從一張八圖透明母稿編譯FDOTHER#5十個固定面板圖示。只在Docker執行。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw

INDICES = [53, 54, 55, 56, 57, 59, 60, 61, 62, 63]
META_SHA = 'a547f2f593b980fd9c1b269d1ba95be8727562d7ed31b44015615775d991efd3'
MASTER = 'item-panel-icons-hd6-master-20261026-r1.png'


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def build(master, original, out):
    source = original/'ui/fdother_005_item_panel'
    if sha(source/'resource.json') != META_SHA:
        raise ValueError('原包版本不符')
    dac = json.loads((original/'palette/fdother_000.json').read_text())['dac_6bit_components']
    if len(dac) != 768 or any(type(x) is not int or not 0 <= x <= 63 for x in dac):
        raise ValueError('原色盤不合法')
    colors = [tuple((v << 2) | (v >> 4) for v in dac[i:i+3])+(255,) for i in range(0, 768, 3)]
    sheet = Image.open(master).convert('RGBA')
    if sheet.width < 1000 or sheet.height < 400 or sheet.getchannel('A').getextrema() != (0, 255):
        raise ValueError('母稿必須是高解析透明八圖atlas')
    cells = []
    for i in range(8):
        x, y = i % 4, i // 4
        cell = sheet.crop((x*sheet.width//4, y*sheet.height//2, (x+1)*sheet.width//4, (y+1)*sheet.height//2))
        alpha = cell.getchannel('A')
        bbox = alpha.getbbox()
        if bbox is None or alpha.crop((0, 0, cell.width, 1)).getextrema()[1] > 64 or alpha.crop((0, cell.height-1, cell.width, cell.height)).getextrema()[1] > 64:
            raise ValueError(f'母稿第{i}格空白或跨列')
        if alpha.crop((0, 0, 1, cell.height)).getextrema()[1] > 64 or alpha.crop((cell.width-1, 0, cell.width, cell.height)).getextrema()[1] > 64:
            raise ValueError(f'母稿第{i}格跨欄')
        cells.append(cell.crop(bbox))
    master_target = out/MASTER
    if master_target.exists():
        if sha(master_target) != sha(master):
            raise ValueError('不能覆寫不同母稿')
    else:
        master_target.write_bytes(master.read_bytes())
    entries = []
    for position, index in enumerate(INDICES):
        cell_index = position if position < 8 else position-3
        raw_path = source/f'entry_{index:03d}/frame.png'
        native = Image.open(raw_path)
        size = (29, 24) if index in [53, 54] else (24, 20)
        if native.mode != 'P' or native.size != size:
            raise ValueError(f'原始圖格{index}形狀不符')
        pixels = bytes(native.getdata())
        bg_index = native.getpixel((0, 0) if index in [53, 54] else (2, 2))
        frame_index = native.getpixel((native.width//2, 0))
        picture = Image.new('RGBA', (size[0]*6, size[1]*6), colors[bg_index])
        ImageDraw.Draw(picture).rectangle((0, 0, picture.width-1, picture.height-1), outline=colors[frame_index], width=3)
        artwork = cells[cell_index].copy()
        artwork.thumbnail((picture.width-12, picture.height-12), Image.Resampling.LANCZOS)
        picture.alpha_composite(artwork, ((picture.width-artwork.width)//2, (picture.height-artwork.height)//2))
        filename = f'item-panel-icon-fdother005-{index:03d}-hd6-r1.png'
        target = out/filename
        if target.exists():
            with Image.open(target) as previous:
                if previous.convert('RGBA').tobytes() != picture.tobytes():
                    raise ValueError(f'不能覆寫不同輸出{filename}')
        else:
            picture.save(target)
        entries.append(dict(asset_id=f'modern.ui.item-panel.raw_{index:03d}.hd6', role='native_ui_icon_hd', status='runtime_candidate', file=filename, width=picture.width, height=picture.height, sha256=sha(target), master_file=MASTER, consumer_contract='native_raw_ui_icon_hd6_v1', alpha_contract='opaque', layout_policy='native_raw_cell_rect_hd6', source_archive='FDOTHER.DAT', source_resource=5, source_metadata_sha256=META_SHA, native_raw_cell=dict(index=index, width=size[0], height=size[1], pixels_sha256=hashlib.sha256(pixels).hexdigest()), source_files=[MASTER], source_sha256=[sha(master_target)], source_refs=[f'asset:FDOTHER_005.raw_{index:03d}', 'docs/data/fd2-reference-files.json', 'docs/knowledge-base/61-modern-theme-runtime-spec.md']))
    return entries


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['master', 'original-pack', 'out-dir', 'catalog', 'output-catalog', 'report']:
        p.add_argument('--'+key, type=Path, required=True)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or not a.out_dir.is_dir() or a.output_catalog.exists() or a.report.exists():
        p.error('必須在Docker、沿既有輸出目錄及全新報告執行')
    catalog = json.loads(a.catalog.read_text())
    if any(e['role'] == 'native_ui_icon_hd' for e in catalog['assets']):
        raise ValueError('輸入清冊已包含面板圖示')
    entries = build(a.master, a.original_pack, a.out_dir)
    catalog['assets'].extend(entries)
    a.output_catalog.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    report = dict(status='COMPILED-CANDIDATE', master_sha256=sha(a.master), source_metadata_sha256=META_SHA, indices=INDICES, catalog_assets=len(catalog['assets']), output_catalog=str(a.output_catalog), output_catalog_sha256=sha(a.output_catalog), files=[dict(file=e['file'], sha256=e['sha256'], native_raw_cell=e['native_raw_cell']) for e in entries], art_policy='八圖母稿；裝備劍／護甲共用同圖，紅藍底與框色由原生圖格取得；美術近似', normal_player_verified=False)
    a.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k:v for k,v in report.items() if k!='files'}, ensure_ascii=False))


if __name__ == '__main__':
    main()
