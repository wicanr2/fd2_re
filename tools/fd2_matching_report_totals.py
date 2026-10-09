#!/usr/bin/env python3
"""從實際雙輪匹配收據重算候選、可連結與布局拒收總數。只在Docker執行。"""
import argparse
import json
from pathlib import Path


def report_totals(repo, manifest):
    paths = manifest['local_outputs']['game_link_receipts']
    if len(paths) % 2:
        raise ValueError('雙輪收據數量不是偶數')
    trials = linked = rejected = 0
    for first, second in zip(paths[::2], paths[1::2]):
        a, b = repo / first, repo / second
        if a.read_bytes() != b.read_bytes():
            raise ValueError('雙輪完整收據不同：' + first)
        report = json.loads(a.read_text(encoding='utf-8'))
        for trial in report['trials']:
            trials += 1
            if trial.get('linkable', True):
                if 'code_sha256' not in trial:
                    raise ValueError('可連結候選缺少產碼雜湊')
                linked += 1
            else:
                if trial.get('comparison_performed') is not False or trial.get('exact_interval_bytes') or not trial.get('layout_rejection'):
                    raise ValueError('布局拒收候選的比較狀態矛盾')
                rejected += 1
    return {'compiled_game_candidates': trials, 'independent_objcopy_checks': linked,
            'source_groups': len(paths) // 2, 'unlinked_layout_candidates': rejected}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--repo', type=Path, default=Path('/repo'))
    parser.add_argument('--manifest', type=Path, default=Path('docs/data/ida/fd2_matching_full_20261008.json'))
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    if not Path('/.dockerenv').exists():
        raise SystemExit('本工具只在Docker執行')
    path = args.manifest if args.manifest.is_absolute() else args.repo / args.manifest
    manifest = json.loads(path.read_text(encoding='utf-8'))
    totals = report_totals(args.repo, manifest)
    mismatches = {key: {'recorded': manifest['matching_summary'].get(key), 'actual': value}
                  for key, value in totals.items() if manifest['matching_summary'].get(key) != value}
    print(json.dumps({'totals': totals, 'mismatches': mismatches}, ensure_ascii=False))
    if args.check and mismatches:
        raise SystemExit(1)


if __name__ == '__main__':
    main()
