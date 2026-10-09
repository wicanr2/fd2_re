#!/usr/bin/env python3
"""核對完整主流程、原始槽位址與未對齊欄位，以及pragma的實際保存證據。"""
import argparse
import hashlib
import json
from pathlib import Path

from fd2_matching_game_restore import BINDINGS,verify_original_bindings


def target(i):
    b=bytes.fromhex(i['loaded_bytes']);n=2 if b[:1]==b'\x0f' else 1
    return int(i['ida_linear_address'],16)+len(b)+int.from_bytes(b[n:],'little',signed=True)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('linked','contract-linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    raw=a.original.read_bytes();image=json.loads(a.evidence.read_text());sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
    assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']
    by={f['inventory']['start']:f for f in image['functions']}
    points={int(i['ida_linear_address'],16):i for f in image['functions'] for c in f['chunks'] for i in c['instructions']}
    f=by['0x25ebb'];assert len(f['chunks'])==1 and f['inventory']['size']==663
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for c in f['chunks'] for i in c['instructions'])
    report=json.loads((a.linked/'report.json').read_text());negative=json.loads((a.contract_linked/'report.json').read_text())
    for r in (report,negative):assert r['input']==image['input'] and r['evidence_sha256']==sha(a.evidence) and len(r['trials'])==45
    assert report['matched_addresses']==['0x25ebb'] and not negative['matched_addresses']
    positive=[t for t in report['trials'] if t['exact_interval_bytes']];assert positive
    for t in positive:assert (a.linked/t['stem']/'candidate.bin').read_bytes()==original
    for c in f['chunks']:
        for i in c['instructions']:
            at=int(i['file_offset'],16);b=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(b)]==b
    assert points[0x25fa6]['loaded_bytes']=='68cb590000'
    assert points[0x26042]['loaded_bytes']=='8d872b310000' and points[0x26048]['loaded_bytes']=='01c3'
    assert points[0x2604a]['loaded_bytes']=='68000a0000' and points[0x2605e]['loaded_bytes']=='81c3000a0000'
    widths={0x26064:('0fb603',0,1),0x2606c:('0fb64301',1,1),0x26075:('8b4302',2,4),
            0x2607d:('8a4306',6,1),0x26085:('8a4307',7,1),0x2608d:('8a4308',8,1),0x26095:('8a4309',9,1)}
    for at,(b,offset,width) in widths.items():assert points[at]['loaded_bytes']==b
    assert points[0x26023]['loaded_bytes']=='83f8ff' and target(points[0x26026])==0x260ab
    assert points[0x2609d]['loaded_bytes']=='813d033c0500ff000000'
    assert points[0x260a9]['loaded_bytes']=='31f6' and target(points[0x260b2])==0x26016
    assert points[0x260cf]['loaded_bytes']=='c7054741050000000000'
    assert points[0x25f10]['loaded_bytes']==points[0x260f5]['loaded_bytes']=='ff1485711d0500'
    assert target(points[0x25dbd])==0x25ebb
    # 實際callee保存契約與保守pragma允許的修改範圍分列。
    assert points[0x4e031]['loaded_bytes']=='56' and points[0x4e03e]['loaded_bytes']=='5e'
    assert points[0x4e037]['loaded_bytes']=='668b06' and points[0x4e03a]['loaded_bytes']=='66894602'
    assert points[0x4e972]['loaded_bytes']=='60' and points[0x4e99d]['loaded_bytes']=='61'
    assert points[0x15e7b]['loaded_bytes']=='53' and points[0x15e9c]['loaded_bytes']=='5b'
    assert points[0x15ea8]['loaded_bytes']=='53' and target(points[0x15f09])==0x22bbe
    assert [points[at]['loaded_bytes'] for at in range(0x22bc1,0x22bc6)]==['5d','5f','5e','5b','c3']
    rejected=[]
    for name in ('sub_1F894','funcs_25E3A','byte_51E63','aFd2Sav_4','unk_50220','sub_2CAD7','sub_10010'):
        bindings=dict(BINDINGS);bindings[name]+=1
        try:verify_original_bindings([f],raw,bindings)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    result={'schema_version':1,'complete_function_bytes':663,'save_buffer_bytes':22987,'slot_header_offset':12587,
        'slot_stride':2600,'unit_copy_bytes':2560,'raw_meta_loads':[{'instruction':hex(at),'offset':o,'width':w} for at,(_,o,w) in widths.items()],
        'raw_ff_sets_selected_zero':True,'zero_retry_target':'0x26016','dispatch_table':'0x51d71','direct_caller':'0x25dbd',
        'actual_callee_contract':{'sub_4E031':'AX only; ESI restored','sub_4E96F':'PUSHA/POPA; EBP restored',
            'sub_15E71':'EBX restored','sub_15E9E':'EBX restored by shared epilogue 0x22bbe'},
        'pragma_allowance_is_actual_clobber':False,'negative_contract_candidates':45,'wrong_symbol_addresses_rejected':rejected,
        'exact_candidates':len(positive),'source_report_sha256':sha(a.linked/'report.json'),'contract_report_sha256':sha(a.contract_linked/'report.json'),
        'author_declarations_and_c_expression':'unknown'}
    a.output.mkdir(parents=True,exist_ok=False);(a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
