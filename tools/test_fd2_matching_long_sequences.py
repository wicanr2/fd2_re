#!/usr/bin/env python3
"""核對完整開場／戰後C、8-bit計數、讀取順序與原始呼叫綁定。"""
import argparse
import hashlib
import json
from pathlib import Path

from fd2_matching_game_restore import BINDINGS,verify_original_bindings
from le_xref import parse_le,parse_fixups,file_to_linear


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text());image=json.loads(a.evidence.read_text());raw=a.original.read_bytes()
    expected={'0x2548c':715,'0x3231b':1626}
    assert hashlib.sha256(a.evidence.read_bytes()).hexdigest()==report['evidence_sha256']
    assert hashlib.sha256(raw).hexdigest()==report['input']['sha256']==image['input']['sha256']
    assert report['matched_addresses']==list(expected) and len(report['trials'])==54
    functions=[f for f in image['functions'] if f['inventory']['start'] in expected];points={};originals={}
    for f in functions:
        address=f['inventory']['start'];ins=[i for c in f['chunks'] for i in c['instructions']]
        originals[address]=b''.join(bytes.fromhex(i['loaded_bytes']) for i in ins)
        assert len(originals[address])==f['inventory']['size']==expected[address]
        for i in ins:
            points[i['ida_linear_address']]=i;at=int(i['file_offset'],16);data=bytes.fromhex(i['file_bytes'])
            assert raw[at:at+len(data)]==data
    for t in (t for t in report['trials'] if t['exact_interval_bytes']):
        assert (a.linked/t['stem']/'candidate.bin').read_bytes()==originals[t['address']]
    for macro in ('INTRO_SEQ_0','INTRO_SEQ_1','POST_SEQ_0','POST_SEQ_1','POST_SEQ_2'):
        t=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-d'+macro]);assert t['exact_interval_bytes']
    word=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-dINTRO_SEQ_2'])
    assert not word['exact_interval_bytes'] and word['candidate_size']==1618
    for at in ('0x3234b','0x32394'):assert points[at]['loaded_bytes']=='30db'
    for at in ('0x32359','0x323a2'):assert points[at]['loaded_bytes']=='fec3'
    for at in ('0x3235b','0x323a4'):assert points[at]['loaded_bytes']=='0fb6c3'
    assert points['0x3235e']['loaded_bytes']=='83f80f' and points['0x323a7']['loaded_bytes']=='83f80d'
    for at,target in (('0x32361',0x3234f),('0x323aa',0x32398),('0x25701',0x256e3),('0x25748',0x2572b)):
        code=bytes.fromhex(points[at]['loaded_bytes']);assert len(code)==2
        assert int(at,16)+2+int.from_bytes(code[1:],'little',signed=True)==target
    assert [int.from_bytes(bytes.fromhex(points[at]['loaded_bytes'])[-4:],'little')
        for at in ('0x32326','0x3252e','0x327eb')]==[32,31,0]
    assert [int.from_bytes(bytes.fromhex(points[at]['loaded_bytes'])[-4:],'little')
        for at in ('0x323e9','0x323fd','0x32506','0x32524','0x327cd','0x327e1')]==[1,0,1,0,1,0]
    assert points['0x254c8']['loaded_bytes']=='8b1d453a0500'
    assert points['0x254d4']['loaded_bytes']=='c643077e' and points['0x254d8']['loaded_bytes']=='c643087e'
    # 原始35BBA呼叫先於全域指標讀取，不能把讀取提前跨過callee。
    call=bytes.fromhex(points['0x254c0']['loaded_bytes']);assert call[0]==0xe8
    assert 0x254c0+5+int.from_bytes(call[1:],'little',signed=True)==0x35bba
    # IDA的caller清單含間接派送關係；回到LE表項及真正的FF14 85讀取端。
    le=parse_le(raw);fixups=parse_fixups(raw,le);dispatch=[]
    callers={i['ida_linear_address']:i for f in image['functions'] if f['inventory']['start'] in ('0x25bf4','0x25ebb')
        for c in f['chunks'] for i in c['instructions']}
    for target,file_offset,base,index,consumers in (
        (0x2548c,0x51c59,0x51de9,28,('0x25e23',)),
        (0x3231b,0x51b71,0x51d71,0,('0x25e3a','0x25f10','0x260f5'))):
        assert fixups[file_offset]==target and file_to_linear(le,file_offset)==base+4*index
        for address in consumers:
            i=callers[address];code=bytes.fromhex(i['loaded_bytes']);assert code[:3]==bytes.fromhex('ff1485')
            assert int.from_bytes(code[3:],'little')==base
        dispatch.append({'target':hex(target),'entry_file_offset':hex(file_offset),'table_base':hex(base),
            'entry_index':index,'consumer_addresses':list(consumers)})
    references=verify_original_bindings(functions,raw,BINDINGS);assert references==report['original_binding_references']
    rejected=[]
    for name in ('sub_13185','sub_24B4D','j___delay','dword_53AFB','sub_32975'):
        binding=dict(BINDINGS);binding[name]+=1
        try:verify_original_bindings(functions,raw,binding)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_function_bytes':expected,'complete_total_bytes':2341,
        'exact_candidates':sum(t['exact_interval_bytes'] for t in report['trials']),
        'intro_counter_width_bits':8,'intro_loop_limits':[15,13],'chapter_store_values':[32,31,0],
        'flag_store_values':[1,0,1,0,1,0],'post_record_read_after_clear':True,
        'new_symbol_wrong_addresses_rejected':rejected,'original_binding_references':len(references),
        'indirect_dispatch_entries':dispatch,'ida_indirect_callers_not_treated_as_direct_e8':True,
        'word_counter_control_unmatched':True,'author_declarations':'unknown','hardware_timing_not_promoted':True,
        'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest()}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
