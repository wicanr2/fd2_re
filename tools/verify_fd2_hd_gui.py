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
    args = parser.parse_args()
    assert Path('/.dockerenv').exists()
    assert args.binary.is_file() and (args.assets/'assets/locales/zh-Hant/pack.json').is_file()
    assert (args.original_pack/'manifest.json').is_file() and args.modern_pack.is_dir()
    assert not args.out.exists(), '禁止覆寫既有驗收'
    args.out.mkdir()
    assert args.out.stat().st_uid == os.getuid()
    audit = args.out/'input-audit.jsonl'
    catalog_path = ROOT/'remake/assets/themes/modern/catalog.json'
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
        command('xdotool','keydown',name)
        try: time.sleep(.08)
        finally: command('xdotool','keyup',name)
        time.sleep(.35)
        state = latest(audit)
        with (args.out/'operations.jsonl').open('a') as stream:
            stream.write(json.dumps({'seconds':time.monotonic()-started,'key':name,'state':state},ensure_ascii=False)+'\n')
        if state.get('error'): raise RuntimeError(state['error'])
    shots = {}
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
            if 'WIDTH=1920' in geometry and 'HEIGHT=1200' in geometry:
                break
            if game.poll() is not None: raise RuntimeError('遊戲在載入高解析主題時退出')
            if latest(audit).get('error'): raise RuntimeError(latest(audit)['error'])
            time.sleep(.1)
        assert 'WIDTH=1920' in geometry and 'HEIGHT=1200' in geometry, geometry
        def shot(name):
            path = args.out/name
            command('import','-window',window,str(path)); return path
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
                if speaker in hd and speaker not in shots:
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
            key('Return')
        else: raise RuntimeError('有界按鍵未抵達第一章')
        time.sleep(1)
        final = latest(audit)
        assert final.get('node')=='battle_ch01' and not final.get('error'),final
        beats = re.findall(r'beat op=dialog ',(args.out/'game.log').read_text())
        assert len(beats)==97,len(beats)
        assert {0,48,66,96}.issubset(shots),shots.keys()
        target = shot('first-chapter.png')
        report = {'schema_version':1,'status':'PLAYER-E2-LIMITED-GPU-REVIEW-REQUIRED',
                  'method':'正式預設START，普通Enter與既有開場略過鍵；無章節注入、改血或強化。',
                  'window':[1920,1200],'binary_sha256':digest(args.binary),
                  'catalog_sha256':digest(catalog_path),'dialogue_beats':len(beats),
                  'screenshots':shots,'final_state':final,'final_png_sha256':digest(target),
                  'wall_seconds':time.monotonic()-started,'recordings_created':False,
                  'limits':'限序章故事頭像與首章交接；非原版視覺對拍、整款HD、跨平台或60fps效能驗收。'}
        (args.out/'hd-gui.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        print('正常START與97個對話拍點通過',flush=True)
    finally:
        for process in reversed(processes):
            if process.poll() is None: process.send_signal(signal.SIGTERM)
        for process in reversed(processes):
            try: process.wait(timeout=5)
            except subprocess.TimeoutExpired: process.kill(); process.wait()
        for stream in streams: stream.close()


if __name__=='__main__':
    main()
