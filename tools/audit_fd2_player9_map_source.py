#!/usr/bin/env python3
"""核對玩家指令9的FDOTHER6特效與FDOTHER5數字固定圖源。"""
import argparse
import hashlib
import json
import os
import struct
from pathlib import Path

from PIL import Image, ImageDraw
from validate_modern_theme_catalog import ROOT

RANGES = {6: list(range(87, 114)), 5: list(range(94, 104))+[116, 117, 118]}
FOLDERS = {5: 'ui/fdother_005_lmi1_opaque', 6: 'effects/fdother_006_lmi1_opaque'}


def digest(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def audit(pack, ranges=None):
    inventory = json.loads((ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json').read_text())
    if digest(pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise ValueError('固定原包manifest不同')
    manifest = json.loads((pack/'manifest.json').read_text())
    rows = {r['path']: r for r in manifest['assets']}
    pinned = next(r for r in json.loads((ROOT/'docs/data/fd2-reference-files.json').read_text())['files'] if r['file'] == 'FDOTHER.DAT')
    if next(r for r in manifest['source_set'] if r['file'] == 'FDOTHER.DAT') != pinned:
        raise ValueError('FDOTHER版本不同')
    palette_path = pack/'palette/fdother_000.json'; palette_row = rows.get(palette_path.relative_to(pack).as_posix())
    if palette_row is None or palette_row.get('source_resource') != 0 or digest(palette_path) != palette_row['sha256']:
        raise ValueError('正式地圖色盤SHA不同')
    dac = json.loads(palette_path.read_text())['dac_6bit_components']
    if len(dac) != 768 or any(type(v) is not int or not 0 <= v <= 63 for v in dac):
        raise ValueError('正式地圖DAC形狀不同')
    palette = [(v << 2) | (v >> 4) for v in dac]
    resources, pictures = [], []
    for resource, indices in (RANGES if ranges is None else ranges).items():
        folder = pack/FOLDERS[resource]; metadata = folder/'bank.json'
        def checked(p):
            row = rows.get(p.relative_to(pack).as_posix())
            if row is None or row.get('source_file') != 'FDOTHER.DAT' or row.get('source_resource') != resource or digest(p) != row['sha256']:
                raise ValueError('LMI固定圖源SHA或資源不同')
        # 舊固定包的#5 bank.json未列metadata，逐PNG已列manifest。
        # 因此再由已列manifest的raw，以既有ParseLMI1契約逐cell獨立重解。
        manifested = metadata.relative_to(pack).as_posix() in rows
        if manifested:
            checked(metadata)
        elif resource != 5:
            raise ValueError('未登錄的LMI metadata')
        bank = json.loads(metadata.read_text())
        if bank.get('kind') != 'fdother_lmi1_bank' or bank['source']['resource'] != resource or bank['source']['sha256'] != pinned['sha256']:
            raise ValueError('LMI bank來源不同')
        frames = []
        raw_path = pack/f'raw/FDOTHER/FDOTHER_{resource:03d}.bin'; checked(raw_path); resource_bytes = raw_path.read_bytes()
        if resource_bytes[:4] != b'LMI1' or digest(raw_path) != bank['source']['raw_sha256']:
            raise ValueError('固定LMI raw來源不同')
        count = struct.unpack_from('<H', resource_bytes, 4)[0]
        for index in indices:
            entry = bank['entries'][index]
            relative = Path(entry['frame'])
            if entry['index'] != index or relative.is_absolute() or '..' in relative.parts:
                raise ValueError('LMI索引或路徑不同')
            p = folder/relative; checked(p)
            with Image.open(p) as im:
                if im.mode != 'P' or im.size != (entry['width'], entry['height']):
                    raise ValueError('LMI Pmode或尺寸不同')
                raw = bytes(im.getdata()); alpha = Image.frombytes('L', im.size, bytes(255 if n else 0 for n in raw))
                colour = im.copy(); colour.putpalette(palette)
                picture = colour.convert('RGBA'); picture.putalpha(alpha)
            if not 0 <= index < count:
                raise ValueError('raw LMI索引越界')
            at = struct.unpack_from('<I', resource_bytes, 6+index*4)[0]
            width, height = struct.unpack_from('<HH', resource_bytes, at); at += 4; decoded = bytearray()
            while len(decoded) < width*height:
                value = resource_bytes[at]; at += 1
                if value <= 0xc0:
                    decoded.append(value)
                else:
                    decoded.extend(bytes([resource_bytes[at]])*min(value-0xc0, width*height-len(decoded))); at += 1
            if (width, height) != (entry['width'], entry['height']) or bytes(decoded) != raw:
                raise ValueError('Pmode圖源與已知raw解碼不一致')
            frame = {'index': index, 'width': entry['width'], 'height': entry['height'], 'pixels_sha256': hashlib.sha256(raw).hexdigest(),
                     'png_sha256': digest(p), 'visible_pixels': sum(n != 0 for n in raw), 'ink_box': alpha.getbbox(), 'native_empty': alpha.getbbox() is None}
            frames.append(frame); pictures.append((resource, frame, picture))
        resources.append({'resource': resource, 'metadata_sha256': digest(metadata), 'metadata_manifested': manifested,
                          'raw_sha256': digest(raw_path), 'frames': frames})
    return {'status': 'PASS-PINNED-PLAYER9-MAP-SOURCES' if ranges is None else 'PASS-PINNED-LMI-SOURCES', 'source_file': pinned, 'pack_manifest_sha256': digest(pack/'manifest.json'),
            'palette_sha256': digest(palette_path), 'palette_resource': 0,
            'resources': resources, 'tool': {'file': 'tools/audit_fd2_player9_map_source.py', 'sha256': digest(Path(__file__))},
            'scope': '固定解碼LMI1圖源，index0透明依既有BlitAt(false)契約。非新增RE或原版oracle。'}, pictures


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('--compound-tails', action='store_true', help='核對32～35既有typed尾段的全部LMI索引，包含原始空拍')
    args = p.parse_args()
    report, sheet = [Path(str(args.out)+suffix) for suffix in ['.json', '.png']]
    if not Path('/.dockerenv').exists() or report.exists() or sheet.exists() or report.parent.stat().st_uid != os.getuid():
        raise SystemExit('Docker內使用本人擁有的新輸出前綴')
    ranges = None
    if args.compound_tails:
        ranges = {5: [74,75,76]+list(range(94,104))+list(range(105,115))+[116,117,118],
                  6: list(range(57,76))+list(range(126,191))}
    data, pictures = audit(args.original_pack, ranges)
    canvas = Image.new('RGB', (1200, ((len(pictures)+9)//10)*200), '#151721'); draw = ImageDraw.Draw(canvas)
    for n, (resource, row, im) in enumerate(pictures):
        x, y = n % 10*120, n//10*200
        scale = min(4, 116/im.width, 150/im.height) if args.compound_tails else 4
        picture = im.resize((max(1,int(im.width*scale)), max(1,int(im.height*scale))), Image.Resampling.NEAREST)
        cell = Image.new('RGBA', (120, 200), '#151721'); cell.alpha_composite(picture, ((120-picture.width)//2, 45)); canvas.paste(cell.convert('RGB'), (x, y))
        draw.text((x+4, y+4), f'{resource}:{row["index"]} {row["width"]}x{row["height"]}', fill='white')
    canvas.save(sheet); data['reference_png_sha256'] = digest(sheet)
    report.write_text(json.dumps(data, ensure_ascii=False, indent=2)+'\n')
    counts = {r['resource']:len(r['frames']) for r in data['resources']}
    print({'status': data['status'], 'effect_frames': counts.get(6,0), 'numeric_sources': counts.get(5,0)})


if __name__ == '__main__': main()
