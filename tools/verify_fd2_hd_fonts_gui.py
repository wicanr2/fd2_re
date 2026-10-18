#!/usr/bin/env python3
"""Docker 內以正常 START、F4、F9 與商店按鍵抽驗 HD 字型，不錄影。"""
import argparse
import json
import os
from pathlib import Path
import subprocess
import time

from verify_fd2_hd_gui import digest, latest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'assets', 'original-pack', 'modern-pack', 'town-save', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    facility = parser.add_mutually_exclusive_group()
    facility.add_argument('--hotel', action='store_true', help='只抽驗旅館固定文字、四槽及保存讀取')
    facility.add_argument('--church', action='store_true', help='抽驗教會即時文字、四語名冊、取消與保存')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and args.binary.is_file()
    assert (args.original_pack/'manifest.json').is_file()
    assert json.loads(args.town_save.read_text())['node'] == 'hotel_ch02'
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
    Path(env['XDG_RUNTIME_DIR']).mkdir(mode=0o700)
    Path(env['XDG_CONFIG_HOME']).mkdir()
    Path(env['XDG_CACHE_HOME']).mkdir()
    alsa=args.out/'asoundrc'
    alsa.write_text('pcm.!default { type pulse }\nctl.!default { type pulse }\n')
    env['ALSA_CONFIG_PATH']=str(alsa)
    processes,streams,shots,operations=[],[],{},[]
    started=time.monotonic()
    window=None

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
                'Up':'ArrowUp','Down':'ArrowDown'}.get(name,name)
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
                time.sleep(.1)
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
            time.sleep(.1)
        if not release_ack:raise TimeoutError('放鍵無收據 '+name)
        released_frame=max(ack['frame'],latest(audit).get('frame',0))
        wait(lambda s:s.get('frame',0)>=released_frame+2)
        operations.append({'key':name,'ack':ack,'release_ack':release_ack})
        return ack

    def shot(name):
        # Wait for subsequent GPU presentation, independently of the input ACK.
        time.sleep(1)
        path=args.out/(name+'.png')
        command('import','-window',window,str(path))
        shots[path.name]={'sha256':digest(path),'state':latest(audit)}
        print('CAPTURE '+path.name,flush=True)

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
        start(['Xvfb',':97','-screen','0','1920x1200x24','-nolisten','tcp'],'xvfb.log')
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
        key('Return')
        wait(lambda s:not s.get('title_phase') and s.get('dialogue_native') is True, timeout=180)
        if args.church:
            def church(mode):
                return wait(lambda s:s.get('node')=='church_ch02' and s.get('church_mode')==mode
                            and s.get('church_ui_busy') is False and s.get('hd_text_runs',0)>0)

            key('F9');wait(lambda s:s.get('node')=='hotel_ch02' and s.get('hotel_mode')=='menu')
            key('Escape');wait(lambda s:s.get('node')=='town_ch02')
            for _ in range(4):key('Left')
            key('Return');church('menu');shot('church-menu')
            key('Return');church('status_roster')
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
            report={'schema':'fd2-hd-church-font-gui-v1','status':'PASS',
                    'binary_sha256':digest(args.binary),'town_save_sha256':digest(args.town_save),
                    'saved_sha256':digest(saved),'theme_save_bytes_equal':True,
                    'source_save_bytes_equal':digest(saved)==digest(args.town_save),
                    'screenshots':shots,'operations':operations,'recording_created':False,
                    'scope':'正常START後F9接續既有hotel_ch02，經城鎮進教會；四語名冊、復活列表／問句與取消、交付物品列表與取消、F2、縮放／說明遮罩，回旅館F5／F9。四語名冊沿既有譯名，其他原生文字未新增翻譯。不宣稱連續戰役或原版視覺對拍。',
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
        (args.out/'operations.json').write_text(json.dumps(operations,ensure_ascii=False,indent=2)+'\n')
        for proc in reversed(processes):
            if proc.poll() is None:proc.terminate()
        for proc in reversed(processes):
            try:proc.wait(timeout=5)
            except subprocess.TimeoutExpired:proc.kill();proc.wait()
        for stream in streams:stream.close()


if __name__=='__main__':main()
