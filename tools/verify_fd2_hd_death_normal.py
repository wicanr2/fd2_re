#!/usr/bin/env python3
"""核對正常死亡事件PNG的可見不透明效果核心；不取代完整fixture或原版oracle。"""
import argparse
import json
from pathlib import Path

from PIL import Image, ImageChops
from verify_fd2_hd_death_gpu import checked, clipped, digest


def validate(catalog_path,pack,run):
    catalog=json.loads(catalog_path.read_text())
    bank=next(a for a in catalog['assets'] if a['role']=='map_overlay_hd_set')
    if ([d['index'] for d in bank['native_lmi_frames']]!=list(range(68,80)) or
            len(bank['files'])!=12 or len(bank['frame_sha256'])!=12):raise ValueError('incomplete source bank')
    effects={d['index']:checked(pack,dict(file=f,sha256=h)) for d,f,h in
             zip(bank['native_lmi_frames'],bank['files'],bank['frame_sha256'])}
    if any(effects[d['index']].size!=(d['width']*6,d['height']*6) for d in bank['native_lmi_frames']):
        raise ValueError('source geometry mismatch')
    receipt=json.loads((run/'receipt.json').read_text())
    if (receipt['schema']!='fd2-shared-death-normal-gui-v1' or receipt['status']!='PASS' or
            receipt['event_id']!=53 or receipt['state_injection'] or receipt['final_node']!='preparation_ch24' or
            not {'normal-attack-commander','normal-victory-confirm'} <= {p['action'] for p in receipt['progress']}):
        raise ValueError('normal event receipt rejected')
    accepted=[];rejected=[]
    for sample in receipt['samples']:
        actual=checked(run,sample).convert('RGB')
        if actual.size!=(1920,1200):raise ValueError('native HD window required')
        candidates=[]
        for state in (sample['before'],sample['after']):
            # Earlier versioned receipts predate explicit action/script metadata.
            if state.get('unit_death_kind','script')!=receipt.get('death_kind','script'):continue
            index=state.get('unit_death_presented_frame',-1)
            if not state.get('unit_death_presented_hd') or not 13<=index<21:continue
            phase='overlay_a' if index<19 else 'overlay_b'
            entries=list(range(68,68+index-12)) if index<19 else [74+index-19]
            viewport=(0,0,1872,1152)
            if tuple(state.get('unit_death_viewport',[]))!=viewport:raise ValueError('viewport mismatch')
            expected=[]
            for entry in entries:
                for target in state['unit_death_targets']:
                    origin=target['Origin'];x=(origin%456-0x8088%456)*6;y=(origin//456-0x8088//456)*6
                    picture=effects[entry];expected.append(dict(entry=entry,rect=[x,y,x+picture.width,y+picture.height]))
            if state['unit_death_layers']!=expected:raise ValueError('source order or origin mismatch')
            world=Image.new('RGBA',(1920,1200),(0,0,0,255));core=Image.new('L',world.size)
            for layer in expected:
                picture=effects[layer['entry']];x,y,x1,y1=layer['rect']
                clipped(world,picture,(x,y),viewport,go_rgba=True)
                opaque=picture.getchannel('A').point([255 if a==255 else 0 for a in range(256)])
                mask=Image.new('L',world.size)
                box=(max(x,0),max(y,0),min(x1,viewport[2]),min(y1,viewport[3]))
                if box[0]<box[2] and box[1]<box[3]:
                    mask.paste(opaque.crop((box[0]-x,box[1]-y,box[2]-x,box[3]-y)),box[:2])
                core=ImageChops.lighter(core,mask)
            # Native 1920x1200, integer positions, no scaling. Check every
            # opaque source pixel, including isolated alpha255 samples.
            count=core.histogram()[255]
            difference=ImageChops.difference(actual,world.convert('RGB'))
            maximum=ImageChops.lighter(ImageChops.lighter(difference.getchannel('R'),difference.getchannel('G')),difference.getchannel('B'))
            errors=ImageChops.multiply(core,maximum.point([255 if x>3 else 0 for x in range(256)])).histogram()[255]
            candidates.append(dict(frame=index,phase=phase,compared_pixels=count,rgb_errors_gt3=errors))
        passing=[c for c in candidates if c['compared_pixels']>=100 and c['rgb_errors_gt3']==0]
        (accepted if passing else rejected).append(dict(file=sample['file'],candidates=passing or candidates))
    phases={c['phase'] for sample in accepted for c in sample['candidates']}
    if phases!={'overlay_a','overlay_b'}:
        raise ValueError('missing independently matched phase '+str(phases)+' '+json.dumps(rejected))
    return dict(status='PASS-NORMAL-DEATH-OPAQUE-CORES',accepted=accepted,rejected=rejected,
                receipt_sha256=digest(run/'receipt.json'),catalog_sha256=digest(catalog_path),
                scope='只核正常X11事件抽樣的不透明來源核心、順序、原點與裁切；透明區、完整26拍及DOS parity另驗。')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    for name in ('catalog','pack','run','out'):parser.add_argument('--'+name,type=Path,required=True)
    args=parser.parse_args();result=validate(args.catalog,args.pack,args.run)
    args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(result['status'],len(result['accepted']))
