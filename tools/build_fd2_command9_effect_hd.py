#!/usr/bin/env python3
"""依READY手繪疊層配方編製敵方指令9的31來源候選，不登錄正式。"""
import argparse
import copy
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageChops
from audit_fd2_command9_effect_source import audit, FIELDS
from build_fd2_gaia_battle_hd import digest, names
from validate_modern_theme_catalog import ROOT, validate

MASTER_KEYS = {'reaper', 'pool', 'columnplain', 'faceglyph'}


def rectangle(value, bounds, crop=False):
    if not isinstance(value, list) or len(value) != 4 or any(type(v) is not int for v in value):
        raise ValueError('圖層矩形格式不同')
    x, y, a, b = value
    w, h = (a-x, b-y) if crop else (a, b)
    if x < 0 or y < 0 or w <= 0 or h <= 0 or x+w > bounds[0] or y+h > bounds[1]:
        raise ValueError('圖層矩形越界')
    return value


def compile_frames(pack, reference, recipe):
    current, _ = audit(pack)
    # 參考圖SHA是稽核CLI附加值；其餘欄位必須由當前固定包重生相同。
    if {k: v for k, v in reference.items() if k != 'reference_png_sha256'} != current:
        raise ValueError('固定圖源稽核不同')
    if (set(recipe) != {'schema_version', 'reference_sha256', 'masters', 'frames', 'policy'}
            or recipe['schema_version'] != 1 or set(recipe['masters']) != MASTER_KEYS
            or not isinstance(recipe['policy'], str) or not recipe['policy']
            or len(recipe['frames']) != 31):
        raise ValueError('未知指令9配方或來源範圍')
    masters, records = {}, {}
    for key, row in recipe['masters'].items():
        if set(row) != {'file', 'sha256', 'crop'}:
            raise ValueError('未知母稿欄位')
        path = (ROOT/row['file']).resolve()
        if not path.is_relative_to(ROOT) or path.stat().st_uid != os.getuid() or digest(path) != row['sha256']:
            raise ValueError('母稿路徑、SHA或擁有權不同')
        with Image.open(path) as im:
            if im.mode != 'RGBA' or not all(256 <= v <= 4096 for v in im.size):
                raise ValueError('母稿RGBA或形狀不同')
            if im.getchannel('A').getextrema()[0] != 0 or im.getchannel('A').getextrema()[1] < 250:
                raise ValueError('母稿透明底或實體核心不足')
            rectangle(row['crop'], im.size, True)
            masters[key] = im.copy()
        records[key] = {'file': str(path.relative_to(ROOT)), 'sha256': digest(path), 'crop': row['crop']}
    folder = pack/'animations/FDOTHER_044'
    native = json.loads((folder/'animation.json').read_text())
    pending = []
    for index, (row, proof) in enumerate(zip(recipe['frames'], current['frames'])):
        if (set(row) != {'frame', 'native_empty', 'clip_native_mask', 'layers'}
                or type(row['frame']) is not int or row['frame'] != index
                or type(row['native_empty']) is not bool or row['native_empty'] != proof['native_empty']
                or type(row['clip_native_mask']) is not bool or row['clip_native_mask'] != (27 <= index <= 29)
                or not isinstance(row['layers'], list)):
            raise ValueError('配方原幀定位、空拍或消散政策不同')
        expected = [] if index == 30 else ['reaper'] if index == 0 else ['pool'] if index <= 4 else ['columnplain']
        if 12 <= index <= 23:
            expected += ['faceglyph']
        if [v.get('master') for v in row['layers']] != expected:
            raise ValueError('配方圖層種類或順序不同')
        size = (proof['width']*6, proof['height']*6)
        out = Image.new('RGBA', size)
        for layer in row['layers']:
            if set(layer) != {'master', 'crop', 'box'}:
                raise ValueError('未知圖層欄位')
            master = masters[layer['master']]
            crop = rectangle(layer['crop'], master.size, True)
            x, y, w, h = rectangle(layer['box'], size)
            picture = master.crop(crop).resize((w, h), Image.Resampling.LANCZOS)
            out.alpha_composite(picture, (x, y))
        if row['clip_native_mask']:
            with Image.open(folder/native['frames'][index]['mask_path']) as mask:
                mask = mask.convert('L').resize(size, Image.Resampling.LANCZOS)
            out.putalpha(ImageChops.multiply(out.getchannel('A'), mask))
        low, high = out.getchannel('A').getextrema()
        if (index == 30 and high != 0) or (index != 30 and (low != 0 or high != 255)):
            raise ValueError('原空拍新增墨跡或非空幀缺少255核心')
        pending.append(out)
    return current, records, pending


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack', 'reference', 'recipe', 'out']:
        p.add_argument('--'+key, required=True, type=Path)
    p.add_argument('--base-catalog', type=Path, default=ROOT/'remake/assets/themes/modern/catalog.json')
    args = p.parse_args()
    out = args.out.resolve()
    if (not Path('/.dockerenv').exists() or out.exists() or not out.is_relative_to(ROOT)
            or not out.parent.is_dir() or out.parent.stat().st_uid != os.getuid()):
        raise SystemExit('在Docker使用本人擁有的新候選目錄')
    recipe, reference = json.loads(args.recipe.read_text()), json.loads(args.reference.read_text())
    if recipe.get('reference_sha256') != digest(args.reference):
        raise SystemExit('配方參考SHA不同')
    native, records, pending = compile_frames(args.original_pack, reference, recipe)
    base = validate(False, args.base_catalog)
    if any(a['role'] == 'battle_effect_hd_set' and a['source_resource'] == 44 for a in base['assets']):
        raise SystemExit('FDOTHER44已登錄，不覆寫')
    private, old_names = ROOT/base['private_root'], names(base)
    source_names = {k: f'command9-ai-{k}-hd6-master-v1.png' for k in sorted(records)}
    frame_names = [f'fdother-044-hd6-v1-f{i:02d}.png' for i in range(31)]
    if (old_names & (set(source_names.values()) | set(frame_names))
            or any(Path(n).name != n or not (private/n).is_file() for n in old_names)):
        raise SystemExit('基底缺件或新檔名衝突')
    out.mkdir(); pack = out/'pack'; pack.mkdir()
    shutil.copyfile(args.base_catalog, out/'base-catalog.json')
    for name in sorted(old_names):
        (pack/name).symlink_to(os.path.relpath(private/name, pack))
    for key, name in source_names.items():
        shutil.copyfile(ROOT/records[key]['file'], pack/name)
    frames = []
    for i, picture in enumerate(pending):
        name = frame_names[i]; picture.save(pack/name)
        frames.append({'frame': i, 'file': name, 'sha256': digest(pack/name), 'native_empty': i == 30,
                       'native_descriptor': {k: native['frames'][i][k] for k in FIELDS}, 'layers': recipe['frames'][i]['layers']})
    candidate = copy.deepcopy(base); candidate['private_root'] = str(pack.relative_to(ROOT))
    candidate['assets'].append({'asset_id': 'modern.fdother.044.hd6.v1', 'role': 'battle_effect_hd_set', 'status': 'runtime_candidate',
        'files': frame_names, 'width': 1920, 'height': 1200, 'frame_count': 31, 'frame_sha256': [v['sha256'] for v in frames],
        'consumer_contract': 'native_fdother_command9_ai_rgba_hd6_v1', 'alpha_contract': 'coverage',
        'source_refs': ['asset:FDOTHER_044', 'docs/data/fd2-reference-files.json'], 'source_archive': 'FDOTHER.DAT', 'source_resource': 44,
        'source_metadata_sha256': native['metadata_sha256'], 'native_header': [native['native_header'][k] for k in ['byte_1', 'byte_2', 'byte_4']],
        'native_frames': [v['native_descriptor'] for v in frames], 'cycle_policy': 'native_command9_ai_frames', 'layout_policy': 'native_descriptor_hd6',
        'source_files': list(source_names.values()), 'source_sha256': [records[k]['sha256'] for k in source_names]})
    path = out/'catalog.json'; path.write_text(json.dumps(candidate, ensure_ascii=False, indent=2)+'\n')
    validate(True, path)
    report = {'status': 'PASS-COMPILED-COMMAND9-AI-CANDIDATE', 'runtime_registered': False,
        'base_catalog_sha256': digest(args.base_catalog), 'catalog_sha256': digest(path), 'reference_sha256': digest(args.reference),
        'recipe_sha256': digest(args.recipe), 'compiler_sha256': digest(Path(__file__)), 'masters': records, 'frames': frames,
        'scope': recipe['policy']+' 非原版RGB parity、PLAYER-E2或集中人工抽驗。'}
    (out/'compiled.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'frames': 31, 'blank_frames': [30], 'assets': len(candidate['assets'])}))


if __name__ == '__main__':
    main()
