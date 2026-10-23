#!/usr/bin/env python3
"""在既有Go Docker驗回復HD，必要測試缺項／SKIP／OOM／逾時即拒收。"""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import subprocess
import sys

from verify_fd2_player9_map_hd import check_real_runtime

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    'unit': {'TestModernHDHealNativeSourcesAndGeometry', 'TestModernHDHealFullThemeAndAtomic'},
    'game': {'TestModernHDHealCompleteGameGPU'},
    'source': {'TestModernHDHealNativeSourcesAndGeometry', 'TestModernHDHealLMIAtomic'},
    'regression': {'TestPlayerNativeCommand13RunsCursorTransactionThroughConfirm',
                   'TestDisabledFullPresentationFailsClosedBeforeAnyCommandTransaction',
                   'TestModernHDDeathAssetLoader', 'TestModernHDPlayer9FrozenInputMatchesSnapshot',
                   'TestModernHDPlayer9FrozenInputRejectsDifferentSnapshot',
                   'TestNativeCommand1012PublishesOnlyAtRecoveredBoundaries',
                   'TestNativeCommand1012RollsBackLatePublishFailure'},
}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--kind', choices=REQUIRED, required=True)
    for name in ['catalog', 'pack', 'out']: p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--require-real-runtime', action='store_true')
    p.add_argument('--timeout', type=int, default=1200)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or not 60 <= a.timeout <= 1800 or not a.out.parent.is_dir():
        raise ValueError('Docker內使用有界預算與既有輸出根')
    if any(Path(str(a.out)+suffix).exists() for suffix in ['.gate.json','.log','.memory.json']):
        raise ValueError('不覆寫收據')
    catalog = json.loads(a.catalog.read_text())
    independent = check_real_runtime(a.pack, catalog) if a.require_real_runtime else None
    env = os.environ.copy()
    env.update(FD2_MUTE='1', FD2_TEST_HEAL_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_HEAL_PACK=str(a.pack.resolve()), FD2_TEST_HEAL_OUT=str(a.out)+'-capture')
    for name in ['FD2_HD_HEAL_UNIT','FD2_HD_HEAL_GPU']: env.pop(name, None)
    if a.kind in {'unit','game','source'}: env['FD2_HD_HEAL_UNIT' if a.kind in {'unit','source'} else 'FD2_HD_HEAL_GPU']='1'
    pattern = '^('+'|'.join(sorted(REQUIRED[a.kind]))+')$'
    shell = '''set -euo pipefail
Xvfb :99 -screen 0 1920x1200x24 -nolisten tcp >/tmp/fd2-heal-xvfb.log 2>&1 &
fd2_heal_xvfb=$!
trap 'kill "$fd2_heal_xvfb" 2>/dev/null || true; wait "$fd2_heal_xvfb" 2>/dev/null || true' EXIT
export DISPLAY=:99
"$@"
'''
    command = [sys.executable,str(ROOT/'tools/fd2_go_test_memory_probe.py'),'--out',str(a.out),'--timeout',str(a.timeout),'--',
               'bash','-c',shell,'fd2-heal','go','test','-json','-count=1',f'-timeout={a.timeout-30}s','./cmd/fd2','-run',pattern]
    result = subprocess.run(command,cwd=ROOT/'remake',env=env)
    if result.returncode: raise SystemExit(result.returncode)
    memory = json.loads(Path(str(a.out)+'.memory.json').read_text())
    if memory['returncode'] or memory['timeout'] or memory['after']['memory.events']['oom'] or memory['after']['memory.events']['oom_kill']:
        raise ValueError('失敗／逾時／記憶體終止')
    events = [json.loads(s) for s in Path(str(a.out)+'.log').read_text().splitlines() if s.startswith('{')]
    passed = {e.get('Test') for e in events if e.get('Action')=='pass'}
    if not REQUIRED[a.kind] <= passed or any(e.get('Action') in {'skip','fail'} for e in events):
        raise ValueError('必要測試缺項、SKIP或失敗')
    if a.kind=='unit':
        text=Path(str(a.out)+'.log').read_text()
        points=re.findall(r'heal(13|14|15|16) HD region inside=(\d+) changed=(\d+) outside_mismatch=(\d+)',text)
        if {p[0] for p in points}!={'13','14','15','16'} or any(int(p[1])<100 or int(p[2])<100 or int(p[3]) for p in points):
            raise ValueError('回復HD變色沒有非空覆蓋或改到範圍外')
    if a.kind=='game':
        receipt=json.loads(Path(str(a.out)+'-capture/receipt.json').read_text())
        if receipt.get('status')!='PASS-HEAL1316-HD-RUNTIME-E1' or receipt.get('cases')!=8 or receipt.get('source_gpu_frames')!=17 or receipt.get('f2_round_trips')!=88:
            raise ValueError('完整回復Game.Draw／來源／F2缺項')
    report=dict(status='PASS',kind=a.kind,required=sorted(REQUIRED[a.kind]),catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(),
                pack=str(a.pack.resolve()),independent_runtime_files=independent,log=str(a.out)+'.log',memory=str(a.out)+'.memory.json',
                scope='固定來源與受控回復Game.Draw；不是正常玩家或原版RGB對拍')
    Path(str(a.out)+'.gate.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','kind':a.kind,'required':len(REQUIRED[a.kind])}))


if __name__=='__main__': main()
