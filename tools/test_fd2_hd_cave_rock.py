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


if __name__ == '__main__':
    unittest.main()
