#!/usr/bin/env python3
"""核對正常死亡事件PNG的可見不透明效果核心；不取代完整fixture或原版oracle。"""
import argparse
import json
from pathlib import Path

from PIL import Image, ImageChops
from verify_fd2_hd_death_gpu import checked, clipped, digest


def validate_status_contract(receipt):
    """狀態消費端的窄契約；不得據此放行章23事件或正常取得中毒。"""
    def require(condition, message):
        if not condition:
            raise ValueError(message)
    require(receipt.get('schema')=='fd2-status-death-normal-gui-v1' and receipt.get('status')=='PASS', 'status schema')
    require(receipt.get('death_kind')=='status' and receipt.get('final_node')=='battle_ch01', 'status route')
    require(receipt.get('in_chapter_injection') is False and receipt.get('hp_lock') is False, 'chapter injection')
    manifest=receipt['source_manifest']
    require(manifest['schema']=='fd2-hd-status-death-source-v1' and manifest['actor_identity']==4,
            'source policy')
    require(manifest['output_sha256']==receipt['source_sha256'], 'source hash')
    require(manifest['in_chapter_injection'] is False and manifest['hp_lock'] is False, 'source injection')
    cases=receipt['cases']
    require(len(cases)==2 and [c['case'] for c in cases]==['full','f2'], 'two replays required')
    for case in cases:
        before,started,after=case['before'],case['started'],case['returned']
        require(started.get('unit_death_scene') and started.get('unit_death_kind')=='status', 'status owner')
        require(not any(after.get(k) for k in ('unit_death_scene','battle_scene','command_scene','ai_busy',
                                             'death_reward_ui','death_reward_message')), 'continuation')
        require(after.get('hd_world') and after['turn']>before['turn'] and after['gold']==before['gold'], 'return or reward')
        require(len(before['units'])==len(after['units']), 'roster changed')
        actors=[(a,b) for a,b in zip(before['units'],after['units']) if a['camp']==0 and a.get('native_identity')==4]
        require(len(actors)==1 and actors[0][0]['hp']>0 and actors[0][1]['hp']==0, 'status HP transition')
        retreat=case['retreat']
        require(retreat in case['movement']['move_reachable'] and
                [actors[0][0]['x'],actors[0][0]['y']]==retreat and actors[0][0]['acted'], 'normal retreat')
        require(all(a['exp']==b['exp'] for a,b in zip(before['units'],after['units'])), 'status awarded EXP')
        require(all(u['hp']>0 for u in after['units'] if u['camp']==0 and u.get('native_identity')!=4), 'other player died')
        toggles=case['f2']
        require(case['theme_toggle']==(case['case']=='f2'), 'toggle case')
        require(len(toggles)==(2 if case['theme_toggle'] else 0), 'toggle count')
        for toggle,theme in zip(toggles,('faithful','hd')):
            state=toggle['state']
            require(toggle['theme']==theme and bool(state.get('hd_theme'))==(theme=='hd') and
                    state.get('unit_death_scene') and state.get('unit_death_kind')=='status' and
                    state.get('unit_death_owner')==started.get('unit_death_owner'), 'active F2')
    require(receipt['warm']['units']==receipt['cold']['units'] and
            receipt['warm']['gold']==receipt['cold']['gold'], 'cold load')
    keys=[o['key'] for o in receipt['operations']]
    require(all(k in keys for k in ('Down','Return','F2','F5','F9')) and
            any(o.get('process_epoch')==1 for o in receipt['operations']), 'normal keyboard and cold process')


def validate_status_source(receipt):
    validate_status_contract(receipt)
    manifest=receipt['source_manifest']
    base,source=Path(manifest['base_file']),Path(receipt['source_file'])
    if (digest(base)!='089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2' or
            digest(base)!=manifest['base_sha256'] or digest(source)!=receipt['source_sha256']):
        raise ValueError('source files changed')
    original=json.loads(base.read_text());expected=json.loads(base.read_text())
    member=expected['party_roster']['4']
    member['HP']=member['MaxHP']//10
    member['native_transient']=[0,0,0,2,0,0]
    if json.loads(source.read_text())!=expected:
        raise ValueError('undisclosed source change')
    changes=[dict(path='party_roster.4.'+field,before=original['party_roster']['4'][field],after=member[field])
             for field in ('HP','native_transient')]
    if manifest['changes']!=changes:
        raise ValueError('source disclosure changed')


def status_sample_states(sample, history):
    states=[sample['before'],sample['after']]
    before,after=states
    # X11 may still show the preceding presentation when Draw's audit has
    # advanced. Admit exactly one earlier frame, only if actually observed
    # in this owner's audit with the same source placement and HD viewport.
    if (before.get('unit_death_owner')!=after.get('unit_death_owner') or
            before.get('unit_death_targets')!=after.get('unit_death_targets')):
        raise ValueError('status screenshot owner or targets changed')
    previous=[s for s in history if s.get('frame',0)<before['frame'] and
              s.get('unit_death_owner')==before.get('unit_death_owner') and
              s.get('unit_death_kind')=='status' and s.get('unit_death_presented_hd') and
              s.get('unit_death_presented_frame')==before.get('unit_death_presented_frame',-1)-1 and
              s.get('unit_death_targets')==before.get('unit_death_targets') and
              s.get('unit_death_viewport')==before.get('unit_death_viewport')]
    if previous:states.append(max(previous,key=lambda s:s['frame']))
    return states


