#!/usr/bin/env python3
"""核對敵方指令9的固定FDOTHER44圖源，保留第30原空拍。"""
import argparse
import json
import os
import platform
from pathlib import Path

from PIL import Image, ImageDraw, __version__ as pillow_version
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ROOT

FIELDS = ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']


def audit(pack):
    inventory = ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json'
    expected = json.loads(inventory.read_text())
    if digest(pack/'manifest.json') != expected['pack_manifest_sha256']:
        raise ValueError('原版包指紋不同')
    manifest = json.loads((pack/'manifest.json').read_text())
    pin = next(v for v in json.loads((ROOT/'docs/data/fd2-reference-files.json').read_text())['files'] if v['file'] == 'FDOTHER.DAT')
    if next(v for v in manifest['source_set'] if v['file'] == 'FDOTHER.DAT') != pin:
        raise ValueError('FDOTHER來源版本不同')
    files = {v['path']: v for v in manifest['assets']}
    folder = pack/'animations/FDOTHER_044'

    def checked(path, index=None):
        row = files.get(path.relative_to(pack).as_posix())
        if (row is None or row.get('source_file') != 'FDOTHER.DAT' or row.get('source_resource') != 44
                or digest(path) != row['sha256'] or index is not None and row.get('source_frame') != index):
            raise ValueError('FDOTHER44圖源SHA或定位不同')
        return row

    metadata = folder/'animation.json'
    checked(metadata)
    doc = json.loads(metadata.read_text())
    if (doc.get('source') != {'file': 'FDOTHER.DAT', 'resource': 44} or len(doc['frames']) != 31
            or doc['native_header'] != {'byte_1': 0, 'byte_2': 31, 'byte_4': 0}):
        raise ValueError('FDOTHER44原始header或31幀不同')
    rows, pictures = [], []
    for index, frame in enumerate(doc['frames']):
        if any(Path(frame[k]).name != frame[k] for k in ['path', 'mask_path']):
            raise ValueError('原圖路徑越界')
        if (frame['x'] < 0 or frame['y'] < 0 or frame['width'] <= 0 or frame['height'] <= 0
                or frame['x']+frame['width'] > 320 or frame['y']+frame['height'] > 200):
            raise ValueError('原圖矩形超出320×200')
        path, mask_path = folder/frame['path'], folder/frame['mask_path']
        a, b = checked(path, index), checked(mask_path, index)
        with Image.open(path) as im, Image.open(mask_path) as mask:
            if im.mode != 'P' or im.size != mask.size or im.size != (frame['width'], frame['height']):
                raise ValueError('原PNG／Mask形狀不同')
            alpha = mask.convert('L')
            visible = sum(n for value, n in enumerate(alpha.histogram()) if value)
            picture = im.convert('RGBA')
            picture.putalpha(alpha)
        rows.append({**{k: frame[k] for k in FIELDS}, 'frame': index,
                     'pixels_sha256': a['sha256'], 'mask_sha256': b['sha256'],
                     'visible_pixels': visible, 'native_empty': visible == 0})
        pictures.append(picture)
    if [v['frame'] for v in rows if v['native_empty']] != [30]:
        raise ValueError('固定來源只有第30幀為空拍')
    return {'status': 'PASS-PINNED-COMMAND9-AI-NATIVE', 'resource': 44,
            'source_file': pin, 'pack_manifest_sha256': digest(pack/'manifest.json'),
            'inventory_sha256': digest(inventory), 'metadata_sha256': digest(metadata),
            'native_header': doc['native_header'], 'frames': rows,
            'tool': {'file': 'tools/audit_fd2_command9_effect_source.py', 'sha256': digest(Path(__file__)),
                     'python': platform.python_version(), 'pillow': pillow_version,
                     'address_space': '分離圖源內相對矩形，非EXE位址'},
            'scope': '固定原版31來源、描述子及PNG／Mask；第30空拍不可補墨跡。非原版執行期或新增語意。'}, pictures


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True, help='新輸出前綴，附加.json／.png')
    args = p.parse_args()
    report, sheet = [Path(str(args.out)+ext) for ext in ['.json', '.png']]
    if (not Path('/.dockerenv').exists() or report.exists() or sheet.exists()
            or not report.parent.is_dir() or report.parent.stat().st_uid != os.getuid()):
        raise SystemExit('在Docker使用本人擁有的新輸出前綴')
    data, pictures = audit(args.original_pack)
    canvas = Image.new('RGB', (1920, 1320), (18, 18, 26))
    draw = ImageDraw.Draw(canvas)
    for row, picture in zip(data['frames'], pictures):
        index = row['frame']
        cell = Image.new('RGBA', (320, 200), (18, 18, 26, 255))
        cell.alpha_composite(picture, (row['x'], row['y']))
        x, y = (index % 6)*320, (index // 6)*220
        canvas.paste(cell.convert('RGB'), (x, y+20))
        draw.text((x+4, y+4), f'FDOTHER44 / {index:02d}', fill='white')
    canvas.save(sheet)
    data['reference_png_sha256'] = digest(sheet)
    report.write_text(json.dumps(data, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': data['status'], 'frames': 31, 'blank_frames': [30]}))


if __name__ == '__main__':
    main()
