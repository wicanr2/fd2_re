"""以普通移動／待機／END抵達屋後；導航只選正式可達格，不寫遊戲狀態。"""
from collections import deque
import hashlib
import json
from pathlib import Path
import time


def walk_behind_house(key, wait, shot, state, map_path, out):
    grid = json.loads(map_path.read_text())
    width, height = grid['w'], grid['h']
    goal = (10, 8)
    steps = []
    started = time.monotonic()
    tool_hash = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()

    def cursor_to(destination):
        for _ in range(100):
            current = state()['cursor']
            if tuple(current) == tuple(destination):
                return
            if current[0] != destination[0]:
                direction = 'Right' if current[0] < destination[0] else 'Left'
            else:
                direction = 'Down' if current[1] < destination[1] else 'Up'
            key(direction)
        raise AssertionError(('普通游標未抵達', state()['cursor'], destination))

    def own():
        actors = [u for u in state()['units'] if u['camp'] == 0]
        assert actors and actors[0]['on_field'] and actors[0]['hp'] > 0, '索爾未存活；保留本次正常抽樣失敗'
        return actors[0]

    def settle_round(old_round, old_turn):
        deadline = time.monotonic()+600
        while time.monotonic() < deadline:
            current = state()
            if current.get('error'):
                raise RuntimeError(current['error'])
            if (current['native_round'] > old_round and current['turn'] > old_turn
                    and not any(current.get(k) for k in ['ai_busy','battle_event','dialogues','banner_t',
                                                          'battle_scene','class_ui_job','system_end_turn_confirm'])):
                return
            if current.get('dialogues'):
                key('Return')
            else:
                time.sleep(.05)
        raise TimeoutError(('普通END未回玩家回合', state()))

    try:
        for turn in range(6):
            actor = own()
            assert not actor['acted'], '回合開始索爾已行動'
            cursor_to((actor['x'], actor['y']))
            key('Return')
            wait(lambda s: s.get('move_preview') is True)
            current = state()
            occupied = {(u['x'],u['y']) for u in current['units']
                        if u['on_field'] and u['hp'] > 0 and u is not actor}
            occupied.discard((actor['x'],actor['y']))
            distances = {goal: 0}
            queue = deque([goal])
            while queue:
                x,y = queue.popleft()
                for point in [(x-1,y),(x+1,y),(x,y-1),(x,y+1)]:
                    nx,ny = point
                    if (0 <= nx < width and 0 <= ny < height and point not in distances
                            and point not in occupied and grid['cost'][ny*width+nx] < 99):
                        distances[point] = distances[(x,y)]+1
                        queue.append(point)
            choices = [tuple(cell) for cell in current['move_reachable']
                       if tuple(cell) in distances and tuple(cell) not in occupied]
            assert choices, '正式可達集合沒有屋後路徑；不注入位置'
            destination = min(choices, key=lambda point: (distances[point],point[1],point[0]))
            assert distances[destination] < distances[(actor['x'],actor['y'])], '普通移動無法前進'
            cursor_to(destination)
            key('Return')
            wait(lambda s: s.get('ring') and not s.get('walking') and s.get('ring_phase') == 'open')
            shot(f'terrain-route-{turn+1}-arrived')
            steps.append({'round':current['native_round'], 'from':[actor['x'],actor['y']],
                          'to':list(destination), 'reachable_before':current['move_reachable'],
                          'after':state()})
            key('Down')
            key('Return')
            wait(lambda s: not s.get('ring') and not s.get('walking') and not s.get('move_preview'))
            assert own()['acted'], '普通待機尚未提交'
            if destination == goal:
                cursor_to((9,6))
                wait(lambda s: s.get('hd_world_terrain_foreground',0) > 0)
                shot('terrain-hd-occlusion')
                baseline = state()
                key('F2'); wait(lambda s: s.get('hd_theme') is False)
                shot('terrain-faithful-occlusion')
                key('F2'); wait(lambda s: s.get('hd_world_terrain_foreground',0) > 0)
                shot('terrain-hd-occlusion-restored')
                for field in ['units','view']:
                    assert state()[field] == baseline[field], ('屋後F2改變狀態',field)
                return steps
            # 空格開系統面板，再以Down→END與原生確認結束回合。
            current = state()
            occupied = {(u['x'],u['y']) for u in current['units'] if u['on_field'] and u['hp'] > 0}
            empty = min(((x,y) for y in range(height) for x in range(width) if (x,y) not in occupied),
                        key=lambda point: (abs(point[0]-current['cursor'][0])+abs(point[1]-current['cursor'][1]),point))
            cursor_to(empty)
            old_round, old_turn = current['native_round'], current['turn']
            key('Return'); wait(lambda s: s.get('ring') and s.get('ring_phase') == 'open')
            key('Down'); key('Return')
            wait(lambda s: s.get('system_end_turn_confirm') and not s.get('class_ui_job'))
            key('Return')
            settle_round(old_round, old_turn)
        raise AssertionError('六個普通回合內未抵達屋後；不重擲或改槽')
    finally:
        (out/'terrain-route.json').write_text(json.dumps(
            {'steps':steps,'final':state(),'goal':list(goal),'state_injection':False,
             'tool_sha256':tool_hash,'tool_source_unchanged':hashlib.sha256(Path(__file__).read_bytes()).hexdigest()==tool_hash,
             'elapsed_seconds':time.monotonic()-started},ensure_ascii=False,indent=2)+'\n')
