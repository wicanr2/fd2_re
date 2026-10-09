#!/usr/bin/env python3
"""驗證原生SDK全檔重建的實際來源、重複覆蓋與偽造收據拒收。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import tempfile
from fd2_matching_library_link import INPUT_SHA


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','library','evidence','original','linked','review','baseline','receipt','first','second','output'):
        p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    filename='native-library-bootstrap-receipt.json'
    assert (a.first/filename).read_bytes()==(a.second/filename).read_bytes()
    assert (a.first/'FD2.EXE').read_bytes()==(a.second/'FD2.EXE').read_bytes()
    assert hashlib.sha256((a.first/'FD2.EXE').read_bytes()).hexdigest()==INPUT_SHA
    old=json.loads(a.receipt.read_text());new=json.loads((a.first/filename).read_text())
    assert new['restored_spans']==old['restored_spans'] and new['functions']==old['functions'] and new['counts']==old['counts']
    before={s['ida_linear_address']:s for s in old['sdk_library_spans']};after={s['ida_linear_address']:s for s in new['sdk_library_spans']}
    assert all(after[k]==v for k,v in before.items())
    assert len(after)==19 and new['sdk_library_code_bytes']==1200
    assert new['sdk_native_new_functions']==8 and new['sdk_native_new_code_bytes']==395
    assert len(after)-len(before)==8 and new['decompilation_complete'] is False
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-native-sdk-bootstrap-test-') as temp:
        root=Path(temp)
        for case in ('report_and_review_forged','candidate_bytes_changed','review_binding_changed','duplicate_span','unknown_classification','c_ownership_overlap'):
            linked=a.linked;review=a.review;receipt=a.receipt;evidence=a.evidence
            if case in ('report_and_review_forged','candidate_bytes_changed'):
                linked=root/(case+'-linked');shutil.copytree(a.linked,linked)
                if case=='candidate_bytes_changed':
                    f=linked/'setbuf/candidate.bin';data=bytearray(f.read_bytes());data[0]^=1;f.write_bytes(data)
                else:
                    rp=linked/'library-link-report.json';r=json.loads(rp.read_text());r['trials'][2]['code_sha256']='0'*64;rp.write_text(json.dumps(r))
                    rv=json.loads(a.review.read_text());rv['matches'][2]['code_sha256']='0'*64
                    rv['probe_sha256']=hashlib.sha256(rp.read_bytes()).hexdigest();review=root/(case+'.json');review.write_text(json.dumps(rv))
            elif case=='review_binding_changed':
                d=json.loads(a.review.read_text());d['probe_sha256']='0'*64;review=root/(case+'.json');review.write_text(json.dumps(d))
            elif case=='duplicate_span':
                d=copy.deepcopy(old);d['sdk_library_spans'].append(copy.deepcopy(d['sdk_library_spans'][0]));receipt=root/(case+'.json');receipt.write_text(json.dumps(d))
            elif case=='c_ownership_overlap':
                d=copy.deepcopy(old);f=next(f for f in d['functions'] if f['ida_linear_address']=='0x3fb2f');f['source_kind']='matched_c'
                receipt=root/(case+'.json');receipt.write_text(json.dumps(d))
            else:
                d=json.loads(a.evidence.read_text());f=next(f for f in d['functions'] if f['inventory']['start']=='0x3fb2f');f['inventory']['classification']['value']='unknown'
                evidence=root/(case+'.json');evidence.write_text(json.dumps(d))
            output=root/(case+'-output')
            cmd=['python3',str(Path(__file__).with_name('fd2_matching_library_native_restore.py'))]
            values={'objects':a.objects,'library':a.library,'evidence':evidence,'original':a.original,'linked':linked,'review':review,
                    'baseline':a.baseline,'receipt':receipt,'output':output}
            for key,value in values.items():cmd+=['--'+key,str(value)]
            result=subprocess.run(cmd,capture_output=True,text=True,timeout=60)
            assert result.returncode and not output.exists(),(case,result.stdout,result.stderr)
            rejected.append({'case':case,'output_created':False})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'sdk_functions':19,'sdk_code_bytes':1200,'new_sdk_functions':8,'new_sdk_code_bytes':395,
        'old_c_ledger_unchanged':True,'old_sdk_spans_unchanged':True,'duplicate_control_counted_once':True,
        'repeat_whole_exe_and_receipts_equal':True,'rejected_before_output':rejected}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
