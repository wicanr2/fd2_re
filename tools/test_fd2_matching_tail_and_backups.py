#!/usr/bin/env python3
"""完整尾呼叫匹配及對話備份負例回歸；不把接近的產碼升格成覆蓋。"""
import argparse
import hashlib
import json
from pathlib import Path

from fd2_matching_game_restore import BINDINGS,verify_original_bindings
from le_xref import parse_le,parse_fixups,file_to_linear


def target(i):
    code=bytes.fromhex(i['loaded_bytes']);at=int(i['ida_linear_address'],16)
    skip=2 if code[:1]==b'\x0f' else 1
    return at+len(code)+int.from_bytes(code[skip:],'little',signed=True)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('sound-linked','events-linked','backups-linked','original','evidence','output'):
        p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    raw=a.original.read_bytes();image=json.loads(a.evidence.read_text());sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
    assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']
    by={f['inventory']['start']:f for f in image['functions']};bytes_by={};points={}
    addresses={'0x3396a':324,'0x34c76':61,'0x35487':86,'0x165ac':688}
    for address,size in addresses.items():
        f=by[address];assert len(f['chunks'])==1 and f['inventory']['size']==size
        ins=f['chunks'][0]['instructions'];bytes_by[address]=b''.join(bytes.fromhex(i['loaded_bytes']) for i in ins)
        assert len(bytes_by[address])==size
        for i in ins:
            at=int(i['file_offset'],16);data=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(data)]==data
            points[int(i['ida_linear_address'],16)]=i
    positives={};reports={}
    for name,path,expected,count in (('sound',a.sound_linked,{'0x3396a'},9),
                                    ('events',a.events_linked,{'0x34c76','0x35487'},18),
                                    ('backups',a.backups_linked,set(),81)):
        report=json.loads((path/'report.json').read_text());reports[name]=sha(path/'report.json')
        assert report['input']==image['input'] and report['evidence_sha256']==sha(a.evidence)
        assert set(report['matched_addresses'])==expected and len(report['trials'])==count
        matched=[t for t in report['trials'] if t['exact_interval_bytes']]
        for t in matched:assert (path/t['stem']/'candidate.bin').read_bytes()==bytes_by[t['address']]
        positives[name]=len(matched)
    tails=[]
    for address,at,to in (('0x3396a',0x33aa9,0x1d4f6),('0x34c76',0x34cae,0x134e4),('0x35487',0x354d8,0x134e4)):
        assert points[at]['loaded_bytes'].startswith('e9') and target(points[at])==to
        tails.append({'function':address,'tail':hex(at),'target':hex(to)})
        binding=dict(BINDINGS);name='sub_1D4F6' if to==0x1d4f6 else 'sub_134E4';binding[name]+=1
        try:verify_original_bindings([by[address]],raw,binding)
        except ValueError:pass
        else:raise AssertionError('錯誤尾呼叫目標沒有拒收')
    assert points[0x33983]['loaded_bytes']=='6a58'
    player=by['0x1d4cb'];original80=next(i for c in player['chunks'] for i in c['instructions'] if i['ida_linear_address']=='0x1d4df')
    assert original80['loaded_bytes']=='6a50'
    for at,value in ((0x339f3,20),(0x33a1c,20),(0x33a45,20),(0x33a6e,60)):
        assert points[at]['loaded_bytes']=='6a'+f'{value:02x}'
    for at in (0x339fd,0x33a26,0x33a4f):assert points[at]['loaded_bytes']=='6858020000'
    assert points[0x34c8c]['loaded_bytes']=='c605fa3a050001' and points[0x34c9d]['loaded_bytes']=='c605fa3a050000'
    le=parse_le(raw);fixups=parse_fixups(raw,le);dispatch=[]
    for address,at,base,index,owner,call in (('0x3396a',0x51bd1,0x51d71,24,'0x25bf4','0x25e3a'),
            ('0x34c76',0x51a1d,0x51b91,35,'0x190ac','0x19511'),('0x35487',0x51a71,0x51b91,56,'0x190ac','0x19511')):
        assert fixups[at]==int(address,16) and file_to_linear(le,at)==base+index*4
        consumer=next(i for c in by[owner]['chunks'] for i in c['instructions'] if i['ida_linear_address']==call)
        code=bytes.fromhex(consumer['loaded_bytes']);offset=int(consumer['file_offset'],16)
        assert code[:3]==raw[offset:offset+3]==bytes.fromhex('ff1485')
        assert fixups[offset+3]==int.from_bytes(code[3:],'little')==base
        dispatch.append({'function':address,'entry_file_offset':hex(at),'entry_index':index,'table_base':hex(base),'consumer':call})
    for at in (0x16629,0x16641):assert points[at]['loaded_bytes']=='f7ff'
    for at in (0x16626,0x1663e):assert points[at]['loaded_bytes']=='c1fa1f'
    assert target(points[0x16610])==0x166cb and target(points[0x16699])==0x1661a
    assert points[0x1664f]['loaded_bytes']=='0fbf4206'
    assert target(points[0x166cd])==0x166e4 and points[0x166e4]['loaded_bytes']=='83fe05'
    assert points[0x166cf]['loaded_bytes']=='682c680000' and points[0x16853]['loaded_bytes']=='b8183a0500'
    assert [target(points[at]) for at in (0x16712,0x16758,0x1679e,0x167e4,0x1682a)]==[0x4e96f]*5
    assert [target(points[at]) for at in (0x1672e,0x16774,0x167ba,0x16800,0x16846)]==[0x168b6]*5
    rejected=[]
    for name in ('dword_53A18','dword_53A1C','dword_53A20','dword_53A24','dword_53A28','sub_15E9E','sub_4E96F'):
        binding=dict(BINDINGS);binding[name]+=1
        try:verify_original_bindings([by['0x165ac']],raw,binding)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    result={'schema_version':1,'complete_matched_functions':{k:addresses[k] for k in ('0x3396a','0x34c76','0x35487')},
        'new_matched_bytes':471,'tail_calls':tails,'indirect_dispatch_entries':dispatch,'exact_candidates':positives,
        'sound_resource_scope':{'transition':88,'player_command':80,'transition_wait_values':[20,20,20,60],'delay_arguments':[600]*3},
        'backup_original_contract':{'bytes':688,'signed_divisions':2,'sum_zero_bypass':'0x166cb','allocation_count':5,
            'allocation_bytes':26668,'capture_calls':5,'draw_calls':5,'return_address':'0x53a18','complete_c_match':False},
        'wrong_backup_bindings_rejected':rejected,'wrong_tail_targets_rejected':True,'source_reports_sha256':reports,
        'author_declarations_and_hardware_time':'unknown'}
    a.output.mkdir(parents=True,exist_ok=False);(a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
