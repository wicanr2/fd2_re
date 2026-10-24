#!/usr/bin/env python3
"""在既有Go Docker嚴格驗證單位轉場HD，拒收缺項、SKIP、OOM及逾時。"""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import subprocess
import sys

from verify_fd2_player9_map_hd import check_real_runtime

ROOT=Path(__file__).resolve().parents[1]
REQUIRED={
    'sky-game':{'TestModernHDSkyKeyGameGPU'},
    'sky-source':{'TestModernHDSkyKeySourceBanks'},
    'sky-unit':{'TestModernHDSkyKeyFullSequenceNativeControl','TestModernHDSkyKeyThemeTransactionRejects','TestModernHDSkyKeyFailureAndLegacy'},
    'sky-native':{'TestNativeCh20SkyKeyPlayerAssetsMatchRecoveredContract','TestNativeCh20SkyKeyAssetValidationRejectsIncompleteANI','TestNativeCh20SkyKeyBeatRejectsEditedPayloadBeforeMutation','TestNativeCh20SkyKeyFailureRestoresPublishedMapTransaction','TestChapterTwentyOneSkyKeyBattleResultReachesTownAndSaveBoundary','TestChapterTwentyOneSkyKeyInsufficientBranchUsesNativeDialogueAndTownSave'},
    'stage-game':{'TestModernHDCh23StageGameGPU'},
    'stage-source':{'TestModernHDCh23StageLoader'},
    'stage-failure':{'TestModernHDCh23LateNativeFailureRestoresBothThemes'},
    'stage-native':{'TestNativeCh23LoopsPreserveSetterTickPaletteAndContinuation','TestNativeCh23FailureRollsBackFirstVisibleDraw','TestNativeCh23RejectsMissingRawFrameBeforeMutation','TestNativeCh23SteadyStageTickAndPostOwner','TestNativeCh23SteadyStageFailureIsAtomic','TestNativeCh23RejectsMissingSeparatedStageBeforeMutation'},
    'stage-unit':{'TestModernHDCh23StageRowsAndTerrainMasks','TestModernHDCh23FigurePalettePreservesAlpha','TestModernHDCh23FullLoopsMatchNativeControl','TestModernHDCh23F2RejectsAndStagesAllOwners','TestModernHDCh23ColdFailureRestoresNativeBoundary','TestModernHDCh23LegacyThemeKeepsNativeStage','TestModernHDCh23StoryPrivateStage'},
    'rowshake-game':{'TestModernHDRowShakeGameGPU'},
    'rowshake-native':{'TestNative24B4DPublishesThirtyAlternatingFramesBeforeContinuation','TestNative24B4DFailsBeforeMutationWithoutNinthTerrainRow','TestNative24B4DStoryPresentationKeepsBattleStateAbsent'},
    'rowshake-unit':{'TestModernHDRowShakeNativeControlAndCadence','TestModernHDRowShakeLayersAndIncomingRow','TestModernHDRowShakeRejectsSourceAndThemeAtomically','TestModernHDRowShakeStoryColdFailureRestoresInitialization','TestModernHDRowShakeStagesAllOwnersBeforePublish','TestModernHDRowShakeRuntimeFailureRestoresBuffers'},
    'indexed-native':{
        'TestResolveNativeIndexedTransitionUsesProvenRelativeCursor',
        'TestResolveNativeIndexedTransitionUsesLoadCHSceneView',
        'TestResolveNativeIndexedTransitionUsesChapter27TwoAxisOffset',
        'TestChapter27IndexedTransitionCallersKeepDistinctOffsets',
        'TestResolveNativeIndexedTransitionRejectsMissingOrUnprovenCursor',
        'TestNativeIndexedTransitionRequiresEveryDrawBeforeContinuation',
        'TestNativeIndexedTransitionStartFailsClosedOnMissingRawAssets',
        'TestNativeIndexedTransitionPreflightsAllNineLUTsBeforePublish',
    },
    'indexed-unit':{
        'TestModernHDIndexedTransitionFrozenFramesAndNativeControl',
        'TestModernHDIndexedTransitionUsesPreviousNativeWork',
        'TestModernHDIndexedTransitionMaskKeepsFullWorkOrigin',
        'TestModernHDIndexedTransitionRejectsBeforePublication',
        'TestModernHDIndexedTransitionStagesAllOwnersAtomically',
        'TestModernHDIndexedTransitionCachedF2Revalidates',
        'TestModernHDIndexedTransitionColdStartRejects',
    },
    'indexed-game':{'TestModernHDIndexedTransitionGameGPU'},
    'native':{
        'TestNativeUnitPresentMutatesOnlyAfterContractAndCompletesAllFrames',
        'TestNativeUnitPresentPreflightFailureLeavesStateAndBuffersUntouched',
        'TestNativeUnitPresentRuntimeFailureRollsBackCoordinateAndBuffers',
        'TestNativeStagingFocusUsesOneStoryViewAfterLoadCH',
        'TestNativeStagingPresentFocusesXYThenMutatesStorySlotAtBridge',
        'TestNativeStagingPresentPreflightFailureStartsNoFocus',
        'TestNativeRelocationUsesSeparateDestinationCursorAndMode6Legality',
    },
    'unit':{
        'TestModernHDUnitPresentCountsUseExplicitWorkOffset',
        'TestModernHDUnitPresentFrozenFramesAndNativeControl',
        'TestModernHDUnitPresentRejectsBeforePublication',
        'TestModernHDUnitPresentStagesPendingLegsAtomically',
        'TestModernHDUnitPresentLegacyEffectAndLinearRowWrap',
    },
    'source':{'TestModernHDUnitPresentActualSources'},
    'game':{'TestModernHDUnitPresentCompleteGameGPU'},
    'story-unit':{
        'TestModernHDStoryMapPrivateComposition',
        'TestModernHDStoryMapAdmission',
        'TestModernHDStoryMapRejectsBeforePublication',
        'TestModernHDStoryMapThemeSwitchAtomic',
    },
    'story-game':{'TestModernHDUnitPresentStoryGameGPU'},
    'motion-unit':{
        'TestModernHDStoryMotionControllerAndIsolation',
        'TestModernHDStoryMotionCameraOverscanAndParallax',
        'TestModernHDStoryMotionF2RejectsWithoutProgress',
        'TestModernHDStoryMotionTerrainRemapsOnce',
        'TestModernHDStoryMotionKeepsIdleNativePlacement',
        'TestModernHDStoryMotionParallaxSubpixels',
    },
    'motion-game':{'TestModernHDStoryMotionGameGPU'},
    'regression':{
        'TestPlayerNativeCommand13RunsCursorTransactionThroughConfirm',
        'TestDisabledFullPresentationFailsClosedBeforeAnyCommandTransaction',
        'TestModernHDDeathAssetLoader',
        'TestModernHDPlayer9FrozenInputMatchesSnapshot',
        'TestModernHDPlayer9FrozenInputRejectsDifferentSnapshot',
        'TestNativeCommand1012PublishesOnlyAtRecoveredBoundaries',
        'TestNativeCommand1012RollsBackLatePublishFailure',
        'TestModernHDHUDUnitSelectionAndHP',
        'TestModernHDHUDUnitAdmissionAndEventShift',
    },
}


