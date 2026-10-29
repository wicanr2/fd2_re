#!/usr/bin/env python3
"""已核准Q版圖源的可重編與原子拒收檢查。"""
from copy import deepcopy
import json
import os
from pathlib import Path
import tempfile
import unittest

from refresh_fd2_hd1_map_sprites import prepare,read,sha

ROOT=Path(__file__).resolve().parents[1]
@unittest.skipUnless(os.environ.get('FD2_CHIBI_PARTY_ROOT'),'explicit accepted inputs required')
class ChibiPartyTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.work=Path(os.environ['FD2_CHIBI_PARTY_ROOT'])
        cls.recipe=ROOT/'remake/assets/themes/modern/hd1-party-chibi-refresh.recipe.json'
        cls.catalog=cls.work/'base-catalog.json'
        cls.candidate=cls.work/'candidate'
        cls.base_view=tempfile.TemporaryDirectory(dir=cls.work)
        cls.addClassCleanup(cls.base_view.cleanup)
        cls.pack=Path(cls.base_view.name)/'base-pack';cls.pack.mkdir()
        names=set()
        for asset in read(cls.catalog)['assets']:
            names.update(asset.get('files',[]));names.update(asset.get('source_files',[]))
            for key in ('file','master_file'):
                if key in asset:names.add(asset[key])
        for name in names:
            source=Path('/modern')/name
            if not source.is_file():raise ValueError('historical registered source missing: '+name)
            (cls.pack/name).symlink_to(source)
        (cls.pack/'catalog.json').write_bytes(cls.catalog.read_bytes())

    def make(self,recipe,out):
        return prepare(recipe,self.catalog,self.pack,Path('/original'),self.work,out)

    def test_rebuild_preserves_other_assets_and_exact_24_hd_pngs(self):
        with tempfile.TemporaryDirectory(dir=self.work) as folder:
            out=Path(folder)/'rebuilt';m=self.make(self.recipe,out)
            expected=read(self.candidate/'manifest.json')
            self.assertEqual(m['groups'],expected['groups'])
            self.assertEqual(read(out/'catalog.json')['assets'],read(self.candidate/'catalog.json')['assets'])
            self.assertEqual(m['other_assets_unchanged'],877)
            for group in m['groups']:
                for role in ('parent','hd'):
                    for name in group[role]['files']:
                        self.assertEqual((out/name).read_bytes(),(self.candidate/name).read_bytes())
                self.assertEqual(sha(out/group['master_file']),group['master_sha256'])

    def test_source_faults_fail_before_publish(self):
        original=read(self.recipe)
        for fault in ('catalog','master','palette','native-bank','design','duplicate-group','invisible-group','unsafe-name'):
            with self.subTest(fault=fault),tempfile.TemporaryDirectory(dir=self.work) as folder:
                d=deepcopy(original);folder=Path(folder);out=folder/'candidate';recipe=folder/'recipe.json'
                if fault=='catalog':d['base_catalog_sha256']='0'*64
                elif fault=='master':d['groups'][0]['master_sha256']='0'*64
                elif fault=='palette':d['palette_sha256']='0'*64
                elif fault=='native-bank':d['native_bank_sha256']='0'*64
                elif fault=='design':d['approved_preview_sha256']='0'*64
                elif fault=='duplicate-group':d['groups'].append(deepcopy(d['groups'][0]))
                elif fault=='invisible-group':d['groups'][0]['group']=39
                else:d['groups'][0]['master_file']='../master.png'
                recipe.write_text(json.dumps(d))
                with self.assertRaises((ValueError,OSError)):self.make(recipe,out)
                self.assertFalse(out.exists())
                self.assertFalse(list(folder.glob('.chibi-*')))

    def test_existing_output_is_preserved(self):
        before=sha(self.candidate/'catalog.json')
        with self.assertRaises(ValueError):self.make(self.recipe,self.candidate)
        self.assertEqual(before,sha(self.candidate/'catalog.json'))

if __name__=='__main__':unittest.main()
