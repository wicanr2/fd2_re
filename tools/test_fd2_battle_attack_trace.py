#!/usr/bin/env python3
"""以實際完整追蹤的缺項／錯配反例驗證分段收據；不修改原始擷取。"""
import argparse
import copy
import json
from pathlib import Path
import shutil
import subprocess
import sys

from PIL import Image
from verify_fd2_battle_attack_trace import ROOT, digest, inspect_run, validate_receipt


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('run', 'binary', 'catalog', 'source-save', 'baseline', 'out'):
        p.add_argument('--'+name, type=Path, required=True)
    a = p.parse_args()
    assert Path('/.dockerenv').exists() and not a.out.exists() and a.out.resolve().is_relative_to(ROOT)
    baseline = json.loads(a.baseline.read_text())
    validate_receipt(baseline, a.baseline, a.catalog)
    a.out.mkdir()
    checks = []

    def shadow(name):
        target = a.out/name
        target.mkdir()
        for file in ('battle-observation.json', 'operations.json', 'input-audit.jsonl'):
            shutil.copyfile(a.run/file, target/file)
        saved = target/'user-data/fd2_re'
        saved.mkdir(parents=True)
        shutil.copyfile(a.run/'user-data/fd2_re/fd2_save.json', saved/'fd2_save.json')
        for sample in baseline['battle_art_samples']:
            (target/sample['file']).symlink_to((a.run/sample['file']).resolve())
        return target

    def change(target, name, mutate):
        path = target/name
        doc = json.loads(path.read_text())
        mutate(doc)
        path.write_text(json.dumps(doc))

    def reject(name, mutate, reason):
        target = shadow(name)
        mutate(target)
        try:
            inspect_run(target, a.binary, a.catalog, a.source_save, 30)
        except ValueError as error:
            assert reason in str(error), (name, str(error))
            checks.append({'case': name, 'status': 'PASS-REJECTED', 'reason': str(error)})
        else:
            raise AssertionError('反例未拒收：'+name)

    reject('missing-cold-epoch', lambda t: change(t, 'operations.json',
           lambda d: d.__setitem__(slice(None), [o for o in d if o['process_epoch'] == 0])), '新程序epoch')
    reject('missing-release', lambda t: change(t, 'operations.json',
           lambda d: d[-1]['release_ack'].update(released_keys=[])), '放鍵ACK')
    reject('wrong-actor', lambda t: change(t, 'battle-observation.json',
           lambda d: d['battle_hd_scenes'][1]['presented'].update(battle_actor_fig=1)), '指定角色HD')
    reject('end-not-hd', lambda t: change(t, 'battle-observation.json',
           lambda d: d['battle_approach_steps'][0]['physical_scenes'][0].update(hd_draw_seen=False)), 'END缺少HD')
    reject('missing-png', lambda t: change(t, 'battle-observation.json',
           lambda d: d['battle_art_samples'].pop()), '演出樣本')
    reject('wrong-png-sha', lambda t: change(t, 'battle-observation.json',
           lambda d: d['battle_art_samples'][0].update(sha256='0'*64)), 'PNG路徑或SHA')

    def missing_save(t):
        def mutate(d):
            first = [i for i, o in enumerate(d) if o['process_epoch'] == 0]
            del d[first[-6]]
        change(t, 'operations.json', mutate)
    reject('missing-f5', missing_save, 'F2／F5／F9序列')
    reject('wrong-save-bytes', lambda t: (t/'user-data/fd2_re/fd2_save.json').write_bytes(
           a.source_save.read_bytes()+b'\n'), '邊界保存與來源不同')

    def wrong_cold(t):
        path = t/'input-audit.jsonl'
        rows = [json.loads(x) for x in path.read_text().splitlines()]
        rows[-1]['units'][0]['hp'] += 1
        path.write_text(''.join(json.dumps(x)+'\n' for x in rows))
    reject('wrong-cold-units', wrong_cold, '冷讀重建不同')

    verifier = ROOT/'tools/verify_fd2_character_battle_gpu.py'

    def gpu_reject(name, receipt, group, reason):
        report = a.out/(name+'-report.json')
        result = subprocess.run([sys.executable, str(verifier), '--figure', '30', '--sample-group', group,
            '--receipt', str(receipt), '--catalog', str(a.catalog), '--source', str(a.catalog.parent/'pack'),
            '--report', str(report)], capture_output=True, text=True, timeout=180)
        output = result.stdout+result.stderr
        assert result.returncode != 0 and not report.exists() and reason in output, (name, output)
        checks.append({'case': name, 'status': 'PASS-REJECTED', 'output_published': False,
                       'reason': output.strip().splitlines()[-1]})

    forged = copy.deepcopy(baseline)
    forged['cold_units'][0]['hp'] += 1
    forged['cold_loaded_units_equal'] = True
    forged_path = a.run/'negative-forged-cold-trace.json'
    assert not forged_path.exists()
    forged_path.write_text(json.dumps(forged))
    try:
        gpu_reject('forged-cold-booleans', forged_path, 'trace', '追蹤收據未對上原始證據')
    finally:
        forged_path.unlink()
    gpu_reject('end-scope-substitution', a.baseline, 'end', '需要同份普通攻防完整GUI收據')
    black = shadow('black-pngs-with-valid-control')
    black_path = black/'black.png'
    Image.new('RGB', (1920, 1200)).save(black_path)
    black_sha = digest(black_path)
    def black_samples(doc):
        for sample in doc['battle_art_samples']:
            path = black/sample['file']
            path.unlink()
            path.symlink_to(black_path.resolve())
            sample['sha256'] = black_sha
    change(black, 'battle-observation.json', black_samples)
    report = inspect_run(black, a.binary, a.catalog, a.source_save, 30)
    control = black/'attack-save-trace.json'
    control.write_text(json.dumps(report))
    gpu_reject('black-pixels-despite-valid-control', control, 'trace', '未在普通攻防GPU核對到')
    (a.out/'guards.json').write_text(json.dumps({'status': 'PASS-ATTACK-TRACE-GUARDS',
        'baseline_sha256': digest(a.baseline), 'checks': checks}, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': 'PASS-ATTACK-TRACE-GUARDS', 'guards': len(checks)}))


if __name__ == '__main__':
    main()
