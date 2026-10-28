"""用明示唯讀原包與同源候選驗真實綁定反例；在Pillow Docker執行。"""
from copy import deepcopy
import json
import os
from pathlib import Path
import tempfile
import unittest

from build_fd2_hd_terrain_reuse import prepare, register


@unittest.skipUnless(os.environ.get('FD2_TEST_TERRAIN_REUSE_ROOT'), 'explicit original/candidate process required')
class TerrainReuseTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo = Path(__file__).resolve().parents[1]
        cls.work = Path(os.environ['FD2_TEST_TERRAIN_REUSE_ROOT'])
        cls.original, cls.pack = Path('/original'), Path('/modern')
        cls.catalog = Path(os.environ.get('FD2_TEST_TERRAIN_BASE_CATALOG', str(cls.repo / 'remake/assets/themes/modern/catalog.json')))
        cls.coverage = cls.work / 'hd1-terrain-coverage-before-r1.json'
        cls.sources = cls.work.parent / 'native-terrain-sources-r2'
        cls.baseline = json.loads(cls.coverage.read_text())

    def build(self, coverage, out):
        return prepare(self.catalog, self.pack, self.repo / 'remake/assets/maps', self.sources,
                       self.original, coverage, out)

    def test_complete_rebuild_preserves_all_base_entries_and_image_names(self):
        with tempfile.TemporaryDirectory(prefix='.reuse-test-', dir=self.work) as temp:
            out = Path(temp) / 'candidate'
            result = self.build(self.coverage, out)
            base = json.loads(self.catalog.read_text())
            candidate = json.loads((out / 'catalog.json').read_text())
            prior = json.loads((self.work / 'hd1-terrain-reuse-candidate-r1/catalog.json').read_text())
            self.assertEqual(base['assets'], candidate['assets'][:len(base['assets'])])
            self.assertEqual(prior['assets'], candidate['assets'])
            self.assertEqual((len(result['bindings']), result['new_catalog_sets'], result['new_image_files']), (46, 11, 0))
            self.assertTrue(all(p.is_symlink() for p in out.glob('*.png')))
            with self.assertRaises(ValueError):
                self.build(self.coverage, out)

    def test_wrong_binding_content_parent_selector_and_partial_scope_rejected(self):
        for fault in ('catalog', 'map-list', 'partial-count', 'content', 'parent', 'role', 'selector', 'phase', 'duplicate'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory(prefix='.reuse-reject-', dir=self.work) as temp:
                report = deepcopy(self.baseline)
                mp = next(m for m in report['maps'] if any(f.get('reuse') for f in m['required']))
                fr = next(f for f in mp['required'] if f.get('reuse'))
                if fault == 'catalog':
                    report['catalog_sha256'] = 'wrong'
                elif fault == 'map-list':
                    report['maps'].pop()
                elif fault == 'partial-count':
                    report['summary']['reusable_slots'] += 1
                elif fault == 'content':
                    fr['native_content_sha256'] = 'wrong'
                elif fault == 'parent':
                    fr['reuse']['frame'] += 1
                elif fault == 'role':
                    fr['role'] = 'foreground'
                elif fault == 'selector':
                    fr['selector']['tile_word'] += 1
                elif fault == 'phase':
                    fr['selector']['event_update_pass'] = 1025
                elif fault == 'duplicate':
                    mp['required'].append(deepcopy(fr))
                    report['summary']['reusable_slots'] += 1
                coverage, out = Path(temp) / 'coverage.json', Path(temp) / 'candidate'
                coverage.write_text(json.dumps(report))
                with self.assertRaises(ValueError):
                    self.build(coverage, out)
                self.assertFalse(out.exists(), 'rejection published partial candidate')

    def test_registration_requires_complete_actual_gpu_and_preserved_save(self):
        gpu_path = self.work / 'hd1-terrain-reuse-gpu-r3-capture/receipt.json'
        for fault in ('valid', 'catalog-sha', 'missing-case', 'wrong-source', 'missing-save'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory(prefix='.reuse-register-', dir=self.work) as temp:
                root = Path(temp)
                catalog = root / 'catalog.json'
                catalog.write_bytes(self.catalog.read_bytes())
                before = catalog.read_bytes()
                gpu = json.loads(gpu_path.read_text())
                for row in gpu['cases']:
                    if row.get('file'):
                        (root / row['file']).symlink_to(gpu_path.parent / row['file'])
                if fault == 'catalog-sha':
                    gpu['catalog_sha256'] = 'wrong'
                elif fault == 'missing-case':
                    gpu['cases'].pop(0)
                elif fault == 'wrong-source':
                    gpu['cases'][0]['source_frame'] += 1
                elif fault == 'missing-save':
                    gpu['save_bytes_unchanged'] = 0
                altered = root / 'gpu.json'
                altered.write_text(json.dumps(gpu))
                args = catalog, self.work / 'hd1-terrain-reuse-candidate-r1', altered
                if fault == 'valid':
                    result = register(*args)
                    self.assertEqual((result['registered_slots'], result['catalog_assets'], result['new_image_files']), (46, 864, 0))
                else:
                    with self.assertRaises(ValueError):
                        register(*args)
                    self.assertEqual(before, catalog.read_bytes())


if __name__ == '__main__':
    unittest.main()
