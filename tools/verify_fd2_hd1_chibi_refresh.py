#!/usr/bin/env python3
"""獨立核對Q版人物的完整矩陣、實際像素與普通操作，再原子登錄。"""
import argparse
from collections import Counter
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image

ROOT=Path(__file__).resolve().parents[1]
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def read(p):return json.loads(p.read_text())
def require(ok,message):
    if not ok:raise ValueError(message)
def rect(r):return [r['Min']['X'],r['Min']['Y'],r['Max']['X'],r['Max']['Y']]
def tree():
    files=sorted(list((ROOT/'remake/cmd').rglob('*.go'))+list((ROOT/'remake/internal').rglob('*.go')),key=lambda p:p.as_posix())
    return hashlib.sha256(''.join(f'{p.relative_to(ROOT/"remake").as_posix()}\0{sha(p)}\n' for p in files).encode()).hexdigest(),len(files)
def core(image,sprite,box):
    # The accepted master has mostly alpha253 interiors. Bound SrcOver by
    # every possible background channel, including integer rounding. Native
    # HP begins at 182, while this unit begins at166: exclude y>=16*6.
    x0,y0,x1,y1=box;require(x1-x0==y1-y0==144,'invalid sprite rectangle')
    actual=image.load();source=sprite.load();checked=matched=0
    for y in range(96):
        for x in range(144):
            tx,ty=x0+x,y0+y
            alpha=source[x,y][3]
            if not (24<=tx<1896 and 24<=ty<1176) or alpha<253:continue
            checked+=1
            if all((source[x,y][i]*alpha+127)//255-1<=actual[tx,ty][i]<=(source[x,y][i]*alpha+255*(255-alpha)+127)//255+1 for i in range(3)):matched+=1
    return checked,matched

def verify_gpu(d,folder,candidate):
    catalog=candidate/'catalog.json';manifest=candidate/'manifest.json';m=read(manifest)
    recipe=ROOT/'remake/assets/themes/modern/hd1-ares-chibi-refresh.recipe.json';r=read(recipe)
    require(m['compiler_sha256']==sha(ROOT/'tools/refresh_fd2_hd1_map_sprites.py') and m['recipe_sha256']==sha(recipe),'compiler or recipe changed')
    require(sha(ROOT/r['approved_preview'])==r['approved_preview_sha256'],'approved design changed')
    require(m['groups'][0]['master_sha256']==r['groups'][0]['master_sha256'] and sha(candidate/m['groups'][0]['master_file'])==r['groups'][0]['master_sha256'],'approved master changed')
    require(d.get('status')=='PASS-HD1-CHIBI-REFRESH-GPU','GPU status')
    require(d.get('catalog_sha256')==sha(catalog) and d.get('manifest_sha256')==sha(manifest),'candidate provenance')
    digest,count=tree();require(d.get('source_tree_sha256')==digest and d.get('source_file_count')==count,'current Go source')
    require(d.get('native_state_unchanged') is True and d.get('save_bytes_unchanged') is True,'state or save check missing')
    expected={f'ares-f{f:02d}-ui-{ui}' for f in range(12) for ui in ('original','hd')}
    cases=d.get('cases',[]);sources=d.get('sources',[])
    require(Counter(r['case'] for r in cases)==Counter(expected) and Counter(r['case'] for r in sources)==Counter(expected),'24-case matrix missing or duplicated')
    require({r['source_frame'] for r in cases if r.get('f2')}=={0,3,6,9},'F2 direction anchors missing')
    lookup={r['case']:r for r in sources};assets=m['groups'][0]['hd'];pixels=0;pngs=0
    require(len(m['groups'])==1 and m['groups'][0]['group']==4,'GPU only verifies Ares group4')
    for row in cases:
        s=lookup[row['case']];f=s['frame'];require(s['group']==4 and s['hud_group']==4 and s['hud_frame']==f%3 and s['save_bytes']>0,'map/HUD source mismatch')
        require(row.get('source_frame')==f and row.get('game_draw') is True and row.get('rgba_tolerance')==2,'not complete Game.Draw at original tolerance')
        if s['interface']=='hd':
            name=row.get('file','');require(name==row['case']+'.png' and sha(folder/name)==row.get('sha256'),'GPU PNG missing or changed')
            image=Image.open(folder/name).convert('RGBA');require(image.size==(1920,1200),'not full window')
            for index,box in ((f,rect(s['map_rect'])),(f%3,rect(s['hud_rect']))):
                source=candidate/assets['files'][index];require(sha(source)==assets['frame_sha256'][index],'actual art source changed')
                checked,matched=core(image,Image.open(source).convert('RGBA'),box)
                require(checked>=600 and matched*100>=checked*98,'actual map/HUD chibi pixels mismatch')
                pixels+=checked
            pngs+=1
    require(pngs==12,'twelve complete PNGs missing')
    return dict(status='PASS-CHIBI-GPU-INDEPENDENT',cases=24,pngs=pngs,near_opaque_head_pixels=pixels,pixel_policy='alpha>=253；逐通道SrcOver背景上下界含一次整數捨入；取人物前96列避開原生HP覆蓋。完整GPU容差2另行保持。',source_tree_sha256=digest,source_file_count=count)

def verify_normal(d,folder,candidate,binary):
    require(d.get('status')=='PASS' and d.get('schema')=='fd2-hd1-chibi-normal-gui-v1','normal status')
    require(d.get('source_sha256')=='089b077b8d334babd5d9f7005ff6ed6499ccb7987520a8f56d59e79bf3edb1d2','normal source')
    require(d.get('catalog_sha256')==sha(candidate/'catalog.json') and d.get('binary_sha256')==sha(binary),'normal candidate or binary')
    require(d.get('gui_tool_sha256')==sha(ROOT/'tools/verify_fd2_hd_fonts_gui.py') and d.get('helper_sha256')==sha(ROOT/'tools/fd2_hd1_chibi_gui.py'),'normal tool changed')
    require(d.get('theme_and_interface_save_bytes_equal') is True and d.get('new_process_title_load') is True and d.get('state_injection') is False,'normal save/cold-load check missing')
    require([(s['state']['hd_theme'],s['state']['interface_mode']) for s in d['styles']]==[(True,'hd'),(True,'original'),(False,'original'),(False,'hd'),(True,'hd')],'normal four styles')
    asset=read(candidate/'manifest.json')['groups'][0]['hd'];sprites=[Image.open(candidate/n).convert('RGBA') for n in asset['files']]
    matches=[]
    for name,shot in d['screenshots'].items():
        require(Path(name).name==name and sha(folder/name)==shot['sha256'],'normal PNG changed')
        image=Image.open(folder/name).convert('RGBA');require(image.size==(1920,1200),'normal window geometry')
        if name not in {'ares-chibi-hd-hd.png','ares-chibi-hd-original.png','ares-chibi-hd-hd-restored.png'}:continue
        states=[shot['gpu_before'],shot['state']]
        boxes={tuple(u['rect']) for s in states for u in s.get('hd_world_units',[]) if u['group']==4}
        options=[(i,list(box),*core(image,sprite,box)) for box in boxes for i,sprite in enumerate(sprites)]
        accepted=[r for r in options if r[2]>=600 and r[3]*100>=r[2]*98]
        require(bool(accepted),'normal actual Ares pixels mismatch')
        hud=[(i,anchor,*core(image,sprites[i],[(anchor+10)*6,996,(anchor+34)*6,1140])) for i in range(3) for anchor in (1,242)]
        hud=[r for r in hud if r[2]>=600 and r[3]*100>=r[2]*98]
        require(bool(hud),'normal HUD retained wrong artwork')
        matches.append(dict(file=name,map=accepted[0],hud=hud[0]))
    require(len(matches)==3,'normal HD source screenshots missing')
    return dict(status='PASS-CHIBI-NORMAL-INDEPENDENT',pngs=len(d['screenshots']),map_and_hud_matches=matches)

def register(formal,candidate,destination):
    c=read(formal);new=read(candidate/'catalog.json');m=read(candidate/'manifest.json')
    require(sha(formal)==m['base_catalog_sha256'],'formal catalog changed during validation')
    require(sha(candidate/'catalog.json')==m['catalog_sha256'],'candidate changed')
    changed=[(a,b) for a,b in zip(c['assets'],new['assets']) if a!=b]
    require(len(c['assets'])==len(new['assets']) and len(changed)==2 and all(a['source_group']==b['source_group']==4 and a['asset_id']==b['asset_id'] for a,b in changed),'unrelated assets changed')
    require(destination.is_dir() and destination.stat().st_uid==os.getuid() and formal.stat().st_uid==os.getuid(),'owned output required')
    names={m['groups'][0]['master_file']}
    for role in ('parent','hd'):names.update(m['groups'][0][role]['files'])
    for name in names:
        source=candidate/name;target=destination/name
        require(Path(name).name==name and source.is_file(),'invalid copy source')
        require(not target.exists() or sha(target)==sha(source),'would overwrite different art')
    for name in names:
        if not (destination/name).exists():shutil.copyfile(candidate/name,destination/name)
    new['private_root']=c['private_root']
    with tempfile.NamedTemporaryFile('w',dir=formal.parent,delete=False) as f:
        f.write(json.dumps(new,ensure_ascii=False,indent=2)+'\n');temp=Path(f.name)
    os.chmod(temp,formal.stat().st_mode&0o777);temp.replace(formal)
    return dict(status='REGISTERED-ARES-CHIBI',catalog_sha256=sha(formal),new_files=len(names))

def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ('gpu','candidate','out'):p.add_argument('--'+n,type=Path,required=True)
    for n in ('normal','binary','formal','destination'):p.add_argument('--'+n,type=Path)
    p.add_argument('--register',action='store_true');p.add_argument('--self-test',action='store_true');a=p.parse_args()
    require(not a.out.exists(),'fresh report required')
    d=read(a.gpu);result=verify_gpu(d,a.gpu.parent,a.candidate);bad=[]
    if a.self_test:
        for fault in ('missing','duplicate','catalog','source-code','false-save','wrong-frame','wrong-hud','wrong-tolerance','missing-f2'):
            x=deepcopy(d)
            if fault=='missing':x['cases'].pop()
            elif fault=='duplicate':x['cases'][-1]=deepcopy(x['cases'][0])
            elif fault=='catalog':x['catalog_sha256']='0'*64
            elif fault=='source-code':x['source_tree_sha256']='0'*64
            elif fault=='false-save':x['save_bytes_unchanged']=False
            elif fault=='wrong-frame':x['sources'][0]['frame']=99
            elif fault=='wrong-hud':x['sources'][0]['hud_group']=0
            elif fault=='wrong-tolerance':x['cases'][0]['rgba_tolerance']=3
            else:
                for r in x['cases']:r['f2']=False
            try:verify_gpu(x,a.gpu.parent,a.candidate)
            except (ValueError,KeyError,IndexError):bad.append(fault)
            else:raise ValueError('mutant accepted: '+fault)
        with tempfile.TemporaryDirectory(dir=a.out.parent) as temp:
            folder=Path(temp);x=deepcopy(d);changed=False
            for row in x['cases']:
                if not row.get('file'):continue
                path=folder/row['file']
                if not changed:
                    Image.new('RGB',(1920,1200)).save(path);row['sha256']=sha(path);changed=True
                else:path.symlink_to((a.gpu.parent/row['file']).resolve())
            try:verify_gpu(x,folder,a.candidate)
            except ValueError as error:
                require('pixels mismatch' in str(error),'black PNG failed for wrong reason');bad.append('actual-black-png')
            else:raise ValueError('actual black PNG accepted')
    result['rejected_mutants']=bad
    if a.normal:require(a.binary is not None,'binary required');result['normal']=verify_normal(read(a.normal),a.normal.parent,a.candidate,a.binary)
    if a.register:
        require(a.normal is not None and a.formal is not None and a.destination is not None,'normal/formal/destination required')
        result['registration']=register(a.formal,a.candidate,a.destination)
    result['inputs']=dict(gpu_sha256=sha(a.gpu),manifest_sha256=sha(a.candidate/'manifest.json'),tool_sha256=sha(Path(__file__)))
    a.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))
if __name__=='__main__':main()
