#!/usr/bin/env python3
"""以正常F5存檔驗證非原生對話HD；可選標題LOAD及重新F5冷讀。Docker內執行，不錄影。"""
import argparse
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import time

from verify_fd2_hd_gui import digest, latest

ROOT = Path(__file__).resolve().parent.parent


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'assets', 'original-pack', 'modern-pack', 'source-receipt', 'out']:
        parser.add_argument('--'+name, type=Path, required=True)
    parser.add_argument('--load-entry', choices=['quick', 'title'], default='quick')
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and args.binary.is_file()
    assert (args.assets/'assets/locales/zh-Hant/pack.json').is_file()
    assert (args.original_pack/'manifest.json').is_file() and args.modern_pack.is_dir()
    assert not args.out.exists(), '禁止覆寫既有驗證'
    source = json.loads(args.source_receipt.read_text())
    source_run = ROOT/Path(source['normal_start_receipt']['file']).parent
    saved_files = source['normal_start_receipt']['report']['save_reload']['files']
    assert len(saved_files) == 1
    relative, saved_sha = next(iter(saved_files.items()))
    saved_input = source_run/relative
    assert digest(saved_input) == saved_sha
    assert json.loads(saved_input.read_text())['node'] == 'battle_ch01'
    args.out.mkdir()
    assert args.out.stat().st_uid == os.getuid()
    saved = args.out/'user-data/fd2_re/fd2_save.json'
    saved.parent.mkdir(parents=True)
    saved.write_bytes(saved_input.read_bytes())
    catalog_path = ROOT/'remake/assets/themes/modern/catalog.json'
    catalog = json.loads(catalog_path.read_text())
    portrait = next(a for a in catalog['assets'] if a['role']=='story_portrait_hd_frame' and a['speaker_id']==0)
    portrait_path = args.modern_pack/portrait['file']
    assert digest(portrait_path) == portrait['sha256']
    audit = args.out/'input-audit.jsonl'
    env = dict(os.environ, DISPLAY=':97', XDG_RUNTIME_DIR='/tmp/fd2-hd-dialogue-pulse',
               XDG_DATA_HOME=str(args.out/'user-data'), XDG_CONFIG_HOME=str(args.out/'config'),
               XDG_CACHE_HOME=str(args.out/'cache'), LIBGL_ALWAYS_SOFTWARE='1',
               GALLIUM_DRIVER='llvmpipe', GOMAXPROCS='2', LP_NUM_THREADS='2',
               PULSE_SERVER='unix:/tmp/fd2-hd-dialogue-pulse/pulse/native',
               FD2_MODERN_THEME_PACK=str(args.modern_pack), FD2_ASSET_PACK=str(args.original_pack),
               FD2_INPUT_AUDIT_LOG=str(audit))
    for name in ['FD2_THEME','FD2_TITLE','FD2_CAMPAIGN','FD2_CAMP_NODE','FD2_SHOT',
                 'FD2_SHOT_FRAME','FD2_NATIVE_SAVE','FD2_SAVE_CURRENT','FD2_ENDING_PREFIX']:
        env.pop(name,None)
    Path(env['XDG_RUNTIME_DIR']).mkdir(mode=0o700)
    Path(env['XDG_CONFIG_HOME']).mkdir()
    Path(env['XDG_CACHE_HOME']).mkdir()
    alsa = args.out/'asoundrc'
    alsa.write_text('pcm.!default { type pulse }\nctl.!default { type pulse }\n')
    env['ALSA_CONFIG_PATH'] = str(alsa)
    processes, streams, screenshots, metrics = [], [], {}, []
    started = time.monotonic()
    window = None

    def command(*items):
        return subprocess.check_output(items,env=env,stderr=subprocess.DEVNULL,text=True)

    def start(items, name):
        stream = (args.out/name).open('w'); streams.append(stream)
        process = subprocess.Popen(items,env=env,cwd=args.assets,stdout=stream,stderr=stream)
        processes.append(process); return process

    def wait_state(predicate, timeout=60):
        deadline = time.monotonic()+timeout
        while time.monotonic()<deadline:
            state = latest(audit)
            if state.get('error'): raise RuntimeError(state['error'])
            if predicate(state): return state
            if time.monotonic()-started>540: raise TimeoutError('整輪超過有界時間')
            time.sleep(.1)
        raise TimeoutError(f'未取得預期狀態：{latest(audit)}')

    def key(name):
        before = latest(audit)
        command('xdotool','keydown',name)
        mapped = {'Return':'Enter','Down':'ArrowDown'}.get(name,name)
        deadline = time.monotonic()+60
        state = None
        try:
            # A short press can be entirely inside a synchronous resource/GPU
            # upload. Hold until the real Update acknowledges this key once.
            while time.monotonic()<deadline:
                if audit.exists():
                    with audit.open('rb') as stream:
                        stream.seek(max(0,audit.stat().st_size-131072))
                        rows=stream.read().splitlines()
                    for row in reversed(rows):
                        try: candidate=json.loads(row)
                        except ValueError: continue
                        if candidate.get('frame',0)>before.get('frame',0) and mapped in (candidate.get('keys') or []):
                            state=candidate;break
                if state is not None:break
                if latest(audit).get('error'):raise RuntimeError(latest(audit)['error'])
                time.sleep(.1)
        finally:
            command('xdotool','keyup',name)
        if state is None:raise TimeoutError(f'{mapped}沒有完成按鍵收據')
        # Observe subsequent Updates before another press, so key-up is consumed
        # even when changing window size stalls the first GPU presentation.
        wait_state(lambda s:s.get('frame',0)>=state['frame']+2)
        with (args.out/'operations.jsonl').open('a') as stream:
            stream.write(json.dumps({'seconds':time.monotonic()-started,'key':name,'state':state},ensure_ascii=False)+'\n')
        return state

    def shot(name):
        path = args.out/name
        command('import','-window',window,str(path))
        screenshots[name] = {'sha256':digest(path)}
        return path

    def window_size(w,h):
        for _ in range(100):
            geometry = command('xdotool','getwindowgeometry','--shell',window)
            if f'WIDTH={w}\n' in geometry and f'HEIGHT={h}\n' in geometry: return
            time.sleep(.1)
        raise AssertionError(geometry)

    def mae(reference, picture, crop):
        result = subprocess.run(['compare','-metric','MAE',str(reference),str(picture)+'['+crop+']','null:'],
                                env=env,capture_output=True,text=True)
        assert result.returncode in (0,1), result.stderr
        found = re.search(r'\(([0-9.eE+-]+)\)',result.stderr)
        assert found,result.stderr
        return float(found[1])

    reference = args.out/'sol-hd-504-mirrored.png'

    def hd_shot(name):
        wait_state(lambda s:s.get('hd_theme') and s.get('dialogue_native') is False
                   and s.get('dialogue_speaker')==0 and s.get('dialogue_phase')==0
                   and s.get('hd_overlay_rect')==[16,213,168,168] and s.get('hd_overlay_mirrored') is True)
        window_size(1920,1200)
        for _ in range(120):
            path = shot(name)
            error = mae(reference,path,'504x504+48+639')
            if error <= .004:
                metrics.append({'file':name,'mae_normalized':error,'crop':'504x504+48+639',
                                'reference':reference.name,'threshold':.004})
                print(f'GPU HD PASS {name} MAE={error}',flush=True)
                return path
            time.sleep(.25)
        raise AssertionError(f'HD圖未呈現：MAE={error}')

    def stable(state):
        return {k:state.get(k) for k in ['node','turn','native_round','units','dialogues','dialogue_speaker','dialogue_native']}

    def launch_load(modern, name):
        nonlocal window,audit
        audit = args.out/(name+'-audit.jsonl')
        env['FD2_INPUT_AUDIT_LOG'] = str(audit)
        if modern: env['FD2_THEME']='modern-handpainted-a'
        else: env.pop('FD2_THEME',None)
        game = start([str(args.binary)],name+'-game.log')
        for _ in range(600):
            if game.poll() is not None: raise RuntimeError('遊戲建立視窗前退出')
            try: window=command('xdotool','search','--name','炎龍騎士團').splitlines()[0];break
            except (subprocess.CalledProcessError,IndexError):time.sleep(.1)
        else: raise TimeoutError('沒有遊戲視窗')
        command('xdotool','windowmove',window,'0','0');command('xdotool','windowfocus',window)
        wait_state(lambda s:bool(s.get('title_phase')))
        for _ in range(160):
            if latest(audit).get('title_phase')=='menu':break
            # Audit rows can lag a cutscene-to-menu transition. Escape skips
            # eligible intro scenes and cannot accidentally confirm START.
            key('Escape')
        else:raise TimeoutError('開場未抵達標題選單')
        if args.load_entry == 'title':
            key('Down')
            wait_state(lambda s:s.get('title_phase')=='menu' and s.get('title_selection')==1)
            key('Return')
            wait_state(lambda s:s.get('title_phase')=='loadslots')
            shot(name+'-slots.png')
            key('Return')
        else:
            key('Return')
            wait_state(lambda s:s.get('title_phase')=='' and s.get('dialogue_native') is True
                       and s.get('dialogue_speaker')==0 and s.get('dialogue_phase')==0)
            key('F9')
        loaded = wait_state(lambda s:s.get('node')=='battle_ch01' and s.get('title_phase')==''
                            and s.get('dialogue_native') is False and s.get('dialogue_phase')==0)
        assert loaded.get('hd_theme') is modern
        return game,loaded

    def stop(game):
        if game.poll() is None:game.terminate()
        try:game.wait(timeout=5)
        except subprocess.TimeoutExpired:game.kill();game.wait()

    try:
        start(['Xvfb',':97','-screen','0','1920x1200x24','-nolisten','tcp'],'xvfb.log')
        pulse=start(['pulseaudio','--daemonize=no','--exit-idle-time=-1','--use-pid-file=no'],'pulse.log')
        for _ in range(100):
            try:command('pactl','info');break
            except subprocess.CalledProcessError:time.sleep(.1)
            if pulse.poll() is not None:raise RuntimeError('PulseAudio初始化失敗')
        else:raise TimeoutError('PulseAudio沒有準備完成')
        command('pactl','load-module','module-null-sink','sink_name=fd2_hd_dialogue','rate=44100','channels=2')
        command('pactl','set-default-sink','fd2_hd_dialogue')
        command('convert',str(portrait_path),'-filter','Triangle','-resize','504x504!','-flop',str(reference))
        baseline=ROOT/source['screenshots']['first-chapter.png']['file']
        assert digest(baseline)==source['screenshots']['first-chapter.png']['sha256']
        old_error=mae(reference,baseline,'504x504+48+639')
        game,faithful=launch_load(False,'faithful')
        window_size(1280,800);time.sleep(1);shot('faithful-load.png')
        key('F2');first=hd_shot('hd-enter.png')
        assert stable(latest(audit))==stable(faithful)
        assert old_error>2*metrics[-1]['mae_normalized'] and old_error>.004
        key('F1');wait_state(lambda s:s.get('help_visible') is True)
        for _ in range(100):
            help_picture=shot('hd-help.png')
            if mae(reference,help_picture,'504x504+48+639')>.02:break
            time.sleep(.25)
        else:raise AssertionError('說明遮罩未實際蓋住HD頭像')
        key('Escape');hd_shot('hd-help-closed.png')
        for w,h in [(1600,1000),(1920,1080)]:
            command('xdotool','windowsize',window,str(w),str(h));window_size(w,h)
            time.sleep(.5);shot(f'hd-resize-{w}x{h}.png')
        command('xdotool','windowsize',window,'1920','1200');hd_shot('hd-size-restored.png')
        key('F2');window_size(1280,800)
        wait_state(lambda s:not s.get('hd_theme') and 'hd_overlay_rect' not in s)
        shot('faithful-return.png')
        key('F2');hd_shot('hd-return.png')
        key('F9');hd_shot('hd-quickload.png')
        quick=latest(audit);assert stable(quick)==stable(faithful)
        assert digest(saved)==saved_sha
        stop(game)
        game,cold=launch_load(True,'hd-cold')
        hd_shot('hd-cold-load.png');assert stable(cold)==stable(faithful)
        assert digest(saved)==saved_sha
        fresh_save = None
        if args.load_entry == 'title':
            saved_mtime = saved.stat().st_mtime_ns
            key('F5')
            assert saved.stat().st_mtime_ns > saved_mtime, 'F5沒有完成新存檔寫入'
            fresh = args.out/'fresh-f5.json'
            fresh.write_bytes(saved.read_bytes())
            fresh_sha = digest(fresh)
            stop(game)
            game,fresh_cold = launch_load(False,'fresh-f5-cold')
            assert stable(fresh_cold)==stable(faithful)
            assert digest(saved)==fresh_sha
            shot('fresh-f5-cold.png')
            fresh_save = {'file':str(fresh),'sha256':fresh_sha,'cold_loaded':fresh_cold,
                          'unchanged_after_cold_load':True}
        report={'schema_version':1,'status':'PLAYER-E2-LIMITED-GPU-REVIEW-REQUIRED',
                'method':('兩個新程序由普通標題LOAD冷讀原樣正常F5存檔；F2/F1/Esc/F9後F5重新保存，第三個新程序標題LOAD冷讀。'
                          if args.load_entry == 'title' else '兩個新程序從正式開場START，在原生對話用普通F9冷讀原樣正常F5存檔；普通F2/F1/Esc/F9，無章節注入、改血或強化。'),
                'load_entry':args.load_entry,'fresh_save':fresh_save,
                'source_receipt':str(args.source_receipt),'source_receipt_sha256':digest(args.source_receipt),
                'source_save':str(saved_input),'source_save_sha256':saved_sha,'save_unchanged':True,
                'binary_sha256':digest(args.binary),'driver_sha256':digest(Path(__file__)),
                'catalog_sha256':digest(catalog_path),'source_portrait_sha256':digest(portrait_path),
                'reference_method':'ImageMagick Triangle放大480至504後鏡像；GPU採線性取樣，允許明示的插值／量化差異，不宣稱零像素差異。',
                'reference_sha256':digest(reference),'old_80_frame_mae':old_error,'gpu_metrics':metrics,
                'faithful_loaded':faithful,'hd_quick_loaded':quick,'hd_cold_loaded':cold,
                'screenshots':screenshots,'wall_seconds':time.monotonic()-started,'recordings_created':False,
                'limits':'正常玩家只驗首章非原生下框與JSON節點讀取；上框幾何為獨立E1，非全戰役／跨平台／原版對拍。'}
        (args.out/'hd-dialogue-gui.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        print('正常JSON冷讀、非原生對話HD、切換及F9通過；入口='+args.load_entry,flush=True)
    finally:
        for process in reversed(processes):stop(process)
        for stream in streams:stream.close()


if __name__=='__main__':
    main()
