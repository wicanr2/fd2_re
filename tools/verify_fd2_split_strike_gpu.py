#!/usr/bin/env python3
"""核對24／29完整最終GPU、固定FIGANI來源及原生空拍。須在資產Docker執行。"""
import argparse
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageMath
from build_fd2_character_battle_hd import DESCRIPTOR, load_native
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_compound_caster_gpu import pinned_native_blank

INVENTORY_SHA = 'd019e2cb233d8bbe8d6b6d688221fdc2fecadc78bf39142e5e3c2f6b57375f51'


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
    d = json.loads(receipt.read_text())
    command = d.get('command_id')
    if command not in (24, 29):
        raise ValueError('需要24／29收據')
    figure, count, start, target_count, seed, mp_cost = (32, 15, 9, 1, 2, 22) if command == 24 else (34, 18, 11, 2, 29, 26)
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
            if rid >= 0:
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
    runtime_sha = None
    if runtime_receipt is not None:
        runtime = json.loads(runtime_receipt.read_text())
        for key in ('status', 'command_id', 'actor_figure', 'catalog_sha256', 'go_seed', 'go_rng_planning_calls',
                    'go_rng_next_matches_native_plan', 'native_rng', 'mp_before', 'mp_after', 'mp_changes',
                    'hp_before', 'hp_after', 'hp_changes', 'continuations', 'owner_released', 'gpu_sources_cleared'):
            if runtime.get(key) != d.get(key):
                raise ValueError('執行包生命週期不同：'+key)
        if runtime.get('hashes_only') is not True or runtime.get('phase_f2') != d['phase_f2'] or runtime.get('source_shots') != shots:
            raise ValueError('執行包最終畫面／來源或相位不同')
        runtime_sha = digest(runtime_receipt)
    return {'status': 'PASS-SPLIT-STRIKE-FULL-GPU', 'command_id': command, 'actor_figure': figure,
            'receipt_sha256': digest(receipt), 'runtime_receipt_sha256': runtime_sha,
            'catalog_sha256': digest(catalog), 'inventory_sha256': digest(inventory),
            'original_manifest_sha256': digest(original_pack/'manifest.json'), 'verifier_sha256': digest(Path(__file__)),
            'native_metadata': native_proofs, 'blank': blank, 'special_frames': sorted(found),
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
