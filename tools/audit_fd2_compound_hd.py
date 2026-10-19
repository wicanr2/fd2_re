#!/usr/bin/env python3
"""核對FDOTHER固定容器及65～68的原始描述子，不重做已閉合語意。"""
import argparse
import hashlib
import json
from pathlib import Path
import struct

from PIL import Image, __version__ as pillow_version
from audit_fd2_battle_animation_hd import REPO, require, sha


def audit(pack):
    manifest_path = pack/'manifest.json'
    manifest = json.loads(manifest_path.read_text())
    reference = json.loads((REPO/'docs/data/fd2-reference-files.json').read_text())
    inputs = []
    for name in ['FD2.EXE', 'FDOTHER.DAT']:
        pinned = next(x for x in reference['files'] if x['file'] == name)
        require(next(x for x in manifest['source_set'] if x['file'] == name) == pinned, '固定來源不符：'+name)
        inputs.append(pinned)
    selected = [x for x in manifest['assets'] if x.get('source_file') == 'FDOTHER.DAT']
    assets = {x['asset_id']: x for x in selected}
    by_path = {x['path']: x for x in selected}
    require(len(assets) == len(selected) == len(by_path), 'FDOTHER清冊身分或路徑重複')
    verified = set()

    def resolve(ident):
        require(ident in assets, '清冊未登錄：'+ident)
        item = assets[ident]
        relative = Path(item['path'])
        require(not relative.is_absolute() and '..' not in relative.parts, '清冊路徑越界')
        path = pack/relative
        if ident not in verified:
            require(sha(path) == item['sha256'], '資產SHA不符：'+ident)
            verified.add(ident)
        return path

    resources = sorted((x for x in manifest['source_resources'] if x['source_file'] == 'FDOTHER.DAT'), key=lambda x:x['source_resource'])
    require([x['source_resource'] for x in resources] == list(range(len(resources))), '原始資源索引不連續')
    offsets, cursor = [], 6+4*len(resources)
    for row in resources:
        offsets.append(cursor)
        cursor += row['raw_bytes']
    header = b'LLLLLL'+b''.join(struct.pack('<I', x) for x in offsets)
    digest, md5 = hashlib.sha256(header), hashlib.md5(header)
    raw_resources = {}
    for row in resources:
        path = resolve(row['raw_asset_id'])
        raw = path.read_bytes()
        require(len(raw) == row['raw_bytes'] and sha(path) == row['raw_sha256'], '原始資源bytes不符')
        digest.update(raw); md5.update(raw)
        if row['source_resource'] in range(65,69):
            raw_resources[row['source_resource']] = raw
    pinned = next(x for x in inputs if x['file'] == 'FDOTHER.DAT')
    require(cursor == pinned['size'] and digest.hexdigest() == pinned['sha256'] and md5.hexdigest() == pinned['md5'], 'raw重組不等於固定FDOTHER.DAT')
    rows = []
    for resource in range(65,69):
        metadata = f'animations/FDOTHER_{resource:03d}/animation.json'
        require(metadata in by_path, '缺少動畫metadata')
        path = resolve(by_path[metadata]['asset_id'])
        doc = json.loads(path.read_text()); raw = raw_resources[resource]
        source = next(x for x in resources if x['source_resource'] == resource)
        require(doc['source'] == {'file':'FDOTHER.DAT','resource':resource}, '動畫來源家族不符')
        require(doc['native_header'] == {'byte_1':raw[1],'byte_2':raw[2],'byte_4':raw[4]}, '原始header不符')
        require(len(doc['frames']) == raw[0] and len(raw) >= 8+4*raw[0], '原始幀數不符')
        frames = []
        for index, frame in enumerate(doc['frames']):
            offset = struct.unpack_from('<I',raw,8+4*index)[0]
            end = struct.unpack_from('<I',raw,12+4*index)[0] if index+1 < raw[0] else len(raw)
            require(8+4*raw[0] <= offset and offset+13 <= end <= len(raw), '描述子越界')
            x,y = struct.unpack_from('<hh',raw,offset); w,h = struct.unpack_from('<HH',raw,offset+9)
            expected = dict(x=x,y=y,width=w,height=h,delay_native=raw[offset+6],raw_byte_4=raw[offset+4],raw_byte_5=raw[offset+5],raw_byte_7=raw[offset+7])
            require(all(frame[k] == v for k,v in expected.items()), '原始描述子不符')
            record = {'frame':index, **expected}
            for key,mode,label in [('path','P','pixels_png_sha256'),('mask_path','L','mask_png_sha256')]:
                require(Path(frame[key]).name == frame[key], '動畫檔名越界')
                relative = str((path.parent/frame[key]).relative_to(pack))
                require(relative in by_path, '影格未登錄')
                item = by_path[relative]
                require(item['asset_id'] in source['output_asset_ids'], '影格不屬於本資源')
                picture_path = resolve(item['asset_id'])
                with Image.open(picture_path) as picture:
                    require(picture.mode == mode and picture.size == (w,h), '影格形狀不符')
                record[label] = sha(picture_path)
            frames.append(record)
        rows.append({'resource':resource,'metadata_sha256':sha(path),'native_header':doc['native_header'],'frames':frames})
    return {'status':'PASS-COMPOUND-SOURCE','schema_version':1,'source_inputs':inputs,'pack_manifest_sha256':sha(manifest_path),
            'archive_reconstruction':{'resources':len(resources),'size':cursor,'sha256':digest.hexdigest(),'md5':md5.hexdigest()},
            'resources':rows,'verified_manifest_assets':len(verified),'pillow_version':pillow_version,'tool_sha256':sha(Path(__file__)),
            'scope':'固定容器、描述子與分離PNG來源；不是GUI或新增原版執行收據。'}


if __name__ == '__main__':
    p=argparse.ArgumentParser(description=__doc__);p.add_argument('--pack',required=True,type=Path);p.add_argument('--out',required=True,type=Path)
    args=p.parse_args();report=audit(args.pack);args.out.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'resources':len(report['resources']),'frames':sum(len(r['frames']) for r in report['resources'])}))
