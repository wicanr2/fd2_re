"""共用死亡HD的正常GUI輔助；由既有GUI主控制器擁有X11及程序生命週期。"""
import json
import copy
import threading
import time
from pathlib import Path

from verify_fd2_hd_gui import digest, latest


STATUS_BASE_SHA = '089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2'


def status_source(base):
    """只修改已審查來源的兩欄；正式遊戲與來源檔保持不變。"""
    source = json.loads(base.read_text())
    if digest(base) != STATUS_BASE_SHA or source['node'] != 'battle_ch01':
        raise ValueError('狀態來源須為已審查首章槽')
    result = copy.deepcopy(source)
    member = result['party_roster']['4']
    if (member['native_transient'] != [0]*6 or member['HP'] != member['MaxHP']
            or not member['has_native_record_byte5'] or member['native_record_byte6'] != 2):
        raise ValueError('身分4的原始狀態不符')
    member['HP'] = member['MaxHP']//10
    member['native_transient'][3] = 2
    changes = [dict(path='party_roster.4.'+field, before=source['party_roster']['4'][field], after=member[field])
               for field in ('HP', 'native_transient')]
    return result, dict(schema='fd2-hd-status-death-source-v1', base_file=str(base),
                        base_sha256=digest(base), actor_identity=4, changes=changes,
                        in_chapter_injection=False, hp_lock=False,
                        scope='既有建構首章槽，另在章前設定低HP與原生中毒計時；不證明正常取得狀態或整章E2。')


