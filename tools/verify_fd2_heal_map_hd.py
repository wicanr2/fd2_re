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


def game_receipt(prefix, start, end):
    receipt=json.loads(Path(str(prefix)+'-capture/receipt.json').read_text())
    rows=receipt.get('completed_cases',[])
    if (receipt.get('status')!='PASS-HEAL1316-HD-RUNTIME-E1' or receipt.get('cases')!=end-start
            or receipt.get('case_start')!=start or receipt.get('case_end')!=end
            or receipt.get('source_gpu_frames')!=17 or receipt.get('f2_round_trips')!=11*(end-start)
            or len(rows)!=end-start or {r.get('index') for r in rows}!=set(range(start,end))
            or any(r.get('command')!=13+r['index']//2 or r.get('multi')!=(r['index']%2==1)
                   or r.get('transactions')!=1 or r.get('continuations')!=1 or r.get('f2_round_trips')!=11 for r in rows)):
        raise ValueError('完整回復Game.Draw／來源／F2／批次身份缺項')
    return receipt


def merge_game_parts(a):
    digest=hashlib.sha256(a.catalog.read_bytes()).hexdigest()
    receipts=[]
    for prefix in a.merge_game_parts:
        gate=json.loads(Path(str(prefix)+'.gate.json').read_text())
        if gate.get('status')!='PASS' or gate.get('kind')!='game' or gate.get('catalog_sha256')!=digest:
            raise ValueError('批次閘門或清冊不一致')
        start,end=gate.get('case_start'),gate.get('case_end')
        if not isinstance(start,int) or not isinstance(end,int) or not 0<=start<end<=8:
            raise ValueError('批次範圍無效')
        memory=json.loads(Path(str(prefix)+'.memory.json').read_text())
        events=[json.loads(s) for s in Path(str(prefix)+'.log').read_text().splitlines() if s.startswith('{')]
        if (memory['returncode'] or memory['timeout'] or memory['after']['memory.events']['oom'] or memory['after']['memory.events']['oom_kill']
                or not REQUIRED['game']<={e.get('Test') for e in events if e.get('Action')=='pass'}
                or any(e.get('Action') in {'skip','fail'} for e in events)):
            raise ValueError('批次缺少通過測試或記憶體證據')
        receipts.append(game_receipt(prefix,start,end))
    rows=[r for d in receipts for r in d['completed_cases']]
    if len(rows)!=8 or {r['index'] for r in rows}!=set(range(8)):
        raise ValueError('八條流程缺項或重複')
    result=dict(status='PASS-HEAL1316-HD-ALL-BATCHES',cases=8,f2_round_trips=sum(r['f2_round_trips'] for r in rows),
                source_gpu_frames_per_part=17,parts=[str(p) for p in a.merge_game_parts],completed_cases=sorted(rows,key=lambda r:r['index']),
                catalog_sha256=digest,scope='全部八條受控Game.Draw；不新增正常玩家或原版RGB對拍')
    Path(str(a.out)+'.gate.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k not in ['completed_cases','parts']}))


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--kind', choices=REQUIRED, required=True)
    for name in ['catalog', 'pack', 'out']: p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--require-real-runtime', action='store_true')
    p.add_argument('--timeout', type=int, default=1200)
    p.add_argument('--case-start',type=int,choices=range(8),default=0)
    p.add_argument('--case-limit',type=int,choices=range(1,9),default=8)
    p.add_argument('--merge-game-parts',type=Path,nargs='+',help='核對各批次無SKIP／OOM、同清冊及全部八條身份，不重跑遊戲')
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or not 60 <= a.timeout <= 1800 or not a.out.parent.is_dir():
        raise ValueError('Docker內使用有界預算與既有輸出根')
    if any(Path(str(a.out)+suffix).exists() for suffix in ['.gate.json','.log','.memory.json']):
        raise ValueError('不覆寫收據')
    if a.merge_game_parts:
        if a.kind!='game':raise ValueError('只可合併完整Game.Draw批次')
        merge_game_parts(a)
        return
    catalog = json.loads(a.catalog.read_text())
    independent = check_real_runtime(a.pack, catalog) if a.require_real_runtime else None
    env = os.environ.copy()
    env.update(FD2_MUTE='1', FD2_TEST_HEAL_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_HEAL_PACK=str(a.pack.resolve()), FD2_TEST_HEAL_OUT=str(a.out)+'-capture')
    for name in ['FD2_HD_HEAL_UNIT','FD2_HD_HEAL_GPU','FD2_HD_HEAL_START','FD2_HD_HEAL_LIMIT']: env.pop(name, None)
    if a.kind in {'unit','game','source'}: env['FD2_HD_HEAL_UNIT' if a.kind in {'unit','source'} else 'FD2_HD_HEAL_GPU']='1'
    if a.kind=='game':env.update(FD2_HD_HEAL_START=str(a.case_start),FD2_HD_HEAL_LIMIT=str(a.case_limit))
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
        game_receipt(a.out,a.case_start,min(8,a.case_start+a.case_limit))
    report=dict(status='PASS',kind=a.kind,required=sorted(REQUIRED[a.kind]),catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(),
                pack=str(a.pack.resolve()),independent_runtime_files=independent,log=str(a.out)+'.log',memory=str(a.out)+'.memory.json',
                scope='固定來源與受控回復Game.Draw；不是正常玩家或原版RGB對拍')
    if a.kind=='game':report.update(case_start=a.case_start,case_end=min(8,a.case_start+a.case_limit))
    Path(str(a.out)+'.gate.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','kind':a.kind,'required':len(REQUIRED[a.kind])}))


if __name__=='__main__': main()
