#!/usr/bin/env python3
"""在Docker依明示清單清理可重建暫存，保留指紋、原因與前後空間統計。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess


def fingerprint(path):
    s = path.lstat()
    return [s.st_dev, s.st_ino, s.st_size, s.st_mtime_ns]


def disk_usage(root):
    return int(subprocess.check_output(['du', '-sx', '-B1', str(root)], text=True).split()[0])


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--plan', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    parser.add_argument('--tracked-list', type=Path, required=True)
    parser.add_argument('--apply', action='store_true')
    args = parser.parse_args()
    assert Path('/.dockerenv').is_file() and os.getuid() != 0
    root = Path(__file__).resolve().parents[1]
    plan = json.loads(args.plan.read_text())
    tracked = set(args.tracked_list.read_bytes().decode().strip('\0').split('\0'))
    blocked = {str(parent) for name in tracked for parent in [Path(name), *Path(name).parents]}
    assert plan['authorization'] == 'user-requested-workspace-cleanup-20261105'
    assert args.report.parent.is_dir() and args.report.parent.stat().st_uid == os.getuid()
    protected = ['work/matching-copy-agent-20261008', 'work/matching-inventory-agent-20261008']
    for row in plan['keep']:
        path = root / row['path']
        assert fingerprint(path) == row['fingerprint'], ('protected file changed', path)
    for row in plan['remove']:
        rel = Path(row['path'])
        assert not rel.is_absolute() and '..' not in rel.parts
        assert rel.parts[0] == 'work' or (len(rel.parts) == 4 and rel.parts[:3] in [
            ('dist-all', 'v.1.1.0-20261103', 'full-local'),
            ('dist-all', 'v.1.1.1-20261104', 'full-local')])
        assert not any(str(rel) == p or str(rel).startswith(p + '/') for p in protected)
        assert str(rel) not in blocked
        path = root / rel
        assert not path.is_symlink() and path.lstat().st_uid == os.getuid(), path
        assert fingerprint(path) == row['fingerprint'], ('candidate changed', path)
        assert row['kind'] in ['file', 'tree']
        assert path.is_file() if row['kind'] == 'file' else path.is_dir()
        # A symlink in a parent must never route deletion outside this workspace.
        assert path.resolve() == path, path
    if not args.apply:
        print(json.dumps(dict(status='PLAN-VALID',candidates=len(plan['remove'])),ensure_ascii=False))
        return
    old = json.loads(args.report.read_text()) if args.report.exists() else dict(schema_version=1, batches=[])
    batch = dict(plan=str(args.plan.relative_to(root)) if args.plan.is_absolute() else str(args.plan),
                 plan_sha256=hashlib.sha256(args.plan.read_bytes()).hexdigest(),
                 tool_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                 before_bytes=disk_usage(root), candidates=len(plan['remove']), removed_count=0, removed=[])
    old['batches'].append(batch)
    def save():
        args.report.write_text(json.dumps(old, ensure_ascii=False, indent=2) + '\n')
    save()
    compact = len(plan['remove']) > 100
    for index, row in enumerate(plan['remove'], 1):
        path = root / row['path']
        if row['kind'] == 'tree':
            shutil.rmtree(path)
        else:
            path.unlink()
        if not compact or row.get('sha256'):
            batch['removed'].append(row)
        batch['removed_count'] = index
        if not compact or index % 1000 == 0:
            save()
            print('removed', f'{index}/{len(plan["remove"])}' if compact else row['path'], flush=True)
    for row in plan['keep']:
        assert fingerprint(root / row['path']) == row['fingerprint']
    batch.update(status='PASS', after_bytes=disk_usage(root))
    batch['reclaimed_bytes'] = batch['before_bytes'] - batch['after_bytes']
    old['reclaimed_bytes'] = sum(b.get('reclaimed_bytes', 0) for b in old['batches'])
    save()
    print(json.dumps(dict(status='PASS',reclaimed_bytes=batch['reclaimed_bytes']),ensure_ascii=False))


if __name__ == '__main__':
    main()
