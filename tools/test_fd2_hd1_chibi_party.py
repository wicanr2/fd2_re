#!/usr/bin/env python3
"""已核准Q版圖源的可重編與原子拒收檢查。"""
from copy import deepcopy
import json
import os
from pathlib import Path
import tempfile
import unittest

from refresh_fd2_hd1_map_sprites import prepare,read,sha,asset_digest

ROOT=Path(__file__).resolve().parents[1]
@unittest.skipUnless(os.environ.get('FD2_CHIBI_PARTY_ROOT'),'explicit accepted inputs required')
class ChibiPartyTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.work=Path(os.environ['FD2_CHIBI_PARTY_ROOT'])
        cls.recipe=Path(os.environ.get('FD2_CHIBI_PARTY_RECIPE',ROOT/'remake/assets/themes/modern/hd1-party-chibi-refresh.recipe.json'))
        cls.catalog=cls.work/'base-catalog.json'
        cls.candidate=Path(os.environ.get('FD2_CHIBI_PARTY_CANDIDATE',cls.work/'candidate'))
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

    def make(self,recipe,out,masters=None):
        return prepare(recipe,self.catalog,self.pack,Path('/original'),self.work if masters is None else masters,out)

    def test_rebuild_preserves_other_assets_and_exact_24_hd_pngs(self):
        with tempfile.TemporaryDirectory(dir=self.work) as folder:
            out=Path(folder)/'rebuilt';m=self.make(self.recipe,out)
            expected=read(self.candidate/'manifest.json')
            self.assertEqual(m['groups'],expected['groups'])
            self.assertEqual(read(out/'catalog.json')['assets'],read(self.candidate/'catalog.json')['assets'])
            self.assertEqual(m['other_assets_unchanged'],len(read(self.catalog)['assets'])-2*len(read(self.recipe)['groups']))
            for group in m['groups']:
                for role in ('parent','hd'):
                    for name in group[role]['files']:
                        self.assertEqual((out/name).read_bytes(),(self.candidate/name).read_bytes())
                self.assertEqual(sha(out/group['master_file']),group['master_sha256'])

    def test_source_faults_fail_before_publish(self):
        original=read(self.recipe)
        faults=['catalog','master','palette','native-bank','design','duplicate-group','invisible-group','unsafe-name']
        if any('reuse_group' in entry for entry in original['groups']):faults+=['reuse-group','reuse-assets','reuse-missing-group']
        for fault in faults:
            with self.subTest(fault=fault),tempfile.TemporaryDirectory(dir=self.work) as folder:
                d=deepcopy(original);folder=Path(folder);out=folder/'candidate';recipe=folder/'recipe.json';masters=self.work
                if fault=='catalog':d['base_catalog_sha256']='0'*64
                elif fault=='master':d['groups'][0]['master_sha256']='0'*64
                elif fault=='palette':d['palette_sha256']='0'*64
                elif fault=='native-bank':d['native_bank_sha256']='0'*64
                elif fault=='design':d['approved_preview_sha256']='0'*64
                elif fault=='duplicate-group':d['groups'].append(deepcopy(d['groups'][0]))
                elif fault=='invisible-group':d['groups'][0]['group']=39
                elif fault=='reuse-group':
                    entry=next(g for g in d['groups'] if 'reuse_group' in g)
                    reference=14 if entry['reuse_group']!=14 else 17
                    assets=[next(a for a in read(self.catalog)['assets'] if a.get('source_group')==reference and a['role']==role) for role in ('map_sprite_set','map_sprite_hd_set')]
                    entry.update(reuse_group=reference,reuse_assets_sha256=asset_digest(assets),master_file=assets[0]['master_file'],master_sha256=sha(Path('/modern')/assets[0]['master_file']))
                    masters=folder/'masters';masters.mkdir()
                    for g in d['groups']:
                        name=g['master_file'];source=Path('/modern')/name if g is entry else self.work/name
                        if not (masters/name).exists():(masters/name).symlink_to(source)
                elif fault=='reuse-assets':next(g for g in d['groups'] if 'reuse_group' in g)['reuse_assets_sha256']='0'*64
                elif fault=='reuse-missing-group':next(g for g in d['groups'] if 'reuse_group' in g).pop('reuse_group')
                else:d['groups'][0]['master_file']='../master.png'
                recipe.write_text(json.dumps(d))
                if fault=='reuse-group':
                    with self.assertRaisesRegex(ValueError,'reused native layers differ'):self.make(recipe,out,masters)
                else:
                    with self.assertRaises((ValueError,OSError)):self.make(recipe,out)
                self.assertFalse(out.exists())
                self.assertFalse(list(folder.glob('.chibi-*')))

    def test_existing_output_is_preserved(self):
        before=sha(self.candidate/'catalog.json')
        with self.assertRaises(ValueError):self.make(self.recipe,self.candidate)
        self.assertEqual(before,sha(self.candidate/'catalog.json'))

if __name__=='__main__':unittest.main()