def validate(catalog_path,pack,run):
    catalog=json.loads(catalog_path.read_text())
    bank=next(a for a in catalog['assets'] if a['role']=='map_overlay_hd_set')
    if ([d['index'] for d in bank['native_lmi_frames']]!=list(range(68,80)) or
            len(bank['files'])!=12 or len(bank['frame_sha256'])!=12):raise ValueError('incomplete source bank')
    effects={d['index']:checked(pack,dict(file=f,sha256=h)) for d,f,h in
             zip(bank['native_lmi_frames'],bank['files'],bank['frame_sha256'])}
    if any(effects[d['index']].size!=(d['width']*6,d['height']*6) for d in bank['native_lmi_frames']):
        raise ValueError('source geometry mismatch')
    receipt=json.loads((run/'receipt.json').read_text())
    status=receipt.get('schema')=='fd2-status-death-normal-gui-v1'
    if status:
        validate_status_source(receipt)
        if receipt['catalog_sha256']!=digest(catalog_path) or digest(run/'user-data/fd2_re/fd2_save.json')!=receipt['saved_sha256']:
            raise ValueError('catalog or saved file changed')
        audit_path=run/'input-audit-first-process.jsonl'
        history=[s for line in audit_path.read_text().splitlines() if (s:=json.loads(line)).get('unit_death_kind')=='status']
    elif (receipt['schema']!='fd2-shared-death-normal-gui-v1' or receipt['status']!='PASS' or
            receipt['event_id']!=53 or receipt['state_injection'] or receipt['final_node']!='preparation_ch24' or
            not {'normal-attack-commander','normal-victory-confirm'} <= {p['action'] for p in receipt['progress']}):
        raise ValueError('normal event receipt rejected')
    accepted=[];rejected=[]
    for sample in receipt['samples']:
        # F2 replay checks switching; complete-HD replay alone supplies A/B proof.
        if status and sample.get('case')!='full':continue
        actual=checked(run,sample).convert('RGB')
        if actual.size!=(1920,1200):raise ValueError('native HD window required')
        candidates=[]
        for state in status_sample_states(sample,history) if status else (sample['before'],sample['after']):
            # Earlier versioned receipts predate explicit action/script metadata.
            if state.get('unit_death_kind','script')!=receipt.get('death_kind','script'):continue
            index=state.get('unit_death_presented_frame',-1)
            if not state.get('unit_death_presented_hd') or not 13<=index<21:continue
            phase='overlay_a' if index<19 else 'overlay_b'
            entries=list(range(68,68+index-12)) if index<19 else [74+index-19]
            viewport=(0,0,1872,1152)
            if tuple(state.get('unit_death_viewport',[]))!=viewport:raise ValueError('viewport mismatch')
            expected=[]
            for entry in entries:
                for target in state['unit_death_targets']:
                    origin=target['Origin'];x=(origin%456-0x8088%456)*6;y=(origin//456-0x8088//456)*6
                    picture=effects[entry];expected.append(dict(entry=entry,rect=[x,y,x+picture.width,y+picture.height]))
            if state['unit_death_layers']!=expected:raise ValueError('source order or origin mismatch')
            world=Image.new('RGBA',(1920,1200),(0,0,0,255));core=Image.new('L',world.size)
            for layer in expected:
                picture=effects[layer['entry']];x,y,x1,y1=layer['rect']
                clipped(world,picture,(x,y),viewport,go_rgba=True)
                opaque=picture.getchannel('A').point([255 if a==255 else 0 for a in range(256)])
                mask=Image.new('L',world.size)
                box=(max(x,0),max(y,0),min(x1,viewport[2]),min(y1,viewport[3]))
                if box[0]<box[2] and box[1]<box[3]:
                    mask.paste(opaque.crop((box[0]-x,box[1]-y,box[2]-x,box[3]-y)),box[:2])
                core=ImageChops.lighter(core,mask)
            # Native 1920x1200, integer positions, no scaling. Check every
            # opaque source pixel, including isolated alpha255 samples.
            count=core.histogram()[255]
            difference=ImageChops.difference(actual,world.convert('RGB'))
            maximum=ImageChops.lighter(ImageChops.lighter(difference.getchannel('R'),difference.getchannel('G')),difference.getchannel('B'))
            errors=ImageChops.multiply(core,maximum.point([255 if x>3 else 0 for x in range(256)])).histogram()[255]
            candidates.append(dict(frame=index,phase=phase,compared_pixels=count,rgb_errors_gt3=errors,
                                   witness_audit_frame=state.get('frame'),
                                   previous_presented_witness=state is not sample['before'] and state is not sample['after']))
        passing=[c for c in candidates if c['compared_pixels']>=100 and c['rgb_errors_gt3']==0]
        (accepted if passing else rejected).append(dict(file=sample['file'],candidates=passing or candidates))
    phases={c['phase'] for sample in accepted for c in sample['candidates']}
    if phases!={'overlay_a','overlay_b'}:
        raise ValueError('missing independently matched phase '+str(phases)+' '+json.dumps(rejected))
    return dict(status='PASS-NORMAL-DEATH-OPAQUE-CORES',accepted=accepted,rejected=rejected,
                receipt_sha256=digest(run/'receipt.json'),catalog_sha256=digest(catalog_path),
                audit_sha256=digest(audit_path) if status else None,
                source_kind='constructed-status-consumer' if status else receipt.get('death_kind','script'),
                scope='只核正常X11抽樣的不透明來源核心、順序、原點與裁切；狀態來源另外揭露章前注入，不證明正常取得狀態。透明區、完整26拍及DOS parity另驗。')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ('catalog','pack','run','out'):parser.add_argument('--'+name,type=Path,required=True)
    args=parser.parse_args();result=validate(args.catalog,args.pack,args.run)
    args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(result['status'],len(result['accepted']))
