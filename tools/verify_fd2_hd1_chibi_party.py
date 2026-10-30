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
PROFILES={'party':GROUPS,'variants':(53,54),'identity-pair':(51,63)}
FRAMES=(0,1,2,3,6,9)

def verify_gpu(d,folder,candidate,recipe=RECIPE,profile='party'):
    groups=PROFILES[profile]
    m=read(candidate/'manifest.json');r=read(recipe)
    require(d.get('status')=='PASS-HD1-CHIBI-PARTY-GPU','GPU status')
    require(d.get('profile','party')==profile,'GPU profile changed')
    if profile!='party':require(d.get('selector_projection_injected') is True,'selector injection disclosure missing')
    require(d.get('catalog_sha256')==sha(candidate/'catalog.json') and d.get('manifest_sha256')==sha(candidate/'manifest.json'),'candidate provenance')
    require(m['recipe_sha256']==sha(recipe) and m['compiler_sha256']==sha(ROOT/'tools/refresh_fd2_hd1_map_sprites.py'),'recipe/compiler changed')
    require(sha(ROOT/r['approved_preview'])==r['approved_preview_sha256'],'approved proportion changed')
    selected=[g['group'] for g in r['groups']]
    require(bool(selected) and len(set(selected))==len(selected) and set(selected)<=set(groups),'recipe groups')
    if profile!='party':require(selected==list(groups),'fixed variant recipe required')
    require([g['group'] for g in m['groups']]==selected,'group manifest')
    catalog=read(candidate/'catalog.json')
    parents={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_set'}
    hd={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_hd_set'}
    records={g:dict(parent=parents[g],hd=hd[g]) for g in groups}
    for row,recipe in zip(m['groups'],r['groups']):
        require(row['master_sha256']==recipe['master_sha256']==sha(candidate/row['master_file']),'master changed')
        for role in ('parent','hd'):
            require(row[role]==records[row['group']][role],'manifest/catalog source mismatch')
            require(len(row[role]['files'])==12,'incomplete frames')
            for name,digest in zip(row[role]['files'],row[role]['frame_sha256']):require(sha(candidate/name)==digest,'frame changed')
    code,count=tree()
    require(d.get('source_tree_sha256')==code and d.get('source_file_count')==count,'Go source changed')
    require(d.get('native_state_unchanged') is True and d.get('save_bytes_unchanged') is True,'state/save missing')
    expected={f'party-g{g:03d}-f{f:02d}-ui-hd' for g in groups for f in FRAMES}|{f'party-g{g:03d}-f00-ui-original' for g in groups}
    cases=d.get('cases',[]);sources=d.get('sources',[])
    require(Counter(x['case'] for x in cases)==Counter(expected) and Counter(x['case'] for x in sources)==Counter(expected),'complete fixed-profile matrix')
    require({x['case'] for x in cases if x.get('f2')}=={f'party-g{g:03d}-f00-ui-hd' for g in groups},'F2 anchors')
    lookup={x['case']:x for x in sources};pixels=pngs=0
    for case in cases:
        s=lookup[case['case']];g,f=s['group'],s['frame']
        require(g in groups and s['hud_group']==g and s['hud_frame']==f%3 and s['save_bytes']>0,'map/HUD provenance')
        if profile!='party':require(s.get('native_identity')==g-50 and s.get('selector_key')==g,'visual selector projection changed')
        require(case['source_frame']==f and case.get('game_draw') is True and case.get('rgba_tolerance')==2,'Game.Draw scope')
        if s['interface']!='hd':continue
        name=case['case']+'.png'
        require(case.get('file')==name and sha(folder/name)==case.get('sha256'),'GPU PNG changed')
        with Image.open(folder/name) as im:image=im.convert('RGBA')
        require(image.size==(1920,1200),'GPU geometry')
        for frame,box in ((f,rect(s['map_rect'])),(f%3,rect(s['hud_rect']))):
            source=candidate/records[g]['hd']['files'][frame]
            require(sha(source)==records[g]['hd']['frame_sha256'][frame],'sample source changed')
            with Image.open(source) as im:sprite=im.convert('RGBA')
            checked,matched=core(image,sprite,box)
            require(checked>=600 and matched*100>=checked*98,'actual map/HUD pixels mismatch')
            pixels+=checked
        pngs+=1
    require(pngs==6*len(groups),'complete profile PNGs')
    return dict(status='PASS-PARTY-GPU-INDEPENDENT',profile=profile,cases=7*len(groups),pngs=pngs,near_opaque_head_pixels=pixels,source_tree_sha256=code,source_file_count=count)

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
    require(len(names)==25*len(groups),'expected one master and24 frames per group')
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
    p.add_argument('--profile',choices=PROFILES,default='party',help='party固定21案例；variants固定53／54，identity-pair固定51／63，各14案例與明示selector投影')
    p.add_argument('--self-test',action='store_true');p.add_argument('--register',action='store_true');a=p.parse_args()
    require(not a.out.exists(),'fresh report required')
    d=read(a.gpu);result=verify_gpu(d,a.gpu.parent,a.candidate,a.recipe,a.profile);result['normal']=verify_normal(read(a.normal),a.normal.parent,a.candidate,a.binary,a.profile)
    rejected=[]
    if a.self_test:
        for fault in ('missing','duplicate','catalog','source-code','false-save','wrong-hud','wrong-tolerance','missing-f2','wrong-profile','wrong-selector','undisclosed-projection'):
            x=deepcopy(d)
            if fault=='missing':x['cases'].pop()
            elif fault=='duplicate':x['cases'][-1]=deepcopy(x['cases'][0])
            elif fault=='catalog':x['catalog_sha256']='0'*64
            elif fault=='source-code':x['source_tree_sha256']='0'*64
            elif fault=='false-save':x['save_bytes_unchanged']=False
            elif fault=='wrong-hud':x['sources'][0]['hud_group']=99
            elif fault=='wrong-tolerance':x['cases'][0]['rgba_tolerance']=3
            elif fault=='missing-f2':
                for row in x['cases']:row['f2']=False
            elif fault=='wrong-profile':x['profile']='party' if a.profile=='variants' else 'variants'
            elif fault=='wrong-selector':
                if a.profile=='party':continue
                x['sources'][0]['selector_key']=99
            else:
                if a.profile=='party':continue
                x.pop('selector_projection_injected',None)
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
