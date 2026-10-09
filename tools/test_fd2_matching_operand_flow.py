#!/usr/bin/env python3
"""以四個完整原始函式驗證文字對應不能取代逐byte匹配。"""
import argparse
import json
from pathlib import Path
from types import SimpleNamespace

from fd2_matching_register_diagnostic import prepare


def main():
    p=argparse.ArgumentParser(description=__doc__)
    for name in ('linked','original','evidence','output'):p.add_argument('--'+name,type=Path,required=True)
    a=p.parse_args();assert Path('/.dockerenv').exists()
    report=json.loads((a.linked/'report.json').read_text())
    assert report['source_key']=='game_operand_flow' and len(report['trials'])==108 and report['matched_addresses']==[]
    expected=(('OPERAND_PANEL_0','0x17eef',209,True,{'eax':'ebx','ebx':'eax'},'0x17f04','0x17f0d'),
              ('OPERAND_STATE_0','0x22af6',208,True,{'esi':'eax'},'0x22b68','0x22b74'),
              ('OPERAND_IDLE_0','0x2b9a1',129,False,None,'0x2ba0b','0x2ba17'),
              ('OPERAND_SLOT_0','0x1b722',46,True,{'eax':'edx','edx':'eax'},'0x1b73f','0x1b747'))
    diagnostics=[]
    for macro,address,size,equivalent,changed,start,end in expected:
        trial=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-d'+macro])
        result=prepare(SimpleNamespace(linked=a.linked,original=a.original,evidence=a.evidence,stem=trial['stem']))
        assert result['ida_linear_address']==address and result['original_size']==result['candidate_size']==size
        assert not result['exact_interval_bytes'] and len(result['differing_byte_offsets'])==3
        assert result['instruction_text_equivalent_under_register_map']==equivalent
        assert result['register_map_interval']['start']==start and result['register_map_interval']['end']==end
        if equivalent:assert {k:v for k,v in result['candidate_to_original_registers'].items() if k!=v}==changed
        else:assert result['candidate_to_original_registers'] is None
        assert result['diagnostic_only'] and not result['increases_c_coverage'];diagnostics.append(result)
    # 明示宣告的錯誤序言仍是完整負例，不從局部register結果推定ABI。
    prologues=[]
    for prefix,size in (('OPERAND_PANEL_',209),('OPERAND_STATE_',208),('OPERAND_IDLE_',129)):
        trial=next(t for t in report['trials'] if t['flags']==['-mf','-3s','-d'+prefix+'1'])
        assert trial['original_size']==size and not trial['exact_interval_bytes'] and trial['first_difference_offset']==1
        prologues.append({'macro':prefix+'1','original_size':size,'candidate_size':trial['candidate_size'],'first_difference_offset':1})
    a.output.mkdir(parents=True,exist_ok=False)
    result={'schema_version':1,'complete_original_function_bytes':592,'diagnostics':diagnostics,
        'exact_matching_remains_false':True,'non_uniform_register_lifetimes_remain_unmapped':True,
        'entry_declaration_negatives':prologues,'new_c_matches':0,'increases_c_coverage':False}
    (a.output/'result.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'complete_functions':4,'complete_original_function_bytes':592,'uniform_text_maps':3,
        'non_uniform_map_rejected':True,'entry_declaration_negatives':3,'new_c_matches':0},ensure_ascii=False))


if __name__=='__main__':main()
