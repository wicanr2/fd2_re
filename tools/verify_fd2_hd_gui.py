#!/usr/bin/env python3
"""以正常START驗證HD序章，直接擷取GPU視窗；不建立錄影。Docker內執行。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import time

ROOT = Path(__file__).resolve().parent.parent


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def latest(path):
    if not path.exists():
        return {}
    with path.open('rb') as stream:
        stream.seek(max(0, path.stat().st_size-131072))
        for row in reversed(stream.read().splitlines()):
            try:
                return json.loads(row)
            except ValueError:
                pass
    return {}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--assets', type=Path, required=True)
    parser.add_argument('--original-pack', type=Path, required=True)
    parser.add_argument('--modern-pack', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--switch-theme', action='store_true', help='由忠實主題正常START，途中驗證雙向F2及首章F5/F9。')
    parser.add_argument('--switch-only', action='store_true', help='限正常START至父王對話的F2／遮罩補抽，不重跑完整序章。')
    args = parser.parse_args()
    assert not args.switch_only or args.switch_theme
    assert Path('/.dockerenv').exists()
    assert args.binary.is_file() and (args.assets/'assets/locales/zh-Hant/pack.json').is_file()
    assert (args.original_pack/'manifest.json').is_file() and args.modern_pack.is_dir()
    assert not args.out.exists(), '禁止覆寫既有驗收'
    args.out.mkdir()
    assert args.out.stat().st_uid == os.getuid()
    audit = args.out/'input-audit.jsonl'
    catalog_path = args.assets/'assets/themes/modern/catalog.json'
    catalog = json.loads(catalog_path.read_text())
    hd = {a['speaker_id']: a for a in catalog['assets'] if a['role']=='story_portrait_hd_frame'}
    coverage = json.loads((ROOT/'docs/data/fd2-modern-intro-coverage-20261018.json').read_text())
    # Line numbers restart in each caller. Keep the original handler address;
    # (story, line) alone silently maps palace king/queen lines to later scenes.
    rows = {(r['handler_address'],Path(r['story']).name,r['line']):r for r in coverage['rows']}
    assert len(rows)==len(coverage['rows']), '序章consumer定位鍵重複'
    processes, streams = [], []
    env = dict(os.environ, DISPLAY=':97', XDG_RUNTIME_DIR='/tmp/fd2-hd-pulse',
               XDG_DATA_HOME=str(args.out/'user-data'), LIBGL_ALWAYS_SOFTWARE='1',
               XDG_CONFIG_HOME=str(args.out/'config'), XDG_CACHE_HOME=str(args.out/'cache'),
               GALLIUM_DRIVER='llvmpipe', GOMAXPROCS='2', LP_NUM_THREADS='2',
               PULSE_SERVER='unix:/tmp/fd2-hd-pulse/pulse/native',
               FD2_THEME='modern-handpainted-a', FD2_MODERN_THEME_PACK=str(args.modern_pack),
               FD2_ASSET_PACK=str(args.original_pack),
               FD2_INPUT_AUDIT_LOG=str(audit), FD2_CUTSCENE_LOG='1')
    # Leave title, campaign, character stats and start node at production defaults.
    for name in ['FD2_TITLE','FD2_CAMPAIGN','FD2_CAMP_NODE','FD2_SHOT','FD2_SHOT_FRAME']:
        env.pop(name,None)
    if args.switch_theme:
        env.pop('FD2_THEME',None)
    Path(env['XDG_RUNTIME_DIR']).mkdir(mode=0o700)
    Path(env['XDG_CONFIG_HOME']).mkdir()
    Path(env['XDG_CACHE_HOME']).mkdir()
    alsa = args.out/'asoundrc'
    alsa.write_text('pcm.!default { type pulse }\nctl.!default { type pulse }\n')
    env['ALSA_CONFIG_PATH'] = str(alsa)
    def start(command, log, cwd=None):
        stream = (args.out/log).open('w'); streams.append(stream)
        process = subprocess.Popen(command, env=env, cwd=cwd, stdout=stream, stderr=stream)
        processes.append(process); return process
    def command(*items):
        return subprocess.check_output(list(items),env=env,stderr=subprocess.DEVNULL,text=True)
    started = time.monotonic()
    def key(name):
        key_started = time.monotonic()
        before = latest(audit)
        command('xdotool','keydown',name)
        try: time.sleep(.08)
        finally: command('xdotool','keyup',name)
        time.sleep(.35)
        state = latest(audit)
        if name in ('F2','F9'):
            # These operations synchronously validate/decode resources. Observe
            # their completed update, rather than treating a fixed delay as ACK.
            deadline = time.monotonic()+60
            while (state.get('frame',0) <= before.get('frame',0)
                   or name not in (state.get('keys') or [])):
                if time.monotonic()>deadline: raise TimeoutError(f'{name}沒有完成輸入收據')
                time.sleep(.1)
                state = latest(audit)
        with (args.out/'operations.jsonl').open('a') as stream:
            stream.write(json.dumps({'seconds':time.monotonic()-started,'key':name,
                                     'wall_seconds':time.monotonic()-key_started,'state':state},ensure_ascii=False)+'\n')
        if state.get('error'): raise RuntimeError(state['error'])
    shots, switches, save_reload = {}, [], None
    try:
        start(['Xvfb',':97','-screen','0','1920x1200x24','-nolisten','tcp'],'xvfb.log')
        pulse = start(['pulseaudio','--daemonize=no','--exit-idle-time=-1','--use-pid-file=no'],'pulse.log')
        for _ in range(100):
            try:
                command('pactl','info'); break
            except subprocess.CalledProcessError: time.sleep(.1)
            if pulse.poll() is not None: raise RuntimeError('PulseAudio初始化失敗，見pulse.log')
        command('pactl','load-module','module-null-sink','sink_name=fd2_hd','rate=44100','channels=2')
        command('pactl','set-default-sink','fd2_hd')
        game = start([str(args.binary)],'game.log',args.assets)
        window = None
        for _ in range(600):
            if game.poll() is not None: raise RuntimeError('遊戲在建立視窗前退出')
            try:
                window = command('xdotool','search','--name','炎龍騎士團').splitlines()[0]; break
            except (subprocess.CalledProcessError,IndexError): time.sleep(.1)
        assert window
        command('xdotool','windowmove',window,'0','0')
        command('xdotool','windowfocus',window)
        for _ in range(600):
            geometry = command('xdotool','getwindowgeometry','--shell',window)
            if ('WIDTH=1280' in geometry and 'HEIGHT=800' in geometry if args.switch_theme
                    else 'WIDTH=1920' in geometry and 'HEIGHT=1200' in geometry):
                break
            if game.poll() is not None: raise RuntimeError('遊戲在載入高解析主題時退出')
            if latest(audit).get('error'): raise RuntimeError(latest(audit)['error'])
            time.sleep(.1)
        assert ('WIDTH=1280' in geometry and 'HEIGHT=800' in geometry if args.switch_theme
                else 'WIDTH=1920' in geometry and 'HEIGHT=1200' in geometry), geometry
        def shot(name):
            path = args.out/name
            command('import','-window',window,str(path)); return path
        def shot_changed(name, reference):
            # A key receipt is an Update ACK, not a completed GPU presentation.
            # Help changes most of the frame; portrait/arrow flicker cannot pass.
            for _ in range(60):
                path = shot(name)
                result = subprocess.run(['compare','-metric','AE',str(reference),str(path),'null:'],
                                        env=env,capture_output=True,text=True)
                if result.returncode in (0,1) and float(result.stderr.strip()) > 100000:
                    return path
                time.sleep(.25)
            raise TimeoutError('說明遮罩尚未實際呈現')
        shot('title.png')
        for attempt in range(1600):
            if time.monotonic()-started>840: raise TimeoutError('正常序章超過有界驗證時間')
            state = latest(audit)
            if state.get('error'): raise RuntimeError(state['error'])
            if state.get('node')=='battle_ch01' and not state.get('dialogues'):
                break
            logs = (args.out/'game.log').read_text(errors='replace')
            beats = re.findall(r'beat op=dialog source=(0x[0-9a-f]+).*?line=(\d+).*?script=(\S+)',logs)
            if beats and state.get('dialogues'):
                source, line, story = beats[-1]
                row = rows.get((source,story,int(line)))
                speaker = row['resolved_dato'] if row else None
                if args.switch_theme and speaker == 48 and not switches:
                    # The cutscene log announces a beat before synchronous
                    # portrait/font preparation finishes. Wait for live frames
                    # and fully revealed text before testing an unrelated key.
                    stable = None
                    stable_frame = None
                    stable_deadline = time.monotonic()+90
                    while time.monotonic() < stable_deadline:
                        time.sleep(.2)
                        current = latest(audit)
                        same = (stable and current.get('dialogue_speaker') == 48
                                and current.get('dialogue_progress', -1) >= 0
                                and current.get('dialogue_progress') == stable.get('dialogue_progress'))
                        if not same:
                            stable_frame = current.get('frame', 0)
                        if same and current.get('frame', 0)-stable_frame >= 60:
                            break
                        stable = current
                    else: raise TimeoutError('父王對話尚未進入穩定輸入等待')
                    before = latest(audit)
                    faithful = shot('f2-faithful-before.png')
                    key('F2')
                    time.sleep(2)
                    after = latest(audit)
                    assert after.get('hd_theme') and after.get('dialogue_speaker')==48,after
                    assert before['node']==after['node'] and before['dialogues']==after['dialogues']
                    assert before['dialogue_progress']==after['dialogue_progress'],(before,after)
                    geometry = command('xdotool','getwindowgeometry','--shell',window)
                    assert 'WIDTH=1920' in geometry and 'HEIGHT=1200' in geometry,geometry
                    hd_enter = shot('f2-hd-enter.png')
                    key('F1'); shot_changed('f2-help.png',hd_enter); key('Escape')
                    time.sleep(1)
                    for w,h in [(1600,1000),(1920,1080)]:
                        command('xdotool','windowsize',window,str(w),str(h)); time.sleep(.6)
                        resized = command('xdotool','getwindowgeometry','--shell',window)
                        assert f'WIDTH={w}\n' in resized and f'HEIGHT={h}\n' in resized,resized
                        shot(f'f2-resize-{w}x{h}.png')
                    key('F2'); time.sleep(.5)
                    restored = command('xdotool','getwindowgeometry','--shell',window)
                    assert 'WIDTH=1280' in restored and 'HEIGHT=800' in restored,restored
                    assert latest(audit).get('hd_theme') is False
                    shot('f2-faithful-return.png')
                    key('F2'); time.sleep(.5)
                    assert latest(audit).get('hd_theme') is True
                    switches.append({'before':before,'after':after,'faithful_sha256':digest(faithful),
                                     'restored_window':[1280,800], 'resize_windows':[[1600,1000],[1920,1080]]})
                if speaker in hd and speaker not in shots:
                    if args.switch_theme and not switches:
                        key('Return'); continue
                    time.sleep(1.2)
                    target = shot(f'speaker-{speaker:03d}.png')
                    reference = args.modern_pack/hd[speaker]['file']
                    matches = []
                    for anchor in ['480x480+1392+30','480x480+48+690']:
                        result = subprocess.run(['compare','-metric','MAE',str(reference),str(target)+'['+anchor+']','null:'],env=env,capture_output=True,text=True)
                        match = re.search(r'\(([0-9.eE+-]+)\)',result.stderr)
                        if match: matches.append((float(match[1]),anchor))
                    if matches and min(matches)[0]<.0001:
                        error,anchor = min(matches)
                        shots[speaker]={'file':target.name,'sha256':digest(target),'mae_normalized':error,'anchor':anchor}
                        print(f'GPU PASS speaker={speaker} anchor={anchor} MAE={error}',flush=True)
            if args.switch_only and switches and 48 in shots:
                break
            key('Return')
        else: raise RuntimeError('有界按鍵未抵達第一章')
        time.sleep(1)
        final = latest(audit)
        expected_node = 'story_ch00_handler' if args.switch_only else 'battle_ch01'
        assert final.get('node')==expected_node and not final.get('error'),final
        if args.switch_theme and not args.switch_only:
            assert switches
            key('F5'); time.sleep(.5)
            saves = list((args.out/'user-data').rglob('fd2_save*.json'))
            assert saves, '普通F5未產生保存檔'
            saved = {str(f.relative_to(args.out)):digest(f) for f in saves}
            key('F2'); assert latest(audit).get('hd_theme') is False
            key('F9'); time.sleep(2)
            loaded = latest(audit)
            assert loaded.get('node')==final['node'],loaded
            assert loaded.get('hd_theme') is False and not loaded.get('error'),loaded
            assert saved == {str(f.relative_to(args.out)):digest(f) for f in saves}
            key('F2'); time.sleep(.5)
            key('F9'); time.sleep(2)
            hd_loaded = latest(audit)
            # Existing F5 JSON saves persist campaign node boundaries, not the
            # live battlefield. Compare equivalent LOADs across the two themes.
            assert hd_loaded.get('node')==loaded['node'] and hd_loaded.get('units')==loaded.get('units'),(loaded,hd_loaded)
            assert hd_loaded.get('hd_theme') is True and not hd_loaded.get('error'),hd_loaded
            assert saved == {str(f.relative_to(args.out)):digest(f) for f in saves}
            save_reload = {'files':saved,'before':final,'faithful_loaded':loaded,'hd_loaded':hd_loaded,
                           'save_unchanged':True,'loaded_units_equal':True,'theme_retained':True,
                           'contract':'既有JSON節點保存；兩種主題LOAD同份存檔的結果相同，不宣稱戰中快照還原。'}
            final = latest(audit)
        beats = re.findall(r'beat op=dialog ',(args.out/'game.log').read_text())
        if not args.switch_only: assert len(beats)==97,len(beats)
        assert ({48} if args.switch_only else {0,48,66,96}).issubset(shots),shots.keys()
        target = shot('f2-switch-final.png' if args.switch_only else 'first-chapter.png')
        report = {'schema_version':1,'status':'PLAYER-E2-LIMITED-GPU-REVIEW-REQUIRED',
                  'method':'正式預設START，普通Enter與既有開場略過鍵；無章節注入、改血或強化。',
                  'window':[1920,1200],'binary_sha256':digest(args.binary),
                  'catalog_sha256':digest(catalog_path),'dialogue_beats':len(beats),
                  'screenshots':shots,'final_state':final,'final_png_sha256':digest(target),
                  'theme_switches':switches,
                  'switch_only':args.switch_only,
                  'save_reload':save_reload,
                  'wall_seconds':time.monotonic()-started,'recordings_created':False,
                  'limits':('限正常START至父王對話的F2與遮罩，未到首章。' if args.switch_only
                            else '限序章故事頭像與首章交接。')+
                           '非原版視覺對拍、整款HD、跨平台或60fps效能驗收。'}
        (args.out/'hd-gui.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        print('正常START的F2遮罩補抽通過' if args.switch_only else '正常START與97個對話拍點通過',flush=True)
    finally:
        for process in reversed(processes):
            if process.poll() is None: process.send_signal(signal.SIGTERM)
        for process in reversed(processes):
            try: process.wait(timeout=5)
            except subprocess.TimeoutExpired: process.kill(); process.wait()
        for stream in streams: stream.close()


if __name__=='__main__':
    main()
