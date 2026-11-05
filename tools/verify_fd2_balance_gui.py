#!/usr/bin/env python3
import argparse,hashlib,json,os,pathlib,shutil,subprocess,time

parser=argparse.ArgumentParser(description='Docker 內以普通鍵鼠驗證能力工具列與獨立進度保存。')
parser.add_argument('--binary',type=pathlib.Path,required=True)
parser.add_argument('--out',type=pathlib.Path,required=True)
parser.add_argument('--resume-save',type=pathlib.Path)
parser.add_argument('--source-save',type=pathlib.Path)
parser.add_argument('--battle-node',default='battle_ch01')
parser.add_argument('--skill-hit',action='store_true')
parser.add_argument('--skip-hd',action='store_true')
args=parser.parse_args()
assert pathlib.Path('/.dockerenv').exists() and args.binary.is_file()
assert not (args.resume_save and args.source_save)
source_save=args.source_save or args.resume_save
resume=source_save is not None
base=args.out;base.mkdir(parents=True,exist_ok=False)
runtime=base/'runtime/AppDir';runtime.mkdir(parents=True)
for name,target in [('assets','/repo/remake/assets'),('pack','/original'),('modern','/modern')]:
    (runtime/name).symlink_to(target,target_is_directory=True)
app=runtime/'AppRun'
app.write_text('#!/bin/sh\nset -eu\ncd "'+str(runtime)+'"\nexport FD2_ASSET_PACK=/original FD2_MODERN_THEME_PACK=/modern\nexec "'+str(args.binary)+'" "$@"\n')
app.chmod(0o755)
env=dict(os.environ,DISPLAY=':95',FD2_CAPTURE_LABEL=('R82 skill UI: controlled JOIN + level-up-writer constructed pre-chapter save; ordinary title LOAD, dialogue keys, mouse, F5; not natural recruitment' if args.skill_hit else 'R82 balance GUI; ordinary START, keys, mouse, F5, cold LOAD; current HP edit restored'))
events=[];owner=None;out=None;start=time.monotonic()
def latest():
    p=out/'input-audit.jsonl'
    if not p.exists():return {}
    with p.open('rb') as f:f.seek(max(0,p.stat().st_size-131072));rows=f.read().splitlines()
    for row in reversed(rows):
        try:return json.loads(row)
        except ValueError:pass
    return {}
def command(*args):return subprocess.check_output(args,env=env,stderr=subprocess.DEVNULL,text=True)
def wait(test,timeout=40):
    deadline=time.monotonic()+timeout
    while time.monotonic()<deadline:
        if owner.poll() is not None:raise RuntimeError('owner exited')
        s=latest()
        if s.get('error'):raise RuntimeError(s['error'])
        if test(s):return s
        time.sleep(.2)
    raise TimeoutError(json.dumps({k:latest().get(k) for k in ['frame','node','title_phase','dialogues','ai_busy','error']}))
def event(kind,detail):events.append({'kind':kind,'detail':detail,'elapsed':round(time.monotonic()-start,3),'audit':latest()})
def key(k,delay=.35):
    command('xdotool','windowfocus',window);command('xdotool','keydown',k);time.sleep(.13);command('xdotool','keyup',k);time.sleep(delay);event('key',k)
def click(x,y):
    command('xdotool','windowfocus',window);command('xdotool','mousemove','--window',window,str(x*3),str(y*3));time.sleep(.15)
    command('xdotool','mousedown','1')
    try:wait(lambda s:s.get('mouse_left_pressed') is True,20)
    finally:command('xdotool','mouseup','1')
    wait(lambda s:s.get('mouse_left_released') is True,20)
    time.sleep(.7)
    event('click',[x,y])
def shot(name):
    p=out/(name+'.png');command('import','-window',window,str(p));event('shot',{'file':p.name,'sha256':hashlib.sha256(p.read_bytes()).hexdigest()})
    print('SHOT',out.name,name,flush=True)
def open_editor():
    command('xdotool','mousemove','--window',window,'1794','3');time.sleep(.5);click(598,12)
def ack():
    for count in range(1,4):
        click(400,264);wait(lambda s:s.get('balance_editor',{}).get('ack')==count)
def choose_person(identity):
    state=latest().get('balance_editor',{})
    assert identity in state.get('ids',[]),identity
    for _ in range(len(state['ids'])):
        if latest().get('balance_editor',{}).get('identity')==identity:return
        click(416,120)
    raise AssertionError('editor did not select identity '+str(identity))
def number(x,y,n):
    click(x,y);command('xdotool','type','--clearmodifiers','--delay','100',str(n));time.sleep(.6);event('number',n)
