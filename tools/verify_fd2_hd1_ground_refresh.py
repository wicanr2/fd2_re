#!/usr/bin/env python3
"""獨立核對HD1底材刷新、完整地圖PNG、來源閉合與登錄邊界。"""
import argparse,copy,hashlib,json,shutil,tempfile
from pathlib import Path
from PIL import Image
from refresh_fd2_hd1_ground import KIND,read,reject,structural,sha
from build_fd2_hd_map_overview import validate as overview_validate
from verify_fd2_battle_hd_consumers import source_tree

def verify(candidate,recipe_path,base_path,gpu_path,coverage):
    proof=read(candidate/'manifest.json');r=read(recipe_path);base=read(base_path);cat=read(candidate/'catalog.json');gpu=read(gpu_path);detail=read(candidate/'gpu-detail.json')
    reject(proof['status']=='PASS-HD1-PAINTED-GROUND-CANDIDATE' and proof['kind']==KIND,'candidate status')
    reject(sha(recipe_path)==proof['recipe_sha256'] and sha(base_path)==proof['base_catalog_sha256'],'candidate input changed')
    reject(sha(candidate/'catalog.json')==proof['catalog_sha256'] and sha(candidate/'gpu-detail.json')==proof['gpu_detail_sha256'],'candidate publication changed')
    changed=structural(base,cat,r,{v['old_file']:v for v in proof['frames']})
    reject(len(proof['frames'])==87 and len(proof['bindings'])==384 and len({v['map'] for v in proof['bindings']})==21,'source closure incomplete')
    reject(len({(v['map'],v['frame']) for v in proof['bindings']})==384,'source closure duplicate')
    for m in proof['masters']:
        p=candidate/'pack'/m['file'];reject(sha(p)==m['sha256'],'master changed')
        with Image.open(p) as im:reject(list(im.size)==m['size'] and im.convert('RGBA').getextrema()[3]==(255,255),'master shape')
    for frame in proof['frames']:
        p=candidate/'pack'/frame['file'];reject(sha(p)==frame['sha256'],'frame changed')
        with Image.open(p) as im:reject(im.size==(144,144) and im.convert('RGBA').getextrema()[3]==(255,255),'frame shape/alpha')
    reject(gpu['status']=='PASS-HD-MAP-OVERVIEW-GPU' and gpu['catalog_sha256']==proof['catalog_sha256'],'GPU catalog/status')
    reject(gpu['detail_manifest_sha256']==proof['gpu_detail_sha256'] and gpu['detail_source_cases']==428,'GPU detail coverage')
    reject((gpu['source_tree_sha256'],gpu['source_file_count'])==source_tree(),'GPU source code changed')
    cases=gpu['cases'];sources=[v for v in cases if not v['game_draw']]
    expected=[f"details-{v['map']:02d}-{v['frame']:04d}-{mode}" for v in detail['bindings'] for mode in ('steady','lut','black','white')]
    reject([v['case'] for v in sources]==expected and len(sources)==428,'GPU source matrix incomplete')
    reject(all(v['rgba_tolerance']==2 and len(v['world_rgba_sha256'])==64 for v in cases),'GPU tolerance/digest')
    rows=overview_validate(gpu,gpu_path.parent,candidate/'catalog.json',coverage)
    reject(len(rows)==6,'full Game.Draw coverage')
    for row in rows:
        with Image.open(gpu_path.parent/row['file']) as im:
            reject(all(high-low>=64 for low,high in im.convert('RGB').getextrema()),'actual gameplay PNG missing')
    return dict(status='PASS-HD1-PAINTED-GROUND-INDEPENDENT',recipe_sha256=sha(recipe_path),candidate_sha256=sha(candidate/'catalog.json'),manifest_sha256=sha(candidate/'manifest.json'),gpu_sha256=sha(gpu_path),source_tree_sha256=gpu['source_tree_sha256'],source_file_count=gpu['source_file_count'],source_cases=428,game_draws=6,f2=6,changed_assets=changed,unchanged_assets=815,slots=384,maps=21,new_files=92,captures=rows,normal_player_verified=False)
