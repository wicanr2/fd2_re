#!/usr/bin/env python3
"""Docker 錄影期間送普通鍵盤輸入，保存時點與原圖。"""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import time


def latest(out):
    path = out / 'input-audit.jsonl'
    if not path.exists():
        return {}
    with path.open('rb') as stream:
        stream.seek(max(0, path.stat().st_size - 131072))
        rows = stream.read().splitlines()
    for row in reversed(rows):
        try:
            return json.loads(row)
        except ValueError:
            pass
    return {}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('out', type=Path)
    p.add_argument('--keys', nargs='*', default=[])
    p.add_argument('--wait', type=float, default=.7)
    p.add_argument('--hold', type=float, default=.4)
    p.add_argument('--shot')
    p.add_argument('--advance-story', type=int, default=0)
    args = p.parse_args()
    assert Path('/.dockerenv').exists()
    window = subprocess.check_output(['xdotool', 'search', '--name', '炎龍騎士團']).decode().splitlines()[0]
    subprocess.run(['xdotool', 'windowfocus', window], check=True)
    start = json.loads((args.out / 'start.json').read_text())['epoch']
    def record(key, label=None):
        row = {'seconds': round(time.time() - start, 3), 'key': key, 'label': label, 'hold': args.hold}
        if key:
            subprocess.run(['xdotool', 'keydown', key], check=True)
            try:
                time.sleep(args.hold)
            finally:
                subprocess.run(['xdotool', 'keyup', key], check=True)
        time.sleep(args.wait)
        row['completed_seconds'] = round(time.time() - start, 3)
        row['state'] = latest(args.out)
        with (args.out / 'operations.jsonl').open('a') as f:
            f.write(json.dumps(row, ensure_ascii=False) + '\n')
    for key in args.keys:
        record(key)
    for _ in range(args.advance_story):
        state = latest(args.out)
        if state.get('node') == 'battle_ch01' and not state.get('dialogues') and not state.get('ai_busy'):
            break
        record('Return', 'ordinary-dialogue-advance')
    if args.shot:
        assert Path(args.shot).name == args.shot and args.shot.endswith('.png')
        target = args.out / args.shot
        subprocess.run(['import', '-window', window, str(target)], check=True)
        record(None, args.shot)
        print('shot_sha256', hashlib.sha256(target.read_bytes()).hexdigest())
    state = latest(args.out)
    print(json.dumps({'seconds': round(time.time() - start, 3), **{k: v for k, v in state.items() if k != 'units'}, 'units': [{k:u[k] for k in ['name','x','y','hp','camp','acted','on_field']} for u in state.get('units',[])]}, ensure_ascii=False))


if __name__ == '__main__':
    main()
