#!/usr/bin/env python3
"""以原版LE重算匯出位元組，拒收偽造匹配並核對完整四函式負例。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

from fd2_matching_game_restore import verify_original_bindings


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects-10a','linked-10a','linked-95','original','evidence','output'):
        p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    image=json.loads(a.evidence.read_text());raw=a.original.read_bytes()
    assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']
    # 全部匯出指令由固定原檔與真正LE fixup重算，空符號清單不略過bytes守衛。
    verify_original_bindings(image['functions'],raw,{'__CHK':0x36cd7})
    total=sum(len(c['instructions']) for f in image['functions'] for c in f['chunks'])
    assert total==75427
    by_address={f['inventory']['start']:f for f in image['functions']}
    addresses=['0x22a85','0x22aa8','0x22af6','0x22bc6']
    functions=[by_address[x] for x in addresses]
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for f in functions for c in f['chunks'] for i in c['instructions'])
    assert len(original)==348
    baselines={}
    for version,path in (('10.0a',a.linked_10a),('9.5',a.linked_95)):
        report=json.loads((path/'report.json').read_text())
        assert report['compiler_version']==version and len(report['trials'])==261
        assert report['evidence_sha256']==hashlib.sha256(a.evidence.read_bytes()).hexdigest()
        t=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-dSTATE_GROUP_0'])
        code=(path/t['stem']/'candidate.bin').read_bytes()
        assert len(code)==348 and t['addresses']==addresses and t['exact_entry_offsets']
        assert not t['exact_interval_bytes'] and code[-27:]==original[-27:]
        baselines[version]=code
    assert baselines['9.5'][:113]==original[:113]
    assert original[106:108]==bytes.fromhex('0f85') and original[112]==0xc3
    newer=baselines['10.0a'];assert newer[106]==0x74 and newer[108]==0xe9
    assert 0x22a85+108+int.from_bytes(newer[107:108],'little',signed=True)==0x22aa7
    assert 0x22a85+113+int.from_bytes(newer[109:113],'little',signed=True)==0x1df58
    for code in baselines.values():
        diffs=[n for n,(left,right) in enumerate(zip(code[113:321],original[113:321])) if left!=right]
        assert diffs==[116,119,124],diffs
    f=by_address['0x22af6'];altered=copy.deepcopy(image)
    fake=next(f for f in altered['functions'] if f['inventory']['start']=='0x22af6')
    rejected=[]
    for address,code in (('0x22b68','0fb67320'),('0x22b6c','83fe08'),('0x22b71','83fe19')):
        one=copy.deepcopy(f)
        point=next(i for c in one['chunks'] for i in c['instructions'] if i['ida_linear_address']==address)
        point['loaded_bytes']=code
        try:verify_original_bindings([one],raw,{'__CHK':0x36cd7})
        except ValueError as error:assert 'LE' in str(error);rejected.append(address)
        else:raise AssertionError('偽造loaded_bytes沒有拒收')
        point=next(i for c in fake['chunks'] for i in c['instructions'] if i['ida_linear_address']==address)
        point['loaded_bytes']=code
    with tempfile.TemporaryDirectory(prefix='fd2-loaded-guard-') as temp:
        evidence=Path(temp)/'forged-ida.json';evidence.write_text(json.dumps(altered))
        output=Path(temp)/'must-not-exist'
        result=subprocess.run(['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link',
            '--objects',str(a.objects_10a),'--original',str(a.original),'--evidence',str(evidence),'--output',str(output)],
            capture_output=True,text=True,timeout=30)
        assert result.returncode and 'LE' in result.stderr and not output.exists(),result.stderr
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'original_loaded_instruction_records':total,'original_group_bytes':348,
        'original_function_intervals':{'0x22a85':35,'0x22aa8':78,'0x22af6':208,'0x22bc6':27},
        'newer_compiler_ret_share_target':'0x22aa7','older_compiler_preserves_original_branch':True,
        'remaining_register_difference_offsets':[116,119,124],
        'exact_27_byte_tail_not_accepted_as_group_match':True,'altered_loaded_bytes_rejected':rejected,
        'false_match_rejected_before_output':True,'original_input_sha256':hashlib.sha256(raw).hexdigest(),
        'source_report_sha256':{version:hashlib.sha256((path/'report.json').read_bytes()).hexdigest()
            for version,path in (('10.0a',a.linked_10a),('9.5',a.linked_95))}}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
