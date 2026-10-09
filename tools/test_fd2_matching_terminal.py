#!/usr/bin/env python3
"""驗證完整終局對話匹配與1718-byte owner負例，保留真實callee保存證據。"""
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
    for name in ('dialogue-linked','body-linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists();raw=a.original.read_bytes();image=json.loads(a.evidence.read_text())
    sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest();assert hashlib.sha256(raw).hexdigest()==image['input']['sha256']
    by={f['inventory']['start']:f for f in image['functions']};points={int(i['ida_linear_address'],16):i for f in image['functions'] for c in f['chunks'] for i in c['instructions']}
    originals={}
    for address,size in (('0x2bce5',1718),('0x2c39b',106)):
        f=by[address];assert f['inventory']['size']==size and len(f['chunks'])==1
        originals[address]=b''.join(bytes.fromhex(i['loaded_bytes']) for c in f['chunks'] for i in c['instructions'])
        assert len(originals[address])==size
        for c in f['chunks']:
            for i in c['instructions']:
                at=int(i['file_offset'],16);b=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(b)]==b
    dialogue=json.loads((a.dialogue_linked/'report.json').read_text());body=json.loads((a.body_linked/'report.json').read_text())
    for r in (dialogue,body):assert r['input']==image['input'] and r['evidence_sha256']==sha(a.evidence)
    assert dialogue['matched_addresses']==['0x2c39b'] and len(dialogue['trials'])==27
    assert not body['matched_addresses'] and len(body['trials'])==72
    positive=[t for t in dialogue['trials'] if t['exact_interval_bytes']];assert len(positive)==4
    for t in positive:assert (a.dialogue_linked/t['stem']/'candidate.bin').read_bytes()==originals['0x2c39b']
    baseline=next(t for t in dialogue['trials'] if t['flags']==['-mf','-3s','-dTERMINAL_DIALOGUE_0'])
    assert not baseline['exact_interval_bytes']
    assert (a.dialogue_linked/baseline['stem']/'candidate.bin').read_bytes()!=originals['0x2c39b']
    assert points[0x2c39b]['loaded_bytes']=='682c000000' and points[0x2c3a5]['loaded_bytes']=='53'
    assert points[0x2c403]['loaded_bytes']=='5b' and points[0x2c404]['loaded_bytes']=='c3'
    assert points[0x2c3ce]['loaded_bytes']=='6814950a00'
    assert points[0x4e037]['loaded_bytes']=='668b06' and points[0x4e03e]['loaded_bytes']=='5e'
    assert points[0x19575]['loaded_bytes']=='53' and points[0x196c9]['loaded_bytes']=='5b'
    assert points[0x4e642]['loaded_bytes']=='53'
    for at in (0x4e6f6,0x4e76d,0x4e7f3):assert points[at]['loaded_bytes']=='5b'
    assert points[0x2bcf3]['loaded_bytes']=='83ec54'
    for at in (0x2bd16,0x2bd26,0x2bd36):assert points[at]['loaded_bytes']=='b905000000'
    for at in (0x2bd24,0x2bd34,0x2bd42):assert points[at]['loaded_bytes']=='f3a5'
    for at,limit,to in ((0x2bf93,40,0x2c026),(0x2c0d6,200,0x2c172),(0x2c313,20,0x2c33b)):
        assert bytes.fromhex(points[at]['loaded_bytes'])[-1 if limit<128 else -4]==limit
        assert target(points[at+len(bytes.fromhex(points[at]['loaded_bytes']))])==to
    for at in (0x2bfdd,0x2c108):assert points[at]['loaded_bytes']=='f7f9'
    calls=[i for c in by['0x2bce5']['chunks'] for i in c['instructions'] if i['loaded_bytes'].startswith('e8') and target(i)==0x2c39b]
    assert len(calls)==12
    rejected=[]
    for name in ('unk_525DC','unk_525F0','unk_52604','sub_2C39B','sub_2C405','sub_20421'):
        bindings=dict(BINDINGS);bindings[name]+=1
        try:verify_original_bindings([by['0x2bce5']],raw,bindings)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    for name in ('sub_4E031','_sub_4E031','sub_1956B'):
        bindings=dict(BINDINGS);bindings[name]+=1
        try:verify_original_bindings([by['0x2c39b']],raw,bindings)
        except ValueError:rejected.append(name)
        else:raise AssertionError(name)
    result={'schema_version':1,'complete_matched_function_bytes':106,'exact_candidates':4,'default_contract_negative_bytes':baseline['candidate_size'],
        'dialogue_destination':'0xa9514','original_direct_calls_from_owner':12,'original_body_bytes':1718,
        'original_body_local_bytes':84,'original_table_copy_bytes':[20,20,20],'original_loop_limits':[40,200,20],
        'negative_body_candidates':72,'original_signed_divisions':2,'wrong_addresses_rejected':rejected,
        'actual_save_evidence':{'sub_4E031':'AX modified; ESI restored','sub_1956B':'EBX restored','sub_4E63D':'EBX restored at all three returns'},
        'pragma_allowance_is_actual_clobber':False,'author_declarations':'unknown',
        'source_report_sha256':sha(a.dialogue_linked/'report.json'),'body_report_sha256':sha(a.body_linked/'report.json')}
    a.output.mkdir(parents=True,exist_ok=False);(a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
