"""既有第23章來源的正常回復術鍵盤驗證，由共用GUI工具擁有程序。"""
import json
import threading
import time
from pathlib import Path

from verify_fd2_hd_gui import digest, latest

SOURCE_SHA = '5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871'


def run_normal_heal(args, audit, key, wait, shot, saved, command, window,
                    restart, operations, shots, tool_hash):
    source_hash, helper_hash = digest(args.town_save), digest(Path(__file__))
    assert source_hash == SOURCE_SHA, '回復術補驗使用已審查的章前來源'
    actor_id = 24
    source = json.loads(args.town_save.read_text())
    roster = source['party_roster'][str(actor_id)]
    learned = [i for i in range(36) if roster['native_command_mask'][i//8] & (1 << (i % 8))]
    assert source['party_deploy'][str(actor_id)] and all(i in learned for i in range(13, 17))
    cases, samples, rejected, errors = [], [], [], []
    stop = threading.Event()
    worker = None

    def checkpoint():
        (args.out/'heal-observation.json').write_text(json.dumps(dict(
            status='OBSERVATION-ONLY', cases=cases, samples=samples, rejected=rejected,
            errors=errors, state_injection=False, recording_created=False,
            source_sha256=source_hash, helper_sha256=helper_hash), ensure_ascii=False, indent=2)+'\n')

    def clean(s):
        return not any(s.get(k) for k in ('heal_scene', 'dialogues', 'banner_t', 'class_ui_job',
                                         'transient_ui', 'battle_scene', 'command_scene',
                                         'unit_death_scene', 'ai_busy', 'walking'))

    def idle():
        deadline = time.monotonic()+180
        while time.monotonic() < deadline:
            s = latest(audit)
            if s.get('error'):
                raise RuntimeError(s['error'])
            if clean(s):
                return s
            if s.get('dialogues') and not s.get('heal_scene'):
                key('Return')
            else:
                time.sleep(.03)
        raise TimeoutError('回復術未正常返回地圖')

    def actor(s):
        matches = [u for u in s['units'] if u.get('native_identity') == actor_id
                   and u['camp'] == 0 and u['on_field'] and u['hp'] > 0]
        assert len(matches) == 1, '希爾法必須正常在場'
        return matches[0]

    def focus():
        a = actor(latest(audit))
        for _ in range(40):
            s = latest(audit)
            if s['cursor'] == [a['x'], a['y']] and not s.get('player_focus'):
                return s
            key('z');wait(lambda s: not s.get('player_focus'))
        raise AssertionError('Z未循環到既有希爾法')

    def capture(command_id, label):
        seen = set()
        try:
            while not stop.is_set() and len(seen) < 40:
                before = latest(audit)
                overlays = before.get('hd_heal_overlays') or []
                signature = (before.get('heal_owner'), before.get('heal_phase'),
                             tuple((o['entry'], tuple(o['rect'])) for o in overlays))
                if (not before.get('heal_scene') or before.get('heal_command_id') != command_id
                        or not before.get('hd_world') or not overlays or signature in seen):
                    stop.wait(.005);continue
                seen.add(signature)
                path = args.out/f'{label}-sample-{len(seen):02d}.png'
                command('import', '-window', window, str(path))
                after = latest(audit)
                row = dict(command_id=command_id, case=label, file=path.name, sha256=digest(path), before=before, after=after)
                if (after.get('heal_owner') == before.get('heal_owner') and after.get('hd_world')
                        and after.get('heal_command_id') == command_id):
                    samples.append(row)
                else:
                    rejected.append(row)
        except Exception as error:
            errors.append(str(error))

    try:
        # 四種完整演出各一次；另以同一來源施放13，專門檢查途中F2。
        for command_id, toggle in [(i, False) for i in range(13, 17)]+[(13, True)]:
            label = f'heal{command_id}'+('-f2' if toggle else '')
            if cases:
                key('F9');wait(lambda s: s.get('node') == args.battle_node and not s.get('heal_scene'))
                idle()
            before = focus();a = actor(before)
            assert command_id in a['command_ids'] and not a['acted']
            shot(f'{label}-focused')
            key('Return');wait(lambda s: s.get('move_preview'))
            # Confirm the actual current cell; do not rewrite coordinates.
            key('Return');wait(lambda s: s.get('ring_phase') == 'open' and not s.get('walking'))
            key('Left');key('Return');wait(lambda s: s.get('command_grid'))
            index = a['command_ids'].index(command_id)
            for _ in range(index):
                key('Down')
            wait(lambda s: s.get('command_selection') == index)
            key('Return');wait(lambda s: s.get('command_targeting') and s.get('command_target_id') == command_id)
            key('Escape');wait(lambda s: s.get('command_grid'))
            cancelled = latest(audit)
            assert actor(cancelled)['mp'] == a['mp'], '取消法術扣了MP'
            key('Return');wait(lambda s: s.get('command_targeting') and s.get('command_target_id') == command_id)
            target_state = latest(audit)
            destination = [a['x'], a['y']]
            assert destination in target_state.get('command_target_cells', []), '正常自身格未被原生selector接受'
            current = target_state['cursor']
            for axis, positive, negative in ((0, 'Right', 'Left'), (1, 'Down', 'Up')):
                for _ in range(abs(destination[axis]-current[axis])):
                    key(positive if destination[axis] > current[axis] else negative)
                current = latest(audit)['cursor']
            shot(f'{label}-target')
            stop.clear();worker = threading.Thread(target=capture, args=(command_id, label), daemon=True);worker.start()
            key('Return')
            started = wait(lambda s: s.get('heal_scene') and s.get('heal_command_id') == command_id, timeout=180)
            toggles = []
            if toggle:
                key('F2');wait(lambda s: not s.get('hd_theme'))
                toggles.append(dict(before=started, after=latest(audit)))
                shot('heal13-faithful-active')
                key('F2');wait(lambda s: s.get('hd_theme'))
                toggles.append(dict(after=latest(audit)))
            returned = idle()
            stop.set();worker.join(timeout=20)
            assert not worker.is_alive() and not errors, errors
            assert returned.get('hd_world') is True and actor(returned)['acted']
            assert actor(returned)['mp'] < a['mp'], '正常回復術沒有扣MP'
            assert [u['hp'] for u in before['units']] == [u['hp'] for u in returned['units']], '滿HP來源的上限改變'
            shot(f'{label}-returned')
            cases.append(dict(case=label, command_id=command_id, theme_toggle=toggle,
                              actor_identity=actor_id, confirmed_cell=destination,
                              before=before, target_state=target_state, cancelled=cancelled,
                              started=started, returned=returned, f2=toggles, state_injection=False))
            checkpoint()
        # Preserve the established F5 chapter-boundary contract, not live battle slots.
        key('F5');wait(lambda s: s.get('frame', 0) > cases[-1]['returned']['frame'])
        hd_saved = saved.read_bytes()
        key('F2');wait(lambda s: not s.get('hd_theme'))
        key('F9');wait(lambda s: s.get('node') == args.battle_node)
        idle();key('F5');assert saved.read_bytes() == hd_saved
        key('F2');wait(lambda s: s.get('hd_theme'))
        key('F9');wait(lambda s: s.get('node') == args.battle_node)
        warm = idle()
        fields = ('native_identity', 'battle_fig', 'x', 'y', 'hp', 'mp', 'lv', 'exp', 'acted', 'on_field', 'command_ids')
        persistent = lambda s: [{k: u.get(k) for k in fields} for u in s['units']]
        restart(args.battle_node, warm['gold'], 'heal-cold-title-load')
        cold = idle()
        assert persistent(warm) == persistent(cold) and saved.read_bytes() == hd_saved
        assert digest(args.town_save) == source_hash and digest(Path(__file__)) == helper_hash
        assert digest(Path(__file__).with_name('verify_fd2_hd_fonts_gui.py')) == tool_hash
        report = dict(schema='fd2-hd-heal-normal-gui-v1', status='PASS',
                      source_file=str(args.town_save), source_sha256=source_hash,
                      catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),
                      binary_sha256=digest(args.binary), gui_tool_sha256=tool_hash,
                      helper_sha256=helper_hash, saved_sha256=digest(saved),
                      cases=cases, samples=samples, rejected=rejected, screenshots=shots,
                      operations=operations, warm=warm, cold=cold,
                      state_injection=False, recording_created=False, new_process_title_load=True,
                      theme_save_bytes_equal=True, cold_units_equal=True,
                      normal_rng_setting='FD2_SEED=1；四種演出各一次，另一次13專驗F2，不重擲挑選，非原版骰序對拍。',
                      save_contract='F5章節邊界保存；不聲明保存戰中MP、位置、Acted。',
                      scope='既有強化來源，正常鍵盤四種滿血回復演出與取消、F2、返回、保存冷讀；非自然難度、完整章節或原版動態。')
        (args.out/'receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    finally:
        stop.set()
        if worker is not None:
            worker.join(timeout=20)
        checkpoint()
