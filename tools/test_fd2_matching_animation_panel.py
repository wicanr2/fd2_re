#!/usr/bin/env python3
"""核對完整演出／介面負例、堆疊實參與callee實際保存，拒絕猜補初值。"""
import argparse
import hashlib
import json
from pathlib import Path
import re

from fd2_matching_game_restore import BINDINGS,verify_original_bindings


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('linked-10a','linked-95','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    raw=a.original.read_bytes();image=json.loads(a.evidence.read_text())
    assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']
    functions={f['inventory']['start']:f for f in image['functions']}
    selected=[functions[x] for x in ('0x28784','0x2ebe0')]
    original={f['inventory']['start']:b''.join(bytes.fromhex(i['loaded_bytes']) for c in f['chunks'] for i in c['instructions']) for f in selected}
    assert {k:len(v) for k,v in original.items()}=={'0x28784':744,'0x2ebe0':943}
    references=verify_original_bindings(selected,raw,BINDINGS)
    for version,path in (('10.0a',a.linked_10a),('9.5',a.linked_95)):
        report=json.loads((path/'report.json').read_text())
        assert report['compiler_version']==version and len(report['trials'])==117 and report['matched_addresses']==[]
        assert report['evidence_sha256']==hashlib.sha256(a.evidence.read_bytes()).hexdigest()
        assert report['original_binding_references']==references
        for t in report['trials']:
            code=(path/t['stem']/'candidate.bin').read_bytes()
            assert not t['exact_interval_bytes'] and code!=original[t['address']]
    points={i['ida_linear_address']:i for f in image['functions'] for c in f['chunks'] for i in c['instructions']}
    assert points['0x28792']['loaded_bytes']=='83ec24' and points['0x2ebee']['loaded_bytes']=='83ec38'
    # PUSH變更ESP；不同運算元的來源是同一個caller局部slot 0x18。
    first=bytes.fromhex(points['0x288e5']['loaded_bytes']);second=bytes.fromhex(points['0x288e9']['loaded_bytes'])
    assert first==bytes.fromhex('ff742424') and second==bytes.fromhex('ff742428')
    assert first[-1]-12==second[-1]-16==24
    call=bytes.fromhex(points['0x288f3']['loaded_bytes']);assert 0x288f3+5+int.from_bytes(call[1:],'little',signed=True)==0x29164
    jump=bytes.fromhex(points['0x288fb']['loaded_bytes']);assert jump[0]==0xeb
    assert 0x288fb+2+int.from_bytes(jump[1:],'little',signed=True)==0x28960
    assert points['0x28963']['loaded_bytes']=='39df'
    assert points['0x29170']['loaded_bytes']=='57' and points['0x2927a']['loaded_bytes']=='5f'
    assert points['0x29113']['loaded_bytes']=='5f' and points['0x29116']['loaded_bytes']=='c3'
    # 實際保存與compiler允許修改分開記錄，不將modify宣告當作clobber證據。
    assert points['0x4df51']['loaded_bytes']=='53' and points['0x4dfbb']['loaded_bytes']=='5b'
    assert points['0x4e8b2']['loaded_bytes']=='60' and points['0x4e8de']['loaded_bytes']=='61'
    assert points['0x4e642']['loaded_bytes']=='53'
    for at in ('0x4e6f6','0x4e76d','0x4e7f3'):assert points[at]['loaded_bytes']=='5b'
    panel=[i for c in functions['0x2ebe0']['chunks'] for i in c['instructions']]
    body=[i for i in panel if i['instruction'].split()[0] not in ('push','pop')]
    assert not any(re.search(r'\b(?:ebx|bx|bl|bh)\b',i['instruction']) for i in body)
    # 原始4個容量由16-bit讀取，不因局部C型別改成32-bit來源。
    assert [points[x]['loaded_bytes'] for x in ('0x2ec5c','0x2ec64','0x2ec6c','0x2ec74')]==[
        '0fb74648','0fb7464a','0fb7464c','0fb7464e']
    rejected=[]
    for name in ('dword_54107','dword_54117','aBgDat','aFiganiDat','aTaiDat','aFdshapDat','sub_2EFB7','sub_2EF8F','sub_29164','sub_2BC9A'):
        binding=dict(BINDINGS);binding[name]+=1
        try:verify_original_bindings(selected,raw,binding)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_original_bytes':{'0x28784':744,'0x2ebe0':943},'complete_total_bytes':1687,
        'candidates':234,'exact_candidates':0,'coverage_increase':0,'original_local_stack_bytes':{'0x28784':36,'0x2ebe0':56},
        'constructor_original_operands':['[esp+24h]','[esp+28h]'],'constructor_effective_local_offsets':[24,24],
        'first_loop_target':'0x28960','incoming_edi_producer':'unknown','initial_zero_not_assumed':True,
        'callee_29164_edi_restore_sites':['0x2927a','0x29113'],
        'graphics_callees_preserve_ebx':['0x4df4c','0x4e8af','0x4e63d'],
        'compiler_clobber_permission_is_not_actual_clobber':True,'panel_capacity_load_bits':16,
        'wrong_symbol_addresses_rejected':rejected,'original_binding_references':len(references),
        'source_reports_sha256':{v:hashlib.sha256((p/'report.json').read_bytes()).hexdigest() for v,p in (('10.0a',a.linked_10a),('9.5',a.linked_95))}}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
