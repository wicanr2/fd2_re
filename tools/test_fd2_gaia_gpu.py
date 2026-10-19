#!/usr/bin/env python3
"""用真實蓋亞收據拒收錯範圍、錯角色與錯圖源；不修改來源。Docker內執行。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ('receipt', 'catalog', 'source', 'report'):
        parser.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and not args.report.exists()
    original = digest(args.receipt)
    receipt = json.loads(args.receipt.read_text())
    catalog = json.loads(args.catalog.read_text())
    assert receipt['status'] == 'PASS' and receipt['schema'] == 'fd2-hd-gaia-focus-gui-v1'
    results = []
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        for name in receipt['screenshots']:
            (root/name).symlink_to((args.receipt.parent/name).resolve())
        for case in ('wrong-scope', 'wrong-group', 'wrong-source'):
            current, cat = copy.deepcopy(receipt), copy.deepcopy(catalog)
            group = 30
            if case == 'wrong-scope':
                current['schema'] = 'unrelated-receipt'
            elif case == 'wrong-group':
                group = 0
            else:
                # Keep a valid catalog digest and genuine source PNG, but
                # substitute Sol for Gaia. Broad alpha bounds must reject it.
                hd = {a['source_group']: a for a in cat['assets'] if a['role'] == 'map_sprite_hd_set'}
                hd[30]['files'] = hd[0]['files'][:]
                hd[30]['frame_sha256'] = hd[0]['frame_sha256'][:]
            rp, cp, output = root/'receipt.json', root/'catalog.json', root/'report.json'
            rp.write_text(json.dumps(current))
            cp.write_text(json.dumps(cat))
            command = [sys.executable, str(Path(__file__).with_name('verify_fd2_hd_world_images.py')),
                       '--receipt', str(rp), '--catalog', str(cp), '--source', str(args.source),
                       '--group', str(group), '--gaia-focus', '--report', str(output)]
            run = subprocess.run(command, capture_output=True, text=True, timeout=30)
            assert run.returncode != 0 and 'AssertionError' in run.stderr and not output.exists(), case
            if case == 'wrong-source':
                assert 'different_pixels' in run.stderr, run.stderr
            results.append({'case': case, 'rejected': True, 'report_published': False})
    assert digest(args.receipt) == original
    args.report.write_text(json.dumps({'status': 'PASS', 'cases': results,
        'source_receipt_unchanged': True, 'receipt_sha256': original,
        'verifier_sha256': digest(Path(__file__).with_name('verify_fd2_hd_world_images.py'))}, indent=2)+'\n')
    print('PASS: three real-receipt rejection cases; source unchanged')


if __name__ == '__main__':
    main()
