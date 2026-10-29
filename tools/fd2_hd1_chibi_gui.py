"""已定版HD1小人物的普通選取、走行取消與保存冷讀抽樣。"""
import json
from pathlib import Path
from verify_fd2_hd_gui import digest,latest

SOURCE_SHA='089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2'

def run_normal_chibi(args,audit,key,wait,shot,saved,command,window,restart,operations,shots,tool_hash):
    assert digest(args.town_save)==SOURCE_SHA and args.battle_node=='battle_ch01'
    initial=latest(audit)
    fields=('native_identity','x','y','hp','mp','lv','exp','acted','on_field')
    units=lambda s:[{k:u.get(k) for k in fields} for u in s['units']]
    ares=next(u for u in initial['units'] if u.get('native_identity')==4 and u['camp']==0 and u['on_field'] and u['hp']>0)
    origin=[ares['x'],ares['y']]
    def cursor(target):
        for axis,positive,negative in ((0,'Right','Left'),(1,'Down','Up')):
            before=latest(audit)['cursor']
            for _ in range(abs(target[axis]-before[axis])):key(positive if target[axis]>before[axis] else negative)
        wait(lambda s:s.get('cursor')==target and not s.get('walking'))
    cursor(origin)
    wait(lambda s:s.get('hd_world') and any(u['group']==4 for u in s.get('hd_world_units',[])),timeout=180)
    styles=[]
    for label,keys in [('hd-hd',[]),('hd-original',['F11']),('original-original',['F2']),('original-hd',['F11']),('hd-hd-restored',['F2'])]:
        for k in keys:
            before=latest(audit);key(k)
            wait(lambda s:s.get('hd_theme')!=before.get('hd_theme') if k=='F2' else s.get('interface_mode')!=before.get('interface_mode'),timeout=180)
        command('xdotool','windowsize',window,'1920','1200')
        state=latest(audit);assert units(state)==units(initial) and not state['error']
        shot('ares-chibi-'+label);styles.append(dict(label=label,state=latest(audit)))
    key('Return');wait(lambda s:s.get('move_preview'))
    state=latest(audit);occupied={(u['x'],u['y']) for u in state['units'] if u['on_field'] and u['hp']>0}
    reachable=state['move_reachable'];x,y=origin
    destination=next(p for p in [[x,y-1],[x+1,y],[x-1,y],[x,y+1]] if p in reachable and tuple(p) not in occupied)
    cursor(destination);key('Return')
    arrival=wait(lambda s:s.get('ring') and s.get('ring_phase')=='open' and not s.get('walking'),timeout=180)
    actual=next(u for u in arrival['units'] if u.get('native_identity')==4 and u['camp']==0)
    assert [actual['x'],actual['y']]==destination and not actual['acted']
    shot('ares-chibi-walk-arrived')
    key('Escape');wait(lambda s:not s.get('ring') and not s.get('walking'),timeout=180)
    if latest(audit).get('move_preview'):key('Escape');wait(lambda s:not s.get('move_preview'))
    returned=latest(audit);assert units(returned)==units(initial)
    shot('ares-chibi-walk-cancelled')
    key('F5');saved_hd=saved.read_bytes()
    key('F2');wait(lambda s:not s.get('hd_theme'));key('F5');assert saved.read_bytes()==saved_hd
    key('F11');wait(lambda s:s.get('interface_mode')=='original');key('F5');assert saved.read_bytes()==saved_hd
    key('F11');wait(lambda s:s.get('interface_mode')=='hd')
    key('F2');wait(lambda s:s.get('hd_theme'))
    restart('battle_ch01',returned['gold'],'ares-chibi-cold-title-load')
    cold=latest(audit);assert units(cold)==units(returned) and saved.read_bytes()==saved_hd
    assert digest(args.town_save)==SOURCE_SHA
    report=dict(schema='fd2-hd1-chibi-normal-gui-v1',status='PASS',source_sha256=SOURCE_SHA,binary_sha256=digest(args.binary),catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),gui_tool_sha256=tool_hash,helper_sha256=digest(Path(__file__)),initial=initial,styles=styles,arrival=arrival,returned=returned,cold=cold,origin=origin,destination=destination,screenshots=shots,operations=operations,saved_sha256=digest(saved),theme_and_interface_save_bytes_equal=True,state_injection=False,recording_created=False,new_process_title_load=True,scope='既有普通首章來源經標題LOAD、游標選亞雷斯、四組呈現、合法一格走行與取消、F5及新程序LOAD；不改寫名冊／HP／移動或規則，不宣稱整章或原版動態對拍。')
    (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
