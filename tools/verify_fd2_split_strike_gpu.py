#!/usr/bin/env python3
"""核對24／29／30完整最終GPU、固定FIGANI來源及原生空拍。須在資產Docker執行。"""
import argparse
import hashlib
import json
from functools import lru_cache
from pathlib import Path

from PIL import Image, ImageChops, ImageMath
from build_fd2_character_battle_hd import DESCRIPTOR, load_native
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_compound_caster_gpu import pinned_native_blank
from audit_fd2_battle_background_sources import audit as audit_backgrounds

INVENTORY_SHA = 'd019e2cb233d8bbe8d6b6d688221fdc2fecadc78bf39142e5e3c2f6b57375f51'


@lru_cache(maxsize=2)
def background_inventory(pack):
    return audit_backgrounds(pack)[0]


def shader_channels(picture, subtract):
    # Go NRGBA -> RGBA uploads floor((c*257*a/255)/256). The shader
    # subtracts from this quantized premultiplied texture, before Over.
    # Computing a straight-alpha LUT first rounds a different operation.
    channels = picture.split()
    alpha = channels[3]
    result = []
    for channel in channels[:3]:
        premul = ImageMath.lambda_eval(
            lambda v: v['float'](v['int'](v['float'](v['c'])*v['float'](v['a'])*257/65280)), c=channel, a=alpha)
        result.append(ImageMath.lambda_eval(lambda v: v['max'](v['c']-v['float'](v['a'])*subtract, 0), c=premul, a=alpha))
    return tuple(result)+(alpha,)


def over_shader(expected, channels, rect, visible):
    crop = (visible[0]-rect[0], visible[1]-rect[1], visible[2]-rect[0], visible[3]-rect[1])
    rgba = expected.crop(visible).split()
    src = [c.crop(crop) for c in channels]
    result = [ImageMath.lambda_eval(
        lambda v: v['convert'](v['s']+v['float'](v['b'])*(255-v['float'](v['a']))/255+0.5, 'L'),
        s=src[i], b=rgba[i], a=src[3]) for i in range(3)]
    expected.paste(Image.merge('RGBA', tuple(result)+(rgba[3],)), (visible[0], visible[1]))


def checked_image(root, name, sha, size=None):
    if not isinstance(name, str) or Path(name).name != name or digest(root/name) != sha:
        raise ValueError('圖片路徑或SHA不同')
    with Image.open(root/name) as im:
        if size is not None and im.size != size:
            raise ValueError('圖片尺寸不同')
        return im.convert('RGBA')


def panel_channels(receipt,layer):
    if (layer.get('archive')!='' or layer.get('resource')!=-1 or layer.get('frame')!=0
            or [layer.get('width'),layer.get('height')]!=[149,42] or layer.get('preserve_palette') is not True
            or 'native_file' in layer or [layer.get('x'),layer.get('y')] not in ([0,154],[171,4])
            or not isinstance(layer.get('panel_name'),str)):
        raise ValueError('HD文字面板來源或不變色色盤契約不同')
    values=layer.get('panel_values',{})
    if set(values)!={'Level','HP','MaxHP','MP','MaxMP'} or any(type(v) is not int for v in values.values()):
        raise ValueError('HD文字面板缺少凍結數值')
    picture=checked_image(receipt.parent,layer.get('panel_file'),layer.get('panel_sha256'),(894,252))
    if picture.getchannel('A').getextrema()!=(255,255) or layer.get('hd_sha256')!='panel-text/'+hashlib.sha256(picture.tobytes()).hexdigest():
        raise ValueError('HD文字面板RGBA內容或GPU鍵不同')
    return shader_channels(picture,0)


def native_transition_channels(receipt,sample,layer,original_pack,row):
    if layer['hd_sha256'] or 'native_file' not in layer:
        raise ValueError('未登錄過渡來源不可偽裝為HD')
    folder=original_pack/'surfaces'/f"BG_{layer['resource']:03d}"
    meta=json.loads((folder/'resource.json').read_text())
    with Image.open(receipt.parent/sample['base_file']) as base, Image.open(folder/meta['frame']) as native, Image.open(folder/meta['mask']) as mask:
        if base.mode!='P' or native.mode!='P':
            raise ValueError('原生過渡回退缺少索引色盤')
        native=native.copy();native.putpalette(base.getpalette())
        expected=native.convert('RGBA');expected.putalpha(mask.convert('L'))
    actual=checked_image(receipt.parent,layer['native_file'],layer['native_sha256'],(row['width'],row['height']))
    if any(high!=0 for _,high in ImageChops.difference(expected,actual).getextrema()):
        raise ValueError('原生過渡回退與固定來源／當次色盤不同')
    return shader_channels(actual.resize((row['width']*6,row['height']*6),Image.Resampling.NEAREST),0)


