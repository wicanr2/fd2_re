#!/usr/bin/env python3
"""依固定原生raw圖源與明確圖集裁切，編製HD服務選單圖示。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil

from PIL import Image, __version__ as pillow_version

def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def compile_icons(recipe_path, master, original, out):
    if not Path('/.dockerenv').exists() or out.exists() or not out.parent.is_dir():
        raise ValueError('需要Docker與既有父目錄中的全新輸出')
    if out.parent.stat().st_uid != os.getuid() or pillow_version != '11.3.0':
        raise ValueError('輸出UID或Pillow版本不同')
    recipe=json.loads(recipe_path.read_text())
    if recipe['schema_version']!=1 or recipe['kind']!='fd2_service_icon_atlas_hd6_v1' or len(recipe['groups'])!=20:
        raise ValueError('配方類型或格數不同')
    if sha(master)!=recipe['master_sha256'] or master.name!=recipe['master_file'] or sha(original/'manifest.json')!=recipe['original_manifest_sha256']:
        raise ValueError('主稿或原包SHA不同')
    if recipe['native_size']!=[24,20] or recipe['output_size']!=[144,120] or recipe['zero_source_positions']!=[[0,0],[23,0],[0,19],[23,19]] or recipe['corner_fill_index']!=74:
        raise ValueError('已證幾何或角落來源不同')
    palette_file=original/'palette/fdother_000.json'
    if sha(palette_file)!=recipe['palette_sha256']:
        raise ValueError('色盤來源不同')
    dac=json.loads(palette_file.read_text())['dac_6bit_components']
    blue=tuple((v<<2)|(v>>4) for v in dac[222:225])
    if list(blue)!=recipe['corner_fill_rgb'] or blue!=(40,69,138):
        raise ValueError('選單原底色不同')
    canvas=Image.open(master).convert('RGBA')
    if list(canvas.size)!=recipe['master_size'] or canvas.getchannel('A').getextrema()!=(255,255):
        raise ValueError('主稿尺寸或不透明不同')
    entries=[];pictures={};sources=[];seen=set()
    for cell,group in enumerate(recipe['groups']):
        if group['cell']!=cell or Path(group['output_file']).name!=group['output_file']:
            raise ValueError('來源格或檔名不同')
        rect=tuple(group['crop'])
        if len(rect)!=4 or not(0<=rect[0]<rect[2]<=canvas.width and 0<=rect[1]<rect[3]<=canvas.height):
            raise ValueError('裁切越界')
        picture=canvas.crop(rect).resize((144,120),Image.Resampling.LANCZOS)
        for binding in group['bindings']:
            resource,index=binding['resource'],binding['index']
            if resource not in {12,13,14,29,63} or index not in range(3,11) or (resource,index) in seen:
                raise ValueError('未知或重複來源')
            folder=Path('ui/fdother_014_church') if resource==14 else Path('shop/FDOTHER_%03d'%resource)
            if sha(original/folder/'resource.json')!=binding['source_metadata_sha256']:
                raise ValueError('metadata不同')
            relative=folder/('entry_%03d/frame.png'%index)
            if str(relative)!=binding['original_file'] or sha(original/relative)!=binding['original_file_sha256']:
                raise ValueError('原始PNG不同')
            native=Image.open(original/relative)
            if native.mode!='P' or native.size!=(24,20):
                raise ValueError('原始圖模式或尺寸不同')
            pixels=native.tobytes();digest=hashlib.sha256(pixels).hexdigest()
            description=dict(index=index,width=24,height=20,pixels_sha256=digest)
            zeros=[[i%24,i//24] for i,v in enumerate(pixels) if v==0]
            if sorted(zeros)!=sorted(recipe['zero_source_positions']) or digest!=group['pixels_sha256'] or description!=binding['native_raw_cell']:
                raise ValueError('raw圖案、共用來源或零角不同')
            seen.add((resource,index));sources.append({'resource':resource,'index':index,'pixels_sha256':digest})
        # Only source-proved zero pixels in stable cleared menu corners.
        for x,y in recipe['zero_source_positions']:
            picture.paste(blue+(255,),(x*6,y*6,(x+1)*6,(y+1)*6))
        pictures[group['output_file']]=picture
    if seen!={(r,i) for r in [12,13,14,29,63] for i in range(3,11)}:
        raise ValueError('五來源40圖示不完整')
    out.mkdir()
    shutil.copyfile(master,out/master.name)
    for name,picture in pictures.items():
        picture.save(out/name)
    for group in recipe['groups']:
        name=group['output_file'];digest=sha(out/name)
        for b in group['bindings']:
            resource,index=b['resource'],b['index']
            entries.append({'asset_id':'modern.ui.service.fdother%03d.raw%03d.hd6'%(resource,index),
                'role':'native_ui_icon_hd','status':'runtime_candidate','file':name,'width':144,'height':120,'sha256':digest,
                'master_file':master.name,'consumer_contract':'native_raw_ui_icon_hd6_v1','alpha_contract':'opaque',
                'layout_policy':'native_raw_cell_rect_hd6','source_archive':'FDOTHER.DAT','source_resource':resource,
                'source_metadata_sha256':b['source_metadata_sha256'],'native_raw_cell':b['native_raw_cell'],
                'source_files':[master.name],'source_sha256':[recipe['master_sha256']],
                'source_refs':['asset:FDOTHER_%03d.raw_%03d'%(resource,index),
                               'remake/assets/themes/modern/facility-service-icons.recipe.json',
                               'tools/build_fd2_service_icons_hd.py','docs/knowledge-base/61-modern-theme-runtime-spec.md']})
    files=[{'file':p.name,'sha256':sha(p)} for p in sorted(out.glob('*.png'))]
    report={'status':'PASS-SERVICE-ICON-HD6-COMPILE','pillow':pillow_version,'recipe_sha256':sha(recipe_path),
            'master_sha256':sha(master),'assets':entries,'files':files,'source_checks':sources,
            'native_corner_blue_pixels':20*4*36,'alpha':'all opaque; no threshold','resampling':'LANCZOS',
            'manual_appearance_accepted':False,'scope':'20 paintings /40 exact raw-source bindings; stable menus only'}
    (out/'receipt.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    return report

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ['recipe','master','original-pack','out']:
        parser.add_argument('--'+name,type=lambda s:Path(s).absolute(),required=True)
    args=parser.parse_args()
    result=compile_icons(args.recipe,args.master,args.original_pack,args.out)
    print(json.dumps({'status':result['status'],'assets':len(result['assets']),'files':len(result['files'])}))
if __name__=='__main__':
    main()
