"""以真實石壁母稿、原生遮罩及GPU收據驗編製／登錄拒收。"""
from copy import deepcopy
import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest

from PIL import Image
from build_fd2_hd_cave_rock import prepare, register


@unittest.skipUnless(os.environ.get('FD2_CAVE_TEST_ROOT'), 'explicit cave inputs required')
class CaveRockTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.root = Path(__file__).resolve().parents[1]
        cls.work = Path(os.environ['FD2_CAVE_TEST_ROOT'])
        cls.catalog = Path(os.environ.get('FD2_CAVE_BASE_CATALOG', str(cls.root / 'remake/assets/themes/modern/catalog.json')))
        floor = os.environ.get('FD2_CAVE_TEST_BATCH') == 'floor'
        cls.recipe = cls.root / ('remake/assets/themes/modern/cave-floor-hd.recipe.json' if floor else 'remake/assets/themes/modern/cave-rock-partial-hd.recipe.json')
        cls.sources = cls.work.parent / 'native-terrain-sources-r2'
        cls.master = cls.work / ('cave-floor-hd-v1-master.png' if floor else 'cave-rock-partial-hd-v1-master.png')
        cls.candidate = cls.work / ('hd1-cave-floor-candidate-r1' if floor else 'hd1-cave-rock-candidate-r1')
        cls.gpu = cls.work / ('hd1-cave-floor-gpu-r1-capture/receipt.json' if floor else 'hd1-cave-rock-game-gpu-r2-capture/receipt.json')
        cls.first = 80 if floor else 32

    def compile(self, out, recipe=None, sources=None, master=None):
        return prepare(recipe or self.recipe, self.catalog, Path('/modern'), Path('/original'),
                       sources or self.sources, self.root / 'remake/assets/maps', master or self.master, out)

    def test_rebuild_and_independent_native_alpha(self):
        native = json.loads((self.sources / 'manifest.json').read_text())['maps'][9]
        with tempfile.TemporaryDirectory(prefix='.cave-rebuild-', dir=self.work) as temp:
            out = Path(temp) / 'candidate'
            result = self.compile(out)
            old = json.loads((self.candidate / 'catalog.json').read_text())
            self.assertEqual(json.loads((out / 'catalog.json').read_text())['assets'], old['assets'])
            for frame in result['frames']:
                p = next(x['file'] for x in native['frames'][frame['frame']]['input_layers'] if x['file'].endswith('/mask.png'))
                mask = Image.open(Path('/original') / p).convert('L').resize((144, 144), Image.Resampling.NEAREST)
                image = Image.open(out / frame['file']).convert('RGBA')
                self.assertEqual(image.getchannel('A').tobytes(), mask.tobytes())
                self.assertTrue(all(a == 0 and r == g == b == 0 or a == 255 for r, g, b, a in image.getdata()))
            with self.assertRaises(ValueError):
                self.compile(out)

    def test_source_failures_publish_nothing(self):
        for fault in ('master-sha', 'recipe-frames', 'native-content', 'overlapping-mask', 'master-geometry'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory(prefix='.cave-reject-', dir=self.work) as temp:
                root = Path(temp)
                recipe = json.loads(self.recipe.read_text())
                source = json.loads((self.sources / 'manifest.json').read_text())
                master = self.master
                if fault == 'master-sha':
                    recipe['master_sha256'] = 'wrong'
                elif fault == 'recipe-frames':
                    recipe['frame_ids'].pop()
                elif fault == 'native-content':
                    source['maps'][9]['frames'][self.first]['native_content_sha256'] = 'wrong'
                elif fault == 'overlapping-mask':
                    source['maps'][9]['frames'][self.first]['input_layers'][2] = deepcopy(source['maps'][9]['frames'][self.first]['input_layers'][1])
                elif fault == 'master-geometry':
                    master = root / 'master.png'
                    Image.new('RGB', (1248, 1254)).save(master)
                    recipe['master_sha256'] = hashlib.sha256(master.read_bytes()).hexdigest()
                rp = root / 'recipe.json'
                rp.write_text(json.dumps(recipe))
                src = root / 'sources'
                src.mkdir()
                (src / 'manifest.json').write_text(json.dumps(source))
                out = root / 'candidate'
                with self.assertRaises(ValueError):
                    self.compile(out, recipe=rp, sources=src, master=master)
                self.assertFalse(out.exists())

    def test_registration_requires_real_complete_consumer_receipt(self):
        original = self.gpu
        for fault in ('valid', 'missing-case', 'wrong-source', 'wrong-hash', 'missing-save'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory(prefix='.cave-register-', dir=self.work) as temp:
                root = Path(temp)
                catalog = root / 'catalog.json'
                before = self.catalog.read_bytes()
                catalog.write_bytes(before)
                gpu = json.loads(original.read_text())
                for case in gpu['cases']:
                    if case.get('file'):
                        (root / case['file']).symlink_to(original.parent / case['file'])
                if fault == 'missing-case':
                    gpu['cases'].pop()
                elif fault == 'wrong-source':
                    gpu['cases'][0]['source_frame'] += 1
                elif fault == 'wrong-hash':
                    gpu['manifest_sha256'] = 'wrong'
                elif fault == 'missing-save':
                    gpu['save_bytes_unchanged'] = 0
                report = root / 'gpu.json'
                report.write_text(json.dumps(gpu))
                if fault == 'valid':
                    result = register(catalog, self.candidate, report)
                    self.assertEqual(result['frames'], 16)
                else:
                    with self.assertRaises(ValueError):
                        register(catalog, self.candidate, report)
                    self.assertEqual(catalog.read_bytes(), before)


@unittest.skipUnless(os.environ.get('FD2_CAVE_DETAILS_ROOT'), 'explicit detail batch required')
class CaveDetailsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.root = Path(__file__).resolve().parents[1]
        cls.work = Path(os.environ['FD2_CAVE_DETAILS_ROOT'])
        cls.rim = os.environ.get('FD2_CAVE_DETAILS_BATCH') == 'rim'
        cls.batch = 'rim' if cls.rim else 'details'
        cls.prefix = 'cave-rim' if cls.rim else 'cave-floor-details'
        cls.expected = (16, 32, 881) if cls.rim else (4, 12, 878)
        cls.first = 0 if cls.rim else 74
        cls.catalog = cls.work / f'hd1-{cls.batch}-base-catalog-r1.json'
        cls.coverage = cls.work / f'hd1-{cls.batch}-base-coverage-r1.json'
        cls.recipe = cls.root / f'remake/assets/themes/modern/{cls.prefix}-hd.recipe.json'
        cls.master = cls.work / f'{cls.prefix}-hd-v1-master.png'
        cls.sources = cls.work.parent / 'native-terrain-sources-r2'
        cls.candidate = cls.work / f'hd1-{cls.batch}-candidate-r1'

    def compile(self, out, recipe=None, coverage=None, sources=None, master=None):
        return prepare(recipe or self.recipe, self.catalog, Path('/modern'), Path('/original'), sources or self.sources,
                       self.root / 'remake/assets/maps', master or self.master, out, coverage_path=coverage or self.coverage)

    def test_complete_rebuild_native_alpha_and_aliases(self):
        with tempfile.TemporaryDirectory(dir=self.work) as temporary:
            out = Path(temporary) / 'candidate'
            result = self.compile(out)
            old = json.loads((self.candidate / 'catalog.json').read_text())
            actual = json.loads((out / 'catalog.json').read_text())
            self.assertEqual(actual['assets'], old['assets'])
            self.assertEqual((len(result['frames']), len(result['bindings'])), self.expected[:2])
            for row in result['frames']:
                self.assertEqual((out / row['file']).read_bytes(), (self.candidate / row['file']).read_bytes())
                with Image.open(out / row['file']) as picture:
                    self.assertEqual(picture.size, (144, 144))
                    self.assertEqual(picture.getchannel('A').getextrema(), (255, 255))
                    if self.rim:
                        source = Image.open(Path('/original') / f'tilesets/fdshap/map_09/tile_{row["frame"]:04d}/frame.png')
                        indices = source.tobytes()
                        for y in range(144):
                            for x in range(144):
                                is_void = indices[(y // 6) * 24 + x // 6] == 0
                                self.assertEqual(picture.getpixel((x, y)) == (0, 0, 0, 255), is_void)
            with self.assertRaises(ValueError):
                self.compile(out)

    def test_bad_sources_publish_no_candidate(self):
        faults = ['master-sha', 'target-missing', 'coverage-kind', 'selector', 'native-sha', 'master-geometry']
        if self.rim:
            faults += ['recipe-frames', 'black-master', 'palette-sha']
        for fault in faults:
            with self.subTest(fault=fault), tempfile.TemporaryDirectory(dir=self.work) as temporary:
                folder = Path(temporary)
                recipe, coverage = json.loads(self.recipe.read_text()), json.loads(self.coverage.read_text())
                source = json.loads((self.sources / 'manifest.json').read_text())
                master = self.master
                if fault == 'master-sha': recipe['master_sha256'] = 'wrong'
                elif fault == 'target-missing': recipe['targets'].pop()
                elif fault == 'coverage-kind': coverage['kind'] = 'wrong'
                elif fault == 'selector':
                    row = next(r for r in coverage['maps'][9]['required'] if r['frame'] == self.first and r['role'] == 'terrain')
                    row['selector']['tile_word'] += 1
                elif fault == 'native-sha': source['maps'][9]['frames'][self.first]['native_content_sha256'] = 'wrong'
                elif fault == 'recipe-frames': recipe['frame_ids'].pop()
                elif fault == 'palette-sha': coverage['palette']['sha256'] = 'wrong'
                else:
                    master = folder / 'bad.png'
                    Image.new('RGB', (1254, 1254) if fault == 'black-master' else (1248, 1254)).save(master)
                    recipe['master_sha256'] = hashlib.sha256(master.read_bytes()).hexdigest()
                cp = folder / 'coverage.json'; cp.write_text(json.dumps(coverage))
                recipe['coverage_sha256'] = hashlib.sha256(cp.read_bytes()).hexdigest()
                sp = folder / 'sources'; sp.mkdir()
                (sp / 'manifest.json').write_text(json.dumps(source))
                recipe['source_manifest_sha256'] = hashlib.sha256((sp / 'manifest.json').read_bytes()).hexdigest()
                rp = folder / 'recipe.json'; rp.write_text(json.dumps(recipe))
                out = folder / 'candidate'
                with self.assertRaises(ValueError): self.compile(out, rp, cp, sp, master)
                self.assertFalse(out.exists())

    def test_complete_gpu_required_for_registration(self):
        original = self.work / f'hd1-{self.batch}-gpu-r1-capture/receipt.json'
        for fault in ('valid', 'missing-source', 'wrong-source', 'missing-game', 'missing-save', 'manifest-sha'):
            with self.subTest(fault=fault), tempfile.TemporaryDirectory(dir=self.work) as temporary:
                folder = Path(temporary)
                catalog = folder / 'catalog.json'; before = self.catalog.read_bytes(); catalog.write_bytes(before)
                gpu = json.loads(original.read_text())
                for row in gpu['cases']:
                    if row.get('file'): (folder / row['file']).symlink_to(original.parent / row['file'])
                if fault == 'missing-source': gpu['cases'].pop(0)
                elif fault == 'wrong-source': gpu['cases'][0]['source_frame'] += 1
                elif fault == 'missing-game': gpu['cases'].pop()
                elif fault == 'missing-save': gpu['maps'][0]['save_bytes'] = 0
                elif fault == 'manifest-sha': gpu['detail_manifest_sha256'] = 'wrong'
                path = folder / 'gpu.json'; path.write_text(json.dumps(gpu))
                if fault == 'valid':
                    result = register(catalog, self.candidate, path)
                    self.assertEqual((result['frames'], result['bindings'], result['catalog_assets']), self.expected)
                else:
                    with self.assertRaises(ValueError): register(catalog, self.candidate, path)
                    self.assertEqual(catalog.read_bytes(), before)


if __name__ == '__main__':
    unittest.main()
