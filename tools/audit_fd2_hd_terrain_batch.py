#!/usr/bin/env python3
"""核對一份真實圖集收據並推進批次狀態；不產生GPU比較結果。"""
import argparse
import hashlib
import json
import re
from pathlib import Path
import shutil

from PIL import Image
from build_fd2_hd_terrain_atlas import read, write

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def audit(batch, atlas, gpu_run='gpu-r1'):
    if (not Path('/.dockerenv').exists() or atlas not in range(8) or
            not re.fullmatch(r'gpu-r[1-9][0-9]*', gpu_run)):
        raise ValueError('需Docker與圖集0至7')
    batch.relative_to(ROOT / 'work')
    work = batch / f'atlas-{atlas:02d}'
    ready = read(batch / 'ready-source.json')
    batch_id = ready.get('batch_id', 'r74')
    if not re.fullmatch(r'r[1-9][0-9]{1,3}', batch_id) or ready['status'] != 'READY':
        raise ValueError('批次名稱或READY不同')
    row = next(a for a in ready['atlases'] if a['atlas'] == atlas)
    recipe_path = ROOT / f'remake/assets/themes/modern/hd1-terrain-{batch_id}-atlas{atlas:02d}.recipe.json'
    recipe, manifest = read(recipe_path), read(work / 'candidate-r2/manifest.json')
    gpu = read(work / (gpu_run + '-capture/receipt.json'))
    count = row['bindings']
    if (manifest['recipe_sha256'] != sha(recipe_path) or len(manifest['bindings']) != count or
            gpu['status'] != 'PASS-PAINTED-TERRAIN-ATLAS-GPU' or
            gpu['catalog_sha256'] != manifest['catalog_sha256'] or
            gpu['manifest_sha256'] != sha(work / 'candidate-r2/manifest.json') or
            (gpu['source_cases'], gpu['complete_game_cases'], len(gpu['cases'])) != (count*4, 2, count*4+2) or
            gpu['save_bytes_unchanged'] != 169):
        raise ValueError('實際來源／GPU收據不完整')
    logs = ['source-r1', gpu_run] + (['compilation-tests-r1', 'registration-tests-r1'] if atlas == 0 else [])
    for name in logs:
        memory = read(work / (name + '.memory.json'))
        if memory['returncode'] != 0 or memory['timeout'] or memory['memory_events_delta']['oom_kill']:
            raise ValueError('程序或記憶體檢查失敗：' + name)
    source_log = (work / 'source-r1.log').read_text()
    if (f'{count} source transactions and {manifest["new_sets"]} wrong-mask admissions checked' not in source_log or
            '--- PASS: TestModernHDPaintedTerrainAtlasSourceTransactions' not in source_log):
        raise ValueError('來源交易沒有真正執行')
    if atlas == 0:
        for name, tests in [('compilation-tests-r1.log', 2), ('registration-tests-r1.log', 1)]:
            text = (work / name).read_text()
            if f'Ran {tests} test' not in text or 'OK' not in text or 'skipped' in text:
                raise ValueError('首份Python驗證缺項')
    prototype_path = work / 'direct-prototype-r1/prototype.json'
    if sha(prototype_path) != row['prototype_sha256']:
        raise ValueError('已審原型不同')
    prototype = {c['cell']: c for c in read(prototype_path)['cases'] if c['accepted']}
    mother = Image.open(work / 'atlas-r1-master.png').convert('RGBA')
    if sha(work / 'atlas-r1-master.png') != row['raw_master_sha256']:
        raise ValueError('原始生成母稿不同')
    for group in recipe['groups']:
        cell = group['cell']
        expected = work / 'direct-prototype-r1' / prototype[cell]['file']
        candidate = work / 'candidate-r2' / group['file']
        registered = Path('/modern') / group['file']
        rebuilt = work / 'rebuilt-from-private-r2' / group['file']
        if not candidate.read_bytes() == expected.read_bytes() == registered.read_bytes() == rebuilt.read_bytes():
            raise ValueError('原型／候選／私人／重編不同')
        with Image.open(registered) as image:
            if image.size != (144, 144) or image.getextrema()[3] != (255, 255):
                raise ValueError('成品尺寸或原生透明不同')
        private_master = Path('/modern') / group['master_file']
        if private_master.read_bytes() != (work / 'candidate-r2' / group['master_file']).read_bytes():
            raise ValueError('私人裁切母稿不同')
        if Image.open(private_master).convert('RGBA').tobytes() != mother.crop(group['crop']).tobytes():
            raise ValueError('完整裁切母稿pixels被改動')
    catalog_path = ROOT / 'remake/assets/themes/modern/catalog.json'
    if (catalog_path.read_bytes() != Path('/modern/catalog.json').read_bytes() or
            read(catalog_path)['assets'] != read(work / 'candidate-r2/catalog.json')['assets']):
        raise ValueError('正式與私人清冊不同')
    before, coverage = read(work / 'base-coverage.json'), read(work / 'formal-coverage-r1.json')
    summary = coverage['summary']
    if (coverage['catalog_sha256'] != sha(catalog_path) or summary['maps'] != 33 or
            summary['registered-hd'] != before['summary']['registered-hd']+count or
            summary['missing-art'] != before['summary']['missing-art']-count or
            summary['missing_unique_native_content'] != before['summary']['missing_unique_native_content']-len(recipe['groups']) or
            summary['required_slots'] != 3998 or summary['native-remap-only'] != 10):
        raise ValueError('官方覆蓋與本批增量不同')
    private_manifest = read(Path('/modern/private-manifest.json'))
    prefix_cells = sum(a['accepted_cells'] for a in ready['atlases'] if a['atlas'] <= atlas)
    if private_manifest['file_count'] != ready['initial_private_files'] + prefix_cells*2:
        raise ValueError('私人白名單數不同')
    for target in [work, catalog_path, ROOT / 'docs/data/fd2-hd1-terrain-coverage.json']:
        if (target.stat().st_uid, target.stat().st_gid) != (1000, 1000):
            raise ValueError('寫入目標擁有權不同')
    snapshot = work / 'registered-catalog.json'
    if snapshot.exists() and snapshot.read_bytes() != catalog_path.read_bytes():
        raise ValueError('已有不同正式清冊快照')
    snapshot.write_bytes(catalog_path.read_bytes())
    shutil.copyfile(work / 'formal-coverage-r1.json', ROOT / 'docs/data/fd2-hd1-terrain-coverage.json')
    proof_names = ['staged-inputs.json', 'candidate-r2/manifest.json', gpu_run + '-capture/receipt.json',
                   'source-r1.log', 'source-r1.memory.json', gpu_run + '.memory.json', 'formal-coverage-r1.json',
                   'registered-catalog.json', 'rebuilt-from-private-r2/receipt.json', 'direct-prototype-r1/prototype.json']
    if atlas == 0:
        proof_names += ['compilation-tests-r1.log', 'compilation-tests-r1.memory.json',
                        'registration-tests-r1.log', 'registration-tests-r1.memory.json']
    result = {'status': 'PASS-ATLAS-GPU-PRIVATE-REBUILD', 'atlas': atlas, 'cells': len(recipe['groups']),
              'bindings': count, 'source_cases': count*4, 'complete_game_cases': 2,
              'new_sets': manifest['new_sets'], 'catalog_assets': len(read(catalog_path)['assets']),
              'catalog_sha256': sha(catalog_path), 'private_files': private_manifest['file_count'],
              'private_manifest_sha256': sha(Path('/modern/private-manifest.json')), 'coverage': summary,
              'prototype_and_private_rebuild_exact': True, 'crop_master_pixels_preserved': True,
              'proofs': [{'file': str((work / name).relative_to(ROOT)), 'sha256': sha(work / name)} for name in proof_names],
              'tool_sha256': sha(Path(__file__)), 'normal_and_final_regression': 'pending whole batch'}
    if gpu_run != 'gpu-r1':
        result['gpu_run'] = gpu_run
    write(work / 'receipt.json', result)
    progress_path = batch / 'progress.json'
    progress = read(progress_path) if progress_path.exists() else {'status': 'IN-PROGRESS', 'completed': []}
    if len(progress['completed']) != atlas:
        raise ValueError('批次狀態不是依序推進')
    progress['completed'].append({'atlas': atlas, 'cells': result['cells'], 'bindings': count,
                                  'receipt': str((work / 'receipt.json').relative_to(ROOT)),
                                  'sha256': sha(work / 'receipt.json')})
    progress.update(registered_cells=sum(a['cells'] for a in progress['completed']),
                    registered_bindings=sum(a['bindings'] for a in progress['completed']), coverage=summary,
                    status='RUNTIME-E1-PENDING-BATCH-REGRESSION')
    write(progress_path, progress)
    state_path = batch.parent / 'hd1-completion-r29/current-state.json'
    state = read(state_path)
    state['active_slice'].update(status='RUNTIME-E1-MULTI-ATLAS-PROGRESS',
                                 scope=f'八份中已核{atlas+1}份，新增{progress["registered_bindings"]}槽；整批正常操作／正式回歸待驗。')
    state['independent_terrain'].update(registered_hd=summary['registered-hd'], missing_slots=summary['missing-art'],
                                        missing_unique=summary['missing_unique_native_content'], work_root=str(batch.relative_to(ROOT)))
    write(state_path, state)
    return {k: v for k, v in result.items() if k != 'proofs'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--batch-root', type=lambda p: Path(p).resolve(), required=True)
    parser.add_argument('--atlas', type=int, required=True)
    parser.add_argument('--gpu-run', default='gpu-r1')
    args = parser.parse_args()
    print(json.dumps(audit(args.batch_root, args.atlas, args.gpu_run), ensure_ascii=False))


if __name__ == '__main__':
    main()
