#!/usr/bin/env python3
"""獨立核對音速刃正常鍵盤、來源、交易與實際HD人物核心。"""
import argparse
import copy
import json
from pathlib import Path

from PIL import Image
from build_fd2_gaia_battle_hd import digest
from verify_fd2_ares_battle_gpu import matches


def require(value, message):
    if not value:raise ValueError(message)


def verify(a, override=None, substitute=None):
    report=json.loads(a.receipt.read_text()) if override is None else override
    require(report.get('status')=='PASS' and report.get('schema')=='fd2-hd-battle-status-gui-v1', '缺正常GUI收據')
    for flag in ('battle_hd_consumer_checked','natural_attack_return_checked','new_process_title_load',
                 'theme_save_bytes_equal','post_battle_save_checked','node_boundary_save_checked','cold_loaded_units_equal'):
        require(report.get(flag) is True,'未完成'+flag)
    require(report['recording_created'] is False and report['battle_command_id']==30
            and report['battle_own_identity']==18 and report['battle_repetitions']==1,'指令、身分或抽樣範圍不同')
    for key,path in [('catalog',a.catalog),('binary',a.binary),('source_save',a.save),('gui_tool',a.gui_tool)]:
        require(report[key+'_sha256']==digest(path),key+'來源不同')
    source=json.loads(a.save.read_text());manifest=json.loads(a.manifest.read_text())
    require(manifest['status']=='PASS-CONTROLLED-COMMAND30-GUI-SOURCE'
            and manifest['output_sha256']==digest(a.save) and manifest['direct_command_grant'] is False
            and manifest['normal_recruitment_proven'] is False,'缺受控來源揭露')
    mask=source['party_roster']['18']['native_command_mask']
    require(mask==[0,0,0,64,0] and source['party_deploy']['18'] is True,'來源沒有正式學到30')
    saved=a.receipt.parent/'user-data/fd2_re/fd2_save.json'
    require(report['saved_sha256']==digest(saved),'冷讀實檔不同')
    require(json.loads(saved.read_text())['party_roster']['18']['native_command_mask']==mask,'保存遺失已學技能')
    observations=report['command30_observations'];toggles=report['command30_theme_checks']
    require(len(observations)==len(toggles)==1,'缺一次完整施放與F2')
    observation,toggle=observations[0],toggles[0]
    require(observation['status']=='PASS-COMMAND30-NORMAL-OBSERVATION'
            and observation['mp_sequence']==[29,5] and observation['target_count']>0,'MP或目標交易不同')
    require(len(observation['hp_changes'])==observation['target_count']
            and all(len(rows)==1 and rows[0][0]>rows[0][1] for rows in observation['hp_changes'].values()),'傷害不是各一次')
    cancelled=report['screenshots']['world-hd-command30-cancelled-target.png']['state']
    require(cancelled['command_grid'] and next(u for u in cancelled['units'] if u['camp']==0 and u['native_identity']==18)['mp']==29,
            '取消未返回指令框或已扣MP')
    audit=a.receipt.parent/'input-audit-first-process.jsonl'
    require(audit.is_file(),'缺首次程序原始audit')
    active=[]
    for line in audit.read_text().splitlines():
        row=json.loads(line)
        if row.get('command_owner')==observation['owner'] and row.get('command_id')==30 and row.get('command_scene'):active.append(row)
    require(len(active)==observation['audit_rows'],'原始audit的owner筆數不同')
    mp=[];changes={};previous=None
    for row in active:
        actor=next(u for u in row['units'] if u['camp']==0 and u['native_identity']==18)
        if not mp or mp[-1]!=actor['mp']:mp.append(actor['mp'])
        hp=[u['hp'] for u in row['units']]
        if previous is not None:
            require(len(previous)==len(hp),'原始audit的名冊索引漂移')
            for index,(before,after) in enumerate(zip(previous,hp)):
                if before!=after:changes.setdefault(str(index),[]).append([before,after])
        previous=hp
    require(mp==observation['mp_sequence'] and changes==observation['hp_changes'],'原始audit與交易摘要不同')
    require(toggle['entered']['command_id']==toggle['native']['command_id']==toggle['restored']['command_id']==30
            and toggle['entered']['command_owner']==toggle['native']['command_owner']==toggle['restored']['command_owner']
            and not toggle['native']['hd_theme'] and toggle['restored']['hd_battle'],'F2改變owner或缺少兩主題')
    require(digest(a.receipt.parent/toggle['file'])==toggle['sha256'],'原生PNG雜湊不同')
    catalog=json.loads(a.catalog.read_text())
    banks={x['source_resource']:x for x in catalog['assets'] if x['role']=='battle_animation_hd_set' and x['source_resource'] in (54,56)}
    require(set(banks)=={54,56},'缺正式蜜蒂待機／效果銀行')
    candidates={}
    for resource,asset in banks.items():
        frames=[]
        for index,(name,sha,frame) in enumerate(zip(asset['files'],asset['frame_sha256'],asset['native_frames'])):
            path=a.pack/name;require(Path(name).name==name and digest(path)==sha,'正式圖源雜湊不同')
            with Image.open(path) as raw:
                im=raw.convert('RGBA');require(im.size==(frame['width']*6,frame['height']*6),'正式圖源尺寸不同')
                pixels=im.load();points=[]
                for y in range(0,im.height,3):
                    for x in range(0,im.width,3):
                        red,green,blue,alpha=pixels[x,y]
                        if alpha>=250:points.append((x,y,red,green,blue,alpha))
                points=points[::max(1,len(points)//5000)]
            frames.append((index,frame,points,name,sha))
        candidates[resource]=frames
    verified=[];unmatched=[]
    for sample in observation['samples']:
        path=a.receipt.parent/sample['file'];require(digest(path)==sample['sha256'],'實際HD PNG雜湊不同')
        with Image.open(substitute or path) as image:
            actual=image.convert('RGB')
            if substitute is not None and actual.size!=(1920,1200):
                require(actual.width*5==actual.height*8,'反例長寬比不同')
                actual=actual.resize((1920,1200),Image.Resampling.NEAREST)
            require(actual.size==(1920,1200),'不是完整遊戲視窗')
        compatible=[]
        for state in (sample['before'],sample['after']):
            require(state.get('command_id')==30 and state.get('command_owner')==observation['owner'],'觀測跨owner')
            layers=state.get('command_source_layers') or []
            for position,layer in enumerate(layers):
                resource=layer.get('resource')
                # Native FIGANI layers use the empty archive discriminator;
                # panel layers also use it, with resource -1, excluded here.
                if layer.get('archive')!='' or resource not in candidates or layer.get('packed_shader'):
                    continue
                current=banks[resource]['native_frames'][layer['frame']]
                dx,dy=layer['x']-current['x'],layer['y']-current['y']
                clip=layer.get('clip') or {};low,high=clip.get('Min',{}),clip.get('Max',{})
                clip_rect=(low.get('X',0)*6,low.get('Y',0)*6,high.get('X',320)*6,high.get('Y',200)*6)
                if clip_rect[0]>=clip_rect[2] or clip_rect[1]>=clip_rect[3]:clip_rect=(0,0,1920,1200)
                occluders=[(p['x']*6,p['y']*6,(p['x']+p['width'])*6,(p['y']+p['height'])*6) for p in layers[position+1:]]
                for index,frame,points,name,sha in candidates[resource]:
                    ox,oy=(frame['x']+dx)*6,(frame['y']+dy)*6
                    core=[point for point in points if max(0,clip_rect[0])<=point[0]+ox<min(1920,clip_rect[2])
                          and max(0,clip_rect[1])<=point[1]+oy<min(1200,clip_rect[3])
                          and not any(x<=point[0]+ox<x1 and y<=point[1]+oy<y1 for x,y,x1,y1 in occluders)]
                    if len(core)>=1000 and matches(actual.load(),core,ox,oy,1920,1200):
                        compatible.append(dict(resource=resource,frame=index,source=name,sha256=sha,core_points=len(core),x=ox,y=oy))
        if compatible:verified.append(dict(file=sample['file'],sha256=sample['sha256'],compatible=compatible))
        else:unmatched.append(sample['file'])
    require(any(c['resource']==56 for sample in verified for c in sample['compatible']),'沒有實際音速刃效果／動作核心')
    return dict(status='PASS-COMMAND30-NORMAL-SCOPED',verified=verified,unmatched_observations=unmatched,
                mp_sequence=observation['mp_sequence'],hp_changes=observation['hp_changes'],
                source_sha256=digest(a.save),binary_sha256=digest(a.binary),catalog_sha256=digest(a.catalog),
                receipt_sha256=digest(a.receipt),audit_sha256=digest(audit),
                scope='受控加入／升級及章前強化來源的普通鍵盤、F2、施放／返回與章節邊界冷讀；相容候選明示，不要求每幀原版一致。')


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('receipt','catalog','pack','binary','save','manifest','gui-tool','out'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--self-test',action='store_true');a=p.parse_args()
    require(Path('/.dockerenv').is_file() and not a.out.exists() and a.out.parent.is_dir(),'限Docker與全新收據')
    report=verify(a)
    if a.self_test:
        original=json.loads(a.receipt.read_text());rejected=[]
        for kind in ('wrong-command','wrong-mp','wrong-damage','missing-cold','wrong-source','wrong-owner','actual-native'):
            bad=copy.deepcopy(original);substitute=None
            if kind=='wrong-command':bad['battle_command_id']=29
            elif kind=='wrong-mp':bad['command30_observations'][0]['mp_sequence']=[29,29]
            elif kind=='wrong-damage':bad['command30_observations'][0]['hp_changes']={}
            elif kind=='missing-cold':bad['new_process_title_load']=False
            elif kind=='wrong-source':bad['source_save_sha256']='0'*64
            elif kind=='wrong-owner':bad['command30_theme_checks'][0]['restored']['command_owner']='other'
            else:substitute=a.receipt.parent/bad['command30_theme_checks'][0]['file']
            try:verify(a,bad,substitute)
            except (ValueError,KeyError,AssertionError):rejected.append(kind)
            else:raise ValueError('未拒收反例：'+kind)
        report['rejected_counterexamples']=rejected
    a.out.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(status=report['status'],verified_pngs=len(report['verified']),unmatched=len(report['unmatched_observations'])),ensure_ascii=False))


if __name__=='__main__':main()
