#!/usr/bin/env python3
"""用既有正常GUI收據驗補取景的完整圖塊、F2狀態與來源拒收。"""
import argparse
import copy
import json
from pathlib import Path
import subprocess
import sys
import tempfile


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--receipt', type=Path, required=True)
    p.add_argument('--legacy-receipt', type=Path, required=True)
    p.add_argument('--report', type=Path, required=True)
    a = p.parse_args()
    if a.report.exists():
        raise ValueError('不得覆寫反例收據')
    original = json.loads(a.receipt.read_text())
    flags = ['--supplemental-frame', '148', '--supplemental-before', 'world-hd-ring.png',
             '--supplemental-faithful', 'world-faithful-ring.png',
             '--supplemental-after', 'world-hd-ring-restored.png']
    tool = Path(__file__).with_name('verify_fd2_hd_terrain_gpu.py')
    results = []
    with tempfile.TemporaryDirectory(dir=a.report.parent) as tmp:
        tmp = Path(tmp)
        for name in original['screenshots']:
            (tmp/name).symlink_to((a.receipt.parent/name).resolve())
        def run(name, receipt, argv, error=None, source=None):
            path = tmp/'receipt.json'
            path.write_text(json.dumps(receipt))
            out = tmp/(name+'.json')
            r = subprocess.run([sys.executable, str(tool), '--receipt', str(source or path),
                                '--report', str(out), *argv], capture_output=True, text=True, timeout=20)
            if error:
                assert r.returncode != 0 and error in r.stderr, (name, r.stdout, r.stderr)
                assert not out.exists()
            else:
                assert r.returncode == 0, (name, r.stdout, r.stderr)
                assert json.loads(out.read_text())['status'] == 'PASS'
            results.append({'case': name, 'expected': error or 'PASS', 'returncode': r.returncode})
        run('positive-supplement', original, flags)
        proof = json.loads((tmp/'positive-supplement.json').read_text())
        assert all(any(t['frame'] == 148 for t in s['not_compared_tiles']) for s in proof['samples'])
        assert all(any(t['frame'] == 148 for t in s['matching_tiles']) for s in proof['supplemental']['samples'])
        run('default-keeps-full-range', original, [], '明示範圍')
        run('not-required-frame', original, ['--supplemental-frame', '52', *flags[2:]], '補驗必須')
        run('duplicate-frame', original, [*flags, '--supplemental-frame', '148'], '補驗必須')
        run('unsafe-path', original, [*flags[:3], '../world-hd-ring.png', *flags[4:]], '單一PNG')
        for name, expected in [('changed-sha', 'GPU圖片被改寫'), ('wrong-view', 'view不一致'),
                               ('wrong-theme', '主題不符'), ('missing-faithful', 'world-faithful-ring.png'),
                               ('no-cold-load', '保存冷讀'), ('missing-full-frame', '明示範圍')]:
            r = copy.deepcopy(original)
            if name == 'changed-sha':
                r['screenshots']['world-hd-ring.png']['sha256'] = '0'*64
            elif name == 'wrong-view':
                r['screenshots']['world-hd-ring-restored.png']['state']['view']['CameraY'] += 1
            elif name == 'wrong-theme':
                r['screenshots']['world-faithful-ring.png']['state']['hd_theme'] = True
            elif name == 'missing-faithful':
                del r['screenshots']['world-faithful-ring.png']
            elif name == 'no-cold-load':
                r['new_process_title_load'] = False
            else:
                for k in ('gpu_before', 'state'):
                    s = r['screenshots']['world-hd-ring-restored.png'][k]
                    s['hd_world_terrain'] = [t for t in s['hd_world_terrain'] if t['frame'] != 148]
            run(name, r, flags, expected)
        run('legacy-default-positive', original, [], source=a.legacy_receipt)
    a.report.write_text(json.dumps({'status': 'PASS-TERRAIN-GUI-GUARDS', 'checks': results,
                                   'scope': '既有正常PNG及收據；不重播遊戲，不新增原版對拍。'},
                                  ensure_ascii=False, indent=2)+'\n')
    print('PASS', len(results), 'terrain GUI checks')


if __name__ == '__main__':
    main()
