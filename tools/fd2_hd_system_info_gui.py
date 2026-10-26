"""普通鍵盤開關戰場資訊；保留原版任意鍵收窗及章節邊界存檔。"""
import json
from pathlib import Path

from verify_fd2_hd_gui import digest, latest

SOURCE_SHA = '5d64f3a0bf9aa56f7e77f51e8cfe45fe8c14ab78f2703b966fbc0e142e7bd871'


def run_normal_system_info(args, audit, key, wait, shot, saved, command, window,
                           restart, operations, shots, tool_hash):
    source_hash, helper_hash = digest(args.town_save), digest(Path(__file__))
    assert source_hash == SOURCE_SHA and args.battle_node == 'battle_ch23'
    initial = latest(audit)
    fields = ('native_identity', 'x', 'y', 'hp', 'mp', 'lv', 'exp', 'acted', 'on_field')
    units = lambda state: [{k: u.get(k) for k in fields} for u in state['units']]
    occupancy = {(u['x'], u['y']) for u in initial['units'] if u['on_field'] and u['hp'] > 0}
    x, y = initial['cursor']
    destination = next((x+dx, y+dy) for dx, dy in [(1, 0), (-1, 0), (0, -1), (0, 1)]
                       if x+dx >= 0 and y+dy >= 0 and (x+dx, y+dy) not in occupancy)
    for axis, positive, negative in ((0, 'Right', 'Left'), (1, 'Down', 'Up')):
        current = latest(audit)['cursor']
        for _ in range(abs(destination[axis]-current[axis])):
            key(positive if destination[axis] > current[axis] else negative)
    assert tuple(latest(audit)['cursor']) == destination

    def idle():
        return wait(lambda s: not s.get('system_info_phase') and not s.get('ring')
                    and not s.get('dialogues') and not s.get('banner_t') and not s.get('walking'), timeout=120)

    def open_info(label):
        key('Return');wait(lambda s: s.get('system_cursor_overlay') and s.get('ring_phase') == 'open')
        key('Up');key('Return')
        wait(lambda s: s.get('system_nested_open') and s.get('ring_phase') == 'open')
        key('Up');key('Return')
        state = wait(lambda s: s.get('system_info_phase') == 'steady', timeout=120)
        assert not state['error'] and units(state) == units(initial)
        shot(label)
        return latest(audit)

    hd = open_info('system-info-hd-steady')
    assert hd['hd_theme'] and hd['hd_world'] and hd['system_info_hd']
    key('F2');wait(lambda s: not s.get('hd_theme'))
    faithful_return = idle()  # 原版資訊窗任意鍵收合，F2也走同一條路徑。
    command('xdotool', 'windowsize', window, '1920', '1200')
    faithful = open_info('system-info-faithful-steady')
    assert faithful['system_info_values'] == hd['system_info_values']
    key('F2');wait(lambda s: s.get('hd_theme'))
    hd_return = idle()
    restored = open_info('system-info-hd-restored')
    assert restored['system_info_values'] == hd['system_info_values'] and restored['system_info_hd']
    key('Escape');returned = idle()
    assert units(returned) == units(initial) and returned['hd_world']
    shot('system-info-returned')
    locale_cases = []
    if args.system_info_locales:
        for locale in ('zh-Hant', 'zh-Hans', 'en', 'ja'):
            for _ in range(4):
                if latest(audit)['locale'] == locale:
                    break
                previous = latest(audit)['locale']
                key('F4');wait(lambda s: s['locale'] != previous and not s.get('error'), timeout=120)
            assert latest(audit)['locale'] == locale
            for modern in (True, False):
                if bool(latest(audit)['hd_theme']) != modern:
                    key('F2');wait(lambda s: bool(s['hd_theme']) == modern, timeout=180)
                command('xdotool', 'windowsize', window, '1920', '1200')
                name = f'system-info-{locale}-' + ('hd' if modern else 'native')
                state = open_info(name)
                assert state['locale'] == locale and state['system_info_values'] == hd['system_info_values']
                assert state['system_info_text_index'] == 641
                if modern or locale != 'zh-Hant':
                    assert state['system_info_lines'] and state['system_info_captions']
                locale_cases.append(dict(locale=locale, modern=modern, screenshot=name+'.png', state=state))
                key('Escape');idle()
        if not latest(audit)['hd_theme']:
            key('F2');wait(lambda s: s['hd_theme'], timeout=180)
        for _ in range(4):
            if latest(audit)['locale'] == 'zh-Hant':
                break
            previous = latest(audit)['locale']
            key('F4');wait(lambda s: s['locale'] != previous and not s.get('error'), timeout=120)
    key('F5');saved_hd = saved.read_bytes()
    key('F2');wait(lambda s: not s.get('hd_theme'))
    key('F5');assert saved.read_bytes() == saved_hd
    key('F2');wait(lambda s: s.get('hd_theme'))
    key('F9');wait(lambda s: s.get('node') == args.battle_node and not s.get('dialogues') and not s.get('banner_t'), timeout=180)
    warm = latest(audit)
    restart(args.battle_node, warm['gold'], 'system-info-cold-title-load')
    cold = latest(audit)
    assert units(warm) == units(cold) and saved.read_bytes() == saved_hd
    assert digest(args.town_save) == source_hash and digest(Path(__file__)) == helper_hash
    assert digest(Path(__file__).with_name('verify_fd2_hd_fonts_gui.py')) == tool_hash
    report = dict(schema='fd2-system-info-normal-gui-v1', status='PASS', source_sha256=source_hash,
                  binary_sha256=digest(args.binary), catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),
                  gui_tool_sha256=tool_hash, helper_sha256=helper_hash, saved_sha256=digest(saved),
                  initial=initial, hd=hd, faithful=faithful, restored=restored, returned=returned,
                  faithful_return=faithful_return, hd_return=hd_return, warm=warm, cold=cold,
                  operations=operations, screenshots=shots, locale_cases=locale_cases, state_injection=False, recording_created=False,
                  theme_save_bytes_equal=True, new_process_title_load=True, cold_units_equal=True,
                  normal_rng_setting='FD2_SEED=1，無戰鬥與重擲，不作原版骰序對拍。',
                  scope='既有第23章強化來源的普通鍵盤資訊窗、F2任意鍵收合、回地圖及章節邊界保存冷讀。locale_cases另列四語／兩主題實際開窗，不宣稱原版動態對拍。')
    (args.out/'receipt.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
