#!/usr/bin/env python3
"""核對四召喚與普通指令0的施術者服裝核心；fixture與普通玩家收據分開驗收。"""
import argparse
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate
from verify_fd2_ares_battle_gpu import matches


def make_native_occlusion_test(entries, source, original_pack, inventory):
    native_masks = {}

    def occluded(x, y, layers):
        for layer in layers:
            ox, oy = x-layer['x']*6, y-layer['y']*6
            if not (0 <= ox < layer['width']*6 and 0 <= oy < layer['height']*6): continue
            if layer['resource'] < 0 or layer['archive'] not in ['', 'FIGANI.DAT']:
                raise SystemExit('不能解釋的前景圖層遮擋')
            clip=layer.get('clip')
            if clip and not (clip['Min']['X']*6<=x<clip['Max']['X']*6 and clip['Min']['Y']*6<=y<clip['Max']['Y']*6):continue
            key = (layer['resource'], layer['frame'])
            if key not in native_masks:
                resource, index = key
                asset = entries.get(resource)
                if asset is not None:
                    path = source/asset['files'][index]
                    if digest(path) != asset['frame_sha256'][index]: raise SystemExit('前景HD圖源SHA不同')
                    with Image.open(path) as im: mask = im.getchannel('A').copy()
                    native_masks[key] = (mask, True)
                else:
                    row = inventory['resources'][resource]
                    folder = original_pack/'animations'/f'FIGANI_{resource:03d}'
                    metadata = folder/'animation.json'
                    if row['resource'] != resource or digest(metadata) != row['metadata_sha256']: raise SystemExit('原生前景metadata不同')
                    frame = json.loads(metadata.read_text())['frames'][index]
                    path = folder/frame['mask_path']
                    if digest(path) != row['frames'][index]['mask_sha256']: raise SystemExit('原生前景mask SHA不同')
                    with Image.open(path) as im: mask = im.convert('L')
                    native_masks[key] = (mask, False)
            mask, hd = native_masks[key]
            if mask.size != (layer['width']*(6 if hd else 1), layer['height']*(6 if hd else 1)):
                raise SystemExit('前景遮罩形狀不同')
            if mask.getpixel((ox, oy) if hd else (ox//6, oy//6)) != 0: return True
        return False
    return occluded


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--receipt', type=Path, action='append', required=True)
    p.add_argument('--figure', type=int, default=4)
    p.add_argument('--core-profile', choices=['blue','gold','opaque'], default='blue', help='服裝色域或全部alpha≥250可見核心；RGB與至少1000點門檻不變')
    p.add_argument('--normal-command0', action='store_true', help='核對普通標題LOAD的指令0施法、自然返回與保存冷讀；不冒充fixture')
    p.add_argument('--command0', action='store_true', help='核對普通指令0的前導、施術與尾段；不套四召喚的滑出契約')
    p.add_argument('--command5', action='store_true', help='核對普通指令5的完整交易、F2及施術來源')
    p.add_argument('--command1', action='store_true', help='核對普通指令1的完整交易、F2及施術來源')
    p.add_argument('--normal-command1', action='store_true', help='核對普通標題LOAD的指令1施法、自然返回與保存冷讀')
    p.add_argument('--ordinary-command', type=int, choices=[2,3,7,8], help='核對指定普通法術的完整交易、F2與施術來源')
    p.add_argument('--normal-ordinary-command', type=int, choices=[2,3,7,8], help='核對指定普通法術的正常標題LOAD及保存冷讀')
    p.add_argument('--normal-end-command', type=int, choices=[0,4,5], help='核對正常END實際施法及完整返回／保存冷讀')
    p.add_argument('--all-cast-frames', action='store_true', help='明示要求每張施術來源都有actor階段最終GPU核心，不將非零相位抽樣冒充全幀')
    for name in ['catalog', 'source', 'report']:
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--original-pack', type=Path, required=True)
    p.add_argument('--inventory', type=Path, required=True)
    args = p.parse_args()
    if sum([args.command0,args.command1,args.command5,args.normal_command0,args.normal_command1,args.normal_end_command is not None,
            args.ordinary_command is not None,args.normal_ordinary_command is not None])>1:
        raise SystemExit('fixture與普通玩家模式不可混用')
    normal=args.normal_command0 or args.normal_command1 or args.normal_ordinary_command is not None or args.normal_end_command is not None
    command_id=(args.normal_end_command if args.normal_end_command is not None else args.ordinary_command if args.ordinary_command is not None
                else args.normal_ordinary_command if args.normal_ordinary_command is not None else 5 if args.command5 else 1 if args.command1 or args.normal_command1 else 0)
    ordinary=args.command0 or args.command1 or args.command5 or args.ordinary_command is not None or normal
    if args.report.exists(): raise SystemExit('不覆寫既有報告')
    catalog = validate(False, args.catalog)
    inventory = json.loads(args.inventory.read_text())
    if digest(args.original_pack/'manifest.json') != inventory['pack_manifest_sha256']:
        raise SystemExit('遮擋來源包指紋不同')
    entries = {a['source_resource']: a for a in catalog['assets'] if a['role'] == 'battle_animation_hd_set'}
    idle, actor = args.figure*3, args.figure*3+1
    phases=['prelude','actor','actor-slide']
    if ordinary:
        phases=['prelude','actor','tail']
        if args.normal_end_command is not None:phases.append('target-idle')
        actor=args.figure*3+2
        row=inventory['resources'][actor]
        if row['resource']!=actor:raise SystemExit('盤點位置不符')
        if row['status']=='empty_header_zero':
            status_path=args.original_pack/'animations'/f'FIGANI_{actor:03d}'/'resource.json'
            status=json.loads(status_path.read_text())
            if (digest(status_path)!=row['status_sha256'] or status.get('reason_code')!='zero_header_word'
                or status.get('header_word_le')!=0 or status.get('source')!={'file':'FIGANI.DAT','resource':actor}):
                raise SystemExit('施術圖回退來源不符')
            actor-=1
        elif row['status']!='decoded':raise SystemExit('未閉合的施術圖來源')
    if idle not in entries or actor not in entries: raise SystemExit('缺少完整施術者圖源')
    core = {}
    for resource in [idle, actor]:
        a = entries[resource]
        for index, (name, sha, frame) in enumerate(zip(a['files'], a['frame_sha256'], a['native_frames'])):
            path = args.source/name
            if digest(path) != sha: raise SystemExit('施術者圖源SHA不同')
            with Image.open(path) as im:
                if im.mode != 'RGBA' or im.size != (frame['width']*6, frame['height']*6): raise SystemExit('施術者圖源形狀不同')
                points = [(i % im.width, i // im.width, r, g, b, alpha)
                          for i, (r, g, b, alpha) in enumerate(im.getdata())
                           if alpha >= 250 and (args.core_profile=='opaque' or ((b >= 70 and b*100 >= r*150 and b*100 >= g*130) if args.core_profile=='blue'
                              else (r>=100 and g>=70 and r>=g and g*100>=b*150 and r*100>=b*180)))]
            if len(points) < 1000: raise SystemExit('本圖源不適用指定色域核心核對')
            core[(resource, index)] = (name, sha, points, frame)
    occluded=make_native_occlusion_test(entries,args.source,args.original_pack,inventory)
    reports = []
    for receipt in args.receipt:
        d = json.loads(receipt.read_text())
        if not normal and (not d.get('status', '').startswith('PASS') or d.get('continuations') != 1 or not d.get('go_rng_unchanged')):
            raise SystemExit('需要完整RunGame交易收據')
        if args.command0 and (d.get('status')!='PASS-ORDINARY-COMMAND0-FIXTURE-RUNTIME-E1' or d.get('command_id')!=0
            or d.get('mp_publications')!=1 or d.get('hp_publications')!=7
            or any(d.get('phase_f2_switches',{}).get(phase)!=1 for phase in ['prelude','actor','actor-pulse','target','tail'])):
            raise SystemExit('需要完整普通指令0交易／F2收據')
        if args.command5 and (d.get('status')!='PASS-ORDINARY-COMMAND5-FIXTURE-RUNTIME-E1' or d.get('command_id')!=5
            or d.get('mp_publications')!=1 or d.get('hp_publications')!=6
            or any(d.get('phase_f2_switches',{}).get(phase)!=1 for phase in ['prelude','actor','actor-pulse','handler-front','target','handler-tail','tail'])):
            raise SystemExit('需要完整普通指令5交易／F2收據')
        if args.command1 and (d.get('status')!='PASS-ORDINARY-COMMAND1-FIXTURE-RUNTIME-E1' or d.get('command_id')!=1
            or d.get('mp_publications')!=1 or d.get('hp_publications')!=8
            or any(d.get('phase_f2_switches',{}).get(phase)!=1 for phase in ['prelude','actor','actor-pulse','command1-target','tail'])):
            raise SystemExit('需要完整普通指令1交易／F2收據')
        if args.ordinary_command is not None and (d.get('status')!=f'PASS-ORDINARY-COMMAND{command_id}-FIXTURE-RUNTIME-E1' or d.get('command_id')!=command_id
            or d.get('mp_publications')!=1 or d.get('hp_publications')!={2:6,3:13,7:5,8:16}[command_id]
            or any(d.get('phase_f2_switches',{}).get(phase)!=1 for phase in ['prelude','actor','actor-pulse','handler-front','target','handler-tail','tail'])):
            raise SystemExit(f'需要完整普通指令{command_id}交易／F2收據')
        samples=d.get('source_shots', {})
        if args.normal_command0 or args.normal_command1 or args.normal_ordinary_command is not None:
            if (d.get('status')!='PASS' or d.get('schema')!='fd2-hd-battle-status-gui-v1'
                or d.get('battle_command_id')!=command_id or d.get('battle_action_kind')!=f'command{command_id}'
                or not d.get('natural_attack_return_checked') or not d.get('post_battle_save_checked')
                or not d.get('new_process_title_load') or not d.get('theme_save_bytes_equal')
                or not d.get('node_boundary_save_checked') or not d.get('cold_loaded_units_equal')
                or not d.get('loaded_units') or d['loaded_units']!=d.get('cold_units')
                or d.get('catalog_sha256')!=digest(args.catalog)):
                raise SystemExit(f'需要完整普通指令{command_id}及邊界保存收據')
            samples={}
            for row in d.get('battle_art_samples',[]):
                name=row['file']
                if Path(name).name!=name or digest(receipt.parent/name)!=row['sha256']:raise SystemExit('普通GPU圖片路徑或SHA不同')
                before,after=row['before'],row['after']
                if not all(q.get('command_scene') and q.get('command_id')==command_id and q.get('hd_battle') for q in [before,after]):continue
                if before.get('command_owner')!=after.get('command_owner'):continue
                for n,state in enumerate([before,after]):
                    phase=state.get('command_phase')
                    if phase not in ['actor','tail'] and not (phase=='prelude' and state.get('command_frame')==8):continue
                    samples[(name,n)]={'file':name,'phase':phase,'subtract':0,'mix':0,'layers':state.get('command_source_layers',[])}
        if args.normal_end_command is not None:
            if (d.get('status')!='PASS' or d.get('schema')!='fd2-hd-battle-status-gui-v1'
                or command_id not in d.get('battle_end_commands',[])
                or not all(d.get(k) for k in ['natural_attack_return_checked','post_battle_save_checked',
                    'new_process_title_load','theme_save_bytes_equal','node_boundary_save_checked','cold_loaded_units_equal'])
                or not d.get('loaded_units') or d['loaded_units']!=d.get('cold_units')
                or d.get('catalog_sha256')!=digest(args.catalog)):
                raise SystemExit('需要完整普通END施法及邊界保存收據')
            samples={}
            for row in d.get('battle_end_command_samples',[]):
                name=row['file']
                if Path(name).name!=name or digest(receipt.parent/name)!=row['sha256']:raise SystemExit('普通GPU圖片路徑或SHA不同')
                before,after=row['before'],row['after']
                if not all(q.get('command_scene') and q.get('command_id')==command_id and q.get('hd_battle') for q in [before,after]):continue
                if before.get('command_owner')!=after.get('command_owner'):continue
                for n,state in enumerate([before,after]):
                    phase=state.get('command_phase')
                    if phase not in ['actor','tail']:continue
                    samples[(name,n)]={'file':name,'phase':phase,'subtract':0,'mix':0,'layers':state.get('command_source_layers',[])}
            for row in d.get('battle_art_samples',[]):
                name=row['file']
                if Path(name).name!=name or digest(receipt.parent/name)!=row['sha256']:raise SystemExit('普通GPU圖片路徑或SHA不同')
                before,after=row['before'],row['after']
                if not all(q.get('battle_scene') and q.get('hd_battle') and q.get('battle_target_fig')==args.figure for q in [before,after]):continue
                if before.get('battle_owner')!=after.get('battle_owner'):continue
                for n,state in enumerate([before,after]):
                    samples[(name,'target',n)]={'file':name,'phase':'target-idle','subtract':0,'mix':0,
                        'layers':state.get('battle_source_layers',[])}
        found = []
        for sample_name, sample in samples.items():
            name=sample.get('file',sample_name)

            if sample['phase'] not in phases or sample['subtract'] != 0 or sample['mix'] != 0: continue
            if Path(name).name != name: raise SystemExit('GPU圖片路徑越界')
            path = receipt.parent/name
            with Image.open(path) as im:
                if im.size != (1920, 1200): raise SystemExit('需要1920×1200最終GPU')
                rgb = im.convert('RGB').load()
                for layer_index, layer in enumerate(sample['layers']):
                    key = (layer['resource'], layer['frame'])
                    if layer['archive'] not in ['', 'FIGANI.DAT'] or key not in core: continue
                    source, sha, all_points, frame = core[key]
                    if (layer['width'], layer['height']) != (frame['width'], frame['height']): raise SystemExit('GPU來源描述子不同')
                    points = [p for p in all_points if not occluded(p[0]+layer['x']*6, p[1]+layer['y']*6, sample['layers'][layer_index+1:])]
                    visible_count = len(points)
                    if visible_count < 1000: continue
                    if args.core_profile!='opaque':points = points[::max(1, visible_count//2500)]
                    if matches(rgb, points, layer['x']*6, layer['y']*6, 1920, 1200):
                        found.append({'file': name, 'sha256': digest(path), 'phase': sample['phase'],
                                      'resource': key[0], 'frame': key[1], 'source_file': source,
                                      'source_sha256': sha, 'checked_pixels': len(points),
                                      'visible_eligible_pixels': visible_count, 'source_eligible_pixels': len(all_points),
                                      'excluded_by_source_occlusion': len(all_points)-visible_count})
        required=(['actor','target-idle'] if args.normal_end_command is not None else ['actor'] if normal else phases)
        for phase in required:
            if (not normal and not d.get('phase_f2_switches', {}).get(phase)) or not any(x['phase'] == phase for x in found):
                raise SystemExit('缺少必要施術者階段的實際HD核心或指定F2：'+phase)
        if args.normal_end_command is not None and not any(x['phase']=='target-idle' and x['resource']==idle for x in found):
            raise SystemExit('沒有核對到同份正常流程的受擊待機')
        if not any(x['phase'] == 'actor' and x['resource'] == actor and x['frame'] > 0 for x in found):
            raise SystemExit('沒有核對到非零施術者動作')
        if args.all_cast_frames:
            expected=set(range(entries[actor]['frame_count']))
            actual={x['frame'] for x in found if x['phase']=='actor' and x['resource']==actor}
            if actual!=expected:raise SystemExit('施術來源全幀GPU核心不完整')
        reports.append({'receipt': str(receipt), 'receipt_sha256': digest(receipt), 'matches': found})
    status=(f'PASS-ORDINARY-COMMAND{command_id}-NORMAL-END-CASTER-GPU' if args.normal_end_command is not None else
            f'PASS-ORDINARY-COMMAND{command_id}-NORMAL-CASTER-GPU' if args.normal_ordinary_command is not None else
            f'PASS-ORDINARY-COMMAND{command_id}-CASTER-GPU' if args.ordinary_command is not None else
            'PASS-ORDINARY-COMMAND5-CASTER-GPU' if args.command5 else
            'PASS-ORDINARY-COMMAND1-CASTER-GPU' if args.command1 else
            f'PASS-ORDINARY-COMMAND{command_id}-NORMAL-CASTER-GPU' if args.normal_command0 or args.normal_command1 else
            'PASS-ORDINARY-COMMAND0-CASTER-GPU' if args.command0 else 'PASS-COMPOUND-CASTER-GPU')
    report = {'status':status, 'figure': args.figure,
              'command_id':command_id if ordinary else None,
              'core_profile':args.core_profile, 'actor_resource':actor, 'expected_phases':['actor','target-idle'] if args.normal_end_command is not None else ['actor'] if normal else phases,
              'catalog_sha256': digest(args.catalog), 'verifier_sha256': digest(Path(__file__)),
              'pixel_matcher_sha256': digest(Path(__file__).with_name('verify_fd2_ares_battle_gpu.py')),
              'occlusion_inventory_sha256': digest(args.inventory),
              'all_cast_frames_required':args.all_cast_frames,
              'reports': reports, 'scope': (f'普通標題LOAD及END指令{command_id}實際視窗的施術者核心，含自然返回／邊界保存冷讀，完整階段另見fixture。' if normal else f'普通指令{command_id}最終GPU的prelude／actor／tail高覆蓋服裝核心，限明示fixture。' if ordinary else '正式RunGame最終GPU的prelude／actor／actor-slide高覆蓋服裝核心，限明示fixture。')+
              '只排除已核SHA的前景原生mask或HD alpha實際遮住的像素，剩餘須至少1000個核心。'
              '相近圖源不判唯一相位，武器／FX及其他施法者另驗；不宣稱原版oracle或整章E2。fixture不當普通玩家證據。'}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'receipts': len(reports), 'matched_pngs': sum(len({x['file'] for x in r['matches']}) for r in reports)}))


if __name__ == '__main__': main()
