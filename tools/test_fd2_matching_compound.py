#!/usr/bin/env python3
"""以真實compiler物件驗證反序入口重排、完整owner與拒收契約。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess

from fd2_matching_game_restore import validate_compound_layout
from fd2_matching_pilot import linked_text
from fd2_matching_sparse import split_coff,read_sparse_pe
from test_fd2_matching_sparse import run_link


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--linked',type=Path,required=True)
    parser.add_argument('--original',type=Path,required=True)
    parser.add_argument('--evidence',type=Path,required=True)
    parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args()
    if not Path('/.dockerenv').exists():raise SystemExit('只在Docker執行')
    report=json.loads((args.linked/'report.json').read_text())
    trial=next(t for t in report['trials'] if t['exact_interval_bytes'])
    source=args.linked/trial['stem'];args.output.mkdir(parents=True,exist_ok=True)
    raw=(source/'candidate.cof').read_bytes();listing=(source/'candidate.dis').read_text()
    positions=[0x231F9,0x231BC,0x230F2]
    converted,layout=split_coff(raw,listing,positions)
    assert converted==(source/'sparse.cof').read_bytes() and layout==trial['compound_layout']
    assert [p['size'] for p in layout['fragments']]==[157,61,202]
    assert len(layout['lifted_branches'])==2 and {b['target_address'] for b in layout['lifted_branches']}=={'0x2328a'}
    defined=tuple('sub_'+format(a,'x') for a in positions)
    original_pe=args.output/'native.exe'
    run_link(source/'candidate.cof',args.output/'native.ld',original_pe,
        '.text 0x230f2 : SUBALIGN(1) { *(_TEXT) }','sub_231F9',defined)
    identity=copy.deepcopy(layout['fragments'])
    for p in identity:p['address']=hex(0x230F2+p['compiler_offset'])
    split_pe=args.output/'identity.exe'
    sections=' '.join(f".m{n} {p['address']} : SUBALIGN(1) {{ *({p['section']}) }}" for n,p in enumerate(identity))
    run_link(source/'sparse.cof',args.output/'identity.ld',split_pe,sections,'sub_231F9',defined)
    assert read_sparse_pe(split_pe,identity)==linked_text(original_pe,0x230F2,420)
    placed=sorted(layout['fragments'],key=lambda p:int(p['address'],16))
    assert read_sparse_pe(source/'linked.exe',placed)==(source/'candidate.bin').read_bytes()
    image=json.loads(args.evidence.read_text());function=next(f for f in image['functions'] if f['inventory']['start']=='0x230f2')
    validate_compound_layout('SCENE_REVERSE','game_compound',function,layout)
    rejected=[]
    for label in ('missing_entry','address_changed','owner_split','prologue_changed','source_changed','compiler_offset_changed','section_changed'):
        test_layout=copy.deepcopy(layout);test_function=copy.deepcopy(function);key='game_compound'
        if label=='missing_entry':test_layout['fragments'].pop()
        elif label=='address_changed':test_layout['fragments'][1]['address']='0x231bd'
        elif label=='owner_split':test_function['inventory']['end']='0x231bc'
        elif label=='prologue_changed':
            i=next(i for c in test_function['chunks'] for i in c['instructions'] if i['ida_linear_address']=='0x231bc');i['loaded_bytes']='6804000000'
        elif label=='source_changed':key='game_scene_flow'
        elif label=='compiler_offset_changed':test_layout['fragments'][1]['compiler_offset']+=1
        else:test_layout['fragments'][0]['section']='_M2'
        try:validate_compound_layout('SCENE_REVERSE',key,test_function,test_layout)
        except ValueError:rejected.append(label)
        else:raise AssertionError(label)
    positive=args.output/'bootstrap'
    command=['python',str(Path(__file__).with_name('fd2_matching_bootstrap.py')),'--original',str(args.original),'--evidence',str(args.evidence),'--restored',str(args.linked),'--output',str(positive)]
    subprocess.run(command,check=True,capture_output=True,text=True)
    receipt=json.loads((positive/'bootstrap-receipt.json').read_text())
    assert receipt['counts']['matched_c']==1 and receipt['whole_file_equal'] and not receipt['decompilation_complete']
    assert (positive/'FD2.EXE').read_bytes()==args.original.read_bytes()
    span=receipt['restored_spans'][0]
    assert span['source_report_directory']==str(args.linked)
    assert span['source_report_sha256']==hashlib.sha256((args.linked/'report.json').read_bytes()).hexdigest()
    for key in ('compiler_version','compiler_inputs','source_key','source_path','source_sha256'):
        assert span[key]==report[key]
    bad=args.output/'bad-report';bad.mkdir();modified=copy.deepcopy(report)
    selected=next(t for t in modified['trials'] if t['exact_interval_bytes'])
    selected['exact_entry_offsets']=False
    (bad/'report.json').write_text(json.dumps(modified))
    target=bad/selected['stem'];target.mkdir();(target/'candidate.bin').write_bytes((source/'candidate.bin').read_bytes())
    command[command.index('--restored')+1]=str(bad);command[-1]=str(args.output/'must-not-exist')
    result=subprocess.run(command,capture_output=True,text=True)
    assert result.returncode!=0 and '複合入口' in result.stderr and not (args.output/'must-not-exist').exists()
    rejected.append('bootstrap_missing_entry_proof')
    result={'identity_layout_bytes':420,'original_owners_counted':1,'compiler_entries':3,'lifted_rel32_branches':2,
        'matched_span_source_provenance_verified':True,
        'target_address':'0x2328a','rejected':rejected,'source_report_sha256':hashlib.sha256((args.linked/'report.json').read_bytes()).hexdigest()}
    (args.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n');print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':main()
