#!/usr/bin/env python3
"""Docker 內以正常 START、F4、F9 與商店按鍵抽驗 HD 字型，不錄影。"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import threading
import time

from verify_fd2_hd_gui import digest, latest
from verify_fd2_hd_panel_transitions import PanelTransitionProbe
from verify_fd2_hd_timed_dialogue import TimedDialogueProbe


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'assets', 'original-pack', 'modern-pack', 'town-save', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
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
    parser.add_argument('--terrain', action='store_true', help='普通方向鍵移鏡頭至首章房屋，抽驗HD地形與F2；需--battle-status')
    parser.add_argument('--timed-dialogue', action='store_true', help='加驗計時對話開收框、穩定HD文字與F2；需--world與明示章前EXP校準槽')
    args = parser.parse_args()
    tool_source_sha256=digest(Path(__file__))
    assert not args.world or args.battle_status
    assert not args.terrain or args.battle_status
    assert not args.timed_dialogue or args.world
    assert not args.transition_frames or args.preparation
    assert not args.panel_transitions or args.status or args.items
    assert Path('/.dockerenv').is_file() and args.binary.is_file()
    assert (args.original_pack/'manifest.json').is_file()
    assert json.loads(args.town_save.read_text())['node'] == ('battle_ch01' if args.battle_status else 'hotel_ch02')
    assert not args.out.exists()
    args.out.mkdir()
    audit = args.out/'input-audit.jsonl'
    saved = args.out/'user-data/fd2_re/fd2_save.json'
    saved.parent.mkdir(parents=True)
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
        assert args.preparation and args.native_save.is_file()
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
            if time.monotonic()-started>660:raise TimeoutError('整輪逾時')
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
        key('Return');wait(lambda s:not s.get('title_phase') and s.get('node')==node and s.get('gold')==gold)
        if args.battle_status:
            loaded_frame=latest(audit)['frame']
            wait(lambda s:s.get('frame',0)>=loaded_frame+20)
            for _ in range(120):
                state=latest(audit)
                if not state.get('dialogues') and not state.get('banner'):break
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
        for _ in range(600):
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
        if args.battle_status:
            key('Down');key('Return')
            wait(lambda s:s.get('title_phase')=='loadslots');shot('battle-title-load')
            key('Return')
            wait(lambda s:not s.get('title_phase') and s.get('node')=='battle_ch01')
            for _ in range(120):
                state=latest(audit)
                if not state.get('dialogues') and not state.get('banner'):break
                key('Return')
            else:raise TimeoutError('首章LOAD對話未結束')
            wait(lambda s:not s.get('dialogues') and not s.get('banner'))
            shot('battle-hd-field')
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
                pan_to([15,5]);pan_to([10,5])
                wait(lambda s:len(s.get('hd_world_terrain',[]))>=18)
                assert latest(audit)['units']==before_pan['units'],'單純移鏡頭改變單位'
                shot('terrain-hd-house')
                house=latest(audit)
                key('F2');wait(lambda s:s.get('hd_theme') is False and not s.get('hd_world'))
                shot('terrain-faithful-house')
                assert latest(audit)['units']==house['units'] and latest(audit)['view']==house['view']
                key('F2');wait(lambda s:s.get('hd_world') is True and len(s.get('hd_world_terrain',[]))>=18)
                shot('terrain-hd-house-restored')
                assert latest(audit)['units']==house['units'] and latest(audit)['view']==house['view']
                key('F9');wait(lambda s:s.get('node')=='battle_ch01')
                for _ in range(120):
                    state=latest(audit)
                    if not state.get('dialogues') and not state.get('banner'):break
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
                        if not state.get('battle_event') and not state.get('dialogues') and not state.get('banner'):
                            return
                        key('Return')
                    raise TimeoutError('普通LOAD戰場對話尚未結束')
                def focus_own(index):
                    state=latest(audit)
                    own=[u for u in state['units'] if u['camp']==0 and u['on_field'] and u['hp']>0]
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
                key('F9');wait(lambda s:s.get('node')=='battle_ch01' and not s.get('ring') and not s.get('move_preview'))
                settle_world_battle()
                def ordinary_physical_attack(prefix):
                    focus_own(2)
                    attacker=latest(audit)
                    shot(prefix+'-attacker-focused')
                    key('Return');wait(lambda s:s.get('move_preview') is True)
                    state=latest(audit)
                    occupied={(u['x'],u['y']) for u in state['units'] if u['on_field'] and u['hp']>0}
                    enemies=[u for u in state['units'] if u['camp']==2 and u['on_field'] and u['hp']>0]
                    choices=[]
                    for target in enemies:
                        for cell in state.get('move_reachable',[]):
                            if tuple(cell) not in occupied and abs(cell[0]-target['x'])+abs(cell[1]-target['y'])==1:
                                choices.append((abs(cell[0]-state['cursor'][0])+abs(cell[1]-state['cursor'][1]),cell,target))
                    assert choices,'正式可達集合沒有相鄰敵人的安全測試位置'
                    _,cell,target=min(choices,key=lambda value:(value[0],value[1][1],value[1][0]))
                    def move_cursor(destination):
                        current=latest(audit)['cursor']
                        for _ in range(abs(destination[0]-current[0])):key('Right' if destination[0]>current[0] else 'Left')
                        current=latest(audit)['cursor']
                        for _ in range(abs(destination[1]-current[1])):key('Down' if destination[1]>current[1] else 'Up')
                        assert latest(audit)['cursor']==destination
                    move_cursor(cell);key('Return')
                    wait(lambda s:s.get('ring') and not s.get('walking') and s.get('ring_phase')=='open')
                    shot(prefix+'-attack-ring')
                    key('Return');wait(lambda s:not s.get('ring'))
                    move_cursor([target['x'],target['y']]);shot(prefix+'-attack-target')
                    key('Return');wait(lambda s:s.get('battle_scene') is True)
                    shot(prefix+'-physical-scene')
                    wait(lambda s:not s.get('battle_scene') and not s.get('walking') and s.get('hd_world'))
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
                key('F9');wait(lambda s:s.get('node')=='battle_ch01' and not s.get('battle_scene'))
                settle_world_battle()
                assert not latest(audit).get('level_up_dialogue') and not latest(audit).get('move_preview')
                shot('world-hd-load-owner-cleared')
                ordinary_physical_attack('world-hd-replay')
                for _ in range(120):
                    state=latest(audit)
                    if not state.get('level_up_dialogue') and not state.get('dialogues') and not state.get('battle_event'):break
                    key('Return')
                else:raise TimeoutError('普通戰鬥經驗對話尚未結束')
                wait(lambda s:s.get('hd_world') and not s.get('battle_scene'))
                shot('world-hd-physical-return')
                if timed_probe:timed_result=timed_probe.finish(validate=True)
                key('F9')
                settle_world_battle()
                focus_own(0)
                wait(lambda s:s.get('hd_world') is True)
                shot('world-hd-source-reloaded')
                focused=latest(audit)
            key('F2');wait(lambda s:s.get('hd_theme') is False);shot('battle-faithful-focused')
            faithful_focused=latest(audit)
            assert faithful_focused['units']==focused['units'] and faithful_focused['view']==focused['view']
            assert not faithful_focused.get('player_status_phase'),'F2誤開角色狀態'
            key('F2');wait(lambda s:s.get('hd_theme') is True);shot('battle-hd-focused-restored')
            assert latest(audit)['units']==focused['units'] and latest(audit)['view']==focused['view']
            assert not latest(audit).get('player_status_phase'),'F2返回誤開角色狀態'
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
            key('F2');wait(lambda s:s.get('hd_theme') is False)
            key('F9');wait(lambda s:s.get('node')=='battle_ch01')
            faithful_loaded=latest(audit)
            key('F5');assert saved.read_bytes()==hd_saved,'兩主題保存不同'
            key('F2');wait(lambda s:s.get('hd_theme') is True)
            key('F9');wait(lambda s:s.get('node')=='battle_ch01')
            hd_loaded=latest(audit)
            assert faithful_loaded['units']==hd_loaded['units'],'兩主題LOAD結果不同'
            assert saved.read_bytes()==hd_saved
            restart_title_load('battle_ch01',hd_loaded['gold'],'battle-cold-title-load')
            assert saved.read_bytes()==hd_saved
            assert digest(Path(__file__))==tool_source_sha256,'GUI工具執行中來源被修改'
            report={'schema':'fd2-hd-battle-status-gui-v1','status':'PASS',
                    'gui_tool_sha256':tool_source_sha256,
                    'world_runtime_checked':args.world,'world_walk_gpu':world_shots,'world_walk_rejected':world_rejected,
                    'timed_dialogue_gpu':timed_result,
                    'binary_sha256':digest(args.binary),'source_save_sha256':digest(args.town_save),
                    'saved_sha256':digest(saved),'screenshots':shots,'operations':operations,
                    'initial':initial,'focused':focused,'returned':returned,
                    'outside_map_gpu':outside,'new_process_title_load':True,
                    'source_save_bytes_equal':saved.read_bytes()==args.town_save.read_bytes(),
                    'theme_save_bytes_equal':True,'loaded_units_equal':True,
                    'recording_created':False,'elapsed_seconds':time.monotonic()-started,
                    'scope':('正常標題LOAD既有普通START／F5首章JSON，Z聚焦、Home檢視、F2、狀態與命令、返回；'
                             + ('含普通走行、指令環、取消、兩次物理攻擊、經驗對話中F9及正常戰場返回。' if args.world else '')
                             + '不注入座標或狀態，不宣稱原版對拍或完整戰役。')}
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
        (args.out/'operations.json').write_text(json.dumps(operations,ensure_ascii=False,indent=2)+'\n')
        for proc in reversed(processes):
            if proc.poll() is None:proc.terminate()
        for proc in reversed(processes):
            try:proc.wait(timeout=5)
            except subprocess.TimeoutExpired:proc.kill();proc.wait()
        for stream in streams:stream.close()


if __name__=='__main__':main()
