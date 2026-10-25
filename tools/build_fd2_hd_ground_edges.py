#!/usr/bin/env python3
"""以原生索引作藝術遮罩，將兩份新手繪材質編成既有稀疏地形。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile

from PIL import Image, ImageFilter
from build_fd2_hd_ground import make_tile
from verify_fd2_terrain_sources import verify, check_binding, sha
from validate_modern_theme_catalog import validate as validate_catalog


def ground_profile(recipe):
    if recipe.get('kind') == 'fd2_grass_earth_hd_v1':
        return 14, 141, 11, False
    if recipe.get('kind') == 'fd2_ground_bases_hd_v1':
        return 5, 52, 18, True
    if recipe.get('kind') == 'fd2_shore_hd_v1':
        return 33, 41, 9, False
    raise ValueError('交界配方契約不符')


def validate_sources(pack, manifest, recipe):
    count, slots, maps, bases = ground_profile(recipe)
    shore=recipe['kind']=='fd2_shore_hd_v1'
    if (recipe.get('schema_version'),recipe.get('size'))!=(1,144):
        raise ValueError('交界配方契約不符')
    groups=recipe['groups'];classes=recipe['index_groups']
    if len(groups)!=count or len({tuple(g['anchor']) for g in groups})!=count:
        raise ValueError('交界來源組數或身分不符')
    if bases and {tuple(g['anchor']) for g in groups}!={(0,f) for f in (52,53,160,164,165)}:
        raise ValueError('底材來源身分不符')
    if shore and {tuple(g['anchor']) for g in groups}!={(0,f) for f in [49,50,*range(64,80),96,97,98,99,100,101,102,105,106,108,109,110,111,113,114]}:
        raise ValueError('水岸來源身分不符')
    keys=('grass','sand','water') if shore else ('grass','earth')
    values=[v for k in keys for v in classes[k]]
    if (any(not classes[k] for k in keys) or any(type(v) is not int or not 0<=v<=255 for v in values) or
            len(set(values))!=len(values)):
        raise ValueError('藝術索引分組重複或越界')
    seen=set();out=[]
    for group in groups:
        pair=group.get('materials',list(keys))
        if shore and (not 1<=len(pair)<=2 or len(set(pair))!=len(pair) or any(k not in keys for k in pair)):
            raise ValueError('水岸材質配對不符')
        allowed={v for k in pair for v in classes[k]}
        actual=[];binding=[];anchor_pixels=None
        for mp in manifest['maps']:
            for fr in mp['frames']:
                if fr['native_content_sha256']!=group['native_content_sha256'] or fr['frame'] not in mp['selected_terrain_frames']:continue
                identity=[mp['map'],fr['frame']]
                paths=[check_binding(pack,e) for e in fr['input_layers']]
                raw,opaque,remap=[Image.open(p).tobytes() for p in paths]
                if (len(raw)!=576 or len(opaque)!=576 or len(remap)!=576 or not all(opaque) or any(remap)
                        or not set(raw)<=allowed):
                    raise ValueError('交界原生圖層或藝術索引不符')
                content=raw+bytes(bool(v) for v in opaque)+bytes(bool(v) for v in remap)
                if hashlib.sha256(content).hexdigest()!=group['native_content_sha256']:
                    raise ValueError('交界原生內容不同')
                if tuple(identity) in seen:raise ValueError('交界槽位重複')
                seen.add(tuple(identity));actual.append(identity);binding.append({'map':mp['map'],'frame':fr['frame'],'input_layers':fr['input_layers']})
                if identity==group['anchor']:anchor_pixels=raw
        if actual!=group['targets'] or anchor_pixels is None:
            raise ValueError('配方與實際交界consumer不同')
        if Path(group['file']).name!=group['file'] or not group['file'].endswith('.png'):
            raise ValueError('交界檔名越界')
        out.append({'group':group,'pixels':anchor_pixels,'bindings':binding})
    if len(seen)!=slots or len({m for m,f in seen})!=maps:
        raise ValueError('交界完整槽位缺少')
    return out


def composite_materials(raw,grass,earth,recipe):
    if recipe['mask_blur_hd_pixels']!=2 or recipe['exact_outer_edge_hd_pixels']!=1:
        raise ValueError('交界遮罩校準範圍不符')
    soil=set(recipe['index_groups']['earth'])
    mask=Image.frombytes('L',(24,24),bytes(255 if v in soil else 0 for v in raw))
    nearest=mask.resize((144,144),Image.Resampling.NEAREST)
    smooth=mask.resize((144,144),Image.Resampling.BICUBIC).filter(ImageFilter.GaussianBlur(2))
    for i in range(144):
        for xy in [(i,0),(i,143),(0,i),(143,i)]:smooth.putpixel(xy,nearest.getpixel(xy))
    result=Image.composite(earth,grass,smooth)
    if result.getextrema()[3]!=(255,255):raise ValueError('交界alpha不同於原生opaque')
    return result,smooth


def build(pack,maps,sources,base_catalog,base_pack,earth_master,recipe_path,out,shore_masters=None):
    if os.path.lexists(out) or not out.parent.is_dir():raise ValueError('輸出須為既有父目錄下的新目錄')
    r=json.loads(recipe_path.read_text())
    count,slots,map_count,bases=ground_profile(r)
    shore=r['kind']=='fd2_shore_hd_v1'
    if sha(base_catalog)!=r['base_catalog_sha256'] or sha(sources/'manifest.json')!=r['source_manifest_sha256']:
        raise ValueError('交界基底綁定不同')
    grass_master=base_pack/r['grass_master_file']
    masters={'grass':grass_master,**(shore_masters or {})} if shore else {'grass':grass_master,'earth':earth_master}
    keys=['grass','sand','water'] if shore else ['grass','earth']
    if set(masters)!=set(keys) or any(p is None or sha(p)!=r[k+'_master_sha256'] for k,p in masters.items()):
        raise ValueError('交界母稿綁定不同')
    source_report=verify(pack,maps,sources);manifest=json.loads((sources/'manifest.json').read_text())
    checked=validate_sources(pack,manifest,r)
    tiles={};edges={}
    for k,p in masters.items():tiles[k],edges[k]=make_tile(p,144,r['edge_blend'])
    catalog=json.loads(base_catalog.read_text());names=set();hashes={}
    for a in catalog['assets']:
        if 'file' in a:names.add(a['file']);hashes[a['file']]=a['sha256']
        if 'master_file' in a:names.add(a['master_file'])
        for key,hkey in [('files','frame_sha256'),('source_files','source_sha256')]:
            if len(a.get(key,[]))!=len(a.get(hkey,[])):raise ValueError('基底圖源綁定缺少')
            for name,h in zip(a.get(key,[]),a.get(hkey,[])):
                if name in hashes and hashes[name]!=h:raise ValueError('基底圖源綁定衝突')
                names.add(name);hashes[name]=h
    for name in names:
        if Path(name).name!=name or not (base_pack/name).is_file() or name in hashes and sha(base_pack/name)!=hashes[name]:
            raise ValueError('基底圖源缺少或不同：'+name)
    targets={tuple(t) for g in r['groups'] for t in g['targets']};old={(a['map_id'],f) for a in catalog['assets'] if a['role']=='native_terrain_hd_set' for f in a['frame_ids']}
    if old&targets:raise ValueError('拒絕覆寫既有地形槽')
    added_materials=['sand','water'] if shore else [] if bases else ['earth']
    newnames={g['file'] for g in r['groups']} | {r[k+'_master_file'] for k in added_materials}
    if len(newnames)!=count+len(added_materials) or names&newnames or any(Path(n).name!=n for n in newnames):raise ValueError('新交界圖源身分衝突')
    if bases:
        masters=[a for a in catalog['assets'] if a['asset_id']==r['earth_master_asset_id']]
        if len(masters)!=1 or masters[0].get('file')!=r['earth_master_file'] or masters[0].get('sha256')!=sha(earth_master):
            raise ValueError('底材既有泥土母稿不符')
    temporary=Path(tempfile.mkdtemp(prefix='.ground-edges-',dir=out.parent))
    try:
        assets=temporary/'pack';assets.mkdir()
        for name in names:(assets/name).symlink_to((base_pack/name).resolve())
        master_ids={'grass':'modern.terrain.grass_olive.master'}
        if bases:master_ids['earth']=r['earth_master_asset_id']
        additions=[]
        for k in added_materials:
            shutil.copyfile(masters[k],assets/r[k+'_master_file']);master_ids[k]=r['asset_id']+'.'+k+'.master'
            size=Image.open(masters[k]).size
            additions.append({'asset_id':master_ids[k],'role':'terrain_hd_master','status':'concept','file':r[k+'_master_file'],'width':size[0],'height':size[1],'sha256':sha(masters[k]),'source_refs':r['source_refs']})
        entries=[];by_map={};preview=Image.new('RGBA',(7*144,max(2,(count+6)//7)*144));edge_data=[]
        for i,item in enumerate(checked):
            group=item['group'];pair=group.get('materials',['grass','earth']);raw=item['pixels']
            if len(pair)==1:picture=tiles[pair[0]].copy();mask=Image.new('L',(144,144))
            else:
                local={**r,'index_groups':{'grass':r['index_groups'][pair[0]],'earth':r['index_groups'][pair[1]]}}
                picture,mask=composite_materials(raw,tiles[pair[0]],tiles[pair[1]],local)
            name=group['file'];picture.save(assets/name);preview.paste(picture,(i%7*144,i//7*144))
            labels={v:k for k,values in r['index_groups'].items() for v in values}
            edge_data.append((picture,[labels[v] for v in raw]))
            entries.append({'anchor':group['anchor'],'file':name,'sha256':sha(assets/name),'mask_rgba_sha256':hashlib.sha256(mask.tobytes()).hexdigest(),'native_content_sha256':group['native_content_sha256'],'targets':item['bindings']})
            for map_id,frame in group['targets']:by_map.setdefault(map_id,[]).append((frame,name,sha(assets/name)))
        same_checks=0
        for pa,ma in edge_data:
            for pb,mb in edge_data:
                for i in range(144):
                    for a,b in [((143,i),(0,i)),((i,143),(i,0))]:
                        if ma[a[1]//6*24+a[0]//6]==mb[b[1]//6*24+b[0]//6]:
                            if pa.getpixel(a)!=pb.getpixel(b):raise ValueError('相同藝術分類的交界RGB不連續')
                            same_checks+=1
        for map_id,items in sorted(by_map.items()):
            refs=[master_ids[k] for k in (['sand','water','grass'] if shore else ['earth','grass'])]
            items.sort();additions.append({'asset_id':r['asset_id']+f'.map{map_id:02d}.native_hd','role':'native_terrain_hd_set','status':'runtime_candidate','consumer_contract':'fdshap_native_hd_sparse_v1','map_id':map_id,'width':144,'height':144,'frame_count':len(items),'frame_ids':[x[0] for x in items],'files':[x[1] for x in items],'frame_sha256':[x[2] for x in items],'semantics':['terrain']*len(items),'alpha_contract':'native_mask_nearest_6','cycle_policy':'native_selector','source_refs':['asset:'+v for v in refs]+r['source_refs']})
        if {a['asset_id'] for a in catalog['assets']}&{a['asset_id'] for a in additions}:raise ValueError('交界清冊身分已存在')
        candidate=copy.deepcopy(catalog);candidate['assets'].extend(additions)
        (temporary/'catalog.json').write_text(json.dumps(candidate,ensure_ascii=False,indent=2)+'\n')
        validate_catalog(False,temporary/'catalog.json')
        preview.save(temporary/'preview.png')
        report={'status':'PASS-SHORE-CANDIDATE' if shore else 'PASS-GROUND-BASES-CANDIDATE' if bases else 'PASS-GROUND-EDGES-CANDIDATE','runtime_integrated':False,'base_catalog_sha256':sha(base_catalog),'recipe_sha256':sha(recipe_path),'source_manifest_sha256':sha(sources/'manifest.json'),'source_verification':source_report,'source_groups':len(entries),'target_slots':len(targets),'maps':sorted(by_map),'frames':entries,'grass_master':{'file':r['grass_master_file'],'sha256':sha(grass_master)},'edge_calibration':{**edges,'equal_material_edge_pixels':same_checks,'outer_hd_pixels':1,'mask_blur_hd_pixels':2},'catalog_assets':len(candidate['assets']),'base_assets_unchanged':len(catalog['assets']),'scope':'新手繪材質與明示藝術遮罩；非玩法分類、原版RGB或正常玩家證據。'}
        if shore:report['masters']=[{'file':r[k+'_master_file'],'sha256':sha(masters[k])} for k in added_materials]
        else:report['earth_master']={'file':r['earth_master_file'],'sha256':sha(earth_master)}
        (temporary/'manifest.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n');temporary.rename(out)
    except BaseException:shutil.rmtree(temporary,ignore_errors=True);raise
    return report


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for n in ['original-pack','maps','sources','base-catalog','base-pack','recipe','out']:p.add_argument('--'+n,type=Path,required=True)
    for n in ['earth-master','sand-master','water-master']:p.add_argument('--'+n,type=Path)
    a=p.parse_args();shore_masters={'sand':a.sand_master,'water':a.water_master} if a.sand_master or a.water_master else None
    r=build(a.original_pack,a.maps,a.sources,a.base_catalog,a.base_pack,a.earth_master,a.recipe,a.out,shore_masters)
    print(json.dumps({k:r[k] for k in ['status','source_groups','target_slots','maps','catalog_assets']}))


if __name__=='__main__':main()
