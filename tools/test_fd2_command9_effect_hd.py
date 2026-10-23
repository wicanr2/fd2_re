#!/usr/bin/env python3
"""用真實固定圖源與正對照檢查指令9配方、圖源及清冊拒收。"""
import argparse
import copy
import io
import json
import os
from pathlib import Path
import shutil
import tempfile

from audit_fd2_command9_effect_source import audit
from build_fd2_command9_effect_hd import compile_frames
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack', 'reference', 'recipe', 'compiled', 'out']:
        p.add_argument('--'+key, type=Path, required=True)
    args = p.parse_args()
    if not Path('/.dockerenv').exists() or args.out.exists() or args.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新測試目錄')
    reference, recipe = json.loads(args.reference.read_text()), json.loads(args.recipe.read_text())
    args.out.mkdir()
    _, _, frames = compile_frames(args.original_pack, reference, recipe)
    compiled = json.loads((args.compiled/'compiled.json').read_text())
    if compiled['status'] != 'PASS-COMPILED-COMMAND9-AI-CANDIDATE' or len(frames) != 31:
        raise ValueError('正對照缺件')
    for frame, record in zip(frames, compiled['frames']):
        buffer = io.BytesIO(); frame.save(buffer, format='PNG')
        import hashlib
        if hashlib.sha256(buffer.getvalue()).hexdigest() != record['sha256']:
            raise ValueError('正對照不能重生相同PNG')
    rejected = []
    for fault in ['reference-sha', 'reference-resource', 'reference-empty', 'schema', 'count', 'unknown', 'master-sha', 'master-path', 'crop-shape', 'crop-outside', 'index-bool', 'index-order', 'native-empty', 'late-ink-layer', 'early-empty', 'wrong-master', 'layer-order', 'unknown-layer', 'outside-box', 'zero-size', 'wrong-clip']:
        r, q = copy.deepcopy(reference), copy.deepcopy(recipe)
        if fault == 'reference-sha': r['frames'][30]['mask_sha256'] = '0'*64
        elif fault == 'reference-resource': r['resource'] = 43
        elif fault == 'reference-empty': r['frames'][30]['native_empty'] = False
        elif fault == 'schema': q['schema_version'] = 2
        elif fault == 'count': q['frames'].pop()
        elif fault == 'unknown': q['override'] = True
        elif fault == 'master-sha': q['masters']['reaper']['sha256'] = '0'*64
        elif fault == 'master-path': q['masters']['reaper']['file'] = '/missing.png'
        elif fault == 'crop-shape': q['masters']['reaper']['crop'] = [1, 2, 3]
        elif fault == 'crop-outside': q['masters']['reaper']['crop'][-1] = 99999
        elif fault == 'index-bool': q['frames'][0]['frame'] = False
        elif fault == 'index-order': q['frames'][0]['frame'] = 1
        elif fault == 'native-empty': q['frames'][0]['native_empty'] = True
        elif fault == 'late-ink-layer': q['frames'][30]['layers'] = copy.deepcopy(q['frames'][0]['layers'])
        elif fault == 'early-empty': q['frames'][0]['layers'] = []
        elif fault == 'wrong-master': q['frames'][0]['layers'][0]['master'] = 'pool'
        elif fault == 'layer-order': q['frames'][12]['layers'].reverse()
        elif fault == 'unknown-layer': q['frames'][0]['layers'][0]['opacity'] = 0
        elif fault == 'outside-box': q['frames'][0]['layers'][0]['box'][0] = -1
        elif fault == 'zero-size': q['frames'][0]['layers'][0]['box'][2] = 0
        elif fault == 'wrong-clip': q['frames'][0]['clip_native_mask'] = True
        try:
            compile_frames(args.original_pack, r, q)
        except (ValueError, FileNotFoundError):
            rejected.append(fault)
        else:
            raise ValueError('配方錯誤未拒收：'+fault)
    # 個別圖源複本只改本地檔，不沿symlink回寫唯讀原包。
    for kind in ['metadata', 'pixels', 'mask', 'manifest']:
        with tempfile.TemporaryDirectory(dir=args.out) as tmp:
            root = Path(tmp); folder = root/'animations/FDOTHER_044'; folder.mkdir(parents=True)
            shutil.copyfile(args.original_pack/'manifest.json', root/'manifest.json')
            for source in (args.original_pack/'animations/FDOTHER_044').iterdir():
                (folder/source.name).symlink_to(source)
            doc = json.loads((folder/'animation.json').read_text())
            path = root/'manifest.json' if kind == 'manifest' else folder/('animation.json' if kind == 'metadata' else doc['frames'][30]['path' if kind == 'pixels' else 'mask_path'])
            if path.is_symlink(): path.unlink()
            path.write_bytes(b'altered-source')
            try: audit(root)
            except ValueError: rejected.append('source-'+kind)
            else: raise ValueError('原圖SHA錯誤未拒收：'+kind)
    catalog = validate(False, args.compiled/'catalog.json')
    for fault in ['consumer', 'count', 'header', 'resource', 'cycle', 'delay', 'unsafe', 'duplicate']:
        c = copy.deepcopy(catalog); a = c['assets'][-1]
        if fault == 'consumer': a['consumer_contract'] = 'native_fdother_ordinary_rgba_hd6_v1'
        elif fault == 'count': a['frame_count'] = 30
        elif fault == 'header': a['native_header'][2] = 1
        elif fault == 'resource': a['source_resource'] = 45
        elif fault == 'cycle': a['cycle_policy'] = 'native_ordinary_frames'
        elif fault == 'delay': a['native_frames'][30]['delay_native'] = -1
        elif fault == 'unsafe': a['files'][30] = '../escape.png'
        elif fault == 'duplicate': c['assets'].append(dict(a, asset_id='duplicate.command9'))
        path = args.out/(fault+'-catalog.json'); path.write_text(json.dumps(c))
        try: validate(False, path)
        except ValueError: rejected.append('catalog-'+fault)
        else: raise ValueError('清冊錯誤未拒收：'+fault)
    report = {'status': 'PASS-COMMAND9-COMPILER-REFUSAL', 'positive_frames': 31, 'negative_controls': rejected,
              'reference_sha256': digest(args.reference), 'recipe_sha256': digest(args.recipe), 'compiler_sha256': digest(Path(__file__).with_name('build_fd2_command9_effect_hd.py')), 'test_sha256': digest(Path(__file__))}
    (args.out/'receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'positive_frames': 31, 'negative_controls': len(rejected)}))


if __name__ == '__main__': main()
