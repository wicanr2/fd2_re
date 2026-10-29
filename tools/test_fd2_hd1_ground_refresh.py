#!/usr/bin/env python3
"""HD1底材刷新來源重編、前置拒收與無關欄位保護。"""
import copy,json,os,tempfile,unittest
from pathlib import Path
from PIL import Image
from refresh_fd2_hd1_ground import prepare,read,sha,structural
ROOT=Path(__file__).resolve().parent.parent
WORK=Path(os.environ.get('FD2_PAINTED_GROUND_ROOT',ROOT/'work/promo-cadence-20261018/hd-fonts/ordinary-effects-r1/hd1-painted-ground-r1'))
RECIPE=ROOT/'remake/assets/themes/modern/hd1-painted-ground-refresh.recipe.json'
class GroundRefreshTests(unittest.TestCase):
    def call(self,out,recipe=RECIPE,base=None,masters=WORK):
        return prepare(recipe,base or WORK/'base-catalog.json',Path('/modern'),Path('/original'),ROOT/'work/promo-cadence-20261018/hd-fonts/native-terrain-sources-r2',ROOT/'remake/assets/maps',masters,out)
    def test_exact_rebuild_and_384_reference_closure(self):
        expected=read(WORK/'candidate/manifest.json')
        with tempfile.TemporaryDirectory(dir=WORK) as directory:
            out=Path(directory)/'candidate'
            got=self.call(out)
            self.assertEqual((got['source_groups'],got['target_slots'],got['maps']),(87,384,21))
            self.assertEqual((out/'catalog.json').read_bytes(),(WORK/'candidate/catalog.json').read_bytes())
            self.assertEqual((out/'gpu-detail.json').read_bytes(),(WORK/'candidate/gpu-detail.json').read_bytes())
            for frame in expected['frames']:
                self.assertEqual((out/'pack'/frame['file']).read_bytes(),(WORK/'candidate/pack'/frame['file']).read_bytes())
            self.assertEqual(got['equal_material_edge_pixels'],expected['equal_material_edge_pixels'])
    def test_preflight_rejects_and_keeps_output_unpublished(self):
        rejected=[]
        with tempfile.TemporaryDirectory(dir=WORK) as directory:
            d=Path(directory);r=read(RECIPE)
            cases=[
                ('base-hash',lambda x:x.update(base_catalog_sha256='0'*64)),
                ('native-hash',lambda x:x.update(source_manifest_sha256='0'*64)),
                ('style-hash',lambda x:x['style'].update(sha256='0'*64)),
                ('profile-hash',lambda x:x['profiles'][0].update(sha256='0'*64)),
                ('master-hash',lambda x:x['materials']['grass'].update(sha256='0'*64)),
                ('master-identity',lambda x:x['materials']['grass'].update(old_file='wrong.png')),
                ('material-missing',lambda x:x['materials'].pop('sand'))]
            for name,mutate in cases:
                bad=copy.deepcopy(r);mutate(bad);p=d/(name+'.json');p.write_text(json.dumps(bad))
                out=d/name
                with self.assertRaises(ValueError):self.call(out,p)
                self.assertFalse(out.exists());rejected.append(name)
            master=d/'masters';master.mkdir()
            for m in r['materials'].values():(master/m['file']).symlink_to((WORK/m['file']).resolve())
            m=r['materials']['grass'];p=master/m['file'];p.unlink()
            with Image.open(WORK/m['file']) as src:im=src.convert('RGBA')
            im.putpixel((0,0),(0,0,0,0));im.save(p)
            m['sha256']=sha(p);bad=d/'alpha.json';bad.write_text(json.dumps(r))
            with self.assertRaises(ValueError):self.call(d/'bad-alpha',bad,masters=master)
            self.assertFalse((d/'bad-alpha').exists());rejected.append('master-alpha')
            existing=d/'existing';existing.mkdir();marker=existing/'keep';marker.write_text('original')
            with self.assertRaises(ValueError):self.call(existing)
            self.assertEqual(marker.read_text(),'original');rejected.append('existing-output')
        self.assertEqual(len(rejected),9)
    def test_structural_rejects_unrelated_identity_role_and_reference_changes(self):
        base=read(WORK/'base-catalog.json');good=read(WORK/'candidate/catalog.json')
        recipe=read(RECIPE);proof=read(WORK/'candidate/manifest.json')
        repl={v['old_file']:v for v in proof['frames']}
        self.assertEqual(structural(base,good,recipe,repl),68)
        mutations=[
            lambda d:d['assets'][0].update(source_refs=['wrong']),
            lambda d:d['assets'][0].update(asset_id='wrong'),
            lambda d:next(a for a in d['assets'] if a['role']=='native_terrain_hd_set' and any(f in repl for f in next(b for b in base['assets'] if b['asset_id']==a['asset_id'])['files'])).update(frame_ids=[999]),
            lambda d:d['assets'].pop()]
        for mutate in mutations:
            bad=copy.deepcopy(good);mutate(bad)
            with self.assertRaises(ValueError):structural(base,bad,recipe,repl)
if __name__=='__main__':unittest.main()
