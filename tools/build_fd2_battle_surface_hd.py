#!/usr/bin/env python3
"""在 Docker 從已審查 BG_004 原稿編製獨立候選包，不修改正式素材。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image
from audit_fd2_battle_background_sources import audit
from validate_modern_theme_catalog import ROOT, validate


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--master', type=Path, required=True)
    p.add_argument('--base-catalog', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or not a.out.parent.is_dir() or a.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('Docker 內使用本人擁有的全新輸出目錄')
    expected = json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['background_toolbar_prototype_20261023']['bg004_art_draft']['master']['sha256']
    if sha(a.master) != expected:
        raise ValueError('已審查原稿 SHA 不同')
    inventory, _ = audit(a.original_pack)
    source = inventory['banks']['BG']['entries'][4]
    if (source['width'], source['height'], source['visible_pixels']) != (320,100,32000):
        raise ValueError('BG_004 原框或全不透明遮罩不同')
    catalog = validate(True, a.base_catalog)
    if any(e['role'] == 'battle_surface_hd' and e['source_archive'] == 'BG.DAT' and e['source_resource'] == 4 for e in catalog['assets']):
        raise ValueError('既有清冊已宣告 BG_004')
    base_root = ROOT/catalog['private_root']
    a.out.mkdir()
    names = set()
    for e in catalog['assets']:
        names.update(e.get('files', [])); names.update(e.get('source_files', []))
        for k in ['file','master_file']:
            if k in e: names.add(e[k])
    for name in sorted(names):
        if Path(name).name != name or not (base_root/name).is_file(): raise ValueError('舊清冊來源不安全或缺件')
        (a.out/name).symlink_to(os.path.relpath(base_root/name, a.out))
    master_name, name = 'battle-bg-004-master-oct23-r1.png', 'battle-bg-004-hd6-oct23-r1.png'
    shutil.copyfile(a.master, a.out/master_name)
    with Image.open(a.master) as im:
        if im.mode != 'RGB' or im.size != (2243,701): raise ValueError('原稿形狀不同')
        im.resize((1920,600), Image.Resampling.LANCZOS).convert('RGBA').save(a.out/name)
    descriptor = dict(x=0,y=0,width=320,height=100,delay_native=0,raw_byte_4=0,raw_byte_5=0,raw_byte_7=0)
    entry = dict(asset_id='modern.battle.bg004.hd6', role='battle_surface_hd', status='runtime_candidate',
                 file=name, width=1920, height=600, sha256=sha(a.out/name), master_file=master_name,
                 source_refs=['asset:BG_004', 'docs/knowledge-base/61-modern-theme-runtime-spec.md'],
                 consumer_contract='native_battle_surface_rgba_hd6_v1', source_archive='BG.DAT', source_resource=4,
                 source_metadata_sha256=source['metadata_sha256'], source_files=[master_name], source_sha256=[expected],
                 native_frames=[descriptor], alpha_contract='coverage', layout_policy='native_surface_hd6')
    catalog['assets'].append(entry)
    catalog['private_root'] = str(a.out.resolve().relative_to(ROOT))
    catalog_path = a.out/'catalog.json'
    catalog_path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    validate(True, catalog_path)
    report = dict(status='PASS-BG004-CANDIDATE', tool_sha256=sha(Path(__file__)), base_catalog_sha256=sha(a.base_catalog),
                  catalog_sha256=sha(catalog_path), master_sha256=expected, compiled_sha256=entry['sha256'],
                  source=source, recipe='unmodified RGB master -> full-frame LANCZOS resize2243x701 to1920x600 -> RGBA; no mask edits',
                  prior_files=len(names), prior_files_are_links=True, production_changed=False,
                  scope='僅編製與完整清冊驗證；尚非獨立實檔包、Game.Draw 或正常玩家驗收。')
    (a.out/'recipe.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'assets':len(catalog['assets']), 'sha256':entry['sha256']}))


if __name__ == '__main__': main()
