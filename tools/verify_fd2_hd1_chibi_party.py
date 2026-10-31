#!/usr/bin/env python3
"""核對固定人物矩陣的受控繪圖與普通交易相容，再登錄同一候選。"""
import argparse
from collections import Counter
from copy import deepcopy
import json
import os
from pathlib import Path
import shutil
import tempfile
from PIL import Image
from verify_fd2_hd1_chibi_refresh import ROOT,sha,read,require,tree,rect,core

RECIPE=ROOT/'remake/assets/themes/modern/hd1-party-chibi-refresh.recipe.json'
GROUPS=(1,2,3)
PROFILES={'party':GROUPS,'variants':(53,54),'identity-pair':(51,63),'base-faces':(1,4),'knights':(55,56),'heroes':(50,52),'enemies':(76,79),'static-king':(48,),'audited-six':(70,73,77,89,125,133),'party-next':(5,6,13,54),'party-four':(7,8,10,11),'party-five':(12,14,15,16),'party-six':(17,18,20,21),'party-seven':(22,23,24,25),'party-eight':(26,27,28,29),'party-nine':(58,59,60,61)}
PROFILE_IDENTITIES={'variants':{53:3,54:4},'identity-pair':{51:1,63:13},'base-faces':{1:1,4:4},'knights':{55:5,56:6},'heroes':{50:0,52:9},'party-next':{5:5,6:6,13:13,54:4},'party-nine':{58:8,59:9,60:10,61:11}}
FRAMES=(0,1,2,3,6,9)
AUDITED_PAIRS={'banded':(70,89),'draconic':(73,77),'robed':(125,133)}
GPU_PARTS={'audited-six':AUDITED_PAIRS,'party-next':{'base-knights':(5,6),'bow-and-ares':(13,54)},'party-four':{'longhair-and-bow':(7,8),'clerics':(10,11)},'party-five':{'martial-and-mage':(12,14),'beast-and-dragon':(15,16)},'party-six':{'dragon-and-swordswoman':(17,18),'greenhair-and-elder':(20,21)},'party-seven':{'stubble-and-bow':(22,23),'bald-and-beast':(24,25)},'party-eight':{'blue-and-green-dragons':(26,27),'blue-and-robed':(28,29)},'party-nine':{'bow-and-unni':(58,59),'two-clerics':(60,61)}}
AUDITED_BINDINGS={70:(5,0,'ally',13,24),73:(24,0,'ally',10,26),77:(2,2,'enemy',8,25),89:(17,0,'enemy',39,7),125:(24,0,'enemy',10,7),133:(1,1,'ally',15,12)}

def verify_parts(d):
    profile=d.get('profile');require(profile in GPU_PARTS,'unsupported split profile')
    pairs=GPU_PARTS[profile]
    parts=d.get('source_parts',[])
    require([x.get('pair') for x in parts]==list(pairs),'complete fixed GPU parts')
    require(d.get('selector_projection_injected') is (profile in ('party-next','party-nine')),'combined projection disclosure changed')
    sources=[];cases=[]
    for part in parts:
        path=ROOT/part['file']
        require(path.resolve().is_relative_to(ROOT) and sha(path)==part['sha256'],'GPU part source changed')
        item=read(path);pair=part['pair'];groups=pairs[pair]
        require(item.get('profile')==profile and item.get('pair')==pair and item.get('selected_groups')==list(groups),'GPU part pair changed')
        require(item.get('selector_projection_injected') is (profile=='party-nine' or profile=='party-next' and 54 in groups),'GPU part projection disclosure changed')
        for key in ('status','catalog_sha256','manifest_sha256','source_tree_sha256','source_file_count','native_state_unchanged','save_bytes_unchanged','authored_roster_fixture'):
            require(item.get(key)==d.get(key),'GPU part common provenance changed')
        expected={f'party-g{g:03d}-f{f:02d}-ui-hd' for g in groups for f in FRAMES}|{f'party-g{g:03d}-f00-ui-original' for g in groups}
        require(Counter(x['case'] for x in item['cases'])==Counter(expected),'incomplete GPU part')
        memory=part['memory'];mp=ROOT/memory['file']
        require(mp.resolve().is_relative_to(ROOT) and sha(mp)==memory['sha256'],'GPU part memory source changed')
        md=read(mp)
        require(md.get('returncode')==0 and md.get('timeout') is False and md['memory_events_delta'].get('oom',0)==0 and md['memory_events_delta'].get('oom_kill',0)==0,'GPU part incomplete or killed')
        sources+=item['sources'];cases+=item['cases']
    require(d['sources']==sources,'GPU part actor metadata changed')
    # File/hash fields are verified independently against real PNG cores below,
    # allowing the black-PNG self-test to test pixels, not merely provenance.
    strip=lambda rows:[{k:v for k,v in x.items() if k not in ('file','sha256')} for x in rows]
    require(strip(d['cases'])==strip(cases),'GPU part render metadata changed')

