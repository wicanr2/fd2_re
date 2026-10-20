#!/usr/bin/env python3
"""以有效GPU基底逐項拒收破損收據；不修改真實擷取。"""
import copy
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest

from PIL import Image
import verify_fd2_hd_death_gpu as verifier


class DeathGPUVerifierTests(unittest.TestCase):
    def test_actual_positive_and_rejected_witnesses(self):
        catalog=Path(os.environ['FD2_TEST_HD_DEATH_CATALOG'])
        pack=Path(os.environ['FD2_TEST_HD_DEATH_PACK'])
        run=Path(os.environ['FD2_TEST_HD_DEATH_RUN'])
        positive=verifier.validate(catalog,pack,run)
        self.assertEqual(positive['frame_count'],26)
        receipt=json.loads((run/'receipt.json').read_text())
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            for path in run.glob('*.png'): (root/path.name).symlink_to(path)
            for kind in ['frames','f2','origin','viewport','missing_overlay','layer_order','black_image','source_order','consumer']:
                with self.subTest(kind=kind):
                    data=copy.deepcopy(receipt)
                    source=json.loads(catalog.read_text())
                    if kind=='frames': data['frames'].pop()
                    elif kind=='consumer':data['consumer']='unknown'
                    elif kind=='f2': data['faithful_draws']=0
                    elif kind=='origin': data['targets'][0]['Origin']+=1
                    elif kind=='viewport': data['frames'][0]['viewport']['Min']['X']=24
                    elif kind=='missing_overlay':
                        data['frames'][13]['overlay_draws'].pop()
                        data['frames'][13]['overlays']-=1
                    elif kind=='layer_order': data['frames'][14]['overlay_draws'].reverse()
                    elif kind=='black_image':
                        path=root/'black.png'
                        Image.new('RGB',(1920,1200)).save(path)
                        data['frames'][0]['file']=path.name
                        data['frames'][0]['sha256']=hashlib.sha256(path.read_bytes()).hexdigest()
                    elif kind=='source_order':
                        bank=next(a for a in source['assets'] if a['role']=='map_overlay_hd_set')
                        bank['native_lmi_frames'][-1]['index']=68
                    candidate=root/'catalog.json'
                    candidate.write_text(json.dumps(source))
                    (root/'receipt.json').write_text(json.dumps(data))
                    with self.assertRaises(ValueError): verifier.validate(candidate,pack,root)


if __name__=='__main__': unittest.main()
