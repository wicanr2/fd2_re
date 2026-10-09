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
    p.add_argument('--draw',action='store_true',help='核對含2D620逐列複製的三函式單元')
    a=p.parse_args();assert Path('/.dockerenv').exists()
    entries=['0x2d3ff','0x2d516']+(['0x2d620'] if a.draw else [])
    size=618 if a.draw else 545
    includes=('DOWN.C','DIGIT.C') if a.draw else ('DIGIT.C',)
    report=json.loads((a.linked/'report.json').read_text());assert report['matched_addresses']==entries
    trial=next(t for t in report['trials'] if t['exact_interval_bytes']);code=(a.linked/trial['stem']/'candidate.bin').read_bytes()
    assert len(code)==size and trial['exact_entry_offsets']
    assert code[0x110:0x117]==bytes.fromhex('83c43c5f5e5bc3') and code[0x21c]==0xe9
    assert 0x2d3ff+0x221+int.from_bytes(code[0x21d:0x221],'little',signed=True)==0x2d50f
    if a.draw:
        assert trial['function_intervals']==[{'address':address,'offset':offset,'size':length}
            for address,offset,length in (('0x2d3ff',0,279),('0x2d516',279,266),('0x2d620',545,73))]
        # 真實產碼必須保留ADD在IMUL之前，以及九次六byte複製的迴圈。
        assert code[576:584]==bytes.fromhex('83c3046b74241c06')
        assert code[590:592]==bytes.fromhex('6a06') and code[608:613]==bytes.fromhex('83fe097ce9')
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-digit-group-test-') as temp:
        root=Path(temp)
        for name in includes:
            for case in ('include_missing','include_bytes','include_time','include_hash'):
                label=name+'-'+case;objects=root/label;shutil.copytree(a.objects,objects);include=objects/name
                if case=='include_missing':include.unlink()
                elif case=='include_bytes':include.write_bytes(include.read_bytes()+b'\n/* changed */\n')
                elif case=='include_time':os.utime(include,(315532801,315532801))
                else:
                    path=objects/'compile-report.json';d=json.loads(path.read_text())
                    next(i for i in d['source_includes'] if i['dos_file']==name)['dos_sha256']='0'*64;path.write_text(json.dumps(d))
                output=root/(label+'-output')
                cmd=['python3',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link','--objects',str(objects),
                     '--original',str(a.original),'--evidence',str(a.evidence),'--output',str(output)]
                result=subprocess.run(cmd,capture_output=True,text=True,timeout=30)
                assert result.returncode and not output.exists(),(label,result.stdout,result.stderr)
                rejected.append({'case':label if a.draw else case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_group_bytes':size,'compiled_entries':len(entries),'shared_cleanup_bytes':'83c43c5f5e5bc3',
            'branch_address':'0x2d61b','branch_target':'0x2d50f','compiler_generated_branch':True,
            'source_report_sha256':hashlib.sha256((a.linked/'report.json').read_bytes()).hexdigest(),'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
