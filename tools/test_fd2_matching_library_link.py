#!/usr/bin/env python3
"""以真實SDK／IDA收據核對原生重定位的完整輸出與輸出前拒收。"""
import argparse
import copy
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
from fd2_matching_library_link import plan, zero_bindings, omf_sections, verify_imports


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','library','evidence','original','linked','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args()
    assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'library-link-report.json').read_text())
    assert len(report['trials'])==9 and all(t['exact_interval_bytes'] for t in report['trials'])
    assert sum(t['size'] for t in report['trials'] if t['original_name']!='memcpy')==395
    for trial in report['trials']:
        assert not trial['binding_segments_contain_code']
        assert omf_sections((a.linked/Path(trial['member']).stem/'bindings.obj').read_bytes())==[]
    image=json.loads(a.evidence.read_text());raw=a.original.read_bytes()
    f=next(f for f in image['functions'] if f['inventory']['start']=='0x3fb2f')
    assert verify_imports(f,raw,{'setvbuf':0x46ce9})
    for wrong in ({'setvbuf':0x46cea},{'_setvbuf':0x46ce9}):
        try:verify_imports(f,raw,wrong)
        except ValueError:pass
        else:raise AssertionError('錯誤位址或下底線別名未拒收')
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-native-sdk-test-') as temp:
        root=Path(temp)
        for case in ('original_bytes','library_bytes','member_bytes','tool_hash','loaded_bytes','callee_address','forged_loaded_and_callee','symbol_name'):
            original=a.original;library=a.library;objects=a.objects;evidence=a.evidence;wlink=Path('/opt/watcom/binl64/wlink')
            if case=='original_bytes':
                original=root/'bad.exe';original.write_bytes(raw[:-1])
            elif case=='library_bytes':
                library=root/'bad.lib';library.write_bytes(a.library.read_bytes()[:-1])
            elif case=='member_bytes':
                objects=root/'objects';objects.mkdir();data=bytearray((a.objects/'setbuf.o').read_bytes());data[-1]^=1;(objects/'setbuf.o').write_bytes(data)
            elif case=='tool_hash':
                wlink=root/'bad-wlink';wlink.write_bytes(b'not the locked tool')
            else:
                d=copy.deepcopy(image)
                target=next(f for f in d['functions'] if f['inventory']['start']=='0x46ce9')
                source=next(f for f in d['functions'] if f['inventory']['start']=='0x3fb2f')
                call=next(i for c in source['chunks'] for i in c['instructions'] if i['instruction'].strip().startswith('call'))
                if case in ('loaded_bytes','forged_loaded_and_callee'):
                    code=bytearray.fromhex(call['loaded_bytes']);code[1]=(code[1]+1)&255;call['loaded_bytes']=code.hex()
                if case in ('callee_address','forged_loaded_and_callee'):target['inventory']['start']='0x46cea'
                if case=='symbol_name':target['inventory']['ida_analysis_name']='_setvbuf'
                evidence=root/(case+'.json');evidence.write_text(json.dumps(d))
            output=root/(case+'-output')
            cmd=['python3',str(Path(__file__).with_name('fd2_matching_library_link.py')),'--objects',str(objects),'--library',str(library),
                 '--evidence',str(evidence),'--original',str(original),'--wlink',str(wlink),'--members','setbuf','--output',str(output)]
            result=subprocess.run(cmd,capture_output=True,text=True,timeout=30)
            assert result.returncode and not output.exists(),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'actual_sdk_trials':9,'new_retained_library_functions':8,'new_sdk_code_bytes':395,
            'zero_binding_objects_have_no_code':True,'exact_symbol_and_address_rejections':2,'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