def run_capture(name):
    global owner,out,window
    out=base/name;out.mkdir(exist_ok=True)
    log=(out/'owner.log').open('w')
    owner=subprocess.Popen(['bash','/repo/tools/capture_fd2_release_promo.sh',str(runtime),str(out),'880','1920x1200','30'],env=env,stdout=log,stderr=log)
    wait(lambda s:bool(s.get('title_phase')),90)
    window=command('xdotool','search','--name','炎龍騎士團').splitlines()[0]
    for _ in range(160):
        if latest().get('title_phase')=='menu':break
        key('Escape')
    assert latest().get('title_phase')=='menu'
def stop_capture():
    if owner and owner.poll() is None:
        (out/'STOP').touch()
        try:owner.wait(timeout=50)
        except subprocess.TimeoutExpired:owner.kill();owner.wait(timeout=3)
def save_path():return out/'user-data/fd2_re/fd2_save.json'
def ready_battle():
    deadline=time.monotonic()+300
    while time.monotonic()<deadline:
        s=latest()
        if s.get('node')==args.battle_node and not any(s.get(k) for k in ['dialogues','ai_busy','walking','battle_event','battle_scene','command_scene','transient_ui','banner_t']):
            # LOAD publishes the node before its chapter-introduction stepper.
            # Require another complete audit interval without a new dialogue.
            before=s['frame'];time.sleep(2.5);after=latest()
            if after.get('frame',0)>before and not any(after.get(k) for k in ['dialogues','ai_busy','walking','battle_event','battle_scene','command_scene','transient_ui','banner_t']):return
        key('Return',.15)
    raise TimeoutError(args.battle_node+' introduction not ready')
result={'pass':False,'binary':str(args.binary),'binary_sha256':hashlib.sha256(args.binary.read_bytes()).hexdigest(),
        'controller_sha256':hashlib.sha256(pathlib.Path(__file__).read_bytes()).hexdigest(),
        'resume_save_sha256':hashlib.sha256(source_save.read_bytes()).hexdigest() if resume else None,
        'battle_node':args.battle_node,'source_policy':('controlled JOIN + level-up writer constructed pre-chapter slot; not natural recruitment' if args.skill_hit else 'ordinary F5 save')}
