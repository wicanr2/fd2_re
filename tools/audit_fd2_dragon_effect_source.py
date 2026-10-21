#!/usr/bin/env python3
"""重生火龍兩圖庫全部36幀的固定技術參考；不是HD或原版執行期證據。"""
import argparse
import json
from pathlib import Path

from PIL import Image, ImageDraw
from build_fd2_gaia_battle_hd import digest


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for key in ['original-pack','reference','out']:
        parser.add_argument('--'+key,type=Path,required=True)
    args=parser.parse_args()
    report,picture=[Path(str(args.out)+suffix) for suffix in ['.json','.png']]
    if not Path('/.dockerenv').exists() or report.exists() or picture.exists():
        raise SystemExit('在Docker使用新的參考圖輸出前綴')
    source=json.loads(args.reference.read_text())
    if (source.get('status')!='PASS-PINNED-NATIVE-ASSET-REFERENCE'
            or digest(args.original_pack/'manifest.json')!=source['pack_manifest_sha256']
            or source['tool']['sha256']!=digest(Path(__file__).with_name('audit_fd2_ordinary_effect_source.py'))):
        raise SystemExit('固定普通效果來源盤點不同')
    sheet=Image.new('RGB',(1440,1080),(38,43,50));draw=ImageDraw.Draw(sheet);records=[]
    for n,resource in enumerate([26,27]):
        folder=args.original_pack/'animations'/f'FDOTHER_{resource:03d}'
        row=next(v for v in source['resources'] if v['resource']==resource)
        metadata=folder/'animation.json'
        if digest(metadata)!=row['metadata_sha256']:raise SystemExit('火龍metadata SHA不同')
        doc=json.loads(metadata.read_text())
        if doc['native_header']!=row['native_header'] or len(doc['frames'])!=18:
            raise SystemExit('火龍header或幀數不同')
        for index,(frame,proof) in enumerate(zip(doc['frames'],row['frames'])):
            if any(Path(frame[k]).name!=frame[k] for k in ['path','mask_path']):raise SystemExit('圖源路徑越界')
            if (digest(folder/frame['path'])!=proof['pixels_sha256']
                    or digest(folder/frame['mask_path'])!=proof['mask_sha256']
                    or any(frame[k]!=proof[k] for k in ['x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7'])):
                raise SystemExit('火龍圖源、mask或描述子不同')
            with Image.open(folder/frame['path']) as im,Image.open(folder/frame['mask_path']) as mask:
                if im.mode!='P' or im.size!=mask.size or im.size!=(frame['width'],frame['height']):
                    raise SystemExit('火龍圖源形狀不同')
                tile=im.convert('RGBA');tile.putalpha(mask.convert('L'))
            tile.thumbnail((224,150),Image.Resampling.NEAREST)
            x,y=index%6*240,(n*3+index//6)*180
            sheet.paste(tile,(x+(240-tile.width)//2,y+25+(150-tile.height)//2),tile)
            draw.text((x+3,y+3),f"{resource}:{index} xy{frame['x']},{frame['y']} {frame['width']}x{frame['height']}",fill='white')
            records.append({'resource':resource,'frame':index,**proof})
    sheet.save(picture)
    report.write_text(json.dumps({'status':'PINNED-NATIVE-REFERENCE-NOT-HD',
        'source_reference_sha256':digest(args.reference),'png_sha256':digest(picture),'frames':records,
        'tool_sha256':digest(Path(__file__))},indent=2)+'\n')
    print('36 pinned native dragon frames; reference only')


if __name__=='__main__':
    main()
