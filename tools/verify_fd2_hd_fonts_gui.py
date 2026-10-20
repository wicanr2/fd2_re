#!/usr/bin/env python3
"""Docker 內以正常 START、F4、F9 與商店按鍵抽驗 HD 字型，不錄影。"""
import argparse
import json
import os
import re
from pathlib import Path
import subprocess
import threading
import time

from verify_fd2_hd_gui import digest, latest
from verify_fd2_hd_panel_transitions import PanelTransitionProbe
from verify_fd2_hd_timed_dialogue import TimedDialogueProbe
from verify_fd2_hd_terrain_walk import walk_behind_house
from verify_fd2_gaia_focus import focus_gaia, sample_gaia
from fd2_hd_death_gui import capture_standalone_ch23_source, run_ch23_death_event


def terrain_frame_ids(value):
    parts=value.split(',')
    if not parts or any(not re.fullmatch(r'[0-9]+',part) for part in parts):
        raise argparse.ArgumentTypeError('地形frame須為逗號分隔的非負整數')
    frames=[int(part) for part in parts]
    if len(set(frames))!=len(frames) or any(frame>65535 for frame in frames):
        raise argparse.ArgumentTypeError('地形frame重複或越界')
    return set(frames)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'assets', 'original-pack', 'modern-pack', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    parser.add_argument('--town-save', type=Path, help='既有JSON來源槽；原生槽轉普通F5時不需指定')
    facility = parser.add_mutually_exclusive_group()
    facility.add_argument('--hotel', action='store_true', help='只抽驗旅館固定文字、四槽及保存讀取')
    facility.add_argument('--church', action='store_true', help='抽驗教會即時文字、四語名冊、取消與保存')
    facility.add_argument('--battle-status', action='store_true', help='以正常標題LOAD抽驗首章戰場狀態面板與底圖')
    facility.add_argument('--status', action='store_true', help='加驗教會角色狀態與命令穩定面板')
    facility.add_argument('--items', action='store_true', help='抽驗商店裝備物品面板、選取、F2與保存')
    facility.add_argument('--preparation', action='store_true', help='抽驗標題 LOAD、整備穩定文字及 F2')
    facility.add_argument('--gold-roll', action='store_true', help='抽驗正常買賣的 HD 金幣滾動及保存')
    parser.add_argument('--native-save', type=Path, help='整備抽樣的既有原生槽，複製後才可寫入')
    parser.add_argument('--transition-frames', action='store_true', help='另以唯讀觀測擷取整備問句轉場PNG')
    parser.add_argument('--panel-transitions', action='store_true', help='以實際移動文字區抽驗狀態／物品轉場')
    parser.add_argument('--gold-roll-theme', choices=['hd','faithful'], default='hd', help='金幣買賣的主題對照')
    parser.add_argument('--world', action='store_true', help='加驗正式HD世界、普通走行、指令環、取消與物理戰鬥返回')
    parser.add_argument('--battle-hd', action='store_true', help='要求兩次普通攻擊確實使用可選HD戰鬥繪製層；需--world')
    parser.add_argument('--battle-art', action='store_true', help='另擷取兩次普通攻擊的有界PNG序列供實際圖源核對；需--battle-hd')
    parser.add_argument('--battle-node', default='battle_ch01', help='來源槽的正式戰場節點，預設首章')
    parser.add_argument('--battle-only', action='store_true', help='抽驗普通攻防、F2及保存冷讀，略過已另驗的狀態／命令字型；需--battle-hd')
    parser.add_argument('--capture-native-battle-save', action='store_true', help='正常原生標題LOAD、出發及F5建立JSON來源槽；需--native-save與--battle-status')
    parser.add_argument('--native-entry-theme', choices=['hd','faithful'], default='hd', help='原生來源擷取的起始主題；第23章獨立整備目前明示使用faithful')
    parser.add_argument('--death-event', type=int, choices=[53], help='第23章普通進軍擊倒隊長，唯讀抽樣共用死亡HD；需--town-save')
    parser.add_argument('--death-kind', choices=['script','action'], default='script', help='抽樣章節事件或普通擊殺的共用死亡；需--death-event與--world固定seed')
    parser.add_argument('--death-commander-hd', action='store_true',
                        help='同一第23章正常路徑加驗隊長待機及116故事HD頭像；不改來源或規則')
    parser.add_argument('--revive-before-departure', type=int, choices=[1],
                        help='第二／三章原生來源出發前，經正常教會服務復活哈諾；需--capture-native-battle-save')
    parser.add_argument('--buy-bekwei-bow', action='store_true',
                        help='原生第九章出發前，普通購買黑暗弓並立即裝備貝克威')
    parser.add_argument('--battle-command', type=int, choices=[0], help='改以正常指令0抽樣；原地候選再由正式目標格驗證，需--battle-only')
    parser.add_argument('--battle-repetitions', type=int, choices=[1,2], default=2,
                        help='普通攻防場數；一場模式仍完整自然返回與保存冷讀，需--battle-only')
    parser.add_argument('--battle-art-samples', type=int, choices=[32,64], default=64,
                        help='每場實際演出PNG取樣上限；完整逐幀Draw另驗')
    parser.add_argument('--startup-timeout', type=int, choices=[60,120,180], default=60,
                        help='明示等待真實視窗的上限，主流程與新程序冷讀共用')
    parser.add_argument('--battle-target-figure', type=int,
                        help='只以普通鍵盤選擇指定敵方圖組；省略時維持最近目標')
    parser.add_argument('--battle-target-row-max', type=int,
                        help='只篩選目前Y不大於此值的同圖組目標，不改遊戲狀態')
    parser.add_argument('--battle-min-approach-rounds', type=int, choices=[0,1,2,3,4,5,6], default=0,
                        help='普通進軍／待機至少幾次END後才出手，不改遊戲狀態')
    parser.add_argument('--battle-own-index', type=int, default=2,
                        help='普通攻擊時以Z選取的在場我方序號，預設2為既有亞雷斯；需--world')
    parser.add_argument('--battle-approach-rounds', type=int, choices=[0,1,2,3,4,5,6], default=0,
                        help='沒有可達攻擊位置時，明示有界普通END次數；需--world')
    parser.add_argument('--gaia', action='store_true', help='首章普通選取蓋亞、北移／取消、F2與新程序LOAD；需--battle-status')
    parser.add_argument('--terrain', action='store_true', help='普通方向鍵移鏡頭至首章房屋，抽驗HD地形與F2；需--battle-status')
    parser.add_argument('--terrain-occlusion', action='store_true', help='最多六個普通回合抵達屋後，抽驗前景；需--terrain與--world')
    parser.add_argument('--terrain-roof-view', action='store_true', help='另以普通游標取景褐頂物件全部八個底圖；需--terrain')
    parser.add_argument('--terrain-props-view', action='store_true', help='普通游標取景綠色與褐色物件九個底圖；需--terrain')
    parser.add_argument('--terrain-frames', type=terrain_frame_ids, default=set(), help='明示本次取景必須包含的frame；需--terrain與--terrain-label')
    parser.add_argument('--terrain-label', help='本次取景的ASCII標籤；需--terrain-frames')
    parser.add_argument('--timed-dialogue', action='store_true', help='加驗計時對話開收框、穩定HD文字與F2；需--world與明示章前EXP校準槽')
    parser.add_argument('--battle-advance', action='store_true', help='無射程目標時沿正式可達集合前進並待機，再正常END')
    parser.add_argument('--battle-advance-column', type=int, help='只限定正常END前的進軍縱列，最終攻擊仍沿正式可達集合')
    parser.add_argument('--faithful-approach', action='store_true', help='正常END期間以F2使用忠實模式，回玩家回合再切HD')
    parser.add_argument('--battle-end-art', action='store_true', help='正常END期間有界擷取明示人物的HD圖層')
    parser.add_argument('--battle-end-commands', type=terrain_frame_ids, default=set(),
                        help='另擷取正常END實際指令0／4／5的圖源，與物理收據分列')
    parser.add_argument('--battle-end-samples-per-frame', type=int, choices=[1,2], default=1,
                        help='每個END圖源幀的有界視窗取樣數；第二張另存，不由job推定已呈現')
    parser.add_argument('--battle-end-figure',type=int,default=2)
    parser.add_argument('--battle-end-figures',type=terrain_frame_ids,default=set(),
                        help='同一普通END擷取多個明示figure，例如133,134；省略時維持單數參數')
    args = parser.parse_args()
    assert not args.death_commander_hd or (args.death_event==53 and args.world and args.battle_hd)
    assert args.revive_before_departure is None or args.capture_native_battle_save
    assert not args.buy_bekwei_bow or (args.capture_native_battle_save
           and args.battle_node=='battle_ch09' and args.revive_before_departure is None)
    assert args.battle_command is None or (args.battle_only and args.battle_hd and not args.battle_end_art and not args.timed_dialogue and not args.battle_advance)
    action_label='command0' if args.battle_command==0 else 'physical'
    assert args.battle_repetitions==2 or (args.battle_only and not args.timed_dialogue)
    assert 0<=args.battle_end_figure<=135
    end_figures=sorted(args.battle_end_figures or {args.battle_end_figure})
    assert all(0<=figure<=135 for figure in end_figures)
    assert not args.battle_end_figures or args.battle_end_art
    assert not args.battle_end_art or (args.battle_only and args.battle_hd and args.battle_approach_rounds>0 and not args.faithful_approach)
    assert not args.battle_end_commands or (args.battle_end_art and args.battle_end_commands <= {0,4,5})
    assert args.battle_end_samples_per_frame==1 or args.battle_end_art
    assert not (args.battle_advance or args.faithful_approach) or args.battle_only
    if not re.fullmatch(r'battle_ch(?:0[1-9]|[12][0-9]|30)', args.battle_node):
        parser.error('戰場節點須為battle_ch01..battle_ch30')
    if args.capture_native_battle_save:
        assert args.native_save and args.battle_status and not args.world
    else:
        assert args.town_save and args.town_save.is_file()
    assert not args.battle_only or args.battle_hd
    assert not (args.gaia or args.terrain) or args.battle_node=='battle_ch01'
    if bool(args.terrain_frames)!=bool(args.terrain_label):
        parser.error('--terrain-frames與--terrain-label須同時指定')
    if args.terrain_label and not re.fullmatch(r'[a-z][a-z0-9-]{0,39}',args.terrain_label):
        parser.error('取景標籤只能使用小寫ASCII字母、數字與連字號')
    tool_source_sha256=digest(Path(__file__))
    assert not args.world or args.battle_status
    assert not args.battle_hd or args.world
    assert not args.battle_art or args.battle_hd
    assert 0 <= args.battle_own_index < 128
    assert 0 <= args.battle_min_approach_rounds <= args.battle_approach_rounds
    assert args.battle_target_figure is None or (args.battle_only and 0<=args.battle_target_figure<=135)
    assert args.battle_target_row_max is None or (args.battle_target_figure is not None and 0<=args.battle_target_row_max<=65535)
    assert args.battle_advance_column is None or (args.battle_advance and 0<=args.battle_advance_column<=65535)
    assert args.battle_own_index == 2 or args.world
    assert not args.battle_approach_rounds or args.world
    battle_approach_steps=[]
    assert not args.gaia or args.battle_status and not args.world and not args.terrain
    assert not args.terrain or args.battle_status
    assert not args.terrain_occlusion or args.terrain and args.world
    assert not args.terrain_roof_view or args.terrain
    assert not args.terrain_props_view or args.terrain and not args.terrain_roof_view
    assert not args.terrain_frames or args.terrain and not (args.terrain_roof_view or args.terrain_props_view)
    assert not args.timed_dialogue or args.world
    assert not args.transition_frames or args.preparation
    assert not args.panel_transitions or args.status or args.items
    assert Path('/.dockerenv').is_file() and args.binary.is_file()
    assert (args.original_pack/'manifest.json').is_file()
    if not args.capture_native_battle_save:
        assert json.loads(args.town_save.read_text())['node'] == (args.battle_node if args.battle_status else 'hotel_ch02')
    assert not args.out.exists()
    args.out.mkdir()
    audit = args.out/'input-audit.jsonl'
    saved = args.out/'user-data/fd2_re/fd2_save.json'
    saved.parent.mkdir(parents=True)
    if not args.capture_native_battle_save:
        saved.write_bytes(args.town_save.read_bytes())
    env = dict(os.environ, DISPLAY=':97', XDG_RUNTIME_DIR='/tmp/fd2-hd-font-pulse',
               XDG_DATA_HOME=str(args.out/'user-data'), XDG_CONFIG_HOME=str(args.out/'config'),
               XDG_CACHE_HOME=str(args.out/'cache'), LIBGL_ALWAYS_SOFTWARE='1',
               GALLIUM_DRIVER='llvmpipe', GOMAXPROCS='2', LP_NUM_THREADS='2',
               PULSE_SERVER='unix:/tmp/fd2-hd-font-pulse/pulse/native',
               FD2_THEME='modern-handpainted-a', FD2_LOCALE='zh-Hant',
               FD2_MODERN_THEME_PACK=str(args.modern_pack), FD2_ASSET_PACK=str(args.original_pack),
               FD2_INPUT_AUDIT_LOG=str(audit), FD2_INPUT_AUDIT_INTERVAL='2')
    for key in ['FD2_TITLE','FD2_CAMPAIGN','FD2_CAMP_NODE','FD2_SHOT','FD2_SHOT_FRAME',
                'FD2_NATIVE_SAVE','FD2_SAVE_CURRENT','FD2_ENDING_PREFIX']:
        env.pop(key,None)
    native_saved = None
    native_source_hash = None
    if args.native_save:
        assert (args.preparation or args.capture_native_battle_save) and args.native_save.is_file()
        native_source_hash = digest(args.native_save)
        native_saved = args.out/'FD2.SAV'
        native_saved.write_bytes(args.native_save.read_bytes())
        env['FD2_NATIVE_SAVE'] = str(native_saved)
    Path(env['XDG_RUNTIME_DIR']).mkdir(mode=0o700)
    Path(env['XDG_CONFIG_HOME']).mkdir()
    Path(env['XDG_CACHE_HOME']).mkdir()
    alsa=args.out/'asoundrc'
    alsa.write_text('pcm.!default { type pulse }\nctl.!default { type pulse }\n')
    env['ALSA_CONFIG_PATH']=str(alsa)
    processes,streams,shots,operations=[],[],{},[]
    battle_hd_scenes=[]
    battle_art_samples=[]
    battle_end_art_samples=[]
    battle_end_command_samples=[]
    battle_post_dialogue_cleanup=[]
    battle_attack_choices=[]
    started=time.monotonic()
    window=None
    audit_epoch=0

    transition_stop=threading.Event()
    transition_thread=None
    transition_shots={}
    transition_rejected=[]
    transition_errors=[]
    transition_gpu_validated=False
    gold_stop=threading.Event()
    gold_thread=None
    gold_shots={}
    gold_errors=[]
    panel_probe=None
    world_stop=threading.Event()
    world_thread=None
    world_shots={}
    world_rejected=[]
    timed_probe=None
    timed_result=None
    if args.world:env['FD2_SEED']='1'

    def capture_world_walk():
        seen=set()
        deadline=time.monotonic()+12
        while not world_stop.is_set() and time.monotonic()<deadline and len(world_shots)<4:
            before=latest(audit)
            if not before.get('walking') or not before.get('hd_world'):
                world_stop.wait(.005);continue
            token=tuple((u.get('work_offset'),u.get('frame')) for u in before.get('hd_world_units',[]))
            if token in seen:world_stop.wait(.005);continue
            seen.add(token)
            path=args.out/('world-walking-'+str(len(seen))+'.png')
            command('import','-window',window,str(path))
            after=latest(audit)
            record={'file':path.name,'sha256':digest(path),'before':before,'after':after,
                    'kind':'actual_gpu_window'}
            if after.get('walking') and after.get('hd_world'):world_shots[path.name]=record
            else:world_rejected.append(record)

    def capture_gold_roll():
        seen=set()
        try:
            while not gold_stop.is_set() and len(seen)<16:
                before=latest(audit)
                token=(before.get('gold_roll_job'),before.get('gold_roll_index'))
                if not token[0] or token in seen:
                    gold_stop.wait(.005);continue
                seen.add(token)
                path=args.out/('gold-roll-'+str(len(seen))+'.png')
                roi='288x54+96+588' if before.get('hd_theme') else '192x36+64+392'
                command('import','-window',window,'-crop',roi,'+repage',str(path))
                gold_shots[path.name]={'before':before,'after':latest(audit),'sha256':digest(path),
                                       'window_crop':roi,'kind':'actual_gpu_window_region'}
        except Exception as error:gold_errors.append(str(error))

    def finish_gold_roll():
        gold_stop.set()
        if gold_thread and gold_thread.ident is not None:
            gold_thread.join(timeout=20)
            if gold_thread.is_alive():gold_errors.append('金幣擷取尚未結束')

    def command(*items):
        return subprocess.check_output(items,env=env,text=True,stderr=subprocess.DEVNULL)

    def start(items,name,cwd=None):
        stream=(args.out/name).open('w');streams.append(stream)
        proc=subprocess.Popen(items,env=env,cwd=cwd,stdout=stream,stderr=stream)
        processes.append(proc);return proc

    def wait(predicate,timeout=60):
        deadline=time.monotonic()+timeout
        while time.monotonic()<deadline:
            state=latest(audit)
            if state.get('error'):raise RuntimeError(state['error'])
            if predicate(state):return state
            limit=2400 if args.battle_end_art else 1800 if args.battle_advance else 1200 if args.terrain_occlusion or args.battle_approach_rounds or args.revive_before_departure is not None else 660
            if time.monotonic()-started>limit:raise TimeoutError('整輪逾時')
            time.sleep(.1)
        raise TimeoutError(str(latest(audit)))

    def key(name):
        before=latest(audit)
        mapped={'Return':'Enter','Left':'ArrowLeft','Right':'ArrowRight',
                'Up':'ArrowUp','Down':'ArrowDown','z':'Z'}.get(name,name)
        command('xdotool','windowfocus',window)
        command('xdotool','keydown',name)
        ack=None
        try:
            deadline=time.monotonic()+60
            while time.monotonic()<deadline:
                if audit.exists():
                    with audit.open('rb') as f:
                        f.seek(max(0,audit.stat().st_size-131072));rows=f.read().splitlines()
                    for row in reversed(rows):
                        try:state=json.loads(row)
                        except ValueError:continue
                        if state.get('frame',0)>before.get('frame',0) and mapped in (state.get('keys') or []):
                            ack=state;break
                if ack:break
                time.sleep(.005)
        finally:command('xdotool','keyup',name)
        if not ack:raise TimeoutError('按鍵無收據 '+name)
        deadline=time.monotonic()+60
        release_ack=None
        while time.monotonic()<deadline:
            with audit.open('rb') as f:
                f.seek(max(0,audit.stat().st_size-131072));rows=f.read().splitlines()
            for row in reversed(rows):
                try:state=json.loads(row)
                except ValueError:continue
                if state.get('frame',0)>ack['frame'] and mapped in (state.get('released_keys') or []):
                    release_ack=state;break
            if release_ack:break
            time.sleep(.005)
        if not release_ack:raise TimeoutError('放鍵無收據 '+name)
        released_frame=max(ack['frame'],latest(audit).get('frame',0))
        wait(lambda s:s.get('frame',0)>=released_frame+2)
        operations.append({'key':name,'ack':ack,'release_ack':release_ack,'process_epoch':audit_epoch})
        return ack

    def shot(name):
        # Wait for subsequent GPU presentation, independently of the input ACK.
        if (args.preparation or args.battle_status) and 'resized' in name:
            before = latest(audit).get('frame', 0)
            command('xdotool','windowfocus',window)
            wait(lambda s:s.get('frame',0)>=before+20, timeout=120)
        time.sleep(1)
        path=args.out/(name+'.png')
        gpu_before=latest(audit)
        command('import','-window',window,str(path))
        shots[path.name]={'sha256':digest(path),'state':latest(audit),
                          'gpu_before':gpu_before,
                          'window_geometry':command('xdotool','getwindowgeometry','--shell',window)}
        print('CAPTURE '+path.name,flush=True)

    def restart_title_load(node, gold, name):
        nonlocal game, window, audit_epoch
        game.terminate();game.wait(timeout=15)
        previous=args.out/'input-audit-first-process.jsonl'
        assert not previous.exists()
        audit.rename(previous)
        audit_epoch=1
        game=start([str(args.binary)],'game-cold-load.log',args.assets)
        window=None
        for _ in range(args.startup_timeout*10):
            if game.poll() is not None:raise RuntimeError('冷讀程序提前退出')
            try:window=command('xdotool','search','--name','炎龍騎士團').splitlines()[0];break
            except (subprocess.CalledProcessError,IndexError):time.sleep(.1)
        assert window
        command('xdotool','windowmove',window,'0','0')
        wait(lambda s:bool(s.get('title_phase')))
        for _ in range(160):
            if latest(audit).get('title_phase')=='menu':break
            key('Escape')
        else:raise TimeoutError('冷讀標題未就緒')
        key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots')
        key('Return');wait(lambda s:not s.get('title_phase') and s.get('node')==node and s.get('gold')==gold)
        if args.battle_status:
            loaded_frame=latest(audit)['frame']
            wait(lambda s:s.get('frame',0)>=loaded_frame+20)
            for _ in range(120):
                state=latest(audit)
                if not state.get('dialogues') and not state.get('banner_t'):break
                key('Return')
            else:raise TimeoutError('冷讀首章對話未結束')
        shot(name)

    def transition_eligible(state):
        frame=state.get('preparation_transition_frame',-1)
        start=state.get('preparation_transition_text_start',-1)
        return (state.get('hd_theme') is True and state.get('preparation_ui_busy') is True
                and state.get('hd_text_runs',0)>0 and start>=0 and start<frame<start+4
                and not state.get('preparation_transition_required'))

    def capture_transitions():
        seen=set()
        try:
            while not transition_stop.is_set() and len(seen)<32:
                before=latest(audit)
                token=(before.get('preparation_transition_job'),before.get('preparation_transition_frame'))
                if not transition_eligible(before) or token in seen:
                    transition_stop.wait(.01);continue
                seen.add(token)
                path=args.out/('preparation-transition-'+str(len(seen))+'.png')
                subprocess.run(['import','-window',window,str(path)],env=env,
                               stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,check=True,timeout=15)
                after=latest(audit)
                valid=(transition_eligible(after)
                       and before.get('preparation_transition_job')==after.get('preparation_transition_job')
                       and before.get('preparation_transition_closing')==after.get('preparation_transition_closing')
                       and before['preparation_transition_frame']<=after['preparation_transition_frame'])
                meta={'sha256':digest(path),'before':before,'after':after,'accepted':valid}
                if valid:
                    meta['anchor']=('preparation-final-question.png' if before.get('preparation_confirm')
                                    else ('preparation-record-question.png' if args.native_save
                                          else 'preparation-departure-question.png'))
                    transition_shots[path.name]=meta
                    print('TRANSITION '+path.name,flush=True)
                else:
                    transition_rejected.append({'file':path.name,**meta})
        except Exception as error:
            transition_errors.append(str(error))

    def finish_transitions(raise_errors=True):
        transition_stop.set()
        if transition_thread and transition_thread.ident is not None:
            transition_thread.join(timeout=20)
            if transition_thread.is_alive():transition_errors.append('轉場擷取尚未結束')
        if transition_errors and raise_errors:raise RuntimeError(str(transition_errors))

    def validate_transition_samples():
        nonlocal transition_gpu_validated
        # CPU state brackets locate candidates; the window can still present
        # an earlier frame. Require actual HD ink and moving choice geometry.
        def region(path,geometry):
            return subprocess.check_output(['convert',str(path),'-crop',geometry,'+repage',
                                            '-depth','8','rgb:-'],env=env,timeout=15)
        def gold_bounds(path):
            raw=region(path,'450x144+1320+996')
            columns=[(i//3)%450 for i in range(0,len(raw),3)
                     if raw[i]>=200 and 100<=raw[i+1]<220 and raw[i+2]<80]
            return (1320+min(columns),1320+max(columns)) if columns else None
        candidates={**transition_shots,**{m['file']:m for m in transition_rejected}}
        transition_shots.clear();transition_rejected.clear()
        for name,meta in candidates.items():
            path=args.out/name
            assert digest(path)==meta['sha256'],'轉場候選PNG被改寫'
            before=meta['before']
            anchor_name=('preparation-final-question.png' if before.get('preparation_confirm')
                         else ('preparation-record-question.png' if args.native_save
                               else 'preparation-departure-question.png'))
            anchor=args.out/anchor_name
            reference=region(anchor,'1302x102+570+714')
            actual=region(path,'1302x102+570+714')
            ink_equal=len(reference)==1302*102*3 and actual==reference
            bounds,stable_bounds=gold_bounds(path),gold_bounds(anchor)
            moving=bool(bounds and stable_bounds and bounds[0]>stable_bounds[0] and bounds[1]<stable_bounds[1])
            meta.update(cpu_phase_contained=meta['accepted'],accepted=ink_equal and moving,
                        anchor=anchor_name,hd_text_region_equal=ink_equal,
                        choice_gold_bounds=bounds,stable_choice_gold_bounds=stable_bounds,
                        moving_choices_verified=moving,exact_frame_claimed=False)
            if meta['accepted']:
                transition_shots[name]=meta
            else:
                transition_rejected.append({'file':name,**meta})
        transition_gpu_validated=True

    def settled_dialogue():
        command('xdotool','windowfocus',window)
        wait(lambda s:s.get('dialogue_phase')==0 and s.get('dialogue_progress',-1)>=0)
        previous=None
        for _ in range(45):
            time.sleep(1);current=latest(audit)
            if previous and current.get('dialogue_progress',0)>0 and current.get('dialogue_progress')==previous.get('dialogue_progress'):
                return current
            previous=current
        raise TimeoutError('逐字顯示未穩定')

    try:
        xvfb=start(['Xvfb',':97','-screen','0','1920x1200x24','-nolisten','tcp'],'xvfb.log')
        for _ in range(100):
            try:command('xdotool','getdisplaygeometry');break
            except subprocess.CalledProcessError:time.sleep(.1)
            if xvfb.poll() is not None:raise RuntimeError('Xvfb啟動失敗')
        else:raise TimeoutError('Xvfb尚未就緒')
        pulse=start(['pulseaudio','--daemonize=no','--exit-idle-time=-1','--use-pid-file=no'],'pulse.log')
        for _ in range(100):
            try:command('pactl','info');break
            except subprocess.CalledProcessError:time.sleep(.1)
            if pulse.poll() is not None:raise RuntimeError('PulseAudio啟動失敗')
        else:raise TimeoutError('PulseAudio尚未就緒')
        command('pactl','load-module','module-null-sink','sink_name=fd2_hd_fonts','rate=44100','channels=2')
        command('pactl','set-default-sink','fd2_hd_fonts')
        game=start([str(args.binary)],'game.log',args.assets)
        for _ in range(args.startup_timeout*10):
            if game.poll() is not None:raise RuntimeError('遊戲提前退出')
            try:window=command('xdotool','search','--name','炎龍騎士團').splitlines()[0];break
            except (subprocess.CalledProcessError,IndexError):time.sleep(.1)
        assert window
        command('xdotool','windowmove',window,'0','0');command('xdotool','windowfocus',window)
        wait(lambda s:bool(s.get('title_phase')))
        for _ in range(160):
            if latest(audit).get('title_phase')=='menu':break
            key('Escape')
        else:raise TimeoutError('標題未就緒')
        if args.death_event:
            run_ch23_death_event(args,audit,key,wait,shot,saved,command,window,
                                restart_title_load,operations,shots,tool_source_sha256)
            return
        if args.capture_native_battle_save:
            if args.battle_node=='battle_ch23':
                capture_standalone_ch23_source(args,audit,key,wait,shot,saved,native_saved,native_source_hash,
                                              tool_source_sha256,operations,shots)
                return
            town_node=args.battle_node.replace('battle_', 'town_', 1)
            prep_node=args.battle_node.replace('battle_', 'preparation_', 1)
            key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots')
            shot('native-title-load');key('Return')
            wait(lambda s:not s.get('title_phase') and s.get('node')==town_node)
            shot('native-town-loaded')
            # 只接受已核對的城鎮選項與出發路由，不把任意章節套入同一鍵序。
            assert args.battle_node in ('battle_ch02','battle_ch03','battle_ch08','battle_ch09','battle_ch11','battle_ch12'), '此原生出發鍵序只支援已明示的第二／三／八／九／十一／十二章'
            nodes=json.loads((args.assets/'assets/scenarios/campaign_full.json').read_text())['nodes']
            suffix=args.battle_node.removeprefix('battle_')
            town=nodes[town_node]
            assert town['native_town_variant']=={'battle_ch02':0,'battle_ch03':2,'battle_ch08':0,'battle_ch09':0,'battle_ch11':2,'battle_ch12':1}[args.battle_node] and [town['options'][i]['to'] for i in (0,2,4)]==[
                'hotel_'+suffix,prep_node,'church_'+suffix], '原生城鎮出發選項不同'
            story_node=nodes[prep_node]['next']
            assert nodes[story_node]['type']=='cutscene' and nodes[story_node]['next']==args.battle_node
            revival = None
            purchase = None
            if args.buy_bekwei_bow:
                before=latest(audit);key('F5');wait(lambda s:s.get('frame',0)>before['frame'])
                baseline=json.loads(saved.read_text());assert baseline['node']==town_node
                (args.out/'purchase-before.json').write_bytes(saved.read_bytes())
                unit=baseline['party_roster']['13']
                assert unit['NativeRecordClass']==4 and unit['NativeIdentity']==13
                assert unit['Inventory']==[0,129] and unit['Equipped']==[True,True]
                goods=nodes['shop_ch09_weapon']['goods']
                good=next(g for g in goods if g['id']==46)
                assert good['price']==1800 and baseline['gold']>=good['price']
                item=next(i for i in json.loads((args.assets/'assets/data/item.json').read_text()) if i['id']==46)
                assert item['type']==5 and item['range']==[2,4]
                types={row['cls']:row['types'] for row in json.loads(
                    (args.assets/'assets/data/class_equip_types.json').read_text())}
                recipients=[ident for ident in baseline['party_join_order']
                            if item['type'] in types[baseline['party_roster'][str(ident)]['NativeRecordClass']]]
                assert 13 in recipients
                def shop(mode):
                    return wait(lambda s:s.get('node')=='shop_ch09_weapon'
                                and s.get('shop_mode')==mode and not s.get('shop_ui_busy'))
                key('Left');key('Return');shop('menu');shot('native-bow-shop')
                key('Return');shop('purchase')
                # 商品是雙欄列表，Right逐項，Down跨兩項。
                for _ in range(goods.index(good)):key('Right')
                shot('native-bow-selected');key('Return');shop('confirm')
                key('Return');shop('recipient_equipment')
                for _ in range(recipients.index(13)):key('Down')
                shot('native-bow-recipient');key('Return');shop('equip_confirm')
                shot('native-bow-equip-confirm');key('Return');shop('purchase')
                assert latest(audit)['gold']==baseline['gold']-good['price']
                shot('native-bow-purchased')
                key('Escape');shop('menu');key('Escape');wait(lambda s:s.get('node')==town_node)
                key('F5');after=json.loads(saved.read_text())
                (args.out/'purchase-after.json').write_bytes(saved.read_bytes())
                equipped=after['party_roster']['13']
                assert equipped['Inventory']==[0,129,46] and equipped['Equipped']==[False,True,True]
                assert equipped['InventorySlots']==[0,129,46,255,255,255,0,0]
                assert equipped['NativeInventoryFlags']==[0,64,64,128,128,128,128,128]
                assert [equipped['AtkMin'],equipped['AtkMax']]==[2,4]
                changed=[k for k in set(unit)|set(equipped) if unit.get(k)!=equipped.get(k)]
                assert set(changed)<=set(['AP','HIT','AtkMin','AtkMax','Inventory','Equipped',
                                          'InventorySlots','NativeInventoryFlags']), changed
                assert all(after['party_roster'][k]==v for k,v in baseline['party_roster'].items() if k!='13')
                assert after['party_join_order']==baseline['party_join_order']
                assert after['gold']==baseline['gold']-1800
                purchase={'identity':13,'item':46,'price':1800,'eligible_recipients':recipients,
                          'gold_before':baseline['gold'],'gold_after':after['gold'],
                          'range_before':[unit['AtkMin'],unit['AtkMax']],'range_after':[2,4],
                          'changed_unit_fields':sorted(changed),'other_roster_unchanged':True,
                          'hp_unchanged':equipped['HP']==unit['HP'],
                          'before_sha256':digest(args.out/'purchase-before.json'),
                          'after_sha256':digest(args.out/'purchase-after.json'),'state_injection':False}
                assert purchase['hp_unchanged']
                key('Right') # 武器店回城仍在選項1，恢復旅館選項0再沿既有出發鍵序。
            if args.revive_before_departure is not None:
                before=latest(audit);key('F5');wait(lambda s:s.get('frame',0)>before['frame'])
                assert saved.is_file(), '城鎮F5未建立名冊來源'
                baseline=json.loads(saved.read_text());assert baseline['node']==town_node
                (args.out/'revive-before.json').write_bytes(saved.read_bytes())
                ident=args.revive_before_departure;unit=baseline['party_roster'][str(ident)]
                candidates=[i for i in baseline['party_join_order']
                            if baseline['party_roster'][str(i)].get('has_native_record_byte5')
                            and baseline['party_roster'][str(i)].get('native_record_byte5',0)&1]
                assert ident in candidates and unit['HP']==0 and unit['MaxHP']>0
                assert unit['HasNativeRecordClass']
                rates=json.loads((args.assets/'assets/data/revive_fee_rates.json').read_text())['rates']
                raw_class=unit['NativeRecordClass'];assert 0<=raw_class<len(rates)
                fee=rates[raw_class]*unit['Lv'];assert 0<=fee<=0xffff and baseline['gold']>=fee
                for _ in range(4):key('Left')
                key('Return')
                def church(mode):
                    return wait(lambda s:s.get('node')==args.battle_node.replace('battle_','church_',1) and s.get('church_mode')==mode
                                and not s.get('church_ui_busy'))
                church('menu');shot('native-revive-church')
                key('Right');key('Right');key('Return');church('revive')
                for _ in range(candidates.index(ident)):key('Down')
                church('revive');shot('native-revive-selected')
                key('Return');church('revive_confirm');shot('native-revive-confirm')
                key('Return');church('revive')
                assert latest(audit)['gold']==baseline['gold']-fee
                shot('native-revive-complete')
                key('Escape');church('menu');key('Escape');wait(lambda s:s.get('node')==town_node)
                key('F5');after=json.loads(saved.read_text())
                (args.out/'revive-after.json').write_bytes(saved.read_bytes())
                expected=dict(unit,HP=unit['MaxHP'],OnField=True)
                restored=unit['native_record_byte5']&~1
                if restored:expected['native_record_byte5']=restored
                else:expected.pop('native_record_byte5',None) # Go json omitempty
                assert after['party_roster'][str(ident)]==expected, '復活寫回不符既有正式契約'
                assert after['party_join_order']==baseline['party_join_order'] and after['gold']==baseline['gold']-fee
                assert all(after['party_roster'][k]==v for k,v in baseline['party_roster'].items() if k!=str(ident))
                revival={'identity':ident,'candidates':candidates,'selection':candidates.index(ident),'raw_class':raw_class,
                         'fee':fee,'gold_before':baseline['gold'],'gold_after':after['gold'],
                         'hp_before':unit['HP'],'hp_after':expected['HP'],
                         'before_sha256':digest(args.out/'revive-before.json'),
                         'after_sha256':digest(args.out/'revive-after.json'),'state_injection':False}
                key('Right');key('Right')
            else:
                key('Left');key('Left')
            key('Return')
            wait(lambda s:s.get('node')==prep_node and not s.get('preparation_ui_busy'))
            shot('native-departure-prompt');key('Return')
            # 章首包含長距離鏡頭移動；軟體繪圖時仍須走完原有虛擬幀。
            deadline=time.monotonic()+480
            while time.monotonic()<deadline:
                state=latest(audit)
                if state.get('error'):raise RuntimeError(state['error'])
                if state.get('node')==args.battle_node:break
                assert state.get('node') in (prep_node,story_node), '原生出發到達非預期節點'
                # story即使對話已清空，也由普通確認鍵呼叫advanceStoryNode。
                # handleNativeStoryInput在fade／storyWalk期間自行拒絕提前推進。
                if state.get('node')==story_node or state.get('dialogues'):
                    key('Return')
                else:time.sleep(.05)
            else:raise TimeoutError('原生出發章首劇情尚未結束')
            for _ in range(120):
                state=latest(audit)
                if not state.get('dialogues') and not state.get('banner_t') and not state.get('battle_event'):break
                key('Return')
            else:raise TimeoutError('原生LOAD章首對話尚未結束')
            before=latest(audit);key('F5');wait(lambda s:s.get('frame',0)>before['frame'])
            assert saved.is_file() and json.loads(saved.read_text())['node']==args.battle_node
            assert digest(args.native_save)==native_source_hash, '唯讀原始槽被修改'
            if purchase is not None:
                actor=next(u for u in latest(audit)['units'] if u.get('native_identity')==13)
                assert actor['battle_fig']==13 and actor['on_field']
                assert [actor['attack_min'],actor['attack_max']]==[2,4]
                purchase['battle_range']=[actor['attack_min'],actor['attack_max']]
                purchase['battle_hp']=actor['hp']
            shot('native-battle-f5')
            report={'schema':'fd2-native-battle-source-gui-v1','status':'PASS',
                    'binary_sha256':digest(args.binary),'gui_tool_sha256':tool_source_sha256,
                    'native_source_sha256':native_source_hash,'saved_sha256':digest(saved),
                    'revival':revival,'purchase':purchase,
                    'native_writable_copy_after_f5_sha256':digest(native_saved),
                    'saved_file':str(saved),'node':args.battle_node,'state_injection':False,
                    'screenshots':shots,'operations':operations,'recording_created':False,
                    'scope':'正常原生標題LOAD、城鎮出發、章首及F5。來源槽祖先依原manifest另揭露，不升格自然難度或整章E2。'}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        if args.battle_status:
            key('Down');key('Return')
            wait(lambda s:s.get('title_phase')=='loadslots');shot('battle-title-load')
            key('Return')
            wait(lambda s:not s.get('title_phase') and s.get('node')==args.battle_node)
            for _ in range(120):
                state=latest(audit)
                if not state.get('dialogues') and not state.get('banner_t'):break
                key('Return')
            else:raise TimeoutError('首章LOAD對話未結束')
            wait(lambda s:not s.get('dialogues') and not s.get('banner_t'))
            shot('battle-hd-field')
            if args.gaia:
                roster=json.loads(args.town_save.read_text())['party_roster']['30']
                assert roster['NativeIdentity']==30 and roster['Fig']==30 and roster['BattleFig']==30
                gaia=sample_gaia(key,wait,shot,lambda:latest(audit))
                assert saved.read_bytes()==args.town_save.read_bytes()
                restart_title_load(args.battle_node,latest(audit)['gold'],'gaia-cold-title-load')
                cold=focus_gaia(key,wait,shot,lambda:latest(audit),'gaia-hd-cold-focused')
                assert cold['units']==gaia['baseline']['units']
                assert saved.read_bytes()==args.town_save.read_bytes()
                assert digest(Path(__file__))==tool_source_sha256
                report={'schema':'fd2-hd-gaia-focus-gui-v1','status':'PASS',
                        'catalog_sha256':digest(args.assets/'assets/themes/modern/catalog.json'),
                        'binary_sha256':digest(args.binary),'gui_tool_sha256':tool_source_sha256,
                        'focus_tool_sha256':digest(Path(__file__).with_name('verify_fd2_gaia_focus.py')),
                        'source_save_sha256':digest(args.town_save),'saved_sha256':digest(saved),
                        'source_save_bytes_equal':True,'new_process_title_load':True,
                        'world_runtime_checked':True,'screenshots':shots,'operations':operations,
                        'gaia':gaia,'cold':cold,'recording_created':False,
                        'scope':'正常標題LOAD既有普通START/F5首章槽；Z選蓋亞、北移三格／取消、F2及新程序LOAD。四方向十二幀另作圖源審查，非原版對拍。'}
                (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
                return
            if args.terrain:
                wait(lambda s:not s.get('player_focus') and s.get('hd_world') is True)
                before_pan=latest(audit)
                def pan_to(destination):
                    for _ in range(90):
                        current=latest(audit)['cursor']
                        if current==destination:return
                        if current[0]!=destination[0]:direction='Right' if current[0]<destination[0] else 'Left'
                        else:direction='Down' if current[1]<destination[1] else 'Up'
                        key(direction)
                    raise AssertionError(('普通游標未抵達房屋觀測位置',latest(audit)['cursor'],destination))
                view_y=8 if args.terrain_frames or args.terrain_props_view else (2 if args.terrain_roof_view else 5)
                pan_to([15,view_y]);pan_to([10,view_y])
                required=(args.terrain_frames if args.terrain_frames else
                          {40,42,43,203,204,205,206,207,208} if args.terrain_props_view else
                          ({28,30,32,33,34,35,272,273} if args.terrain_roof_view else set()))
                def terrain_visible(s):
                    return (len(s.get('hd_world_terrain',[])) >= (len(required) if args.terrain_frames else 8 if args.terrain_roof_view else 18)
                            and required.issubset({t['frame'] for t in s.get('hd_world_terrain',[])}))
                wait(terrain_visible)
                assert latest(audit)['units']==before_pan['units'],'單純移鏡頭改變單位'
                label=args.terrain_label or ('props' if args.terrain_props_view else ('roof-object' if args.terrain_roof_view else 'house'))
                shot('terrain-hd-'+label)
                house=latest(audit)
                key('F2');wait(lambda s:s.get('hd_theme') is False and not s.get('hd_world'))
                shot('terrain-faithful-'+label)
                assert latest(audit)['units']==house['units'] and latest(audit)['view']==house['view']
                key('F2');wait(lambda s:s.get('hd_world') is True and terrain_visible(s))
                shot('terrain-hd-'+label+'-restored')
                assert latest(audit)['units']==house['units'] and latest(audit)['view']==house['view']
                if args.terrain_occlusion:
                    walk_behind_house(key,wait,shot,lambda:latest(audit),
                                      args.assets/'assets/maps/map0/map.json',args.out)
                key('F9');wait(lambda s:s.get('node')==args.battle_node)
                for _ in range(120):
                    state=latest(audit)
                    if not state.get('dialogues') and not state.get('banner_t'):break
                    key('Return')
                else:raise TimeoutError('房屋抽驗後普通F9未返回戰場')
                wait(lambda s:not s.get('player_focus') and s.get('hd_world') is True)
            initial=latest(audit)
            key('z');wait(lambda s:not s.get('player_focus'))
            shot('battle-hd-focused')
            focused=latest(audit)
            if args.world:
                def settle_world_battle():
                    for _ in range(120):
                        state=latest(audit)
                        if state.get('level_up_dialogue') and state.get('dialogue_native') is False:
                            raise AssertionError('LOAD後經驗對話owner仍佔有新的非原生章首對話')
                        if state.get('class_ui_job'):
                            wait(lambda s:not s.get('class_ui_job'))
                            continue
                        if state.get('unit_death_scene'):
                            wait(lambda s:not s.get('unit_death_scene'), timeout=180)
                            continue
                        if state.get('death_reward_ui'):
                            raise AssertionError('普通流程遇到未涵蓋的滿背包掉落互動，停止抽驗')
                        if state.get('death_reward_message'):
                            if state.get('death_reward_message_wait'):
                                key('Return')
                            else:
                                wait(lambda s:not s.get('death_reward_message') or s.get('death_reward_message_wait') or s.get('class_ui_job'))
                            continue
                        if not any(state.get(k) for k in ['battle_event','dialogues','banner_t','level_up_dialogue']):
                            return
                        key('Return')
                    raise TimeoutError('普通LOAD戰場對話尚未結束')
                def focus_own(index):
                    state=latest(audit)
                    own=[u for u in state['units'] if u['camp']==0 and u['on_field'] and u['hp']>0]
                    assert index < len(own), '指定我方序號不在目前正常在場名冊'
                    target=[own[index]['x'],own[index]['y']]
                    for _ in range(len(own)+1):
                        key('z');wait(lambda s:not s.get('player_focus'))
                        if latest(audit)['cursor']==target:return
                    raise AssertionError('正常循環選人未抵達指定在場單位')
                wait(lambda s:s.get('hd_world') is True)
                baseline=latest(audit)
                key('Return');wait(lambda s:s.get('move_preview') is True)
                shot('world-hd-move-preview')
                origin=latest(audit)['cursor']
                if args.battle_only:
                    state=latest(audit)
                    occupied={(u['x'],u['y']) for u in state['units'] if u['on_field'] and u['hp']>0}
                    candidates=[cell for cell in state['move_reachable'] if tuple(cell) not in occupied
                                and 2<=abs(cell[0]-origin[0])+abs(cell[1]-origin[1])<=3]
                    assert candidates,'正式可達集合沒有短走行抽驗位置'
                    destination=min(candidates,key=lambda cell:(cell!=[origin[0]+3,origin[1]],cell[1],cell[0]))
                    for _ in range(abs(destination[0]-origin[0])):key('Right' if destination[0]>origin[0] else 'Left')
                    current=latest(audit)['cursor']
                    for _ in range(abs(destination[1]-current[1])):key('Down' if destination[1]>current[1] else 'Up')
                else:
                    for _ in range(3):key('Right')
                destination=latest(audit)['cursor']
                assert destination in latest(audit).get('move_reachable',[]),'普通走行目的地不在正式可達集合'
                shot('world-hd-move-destination')
                world_thread=threading.Thread(target=capture_world_walk,daemon=True)
                world_thread.start()
                key('Return');wait(lambda s:s.get('ring') is True and not s.get('walking'))
                wait(lambda s:s.get('ring_phase')=='open')
                shot('world-hd-ring')
                ring=latest(audit)
                key('F2');wait(lambda s:s.get('hd_theme') is False and not s.get('hd_world'));shot('world-faithful-ring')
                key('F2');wait(lambda s:s.get('hd_world') is True);shot('world-hd-ring-restored')
                assert latest(audit)['units']==ring['units'] and latest(audit)['view']==ring['view']
                key('Escape');wait(lambda s:not s.get('ring') and not s.get('walking'))
                if latest(audit).get('move_preview'):key('Escape');wait(lambda s:not s.get('move_preview'))
                shot('world-hd-move-cancelled')
                world_stop.set();world_thread.join(timeout=15)
                assert not world_thread.is_alive() and world_shots,'未取得有效實際走行GPU'
                key('F9');wait(lambda s:s.get('node')==args.battle_node and not s.get('ring') and not s.get('move_preview'))
                settle_world_battle()
                def ordinary_physical_attack(prefix):
                    def move_cursor(destination):
                        current=latest(audit)['cursor']
                        for _ in range(abs(destination[0]-current[0])):key('Right' if destination[0]>current[0] else 'Left')
                        current=latest(audit)['cursor']
                        for _ in range(abs(destination[1]-current[1])):key('Down' if destination[1]>current[1] else 'Up')
                        assert latest(audit)['cursor']==destination
                    for approach in range(args.battle_approach_rounds+1):
                        focus_own(args.battle_own_index)
                        shot(prefix+('-attacker-focused' if approach==0 else '-approached-attacker-focused'))
                        key('Return');wait(lambda s:s.get('move_preview') is True)
                        state=latest(audit)
                        occupied={(u['x'],u['y']) for u in state['units'] if u['on_field'] and u['hp']>0}
                        enemies=[u for u in state['units'] if u['camp']==2 and u['on_field'] and u['hp']>0]
                        targets=[u for u in enemies if args.battle_target_figure is None or u['battle_fig']==args.battle_target_figure]
                        if args.battle_target_row_max is not None:
                            targets=[u for u in targets if u['y']<=args.battle_target_row_max]
                            assert targets, '目前指定列範圍沒有活躍目標'
                        attacker=next(u for u in state['units'] if u['camp']==0 and u['on_field'] and u['hp']>0
                                      and [u['x'],u['y']]==state['cursor'])
                        # 與battle.State.InAttackRange相同的已載入欄位及零值政策；不改正式規則。
                        minimum,maximum=attacker.get('attack_min',0) or 1,attacker.get('attack_max',0) or 1
                        if args.battle_command==0:
                            assert 0 in (attacker.get('command_ids') or []) and attacker['mp']>=2,'目前單位不能使用正式指令0'
                            record=next(r for r in json.loads((args.assets/'assets/spells.json').read_text()) if r['id']==0)
                            assert record['dist']==5 and record['range']==0 and record['target']==0
                            minimum,maximum=1,record['dist']  # 僅預篩；正式目標格仍須接納。
                        assert 1<=minimum<=maximum,'目前觀測的正式射程無效'
                        choices=[]
                        for target in targets:
                            for cell in state.get('move_reachable',[]):
                                if (tuple(cell) not in occupied or cell==state['cursor']) and minimum<=abs(cell[0]-target['x'])+abs(cell[1]-target['y'])<=maximum:
                                    choices.append((abs(cell[0]-state['cursor'][0])+abs(cell[1]-state['cursor'][1]),cell,target))
                        if args.battle_command==0:
                            # 沿0x18A5F的移動後禁用規則，只在本回合原地施法。
                            origin=state['cursor']
                            choices=[(0,origin,target) for target in targets
                                     if minimum<=abs(origin[0]-target['x'])+abs(origin[1]-target['y'])<=maximum]
                        command_sampled=(not args.battle_end_commands or any(
                            sample['before'].get('command_actor_fig') in end_figures
                            and sample['before'].get('command_id') in args.battle_end_commands
                            for sample in battle_end_command_samples if sample['scene']==prefix))
                        if choices and approach>=args.battle_min_approach_rounds and command_sampled:break
                        assert approach<args.battle_approach_rounds,'正式可達集合沒有符合目前射程的測試位置'
                        advance_cell=None
                        if args.battle_advance:
                            def enemy_distance(cell):
                                return min(abs(cell[0]-enemy['x'])+abs(cell[1]-enemy['y']) for enemy in targets or enemies)
                            cells=[cell for cell in state.get('move_reachable',[]) if tuple(cell) not in occupied or cell==state['cursor']]
                            if args.battle_advance_column is not None:
                                cells=[cell for cell in cells if cell[0]==args.battle_advance_column]
                            assert cells,'沒有可用的正常前進格'
                            if choices:
                                # 已有合法射程時，正常移動到該格待機，保留明示的END抽樣輪數。
                                advance_choices=[choice for choice in choices if choice[1] in cells]
                                assert advance_choices,'指定前進縱列沒有正式射程格'
                                advance_cell=min(advance_choices,key=lambda value:(value[0],value[1][1],value[1][0]))[1]
                            else:
                                advance_cell=min(cells,key=lambda cell:(enemy_distance(cell),cell[1],cell[0]))
                                assert enemy_distance(advance_cell)<enemy_distance(state['cursor']),'正常可達集合無法縮短接敵距離'
                            move_cursor(advance_cell);key('Return')
                            wait(lambda s:s.get('ring') and not s.get('walking') and s.get('ring_phase')=='open')
                            key('Down');key('Return')
                            wait(lambda s:not s.get('ring') and not s.get('class_ui_job') and not s.get('move_preview'))
                            advanced=latest(audit)
                            unit=next(u for u in advanced['units'] if u['camp']==0 and u['native_identity']==attacker['native_identity'])
                            assert [unit['x'],unit['y']]==advance_cell and unit['acted'],'正常移動待機未提交'
                        else:
                            key('Escape');wait(lambda s:not s.get('move_preview'))
                        if args.faithful_approach:
                            key('F2');wait(lambda s:s.get('hd_theme') is False)
                        occupied={(u['x'],u['y']) for u in latest(audit)['units'] if u['on_field'] and u['hp']>0}
                        grid=json.loads((args.assets/f'assets/maps/map{int(args.battle_node[-2:])-1}/map.json').read_text())
                        empty=min(([x,y] for y in range(grid['h']) for x in range(grid['w']) if (x,y) not in occupied),
                                  key=lambda p:(abs(p[0]-state['cursor'][0])+abs(p[1]-state['cursor'][1]),p))
                        move_cursor(empty)
                        key('Return');wait(lambda s:s.get('ring') and s.get('ring_phase')=='open')
                        key('Down');key('Return')
                        wait(lambda s:s.get('system_end_turn_confirm') and not s.get('class_ui_job'))
                        key('Return')
                        deadline=time.monotonic()+(900 if args.battle_end_art else 600 if args.battle_advance else 300)
                        end_seen={};end_scenes={};command_seen={};command_scenes={}
                        while time.monotonic()<deadline:
                            current=latest(audit)
                            if current.get('error'):raise RuntimeError(current['error'])
                            if args.battle_end_art:
                                assert current.get('hd_theme') is True,'正常END意外退出HD主題'
                                owner=current.get('battle_owner')
                                if current.get('battle_scene') and owner:
                                    token=(owner,current.get('battle_actor_fig'),current.get('battle_target_fig'))
                                    stats=end_scenes.setdefault(token,{'owner':owner,'actor_fig':token[1],'target_fig':token[2],
                                        'first_frame':current['frame'],'last_frame':current['frame'],'hd_draw_seen':False})
                                    stats['last_frame']=current['frame'];stats['hd_draw_seen']|=current.get('hd_battle') is True
                                if current.get('battle_scene') and current.get('hd_battle') and current.get('battle_phase') in ('body','counter','departure'):
                                    for layer in current.get('battle_source_layers') or []:
                                        resource,index=layer['resource'],layer['frame']
                                        count=end_seen.get((resource,index),0)
                                        if (resource not in {figure*3+kind for figure in end_figures for kind in (0,1)}
                                                or layer['fill']!=-1 or count>=args.battle_end_samples_per_frame):continue
                                        end_seen[(resource,index)]=count+1
                                        suffix='' if count==0 else f'-sample{count+1}'
                                        path=args.out/f'{prefix}-end-{approach}-{resource:03d}-{index:02d}{suffix}.png'
                                        command('import','-window',window,str(path))
                                        battle_end_art_samples.append({'scene':prefix,'approach':approach,'file':path.name,
                                            'sha256':digest(path),'before':current,'after':latest(audit),
                                            'scope':'當前job只用於定位；是否真的呈現由PNG另驗'})
                                if current.get('command_scene') and current.get('command_id') in args.battle_end_commands:
                                    owner=current.get('command_owner')
                                    assert owner,'施法場景缺少正式owner'
                                    stats=command_scenes.setdefault(owner,{'owner':owner,'command_id':current['command_id'],
                                        'actor_fig':current.get('command_actor_fig'),'target_fig':current.get('command_target_fig'),
                                        'first_frame':current['frame'],'hd_draw_seen':False})
                                    stats['last_frame']=current['frame'];stats['hd_draw_seen']|=current.get('hd_battle') is True
                                    if current.get('hd_battle') and current.get('command_phase') in ('actor','tail','handler-front','handler-tail'):
                                        for layer in current.get('command_source_layers') or []:
                                            resource,index=layer['resource'],layer['frame']
                                            token=(current['command_id'],resource,index)
                                            count=command_seen.get(token,0)
                                            if (layer['archive'] not in ('','FIGANI.DAT') or resource not in
                                                {figure*3+kind for figure in end_figures for kind in (0,2)}
                                                or layer['fill']!=-1 or count>=args.battle_end_samples_per_frame):continue
                                            command_seen[token]=count+1
                                            suffix='' if count==0 else f'-sample{count+1}'
                                            path=args.out/f'{prefix}-end-{approach}-cmd{current["command_id"]}-{resource:03d}-{index:02d}{suffix}.png'
                                            command('import','-window',window,str(path))
                                            battle_end_command_samples.append({'scene':prefix,'approach':approach,'file':path.name,
                                                'sha256':digest(path),'before':current,'after':latest(audit),
                                                'scope':'正常END的實際施法；job定位與PNG呈現分開核對'})
                            if current.get('title_phase'):
                                raise RuntimeError('普通END已回到標題，不能取得指定攻擊：'+str(current.get('title_phase')))
                            if (current['native_round']>state['native_round'] and current['turn']>state['turn']
                                    and not any(current.get(k) for k in ['ai_busy','battle_event','dialogues','banner_t',
                                                                       'battle_scene','command_scene','unit_death_scene','class_ui_job','system_end_turn_confirm'])):break
                            if current.get('dialogues'):key('Return')
                            else:time.sleep(.05)
                        else:raise TimeoutError('普通END未回玩家回合')
                        if args.battle_end_art:
                            # 換手可以只有走行；實際戰鬥仍逐場驗HD，整份收據另要求END樣本。
                            assert all(s['hd_draw_seen'] for s in end_scenes.values()),'END有物理場景沒有HD呈現證據'
                            assert all(s['hd_draw_seen'] for s in command_scenes.values()),'END有明示施法場景沒有HD呈現證據'
                        own_before=[u for u in state['units'] if u['camp']==0 and u['on_field'] and u['hp']>0]
                        own_after=[u for u in current['units'] if u['camp']==0 and u['on_field'] and u['hp']>0]
                        survivors={(u['native_identity'],u['name']) for u in own_after}
                        assert all((u['native_identity'],u['name']) in survivors for u in own_before),'普通END有我方離場，停止抽驗'
                        if args.faithful_approach:
                            key('F2');wait(lambda s:s.get('hd_theme') is True and s.get('hd_world'))
                        battle_approach_steps.append({'scene':prefix,'before':state,'after':current,
                            'advance_cell':advance_cell,'end_in_faithful_theme':args.faithful_approach,
                            'physical_scenes':list(end_scenes.values()),'command_scenes':list(command_scenes.values()),'state_injection':False})
                        shot(prefix+'-approach-round-return')
                    _,cell,target=min(choices,key=lambda value:(value[0],value[1][1],value[1][0]))
                    battle_attack_choices.append({'scene':prefix,'attacker':attacker,'range':[minimum,maximum],
                                                  'cell':cell,'target':target,'state_injection':False,
                                                  'action_kind':action_label,'command_candidate_policy':'stationary_prefilter_then_formal_target_field' if args.battle_command==0 else None})
                    move_cursor(cell);key('Return')
                    wait(lambda s:s.get('ring') and not s.get('walking') and s.get('ring_phase')=='open')
                    shot(prefix+'-attack-ring')
                    if args.battle_command==0:
                        key('Left');key('Return')
                        wait(lambda s:s.get('command_grid') and s.get('command_selection')==0)
                        key('Return');wait(lambda s:s.get('command_targeting') and s.get('command_target_id')==0)
                        key('Escape');wait(lambda s:s.get('command_grid'))
                        shot(prefix+'-command0-cancelled-target')
                        key('Return');wait(lambda s:s.get('command_targeting') and s.get('command_target_id')==0)
                        assert [target['x'],target['y']] in latest(audit).get('command_target_cells',[]),'正式法術目標格拒收'
                    else:
                        key('Return');wait(lambda s:not s.get('ring'))
                    move_cursor([target['x'],target['y']]);shot(prefix+'-attack-target')
                    scene_started=time.monotonic()
                    scene_key='command_scene' if args.battle_command==0 else 'battle_scene'
                    key('Return');wait(lambda s:s.get(scene_key) is True)
                    if args.battle_hd:
                        wait(lambda s:s.get(scene_key) and s.get('hd_battle'))
                    shot(prefix+'-'+action_label+'-scene')
                    if args.battle_hd:
                        assert shots[prefix+'-'+action_label+'-scene.png']['state'].get('hd_battle'), '未擷取到HD戰鬥繪製層'
                    if args.battle_art:
                        # 只讀實際視窗，不暫停、改寫或推進遊戲狀態。
                        for index in range(args.battle_art_samples):
                            before=latest(audit)
                            if not before.get(scene_key):break
                            if before.get('hd_battle'):
                                path=args.out/f'{prefix}-art-{index:02d}.png'
                                command('import','-window',window,str(path))
                                battle_art_samples.append({'scene':prefix,'file':path.name,'sha256':digest(path),
                                                           'before':before,'after':latest(audit)})
                            time.sleep(.04)
                    returned_scene=wait(lambda s:not s.get(scene_key) and not s.get('unit_death_scene') and not s.get('walking') and s.get('hd_world'),
                                        timeout=180 if args.battle_hd else 60)
                    if args.battle_hd:
                        battle_hd_scenes.append({'name':prefix,'elapsed_seconds':time.monotonic()-scene_started,
                                                 'presented':shots[prefix+'-'+action_label+'-scene.png']['state'],
                                                 'returned':returned_scene,
                                                 'scope':'軟體GPU與視窗擷取的牆鐘時間，不是正式平台效能驗收'})
                if args.timed_dialogue:
                    timed_probe=TimedDialogueProbe(args.out,audit,lambda p:command('import','-window',window,str(p)),env)
                    timed_probe.thread.start()
                ordinary_physical_attack('world-hd')
                # The physical scene can finish before its experience dialogue.
                # Keep this distinct from a fully returned battle map.
                shot('world-hd-physical-before-load')
                if args.timed_dialogue:
                    wait(lambda s:s.get('timed_dialogue_wait') and s.get('hd_timed_dialogue') and s.get('hd_timed_phase')==1)
                    shot('timed-hd-stable')
                    frozen=latest(audit)
                    key('F2');wait(lambda s:not s.get('hd_theme'));shot('timed-faithful-stable')
                    key('F2');wait(lambda s:s.get('hd_timed_dialogue') and s.get('hd_timed_phase')==1);shot('timed-hd-restored')
                    for field in ('units','view','timed_dialogue_owner','dialogue_progress'):
                        assert latest(audit)[field]==frozen[field],('計時對話F2改變狀態',field)
                    for _ in range(120):
                        state=latest(audit)
                        if not state.get('level_up_dialogue') and not state.get('dialogues'):break
                        key('Return')
                    else:raise TimeoutError('第一筆升級對話尚未結束')
                if args.battle_repetitions==2:
                    key('F9');wait(lambda s:s.get('node')==args.battle_node and not s.get('battle_scene'))
                    settle_world_battle()
                    assert not latest(audit).get('level_up_dialogue') and not latest(audit).get('move_preview')
                    shot('world-hd-load-owner-cleared')
                    ordinary_physical_attack('world-hd-replay')
                for _ in range(120):
                    state=latest(audit)
                    if not state.get('level_up_dialogue') and not state.get('dialogues') and not state.get('battle_event'):break
                    key('Return')
                else:raise TimeoutError('普通戰鬥經驗對話尚未結束')
                settle_world_battle()
                wait(lambda s:s.get('hd_world') and not s.get('battle_scene'))
                shot('world-hd-physical-return')
                if timed_probe:timed_result=timed_probe.finish(validate=True)
                if not args.battle_only:key('F9')
                settle_world_battle()
                # 最後一次翻頁可能剛好落在自動關框後，正常開出狀態面板。
                # 沿正式Escape先關狀態／命令頁，再用Z選人。
                # nativeMovePlan可在g.sel=nil後保留；它本身不是開著的輸入面板。
                for _ in range(8):
                    state=latest(audit)
                    phase=state.get('player_status_phase')
                    if phase:
                        state=wait(lambda s:s.get('player_status_phase') in (None,'status','commands'))
                        phase=state.get('player_status_phase')
                        if not phase:continue
                        key('Escape')
                        wait(lambda s:s.get('player_status_phase') in (None,'status','commands') and s.get('player_status_phase')!=phase)
                    elif state.get('ring'):
                        key('Escape')
                    else:break
                    battle_post_dialogue_cleanup.append({'before':state,'after':latest(audit),'key':'Escape','state_injection':False})
                else:raise AssertionError('普通戰後狀態面板或選取尚未取消')
                focus_own(0)
                wait(lambda s:s.get('hd_world') is True)
                shot('world-hd-postbattle-before-save' if args.battle_only else 'world-hd-source-reloaded')
                focused=latest(audit)
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('battle-faithful-focused')
            faithful_focused=latest(audit)
            assert faithful_focused['units']==focused['units'] and faithful_focused['view']==focused['view']
            assert not faithful_focused.get('player_status_phase'),'F2誤開角色狀態'
            key('F2');wait(lambda s:s.get('hd_theme') is True);shot('battle-hd-focused-restored')
            assert latest(audit)['units']==focused['units'] and latest(audit)['view']==focused['view']
            assert not latest(audit).get('player_status_phase'),'F2返回誤開角色狀態'
            outside=[]
            returned=latest(audit)
            if not args.battle_only:
                key('Home');wait(lambda s:s.get('player_status_phase')=='status')
                shot('battle-hd-status')
                key('F2');wait(lambda s:s.get('hd_theme') is False);shot('battle-faithful-status')
                key('F2');wait(lambda s:s.get('hd_theme') is True);shot('battle-hd-status-restored')
                # Sol has no command IDs. The first confirmation closes his panel.
                key('Return');wait(lambda s:not s.get('player_status_phase'))
                shot('battle-hd-returned')
                returned=latest(audit)
                assert focused['units']==returned['units']
                assert focused['cursor']==returned['cursor']
                key('z');wait(lambda s:not s.get('player_focus'))
                shot('battle-hd-command-unit-focused')
                command_focused=latest(audit)
                key('Home');wait(lambda s:s.get('player_status_phase')=='status')
                shot('battle-hd-command-unit-status')
                command('xdotool','windowsize',window,'1600','1000');shot('battle-status-resized')
                command('xdotool','windowsize',window,'1920','1200')
                key('F1');wait(lambda s:s.get('help_visible') is True);shot('battle-status-help')
                key('Escape');wait(lambda s:s.get('help_visible') is False)
                key('Return');wait(lambda s:s.get('player_status_phase')=='commands')
                shot('battle-hd-commands')
                key('Return');wait(lambda s:not s.get('player_status_phase'))
                shot('battle-hd-command-unit-returned')
                returned=latest(audit)
                assert command_focused['units']==returned['units']
                assert command_focused['cursor']==returned['cursor']
                assert digest(saved)==digest(args.town_save),'面板操作改寫保存'
                outside=[]
                status_reference='world-hd-source-reloaded' if args.world else 'battle-hd-focused'
                for reference, targets in [
                        (status_reference,['battle-hd-status','battle-hd-status-restored','battle-hd-returned']),
                        ('battle-hd-command-unit-focused',['battle-hd-command-unit-status','battle-hd-commands','battle-hd-command-unit-returned'])]:
                    for target in targets:
                        assert shots[reference+'.png']['state']['view']==shots[target+'.png']['state']['view'], '面板外比較鏡頭不同'
                        # The top five native rows are outside every stable panel.
                        # This is actual map RGB, not an audit-based Draw assertion.
                        roi='1800x30+60+0'
                        result=subprocess.run(['compare','-metric','AE',str(args.out/(reference+'.png'))+'['+roi+']',
                                               str(args.out/(target+'.png'))+'['+roi+']','null:'],
                                              env=env,capture_output=True,text=True,check=False)
                        assert result.returncode in (0,1),result.stderr
                        difference=float(result.stderr.strip())
                        outside.append({'reference':reference,'target':target,'crop':roi,'different_pixels':difference})
                        assert difference==0,('面板外現代地圖被覆蓋',outside[-1])
            key('F5');wait(lambda s:s.get('frame',0)>returned['frame'])
            hd_saved=saved.read_bytes()
            persistence_fields=('native_identity','battle_fig','name','x','y','hp','mp','lv','exp','camp','acted','on_field','attack_min','attack_max')
            def persistence(state):
                return [{field:unit.get(field) for field in persistence_fields} for unit in state['units']]
            warm_units=persistence(returned)
            key('F2');wait(lambda s:s.get('hd_theme') is False)
            key('F9');wait(lambda s:s.get('node')==args.battle_node)
            faithful_loaded=latest(audit)
            key('F5');assert saved.read_bytes()==hd_saved,'兩主題保存不同'
            key('F2');wait(lambda s:s.get('hd_theme') is True)
            key('F9');wait(lambda s:s.get('node')==args.battle_node)
            hd_loaded=latest(audit)
            loaded_units=persistence(hd_loaded)
            assert faithful_loaded['units']==hd_loaded['units'],'兩主題LOAD結果不同'
            assert saved.read_bytes()==hd_saved
            restart_title_load(args.battle_node,hd_loaded['gold'],'battle-cold-title-load')
            assert saved.read_bytes()==hd_saved
            cold_units=persistence(latest(audit))
            if args.battle_only:
                assert loaded_units==cold_units,'章節邊界保存／新程序LOAD的重建狀態不同'
            if args.battle_end_art:
                end_samples=battle_end_art_samples+battle_end_command_samples if args.battle_end_commands else battle_end_art_samples
                assert all(any(s['scene']==name for s in end_samples) for name in ('world-hd','world-hd-replay')[:args.battle_repetitions]),'未取得明示場數的普通END圖源樣本'
                if args.battle_end_commands:
                    assert battle_end_command_samples,'未取得明示指令的普通END施法樣本'
            assert digest(Path(__file__))==tool_source_sha256,'GUI工具執行中來源被修改'
            report={'schema':'fd2-hd-battle-status-gui-v1','status':'PASS',
                    'catalog_sha256':digest(args.assets/'assets/themes/modern/catalog.json'),
                    'gui_tool_sha256':tool_source_sha256,
                    'world_runtime_checked':args.world,'world_walk_gpu':world_shots,'world_walk_rejected':world_rejected,
                    'battle_hd_consumer_checked':args.battle_hd,
                    'battle_command_id':args.battle_command,
                    'battle_action_kind':action_label,
                    'battle_repetitions':args.battle_repetitions,'battle_art_sample_limit':args.battle_art_samples,
                    'battle_end_figures':end_figures,
                    'natural_attack_return_checked':args.battle_only,
                    'battle_own_index':args.battle_own_index,
                    'battle_target_figure':args.battle_target_figure,
                    'battle_target_row_max':args.battle_target_row_max,
                    'battle_advance_column':args.battle_advance_column,
                    'battle_min_approach_rounds':args.battle_min_approach_rounds,
                    'battle_node':args.battle_node,'battle_only':args.battle_only,
                    'battle_attack_choices':battle_attack_choices,
                    'battle_approach_rounds_limit':args.battle_approach_rounds,
                    'battle_advance':args.battle_advance,'faithful_approach':args.faithful_approach,
                    'battle_approach_steps':battle_approach_steps,
                    'battle_source_roster':{
                        ident:{field:unit.get(field) for field in ('NativeIdentity','Fig','BattleFig')}
                        for ident,unit in json.loads(args.town_save.read_text())['party_roster'].items()
                    },
                    'battle_hd_scenes':battle_hd_scenes,
                    'battle_art_samples':battle_art_samples,
                    'battle_end_art_checked':args.battle_end_art,'battle_end_figure':args.battle_end_figure,
                    'battle_end_samples_per_frame':args.battle_end_samples_per_frame,
                    'battle_end_art_samples':battle_end_art_samples,
                    'battle_end_commands':sorted(args.battle_end_commands),
                    'battle_end_command_samples':battle_end_command_samples,
                    'battle_post_dialogue_cleanup':battle_post_dialogue_cleanup,
                    'timed_dialogue_gpu':timed_result,
                    'binary_sha256':digest(args.binary),'source_save_sha256':digest(args.town_save),
                    'saved_sha256':digest(saved),'screenshots':shots,'operations':operations,
                    'initial':initial,'focused':focused,'returned':returned,
                    'outside_map_gpu':outside,'new_process_title_load':True,
                    'source_save_bytes_equal':saved.read_bytes()==args.town_save.read_bytes(),
                    'theme_save_bytes_equal':True,'loaded_units_equal':True,
                    'post_battle_save_checked':args.battle_only,
                    'node_boundary_save_checked':args.battle_only,
                    'cold_loaded_units_equal':loaded_units==cold_units,
                    'warm_units':warm_units,'loaded_units':loaded_units,'cold_units':cold_units,
                    'save_contract':'F5保存章節邊界進度；LOAD重建戰場，不保存戰中位置、Acted或敵方HP。',
                    'recording_created':False,'elapsed_seconds':time.monotonic()-started,
                    'scope':((f'正常標題LOAD來源JSON，普通走行、{args.battle_repetitions}場攻防、F2及戰後保存／新程序LOAD；來源祖先另見建槽收據。未重驗字型面板。' if args.battle_only else '正常標題LOAD既有普通START／F5首章JSON，Z聚焦、Home檢視、F2、狀態與命令、返回；')
                             + (f'含普通走行、指令環、取消、{args.battle_repetitions}場{action_label}演出與正常戰場返回。'+('含經驗對話中F9。' if args.battle_repetitions==2 else '') if args.world else '')
                             + '不注入座標或狀態，不宣稱原版對拍或完整戰役。')}
            if args.terrain_frames:
                report['terrain_expectation']={'label':args.terrain_label,'frames':sorted(args.terrain_frames),'view_y':8}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        if args.preparation:
            if args.transition_frames:
                transition_thread=threading.Thread(target=capture_transitions,daemon=True)
                transition_thread.start()
            key('Down');key('Return')
            wait(lambda s:s.get('title_phase')=='loadslots' and s.get('title_slot')==0)
            shot('title-load-slots')
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-title-load-slots')
            key('F2');wait(lambda s:s.get('hd_theme') is True);shot('hd-title-load-slots-restored')
            for slot in range(1,4):
                key('Down');wait(lambda s:s.get('title_slot')==slot);shot('title-slot-'+str(slot))
            key('Down');wait(lambda s:s.get('title_slot')==3)
            key('Escape');wait(lambda s:s.get('title_phase')=='menu')
            key('Return');wait(lambda s:s.get('title_phase')=='loadslots')
            key('Return');wait(lambda s:not s.get('title_phase'))
            def prep(**values):
                return wait(lambda s:s.get('node','').startswith('preparation_')
                            and not s.get('preparation_ui_busy')
                            and all(s.get(k)==v for k,v in values.items()))
            if native_saved:
                prep(preparation_selecting=False,preparation_slots=False,preparation_confirm=False)
                shot('preparation-record-question')
                assert digest(native_saved)==digest(args.native_save),'LOAD改寫來源複本'
                key('Return');prep(preparation_slots=True);shot('preparation-record-slots')
                key('Down');prep(preparation_slot=1)
                key('Return');prep(preparation_slots=True)
                hd_saved=native_saved.read_bytes();shot('preparation-record-saved')
                key('F2');wait(lambda s:s.get('hd_theme') is False)
                key('Return');prep(preparation_slots=True)
                assert native_saved.read_bytes()==hd_saved,'兩主題原生保存不同'
                key('F2');wait(lambda s:s.get('hd_theme') is True)
                key('Escape');prep(preparation_selecting=True,preparation_slots=False)
                shot('preparation-roster')
                key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-preparation-roster')
                key('F2');wait(lambda s:s.get('hd_theme') is True);shot('hd-preparation-roster-restored')
                command('xdotool','windowsize',window,'1600','1000');shot('preparation-resized')
                command('xdotool','windowsize',window,'1920','1200')
                key('F1');wait(lambda s:s.get('help_visible') is True);shot('preparation-help')
                key('Escape');wait(lambda s:s.get('help_visible') is False)
                key('Right');prep(preparation_selection=1);shot('preparation-next-unit')
                key('Left');prep(preparation_selection=0)
                for _ in range(latest(audit)['preparation_limit']):
                    key('Return')
                prep(preparation_selecting=False)
                shot('preparation-final-question')
                key('Escape');prep(preparation_selecting=True);shot('preparation-returned-roster')
                assert native_saved.read_bytes()==hd_saved,'選人或取消改寫原生存檔'
                assert digest(args.native_save)==native_source_hash,'原始槽被改寫'
                saved_hash=digest(native_saved)
                scope='正常標題LOAD既有第24章原生槽；祖先為攻略建構政策與清敵續玩，並非自然難度。保存槽1、選人、確認或必選訊息與取消均走普通鍵盤；不宣稱連續戰役或原版對拍。'
            else:
                wait(lambda s:s.get('node')=='hotel_ch02')
                key('Escape');wait(lambda s:s.get('node')=='town_ch02')
                key('Left');key('Left');key('Return')
                prep(preparation_selecting=False);shot('preparation-departure-question')
                key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-preparation-question')
                key('F2');wait(lambda s:s.get('hd_theme') is True);shot('hd-preparation-question-restored')
                key('Right');prep(preparation_choice=1);shot('preparation-departure-no')
                key('Return');wait(lambda s:s.get('node')=='town_ch02')
                key('Right');key('Right');key('Return')
                wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
                key('F5');hd_saved=saved.read_bytes()
                key('F2');wait(lambda s:s.get('hd_theme') is False)
                key('F5');assert saved.read_bytes()==hd_saved,'兩主題JSON保存不同'
                key('F9');wait(lambda s:s.get('node')=='hotel_ch02')
                shot('faithful-hotel-after-preparation-load')
                assert digest(saved)==digest(args.town_save),'出發取消改變來源狀態'
                saved_hash=digest(saved)
                scope='正常標題LOAD既有hotel_ch02 JSON槽，城鎮整備出發問句選NO返回，旅館F5／F9；非連續全戰役，不宣稱原版對拍。'
            finish_transitions()
            if args.transition_frames:
                validate_transition_samples()
                directions={m['before']['preparation_transition_closing'] for m in transition_shots.values()}
                assert directions=={False,True},'未取得開框及收框中的有效問句轉場樣本'
            assert shots['title-load-slots.png']['sha256']==shots['hd-title-load-slots-restored.png']['sha256'],'標題F2往返圖不同'
            report={'schema':'fd2-hd-preparation-font-gui-v1','status':'PASS',
                    'binary_sha256':digest(args.binary),'town_save_sha256':digest(args.town_save),
                    'native_source_sha256':digest(args.native_save) if args.native_save else None,
                    'saved_sha256':saved_hash,'theme_save_bytes_equal':True,
                    'screenshots':shots,'operations':operations,'recording_created':False,
                    'transition_screenshots':transition_shots,'transition_rejected':transition_rejected,
                    'transition_gpu_validated':transition_gpu_validated,
                    'scope':scope,'elapsed_seconds':time.monotonic()-started}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        key('Return')
        wait(lambda s:not s.get('title_phase') and s.get('dialogue_native') is True, timeout=180)
        if args.panel_transitions:
            panel_probe=PanelTransitionProbe(args.out,lambda:latest(audit),command,lambda:window,env)
            panel_probe.start()
        if args.gold_roll:
            transaction_hd=args.gold_roll_theme=='hd'
            if not transaction_hd:
                key('F2');wait(lambda s:s.get('hd_theme') is False)
            def shop(mode):
                return wait(lambda s:s.get('node')=='shop_ch02_weapon' and s.get('shop_mode')==mode
                            and s.get('shop_ui_busy') is False)
            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            key('Left');key('Return');shop('menu');shot('gold-before-purchase')
            gold_thread=threading.Thread(target=capture_gold_roll,daemon=True);gold_thread.start()
            key('Return');shop('purchase')
            key('Return');shop('confirm')
            key('Return');shop('recipient_equipment')
            key('Return');shop('equip_confirm')
            key('Right');key('Return');shop('purchase')
            assert latest(audit).get('gold')==950,'購買布衣的餘額錯誤'
            shot('gold-after-purchase')
            key('F2');wait(lambda s:s.get('hd_theme') is not transaction_hd);shot('gold-alternate-after-purchase')
            key('F2');wait(lambda s:s.get('hd_theme') is transaction_hd);shot('gold-purchase-theme-restored')
            key('Escape');shop('menu')
            key('Right');wait(lambda s:s.get('shop_service')==1)
            key('Return');shop('sell_roster')
            key('Return');shop('sell_items')
            # Sol starts with short sword, leather armor and herb. The ordinary
            # purchase appends cloth in the fourth active raw inventory slot.
            key('Down');key('Right');key('Return');shop('sell_confirm');shot('gold-cloth-sale-confirmation')
            key('Return');shop('sell_roster')
            assert latest(audit).get('gold')==987,'出售布衣的餘額錯誤'
            shot('gold-after-sale')
            finish_gold_roll()
            assert not gold_errors,gold_errors
            accepted,rejected={},[]
            def rgb(path,roi):
                return subprocess.check_output(['convert',str(path),'-crop',roi,'+repage','-depth','8','RGB:-'],env=env)
            for name,meta in gold_shots.items() if transaction_hd else []:
                credit=meta['before']['gold_roll_credit']
                before=args.out/('gold-after-purchase.png' if credit else 'gold-before-purchase.png')
                after=args.out/('gold-after-sale.png' if credit else 'gold-after-purchase.png')
                path=args.out/name
                prefix=rgb(path,'144x54+0+0')
                changed=rgb(path,'144x54+144+0')
                same_prefix=prefix==rgb(after,'144x54+96+588')
                moving=changed not in [rgb(before,'144x54+240+588'),rgb(after,'144x54+240+588')]
                meta.update(static_hd_prefix_rgb_equal=same_prefix,moving_digits_distinct_from_both_stable=moving,
                            exact_frame_claimed=False)
                if same_prefix and moving:accepted[name]=meta
                else:rejected.append({'file':name,**meta})
            if transaction_hd:
                assert {m['before']['gold_roll_credit'] for m in accepted.values()}=={False,True},'未取得兩個方向的實際 GPU 滾動樣本'
            key('Escape');shop('menu');key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            key('Right');key('Return');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            assert digest(saved)==digest(args.town_save),'來源複本在保存前被改寫'
            key('F5');hd_saved=saved.read_bytes()
            source=json.loads(args.town_save.read_text());result=json.loads(hd_saved)
            expected=json.loads(args.town_save.read_text());expected['gold']=987
            # Existing 0x1b8e7 removal shifts all remaining raw cells, including
            # unused item bytes; the final item byte stays, only its flag is set.
            unit=expected['party_roster']['0']
            flags=unit['NativeInventoryFlags'];slots=unit['InventorySlots']
            raw_slot=next(i for i,flag in enumerate(flags) if flag&0x80)
            slots[raw_slot:-1]=slots[raw_slot+1:]
            flags[raw_slot:-1]=flags[raw_slot+1:];flags[-1]=0x80
            assert result==expected,'保存結果與既有買賣原始槽位移規則不同'
            key('F2');wait(lambda s:s.get('hd_theme') is not transaction_hd)
            key('F5');assert saved.read_bytes()==hd_saved,'兩主題保存結果不同'
            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('gold')==987)
            shot('gold-alternate-f9-load')
            key('F2');wait(lambda s:s.get('hd_theme') is transaction_hd);shot('gold-transaction-theme-f9-load')
            game.terminate();game.wait(timeout=15)
            # Frame counters restart in a new process. Retain its predecessor's
            # log separately so old high-frame key ACKs cannot satisfy new keys.
            audit.rename(args.out/'input-audit-first-process.jsonl')
            audit_epoch=1
            game=start([str(args.binary)],'game-cold-load.log',args.assets)
            window=None
            for _ in range(600):
                if game.poll() is not None:raise RuntimeError('冷讀程序提前退出')
                try:window=command('xdotool','search','--name','炎龍騎士團').splitlines()[0];break
                except (subprocess.CalledProcessError,IndexError):time.sleep(.1)
            assert window
            command('xdotool','windowmove',window,'0','0')
            wait(lambda s:bool(s.get('title_phase')))
            for _ in range(160):
                if latest(audit).get('title_phase')=='menu':break
                key('Escape')
            else:raise TimeoutError('冷讀標題未就緒')
            key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots')
            key('Return');wait(lambda s:not s.get('title_phase') and s.get('node')=='hotel_ch02' and s.get('gold')==987)
            assert saved.read_bytes()==hd_saved,'新程序LOAD改寫保存檔'
            shot('gold-new-process-title-load')
            report={'schema':'fd2-hd-gold-roll-gui-v1','status':'PASS','binary_sha256':digest(args.binary),
                    'gui_tool_sha256':digest(Path(__file__)),
                    'town_save_sha256':digest(args.town_save),'saved_sha256':digest(saved),
                    'transaction_theme':args.gold_roll_theme,'theme_save_bytes_equal':True,
                    'expected_save_changes':{'gold':[1000,987],'party_roster/0/InventorySlots/5':[255,0]},
                    'new_process_title_load':True,
                    'screenshots':shots,'gold_roll_screenshots':accepted,'gold_roll_rejected':rejected,
                    'operations':operations,'recording_created':False,'elapsed_seconds':time.monotonic()-started,
                    'scope':'正常START後F9接續既有hotel_ch02，普通購買布衣、拒絕立即裝備並出售同件。1000→950→987；F2、F5及F9。CPU只定位候選，PNG以固定HD前四位與相異動態後四位驗證，不宣稱精確同幀、連續戰役或原版逐幀對拍。'}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        if args.items:
            def shop(mode, selection=None):
                return wait(lambda s:s.get('node')=='shop_ch02_weapon' and s.get('shop_mode')==mode
                            and s.get('shop_ui_busy') is False
                            and (mode!='equip_panel' or (s.get('item_anim_step')==11 and not s.get('item_closing')))
                            and (selection is None or s.get('item_selection')==selection))
            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            key('Left');key('Return');shop('menu');shot('shop-menu')
            key('Right');key('Right');wait(lambda s:s.get('shop_service')==2)
            key('Return');shop('equip_roster');shot('equip-roster')
            key('Return');shop('equip_panel',0);shot('item-short-sword')
            equip_before=latest(audit).get('equip_unit');assert equip_before and equip_before['id']==0
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-item-short-sword')
            key('F2');wait(lambda s:s.get('hd_theme') is True);shop('equip_panel',0);shot('hd-item-short-sword-restored')
            command('xdotool','windowsize',window,'1600','1000');shot('item-panel-resized')
            command('xdotool','windowsize',window,'1920','1200')
            key('F1');wait(lambda s:s.get('help_visible') is True);shot('item-panel-help')
            key('Escape');wait(lambda s:s.get('help_visible') is False)
            key('Down');shop('equip_panel',1);shot('item-armor')
            key('Down');shop('equip_panel',2);shot('item-herb')
            key('Return');shop('equip_panel',2);shot('item-herb-rejected')
            assert latest(audit).get('equip_unit')==equip_before,'裝備藥草改變角色'
            key('Up');key('Up');shop('equip_panel',0)
            key('Return');shop('equip_panel',0);shot('item-short-sword-reequipped')
            assert latest(audit).get('equip_unit')==equip_before,'重新裝備既有短劍改變資料'
            key('Escape');shop('equip_roster');shot('equip-roster-returned')
            key('Escape');shop('menu')
            key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            key('Right');key('Return');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            assert digest(saved)==digest(args.town_save),'來源複本在保存前被改寫'
            key('F5');hd_saved=saved.read_bytes()
            key('F2');wait(lambda s:s.get('hd_theme') is False)
            key('F5');assert saved.read_bytes()==hd_saved,'兩主題保存結果不同'
            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            shot('faithful-hotel-after-items-load')
            panels=panel_probe.validate(items=True) if panel_probe else None
            if args.panel_transitions:
                assert hd_saved==args.town_save.read_bytes(),'物品轉場改變來源保存'
                restart_title_load('hotel_ch02',1000,'hotel-after-items-new-process-load')
                assert saved.read_bytes()==hd_saved,'新程序LOAD改寫保存'
            report={'schema':'fd2-hd-item-font-gui-v1','status':'PASS',
                    'panel_transitions':panels,
                    'new_process_title_load':args.panel_transitions,
                    'binary_sha256':digest(args.binary),'town_save_sha256':digest(args.town_save),
                    'saved_sha256':digest(saved),'theme_save_bytes_equal':True,
                    'source_save_bytes_equal':digest(saved)==digest(args.town_save),
                    'screenshots':shots,'operations':operations,'recording_created':False,
                    'scope':'正常START後F9接續既有hotel_ch02，經城鎮進武器店裝備服務；普通選索爾、移動至皮甲／藥草、拒絕裝備藥草、重新裝備既有短劍；F2、縮放／說明遮罩、返回旅館F5／F9。原生物品文字沿既有語言，未新增翻譯，不宣稱連續戰役或原版對拍。',
                    'elapsed_seconds':time.monotonic()-started}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        if args.church or args.status:
            def church(mode):
                return wait(lambda s:s.get('node')=='church_ch02' and s.get('church_mode')==mode
                            and s.get('church_ui_busy') is False and s.get('hd_text_runs',0)>0)

            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            for _ in range(4):key('Left')
            key('Return');church('menu');shot('church-menu')
            key('Return');church('status_roster')
            if args.status:
                key('Return');church('status_view');shot('church-status-sol')
                key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-church-status-sol')
                key('F2');wait(lambda s:s.get('hd_theme') is True);church('status_view');shot('hd-church-status-sol-restored')
                command('xdotool','windowsize',window,'1600','1000');shot('church-status-resized')
                command('xdotool','windowsize',window,'1920','1200')
                key('F1');wait(lambda s:s.get('help_visible') is True);shot('church-status-help')
                key('Escape');wait(lambda s:s.get('help_visible') is False)
                key('Return');church('status_roster')
                key('Right');church('status_roster')
                key('Return');church('status_view')
                assert latest(audit).get('church_status_id')==9,'命令角色不是原槽ID9'
                shot('church-status-yuni')
                key('Return');church('status_commands');shot('church-commands-yuni')
                key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-church-commands-yuni')
                key('F2');wait(lambda s:s.get('hd_theme') is True);church('status_commands');shot('hd-church-commands-yuni-restored')
                key('Return');church('status_roster')
                key('Left');church('status_roster')
            for locale in ['zh-Hant','zh-Hans','ja','en']:
                wait(lambda s:s.get('locale')==locale);church('status_roster')
                shot('church-roster-'+locale);key('F4')
            before=church('status_roster')
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-church-roster')
            key('F2');wait(lambda s:s.get('hd_theme') is True);church('status_roster')
            after=latest(audit)
            for field in ['node','church_mode','church_selection','gold']:
                assert before[field]==after[field],field
            shot('hd-church-roster-restored')
            command('xdotool','windowsize',window,'1600','1000');shot('church-roster-resized')
            command('xdotool','windowsize',window,'1920','1200')
            key('F1');wait(lambda s:s.get('help_visible') is True);shot('church-help')
            key('Escape');wait(lambda s:s.get('help_visible') is False)
            key('Escape');church('menu')
            key('Right');key('Right');key('Return');church('revive');shot('church-revive-list')
            key('Return');church('revive_confirm');shot('church-revive-question')
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-church-revive-question')
            key('F2');wait(lambda s:s.get('hd_theme') is True);church('revive_confirm')
            key('Escape');church('revive')
            key('Escape');church('menu')
            key('Right');key('Return');church('transfer_source');shot('church-transfer-source')
            key('Return');church('transfer_item');shot('church-transfer-item')
            key('Escape');church('transfer_source')
            key('Escape');church('menu')
            assert latest(audit).get('gold')==1000,'取消服務改變金幣'
            key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            # Re-enter the hotel through its authored town option.
            key('Left')
            key('Return');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            assert digest(saved)==digest(args.town_save),'未保存前來源槽被改寫'
            key('F5');hd_saved=saved.read_bytes()
            key('F2');wait(lambda s:s.get('hd_theme') is False)
            key('F5');assert saved.read_bytes()==hd_saved,'兩主題保存結果不同'
            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            shot('faithful-hotel-after-church-load')
            panels=panel_probe.validate() if panel_probe else None
            if args.panel_transitions:
                assert hd_saved==args.town_save.read_bytes(),'狀態轉場改變來源保存'
                restart_title_load('hotel_ch02',1000,'hotel-after-status-new-process-load')
                assert saved.read_bytes()==hd_saved,'新程序LOAD改寫保存'
            report={'schema':'fd2-hd-status-font-gui-v1' if args.status else 'fd2-hd-church-font-gui-v1','status':'PASS',
                    'panel_transitions':panels,
                    'new_process_title_load':args.panel_transitions,
                    'binary_sha256':digest(args.binary),'town_save_sha256':digest(args.town_save),
                    'saved_sha256':digest(saved),'theme_save_bytes_equal':True,
                    'source_save_bytes_equal':digest(saved)==digest(args.town_save),
                    'screenshots':shots,'operations':operations,'recording_created':False,
                    'scope':('另驗Sol狀態與ID9命令、F2、縮放與說明遮罩；' if args.status else '')+'正常START後F9接續既有hotel_ch02，經城鎮進教會；四語名冊、復活列表／問句與取消、交付物品列表與取消、F2、縮放／說明遮罩，回旅館F5／F9。四語名冊沿既有譯名，其他原生文字未新增翻譯。不宣稱連續戰役或原版視覺對拍。',
                    'elapsed_seconds':time.monotonic()-started}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        if args.hotel:
            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            for locale in ['zh-Hant','zh-Hans','ja','en']:
                wait(lambda s:s.get('locale')==locale and s.get('hd_text_runs',0)>0)
                shot('hotel-menu-'+locale);key('F4')
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-hotel-menu')
            key('F2');wait(lambda s:s.get('hd_theme') is True)
            key('Right');key('Return')
            wait(lambda s:s.get('hotel_mode')=='slots')
            for slot in range(4):
                wait(lambda s:s.get('hotel_slot')==slot and s.get('hd_text_runs',0)>0)
                shot('hotel-slots-'+str(slot))
                if slot<3:key('Down')
            key('Up');key('Up');key('Return')
            wait(lambda s:s.get('hotel_mode')=='saved');shot('hotel-saved')
            secondary=saved.with_name('fd2_save_1.json')
            hd_saved=secondary.read_bytes()
            key('Return');key('Right');key('Return');key('Down');key('Return')
            wait(lambda s:s.get('hotel_mode')=='loaded' and s.get('gold')==1000)
            shot('hotel-loaded')
            assert secondary.read_bytes()==hd_saved,'LOAD修改了保存bytes'
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-hotel-loaded')
            key('Return');key('Left');key('Return');key('Down');key('Return')
            wait(lambda s:s.get('hotel_mode')=='saved')
            assert secondary.read_bytes()==hd_saved,'兩主題保存結果不同'
            key('F2');wait(lambda s:s.get('hd_theme') is True);shot('hd-hotel-saved-restored')
            assert digest(saved)==digest(args.town_save),'來源槽被改寫'
            report={'schema':'fd2-hd-hotel-font-gui-v1','status':'PASS',
                    'binary_sha256':digest(args.binary),'town_save_sha256':digest(args.town_save),
                    'saved_sha256':digest(secondary),'theme_save_bytes_equal':True,
                    'screenshots':shots,'operations':operations,'recording_created':False,
                    'scope':'正常START後F9接續既有hotel_ch02槽，正常旅館選槽、保存與服務LOAD；四語保留原有原生文字，不新增翻譯。不宣稱連續戰役或原版視覺對拍。',
                    'elapsed_seconds':time.monotonic()-started}
            (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
            return
        for locale in ['zh-Hant','zh-Hans','ja','en']:
            settled_dialogue();shot('native-dialogue-'+locale)
            assert latest(audit).get('locale')==locale
            if locale=='zh-Hant':
                before=latest(audit)
                key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-native-dialogue')
                key('F2');wait(lambda s:s.get('hd_theme') is True);shot('hd-native-restored')
                after=latest(audit)
                for field in ['node','dialogue_speaker','dialogue_progress','dialogues']:
                    assert before[field]==after[field],(field,before,after)
            key('F4')
        key('F1');wait(lambda s:s.get('help_visible') is True);shot('hd-help')
        key('Escape');key('F7');key('F7');key('F1');shot('hd-help-largest-font')
        key('Escape');key('F7');key('F7');key('F7');key('F1')
        command('xdotool','windowsize',window,'1600','1000');shot('hd-help-resized')
        command('xdotool','windowsize',window,'1920','1200')
        key('Escape')
        key('F9');wait(lambda s:s.get('node')=='hotel_ch02')
        key('Escape');wait(lambda s:s.get('node')=='town_ch02')
        for _ in range(3):key('Left')
        key('Return');wait(lambda s:s.get('node')=='shop_ch02_item')
        time.sleep(2);shot('shop-menu-zh-Hant')
        key('Return');time.sleep(2)
        for locale in ['zh-Hant','zh-Hans','ja','en']:
            wait(lambda s:s.get('locale')==locale and s.get('shop_mode')=='purchase')
            shot('shop-list-'+locale);key('F4')
        # One ordinary purchase, followed by a state-based save receipt.
        for _ in range(3):key('Return');time.sleep(.5)
        wait(lambda s:s.get('gold')==990 and s.get('shop_mode')=='purchase')
        shot('shop-after-purchase')
        key('F2');wait(lambda s:s.get('hd_theme') is False);shot('faithful-shop')
        key('F1');wait(lambda s:s.get('help_visible') is True);shot('faithful-help')
        key('Escape')
        key('F2');wait(lambda s:s.get('hd_theme') is True);shot('hd-shop-restored')
        assert digest(saved)==digest(args.town_save),'未保存前來源槽被改寫'
        report={'schema':'fd2-hd-font-gui-v1','status':'PASS','binary_sha256':digest(args.binary),
                'town_save_sha256':digest(args.town_save),'screenshots':shots,'operations':operations,
                'scope':'正常START序章首句四語、說明頁；明示F9接續hotel_ch02後正常城鎮與商店。非連續全戰役，不宣稱原版視覺對拍。',
                'recording_created':False,'elapsed_seconds':time.monotonic()-started}
        (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    finally:
        if timed_probe and timed_result is None:timed_probe.finish()
        world_stop.set()
        if world_thread and world_thread.ident is not None:world_thread.join(timeout=15)
        if args.world:
            (args.out/'battle-observation.json').write_text(json.dumps(
                {'status':'OBSERVATION-ONLY','battle_hd_scenes':battle_hd_scenes,
                 'battle_art_samples':battle_art_samples,'battle_end_art_samples':battle_end_art_samples,
                 'battle_end_command_samples':battle_end_command_samples,
                 'battle_approach_steps':battle_approach_steps,'battle_post_dialogue_cleanup':battle_post_dialogue_cleanup,
                 'catalog_sha256':digest(args.assets/'assets/themes/modern/catalog.json'),
                 'binary_sha256':digest(args.binary),'source_save_sha256':digest(args.town_save),
                 'gui_tool_sha256':tool_source_sha256,'tool_source_unchanged':digest(Path(__file__))==tool_source_sha256,
                 'recording_created':False},ensure_ascii=False,indent=2)+'\n')
            (args.out/'world-observation.json').write_text(json.dumps(
                {'accepted_walk_gpu':world_shots,'rejected_walk_gpu':world_rejected,
                 'gui_tool_sha256':tool_source_sha256,'tool_source_unchanged':digest(Path(__file__))==tool_source_sha256,
                 'recording_created':False},ensure_ascii=False,indent=2)+'\n')
        if panel_probe:
            panel_probe.finish()
            if not (args.out/'panel-transition-observation.json').exists():panel_probe.save()
        finish_gold_roll()
        if args.gold_roll:
            (args.out/'gold-roll-observation.json').write_text(json.dumps(
                {'samples':gold_shots,'errors':gold_errors},ensure_ascii=False,indent=2)+'\n')
        finish_transitions(False)
        if args.transition_frames:
            (args.out/'transition-observation.json').write_text(json.dumps(
                {'accepted':transition_shots,'rejected':transition_rejected,'errors':transition_errors,
                 'gpu_validation_performed':transition_gpu_validated},
                ensure_ascii=False,indent=2)+'\n')
        if args.terrain:
            (args.out/'terrain-observation.json').write_text(json.dumps(
                {'screenshots':shots,'gui_tool_sha256':tool_source_sha256,
                 'tool_source_unchanged':digest(Path(__file__))==tool_source_sha256,
                 'recording_created':False},ensure_ascii=False,indent=2)+'\n')
        (args.out/'operations.json').write_text(json.dumps(operations,ensure_ascii=False,indent=2)+'\n')
        for proc in reversed(processes):
            if proc.poll() is None:proc.terminate()
        for proc in reversed(processes):
            try:proc.wait(timeout=5)
            except subprocess.TimeoutExpired:proc.kill();proc.wait()
        for stream in streams:stream.close()


if __name__=='__main__':main()
