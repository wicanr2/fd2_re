#!/usr/bin/env python3
"""以完整577-byte物件驗證索引初始化、第一次檢查及實際來源綁定。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile


def branch_target(code):
    assert code[0x105]==0xe9
    return 0x2f642+0x10a+int.from_bytes(code[0x106:0x10a],'little',signed=True)


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text());assert report['compiler_version']=='9.5'
    assert report['matched_addresses']==['0x2f642']
    positive=[t for t in report['trials'] if t['exact_interval_bytes']]
    assert len(positive)==2 and {t['flags'][-1] for t in positive}=={'-dITEM_INDEX_1','-dITEM_INDEX_2'}
    assert all(t['flags'][:2]==['-mf','-3s'] and len(t['flags'])==3 for t in positive)
    code=(a.linked/positive[0]['stem']/'candidate.bin').read_bytes()
    assert len(code)==577 and code[:5]==bytes.fromhex('6854000000') and code[-7:]==bytes.fromhex('83c4205f5e5bc3')
    assert branch_target(code)==0x2f66e and code[0x103:0x105]==bytes.fromhex('31fa')
    assert code[0x2c:0x31]==bytes.fromhex('83fa087d17')
    control=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-dITEM_INDEX_0'])
    before=(a.linked/control['stem']/'candidate.bin').read_bytes()
    assert not control['exact_interval_bytes'] and len(before)==577
    assert [n for n,(x,y) in enumerate(zip(code,before)) if x!=y]==[262]
    assert branch_target(before)==0x2f673
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-item-transaction-test-') as temp:
        root=Path(temp)
        for case in ('source_missing','source_bytes','source_hash','object_bytes'):
            objects=root/case;shutil.copytree(a.objects,objects)
            if case=='source_missing':(objects/'GAME.C').unlink()
            elif case=='source_bytes':(objects/'GAME.C').write_bytes((objects/'GAME.C').read_bytes()+b'\n/* changed */\n')
            elif case=='source_hash':
                path=objects/'compile-report.json';data=json.loads(path.read_text());data['source_sha256']='0'*64;path.write_text(json.dumps(data))
            else:
                path=objects/(positive[0]['stem']+'.OBJ');data=bytearray(path.read_bytes());data[-1]^=1;path.write_bytes(data)
            output=root/(case+'-output')
            result=subprocess.run(['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link',
                '--objects',str(objects),'--original',str(a.original),'--evidence',str(a.evidence),'--output',str(output)],
                capture_output=True,text=True,timeout=30)
            assert result.returncode and (not output.exists() if case!='object_bytes' else not (output/positive[0]['stem']).exists()),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'candidate_output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    receipt={'schema_version':1,'complete_function_bytes':577,'compiler_generated_branch':True,
        'branch_address':'0x2f747','branch_target':'0x2f66e','old_control_target':'0x2f673',
        'old_control_differing_bytes':1,'matching_forms':['ITEM_INDEX_1','ITEM_INDEX_2'],
        'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest(),'rejected':rejected}
    (a.output/'result.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n');print(json.dumps(receipt,ensure_ascii=False))


if __name__=='__main__':main()
