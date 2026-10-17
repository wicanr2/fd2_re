#!/usr/bin/env python3
"""從正式完整包獨立核對三平台內容、引擎與素材，Linux解包到容器暫存。"""
import argparse
import hashlib
import json
import plistlib
from pathlib import Path
import struct
import subprocess
import tempfile
import zipfile


def sha(path):
    h = hashlib.sha256()
    with Path(path).open('rb') as f:
        for block in iter(lambda:f.read(1048576),b''):
            h.update(block)
    return h.hexdigest()


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('release',type=Path)
    args=p.parse_args()
    assert Path('/.dockerenv').exists()
    full=args.release/'full-local'
    meta=json.loads((full/'VALIDATION.json').read_text())
    asset_document=json.loads((full/'ASSET-SHA256.json').read_text())
    rows=asset_document['common']
    assert len(rows)==meta['asset_file_count']
    assert sha(full/'ASSET-SHA256.json')==meta['asset_files_manifest_sha256']
    result={'version':meta['version'],'status':'STATIC-CONTENT-PASSED','public_distribution':False,'asset_files_per_platform':len(rows),'platforms':{}}
    for entry in meta['packages']:
        file=args.release/entry['path']
        assert file.stat().st_size==entry['bytes'] and sha(file)==entry['sha256'],file
    for platform in ['windows','macos']:
        archive=next(full.glob('*'+platform+'*.zip'))
        with zipfile.ZipFile(archive) as z:
            assert z.testzip() is None
            names=z.namelist()
            assert len(names)==len(set(names))
            for info in z.infolist():
                if any(ord(c)>127 for c in info.filename):
                    assert info.flag_bits&0x800,info.filename
            base=names[0].split('/')[0]+'/'
            resources=base+('FD2.app/Contents/Resources/' if platform=='macos' else '')
            overrides={row['path']:row for row in asset_document['platform_overrides'].get(platform,[])}
            for common in rows:
                row=overrides.get(common['path'],common)
                raw=z.read(resources+row['path'])
                assert len(raw)==row['bytes'] and hashlib.sha256(raw).hexdigest()==row['sha256'],row['path']
            assert z.read(resources+'README.txt').startswith(b'\xef\xbb\xbf')
            assert b'\r\n' in z.read(resources+'README.txt')
            assert b'RRSAL' in z.read(resources+'LICENSE')
            assert json.loads(z.read(resources+'FD2-LOCAL-FULL.json'))['version']==meta['version']
            if platform=='windows':
                binary=z.read(base+'fd2.exe')
                pe=struct.unpack_from('<I',binary,0x3c)[0]
                assert binary[:2]==b'MZ' and binary[pe:pe+4]==b'PE\0\0'
                assert struct.unpack_from('<H',binary,pe+4)[0]==0x8664
                assert struct.unpack_from('<H',binary,pe+92)[0]==2
                launch=z.read(base+'開始遊戲.bat')
                assert launch.isascii() and b'\r\n' in launch and not launch.startswith(b'\xef\xbb\xbf')
                assert b'FD2_ASSET_PACK=%~dp0pack' in launch and b'%~dp0fd2.exe' in launch
                assert z.getinfo(base+'開始遊戲.bat').flag_bits&0x800
                architecture=['x86_64']
            else:
                binary=z.read(base+'FD2.app/Contents/MacOS/fd2-engine')
                plist=plistlib.loads(z.read(base+'FD2.app/Contents/Info.plist'))
                assert plist['CFBundleExecutable']=='fd2' and plist['FD2ReleaseVersion']==meta['version']
                magic,count=struct.unpack_from('>II',binary)
                assert magic==0xcafebabe and count==2
                cpus={struct.unpack_from('>I',binary,8+i*20)[0] for i in range(count)}
                assert cpus=={0x01000007,0x0100000c}
                for name in ['fd2','fd2-engine']:
                    info=z.getinfo(base+'FD2.app/Contents/MacOS/'+name)
                    assert (info.external_attr>>16)&0o111
                launch=z.read(base+'FD2.app/Contents/MacOS/fd2')
                assert b'../Resources/pack' in launch and b'fd2-engine' in launch
                architecture=['x86_64','arm64']
            assert hashlib.sha256(binary).hexdigest()==meta['engine_binary_sha256'][platform]
            result['platforms'][platform]={'zip_crc':True,'assets_verified':len(rows),'architecture':architecture,'engine_identical_to_release':True,'launcher':True,'native_release_crlf_files':len(overrides),'native_full_runtime':'not_run'}
    image=next(full.glob('*.AppImage'))
    with image.open('rb') as f:
        head=f.read(20)
    assert head[:4]==b'\x7fELF' and head[8:11]==b'AI\x02'
    with tempfile.TemporaryDirectory(prefix='fd2-full-audit-') as temporary:
        subprocess.run([str(image.resolve()),'--appimage-extract'],cwd=temporary,stdout=subprocess.DEVNULL,check=True)
        root=Path(temporary)/'squashfs-root'
        for row in rows:
            file=root/row['path']
            assert file.stat().st_size==row['bytes'] and sha(file)==row['sha256'],file
        binary=root/'usr/bin/fd2'
        assert sha(binary)==meta['engine_binary_sha256']['linux']
        raw=binary.read_bytes()[:20]
        assert raw[:4]==b'\x7fELF' and raw[4]==2 and struct.unpack_from('<H',raw,18)[0]==62
        assert 'FD2_ASSET_PACK="${APPDIR}/pack"' in (root/'AppRun').read_text()
        assert b'RRSAL' in (root/'LICENSE').read_bytes()
        result['platforms']['linux']={'assets_verified':len(rows),'architecture':['x86_64'],'engine_identical_to_release':True,'appimage_type':2,'launcher':True}
    (args.release/'smoke/full-local-content.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__=='__main__':
    main()
