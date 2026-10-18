#!/usr/bin/env python3
"""在Docker以兩個實際GPU程序比較Ebiten預設與正式最終呈現器，不建立錄影。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import time


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--binary', type=Path, required=True)
    p.add_argument('--out', type=Path, required=True)
    a = p.parse_args()
    assert Path('/.dockerenv').exists() and a.binary.is_file()
    assert not a.out.exists()
    a.out.mkdir()
    env = dict(os.environ, DISPLAY=':96', LIBGL_ALWAYS_SOFTWARE='1',
               GALLIUM_DRIVER='llvmpipe', LP_NUM_THREADS='2', GOMAXPROCS='2')
    results = []
    def cmd(*items):
        return subprocess.check_output(items, env=env, text=True, stderr=subprocess.DEVNULL)
    with (a.out/'xvfb.log').open('w') as log:
        xvfb = subprocess.Popen(['Xvfb', ':96', '-screen', '0', '1920x1200x24', '-nolisten', 'tcp'],
                                env=env, stdout=log, stderr=log)
        try:
            time.sleep(.5)
            for w, h, off in [(1280,800,False),(1600,1000,False),(320,200,False),
                               (1920,1080,False),(1600,1000,True)]:
                files = []
                for mode in ['default','adapter']:
                    name = f'{w}x{h}-off{int(off)}-{mode}'
                    output = a.out/(name+'.log')
                    childenv = dict(env, FD2_TEST_SCREEN_FILTER_GPU=mode,
                                    FD2_TEST_SCREEN_WIDTH=str(w), FD2_TEST_SCREEN_HEIGHT=str(h),
                                    FD2_TEST_SCREEN_FILTER_OFF=str(int(off)))
                    with output.open('w') as stream:
                        game = subprocess.Popen([str(a.binary)], env=childenv, stdout=stream, stderr=stream)
                        try:
                            for _ in range(300):
                                if game.poll() is not None: raise RuntimeError(output.read_text())
                                if 'GPU_READY' in output.read_text(): break
                                time.sleep(.1)
                            else: raise TimeoutError(name)
                            window = cmd('xdotool','search','--name','FD2 screen filter GPU probe').splitlines()[0]
                            cmd('xdotool','windowmove',window,'0','0')
                            geometry = cmd('xdotool','getwindowgeometry','--shell',window)
                            assert f'WIDTH={w}\n' in geometry and f'HEIGHT={h}\n' in geometry,geometry
                            shot = a.out/(name+'.png')
                            cmd('import','-window',window,str(shot))
                            files.append(shot)
                        finally:
                            game.send_signal(signal.SIGTERM) if game.poll() is None else None
                            try: game.wait(timeout=5)
                            except subprocess.TimeoutExpired: game.kill(); game.wait()
                comparison = subprocess.run(['compare','-metric','AE',str(files[0]),str(files[1]),'null:'],
                                            env=env, text=True, capture_output=True)
                assert comparison.returncode == 0 and comparison.stderr.strip() == '0',comparison.stderr
                results.append({'window':[w,h], 'filter_disabled':off, 'different_pixels':0,
                                'screenshots':[{ 'file':f.name,'sha256':hashlib.sha256(f.read_bytes()).hexdigest()} for f in files]})
                print(f'GPU PASS {w}x{h} filter_off={off} different_pixels=0',flush=True)
        finally:
            xvfb.terminate()
            try: xvfb.wait(timeout=5)
            except subprocess.TimeoutExpired: xvfb.kill(); xvfb.wait()
    (a.out/'screen-filter-gpu.json').write_text(json.dumps({
        'status':'PASS','method':'獨立程序的實際GPU視窗；預設Ebiten與正式adapter全畫面比較。',
        'binary_sha256':hashlib.sha256(a.binary.read_bytes()).hexdigest(), 'cases':results,
        'recordings_created':False,'limits':'Linux llvmpipe、鎖定Ebiten v2.6.6；不是原版遊戲對拍或跨平台效能。'
    },ensure_ascii=False,indent=2)+'\n')


if __name__ == '__main__':
    main()
