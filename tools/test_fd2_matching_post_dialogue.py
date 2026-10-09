#!/usr/bin/env python3
"""以完整原始函式核對17-byte備份、章節副作用與真正間接派送表項。"""
import argparse
import hashlib
import json
from pathlib import Path

from fd2_matching_game_restore import BINDINGS,verify_original_bindings
from le_xref import parse_le,parse_fixups,file_to_linear


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    raw=a.original.read_bytes();image=json.loads(a.evidence.read_text());report=json.loads((a.linked/'report.json').read_text())
    assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']==report['input']['sha256']
    assert hashlib.sha256(a.evidence.read_bytes()).hexdigest()==report['evidence_sha256']
    assert report['matched_addresses']==['0x24754'] and len(report['trials'])==27
    f=next(f for f in image['functions'] if f['inventory']['start']=='0x24754')
    ins=[i for c in f['chunks'] for i in c['instructions']];points={i['ida_linear_address']:i for i in ins}
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for i in ins);assert len(original)==f['inventory']['size']==960
    for i in ins:
        at=int(i['file_offset'],16);data=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(data)]==data
    positive=[t for t in report['trials'] if t['exact_interval_bytes']]
    for t in positive:assert (a.linked/t['stem']/'candidate.bin').read_bytes()==original
    for n in range(3):
        t=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-dPOST_DIALOGUE_'+str(n)]);assert t['exact_interval_bytes']
    assert points['0x24761']['loaded_bytes']=='83ec3c'
    for at in ('0x24764','0x24775','0x24786'):assert points[at]['loaded_bytes']=='b904000000'
    for at in ('0x24772','0x24783','0x24792'):assert points[at]['loaded_bytes']=='f3a5'
    for at in ('0x24774','0x24785','0x24794'):assert points[at]['loaded_bytes']=='a4'
    assert points['0x24951']['loaded_bytes']=='ff35793a0500'
    assert points['0x24957']['loaded_bytes']=='ff05033c0500'
    call=bytes.fromhex(points['0x2495d']['loaded_bytes']);assert 0x2495d+5+int.from_bytes(call[1:],'little',signed=True)==0x15f84
    assert points['0x24a36']['loaded_bytes']=='83c302' and points['0x24a39']['loaded_bytes']=='83fb40'
    loop=bytes.fromhex(points['0x24a3c']['loaded_bytes']);assert 0x24a3c+2+int.from_bytes(loop[1:],'little',signed=True)==0x24a1c
    le=parse_le(raw);fixups=parse_fixups(raw,le)
    assert fixups[0x51c41]==0x24754 and file_to_linear(le,0x51c41)==0x51de9+22*4
    consumer=next(i for owner in image['functions'] for c in owner['chunks'] for i in c['instructions'] if i['ida_linear_address']=='0x25e23')
    assert consumer['loaded_bytes']=='ff1485e91d0500'
    rejected=[]
    for name in ('unk_522A3','unk_522B4','unk_522C5','sub_24B14','sub_24BDE','sub_10652'):
        binding=dict(BINDINGS);binding[name]+=1
        try:verify_original_bindings([f],raw,binding)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_original_function_bytes':960,'original_local_stack_bytes':60,
        'table_copy_bytes':[17,17,17],'table_stack_offsets':[40,20,0],'exact_candidates':len(positive),
        'chapter_increment_after_context_push_before_call':True,'fade_step':2,'fade_limit':64,'fade_loop_target':'0x24a1c',
        'dispatch_entry':{'entry_file_offset':'0x51c41','table_base':'0x51de9','entry_index':22,'consumer':'0x25e23'},
        'wrong_symbol_addresses_rejected':rejected,'author_source_and_expression_order':'unknown',
        'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest()}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
