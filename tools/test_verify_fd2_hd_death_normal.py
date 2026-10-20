#!/usr/bin/env python3
"""從真實正常GUI基底核對錯圖、缺相位與錯定位均拒收。"""
import copy
import json
import os
from pathlib import Path
import tempfile
import unittest

from PIL import Image
import verify_fd2_hd_death_normal as verifier


class NormalDeathVerifierTests(unittest.TestCase):
    def test_real_positive_and_rejections(self):
        catalog=Path(os.environ['FD2_TEST_HD_DEATH_CATALOG'])
        pack=Path(os.environ['FD2_TEST_HD_DEATH_PACK'])
        run=Path(os.environ['FD2_TEST_HD_DEATH_RUN'])
        self.assertEqual(verifier.validate(catalog,pack,run)['status'],'PASS-NORMAL-DEATH-OPAQUE-CORES')
        receipt=json.loads((run/'receipt.json').read_text())
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            for path in run.glob('*.png'):(root/path.name).symlink_to(path)
            black=root/'black.png';Image.new('RGB',(1920,1200)).save(black)
            for kind in ('black','sha','missing_b','origin','viewport','source_order','injection','continuation','kind'):
                with self.subTest(kind=kind):
                    data=copy.deepcopy(receipt);source=json.loads(catalog.read_text())
                    if kind=='injection':data['state_injection']=True
                    elif kind=='continuation':data['final_node']='battle_ch23'
                    elif kind=='source_order':
                        next(a for a in source['assets'] if a['role']=='map_overlay_hd_set')['native_lmi_frames'].reverse()
                    elif kind=='missing_b':
                        data['samples']=[s for s in data['samples'] if all(v.get('unit_death_presented_frame',99)<19 for v in (s['before'],s['after']))]
                    elif kind=='kind':
                        for sample in data['samples']:
                            for state in (sample['before'],sample['after']):
                                state['unit_death_kind']='action' if data.get('death_kind','script')=='script' else 'script'
                    else:
                        for sample in data['samples']:
                            if kind=='black':sample.update(file=black.name,sha256=verifier.digest(black))
                            elif kind=='sha':sample['sha256']='0'*64
                            else:
                                for state in (sample['before'],sample['after']):
                                    if not 13<=state.get('unit_death_presented_frame',-1)<21:continue
                                    if kind=='origin':state['unit_death_targets'][0]['Origin']+=1
                                    else:state['unit_death_viewport'][0]=24
                    candidate=root/'catalog.json';candidate.write_text(json.dumps(source))
                    (root/'receipt.json').write_text(json.dumps(data))
                    with self.assertRaises(ValueError):verifier.validate(candidate,pack,root)


if __name__=='__main__':unittest.main()
