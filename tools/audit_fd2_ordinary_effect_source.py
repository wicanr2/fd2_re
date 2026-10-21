#!/usr/bin/env python3
"""核普通法術FDOTHER圖源，輸出可重生技術參考，不宣稱執行期或HD美術完成。"""
import argparse
import json
import os
from pathlib import Path
import platform

from PIL import Image, ImageDraw, __version__ as pillow_version
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import ORDINARY_EFFECT_RESOURCES, ROOT


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--original-pack', type=Path, required=True)
    parser.add_argument('--out', type=Path, required=True, help='新輸出前綴，附加.json與.png')
    args = parser.parse_args()
    report, reference = (Path(str(args.out)+suffix) for suffix in ['.json','.png'])
    if (not Path('/.dockerenv').exists() or report.exists() or reference.exists()
            or not args.out.parent.is_dir() or args.out.parent.stat().st_uid != os.getuid()):
        raise SystemExit('在Docker使用本人擁有的新輸出前綴')
    pack = args.original_pack
    inventory = ROOT/'docs/data/fd2-battle-hd-source-inventory-20261019.json'
    expected = json.loads(inventory.read_text())
    if digest(pack/'manifest.json') != expected['pack_manifest_sha256']:
        raise SystemExit('原版包指紋不同')
    manifest = json.loads((pack/'manifest.json').read_text())
    pinned = next(a for a in json.loads((ROOT/'docs/data/fd2-reference-files.json').read_text())['files'] if a['file']=='FDOTHER.DAT')
    if next(a for a in manifest['source_set'] if a['file']=='FDOTHER.DAT') != pinned:
        raise SystemExit('FDOTHER來源指紋不同')
    files = {a['path']:a for a in manifest['assets']}
    resources = sorted(ORDINARY_EFFECT_RESOURCES)
    sheet = Image.new('RGB',(1200,((len(resources)+3)//4)*220),(38,43,50))
    draw = ImageDraw.Draw(sheet)
    records = []
    for n, resource in enumerate(resources):
        folder = pack/'animations'/f'FDOTHER_{resource:03d}'
        def checked(path, index=None):
            row = files.get(path.relative_to(pack).as_posix())
            if (row is None or row.get('source_file')!='FDOTHER.DAT' or row.get('source_resource')!=resource
                    or digest(path)!=row['sha256'] or index is not None and row.get('source_frame')!=index):
                raise SystemExit('原生效果圖源SHA或定位不同')
            return row
        metadata = folder/'animation.json'
        checked(metadata)
        doc = json.loads(metadata.read_text())
        if doc.get('source') != {'file':'FDOTHER.DAT','resource':resource} or not doc.get('frames'):
            raise SystemExit('原生效果metadata定位不同或缺幀')
        frames, pictures = [], []
        for index, frame in enumerate(doc['frames']):
            if any(Path(frame[k]).name != frame[k] for k in ['path','mask_path']):
                raise SystemExit('原生效果圖源路徑越界')
            path, mask_path = folder/frame['path'], folder/frame['mask_path']
            a, b = checked(path,index), checked(mask_path,index)
            with Image.open(path) as im, Image.open(mask_path) as mask:
                if im.mode!='P' or im.size!=mask.size or im.size!=(frame['width'],frame['height']):
                    raise SystemExit('原生效果PNG／mask形狀不同')
                picture = im.convert('RGBA')
                picture.putalpha(mask.convert('L'))
                visible = sum(v!=0 for v in mask.getdata())
            frames.append({**{k:frame[k] for k in ['x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7']},
                           'pixels_sha256':a['sha256'],'mask_sha256':b['sha256'],'visible_pixels':visible})
            pictures.append(picture)
        best = max(range(len(frames)),key=lambda i:frames[i]['visible_pixels'])
        pic = pictures[best]
        scale = min(280/pic.width,170/pic.height,3)
        thumb = pic.resize((round(pic.width*scale),round(pic.height*scale)),Image.Resampling.NEAREST)
        x, y = n%4*300, n//4*220
        sheet.paste(thumb,(x+(300-thumb.width)//2,y+30+(170-thumb.height)//2),thumb)
        draw.text((x+5,y+5),f'FDOTHER{resource} frame{best} / {len(frames)}',fill='white')
        records.append({'resource':resource,'source_file':'FDOTHER.DAT','metadata_sha256':digest(metadata),
                        'native_header':doc['native_header'],'frames':frames,'reference_frame':best})
    sheet.save(reference)
    report.write_text(json.dumps({'status':'PASS-PINNED-NATIVE-ASSET-REFERENCE',
        'pack_manifest_sha256':digest(pack/'manifest.json'),'source_file':pinned,'resources':records,
        'reference_png_sha256':digest(reference),'inventory_sha256':digest(inventory),
        'tool':{'file':'tools/audit_fd2_ordinary_effect_source.py','sha256':digest(Path(__file__)),
                'python':platform.python_version(),'pillow':pillow_version,
                'address_space':'分離圖源內相對矩形，非EXE位址'},
        'scope':'固定分離包的圖源與描述子機械式核對；不是原版執行期或新增語意。'},
        ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':'PASS','resources':len(records),'frames':sum(len(x['frames']) for x in records)}))


if __name__ == '__main__':
    main()
