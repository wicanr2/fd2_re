#!/usr/bin/env python3
"""LMI死亡效果編製器的真實來源及預寫入拒收。"""
import copy
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest

from PIL import Image
import build_fd2_lmi_death_hd as compiler


class LMIDeathHDTests(unittest.TestCase):
    def test_actual_sources_and_atomic_rejection(self):
        original=Path(os.environ['FD2_TEST_ORIGINAL_PACK'])
        source=Path(os.environ['FD2_TEST_LMI_RECIPE'])
        recipe=json.loads(source.read_text())
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            for item in recipe['sources'].values():
                (root/item['file']).write_bytes((source.parent/item['file']).read_bytes())
            path=root/'recipe.json'
            path.write_text(json.dumps(recipe))
            asset=compiler.build(original,path,root/'positive')
            self.assertEqual([d['index'] for d in asset['native_lmi_frames']],list(range(68,80)))
            self.assertEqual(len(asset['files']),12)
            for name in ['bank','source_hash','source_alpha','missing','opacity','unknown_source','extra_field','unused_source']:
                with self.subTest(name=name):
                    bad=copy.deepcopy(recipe)
                    if name=='bank': bad['bank_sha256']='0'*64
                    elif name=='source_hash': bad['sources']['puff']['sha256']='0'*64
                    elif name=='missing': bad['entries'].pop('79')
                    elif name=='opacity': bad['entries']['79']['opacity']=0
                    elif name=='unknown_source': bad['entries']['79']['source']='unknown'
                    elif name=='extra_field': bad['entries']['79']['hidden']='x'
                    elif name in {'source_alpha','unused_source'}:
                        p=root/'invalid.png'
                        Image.new('RGBA',(32,32),(255,255,255,255)).save(p)
                        bad['sources']['invalid']={'file':p.name,'sha256':hashlib.sha256(p.read_bytes()).hexdigest()}
                        if name=='source_alpha': bad['entries']['79']['source']='invalid'
                        else:
                            p.write_bytes((source.parent/recipe['sources']['puff']['file']).read_bytes())
                            bad['sources']['invalid']['sha256']=hashlib.sha256(p.read_bytes()).hexdigest()
                    path.write_text(json.dumps(bad))
                    out=root/name
                    with self.assertRaises((ValueError,FileNotFoundError)): compiler.build(original,path,out)
                    self.assertFalse(out.exists())


if __name__=='__main__': unittest.main()
