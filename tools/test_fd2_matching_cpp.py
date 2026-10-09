#!/usr/bin/env python3
"""以真實C++物件驗證固定C包裝、前端來源與輸出前拒收。只在Docker執行。"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import os

from fd2_matching_game_restore import CPP_WRAPPER, SOURCE_DATE_EPOCH, validate_cpp_wrapper


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--objects', type=Path, required=True)
    parser.add_argument('--linked', type=Path, required=True)
    parser.add_argument('--original', type=Path, required=True)
    parser.add_argument('--evidence', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('本工具只在Docker執行')
    args.output.mkdir(parents=True, exist_ok=True)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError('輸出擁有權不符')
    source = json.loads((args.objects/'compile-report.json').read_text())
    linked = json.loads((args.linked/'report.json').read_text())
    assert source['compiler_frontend'] == linked['compiler_frontend'] == 'cpp'
    assert linked['matched_addresses'] == ['0x2d3ff']
    validate_cpp_wrapper(args.objects, source)
    tool = Path(__file__).with_name('fd2_matching_game_restore.py')
    rejected = []
    for name in ('missing', 'bytes_changed', 'time_changed', 'hash_changed', 'frontend_changed', 'component_changed', 'version_unregistered'):
        objects = args.output/name/'objects'
        objects.mkdir(parents=True)
        report = json.loads(json.dumps(source))
        shutil.copy2(args.objects/'GAME.C', objects/'GAME.C')
        if name != 'missing':
            shutil.copy2(args.objects/'GAME.CPP', objects/'GAME.CPP')
        if name == 'bytes_changed':
            (objects/'GAME.CPP').write_bytes(CPP_WRAPPER+b'/* changed */\n')
            os.utime(objects/'GAME.CPP', (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
        elif name == 'time_changed':
            os.utime(objects/'GAME.CPP', (SOURCE_DATE_EPOCH+2, SOURCE_DATE_EPOCH+2))
        elif name == 'hash_changed':
            report['cpp_wrapper']['sha256'] = '0'*64
        elif name == 'frontend_changed':
            report['compiler_frontend'] = 'c'
        elif name == 'component_changed':
            report['compiler_inputs']['WPP386.EXE'] = '0'*64
        elif name == 'version_unregistered':
            report['compiler_version'] = '9.5'
        (objects/'compile-report.json').write_text(json.dumps(report)+'\n')
        destination = args.output/name/'unexpected-output'
        p = subprocess.run([sys.executable, str(tool), 'link', '--objects', str(objects),
                            '--evidence', str(args.evidence), '--original', str(args.original),
                            '--output', str(destination)], capture_output=True, text=True)
        assert p.returncode and not destination.exists(), (name, p.stdout, p.stderr)
        expected = ('實際C介面包裝', 'C介面包裝的SHA', 'compiler 組件登記不符', 'C++前端尚未鎖版')
        assert any(text in p.stderr for text in expected), (name, p.stderr)
        rejected.append(name)
    bootstrap = args.output/'bootstrap'
    p = subprocess.run([sys.executable, str(tool.with_name('fd2_matching_bootstrap.py')),
                        '--original', str(args.original), '--evidence', str(args.evidence),
                        '--restored', str(args.linked), '--output', str(bootstrap)],
                       capture_output=True, text=True)
    assert p.returncode == 0, p.stderr
    rebuilt = json.loads((bootstrap/'bootstrap-receipt.json').read_text())
    assert rebuilt['whole_file_equal'] and not rebuilt['decompilation_complete']
    assert rebuilt['counts']['matched_c'] == 1
    span = rebuilt['restored_spans'][0]
    assert span['ida_linear_address'] == '0x2d3ff' and span['size'] == 279
    assert span['compiler_frontend'] == 'cpp' and span['cpp_wrapper'] == linked['cpp_wrapper']
    assert span['compiler_inputs'] == linked['compiler_inputs']
    result = {'matched_c_bytes': 279, 'source_frontend_preserved': True,
              'wrapper_sha256': hashlib.sha256(CPP_WRAPPER).hexdigest(),
              'rejected_before_output': rejected, 'whole_file_equal': True,
              'scope': '只驗證C語法來源及C++前端包裝，不推定原作語言或全檔C完成。'}
    (args.output/'result.json').write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
