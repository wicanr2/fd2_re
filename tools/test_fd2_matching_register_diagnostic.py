#!/usr/bin/env python3
"""以真實340-byte負例及反例驗證診斷不會接受錯誤匹配。"""
import argparse
import json
from pathlib import Path
import shutil
import subprocess
import tempfile

from fd2_matching_register_diagnostic import diagnose,prepare


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('original','evidence','linked','output'):p.add_argument('--'+name,type=Path,required=True)
    p.add_argument('--stem',default='G00');a=p.parse_args();assert Path('/.dockerenv').exists()
    checked=prepare(a)
    assert not checked['exact_interval_bytes'] and checked['instruction_text_equivalent_under_register_map']
    changed={k:v for k,v in checked['candidate_to_original_registers'].items() if k!=v}
    assert changed=={'ebx':'esi','esi':'ebx'} and len(checked['differing_byte_offsets'])==14
    assert checked['register_map_interval']['start']=='0x2d6ac' and checked['register_map_interval']['end']=='0x2d777'
    image=json.loads(a.evidence.read_text());f=next(f for f in image['functions'] if f['inventory']['start']=='0x2d669')
    original=b''.join(bytes.fromhex(i['loaded_bytes']) for c in f['chunks'] for i in c['instructions'])
    identity=diagnose(original,original,0x2d669);assert identity['exact_interval_bytes'] and identity['instruction_text_equivalent_under_register_map']
    negatives=[]
    for case,offset in (('immediate',1),('branch',0xb8),('inconsistent_register_map',0x44)):
        candidate=bytearray(original)
        if case=='inconsistent_register_map':candidate[0x44]=0xdb;candidate[0x6d]=0x47
        else:candidate[offset]^=1
        result=diagnose(original,bytes(candidate),0x2d669)
        assert not result['exact_interval_bytes'] and not result['instruction_text_equivalent_under_register_map'],case
        assert result['diagnostic_only'] and not result['increases_c_coverage'];negatives.append(case)
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-register-diagnostic-test-') as temp:
        root=Path(temp)
        for case in ('candidate_bytes','false_match','false_first_difference','false_original_size','evidence_hash','input_identity','original_bytes'):
            linked=root/case;shutil.copytree(a.linked,linked);original_path=a.original
            path=linked/'report.json';report=json.loads(path.read_text());trial=next(t for t in report['trials'] if t['stem']==a.stem)
            if case=='candidate_bytes':
                code=linked/a.stem/'candidate.bin';data=bytearray(code.read_bytes());data[0]^=1;code.write_bytes(data)
            elif case=='false_match':trial['exact_interval_bytes']=True
            elif case=='false_first_difference':trial['first_difference_offset']=0
            elif case=='false_original_size':trial['original_size']+=1
            elif case=='evidence_hash':report['evidence_sha256']='0'*64
            elif case=='input_identity':report['input']['size']+=1
            else:
                original_path=root/'changed-original.exe';data=bytearray(a.original.read_bytes());data[-1]^=1;original_path.write_bytes(data)
            path.write_text(json.dumps(report));output=root/(case+'-output')
            result=subprocess.run(['python3',str(Path(__file__).with_name('fd2_matching_register_diagnostic.py')),
                '--original',str(original_path),'--evidence',str(a.evidence),'--linked',str(linked),'--stem',a.stem,
                '--output',str(output)],capture_output=True,text=True,timeout=30)
            assert result.returncode and not output.exists(),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_function_bytes':340,'real_mismatching_candidate_remains_rejected':True,
        'differing_bytes':14,'diagnostic_mapping':changed,'exact_identity_control':True,
        'non_mapping_negatives':negatives,'rejected_before_output':rejected,'increases_c_coverage':False}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
