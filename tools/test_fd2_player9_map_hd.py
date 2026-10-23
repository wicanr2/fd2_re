#!/usr/bin/env python3
"""以真實候選與原包測玩家9編製拒收、清冊邊界及正對照。"""
import copy
import json
import os
from pathlib import Path
import shutil
import tempfile
import unittest

from PIL import Image
import build_fd2_player9_map_hd as compiler
import validate_modern_theme_catalog as catalog


class Player9CompilerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        value=os.environ.get('FD2_TEST_PLAYER9_RECIPE')
        if not value:
            raise unittest.SkipTest('explicit real player9 recipe required')
        cls.original=Path(os.environ['FD2_TEST_ORIGINAL_PACK'])
        cls.recipe_path=Path(value)
        cls.recipe=json.loads(cls.recipe_path.read_text())
        cls.font=Path(os.environ['FD2_TEST_PLAYER9_FONT'])

    def inputs(self,root):
        for source in self.recipe['masters'].values():
            shutil.copyfile(self.recipe_path.parent/source['file'],root/source['file'])
        recipe=copy.deepcopy(self.recipe)
        return recipe

    def write(self,root,recipe):
        p=root/'recipe.json';p.write_text(json.dumps(recipe));return p

    def test_actual_positive_40_and_immutable_masters(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp);recipe=self.inputs(root);path=self.write(root,recipe)
            before={k:compiler.sha(root/r['file']) for k,r in recipe['masters'].items()}
            assets=compiler.build(self.original,path,self.font,root/'candidate')
            self.assertEqual([a['frame_count'] for a in assets],[27,13])
            self.assertEqual([a['source_resource'] for a in assets],[6,5])
            for asset in assets:
                for name,digest,desc in zip(asset['files'],asset['frame_sha256'],asset['native_lmi_frames']):
                    self.assertEqual(compiler.sha(root/'candidate'/name),digest)
                    with Image.open(root/'candidate'/name) as im:
                        self.assertEqual(im.size,(desc['width']*6,desc['height']*6))
                        low,high=im.getchannel('A').getextrema();self.assertEqual(low,0);self.assertGreater(high,0)
            self.assertEqual(before,{k:compiler.sha(root/r['file']) for k,r in recipe['masters'].items()})
            public=root/'public';public.mkdir();public_recipe=self.write(public,recipe)
            redirected=compiler.build(self.original,public_recipe,self.font,root/'redirected',root)
            self.assertEqual(redirected,assets)
            with self.assertRaisesRegex(ValueError,'already exists'):
                compiler.build(self.original,path,self.font,root/'candidate')

    def test_refusals_are_atomic_with_real_positive_inputs(self):
        for fault in ['schema','missing-master','unsafe-name','master-sha','rgb','opaque','empty','font-sha','late-native','palette-source']:
            with self.subTest(fault=fault),tempfile.TemporaryDirectory() as tmp:
                root=Path(tmp);recipe=self.inputs(root);font=self.font
                row=recipe['masters']['hood-closed'];p=root/row['file']
                if fault=='schema':recipe['unknown']=True
                elif fault=='missing-master':del recipe['masters']['dark-ring']
                elif fault=='unsafe-name':row['file']='../outside.png'
                elif fault=='master-sha':row['sha256']='0'*64
                elif fault in ['rgb','opaque','empty']:
                    with Image.open(p) as src: im=src.convert('RGB' if fault=='rgb' else 'RGBA')
                    if fault in ['opaque','empty']:im.putalpha(255 if fault=='opaque' else 0)
                    im.save(p);row['sha256']=compiler.sha(p)
                elif fault=='font-sha':font=root/'wrong.ttc';font.write_bytes(b'wrong font')
                elif fault=='late-native':recipe['native_source']['resources'][0]['frames'][-1]['pixels_sha256']='0'*64
                elif fault=='palette-source':recipe['native_source']['palette_sha256']='0'*64
                path=self.write(root,recipe);out=root/'rejected'
                with self.assertRaises(ValueError):compiler.build(self.original,path,font,out)
                self.assertFalse(out.exists())

    def test_original_png_and_raw_tampering_are_refused(self):
        for fault in ['png','raw']:
            with self.subTest(fault=fault),tempfile.TemporaryDirectory() as tmp:
                root=Path(tmp);pack=root/'pack';pack.mkdir();paths={'manifest.json','palette/fdother_000.json'}
                for resource,indices in compiler.RANGES.items():
                    folder=compiler.FOLDERS[resource];bank=json.loads((self.original/folder/'bank.json').read_text())
                    paths.add(folder+'/bank.json');paths.add(f'raw/FDOTHER/FDOTHER_{resource:03d}.bin')
                    paths.update(folder+'/'+bank['entries'][i]['frame'] for i in indices)
                bank=json.loads((self.original/compiler.FOLDERS[6]/'bank.json').read_text())
                changed=(compiler.FOLDERS[6]+'/'+bank['entries'][113]['frame']) if fault=='png' else 'raw/FDOTHER/FDOTHER_006.bin'
                for name in paths:
                    target=pack/name;target.parent.mkdir(parents=True,exist_ok=True)
                    if name==changed:target.write_bytes((self.original/name).read_bytes()+b'tampered')
                    else:target.symlink_to(self.original/name)
                with self.assertRaisesRegex(ValueError,'SHA'):
                    compiler.build(pack,self.recipe_path,self.font,root/'rejected')
                self.assertFalse((root/'rejected').exists())


class Player9CatalogTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        path=os.environ.get('FD2_TEST_PLAYER9_MAP_CATALOG')
        if not path:raise unittest.SkipTest('explicit complete catalog required')
        cls.path=Path(path);cls.data=json.loads(cls.path.read_text())

    def test_full_positive(self):
        self.assertEqual(len(catalog.validate(False,self.path)['assets']),673)

    def test_exact_disjoint_ranges_and_coupled_banks(self):
        for fault in ['count','resource','index','repeat-entry','consumer','partial5','partial6','duplicate-death','duplicate-effect']:
            with self.subTest(fault=fault),tempfile.TemporaryDirectory() as tmp:
                data=copy.deepcopy(self.data)
                effects=next(a for a in data['assets'] if a['role']=='map_overlay_hd_set' and a['source_resource']==6)
                results=next(a for a in data['assets'] if a['role']=='map_overlay_hd_set' and a['frame_count']==13)
                if fault=='count':effects['frame_count']=26
                elif fault=='resource':effects['source_resource']=7
                elif fault=='index':results['native_lmi_frames'][-1]['index']=119
                elif fault=='repeat-entry':results['native_lmi_frames'][-1]['index']=117
                elif fault=='consumer':effects['consumer_contract']='native_fdother_ordinary_rgba_hd6_v1'
                elif fault=='partial5':data['assets'].remove(results)
                elif fault=='partial6':data['assets'].remove(effects)
                else:
                    extra=copy.deepcopy(effects if fault=='duplicate-effect' else next(a for a in data['assets'] if a['role']=='map_overlay_hd_set' and a['frame_count']==12))
                    extra['asset_id']+='-duplicate';data['assets'].append(extra)
                path=Path(tmp)/'catalog.json';path.write_text(json.dumps(data))
                with self.assertRaises(ValueError):catalog.validate(False,path)


if __name__=='__main__':unittest.main()
