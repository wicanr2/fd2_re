#!/usr/bin/env python3
"""在Docker驗證候選編製的序列／平行一致性及拒收守衛，不變更正式清冊。"""
import argparse
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

from PIL import Image

import build_modern_hd_map_sprites as builder
import verify_modern_hd_map_sprites as verifier


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--source', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file()
    assert not args.report.exists() and args.report.parent.stat().st_uid == 1000
    assets = [a for a in json.loads(args.catalog.read_text())['assets']
              if a.get('role') == 'map_sprite_set' and a['source_group'] in (0, 52)]
    assert {a['source_group'] for a in assets} == {0, 52}
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        source = root/'source'
        source.mkdir()
        for asset in assets:
            for name in asset['files']+[asset['master_file']]:
                shutil.copyfile(args.source/name, source/name)
        repeat = next(a for a in assets if a['source_group'] == 52)
        components = builder.master_components(Image.open(source/repeat['master_file']), 12)
        components[6][0].save(source/'canonical-phase.png')
        overrides = root/'frames.json'
        overrides.write_text(json.dumps({'52': {'6': 'canonical-phase.png'}}))
        command = [sys.executable, str(Path(builder.__file__)), '--catalog', str(args.catalog),
                   '--source', str(source), '--frame-overrides', str(overrides),
                   '--group', '0', '--group', '52']
        for workers in (1, 2):
            subprocess.run(command+['--output', str(root/f'output-{workers}'), '--workers', str(workers)],
                           check=True, stdout=subprocess.DEVNULL, timeout=180)
        output, parallel = root/'output-1', root/'output-2'
        assert sorted(p.name for p in output.iterdir()) == sorted(p.name for p in parallel.iterdir())
        assert all(p.read_bytes() == (parallel/p.name).read_bytes() for p in output.iterdir())
        passed = ['serial and two-worker PNG/manifest bytes identical']
        manifest = output/'hd-map-sprites.json'
        data = json.loads(manifest.read_text())
        frames = next(g for g in data['groups'] if g['source_group'] == 52)['frames']
        assert frames[8]['source_file'] == frames[6]['source_file'] == 'canonical-phase.png'
        for key in ('source_sha256', 'sha256', 'source_cell', 'source_ink', 'fit_rect'):
            assert frames[8][key] == frames[6][key]
        assert verifier.verify(manifest, args.catalog, source, True)['status'] == 'PASS'
        passed.append('canonical override repeats preserve bytes, provenance and fit')
        try:
            verifier.verify(manifest, args.catalog, source)
        except AssertionError:
            passed.append('subset refused without explicit scope')
        else:
            raise AssertionError('accepted incomplete coverage')
        original_manifest = manifest.read_bytes()

        def rejects(name):
            try:
                verifier.verify(manifest, args.catalog, source, True)
            except AssertionError:
                passed.append(name)
            else:
                raise AssertionError('accepted invalid candidate: '+name)

        frames[8]['source_file'] = repeat['master_file']
        manifest.write_text(json.dumps(data))
        rejects('wrong repeated-frame provenance refused')
        manifest.write_bytes(original_manifest)
        data = json.loads(original_manifest)
        data['groups'][0]['frames'][0]['source_ink'][2] += 10
        manifest.write_text(json.dumps(data))
        rejects('source crop dimension mismatch refused')
        manifest.write_bytes(original_manifest)
        data = json.loads(original_manifest)
        data['groups'][0]['frames'][0]['fit_rect'][1] += 8
        manifest.write_text(json.dumps(data))
        rejects('stretched proportional fit refused')
        manifest.write_bytes(original_manifest)
        extra = output/'unexpected.bin'
        extra.write_bytes(b'unlisted')
        rejects('unlisted output refused')
        extra.unlink()
        image_path = output/frames[1]['file']
        original_png = image_path.read_bytes()
        image_path.write_bytes(original_png+b'modified')
        rejects('changed PNG bytes refused')
        image_path.write_bytes(original_png)
        overrides.write_text(json.dumps({'52': {'8': 'canonical-phase.png'}}))
        invalid = root/'invalid-output'
        result = subprocess.run(command+['--output', str(invalid)], stdout=subprocess.DEVNULL,
                                stderr=subprocess.PIPE, timeout=90)
        assert result.returncode != 0 and b'repeated phase' in result.stderr
        assert not invalid.exists()
        passed.append('conflicting repeat override refused before output')
        report = {'status': 'PASS', 'checks': passed,
                  'compiler_sha256': verifier.digest(Path(builder.__file__)),
                  'verifier_sha256': verifier.digest(Path(verifier.__file__)),
                  'test_script_sha256': verifier.digest(Path(__file__)),
                  'scope': 'candidate asset tooling only; no gameplay or original parity'}
        args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2)+'\n')
        print(json.dumps(report, ensure_ascii=False))


if __name__ == '__main__':
    main()