def assemble_gpu_parts(out,parts):
    require(bool(parts) and not out.exists() and not out.parent.exists() and out.parent.parent.is_dir() and out.parent.parent.stat().st_uid==os.getuid(),'fresh owned combined capture required')
    inputs=[read(p) for p in parts]
    profile=inputs[0].get('profile');require(profile in GPU_PARTS,'unsupported split profile')
    pairs=GPU_PARTS[profile];require(len(parts)==len(pairs),'complete fixed GPU parts required')
    d=deepcopy(inputs[0]);d['pair']='combined';d['selected_groups']=list(PROFILES[profile]);d['cases']=[];d['sources']=[];d['source_parts']=[];d['selector_projection_injected']=profile in ('party-next','party-nine');d['scope']='complete fixed GPU pairs and completed memory reports; controlled presentation, per-actor selector projection disclosed, not natural chapter progression'
    links=[]
    for (pair,groups),path,item in zip(pairs.items(),parts,inputs):
        require(path.parent.name.endswith('-capture'),'GPU capture naming contract')
        memory=path.parent.parent/(path.parent.name.removesuffix('-capture')+'.memory.json')
        d['source_parts'].append(dict(pair=pair,file=str(path.resolve().relative_to(ROOT)),sha256=sha(path),memory=dict(file=str(memory.resolve().relative_to(ROOT)),sha256=sha(memory))))
        d['cases']+=item['cases'];d['sources']+=item['sources']
        for row in item['cases']:
            if not row.get('file'):continue
            name=row['file'];source=path.parent/name
            require(Path(name).name==name and source.is_file() and sha(source)==row['sha256'],'GPU part PNG changed')
            links.append((name,source))
    require(len({name for name,src in links})==6*len(PROFILES[profile]),'duplicate or missing GPU PNGs')
    verify_parts(d)
    out.parent.mkdir()
    for name,source in links:(out.parent/name).symlink_to(source.resolve())
    out.write_text(json.dumps(d,ensure_ascii=False,indent=2)+'\n')

def verify_promoted_bindings(r):
    groups=PROFILES['party-nine'];identities=PROFILE_IDENTITIES['party-nine']
    bindings=r.get('party_bindings',[])
    require([b['group'] for b in bindings]==list(groups) and r.get('gpu_pairs')=={k:list(v) for k,v in GPU_PARTS['party-nine'].items()},'fixed promoted party bindings')
    source=r.get('promotion_source',{});path='remake/assets/data/class_change_targets.json'
    require(source.get('file')==path and sha(ROOT/path)==source.get('sha256'),'promoted source changed')
    rows=read(ROOT/path)['current_portraits']
    for b,chapter in zip(bindings,[3,1,6,11]):
        g=b['group'];identity=identities[g];path=f'remake/assets/scenarios/ch{chapter:02d}.json'
        require(b.get('native_identity')==b.get('portrait')==b.get('fig')==identity and b.get('target_field')=='optional_target' and b.get('selector_projection_injected') is True,'promoted identity/field/projection mismatch')
        require(b.get('scenario_file')==path and sha(ROOT/path)==b.get('scenario_sha256'),'promoted scenario changed')
        matches=[u for u in read(ROOT/path)['party'] if u.get('native_identity')==identity]
        require(len(matches)==1 and matches[0].get('fig')==matches[0].get('portrait')==identity,'promoted authored identity changed')
        targets=[row for row in rows if row['portrait']==identity]
        require(len(targets)==1 and targets[0].get('optional_target')==g,'promoted optional target changed')

