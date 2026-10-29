"""以普通鍵盤進END、切換四組呈現、選NO返回；不執行敵方回合。"""
import json
from pathlib import Path

from verify_fd2_hd_gui import digest,latest
from fd2_hd_system_info_gui import SOURCE_SHA

def run_normal_system_confirmation(args,audit,key,wait,shot,saved,command,window,restart,operations,shots,tool_hash):
    source_hash=digest(args.town_save)
    assert source_hash==SOURCE_SHA and args.battle_node=='battle_ch23'
    initial=latest(audit)
    fields=('native_identity','x','y','hp','mp','lv','exp','acted','on_field')
    units=lambda s:[{k:u.get(k) for k in fields} for u in s['units']]
    occupancy={(u['x'],u['y']) for u in initial['units'] if u['on_field'] and u['hp']>0}
    x,y=initial['cursor']
    destination=next((x+dx,y+dy) for dx,dy in [(1,0),(-1,0),(0,-1),(0,1)] if x+dx>=0 and y+dy>=0 and (x+dx,y+dy) not in occupancy)
    for axis,positive,negative in ((0,'Right','Left'),(1,'Down','Up')):
        current=latest(audit)['cursor']
        for _ in range(abs(destination[axis]-current[axis])):key(positive if destination[axis]>current[axis] else negative)
    key('Return');wait(lambda s:s.get('system_cursor_overlay') and s.get('ring_phase')=='open')
    key('Down');key('Return')
    wait(lambda s:s.get('system_end_turn_confirm') and not s.get('class_ui_job'),timeout=180)
    states=[]
    for label,keys in [('hd-hd',[]),('hd-original',['F11']),('original-original',['F2']),('original-hd',['F11']),('hd-hd-restored',['F2'])]:
        for k in keys:
            before=latest(audit);key(k)
            if k=='F2':wait(lambda s:s.get('hd_theme')!=before.get('hd_theme') and s.get('system_end_turn_confirm') and not s.get('class_ui_job'),timeout=180)
            else:wait(lambda s:s.get('interface_mode')!=before.get('interface_mode') and s.get('system_end_turn_confirm') and not s.get('class_ui_job'),timeout=180)
        command('xdotool','windowsize',window,'1920','1200')
        state=latest(audit);assert not state['error'] and units(state)==units(initial)
        shot('system-confirmation-'+label);states.append(dict(label=label,state=latest(audit)))
    key('Right');key('Return')
    returned=wait(lambda s:not s.get('system_end_turn_ui') and not s.get('class_ui_job') and not s.get('ring') and not s.get('banner_t'),timeout=180)
    assert units(returned)==units(initial) and returned['gold']==initial['gold']
    shot('system-confirmation-returned')
    key('F5');saved_hd=saved.read_bytes()
    key('F2');wait(lambda s:not s.get('hd_theme'));key('F5');assert saved.read_bytes()==saved_hd
    key('F11');wait(lambda s:s.get('interface_mode')=='original');key('F5');assert saved.read_bytes()==saved_hd
    key('F11');wait(lambda s:s.get('interface_mode')=='hd')
    key('F2');wait(lambda s:s.get('hd_theme'))
    assert digest(args.town_save)==source_hash
    report=dict(schema='fd2-system-confirmation-normal-gui-v1',status='PASS',source_sha256=source_hash,binary_sha256=digest(args.binary),catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),gui_tool_sha256=tool_hash,helper_sha256=digest(Path(__file__)),initial=initial,states=states,returned=returned,operations=operations,screenshots=shots,saved_sha256=digest(saved),theme_and_interface_save_bytes_equal=True,state_injection=False,recording_created=False,new_process_title_load=False,scope='既有第23章強化來源經標題LOAD、普通END、F2／F11、NO及自然返回，保存bytes保持；不新增自然難度、原版動態、整章或冷讀聲明。')
    (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
