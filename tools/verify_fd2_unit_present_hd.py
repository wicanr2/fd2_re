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
    'commands1012-stage-core':{'TestNativeCommandPreludeStageUsesSingleSnapshot','TestModernHDCommand1012StageIsolationAndReject','TestModernHDCommand1012StageColdReject','TestModernHDCommandPreludeStageMarker','TestModernHDCommand1012StageAIEntryControl'},
    'commands1012-stage-ai':{'TestModernHDCommand1012StageAIEntryControl'},
    'commands1012-stage-game':{'TestModernHDCommand1012StageGameGPU'},
    'commands1012-stage-unit':{'TestNativeCommandPreludeStageUsesSingleSnapshot','TestModernHDCommand1012StageIsolationAndReject','TestModernHDCommand1012StageColdReject','TestModernHDCommandPreludeStageMarker'},
    'heal-stage-game':{'TestModernHDHealStageGameGPU'},
    'heal-stage-unit':{'TestModernHDHealStageRemapMatchesNativeMarker','TestModernHDHealStageSourceIsolationAndReject','TestNativeHealStageLateFailureRestoresTransaction'},
    'heal-stage-source':{'TestNativeHealStageFrontUsesOriginalSnapshot'},
    'player9-stage-game':{'TestModernHDPlayer9StageGameGPU'},
    'player9-stage-unit':{'TestModernHDPlayer9StageFreezeUsesSeed','TestModernHDPlayer9ActualStage','TestModernHDPlayer9StageRejectsEditedSource'},
    'item-miss':{'TestNativeMissDescriptorConsumers'},
    'item-stage-source':{'TestModernHDCompoundStageRemapStartsFromSeed'},
    'item-game':{'TestModernHDItemMapGameGPU'},
    'item-native':{'TestExecuteNativeAIItemRestoreUsesOriginalIndexedTail','TestExecuteNativeAIItemDamageUsesOriginalIndexed1CD17Tail','TestPlayerDamageItemsUseOriginalIndexedTails','TestExecuteNativeAIItemType21UsesOriginalIndexed1CAC7Tail','TestNativeAIItemPresentationLateCancelRollsBack','TestNativeAIItemDamagePresentationMissingIndexedContextIsAtomic','TestNativeAIItemPresentationRosterChangeCancelsWithoutPanic'},
    'item-unit':{'TestModernHDItemMapNativeControl','TestModernHDPlayerRestoreKeepsNativeRoute','TestModernHDItemMapRejectsAndF2','TestModernHDCompoundFrozenStage','TestModernHDCompoundActualStage','TestModernHDCompoundStageRemapStartsFromSeed','TestModernHDItemMapFailureRestoresTransaction','TestModernHDItemMapMultipleTargets','TestModernHDItemMapPalettePreservesAlpha'},
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
    if kind=='commands1012-stage-game':
        d=receipt; ids=d.get('case_ids',[]); cases=d.get('cases',[]); rows=d.get('rows',[])
        if d.get('status')!='PASS-COMMAND1012-STAGE-GAME' or d.get('map_index')!=23 or d.get('seed')!=7 or d.get('clock_unix')!=1000 or d.get('rgba_tolerance')!=2 or ids not in [list(range(9)),[0,8]] or [c.get('case') for c in cases]!=ids:
            raise ValueError('commands1012 stage identity or cases missing')
        expected_rows=0;expected_pngs=0;expected_f2=0
        for c in cases:
            ident=c['case']; prelude=ident//3!=0; count=124 if prelude else 113; f2=11 if prelude else 8
            selected=[r for r in rows if r.get('case')==ident]
            if c.get('command')!=10+ident//3 or c.get('variant')!=ident%3 or c.get('target_count')!=(2 if ident%3==2 else 1) or c.get('draws')!=count or c.get('f2_round_trips')!=f2 or c.get('continuations')!=1 or c.get('native_control_unchanged') is not True or len(selected)!=count:
                raise ValueError('commands1012 stage branches or final state missing')
            for phase,want in [('prelude',list(range(11)) if prelude else []),('main',list(range(60))),('result',list(range(22))),('hold',list(range(30,-1,-1)))]:
                if [r['frame'] for r in selected if r['phase']==phase]!=want:raise ValueError('commands1012 stage phase coverage differs')
            if sum(r.get('f2_round_trip') is True for r in selected)!=f2:raise ValueError('commands1012 stage F2 rows differ')
            expected_rows+=count;expected_pngs+=8 if prelude else 5;expected_f2+=f2
        if len(rows)!=expected_rows or len(list(Path(str(prefix)+'-capture').glob('*.png')))!=expected_pngs:raise ValueError('commands1012 stage rows or captures missing')
        return dict(case_ids=ids,frames=expected_rows,f2_round_trips=expected_f2,pngs=expected_pngs)
    if kind=='heal-stage-game':
        d=json.loads(Path(str(prefix)+'-capture/receipt.json').read_text())
        ids=d.get('case_ids',[]);cases=d.get('cases',[]);rows=d.get('rows',[])
        if d.get('status')!='PASS-HEAL-STAGE-HD-GAME' or d.get('map_index')!=23 or d.get('seed')!=7 or d.get('clock_unix')!=1000 or d.get('rgba_tolerance')!=2 or ids not in [list(range(8)),[0,7]] or len(cases)!=len(ids) or len(rows)!=59*len(ids):
            raise ValueError('heal stage identity or full frame coverage missing')
        if [c.get('case') for c in cases]!=ids or any(c.get('command')!=13+c['case']//2 or c.get('multi')!=(c['case']%2==1) or c.get('frames')!=59 or c.get('f2_round_trips')!=11 or c.get('transactions')!=1 or c.get('continuations')!=1 or c.get('native_control_unchanged') is not True for c in cases):
            raise ValueError('heal stage final state or branches missing')
        for case in ids:
            selected=[r for r in rows if r.get('case')==case]
            if any(r.get('mid_hold_native_verified') is not True for r in selected if r['phase']==1):raise ValueError('heal native middle hold not verified')
            if len(selected)!=59 or {p:len([r for r in selected if r['phase']==p]) for p in range(7)}!={0:16,1:1,2:1,3:7,4:11,5:22,6:1} or any(r['transactions']!=(1 if r['phase'] in [5,6] else 0) for r in selected):
                raise ValueError('heal stage phase or transaction boundary missing')
        if len(list(Path(str(prefix)+'-capture').glob('*.png')))!=9*len(ids):raise ValueError('heal stage captures missing')
        return dict(case_ids=ids,frames=59*len(ids),f2_round_trips=11*len(ids),pngs=9*len(ids))
    if kind=='player9-stage-game':
        d=json.loads(Path(str(prefix)+'-capture/receipt.json').read_text())
        cases=d.get('cases',[])
        if d.get('status')!='PASS-PLAYER9-STAGE-HD-GAME' or d.get('map_index')!=23 or d.get('seed')!=7 or d.get('clock_unix')!=1000 or d.get('rgba_tolerance')!=2 or len(cases)!=2 or len(d.get('rows',[]))!=116:
            raise ValueError('player9 stage identity or complete frames missing')
        if {c.get('hit') for c in cases}!={True,False} or any(c.get('frames')!=58 or c.get('f2_round_trips')!=8 or c.get('continuations')!=1 or c.get('native_control_unchanged') is not True for c in cases) or len(list(Path(str(prefix)+'-capture').glob('*.png')))!=18:
            raise ValueError('player9 stage F2, return or captures missing')
        return dict(frames=116,cases=2,f2_round_trips=16,pngs=18)
    if kind=='item-game':
        cases=receipt.get('cases',[]); rows=receipt.get('rows',[])
        names={f'{name}-player-{player}' for name in ['restore5','restore13','damage20','damage20-command0','damage24','type21-command1','type21-command6','type21-command7'] for player in ['false','true'] if not (name.startswith('restore') and player=='true')}
        if (receipt.get('status')!='PASS-ITEM-MAP-HD-GAME' or receipt.get('fixed_clock_unix')!=1000
                or receipt.get('native_seed')!=7 or receipt.get('rgba_tolerance')!=2
                or receipt.get('map_index')!=0 or receipt.get('initial_actor_tile')!=[4,3]
                or receipt.get('initial_target_tile')!=[5,3] or receipt.get('camera_tile')!=[0,0]
                or len(cases)!=14 or {r.get('case') for r in cases}!=names or len(rows)!=556):
            raise ValueError('物品玩家／AI完整流程或固定條件缺項')
        for c in cases:
            count=38 if c['case'].startswith('type21-command1-') else 40
            selected=[r for r in rows if r.get('case')==c['case']]
            if (c.get('frames')!=count or c.get('updates',0)<=count or c.get('f2_round_trips')!=3
                    or c.get('continuations')!=1 or c.get('native_control_unchanged') is not True
                    or len(selected)!=count or {r.get('frame') for r in selected}!=set(range(count))
                    or not re.fullmatch('[0-9a-f]{64}',c.get('return_final_sha256',''))):
                raise ValueError('物品拍數、停留、主題往返、續行或返回畫布缺項')
            if any(any(not re.fullmatch('[0-9a-f]{64}',r.get(k,'')) for k in ['final_sha256','native_vga_sha256']) for r in selected):
                raise ValueError('物品逐幀收據缺項')
            source6={l['Entry'] for r in selected for l in (r.get('sources') or []) if l['Resource']==6}
            expected=(set(range(57,64)) if c['case'].startswith('restore') else
                      set(range(64,76)) if c['case'].startswith(('type21-command6-','type21-command7-')) else
                      set(range(49,57)) | ({74,75} if c['case'].startswith('type21-') else set()))
            if source6!=expected:
                raise ValueError('物品逐流程的光效／切換來源缺項')
            publish=16 if c['case'].startswith('type21-command1-') else 18
            if any(r.get('published')!=(r['frame']>=publish) for r in selected):
                raise ValueError('物品結算時點與逐幀收據不同')
        if len(list(Path(str(prefix)+'-capture').glob('*.png')))!=50:
            raise ValueError('物品三階段抽樣圖缺項')
        digits={l['Entry'] for r in rows for l in (r.get('sources') or []) if l['Resource']==5}
        if not {116,117,118}<=digits or digits & {74,75,76}:
            raise ValueError('未命中字樣的實際繪圖來源缺項')
        return dict(source6_indices=list(range(49,76)),miss_indices=[116,117,118],published_boundaries=[16,18],frames=556)
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
    p.add_argument('--heal-stage-sample',action='store_true')
    p.add_argument('--commands1012-stage-sample',action='store_true')
    p.add_argument('--check-existing',action='store_true',help='以目前嚴格檢查器另存稽核，不重跑或覆寫既有收據')
    p.add_argument('--require-real-runtime',action='store_true')
    p.add_argument('--compare-with',type=Path,help='僅item-game既有收據稽核：比較另一個輸出前綴的完整收據與50張PNG')
    a=p.parse_args()
    if a.heal_stage_sample and a.kind!='heal-stage-game':raise ValueError('heal stage sample only')
    if a.commands1012_stage_sample and a.kind!='commands1012-stage-game':raise ValueError('commands1012 stage sample only')
    if a.compare_with and (a.kind!='item-game' or not a.check_existing):
        raise ValueError('--compare-with只用於item-game --check-existing')
    if not Path('/.dockerenv').exists() or not 60<=a.timeout<=1800 or not a.out.parent.is_dir() or not a.catalog.is_file() or not a.pack.is_dir():
        raise ValueError('Docker內使用有效輸入、既有工作根與有界時間')
    for suffix in (['.audit.json'] if a.check_existing else ['.gate.json','.memory.json','.log']):
        if Path(str(a.out)+suffix).exists():
            raise ValueError('不可覆寫收據')
    independent=check_real_runtime(a.pack,json.loads(a.catalog.read_text())) if a.require_real_runtime else None
    env=os.environ.copy()
    for name in ['FD2_HD_COMMAND1012_STAGE_GPU','FD2_HD_COMMAND1012_STAGE_CASES']:
        env.pop(name,None)
    for name in ['FD2_HD_UNIT_PRESENT_SOURCE','FD2_HD_UNIT_PRESENT_GPU','FD2_HD_UNIT_PRESENT_STORY_GPU','FD2_HD_STORY_MOTION_GPU','FD2_HD_INDEXED_TRANSITION_GPU','FD2_HD_ROW_SHAKE_GPU','FD2_HD_CH23_STAGE_GPU','FD2_HD_SKY_KEY_GPU','FD2_HD_ITEM_MAP_GPU','FD2_HD_ITEM_MAP_PROBE','FD2_HD_PLAYER9_STAGE_GPU','FD2_HD_HEAL_STAGE_GPU','FD2_HD_HEAL_STAGE_CASES']:
        env.pop(name,None)
    env.update(FD2_MUTE='1',FD2_TEST_UNIT_PRESENT_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_UNIT_PRESENT_PACK=str(a.pack.resolve()),FD2_TEST_UNIT_PRESENT_OUT=str(a.out)+'-capture')
    if a.kind=='source':env['FD2_HD_UNIT_PRESENT_SOURCE']='1'
    if a.kind=='item-game':env['FD2_HD_ITEM_MAP_GPU']='1'
    if a.kind=='player9-stage-game':env['FD2_HD_PLAYER9_STAGE_GPU']='1'
    if a.kind=='commands1012-stage-game':
        env['FD2_HD_COMMAND1012_STAGE_GPU']='1'
        if a.commands1012_stage_sample:env['FD2_HD_COMMAND1012_STAGE_CASES']='0,8'
    if a.kind=='heal-stage-game':
        env['FD2_HD_HEAL_STAGE_GPU']='1'
        if a.heal_stage_sample:env['FD2_HD_HEAL_STAGE_CASES']='0,7'
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
    receipt_evidence=None
    if a.kind in {'source','game','story-game','motion-game','indexed-game','rowshake-game','stage-game','sky-game','item-game','player9-stage-game','heal-stage-game','commands1012-stage-game'}:
        receipt_evidence=check_receipt(a.kind,a.out)
        if a.kind=='commands1012-stage-game' and receipt_evidence['case_ids']!=([0,8] if a.commands1012_stage_sample else list(range(9))):raise ValueError('commands1012 stage sample differs from request')
        if a.kind=='heal-stage-game' and receipt_evidence['case_ids']!=([0,7] if a.heal_stage_sample else list(range(8))):raise ValueError('heal stage sample differs from request')
    report=dict(status='PASS',kind=a.kind,required=sorted(REQUIRED[a.kind]),catalog=str(a.catalog.resolve()),
                catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(),pack=str(a.pack.resolve()),
                log=str(a.out)+'.log',memory=str(a.out)+'.memory.json',scope='受控轉場與Game.Draw；不新增正常玩家或原版RGB對拍')
    report['independent_runtime_files']=independent
    if receipt_evidence is not None:report['receipt_evidence']=receipt_evidence
    if a.compare_with:
        other=json.loads(Path(str(a.compare_with)+'.gate.json').read_text())
        if other.get('status')!='PASS' or other.get('kind')!=a.kind or other.get('catalog_sha256')!=report['catalog_sha256']:
            raise ValueError('另一包的閘門或清冊不同')
        check_receipt(a.kind,a.compare_with)
        left,right=Path(str(a.out)+'-capture'),Path(str(a.compare_with)+'-capture')
        if (left/'receipt.json').read_bytes()!=(right/'receipt.json').read_bytes():
            raise ValueError('兩包完整流程收據不同')
        pngs=lambda root:{p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in root.glob('*.png')}
        if pngs(left)!=pngs(right):raise ValueError('兩包完整PNG不同')
        report['cross_pack']=dict(other=str(a.compare_with),pngs=50,frames=556,
                                 receipt_sha256=hashlib.sha256((left/'receipt.json').read_bytes()).hexdigest())
    report['verifier_sha256']=hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    Path(str(a.out)+('.audit.json' if a.check_existing else '.gate.json')).write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','kind':a.kind,'required':len(REQUIRED[a.kind])}))


if __name__=='__main__':
    main()