def negative_checks(candidate,recipe,base,gpu_path,coverage):
    original=read(gpu_path)
    mutants=[
      ('missing-case',lambda d:d['cases'].pop(0)),
      ('wrong-catalog',lambda d:d.update(catalog_sha256='0'*64)),
      ('wrong-detail',lambda d:d.update(detail_manifest_sha256='0'*64)),
      ('wrong-code',lambda d:d.update(source_tree_sha256='0'*64)),
      ('wrong-tolerance',lambda d:d['cases'][0].update(rgba_tolerance=255)),
      ('missing-f2',lambda d:next(v for v in d['cases'] if v['game_draw']).update(f2=False)),
      ('false-save',lambda d:d['maps'][0].update(save_bytes=0)),
      ('missing-map',lambda d:d['maps'].pop()),
      ('duplicate-source',lambda d:d['cases'].__setitem__(0,copy.deepcopy(d['cases'][1]))),
      ('wrong-png-sha',lambda d:next(v for v in d['cases'] if v['game_draw']).update(sha256='0'*64))]
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='.painted-negatives-',dir=gpu_path.parent.parent) as directory:
        folder=Path(directory)
        for v in original['cases']:
            if v.get('file'):(folder/v['file']).symlink_to((gpu_path.parent/v['file']).resolve())
        path=folder/'receipt.json'
        for name,mutate in mutants:
            d=copy.deepcopy(original);mutate(d);path.write_text(json.dumps(d))
            try:verify(candidate,recipe,base,path,coverage)
            except (ValueError,FileNotFoundError):rejected.append(name)
            else:raise ValueError('negative admitted: '+name)
        d=copy.deepcopy(original);v=next(v for v in d['cases'] if v['game_draw'])
        pic=folder/'negative-black.png';Image.new('RGBA',(1920,1200),(0,0,0,255)).save(pic)
        v.update(file=pic.name,sha256=sha(pic));path.write_text(json.dumps(d))
        try:verify(candidate,recipe,base,path,coverage)
        except ValueError:rejected.append('actual-black-png')
        else:raise ValueError('actual black PNG admitted')
        bad=copy.deepcopy(read(candidate/'catalog.json'));bad['assets'][0]['source_refs']=['wrong']
        try:structural(read(base),bad,read(recipe),{v['old_file']:v for v in read(candidate/'manifest.json')['frames']})
        except ValueError:rejected.append('unrelated-asset-field')
        else:raise ValueError('unrelated field admitted')
    return rejected
def verify_normal_ground(path,candidate,binary):
    from verify_fd2_hd1_chibi_party import verify_normal
    d=read(path);folder=path.parent
    verify_normal(d,folder,candidate,binary)
    saved=folder/'user-data/fd2_re/fd2_save.json'
    reject(sha(saved)==d['saved_sha256'],'actual normal save changed')
    frames={t['frame']:v for v in read(candidate/'manifest.json')['frames'] for t in v['targets'] if t['map']==0}
    selected=[('ares-chibi-hd-hd.png',d['styles'][0]['state']),
              ('ares-chibi-hd-original.png',d['styles'][1]['state']),
              ('ares-chibi-hd-hd-restored.png',d['styles'][4]['state']),
              ('ares-chibi-walk-cancelled.png',d['returned']),
              ('ares-chibi-cold-title-load.png',d['cold'])]
    from PIL import ImageChops
    checked=[]
    for name,state in selected:
        reject(state.get('hd_world') is True,'normal HD terrain missing')
        image=Image.open(folder/name).convert('RGBA');count=0;identities=set()
        for tile in state['hd_world_terrain']:
            rect=tile['rect'];frame=tile['frame'];v=frames.get(frame)
            # The unchanged first-chapter source has an unobstructed top row.
            # Fixed geometry avoids accepting a different convenient sample.
            if not v or rect[1]!=24 or rect[3]!=168 or rect[0]<0 or rect[2]>1920:continue
            expected=Image.open(candidate/'pack'/v['file']).convert('RGBA')
            reject(hashlib.sha256(expected.tobytes()).hexdigest()==tile['rgba_sha256'],'normal terrain source pixels changed')
            diff=ImageChops.difference(image.crop(tuple(rect)),expected)
            reject(all(high<=2 for low,high in diff.getextrema()),'actual normal terrain PNG mismatch')
            count+=1;identities.add(frame)
        reject(count>=8 and len(identities)>=3,'normal terrain sample incomplete')
        checked.append(dict(file=name,sha256=sha(folder/name),tiles=count,frames=sorted(identities)))
    return dict(status='PASS-NORMAL-HD1-PAINTED-GROUND',report_sha256=sha(path),binary_sha256=sha(binary),
                actual_save_sha256=sha(saved),pngs=10,terrain_pngs=5,terrain_tiles=sum(v['tiles'] for v in checked),
                rgba_tolerance=2,state_injection=False,checks=checked,
                scope='普通首章LOAD、四組呈現、合法一格走行取消、F5及新程序LOAD；固定頂列新版圖源與實際PNG核對，非整章或原版parity。')