def verify_gpu(d,folder,candidate,recipe=RECIPE,profile='party'):
    groups=PROFILES[profile]
    static=profile=='static-king'
    frames=(0,1,2) if static else FRAMES
    frame_count=3 if static else 12
    m=read(candidate/'manifest.json');r=read(recipe)
    require(d.get('status')=='PASS-HD1-CHIBI-PARTY-GPU','GPU status')
    require(d.get('profile','party')==profile,'GPU profile changed')
    if profile=='audited-six':
        require(d.get('selector_projection_injected') is False and d.get('authored_roster_fixture') is True and d.get('selected_groups')==list(groups) and d.get('pair')=='combined','authored roster disclosure missing')
        verify_parts(d)
    elif profile in ('party-next','party-four','party-five','party-six','party-seven','party-eight','party-nine'):
        require(d.get('selector_projection_injected') is (profile in ('party-next','party-nine')) and d.get('authored_roster_fixture') is False and d.get('selected_groups')==list(groups) and d.get('pair')=='combined','split party disclosure missing')
        verify_parts(d)
    elif profile=='base-faces':
        require(d.get('selector_projection_injected') is False and d.get('selected_groups')==list(groups),'base identity projection forbidden')
    elif static:
        require(d.get('selector_projection_injected') is False and d.get('authored_story_fixture') is True and d.get('story_private_foreground_approximation') is True,'static story disclosure missing')
    elif profile=='enemies':
        require(d.get('selector_projection_injected') is False and d.get('authored_enemy_fixture') is True,'authored enemy disclosure missing')
    elif profile!='party':require(d.get('selector_projection_injected') is True,'selector injection disclosure missing')
    require(d.get('catalog_sha256')==sha(candidate/'catalog.json') and d.get('manifest_sha256')==sha(candidate/'manifest.json'),'candidate provenance')
    require(m['recipe_sha256']==sha(recipe) and m['compiler_sha256']==sha(ROOT/'tools/refresh_fd2_hd1_map_sprites.py'),'recipe/compiler changed')
    require(sha(ROOT/r['approved_preview'])==r['approved_preview_sha256'],'approved proportion changed')
    selected=[g['group'] for g in r['groups']]
    require(bool(selected) and len(set(selected))==len(selected) and set(selected)<=set(groups),'recipe groups')
    if profile!='party':require(selected==list(groups),'fixed variant recipe required')
    if profile=='party-nine':verify_promoted_bindings(r)
    if profile=='heroes':
        bindings=[dict(group=50,native_identity=0,target_field='optional_target'),dict(group=52,native_identity=9,target_field='special_target')]
        require(r.get('selector_bindings')==bindings,'explicit special identity binding missing')
        source=r['promotion_source']
        require(source['file']=='remake/assets/data/class_change_targets.json' and sha(ROOT/source['file'])==source['sha256'],'promotion source changed')
        rows=read(ROOT/source['file'])['current_portraits']
        for b in bindings:
            matches=[row for row in rows if row['portrait']==b['native_identity']]
            require(len(matches)==1 and matches[0].get(b['target_field'])==b['group'],'authored promotion identity mismatch')
    if profile in ('base-faces','party-four','party-five','party-six','party-seven','party-eight'):
        bindings=r.get('party_bindings',[])
        require([b['group'] for b in bindings]==list(groups),'fixed base party bindings')
        chapters={'base-faces':[2,1],'party-four':[19,3,6,11],'party-five':[8,12,16,18],'party-six':[13,17,23,19],'party-seven':[24,22,22,21],'party-eight':[26,20,21,26]}[profile]
        if profile in ('party-four','party-five','party-six','party-seven','party-eight'):require(r.get('gpu_pairs')=={k:list(v) for k,v in GPU_PARTS[profile].items()},'fixed unprojected party pairs')
        for b,chapter in zip(bindings,chapters):
            g=b['group'];path=f'remake/assets/scenarios/ch{chapter:02d}.json'
            require(b['scenario_file']==path and sha(ROOT/path)==b['scenario_sha256'],'base scenario changed')
            matches=[u for u in read(ROOT/path)['party'] if u.get('native_identity')==g]
            require(len(matches)==1 and matches[0].get('fig')==matches[0].get('portrait')==g and b['native_identity']==b['portrait']==b['fig']==g,'authored base identity changed')
            if profile in ('party-four','party-five','party-six','party-seven','party-eight'):require(b.get('selector_projection_injected') is False,'base party recipe projection forbidden')
    if profile=='party-next':
        bindings=r.get('party_bindings',[])
        require([b['group'] for b in bindings]==list(groups) and r.get('gpu_pairs')=={k:list(v) for k,v in GPU_PARTS[profile].items()},'fixed next party bindings')
        promotion=r['promotion_source'];path='remake/assets/data/class_change_targets.json'
        require(promotion['file']==path and sha(ROOT/path)==promotion['sha256'],'next promotion source changed')
        targets=[row for row in read(ROOT/path)['current_portraits'] if row['portrait']==4]
        require(len(targets)==1 and targets[0].get('optional_target')==54,'explicit Ares target changed')
        for b,chapter in zip(bindings,[9,10,7,1]):
            g=b['group'];identity=PROFILE_IDENTITIES[profile][g];path=f'remake/assets/scenarios/ch{chapter:02d}.json'
            require(b['scenario_file']==path and sha(ROOT/path)==b['scenario_sha256'],'next party scenario changed')
            matches=[u for u in read(ROOT/path)['party'] if u.get('native_identity')==identity]
            require(len(matches)==1 and matches[0].get('fig')==matches[0].get('portrait')==identity and b['native_identity']==b['portrait']==b['fig']==identity and b['selector_projection_injected'] is (g==54),'next party identity mismatch')
            if g==54:require(b.get('target_field')=='optional_target','explicit Ares target field missing')
    if profile=='enemies':
        bindings=r.get('enemy_bindings',[])
        require([b['group'] for b in bindings]==[76,79],'fixed enemy bindings')
        for b,map_id,pos in zip(bindings,[0,18],[(19,15),(20,4)]):
            g=b['group'];path=f'remake/assets/maps/map{map_id}/map{map_id}_units.json'
            require(b['roster_file']==path and sha(ROOT/path)==b['roster_sha256'],'enemy roster changed')
            require(b['map']==map_id and b['camp']=='enemy' and b['portrait']==b['battle_fig']==g,'enemy binding changed')
            require(b['position']==dict(x_word=pos[0],y_word=pos[1],raw_key=g),'enemy original position changed')
            matches=[u for u in read(ROOT/path)['units'] if u.get('map_selector_key')==g and u.get('native_position_record')==b['position']]
            require(bool(matches) and all(u.get('camp')=='enemy' and u.get('portrait')==g and u.get('battle_fig')==g for u in matches),'authored enemy mismatch')
    if profile=='audited-six':
        bindings=r.get('roster_bindings',[])
        require([b['group'] for b in bindings]==list(groups) and r.get('gpu_pairs')=={k:list(v) for k,v in AUDITED_PAIRS.items()},'fixed audited bindings')
        for b in bindings:
            g=b['group'];map_id,initial,camp,x,y=AUDITED_BINDINGS[g];path=f'remake/assets/maps/map{map_id}/map{map_id}_units.json'
            require(b['roster_file']==path and sha(ROOT/path)==b['roster_sha256'],'audited roster changed')
            require(b['map']==map_id and b['initial_group']==initial and b['camp']==camp and b['portrait']==b['battle_fig']==g and b['position']==dict(x_word=x,y_word=y,raw_key=g),'audited binding changed')
            row=read(ROOT/path)['units'][b['row']]
            require(row.get('map_selector_key')==g and row.get('portrait')==row.get('battle_fig')==g and row.get('group')==initial and row.get('camp')==camp and row.get('native_position_record')==b['position'] and row.get('native_constructor')==b['native_constructor'],'authored audited source mismatch')
    if static:
        b=r.get('story_binding',{})
        require(b.get('group')==48 and b.get('map')==32 and b.get('frame_count')==3 and b.get('position')==dict(x_word=7,y_word=5,raw_key=48),'static story binding changed')
        paths=dict(roster='remake/assets/maps/map32/map32_units.json',campaign='remake/assets/scenarios/campaign_full.json',dialogue='remake/assets/story/ch00_palace.json')
        for kind,path in paths.items():require(b.get(kind+'_file')==path and sha(ROOT/path)==b.get(kind+'_sha256'),'static story source changed')
        require(b.get('actor_node')=='story_ch01_palace_throne' and b.get('speaker')==48 and b.get('speaker_name')=='父王','static actor identity changed')
        roster=[u for u in read(ROOT/paths['roster'])['units'] if u.get('map_selector_key')==48 and u.get('native_position_record')==b['position']]
        require(len(roster)==1 and roster[0].get('camp')=='own' and roster[0].get('portrait')==roster[0].get('battle_fig')==48,'static authored roster mismatch')
        scene=read(ROOT/paths['campaign'])['nodes'][b['actor_node']]
        require(scene.get('map')=='assets/maps/map32' and scene.get('script')=='assets/story/ch00_palace.json' and dict(fig=48,x=7,y=5,dir=0) in scene.get('actors',[]),'static authored actor mismatch')
        lines=[l for s in read(ROOT/paths['dialogue'])['scenes'] for l in s['lines'] if l.get('speaker')==48]
        require(bool(lines) and all(l.get('speaker_name')=='父王' for l in lines),'static dialogue identity mismatch')
    require([g['group'] for g in m['groups']]==selected,'group manifest')
    catalog=read(candidate/'catalog.json')
    parents={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_set'}
    hd={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_hd_set'}
    records={g:dict(parent=parents[g],hd=hd[g]) for g in groups}
    for row,recipe in zip(m['groups'],r['groups']):
        require(row['master_sha256']==recipe['master_sha256']==sha(candidate/row['master_file']),'master changed')
        for role in ('parent','hd'):
            require(row[role]==records[row['group']][role],'manifest/catalog source mismatch')
            require(row[role]['frame_count']==frame_count and len(row[role]['files'])==frame_count,'incomplete frames')
            for name,digest in zip(row[role]['files'],row[role]['frame_sha256']):require(sha(candidate/name)==digest,'frame changed')
    code,count=tree()
    require(d.get('source_tree_sha256')==code and d.get('source_file_count')==count,'Go source changed')
    require(d.get('native_state_unchanged') is True and d.get('save_bytes_unchanged') is True,'state/save missing')
    prefix='static' if static else 'party'
    expected={f'{prefix}-g{g:03d}-f{f:02d}-ui-hd' for g in groups for f in frames}|({f'{prefix}-g{g:03d}-f{f:02d}-ui-original' for g in groups for f in frames} if static else {f'party-g{g:03d}-f00-ui-original' for g in groups})
    cases=d.get('cases',[]);sources=d.get('sources',[])
    require(Counter(x['case'] for x in cases)==Counter(expected) and Counter(x['case'] for x in sources)==Counter(expected),'complete fixed-profile matrix')
    require({x['case'] for x in cases if x.get('f2')}=={f'{prefix}-g{g:03d}-f00-ui-hd' for g in groups},'F2 anchors')
    lookup={x['case']:x for x in sources};pixels=pngs=0
    for case in cases:
        s=lookup[case['case']];g,f=s['group'],s['frame']
        require(g in groups and f in frames and s['save_bytes']>0,'map provenance')
        if static:
            require(s.get('hud_hidden') is True and s.get('camp')==0 and s.get('portrait')==s.get('battle_fig')==s.get('selector_key')==48 and s.get('has_native_identity') is False and s.get('position')==r['story_binding']['position'] and s.get('map')==32 and s.get('roster_sha256')==r['story_binding']['roster_sha256'],'static story provenance changed')
        else:require(s['hud_group']==g and s['hud_frame']==f%3,'map/HUD provenance')
        if profile=='enemies':
            binding=next(b for b in r['enemy_bindings'] if b['group']==g)
            require(s.get('camp')==2 and s.get('portrait')==s.get('battle_fig')==s.get('selector_key')==g and s.get('position')==binding['position'] and s.get('map')==binding['map'] and s.get('roster_sha256')==binding['roster_sha256'] and s.get('constructor')=='AppendGroupWithNativePlacement' and s.get('placement_gate')==0,'authored enemy provenance changed')
        elif profile=='audited-six':
            b=next(b for b in r['roster_bindings'] if b['group']==g)
            require(s.get('camp')=={'own':0,'ally':1,'enemy':2}[b['camp']] and s.get('portrait')==s.get('battle_fig')==s.get('selector_key')==g and s.get('has_native_identity') is False and s.get('position')==b['position'] and s.get('map')==b['map'] and s.get('roster_sha256')==b['roster_sha256'] and s.get('constructor')=='AppendGroupWithNativePlacement' and s.get('placement_gate')==0,'audited actor provenance changed')
        elif profile in ('base-faces','party-four','party-five','party-six','party-seven','party-eight'):
            b=next(b for b in r['party_bindings'] if b['group']==g)
            require(s.get('native_identity')==s.get('portrait')==s.get('battle_fig')==s.get('selector_key')==g and s.get('has_native_identity') is True and s.get('camp')==0 and s.get('scenario_sha256')==b['scenario_sha256'],'authored base provenance changed')
            if profile in ('party-four','party-five','party-six','party-seven','party-eight'):require(s.get('selector_projection_injected') is False,'base party actor projection forbidden')
        elif profile=='party-nine':
            b=next(b for b in r['party_bindings'] if b['group']==g);identity=PROFILE_IDENTITIES[profile][g]
            require(s.get('native_identity')==s.get('portrait')==s.get('battle_fig')==identity and s.get('selector_key')==g and s.get('has_native_identity') is True and s.get('camp')==0 and s.get('scenario_sha256')==b['scenario_sha256'] and s.get('selector_projection_injected') is True,'promoted actor provenance changed')
        elif profile=='party-next':
            b=next(b for b in r['party_bindings'] if b['group']==g);identity=PROFILE_IDENTITIES[profile][g]
            require(s.get('native_identity')==s.get('portrait')==s.get('battle_fig')==identity and s.get('selector_key')==g and s.get('has_native_identity') is True and s.get('camp')==0 and s.get('scenario_sha256')==b['scenario_sha256'] and s.get('selector_projection_injected') is (g==54),'next party actor provenance changed')
        elif profile not in ('party','static-king'):require(s.get('native_identity')==PROFILE_IDENTITIES[profile][g] and s.get('selector_key')==g,'visual selector projection changed')
        require(case['source_frame']==f and case.get('game_draw') is True and case.get('rgba_tolerance')==2,'Game.Draw scope')
        if s['interface']!='hd':continue
        name=case['case']+'.png'
        require(case.get('file')==name and sha(folder/name)==case.get('sha256'),'GPU PNG changed')
        with Image.open(folder/name) as im:image=im.convert('RGBA')
        require(image.size==(1920,1200),'GPU geometry')
        samples=((f,rect(s['map_rect'])),) if static else ((f,rect(s['map_rect'])),(f%3,rect(s['hud_rect'])))
        for frame,box in samples:
            source=candidate/records[g]['hd']['files'][frame]
            require(sha(source)==records[g]['hd']['frame_sha256'][frame],'sample source changed')
            with Image.open(source) as im:sprite=im.convert('RGBA')
            checked,matched=core(image,sprite,box)
            require(checked>=600 and matched*100>=checked*98,'actual map/HUD pixels mismatch')
            pixels+=checked
        pngs+=1
    require(pngs==(3 if static else 6)*len(groups),'complete profile PNGs')
    return dict(status='PASS-PARTY-GPU-INDEPENDENT',profile=profile,cases=(6 if static else 7)*len(groups),pngs=pngs,near_opaque_head_pixels=pixels,source_tree_sha256=code,source_file_count=count)

def verify_normal(d,folder,candidate,binary,profile='party'):
    # The unchanged ordinary first-chapter source has no groups1/2/3. This
    # checks full candidate admission and real input/save compatibility only.
    require(d.get('status')=='PASS' and d.get('schema')=='fd2-hd1-chibi-normal-gui-v1','normal status')
    require(d.get('source_sha256')=='089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2','normal source')
    require(d.get('catalog_sha256')==sha(candidate/'catalog.json') and d.get('binary_sha256')==sha(binary),'normal provenance')
    require(d.get('gui_tool_sha256')==sha(ROOT/'tools/verify_fd2_hd_fonts_gui.py') and d.get('helper_sha256')==sha(ROOT/'tools/fd2_hd1_chibi_gui.py'),'normal tools changed')
    require(d.get('theme_and_interface_save_bytes_equal') is True and d.get('new_process_title_load') is True and d.get('state_injection') is False,'normal transaction')
    require([(s['state']['hd_theme'],s['state']['interface_mode']) for s in d['styles']]==[(True,'hd'),(True,'original'),(False,'original'),(False,'hd'),(True,'hd')],'normal four styles')
    for name,shot in d['screenshots'].items():
        require(Path(name).name==name and sha(folder/name)==shot['sha256'],'normal PNG changed')
        with Image.open(folder/name) as im:require(im.size==(1920,1200),'normal geometry')
    require(len(d['screenshots'])==10,'normal screenshots')
    visible={u.get('native_identity') for u in d['initial']['units'] if u.get('camp')==0 and u.get('on_field')}
    require(not visible.intersection(GROUPS),'source scope changed')
    scope='unchanged chapter1 source; new groups1/2/3 actual pixels verified only by controlled GPU, not natural membership'
    if profile=='variants':scope='unchanged chapter1 source with base selectors0/4/9/30; groups53/54 actual pixels verified only by controlled raw-selector projections, not promotion or natural membership'
    if profile=='identity-pair':scope='unchanged chapter1 source with base selectors0/4/9/30; groups51/63 actual pixels verified only by controlled authored identities1/13 raw-selector projections, not promotion or natural membership'
    if profile=='base-faces':
        from verify_fd2_hd1_chibi_refresh import verify_normal as verify_ares_normal
        result=verify_ares_normal(d,folder,candidate,binary,group=4)
        result['scope']='unchanged ordinary chapter1 source verifies Ares4 actual map/HUD pixels, four controls, walk/cancel and save/cold LOAD; Hano1 pixels only by authored chapter2 fixture, not natural recruitment'
        return result
    if profile=='knights':scope='unchanged chapter1 source with base selectors0/4/9/30; groups55/56 actual pixels verified only by controlled authored identities5/6 raw-selector projections, not promotion or natural membership'
    if profile=='heroes':scope='unchanged chapter1 source contains Sol0 and Unni9 base selectors, not transformed selectors50/52; new pixels verified only by controlled authored identity0/9 projections, no promotion-flow or natural transformed appearance claim'
    if profile=='enemies':scope='unchanged ordinary chapter1 save validates complete-pack admission, independent controls and save/cold LOAD; selected enemy pixels only by controlled authored initial groups map0/10 and map18/0, not natural encounter or chapter19 progression'
    if profile=='static-king':scope='unchanged ordinary chapter1 save validates full-pack admission, independent controls and save/cold LOAD; static48 only by controlled original map32 story carrier, no natural prologue or new story-save feature claim'
    if profile=='audited-six':scope='unchanged chapter1 ordinary save checks complete six-group pack admission, four controls, walk/cancel and save/cold LOAD; six new actors only by fixed authored roster pair fixtures, not natural chapter progression or persistent player identities'
    if profile=='party-next':scope='unchanged chapter1 ordinary save checks full four-group candidate, independent controls, walk/cancel and save/cold LOAD; groups5/6/13 use controlled authored party,54 is explicit identity4 selector projection; not natural recruitment or promotion'
    if profile=='party-four':scope='unchanged chapter1 ordinary save checks full candidate, four controls, walk/cancel and save/cold LOAD; groups7/8/10/11 pixels only through controlled authored party with no selector projection, not natural chapter recruitment'
    if profile=='party-five':scope='unchanged chapter1 ordinary save checks full candidate, four controls, walk/cancel and save/cold LOAD; groups12/14/15/16 pixels only through controlled authored party without selector projection, not natural recruitment'
    if profile=='party-six':scope='unchanged chapter1 ordinary save checks full candidate, four controls, walk/cancel and save/cold LOAD; groups17/18/20/21 pixels only through controlled authored party without selector projection, not natural recruitment'
    if profile=='party-seven':scope='unchanged chapter1 ordinary save checks full candidate, four controls, walk/cancel and save/cold LOAD; groups22/23/24/25 pixels only through controlled authored party without selector projection, not natural recruitment'
    if profile=='party-eight':scope='unchanged chapter1 ordinary save checks full candidate, four controls, walk/cancel and save/cold LOAD; groups26/27/28/29 pixels only through controlled authored party without selector projection, not natural recruitment'
    if profile=='party-nine':scope='unchanged chapter1 ordinary save checks full candidate, four controls, walk/cancel and save/cold LOAD;58/59/60/61 pixels only through disclosed selector/byte6/cache projection with base identities8/9/10/11, not natural promotion'
    return dict(status='PASS-FULL-CANDIDATE-NORMAL-TRANSACTION',pngs=10,scope=scope)

def register(formal,candidate,destination,recipe=RECIPE,profile='party'):
    old=read(formal);new=read(candidate/'catalog.json');m=read(candidate/'manifest.json')
    require(sha(formal)==m['base_catalog_sha256'],'formal changed')
    require(len(old['assets'])==len(new['assets'])==883,'asset count changed')
    changed=[(a,b) for a,b in zip(old['assets'],new['assets']) if a!=b]
    groups=[g['group'] for g in read(recipe)['groups']]
    require(bool(groups) and len(set(groups))==len(groups) and set(groups)<=set(PROFILES[profile]),'registration groups')
    require(len(changed)==2*len(groups) and Counter((a['source_group'],a['role']) for a,b in changed)==Counter((g,role) for g in groups for role in ('map_sprite_set','map_sprite_hd_set')),'unrelated changes')
    require(all(a['asset_id']==b['asset_id'] and a['source_group']==b['source_group'] and a['role']==b['role'] for a,b in changed),'asset identity changed')
    require(formal.stat().st_uid==destination.stat().st_uid==os.getuid(),'owned output required')
    names=set()
    for g in m['groups']:
        names.add(g['master_file'])
        for role in ('parent','hd'):names.update(g[role]['files'])
    require(len(names)==(7 if profile=='static-king' else 25)*len(groups),'expected one master and complete native frames per group')
    for name in names:
        require(Path(name).name==name and (candidate/name).is_file(),'unsafe file')
        require(not (destination/name).exists() or sha(destination/name)==sha(candidate/name),'different existing art')
    for name in names:
        if not (destination/name).exists():shutil.copyfile(candidate/name,destination/name)
    new['private_root']=old['private_root']
    with tempfile.NamedTemporaryFile('w',dir=formal.parent,delete=False) as f:
        f.write(json.dumps(new,ensure_ascii=False,indent=2)+'\n');temp=Path(f.name)
    os.chmod(temp,formal.stat().st_mode&0o777);temp.replace(formal)
    return dict(status='REGISTERED-CHIBI-PARTY',catalog_sha256=sha(formal),changed_entries=len(changed),unchanged_entries=883-len(changed),new_files=len(names))

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ('gpu','candidate','normal','binary','out'):p.add_argument('--'+n,type=Path,required=True)
    for n in ('formal','destination'):p.add_argument('--'+n,type=Path)
    p.add_argument('--recipe',type=Path,default=RECIPE,help='明示本批配方；必須符合所選固定矩陣')
    p.add_argument('--profile',choices=PROFILES,default='party',help='party固定21；base-faces1／4無投影；其他固定雙組14；static-king48固定六；audited-six三份42，party-next兩份28揭露54投影，party-four／party-five／party-six／party-seven／party-eight兩份28無投影；party-nine兩份28明示四組投影')
    p.add_argument('--gpu-parts',nargs='+',type=Path,help='audited-six固定三份、party-next／party-four／party-five／party-six／party-seven／party-eight／party-nine固定兩份GPU收據；--gpu指定新的合併目錄內收據')
    p.add_argument('--self-test',action='store_true');p.add_argument('--register',action='store_true');a=p.parse_args()
    require(not a.out.exists(),'fresh report required')
    if a.gpu_parts:
        require(a.profile in GPU_PARTS,'GPU parts require a fixed split profile')
        assemble_gpu_parts(a.gpu,a.gpu_parts)
    d=read(a.gpu);result=verify_gpu(d,a.gpu.parent,a.candidate,a.recipe,a.profile);result['normal']=verify_normal(read(a.normal),a.normal.parent,a.candidate,a.binary,a.profile)
    rejected=[]
    if a.self_test:
        for fault in ('missing','duplicate','catalog','source-code','false-save','wrong-hud','wrong-tolerance','missing-f2','wrong-profile','wrong-selector','wrong-identity','undisclosed-projection','wrong-base-source','false-base-identity','wrong-actor-projection','wrong-enemy-camp','wrong-enemy-position','wrong-enemy-roster','wrong-static-phase','wrong-story-source','wrong-story-position','missing-part','duplicate-part','wrong-part-source','wrong-part-group','wrong-part-memory'):
            x=deepcopy(d)
            if fault=='missing':x['cases'].pop()
            elif fault=='duplicate':x['cases'][-1]=deepcopy(x['cases'][0])
            elif fault=='catalog':x['catalog_sha256']='0'*64
            elif fault=='source-code':x['source_tree_sha256']='0'*64
            elif fault=='false-save':x['save_bytes_unchanged']=False
            elif fault=='wrong-hud':
                if a.profile=='static-king':x['sources'][0]['hud_hidden']=False
                else:x['sources'][0]['hud_group']=99
            elif fault=='wrong-tolerance':x['cases'][0]['rgba_tolerance']=3
            elif fault=='missing-f2':
                for row in x['cases']:row['f2']=False
            elif fault=='wrong-profile':x['profile']='party' if a.profile=='variants' else 'variants'
            elif fault=='wrong-selector':
                if a.profile=='party':continue
                x['sources'][0]['selector_key']=99
            elif fault=='wrong-identity':
                if a.profile=='party':continue
                if a.profile in ('enemies','static-king','audited-six'):x['sources'][0]['portrait']=99
                else:
                    subject=next(row for row in x['sources'] if row['group']==52) if a.profile=='heroes' else x['sources'][0]
                    subject['native_identity']=2 if a.profile=='heroes' else 99
            elif fault=='undisclosed-projection':
                if a.profile=='party':continue
                x.pop('selector_projection_injected',None)
            elif fault=='wrong-base-source':
                if a.profile not in ('base-faces','party-next','party-four','party-five','party-six','party-seven','party-eight','party-nine'):continue
                x['sources'][0]['scenario_sha256']='0'*64
            elif fault=='false-base-identity':
                if a.profile not in ('base-faces','party-next','party-four','party-five','party-six','party-seven','party-eight','party-nine'):continue
                x['sources'][0]['has_native_identity']=False
            elif fault=='wrong-actor-projection':
                if a.profile not in ('party-next','party-four','party-five','party-six','party-seven','party-eight','party-nine'):continue
                x['sources'][0]['selector_projection_injected']=not x['sources'][0].get('selector_projection_injected',False)
            elif fault=='wrong-enemy-camp':
                if a.profile not in ('enemies','audited-six'):continue
                x['sources'][0]['camp']=0
            elif fault=='wrong-enemy-position':
                if a.profile not in ('enemies','audited-six'):continue
                x['sources'][0]['position']['raw_key']=0
            elif fault=='wrong-enemy-roster':
                if a.profile not in ('enemies','audited-six'):continue
                x['sources'][0]['roster_sha256']='0'*64
            elif fault=='wrong-static-phase':
                if a.profile!='static-king':continue
                x['sources'][0]['frame']=3
            elif fault=='wrong-story-source':
                if a.profile!='static-king':continue
                x['sources'][0]['roster_sha256']='0'*64
            elif fault=='wrong-story-position':
                if a.profile!='static-king':continue
                x['sources'][0]['position']['raw_key']=0
            elif fault=='missing-part':
                if a.profile not in GPU_PARTS:continue
                x['source_parts'].pop()
            elif fault=='duplicate-part':
                if a.profile not in GPU_PARTS:continue
                x['source_parts'][-1]=deepcopy(x['source_parts'][0])
            elif fault=='wrong-part-source':
                if a.profile not in GPU_PARTS:continue
                x['source_parts'][0]['sha256']='0'*64
            elif fault=='wrong-part-group':
                if a.profile not in GPU_PARTS:continue
                x['source_parts'][0]['pair']='robed'
            else:
                if a.profile not in GPU_PARTS:continue
                x['source_parts'][0]['memory']['sha256']='0'*64
            try:verify_gpu(x,a.gpu.parent,a.candidate,a.recipe,a.profile)
            except (ValueError,KeyError,IndexError):rejected.append(fault)
            else:raise ValueError('mutant accepted: '+fault)
        with tempfile.TemporaryDirectory(dir=a.out.parent) as t:
            folder=Path(t);x=deepcopy(d);changed=False
            for row in x['cases']:
                if not row.get('file'):continue
                out=folder/row['file']
                if not changed:Image.new('RGB',(1920,1200)).save(out);row['sha256']=sha(out);changed=True
                else:out.symlink_to((a.gpu.parent/row['file']).resolve())
            try:verify_gpu(x,folder,a.candidate,a.recipe,a.profile)
            except ValueError as e:require('pixels mismatch' in str(e),'black PNG wrong failure');rejected.append('actual-black-png')
            else:raise ValueError('black PNG accepted')
    if a.self_test and a.profile=='party-nine':
        for fault in ('promotion-source-sha','promotion-target-field','promotion-native-identity','promotion-unreported-selector'):
            r=deepcopy(read(a.recipe))
            if fault=='promotion-source-sha':r['promotion_source']['sha256']='0'*64
            elif fault=='promotion-target-field':r['party_bindings'][0]['target_field']='default_target'
            elif fault=='promotion-native-identity':r['party_bindings'][1]['native_identity']=2
            else:r['party_bindings'][0]['selector_projection_injected']=False
            try:verify_promoted_bindings(r)
            except (ValueError,KeyError,IndexError):rejected.append(fault)
            else:raise ValueError('promoted recipe mutant accepted: '+fault)
    result['rejected_mutants']=rejected
    if a.register:
        require(a.formal is not None and a.destination is not None,'formal/destination required')
        result['registration']=register(a.formal,a.candidate,a.destination,a.recipe,a.profile)
    result['inputs']={k+'_sha256':sha(v) for k,v in dict(gpu=a.gpu,normal=a.normal,binary=a.binary,recipe=a.recipe,tool=Path(__file__)).items()}
    a.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
