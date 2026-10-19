#!/usr/bin/env python3
"""由新母稿與已驗原版FX遮罩編製熾天使五幀；不改正式清冊。"""
import argparse
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image, ImageChops, ImageDraw, ImageFilter
from build_fd2_gaia_battle_hd import names
from validate_modern_theme_catalog import ROOT, validate


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ['pack','inventory','body','energy','out']:
        p.add_argument('--'+name,required=True,type=Path)
    p.add_argument('--base-catalog',type=Path,default=ROOT/'remake/assets/themes/modern/catalog.json')
    p.add_argument('--charge',type=Path,help='固定世界畫布的近身電光；原身體仍最後疊回')
    p.add_argument('--fx-edge-softness',type=float,default=.65,help='原生遮罩抗鋸齒半徑，僅美術編製')
    p.add_argument('--native-fx-colors',action='store_true',help='重生初期保留原色邊框的草稿')
    args=p.parse_args();out=args.out.resolve()
    if not 0<=args.fx_edge_softness<=2:raise SystemExit('FX抗鋸齒半徑超出有界範圍')
    if out.exists() or not out.is_relative_to(ROOT) or out.parent.stat().st_uid!=os.getuid():
        raise SystemExit('使用儲存庫內本人擁有的新候選目錄')
    inventory=json.loads(args.inventory.read_text())
    if inventory.get('status')!='PASS-COMPOUND-SOURCE' or sha(args.pack/'manifest.json')!=inventory['pack_manifest_sha256']:
        raise SystemExit('原始來源未驗證或盤點不同')
    row=next(x for x in inventory['resources'] if x['resource']==65)
    folder=args.pack/'animations/FDOTHER_065';metadata=folder/'animation.json'
    if sha(metadata)!=row['metadata_sha256']:
        raise SystemExit('metadata來源改變')
    native=json.loads(metadata.read_text())
    pixels,masks=[],[]
    for frame,proof in zip(native['frames'],row['frames']):
        for key,label in [('path','pixels_png_sha256'),('mask_path','mask_png_sha256')]:
            if sha(folder/frame[key])!=proof[label]:raise SystemExit('原始PNG改變')
        with Image.open(folder/frame['path']) as pic,Image.open(folder/frame['mask_path']) as mask:
            stage=Image.new('L',(320,200));stage.paste(Image.frombytes('L',pic.size,pic.tobytes()),(frame['x'],frame['y']))
            alpha=Image.new('L',(320,200));alpha.paste(mask,(frame['x'],frame['y']))
            pixels.append(stage.tobytes());masks.append(alpha.tobytes())
    if len(pixels)!=5:raise SystemExit('原始五幀契約改變')
    base=pixels[0];body_points=[i for i,v in enumerate(base) if v!=0 and masks[0][i]]
    shifts=[0,0,0,0,1];body_proofs=[]
    for index,dy in enumerate(shifts):
        visible=[i for i in body_points if i//320+dy<200]
        if any(pixels[index][i+dy*320]!=base[i] or not masks[index][i+dy*320] for i in visible):
            raise SystemExit('身體不是預期的原始重用／平移；停止編製')
        body_proofs.append({'frame':index,'dy_native':dy,'matched_body_pixels':len(visible),'clipped_body_pixels':len(body_points)-len(visible)})
    with Image.open(args.body) as source:
        if source.mode!='RGBA' or source.getchannel('A').getextrema()!=(0,255):raise SystemExit('身體母稿alpha不符')
        body=source.crop(source.getchannel('A').getbbox()).copy()
    with Image.open(args.energy) as source:
        if source.convert('RGBA').getchannel('A').getextrema()!=(255,255):raise SystemExit('電光材質須為不透明整幅')
        texture=source.convert('RGB').resize((1920,1200),Image.Resampling.LANCZOS)
    charge=None
    if args.charge:
        with Image.open(args.charge) as source:
            if source.mode!='RGBA':raise SystemExit('近身電光母稿須為RGBA')
            low,high=source.getchannel('A').getextrema()
            if low!=0 or high==0:raise SystemExit('近身電光母稿須保留非空coverage alpha')
            if abs(source.width/source.height-1.6)>.01:raise SystemExit('近身電光世界畫布比例不符')
            charge=source.resize((1920,1200),Image.Resampling.LANCZOS)
    first=native['frames'][0];scale=min(first['width']*6/body.width,first['height']*6/body.height)
    body=body.resize((round(body.width*scale),round(body.height*scale)),Image.Resampling.LANCZOS)
    origin=(first['x']*6+(first['width']*6-body.width)//2,first['y']*6+first['height']*6-body.height)
    dac_path=args.pack/'palette/fdother_000.json'
    manifest=json.loads((args.pack/'manifest.json').read_text())
    palette_asset=next(x for x in manifest['assets'] if x['path']=='palette/fdother_000.json')
    if sha(dac_path)!=palette_asset['sha256']:raise SystemExit('色盤SHA改變')
    dac=json.loads(dac_path.read_text())['dac_6bit_components']
    if len(dac)!=768 or any(type(v)is not int or v not in range(64) for v in dac):raise SystemExit('原始DAC不符')
    palette=[tuple((v<<2)|(v>>4) for v in dac[i:i+3]) for i in range(0,768,3)]
    stages,records=[],[]
    for index,(frame,dy) in enumerate(zip(native['frames'],shifts)):
        body_mask={i+dy*320 for i in body_points if i//320+dy<200}
        effect=[bool(masks[index][i]) and value!=0 and i not in body_mask for i,value in enumerate(pixels[index])]
        mask=Image.frombytes('L',(320,200),bytes(255 if v else 0 for v in effect))
        if args.fx_edge_softness:mask=mask.filter(ImageFilter.GaussianBlur(args.fx_edge_softness))
        mask=mask.resize((1920,1200),Image.Resampling.LANCZOS)
        colors=Image.new('RGB',(320,200));colors.putdata([palette[value] if effect[i] else (255,255,255) for i,value in enumerate(pixels[index])])
        colors=colors.resize((1920,1200),Image.Resampling.NEAREST)
        stage=(ImageChops.multiply(texture,colors) if args.native_fx_colors else texture).convert('RGBA');stage.putalpha(mask)
        if index>0 and charge is not None:stage.alpha_composite(charge,(0,dy*6))
        stage.alpha_composite(body,(origin[0],origin[1]+dy*6))
        stages.append(stage)
        record={'frame':index,'dy_native':dy,'body_origin_hd':[origin[0],origin[1]+dy*6],
                'body_size_hd':list(body.size),'fx_native_pixels':sum(effect),'descriptor':row['frames'][index]}
        records.append(record)
    catalog=copy.deepcopy(validate(False,args.base_catalog))
    # 可從目前清冊重生，僅替換同一資源，其他條目的身分及bytes保持。
    catalog['assets']=[x for x in catalog['assets'] if not(x['role']=='battle_effect_hd_set' and x['source_resource']==65)]
    private=ROOT/catalog['private_root'];base_names=names(catalog)
    for name in base_names:
        if Path(name).name!=name or not(private/name).is_file():raise SystemExit('既有清冊缺件')
    out.mkdir();pack=out/'pack';pack.mkdir()
    for name in sorted(base_names):(pack/name).symlink_to(os.path.relpath(private/name,pack))
    source_files=['seraph-body-hd6-master-v1.png','seraph-electric-hd6-master-v1.png']
    source_paths=[args.body,args.energy]
    if args.charge:
        source_files.append('seraph-charge-world-hd6-master-v1.png');source_paths.append(args.charge)
    for source,name in zip(source_paths,source_files):shutil.copyfile(source,pack/name)
    files=[]
    for index,(stage,frame) in enumerate(zip(stages,native['frames'])):
        name=f'fdother-065-hd6-v1-f{index:02d}.png'
        stage.crop((frame['x']*6,frame['y']*6,(frame['x']+frame['width'])*6,(frame['y']+frame['height'])*6)).save(pack/name)
        files.append(name);records[index].update(file=name,sha256=sha(pack/name))
    catalog['private_root']=str(pack.relative_to(ROOT))
    catalog['assets'].append({'asset_id':'modern.fdother.065.hd6.v1','role':'battle_effect_hd_set','status':'runtime_candidate',
        'files':files,'width':1920,'height':1200,'frame_count':5,'frame_sha256':[sha(pack/n) for n in files],
        'consumer_contract':'native_fdother_compound_rgba_hd6_v1','alpha_contract':'coverage',
        'source_refs':['asset:FDOTHER_065','docs/data/fd2-reference-files.json'],'source_archive':'FDOTHER.DAT','source_resource':65,
        'source_metadata_sha256':row['metadata_sha256'],'native_header':[row['native_header'][k] for k in ['byte_1','byte_2','byte_4']],
        'native_frames':[{k:f[k] for k in ['x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7']} for f in row['frames']],
        'cycle_policy':'native_compound_frames','layout_policy':'native_descriptor_hd6',
        'source_files':source_files,'source_sha256':[sha(pack/n) for n in source_files]})
    path=out/'catalog.json';path.write_text(json.dumps(catalog,ensure_ascii=False,indent=2)+'\n');validate(True,path)
    sheet=Image.new('RGB',(1920,480),(30,35,42));draw=ImageDraw.Draw(sheet)
    for index,stage in enumerate(stages):
        thumb=stage.resize((384,240),Image.Resampling.LANCZOS);sheet.paste(thumb,(index*384,240),thumb)
        frame=native['frames'][index]
        with Image.open(folder/frame['path']) as image,Image.open(folder/frame['mask_path']) as alpha:
            source=image.convert('RGBA');source.putalpha(alpha)
        original=Image.new('RGBA',(320,200));original.alpha_composite(source,(frame['x'],frame['y']))
        thumb=original.resize((384,240),Image.Resampling.NEAREST);sheet.paste(thumb,(index*384,0),thumb)
        draw.text((index*384+5,5),f'native source {index}',fill='white');draw.text((index*384+5,245),f'HD candidate {index}',fill='white')
    sheet.save(out/'review.png')
    report={'status':'PASS-COMPILED-SERAPH-CANDIDATE','scope':'明示母稿＋原始FX遮罩，身體縮放一次並保留最後一幀dy1；不是五張獨立繪畫或GUI。',
            'runtime_registered':False,'base_catalog_sha256':sha(args.base_catalog),'catalog_sha256':sha(path),'inventory_sha256':sha(args.inventory),
            'body_source_sha256':sha(args.body),'energy_source_sha256':sha(args.energy),'palette_sha256':sha(dac_path),
            'body_source_proof':body_proofs,'frames':records,'compiler_sha256':sha(Path(__file__)),'review_sha256':sha(out/'review.png')}
    report['fx_edge_softness_native']=args.fx_edge_softness
    report['native_fx_colors']=args.native_fx_colors
    if args.charge:report['charge_source_sha256']=sha(args.charge)
    (out/'compiled.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'frames':len(files),'catalog_assets':len(catalog['assets'])}))


if __name__=='__main__':main()
