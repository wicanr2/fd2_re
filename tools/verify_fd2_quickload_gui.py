#!/usr/bin/env python3
"""Docker內以普通Home／F9驗證資訊面板讀檔；起點為明示建構槽。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import time
from operate_fd2_promo import latest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('out', type=Path)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists()
    for _ in range(180):
        result = subprocess.run(['xdotool', 'search', '--name', '炎龍騎士團'],
                                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if result.returncode == 0:
            break
        time.sleep(.5)
    else:
        raise RuntimeError('window did not appear')

    def press(key, shot):
        hold = '.06' if key in ('Left', 'Right', 'Up', 'Down') else '.25'
        with (args.out / 'driver.log').open('a') as log:
            subprocess.run(['python3', str(Path(__file__).with_name('operate_fd2_promo.py')),
                            str(args.out), '--keys', key, '--hold', hold, '--wait', '3',
                            '--shot', shot], check=True, stdout=log, stderr=log)
        return latest(args.out)

    battle = press('F9', 'battle-loaded.png')
    assert battle.get('node') == 'battle_ch01' and not battle.get('error'), battle
    for index in range(30):
        if not battle.get('dialogues'):
            break
        battle = press('Return', 'briefing.png')
    assert not battle.get('dialogues'), 'battle briefing did not finish'
    # A node-boundary save can retain party positions from its source chapter.
    # Move the ordinary cursor to a real surviving unit; never assume (7,14).
    unit = next(u for u in battle['units'] if u['camp'] == 0 and u['hp'] > 0 and u['on_field'])
    for _ in range(24):
        x, y = battle['cursor']
        if [x, y] == [unit['x'], unit['y']]:
            break
        key = ('Right' if unit['x'] > x else 'Left') if x != unit['x'] else (
            'Down' if unit['y'] > y else 'Up')
        battle = press(key, 'cursor-step.png')
    assert battle['cursor'] == [unit['x'], unit['y']], battle
    before = press('Home', 'status-before.png')
    assert before.get('player_status_phase') and not before.get('error'), before
    reference = args.out / 'hotel-reference.json'
    shutil.copyfile(reference, args.out / 'user-data/fd2_re/fd2_save.json')
    after = press('F9', 'hotel-after.png')
    assert after.get('node') == 'hotel_ch02' and not after.get('player_status_phase')
    assert not after.get('error'), after
    town = press('Escape', 'town-after.png')
    assert town.get('node') == 'town_ch02' and not town.get('player_status_phase')
    names = ['battle-loaded.png', 'status-before.png', 'hotel-after.png', 'town-after.png',
             'input-audit.jsonl', 'operations.jsonl', 'start.json']
    hashes = {n: hashlib.sha256((args.out / n).read_bytes()).hexdigest() for n in names}
    assert hashes['status-before.png'] != hashes['hotel-after.png']
    report = {'schema_version': 1, 'status': 'RUNTIME-E1-GPU-REVIEW-REQUIRED',
              'method': '普通F9載入明示battle_ch01建構槽，Home開資訊，再F9載入既有酒店槽，Escape返回城鎮。',
              'limits': 'FD2_TITLE=0略過標題；初始槽由酒店JSON只改node；不宣稱一般開頭完整PLAYER-E2、原版對拍或傷害一致。',
              'before': before, 'after': after, 'town': town, 'files': hashes,
              'hotel_reference_sha256': hashlib.sha256(reference.read_bytes()).hexdigest()}
    (args.out / 'quickload-gui.json').write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n')
    print(json.dumps({'status': report['status'], 'files': hashes}, ensure_ascii=False))


if __name__ == '__main__':
    main()
