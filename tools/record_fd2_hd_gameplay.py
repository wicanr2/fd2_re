#!/usr/bin/env python3
"""在 Docker 內錄製既有正常鍵盤驗證器的實際 X11 畫面與遊戲音訊。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import time

from verify_fd2_hd_gui import latest


def stop(process):
    if process is None or process.poll() is not None:
        return
    for sig, seconds in [(signal.SIGINT, 5), (signal.SIGTERM, 3), (signal.SIGKILL, 2)]:
        process.send_signal(sig)
        try:
            process.wait(timeout=seconds)
            return
        except subprocess.TimeoutExpired:
            pass
    raise RuntimeError('錄製程序未能在上限內停止')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--timeout', type=int, default=1100)
    parser.add_argument('verifier_args', nargs=argparse.REMAINDER)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert 60 <= args.timeout <= 2400 and not args.out.exists()
    args.out.mkdir()
    tail = args.verifier_args
    if tail[:1] == ['--']:
        tail = tail[1:]
    repo = Path(__file__).resolve().parents[1]
    command = ['python3', str(repo/'tools/verify_fd2_hd_fonts_gui.py'), *tail]
    env = dict(os.environ, DISPLAY=':97', PULSE_SERVER='unix:/tmp/fd2-hd-font-pulse/pulse/native')
    processes = []
    started = time.monotonic()
    logs = []
    result = {'schema_version': 1, 'command': command, 'kind': 'actual_x11_and_game_audio',
              'source_injection': '依驗證器的明示來源槽；錄製器不修改遊戲或存檔', 'status': 'FAIL'}
    def launch(cmd, name):
        stream = (args.out/name).open('w'); logs.append(stream)
        p = subprocess.Popen(cmd, env=env, stdout=stream, stderr=stream)
        processes.append(p)
        return p
    try:
        verifier = launch(command, 'verifier.log')
        deadline = time.monotonic()+90
        while time.monotonic() < deadline:
            if verifier.poll() is not None:
                raise RuntimeError('遊戲驗證器在錄影就緒前結束')
            test = subprocess.run(['pactl', 'list', 'short', 'sinks'], env=env, capture_output=True, text=True)
            if 'fd2_hd_fonts' in test.stdout and Path('/tmp/.X11-unix/X97').exists():
                break
            time.sleep(.25)
        else:
            raise TimeoutError('X11或音訊接收器未就緒')
        recording_started = time.monotonic()
        video = launch(['ffmpeg','-nostdin','-y','-hide_banner','-loglevel','warning',
                        '-thread_queue_size','256','-f','x11grab','-framerate','30',
                        '-video_size','1920x1200','-draw_mouse','0','-i',':97.0',
                        '-an','-c:v','libx264','-preset','veryfast','-crf','18',
                        '-threads','2','-pix_fmt','yuv420p',str(args.out/'video.mkv')], 'video.log')
        audio = launch(['ffmpeg','-nostdin','-y','-hide_banner','-loglevel','warning',
                        '-f','pulse','-i','fd2_hd_fonts.monitor','-vn',
                        '-c:a','pcm_s16le',str(args.out/'audio.wav')], 'audio.log')
        result['recording_offset_seconds'] = recording_started-started
        # Read-only timing index for editing. Software rendering can stretch
        # wall-clock duration; retain the actual frames and scene boundaries.
        verifier_out = Path(tail[tail.index('--out')+1])
        timing_rows, previous = 0, None
        with (args.out/'frame-times.jsonl').open('w') as timing:
            while verifier.poll() is None:
                if time.monotonic()-started >= args.timeout:
                    raise TimeoutError('正常鍵盤錄影超過明示預算')
                state = latest(verifier_out/'input-audit.jsonl')
                signature = (state.get('frame'), state.get('title_phase'))
                if state and signature != previous:
                    timing.write(json.dumps(dict(seconds=time.monotonic()-recording_started,
                        state={k:state.get(k) for k in ['frame','node','title_phase','battle_scene',
                            'command_scene','command_id','command_phase','command_frame','command_target_index',
                            'hd_theme','hd_battle','walking','dialogues','cursor']}))+'\n')
                    timing.flush()
                    timing_rows += 1
                    previous = signature
                time.sleep(.1)
        code = verifier.wait()
        result['frame_timing_rows'] = timing_rows
        result['verifier_exit_code'] = code
        if code != 0:
            raise RuntimeError(f'正常鍵盤驗證器失敗：{code}')
        result['status'] = 'PASS'
    finally:
        for process in reversed(processes):
            stop(process)
        for stream in logs:
            stream.close()
        result['elapsed_seconds'] = time.monotonic()-started
        result['files'] = [{'file': p.name, 'bytes': p.stat().st_size,
                            'sha256': hashlib.sha256(p.read_bytes()).hexdigest()}
                           for p in sorted(args.out.iterdir()) if p.suffix in ['.mkv','.wav']]
        (args.out/'recording.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__ == '__main__':
    main()
