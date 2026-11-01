"""用實際候選核對圖集重編與錯誤來源拒收；只在Docker執行。"""
import copy
import json
import os
from pathlib import Path
import shutil
import tempfile
import unittest

from PIL import Image

import build_fd2_hd_terrain_atlas as atlas


class TerrainAtlasTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        value = os.environ.get('FD2_TERRAIN_ATLAS_TEST_ROOT')
        if not value:
            raise unittest.SkipTest('explicit real atlas work root required')
        cls.work = Path(value).resolve()
        cls.repo = Path(__file__).resolve().parents[1]
        cls.recipe = cls.repo / 'remake/assets/themes/modern/hd1-terrain-atlas-pilot16.recipe.json'
        cls.base = cls.work / 'base-catalog.json'
        cls.candidate = cls.work / 'candidate-r2'

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(dir=self.work)
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)

    def prepare(self, recipe, out, master=None):
        return atlas.prepare(recipe, self.base, Path('/modern'), Path('/original'),
                             self.repo / 'work/promo-cadence-20261018/hd-fonts/native-terrain-sources-r2',
                             self.repo / 'remake/assets/maps', master or self.work / 'terrain-pilot24-atlas-r1.png',
                             self.work / 'base-coverage.json', out)

    def test_rebuild_preserves_every_asset_and_selected_pixel(self):
        out = self.root / 'rebuild'
        result = self.prepare(self.recipe, out)
        original = atlas.read(self.candidate / 'manifest.json')
        self.assertEqual(result['frames'], original['frames'])
        self.assertEqual(result['bindings'], original['bindings'])
        self.assertEqual(atlas.read(out / 'catalog.json')['assets'], atlas.read(self.candidate / 'catalog.json')['assets'])
        self.assertEqual(result['new_image_files'], 17)
        for frame in result['frames']:
            for field in ('file', 'master_file'):
                if field not in frame:
                    continue
                self.assertEqual((out / frame[field]).read_bytes(), (self.candidate / frame[field]).read_bytes())
        base = atlas.read(self.base)['assets']
        self.assertEqual(atlas.read(out / 'catalog.json')['assets'][:len(base)], base)

    def test_wrong_sources_never_publish_a_candidate(self):
        faults = ['master-sha', 'catalog-sha', 'coverage-sha', 'manifest-sha', 'duplicate-cell',
                  'missing-cell', 'missing-target', 'wrong-selector', 'wrong-native', 'crop', 'path',
                  'existing-file', 'geometry', 'alpha', 'shade-index', 'material-sha', 'shade-classification']
        for fault in faults:
            with self.subTest(fault=fault):
                recipe = copy.deepcopy(atlas.read(self.recipe))
                master = None
                key = {'master-sha': 'master_sha256', 'catalog-sha': 'base_catalog_sha256',
                       'coverage-sha': 'coverage_sha256', 'manifest-sha': 'source_manifest_sha256'}.get(fault)
                if key:
                    recipe[key] = '0' * 64
                elif fault == 'duplicate-cell':
                    recipe['groups'][1]['cell'] = recipe['groups'][0]['cell']
                elif fault == 'missing-cell':
                    recipe['groups'].pop()
                elif fault == 'missing-target':
                    recipe['groups'][0]['targets'].pop()
                elif fault == 'wrong-selector':
                    recipe['groups'][0]['targets'][0]['selector']['tile_word'] ^= 1
                elif fault == 'wrong-native':
                    recipe['groups'][0]['native_content_sha256'] = '0' * 64
                elif fault == 'crop':
                    next(g for g in recipe['groups'] if g['compile_mode'] == 'atlas_cell')['crop'][0] = 0
                elif fault == 'path':
                    next(g for g in recipe['groups'] if g['compile_mode'] == 'atlas_cell')['master_file'] = '../outside.png'
                elif fault == 'shade-index':
                    recipe['shade_policy']['index_shades']['137'] = 255
                elif fault == 'material-sha':
                    recipe['shade_policy']['material']['sha256'] = '0' * 64
                elif fault == 'shade-classification':
                    recipe['shade_policy']['index_groups']['grass'].remove(137)
                elif fault == 'existing-file':
                    recipe['groups'][0]['file'] = next(a['file'] for a in atlas.read(self.base)['assets'] if 'file' in a)
                elif fault in ('geometry', 'alpha'):
                    master = self.root / (fault + '.png')
                    picture = Image.new('RGBA', (1, 1) if fault == 'geometry' else (1024, 1536), (20, 40, 60, 255))
                    if fault == 'alpha':
                        picture.putpixel((0, 0), (20, 40, 60, 254))
                    picture.save(master)
                    recipe['master_sha256'] = atlas.sha(master)
                path, out = self.root / (fault + '.json'), self.root / (fault + '-candidate')
                atlas.write(path, recipe)
                with self.assertRaises(ValueError):
                    self.prepare(path, out, master)
                self.assertFalse(out.exists())
                self.assertFalse(list(self.root.glob('.terrain-atlas-*')))

    def test_registration_requires_complete_real_gpu_evidence(self):
        value = os.environ.get('FD2_TERRAIN_ATLAS_GPU_RECEIPT')
        if not value:
            self.skipTest('real GPU receipt required after source compilation')
        source = Path(value)
        real = atlas.read(source)
        for row in real['cases']:
            if row.get('file'):
                (self.root / row['file']).symlink_to(source.parent / row['file'])
        base = self.root / 'catalog.json'
        for fault in ['status', 'catalog', 'manifest', 'missing-case', 'wrong-frame', 'no-f2', 'no-save', 'png']:
            with self.subTest(fault=fault):
                shutil.copyfile(self.base, base)
                before = base.read_bytes()
                gpu = copy.deepcopy(real)
                if fault == 'status':
                    gpu['status'] = 'guessed'
                elif fault == 'catalog':
                    gpu['catalog_sha256'] = '0' * 64
                elif fault == 'manifest':
                    gpu['manifest_sha256'] = '0' * 64
                elif fault == 'missing-case':
                    gpu['cases'].pop(0)
                elif fault == 'wrong-frame':
                    gpu['cases'][0]['source_frame'] += 1
                elif fault == 'no-f2':
                    next(r for r in gpu['cases'] if r['game_draw'])['f2'] = False
                elif fault == 'no-save':
                    gpu['save_bytes_unchanged'] = 0
                elif fault == 'png':
                    next(r for r in gpu['cases'] if r.get('file'))['sha256'] = '0' * 64
                path = self.root / (fault + '-gpu.json')
                atlas.write(path, gpu)
                with self.assertRaises(ValueError):
                    atlas.register(base, self.candidate, path)
                self.assertEqual(before, base.read_bytes())
        atlas.write(self.root / 'actual-gpu.json', real)
        shutil.copyfile(self.base, base)
        result = atlas.register(base, self.candidate, self.root / 'actual-gpu.json')
        self.assertEqual(result['bindings'], 174)
        self.assertEqual(result['catalog_assets'], 906)


if __name__ == '__main__':
    unittest.main()
