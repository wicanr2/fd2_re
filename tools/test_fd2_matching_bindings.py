#!/usr/bin/env python3
"""以原始IDA指令與LE重定位驗證具名外部符號，包含矛盾及漏失反例。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess

from fd2_matching_game_restore import BINDINGS, verify_original_bindings


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--objects', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('只在Docker執行')
    raw = args.original.read_bytes()
    image = json.loads(args.evidence.read_text())
    assert len(raw) == image['input']['size'] and hashlib.sha256(raw).hexdigest() == image['input']['sha256']
    function = next(f for f in image['functions'] if f['inventory']['start'] == '0x190ac')
    references = verify_original_bindings([function], raw, BINDINGS)
    table = [r for r in references if r['symbol'] == 'funcs_1199C']
    assert table == [{'symbol':'funcs_1199C', 'ida_linear_address':'0x19511', 'value':'0x51b91', 'kind':'original_le_fixup'}]
    rejected = []
    wrong = dict(BINDINGS)
    wrong['funcs_1199C'] = 0x1199C
    try:
        verify_original_bindings([function], raw, wrong)
    except ValueError as error:
        assert 'funcs_1199C' in str(error)
        rejected.append('name_digits_used_as_address')
    else:
        raise AssertionError('舊錯誤綁定沒有拒收')
    contradiction = copy.deepcopy(function)
    i = next(i for c in contradiction['chunks'] for i in c['instructions'] if i['ida_linear_address'] == '0x19511')
    i['loaded_bytes'] = 'ff1485991b0500'
    try:
        verify_original_bindings([contradiction, function], raw, BINDINGS)
    except ValueError:
        rejected.append('earlier_reference_conflicts_with_later_valid_reference')
    else:
        raise AssertionError('後一筆正確參照掩蓋矛盾')
    bad_raw = bytearray(raw)
    at = int(i['file_offset'],16)
    bad_raw[at] ^= 1
    try:
        verify_original_bindings([function], bytes(bad_raw), BINDINGS)
    except ValueError:
        rejected.append('original_instruction_changed')
    else:
        raise AssertionError('原始指令變動沒有拒收')
    # 公開CLI以同一驗證器拒收未知具名位址，必須在任何連結輸出前結束。
    args.output.mkdir(parents=True,exist_ok=True)
    altered = copy.deepcopy(image)
    f = next(f for f in altered['functions'] if f['inventory']['start']=='0x190ac')
    j = next(j for c in f['chunks'] for j in c['instructions'] if j['ida_linear_address']=='0x19511')
    j['loaded_bytes'] = 'ff1485991b0500'
    evidence = args.output / 'contradictory-evidence.json'
    evidence.write_text(json.dumps(altered))
    output = args.output / 'must-not-exist'
    result = subprocess.run(['python',str(Path(__file__).with_name('fd2_matching_game_restore.py')),'link','--objects',str(args.objects),'--original',str(args.original),'--evidence',str(evidence),'--output',str(output)],capture_output=True,text=True)
    assert result.returncode != 0 and 'LE' in result.stderr and not output.exists(), result.stderr
    rejected.append('public_link_rejects_before_output')
    receipt = {'input_sha256':image['input']['sha256'], 'tool':image['tool'],
               'reference_count':len(references), 'indirect_table_reference':table[0], 'rejected':rejected}
    (args.output/'result.json').write_text(json.dumps(receipt,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(receipt,ensure_ascii=False))


if __name__=='__main__': main()
