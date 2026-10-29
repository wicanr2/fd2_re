"""普通START的古金對話框四組切換。由verify_fd2_hd_gui.py驅動，不注入狀態。"""
import json
import time
from pathlib import Path
from verify_fd2_hd_gui import digest,latest

def acknowledged_key(audit,command,name,operations):
    before=latest(audit);mapped={'Return':'Enter','Down':'ArrowDown'}.get(name,name)
    deadline=time.monotonic()+120;ack=None
    command('xdotool','keydown',name)
    try:
        while time.monotonic()<deadline:
            if audit.exists():
                with audit.open('rb') as stream:
                    stream.seek(max(0,audit.stat().st_size-262144));rows=stream.read().splitlines()
                for raw in reversed(rows):
                    try:s=json.loads(raw)
                    except ValueError:continue
                    if s.get('frame',0)>before.get('frame',0) and mapped in (s.get('keys') or []):ack=s;break
            if ack is not None:break
            if latest(audit).get('error'):raise RuntimeError(latest(audit)['error'])
            time.sleep(.05)
    finally:command('xdotool','keyup',name)
    if ack is None:raise TimeoutError(f'{mapped}沒有正常Update按鍵收據')
    # ACK may already be several frames old when the poll observes it. Waiting
    # relative to ACK can return before this key-up reaches Update, merging two
    # consecutive presses into one held key. Require the actual release event.
    released=None
    while time.monotonic()<deadline:
        with audit.open('rb') as stream:
            stream.seek(max(0,audit.stat().st_size-262144));rows=stream.read().splitlines()
        for raw in reversed(rows):
            try:s=json.loads(raw)
            except ValueError:continue
            if s.get('frame',0)>ack['frame'] and mapped in (s.get('released_keys') or []):released=s;break
        if released is not None:break
        time.sleep(.05)
    if released is None:raise TimeoutError(f'{mapped}沒有正常Update放開收據')
    with operations.open('a') as stream:stream.write(json.dumps(dict(key=name,ack=ack,released=released),ensure_ascii=False)+'\n')
    return ack

def run_gold_dialogue(args,audit,key,shot,command,window,started):
    def wait(predicate,limit=120):
        end=time.monotonic()+limit
        while time.monotonic()<end:
            s=latest(audit)
            if s.get('error'):raise RuntimeError(s['error'])
            if predicate(s):return s
            time.sleep(.1)
        raise TimeoutError(str(latest(audit)))
    previous,stable_at=None,time.monotonic()
    while True:
        current=wait(lambda s:s.get('dialogue_speaker')==48 and s.get('dialogue_phase')==0)
        if previous is None or previous.get('dialogue_progress')!=current.get('dialogue_progress'):stable_at=time.monotonic()
        if time.monotonic()-stable_at>2:break
        previous=current;time.sleep(.2)
    initial=latest(audit)
    def state(s):return {k:s.get(k) for k in ('node','units','dialogues','dialogue_speaker','dialogue_native','dialogue_progress','dialogue_phase')}
    rows=[]
    for label,keys,art,ui in (
        ('hd-hd',[],True,'hd'),('original-hd',['F2'],False,'hd'),
        ('original-original',['F11'],False,'original'),('hd-original',['F2'],True,'original'),
        ('hd-hd-restored',['F11'],True,'hd')):
        for name in keys:key(name)
        s=wait(lambda s:s.get('hd_theme') is art and s.get('interface_mode')==ui)
        command('xdotool','windowsize',window,'1920','1200')
        # Updates acknowledge controls; wait for the actual final drawing too.
        wait(lambda s:bool(s.get('hd_gold_dialogue_rects'))==(ui=='hd'))
        time.sleep(1)
        assert state(latest(audit))==state(initial)
        path=shot('gold-'+label+'.png')
        rows.append(dict(label=label,art=art,ui=ui,state=latest(audit),capture=path.name,sha256=digest(path)))
    key('Return')
    advanced=wait(lambda s:s.get('dialogue_speaker')!=48 or s.get('dialogues')!=initial.get('dialogues') or s.get('dialogue_progress')!=initial.get('dialogue_progress'))
    report=dict(status='PASS-GOLD-DIALOGUE-NORMAL-START',normal_player_verified=True,state_injection=False,
                scope='普通預設START至父王，四組美術／介面與正常Enter續句；像素另行獨立核對，不宣稱整章或原版對拍。',
                binary_sha256=digest(args.binary),catalog_sha256=digest(args.assets/'assets/themes/modern/catalog.json'),
                driver_sha256=digest(Path(__file__)),initial=initial,rows=rows,advanced=advanced,
                elapsed_seconds=time.monotonic()-started,recording_created=False)
    (args.out/'gold-gui.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')

def start_campaign(audit,key):
    deadline=time.monotonic()+180
    while latest(audit).get('title_phase')!='menu':
        if time.monotonic()>deadline:raise TimeoutError('開場未抵達標題選單')
        key('Escape')
    assert latest(audit).get('title_selection')==0
    # Confirm starts the native menu flash. Repeated confirmations reset it;
    # wait for the actual transition instead of repeatedly sending Enter.
    key('Return')
    while latest(audit).get('title_phase'):
        if latest(audit).get('error'):raise RuntimeError(latest(audit)['error'])
        if time.monotonic()>deadline:raise TimeoutError('START閃爍後沒有進入戰役')
        time.sleep(.1)
