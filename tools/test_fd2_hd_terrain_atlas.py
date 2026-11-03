"""用實際候選核對圖集重編與錯誤來源拒收；只在Docker執行。"""
import copy
import hashlib
import json
import os
from pathlib import Path
import shutil
import tempfile
import unittest

from PIL import Image

import build_fd2_hd_terrain_atlas as atlas
from build_fd2_hd_terrain_reuse import native


class TerrainAtlasTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        value = os.environ.get('FD2_TERRAIN_ATLAS_TEST_ROOT')
        if not value:
            raise unittest.SkipTest('explicit real atlas work root required')
        cls.work = Path(value).resolve()
        cls.repo = Path(__file__).resolve().parents[1]
        cls.recipe = Path(os.environ.get('FD2_TERRAIN_ATLAS_TEST_RECIPE', str(cls.repo / 'remake/assets/themes/modern/hd1-terrain-atlas-pilot16.recipe.json')))
        cls.master = Path(os.environ.get('FD2_TERRAIN_ATLAS_TEST_MASTER', str(cls.work / 'terrain-pilot24-atlas-r1.png')))
        cls.base = cls.work / 'base-catalog.json'
        cls.candidate = cls.work / 'candidate-r2'

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(dir=self.work)
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)

    def prepare(self, recipe, out, master=None):
        return atlas.prepare(recipe, self.base, Path('/modern'), Path('/original'),
                             self.repo / 'work/promo-cadence-20261018/hd-fonts/native-terrain-sources-r2',
                             self.repo / 'remake/assets/maps', master or self.master,
                             self.work / 'base-coverage.json', out)

    def test_rebuild_preserves_every_asset_and_selected_pixel(self):
        out = self.root / 'rebuild'
        result = self.prepare(self.recipe, out)
        original = atlas.read(self.candidate / 'manifest.json')
        self.assertEqual(result['frames'], original['frames'])
        self.assertEqual(result['bindings'], original['bindings'])
        self.assertEqual(atlas.read(out / 'catalog.json')['assets'], atlas.read(self.candidate / 'catalog.json')['assets'])
        recipe = atlas.read(self.recipe)
        self.assertEqual(result['new_image_files'], recipe['expected_cells'] + recipe['expected_crop_masters'])
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
        if any('edge_pad_master_pixels' in g for g in atlas.read(self.recipe)['groups']):
            faults += ['edge-pad-negative', 'edge-pad-odd', 'edge-pad-type']
        if atlas.read(self.recipe)['grid'] == [8, 12]:
            faults += ['dense-grid-shape', 'dense-edge-pad-two', 'dense-nondirect-mode']
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
                    group = next((g for g in recipe['groups'] if g['compile_mode'] == 'atlas_cell'), recipe['groups'][0])
                    group['crop'] = [0, 0, 1, 1]
                elif fault == 'path':
                    group = next((g for g in recipe['groups'] if g['compile_mode'] == 'atlas_cell'), recipe['groups'][0])
                    group['master_file'] = '../outside.png'
                elif fault == 'shade-index':
                    recipe['shade_policy']['index_shades']['137'] = 255
                elif fault == 'material-sha':
                    recipe['shade_policy']['material']['sha256'] = '0' * 64
                elif fault == 'shade-classification':
                    recipe['shade_policy']['index_groups']['grass'].remove(137)
                elif fault == 'dense-grid-shape':
                    recipe['grid'] = [8, 6]
                elif fault == 'dense-edge-pad-two':
                    recipe['groups'][0]['edge_pad_master_pixels'] = 2
                elif fault == 'dense-nondirect-mode':
                    recipe['groups'][0]['compile_mode'] = 'atlas_cell'
                elif fault.startswith('edge-pad-'):
                    recipe['groups'][0]['edge_pad_master_pixels'] = {
                        'edge-pad-negative': -2, 'edge-pad-odd': 1, 'edge-pad-type': True}[fault]
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

    def test_masked_objects_keep_native_boundary_and_painted_rgb(self):
        recipe = atlas.read(self.recipe)
        groups = [g for g in recipe['groups'] if g['compile_mode'] == 'atlas_masked_object']
        if not groups:
            self.skipTest('this recipe has no masked object source')
        policy, grass, _ = atlas.grass_policy(recipe, atlas.read(self.base), Path('/modern'))
        source = atlas.read(self.repo / 'work/promo-cadence-20261018/hd-fonts/native-terrain-sources-r2/manifest.json')
        indices = set(policy['index_groups']['grass'])
        checked = 0
        for group in groups:
            raw = native(Path('/original'), source, *group['anchor'])[:576]
            master = Image.open(self.candidate / group['master_file']).convert('RGBA')
            painting = master.resize((144, 144), Image.Resampling.LANCZOS)
            actual = Image.open(self.candidate / group['file']).convert('RGBA')
            self.assertEqual(actual.getextrema()[3], (255, 255))
            for offset in range(144):
                for x, y in ((offset, 0), (offset, 143), (0, offset), (143, offset)):
                    index = raw[y // 6 * 24 + x // 6]
                    if index in indices:
                        value = policy['index_shades'][str(index)]
                        expected = tuple(round(v * value / 255) for v in grass.getpixel((x, y))[:3]) + (255,)
                    else:
                        expected = painting.getpixel((x, y))
                    self.assertEqual(actual.getpixel((x, y)), expected, (group['cell'], x, y))
                    checked += 1
        self.assertEqual(checked, len(groups) * 576)
        # A broad isolated object has an unmixed painted center, while the
        # surrounding grass stays at its source material and original opacity.
        raw = bytes(32 if 5 <= x < 19 and 5 <= y < 19 else 68 for y in range(24) for x in range(24))
        actual, _, _ = atlas.masked_object_tile(raw, painting, grass, policy)
        self.assertEqual(actual.getpixel((72, 72)), painting.getpixel((72, 72)))
        self.assertEqual(actual.getpixel((0, 0)), grass.getpixel((0, 0)))

    def test_masked_object_wrong_partition_never_publishes(self):
        original = atlas.read(self.recipe)
        group = next((g for g in original['groups'] if g['compile_mode'] == 'atlas_masked_object'), None)
        if group is None:
            self.skipTest('this recipe has no masked object source')
        for fault in ('indices', 'count'):
            with self.subTest(fault=fault):
                recipe = copy.deepcopy(original)
                changed = next(g for g in recipe['groups'] if g['cell'] == group['cell'])
                if fault == 'indices':
                    changed['object_indices'].append(next(v for v in range(256) if v not in changed['object_indices']))
                else:
                    changed['object_source_pixels'] += 1
                path, out = self.root / (fault + '.json'), self.root / (fault + '-candidate')
                atlas.write(path, recipe)
                with self.assertRaises(ValueError):
                    self.prepare(path, out)
                self.assertFalse(out.exists())
                self.assertFalse(list(self.root.glob('.terrain-atlas-*')))
        policy, grass, _ = atlas.grass_policy(original, atlas.read(self.base), Path('/modern'))
        painting = Image.new('RGBA', (144, 144), (10, 20, 30, 255))
        for raw in (bytes([68]) * 576, bytes([32]) * 576):
            with self.assertRaises(ValueError):
                atlas.masked_object_tile(raw, painting, grass, policy)
        raw = bytes(32 if x < 12 else 68 for y in range(24) for x in range(24))
        for bad in (Image.new('RGBA', (143, 144)), Image.new('RGBA', (144, 144), (10, 20, 30, 254))):
            with self.assertRaises(ValueError):
                atlas.masked_object_tile(raw, bad, grass, policy)

    def test_rgba_objects_keep_master_alpha_and_green_volume(self):
        recipe = atlas.read(self.recipe)
        groups = [g for g in recipe['groups'] if g['compile_mode'] == 'atlas_rgba_object']
        if not groups:
            self.skipTest('this recipe has no RGBA object source')
        policy, grass, _ = atlas.grass_policy(recipe, atlas.read(self.base), Path('/modern'))
        for group in groups:
            master = Image.open(self.candidate / group['master_file']).convert('RGBA')
            alpha = master.getchannel('A').tobytes()
            self.assertEqual(hashlib.sha256(alpha).hexdigest(), group['master_alpha_sha256'])
            self.assertEqual(sum(bool(v) for v in alpha), group['master_alpha_nonzero'])
            self.assertEqual(Image.open(self.candidate / group['file']).getextrema()[3], (255, 255))
        # A green object must remain visible even when every underlying source
        # index belongs to grass. Artistic alpha never becomes game coverage.
        painting = Image.new('RGBA', (144, 144))
        painting.paste((30, 140, 70, 255), (40, 40, 104, 104))
        actual, _ = atlas.rgba_object_tile(bytes([68]) * 576, painting, grass, policy)
        self.assertEqual(actual.getpixel((72, 72)), (30, 140, 70, 255))
        self.assertEqual(actual.getpixel((0, 0)), grass.getpixel((0, 0)))
        self.assertEqual(actual.getextrema()[3], (255, 255))

    def test_rgba_object_alpha_contract_fails_closed(self):
        original = atlas.read(self.recipe)
        if not any(g['compile_mode'] == 'atlas_rgba_object' for g in original['groups']):
            self.skipTest('this recipe has no RGBA object source')
        for fault in ('alpha-sha', 'alpha-count', 'opaque-contract', 'mixed-mode'):
            with self.subTest(fault=fault):
                recipe = copy.deepcopy(original)
                if fault == 'alpha-sha':
                    recipe['groups'][0]['master_alpha_sha256'] = '0' * 64
                elif fault == 'alpha-count':
                    recipe['groups'][0]['master_alpha_nonzero'] += 1
                elif fault == 'opaque-contract':
                    recipe['master_alpha_contract'] = 'opaque'
                else:
                    recipe['groups'][0]['compile_mode'] = 'atlas_masked_object'
                path, out = self.root / (fault + '.json'), self.root / (fault + '-candidate')
                atlas.write(path, recipe)
                with self.assertRaises(ValueError):
                    self.prepare(path, out)
                self.assertFalse(out.exists())
                self.assertFalse(list(self.root.glob('.terrain-atlas-*')))
        policy, grass, _ = atlas.grass_policy(original, atlas.read(self.base), Path('/modern'))
        raw = bytes([68]) * 576
        for bad in (Image.new('RGBA', (143, 144)), Image.new('RGB', (144, 144)),
                    Image.new('RGBA', (144, 144)), Image.new('RGBA', (144, 144), (10, 20, 30, 255))):
            with self.assertRaises(ValueError):
                atlas.rgba_object_tile(raw, bad, grass, policy)
        painting = Image.new('RGBA', (144, 144))
        painting.putpixel((72, 72), (10, 20, 30, 255))
        for bad in (raw[:-1], bytes([32]) * 576):
            with self.assertRaises(ValueError):
                atlas.rgba_object_tile(bad, painting, grass, policy)

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
        self.assertEqual(result['bindings'], atlas.read(self.recipe)['expected_bindings'])
        self.assertEqual(result['catalog_assets'], len(atlas.read(self.base)['assets']) + len(atlas.read(self.candidate / 'manifest.json')['new_assets']))


if __name__ == '__main__':
    unittest.main()
