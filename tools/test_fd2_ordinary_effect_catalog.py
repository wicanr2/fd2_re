#!/usr/bin/env python3
"""以現有召喚清冊建立可丟棄格式案例，驗普通效果准入與拒收，不登錄美術。"""
import argparse
import copy
import json
import os
from pathlib import Path
import tempfile

from build_fd2_gaia_battle_hd import digest
from validate_modern_theme_catalog import validate


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--catalog', type=Path, required=True)
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    if (not Path('/.dockerenv').exists() or args.report.exists()
            or args.report.parent.stat().st_uid != os.getuid()):
        raise SystemExit('在Docker使用本人擁有的新報告路徑')
    catalog = validate(False, args.catalog)
    checks = []
    with tempfile.TemporaryDirectory(prefix='fd2-effect-catalog-') as directory:
        path = Path(directory)/'catalog.json'
        def check(resource, consumer, cycle, accepted):
            candidate = copy.deepcopy(catalog)
            entry = next(a for a in candidate['assets'] if a['role']=='battle_effect_hd_set' and a['source_resource']==65)
            entry.update(source_resource=resource, consumer_contract=consumer, cycle_policy=cycle)
            path.write_text(json.dumps(candidate))
            try:
                validate(False, path)
            except ValueError as e:
                if accepted or 'invalid HD battle resource' not in str(e):
                    raise AssertionError((resource,consumer,str(e))) from e
            else:
                if not accepted:
                    raise AssertionError(('out-of-scope contract admitted',resource,consumer))
            checks.append({'resource':resource,'consumer':consumer,'accepted':accepted})
        for resource in [18,19,20,21,22,23,24,25,26,27,28,30,32,33,37,38,39,43]:
            check(resource,'native_fdother_ordinary_rgba_hd6_v1','native_ordinary_frames',True)
            check(resource,'native_fdother_compound_rgba_hd6_v1','native_compound_frames',False)
        for resource in [-1,0,5,17,29,31,34,36,40,42,44,64,69,409,None,[],{},True]:
            check(resource,'native_fdother_ordinary_rgba_hd6_v1','native_ordinary_frames',False)
    report={'status':'PASS-ORDINARY-EFFECT-CATALOG','base_catalog_sha256':digest(args.catalog),
            'checks':checks,'scope':'只驗清冊格式與契約；原header／描述子由Go loader驗證，不新增素材或宣稱美術完成。'}
    args.report.write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'status':report['status'],'cases':len(checks)}))


if __name__ == '__main__':
    main()
