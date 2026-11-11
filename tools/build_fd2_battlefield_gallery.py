#!/usr/bin/env python3
"""從30章完整Game.Draw收據產生每場獨立圖片與Markdown圖冊。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def build(args):
    assert Path('/.dockerenv').exists() and os.getuid() != 0
    for path in (args.out_markdown, args.out_images, args.report):
        assert not path.exists() and path.parent.stat().st_uid == os.getuid(), path
    receipts = [read(path) for path in args.receipt]
    receipt = receipts[0]
    captured_catalog = args.source_catalog or args.catalog
    captured_coverage = args.source_coverage or args.coverage
    assert (args.source_catalog is None) == (args.source_coverage is None)
    source_folders = {}
    for path, part in zip(args.receipt, receipts):
        assert part['status'] == 'PASS-HD-MAP-OVERVIEW-GPU'
        assert part['catalog_sha256'] == sha(captured_catalog)
        assert part['coverage_sha256'] == sha(captured_coverage)
        assert part['source_tree_sha256'] == receipt['source_tree_sha256']
        for row in part['maps']:
            assert row['map'] not in source_folders, '戰場擷取重複'
            source_folders[row['map']] = path.parent
    coverage = read(args.coverage)
    assert coverage['catalog_sha256'] == sha(args.catalog)
    if args.source_catalog:
        before, after = read(captured_catalog), read(args.catalog)
        before.pop('private_root', None); after.pop('private_root', None)
        assert before == after, '正式圖源與GPU擷取候選不同'
        before, after = read(captured_coverage), copy.deepcopy(coverage)
        before.pop('catalog_sha256'); after.pop('catalog_sha256')
        assert before == after, '正式覆蓋與GPU擷取候選不同'
    states = sorted((row for part in receipts for row in part['maps']), key=lambda row: row['map'])
    cases = sorted((row for part in receipts for row in part['cases'] if row.get('game_draw')), key=lambda row: row['map'])
    assert [row['map'] for row in states] == list(range(30))
    assert [row['map'] for row in cases] == list(range(30))
    args.out_images.mkdir()
    lines = ['# HD1 戰場地形圖冊', '',
             '30 個主線戰場各列一張獨立 HD 畫面，依正式章名排列。點選圖片可放大。', '',
             '畫面由重製引擎完整繪製，以受控相機選取代表區域。已核對原版／HD往返、角色資料、亂數與章界保存保持；這些截圖不代表逐章通關或原版畫面對拍。', '',
             '[返回專案首頁](../README.md) · [來源與驗證紀錄](data/fd2-hd1-battlefield-gallery-20261111.json)', '',
             '## 戰場索引', '']
    for state in states:
        chapter = state['chapter']
        lines.append(f'- [第 {chapter} 章：{state["name"]}](#battlefield-{chapter:02d})')
    rows = []
    for state, case in zip(states, cases):
        map_id, chapter = state['map'], state['chapter']
        assert chapter == map_id+1 and state['painted_terrain_draws'] > 0 and state['save_bytes'] > 0
        assert case.get('f2') and case['rgba_tolerance'] == 2
        source = source_folders[map_id]/case['file']
        assert Path(case['file']).name == case['file'] and sha(source) == case['sha256']
        scenario = args.repo/f'remake/assets/scenarios/ch{chapter:02d}.json'
        canonical = read(scenario)
        assert canonical['map'] == map_id and canonical['chapter'] == chapter and canonical['name'] == state['name']
        assert sha(scenario) == state['scenario_sha256']
        name = f'chapter-{chapter:02d}-map-{map_id:02d}.png'
        output = args.out_images/name
        with Image.open(source) as image:
            assert image.size == (1920, 1200)
            image.convert('RGB').resize((1280, 800), Image.Resampling.LANCZOS).save(output, optimize=True)
        relative = os.path.relpath(output, args.out_markdown.parent).replace(os.sep, '/')
        lines.extend(['', f'<a id="battlefield-{chapter:02d}"></a>', '',
                      f'## 第 {chapter} 章：{state["name"]}', '',
                      f'[![第{chapter}章{state["name"]}的HD戰場地形]({relative})]({relative})', '',
                      f'原生地圖 {map_id:02d}，代表區域取景。'])
        rows.append({**state, 'source_file': case['file'], 'source_sha256': case['sha256'],
                     'scaled_unit_unstable_pixels': case.get('scaled_unit_unstable_pixels', 0),
                     'file': str(output.relative_to(args.repo)), 'size': [1280, 800], 'sha256': sha(output)})
    args.out_markdown.write_text('\n'.join(lines)+'\n')
    report = dict(status='PASS-30-BATTLEFIELD-GALLERY', maps=rows,
                  catalog_sha256=sha(args.catalog), coverage_sha256=sha(args.coverage),
                  captured_catalog_sha256=sha(captured_catalog), captured_coverage_sha256=sha(captured_coverage),
                  producer_receipts=[{'file': str(path.relative_to(args.repo)), 'sha256': sha(path)} for path in args.receipt],
                  source_tree_sha256=receipt['source_tree_sha256'], tool_sha256=sha(Path(__file__)),
                  markdown_sha256=sha(args.out_markdown), public_distribution='使用者明確要求公開的戰場截圖與索引；不含可重組圖源包。',
                  scope='30章代表畫面的受控完整Game.Draw；每場獨立PNG，僅等比例縮圖，不宣稱完整地圖、正常通關或DOS原版對拍。')
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(dict(status=report['status'], screenshots=len(rows), markdown=str(args.out_markdown)), ensure_ascii=False))


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument('--receipt', type=Path, action='append', required=True)
    for key in ('catalog', 'coverage', 'out-markdown', 'out-images', 'report'):
        parser.add_argument('--'+key, type=Path, required=True)
    parser.add_argument('--source-catalog', type=Path)
    parser.add_argument('--source-coverage', type=Path)
    build(parser.parse_args())
