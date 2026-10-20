"""共用死亡HD的正常GUI輔助；由既有GUI主控制器擁有X11及程序生命週期。"""
import json
import time
from pathlib import Path

from verify_fd2_hd_gui import digest, latest


def run_ch23_death_event(args,audit,key,wait,shot,saved,command,window,restart,operations,shots,tool_hash):
    assert args.death_event==53 and args.battle_node=='battle_ch23' and args.town_save and args.world
    source_hash=digest(args.town_save)
    progress=[]; samples=[]; commander_samples=[]; portrait_samples=[]; commander_seen=set()
    commander_hd=args.death_commander_hd
    command('xdotool','windowsize',window,'1920','1200')
    def checkpoint():
        (args.out/'death-observation.json').write_text(json.dumps(dict(
            progress=progress,samples=samples,commander_samples=commander_samples,
            portrait_samples=portrait_samples,state=latest(audit),state_injection=False),ensure_ascii=False,indent=2)+'\n')
    def sample_commander(s):
        layers=[l for l in s.get('battle_source_layers') or []
                if l.get('archive','') in ('','FIGANI.DAT') and l['resource']==348 and l['fill']==-1]
        signature=(s.get('battle_owner'),s.get('battle_phase'),tuple((l['x'],l['y']) for l in layers))
        if (not commander_hd or not s.get('battle_scene') or not s.get('hd_battle')
                or not layers or signature in commander_seen or len(commander_samples)>=16):return
        commander_seen.add(signature);path=args.out/f'commander116-target-{len(commander_samples):02d}.png'
        command('import','-window',window,str(path))
        commander_samples.append(dict(file=path.name,sha256=digest(path),before=s,after=latest(audit)));checkpoint()
    def sample_portrait(s):
        if (not commander_hd or portrait_samples or s.get('dialogue_speaker')!=116
                or not s.get('hd_theme') or s.get('hd_overlay_rect') is None):return
        before=wait(lambda v:v.get('frame',0)>s['frame']+4 and v.get('dialogue_speaker')==116
                    and v.get('hd_theme') and v.get('hd_overlay_rect') is not None)
        path=args.out/'commander116-story-hd.png';command('import','-window',window,str(path))
        portrait_samples.append(dict(file=path.name,sha256=digest(path),before=before,after=latest(audit)))
        key('F2');wait(lambda v:not v.get('hd_theme'));shot('commander116-story-faithful')
        key('F2');wait(lambda v:v.get('hd_theme') and v.get('dialogue_speaker')==116 and v.get('hd_overlay_rect') is not None)
        shot('commander116-story-hd-restored');checkpoint()
    def clean(s):
        return not any(s.get(k) for k in ('dialogues','battle_scene','command_scene','battle_event','banner_t','walking','class_ui_job','unit_death_scene','ai_busy','system_end_turn_confirm'))
    def move(cell):
        for axis,positive,negative in ((0,'Right','Left'),(1,'Down','Up')):
            origin=latest(audit)['cursor'][axis]
            for _ in range(abs(cell[axis]-origin)):key(positive if cell[axis]>origin else negative)
        assert latest(audit)['cursor']==cell
    def attack(cell,target):
        move(cell);key('Return');wait(lambda s:s.get('ring_phase')=='open' and not s.get('walking'))
        key('Return');wait(lambda s:not s.get('ring'))
        move([target['x'],target['y']]);key('Return')
    def idle(timeout=600,after_round=None):
        deadline=time.monotonic()+timeout
        while time.monotonic()<deadline:
            s=latest(audit)
            if s.get('error'):raise RuntimeError(s['error'])
            if clean(s) and (after_round is None or (s.get('native_round',0)>after_round[0] and s.get('turn',0)>after_round[1])):return s
            if s.get('dialogues') and not any(s.get(k) for k in ('battle_scene','command_scene','unit_death_scene')):key('Return')
            else:time.sleep(.03)
        raise TimeoutError('正常回合未返回 '+str(latest(audit)))
    key('F2');wait(lambda s:not s.get('hd_theme'))
    key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots')
    key('Return');wait(lambda s:s.get('node')=='battle_ch23' and not s.get('title_phase'))
    idle();shot('death23-start-faithful')
    for approach in range(11):
        s=idle()
        gaia=next(u for u in s['units'] if u['camp']==0 and u.get('native_identity')==30 and u['on_field'] and u['hp']>0)
        for _ in range(32):
            s=latest(audit)
            if s['cursor']==[gaia['x'],gaia['y']] and not s.get('player_focus'):break
            key('z');wait(lambda v:not v.get('player_focus'))
        else:raise AssertionError('Z未選到蓋亞')
        key('Return');wait(lambda s:s.get('move_preview'))
        s=latest(audit);target=next(u for u in s['units'] if u['camp']==2 and u['battle_fig']==116 and u['on_field'] and u['hp']>0)
        occupied={(u['x'],u['y']) for u in s['units'] if u['on_field'] and u['hp']>0}
        cells=[p for p in s['move_reachable'] if tuple(p) not in occupied or p==s['cursor']]
        distance=lambda p:abs(p[0]-target['x'])+abs(p[1]-target['y'])
        legal=[p for p in cells if 1<=distance(p)<=1]
        if legal:
            cell=min(legal,key=lambda p:(abs(p[0]-gaia['x'])+abs(p[1]-gaia['y']),p))
            if commander_hd:
                key('F2');wait(lambda v:v.get('hd_theme'))
            attack(cell,target)
            progress.append(dict(action='normal-attack-commander',before=s,cell=cell));checkpoint()
            break
        assert approach<10,'十次END仍無法正常接敵'
        cell=min(cells,key=lambda p:(distance(p),p[1],p[0]))
        if distance(cell)>=distance(s['cursor']):
            blockers=[(p,u) for p in cells for u in s['units'] if u['camp']==2 and u['hp']>0 and u['on_field']
                      and abs(p[0]-u['x'])+abs(p[1]-u['y'])==1 and distance([u['x'],u['y']])<distance(s['cursor'])]
            assert blockers,'普通路徑無法前進或清除阻擋敵軍'
            cell,blocker=min(blockers,key=lambda v:(distance([v[1]['x'],v[1]['y']]),distance(v[0]),v[0]))
            attack(cell,blocker)
            idle()
            progress.append(dict(action='normal-attack-blocker',cell=cell,target=blocker));checkpoint()
        else:
            move(cell);key('Return');wait(lambda s:s.get('ring_phase')=='open' and not s.get('walking'))
            key('Down');key('Return');wait(lambda s:not s.get('ring') and not s.get('class_ui_job') and not s.get('move_preview'))
        s=latest(audit);occupied={(u['x'],u['y']) for u in s['units'] if u['on_field'] and u['hp']>0}
        grid=json.loads((args.assets/'assets/maps/map22/map.json').read_text())
        empty=min(([x,y] for y in range(grid['h']) for x in range(grid['w']) if (x,y) not in occupied),key=lambda p:(abs(p[0]-s['cursor'][0])+abs(p[1]-s['cursor'][1]),p))
        move(empty);key('Return');wait(lambda s:s.get('ring_phase')=='open')
        key('Down');key('Return');wait(lambda s:s.get('system_end_turn_confirm') and not s.get('class_ui_job'))
        key('Return');after=idle(after_round=(s['native_round'],s['turn']))
        progress.append(dict(action='normal-move-wait-end',cell=cell,before=s,after=after));checkpoint()
    event_seen=False;hd_on=False;seen=set();tail=False
    deadline=time.monotonic()+360
    while time.monotonic()<deadline:
        s=latest(audit)
        if s.get('error'):raise RuntimeError(s['error'])
        sample_commander(s);sample_portrait(s)
        if s.get('unit_death_scene') and s.get('unit_death_kind')==args.death_kind:
            event_seen=True
            if not hd_on:
                if not s.get('hd_theme'):
                    key('F2');wait(lambda s:s.get('hd_theme'))
                hd_on=True;continue
            frame=s.get('unit_death_presented_frame',-1)
            if s.get('unit_death_presented_hd') and frame not in seen and not tail:
                seen.add(frame);path=args.out/f'death23-event-{frame:02d}.png'
                command('import','-window',window,str(path))
                samples.append(dict(file=path.name,sha256=digest(path),before=s,after=latest(audit)));checkpoint()
            if frame>=22 and not tail:
                key('F2');wait(lambda s:not s.get('hd_theme'));tail=True
        elif event_seen:
            if bool(latest(audit).get('hd_theme'))!=commander_hd:
                key('F2');wait(lambda s:bool(s.get('hd_theme'))==commander_hd)
            break
        elif s.get('dialogues') and not s.get('battle_scene'):key('Return')
        else:time.sleep(.005)
    else:raise TimeoutError('普通擊倒隊長後未觀測到死亡事件')
    assert event_seen and hd_on and any(v['before'].get('unit_death_phase')=='overlay_a' for v in samples) and any(v['before'].get('unit_death_phase')=='overlay_b' for v in samples)
    deadline=time.monotonic()+480
    victory_ack=False
    while time.monotonic()<deadline:
        s=latest(audit)
        if s.get('error'):raise RuntimeError(s['error'])
        sample_portrait(s)
        if s.get('node')=='preparation_ch24' and not s.get('preparation_ui_busy'):break
        if (not victory_ack and s.get('node')=='battle_ch23' and clean(s)
                and not any(u['camp']==2 and u['hp']>0 and u['on_field'] for u in s['units'])):
            key('Return');victory_ack=True
            progress.append(dict(action='normal-victory-confirm',before=s));checkpoint()
        elif s.get('dialogues') or s.get('node','').startswith('story_'):key('Return')
        else:time.sleep(.03)
    else:raise TimeoutError('死亡演出後未進入第24章整備')
    if latest(audit).get('hd_theme'):
        key('F2');wait(lambda s:not s.get('hd_theme'))
    shot('death23-returned-preparation-faithful')
    latest_frame=latest(audit)['frame']
    key('F5');wait(lambda s:s.get('frame',0)>latest_frame+2)
    assert saved.is_file() and json.loads(saved.read_text())['node']=='preparation_ch24'
    faithful_hash=digest(saved)
    key('F2');wait(lambda s:s.get('hd_theme'))
    shot('death23-returned-preparation-hd')
    key('F5');wait(lambda s:s.get('frame',0)>latest_frame+5)
    assert digest(saved)==faithful_hash,'F2改變正式存檔'
    before=latest(audit)
    restart('preparation_ch24',before['gold'],'death23-cold-load-preparation')
    assert digest(args.town_save)==source_hash
    if commander_hd:
        assert commander_samples and len(portrait_samples)==1,'隊長待機或116故事頭像未取得正常視窗樣本'
    checkpoint()
    (args.out/'receipt.json').write_text(json.dumps(dict(
        schema='fd2-shared-death-normal-gui-v1',status='PASS',binary_sha256=digest(args.binary),
        gui_tool_sha256=tool_hash,helper_sha256=digest(Path(__file__)),source_sha256=source_hash,
        source_file=str(args.town_save),saved_sha256=faithful_hash,samples=samples,progress=progress,
        operations=operations,screenshots=shots,event_id=53,death_kind=args.death_kind,state_injection=False,recording_created=False,
        commander_hd=commander_hd,commander_samples=commander_samples,portrait_samples=portrait_samples,
        catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),
        final_node=latest(audit)['node'],scope='章前強化來源；正常移動、END、攻擊隊長與死亡事件抽樣、自然戰後、整備F2和保存冷讀。非自然難度、整章E2或新原版對拍。'),ensure_ascii=False,indent=2)+'\n')


