#!/usr/bin/env python3
"""Docker內由正常標題LOAD量測原版與HD地圖；不注入執行中狀態。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import time

from verify_fd2_hd_gui import latest


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ['binary','assets','original-pack','modern-pack','source-save','out']:
        p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--seconds',type=int,choices=[35,45],default=35)
    a=p.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid()!=0 and not a.out.exists()
    assert a.out.parent.stat().st_uid==os.getuid()
    a.out.mkdir()
    save=a.out/'user-data/fd2_re/fd2_save.json'
    save.parent.mkdir(parents=True)
    shutil.copyfile(a.source_save,save)
    audit=a.out/'input-audit.jsonl'
    runtime=a.out/'runtime'
    runtime.mkdir(mode=0o700)
    env=dict(os.environ,DISPLAY=':96',XDG_RUNTIME_DIR=str(runtime),
        XDG_DATA_HOME=str(a.out/'user-data'),XDG_CONFIG_HOME=str(a.out/'config'),
        XDG_CACHE_HOME=str(a.out/'cache'),LIBGL_ALWAYS_SOFTWARE='1',GALLIUM_DRIVER='llvmpipe',
        GOMAXPROCS='2',LP_NUM_THREADS='2',PULSE_SERVER='unix:'+str(runtime/'pulse/native'),
        FD2_THEME='modern-handpainted-a',FD2_LOCALE='zh-Hant',FD2_TOOLBAR_LAYOUT='floating',
        FD2_MODERN_THEME_PACK=str(a.modern_pack),FD2_ASSET_PACK=str(a.original_pack),
        FD2_INPUT_AUDIT_LOG=str(audit),FD2_INPUT_AUDIT_INTERVAL='10',
        FD2_PERFORMANCE_REPORT=str(a.out/'performance.jsonl'),
        FD2_PERFORMANCE_CPU_PROFILE=str(a.out/'world.pprof'),FD2_PERFORMANCE_CPU_PROFILE_SCENE='world')
    for key in ['FD2_TITLE','FD2_CAMPAIGN','FD2_CAMP_NODE','FD2_SHOT','FD2_NATIVE_SAVE','FD2_SAVE_CURRENT']:
        env.pop(key,None)
    for name in ['config','cache']:(a.out/name).mkdir()
    alsa=a.out/'asoundrc'
    alsa.write_text('pcm.!default { type pulse }\nctl.!default { type pulse }\n')
    env['ALSA_CONFIG_PATH']=str(alsa)
    processes=[]
    streams=[]
    operations=[]
    windows=[]
    def start(cmd,name,cwd=None):
        stream=(a.out/name).open('w');streams.append(stream)
        proc=subprocess.Popen(cmd,env=env,cwd=cwd,stdout=stream,stderr=stream)
        processes.append(proc)
        return proc
    def command(*args):
        return subprocess.check_output(args,env=env,text=True,stderr=subprocess.DEVNULL,timeout=15)
    def wait(predicate,timeout=90):
        until=time.monotonic()+timeout
        while time.monotonic()<until:
            s=latest(audit)
            if s.get('error'):raise RuntimeError(s['error'])
            if predicate(s):return s
            time.sleep(.05)
        raise TimeoutError(str(latest(audit)))
    def rows():
        if not audit.exists():return []
        with audit.open('rb') as f:
            f.seek(max(0,audit.stat().st_size-131072));lines=f.read().splitlines()
        out=[]
        for line in lines:
            try:out.append(json.loads(line))
            except ValueError:pass
        return out
    def key(name):
        before=latest(audit)
        mapped={'Return':'Enter','Down':'ArrowDown'}.get(name,name)
        command('xdotool','windowfocus',window)
        command('xdotool','keydown',name)
        try:
            until=time.monotonic()+30
            ack=None
            while time.monotonic()<until:
                ack=next((r for r in reversed(rows()) if r.get('frame',0)>before.get('frame',0) and mapped in (r.get('keys') or [])),None)
                if ack:break
                time.sleep(.005)
            assert ack, '按鍵未受領 '+name
        finally:command('xdotool','keyup',name)
        until=time.monotonic()+30
        release=None
        while time.monotonic()<until:
            release=next((r for r in reversed(rows()) if r.get('frame',0)>ack['frame'] and mapped in (r.get('released_keys') or [])),None)
            if release:break
            time.sleep(.005)
        assert release, '放鍵未受領 '+name
        wait(lambda s:s.get('frame',0)>=release['frame']+2)
        operations.append(dict(key=name,before=before,ack=ack,release=release,after=latest(audit)))
    try:
        start(['Xvfb',':96','-screen','0','1920x1200x24','-nolisten','tcp'],'xvfb.log')
        for _ in range(100):
            try:command('xdotool','getdisplaygeometry');break
            except subprocess.CalledProcessError:time.sleep(.1)
        else:raise TimeoutError('Xvfb')
        start(['pulseaudio','--daemonize=no','--exit-idle-time=-1','--use-pid-file=no'],'pulse.log')
        for _ in range(100):
            try:command('pactl','info');break
            except subprocess.CalledProcessError:time.sleep(.1)
        else:raise TimeoutError('PulseAudio')
        command('pactl','load-module','module-null-sink','sink_name=fd2_profile','rate=44100','channels=2')
        command('pactl','set-default-sink','fd2_profile')
        game=start([str(a.binary)],'game.log',a.assets)
        window=None
        for _ in range(1800):
            if game.poll() is not None:raise RuntimeError('遊戲提前退出')
            try:window=command('xdotool','search','--name','炎龍騎士團').splitlines()[0];break
            except (subprocess.CalledProcessError,IndexError):time.sleep(.1)
        assert window
        wait(lambda s:bool(s.get('title_phase')))
        for _ in range(100):
            if latest(audit).get('title_phase')=='menu':break
            key('Escape')
        else:raise TimeoutError('標題選單')
        key('Down');key('Return')
        wait(lambda s:s.get('title_phase')=='loadslots')
        key('Return')
        wait(lambda s:not s.get('title_phase') and s.get('node')=='battle_ch01')
        for _ in range(100):
            s=latest(audit)
            if not s.get('dialogues') and not s.get('banner_t'):break
            key('Return')
        else:raise TimeoutError('首章對話／橫幅')
        for label,hd in [('hd-balanced',True),('faithful',False)]:
            if latest(audit).get('hd_theme') is not hd:
                key('F2');wait(lambda s:s.get('hd_theme') is hd)
            command('xdotool','windowsize',window,'1920','1200')
            command('xdotool','windowmove',window,'0','0')
            time.sleep(3)
            first=latest(audit)
            stamp=latest(a.out/'performance.jsonl').get('elapsed_seconds',0)
            print('OBSERVE '+label,flush=True)
            time.sleep(a.seconds)
            last=latest(audit)
            windows.append(dict(label=label,start_elapsed_seconds=stamp,
                end_elapsed_seconds=latest(a.out/'performance.jsonl').get('elapsed_seconds'),
                first=first,last=last,window=command('xdotool','getwindowgeometry','--shell',window)))
            command('import','-window',window,str(a.out/(label+'.png')))
        assert sha(save)==sha(a.source_save)
        report=dict(status='PASS-OBSERVATION',binary_sha256=sha(a.binary),source_save_sha256=sha(a.source_save),
            gui_tool_sha256=sha(__file__),renderer='Xvfb/llvmpipe',render_threads=2,gomaxprocs=2,
            window=[1920,1200],windows=windows,operations=operations,source_save_unchanged=True,
            scope='已揭露建構祖先的首章存檔，正常標題LOAD、Enter及F2；靜止地圖抽樣，不代表戰鬥或Windows實機60fps。')
        (a.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        print('PASS-OBSERVATION',flush=True)
    finally:
        for proc in reversed(processes):
            if proc.poll() is None:
                proc.terminate()
                try:proc.wait(timeout=15)
                except subprocess.TimeoutExpired:proc.kill();proc.wait(timeout=5)
        for stream in streams:stream.close()


if __name__=='__main__':main()
