#!/usr/bin/env python3
"""核對112離線眼光與原生光色走廊，不將候選美術稱為原版parity。"""
import argparse
import hashlib
import json
import math
import os
from pathlib import Path
import statistics

from PIL import Image, ImageChops

from build_fd2_character_battle_hd import load_native
from validate_modern_theme_catalog import ROOT, validate


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify(compiled, pack, inventory, original):
    report = json.loads(compiled.read_text())
    assert report['status'] == 'PASS-COMPILED-CHARACTER-CANDIDATE'
    assert report['figure'] == 112 and report['ray_projection'] is True
    assert report['inventory_sha256'] == sha(inventory)
    assert report['overlay_compiler_sha256'] == sha(ROOT/'tools/build_fd2_robot112_eye_hd.py')
    parent_path = ROOT/report['parent_compiled']['file']
    assert sha(parent_path) == report['parent_compiled']['sha256']
    parent = json.loads(parent_path.read_text())
    parent_pack = parent_path.parent/'pack'
    parent_catalog = ROOT/report['parent_catalog']['file']
    assert sha(parent_catalog) == report['parent_catalog']['sha256'] == parent['catalog_sha256']
    assert parent['figure'] == 112 and not parent.get('ray_projection')
    assert report['recipe_sha256'] == parent['recipe_sha256']
    candidate_path = compiled.parent/'compiler-catalog.json'
    assert sha(candidate_path) == report['catalog_sha256']
    catalog = validate(False, candidate_path)
    old = json.loads(parent_catalog.read_text())
    assert catalog['assets'][:-1] == old['assets'][:-1]
    asset, old_asset = catalog['assets'][-1], old['assets'][-1]
    assert asset['source_resource'] == old_asset['source_resource'] == 337
    for key in set(old_asset)-{'frame_sha256', 'source_files', 'source_sha256'}:
        assert asset[key] == old_asset[key], key
    source = report['ray_source']
    assert source['axis'] == [[538, 9], [538, 1456]]
    assert source['eye'] == [528, 54] and source['body_pose'] == 'robot112idle'
    assert source['source_core_width'] == 27
    raw = ROOT/source['file']
    assert sha(raw) == source['sha256'] == asset['source_sha256'][-1]
    assert sha(pack/asset['source_files'][-1]) == sha(raw)
    with Image.open(raw) as image:
        assert image.mode == 'RGBA' and image.getchannel('A').getextrema()[0] == 0
        for point in source['axis']:
            assert image.getpixel(tuple(point))[3] >= 192
    body_source = parent['sources']['robot112idle']
    assert sha(ROOT/body_source['file']) == body_source['sha256']
    with Image.open(ROOT/body_source['file']) as image:
        assert image.getpixel((528, 54))[3] >= 192
    frames = {(v['resource'], v['frame']): v for v in report['frames']}
    parents = {(v['resource'], v['frame']): v for v in parent['frames']}
    assert len(frames) == len(report['frames']) == len(parents) == 21
    assert set(frames) == {(336, 0)} | {(337, i) for i in range(20)}
    records, unchanged, attached, detached = [], [], [], []
    for resource in [336, 337]:
        native, originals = load_native(original, json.loads(inventory.read_text()), resource)
        for i, (frame, original_image) in enumerate(zip(native['frames'], originals)):
            record, before = frames[resource, i], parents[resource, i]
            assert record['native_descriptor'] == before['native_descriptor']
            assert all(record['native_descriptor'][k] == frame[k] for k in record['native_descriptor'])
            assert sha(parent_pack/before['file']) == before['sha256']
            assert sha(pack/record['file']) == record['sha256']
            projection = record.get('ray_projection')
            if resource == 336 or i not in range(2, 17):
                assert projection is None and record['sha256'] == before['sha256']
                unchanged.append([resource, i])
                continue
            assert record['parent_png_sha256'] == before['sha256']
            assert record['layers'] == before['layers']
            assert projection['width_hd'] == 6
            assert projection['attached_to_eye'] == (i <= 15)
            layer_path = ROOT/projection['layer_file']
            assert sha(layer_path) == projection['layer_sha256']
            with Image.open(layer_path) as image:
                layer = image.copy()
            with Image.open(parent_pack/before['file']) as image:
                composite = image.copy()
            composite.alpha_composite(layer)
            with Image.open(pack/record['file']) as image:
                assert ImageChops.difference(composite, image).getbbox(alpha_only=False) is None
                assert image.getchannel('A').getextrema() == (0, 255)
            alpha = layer.getchannel('A')
            assert alpha.getextrema()[0] == 0 and alpha.getextrema()[1] >= 250
            body = next(v for v in before['layers'] if v['pose'] == 'robot112idle')
            eye = [body['origin'][j]+[528,54][j]*body['scale_xy'][j] for j in range(2)]
            start = projection['start_hd']
            if i <= 15:
                assert math.dist(start, eye) <= 0.001
                assert any(alpha.getpixel((x,y)) >= 192
                    for y in range(max(0,round(eye[1])-8),min(alpha.height,round(eye[1])+9))
                    for x in range(max(0,round(eye[0])-8),min(alpha.width,round(eye[0])+9)))
                attached.append(i)
            else:
                assert math.dist(start, eye) > 600 and start[1] >= 800
                assert not any(alpha.getpixel((x,y))
                    for y in range(max(0,round(eye[1])-8),min(alpha.height,round(eye[1])+9))
                    for x in range(max(0,round(eye[0])-8),min(alpha.width,round(eye[0])+9)))
                detached.append(i)
            # 獨立從固定原PNG讀取亮金色像素，避免只驗同一仿射矩陣自洽。
            points = []
            for y in range(original_image.height):
                for x in range(original_image.width):
                    red, green, blue, a = original_image.getpixel((x,y))
                    wx, wy = x+frame['x'], y+frame['y']
                    if (a and red >= 190 and green >= 140 and red > blue+30 and green > blue+20
                        and (20 <= wy <= 100 if i<=15 else wy>=135 and wx<=274)):
                        points.append(((x+.5)*6,(y+.5)*6))
            assert len(points) >= 10
            # 手繪頭部的比例與原版不同。將原生光痕起點移到實際HD眼部，
            # 地面終點保持；再核對整條走廊，不能假定兩份身體逐像素相同。
            mx=statistics.mean(x for x,y in points)
            my=statistics.mean(y for x,y in points)
            slope=sum((y-my)*(x-mx) for x,y in points)/sum((y-my)**2 for x,y in points)
            original_y=(6-frame['y'])*6+3
            original_root=[mx+slope*(original_y-my),original_y]
            endpoint_y=(174-frame['y'])*6
            delta=[start[j]-original_root[j] for j in range(2)] if i<=15 else [0,0]
            assert math.hypot(*delta)<=36
            worst = 0
            for px, py in points:
                fraction=(endpoint_y-py)/(endpoint_y-original_root[1]) if i<=15 else 0
                px,py=px+delta[0]*fraction,py+delta[1]*fraction
                distances = [math.hypot(x-px,y-py)
                    for y in range(max(0,math.floor(py)-8),min(alpha.height,math.ceil(py)+9))
                    for x in range(max(0,math.floor(px)-8),min(alpha.width,math.ceil(px)+9))
                    if alpha.getpixel((x,y))>=192]
                assert distances and min(distances)<=8, (i,px,py,'原生光色走廊沒有HD光痕')
                worst=max(worst,min(distances))
            assert record['clipped_high_coverage_pixels'] == before['clipped_high_coverage_pixels']+projection['clipped_high_coverage_pixels']
            records.append(dict(frame=i,native_color_points=len(points),maximum_distance_hd=worst,attached_to_eye=i<=15))
    assert attached==list(range(2,16)) and detached==[16] and len(unchanged)==6
    geometry = report['native_art_geometry']
    assert len(geometry)==20 and all(v['unassigned_warm_pixels']==0 for v in geometry)
    assert geometry[19]['ray'] is None and geometry[19]['warm_pixels']==6
    assert geometry[19]['fires']==[dict(world_box=[299,111,300,130],kind='embers',native_pixels=6)]
    # 六個原生暗點均落在實際非空火星層範圍，不能只在報告列出一個方框。
    last=frames[337,19];ember=next(v for v in last['layers'] if v['pose']=='gold112embers3')
    assert ember['pose_used'] and not ember['native_empty_source']
    with Image.open(pack/last['file']) as image:
        assert image.getchannel('A').crop(((299-13)*6,110*6,287*6,130*6)).getextrema()[1]>=250
    assert asset['frame_sha256']==[frames[337,i]['sha256'] for i in range(20)]
    return dict(status='PASS-ROBOT112-EYE-SOURCE-GEOMETRY',frames=21,ray_frames=15,attached=attached,
        detached=detached,unchanged=unchanged,corridors=records,last_dark_native_pixels=6,
        scope='固定原PNG光色走廊與來源接點、父合成與SHA。美術幾何近似，不是原版RGB或正常玩家驗證。')


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for key in ['compiled','pack','inventory','original','report']:
        parser.add_argument('--'+key,type=Path,required=True)
    args=parser.parse_args()
    assert Path('/.dockerenv').exists() and not args.report.exists()
    assert args.report.parent.stat().st_uid==os.getuid()
    result=verify(args.compiled,args.pack,args.inventory,args.original)
    args.report.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({k:result[k] for k in ['status','frames','ray_frames','last_dark_native_pixels']}))


if __name__=='__main__':
    main()
