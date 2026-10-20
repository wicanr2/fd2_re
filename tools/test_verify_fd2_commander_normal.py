#!/usr/bin/env python3
"""以實檔正常隊長收據核對錯圖、錯speaker及未完成路徑均拒收。"""
import argparse
import copy
import json
from pathlib import Path
import tempfile

from build_fd2_gaia_battle_hd import digest
from verify_fd2_commander_normal import verify


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ['receipt', 'catalog', 'source', 'report']:
        parser.add_argument('--'+name, type=Path, required=True)
    args = parser.parse_args()
    assert Path('/.dockerenv').exists() and not args.report.exists()
    original = json.loads(args.receipt.read_text())
    baseline = verify(args.receipt, args.catalog, args.source)
    checks = []
    def check(name, mutate):
        doc = copy.deepcopy(original)
        mutate(doc)
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', dir=args.receipt.parent) as out:
            out.write(json.dumps(doc));out.flush()
            try:
                verify(Path(out.name), args.catalog, args.source)
            except ValueError as error:
                checks.append(dict(case=name, reason=str(error)))
            else:
                raise AssertionError('錯收：'+name)
    check('not-pass', lambda j:j.update(status='OBSERVATION-ONLY'))
    check('state-injection', lambda j:j.update(state_injection=True))
    check('wrong-catalog', lambda j:j.update(catalog_sha256='0'*64))
    check('wrong-save', lambda j:j.update(saved_sha256='0'*64))
    check('missing-idle', lambda j:j.update(commander_samples=[]))
    check('wrong-idle-png', lambda j:j['commander_samples'][0].update(sha256='0'*64))
    check('missing-portrait', lambda j:j.update(portrait_samples=[]))
    check('wrong-speaker', lambda j:j['portrait_samples'][0]['before'].update(dialogue_speaker=111))
    check('wrong-rectangle', lambda j:j['portrait_samples'][0]['after'].update(hd_overlay_rect=[0,0,160,160]))
    check('missing-cold-load', lambda j:j.update(operations=[o for o in j['operations'] if o.get('process_epoch')!=1]))
    report=dict(status='PASS-COMMANDER-NORMAL-GUARDS',guards=len(checks),checks=checks,
                receipt_sha256=digest(args.receipt),verifier_sha256=baseline['verifier_sha256'])
    args.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(dict(status=report['status'],guards=report['guards'])))


if __name__=='__main__':main()
