#!/usr/bin/env python3
"""以真實24／29／30畫面正對照核對生命週期、圖源與最終畫面拒收。"""
import argparse
import copy
import json
import os
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from verify_fd2_split_strike_gpu import verify


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'catalog', 'source', 'original-pack', 'inventory', 'out'):
        parser.add_argument('--'+name, type=Path, required=True)
    parser.add_argument('--require-flame-embers', action='store_true')
    a = parser.parse_args()
    if not Path('/.dockerenv').exists() or a.out.exists() or a.out.parent.stat().st_uid != os.getuid():
        raise SystemExit('在Docker使用本人擁有的新輸出目錄')
    a.out.mkdir()
    call = [a.catalog, a.source, a.original_pack, a.inventory]
    base = json.loads(a.receipt.read_text())
    require_embers = a.require_flame_embers or base.get('flame_embers') == 'native-repeat-v1'
    positive = verify(a.receipt, *call, require_flame_embers=require_embers)
    (a.out/'positive.json').write_text(json.dumps(positive, ensure_ascii=False, indent=2)+'\n')
    results = []

    def reject(name, mutate, expected, corrupt=None):
        folder = a.out/name
        folder.mkdir()
        for shot in base['source_shots'].values():
            files = [shot['file'], shot['base_file']]+[l['native_file'] for l in shot['layers'] if 'native_file' in l]
            files += [l['panel_file'] for l in shot['layers'] if 'panel_file' in l]
            for filename in files:
                path = folder/filename
                if not path.exists():
                    path.symlink_to((a.receipt.parent/filename).resolve())
        doc = copy.deepcopy(base)
        mutate(doc)
        if corrupt:
            corrupt(doc, folder)
        path = folder/'receipt.json'
        path.write_text(json.dumps(doc))
        try:
            verify(path, *call, require_flame_embers=require_embers)
        except ValueError as error:
            if expected not in str(error):
                raise AssertionError((name, str(error))) from error
            results.append({'case': name, 'status': 'PASS-REJECTED', 'reason': str(error)})
        else:
            raise AssertionError('反例被接受：'+name)

    for name, mutate in [
        ('wrong-command', lambda d: d.update(command_id=0)),
        ('missing-f2', lambda d: d['phase_f2'].update(actor=0)),
        ('missing-continuation', lambda d: d.update(continuations=0)),
        ('retained-owner', lambda d: d.update(owner_released=False)),
        ('changed-mp', lambda d: d.update(mp_after=d['mp_after']+1)),
        ('missing-hp', lambda d: d.update(hp_changes=[0])),
        ('extra-rng', lambda d: d.update(go_rng_planning_calls=7)),
        ('wrong-seed', lambda d: d.update(go_seed=999)),
    ]:
        reject(name, mutate, '需要24' if name == 'wrong-command' else '生命週期')
    first = next(iter(base['source_shots']))
    def figure(doc):
        return next(l for l in doc['source_shots'][first]['layers'] if l.get('archive','') in ('','FIGANI.DAT') and l['resource']>=0)
    reject('wrong-screen-sha', lambda d: d['source_shots'][first].update(sha256='bad'), '圖片路徑或SHA')
    reject('changed-origin', lambda d: figure(d).update(dx=1), '目的矩形')
    reject('changed-descriptor', lambda d: figure(d).update(x=0, rect=[0, 0, 6, 6]), '目的矩形')
    reject('changed-hd-source', lambda d: figure(d).update(hd_sha256='bad'), 'HD來源SHA')
    reject('hidden-hd', lambda d: figure(d).update(clip=[0, 0, 0, 0]), '完整GPU')
    if require_embers:
        particle_shot = next(name for name, shot in base['source_shots'].items() if shot.get('particles'))
        def particle(doc):
            return doc['source_shots'][particle_shot]['particles'][0]
        reject('missing-flame-contract', lambda d: d.pop('flame_embers'), '連續火星契約')
        reject('missing-active-ember-f2', lambda d: d['flame_embers_f2'].update({'0': 0}), '火星F2')
        reject('missing-particle', lambda d: d['source_shots'][particle_shot]['particles'].pop(), '火星數量')
        reject('wrong-particle-sha', lambda d: particle(d).update(source_sha256='bad'), '火星來源')
        reject('wrong-particle-crop', lambda d: particle(d).update(source_crop=[213, 30, 226, 46]), '火星來源')
        reject('wrong-particle-position', lambda d: particle(d)['rect'].__setitem__(0, particle(d)['rect'][0]+1), '火星位置')
        reject('wrong-particle-opacity', lambda d: particle(d).update(opacity=0), '火星透明度')
        reject('nan-particle-opacity', lambda d: particle(d).update(opacity=float('nan')), '火星透明度')
        blank = next(name for name, shot in base['source_shots'].items()
                     if shot['phase'] == 'target-0' and shot['cursor']['Effect'] == 17)
        reject('particle-on-native-blank', lambda d: d['source_shots'][blank].update(
            particles=copy.deepcopy(d['source_shots'][particle_shot]['particles'])), '非火星相位')
        def missing_repeat(doc):
            doc['source_shots'] = {name: shot for name, shot in doc['source_shots'].items()
                                   if not (shot['phase'] == 'target-0' and shot['cursor']['Effect'] == 14
                                           and shot['cursor']['Repeat'] == 1)}
        reject('missing-particle-repeat', missing_repeat, '連續火星節拍')
    catalog = json.loads(a.catalog.read_text())
    extended = {asset['source_resource']: asset for asset in catalog['assets']
                if asset.get('role') == 'battle_animation_hd_set'
                and asset.get('consumer_contract') == 'native_figani_rgba_hd6_v2'}
    for shot_name, shot in base['source_shots'].items():
        selected = next((i for i, layer in enumerate(shot['layers'])
                         if layer.get('archive', '') in ('', 'FIGANI.DAT') and layer['resource'] in extended
                         and layer['rect'] != [(layer['x']+layer['dx'])*6, (layer['y']+layer['dy'])*6,
                                               (layer['x']+layer['dx']+layer['width'])*6,
                                               (layer['y']+layer['dy']+layer['height'])*6]), None)
        if selected is not None:
            def native_box(doc):
                layer = doc['source_shots'][shot_name]['layers'][selected]
                layer['rect'] = [(layer['x']+layer['dx'])*6, (layer['y']+layer['dy'])*6,
                                 (layer['x']+layer['dx']+layer['width'])*6, (layer['y']+layer['dy']+layer['height'])*6]
            reject('native-box-for-extended-hd', native_box, '目的矩形')
            break
    panel_layers = [(name, i) for name, shot in base['source_shots'].items()
                    for i, layer in enumerate(shot['layers']) if 'panel_file' in layer]
    if panel_layers:
        panel_shot, panel_index = panel_layers[0]
        def panel(doc):
            return doc['source_shots'][panel_shot]['layers'][panel_index]
        if require_embers:
            reject('panel-not-screen-fixed', lambda d: panel(d).update(screen_fixed=False), '固定HUD')
        reject('panel-sprite-palette', lambda d: panel(d).update(preserve_palette=False), '面板來源')
        def changed_panel_origin(doc):
            layer=panel(doc);layer['x']+=1;layer['rect'][0]+=6;layer['rect'][2]+=6
        def changed_panel_size(doc):
            layer=panel(doc);layer['width']-=1;layer['rect'][2]-=6
        reject('panel-wrong-origin', changed_panel_origin, '面板來源')
        reject('panel-wrong-size', changed_panel_size, '面板來源')
        reject('panel-missing-values', lambda d: panel(d).pop('panel_values'), '凍結數值')
        reject('panel-noninteger-values', lambda d: panel(d)['panel_values'].update(HP=1.5), '凍結數值')
        reject('panel-wrong-content-key', lambda d: panel(d).update(hd_sha256='panel-text/bad'), 'RGBA內容')
        reject('panel-wrong-png-sha', lambda d: panel(d).update(panel_sha256='bad'), '圖片路徑或SHA')
        reject('panel-outside-path', lambda d: panel(d).update(panel_file='../outside.png'), '圖片路徑或SHA')
    if any(l.get('archive') in ('BG.DAT','TAI.DAT') for l in base['source_shots'][first]['layers']):
        def surface(doc):return next(l for l in doc['source_shots'][first]['layers'] if l.get('archive')=='BG.DAT')
        reject('wrong-surface-sha',lambda d:surface(d).update(hd_sha256='bad'),'場景原描述子')
        reject('surface-frame-out-of-range',lambda d:surface(d).update(frame=1),'場景來源或單圖')
        reject('hidden-surface',lambda d:surface(d).update(clip=[0,0,0,0]),'完整GPU')
        def changed_surface_origin(doc):
            l=surface(doc);l['y']+=1;l['rect'][1]+=6;l['rect'][3]+=6
        reject('self-consistent-surface-origin',changed_surface_origin,'場景原描述子')

    transitions=[(name,i) for name,shot in base['source_shots'].items()
                 for i,layer in enumerate(shot['layers'])
                 if layer.get('archive')=='BG.DAT' and layer['resource'] in (0,1,2)]
    if transitions:
        shot_name,layer_index=transitions[0]
        def transition(doc):return doc['source_shots'][shot_name]['layers'][layer_index]
        reject('wrong-transition-sha',lambda d:transition(d).update(hd_sha256='bad'),
               '場景原描述子' if transition(base)['hd_sha256'] else '未登錄過渡')
        def changed_transition_origin(doc):
            layer=transition(doc);layer['y']+=1;layer['rect'][1]+=6;layer['rect'][3]+=6
        reject('self-consistent-transition-origin',changed_transition_origin,'場景原描述子')
        if not transition(base)['hd_sha256']:
            def corrupt_native_transition(doc,folder):
                layer=transition(doc);path=folder/layer['native_file'];path.unlink()
                picture=Image.open(a.receipt.parent/layer['native_file']).convert('RGBA')
                r,g,b,alpha=picture.getpixel((0,0));picture.putpixel((0,0),(r^255,g,b,alpha));picture.save(path)
                layer['native_sha256']=digest(path)
            reject('self-consistent-native-transition-rgb',lambda d:None,'原生過渡回退與固定來源',corrupt_native_transition)

    def missing_blank(doc):
        end = {24: 14, 29: 17, 30: 13}[doc['command_id']]
        doc['source_shots'] = {k: v for k, v in doc['source_shots'].items()
                               if not (v['phase'].startswith('target-') and v['cursor']['Effect'] == end)}
    reject('missing-native-blank', missing_blank, '來源幀或空拍缺少')
    reject('missing-transition', lambda d: d.update(source_shots={k: v for k, v in d['source_shots'].items()
           if not (v['phase'] == 'transition-in-0' and v['cursor']['Transition'] == 19)}), '來源幀或空拍缺少')

    def corrupt_screen(doc, folder):
        shot = doc['source_shots'][first]
        path = folder/shot['file']
        path.unlink()
        Image.new('RGB', (1920, 1200), (0, 255, 0)).save(path)
        shot['sha256'] = digest(path)
    reject('self-consistent-wrong-gpu', lambda d: None, '完整GPU', corrupt_screen)
    report = {'status': 'PASS-SPLIT-STRIKE-GPU-GUARDS', 'command_id': base['command_id'],
              'positive_sha256': digest(a.out/'positive.json'), 'checks': results,
              'scope': '真實最終GPU與固定原圖正對照的拒收，不提升正常玩家或原版parity。'}
    (a.out/'guards.json').write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({'status': report['status'], 'cases': len(results)}))


if __name__ == '__main__':
    main()