REQUIRED['sky-core']=REQUIRED['sky-source'] | REQUIRED['sky-unit'] | REQUIRED['stage-unit'] | REQUIRED['story-unit']

def check_receipt(kind, prefix):
    receipt=json.loads(Path(str(prefix)+'-capture/receipt.json').read_text())
    if kind=='sky-game':
        rows=receipt.get('rows',[])
        if (receipt.get('status')!='PASS-SKY-KEY-HD-GAME' or receipt.get('fixed_clock_unix')!=1000
                or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2 or receipt.get('map_index')!=20
                or receipt.get('f2_round_trips')!=12 or receipt.get('continuations')!=1
                or receipt.get('native_control_unchanged') is not True or len(rows)!=202):
            raise ValueError('天空之鑰完整流程、GPU、F2或控制組缺項')
        expected={0:4,1:68,2:96,3:1,4:1,5:32}
        for phase,count in expected.items():
            selected=[r for r in rows if r.get('phase')==phase]
            if len(selected)!=count:
                raise ValueError('天空之鑰階段缺項')
            if phase in {1,2,5} and [r.get('frame') for r in selected]!=list(range({1:1,2:0,5:69}[phase],{1:69,2:96,5:101}[phase])):
                raise ValueError('天空之鑰畫格順序不同')
            for row in selected:
                if row.get('rng')!=7 or any(not re.fullmatch('[0-9a-f]{64}',row.get(k,'')) for k in ['final_sha256','native_vga_sha256','dac_sha256']):
                    raise ValueError('天空之鑰狀態或圖面缺項')
        pngs=list(Path(str(prefix)+'-capture').glob('*.png'))
        if len(pngs)!=13:raise ValueError('天空之鑰抽樣PNG缺項')
        return
    if kind=='stage-game':
        rows=receipt.get('rows',[]); returns=receipt.get('returns',[])
        if (receipt.get('status')!='PASS-CH23-STAGE-HD-GAME' or receipt.get('fixed_clock_unix')!=1000
                or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2
                or receipt.get('map_index')!=23 or receipt.get('f2_round_trips')!=6
                or receipt.get('continuations')!=2 or receipt.get('native_control_unchanged') is not True
                or len(rows)!=300 or len(returns)!=2):
            raise ValueError('場景完整流程、GPU、F2或控制組缺項')
        for phase,count,continuations in [('initial',240,1),('palette',60,2)]:
            selected=[r for r in rows if r.get('phase')==phase]
            end=[r for r in returns if r.get('phase')==phase]
            if (len(selected)!=count or {r.get('draw') for r in selected}!=set(range(count))
                    or len(end)!=1 or end[0].get('draws')!=count or end[0].get('continuations')!=continuations
                    or not re.fullmatch('[0-9a-f]{64}',end[0].get('final_sha256',''))):
                raise ValueError('240／60次演出或正常繪圖返回缺項')
            for row in selected:
                if (row.get('rng')!=7 or not 0<=row.get('row_shift',-1)<192
                        or row.get('raw_esi')!=(row['draw'] if phase=='palette' else 0)
                        or any(not re.fullmatch('[0-9a-f]{64}',row.get(k,'')) for k in ['final_sha256','native_vga_sha256','dac_sha256'])):
                    raise ValueError('循環列位移、DAC或完整圖面缺項')
        return
    if kind=='rowshake-game':
        cases=receipt.get('cases',[]); rows=receipt.get('rows',[])
        expected={'battle-20':20,'battle-30':30,'story-20':20,'story-30':30}
        if (receipt.get('status')!='PASS-ROW-SHAKE-HD-GAME' or receipt.get('fixed_clock_unix')!=1000
                or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2
                or len(cases)!=4 or {c.get('name') for c in cases}!=set(expected) or len(rows)!=100):
            raise ValueError('交替列位移矩陣或固定條件缺項')
        for c in cases:
            count=expected[c['name']]; selected=[r for r in rows if r.get('case')==c['name']]
            if (c.get('draws')!=count or c.get('f2_round_trips')!=3 or c.get('continuations')!=1
                    or c.get('native_control_unchanged') is not True or len(selected)!=count
                    or {r.get('step') for r in selected}!=set(range(count))
                    or not re.fullmatch('[0-9a-f]{64}',c.get('return_final_sha256',''))):
                raise ValueError('20／30拍、F2或返回缺項')
            for r in selected:
                if r.get('row')!=r['step']%2 or r.get('rng')!=7 or any(not re.fullmatch('[0-9a-f]{64}',r.get(k,'')) for k in ['final_sha256','native_vga_sha256']):
                    raise ValueError('列位移、亂數或完整圖面缺項')
        return
    if kind=='indexed-game':
        cases=receipt.get('cases',[]); rows=receipt.get('rows',[]); alpha=receipt.get('alpha_sources',[])
        if (receipt.get('status')!='PASS-INDEXED-TRANSITION-HD-GAME' or receipt.get('fixed_clock_unix')!=1000
                or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2
                or len(cases)!=4 or {r.get('name') for r in cases}!={'ch21','ch22','ch27','geometry'}
                or len(rows)!=288 or len(alpha)!=3 or {r.get('delta') for r in alpha}!={0,16,62}
                or any(r.get('alphas')!=[0,128,255] or not re.fullmatch('[0-9a-f]{64}',r.get('world_sha256','')) for r in alpha)):
            raise ValueError('徑向／DAC完整流程、固定條件或alpha來源缺項')
        expected_sources={'ch21':'0x245ce','ch22':'0x336e5','ch27':'0x33ce2','geometry':''}
        for case in cases:
            if (case.get('source')!=expected_sources[case['name']] or case.get('draws')!=72
                    or case.get('f2_round_trips')!=8 or case.get('tail_updates')!=31 or case.get('continuations')!=1
                    or case.get('native_control_unchanged') is not True
                    or not re.fullmatch('[0-9a-f]{64}',case.get('return_final_sha256',''))):
                raise ValueError('caller、一次續行、停留、主題往返或HD返回缺項')
            selected=[r for r in rows if r.get('case')==case['name']]
            expected={f'pass/{i}' for i in range(9)}|{f'tail/{i}' for i in range(31)}|{f'palette/{i}' for i in range(32)}
            if len(selected)!=72 or {r.get('key') for r in selected}!=expected:
                raise ValueError('九階、31停留tick或32階DAC缺項')
            final_vga=next(r['native_vga_sha256'] for r in selected if r['key']=='pass/8')
            for row in selected:
                if (row.get('key')!=f"{row.get('phase')}/{row.get('number')}" or row.get('rng')!=7
                        or any(not re.fullmatch('[0-9a-f]{64}',row.get(k,'')) for k in ['final_sha256','native_vga_sha256','dac_sha256'])
                        or row.get('palette_add6')!=(row['number']*2 if row['phase']=='palette' else 0)
                        or row['phase']!='pass' and row['native_vga_sha256']!=final_vga):
                    raise ValueError('原始圖面、GPU或baseline DAC階段不一致')
            if len({r['dac_sha256'] for r in selected if r['phase']=='palette'})!=32:
                raise ValueError('32個實際DAC快照缺項')
        return
    if kind=='motion-game':
        cases=receipt.get('cases',[])
        if (receipt.get('status')!='PASS-STORY-MOTION-HD-GAME' or receipt.get('fixed_clock_unix')!=1000
                or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2
                or receipt.get('parallax_fixture')!='pan: original FDOTHER 16/17 on map-zero carrier; controlled RUNTIME-E1'
                or len(cases)!=8 or {r.get('name') for r in cases}!= {'east','west','north','south','turn-x','turn-y','pan','follow'}
                or any(r.get('draws')!=14 or r.get('continuations')!=1 or r.get('f2_round_trips')!=3
                       or r.get('native_control_unchanged') is not True for r in cases)
                or len(receipt.get('rows',[]))!=112
                or any(not re.fullmatch('[0-9a-f]{64}',r.get('final_sha256','')) for r in receipt['rows'])):
            raise ValueError('故事走位、移鏡、GPU、續行或F2矩陣缺項')
        for case in cases:
            rows=[r for r in receipt['rows'] if r.get('case')==case['name']]
            if len(rows)!=14 or {r.get('step') for r in rows}!=set(range(14)) or any(r.get('units',0)<=0 for r in rows):
                raise ValueError('逐流程14次實際畫布缺項')
        endpoints={'east':(216,120),'west':(72,120),'north':(144,72),'south':(144,192),
                   'turn-x':(216,144),'turn-y':(168,192),'pan':(144,120),'follow':(144,48)}
        for row in receipt['rows']:
            expected=(144,120) if row['step']==0 else endpoints[row['case']] if row['step']==13 else None
            if expected and (row['actor_x'],row['actor_y'])!=expected:
                raise ValueError('控制器起終點與流程契約不同')
        return
    sources=receipt.get('sources',[])
    if kind=='source':
        if (receipt.get('status')!='PASS-UNIT-PRESENT-HD-SOURCES'
                or receipt.get('frames')!=11 or receipt.get('native_frames')!=45 or receipt.get('rejections')!=5
                or len(sources)!=11 or {r.get('index') for r in sources}!=set(range(114,125))
                or any(r.get('varying_native_blocks',0)<=0 for r in sources)):
            raise ValueError('實際11張來源、原生幀或拒收測試缺項')
        return
    cases=receipt.get('cases',[])
    story=kind=='story-game'
    expected_cases={'story-staging','battle-present'} if story else {'command23','staging'}
    expected_status='PASS-UNIT-PRESENT-HD-STORY-GAME' if story else 'PASS-UNIT-PRESENT-HD-GAME'
    if (receipt.get('status')!=expected_status or receipt.get('fixed_clock_unix')!=1000
            or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2
            or len(sources)!=22 or {(r.get('entry'),r.get('placement')) for r in sources}!={(i,p) for i in range(114,125) for p in range(2)}
            or len(cases)!=2 or {r.get('name') for r in cases}!=expected_cases):
        raise ValueError('完整遊戲、固定條件或22個來源繪製缺項')
    rows=receipt.get('rows',[])
    for case in cases:
        name=case['name']; legs=2 if name=='command23' else 1
        f2=20 if name=='command23' else 9 if name=='battle-present' else 10
        selected=[r for r in rows if r.get('case')==name]
        if (case.get('legs')!=legs or case.get('continuations')!=1 or case.get('native_control_unchanged') is not True
                or case.get('f2_round_trips')!=f2 or case.get('draws')!=len(selected)
                or any(r.get('rng')!=7 or not re.fullmatch('[0-9a-f]{64}',r.get('final_sha256','')) for r in selected)):
            raise ValueError('轉場、續行、主題切換或原生控制組缺項')
        if story and not re.fullmatch('[0-9a-f]{64}',case.get('return_final_sha256','')):
            raise ValueError('缺少回到HD世界的完整GPU畫布')
        keys={r['key'] for r in selected}
        if name=='command23' and not {f'palette/{i}' for i in range(8)}<=keys:
            raise ValueError('命令23前導閃光缺項')
        for leg in range(legs):
            found={int(m[2]):m[1] for key in keys if (m:=re.fullmatch(r'leg'+str(leg)+r'/(intro|contract|bridge|release)/(\d+)',key))}
            count=len(found)
            if count not in {45,51} or set(found)!=set(range(count)):
                raise ValueError('原生45／51幀轉場缺項')
            for index,phase in found.items():
                expected='intro' if index<11 else 'contract' if index<17 else 'bridge' if index<count-10 else 'release'
                if phase!=expected:
                    raise ValueError('轉場階段或橋接列數不同')


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--kind',choices=REQUIRED,required=True)
    for name in ['catalog','pack','out']:
        p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--timeout',type=int,default=900)
    p.add_argument('--check-existing',action='store_true',help='以目前嚴格檢查器另存稽核，不重跑或覆寫既有收據')
    p.add_argument('--require-real-runtime',action='store_true')
    a=p.parse_args()
    if not Path('/.dockerenv').exists() or not 60<=a.timeout<=1800 or not a.out.parent.is_dir() or not a.catalog.is_file() or not a.pack.is_dir():
        raise ValueError('Docker內使用有效輸入、既有工作根與有界時間')
    for suffix in (['.audit.json'] if a.check_existing else ['.gate.json','.memory.json','.log']):
        if Path(str(a.out)+suffix).exists():
            raise ValueError('不可覆寫收據')
    independent=check_real_runtime(a.pack,json.loads(a.catalog.read_text())) if a.require_real_runtime else None
    env=os.environ.copy()
    for name in ['FD2_HD_UNIT_PRESENT_SOURCE','FD2_HD_UNIT_PRESENT_GPU','FD2_HD_UNIT_PRESENT_STORY_GPU','FD2_HD_STORY_MOTION_GPU','FD2_HD_INDEXED_TRANSITION_GPU','FD2_HD_ROW_SHAKE_GPU','FD2_HD_CH23_STAGE_GPU','FD2_HD_SKY_KEY_GPU']:
        env.pop(name,None)
    env.update(FD2_MUTE='1',FD2_TEST_UNIT_PRESENT_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_UNIT_PRESENT_PACK=str(a.pack.resolve()),FD2_TEST_UNIT_PRESENT_OUT=str(a.out)+'-capture')
    if a.kind=='source':env['FD2_HD_UNIT_PRESENT_SOURCE']='1'
    if a.kind=='sky-game':env['FD2_HD_SKY_KEY_GPU']='1'
    if a.kind=='stage-game':env['FD2_HD_CH23_STAGE_GPU']='1'
    if a.kind=='game':env['FD2_HD_UNIT_PRESENT_GPU']='1'
    if a.kind=='story-game':env['FD2_HD_UNIT_PRESENT_STORY_GPU']='1'
    if a.kind=='motion-game':env['FD2_HD_STORY_MOTION_GPU']='1'
    if a.kind=='rowshake-game':env['FD2_HD_ROW_SHAKE_GPU']='1'
    if a.kind=='indexed-game':env['FD2_HD_INDEXED_TRANSITION_GPU']='1'
    pattern='^('+'|'.join(sorted(REQUIRED[a.kind]))+')$'
    shell='''set -euo pipefail
Xvfb :99 -screen 0 1920x1200x24 -nolisten tcp >/tmp/fd2-unit-present-xvfb.log 2>&1 &
fd2_xvfb_pid=$!
trap 'kill "$fd2_xvfb_pid" 2>/dev/null || true; wait "$fd2_xvfb_pid" 2>/dev/null || true' EXIT
export DISPLAY=:99
"$@"
'''
    command=[sys.executable,str(ROOT/'tools/fd2_go_test_memory_probe.py'),'--out',str(a.out),'--timeout',str(a.timeout),'--',
             'bash','-c',shell,'fd2-unit-present','go','test','-json','-count=1',f'-timeout={a.timeout-30}s','./cmd/fd2','-run',pattern]
    if not a.check_existing:
        result=subprocess.run(command,cwd=ROOT/'remake',env=env)
        if result.returncode:
            raise SystemExit(result.returncode)
    else:
        old=json.loads(Path(str(a.out)+'.gate.json').read_text())
        if old.get('status')!='PASS' or old.get('kind')!=a.kind or old.get('catalog_sha256')!=hashlib.sha256(a.catalog.read_bytes()).hexdigest():
            raise ValueError('原閘門身份或清冊不同')
    memory=json.loads(Path(str(a.out)+'.memory.json').read_text())
    if memory['returncode'] or memory['timeout'] or memory['after']['memory.events']['oom'] or memory['after']['memory.events']['oom_kill']:
        raise ValueError('失敗、逾時或OOM')
    events=[json.loads(s) for s in Path(str(a.out)+'.log').read_text().splitlines() if s.startswith('{')]
    if not REQUIRED[a.kind]<={e.get('Test') for e in events if e.get('Action')=='pass'} or any(e.get('Action') in {'skip','fail'} for e in events):
        raise ValueError('必要測試缺項、SKIP或失敗')
    if a.kind in {'source','game','story-game','motion-game','indexed-game','rowshake-game','stage-game','sky-game'}:
        check_receipt(a.kind,a.out)
    report=dict(status='PASS',kind=a.kind,required=sorted(REQUIRED[a.kind]),catalog=str(a.catalog.resolve()),
                catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(),pack=str(a.pack.resolve()),
                log=str(a.out)+'.log',memory=str(a.out)+'.memory.json',scope='受控轉場與Game.Draw；不新增正常玩家或原版RGB對拍')
    report['independent_runtime_files']=independent
    report['verifier_sha256']=hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    Path(str(a.out)+('.audit.json' if a.check_existing else '.gate.json')).write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','kind':a.kind,'required':len(REQUIRED[a.kind])}))


if __name__=='__main__':
    main()