def run_status_death(args,audit,key,wait,shot,saved,command,window,
                     restart,operations,shots,tool_hash):
    source_hash = digest(args.town_save)
    source = json.loads(args.town_save.read_text())
    manifest_path = args.town_save.with_name('manifest.json')
    manifest = json.loads(manifest_path.read_text())
    assert manifest['schema'] == 'fd2-hd-status-death-source-v1'
    assert manifest['base_sha256'] == STATUS_BASE_SHA and manifest['output_sha256'] == source_hash
    assert manifest['in_chapter_injection'] is False and manifest['hp_lock'] is False
    assert source['party_roster']['4']['native_transient'] == [0,0,0,2,0,0]
    cases,samples,rejected,errors = [],[],[],[]
    stop = threading.Event()
    worker = None

    def checkpoint():
        (args.out/'status-death-observation.json').write_text(json.dumps(dict(
            status='OBSERVATION-ONLY',cases=cases,samples=samples,rejected=rejected,errors=errors,
            source_sha256=source_hash,source_manifest=manifest,
            in_chapter_injection=False,hp_lock=False,recording_created=False),ensure_ascii=False,indent=2)+'\n')

    def clean(s):
        return not any(s.get(k) for k in ('dialogues','banner_t','class_ui_job','transient_ui',
                    'battle_scene','command_scene','unit_death_scene','ai_busy','walking','system_end_turn_confirm'))

    def actor(s):
        rows=[u for u in s['units'] if u.get('native_identity')==4 and u['camp']==0]
        assert len(rows)==1
        return rows[0]

    def idle():
        deadline=time.monotonic()+240
        while time.monotonic()<deadline:
            s=latest(audit)
            if s.get('error'):raise RuntimeError(s['error'])
            if clean(s):return s
            if s.get('dialogues') and not s.get('unit_death_scene'):key('Return')
            else:time.sleep(.02)
        raise TimeoutError('狀態致死未返回玩家操作')

    def capture(label):
        seen=set()
        try:
            while not stop.is_set():
                before=latest(audit)
                frame=before.get('unit_death_presented_frame',-1)
                signature=(before.get('unit_death_owner'),frame,before.get('unit_death_presented_hd'))
                if (not before.get('unit_death_scene') or before.get('unit_death_kind')!='status'
                        or not before.get('unit_death_presented_hd') or frame<13 or frame>24
                        or signature in seen or len(seen)>=16):
                    stop.wait(.005);continue
                seen.add(signature)
                path=args.out/f'status-death-{label}-{frame:02d}.png'
                command('import','-window',window,str(path))
                after=latest(audit)
                row=dict(case=label,file=path.name,sha256=digest(path),before=before,after=after)
                if after.get('unit_death_owner')==before.get('unit_death_owner'):
                    samples.append(row)
                else:rejected.append(row)
        except Exception as error:errors.append(str(error))

    try:
        for toggle in (False,True):
            label='f2' if toggle else 'full'
            if cases:
                key('F9');wait(lambda s:s.get('node')==args.battle_node and not s.get('unit_death_scene'))
            before=idle()
            assert before.get('hd_world') and actor(before)['hp']==source['party_roster']['4']['HP']>0
            # 首章西側敵軍可先攻擊低HP角色。沿真正可達集合向遠離敵軍處退避，
            # 不提高防禦、不搬座標，也不把普通攻擊死亡當作status通過。
            a=actor(before)
            for _ in range(32):
                if latest(audit)['cursor']==[a['x'],a['y']] and not latest(audit).get('player_focus'):break
                key('z');wait(lambda s:not s.get('player_focus'))
            else:raise AssertionError('Z未選到狀態測試角色')
            key('Return');movement=wait(lambda s:s.get('move_preview'))
            enemies=[u for u in movement['units'] if u['camp']==2 and u['hp']>0 and u['on_field']]
            occupied={(u['x'],u['y']) for u in movement['units'] if u['hp']>0 and u['on_field']}
            candidates=[p for p in movement['move_reachable'] if tuple(p) not in occupied]
            distance=lambda p:min(abs(p[0]-u['x'])+abs(p[1]-u['y']) for u in enemies)
            retreat=max(candidates,key=lambda p:(distance(p),p[0],p[1]))
            assert distance(retreat)>distance([a['x'],a['y']]),'無正常安全退避格'
            for axis,positive,negative in ((0,'Right','Left'),(1,'Down','Up')):
                current=latest(audit)['cursor'][axis]
                for _ in range(abs(retreat[axis]-current)):key(positive if retreat[axis]>current else negative)
            key('Return');wait(lambda s:s.get('ring_phase')=='open' and not s.get('walking'))
            key('Down');key('Return');wait(lambda s:not s.get('ring') and not s.get('move_preview') and not s.get('class_ui_job'))
            moved=idle()
            assert actor(moved)['hp']==actor(before)['hp'] and actor(moved)['acted']
            assert [actor(moved)['x'],actor(moved)['y']]==retreat
            shot(f'status-death-{label}-retreated')
            before=moved
            occupied={(u['x'],u['y']) for u in before['units'] if u['on_field'] and u['hp']>0}
            # 取目前游標旁的空地；逐鍵移動，不改座標。
            nodes=json.loads((args.assets/'assets/scenarios/campaign_full.json').read_text())['nodes']
            grid=json.loads((args.assets/nodes[args.battle_node]['map']/'map.json').read_text())
            empty=min(([x,y] for y in range(grid['h']) for x in range(grid['w']) if (x,y) not in occupied),
                      key=lambda p:(abs(p[0]-before['cursor'][0])+abs(p[1]-before['cursor'][1]),p))
            for axis,positive,negative in ((0,'Right','Left'),(1,'Down','Up')):
                current=latest(audit)['cursor'][axis]
                for _ in range(abs(empty[axis]-current)):key(positive if empty[axis]>current else negative)
            key('Return');wait(lambda s:s.get('ring_phase')=='open')
            key('Down');key('Return');wait(lambda s:s.get('system_end_turn_confirm') and not s.get('class_ui_job'))
            stop.clear();worker=threading.Thread(target=capture,args=(label,),daemon=True);worker.start()
            key('Return')
            deadline=time.monotonic()+240
            started=None
            while time.monotonic()<deadline:
                s=latest(audit)
                if s.get('error'):raise RuntimeError(s['error'])
                if s.get('unit_death_scene') and s.get('unit_death_kind')=='status':started=s;break
                if s.get('dialogues'):key('Return')
                else:time.sleep(.005)
            assert started,'未觀測到正式狀態死亡owner'
            toggles=[]
            if toggle:
                key('F2');faithful=wait(lambda s:not s.get('hd_theme'))
                assert faithful.get('unit_death_scene') and faithful.get('unit_death_kind')=='status'
                toggles.append(dict(theme='faithful',state=faithful))
                key('F2');hd=wait(lambda s:s.get('hd_theme'))
                assert hd.get('unit_death_scene') and hd.get('unit_death_kind')=='status'
                toggles.append(dict(theme='hd',state=hd))
            returned=idle()
            stop.set();worker.join(timeout=20)
            assert not worker.is_alive() and not errors,errors
            assert actor(returned)['hp']==0 and returned.get('hd_world')
            assert returned['turn']>before['turn'] and returned['gold']==before['gold']
            assert [u['exp'] for u in returned['units']]==[u['exp'] for u in before['units']]
            assert all(u['hp']>0 for u in returned['units'] if u['camp']==0 and u.get('native_identity')!=4)
            assert not any(returned.get(k) for k in ('death_reward_ui','death_reward_message'))
            shot(f'status-death-{label}-returned')
            cases.append(dict(case=label,theme_toggle=toggle,before=before,movement=movement,retreat=retreat,
                              started=started,returned=returned,f2=toggles))
            checkpoint()
        key('F5');wait(lambda s:s.get('frame',0)>cases[-1]['returned']['frame']+2)
        saved_bytes=saved.read_bytes()
        key('F2');wait(lambda s:not s.get('hd_theme'))
        key('F9');wait(lambda s:s.get('node')==args.battle_node);idle()
        key('F5');assert saved.read_bytes()==saved_bytes,'主題切換改變章界存檔'
        key('F2');wait(lambda s:s.get('hd_theme'))
        key('F9');wait(lambda s:s.get('node')==args.battle_node)
        warm=idle()
        restart(args.battle_node,warm['gold'],'status-death-cold-title-load')
        cold=idle()
        assert cold['units']==warm['units'] and saved.read_bytes()==saved_bytes
        assert digest(args.town_save)==source_hash
        checkpoint()
        report=dict(schema='fd2-status-death-normal-gui-v1',status='PASS',death_kind='status',
                    source_file=str(args.town_save),source_sha256=source_hash,source_manifest=manifest,
                    binary_sha256=digest(args.binary),gui_tool_sha256=tool_hash,
                    helper_sha256=digest(Path(__file__)),catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),
                    cases=cases,samples=samples,rejected=rejected,operations=operations,screenshots=shots,
                    saved_sha256=digest(saved),warm=warm,cold=cold,final_node=cold['node'],
                    in_chapter_injection=False,hp_lock=False,recording_created=False,
                    scope='明示章前低HP／中毒來源；正常LOAD、END／YES、狀態死亡及F2、返回，章界保存冷讀。非正常取得中毒、章內存檔、自然難度、整章E2或新DOS對拍。')
        (args.out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    finally:
        stop.set()
        if worker:worker.join(timeout=20)
        checkpoint()


def run_ch23_death_event(args,audit,key,wait,shot,saved,command,window,restart,operations,shots,tool_hash):
    assert args.death_event==53 and args.battle_node=='battle_ch23' and args.town_save and args.world
    source_hash=digest(args.town_save)
    progress=[]; samples=[]; commander_samples=[]; portrait_samples=[]; commander_seen=set()
    commander_hd=args.death_commander_hd
    actor_art=getattr(args,'death_actor_art',False)
    actor_samples=[]; actor_seen=set()
    command('xdotool','windowsize',window,'1920','1200')
    def checkpoint():
        (args.out/'death-observation.json').write_text(json.dumps(dict(
            progress=progress,samples=samples,commander_samples=commander_samples,
            portrait_samples=portrait_samples,actor_art=actor_art,actor_art_samples=actor_samples,
            state=latest(audit),state_injection=False),ensure_ascii=False,indent=2)+'\n')
    def sample_actor(s):
        layers=[l for l in s.get('battle_source_layers') or []
                if l.get('archive','') in ('','FIGANI.DAT') and l['resource'] in (90,91) and l['fill']==-1]
        signature=(s.get('battle_owner'),s.get('battle_phase'),
                   tuple((l['resource'],l['frame'],l['x'],l['y']) for l in layers))
        if (not actor_art or not s.get('battle_scene') or not s.get('hd_battle')
                or not layers or signature in actor_seen or len(actor_samples)>=48):return
        actor_seen.add(signature)
        path=args.out/f'gaia30-action-{len(actor_samples):02d}.png'
        command('import','-window',window,str(path))
        actor_samples.append(dict(scene='gaia-commander',file=path.name,sha256=digest(path),before=s,after=latest(audit)))
        checkpoint()
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
        sample_actor(s);sample_commander(s);sample_portrait(s)
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
    if actor_art:
        assert len(actor_samples)>=3,'缺少普通蓋亞攻擊觀測樣本'
    checkpoint()
    (args.out/'receipt.json').write_text(json.dumps(dict(
        schema='fd2-shared-death-normal-gui-v1',status='PASS',binary_sha256=digest(args.binary),
        gui_tool_sha256=tool_hash,helper_sha256=digest(Path(__file__)),source_sha256=source_hash,
        source_file=str(args.town_save),saved_sha256=faithful_hash,samples=samples,progress=progress,
        operations=operations,screenshots=shots,event_id=53,death_kind=args.death_kind,state_injection=False,recording_created=False,
        commander_hd=commander_hd,commander_samples=commander_samples,portrait_samples=portrait_samples,
        actor_art=actor_art,actor_figure=30,actor_art_samples=actor_samples,
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


if __name__=='__main__':
    import argparse
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--build-status-source',action='store_true',required=True)
    parser.add_argument('--source',type=Path,required=True)
    parser.add_argument('--out',type=Path,required=True)
    args=parser.parse_args()
    result,manifest=status_source(args.source)
    assert not args.out.exists() and args.out.parent.is_dir()
    args.out.mkdir()
    output=args.out/'fd2_save.json'
    output.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    manifest['output_sha256']=digest(output)
    (args.out/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
    print(manifest['output_sha256'])
