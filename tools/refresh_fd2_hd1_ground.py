#!/usr/bin/env python3
"""以定版手繪底材刷新既有HD1地形；原生來源、集合身分與舊檔保持。"""
import argparse,copy,hashlib,json,os,shutil,tempfile
from pathlib import Path
from PIL import Image
from build_fd2_hd_ground import make_tile,ground_sources
from build_fd2_hd_ground_edges import validate_sources,composite_materials,composite_relief
from verify_fd2_terrain_sources import verify,sha
from validate_modern_theme_catalog import validate as validate_catalog

ROOT=Path(__file__).resolve().parent.parent
KIND='fd2_hd1_painted_ground_refresh_v1'
PROFILES=('ground-olive-hd.recipe.json','ground-grass-earth-hd.recipe.json','ground-bases-hd.recipe.json','shore-hd.recipe.json','relief-hd.recipe.json')
def read(p):return json.loads(Path(p).read_text())
def reject(ok,why):
    if not ok:raise ValueError(why)
def named(s):return isinstance(s,str) and Path(s).name==s and s.endswith('.png')
def names(catalog):
    out={}
    for a in catalog['assets']:
        if 'file' in a:
            reject(a['file'] not in out or out[a['file']]==a['sha256'],'conflicting file')
            out[a['file']]=a['sha256']
        for f,h in zip(a.get('files',[]),a.get('frame_sha256',[])):
            reject(f not in out or out[f]==h,'conflicting frame');out[f]=h
        for f,h in zip(a.get('source_files',[]),a.get('source_sha256',[])):
            reject(f not in out or out[f]==h,'conflicting source');out[f]=h
    reject(all(named(f) or Path(f).name==f for f in out),'unsafe base filename')
    return out
def structural(base,new,recipe,replacements):
    reject(len(base['assets'])==len(new['assets'])==883,'asset count changed')
    by_id={a['asset_id']:a for a in new['assets']}
    reject(len(by_id)==883,'duplicate asset identity')
    masters={v['asset_id']:v for v in recipe['materials'].values()}
    changed=0
    for old in base['assets']:
        got=by_id.get(old['asset_id']);reject(got is not None,'missing asset')
        want=copy.deepcopy(old)
        if old['asset_id'] in masters:
            m=masters[old['asset_id']]
            want.update(file=m['file'],sha256=m['sha256'],width=m['size'][0],height=m['size'][1])
        if old['role']=='native_terrain_hd_set':
            for i,f in enumerate(want['files']):
                if f in replacements:
                    want['files'][i],want['frame_sha256'][i]=replacements[f]['file'],replacements[f]['sha256']
        reject(got==want,'unrelated asset field changed: '+old['asset_id'])
        changed+=got!=old
    reject(changed==68,'expected five masters and63 terrain sets')
    return changed
