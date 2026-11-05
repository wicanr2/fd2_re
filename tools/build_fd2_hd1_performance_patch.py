#!/usr/bin/env python3
"""建立已有完整版可使用的Windows HD1效能替換測試包，不含原版或HD素材。"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import struct
import zipfile


ROOT = Path(__file__).resolve().parents[1]


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ['binary', 'build-info', 'gui-receipt', 'tested-linux', 'quality-gpu-log', 'out']:
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--version', required=True)
    p.add_argument('--source-commit', required=True)
    args = p.parse_args()
    assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}', args.version)
    assert re.fullmatch(r'[0-9a-f]{40}', args.source_commit)
    assert args.out.resolve() == (ROOT/'dist-all'/args.version).resolve()
    assert not args.out.exists() and args.out.parent.is_dir()
    assert args.out.parent.stat().st_uid == os.getuid()
    raw = args.binary.read_bytes()
    assert raw[:2] == b'MZ'
    pe = struct.unpack_from('<I', raw, 0x3c)[0]
    assert raw[pe:pe+4] == b'PE\0\0'
    assert struct.unpack_from('<H', raw, pe+4)[0] == 0x8664
    assert struct.unpack_from('<H', raw, pe+24)[0] == 0x20b
    assert struct.unpack_from('<H', raw, pe+24+68)[0] == 2
    assert args.version.encode() in raw
    build_info = args.build_info.read_text()
    assert 'GOOS=windows' in build_info and 'GOARCH=amd64' in build_info
    # Go 1.22 load/pkg.go deliberately omits -ldflags under -trimpath.
    # Validate the actual embedded version above, not an absent metadata field.
    assert '\tbuild\t-trimpath=true' in build_info and 'CGO_ENABLED=0' in build_info
    gui = json.loads(args.gui_receipt.read_text())
    assert gui['status'] == 'PASS'
    assert gui['binary_sha256'] == digest(args.tested_linux.read_bytes())
    for field in ['natural_attack_return_checked', 'theme_save_bytes_equal', 'node_boundary_save_checked',
                  'cold_loaded_units_equal', 'new_process_title_load']:
        assert gui[field] is True, field
    gpu_log = args.quality_gpu_log.read_text()
    assert '--- PASS: TestHDTextureQualityBattleGeometryGPU' in gpu_log and 'FAIL' not in gpu_log
    sources = {}
    for file in sorted((ROOT/'remake').rglob('*.go')):
        if not file.name.endswith('_test.go'):
            sources[file.relative_to(ROOT).as_posix()] = digest(file.read_bytes())
    for name in ['remake/go.mod', 'remake/go.sum']:
        sources[name] = digest((ROOT/name).read_bytes())
    source_hash = digest(json.dumps(sources, sort_keys=True, separators=(',', ':')).encode())
    metadata = {
        'schema': 'fd2-hd1-windows-performance-patch-v1', 'version': args.version,
        'status': 'LOCAL-WINDOWS-TEST-PATCH', 'source_commit': args.source_commit,
        'source_tree_sha256': source_hash, 'source_files_sha256': sources,
        'binary_sha256': digest(raw), 'build_info_sha256': digest(args.build_info.read_bytes()),
        'version_verification': 'PE內含指定完整版號；Go trimpath建置資訊省略ldflags，另核對架構與CGO。',
        'linux_normal_gui_receipt_sha256': digest(args.gui_receipt.read_bytes()),
        'linux_normal_gui_binary_sha256': gui['binary_sha256'],
        'post_gui_delta': '最後僅調整錯誤計畫的畫布保留時機；正常GUI二進位與最終封包分列。',
        'final_quality_gpu_log_sha256': digest(args.quality_gpu_log.read_bytes()),
        'windows_runtime_validation': 'PENDING-USER-HARDWARE',
        'base_install': 'v.1.1.2-20261105完整版；保留既有pack、modern及字型。',
        'default_quality': 'balanced', 'quality_modes': ['full', 'balanced', 'light'],
        'rights': 'RRSAL-1.0引擎替換檔；不含原版資料、HD素材、音樂或字型。',
        'limits': 'Windows交叉編譯及PE／版本／內容已檢查；Windows實機啟動與60fps仍待驗證，沒有建立公開Release。',
    }
    readme = f'''# HD1效能替換測試包 {args.version}

請在現有v.1.1.2-20261105 Windows完整版目錄使用。

1. 先備份原本的fd2.exe。
2. 將本包的fd2.exe與兩份bat放到原fd2.exe所在目錄，保留pack、modern與其他檔案。
3. 用原本的「開始遊戲.bat」啟動。工具列「影像」預設均衡，可切完整／輕量並查看後端及fps。

均衡與輕量將地圖／戰鬥GPU畫布降為1280×800、960×600，原計畫座標及顯示尺寸保持。
文字、獨立介面與原始母稿保持。開場、黃金之鑰等專用場景目前維持原品質。

需要回報效能時，使用start-fd2-performance.bat，正常遊玩後關閉遊戲。
它把逐秒fps、GPU後端、實體畫布尺寸、程序日誌與Windows配接器清單存到
%LOCALAPPDATA%\\fd2_re\\performance。配接器清單不代表遊戲選中的硬體。
也可用start-fd2-directx11.bat比較既有DirectX 11路徑，設定只作用於該次程序。

Linux軟體GPU已驗證正常選人、攻擊返回、F2、保存與新程序LOAD。
其後錯誤計畫的畫布保留修正另經三品質GPU回歸；收據與二進位雜湊分列。
Windows實機啟動、批次檔與60fps仍待測試。不含原版或HD素材。

來源提交：{args.source_commit}
引擎SHA-256：{digest(raw)}
Microsoft配接器欄位：https://learn.microsoft.com/en-us/windows/win32/cimwin32prov/win32-videocontroller
'''
    files = {'fd2.exe': raw, 'README.md': readme.encode(), 'LICENSE': (ROOT/'LICENSE').read_bytes(),
             'SOURCE-PROVENANCE.json': (json.dumps(metadata, ensure_ascii=False, indent=2)+'\n').encode(),
             'BUILD-INFO.txt': args.build_info.read_bytes()}
    for source, name in [('run_fd2_windows_performance.bat', 'start-fd2-performance.bat'),
                         ('run_fd2_windows_directx11.bat', 'start-fd2-directx11.bat')]:
        files[name] = (ROOT/'tools'/source).read_bytes().replace(b'\r\n', b'\n').replace(b'\n', b'\r\n')
    patch = args.out/'patch'
    patch.mkdir(parents=True)
    archive = patch/f'FD2-{args.version}-windows-x86_64-performance-patch.zip'
    with zipfile.ZipFile(archive, 'w', compression=zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for name, content in files.items():
            z.writestr(name, content)
    with zipfile.ZipFile(archive) as z:
        assert z.testzip() is None and set(z.namelist()) == set(files)
        for name, content in files.items():
            assert digest(z.read(name)) == digest(content), name
    manifest = dict(metadata)
    manifest['files'] = [{'file': archive.relative_to(args.out).as_posix(), 'size_bytes': archive.stat().st_size,
                          'sha256': digest(archive.read_bytes()),
                          'contents_sha256': {name: digest(content) for name, content in files.items()}}]
    (args.out/'SHA256SUMS.json').write_text(json.dumps(manifest, ensure_ascii=False, indent=2)+'\n')
    assert archive.stat().st_uid == os.getuid()
    print(json.dumps({'version': args.version, 'archive': str(archive), 'sha256': manifest['files'][0]['sha256'],
                      'status': manifest['status']}, ensure_ascii=False))


if __name__ == '__main__':
    main()
