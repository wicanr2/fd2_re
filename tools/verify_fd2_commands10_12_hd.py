#!/usr/bin/env python3
"""在既有Go容器重跑指令10～12完整HD；拒收略過、失敗及不完整演出。"""
import argparse
import collections
import hashlib
import json
import os
import re
from pathlib import Path
import subprocess
import sys

from verify_fd2_player9_map_hd import check_real_runtime

ROOT = Path(__file__).resolve().parent.parent
REQUIRED_TESTS = {
    'unit': {'TestModernHDCommand1012SamplingIndependentCoordinates', 'TestModernHDCommand1012FullThemeAndAtomicF2'},
    'game': {'TestModernHDCommand1012CompleteGameGPU'},
    'regression': {
        'TestNativeCommand1012PublishesOnlyAtRecoveredBoundaries', 'TestNativeCommand1012RollsBackLatePublishFailure',
        'TestNativeCommand1012PreludeCannotDisappearSilently', 'TestExecuteNativeAICommand10UsesOriginalIndexedOwner',
        'TestNative2189APrecomputesTenFramesAndContinuesAfterPresentation', 'TestNative2189AMissingLUTFailsBeforeMutation',
        'TestNative2189AAcceptsRecoveredCommand11CallShape', 'TestNative2189ARejectsUnrecoveredCallShape',
        'TestModernHDPlayer9NativeSourcesMatchFullVGA', 'TestModernHDPlayer9FrozenInputMatchesSnapshot',
        'TestModernHDPlayer9FrozenInputRejectsDifferentSnapshot', 'TestModernHDDeathScheduleAndAtomicPreflight',
        'TestModernHDDeathAssetLoader', 'TestModernHDDeathRetainsAuxiliarySeedWithoutSteadyWriter',
        'TestModernMapSpriteAlphaPreservesReferenceFormatsAndStride',
    },
}


