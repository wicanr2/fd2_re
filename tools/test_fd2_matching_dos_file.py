#!/usr/bin/env python3
"""核對五個完整原始函式、EBX跨呼叫存活與新增符號拒收。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile

from fd2_matching_game_restore import BINDINGS,verify_original_bindings


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text());image=json.loads(a.evidence.read_text());raw=a.original.read_bytes()
    expected={'0x361cc':137,'0x36284':109,'0x36344':311,'0x36900':85,'0x36955':173}
    assert hashlib.sha256(a.evidence.read_bytes()).hexdigest()==report['evidence_sha256']
    assert hashlib.sha256(raw).hexdigest()==report['input']['sha256']==image['input']['sha256']
    assert report['matched_addresses']==list(expected) and len(report['trials'])==126
    functions=[f for f in image['functions'] if f['inventory']['start'] in expected]
    originals={}
    for f in functions:
        address=f['inventory']['start'];instructions=[i for c in f['chunks'] for i in c['instructions']]
        originals[address]=b''.join(bytes.fromhex(i['loaded_bytes']) for i in instructions)
        assert len(originals[address])==f['inventory']['size']==expected[address]
        for i in instructions:
            at=int(i['file_offset'],16);data=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(data)]==data
    for t in (t for t in report['trials'] if t['exact_interval_bytes']):
        assert (a.linked/t['stem']/'candidate.bin').read_bytes()==originals[t['address']]
    macros=('RAW_ALLOC_2','RAW_RANGE_2','RAW_LX_1','RAW_LENGTH_2','RAW_LOAD_2')
    for macro in macros:
        t=next(t for t in report['trials'] if t['flags']==['-mf','-4s','-d'+macro]);assert t['exact_interval_bytes']
    assert all(not t['exact_interval_bytes'] for t in report['trials'] if t['flags'][-1][2:] not in macros)
    # 原始兩次無號判定重用旗標；合併成單一if的等長控制仍有差異。
    points={i['ida_linear_address']:i for f in functions for c in f['chunks'] for i in c['instructions']}
    assert points['0x36291']['loaded_bytes']=='7304' and points['0x36299']['loaded_bytes']=='7302'
    assert points['0x362e8']['loaded_bytes']=='25ff000000'
    assert originals['0x36344'][3:9]==bytes.fromhex('81eccc000000')
    assert points['0x36443']['loaded_bytes']=='03bc24ac000000'
    assert originals['0x36955'][:4]==bytes.fromhex('565731d2')
    assert originals['0x36955'][-5:]==bytes.fromhex('89d85f5ec3')
    # 被呼叫函式的局部保存證據不外推完整transitive clobber集合。
    for addr in ('0x3d056','0x3cc4b'):
        f=next(f for f in image['functions'] if f['inventory']['start']==addr)
        ins=[i for c in f['chunks'] for i in c['instructions']]
        assert ins[0]['loaded_bytes']=='53' and ins[-2]['loaded_bytes']=='5b' and ins[-1]['loaded_bytes']=='c3'
    rejected=[]
    for name in ('open','_open','filelength','_filelength','close','_close','sub_36107','_sub_36107','dword_360FF','aLx','off_5275C'):
        binding=dict(BINDINGS);binding[name]+=1
        try:verify_original_bindings(functions,raw,binding)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    with tempfile.TemporaryDirectory(prefix='fd2-dos-file-test-') as temp:
        altered=copy.deepcopy(image)
        f=next(f for f in altered['functions'] if f['inventory']['start']=='0x36344')
        point=next(i for c in f['chunks'] for i in c['instructions'] if i['ida_linear_address']=='0x3634d')
        point['loaded_bytes']='a1fe600300'
        evidence=Path(temp)/'altered.json';evidence.write_text(json.dumps(altered))
        output=Path(temp)/'must-not-exist'
        result=subprocess.run(['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link',
            '--objects',str(a.objects),'--original',str(a.original),'--evidence',str(evidence),'--output',str(output)],
            capture_output=True,text=True,timeout=30)
        assert result.returncode and 'LE' in result.stderr and not output.exists(),result.stderr
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_function_bytes':expected,'complete_total_bytes':sum(expected.values()),
        'exact_candidates':sum(t['exact_interval_bytes'] for t in report['trials']),
        'range_unsigned_branches':['0x36291','0x36299'],'lx_local_stack_bytes':204,
        'load_data_pointer_survives_close':True,'unmatched_controls_preserved':True,
        'callee_ebx_save_restore':['0x3d056','0x3cc4b'],'complete_transitive_clobbers':'unknown',
        'binding_alias_rejected':rejected,'altered_le_rejected_before_output':True,'classification_changed':False,
        'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest()}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
