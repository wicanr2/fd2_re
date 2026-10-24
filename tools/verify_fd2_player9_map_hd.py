#!/usr/bin/env python3
"""在既有Go容器重跑玩家9的完整主題、來源GPU或Game.Draw閘門。"""
import argparse
import collections
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT=Path(__file__).resolve().parent.parent
PATTERNS={
 'unit':r'TestModernHDPlayer9(NativeSourcesMatchFullVGA|FrozenInputMatchesSnapshot|FrozenInputRejectsDifferentSnapshot|FullThemePreflightAndF2Atomic)$|^TestModernMapSpriteAlphaPreservesReferenceFormatsAndStride$',
 'preflight':r'^TestModernHDPlayer9FullThemePreflightAndF2Atomic$',
 'source':r'^TestModernHDPlayer9MapSourcesGPU$',
 'game':r'^TestModernHDPlayer9MapCompleteGameGPU$',
 'regression':r'^TestModernHDDeath(ScheduleAndAtomicPreflight|AssetLoader|RetainsAuxiliarySeedWithoutSteadyWriter|RuntimeGPU)$|^TestModernHDCommand9ContractAndNativeBlank$',
}


def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()


def check_real_runtime(pack,catalog):
    excluded={'terrain_hd_master','portrait_concept','battlefield_concept','battlefield_tileset_concept','battle_hud_concept'}
    wanted={}
    for asset in catalog['assets']:
        if asset['role'] in excluded:continue
        rows=list(zip(asset.get('files',[]),asset.get('frame_sha256',[])))
        if asset['role']=='story_animation_hd_set':
            rows.extend((n,h) for n,h in zip(asset['source_files'],asset['source_sha256']) if Path(n).suffix=='.json')
        if asset.get('file'):rows.append((asset['file'],asset['sha256']))
        for name,digest in rows:
            if Path(name).name!=name or name in wanted and wanted[name]!=digest:
                raise ValueError('unsafe or conflicting runtime file')
            wanted[name]=digest
    for name,digest in wanted.items():
        p=pack/name
        if p.is_symlink() or not p.is_file() or p.stat().st_nlink!=1 or sha(p)!=digest:
            raise ValueError(f'independent runtime file failed: {name}')
    return len(wanted)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--kind',choices=PATTERNS,required=True)
    p.add_argument('--catalog',type=Path,required=True);p.add_argument('--pack',type=Path,required=True)
    p.add_argument('--out',type=Path,required=True,help='既有父目錄中的全新收據prefix')
    p.add_argument('--capture-out',type=Path)
    p.add_argument('--timeout',type=int,default=900)
    p.add_argument('--require-real-runtime',action='store_true')
    a=p.parse_args()
    if not Path('/.dockerenv').exists():raise SystemExit('只可在維護中的Go Docker image執行')
    if not a.catalog.is_file() or not a.pack.is_dir() or not a.out.parent.is_dir() or not 30<=a.timeout<=1800:
        raise SystemExit('輸入或有界輸出設定無效')
    out=a.out.resolve();capture=(a.capture_out or out.with_name(out.name+'-capture')).resolve()
    if Path(str(out)+'.gate.json').exists():raise SystemExit('不可覆寫既有收據')
    catalog=json.loads(a.catalog.read_text())
    independent=check_real_runtime(a.pack,catalog) if a.require_real_runtime else None
    env=os.environ.copy()
    env.update(FD2_TEST_PLAYER9_MAP_CATALOG=str(a.catalog.resolve()),FD2_TEST_PLAYER9_MAP_PACK=str(a.pack.resolve()),
               FD2_TEST_PLAYER9_MAP_OUT=str(capture),FD2_TEST_MODERN_BATTLE_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_MODERN_BATTLE_PACK=str(a.pack.resolve()),FD2_TEST_HD_DEATH_OUT=str(capture))
    for key in ['FD2_HD_PLAYER9_MAP_GPU','FD2_HD_PLAYER9_MAP_SOURCE_GPU','FD2_TEST_HD_DEATH_GPU']:env.pop(key,None)
    if a.kind=='game':env['FD2_HD_PLAYER9_MAP_GPU']='1'
    if a.kind=='source':env['FD2_HD_PLAYER9_MAP_SOURCE_GPU']='1'
    if a.kind=='regression':env['FD2_TEST_HD_DEATH_GPU']='1'
    shell=r'''set -euo pipefail
Xvfb :99 -screen 0 1920x1200x24 -nolisten tcp >/tmp/fd2-player9-xvfb.log 2>&1 &
fd2_xvfb_pid=$!
trap 'kill "$fd2_xvfb_pid" 2>/dev/null || true; wait "$fd2_xvfb_pid" 2>/dev/null || true' EXIT
export DISPLAY=:99
"$@"
'''
    go=['go','test','-json',f'-timeout={max(20,a.timeout-30)}s','-count=1','./cmd/fd2','-run',PATTERNS[a.kind]]
    command=[sys.executable,str(ROOT/'tools/fd2_go_test_memory_probe.py'),'--out',str(out),'--timeout',str(a.timeout),'--',
             'bash','-c',shell,'fd2-player9-gate',*go]
    result=subprocess.run(command,env=env,cwd=ROOT/'remake')
    if result.returncode:raise SystemExit(result.returncode)
    rows=[json.loads(line) for line in Path(str(out)+'.log').read_text().splitlines() if line.startswith('{')]
    if any(x.get('Action') in {'skip','fail'} for x in rows) or not any(x.get('Action')=='pass' and x.get('Test') for x in rows):
        raise SystemExit('閘門沒有完整執行或含SKIP／失敗')
    summary={'status':'PASS','kind':a.kind,'catalog_sha256':sha(a.catalog),'catalog_assets':len(catalog['assets']),
             'pack':str(a.pack),'independent_runtime_files':independent,'command':go,
             'memory_receipt':str(out)+'.memory.json','log':str(out)+'.log','scope':'RUNTIME-E1；不等於原版parity、正常玩家或人工造型確認。'}
    if a.kind in {'source','game'}:
        report=json.loads((capture/'gpu.json').read_text());summary['gpu']=str(capture/'gpu.json')
        if a.kind=='source' and (report.get('full_gpu_worlds')!=120 or len(report.get('rows',[]))!=120):raise SystemExit('來源GPU案例不完整')
        if a.kind=='game':
            game=[r for r in report.get('rows',[]) if r['kind']=='game']
            counts=collections.Counter((r['hit'],r['phase']) for r in game)
            for hit in [True,False]:
                for phase,n in enumerate([8,27,22,31]):
                    if counts[(hit,phase)]!=n:raise SystemExit('完整演出節拍不完整')
            if sum(r.get('f2_round_trip',False) for r in game)!=16:raise SystemExit('F2往返不完整')
    Path(str(out)+'.gate.json').write_text(json.dumps(summary,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(summary,ensure_ascii=False))


if __name__=='__main__':main()
