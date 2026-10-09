#!/usr/bin/env python3
"""以真實C物件核對數字群組的共同清理尾端與include來源。"""
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
    report=json.loads((a.linked/'report.json').read_text());assert report['matched_addresses']==['0x2d3ff','0x2d516']
    trial=next(t for t in report['trials'] if t['exact_interval_bytes']);code=(a.linked/trial['stem']/'candidate.bin').read_bytes()
    assert len(code)==545 and trial['exact_entry_offsets']
    assert code[0x110:0x117]==bytes.fromhex('83c43c5f5e5bc3') and code[0x21c]==0xe9
    assert 0x2d3ff+0x221+int.from_bytes(code[0x21d:0x221],'little',signed=True)==0x2d50f
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-digit-group-test-') as temp:
        root=Path(temp)
        for case in ('include_missing','include_bytes','include_time','include_hash'):
            objects=root/case;shutil.copytree(a.objects,objects);include=objects/'DIGIT.C'
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
    result={'schema_version':1,'complete_group_bytes':545,'compiled_entries':2,'shared_cleanup_bytes':'83c43c5f5e5bc3',
            'branch_address':'0x2d61b','branch_target':'0x2d50f','compiler_generated_branch':True,
            'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest(),'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
