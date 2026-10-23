#!/usr/bin/env python3
"""在 Docker 從已審查 BG／TAI 原稿編製獨立候選包。"""
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


def compile_reviewed_batch(a):
    state = json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())['forest_coast_surface_followup_20261024']
    if state['status'] not in {'READY-ART-REVIEWED', 'CONFORMED-SCOPED/RUNTIME-E1'} or sha(a.recipe) != state['art_recipe']['sha256']:
        raise ValueError('配方尚未READY或SHA不同')
    recipe = json.loads(a.recipe.read_text())
    if recipe.get('schema_version') != 1 or {(e['archive'],e['resource']) for e in recipe['surfaces']} != {('BG.DAT',5),('BG.DAT',6),('TAI.DAT',5),('TAI.DAT',6)} or len(recipe['surfaces']) != 4:
        raise ValueError('批次來源集合不同')
    inventory, _ = audit(a.original_pack)
    catalog = validate(True, a.base_catalog)
    existing = {(e.get('source_archive'),e.get('source_resource')) for e in catalog['assets'] if e['role']=='battle_surface_hd'}
    prepared = []
    for row in recipe['surfaces']:
        archive, index = row['archive'], row['resource']
        source = inventory['banks'][archive[:-4]]['entries'][index]
        pinned = next(r for r in state['sources'][archive[:-4]] if r['index']==index)
        if json.loads(json.dumps(source)) != pinned or (archive,index) in existing:
            raise ValueError('固定原版圖源不同或已登錄')
        master = ROOT/row['master']
        if sha(master) != row['master_sha256']:
            raise ValueError('原稿SHA不同')
        with Image.open(master) as im:
            if im.mode != row['master_mode'] or list(im.size) != row['master_size']:
                raise ValueError('原稿形狀不同')
            if archive == 'TAI.DAT':
                bbox = im.getchannel('A').point(lambda v:255 if v>=64 else 0).getbbox()
                if list(bbox or []) != row['alpha64_bbox']:
                    raise ValueError('原稿核心框不同')
                crop = [max(0,bbox[0]-2),max(0,bbox[1]-2),min(im.width,bbox[2]+2),min(im.height,bbox[3]+2)]
                if crop != row['crop']:
                    raise ValueError('裁切配方不同')
                im = im.crop(crop)
            elif row['crop'] is not None:
                raise ValueError('背景不接受額外裁切')
            picture = im.resize((source['width']*6,source['height']*6),Image.Resampling.LANCZOS).convert('RGBA')
            if picture.getchannel('A').getextrema() != ((255,255) if archive=='BG.DAT' else (0,255)):
                raise ValueError('成品透明／不透明端點不同')
            if picture.tobytes() != Image.open(ROOT/row['review']['file']).convert('RGBA').tobytes() or sha(ROOT/row['review']['file']) != row['review']['sha256']:
                raise ValueError('編製結果與已審圖不同')
        descriptor = dict(x=0,y=0,width=source['width'],height=source['height'],delay_native=0,raw_byte_4=0,raw_byte_5=0,raw_byte_7=0)
        entry = dict(asset_id=f'modern.battle.{archive[:-4].lower()}{index:03d}.hd6',role='battle_surface_hd',status='runtime_candidate',
                     file=row['file'],width=source['width']*6,height=source['height']*6,master_file=row['master_file'],
                     source_refs=[f'asset:{archive[:-4]}_{index:03d}','docs/knowledge-base/61-modern-theme-runtime-spec.md'],
                     consumer_contract='native_battle_surface_rgba_hd6_v1',source_archive=archive,source_resource=index,
                     source_metadata_sha256=source['metadata_sha256'],source_files=[row['master_file']],source_sha256=[row['master_sha256']],
                     native_frames=[descriptor],alpha_contract='coverage',layout_policy='native_surface_hd6')
        prepared.append((row,master,picture,entry))
    names = set()
    for e in catalog['assets']:
        names.update(e.get('files',[])); names.update(e.get('source_files',[]))
        names.update(e[k] for k in ['file','master_file'] if k in e)
    additions = [r[k] for r,_,_,_ in prepared for k in ['file','master_file']]
    base = ROOT/catalog['private_root']
    if len(set(additions)) != 8 or set(additions)&names or any(Path(n).name != n for n in names|set(additions)) or any(not (base/n).is_file() for n in names):
        raise ValueError('白名單名稱碰撞或來源缺件')
    a.out.mkdir()
    for name in sorted(names):
        (a.out/name).symlink_to(os.path.relpath(base/name,a.out))
    for row,master,picture,entry in prepared:
        shutil.copyfile(master,a.out/row['master_file'])
        picture.save(a.out/row['file'])
        entry['sha256'] = sha(a.out/row['file'])
        catalog['assets'].append(entry)
    catalog['private_root'] = str(a.out.resolve().relative_to(ROOT))
    path = a.out/'catalog.json'
    path.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n')
    validate(True,path)
    report = dict(status='PASS-REVIEWED-SURFACE-BATCH',recipe_sha256=sha(a.recipe),base_catalog_sha256=sha(a.base_catalog),
                  catalog_sha256=sha(path),tool_sha256=sha(Path(__file__)),old_files_are_links=True,prior_files=len(names),
                  surfaces=[e for _,_,_,e in prepared],production_changed=False)
    (a.out/'recipe.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'assets':len(catalog['assets']),'new_surfaces':len(prepared)}))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--original-pack', type=Path, required=True)
    inputs = p.add_mutually_exclusive_group(required=True)
    inputs.add_argument('--master', type=Path)
    inputs.add_argument('--recipe', type=Path)
    p.add_argument('--archive', choices=['BG.DAT', 'TAI.DAT'], default='BG.DAT')
    p.add_argument('--base-catalog', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or not a.out.parent.is_dir() or a.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('Docker 內使用本人擁有的全新輸出目錄')
    if a.recipe:
        compile_reviewed_batch(a)
        return
    state = json.loads((ROOT/'docs/data/fd2-modern-hd-battlefield-20261019.json').read_text())
    draft = (state['background_toolbar_prototype_20261023']['bg004_art_draft'] if a.archive == 'BG.DAT'
             else state['tai004_physical_followup_20261023']['art_draft'])
    expected = draft['master']['sha256']
    if sha(a.master) != expected:
        raise ValueError('已審查原稿 SHA 不同')
    inventory, _ = audit(a.original_pack)
    prefix = a.archive[:-4]
    source = inventory['banks'][prefix]['entries'][4]
    wanted = (320,100,32000) if prefix == 'BG' else (154,42,5366)
    if (source['width'], source['height'], source['visible_pixels']) != wanted:
        raise ValueError('固定單圖原框或遮罩不同')
    catalog = validate(True, a.base_catalog)
    if any(e['role'] == 'battle_surface_hd' and e['source_archive'] == a.archive and e['source_resource'] == 4 for e in catalog['assets']):
        raise ValueError('既有清冊已宣告此單圖')
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
    master_name, name = f'battle-{prefix.lower()}-004-master-oct23-r1.png', f'battle-{prefix.lower()}-004-hd6-oct23-r1.png'
    shutil.copyfile(a.master, a.out/master_name)
    with Image.open(a.master) as im:
        if prefix == 'BG':
            if im.mode != 'RGB' or im.size != (2243,701): raise ValueError('原稿形狀不同')
            picture = im.resize((1920,600), Image.Resampling.LANCZOS).convert('RGBA')
            recipe = 'unmodified RGB master -> full-frame LANCZOS resize2243x701 to1920x600 -> RGBA; no mask edits'
        else:
            if im.mode != 'RGBA' or im.size != (2170,725): raise ValueError('透明原稿形狀不同')
            bbox = im.getchannel('A').point(lambda v:255 if v>=64 else 0).getbbox()
            if list(bbox or []) != draft['alpha64_bbox']: raise ValueError('原稿核心框不同')
            crop = (max(0,bbox[0]-2),max(0,bbox[1]-2),min(im.width,bbox[2]+2),min(im.height,bbox[3]+2))
            if list(crop) != draft['compile_crop']: raise ValueError('裁切配方不同')
            picture = im.crop(crop).resize((924,252),Image.Resampling.LANCZOS)
            if picture.getchannel('A').getextrema() != (0,255): raise ValueError('編製後缺少透明背景或255核心，不可放寬端點')
            recipe = f'unmodified RGBA master -> alpha64 bbox expanded2 crop{crop} -> LANCZOS924x252; preserve in-crop alpha; no alpha painting'
        picture.save(a.out/name)
    descriptor = dict(x=0,y=0,width=source['width'],height=source['height'],delay_native=0,raw_byte_4=0,raw_byte_5=0,raw_byte_7=0)
    entry = dict(asset_id=f'modern.battle.{prefix.lower()}004.hd6', role='battle_surface_hd', status='runtime_candidate',
                 file=name, width=source['width']*6, height=source['height']*6, sha256=sha(a.out/name), master_file=master_name,
                 source_refs=[f'asset:{prefix}_004', 'docs/knowledge-base/61-modern-theme-runtime-spec.md'],
                 consumer_contract='native_battle_surface_rgba_hd6_v1', source_archive=a.archive, source_resource=4,
                 source_metadata_sha256=source['metadata_sha256'], source_files=[master_name], source_sha256=[expected],
                 native_frames=[descriptor], alpha_contract='coverage', layout_policy='native_surface_hd6')
    catalog['assets'].append(entry)
    catalog['private_root'] = str(a.out.resolve().relative_to(ROOT))
    catalog_path = a.out/'catalog.json'
    catalog_path.write_text(json.dumps(catalog, ensure_ascii=False, indent=2)+'\n')
    validate(True, catalog_path)
    report = dict(status=f'PASS-{prefix}004-CANDIDATE', tool_sha256=sha(Path(__file__)), base_catalog_sha256=sha(a.base_catalog),
                  catalog_sha256=sha(catalog_path), master_sha256=expected, compiled_sha256=entry['sha256'],
                  source=source, recipe=recipe,
                  prior_files=len(names), prior_files_are_links=True, production_changed=False,
                  scope='僅編製與完整清冊驗證；尚非獨立實檔包、Game.Draw 或正常玩家驗收。')
    (a.out/'recipe.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'assets':len(catalog['assets']), 'sha256':entry['sha256']}))


if __name__ == '__main__': main()
