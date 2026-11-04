#!/usr/bin/env python3
"""附接錄影容器，以普通 START 錄製皇城切換與第一關站位，不注入狀態。"""

import argparse
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time

from fd2_hd_gold_dialogue_gui import acknowledged_key
from verify_fd2_hd_gui import digest, latest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--out', type=Path, required=True)
    parser.add_argument('--binary', type=Path, required=True)
    parser.add_argument('--skip-opening', action='store_true')
    parser.add_argument('--faithful-transit', action='store_true', help='皇城切換後以原版畫風走完序章，第一關進場時恢復HD；只送F2。')
    parser.add_argument('--menu-only', action='store_true', help='正常鍵盤略過開場後，只錄實際開始選單。')
    parser.add_argument('--startup-timeout', type=int, choices=(360,900), default=360)
    parser.add_argument('--duration', type=int, choices=(900, 1800, 2700, 3600), default=1800)
    parser.add_argument('--capture-appdir', type=Path, help='由本程序擁有既有錄影工具及其清理，供有界容器使用。')
    parser.add_argument('--capture-size', choices=('640x400', '1280x800', '1920x1200'), default='1920x1200')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert args.out.is_dir() and args.binary.is_file()
    assert args.out.stat().st_uid == os.getuid()
    audit = args.out / 'input-audit.jsonl'
    started = time.monotonic()
    driver_source_sha256 = digest(Path(__file__))
    deadline = started + args.duration - 50
    events = []
    owner, owner_stream, received_signal = None, None, None

    def interrupted(number, _frame):
        nonlocal received_signal
        received_signal = signal.Signals(number).name
        raise RuntimeError('錄影控制程序收到' + received_signal)

    signal.signal(signal.SIGTERM, interrupted)
    signal.signal(signal.SIGINT, interrupted)

    def command(*items):
        return subprocess.check_output(items, text=True, stderr=subprocess.DEVNULL)

    def wait(predicate, limit=120):
        end = min(time.monotonic() + limit, deadline)
        while time.monotonic() < end:
            if owner is not None and owner.poll() is not None:
                raise RuntimeError('錄影擁有程序已提前結束，請查owner.log')
            state = latest(audit)
            if state.get('error'):
                raise RuntimeError(state['error'])
            if predicate(state):
                return state
            time.sleep(.1)
        raise TimeoutError(str({k: v for k, v in latest(audit).items()
                                if not k.startswith('hd_world_')}))

    def record(label, state=None):
        state = latest(audit) if state is None else state
        start = json.loads((args.out / 'start.json').read_text())
        item = dict(label=label, seconds=time.monotonic() - start['monotonic'],
                    state={k: v for k, v in state.items()
                           if not k.startswith('hd_world_')})
        events.append(item)
        with (args.out / 'promo-events.jsonl').open('a') as stream:
            stream.write(json.dumps(item, ensure_ascii=False) + '\n')
        return item

    def key(name):
        record('before-key-' + name)
        result = acknowledged_key(audit, command, name, args.out / 'operations.jsonl')
        if name in ('F2', 'F11'):
            size = json.loads((args.out / 'start.json').read_text())['window_size'].split('x')
            command('xdotool', 'windowsize', window, *size)
        record('after-key-' + name)
        return result

    def shot(label):
        command('import', '-window', window, str(args.out / (label + '.png')))
        record(label)

    def dialogue_identity(state):
        return {k: state.get(k) for k in ('node', 'units', 'dialogues',
                'dialogue_speaker', 'dialogue_progress', 'dialogue_phase')}

    try:
        if args.capture_appdir:
            assert (args.capture_appdir / 'AppRun').is_file()
            os.environ['DISPLAY'] = ':95'
            env = dict(os.environ)
            env.setdefault('FD2_CAPTURE_STARTUP_TIMEOUT', str(args.startup_timeout))
            owner_stream = (args.out / 'owner.log').open('w')
            owner = subprocess.Popen(['bash', str(Path(__file__).with_name('capture_fd2_release_promo.sh')),
                                      str(args.capture_appdir), str(args.out), str(args.duration), args.capture_size, '30'],
                                     env=env, stdout=owner_stream, stderr=owner_stream, start_new_session=True)
        wait(lambda s: bool(s.get('title_phase')), args.startup_timeout)
        window = command('xdotool', 'search', '--name', '炎龍騎士團').splitlines()[0]
        command('xdotool', 'windowfocus', window)
        if args.skip_opening:
            while latest(audit).get('title_phase') != 'menu':
                key('Escape')
        wait(lambda s: s.get('title_phase') == 'menu', args.duration - 50)
        assert latest(audit)['title_selection'] == 0
        opening = None
        if not args.skip_opening:
            rows = []
            with audit.open() as stream:
                for line in stream:
                    state = json.loads(line)
                    if state.get('title_phase') == 'cutscene':
                        rows.append({k: state.get(k) for k in ('frame', 'opening_step', 'opening_frame', 'opening_step_count', 'hd_theme', 'hd_opening')})
            assert rows and all(row['hd_opening'] and row['hd_theme'] for row in rows)
            expected = set(range(rows[0]['opening_step_count']))
            assert {row['opening_step'] for row in rows} == expected
            opening = dict(status='PASS-NATURAL-HD-OPENING-TO-MENU', steps=sorted(expected), rows=rows,
                           scope='普通開場排程完整抵達選單，全部Update收據保持HD；各影格實際繪製另由GPU收據驗證。')
            (args.out / 'natural-opening.json').write_text(json.dumps(opening, ensure_ascii=False, indent=2) + '\n')
        wait(lambda s: s.get('hd_text_runs', 0) == 6)
        time.sleep(2)
        shot('start-menu')
        time.sleep(3)
        if args.menu_only:
            (args.out/'menu-promo-gui.json').write_text(json.dumps(dict(
                status='PASS-NORMAL-KEYBOARD-TITLE-MENU',state_injection=False,
                binary_sha256=digest(args.binary),driver_sha256=driver_source_sha256,
                events=events),ensure_ascii=False,indent=2)+'\n')
            return
        key('Return')
        wait(lambda s: not s.get('title_phase'))
        royal_deadline = min(time.monotonic() + 240, deadline)
        while True:
            state = latest(audit)
            if state.get('dialogue_speaker') == 48 and state.get('dialogue_phase') == 0:
                break
            if state.get('error'):
                raise RuntimeError(state['error'])
            if time.monotonic() >= royal_deadline:
                raise TimeoutError('普通序章未抵達父王對話')
            if state.get('dialogues') and state.get('dialogue_phase') == 0:
                key('Return')
                time.sleep(.65)
            else:
                time.sleep(.2)
        previous = None
        stable_at = time.monotonic()
        while True:
            state = wait(lambda s: s.get('dialogue_speaker') == 48
                         and s.get('dialogue_phase') == 0)
            if previous is None or state.get('dialogue_progress') != previous.get('dialogue_progress'):
                stable_at = time.monotonic()
            if time.monotonic() - stable_at >= 2:
                break
            previous = state
            time.sleep(.2)
        initial = dialogue_identity(latest(audit))
        for label, keys, art, ui in (
            ('royal-hd-hd', (), True, 'hd'),
            ('royal-original-hd', ('F2',), False, 'hd'),
            ('royal-original-original', ('F11',), False, 'original'),
            ('royal-hd-original', ('F2',), True, 'original'),
            ('royal-hd-hd-restored', ('F11',), True, 'hd'),
        ):
            for name in keys:
                key(name)
            wait(lambda s: s.get('hd_theme') is art and s.get('interface_mode') == ui)
            wait(lambda s: bool(s.get('hd_gold_dialogue_rects')) == (ui == 'hd'))
            time.sleep(2)
            assert dialogue_identity(latest(audit)) == initial
            shot(label)
            time.sleep(2)
        record('royal-dialogue-advance')
        if args.faithful_transit:
            key('F2')
            wait(lambda s: not s.get('hd_theme'))
        while True:
            state = latest(audit)
            if args.faithful_transit and state.get('node') == 'battle_ch01' and not state.get('hd_theme'):
                key('F2')
                wait(lambda s: s.get('hd_theme'))
                record('first-battle-hd-restored')
            if state.get('node') == 'battle_ch01' and not state.get('dialogues'):
                break
            if time.monotonic() >= deadline:
                raise TimeoutError('普通序章未於錄製預算內抵達第一關')
            if state.get('dialogues') and state.get('dialogue_phase') == 0:
                key('Return')
                time.sleep(.65)
            else:
                time.sleep(.2)
        wait(lambda s: s.get('node') == 'battle_ch01' and not s.get('dialogues')
             and not any(s.get(k) for k in ('walking', 'transient_ui', 'banner',
                     'ai_busy', 'native_full_dac_black', 'native_map_dac_black')), 180)
        allies = [u for u in latest(audit)['units'] if u['camp'] == 0 and u['on_field']]
        assert {u['native_identity'] for u in allies} == {0, 4, 9, 30}
        shot('first-battle-deployed')
        time.sleep(4)
        key('F5')
        save = args.out / 'user-data/fd2_re/fd2_save.json'
        wait(lambda s: save.is_file())
        time.sleep(1)
        record('first-battle-saved')
        report = dict(status='PASS-NORMAL-START-ROYAL-DEPLOYMENT', state_injection=False,
                      opening_skipped=args.skip_opening,
                      faithful_transit=args.faithful_transit,
                      natural_opening=opening,
                      scope='重製端普通START、皇城四組切換、第一關入場與F5；不宣稱DOS原版對拍或整章通關。',
                      binary_sha256=digest(args.binary), driver_sha256=driver_source_sha256,
                      driver_source_unchanged=digest(Path(__file__)) == driver_source_sha256,
                      save_sha256=digest(save), save=str(save), events=events,
                      elapsed_seconds=time.monotonic() - started)
        (args.out / 'opening-promo-gui.json').write_text(
            json.dumps(report, ensure_ascii=False, indent=2) + '\n')
        print(json.dumps({k: v for k, v in report.items() if k != 'events'}, ensure_ascii=False))
    finally:
        failure = str(sys.exc_info()[1]) if sys.exc_info()[1] is not None else None
        (args.out / 'STOP').touch()
        owner_exit = None
        if owner:
            try:
                owner_exit = owner.wait(timeout=45)
            except subprocess.TimeoutExpired:
                os.killpg(owner.pid, signal.SIGTERM)
                try:
                    owner_exit = owner.wait(timeout=15)
                except subprocess.TimeoutExpired:
                    os.killpg(owner.pid, signal.SIGKILL)
                    owner_exit = owner.wait(timeout=5)
        if owner_stream:
            owner_stream.close()
        (args.out / 'promo-events.json').write_text(json.dumps(events, ensure_ascii=False, indent=2)+'\n')
        (args.out / 'controller-result.json').write_text(json.dumps(
            dict(status='PASS-GUI' if failure is None else 'FAILED-OR-INTERRUPTED', error=failure,
                 received_signal=received_signal, recording_owner_exit=owner_exit,
                 elapsed_seconds=time.monotonic()-started), ensure_ascii=False, indent=2)+'\n')


if __name__ == '__main__':
    main()
