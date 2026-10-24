#!/usr/bin/env python3
"""在Pillow Docker以固定原圖／母稿編製天空之鑰候選，不修改正式清冊。"""
import argparse
from collections import Counter
import hashlib
import json
import math
import os
from pathlib import Path
import shutil

from PIL import Image

ROOT = Path(__file__).resolve().parents[1]
COLOURS = {'white': range(185, 192), 'darkblue': range(88, 96),
           'lightblue': range(200, 208), 'green': range(152, 160), 'red': range(32, 40)}
# Reviewed art-space anchors, in the unmodified 1254px front master.
ART_ANCHORS = {'white': (279, 505), 'darkblue': (394, 876),
               'lightblue': (875, 876), 'green': (627, 249), 'red': (970, 505)}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json(path, data):
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + '\n')


def components(pixels, width, height, allowed):
    remaining = {i for i, value in enumerate(pixels) if allowed(i, value)}
    groups = []
    while remaining:
        start = min(remaining); remaining.remove(start); todo = [start]; found = []
        while todo:
            i = todo.pop(); found.append(i); x, y = i % width, i // width
            for yy in range(max(0, y-1), min(height, y+2)):
                for xx in range(max(0, x-1), min(width, x+2)):
                    n = yy*width+xx
                    if n in remaining:
                        remaining.remove(n); todo.append(n)
        xs = [i % width for i in found]; ys = [i // width for i in found]
        groups.append({'n': len(found), 'center': [sum(xs)/len(xs)+.5, sum(ys)/len(ys)+.5],
                       'bbox': [min(xs), min(ys), max(xs)+1, max(ys)+1], 'indices': found})
    return sorted(groups, key=lambda c: (-c['n'], c['bbox']))


def bounds(points):
    return [min(x for x,y in points), min(y for x,y in points),
            max(x for x,y in points)+1, max(y for x,y in points)+1] if points else None


def solve(matrix, rhs):
    rows = [list(row)+[b] for row,b in zip(matrix,rhs)]
    for k in range(len(rows)):
        at = max(range(k,len(rows)), key=lambda r: abs(rows[r][k]))
        rows[k],rows[at]=rows[at],rows[k]
        if abs(rows[k][k]) < 1e-10: raise ValueError('退化的美術錨點')
        v=rows[k][k];rows[k]=[x/v for x in rows[k]]
        for r in range(len(rows)):
            if r!=k:
                v=rows[r][k];rows[r]=[x-v*y for x,y in zip(rows[r],rows[k])]
    return [r[-1] for r in rows]


def fit_affine(anchors):
    rows=[[(ART_ANCHORS[k][0]-627)/600,(ART_ANCHORS[k][1]-627)/600,1] for k in anchors]
    matrix=[[sum(r[i]*r[j] for r in rows) for j in range(3)] for i in range(3)]
    result=[]
    for channel in range(2):
        result+=solve(matrix,[sum(r[k]*v[channel] for r,v in zip(rows,anchors.values())) for k in range(3)])
    return result


def regression(xs,ys):
    n=len(xs);sx=sum(xs);sy=sum(ys);sxx=sum(x*x for x in xs);sxy=sum(x*y for x,y in zip(xs,ys))
    a=(n*sxy-sx*sy)/(n*sxx-sx*sx)
    return a,(sy-a*sx)/n


def prepare_ani(original):
    folder=original/'animations/ANI_000';doc=json.loads((folder/'animation.json').read_text())
    if doc['frame_count']!=96 or doc['source']['sha256']!='be909c71d0f1121b6632ae931d978e990f6d54c830f4e0509cd6862187c4d963':
        raise ValueError('固定ANI0不同')
    frames=[]
    for i,row in enumerate(doc['frames']):
        im=Image.open(folder/row['frame']);im.load()
        dac=json.loads((folder/row['palette']).read_text())['dac_6bit_components']
        if im.mode!='P' or im.size!=(320,200) or im.getpalette()!=[(v<<2)|(v>>4) for v in dac]:
            raise ValueError('ANI索引圖或實際DAC不一致')
        pixels=im.tobytes();anchors={};all_anchors={}
        for name,indices in COLOURS.items():
            cs=components(pixels,320,200,lambda n,v:v in indices)
            if not cs:continue
            c=cs[0];b=c['bbox'];all_anchors[name]={k:c[k] for k in ['n','center','bbox']}
            # The smallest complete gemstone in the last frame still has
            # 68 pixels. Tiny blue glints on a white gem are not anchors.
            if c['n']>=40 and b[0]>0 and b[2]<320 and b[1]>20 and b[3]<163:
                anchors[name]=c['center']
        frame={'index':i,'pixel_sha256':hashlib.sha256(pixels).hexdigest(),
               'dac_sha256':hashlib.sha256(bytes(dac)).hexdigest(),'source_colours':sorted(set(pixels)),'anchors':all_anchors,
               'usable_anchors':anchors,'blank':im.convert('RGB').getbbox() is None}
        frames.append(frame)
    ref=next(f for f in frames if len(f['usable_anchors'])>=3)
    ref_matrix=fit_affine(ref['usable_anchors']);ref_points=ref['usable_anchors']
    ref_vector=[ref_points['lightblue'][k]-ref_points['white'][k] for k in range(2)]
    scales=[]
    for f in frames:
        a=f['usable_anchors']
        if 28<=f['index']<=48 and 'white' in a and 'lightblue' in a:
            length=math.dist(a['white'],a['lightblue'])/math.hypot(*ref_vector)
            scales.append((f['index'],length))
    # The early close-up lacks enough visible landmarks. This bounded inverse
    # zoom is an explicit art approximation; it is never native ABI evidence.
    models=[]
    for offset in range(-12,13):
        xs=[1/(i+offset) for i,v in scales];ys=[v for i,v in scales]
        k=sum(x*y for x,y in zip(xs,ys))/sum(x*x for x in xs)
        error=sum((k*x-y)**2 for x,y in zip(xs,ys))
        models.append((error,offset,k))
    _,offset,k=min(models)
    visible=[f for f in frames if 20<=f['index']<=45 and 'white' in f['usable_anchors']]
    white_models=[regression([1/f['index'] for f in visible],[f['usable_anchors']['white'][c] for f in visible]) for c in range(2)]
    for f in frames:
        if f['blank']:f['affine']=None;f['method']='native-black';continue
        anchors=f['usable_anchors'];i=f['index']
        if len(anchors)>=3:
            f['affine']=fit_affine(anchors);f['method']='native-gem-affine';continue
        scale=min(16,max(.1,k/(i+offset)));angle=0
        if 'white' in anchors and 'lightblue' in anchors:
            vec=[anchors['lightblue'][c]-anchors['white'][c] for c in range(2)]
            scale=math.hypot(*vec)/math.hypot(*ref_vector)
            angle=math.atan2(vec[1],vec[0])-math.atan2(ref_vector[1],ref_vector[0])
        co,si=math.cos(angle)*scale,math.sin(angle)*scale
        a,b,_,d,e,_=ref_matrix
        linear=[co*a-si*d,co*b-si*e,si*a+co*d,si*b+co*e]
        target=anchors.get('white',[a/i+b for a,b in white_models]);u,v=[(x-627)/600 for x in ART_ANCHORS['white']]
        f['affine']=[linear[0],linear[1],target[0]-linear[0]*u-linear[1]*v,
                     linear[2],linear[3],target[1]-linear[2]*u-linear[3]*v]
        f['method']='two-gem-affine-extension' if len(anchors)>=2 else 'bounded-closeup-art-approximation'
    return doc,frames,{'reference_frame':ref['index'],'inverse_zoom_offset':offset,'inverse_zoom_k':k,'early_max_scale':16,'white_models':white_models}


def warp_lock(master,affine):
    a,b,c,d,e,f=affine;det=a*e-b*d
    if abs(det)<1e-6:raise ValueError('金鎖投影退化')
    coeff=(100*e/det,-100*b/det,627+600*(-e*c+b*f)/det,
           -100*d/det,100*a/det,627+600*(d*c-a*f)/det)
    image=master.transform((1920,1200),Image.Transform.AFFINE,coeff,Image.Resampling.BICUBIC)
    result=Image.new('RGBA',(1920,1200),(0,0,0,255))
    # Original clip's content band is y20..162; no invented picture outside it.
    result.alpha_composite(image.crop((0,120,1920,978)),(0,120))
    return result


def crop_component(im,box):
    part=im.crop(box);alpha=part.getchannel('A');core=alpha.point(lambda v:255 if v>=16 else 0).getbbox()
    if core is None:raise ValueError('空白美術分量')
    return part.crop((max(0,core[0]-2),max(0,core[1]-2),min(part.width,core[2]+2),min(part.height,core[3]+2)))


def paste_scaled(canvas,picture,box,origin,white=False):
    x0,y0,x1,y1=box
    size=(max(1,round((x1-x0)*6)),max(1,round((y1-y0)*6)))
    picture=picture.resize(size,Image.Resampling.LANCZOS)
    if white:
        alpha=picture.getchannel('A');picture=Image.new('RGBA',size,'white');picture.putalpha(alpha)
    x,y=round((x0-origin[0])*6),round((y0-origin[1])*6)
    # Pillow's compositor clips the source to the native descriptor rectangle.
    canvas.alpha_composite(picture,(x,y))


def fd34_plan(folder):
    doc=json.loads((folder/'bank.json').read_text());rows=[]
    if len(doc['frames'])!=101 or doc['source']['raw_sha256']!='53f120fa4b1fab74c6b3998ec3ef8a9a2363461980ad38a7ffef2400e79b0c4d':raise ValueError('FDOTHER34版本不同')
    # The body-only opening frames and the body/key hold contain no lock.
    # Their literal gold pixels distinguish the coat from the separate prop.
    body_gold=set()
    for i in list(range(1,21))+list(range(60,70)):
        f=doc['frames'][i];im=Image.open(folder/f['frame']);mask=Image.open(folder/f['mask']).tobytes()
        for n,v in enumerate(im.tobytes()):
            x,y=f['x']+n%im.width,f['y']+n//im.width
            if mask[n] and 80<=v<=87 and 146<=x<180 and 94<=y<118:body_gold.add((x,y,v))
    for f in doc['frames']:
        im=Image.open(folder/f['frame']);mask=Image.open(folder/f['mask']);im.load();mask.load();pix=im.tobytes();m=mask.tobytes();w,h=im.size;x0,y0=f['x'],f['y']
        if im.mode!='P' or mask.mode!='L' or im.size!=(f['width'],f['height']) or mask.size!=im.size:raise ValueError('FDOTHER34形狀不同')
        position=lambda n:(x0+n%w,y0+n//w)
        outside=lambda n:not(148<=position(n)[0]<172 and 94<=position(n)[1]<118)
        allcs=components(pix,w,h,lambda n,v:bool(m[n]) and outside(n))
        coin=None;flash_coin=False
        top=[c for c in allcs if c['n']>=20 and y0+c['bbox'][3]<70 and c['bbox'][2]-c['bbox'][0]>=15]
        if top:
            c=max(top,key=lambda c:c['n']);coin=[x0+c['bbox'][0],y0+c['bbox'][1],x0+c['bbox'][2],y0+c['bbox'][3]]
            flash_coin=sum(pix[n]==253 for n in c['indices'])>c['n']*.5
        else:
            gold=[position(n) for n,v in enumerate(pix) if m[n] and 80<=v<=87 and (*position(n),v) not in body_gold]
            if f['index']>=26 and len(gold)>=12:
                b=bounds(gold)
                if b[2]-b[0]>=8 and b[3]-b[1]>=8:
                    cx,cy=(b[0]+b[2])/2,(b[1]+b[3])/2
                    diameter=min(22,max(16,max(b[2]-b[0],b[3]-b[1])+4))
                    coin=[cx-diameter/2,cy-diameter/2,cx+diameter/2,cy+diameter/2]
        orbs=[]
        for name,indices in COLOURS.items():
            cs=components(pix,w,h,lambda n,v:bool(m[n]) and v in indices and outside(n))
            if cs and cs[0]['n']>=8 and min(cs[0]['bbox'][2]-cs[0]['bbox'][0],cs[0]['bbox'][3]-cs[0]['bbox'][1])>=5:
                b=cs[0]['bbox'];box=[x0+b[0]-1,y0+b[1]-1,x0+b[2]+1,y0+b[3]+1]
                if coin and not(box[2]<=coin[0] or box[0]>=coin[2] or box[3]<=coin[1] or box[1]>=coin[3]):continue
                orbs.append({'kind':name,'bbox':box,'flash':False})
        for c in allcs:
            b=c['bbox'];box=[x0+b[0],y0+b[1],x0+b[2],y0+b[3]]
            if c['n']<30 or b[2]-b[0]<8 or sum(pix[n]==253 for n in c['indices'])<c['n']*.6:continue
            if coin and box[1]<coin[3] and box[3]>coin[1]:continue
            orbs.append({'kind':'white','bbox':box,'flash':True})
        excluded=[x['bbox'] for x in orbs]+([coin] if coin else [])
        key=[position(n) for n,v in enumerate(pix) if m[n] and 224<=v<=239 and not any(b[0]<=position(n)[0]<b[2] and b[1]<=position(n)[1]<b[3] for b in excluded)]
        keybox=bounds(key) if len(key)>=4 else None
        canvas=bytearray(64000);canvas_ff=bytearray([255])*64000
        for n,v in enumerate(pix):
            if m[n]:canvas[(y0+n//w)*320+x0+n%w]=v;canvas_ff[(y0+n//w)*320+x0+n%w]=v
        row={'index':f['index'],'x':x0,'y':y0,'width':w,'height':h,'native_canvas_sha256':hashlib.sha256(canvas).hexdigest(),
             'native_canvas_ff_sha256':hashlib.sha256(canvas_ff).hexdigest(),
             'native_pixels_sha256':hashlib.sha256(pix).hexdigest(),'native_mask_sha256':hashlib.sha256(m).hexdigest(),
             'source_colours':sorted({v for n,v in enumerate(pix) if m[n]}),'orbs':orbs,'coin':coin,'coin_flash':flash_coin,'key':keybox,'raised':keybox is not None}
        rows.append(row)
    return doc,rows


def descriptor(row):
    return {**{k:row[k] for k in ['x','y','width','height']},'delay_native':0,'raw_byte_4':0,'raw_byte_5':0,'raw_byte_7':0}


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ['original-pack','source','catalog','masters','out']:parser.add_argument('--'+name,type=Path,required=True)
    a=parser.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or not a.out.parent.is_dir() or a.out.parent.stat().st_uid!=os.getuid():raise ValueError('Docker內使用全新本人輸出')
    if sha(a.catalog)!='3d28672ef5f5b5803e91b7e72b244146dd656204aa42fe2ac47290bc591641cc':raise ValueError('766項基底不同')
    masters=['sky-key-lock-master-oct25-r1.png','sky-key-raised-master-oct25-r1.png','sky-key-components-master-oct25-r2.png']
    expected=['f6a043c426ce9e720268af7ba844a1ff4d4e6e1fae292ff3d0c0774aba1144c6','79c5792c771dd3dd0ac024cd8d6f26bd7eacc75a98fe2ff1bd8f83309cb33c0c','e87b715e79de0d775459f6a230a2fdcb7c2857bc06ea2648916e9c85562e2603']
    for name,digest in zip(masters,expected):
        if digest and sha(a.masters/name)!=digest:raise ValueError('母稿不同')
    inventory=json.loads((ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json').read_text())
    if sha(a.original_pack/'manifest.json')!=inventory['pack_manifest_sha256']:raise ValueError('原包manifest不同')
    manifest=json.loads((a.original_pack/'manifest.json').read_text());pinned={r['path']:r['sha256'] for r in manifest['assets']}
    for folder in [a.original_pack/'animations/ANI_000',a.original_pack/'animations/fdother_034_ch20_sky_key']:
        for f in sorted(folder.rglob('*')):
            if f.is_file() and (str(f.relative_to(a.original_pack)) not in pinned or sha(f)!=pinned[str(f.relative_to(a.original_pack))]):raise ValueError('原圖檔案不同：'+str(f))
    catalog=json.loads(a.catalog.read_text())
    parent=next(x for x in catalog['assets'] if x['asset_id']=='modern.fdicon.group_024.style_a.hd6')
    if sha(a.source/parent['files'][0])!=parent['frame_sha256'][0]:raise ValueError('人物24HD來源不同')
    a.out.mkdir()
    names=set()
    for asset in catalog['assets']:
        names.update(asset.get('files',[]));names.update(asset.get('source_files',[]));names.update(asset[k] for k in ['file','master_file'] if k in asset)
    for name in sorted(names):
        if Path(name).name!=name or not(a.source/name).is_file():raise ValueError('舊白名單缺件')
        (a.out/name).symlink_to(os.path.relpath(a.source/name,a.out))
    for name in masters:shutil.copyfile(a.masters/name,a.out/name)
    lock=Image.open(a.out/masters[0]).convert('RGBA');raised=Image.open(a.out/masters[1]).convert('RGBA');raised=raised.crop(raised.getchannel('A').getbbox())
    front=Image.open(a.source/'fdicon-024-style-a-hd6-f00.png').convert('RGBA')
    sheet=Image.open(a.out/masters[2]).convert('RGBA')
    if lock.size!=(1254,1254) or sheet.size!=(1536,1024):raise ValueError('母稿形狀不同')
    parts={name:crop_component(sheet,(i%3*512,i//3*512,(i%3+1)*512,(i//3+1)*512)) for i,name in enumerate(['white','darkblue','lightblue','green','red','key'])}
    fdmeta,fdrows=fd34_plan(a.original_pack/'animations/fdother_034_ch20_sky_key')
    if any(r['coin'] for r in fdrows[1:21]+fdrows[60:70]) or [len(r['orbs']) for r in fdrows[21:26]]!=[1,2,3,4,5]:raise ValueError('原生元件分類不符：衣袍／金鎖或五球白閃')
    fdplan='sky-key-fdother034-layout-oct25-r1.json';write_json(a.out/fdplan,{'schema_version':1,'kind':'sky_key_fdother34_hd6','source_metadata_sha256':sha(a.original_pack/'animations/fdother_034_ch20_sky_key/bank.json'),'frames':fdrows,'base_frame_policy':'native frame0 patch, not a replacement artwork','art_alpha_policy':'preserve source coverage; component source max254 is not promoted to255'})
    fdnames=[]
    for row in fdrows[1:]:
        out=Image.new('RGBA',(row['width']*6,row['height']*6));origin=(row['x'],row['y'])
        paste_scaled(out,raised if row['raised'] else front,[148,94,172,118],origin)
        if row['key']:paste_scaled(out,parts['key'],row['key'],origin)
        for orb in row['orbs']:paste_scaled(out,parts[orb['kind']],orb['bbox'],origin,orb['flash'])
        if row['coin']:paste_scaled(out,lock,row['coin'],origin,row['coin_flash'])
        name=f"sky-key-fdother034-hd6-f{row['index']:03}.png";out.save(a.out/name);fdnames.append(name)
    animeta,anirows,model=prepare_ani(a.original_pack)
    scales=[math.sqrt(abs(r['affine'][0]*r['affine'][4]-r['affine'][1]*r['affine'][3])) for r in anirows if r['affine']]
    if any(b/a>1.6 for a,b in zip(scales,scales[1:])):raise ValueError('縮遠候選有不連續跳變')
    aniplan='sky-key-ani000-layout-oct25-r1.json';write_json(a.out/aniplan,{'schema_version':1,'kind':'sky_key_ani0_hd6','source_metadata_sha256':sha(a.original_pack/'animations/ANI_000/animation.json'),'art_anchors':ART_ANCHORS,'early_model':model,'frames':anirows,'scope':'gem-aligned affine art approximation, not recovered original 3D transforms'})
    aninames=[]
    for row in anirows:
        out=Image.new('RGBA',(1920,1200),(0,0,0,255)) if row['blank'] else warp_lock(lock,row['affine'])
        name=f"sky-key-ani000-hd6-f{row['index']:03}.png";out.save(a.out/name);aninames.append(name)
    entries=[]
    for archive,resource,files,metadata,rows,plan,source_names,contract in [
        ('FDOTHER.DAT',34,fdnames,fdmeta,fdrows[1:],fdplan,masters+['fdicon-024-style-a-hd6-f00.png',fdplan],'sky_key_fdother34_rgba_hd6_v1'),
        ('ANI.DAT',0,aninames,animeta,[{'x':0,'y':0,'width':320,'height':200}]*96,aniplan,[masters[0],aniplan],'sky_key_ani0_rgba_hd6_v1')]:
        entries.append({'asset_id':f'modern.sky-key.{archive[:-4].lower()}.{resource:03}.hd6','role':'story_animation_hd_set','status':'runtime_candidate',
                        'master_file':masters[1] if resource==34 else masters[0],'files':files,'width':max(r['width'] for r in rows)*6,'height':max(r['height'] for r in rows)*6,
                        'frame_count':len(files),'frame_sha256':[sha(a.out/n) for n in files],
                        'source_archive':archive,'source_resource':resource,'source_metadata_sha256':json.loads((a.out/plan).read_text())['source_metadata_sha256'],
                        'source_files':source_names,'source_sha256':[sha(a.out/n) for n in source_names],'native_frames':[descriptor(r) for r in rows],
                        'consumer_contract':contract,'layout_policy':'native_frame_xy_hd6','alpha_contract':'coverage',
                        'source_refs':['docs/knowledge-base/61-modern-theme-runtime-spec.md','docs/data/ida/fd2_ch20_sky_key_sequence_ida.txt']})
    catalog['assets']+=entries;catalog['private_root']=str(a.out.relative_to(ROOT));write_json(a.out/'catalog.json',catalog)
    write_json(a.out/'recipe.json',{'status':'DRAFT-COMPILED-SKY-KEY','tool_sha256':sha(Path(__file__)),'base_catalog_sha256':sha(a.catalog),'catalog_sha256':sha(a.out/'catalog.json'),'art_masters':[{'file':n,'sha256':sha(a.out/n)} for n in masters],'fdother_frames':100,'ani_frames':96,'original_manifest_sha256':sha(a.original_pack/'manifest.json'),'new_entries':entries,'production_changed':False})
    print(json.dumps({'status':'DRAFT-COMPILED-SKY-KEY','new_hd_frames':196,'base_assets':len(catalog['assets'])-2,'candidate_assets':len(catalog['assets'])}))


if __name__=='__main__':main()
