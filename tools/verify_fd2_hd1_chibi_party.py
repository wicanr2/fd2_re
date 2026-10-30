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
PROFILES={'party':GROUPS,'variants':(53,54),'identity-pair':(51,63),'knights':(55,56),'heroes':(50,52),'enemies':(76,79),'static-king':(48,)}
PROFILE_IDENTITIES={'variants':{53:3,54:4},'identity-pair':{51:1,63:13},'knights':{55:5,56:6},'heroes':{50:0,52:9}}
FRAMES=(0,1,2,3,6,9)

def verify_gpu(d,folder,candidate,recipe=RECIPE,profile='party'):
    groups=PROFILES[profile]
    static=profile=='static-king'
    frames=(0,1,2) if static else FRAMES
    frame_count=3 if static else 12
    m=read(candidate/'manifest.json');r=read(recipe)
    require(d.get('status')=='PASS-HD1-CHIBI-PARTY-GPU','GPU status')
    require(d.get('profile','party')==profile,'GPU profile changed')
    if static:
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
    if profile=='heroes':
        bindings=[dict(group=50,native_identity=0,target_field='optional_target'),dict(group=52,native_identity=9,target_field='special_target')]
        require(r.get('selector_bindings')==bindings,'explicit special identity binding missing')
        source=r['promotion_source']
        require(source['file']=='remake/assets/data/class_change_targets.json' and sha(ROOT/source['file'])==source['sha256'],'promotion source changed')
        rows=read(ROOT/source['file'])['current_portraits']
        for b in bindings:
            matches=[row for row in rows if row['portrait']==b['native_identity']]
            require(len(matches)==1 and matches[0].get(b['target_field'])==b['group'],'authored promotion identity mismatch')
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
    if profile=='knights':scope='unchanged chapter1 source with base selectors0/4/9/30; groups55/56 actual pixels verified only by controlled authored identities5/6 raw-selector projections, not promotion or natural membership'
    if profile=='heroes':scope='unchanged chapter1 source contains Sol0 and Unni9 base selectors, not transformed selectors50/52; new pixels verified only by controlled authored identity0/9 projections, no promotion-flow or natural transformed appearance claim'
    if profile=='enemies':scope='unchanged ordinary chapter1 save validates complete-pack admission, independent controls and save/cold LOAD; selected enemy pixels only by controlled authored initial groups map0/10 and map18/0, not natural encounter or chapter19 progression'
    if profile=='static-king':scope='unchanged ordinary chapter1 save validates full-pack admission, independent controls and save/cold LOAD; static48 only by controlled original map32 story carrier, no natural prologue or new story-save feature claim'
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
    p.add_argument('--profile',choices=PROFILES,default='party',help='party固定21；variants53／54、identity-pair51／63、knights55／56、heroes50／52各固定14，heroes明示0／9身份；enemies76／79保持敵方原生名冊；static-king48固定六繪圖及三拍')
    p.add_argument('--self-test',action='store_true');p.add_argument('--register',action='store_true');a=p.parse_args()
    require(not a.out.exists(),'fresh report required')
    d=read(a.gpu);result=verify_gpu(d,a.gpu.parent,a.candidate,a.recipe,a.profile);result['normal']=verify_normal(read(a.normal),a.normal.parent,a.candidate,a.binary,a.profile)
    rejected=[]
    if a.self_test:
        for fault in ('missing','duplicate','catalog','source-code','false-save','wrong-hud','wrong-tolerance','missing-f2','wrong-profile','wrong-selector','wrong-identity','undisclosed-projection','wrong-enemy-camp','wrong-enemy-position','wrong-enemy-roster','wrong-static-phase','wrong-story-source','wrong-story-position'):
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
                if a.profile in ('enemies','static-king'):x['sources'][0]['portrait']=99
                else:
                    subject=next(row for row in x['sources'] if row['group']==52) if a.profile=='heroes' else x['sources'][0]
                    subject['native_identity']=2 if a.profile=='heroes' else 99
            elif fault=='undisclosed-projection':
                if a.profile=='party':continue
                x.pop('selector_projection_injected',None)
            elif fault=='wrong-enemy-camp':
                if a.profile!='enemies':continue
                x['sources'][0]['camp']=0
            elif fault=='wrong-enemy-position':
                if a.profile!='enemies':continue
                x['sources'][0]['position']['raw_key']=0
            elif fault=='wrong-enemy-roster':
                if a.profile!='enemies':continue
                x['sources'][0]['roster_sha256']='0'*64
            elif fault=='wrong-static-phase':
                if a.profile!='static-king':continue
                x['sources'][0]['frame']=3
            elif fault=='wrong-story-source':
                if a.profile!='static-king':continue
                x['sources'][0]['roster_sha256']='0'*64
            else:
                if a.profile!='static-king':continue
                x['sources'][0]['position']['raw_key']=0
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
    result['rejected_mutants']=rejected
    if a.register:
        require(a.formal is not None and a.destination is not None,'formal/destination required')
        result['registration']=register(a.formal,a.candidate,a.destination,a.recipe,a.profile)
    result['inputs']={k+'_sha256':sha(v) for k,v in dict(gpu=a.gpu,normal=a.normal,binary=a.binary,recipe=a.recipe,tool=Path(__file__)).items()}
    a.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
