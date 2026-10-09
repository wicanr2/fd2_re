#!/usr/bin/env python3
"""用真實雙函式物件核對編譯器共同尾端與include來源拒收。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text())
    assert report['matched_addresses']==['0x12c60','0x12cea']
    trial=next(t for t in report['trials'] if t['exact_interval_bytes'])
    code=(a.linked/trial['stem']/'candidate.bin').read_bytes()
    assert len(code)==283 and trial['exact_entry_offsets']
    assert code[0xe8:0xea]==bytes.fromhex('749c') and code[0x86:0x8a]==bytes.fromhex('5f5e5bc3')
    assert 0x12c60+0xea+int.from_bytes(code[0xe9:0xea],'little',signed=True)==0x12ce6
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-map-group-test-') as temp:
        root=Path(temp)
        for case in ('include_missing','include_bytes','include_time','include_hash'):
            objects=root/case;shutil.copytree(a.objects,objects);include=objects/'REC.C'
            if case=='include_missing':include.unlink()
            elif case=='include_bytes':include.write_bytes(include.read_bytes()+b'\n/* changed */\n')
            elif case=='include_time':os.utime(include,(315532801,315532801))
            else:
                path=objects/'compile-report.json';d=json.loads(path.read_text());d['source_includes'][0]['dos_sha256']='0'*64;path.write_text(json.dumps(d))
            output=root/(case+'-output')
            cmd=['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link','--objects',str(objects),
                 '--original',str(a.original),'--evidence',str(a.evidence),'--output',str(output)]
            result=subprocess.run(cmd,capture_output=True,text=True,timeout=30)
            assert result.returncode and not output.exists(),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_group_bytes':283,'compiled_entries':2,'shared_return_bytes':'5f5e5bc3',
            'branch_address':'0x12d48','branch_target':'0x12ce6','compiler_generated_branch':True,
            'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest(),'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
