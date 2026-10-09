#!/usr/bin/env python3
"""將已保存的歷史Watcom DOS組件核對後準備到明確本機輸出，只在Docker執行。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

FILES = {
    '9.01': {
        'WCC386.EXE': ('disk01/wcc386.dos/wcc386.exe', '99d79830e8bf2bf04582215583cd6c276e21226d86a9bdea197a99abb75eabe0'),
        'DOS4GW.EXE': ('disk04/dos4gw.wpk/dos4gw.exe', '535d649996de16d1e495633a9aaf60244bc717e7d6951a45a8092c3492ce4005'),
    },
    '9.5': {
        'WCC386.EXE': ('wcc386.dos/wcc386.exe', '3fe098187af3ed4bccbf184b0f0d1a21a18cf976fe72c6178f19b8eea8b13e78'),
        'DOS4GW.EXE': ('dos4gw.wpk/dos4gw.exe', 'b401506365892bd7bcb4362599279504a863f05bf37d323f732fd348bd1ef3c5'),
    },
}

def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', choices=FILES, required=True)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('本工具只在Docker執行')
    if not args.source.is_dir() or not args.output.is_dir():
        raise ValueError('來源與明確輸出目錄必須先存在')
    if args.output.stat().st_uid != os.getuid():
        raise ValueError('輸出目錄擁有權不符')
    verified = {}
    for name, (relative, expected) in FILES[args.version].items():
        source = args.source / relative
        if not source.is_file() or digest(source) != expected:
            raise ValueError('原廠組件與固定雜湊不符：' + relative)
        target = args.output / name
        if target.exists() and (not target.is_file() or target.stat().st_uid != os.getuid() or digest(target) != expected):
            raise ValueError('既有輸出不是同一組件或擁有權不符')
        verified[name] = {'source': relative, 'size': source.stat().st_size, 'sha256': expected}
    manifest = args.output / 'prepared-inputs.json'
    if manifest.exists() and manifest.stat().st_uid != os.getuid():
        raise ValueError('既有manifest擁有權不符')
    for name, entry in verified.items():
        target = args.output / name
        if not target.exists():
            target.write_bytes((args.source / entry['source']).read_bytes())
        if digest(target) != entry['sha256']:
            raise ValueError('複製後組件雜湊不符')
    result = {'schema_version': 1, 'compiler_version': args.version, 'source_root': str(args.source),
              'files': verified, 'driver_sha256': digest(Path(__file__)),
              'rights': '原廠工具僅本機研究，不加入Git或公開包。'}
    manifest.write_text(json.dumps(result, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(json.dumps(result, ensure_ascii=False))

if __name__ == '__main__':
    main()