try:
    if args.skill_hit:
        assert source_save
        original_source=source_save.read_bytes();source=json.loads(original_source)
        assert source['node']==args.battle_node
        dst=base/'skill/user-data/fd2_re';dst.mkdir(parents=True,exist_ok=False);shutil.copy2(source_save,dst/'fd2_save.json')
        run_capture('skill');key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots');key('Return')
        wait(lambda s:s.get('node')==args.battle_node and not s.get('title_phase'));ready_battle();shot('battle-ready')
        party_ids=sorted(int(k) for k in source['party_roster'])
        assert 9 in party_ids and 18 in party_ids
        open_editor();shot('warning-zero');ack();click(384,88)
        choose_person(9)
        shot('yuni-learned-skill')
        # Command zero is already learned in the controlled source, so the
        # first displayed command needs no skill grant or command injection.
        assert source['party_roster']['9']['native_command_mask'][0]&1
        number(480,193,100);click(505,334);shot('yuni-hit-100');click(548,55);key('F5')
        wait(lambda s:json.loads(save_path().read_text()).get('balance',{}).get('skill_hit',{}).get('9',{}).get('0')==100)
        shutil.copy2(save_path(),base/'saved-yuni-hit100.json');result['yuni_skill_0_hit100_saved']=True
        open_editor();ack();click(384,88)
        choose_person(9)
        click(138,334);shot('yuni-reset-original');click(548,55);key('F5')
        wait(lambda s:not json.loads(save_path().read_text()).get('balance',{}).get('skill_hit',{}).get('9'))
        result['yuni_override_removed']=True
        open_editor();ack();click(384,88)
        choose_person(18)
        shot('middy-sonic-no-hit-roll');click(505,334);shot('middy-apply-preserves-rules');click(548,55);key('F5')
        wait(lambda s:not json.loads(save_path().read_text()).get('balance',{}).get('skill_hit',{}).get('18'))
        shutil.copy2(save_path(),base/'saved-middy-unchanged.json');result['middy_no_hit_override']=True
        assert source_save.read_bytes()==original_source
        result['source_unchanged']=True;result['pass']=True
        raise SystemExit(0)
    if resume:
        old=source_save
        dst=base/'cold/user-data/fd2_re';dst.mkdir(parents=True,exist_ok=False);shutil.copy2(old,dst/'fd2_save.json')
        run_capture('cold');key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots');key('Return')
        wait(lambda s:s.get('node')==args.battle_node and not s.get('title_phase'));ready_battle()
        shot('cold-battle');open_editor();shot('cold-warning-0');ack();shot('cold-current')
        original_hp=json.loads(save_path().read_text())['party_roster']['0']['MaxHP']
        number(480,155,original_hp+1);click(505,334);shot('current-hp-plus-one');click(548,55);key('F5')
        wait(lambda s:json.loads(save_path().read_text())['party_roster']['0']['MaxHP']==original_hp+1)
        shutil.copy2(save_path(),base/'saved-plus-one.json');result['current_hp_edit_saved']=original_hp+1
        open_editor();ack();number(480,155,original_hp);click(505,334);click(548,55);key('F5')
        wait(lambda s:json.loads(save_path().read_text())['party_roster']['0']['MaxHP']==original_hp)
        result['current_hp_restored']=original_hp
        open_editor();ack();click(254,88);shot('cold-growth-110');click(514,88);shot('cold-enemy-120')
        click(138,334);shot('enemy-reset-100');click(254,88);click(138,334);shot('growth-reset-100')
        click(548,55);key('F5');wait(lambda s:'balance' not in json.loads(save_path().read_text()))
        result['reset_balance_omitted']=True
        if not args.skip_hd:
            key('F2');wait(lambda s:s.get('hd_theme') is True,360);key('F11');wait(lambda s:s.get('hd_interface') is True,60)
            before=latest()['frame'];wait(lambda s:s.get('frame',0)>=before+120,120)
            open_editor();ack();shot('hd-current');click(254,88);shot('hd-growth');click(514,88);shot('hd-enemy');click(548,55)
        result['pass']=True
        raise SystemExit(0)
    run_capture('start')
    key('Return');wait(lambda s:s.get('title_phase')=='')
    ready_battle()
    shot('battle-original');open_editor();shot('warning-0')
    click(400,264);shot('warning-1');click(400,264);shot('warning-2');key('Escape');open_editor();shot('warning-reopened-0')
    ack();shot('current-abilities');click(254,88);shot('growth-default')
    for i in range(5):number(480,155+i*25,110)
    click(505,334);shot('growth-110');click(384,88);shot('skill-hit');click(514,88);number(480,193,120);click(505,334);shot('enemy-120')
    click(548,55);key('F5');wait(lambda s:save_path().exists())
    saved=json.loads(save_path().read_text());assert saved['balance']['enemy_percent']==120,saved.get('balance')
    result['saved_balance']=saved['balance'];result['before_cold_units']=latest().get('units')
    first_save=save_path();first_settings=first_save.parent/'display-settings.json'
    second=base/'cold/user-data/fd2_re';second.mkdir(parents=True,exist_ok=False)
    shutil.copy2(first_save,second/'fd2_save.json')
    stop_capture();run_capture('cold')
    key('Down');key('Return');wait(lambda s:s.get('title_phase')=='loadslots');key('Return')
    wait(lambda s:s.get('node')==args.battle_node and not s.get('title_phase'))
    ready_battle()
    result['after_cold_units']=latest().get('units');shot('cold-battle');open_editor();shot('cold-warning-0');ack()
    click(254,88);shot('cold-growth-110');click(514,88);shot('cold-enemy-120')
    click(138,334);shot('enemy-reset-100');click(254,88);click(138,334);shot('growth-reset-100')
    click(548,55);key('F5');wait(lambda s:save_path().exists())
    restored=json.loads(save_path().read_text());assert 'balance' not in restored,restored.get('balance')
    result['reset_balance_omitted']=True
    key('F2');wait(lambda s:s.get('hd_theme') is True,360);key('F11');wait(lambda s:s.get('hd_interface') is True,60)
    before=latest()['frame'];wait(lambda s:s.get('frame',0)>=before+120,120)
    open_editor();ack();shot('hd-current');click(254,88);shot('hd-growth');click(514,88);shot('hd-enemy');click(548,55)
    result['pass']=True
except Exception as error:
    result['error']=type(error).__name__+': '+str(error);print('ERROR',result['error'],flush=True)
finally:
    stop_capture();result['elapsed']=time.monotonic()-start;result['last_state']=latest()
    (base/'operations.json').write_text(json.dumps(events,ensure_ascii=False,indent=2))
    (base/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2))
    print('RESULT',{k:v for k,v in result.items() if k not in ['last_state','before_cold_units','after_cold_units']},flush=True)