def register(candidate,recipe,base,gpu,coverage,formal,destination,guards,normal,binary):
    result=verify(candidate,recipe,base,gpu,coverage)
    previous=read(guards)
    reject(previous['status']=='PASS-HD1-PAINTED-GROUND-INDEPENDENT' and previous['tool_sha256']==sha(Path(__file__)),'independent guards source changed')
    reject(set(previous.get('rejected_mutants',[]))=={'missing-case','wrong-catalog','wrong-detail','wrong-code','wrong-tolerance','missing-f2','false-save','missing-map','duplicate-source','wrong-png-sha','actual-black-png','unrelated-asset-field'},'independent guards incomplete')
    reject(all(previous[k]==v for k,v in result.items()),'guards do not bind this candidate/GPU')
    result['normal']=verify_normal_ground(normal,candidate,binary)
    reject(sha(formal)==sha(base),'formal catalog changed')
    proof=read(candidate/'manifest.json')
    files=[*proof['masters'],*proof['frames']]
    reject(len(files)==92,'new file count')
    reject(destination.is_dir() and destination.stat().st_uid==formal.stat().st_uid==1000,'owned output required')
    for v in files:
        p=destination/v['file'];reject(not p.exists() or sha(p)==v['sha256'],'different existing versioned file')
    for v in files:
        p=destination/v['file']
        if not p.exists():shutil.copyfile(candidate/'pack'/v['file'],p)
        reject(p.stat().st_uid==1000 and sha(p)==v['sha256'],'published file differs')
    with tempfile.NamedTemporaryFile(dir=formal.parent,delete=False) as stream:
        temp=Path(stream.name);stream.write((candidate/'catalog.json').read_bytes())
    temp.chmod(formal.stat().st_mode&0o777);temp.replace(formal)
    result.update(status='REGISTERED-HD1-PAINTED-GROUND',formal_catalog_sha256=sha(formal))
    return result
def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('candidate','recipe','base','gpu','coverage','out'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--self-test',action='store_true');p.add_argument('--register',action='store_true')
    p.add_argument('--formal',type=Path);p.add_argument('--destination',type=Path);p.add_argument('--guards',type=Path)
    p.add_argument('--normal',type=Path);p.add_argument('--binary',type=Path)
    a=p.parse_args();reject(not a.out.exists(),'fresh report required')
    if a.register:
        reject(all(v is not None for v in (a.formal,a.destination,a.guards,a.normal,a.binary)),'registration paths and normal receipt required')
        result=register(a.candidate,a.recipe,a.base,a.gpu,a.coverage,a.formal,a.destination,a.guards,a.normal,a.binary)
    else:
        result=verify(a.candidate,a.recipe,a.base,a.gpu,a.coverage)
        if a.normal is not None:
            reject(a.binary is not None,'normal binary required')
            result['normal']=verify_normal_ground(a.normal,a.candidate,a.binary)
    if a.self_test:result['rejected_mutants']=negative_checks(a.candidate,a.recipe,a.base,a.gpu,a.coverage)
    result['tool_sha256']=sha(Path(__file__))
    a.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:v for k,v in result.items() if k!='captures'},ensure_ascii=False))
if __name__=='__main__':main()
