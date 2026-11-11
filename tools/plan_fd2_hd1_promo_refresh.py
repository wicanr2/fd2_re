#!/usr/bin/env python3
"""核對新錄影後編製HD1推廣片計畫；未改動的鏡頭明示沿用。"""
import argparse
import bisect
import hashlib
import json
import os
from pathlib import Path
import re
import struct


def sha(path):
    h=hashlib.sha256()
    with Path(path).open('rb') as stream:
        for data in iter(lambda:stream.read(1024*1024),b''):h.update(data)
    return h.hexdigest()


def load(path):
    return json.loads(Path(path).read_text())


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ['previous','captures','opening','cards','binary','audio-reference','out']:
        parser.add_argument('--'+name,type=Path,required=True)
    parser.add_argument('--version',required=True)
    parser.add_argument('--engine-commit',required=True)
    parser.add_argument('--opening-start',type=float,required=True)
    parser.add_argument('--opening-end',type=float,help='在選單已另有鏡頭時，剪除開場結尾等待。')
    parser.add_argument('--intro-capture',type=Path,help='HD開場可由另一份已完成自然開場階段的錄影提供。')
    parser.add_argument('--battle-binary',type=Path,help='後續只修故事路徑時，明示先前戰鬥實錄的程式版本。')
    parser.add_argument('--deployment-selection',type=Path,help='人工依實際畫面選定首關站位的來源時碼與SHA。')
    args=parser.parse_args()
    assert Path('/.dockerenv').exists() and os.getuid()!=0
    assert not args.out.exists() and args.out.parent.stat().st_uid==os.getuid()
    repo=Path(__file__).resolve().parents[1]
    # Earlier manifests used /repo as the container mount; source bytes stay the same.
    plan=json.loads(args.previous.read_text().replace('/repo/',str(repo)+'/'))
    previous_sha=sha(args.previous)
    plan['version']=args.version
    assert re.fullmatch(r'[0-9a-f]{40}',args.engine_commit)
    plan['target_release_engine_commit']=args.engine_commit
    binary_sha=sha(args.binary)
    battle_binary=args.battle_binary or args.binary
    battle_sha=sha(battle_binary)
    input_paths={args.previous,args.binary,repo/'remake/assets/themes/modern/catalog.json',Path(__file__)}
    revised={};captures=[];sources={};pack_roots=None
    specifications=[('ares',4,None),('gaia',30,None),('yuni',9,0),('middy',18,30),('tino',2,29),('rodman',19,28)]
    for name,identity,command in specifications:
        candidates=[]
        for recording in args.captures.glob(name+'-record-r*/recording.json'):
            row=load(recording)
            if row.get('status')=='PASS' and row.get('verifier_exit_code')==0:
                tail=row['command'];gui=Path(tail[tail.index('--out')+1]);action=gui/'promo-action.json'
                if action.is_file() and load(action).get('binary_sha256')==battle_sha:
                    candidates.append((recording,gui,action))
        assert len(candidates)==1,(name,'exactly one current successful capture required')
        recording,gui,action_path=candidates[0];action=load(action_path)
        command_line=load(recording)['command']
        roots=(command_line[command_line.index('--original-pack')+1],command_line[command_line.index('--modern-pack')+1])
        assert pack_roots is None or pack_roots==roots
        pack_roots=roots
        assert action['status']=='PASS-NORMAL-KEYBOARD-PROMO-ACTION'
        assert action['actor_identity']==identity and action['command_id']==command
        assert action['runtime_state_injection'] is False
        observation=gui/'battle-observation.json'
        assert load(observation)['catalog_sha256']==sha(repo/'remake/assets/themes/modern/catalog.json')
        rec=recording.parent;video=rec/'video.mkv';timing=rec/'frame-times.jsonl'
        expected=next(r for r in load(recording)['files'] if r['file']==video.name)
        assert sha(video)==expected['sha256']
        rows=[json.loads(line) for line in timing.read_text().splitlines()]
        def active(row):
            s=row['state']
            return (s.get('battle_scene') and s.get('battle_actor_fig')==identity) if command is None else (s.get('command_scene') and s.get('command_id')==command)
        observations=[r for r in rows if active(r)]
        assert len(observations)>=4 and all(r['state'].get('hd_theme') for r in observations),'action contains original art'
        label='physical' if command is None else 'command'+str(command)
        window=gui/('world-hd-'+label+'-scene.png')
        width,height=struct.unpack('>II',window.read_bytes()[16:24]);assert width*5==height*8
        start=max(0,observations[0]['seconds']-.65)
        end=observations[-1]['seconds']+.15
        revised['gaia-hd' if name=='gaia' else name]=dict(source=str(video),source_sha256=sha(video),
            start=start,source_duration=end-start,minimum_playback_rate=1,
            window_rect=[0,0,width,height],capture_receipt=str(action_path),
            recording_receipt=str(recording),timing=str(timing),binary=str(battle_binary),binary_sha256=battle_sha,
            capture_scope='目前程式普通LOAD、選人、移動、出招及返回；來源具明示建構祖先。動作剪輯不代表實機幀率。')
        captures.append(dict(scene='gaia-hd' if name=='gaia' else name,identity=identity,command=command,
            observations=len(observations),recording=str(recording),recording_sha256=sha(recording),
            action_receipt=str(action_path),action_sha256=sha(action_path),timing_sha256=sha(timing)))
        audit=gui/'input-audit.jsonl'
        raw=[json.loads(line) for line in audit.read_text().splitlines()]
        sources['gaia-hd' if name=='gaia' else name]=dict(rows=rows,raw=raw,identity=identity,command=command,
            start=start,end=end,timing=timing)
        if name=='gaia':
            transitions=[rows[i]['seconds'] for i in range(1,len(rows))
                if rows[i-1]['state'].get('hd_theme') is False and rows[i]['state'].get('hd_theme') is True
                and not rows[i]['state'].get('battle_scene') and rows[i]['seconds']<start]
            assert transitions,'normal map F2 restoration missing'
            t=transitions[-1]
            revised['map-switch']=dict(source=str(video),source_sha256=sha(video),start=max(0,t-1.5),source_duration=4,
                minimum_playback_rate=1,capture_receipt=str(action_path),capture_scope='普通F2恢復HD地圖；無狀態注入。')
        input_paths.update([video,recording,action_path,timing,window,audit,observation,battle_binary])
    opening=load(args.opening/'opening-promo-gui.json')
    assert opening['status']=='PASS-NORMAL-START-ROYAL-DEPLOYMENT' and opening['state_injection'] is False
    assert opening['binary_sha256']==binary_sha and opening['driver_source_unchanged']
    if not args.intro_capture:
        assert opening['opening_skipped'] is False
        assert opening['natural_opening']['status']=='PASS-NATURAL-HD-OPENING-TO-MENU'
    events=opening['events']
    event=lambda label:next(r for r in events if r['label']==label)
    source=args.opening/'video.mkv'
    assert source.is_file() and args.opening_start>=0
    source_hash=sha(source)
    def opening_clip(start,end,**extra):
        assert 0<=start<end
        return dict(source=str(source),source_sha256=source_hash,start=start,source_duration=end-start,
            minimum_playback_rate=1,capture_receipt=str(args.opening/'opening-promo-gui.json'),
            capture_scope='目前程式普通START、逐幕HD開場、開始選單、皇城四組切換與第一關站位；無狀態注入。',**extra)
    menu=event('start-menu')['seconds']
    if args.intro_capture:
        natural=load(args.intro_capture/'natural-opening.json')
        assert natural['status']=='PASS-NATURAL-HD-OPENING-TO-MENU'
        assert natural['rows'] and all(r['hd_theme'] and r['hd_opening'] for r in natural['rows'])
        intro_events=[json.loads(line) for line in (args.intro_capture/'promo-events.jsonl').read_text().splitlines()]
        intro_menu=next(r['seconds'] for r in intro_events if r['label']=='start-menu')
        intro_end=args.opening_end if args.opening_end is not None else intro_menu-2
        assert args.opening_start<intro_end<=intro_menu
        intro_video=args.intro_capture/'video.mkv'
        revised['opening']=dict(source=str(intro_video),source_sha256=sha(intro_video),
            start=args.opening_start,source_duration=intro_end-args.opening_start,minimum_playback_rate=1,
            capture_receipt=str(args.intro_capture/'natural-opening.json'),
            binary=str(battle_binary),binary_sha256=battle_sha,
            capture_scope='獨立take完成逐幕HD開場到選單；同take後段的舊名冊缺陷不作首關通過證據。')
        input_paths.update([intro_video,args.intro_capture/'natural-opening.json',args.intro_capture/'promo-events.jsonl'])
    else:
        revised['opening']=opening_clip(args.opening_start,menu-2)
    revised['start-menu']=opening_clip(menu-.5,menu+2.5,intentional_static=True)
    revised['royal-switch']=opening_clip(event('royal-hd-hd')['seconds']-1,event('royal-hd-hd-restored')['seconds']+1,
        intentional_static=True,intentional_static_reason='同句皇城對話等待按鍵，比較四組原版／HD設定；切換間保留閱讀時間。')
    # Keep normal START arrival and the stable first-battle formation.
    deployed=event('first-battle-deployed')
    assert deployed['state']['hd_theme'] and deployed['state']['node']=='battle_ch01'
    entry=next((r['seconds'] for r in events if r['state'].get('node')=='battle_ch01'),deployed['seconds']-7)
    revised['first-deployment']=opening_clip(min(entry,deployed['seconds']-7),deployed['seconds']+3)
    if args.deployment_selection:
        selection=load(args.deployment_selection)
        assert selection['source_sha256']==source_hash and selection['reviewed'] is True
        assert 0<=selection['start']<selection['end']<=deployed['seconds']+3
        revised['first-deployment']=opening_clip(selection['start'],selection['end'],selection_evidence=str(args.deployment_selection))
        input_paths.add(args.deployment_selection)
    input_paths.update([source,args.opening/'opening-promo-gui.json',args.opening/'controller-inputs.json'])
    if (args.opening/'natural-opening.json').exists():input_paths.add(args.opening/'natural-opening.json')
    cards=load(args.cards/'character-cards.json')
    assert cards['character_count']==32 and cards['page_count']==16
    assert cards['catalog_sha256']==sha(repo/'remake/assets/themes/modern/catalog.json')
    assert all(c['portrait_style']=='HD' for p in cards['pages'] for c in p['characters'])
    for page in cards['pages']:
        path=args.cards/page['file'];assert sha(path)==page['sha256'];input_paths.add(path)
        revised[path.stem]=dict(source=str(path),source_sha256=sha(path))
    input_paths.add(args.cards/'character-cards.json')
    new_scenes=[]
    for original in plan['scenes']:
        scene=dict(original)
        if scene['id'] in revised:
            if scene['kind']=='game':
                scene={k:original[k] for k in ['id','kind','duration','caption'] if k in original}
            scene.update(revised[scene['id']]);scene['refresh']='current-runtime-capture' if scene['kind']=='game' else 'current-catalog-card'
        else:
            scene['refresh']='reused-unchanged-source'
        if scene.get('source'):input_paths.add(Path(scene['source']))
        new_scenes.append(scene)
    plan['scenes']=new_scenes
    reference=load(args.audio_reference)
    assert reference['kind']=='normal-speed-music-separate-native-sfx'
    assets=load(repo/'remake/assets/themes/modern/catalog.json')['assets']
    admitted={a['source_resource']:a for a in assets if a['role']=='battle_physical_sfx_hd'}
    cue_plan=dict(reference)
    cue_plan.update(version=args.version,source_movie_sha256='SET-AFTER-VIDEO-ENCODE',cues=[])
    offsets={};cursor=0
    for s in new_scenes:offsets[s['id']]=cursor;cursor+=s['duration']
    scenes={s['id']:s for s in new_scenes}
    def source_time(data,game_frame):
        rows=data['rows'];frames=[r['state']['frame'] for r in rows]
        at=bisect.bisect_left(frames,game_frame)
        if at==0:return rows[0]['seconds']
        if at==len(rows):return rows[-1]['seconds']
        a,b=rows[at-1],rows[at]
        f=(game_frame-a['state']['frame'])/(b['state']['frame']-a['state']['frame'])
        return a['seconds']+f*(b['seconds']-a['seconds'])
    for old_cue in reference['cues']:
        cue=dict(old_cue);name=cue['scene']
        path=cue['file'].replace('/original/',pack_roots[0]+'/').replace('/modern/',pack_roots[1]+'/').replace('/repo/',str(repo)+'/')
        cue['file']=path
        if name in sources:
            data=sources[name];native=cue.get('source_pose',cue.get('native_frame',0))
            if data['command'] is None:
                resource=data['identity']*3+1
                candidates=[r for r in data['raw'] if r.get('battle_actor_fig')==data['identity'] and r.get('hd_theme')
                    and any(l.get('resource')==resource and l.get('frame')==native for l in r.get('battle_source_layers',[]))]
            else:
                phase=('prelude' if cue['sample']==0 else 'target') if name=='yuni' else 'target-0'
                candidates=[r for r in data['raw'] if r.get('command_id')==data['command'] and r.get('hd_theme')
                    and r.get('command_phase')==phase and r.get('command_frame',-1)>=native]
            assert candidates,(name,'cue reference frame not observed',native)
            observed=candidates[0]['frame'];seconds=source_time(data,observed)
            scene=scenes[name];rate=max(1,scene['source_duration']/scene['duration'])
            relative=(seconds-scene['start'])/rate
            assert 0<=relative<scene['duration'],(name,relative)
            cue.update(at_seconds=offsets[name]+relative,observed_game_frame=observed,
                observed_recording_seconds=seconds,playback_rate=1,
                synchronization='目前正常錄影姿勢／命令拍點映射至剪輯；觀測與擷取有誤差，非硬體wall-clock。')
            # Only the physical path uses these admitted heavy impact replacements.
            if name in ('ares','gaia-hd') and cue['sample'] in admitted[cue['bank']]['cue_ids']:
                asset=admitted[cue['bank']]
                cue.update(file=str(Path(pack_roots[1])/asset['file']),sha256=asset['sha256'],
                    source_kind='hd-original-b',asset_id=asset['asset_id'])
        assert sha(cue['file'])==cue['sha256']
        cue_plan['cues'].append(cue);input_paths.add(Path(cue['file']))
    cue_plan['music']=dict(reference['music'])
    cue_plan['music']['file']=cue_plan['music']['file'].replace('/repo/',str(repo)+'/')
    assert sha(cue_plan['music']['file'])==cue_plan['music']['sha256']
    input_paths.update([args.audio_reference,Path(cue_plan['music']['file'])])
    audio_path=args.out.with_name(args.out.stem+'-audio.json');assert not audio_path.exists()
    audio_path.write_text(json.dumps(cue_plan,ensure_ascii=False,indent=2)+'\n')
    plan['inputs']=[dict(path=str(p),sha256=sha(p)) for p in sorted(input_paths)]
    plan['refresh']={'status':'CURRENT-CAPTURES-VERIFIED-EDIT-PENDING','previous_manifest_sha256':previous_sha,
        'binary_sha256':binary_sha,'battle_binary_sha256':battle_sha,'catalog_sha256':cards['catalog_sha256'],'captures':captures,
        'new_scenes':list(revised),'reused_scenes':[s['id'] for s in new_scenes if s['refresh']=='reused-unchanged-source'],
        'audio_policy':'影片只剪畫面；以原速原遊戲配樂、明示音效及已選曉雨快8%旁白另混。','audio_plan':str(audio_path),
        'scope':'剪輯調整畫面時間與必要停格，不作即時效能或Windows60fps聲明。'}
    for key in ['movie','segments','narration','gaia_repair','previous_movie','normal_speed_audio_repair']:
        plan.pop(key,None)
    assert len(plan['scenes'])==43 and sum(s['duration'] for s in plan['scenes'])==313
    args.out.write_text(json.dumps(plan,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'scenes':43,'duration':313,'fresh_scenes':len(revised),'plan':str(args.out)},ensure_ascii=False))


if __name__=='__main__':main()