def capture_standalone_ch23_source(args,audit,key,wait,shot,saved,native_saved,source_hash,tool_hash,operations,shots):
    assert args.battle_node=='battle_ch23' and args.native_entry_theme=='faithful'
    nodes=json.loads((args.assets/'assets/scenarios/campaign_full.json').read_text())['nodes']
    prep=nodes['preparation_ch23']
    assert prep['type']=='preparation' and not prep.get('cancel')
    assert prep['required_party_identities']==[24] and prep['preparation_front_identities']==[24]
    assert prep['next']=='story_ch23' and nodes['story_ch23']['next']=='battle_ch23'
    key('F2');wait(lambda s:not s.get('hd_theme'))
    key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots')
    shot('native23-title-load');key('Return')
    def ready(**fields):
        return wait(lambda s:s.get('node')=='preparation_ch23' and not s.get('preparation_ui_busy')
                    and all(s.get(k)==v for k,v in fields.items()))
    ready(preparation_selecting=False,preparation_slots=False)
    shot('native23-record-question')
    assert digest(native_saved)==source_hash
    key('Escape');ready(preparation_selecting=True)
    initial=latest(audit)
    limit=initial['preparation_limit']
    assert 1<=limit<=26 and initial['preparation_selected']==0
    for _ in range(limit):
        key('Return')
        wait(lambda s:not s.get('preparation_ui_busy'))
    ready(preparation_selecting=False,preparation_confirm=True)
    assert not latest(audit).get('preparation_required')
    shot('native23-selected-confirm');key('Return')
    deadline=time.monotonic()+480
    while time.monotonic()<deadline:
        state=latest(audit)
        if state.get('error'):raise RuntimeError(state['error'])
        if state.get('node')=='battle_ch23' and not state.get('dialogues') and not state.get('battle_event') and not state.get('banner_t'):
            break
        assert state.get('node') in ('preparation_ch23','story_ch23','battle_ch23')
        if state.get('node')=='story_ch23' or state.get('dialogues'):key('Return')
        else:time.sleep(.05)
    else:raise TimeoutError('第23章獨立整備正常出發未完成')
    before=latest(audit);key('F5');wait(lambda s:s.get('frame',0)>before['frame'])
    assert saved.is_file() and json.loads(saved.read_text())['node']=='battle_ch23'
    assert digest(args.native_save)==source_hash and digest(native_saved)==source_hash
    shot('native23-battle-f5')
    report=dict(schema='fd2-native-battle-source-gui-v1',status='PASS',binary_sha256=digest(args.binary),
                gui_tool_sha256=tool_hash,helper_sha256=digest(Path(__file__)),native_source_sha256=source_hash,
                saved_sha256=digest(saved),saved_file=str(saved),node='battle_ch23',entry_theme='faithful',
                state_injection=False,units=latest(audit)['units'],screenshots=shots,operations=operations,
                recording_created=False,preparation_limit=limit,
                scope='正常原生標題LOAD、獨立整備必選24、確認出發、章首與F5。章前強化與既有傷亡依manifest揭露；非自然難度或整章E2。')
    (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
