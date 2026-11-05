#!/usr/bin/env python3
"""編製逐人、轉職與其餘圖組的原版／HD全幀總覽，保留來源與未知範圍。"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import shutil

from PIL import Image, ImageDraw, ImageFont


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['original', 'modern', 'catalog', 'inventory', 'font', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--reuse-overview', type=Path, help='逐人來源與文字相同且圖SHA正確時沿用既有總覽')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    assert not args.out.exists() and args.out.parent.stat().st_uid == os.getuid()
    root = Path(__file__).resolve().parents[1]
    inventory = json.loads(args.inventory.read_text())
    assert inventory['status'] == 'PASS-SOURCE-INVENTORY'
    assert inventory['archive_reconstruction']['matches_pinned_original']
    natives = {r['resource']: r for r in inventory['resources']}
    catalog = json.loads(args.catalog.read_text())
    modern = {r['source_resource']: r for r in catalog['assets']
              if r['role'] == 'battle_animation_hd_set'}
    names = json.loads((root / 'remake/assets/locales/zh-Hant/entities.json').read_text())
    targets = json.loads((root / 'remake/assets/data/class_change_targets.json').read_text())
    jobs = {r['portrait']: r['class_id'] for r in targets['target_portraits']}
    identities = {i: names['characters'][str(i)]['name'] for i in range(32)}
    owners = {i: i for i in range(32)}
    for row in targets['current_portraits']:
        for key in ['default_target', 'optional_target', 'special_target']:
            if key in row:
                figure = row[key]
                job = names['class_names'][str(jobs[figure])]['name']
                identities[figure] = identities[row['portrait']] + '／' + job
                owners[figure] = row['portrait']
    face = lambda size: ImageFont.truetype(str(args.font), size, index=3)
    args.out.mkdir()
    manifest = dict(schema_version=1, status='SOURCE-CONTACT-SHEETS',
                    scope='來源圖組全幀對照，未合成戰場；不宣稱每幀都有正式caller、正常取得或原版執行對拍。',
                    catalog_sha256=sha(args.catalog), inventory_sha256=sha(args.inventory),
                    original_manifest_sha256=sha(args.original / 'manifest.json'),
                    producer_sha256=sha(__file__), font_sha256=sha(args.font), figures=[])
    previous = {}
    if args.reuse_overview:
        prior_file = args.reuse_overview / 'overview-manifest.json'
        prior = json.loads(prior_file.read_text())
        assert prior['status'] == manifest['status']
        assert all(prior[key] == manifest[key] for key in
                   ['inventory_sha256', 'original_manifest_sha256', 'font_sha256'])
        previous = {row['figure']: row for row in prior['figures']}
        manifest['reuse_manifest_sha256'] = sha(prior_file)
        manifest['reused_figures'] = []
    fields = ['x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7']
    for figure in range(136):
        banks = []
        for resource in range(figure * 3, figure * 3 + 3):
            proof = natives[resource]
            bank = dict(resource=resource, status=proof['status'], frames=[])
            if proof['status'] == 'decoded':
                directory = args.original / 'animations' / f'FIGANI_{resource:03}'
                metadata = directory / 'animation.json'
                assert sha(metadata) == proof['metadata_sha256']
                doc = json.loads(metadata.read_text())
                assert len(doc['frames']) == len(proof['frames'])
                hd = modern.get(resource)
                if hd:
                    assert hd['source_metadata_sha256'] == proof['metadata_sha256']
                    assert hd['frame_count'] == len(doc['frames']) == len(hd['files'])
                for index, (frame, evidence) in enumerate(zip(doc['frames'], proof['frames'])):
                    assert all(frame[k] == evidence[k] for k in fields)
                    assert sha(directory / frame['path']) == evidence['pixels_sha256']
                    assert sha(directory / frame['mask_path']) == evidence['mask_sha256']
                    bank['frames'].append(dict(index=index, descriptor={k: frame[k] for k in fields},
                        original=str(directory / frame['path']), mask=str(directory / frame['mask_path']),
                        original_sha256=evidence['pixels_sha256'], mask_sha256=evidence['mask_sha256'],
                        hd=str(args.modern / hd['files'][index]) if hd else None,
                        hd_sha256=hd['frame_sha256'][index] if hd else None))
            banks.append(bank)
        title = identities.get(figure, f'圖組 {figure:03}／身份未綁定')
        old = previous.get(figure)
        filename = f'figure-{figure:03}.png'
        if old and old['name'] == title and old['owner'] == owners.get(figure) and old['banks'] == banks:
            for bank in banks:
                for frame in bank['frames']:
                    if frame['hd']:
                        assert sha(frame['hd']) == frame['hd_sha256']
            source = args.reuse_overview / old['sheet']
            assert sha(source) == old['sha256']
            shutil.copyfile(source, args.out / filename)
            manifest['figures'].append(dict(figure=figure, owner=owners.get(figure), name=title,
                sheet=filename, sha256=old['sha256'], banks=banks))
            manifest['reused_figures'].append(figure)
            print(f'{figure:03} {title} 來源與圖SHA相同，沿用', flush=True)
            continue
        height = 190 + sum(64 + max(1, math.ceil(len(b['frames']) / 4)) * 348 for b in banks)
        canvas = Image.new('RGBA', (1060, height), '#0b192a')
        draw = ImageDraw.Draw(canvas)
        draw.text((24, 15), title, font=face(36), fill='#f3dfaa')
        draw.text((24, 68), f'BattleFig {figure:03}｜每格上排原版，下排 HD', font=face(25), fill='#c5d4e1')
        draw.text((24, 108), '依原始座標縮圖；完整圖源審閱，非戰場錄影或正常取得證據。', font=face(22), fill='#aabed0')
        top = 170
        for number, bank in enumerate(banks):
            resource = bank['resource']
            label = ['第一銀行／待機來源', '第二銀行／攻擊來源', '第三銀行／動作與特效來源'][number]
            draw.text((24, top), f'{label}　FIGANI {resource:03}　{len(bank["frames"])} 幀', font=face(23), fill='#e9c67d')
            top += 54
            if not bank['frames']:
                draw.text((34, top + 30), bank['status'] + '：原始資料無可列影格，不補造。', font=face(22), fill='#aabed0')
            for index, row in enumerate(bank['frames']):
                left, y = 24 + (index % 4) * 258, top + (index // 4) * 348
                draw.text((left, y), f'f{index:02}  delay {row["descriptor"]["delay_native"]}', font=face(19), fill='#dbe6f1')
                frame = row['descriptor']
                original = Image.open(row['original']).convert('RGBA')
                original.putalpha(Image.open(row['mask']).convert('L'))
                full = Image.new('RGBA', (320, 200))
                full.alpha_composite(original, (frame['x'], frame['y']))
                canvas.alpha_composite(full.resize((240, 150), Image.Resampling.NEAREST), (left, y + 28))
                if row['hd']:
                    assert sha(row['hd']) == row['hd_sha256']
                    picture = Image.open(row['hd']).convert('RGBA')
                    if picture.size == (frame['width'] * 6, frame['height'] * 6):
                        full = Image.new('RGBA', (1920, 1200))
                        full.alpha_composite(picture, (frame['x'] * 6, frame['y'] * 6))
                    else:
                        assert picture.size == (1920, 1200), (resource, index, picture.size)
                        full = picture
                    canvas.alpha_composite(full.resize((240, 150), Image.Resampling.LANCZOS), (left, y + 184))
                else:
                    draw.text((left + 12, y + 235), 'HD 尚未登錄', font=face(21), fill='#d7a984')
            top += max(1, math.ceil(len(bank['frames']) / 4)) * 348 + 10
        canvas.convert('RGB').save(args.out / filename, optimize=True)
        manifest['figures'].append(dict(figure=figure, owner=owners.get(figure), name=title,
            sheet=filename, sha256=sha(args.out / filename), banks=banks))
        print(f'{figure:03} {title}', flush=True)
    lines = ['# 戰鬥動畫逐人總覽', '', manifest['scope'], '',
             '每格上排為原版解碼圖源，下排為目前正式 HD 圖源。保留原始座標，縮圖不代表原版同狀態執行。',
             '第三銀行依實際內容列出，不一概叫作魔法；空標頭、全透明影格與未登錄 HD 均明示。', '',
             '完整全幀來源對照僅供本機審閱。原始素材及可重組圖源不加入公開庫。', '', '## 伙伴索引', '',
             '| 角色 | 基底圖組 | 轉職／其他已證實形態 |', '|---|---|---|']
    for identity in range(32):
        variants = [f'[{identities[f]}](#figure-{f:03})' for f in sorted(owners) if f != identity and owners[f] == identity]
        lines.append(f'| {identities[identity]} | [圖組 {identity:03}](#figure-{identity:03}) | {"、".join(variants) or "原始轉職表未列"} |')
    for row in manifest['figures']:
        figure = row['figure']
        lines += ['', f'<a id="figure-{figure:03}"></a>', f'## {row["name"]}', '',
                  '| 銀行 | 原始狀態 | 幀數 | HD來源 |', '|---|---|---:|---|']
        for bank in row['banks']:
            hd_count = sum(bool(f['hd']) for f in bank['frames'])
            lines.append(f'| FIGANI {bank["resource"]:03} | {bank["status"]} | {len(bank["frames"])} | {hd_count} 幀 |')
        lines += ['', f'![{row["name"]}原版與HD全幀對照]({row["sheet"]})', '']
    lines += ['', '## 未綁定資源', '',
              'FIGANI 408 不強配任何角色。它保留在原始資源清冊，未據此新增角色或正式招式。', '',
              '來源與逐幀SHA-256見 `overview-manifest.json`。']
    (args.out / 'README.md').write_text('\n'.join(lines) + '\n')
    manifest['markdown_sha256'] = sha(args.out / 'README.md')
    (args.out / 'overview-manifest.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + '\n')


if __name__ == '__main__':
    main()
