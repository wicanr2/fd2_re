#!/usr/bin/env python3
"""核對完整AI物品target-list、byte算式、夾限與原始保存契約。"""
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
    for name in ('linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists();raw=a.original.read_bytes();image=json.loads(a.evidence.read_text())
    sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']
    by={f['inventory']['start']:f for f in image['functions']}
    points={int(i['ida_linear_address'],16):i for f in image['functions'] for c in f['chunks'] for i in c['instructions']}
    f=by['0x15055'];assert f['inventory']['size']==700 and len(f['chunks'])==1
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for c in f['chunks'] for i in c['instructions'])
    r=json.loads((a.linked/'report.json').read_text());assert r['input']==image['input'] and r['evidence_sha256']==sha(a.evidence)
    assert r['matched_addresses']==['0x15055'] and len(r['trials'])==90
    positive=[t for t in r['trials'] if t['exact_interval_bytes']];assert positive
    for t in positive:assert (a.linked/t['stem']/'candidate.bin').read_bytes()==original
    for n in (8,9):assert any(t['flags']==['-mf','-3s','-dAI_ITEM_EXEC_'+str(n)] for t in positive)
    near=next(t for t in r['trials'] if t['flags']==['-mf','-3s','-dAI_ITEM_EXEC_6'])
    near_code=(a.linked/near['stem']/'candidate.bin').read_bytes();assert not near['exact_interval_bytes'] and near_code!=original
    assert points[0x15063]['loaded_bytes']=='83ec24' and points[0x150c5]['loaded_bytes']=='88442420'
    assert points[0x150a2]['loaded_bytes']=='0fb64011' and points[0x150b5]['loaded_bytes']=='0fb65811'
    assert target(points[0x150cf])==0x150fb and target(points[0x1514d])==0x15168
    assert target(points[0x15163])==0x152e5
    assert target(points[0x150f1])==0x149f8 and target(points[0x15114])==0x14818
    assert points[0x1511c]['loaded_bytes']=='89c7' and points[0x152e8]['loaded_bytes']=='57'
    assert points[0x152e5]['loaded_bytes']=='89e0' and target(points[0x152f0])==0x20c6f
    assert points[0x151e7]['loaded_bytes']=='806c242010'
    assert points[0x151ec]['loaded_bytes']=='a1373c0500' and points[0x151f7]['loaded_bytes']=='0fb65c2420'
    assert points[0x1522e]['loaded_bytes']=='8b1d3b3c0500' and points[0x1523a]['loaded_bytes']=='0fb6442420'
    assert target(points[0x15211])==0x15220 and target(points[0x15222])==0x1522e
    assert target(points[0x15254])==0x15263 and target(points[0x15265])==0x15271
    assert points[0x151b7]['loaded_bytes']=='bb40000000' and target(points[0x151db])==0x151be
    assert points[0x15299]['loaded_bytes']=='bb01000000' and points[0x152bb]['loaded_bytes']=='83fb09'
    assert target(points[0x152be])==0x152a0 and points[0x152fd]['loaded_bytes']=='c705c83e050000000000'
    assert points[0x4e577]['loaded_bytes']=='f7e2' and points[0x4e581]['loaded_bytes']=='5d'
    assert points[0x4dc01]['loaded_bytes']=='53' and points[0x4dc2f]['loaded_bytes']=='5b'
    assert points[0x4dc30]['loaded_bytes']=='5e' and points[0x4dc31]['loaded_bytes']=='5f'
    for at in (0x15006,0x15039):assert target(points[at])==0x15055
    for c in f['chunks']:
        for i in c['instructions']:
            at=int(i['file_offset'],16);b=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(b)]==b
    rejected=[]
    for name in ('sub_20C6F','sub_28784','sub_4E56C','sub_4DBFC','sub_149F8','sub_14818','dword_53C37','dword_53C3B'):
        bindings=dict(BINDINGS);bindings[name]+=1
        try:verify_original_bindings([f],raw,bindings)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    result={'schema_version':1,'complete_function_bytes':700,'original_local_bytes':36,'original_byte_offset':32,
        'target_list_helpers':['0x149f8','0x14818'],'count_saved_at':'0x1511c','target_list_consumer':'0x152f0',
        'small_command_join':'0x152e5','raw_byte_subtract':16,'signed_coordinate_clamps_preserved':True,
        'fade_initial':64,'flash_first':1,'flash_limit':9,'original_callers':['0x15006','0x15039'],
        'actual_callee_contract':{'sub_4E56C':'EAX/EDX only; EBP restored','sub_4DBFC':'EBX/ESI/EDI restored'},
        'pragma_allowance_is_actual_clobber':False,'wrong_addresses_rejected':rejected,'exact_candidates':len(positive),
        'close_candidate_bytes':len(near_code),'close_candidate_not_accepted':True,'author_declarations_and_expressions':'unknown',
        'source_report_sha256':sha(a.linked/'report.json')}
    a.output.mkdir(parents=True,exist_ok=False);(a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
