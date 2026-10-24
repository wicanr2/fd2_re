#!/usr/bin/env python3
"""在既有 Go Docker 驗證明示 BG／TAI 候選，拒收 SKIP、缺項、OOM 或逾時。"""
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
    'hud-unit': {'TestModernHDHUDUnitSelectionAndHP','TestModernHDHUDUnitAdmissionAndEventShift',
                 'TestModernHDMapTransactionUsesOneNativeClockAndIndependentTerrain','TestModernHDMapRejectedArtDoesNotPublishNativeState'},
    'hud-game': {'TestModernHDHUDUnitSourcesAndGameGPU'},
    'modifier-map-fixture': {'TestModernModifierMapFixtureAdmission','TestModernModifierMapControlSnapshot'},
    'modifier-map-source': {'TestModernModifierMapSourcesAndAtomic'},
    'modifier-map-game': {'TestModernModifierMapRuntimeGPU'},
    'modifier-map-zero-seed': {'TestModernModifierMapZeroSeedApplicationGPU'},
    'modifier-map-batch': {'TestModernModifierMapBatchGPU'},
    'modifier-map-regression': {'TestExecuteNativeAICommand17UsesOriginalIndexedTail','TestNativeAICommandModifierLateCancelRollsBack',
                                'TestExecuteNativeAICommand20UsesOriginalIndexedTail','TestExecuteNativeAICommand26UsesOriginalIndexedTail',
                                'TestPlayerNativeCommand20UsesPaletteThenOriginalIndexedTail','TestPlayerNativeCommand25UsesOriginalTailWithoutEmptyNumericHold'},
    'compound-tail-art-source': {'TestModernCompoundTailArtAtomic','TestModernCompoundTailArtSourceGPU'},
    'modifier-tail-art-source': {'TestModernModifierTailArtAtomic','TestModernModifierTailArtSourceGPU'},
    'compound-map-unit': {'TestModernCompoundMapBindingAtomic'},
    'compound-map-source': {'TestModernCompoundMapAllSources'},
    'compound-map-game': set(),
    'ordinary-scenery-unit': {'TestModernBattleSurfaceLUTColourAndAlpha','TestModernBattleSurfaceLUTEndpoints',
                             'TestModernBattleSurfaceLUTIdentityAndSourceGuard','TestModernOrdinarySceneryAtomic','TestModernOrdinaryAI9OppositeSideRejected'},
    'ordinary-scenery-fallback': {'TestModernBattleSurfaceLUTFallbackGPU'},
    'ordinary-scenery-source': {'TestModernCommandPanelAllOwners'},
    'ordinary-scenery-game': set(),
    'compound-scenery-unit': {'TestModernCompoundSceneryAtomic'},
    'compound-scenery-regression': {'TestStartNativeCommand32PresentationUsesOriginalAssetsEndToEnd',
                                    'TestStartNativeCommand33PresentationUsesOriginalAssetsEndToEnd',
                                    'TestStartNativeCommand34PresentationUsesOriginalAssetsEndToEnd',
                                    'TestStartNativeCommand35PresentationUsesOriginalAssetsEndToEnd'},
    'compound-scenery-source': {'TestModernCompoundScenerySources'},
    'compound-scenery-game': set(),
    'command-panel-unit': {'TestModernCommandPanelCollectorAtomic','TestModernCommandPanelPalette'},
    'command-panel-owner': {'TestModernCommandPanelAllOwners'},
    'command-panel-game': set(),
	'panel-unit': {'TestModernBattlePanelTextFrozenSources', 'TestModernBattlePanelTextFourLocaleInk', 'TestModernBattlePanelTextFourLocalePreview'},
	'panel-source': {'TestModernBattlePanelTextGPU'},
	'panel-game': {'TestModernHDBattleRuntimeGPU'},
    'panel-scroll': {'TestModernHDScrollRuntimeGPU'},
    'scenery-legacy': {'TestModernBattleSceneryLegacyFallbackGPU'},
    'scenery-native': {'TestModernBattleSceneryAllNativeGeometry'},
    'scenery-unit': {'TestModernBattleSceneryAllSourcesAndAtomic'},
    'scenery-source': {'TestModernBattleScenerySourcesGPU'},
    'transition-source': {'TestModernHDTransitionOnlyGPU'},
    'split-unit': {'TestModernHDSplitStrikeSceneryNativeAndAtomic'},
    'split-game': {'TestModernHDSplitStrikeRuntimeGPU'},
    'split-native-gpu': {'TestModernHDSplitStrikeNativeSourceGPU'},
    'split-source': {'TestModernHDSplitStrikeSceneryOnlyGPU'},
    'scroll-unit': {'TestModernScrollSceneryNativeAndAtomic'},
    'scroll-source': {'TestModernScrollSceneryGPU'},
    'scroll-game': {'TestModernHDScrollRuntimeGPU'},
    'scroll-regression': {'TestNativePhysicalBodyMultiStrikeCounterAndPulse', 'TestNativePhysicalBodyMissingSoundStopsBeforeSettlement',
                          'TestModernScrollMetadataRejectsBeforePartialSources', 'TestModernScrollSourceClipRejectsBeforeScaling',
                          'TestModernHDSplitStrikePreflightAndF2Atomic', 'TestModernHDSplitStrikeLateFailureRestoresTransaction'},
    'source-unit': {'TestModernBattleSurfaceHDSourceAndAtomic', 'TestModernBattleSurfacePhysicalNativeReconstruction'},
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
    p.add_argument('--modifier-command',type=int,choices=[17,18,19,20,21,22,23,25,26,27],default=17)
    p.add_argument('--modifier-failed',action='store_true')
    p.add_argument('--modifier-start',type=int,choices=range(21),default=0)
    p.add_argument('--modifier-limit',type=int,choices=range(1,7),default=6)
    p.add_argument('--panel-locale', choices=['zh-Hant','zh-Hans','ja','en'], default='zh-Hant')
    p.add_argument('--panel-owner',type=int,choices=[0,1,2,3,4,5,6,7,8,9,24,29,32,33,34,35],default=0)
    p.add_argument('--surface', type=int, choices=range(56), default=4)
    p.add_argument('--scenery-start',type=int,default=0)
    p.add_argument('--scenery-limit',type=int,default=16)
    p.add_argument('--transition',action='store_true',help='要求BG0..2真實HD來源與完整過渡擷取')
    p.add_argument('--pair', choices=['4:5','5:6','6:4','7:10','34:36','36:37'], default='4:5')
    p.add_argument('--command',type=int,choices=[24,29],default=24)
    p.add_argument('--triple',choices=['4:5:6','5:6:4','6:4:5','7:9:10','34:36:37','40:43:47'],default='4:5:6')
    p.add_argument('--hashes-only',action='store_true')
    p.add_argument('--with-pedestal', action='store_true')
    p.add_argument('--require-real-runtime', action='store_true')
    p.add_argument('--all-compound-effects', action='store_true', help='四召喚地圖尾段不允許任何可見LMI回退')
    p.add_argument('--all-modifier-effects', action='store_true', help='狀態法術地圖尾段不允許任何可見LMI回退')
    p.add_argument('--timeout', type=int, default=1200)
    a = p.parse_args()
    if a.kind=='modifier-map-zero-seed' and (a.modifier_command!=26 or not a.enemy or a.modifier_failed):
        raise SystemExit('固定0補測只使用既有敵方26正例')
    if a.kind=='modifier-map-batch' and a.modifier_start+a.modifier_limit>21:
        raise SystemExit('狀態法術分片越界')
    if a.all_compound_effects and a.kind!='compound-map-game':raise SystemExit('--all-compound-effects只用於compound-map-game')
    if a.all_modifier_effects and a.kind not in ['modifier-map-source','modifier-map-game','modifier-map-zero-seed','modifier-map-batch']:
        raise SystemExit('--all-modifier-effects只用於狀態法術來源或GPU流程')
    if a.kind in ['ordinary-scenery-source','ordinary-scenery-game'] and a.panel_owner not in range(10):
        raise SystemExit('一般場景需要明示0～9的owner，9只指敵方戰鬥')
    if a.kind in ['ordinary-scenery-source','ordinary-scenery-game'] and a.panel_owner==9 and a.enemy:
        raise SystemExit('敵方9原owner只接受raw side0；另一側由ordinary-scenery-unit驗原子拒收')
    if a.kind in ['compound-scenery-source','compound-scenery-game'] and a.panel_owner not in [32,33,34,35]:
        raise SystemExit('召喚背景需要明示32～35的owner')
    if a.kind in ['command-panel-game','compound-scenery-game','ordinary-scenery-game','compound-map-game']:
        name={0:'TestModernHDCommand0RuntimeGPU',1:'TestModernHDCommand1RuntimeGPU',4:'TestModernHDCommand5RuntimeGPU',5:'TestModernHDCommand5RuntimeGPU',24:'TestModernHDSplitStrikeRuntimeGPU',29:'TestModernHDSplitStrikeRuntimeGPU',32:'TestModernHDCompoundRuntimeGPU',33:'TestModernHDFairyRuntimeGPU',34:'TestModernHDStagedSummonRuntimeGPU',35:'TestModernHDStagedSummonRuntimeGPU'}.get(a.panel_owner,'TestModernHDOrdinaryRemainingRuntimeGPU')
        REQUIRED[a.kind]={name}
    if not Path('/.dockerenv').exists() or not 60 <= a.timeout <= 1800 or not a.out.parent.is_dir():
        raise SystemExit('Docker 內使用有界時間與既有輸出父目錄')
    for suffix in ['.gate.json','.log','.memory.json']:
        if Path(str(a.out)+suffix).exists(): raise SystemExit('不覆寫收據')
    catalog = json.loads(a.catalog.read_text())
    independent = check_real_runtime(a.pack, catalog) if a.require_real_runtime else None
    env = os.environ.copy()
    env.pop('FD2_HD_HUD_GPU',None)
    env.pop('FD2_HD_HUD_OUT',None)
    if a.kind=='hud-game':
        env.update(FD2_HD_HUD_GPU='1',FD2_HD_HUD_OUT=str(a.out)+'-capture',
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
    env.update(FD2_MUTE='1', FD2_TEST_MODERN_BATTLE_CATALOG=str(a.catalog.resolve()), FD2_TEST_MODERN_BATTLE_PACK=str(a.pack.resolve()))
    for name in ['FD2_HD_MODIFIER_MAP_UNIT','FD2_HD_MODIFIER_MAP_GPU','FD2_HD_MODIFIER_MAP_REPORT','FD2_HD_MODIFIER_MAP_FIXTURE','FD2_HD_MODIFIER_MAP_ZERO','FD2_HD_MODIFIER_MAP_BATCH','FD2_HD_MODIFIER_REQUIRE_EFFECTS']:env.pop(name,None)
    if a.all_modifier_effects:env['FD2_HD_MODIFIER_REQUIRE_EFFECTS']='1'
    if a.kind=='modifier-map-fixture':env['FD2_HD_MODIFIER_MAP_FIXTURE']='1'
    if a.kind=='modifier-map-source':
        env.update(FD2_HD_MODIFIER_MAP_UNIT='1',FD2_HD_MODIFIER_MAP_REPORT=str(a.out),
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
    if a.kind in ['modifier-map-game','modifier-map-zero-seed','modifier-map-batch']:
        env.update(FD2_HD_MODIFIER_MAP_GPU='1',FD2_HD_MODIFIER_COMMAND=str(a.modifier_command),
                   FD2_HD_MODIFIER_PLAYER='0' if a.enemy else '1',FD2_HD_MODIFIER_FAILED='1' if a.modifier_failed else '0',
                   FD2_HD_MODIFIER_MAP_REPORT=str(a.out),FD2_HD_MODIFIER_CAPTURE=str(a.out)+'-capture',
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
        if a.kind=='modifier-map-zero-seed':env['FD2_HD_MODIFIER_MAP_ZERO']='1'
        if a.kind=='modifier-map-batch':
            env.update(FD2_HD_MODIFIER_MAP_BATCH='1',FD2_HD_MODIFIER_START=str(a.modifier_start),FD2_HD_MODIFIER_LIMIT=str(a.modifier_limit))
    for name in ['FD2_HD_BATTLE_LIFECYCLE_GPU','FD2_HD_BATTLE_REAL','FD2_HD_BATTLE_ENEMY_ACTOR','FD2_HD_SURFACE_GPU','FD2_HD_BATTLE_BACKGROUND','FD2_HD_BATTLE_PEDESTAL']:
        env.pop(name,None)
    for name in ['FD2_HD_SCROLL_SIDE','FD2_HD_SCROLL_SCENERY','FD2_HD_SCROLL_SCENERY_GPU','FD2_HD_SCROLL_REAL','FD2_HD_SCROLL_TARGET','FD2_HD_SCROLL_PROJECTILES','FD2_HD_SCROLL_TARGET_FRAMES']:
        env.pop(name,None)
    for name in ['FD2_HD_SPLIT_SCENERY','FD2_HD_SPLIT_SCENERY_SIDE','FD2_HD_SPLIT_STRIKE_GPU','FD2_SPLIT_HASHES_ONLY']:
        env.pop(name,None)
    env.pop('FD2_HD_TRANSITION_REQUIRED',None)
    env.pop('FD2_HD_TRANSITION_GPU',None)
    env.pop('FD2_HD_SCENERY_BATCH',None)
    for name in ['FD2_HD_PANEL_GPU','FD2_HD_PANEL_TEXT','FD2_HD_PANEL_LOCALE','FD2_HD_PANEL_PREVIEW']:env.pop(name,None)
    for name in ['FD2_HD_COMMAND_PANEL_OWNER','FD2_HD_COMMAND_PANEL_REPORT','FD2_HD_COMMAND_PANEL_TEXT']:env.pop(name,None)
    for name in ['FD2_HD_COMPOUND_SCENERY','FD2_HD_COMPOUND_SCENERY_SIDE','FD2_HD_COMPOUND_SCENERY_SURFACE']:env.pop(name,None)
    for name in ['FD2_HD_ORDINARY_SCENERY','FD2_HD_ORDINARY_SCENERY_SIDE','FD2_HD_ORDINARY_SCENERY_SURFACE']:env.pop(name,None)
    env.pop('FD2_HD_ORDINARY_LUT_FALLBACK_GPU',None)
    env.pop('FD2_HD_COMPOUND_MAP',None)
    env.pop('FD2_HD_COMPOUND_MAP_REPORT',None)
    for name in ['FD2_HD_COMPOUND_TAIL_ART_UNIT','FD2_HD_COMPOUND_TAIL_ART_GPU','FD2_HD_COMPOUND_TAIL_ART_REPORT']:env.pop(name,None)
    if a.kind=='compound-tail-art-source':
        env.update(FD2_HD_COMPOUND_TAIL_ART_UNIT='1',FD2_HD_COMPOUND_TAIL_ART_GPU='1',
                   FD2_HD_COMPOUND_TAIL_ART_REPORT=str(a.out)+'.source.json',
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
    for name in ['FD2_HD_MODIFIER_TAIL_ART_UNIT','FD2_HD_MODIFIER_TAIL_ART_GPU','FD2_HD_MODIFIER_TAIL_ART_REPORT']:env.pop(name,None)
    if a.kind=='modifier-tail-art-source':
        env.update(FD2_HD_MODIFIER_TAIL_ART_UNIT='1',FD2_HD_MODIFIER_TAIL_ART_GPU='1',
                   FD2_HD_MODIFIER_TAIL_ART_REPORT=str(a.out)+'.source.json',
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
    if a.kind in ['compound-map-source','compound-map-game']:
        if a.panel_owner not in [32,33,34,35]:raise SystemExit('需要32～35召喚owner')
        env.update(FD2_HD_COMPOUND_MAP='1',FD2_HD_COMPOUND_MAP_REPORT=str(a.out)+'.map-owner.json',
                   FD2_HD_COMMAND_PANEL_OWNER=str(a.panel_owner),FD2_HD_COMMAND_PANEL_TEXT='1',FD2_HD_PANEL_LOCALE=a.panel_locale,
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()),
                   FD2_HD_COMPOUND_SCENERY='1',FD2_HD_COMPOUND_SCENERY_SIDE='2' if a.enemy else '0',FD2_HD_COMPOUND_SCENERY_SURFACE=str(a.surface))
    if a.kind=='ordinary-scenery-fallback':env['FD2_HD_ORDINARY_LUT_FALLBACK_GPU']='1'
    if a.kind in ['ordinary-scenery-source','ordinary-scenery-game']:
        env.update(FD2_HD_ORDINARY_SCENERY='1',FD2_HD_ORDINARY_SCENERY_SIDE='2' if a.enemy else '0',
                   FD2_HD_ORDINARY_SCENERY_SURFACE=str(a.surface),FD2_HD_COMMAND_PANEL_TEXT='1',
                   FD2_HD_COMMAND_PANEL_OWNER=str(a.panel_owner),FD2_HD_PANEL_LOCALE=a.panel_locale)
    if a.kind in ['compound-scenery-source','compound-scenery-game']:
        env.update(FD2_HD_COMPOUND_SCENERY='1',FD2_HD_COMPOUND_SCENERY_SIDE='2' if a.enemy else '0',FD2_HD_COMPOUND_SCENERY_SURFACE=str(a.surface),
                   FD2_HD_COMMAND_PANEL_TEXT='1',FD2_HD_COMMAND_PANEL_OWNER=str(a.panel_owner),FD2_HD_PANEL_LOCALE=a.panel_locale)
    if a.kind=='compound-scenery-source':env['FD2_HD_COMMAND_PANEL_REPORT']=str(a.out)+'.owner.json'
    if a.kind in ['command-panel-owner','ordinary-scenery-source']:env.update(FD2_HD_COMMAND_PANEL_OWNER=str(a.panel_owner),FD2_HD_COMMAND_PANEL_REPORT=str(a.out)+'.owner.json')
    if a.kind in ['command-panel-game','compound-scenery-game','ordinary-scenery-game','compound-map-game']:
        env.update(FD2_HD_COMMAND_PANEL_TEXT='1',FD2_HD_PANEL_LOCALE=a.panel_locale,FD2_HD_COMMAND_PANEL_OWNER=str(a.panel_owner),
                   FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()),FD2_TEST_MODERN_EFFECT_OUT=str(a.out)+'-capture')
        if a.panel_owner in [0,1,4,5]:env[f'FD2_HD_COMMAND{5 if a.panel_owner==4 else a.panel_owner}_GPU']='1'
        elif a.panel_owner in [2,3,6,7,8,9]:env.update(FD2_HD_ORDINARY_GPU='1',FD2_TEST_ORDINARY_COMMAND=str(a.panel_owner))
        elif a.panel_owner in [24,29]:env.update(FD2_HD_SPLIT_STRIKE_GPU='1',FD2_TEST_SPLIT_STRIKE_COMMAND=str(a.panel_owner))
        elif a.panel_owner==32:env['FD2_HD_COMPOUND_GPU']='1'
        elif a.panel_owner==33:env['FD2_HD_FAIRY_GPU']='1'
        else:env['FD2_HD_STAGED_COMMAND']=str(a.panel_owner)
    if a.kind=='panel-unit':env['FD2_HD_PANEL_PREVIEW']=str(a.out)+'.png'
    if a.kind=='panel-source':env['FD2_HD_PANEL_GPU']='1'
    if a.kind in ['panel-game','panel-scroll']:env.update(FD2_HD_PANEL_TEXT='1',FD2_HD_PANEL_LOCALE=a.panel_locale)
    if a.kind in ['scenery-native','scenery-unit','scenery-source','scenery-legacy']:env['FD2_HD_SCENERY_BATCH']='1'
    if a.kind=='scenery-source':
        if not 0<=a.scenery_start<86 or not 1<=a.scenery_limit<=16:raise SystemExit('完整場景GPU使用有界16來源切片')
        env.update(FD2_HD_SCENERY_START=str(a.scenery_start),FD2_HD_SCENERY_LIMIT=str(a.scenery_limit),FD2_TEST_HD_SCENERY_OUT=str(a.out)+'.source.json')
    if a.kind=='scenery-legacy':env['FD2_TEST_HD_SCENERY_OUT']=str(a.out)+'.legacy.json'
    if a.transition:env['FD2_HD_TRANSITION_REQUIRED']='1'
    env['FD2_HD_BATTLE_SURFACE_INDEX']=str(a.surface)
    if a.with_pedestal: env['FD2_HD_BATTLE_PEDESTAL']=str(a.surface)
    if a.kind in ['game','panel-game']:
        env.update(FD2_HD_BATTLE_LIFECYCLE_GPU='1',FD2_HD_BATTLE_REAL='1',FD2_HD_BATTLE_BACKGROUND=str(a.surface),
                   FD2_TEST_MODERN_BATTLE_FIGURE='76' if a.enemy else '0',FD2_TEST_HD_BATTLE_OUT=str(a.out)+'-capture')
        if a.enemy: env['FD2_HD_BATTLE_ENEMY_ACTOR']='1'
    elif a.kind == 'source': env['FD2_HD_SURFACE_GPU']='1'
    elif a.kind == 'legacy-gpu': env['FD2_HD_BATTLE_GPU']='1'
    elif a.kind == 'scroll-source': env['FD2_HD_SCROLL_SCENERY_GPU']='1'
    elif a.kind == 'transition-source':env['FD2_HD_TRANSITION_GPU']='1'
    elif a.kind == 'scroll-regression':
        env.update(FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
    elif a.kind in ['scroll-game','panel-scroll']:
        env.update(FD2_HD_SCROLL_SIDE='2' if a.enemy else '0',FD2_HD_SCROLL_REAL='1',FD2_HD_SCROLL_SCENERY=a.pair,
                   FD2_TEST_HD_SCROLL_OUT=str(a.out)+'-capture')
    if a.kind in ['split-unit','split-game','split-native-gpu','split-source']:
        env.update(FD2_TEST_MODERN_EFFECT_CATALOG=str(a.catalog.resolve()),FD2_TEST_MODERN_EFFECT_PACK=str(a.pack.resolve()))
    if a.kind in ['split-game','split-native-gpu','split-source']:
        env.update(FD2_HD_SPLIT_STRIKE_GPU='1',FD2_TEST_MODERN_EFFECT_OUT=str(a.out)+'-capture')
    if a.kind=='split-game':
        env.update(FD2_TEST_SPLIT_STRIKE_COMMAND=str(a.command),FD2_HD_SPLIT_SCENERY=a.triple,FD2_HD_SPLIT_SCENERY_SIDE='2' if a.enemy else '0')
        if a.hashes_only:env['FD2_SPLIT_HASHES_ONLY']='1'
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
    if a.kind=='hud-game':
        d=json.loads((Path(str(a.out)+'-capture')/'gpu.json').read_text())
        rows=d.get('cases',[])
        if (d.get('status')!='PASS-HUD-UNIT-HD-RUNTIME-E1' or d.get('groups')!=125 or d.get('source_states')!=500
                or d.get('fixed_clock_unix')!=1000 or d.get('native_rng_seed')!=7 or len(rows)!=6
                or sum(r.get('game_draw',False) and r.get('f2',False) for r in rows)!=4):
            raise SystemExit('HUD缺少完整圖源、Game.Draw或F2驗證')
    if a.kind in ['command-panel-owner','ordinary-scenery-source']:
        receipt=json.loads(Path(str(a.out)+'.owner.json').read_text())
        if (receipt.get('status')!='PASS-TYPED-COMMAND-PANEL' or receipt.get('command_id')!=a.panel_owner
                or receipt.get('bound_battle_frames',0)<=0 or not receipt.get('native_sources_equal')
                or not receipt.get('unit_state_equal') or not receipt.get('rng_equal')):
            raise SystemExit('缺少指令面板原生來源、狀態或RNG核對')
        if a.kind=='ordinary-scenery-source':
            scenery=receipt.get('ordinary_scenery',{})
            counts=scenery.get('sources',{})
            if (scenery.get('side')!=('2' if a.enemy else '0') or scenery.get('control_surface')!=str(a.surface)
                    or sum(v for k,v in counts.items() if k.startswith('BG.DAT:'))!=receipt['bound_battle_frames']
                    or sum(v for k,v in counts.items() if k.startswith('TAI.DAT:'))!=receipt['bound_battle_frames']):
                raise SystemExit('缺少一般指令每幀的BG／TAI來源')
    if a.kind=='compound-scenery-source':
        receipt=json.loads(Path(str(a.out)+'.owner.json').read_text())
        if (receipt.get('status')!='PASS-COMPOUND-SCENERY-SOURCES' or receipt.get('command')!=a.panel_owner
                or receipt.get('surface')!=a.surface
                or receipt.get('side')!=(2 if a.enemy else 0) or receipt.get('battle_frames',0)<=0
                or receipt.get('platform_frames')!=(receipt['battle_frames'] if a.enemy else 0)):
            raise SystemExit('缺少雙側召喚背景／台座來源核對')
    if a.kind in ['command-panel-game','compound-scenery-game','ordinary-scenery-game','compound-map-game']:
        receipt=json.loads(Path(str(a.out)+'-capture/receipt.json').read_text())
        panel=receipt.get('panel_text',{})
        if (panel.get('locale')!=a.panel_locale or panel.get('audited_draws',0)<=0 or panel.get('matched_rgba_pixels',0)<100
                or receipt.get('continuations')!=1):
            raise SystemExit('缺少實際Game.Draw面板字形核對或一次續行')
        if a.kind=='ordinary-scenery-game':
            scenery=receipt.get('ordinary_scenery',{})
            phases=receipt.get('hd_phase_draws',{})
            if (scenery.get('side')!=('2' if a.enemy else '0') or scenery.get('control_surface')!=str(a.surface)
                    or scenery.get('audited_draws',0)<=0 or scenery.get('matched_pixels',0)<100 or scenery.get('changed_lut_pixels',0)<100
                    or not phases or any(scenery.get('sampled_phases',{}).get(p,0)<=0 for p in phases)
                    or not any(k.startswith('BG.DAT:') and v>0 for k,v in scenery.get('sources',{}).items())
                    or not (any(k.startswith('TAI.DAT:') and v>0 for k,v in scenery.get('sources',{}).items())
                            or any(k.startswith('TAI.DAT:') and v>0 for k,v in scenery.get('native_blank_sources',{}).items()))):
                raise SystemExit('缺少一般指令各階段HD場景／實際查表變色')
        if a.kind=='compound-scenery-game':
            scenery=receipt.get('compound_scenery',{})
            phases=receipt.get('hd_phase_draws',{})
            if (scenery.get('side')!=('2' if a.enemy else '0') or scenery.get('surface')!=str(a.surface) or scenery.get('audited_draws',0)<=0
                    or scenery.get('matched_pixels',0)<100 or not scenery.get('sources')
                    or any(scenery.get('sampled_phases',{}).get(p,0)<=0 for p in phases)
                    or scenery['sources'].get(f'BG.DAT:{a.surface}',0)<=0
                    or (a.enemy and scenery['sources'].get(f'TAI.DAT:{a.surface}',0)<=0)):
                raise SystemExit('缺少召喚各階段的實際HD背景／台座像素')
    if a.kind in ['game','panel-game']:
        receipt = json.loads(Path(str(a.out)+'-capture/receipt.json').read_text())
        if receipt.get('background_selection',{}).get('BaseBG') != a.surface or not receipt.get('background_hd_sha256') or receipt['continuations'] != 1:
            raise SystemExit('缺少所選 BG Game.Draw 或一次返回')
        if a.with_pedestal and (receipt.get('background_selection',{}).get('TAI') != a.surface or not receipt.get('pedestal_hd_sha256')):
            raise SystemExit('缺少所選 TAI Game.Draw')
        if a.kind=='panel-game' and (receipt.get('panel_text',{}).get('locale')!=a.panel_locale or receipt.get('panel_text',{}).get('frozen_snapshots',0)<4 or receipt.get('panel_text',{}).get('actual_gpu_draws',0)<=0):
            raise SystemExit('缺少面板凍結來源或實際GPU繪製')
    if a.kind in ['scroll-game','panel-scroll']:
        receipt=json.loads(Path(str(a.out)+'-capture/receipt.json').read_text())
        phases=['prelude','departure','scroll-out','scroll-in','body','counter']
        if (receipt.get('scenery_pair')!=a.pair or receipt.get('side')!=(2 if a.enemy else 0)
                or receipt.get('catalog_sha256')!=hashlib.sha256(a.catalog.read_bytes()).hexdigest()
                or not receipt.get('scene_selection',{}).get('HasSeparateBackgrounds')
                or receipt.get('continuations')!=1 or receipt.get('battle_dac_steps')!=64 or receipt.get('map_dac_steps')!=65
                or any(receipt.get('phase_f2',{}).get(p)!=1 or receipt.get('phase_hd_draws',{}).get(p,0)<=0 for p in phases)
                or len(receipt.get('scenery_sources',[]))!=3
                or any(receipt.get('scenery_actual_capture_draws',{}).get(s['sha256'],0)<=0 for s in receipt.get('scenery_sources',[]))):
            raise SystemExit('缺少分離場景完整Game.Draw／來源／F2／DAC／一次續行')
        if a.transition and (len(receipt.get('transition_sources',[]))!=3
                or any(receipt.get('scenery_actual_capture_draws',{}).get(s['sha256'],0)<=0 for s in receipt.get('transition_sources',[]))):
            raise SystemExit('缺少全部三個過渡來源的實際Draw')
        if a.kind=='panel-scroll' and (receipt.get('panel_text',{}).get('locale')!=a.panel_locale or any(receipt.get('panel_text',{}).get('actual_gpu_phase_draws',{}).get(phase,0)<=0 for phase in phases)):
            raise SystemExit('缺少六階段實際HD面板')
    if a.kind=='split-game':
        receipt=json.loads(Path(str(a.out)+'-capture/receipt.json').read_text())
        if (receipt.get('scenery_triple')!=a.triple or receipt.get('scenery_side')!=('2' if a.enemy else '0')
                or receipt.get('command_id')!=a.command or receipt.get('continuations')!=1
                or receipt.get('mp_changes')!=1 or receipt.get('hp_changes')!=([1] if a.command==24 else [1,1])
                or len(receipt.get('phase_f2',{}))!=(5 if a.command==24 else 8)
                or any(v!=1 for v in receipt.get('phase_f2',{}).values()) or not receipt.get('scenery_actual_sources')
                or receipt.get('catalog_sha256')!=hashlib.sha256(a.catalog.read_bytes()).hexdigest()):
            raise SystemExit('缺少24／29完整場景Game.Draw／F2／交易／一次續行')
        if a.transition and receipt.get('transition_required') is not True:
            raise SystemExit('缺少全部三個過渡來源的實際Draw')
    if a.kind=='scenery-source':
        receipt=json.loads(Path(str(a.out)+'.source.json').read_text())
        if (receipt.get('status')!='PASS-COMPLETE-SCENERY-SOURCE-GPU' or receipt.get('start')!=a.scenery_start
                or receipt.get('end')!=min(86,a.scenery_start+a.scenery_limit) or receipt.get('total_assets')!=86
                or len(receipt.get('assets',[]))!=receipt['end']-receipt['start']
                or receipt.get('completed_gpu_cases')!=2*len(receipt['assets'])):
            raise SystemExit('完整場景GPU切片缺少實際繪圖')
    if a.kind=='scenery-legacy':
        receipt=json.loads(Path(str(a.out)+'.legacy.json').read_text())
        if (receipt.get('status')!='PASS-OLD684-NEW-SCENERY-FALLBACK-GPU'
                or receipt.get('legacy_catalog_sha256')!=hashlib.sha256(a.catalog.read_bytes()).hexdigest()
                or receipt.get('legacy_registered_surfaces')!=9 or receipt.get('completed_gpu_cases')!=16
                or len(set(receipt.get('source_ids',[])))!=8):
            raise SystemExit('舊684主題缺少新場景原圖GPU回退')
    if a.kind=='compound-map-source':
        receipt=json.loads(Path(str(a.out)+'.map-owner.json').read_text())
        if (receipt.get('status')!='PASS-COMPOUND-MAP-SOURCES' or receipt.get('command')!=a.panel_owner or receipt.get('map_frames',0)<80
                or receipt.get('phases',{}).get('map-ramp')!=41 or receipt.get('atomic_rejections')!=2
                or not all(receipt.get(k) for k in ['native_reconstruction','rng_unchanged','unit_state_unchanged'])):
            raise SystemExit('召喚尾段來源／原生重組／原子拒收未驗')
    if a.kind=='compound-map-game':
        m=receipt.get('compound_map',{})
        if (m.get('audited_draws',0)<80 or m.get('matched_pixels',0)<10000 or m.get('registered_layers',0)<=0
                or m.get('registered_overlay_pixels',0)<=0 or m.get('changed_overlay_pixels',0)<=0):
            raise SystemExit('召喚尾段缺實際HD世界與GPU回讀')
        if a.all_compound_effects and m.get('native_fallback_layers')!=0:
            raise SystemExit('召喚尾段仍有原生LMI回退')
    if a.kind=='compound-tail-art-source':
        d=json.loads(Path(str(a.out)+'.source.json').read_text())
        required=set(range(64,76))|set(range(126,191))
        if (d.get('sources')!=77 or d.get('worlds')!=154 or len(d.get('cases',[]))!=154
                or {c['index'] for c in d['cases']}!=required
                or any({c['placement'] for c in d['cases'] if c['index']==i}!={0,1} for i in required)):
            raise SystemExit('77份尾段光效缺少全部GPU來源或邊界取樣')
    if a.kind=='modifier-map-source':
        d=json.loads(Path(str(a.out)+'.sources.json').read_text())
        wanted={(i,True,False) for i in [17,18,19,20,21,22,23,25,26,27]}|{(i,False,False) for i in [17,18,19,20,21,22,26,27]}|{(i,True,True) for i in [20,22,25]}
        rows=d.get('cases',[])
        if (len(rows)!=21 or {(r['case']['ID'],r['case']['Player'],r['case']['Failed']) for r in rows}!=wanted
                or any(r.get('frames',0)<8 or r.get('atomic_rejections')!=2 for r in rows)):
            raise SystemExit('狀態法術缺少完整來源或錯來源原子拒收')
        if a.all_modifier_effects and any(r.get('native_fallback_layers')!=0 for r in rows):
            raise SystemExit('狀態法術來源仍有原生LMI回退')
    if a.kind=='modifier-tail-art-source':
        d=json.loads(Path(str(a.out)+'.source.json').read_text())
        required=set(range(191,230))
        if (d.get('sources')!=39 or d.get('worlds')!=78 or len(d.get('cases',[]))!=78
                or {c['index'] for c in d['cases']}!=required
                or any({c['placement'] for c in d['cases'] if c['index']==i}!={0,1} for i in required)):
            raise SystemExit('39份狀態光效缺少完整GPU來源或邊界取樣')
    if a.kind in ['modifier-map-game','modifier-map-zero-seed','modifier-map-batch']:
        suffix='.batch.json' if a.kind=='modifier-map-batch' else '.game.json'
        report_data=json.loads(Path(str(a.out)+suffix).read_text())
        rows=report_data.get('cases',[]) if a.kind=='modifier-map-batch' else [report_data]
        expected={(a.modifier_command,not a.enemy,a.modifier_failed)}
        if a.kind=='modifier-map-batch':
            cases=[]
            for i in [17,18,19,20,21,22,23,25,26,27]:
                cases.append((i,True,False))
                if i not in [23,25]:cases.append((i,False,False))
            cases += [(i,True,True) for i in [20,22,25]]
            expected=set(cases[a.modifier_start:a.modifier_start+a.modifier_limit])
        if len(rows)!=len(expected) or {(d['case']['ID'],d['case']['Player'],d['case']['Failed']) for d in rows}!=expected:
            raise SystemExit('狀態法術缺少明示的全部流程')
        for d in rows:
            if a.all_modifier_effects and d.get('native_fallback_layers')!=0:
                raise SystemExit('狀態法術尾段仍有原生LMI回退')
            seed=0 if a.kind=='modifier-map-zero-seed' else 7
            if a.kind=='modifier-map-zero-seed' and d.get('application_successes',0)<=0:
                raise SystemExit('固定0補測未觸發狀態套用')
            if (d.get('seed')!=seed or d.get('native_control_seed')!=seed or d.get('clock_unix_seconds')!=1000 or d.get('continuations')!=1
                or not d.get('native_control_units_rng_equal_after_every_step') or d.get('audited_draws',0)<8
                or d.get('matched_pixels',0)<10000 or d.get('different_native_hd_lattice_pixels',0)<=0
                or d.get('faithful_palette_pixels',0)<64000 or not d.get('phase_f2_switches')
                or not all(d['phase_f2_switches'].values()) or not d.get('source_shots')):
                raise SystemExit('狀態法術缺少實際GPU、原生對照、F2或返回')
    report = dict(status='PASS',kind=a.kind,enemy=a.enemy,required=sorted(REQUIRED[a.kind]),
                  catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(),
                  catalog=str(a.catalog.resolve()),pack=str(a.pack.resolve()),with_pedestal=a.with_pedestal,surface=a.surface,pair=a.pair,
                  command_id=a.command,triple=a.triple,hashes_only=a.hashes_only,transition_required=a.transition,
                  panel_locale=a.panel_locale,
                  panel_owner=a.panel_owner,
                  modifier_command=a.modifier_command,modifier_failed=a.modifier_failed,
                  modifier_start=a.modifier_start,modifier_limit=a.modifier_limit,
                  all_compound_effects=a.all_compound_effects,
                  all_modifier_effects=a.all_modifier_effects,
                  scenery_start=a.scenery_start,scenery_limit=a.scenery_limit,
                  independent_runtime_files=independent, memory=str(a.out)+'.memory.json',log=str(a.out)+'.log',
                  scope='固定來源／GPU／明示物理戰鬥fixture，非正常玩家或新原版RGB對拍')
    Path(str(a.out)+'.gate.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','kind':a.kind,'required':len(REQUIRED[a.kind])}))


if __name__ == '__main__': main()
