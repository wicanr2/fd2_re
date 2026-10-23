#!/usr/bin/env python3
"""在 Docker 核對固定 BG／TAI 分離圖源，輸出清冊及研究總覽。"""
import argparse
from collections import Counter, defaultdict
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageDraw, __version__ as pillow_version

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def audit(pack):
    pinned = json.loads((ROOT / 'docs/data/fd2-battle-hd-source-inventory-20261019.json').read_text())
    if sha(pack / 'manifest.json') != pinned['pack_manifest_sha256']:
        raise ValueError('固定原包 manifest 不符')
    manifest = json.loads((pack / 'manifest.json').read_text())
    rows = {r['path']: r for r in manifest['assets']}
    by_id = {r['asset_id']: r for r in manifest['assets']}
    sources = {r['file']: r for r in manifest['source_set']}
    references = {r['file']: r for r in json.loads((ROOT / 'docs/data/fd2-reference-files.json').read_text())['files']}

    def checked(relative, source_file, resource=None):
        p = pack / relative
        r = rows.get(relative)
        if not r or r['source_file'] != source_file or (resource is not None and r.get('source_resource') != resource) or sha(p) != r['sha256']:
            raise ValueError(f'圖源 SHA 或資源不符：{relative}')
        return p

    for name in ['BG.DAT', 'TAI.DAT', 'FDOTHER.DAT', 'FDSHAP.DAT']:
        if sources[name] != references[name]:
            raise ValueError(f'原版來源版本不符：{name}')
    palette_path = checked('palette/fdother_000.json', 'FDOTHER.DAT', 0)
    dac = json.loads(palette_path.read_text())['dac_6bit_components']
    if len(dac) != 768 or any(type(n) is not int or not 0 <= n <= 63 for n in dac):
        raise ValueError('戰鬥色盤形狀不符')
    palette = [(n << 2) | (n >> 4) for n in dac]
    resource_rows = {(r['source_file'], r['source_resource']): r for r in manifest['source_resources']}
    report = {'schema_version': 1, 'status': 'PASS-PINNED-BACKGROUND-INVENTORY',
              'pack_manifest_sha256': sha(pack / 'manifest.json'),
              'palette': {'resource': 0, 'sha256': sha(palette_path)},
              'source_files': {n: sources[n] for n in ['BG.DAT', 'TAI.DAT', 'FDOTHER.DAT', 'FDSHAP.DAT']},
              'banks': {}, 'terrain_control_tables': [],
              'scope': '固定分離圖源及地形表盤點。總覽是解碼研究圖，非原版執行截圖、HD 美術或玩家路徑證據。control[2] 出現次數只描述表內容，不證明章內可達。',
              'tool': {'file': 'tools/audit_fd2_battle_background_sources.py', 'sha256': sha(Path(__file__)), 'pillow': pillow_version}}
    pictures = {}
    for bank in ['BG', 'TAI']:
        entries, images, groups = [], [], defaultdict(list)
        indices = sorted(i for name, i in resource_rows if name == bank + '.DAT')
        if indices != list(range(57)):
            raise ValueError('固定圖庫資源數量不同')
        for index in indices:
            folder = f'surfaces/{bank}_{index:03d}'
            p = checked(folder + '/resource.json', bank + '.DAT', index)
            meta = json.loads(p.read_text())
            source = meta['source']
            if source['sha256'] != sources[bank + '.DAT']['sha256'] or source['resource'] != index:
                raise ValueError('surface metadata 來源不符')
            rr = resource_rows[(bank + '.DAT', index)]
            raw_row = by_id[rr['raw_asset_id']]
            raw = checked(raw_row['path'], bank + '.DAT', index)
            if raw.stat().st_size != rr['raw_bytes'] or sha(raw) != rr['raw_sha256'] or source['raw_size'] != rr['raw_bytes']:
                raise ValueError('原始資源大小或 SHA 不符')
            entry = {'index': index, 'metadata_sha256': sha(p), 'raw_sha256': sha(raw), 'raw_size': raw.stat().st_size, 'status': meta['status']}
            if meta['status'] != 'decoded':
                if index != 56 or raw.stat().st_size != 0 or meta.get('reason_code') != 'decode_failed':
                    raise ValueError('未預期的不可解碼圖源')
                entry['classification'] = 'empty-raw-no-art'; entries.append(entry); images.append((entry, None)); continue
            fp = checked(folder + '/' + meta['frame'], bank + '.DAT', index)
            mp = checked(folder + '/' + meta['mask'], bank + '.DAT', index)
            with Image.open(fp) as im, Image.open(mp) as mask:
                if im.mode != 'P' or im.size != (meta['width'], meta['height']) or mask.size != im.size:
                    raise ValueError('圖源 mode 或尺寸不符')
                alpha = mask.convert('L')
                if set(alpha.getdata()) - {0, 255}:
                    raise ValueError('原 Mask 非二值')
                signature = hashlib.sha256(str(im.size).encode() + bytes(im.getdata()) + alpha.tobytes()).hexdigest()
                coloured = im.copy(); coloured.putpalette(palette)
                picture = coloured.convert('RGBA'); picture.putalpha(alpha)
                visible = sum(v != 0 for v in alpha.getdata())
                entry.update(width=im.width, height=im.height, frame_sha256=sha(fp), mask_sha256=sha(mp),
                             indexed_mask_sha256=signature, visible_pixels=visible,
                             ink_box=alpha.getbbox(), classification='empty-mask-no-art' if visible == 0 else 'small-source-review' if im.size == (10, 3) else 'scene-source')
                groups[signature].append(index)
                entries.append(entry); images.append((entry, picture))
        report['banks'][bank] = {'entries': entries, 'decoded': sum(r['status'] == 'decoded' for r in entries),
                                 'dimension_counts': dict(Counter(f"{r['width']}x{r['height']}" for r in entries if 'width' in r)),
                                 'identical_indexed_mask_groups': [g for g in groups.values() if len(g) > 1],
                                 'distinct_decoded_indexed_masks': len(groups),
                                 'visible_entries': sum(r.get('visible_pixels', 0) > 0 for r in entries),
                                 'distinct_visible_indexed_masks': len({r['indexed_mask_sha256'] for r in entries if r.get('visible_pixels', 0) > 0})}
        pictures[bank] = images
    # 正式 SelectNativePhysicalScene 消費四位元組 control 的第三項。
    for row in sorted((r for r in manifest['assets'] if r['path'].startswith('tilesets/fdshap/') and r['path'].endswith('/bank.json')), key=lambda r: r['path']):
        p = checked(row['path'], 'FDSHAP.DAT')
        bank = json.loads(p.read_text()); controls = bank['controls']
        if len(controls) % 4 or any(type(v) is not int or not 0 <= v <= 255 for v in controls):
            raise ValueError('地形 control 形狀不符')
        selectors = Counter(controls[2::4])
        if any(i >= 57 for i in selectors):
            raise ValueError('地形 control[2] 超出固定圖庫')
        report['terrain_control_tables'].append({'map_index': bank['map_index'], 'metadata_sha256': sha(p),
                                                'records': len(controls) // 4, 'control2_counts': dict(sorted(selectors.items()))})
    return report, pictures


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--original-pack', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True, help='已存在目錄中的全新輸出前綴')
    args = parser.parse_args()
    targets = [Path(str(args.out) + s) for s in ['.json', '-BG.png', '-TAI.png']]
    if not Path('/.dockerenv').exists() or any(p.exists() for p in targets) or args.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('請在 Docker 使用本人擁有的全新輸出前綴')
    report, pictures = audit(args.original_pack)
    report['sheets'] = []
    for bank, target in zip(['BG', 'TAI'], targets[1:]):
        canvas = Image.new('RGB', (1344, 2400), '#171d29'); draw = ImageDraw.Draw(canvas)
        for n, (entry, picture) in enumerate(pictures[bank]):
            x, y = n % 4 * 336, n // 4 * 160
            label = f"{bank}_{entry['index']:03d} "
            label += f"{entry['width']}x{entry['height']}" if picture else 'EMPTY RAW / NO ART'
            draw.text((x + 8, y + 5), label, fill='white')
            if picture:
                # 只對 TAI 使用固定兩倍最近鄰，保留來源形狀。
                scale = 2 if bank == 'TAI' else 1
                preview = picture.resize((picture.width * scale, picture.height * scale), Image.Resampling.NEAREST)
                cell = Image.new('RGBA', (320, 120), '#171d29')
                cell.alpha_composite(preview, ((320 - preview.width) // 2, 0))
                canvas.paste(cell.convert('RGB'), (x + 8, y + 28))
        canvas.save(target)
        report['sheets'].append({'file': str(target), 'sha256': sha(target)})
    targets[0].write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'status': report['status'], 'banks': {k: {a: v[a] for a in ['decoded', 'dimension_counts', 'distinct_decoded_indexed_masks']} for k, v in report['banks'].items()}, 'terrain_tables': len(report['terrain_control_tables'])}, ensure_ascii=False))


if __name__ == '__main__':
    main()
