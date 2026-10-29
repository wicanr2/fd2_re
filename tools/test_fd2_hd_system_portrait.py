"""驗DATO75真實來源、重編與完整GPU登錄守衛。"""
from copy import deepcopy
import json
import os
from pathlib import Path
import tempfile
import unittest

from build_fd2_hd_system_portrait import prepare, register, sha


@unittest.skipUnless(os.environ.get('FD2_SYSTEM_PORTRAIT_ROOT'),'explicit portrait inputs required')
class SystemPortraitTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.root=Path(__file__).resolve().parents[1]
        cls.work=Path(os.environ['FD2_SYSTEM_PORTRAIT_ROOT'])
        cls.recipe=cls.root/'remake/assets/themes/modern/system-confirmation-portrait.recipe.json'
        cls.base=cls.work/'hd1-system-base-catalog-r1.json'
        cls.master=cls.work/'dato-075-system-portrait-handpainted-v1-master.png'
        cls.candidate=cls.work/'hd1-system-candidate-r1'

    def test_complete_rebuild_preserves_old_assets_and_pngs(self):
        with tempfile.TemporaryDirectory(dir=self.work) as temp:
            out=Path(temp)/'candidate'
            result=prepare(self.recipe,self.base,Path('/modern'),Path('/original'),self.master,out)
            self.assertEqual(json.loads((out/'catalog.json').read_text())['assets'],json.loads((self.candidate/'catalog.json').read_text())['assets'])
            for row in result['outputs']:
                self.assertEqual((out/row['file']).read_bytes(),(self.candidate/row['file']).read_bytes())
            with self.assertRaises(ValueError):prepare(self.recipe,self.base,Path('/modern'),Path('/original'),self.master,out)

    def test_bad_sources_publish_nothing(self):
        for fault in ('master_sha256','base_catalog_sha256','native_png_sha256','speaker_id'):
            with self.subTest(fault=fault),tempfile.TemporaryDirectory(dir=self.work) as temp:
                root=Path(temp); recipe=json.loads(self.recipe.read_text());recipe[fault]=0 if fault=='speaker_id' else 'wrong'
                path=root/'recipe.json';path.write_text(json.dumps(recipe));out=root/'candidate'
                with self.assertRaises(ValueError):prepare(path,self.base,Path('/modern'),Path('/original'),self.master,out)
                self.assertFalse(out.exists())

    def test_complete_gpu_receipt_required(self):
        source=Path(os.environ['FD2_SYSTEM_PORTRAIT_GPU'])
        for fault in ('valid','wrong-catalog','wrong-manifest','missing-caller','missing-modes','missing-save','missing-rows','wrong-png'):
            with self.subTest(fault=fault),tempfile.TemporaryDirectory(dir=self.work) as temp:
                root=Path(temp);catalog=root/'catalog.json';catalog.write_bytes(self.base.read_bytes());before=catalog.read_bytes();gpu=json.loads(source.read_text())
                for row in gpu['rows']:
                    if row.get('file') and not (root/row['file']).exists():(root/row['file']).symlink_to(source.parent/row['file'])
                if fault=='wrong-catalog':gpu['catalog_sha256']='wrong'
                elif fault=='wrong-manifest':gpu['portrait_manifest_sha256']='wrong'
                elif fault=='missing-caller':gpu['caller_count']=4
                elif fault=='missing-modes':gpu['display_modes']=3
                elif fault=='missing-save':gpu['save_bytes_unchanged']=0
                elif fault=='missing-rows':gpu['rows']=gpu['rows'][:99]
                elif fault=='wrong-png':next(r for r in gpu['rows'] if r.get('file'))['sha256']='wrong'
                path=root/'gpu.json';path.write_text(json.dumps(gpu))
                if fault=='valid':
                    result=register(catalog,self.candidate,path);self.assertEqual(result['status'],'REGISTERED-SYSTEM-CONFIRMATION-PORTRAIT')
                    self.assertEqual(len(json.loads(catalog.read_text())['assets']),883)
                else:
                    with self.assertRaises(ValueError):register(catalog,self.candidate,path)
                    self.assertEqual(catalog.read_bytes(),before)


if __name__=='__main__':unittest.main()
