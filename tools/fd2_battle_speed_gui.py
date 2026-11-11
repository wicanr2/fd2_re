"""既有普通LOAD驗證中的真實速度選單操作；不寫遊戲狀態。"""
import json
import time

from verify_fd2_hd_gui import latest, digest


def configure_battle_speed(args, audit, window, command, wait, shot, saved):
    initial = latest(audit)
    fields = ('native_identity', 'x', 'y', 'hp', 'mp', 'lv', 'exp', 'acted', 'on_field')
    units = lambda state: [{k: u.get(k) for k in fields} for u in state['units']]
    original_save = digest(saved)

    def click(x, y):
        before = latest(audit)['frame']
        command('xdotool', 'mousemove', '--window', window, str(x*3), str(y*3))
        time.sleep(.15)
        command('xdotool', 'click', '1')
        return wait(lambda s: s.get('frame', 0) >= before+2)

    click(402, 12)
    shot('battle-speed-menu-default')
    checks = []
    for x, y, game, battle in [(509, 96, 1, 2), (529, 64, 3, 2), (405, 64, 1, 2)]:
        click(x, y)
        state = wait(lambda s: s.get('game_speed') == game and s.get('battle_animation_speed') == battle)
        assert state['effective_game_speed'] == game, '戰場一般操作被戰鬥倍率影響'
        assert units(state) == units(initial) and state['cursor'] == initial['cursor'] and state['camera'] == initial['camera']
        assert digest(saved) == original_save, '速度設定改寫戰役'
        settings = next((args.out/'user-data').rglob('fd2_settings.json'))
        configured = json.loads(settings.read_text())
        assert configured['game_speed'] == game and configured['battle_animation_speed'] == battle
        checks.append(dict(game=game, battle=battle, effective=game, campaign_unchanged=True))
    shot('battle-speed-menu-independent-2x')
    click(467, 150)
    command('xdotool', 'mousemove', '--window', window, '1900', '1100')
    report = dict(status='PASS-NORMAL-TOOLBAR-CONFIGURATION', checks=checks,
                  source_save_sha256=original_save, runtime_state_injection=False)
    (args.out/'battle-speed-ui.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')


def capture_gold_ring_styles(args, audit, key, wait, shot, prefix):
    initial = latest(audit)
    assert initial['ring'] and initial['hd_theme'] and initial['hd_interface']
    protected = lambda s: [(u.get('native_identity'),u.get('x'),u.get('y'),u.get('hp'),u.get('mp'),u.get('acted')) for u in s['units']]
    rows = []
    for art, ui in [(True,True),(True,False),(False,False),(False,True),(True,True)]:
        state = latest(audit)
        if state['hd_theme'] != art:
            key('F2');wait(lambda s:s.get('hd_theme') is art)
        if latest(audit)['hd_interface'] != ui:
            key('F11');wait(lambda s:s.get('hd_interface') is ui)
        state = wait(lambda s:s.get('ring') and s.get('hd_theme') is art and s.get('hd_interface') is ui)
        assert protected(state) == protected(initial) and state['cursor'] == initial['cursor']
        name = prefix+f'-gold-ring-art{int(art)}-ui{int(ui)}'
        shot(name)
        rows.append(dict(art_hd=art,interface_hd=ui,file=name+'.png',units_unchanged=True))
    (args.out/'gold-command-ui.json').write_text(json.dumps(dict(status='PASS-NORMAL-RING-FOUR-STYLES',rows=rows,runtime_state_injection=False),indent=2)+'\n')
