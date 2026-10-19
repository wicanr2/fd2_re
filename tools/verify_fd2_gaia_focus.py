"""沿共用GUI按鍵與收據，普通選取蓋亞、移動及取消；不寫入遊戲狀態。"""


def focus_gaia(key, wait, shot, state, label):
    own = [u for u in state()['units'] if u['camp'] == 0 and u['on_field'] and u['hp'] > 0]
    assert len(own) == 4 and own[3]['hp'] == 50, '不是既定普通首章起點'
    target = [own[3]['x'], own[3]['y']]
    for _ in range(5):
        key('z')
        wait(lambda s: not s.get('player_focus'))
        if state()['cursor'] == target:
            break
    else:
        raise AssertionError('普通循環選角未抵達蓋亞')
    wait(lambda s: s.get('hd_world') and any(u['group'] == 30 for u in s.get('hd_world_units', [])))
    shot(label)
    return state()


def sample_gaia(key, wait, shot, state):
    baseline = focus_gaia(key, wait, shot, state, 'gaia-hd-focused')
    key('F2'); wait(lambda s: s.get('hd_theme') is False)
    shot('gaia-faithful-focused')
    assert state()['units'] == baseline['units'] and state()['view'] == baseline['view']
    key('F2'); wait(lambda s: s.get('hd_world') is True)
    shot('gaia-hd-focused-restored')
    assert state()['units'] == baseline['units'] and state()['view'] == baseline['view']
    key('Return'); wait(lambda s: s.get('move_preview') is True)
    origin = state()['cursor']
    destination = [origin[0], origin[1]-3]
    assert destination in state()['move_reachable'], '北移三格不在正式可達集合'
    occupied = {(u['x'], u['y']) for u in state()['units'] if u['on_field'] and u['hp'] > 0}
    assert tuple(destination) not in occupied
    for _ in range(3):
        key('Up')
    assert state()['cursor'] == destination
    key('Return')
    wait(lambda s: s.get('ring') and not s.get('walking') and s.get('ring_phase') == 'open')
    shot('gaia-hd-walk-arrived')
    arrival = state()
    own = [u for u in arrival['units'] if u['camp'] == 0]
    assert [own[3]['x'], own[3]['y']] == destination
    key('Escape'); wait(lambda s: not s.get('ring') and not s.get('walking'))
    if state().get('move_preview'):
        key('Escape'); wait(lambda s: not s.get('move_preview'))
    own = [u for u in state()['units'] if u['camp'] == 0]
    assert [own[3]['x'], own[3]['y']] == origin and not own[3]['acted']
    shot('gaia-hd-walk-cancelled')
    return {'baseline': baseline, 'arrival': arrival, 'cancelled': state(),
            'ordinary_walk': {'from': origin, 'to': destination, 'direction': 'north'},
            'limits': '普通北移與取消；四方向十二幀另作圖源審查，未宣稱全部方向皆在本次GUI行走。'}
