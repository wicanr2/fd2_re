#!/usr/bin/env python3
"""由已發布引擎與本機合法素材組裝三平台非 HD 完整包，只在 Docker 執行。"""
from __future__ import annotations

import argparse
import calendar
from datetime import datetime
import hashlib
import json
import os
import re
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import zipfile

sys.path.insert(0, str(Path(__file__).resolve().parent))
from validate_separated_asset_pack import validate


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as stream:
        for block in iter(lambda: stream.read(1048576), b''):
            h.update(block)
    return h.hexdigest()


def dump(path, value):
    Path(path).write_text(json.dumps(value, ensure_ascii=False, indent=2) + '\n')


def owned(path):
    path = Path(path)
    assert path.stat().st_uid == os.getuid(), f'輸出擁有權不符：{path}'


def extract_zip(path, destination):
    with zipfile.ZipFile(path) as archive:
        assert archive.testzip() is None
        for entry in archive.infolist():
            relative = Path(entry.filename)
            assert not relative.is_absolute() and '..' not in relative.parts
            target = destination / relative
            if entry.is_dir():
                target.mkdir(parents=True, exist_ok=True)
            else:
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(archive.read(entry))
                target.chmod((entry.external_attr >> 16) & 0o777 or 0o644)
    return next(destination.iterdir())


def copy_runtime(source, target):
    for path in sorted(source.rglob('*')):
        if path.is_file() and not path.name.endswith(('.bak', '.tmp')):
            dest = target / path.relative_to(source)
            # The immutable release supplies the authoritative public scripts.
            if dest.exists():
                continue
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.copyfile(path, dest)


