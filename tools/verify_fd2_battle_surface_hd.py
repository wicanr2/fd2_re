#!/usr/bin/env python3
"""在既有 Go Docker 驗證 BG_004 候選，拒收 SKIP、缺項、OOM 或逾時。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

from verify_fd2_player9_map_hd import check_real_runtime

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    'unit': {'TestModernBattleSurfaceHDSourceAndAtomic', 'TestModernBattleSurfacePhysicalNativeReconstruction', 'TestModernBattleSurfaceCompleteTheme'},
    'source': {'TestModernBattleSurfaceHDGPU'},
    'legacy-gpu': {'TestModernHDBattleGPUBridge'},
    'game': {'TestModernHDBattleRuntimeGPU'},
    'regression': {'TestNativePhysicalSceneMatchesChapter8NormalOracleSelection', 'TestNativePhysicalSurfaceRetainsOpaqueIndexZero',
                   'TestNativePhysicalSceneMissingBGStopsPlayerAndMode11BeforeSettlement',
                   'TestModernHDBattleAssetsCompleteAndAtomic', 'TestModernHDBattleAssetsLazySetAtomic',
                   'TestModernScrollSurfaceKeepsOpaqueZeroAndTransparentMask', 'TestModernScrollMetadataRejectsBeforePartialSources'},
}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--kind', choices=REQUIRED, required=True)
    p.add_argument('--catalog', type=Path, required=True)
    p.add_argument('--pack', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('--enemy', action='store_true')
    p.add_argument('--require-real-runtime', action='store_true')
    p.add_argument('--timeout', type=int, default=1200)
    a = p.parse_args()
    if not Path('/.dockerenv').exists() or not 60 <= a.timeout <= 1800 or not a.out.parent.is_dir():
        raise SystemExit('Docker 內使用有界時間與既有輸出父目錄')
    for suffix in ['.gate.json','.log','.memory.json']:
        if Path(str(a.out)+suffix).exists(): raise SystemExit('不覆寫收據')
    catalog = json.loads(a.catalog.read_text())
    independent = check_real_runtime(a.pack, catalog) if a.require_real_runtime else None
    env = os.environ.copy()
    env.update(FD2_MUTE='1', FD2_TEST_MODERN_BATTLE_CATALOG=str(a.catalog.resolve()), FD2_TEST_MODERN_BATTLE_PACK=str(a.pack.resolve()))
    for name in ['FD2_HD_BATTLE_LIFECYCLE_GPU','FD2_HD_BATTLE_REAL','FD2_HD_BATTLE_ENEMY_ACTOR','FD2_HD_SURFACE_GPU','FD2_HD_BATTLE_BACKGROUND']:
        env.pop(name,None)
    if a.kind == 'game':
        env.update(FD2_HD_BATTLE_LIFECYCLE_GPU='1',FD2_HD_BATTLE_REAL='1',FD2_HD_BATTLE_BACKGROUND='4',
                   FD2_TEST_MODERN_BATTLE_FIGURE='76' if a.enemy else '0',FD2_TEST_HD_BATTLE_OUT=str(a.out)+'-capture')
        if a.enemy: env['FD2_HD_BATTLE_ENEMY_ACTOR']='1'
    elif a.kind == 'source': env['FD2_HD_SURFACE_GPU']='1'
    elif a.kind == 'legacy-gpu': env['FD2_HD_BATTLE_GPU']='1'
    pattern = '^('+'|'.join(sorted(REQUIRED[a.kind]))+')$'
    shell = '''set -euo pipefail
Xvfb :99 -screen 0 1920x1200x24 -nolisten tcp >/tmp/fd2-surface-xvfb.log 2>&1 &
fd2_surface_xvfb=$!
trap 'kill "$fd2_surface_xvfb" 2>/dev/null || true; wait "$fd2_surface_xvfb" 2>/dev/null || true' EXIT
export DISPLAY=:99
"$@"
'''
    go = ['go','test','-json','-count=1',f'-timeout={a.timeout-30}s','./cmd/fd2','-run',pattern]
    command = [sys.executable,str(ROOT/'tools/fd2_go_test_memory_probe.py'),'--out',str(a.out),'--timeout',str(a.timeout),'--','bash','-c',shell,'fd2-background',*go]
    result = subprocess.run(command,cwd=ROOT/'remake',env=env)
    if result.returncode: raise SystemExit(result.returncode)
    memory = json.loads(Path(str(a.out)+'.memory.json').read_text())
    if memory['returncode'] or memory['timeout'] or memory['after']['memory.events']['oom'] or memory['after']['memory.events']['oom_kill']:
        raise SystemExit('失敗、逾時或 OOM')
    events = [json.loads(s) for s in Path(str(a.out)+'.log').read_text().splitlines() if s.startswith('{')]
    passed = {e.get('Test') for e in events if e.get('Action')=='pass'}
    if not REQUIRED[a.kind] <= passed or any(e.get('Action') in {'skip','fail'} for e in events):
        raise SystemExit('必要測試缺項、SKIP 或失敗')
    if a.kind == 'game':
        receipt = json.loads(Path(str(a.out)+'-capture/receipt.json').read_text())
        if receipt.get('background_selection',{}).get('BaseBG') != 4 or not receipt.get('background_hd_sha256') or receipt['continuations'] != 1:
            raise SystemExit('缺少實際 BG_004 Game.Draw 或一次返回')
    report = dict(status='PASS',kind=a.kind,enemy=a.enemy,required=sorted(REQUIRED[a.kind]),
                  catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(),
                  independent_runtime_files=independent, memory=str(a.out)+'.memory.json',log=str(a.out)+'.log',
                  scope='固定來源／GPU／明示物理戰鬥fixture，非正常玩家或新原版RGB對拍')
    Path(str(a.out)+'.gate.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','kind':a.kind,'required':len(REQUIRED[a.kind])}))


if __name__ == '__main__': main()