def verify(receipt, catalog, source, original_pack, inventory, runtime_receipt=None):
    if digest(inventory) != INVENTORY_SHA:
        raise ValueError('固定來源盤點不同')
    proof = json.loads(inventory.read_text())
    if (proof.get('status') != 'PASS-SOURCE-INVENTORY'
        or proof.get('archive_reconstruction', {}).get('matches_pinned_original') is not True
        or digest(original_pack/'manifest.json') != proof['pack_manifest_sha256']):
        raise ValueError('原始包或重建證據不同')
    doc = validate(False, catalog)
    entries = {a['source_resource']: a for a in doc['assets'] if a['role'] == 'battle_animation_hd_set'}
    surfaces = {(a['source_archive'],a['source_resource']):a for a in doc['assets'] if a['role']=='battle_surface_hd'}
    d = json.loads(receipt.read_text())
    command = d.get('command_id')
    profiles = {24: (32, 15, 9, 1, 2, 22), 29: (34, 18, 11, 2, 29, 26),
                30: (18, 14, 9, 2, 30, 24)}
    if command not in profiles:
        raise ValueError('需要24／29／30收據')
    figure, count, start, target_count, seed, mp_cost = profiles[command]
    resource = figure*3+2
    phases = {'prelude', 'actor'} | {p+str(i) for i in range(target_count)
                                                for p in ('transition-out-', 'transition-in-', 'target-')}
    if (d.get('status') != 'PASS-SPLIT-STRIKE-RUNTIME-E1' or d.get('actor_figure') != figure
        or d.get('catalog_sha256') != digest(catalog) or d.get('hashes_only') is not False
        or d.get('continuations') != 1 or d.get('owner_released') is not True
        or d.get('gpu_sources_cleared') is not True or d.get('mp_changes') != 1
        or d.get('mp_before', 0)-d.get('mp_after', 0) != mp_cost
        or d.get('hp_changes') != [1]*target_count or len(d.get('hp_before', [])) != target_count
        or len(d.get('hp_after', [])) != target_count
        or any(a <= b for a, b in zip(d['hp_before'], d['hp_after']))
        or d.get('go_seed') != seed or d.get('go_rng_planning_calls') != target_count
        or d.get('go_rng_next_matches_native_plan') is not True or d.get('native_rng') != 0
        or set(d.get('phase_draws', {})) != phases or set(d.get('phase_f2', {})) != phases
        or any(d['phase_draws'][p] <= 0 or d['phase_f2'][p] != 1 for p in phases)
        or d.get('hd_draws', 0) <= 0 or d.get('final_draws', 0) <= 0):
        raise ValueError('生命週期／F2／標記／亂數／返回收據不完整')
    rows, pictures, native_proofs = {}, {}, {}

    def bank(rid):
        if rid not in rows:
            row, _ = load_native(original_pack, proof, rid)
            asset = entries[rid]
            if (asset['source_metadata_sha256'] != row['metadata_sha256']
                or asset['native_header'] != [row['native_header'][f'byte_{i}'] for i in (1, 2, 4)]
                or len(asset['files']) != len(row['frames'])
                or any(any(a[k] != b[k] for k in DESCRIPTOR)
                       for a, b in zip(asset['native_frames'], row['frames']))):
                raise ValueError('清冊描述子不是固定原圖')
            rows[rid] = row
            native_proofs[str(rid)] = row['metadata_sha256']
        return rows[rid], entries[rid]

    row, asset = bank(resource)
    if len(row['frames']) != count:
        raise ValueError('分段打擊幀數不同')
    blank = pinned_native_blank(original_pack, proof, resource, count-1, source/asset['files'][-1])
    found, coverage, outputs, worst = set(), {p: set() for p in phases}, {}, 0
    used_surfaces=set(); surface_proofs={}
    shots = d.get('source_shots', {})
    if not shots:
        raise ValueError('缺少實際GPU畫面')
    for label, sample in shots.items():
        phase, cursor = sample['phase'], sample['cursor']
        if phase not in phases:
            raise ValueError('未知相位')
        index = cursor['Prelude'] if phase == 'prelude' else cursor['Transition'] if phase.startswith('transition-') else cursor['Effect']
        coverage[phase].add(index)
        expected = checked_image(receipt.parent, sample['base_file'], sample['base_sha256'], (320, 200)).resize((1920, 1200), Image.Resampling.NEAREST)
        subtract = sample['subtract']
        want_subtract = (8-index)*6/63 if phase == 'prelude' else 0
        if abs(subtract-want_subtract) > 0.000001:
            raise ValueError('原生DAC減色不同')
        for layer in sample['layers']:
            x, y, w, h = (layer[k] for k in ('x', 'y', 'width', 'height'))
            rect = [(x+layer['dx'])*6, (y+layer['dy'])*6,
                    (x+layer['dx']+w)*6, (y+layer['dy']+h)*6]
            if rect != layer['rect'] or layer['fill'] != -1:
                raise ValueError('圖層目的矩形或填色不同')
            rid, frame = layer['resource'], layer['frame']
            archive=layer.get('archive','')
            if archive not in ('','FIGANI.DAT','BG.DAT','TAI.DAT'):
                raise ValueError('不支援的分段打擊archive')
            if archive in ('BG.DAT','TAI.DAT'):
                if frame!=0 or not isinstance(rid,int) or not 0<=rid<56:
                    raise ValueError('場景來源或單圖索引不同')
                row=background_inventory(original_pack)['banks'][archive[:-4]]['entries'][rid]
                origin=(0,50) if archive=='BG.DAT' else (164,157)
                if [x,y,w,h]!=[origin[0],origin[1],row['width'],row['height']]:
                    raise ValueError('場景原描述子／metadata／HD身分不同')
                if (archive,rid) not in surfaces:
                    if archive!='BG.DAT' or rid not in (0,1,2):
                        raise ValueError('場景來源或單圖索引不同')
                    picture=native_transition_channels(receipt,sample,layer,original_pack,row)
                else:
                    asset=surfaces[archive,rid]
                    if (asset['source_metadata_sha256']!=row['metadata_sha256'] or asset['width']!=w*6 or asset['height']!=h*6
                            or asset['native_frames']!=[dict(x=0,y=0,width=w,height=h,delay_native=0,raw_byte_4=0,raw_byte_5=0,raw_byte_7=0)]
                            or layer['hd_sha256']!=asset['sha256'] or 'native_file' in layer):
                        raise ValueError('場景原描述子／metadata／HD身分不同')
                    key=archive,rid,subtract
                    if key not in pictures:
                        pictures[key]=shader_channels(checked_image(source,asset['file'],asset['sha256'],(w*6,h*6)),subtract)
                    picture=pictures[key];used_surfaces.add((archive,rid))
                surface_proofs[f'{archive}:{rid}']=row['metadata_sha256']
            elif 'panel_file' in layer:
                picture=panel_channels(receipt,layer)
            elif rid >= 0:
                native, registered = bank(rid)
                if not 0 <= frame < len(native['frames']):
                    raise ValueError('圖源索引越界')
                raw = native['frames'][frame]
                if [x, y, w, h] != [raw[k] for k in ('x', 'y', 'width', 'height')]:
                    raise ValueError('實際Draw原描述子不同')
                sha = registered['frame_sha256'][frame]
                if layer['hd_sha256'] != sha or 'native_file' in layer:
                    raise ValueError('HD來源SHA或來源類別不同')
                key = rid, frame, subtract
                if key not in pictures:
                    im = checked_image(source, registered['files'][frame], sha, (w*6, h*6))
                    pictures[key] = shader_channels(im, subtract)
                picture = pictures[key]
                if rid == resource and phase in ('actor', 'target-0', 'target-1'):
                    found.add(frame)
            else:
                if layer['hd_sha256']:
                    raise ValueError('原生圖層不可偽裝為HD')
                picture = shader_channels(checked_image(receipt.parent, layer['native_file'], layer['native_sha256'], (w, h)).resize((w*6, h*6), Image.Resampling.NEAREST), 0)
            clip = layer.get('clip', [0, 0, 1920, 1200])
            if len(clip) != 4 or any(type(v) is not int or v % 6 for v in clip):
                raise ValueError('裁切區域不是原生座標')
            visible = [max(0, rect[0], clip[0]), max(0, rect[1], clip[1]),
                       min(1920, rect[2], clip[2]), min(1200, rect[3], clip[3])]
            if visible[0] < visible[2] and visible[1] < visible[3]:
                over_shader(expected, picture, rect, visible)
        actual = checked_image(receipt.parent, sample['file'], sample['sha256'], (1920, 1200))
        error = max(v[1] for v in ImageChops.difference(expected, actual).getextrema())
        worst = max(worst, error)
        if error > 1:
            raise ValueError(f'完整GPU與獨立CPU Over不同：{label}，最大差{error}')
        outputs[label] = sample['sha256']
    required = {'prelude': set(range(9)), 'actor': set(range(start))}
    for i in range(target_count):
        required.update({f'transition-out-{i}': set(range(10)),
                         f'transition-in-{i}': set(range(10, 20)), f'target-{i}': set(range(start, count))})
    if coverage != required or found != set(range(count)):
        raise ValueError('完整原生相位、來源幀或空拍缺少實際Draw')
    if d.get('scenery_triple') is not None:
        triple=d['scenery_triple'];side=d.get('scenery_side')
        if triple not in ('4:5:6','5:6:4','6:4:5','7:9:10','34:36:37','40:43:47') or side not in ('0','2'):
            raise ValueError('明示場景測試槽不同')
        indices=list(map(int,triple.split(':')))
        wanted={('BG.DAT',i) for i in indices[:target_count+1]}
        wanted.update(('BG.DAT',i) for i in range(3) if ('BG.DAT',i) in surfaces)
        if side=='2':wanted.add(('TAI.DAT',indices[0]))
        if used_surfaces!=wanted:
            raise ValueError('缺少全部實際BG／TAI來源Draw')
    runtime_sha = None
    if runtime_receipt is not None:
        runtime = json.loads(runtime_receipt.read_text())
        for key in ('status', 'command_id', 'actor_figure', 'catalog_sha256', 'go_seed', 'go_rng_planning_calls',
                    'go_rng_next_matches_native_plan', 'native_rng', 'mp_before', 'mp_after', 'mp_changes',
                    'hp_before', 'hp_after', 'hp_changes', 'continuations', 'owner_released', 'gpu_sources_cleared',
                    'scenery_triple','scenery_side','scenery_actual_sources'):
            if runtime.get(key) != d.get(key):
                raise ValueError('執行包生命週期不同：'+key)
        if runtime.get('hashes_only') is not True or runtime.get('phase_f2') != d['phase_f2'] or runtime.get('source_shots') != shots:
            raise ValueError('執行包最終畫面／來源或相位不同')
        runtime_sha = digest(runtime_receipt)
    return {'status': 'PASS-SPLIT-STRIKE-FULL-GPU', 'command_id': command, 'actor_figure': figure,
            'receipt_sha256': digest(receipt), 'runtime_receipt_sha256': runtime_sha,
            'catalog_sha256': digest(catalog), 'inventory_sha256': digest(inventory),
            'original_manifest_sha256': digest(original_pack/'manifest.json'), 'verifier_sha256': digest(Path(__file__)),
            'native_metadata': native_proofs, 'surface_metadata':surface_proofs,'blank': blank, 'special_frames': sorted(found),
            'phase_indices': {p: sorted(v) for p, v in coverage.items()}, 'screens': outputs, 'worst_channel_error': worst,
            'scope': '實際Game.Draw最終1920全畫布對獨立Pillow Over及固定FIGANI描述子／成品；'
                     '原生背景與viewport幾何另由雙側來源矩陣驗證。非原版RGB parity、正常玩家或人工美術驗收。'}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ('receipt', 'catalog', 'source', 'original-pack', 'inventory', 'report'):
        parser.add_argument('--'+key, type=Path, required=True)
    parser.add_argument('--runtime-receipt', type=Path)
    args = parser.parse_args()
    if args.report.exists():
        raise SystemExit('不覆寫既有收據')
    result = verify(args.receipt, args.catalog, args.source, args.original_pack, args.inventory, args.runtime_receipt)
    args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': result['status'], 'command': result['command_id'], 'screens': len(result['screens'])}))


if __name__ == '__main__':
    main()
