#!/usr/bin/env python3
"""核對敵方指令9的實際Game.Draw核心、119幀來源與原生空拍。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from audit_fd2_command9_effect_source import audit
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches
from verify_fd2_compound_caster_gpu import make_native_occlusion_test

PHASES = ['prelude', 'actor', 'actor-pulse', 'command9-slide-in', 'command9-front', 'command9-target', 'command9-tail', 'command9-slide-out', 'tail']


def verify(receipt_path, catalog_path, source, original_pack, inventory_path):
    d = json.loads(receipt_path.read_text())
    if (d.get('status') != 'PASS-ORDINARY-COMMAND9-FIXTURE-RUNTIME-E1' or d.get('command_id') != 9
            or d.get('continuations') != 1 or d.get('mp_publications') != 1 or d.get('hp_publications') != 20
            or d.get('go_rng_unchanged') is not True
            or any(d.get('phase_f2_switches', {}).get(k) != 1 or d.get('hd_phase_draws', {}).get(k, 0) <= 0 for k in PHASES)):
        raise ValueError('需要指令9完整交易及每階段F2收據')
    catalog = validate(False, catalog_path)
    entries = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'}
    effects = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_effect_hd_set'}
    if 44 not in effects:
        raise ValueError('FDOTHER44缺件')
    native, _ = audit(original_pack)
    effect = effects[44]
    if (effect['frame_count'] != 31 or effect['source_metadata_sha256'] != native['metadata_sha256']
            or effect['consumer_contract'] != 'native_fdother_command9_ai_rgba_hd6_v1'):
        raise ValueError('FDOTHER44契約或來源不同')
    inventory = json.loads(inventory_path.read_text())
    occluded = make_native_occlusion_test(entries, source, original_pack, inventory, effects)
    samples = d.get('source_shots', {})
    for phase, start, count in [('command9-slide-in', 0, 11), ('command9-front', 11, 20), ('command9-target', 31, 60), ('command9-tail', 91, 20), ('command9-slide-out', 111, 8)]:
        expected = {f'{phase}-{i:03d}.png' for i in range(start, start+count)}
        actual = {k for k, v in samples.items() if v.get('phase') == phase}
        if actual != expected:
            raise ValueError('119 handler的來源影像不完整或重複')
    cache, rows, blanks, seen = {}, [], [], set()

    def core(archive, resource, index):
        key = (archive, resource, index)
        if key in cache:
            return cache[key]
        entry = (entries if archive == 'FIGANI.DAT' else effects).get(resource)
        if entry is None or not 0 <= index < entry['frame_count']:
            raise ValueError('實際Draw未綁完整HD來源')
        path = source/entry['files'][index]
        if digest(path) != entry['frame_sha256'][index]:
            raise ValueError('HD成品SHA不同')
        with Image.open(path) as im:
            frame = entry['native_frames'][index]
            if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6):
                raise ValueError('HD形狀不同')
            histogram = im.getchannel('A').histogram(); eligible = sum(histogram[250:])
            if key == ('FDOTHER.DAT', 44, 30):
                if histogram[0] != im.width*im.height or not native['frames'][30]['native_empty']:
                    raise ValueError('原生空拍新增墨跡')
                result = ([], frame, digest(path), eligible)
            else:
                if eligible < 1000:
                    raise ValueError('來源少於1000個高覆蓋核心')
                stride, n, points = max(1, eligible//6000), 0, []
                for at, rgba in enumerate(im.getdata()):
                    if rgba[3] >= 250:
                        if n % stride == 0:
                            points.append((at % im.width, at//im.width, *rgba))
                        n += 1
                result = (points, frame, digest(path), eligible)
        cache[key] = result
        return result

    for name, sample in samples.items():
        if Path(name).name != name or sample['phase'] not in PHASES:
            raise ValueError('來源影像名稱或階段不同')
        path = receipt_path.parent/name
        with Image.open(path) as im:
            if im.size != (1920, 1200):
                raise ValueError('需要最終1920×1200GPU影像')
            pixels = im.convert('RGB').load()
            for n, layer in enumerate(sample['layers']):
                archive = layer['archive'] or 'FIGANI.DAT'; key = (archive, layer['resource'], layer['frame'])
                if archive not in ['FIGANI.DAT', 'FDOTHER.DAT'] or layer['resource'] < 0 or layer.get('fill', -1) != -1:
                    raise ValueError('不能解釋的來源圖層')
                points, frame, sha, eligible = core(*key)
                if (layer['width'], layer['height']) != (frame['width'], frame['height']):
                    raise ValueError('GPU描述子不同')
                if key == ('FDOTHER.DAT', 44, 30):
                    if sample['phase'] != 'command9-target' or n != len(sample['layers'])-1:
                        raise ValueError('原生空拍排程不同')
                    blanks.append({'file': name, 'sha256': digest(path), 'native_mask_sha256': native['frames'][30]['mask_sha256'], 'source_sha256': sha})
                    seen.add(key); continue
                if sample['subtract'] != 0 or sample['mix'] != 0:
                    continue
                x, y = layer['x']*6, layer['y']*6
                visible = [p for p in points if 0 <= p[0]+x < 1920 and 0 <= p[1]+y < 1200 and not occluded(p[0]+x, p[1]+y, sample['layers'][n+1:])]
                if len(visible) < 1000:
                    continue
                if not matches(pixels, visible, x, y, 1920, 1200):
                    raise ValueError(f'最終GPU核心不同：{name}/{key}')
                rows.append({'file': name, 'sha256': digest(path), 'phase': sample['phase'], 'archive': archive, 'resource': key[1], 'frame': key[2],
                             'source_sha256': sha, 'checked_pixels': len(visible), 'source_eligible_pixels': eligible})
                seen.add(key)
    if {k[2] for k in seen if k[:2] == ('FDOTHER.DAT', 44)} != set(range(31)) or len(blanks) != 1:
        raise ValueError('未核對全部31特效來源與一次原生空拍')
    actor = d['actor_figure']*3+2
    source_row = inventory['resources'][actor]
    if source_row['status'] == 'empty_header_zero':
        actor -= 1
    elif source_row['status'] != 'decoded':
        raise ValueError('施術回退來源未知')
    cast_seen = {r['frame'] for r in rows if r['phase'] == 'actor' and r['archive'] == 'FIGANI.DAT' and r['resource'] == actor}
    if actor not in entries or cast_seen != set(range(entries[actor]['frame_count'])):
        raise ValueError('施術來源全幀核心不完整')
    if not any(r['file'] == blanks[0]['file'] and r['archive'] == 'FIGANI.DAT' and r['resource'] == d['target_figure']*3 for r in rows):
        raise ValueError('原生空拍同次Draw缺少目標可見核心')
    return {'status': 'PASS-COMMAND9-AI-FINAL-GPU', 'receipt_sha256': digest(receipt_path), 'catalog_sha256': digest(catalog_path),
            'verifier_sha256': digest(Path(__file__)), 'native_auditor_sha256': native['tool']['sha256'],
            'pixel_matcher_sha256': digest(Path(__file__).with_name('verify_fd2_ares_battle_gpu.py')),
            'occlusion_tool_sha256': digest(Path(__file__).with_name('verify_fd2_compound_caster_gpu.py')),
            'handler_frames': 119, 'effect_frames': 31, 'cast_frames': len(cast_seen), 'matches': rows, 'native_blank_draws': blanks,
            'scope': '完整Game.Draw fixture、F2及全部31效果／施術來源最終GPU，每份非空至少1000可見高覆蓋核心。原空拍另核固定Mask／alpha及同次目標核心。非正常玩家、原版RGB parity或PLAYER-E2。'}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for key in ['receipt', 'catalog', 'source', 'original-pack', 'inventory', 'report']:
        p.add_argument('--'+key, type=Path, required=True)
    args = p.parse_args()
    if args.report.exists():
        raise SystemExit('不覆寫既有收據')
    report = verify(args.receipt, args.catalog, args.source, args.original_pack, args.inventory)
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'handler_frames': 119, 'effect_frames': 31, 'cast_frames': report['cast_frames'], 'matches': len(report['matches'])}))


if __name__ == '__main__':
    main()
