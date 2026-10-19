#!/usr/bin/env python3
"""核對普通GUI的144圖源細節與雙主題取景；不是原版oracle或逐幀parity。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def compare_unit_source(actual, source, rect, coverage):
    left, top, right, bottom = rect
    assert source.size == (144, 144) and actual.size == (1920, 1200)
    checked = different = near_checked = near_different = 0
    for y in range(144):
        for x in range(144):
            pixel = source.getpixel((x, y))
            if not (24 <= left+x < 1896 and 24 <= top+y < 1176):
                continue
            actual_pixel = actual.getpixel((left+x, top+y))
            if coverage and 250 <= pixel[3] < 255:
                alpha = pixel[3]
                near_checked += 1
                near_different += any(value < channel*alpha//255-1 or
                    value > (channel*alpha+254)//255+255-alpha+1
                    for value, channel in zip(actual_pixel, pixel[:3]))
            if pixel[3] == 255:
                checked += 1
                different += actual_pixel != pixel[:3]
    return {'opaque_core_pixels': checked, 'different_pixels': different,
            'near_opaque_pixels': near_checked, 'outside_source_over_bounds': near_different,
            'accepted': checked+near_checked > 3000 and not different and not near_different}


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ('receipt','catalog','source','report'):
        parser.add_argument('--'+name,type=Path,required=True)
    parser.add_argument('--group', type=int, default=0, help='核對指定角色的實際144圖源；預設索爾0')
    parser.add_argument('--gaia-focus', action='store_true', help='只驗蓋亞補抽的固定三張角色圖源；不套用舊原生地形對照')
    args=parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not args.report.exists() and args.report.parent.stat().st_uid==1000
    receipt=json.loads(args.receipt.read_text())
    assert receipt['status']=='PASS' and receipt['world_runtime_checked'] is True
    if args.gaia_focus:
        assert args.group == 30 and receipt['schema'] == 'fd2-hd-gaia-focus-gui-v1'
        assert receipt['new_process_title_load'] and receipt['source_save_bytes_equal']
    catalog=json.loads(args.catalog.read_text())
    assets={a['source_group']:a for a in catalog['assets'] if a['role']=='map_sprite_hd_set'}
    assert len(assets)==125
    assert args.group in assets, 'unknown or invisible source group'
    shots=receipt['screenshots']
    samples=[]
    # Exact mode keeps the recorded phase. Gaia focus is explicitly an art
    # identity check over all three declared cycles in the same direction;
    # it never upgrades a different phase to same-state evidence.
    sample_names = (('gaia-hd-focused.png', 'gaia-hd-focused-restored.png', 'gaia-hd-cold-focused.png')
                    if args.gaia_focus else
                    ('battle-hd-focused.png','battle-hd-focused-restored.png','world-hd-source-reloaded.png'))
    for name in sample_names:
        record=shots[name]
        path=args.receipt.parent/name
        assert digest(path)==record['sha256']
        before,after=record['gpu_before'],record['state']
        first=[u for u in before.get('hd_world_units',[]) if u['group']==args.group]
        last=[u for u in after.get('hd_world_units',[]) if u['group']==args.group]
        if not before.get('hd_world') or not after.get('hd_world') or len(first)!=1 or first!=last:
            samples.append({'file':name,'status':'NOT-COMPARABLE','reason':'source phase/position changed during GPU capture'})
            continue
        unit=first[0]
        left,top,right,bottom=unit['rect']
        assert right-left==bottom-top==144
        offset=unit['work_offset']
        assert left==(offset%456-0x8088%456+4)*6
        assert top==(offset//456-0x8088//456+4)*6
        asset=assets[args.group]
        actual=Image.open(path).convert('RGB')
        phases = range(unit['frame']//3*3, unit['frame']//3*3+3) if args.gaia_focus else [unit['frame']]
        candidates=[]
        for phase in phases:
            source_name=asset['files'][phase]
            assert Path(source_name).name==source_name
            source_path=args.source/source_name
            assert digest(source_path)==asset['frame_sha256'][phase]
            source=Image.open(source_path).convert('RGBA')
            comparison=compare_unit_source(actual,source,unit['rect'],args.gaia_focus)
            candidates.append(dict(comparison,source=source_name,source_frame=phase,
                                   source_sha256=digest(source_path)))
        accepted=[c for c in candidates if c['accepted']]
        assert len(accepted)==1, candidates
        sample=dict(accepted[0],file=name,status='PASS',rect=unit['rect'],reported_frame=unit['frame'])
        if args.gaia_focus:
            sample.update(comparison='same-direction registered cycle identity',
                          same_state_frame=sample['source_frame']==unit['frame'],phase_candidates=candidates)
        samples.append(sample)
    assert sum(s['status']=='PASS' for s in samples)>=2,'insufficient stable GPU source samples'
    if args.gaia_focus:
        result = {'status': 'PASS', 'source_group': 30, 'source_samples': samples,
                  'receipt_sha256': digest(args.receipt), 'catalog_sha256': digest(args.catalog),
                  'verifier_sha256': digest(Path(__file__)),
                  'scope': 'Gaia art identity among three registered cycles in the reported direction; alpha255 RGB exact, alpha250..254 source-over bounds with unknown backdrop and one quantization code value. Phase differences disclosed, not same-state frame parity; no original oracle.',
                  'blend_reference':'https://ebitengine.org/en/examples/blur.html'}
        args.report.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
        print(json.dumps(result, ensure_ascii=False))
        return
    hd_name='world-hd-source-reloaded.png'
    hd,faithful=shots[hd_name],shots['battle-faithful-focused.png']
    assert hd['state']['view']==faithful['state']['view']
    h=Image.open(args.receipt.parent/hd_name).convert('RGB')
    f=Image.open(args.receipt.parent/'battle-faithful-focused.png').convert('RGB')
    assert h.size==(1920,1200) and f.size==(1280,800)
    terrain_difference=sum(h.getpixel((x*6+3,y*6+3))!=f.getpixel((x*4+2,y*4+2))
                           for y in range(4,9) for x in range(4,196))
    result={'status':'PASS','scope':'actual remake GPU versus registered 144 source; faithful remake geometry only, no original oracle',
            'source_group':args.group,
            'receipt_sha256':digest(args.receipt),'catalog_sha256':digest(args.catalog),
            'verifier_sha256':digest(Path(__file__)),'source_samples':samples,
            'same_view':True,'fixed_static_terrain_samples':960,'terrain_different_pixels':terrain_difference,
            'normal_walk_samples':len(receipt['world_walk_gpu']),
            'physical_scene_and_return':all(n in shots for n in ('world-hd-physical-scene.png','world-hd-physical-return.png'))}
    assert result['normal_walk_samples']>0 and result['physical_scene_and_return']
    assert terrain_difference==0,'static terrain coordinates/color differ across themes'
    args.report.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