def validate_go_outputs(out, kind):
    memory = json.loads(Path(str(out)+'.memory.json').read_text())
    if memory['returncode'] != 0 or memory['timeout'] or memory['after']['memory.events']['oom'] or memory['after']['memory.events']['oom_kill']:
        raise ValueError('執行失敗、逾時或記憶體耗盡')
    text = Path(str(out)+'.log').read_text()
    events = [json.loads(s) for s in text.splitlines() if s.startswith('{')]
    passed = {e.get('Test') for e in events if e.get('Action') == 'pass'}
    if any(e.get('Action') in {'skip', 'fail'} for e in events) or not REQUIRED_TESTS[kind] <= passed:
        raise ValueError('必要測試缺項、SKIP或失敗')
    if kind == 'unit':
        points = re.findall(r'command(11|12) independent radial HD points inside=(\d+) changed=(\d+)', text)
        if {p[0] for p in points} != {'11', '12'} or any(int(p[1]) < 100 or int(p[2]) < 100 for p in points):
            raise ValueError('HD徑向變色沒有非空覆蓋')
    return sorted(REQUIRED_TESTS[kind])


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--kind', choices=['unit', 'game', 'regression'], required=True)
    p.add_argument('--id', type=int, choices=[10, 11, 12], default=10)
    p.add_argument('--catalog', type=Path, required=True)
    p.add_argument('--pack', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('--timeout', type=int, default=900)
    p.add_argument('--require-real-runtime', action='store_true')
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or not 30 <= a.timeout <= 1800:
        raise SystemExit('只在既有Go Docker與有界時間內執行')
    if not a.catalog.is_file() or not a.pack.is_dir() or not a.out.parent.is_dir():
        raise SystemExit('明示完整清冊、素材與既有輸出父目錄')
    out = a.out.resolve()
    if Path(str(out)+'.gate.json').exists():
        raise SystemExit('不覆寫收據')
    catalog = json.loads(a.catalog.read_text())
    independent = check_real_runtime(a.pack, catalog) if a.require_real_runtime else None
    capture = out.with_name(out.name+'-capture')
    env = os.environ.copy()
    env.update(FD2_TEST_COMMAND1012_CATALOG=str(a.catalog.resolve()), FD2_TEST_COMMAND1012_PACK=str(a.pack.resolve()),
               FD2_TEST_COMMAND1012_ID=str(a.id), FD2_TEST_COMMAND1012_OUT=str(capture))
    env.pop('FD2_HD_COMMAND1012_GPU', None)
    env.pop('FD2_HD_COMMAND1012_UNIT', None)
    if a.kind == 'game':
        env['FD2_HD_COMMAND1012_GPU'] = '1'
        pattern = '^TestModernHDCommand1012CompleteGameGPU$'
    elif a.kind == 'unit':
        env['FD2_HD_COMMAND1012_UNIT'] = '1'
        pattern = '^TestModernHDCommand1012(SamplingIndependentCoordinates|FullThemeAndAtomicF2)$'
    else:
        pattern = ('^(TestNativeCommand1012.*|TestNative2189A.*|TestExecuteNativeAICommand10UsesOriginalIndexedOwner|'
                   'TestModernHDPlayer9(NativeSourcesMatchFullVGA|FrozenInputMatchesSnapshot|FrozenInputRejectsDifferentSnapshot)|'
                   'TestModernHDDeath(ScheduleAndAtomicPreflight|AssetLoader|RetainsAuxiliarySeedWithoutSteadyWriter)|'
                   'TestModernMapSpriteAlphaPreservesReferenceFormatsAndStride)$')
    shell = '''set -euo pipefail
Xvfb :99 -screen 0 1920x1200x24 -nolisten tcp >/tmp/fd2-command1012-xvfb.log 2>&1 &
fd2_xvfb_pid=$!
trap 'kill "$fd2_xvfb_pid" 2>/dev/null || true; wait "$fd2_xvfb_pid" 2>/dev/null || true' EXIT
export DISPLAY=:99
"$@"
'''
    go = ['go', 'test', '-json', '-count=1', f'-timeout={a.timeout-30}s', './cmd/fd2', '-run', pattern]
    cmd = [sys.executable, str(ROOT/'tools/fd2_go_test_memory_probe.py'), '--out', str(out), '--timeout', str(a.timeout),
           '--', 'bash', '-c', shell, 'fd2-command1012-hd', *go]
    result = subprocess.run(cmd, cwd=ROOT/'remake', env=env)
    if result.returncode:
        raise SystemExit(result.returncode)
    required = validate_go_outputs(out, a.kind)
    report = {'status': 'PASS', 'kind': a.kind, 'command': a.id, 'catalog_sha256': hashlib.sha256(a.catalog.read_bytes()).hexdigest(),
              'independent_runtime_files': independent, 'pack': str(a.pack), 'memory': str(out)+'.memory.json', 'log': str(out)+'.log',
              'required_tests_passed': required, 'scope': 'RUNTIME-E1；非正常玩家、原版parity或集中人工造型確認。'}
    if a.kind == 'game':
        gpu = json.loads((capture/'gpu.json').read_text())
        counts = collections.Counter((r['target_count'], r['hit'], r['phase']) for r in gpu['rows'])
        for targets, hit in [(1, True), (1, False), (2, True)]:
            for phase, n in [(1, 60), (2, 22), (3, 31)]:
                if counts[(targets, hit, phase)] != n:
                    raise SystemExit('主／結果／hold節拍不完整')
            if counts[(targets, hit, 0)] != (0 if a.id == 10 else 11):
                raise SystemExit('前導十輪及restore不完整')
        if sum(r.get('f2_round_trip', False) for r in gpu['rows']) != (24 if a.id == 10 else 33):
            raise SystemExit('F2覆蓋不完整')
        report.update(full_hd_draws=len(gpu['rows']), gpu=str(capture/'gpu.json'))
    Path(str(out)+'.gate.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
