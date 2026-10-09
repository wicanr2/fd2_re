#!/usr/bin/env python3
"""核對完整六模式分派、共用返回及原始四位元組記錄的+7寫入。"""
import argparse
import hashlib
import json
from pathlib import Path

from fd2_matching_game_restore import BINDINGS,verify_original_bindings


def branch_target(i):
    code=bytes.fromhex(i['loaded_bytes']);at=int(i['ida_linear_address'],16)
    return at+len(code)+int.from_bytes(code[2:] if code[:2]==b'\x0f\x85' else code[1:],'little',signed=True)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    raw=a.original.read_bytes();image=json.loads(a.evidence.read_text());report=json.loads((a.linked/'report.json').read_text())
    assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']==report['input']['sha256']
    assert hashlib.sha256(a.evidence.read_bytes()).hexdigest()==report['evidence_sha256']
    assert report['matched_addresses']==['0x122dc'] and len(report['trials'])==18
    f=next(f for f in image['functions'] if f['inventory']['start']=='0x122dc')
    assert len(f['chunks'])==1 and f['inventory']['size']==1051
    ins=f['chunks'][0]['instructions'];points={int(i['ida_linear_address'],16):i for i in ins}
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for i in ins)
    for i in ins:
        at=int(i['file_offset'],16);code=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(code)]==code
    positive=[t for t in report['trials'] if t['exact_interval_bytes']]
    assert positive
    for t in positive:assert (a.linked/t['stem']/'candidate.bin').read_bytes()==original
    edges=[]
    for mode,at,fail in ((1,0x122e6,0x122fc),(2,0x122fc,0x12309),(3,0x12309,0x12385),
                         (4,0x12385,0x124b8),(5,0x124b8,0x126d0),(6,0x126d0,0x126f6)):
        assert points[at]['loaded_bytes']=='833d831a0500'+f'{mode:02x}'
        assert branch_target(points[at+7])==fail
        edges.append({'mode':mode,'compare':hex(at),'otherwise':hex(fail)})
    calls=[i for i in ins if bytes.fromhex(i['loaded_bytes'])[:1]==b'\xe8']
    targets=[branch_target(i) for i in calls]
    assert targets.count(0x126f7)==37 and targets.count(0x36cd7)==1
    assert points[0x12384]['loaded_bytes']==points[0x126f6]['loaded_bytes']=='c3'
    assert branch_target(points[0x126cb])==0x1237c
    assert points[0x126de]['loaded_bytes']=='0faf05c13a0500'
    assert points[0x126e5]['loaded_bytes']=='0305b13a0500'
    assert points[0x126eb]['loaded_bytes']=='8b15513a0500'
    assert points[0x126f1]['loaded_bytes']=='c644820700'
    callers=[]
    for owner,at in (('0x11cac',0x11cf0),('0x18b84',0x18bf7)):
        caller=next(f for f in image['functions'] if f['inventory']['start']==owner)
        i=next(i for c in caller['chunks'] for i in c['instructions'] if int(i['ida_linear_address'],16)==at)
        assert branch_target(i)==0x122dc
        offset=int(i['file_offset'],16);assert raw[offset:offset+5]==bytes.fromhex(i['file_bytes'])
        callers.append({'owner':owner,'call':hex(at)})
    rejected=[]
    for name in ('sub_126F7','dword_51A83','dword_53AB1','dword_53AB5','dword_53AC1','dword_53A51'):
        binding=dict(BINDINGS);binding[name]+=1
        try:verify_original_bindings([f],raw,binding)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_original_function_bytes':1051,'six_mode_edges':edges,
        'explicit_helper_calls':37,'mode5_shared_call_target':'0x1237c','raw_writer':'0x126f1',
        'record_byte_expression':'(dword_53AB5*dword_53AC1+dword_53AB1)*4+7',
        'original_direct_callers':callers,'wrong_symbol_addresses_rejected':rejected,'exact_candidates':len(positive),
        'author_declarations_and_source':'unknown','source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest()}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
