#!/usr/bin/env python3
"""核對原始ESI先讀後寫與完整140-byte函式，不猜初值或作者宣告。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text());image=json.loads(a.evidence.read_text());raw=a.original.read_bytes()
    assert hashlib.sha256(a.evidence.read_bytes()).hexdigest()==report['evidence_sha256']
    assert hashlib.sha256(raw).hexdigest()==report['input']['sha256']==image['input']['sha256']
    assert report['matched_addresses']==['0x12dac'] and len(report['trials'])==126
    f=next(f for f in image['functions'] if f['inventory']['start']=='0x12dac')
    instructions=[i for c in f['chunks'] for i in c['instructions']]
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for i in instructions)
    assert len(original)==f['inventory']['size']==140
    for i in instructions:
        at=int(i['file_offset'],16);data=bytes.fromhex(i['file_bytes']);assert raw[at:at+len(data)]==data
    positive=[t for t in report['trials'] if t['exact_interval_bytes']]
    assert len(positive)==5 and all(t['flags'][-1]=='-dINPUT_ESI_1' for t in positive)
    for t in positive:assert (a.linked/t['stem']/'candidate.bin').read_bytes()==original
    assert original[:5]==bytes.fromhex('6818000000') and original[10:12]==bytes.fromhex('5356')
    assert original[31:36]==bytes.fromhex('0fbf0039c6')
    assert original[53:56]==bytes.fromhex('0fbf30') and original[-3:]==bytes.fromhex('5e5bc3')
    # 原始第一次CMP ESI,EAX早於MOVSX ESI；函式入口輸入的producer仍未知。
    before_read=[i for i in instructions if int(i['ida_linear_address'],16)<0x12dce]
    assert all('esi' not in i['instruction'] or i['instruction'].strip()=='push    esi' for i in before_read)
    assert all(not t['exact_interval_bytes'] for t in report['trials'] if t['flags'][-1]!='-dINPUT_ESI_1')
    source=(a.objects/'GAME.C').read_text()
    assert 'parm caller [esi] value [eax] modify exact [eax ecx edx gs]' in source
    assert 'int sub_12DAC(int previous)' in source
    # 此helper修改AX但保存ESI，不把非匹配callee宣告當成原始clobber事實。
    helper=next(f for f in image['functions'] if f['inventory']['start']=='0x4e031')
    code=b''.join(bytes.fromhex(i['loaded_bytes']) for c in helper['chunks'] for i in c['instructions'])
    assert code==bytes.fromhex('56be1a040000668b06668946025ec3')
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-input-esi-test-') as temp:
        root=Path(temp)
        for case in ('source_missing','source_bytes','source_hash'):
            objects=root/case;shutil.copytree(a.objects,objects)
            if case=='source_missing':(objects/'GAME.C').unlink()
            elif case=='source_bytes':(objects/'GAME.C').write_bytes((objects/'GAME.C').read_bytes()+b'\n/* changed */\n')
            else:
                path=objects/'compile-report.json';d=json.loads(path.read_text());d['source_sha256']='0'*64;path.write_text(json.dumps(d))
            output=root/(case+'-output')
            result=subprocess.run(['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link',
                '--objects',str(objects),'--original',str(a.original),'--evidence',str(a.evidence),'--output',str(output)],
                capture_output=True,text=True,timeout=30)
            assert result.returncode and not output.exists(),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_function_bytes':{'0x12dac':140},'exact_candidates':len(positive),
        'incoming_esi_read':'0x12dce','first_esi_update':'0x12de1','incoming_esi_producer':'unknown',
        'saved_registers':['ebx','esi'],'signed_clock_read_bits':16,'unmatched_controls_preserved':True,
        'callee_4e031_complete_bytes':15,'author_declarations':'unknown','actual_gs_clobber':'unknown',
        'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest(),'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
