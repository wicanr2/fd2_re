#!/usr/bin/env python3
"""以實際編譯收據檢查共用尾段組合及不完整中繼資料的拒收。"""

import argparse
import copy
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, default=Path('/repo'))
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--linked', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('本工具只在 Docker 執行')
    if args.output.stat().st_uid != 1000:
        raise ValueError('輸出擁有權不符')
    report = json.loads((args.linked / 'report.json').read_text())
    trial = next(t for t in report['trials'] if t['exact_interval_bytes'] and len(t.get('addresses', [])) > 1)
    expected_functions = sum(p.get('kind', 'ida_function') == 'ida_function' for p in trial['function_intervals'])
    unowned = [p for p in trial['function_intervals'] if p.get('kind') == 'unowned_code']
    cases = ('valid', 'entry_unproven', 'offset_changed', 'last_function_missing')
    if unowned:
        cases += ('unowned_evidence_changed',)
    with tempfile.TemporaryDirectory(dir=args.output, prefix='groups-') as tmp:
        root = Path(tmp)
        for case in cases:
            directory = root / case
            candidate = directory / trial['stem']
            candidate.mkdir(parents=True)
            shutil.copyfile(args.linked / trial['stem'] / 'candidate.bin', candidate / 'candidate.bin')
            data = copy.deepcopy(report)
            data['trials'] = [copy.deepcopy(trial)]
            changed = data['trials'][0]
            if case == 'entry_unproven':
                changed['exact_entry_offsets'] = False
            elif case == 'offset_changed':
                changed['function_intervals'][1]['offset'] += 1
            elif case == 'last_function_missing':
                changed['function_intervals'].pop()
                changed['addresses'].pop()
            elif case == 'unowned_evidence_changed':
                next(p for p in changed['function_intervals'] if p.get('kind') == 'unowned_code')['evidence_sha256'] = '0' * 64
            (directory / 'report.json').write_text(json.dumps(data))
            output = root / (case + '-output')
            result = subprocess.run([sys.executable, str(args.repo / 'tools/fd2_matching_bootstrap.py'),
                '--original', str(args.original), '--evidence', str(args.evidence),
                '--restored', str(directory), '--output', str(output)], capture_output=True, text=True)
            if case == 'valid':
                assert result.returncode == 0, result.stderr
                receipt = json.loads((output / 'bootstrap-receipt.json').read_text())
                assert receipt['counts']['matched_c'] == expected_functions
                assert sum(s['size'] for s in receipt['restored_spans']) == trial['candidate_size']
                assert receipt.get('matched_unowned_code_bytes', 0) == sum(p['size'] for p in unowned)
                assert all(s['compiled_group'] == trial['address'] for s in receipt['restored_spans'])
                assert (output / 'FD2.EXE').read_bytes() == args.original.read_bytes()
                assert receipt['whole_file_equal'] and not receipt['decompilation_complete']
            else:
                expected = {'entry_unproven': '未證明各入口', 'offset_changed': '不連續',
                            'last_function_missing': '未覆蓋完整編譯碼',
                            'unowned_evidence_changed': '沒有綁定證據'}[case]
                assert result.returncode != 0 and expected in result.stderr, result.stderr
                assert not (output / 'FD2.EXE').exists() and not (output / 'bootstrap-receipt.json').exists()
    print(json.dumps({'positive_group_bytes': trial['candidate_size'], 'positive_functions': expected_functions,
        'matched_unowned_code_bytes': sum(p['size'] for p in unowned),
        'rejected': list(cases[1:]), 'source_report_sha256': hashlib.sha256((args.linked / 'report.json').read_bytes()).hexdigest()}, ensure_ascii=False))


if __name__ == '__main__':
    main()
