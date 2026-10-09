#!/usr/bin/env python3
"""驗證整個SDK群組、雙輪全檔與未知／C目標拒收，不提高C覆蓋。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import tempfile
from types import SimpleNamespace

from fd2_matching_library_extension import prepare
from fd2_matching_library_probe import omf_sections


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('objects','library','evidence','original','baseline','receipt','first','second','output'):
        p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists();sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
    image=json.loads(a.evidence.read_text());base=json.loads(a.receipt.read_text());original=a.original.read_bytes()
    reports=[json.loads((path/'sdk-extension-report.json').read_text()) for path in (a.first,a.second)]
    receipts=[json.loads((path/'sdk-extension-bootstrap-receipt.json').read_text()) for path in (a.first,a.second)]
    assert (a.first/'sdk-extension-report.json').read_bytes()==(a.second/'sdk-extension-report.json').read_bytes()
    assert (a.first/'sdk-extension-bootstrap-receipt.json').read_bytes()==(a.second/'sdk-extension-bootstrap-receipt.json').read_bytes()
    assert (a.first/'FD2.EXE').read_bytes()==(a.second/'FD2.EXE').read_bytes()==original
    assert sha(a.first/'FD2.EXE')==image['input']['sha256']
    receipt=receipts[0];report=reports[0]
    assert receipt['counts']==base['counts'] and receipt['restored_spans']==base['restored_spans']
    assert receipt['functions']==base['functions'] and receipt['sdk_library_spans'][:len(base['sdk_library_spans'])]==base['sdk_library_spans']
    assert receipt['sdk_library_function_count']==25 and receipt['sdk_library_code_bytes']==1777
    assert receipt['sdk_extension_new_functions']==6 and receipt['sdk_extension_new_code_bytes']==577
    assert not receipt['classification_changed'] and not receipt['decompilation_complete']
    expected={'0x3d919','0x46ce9','0x46a80','0x4d7bc','0x4d84c','0x4d897'}
    new=receipt['sdk_extension_rebuilt_spans'];assert {s['ida_linear_address'] for s in new}==expected
    assert all(s['original_classification']['value']=='runtime' for s in new)
    sdk=omf_sections((a.objects/'amodf.o').read_bytes());assert len(sdk)==1 and sdk[0]['size']==302
    for path in (a.first,a.second):
        assert (path/'amodf/candidate.bin').read_bytes()==sdk[0]['data']
        assert (path/'amodf/linked.raw').read_bytes()==sdk[0]['data']
    group=report['whole_group'];assert group['whole_code_equal'] and group['raw_container_prefix_zero_bytes']==0
    assert [f['size'] for f in group['functions']]==[144,75,83]
    assert {(p['name'],p['offset']) for p in group['publics']}=={('__ModF',0),('__ZBuf2F',144)}
    assert group['caller_windows'] and len(report['native_report']['trials'])==3
    assert all(t['exact_interval_bytes'] for t in report['native_report']['trials'])
    rejected=[]
    with tempfile.TemporaryDirectory(prefix='fd2-sdk-extension-guard-') as temp:
        temp=Path(temp)
        for label in ('unknown_function','cut_boundary','shift_public','forged_loaded_byte','split_chunk',
                      'unknown_ledger','c_ledger','c_span_overlap','ambiguous_public','object_not_in_library'):
            changed=copy.deepcopy(image);ledger=copy.deepcopy(base);objects=a.objects
            f=next(f for f in changed['functions'] if f['inventory']['start']=='0x4d897')
            if label=='unknown_function':f['inventory']['classification']['value']='unknown'
            elif label=='cut_boundary':
                f['inventory']['size']-=1;f['inventory']['end']=hex(int(f['inventory']['end'],16)-1)
            elif label=='shift_public':
                public=next(f for f in changed['functions'] if f['inventory']['start']=='0x4d84c');public['inventory']['start']='0x4d84d'
            elif label=='forged_loaded_byte':
                i=f['chunks'][0]['instructions'][0];b=bytearray.fromhex(i['loaded_bytes']);b[0]^=1;i['loaded_bytes']=b.hex()
            elif label=='split_chunk':f['chunks'].append(copy.deepcopy(f['chunks'][0]))
            elif label in ('unknown_ledger','c_ledger'):
                row=next(f for f in ledger['functions'] if f['ida_linear_address']=='0x4d897')
                row['source_kind']='unrestored_original_code' if label=='unknown_ledger' else 'matched_c'
            elif label=='c_span_overlap':ledger['restored_spans'].append({'ida_linear_address':'0x4d897','source_kind':'matched_c','size':83})
            elif label=='ambiguous_public':
                changed['functions'].append(copy.deepcopy(next(f for f in changed['functions'] if f['inventory']['start']=='0x4d84c')))
            elif label=='object_not_in_library':
                objects=temp/'bad-objects';objects.mkdir();bad=bytearray((a.objects/'amodf.o').read_bytes());bad[-1]^=1;(objects/'amodf.o').write_bytes(bad)
            evidence=temp/(label+'-ida.json');evidence.write_text(json.dumps(changed))
            receipt_path=temp/(label+'-receipt.json');receipt_path.write_text(json.dumps(ledger))
            output=temp/(label+'-must-not-exist')
            args=SimpleNamespace(objects=objects,library=a.library,evidence=evidence,original=a.original,
                baseline=a.baseline,receipt=receipt_path,output=output,wlink=Path('/opt/watcom/binl64/wlink'),wdis=Path('/opt/watcom/binl64/wdis'))
            try:prepare(args)
            except ValueError:rejected.append(label)
            else:raise AssertionError(label)
            assert not output.exists()
    result={'schema_version':1,'new_sdk_source_functions':6,'new_sdk_source_bytes':577,'total_sdk_source_functions':25,
        'total_sdk_source_bytes':1777,'whole_code_group_bytes':302,'complete_function_bytes':[144,75,83],
        'old_sdk_sources_unchanged':len(base['sdk_library_spans']),'c_spans_unchanged':len(base['restored_spans']),
        'c_counts_unchanged':base['counts'],'reports_repeat_equal':True,'whole_file_repeat_equal':True,
        'classification_changed':False,'decompilation_complete':False,'pre_output_rejections':rejected,
        'source_report_sha256':sha(a.first/'sdk-extension-report.json')}
    a.output.mkdir(parents=True,exist_ok=False);(a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