def write_zip(root, output, version):
    assert not output.exists(), f'禁止覆寫：{output}'
    with zipfile.ZipFile(output, 'w', zipfile.ZIP_DEFLATED, compresslevel=6) as z:
        for path in sorted(root.rglob('*')):
            if not path.is_file():
                continue
            date = datetime.strptime(version.rsplit('-', 1)[1], '%Y%m%d')
            entry = zipfile.ZipInfo((Path(root.name) / path.relative_to(root)).as_posix(), (date.year, date.month, date.day, 0, 0, 0))
            entry.create_system = 3
            entry.external_attr = (stat.S_IFREG | (path.stat().st_mode & 0o777)) << 16
            entry.compress_type = zipfile.ZIP_DEFLATED
            z.writestr(entry, path.read_bytes())
    with zipfile.ZipFile(output) as z:
        assert z.testzip() is None
        for path in root.rglob('*'):
            if path.is_file():
                name = (Path(root.name) / path.relative_to(root)).as_posix()
                assert hashlib.sha256(z.read(name)).hexdigest() == sha(path), name


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['stage', 'package'])
    parser.add_argument('--version', default='v.1.0.0-20261017')
    args = parser.parse_args()
    assert Path('/.dockerenv').exists(), '只在 Docker 內執行'
    repo = Path(__file__).resolve().parents[1]
    version = args.version
    assert re.fullmatch(r'v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}', version), version
    datetime.strptime(version.rsplit('-', 1)[1], '%Y%m%d')
    assert version == (repo / 'VERSION').read_text().strip()
    release = repo / 'dist-all' / version
    work = repo / 'work' / ('full-local-' + version)
    owned(repo / 'work'); owned(release)
    if args.action == 'stage':
        assert not work.exists(), '已有 staging，先核對，不自動覆寫'
        pack = repo / 'remake/generated-assets/fd2-original-b97caf22'
        runtime = repo / 'remake/assets'
        errors = validate(pack / 'manifest.json', repo / 'org_game/炎龍騎士團/FLAME2', runtime, repo / 'docs/data/fd2-source-resource-coverage-summary.json')
        assert not errors, errors[:20]
        work.mkdir()
        source = json.loads((release / 'SHA256SUMS.json').read_text())
        for entry in source['files']:
            original = release / 'patch' / entry['file']
            assert original.stat().st_size == entry['size'] and sha(original) == entry['sha256'], original
        # Public release hashes are immutable inputs, also checked independently before this tool.
        dump(work / 'public-manifest.json', source)
        inputs = []
        for path in sorted((release / 'patch').iterdir()):
            inputs.append({'name': path.name, 'bytes': path.stat().st_size, 'sha256': sha(path)})
        linux = work / 'linux/AppDir'
        linux.parent.mkdir()
        appimage = next((release / 'patch').glob('*linux*.AppImage'))
        subprocess.run([str(appimage), '--appimage-extract'], cwd=linux.parent, check=True, stdout=subprocess.DEVNULL)
        (linux.parent / 'squashfs-root').rename(linux)
        win = extract_zip(next((release / 'patch').glob('*windows*.zip')), work / 'windows')
        mac = extract_zip(next((release / 'patch').glob('*macos*.zip')), work / 'macos')
        mac_full = mac.with_name(f'FD2-{version}-macos-universal-full-local')
        mac.rename(mac_full)
        mac = mac_full
        engine = work / 'windows' / (f'FD2-{version}-windows-x86_64-full-local')
        win.rename(engine)
        # Keep engine assets and separated original assets in their own namespaces.
        mac_resources = mac / 'FD2.app/Contents/Resources'
        for target in [linux, engine, mac_resources]:
            public_hashes = {p.relative_to(target / 'assets'): sha(p) for p in (target / 'assets').rglob('*') if p.is_file()}
            copy_runtime(runtime, target / 'assets')
            for relative, expected in public_hashes.items():
                assert sha(target / 'assets' / relative) == expected, f'公開素材覆蓋漂移：{relative}'
            shutil.copytree(pack, target / 'pack')
        linux_binary = linux / 'usr/bin/fd2'
        win_binary = engine / 'fd2.exe'
        mac_binary = mac / 'FD2.app/Contents/MacOS/fd2'
        binaries = {'linux': sha(linux_binary), 'windows': sha(win_binary), 'macos': sha(mac_binary)}
        app_run = (linux / 'AppRun').read_text()
        app_run = app_run.replace('exec "${APPDIR}/usr/bin/fd2"', 'export FD2_ASSET_PACK="${APPDIR}/pack"\nexec "${APPDIR}/usr/bin/fd2"')
        (linux / 'AppRun').write_text(app_run); (linux / 'AppRun').chmod(0o755)
        bat = '@echo off\r\nset "FD2_ASSET_PACK=%~dp0pack"\r\n"%~dp0fd2.exe" %*\r\n'
        (engine / '開始遊戲.bat').write_bytes(bat.encode('utf-8'))
        mac_binary.rename(mac_binary.with_name('fd2-engine'))
        mac_binary.write_text('#!/bin/sh\nset -eu\nhere=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)\nexport FD2_ASSET_PACK="$here/../Resources/pack"\nexec "$here/fd2-engine" "$@"\n')
        mac_binary.chmod(0o755)
        metadata = {'schema_version': 1, 'kind': 'fd2-local-full-non-hd', 'version': version, 'engine_source_commit': source['source_commit'], 'public_distribution': False, 'theme': 'faithful-original-pixel-art', 'hd': False, 'asset_pack_manifest_sha256': sha(pack / 'manifest.json'), 'public_inputs': inputs, 'engine_binary_sha256': binaries, 'save_location': '$XDG_DATA_HOME/fd2_re or ~/.local/share/fd2_re', 'limits': ['Windows／macOS完整包尚未實機驗收，已發布的相同引擎已通過原生自檢。', 'Windows／macOS未簽署，macOS未公證。', '30章為受控流程抽樣，並非所有分支或自然難度驗收。']}
        text = f'炎龍騎士團 2 重製 {version} 本機完整版\n\n本包已含完整合法來源的分離素材，採原版像素風格，尚未 HD。\n只供原版合法持有者本機使用，不得公開上傳或再散布。\n\nWindows：解壓整個資料夾後執行「開始遊戲.bat」。\nmacOS：解壓後開啟 FD2.app。尚未簽署或公證。\nLinux：執行完整包 AppImage；無 FUSE 時加 --appimage-extract-and-run。\n存檔與設定：$XDG_DATA_HOME/fd2_re；未設定時為家目錄的 .local/share/fd2_re。\n\n方向鍵移動，Enter／Space 確認，Esc 返回。F5 快存，F9 快讀。\n使用全新設定時是原版忠實主題；沿用舊設定請確認沒有選擇現代主題。\n引擎授權見 LICENSE；原版素材不適用引擎授權。\n'
        for target in [linux, engine, mac_resources]:
            dump(target / 'FD2-LOCAL-FULL.json', metadata)
            (target / '完整包說明.txt').write_bytes(('\ufeff' + text.replace('\n', '\r\n')).encode('utf-8'))
            (target / 'README.txt').write_bytes(('\ufeff' + text.replace('\n', '\r\n')).encode('utf-8'))
        asset_rows = []
        for kind in ['assets', 'pack']:
            for path in sorted((linux / kind).rglob('*')):
                if path.is_file():
                    asset_rows.append({'path': path.relative_to(linux).as_posix(), 'bytes': path.stat().st_size, 'sha256': sha(path)})
        asset_document = {'schema_version': 1, 'common': asset_rows, 'platform_overrides': {}}
        for platform, target in [('windows', engine), ('macos', mac_resources)]:
            overrides = []
            for row in asset_rows:
                file = target / row['path']
                actual = sha(file)
                if actual == row['sha256']:
                    continue
                base = (linux / row['path']).read_bytes()
                raw = file.read_bytes()
                assert base.replace(b'\r\n', b'\n') == raw.replace(b'\r\n', b'\n'), file
                raw.decode('utf-8')
                overrides.append({'path': row['path'], 'bytes': len(raw), 'sha256': actual, 'reason': 'native_release_crlf_only'})
            asset_document['platform_overrides'][platform] = overrides
        dump(work / 'asset-files.json', asset_document)
        metadata['asset_file_count'] = len(asset_rows)
        metadata['asset_files_manifest_sha256'] = sha(work / 'asset-files.json')
        metadata['public_platform_crlf_overrides'] = {key: len(value) for key, value in asset_document['platform_overrides'].items()}
        metadata['staging'] = {'linux': str(linux.relative_to(repo)), 'windows': str(engine.relative_to(repo)), 'macos': str(mac.relative_to(repo))}
        for target in [linux, engine, mac_resources]:
            dump(target / 'FD2-LOCAL-FULL.json', metadata)
        dump(work / 'staging.json', metadata)
        print(json.dumps({'staged': metadata['staging'], 'asset_files': len(asset_rows), 'asset_bytes': sum(r['bytes'] for r in asset_rows)}))
        return
    metadata = json.loads((work / 'staging.json').read_text())
    asset_document = json.loads((work / 'asset-files.json').read_text())
    rows = asset_document['common']
    for platform, relative in metadata['staging'].items():
        root = repo / relative
        if platform == 'macos':
            root = root / 'FD2.app/Contents/Resources'
        overrides = {row['path']: row for row in asset_document['platform_overrides'].get(platform, [])}
        for common in rows:
            row = overrides.get(common['path'], common)
            path = root / row['path']
            assert path.stat().st_size == row['bytes'] and sha(path) == row['sha256'], path
    out = release / 'full-local'
    assert not out.exists(), '正式完整版已存在，禁止覆寫'
    out.mkdir()
    packages = []
    for platform, arch in [('windows', 'x86_64'), ('macos', 'universal')]:
        root = repo / metadata['staging'][platform]
        output = out / f'FD2-{version}-{platform}-{arch}-full-local.zip'
        write_zip(root, output, version)
        packages.append({'path': output.relative_to(release).as_posix(), 'bytes': output.stat().st_size, 'sha256': sha(output)})
    output = out / f'FD2-{version}-linux-x86_64-full-local.AppImage'
    epoch = calendar.timegm(datetime.strptime(version.rsplit('-', 1)[1], '%Y%m%d').timetuple())
    environment = dict(os.environ, ARCH='x86_64', APPIMAGE_EXTRACT_AND_RUN='1', SOURCE_DATE_EPOCH=str(epoch))
    subprocess.run(['/opt/appimage-tools/appimagetool.AppImage', '--runtime-file', '/opt/appimage-tools/runtime-x86_64', '--mksquashfs-opt', '-processors', '--mksquashfs-opt', '2', str(repo / metadata['staging']['linux']), str(output)], env=environment, check=True)
    output.chmod(0o755)
    packages.append({'path': output.relative_to(release).as_posix(), 'bytes': output.stat().st_size, 'sha256': sha(output)})
    metadata['packages'] = packages
    dump(out / 'VALIDATION.json', metadata)
    shutil.copyfile(work / 'asset-files.json', out / 'ASSET-SHA256.json')
    dump(out / 'SHA256SUMS.json', {'version': version, 'public_distribution': False, 'files': packages})
    print(json.dumps(packages))


if __name__ == '__main__':
    main()