def prepare(recipe_path,catalog_path,pack,original,sources,maps,masters,out):
    reject(Path('/.dockerenv').exists(),'Docker required')
    reject(not os.path.lexists(out) and out.parent.is_dir(),'fresh output required')
    r,base,source=read(recipe_path),read(catalog_path),read(sources/'manifest.json')
    reject(r['kind']==KIND and r['schema_version']==1,'unsupported recipe')
    reject(sha(catalog_path)==r['base_catalog_sha256'],'base catalog changed')
    reject(sha(sources/'manifest.json')==r['source_manifest_sha256'],'native manifest changed')
    reject(sha(ROOT/r['style']['file'])==r['style']['sha256'],'approved style changed')
    reject(tuple(p['file'] for p in r['profiles'])==PROFILES,'profile allow-list changed')
    profiles=[]
    for v in r['profiles']:
        p=ROOT/'remake/assets/themes/modern'/v['file']
        reject(sha(p)==v['sha256'],'source recipe changed')
        profiles.append(read(p))
    by_id={a['asset_id']:a for a in base['assets']}
    tiles,edge_data={},{}
    reject(set(r['materials'])=={'grass','earth','sand','water','stone'},'material allow-list changed')
    for key,m in r['materials'].items():
        old=by_id[m['asset_id']]
        reject(old['role']=='terrain_hd_master' and old['file']==m['old_file'] and old['sha256']==m['old_sha256'],'master identity changed')
        reject(named(m['file']) and m['file']!=m['old_file'] and sha(masters/m['file'])==m['sha256'],'new master binding changed')
        with Image.open(masters/m['file']) as im:reject(list(im.size)==m['size'],'master dimensions changed')
        tiles[key],edge_data[key]=make_tile(masters/m['file'],144,8)
    baseline=names(base)
    for f,h in baseline.items():reject((pack/f).is_file() and sha(pack/f)==h,'base file changed: '+f)
    reject(not set(m['file'] for m in r['materials'].values())&set(baseline),'new master collides with old file')
    source_proof=verify(original,maps,sources)
    work=Path(tempfile.mkdtemp(prefix='.painted-ground-',dir=out.parent))
    try:
        assets=work/'pack';assets.mkdir()
        for f in baseline:(assets/f).symlink_to((pack/f).resolve())
        for m in r['materials'].values():shutil.copyfile(masters/m['file'],assets/m['file'])
        entries=[];seams=[]
        grass=profiles[0]
        pure=ground_sources(original,source,grass)
        batches=[(grass,[dict(group=dict(anchor=grass['anchor'],file=grass['tile_file'],native_content_sha256=grass['native_content_sha256']),bindings=pure,pixels=None)])]
        batches.extend((p,validate_sources(original,source,p)) for p in profiles[1:])
        for profile,rows in batches:
            relief=profile['kind']=='fd2_relief_hd_v1'
            labels={v:k for k,vs in profile.get('index_groups',{}).items() for v in vs}
            for item in rows:
                group,raw=item['group'],item['pixels']
                if raw is None:
                    picture=tiles['grass'].copy();mask=None;classes=[('grass',255)]*576
                elif relief:
                    picture,mask=composite_relief(raw,{k:tiles[k] for k in ('grass','earth','stone')},profile)
                    classes=[(labels[v],profile['index_shades'][str(v)]) for v in raw]
                else:
                    pair=group.get('materials',['grass','earth'])
                    if len(pair)==1:picture=tiles[pair[0]].copy();mask=None
                    else:
                        local={**profile,'index_groups':dict(grass=profile['index_groups'][pair[0]],earth=profile['index_groups'][pair[1]])}
                        picture,mask=composite_materials(raw,tiles[pair[0]],tiles[pair[1]],local)
                    classes=[(labels[v],255) for v in raw]
                old=group['file'];new=old.replace('-hd-v1.png','-hd1-painted-v2.png')
                reject(old in baseline and new!=old and new not in baseline and named(new),'frame identity or version differs')
                picture.save(assets/new)
                reject(picture.size==(144,144) and picture.getextrema()[3]==(255,255),'opaque frame contract')
                entries.append(dict(anchor=group['anchor'],old_file=old,old_sha256=baseline[old],file=new,sha256=sha(assets/new),native_content_sha256=group['native_content_sha256'],targets=item['bindings']))
                seams.append((picture,classes))
        reject(len(entries)==87 and len({v['old_file'] for v in entries})==87,'frame allow-list incomplete')
        equal=0
        for a,ca in seams:
            for b,cb in seams:
                for i in range(144):
                    for pa,pb in [((143,i),(0,i)),((i,143),(i,0))]:
                        if ca[pa[1]//6*24+pa[0]//6]==cb[pb[1]//6*24+pb[0]//6]:
                            reject(a.getpixel(pa)==b.getpixel(pb),'matching material seam differs');equal+=1
        replacements={v['old_file']:v for v in entries};candidate=copy.deepcopy(base)
        for a in candidate['assets']:
            for m in r['materials'].values():
                if a['asset_id']==m['asset_id']:a.update(file=m['file'],sha256=m['sha256'],width=m['size'][0],height=m['size'][1])
            if a['role']=='native_terrain_hd_set':
                for i,f in enumerate(a['files']):
                    if f in replacements:a['files'][i],a['frame_sha256'][i]=replacements[f]['file'],replacements[f]['sha256']
        structural(base,candidate,r,replacements)
        required={(mp['map'],v['frame']):v for mp in read(ROOT/'docs/data/fd2-hd1-terrain-coverage.json')['maps'] for v in mp['required'] if v['role']=='terrain'}
        bindings=[]
        for row in entries:
            for target in row['targets']:
                pair=target['map'],target['frame'];v=required[pair]
                reject(v['native_content_sha256']==row['native_content_sha256'] and v['source_pixels']==576 and v['remap_pixels']==0,'coverage/native source differs')
                bindings.append(dict(map=pair[0],**{k:v[k] for k in ('frame','role','source_pixels','remap_pixels','selector')},file=row['file'],sha256=row['sha256'],input_layers=target['input_layers']))
        expected={(v['map'],v['frame']) for v in bindings}
        actual={(a['map_id'],f) for a in base['assets'] if a['role']=='native_terrain_hd_set' for f,name in zip(a['frame_ids'],a['files']) if name in replacements}
        reject(expected==actual and len(expected)==384 and len({m for m,f in expected})==21,'catalog reference closure differs')
        anchors={tuple(v['anchor']) for v in entries}
        gpu=[v for v in bindings if (v['map'],v['frame']) in anchors]
        for m in sorted({m for m,f in expected}-{v['map'] for v in gpu}):gpu.append(next(v for v in bindings if v['map']==m))
        reject(len(gpu)==107,'GPU source coverage differs')
        (work/'catalog.json').write_text(json.dumps(candidate,ensure_ascii=False,indent=2)+'\n')
        validate_catalog(False,work/'catalog.json')
        detail=dict(kind=KIND,catalog_sha256=sha(work/'catalog.json'),bindings=gpu)
        (work/'gpu-detail.json').write_text(json.dumps(detail,ensure_ascii=False,indent=2)+'\n')
        preview=Image.new('RGBA',(5*144,2*144))
        for i,(key,m) in enumerate(r['materials'].items()):
            before,_=make_tile(pack/m['old_file'],144,8);preview.paste(before,(i*144,0));preview.paste(tiles[key],(i*144,144))
        preview.save(work/'materials-preview.png')
        report=dict(status='PASS-HD1-PAINTED-GROUND-CANDIDATE',kind=KIND,base_catalog_sha256=sha(catalog_path),catalog_sha256=sha(work/'catalog.json'),recipe_sha256=sha(recipe_path),source_manifest_sha256=sha(sources/'manifest.json'),source_verification=source_proof,frames=entries,bindings=bindings,masters=list(r['materials'].values()),gpu_detail_sha256=sha(work/'gpu-detail.json'),source_groups=87,target_slots=384,maps=21,changed_assets=68,unchanged_assets=815,catalog_assets=883,new_files=92,material_edges=edge_data,equal_material_edge_pixels=equal,runtime_integrated=False)
        (work/'manifest.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
        work.rename(out)
    except BaseException:shutil.rmtree(work,ignore_errors=True);raise
    return report
def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('recipe','catalog','pack','original','sources','maps','masters','out'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();r=prepare(a.recipe,a.catalog,a.pack,a.original,a.sources,a.maps,a.masters,a.out)
    print(json.dumps({k:r[k] for k in ('status','source_groups','target_slots','maps','changed_assets','new_files')}))
if __name__=='__main__':main()
