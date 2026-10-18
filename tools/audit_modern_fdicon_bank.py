#!/usr/bin/env python3
"""核對實際FDICON header與全部分離幀，盤點現代清冊未涵蓋的selector。"""
import argparse
import hashlib
import json
from pathlib import Path
import struct

from PIL import Image


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--bank', type=Path, required=True)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--reference', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    parser.add_argument('--palette', type=Path, help='FDOTHER#0分離DAC；只供明示原版參照圖')
    parser.add_argument('--render-selector', type=int, action='append', default=[])
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not args.report.exists() and args.report.parent.stat().st_uid == 1000
    raw = args.original.read_bytes()
    reference = next(f for f in json.loads(args.reference.read_text())['files'] if f['file'] == 'FDICON.B24')
    assert len(raw) == reference['size'] and digest(raw) == reference['sha256']
    assert hashlib.md5(raw).hexdigest() == reference['md5']
    width, height, count = struct.unpack_from('<HHH', raw)
    assert (width, height) == (24, 24) and count % 12 == 0
    bank = json.loads(args.bank.read_text())
    assert bank['source']['sha256'] == reference['sha256'] and len(bank['sprites']) == count
    registered = {a['source_group'] for a in json.loads(args.catalog.read_text())['assets']
                  if a.get('role') == 'map_sprite_set'}
    frames, layers = [], {}
    for index, entry in enumerate(bank['sprites']):
        assert entry['index'] == index
        decoded = {}
        for key in ('frame', 'mask', 'remap_mask'):
            name = entry[key]
            assert name == f'sprite_{index:04d}/{key}.png'
            path = args.bank.parent/name
            layers[name] = digest(path.read_bytes())
            with Image.open(path) as image:
                assert image.size == (24, 24) and image.mode in ('L', 'P')
                decoded[key] = image.tobytes()
        writes = sum(v != 0 for v in decoded['mask'])
        remaps = sum(v != 0 for v in decoded['remap_mask'])
        combined = decoded['frame']+decoded['mask']+decoded['remap_mask']
        frames.append({'index': index, 'source_write': writes, 'destination_remap': remaps, 'decoded_sha256': digest(combined)})
    groups = []
    canonical = {}
    for group in range(count//12):
        selected = frames[group*12:(group+1)*12]
        visible = [i for i, f in enumerate(selected) if f['source_write']]
        key = tuple(f['decoded_sha256'] for f in selected)
        entry = {'selector': group, 'visible_frames': visible, 'registered': group in registered,
                 'frames': selected, 'exact_decoded_alias_of': canonical.get(key)}
        canonical.setdefault(key, group)
        groups.append(entry)
    missing = [g['selector'] for g in groups if g['visible_frames'] and not g['registered']]
    blank = [g['selector'] for g in groups if not g['visible_frames']]
    report = {'schema_version': 1, 'status': 'PASS', 'scope': '固定原版header及分離mask盤點；不推定人物身分或全部玩家可達性',
              'original': reference, 'header_file_offset': 0, 'header_bytes_hex': raw[:6].hex(),
              'header': {'width': width, 'height': height, 'frames': count, 'groups': count//12},
              'reference_sha256': digest(args.reference.read_bytes()), 'bank_sha256': digest(args.bank.read_bytes()),
              'catalog_sha256': digest(args.catalog.read_bytes()), 'tool_sha256': digest(Path(__file__).read_bytes()),
              'registered_groups': sorted(registered), 'unregistered_visible_selectors': missing,
              'no_source_write_selectors': blank, 'groups': groups, 'separated_layer_sha256': layers}
    if args.render_selector:
        assert args.palette is not None
        palette_data = json.loads(args.palette.read_text())
        expected = next(f for f in json.loads(args.reference.read_text())['files'] if f['file'] == 'FDOTHER.DAT')
        assert palette_data['source']['sha256'] == expected['sha256'] and palette_data['source']['resource'] == 0
        dac = palette_data['dac_6bit_components']
        assert len(dac) == 768 and all(type(v) is int and 0 <= v <= 63 for v in dac)
        rgb = [(v << 2) | (v >> 4) for v in dac]
        report['palette_sha256'] = digest(args.palette.read_bytes())
        report['palette_formula_source'] = 'remake/internal/fdother/palette.go:ParseVGAPalette'
        report['original_reference_images'] = []
        for selector in args.render_selector:
            assert 0 <= selector < count//12
            sheet = Image.new('RGBA', (3*208, 4*208))
            for phase in range(12):
                entry = bank['sprites'][selector*12+phase]
                with Image.open(args.bank.parent/entry['frame']) as image:
                    pixels = Image.frombytes('P', (24, 24), image.tobytes())
                pixels.putpalette(rgb)
                pixels = pixels.convert('RGBA')
                with Image.open(args.bank.parent/entry['mask']) as mask:
                    pixels.putalpha(mask.convert('L').point(lambda value: 255 if value else 0))
                pixels = pixels.resize((192, 192), Image.Resampling.NEAREST)
                sheet.alpha_composite(pixels, ((phase%3)*208+8, (phase//3)*208+8))
            path = args.report.parent/f'native-fdicon-{selector:03d}-reference.png'
            assert not path.exists()
            sheet.save(path)
            report['original_reference_images'].append({'selector': selector, 'file': path.name, 'sha256': digest(path.read_bytes()),
                                                        'scope': '原版分離幀按正式DAC上色；非重製／非執行期擷圖'})
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({k: report[k] for k in ('status', 'header', 'unregistered_visible_selectors', 'no_source_write_selectors')}, ensure_ascii=False))


if __name__ == '__main__':
    main()
