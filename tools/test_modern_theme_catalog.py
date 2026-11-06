#!/usr/bin/env python3
"""Deterministic tests for the modern-theme prototype catalog."""
from __future__ import annotations

import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import validate_modern_theme_catalog as validator


class ModernThemeCatalogTests(unittest.TestCase):
    def test_battle_png_audit_checks_explicit_geometry_and_v1_rejection(self):
        from PIL import Image
        from audit_fd2_battle_hd_coverage import check_pngs
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            image = Image.new('RGBA', (12, 18))
            image.putpixel((2, 2), (255, 0, 0, 255))
            image.save(root/'frame.png')
            asset = dict(files=['frame.png'], frame_sha256=[hashlib.sha256((root/'frame.png').read_bytes()).hexdigest()],
                         native_frames=[dict(width=2, height=2)], source_archive='FIGANI.DAT',
                         consumer_contract='native_figani_rgba_hd6_v2',
                         hd_frames=[dict(x=100, y=100, width=12, height=18)])
            check_pngs(asset, root)
            for case in ['missing', 'negative', 'overflow', 'size', 'v1']:
                bad = copy.deepcopy(asset)
                if case == 'missing': bad.pop('hd_frames')
                elif case == 'negative': bad['hd_frames'][0]['x'] = -1
                elif case == 'overflow': bad['hd_frames'][0]['x'] = 1920
                elif case == 'size': bad['hd_frames'][0]['height'] = 12
                elif case == 'v1': bad['consumer_contract'] = 'native_figani_rgba_hd6_v1'
                with self.subTest(case=case), self.assertRaises(ValueError): check_pngs(bad, root)

    def test_explicit_hd_bounds_keep_native_descriptors_and_reject_partial_rectangles(self):
        base = validator.validate(False)
        asset = next(a for a in base['assets'] if a['role'] == 'battle_animation_hd_set')
        original_native = copy.deepcopy(asset['native_frames'])
        asset['consumer_contract'] = 'native_figani_rgba_hd6_v2'
        asset['layout_policy'] = 'explicit_hd_frame_rects_v2'
        asset['hd_frames'] = [dict(x=0, y=0, width=12, height=18) for _ in asset['files']]
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / 'catalog.json'
            path.write_text(json.dumps(base))
            validator.validate(False, path)
            self.assertEqual(asset['native_frames'], original_native)
            for case in ['missing', 'short', 'key', 'extra', 'bool', 'fraction', 'negative', 'empty', 'right', 'bottom', 'v1']:
                with self.subTest(case=case):
                    bad = copy.deepcopy(base)
                    a = next(a for a in bad['assets'] if a['asset_id'] == asset['asset_id'])
                    if case == 'missing': a.pop('hd_frames')
                    elif case == 'short': a['hd_frames'].pop()
                    elif case == 'key': a['hd_frames'][0].pop('x')
                    elif case == 'extra': a['hd_frames'][0]['extra'] = 0
                    elif case == 'bool': a['hd_frames'][0]['x'] = False
                    elif case == 'fraction': a['hd_frames'][0]['width'] = 12.5
                    elif case == 'negative': a['hd_frames'][0]['x'] = -1
                    elif case == 'empty': a['hd_frames'][0]['width'] = 0
                    elif case == 'right': a['hd_frames'][0]['x'] = 1910
                    elif case == 'bottom': a['hd_frames'][0]['y'] = 1190
                    elif case == 'v1':
                        a['consumer_contract'] = 'native_figani_rgba_hd6_v1'
                        a['layout_policy'] = 'contain_native_descriptor_hd6_candidate'
                    path.write_text(json.dumps(bad))
                    with self.assertRaises(ValueError): validator.validate(False, path)

    def test_hd_physical_sound_group_and_camera_profile_are_optional_and_atomic(self):
        base = validator.validate(False)
        base['assets'] = [a for a in base['assets'] if a['role'] not in
                          ('battle_physical_sfx_hd', 'battle_presentation_profile')]
        audio = [dict(asset_id=f'modern.test.physical.{bank}', role='battle_physical_sfx_hd',
                      status='runtime_candidate', file=f'impact-{bank}.ogg', sha256='0'*64,
                      source_refs=['tools/build_fd2_hd1_impact_preview.py'],
                      consumer_contract='physical_impact_ogg_v1', source_archive='FDOTHER.DAT',
                      source_resource=bank, source_metadata_sha256='0'*64, cue_ids=cues,
                      sample_rate=48000, channels=1)
                 for bank, cues in [(49, [1, 3, 4, 5]), (53, [2])]]
        profile = dict(asset_id='modern.test.presentation.b', role='battle_presentation_profile',
                       status='runtime_candidate', file='camera.json', sha256='0'*64,
                       source_refs=['docs/knowledge-base/61-modern-theme-runtime-spec.md'],
                       consumer_contract='hd1_battle_presentation_v1')
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'catalog.json'
            good = copy.deepcopy(base); good['assets'] += audio + [profile]
            path.write_text(json.dumps(good)); validator.validate(False, path)
            for case in ['partial', 'miss', 'spin', 'other-bank', 'duplicate-cue',
                         'rate', 'channels', 'path', 'duplicate-profile', 'profile-contract']:
                with self.subTest(case=case):
                    bad = copy.deepcopy(good); a = bad['assets'][-3]
                    if case == 'partial': bad['assets'].pop(-2)
                    elif case == 'miss': a['cue_ids'][0] = 0
                    elif case == 'spin': a['cue_ids'][0] = 2
                    elif case == 'other-bank': a['source_resource'] = 50
                    elif case == 'duplicate-cue': a['cue_ids'].append(1)
                    elif case == 'rate': a['sample_rate'] = 22050
                    elif case == 'channels': a['channels'] = 2
                    elif case == 'path': a['file'] = '../impact.ogg'
                    elif case == 'duplicate-profile':
                        duplicate = copy.deepcopy(profile); duplicate['asset_id'] += '.second'
                        bad['assets'].append(duplicate)
                    elif case == 'profile-contract': bad['assets'][-1]['consumer_contract'] = 'unknown'
                    path.write_text(json.dumps(bad))
                    with self.assertRaises(ValueError): validator.validate(False, path)

    def test_native_terrain_shared_png_keeps_unique_frames_and_consistent_hash(self):
        catalog = validator.validate(False)
        entry = next(a for a in catalog['assets'] if a['asset_id']=='modern.terrain.ground_bases.map00.native_hd')
        first, repeated = entry['frame_ids'].index(52), entry['frame_ids'].index(244)
        self.assertEqual(entry['files'][first], entry['files'][repeated])
        self.assertEqual(entry['frame_sha256'][first], entry['frame_sha256'][repeated])
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'catalog.json'
            for case in ('positive', 'hash', 'frame', 'path'):
                bad = copy.deepcopy(catalog)
                item = next(a for a in bad['assets'] if a['asset_id']==entry['asset_id'])
                if case=='hash':item['frame_sha256'][repeated]='0'*64
                if case=='frame':item['frame_ids'][repeated]=52
                if case=='path':item['files'][repeated]='../bad.png'
                path.write_text(json.dumps(bad))
                with self.subTest(case=case):
                    if case=='positive':validator.validate(False,path)
                    else:
                        with self.assertRaisesRegex(ValueError, 'conflicting native HD|native HD terrain identity|unsafe private file'):
                            validator.validate(False,path)

    def test_tracked_catalog_rejects_candidate_root(self):
        data = validator.validate(False)
        data['private_root'] = 'work/disposable-hd-candidate'
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'catalog.json'
            path.write_text(json.dumps(data))
            validator.validate(False, path)  # An explicitly selected candidate is allowed.
            with mock.patch.object(validator, 'CATALOG', path):
                for selected in [None, path]:
                    with self.subTest(selected=selected), self.assertRaisesRegex(ValueError, 'tracked catalog must use private_root'):
                        validator.validate(False, selected)

    def test_lmi_overlay_bank_requires_ordered_complete_descriptors(self):
        base = validator.validate(False)
        base['assets'] = [a for a in base['assets'] if a['role'] != 'map_overlay_hd_set']
        asset = dict(asset_id='modern.test.lmi.hd6',role='map_overlay_hd_set',status='runtime_candidate',
                     files=[f'entry-{i}.png' for i in range(12)],width=1920,height=1200,frame_count=12,
                     frame_sha256=['0'*64]*12,source_archive='FDOTHER.DAT',source_resource=5,
                     source_metadata_sha256='0'*64,consumer_contract='native_lmi1_rgba_hd6_v1',
                     alpha_contract='coverage',cycle_policy='native_lmi1_entries',layout_policy='native_lmi1_rect_hd6',
                     native_lmi_frames=[dict(index=i,width=72,height=71,pixels_sha256='0'*64) for i in range(68,80)],
                     source_files=['master.png'],source_sha256=['0'*64],source_refs=['tools/build_fd2_lmi_death_hd.py'])
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'catalog.json'
            good=copy.deepcopy(base)
            good['assets'].append(asset)
            path.write_text(json.dumps(good))
            validator.validate(False,path)
            for name in ['order','geometry','hash','path','missing','source','contract','duplicate']:
                with self.subTest(name=name):
                    bad=copy.deepcopy(good)
                    entry=bad['assets'][-1]
                    if name=='order': entry['native_lmi_frames'][-1]['index']=68
                    elif name=='geometry': entry['native_lmi_frames'][-1]['width']=0
                    elif name=='hash': entry['native_lmi_frames'][-1]['pixels_sha256']='bad'
                    elif name=='path': entry['files'][-1]='../bad.png'
                    elif name=='missing': entry['files'].pop()
                    elif name=='source': entry['source_resource']=6
                    elif name=='contract': entry['consumer_contract']='native_figani_rgba_hd6_v1'
                    elif name=='duplicate':
                        duplicate=copy.deepcopy(entry)
                        duplicate['asset_id']='modern.test.lmi.duplicate'
                        bad['assets'].append(duplicate)
                    path.write_text(json.dumps(bad))
                    with self.assertRaises(ValueError): validator.validate(False,path)

    def test_battle_animation_resource_requires_complete_safe_descriptors(self):
        base = validator.validate(False)
        asset = {'asset_id':'modern.test.figani.hd6','role':'battle_animation_hd_set','status':'runtime_candidate',
                 'files':['test0.png'],'width':1920,'height':1200,'frame_count':1,'frame_sha256':['0'*64],
                 'source_archive':'FIGANI.DAT','source_resource':90,'source_metadata_sha256':'0'*64,
                 'native_header':[0,1,0],'native_frames':[{'x':1,'y':2,'width':2,'height':2,'delay_native':2,
                    'raw_byte_4':0,'raw_byte_5':0,'raw_byte_7':0}],
                 'consumer_contract':'native_figani_rgba_hd6_v1','alpha_contract':'coverage',
                 'cycle_policy':'native_descriptor_key_poses','layout_policy':'contain_native_descriptor_hd6_candidate',
                 'source_files':['test-master.png'],'source_sha256':['0'*64],'source_refs':['docs/data/fd2-reference-files.json']}
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'catalog.json'
            good=copy.deepcopy(base)
            # 合成fixture先隔離同一來源，不依賴該號碼尚未正式登錄。
            good['assets']=[a for a in good['assets'] if not (
                a['role']=='battle_animation_hd_set' and a['source_resource']==asset['source_resource'])]
            good['assets'].append(asset)
            path.write_text(json.dumps(good));validator.validate(False,path)
            for key in ['missing','delay','path','header','duplicate','alpha']:
                bad=copy.deepcopy(good);entry=bad['assets'][-1]
                if key=='missing':entry['native_frames']=[]
                if key=='delay':entry['native_frames'][0]['delay_native']=256
                if key=='path':entry['files']=['../test0.png']
                if key=='header':entry['native_header']=[1,2,0]
                if key=='duplicate':
                    clone=copy.deepcopy(entry);clone['asset_id']='modern.test.figani.duplicate';bad['assets'].append(clone)
                if key=='alpha':entry['alpha_contract']='binary'
                path.write_text(json.dumps(bad))
                with self.subTest(key=key), self.assertRaises(ValueError):validator.validate(False,path)

    def test_distinct_character_concepts_preserve_existing_design_and_reject_duplicate_id(self):
        catalog = validator.validate(False)
        original = next(a for a in catalog['assets'] if a['role'] == 'portrait_concept')
        added = copy.deepcopy(original)
        added['asset_id'] = 'modern.test.character.design'
        catalog['assets'].append(added)
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'catalog.json'
            path.write_text(json.dumps(catalog))
            self.assertEqual(validator.validate(False, path)['assets'][0], original)
            catalog['assets'].append(copy.deepcopy(added))
            path.write_text(json.dumps(catalog))
            with self.assertRaisesRegex(ValueError, 'duplicate asset_id'):
                validator.validate(False, path)

    def test_tracked_catalog_and_private_files_are_consistent(self):
        catalog = validator.validate(verify_private=True)
        self.assertEqual(catalog["theme_id"], "modern-handpainted-a")
        self.assertEqual({asset["status"] for asset in catalog["assets"]}, {"concept", "runtime_candidate"})
        # Sol has moved to the HD portrait role. Check an explicit remaining
        # 80px asset rather than treating catalog order as character identity.
        candidate = next(asset for asset in catalog["assets"]
                         if asset['asset_id'] == 'modern.celia.portrait.style_a.frame0')
        self.assertEqual(
            (candidate["width"], candidate["height"], candidate["speaker_id"], candidate["frame"]),
            (80, 80, 8, 0),
        )
        hano = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_001.style_a"
        )
        self.assertEqual(
            (
                hano["source_group"],
                hano["frame_count"],
                hano["alpha_contract"],
                hano["master_file"],
            ),
            (1, 12, "binary", "fdicon-001-hd1-base-face-q-v1-master.png"),
        )
        sprite_groups = {
            asset["source_group"]
            for asset in catalog["assets"]
            if asset.get("role") == "map_sprite_set"
        }
        hd_groups = {asset['source_group'] for asset in catalog['assets']
                     if asset['role'] == 'map_sprite_hd_set'}
        self.assertEqual(sprite_groups, validator.VISIBLE_FDICON_GROUPS if hd_groups else set(range(96)) - {39, 57})
        if hd_groups:
            self.assertEqual(hd_groups, validator.VISIBLE_FDICON_GROUPS)
        for group in (48, 66, 67, 74, 75):
            static = next(asset for asset in catalog["assets"] if asset.get("source_group") == group)
            self.assertEqual(
                (static["consumer_contract"], static["frame_count"], static["cycle_policy"]),
                ("fdicon_static_sprite_3x24_v1", 3, "static_three_phase"),
            )
            self.assertEqual(len(set(static["frame_sha256"])), 3)
        yuni = next(asset for asset in catalog["assets"] if asset.get("source_group") == 52)
        self.assertEqual(yuni["cycle_policy"], "source_exact_repeats")
        self.assertEqual(yuni["frame_sha256"][6], yuni["frame_sha256"][8])
        # Individual Q redraws may differ across originally shared groups.
        # Within-group native phase equivalence and parent/HD pairing remain
        # checked by HDMapCatalogContractTests and the full validator.
        self.assertEqual(len(set(yuni["frame_sha256"])), 11)
        soldier = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_076.style_a"
        )
        self.assertEqual(soldier["source_group"], 76)
        self.assertIn("remake/assets/scenarios/ch03.json", soldier["source_refs"])
        captain = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_077.style_a"
        )
        self.assertEqual(captain["source_group"], 77)
        self.assertIn("remake/assets/maps/map2/map2_units.json", captain["source_refs"])
        swordsman = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_082.style_a"
        )
        self.assertEqual(swordsman["source_group"], 82)
        self.assertIn("remake/assets/maps/map7/map7_units.json", swordsman["source_refs"])
        horned_swordsman = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_083.style_a"
        )
        self.assertEqual(horned_swordsman["source_group"], 83)
        self.assertIn("remake/assets/maps/map17/map17_units.json", horned_swordsman["source_refs"])
        heavy_unit = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_080.style_a"
        )
        self.assertEqual(heavy_unit["source_group"], 80)
        self.assertIn("remake/assets/maps/map16/map16_units.json", heavy_unit["source_refs"])
        hooded_unit = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_091.style_a"
        )
        self.assertEqual(hooded_unit["source_group"], 91)
        self.assertIn("remake/assets/maps/map21/map21_units.json", hooded_unit["source_refs"])
        hooded_swordsman = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_088.style_a"
        )
        self.assertEqual(hooded_swordsman["source_group"], 88)
        self.assertIn("remake/assets/maps/map21/map21_units.json", hooded_swordsman["source_refs"])
        bucket_helmet_soldier = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_085.style_a"
        )
        self.assertEqual(bucket_helmet_soldier["source_group"], 85)
        self.assertIn("remake/assets/maps/map20/map20_units.json", bucket_helmet_soldier["source_refs"])
        broad_shouldered_soldier = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_078.style_a"
        )
        self.assertEqual(broad_shouldered_soldier["source_group"], 78)
        self.assertIn("remake/assets/maps/map17/map17_units.json", broad_shouldered_soldier["source_refs"])
        rust_hooded_soldier = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_090.style_a"
        )
        self.assertEqual(rust_hooded_soldier["source_group"], 90)
        self.assertIn("remake/assets/maps/map3/map3_units.json", rust_hooded_soldier["source_refs"])
        self.assertIn("remake/assets/maps/map8/map8_units.json", rust_hooded_soldier["source_refs"])
        fortress_guard = next(
            asset
            for asset in catalog["assets"]
            if asset.get("asset_id") == "modern.fdicon.group_086.style_a"
        )
        self.assertEqual(fortress_guard["source_group"], 86)
        self.assertIn("remake/assets/maps/map25/map25_units.json", fortress_guard["source_refs"])
        self.assertIn("remake/assets/maps/map31/map31_units.json", fortress_guard["source_refs"])


class HDMapCatalogContractTests(unittest.TestCase):
    def setUp(self):
        self.catalog = validator.validate(False)
        self.catalog['assets'] = [a for a in self.catalog['assets'] if a['role'] != 'map_sprite_hd_set']
        parents = {a['source_group']: a for a in self.catalog['assets'] if a['role'] == 'map_sprite_set'}
        for group in sorted(validator.VISIBLE_FDICON_GROUPS-set(parents)):
            parent = copy.deepcopy(parents[0])
            parent.update(asset_id=f'modern.test.group_{group:03d}', source_group=group)
            self.catalog['assets'].append(parent)
            parents[group] = parent
        for group, parent in parents.items():
            child = copy.deepcopy(parent)
            child.update(asset_id=parent['asset_id']+'.hd6', role='map_sprite_hd_set',
                         width=144, height=144, alpha_contract='coverage',
                         layout_policy='contain_22px_cell_feet_v1',
                         consumer_contract='fdicon_hd_static_sprite_3x144_v1' if parent['frame_count']==3
                         else 'fdicon_hd_map_sprite_12x144_v1',
                         master_file='selected-master.png', frame_overrides={},
                         source_files=['selected-master.png'], source_sha256=['a'*64])
            child['frame_sha256'] = [hashlib.sha256(('hd-'+h).encode()).hexdigest() for h in parent['frame_sha256']]
            self.catalog['assets'].append(child)

    def validate(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'catalog.json'
            path.write_text(json.dumps(self.catalog))
            return validator.validate(False, path)

    def child(self, group=0):
        return next(a for a in self.catalog['assets'] if a['role']=='map_sprite_hd_set' and a['source_group']==group)

    def test_full_native_collection_and_source_repeat_equivalence(self):
        self.validate()
        self.assertEqual(self.child(52)['frame_sha256'][6], self.child(52)['frame_sha256'][8])

    def test_reject_incomplete_collection_or_parent(self):
        original = copy.deepcopy(self.catalog)
        for role in ('map_sprite_set', 'map_sprite_hd_set'):
            with self.subTest(role=role):
                self.catalog = copy.deepcopy(original)
                self.catalog['assets'] = [a for a in self.catalog['assets']
                                          if not (a['role']==role and a['source_group']==135)]
                with self.assertRaisesRegex(ValueError, 'complete native bank'):
                    self.validate()

    def test_reject_zero_write_and_out_of_bank_groups(self):
        for group in (104, 139, 140, -1):
            with self.subTest(group=group):
                self.child()['source_group'] = group
                with self.assertRaisesRegex(ValueError, 'map sprite identity'):
                    self.validate()
                next(a for a in self.catalog['assets'] if a['role']=='map_sprite_hd_set' and a['source_group']==group)['source_group'] = 0

    def test_reject_source_provenance_gaps_and_unsafe_names(self):
        child = self.child()
        for sources, hashes in ((['selected-master.png','unused.png'], ['a'*64,'b'*64]),
                                (['../selected-master.png'], ['a'*64]),
                                (['selected-master.png'], ['invalid']),
                                ([{}], ['a'*64])):
            with self.subTest(sources=sources):
                child['source_files'], child['source_sha256'] = sources, hashes
                with self.assertRaises(ValueError):
                    self.validate()

    def test_reject_override_of_repeated_phase(self):
        self.child(52)['frame_overrides'] = {'8':'selected-master.png'}
        with self.assertRaisesRegex(ValueError, 'repeated phase'):
            self.validate()


class NativeTerrainCatalogTests(unittest.TestCase):
    def setUp(self):
        self.catalog = validator.validate(False)
        self.asset = next(a for a in self.catalog['assets'] if a['role'] == 'native_terrain_hd_set')

    def validate(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp)/'catalog.json'
            path.write_text(json.dumps(self.catalog))
            return validator.validate(False, path)

    def test_sparse_terrain_and_foreground_identity(self):
        self.validate()
        self.assertEqual(self.asset['semantics'].count('terrain'), 24)
        self.assertEqual(self.asset['semantics'].count('foreground'), 6)
        self.assertEqual(self.asset['frame_count'], 30)

    def test_reject_duplicate_identity_in_or_across_sets(self):
        self.asset['frame_ids'][1] = self.asset['frame_ids'][0]
        with self.assertRaisesRegex(ValueError, 'terrain identity'):
            self.validate()
        self.setUp()
        duplicate = copy.deepcopy(self.asset)
        duplicate['asset_id'] += '.duplicate'
        self.catalog['assets'].append(duplicate)
        with self.assertRaisesRegex(ValueError, 'duplicate native HD terrain identity'):
            self.validate()

    def test_reject_invalid_shape_role_hash_or_file(self):
        for field, value in [('frame_count', 0), ('map_id', 33), ('width', 24),
                             ('semantics', ['unknown']*30), ('frame_sha256', ['invalid']*30),
                             ('files', ['../outside.png']+self.asset['files'][1:])]:
            with self.subTest(field=field):
                original = self.asset[field]
                self.asset[field] = value
                with self.assertRaises(ValueError):
                    self.validate()
                self.asset[field] = original


class EmbeddedUIIconCatalogTests(unittest.TestCase):
    def test_embedded_ui_symbol_requires_exact_source_rect_and_alpha_contract(self):
        base=validator.validate(False)
        identity='modern.ui.preparation.frame_137.count-head.hd6'
        index=next(i for i,a in enumerate(base['assets']) if a['asset_id']==identity)
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'catalog.json'
            for fault in ['frame','hash','rect','size','contract','source','metadata','alpha','master-proof','duplicate']:
                bad=copy.deepcopy(base);asset=bad['assets'][index]
                if fault=='frame':asset['native_lmi_frames'][0]['index']=136
                elif fault=='hash':asset['native_lmi_frames'][0]['pixels_sha256']='0'*64
                elif fault=='rect':asset['native_source_rect']=[21,22,45,40]
                elif fault=='size':asset['width']=143
                elif fault=='contract':asset['consumer_contract']='native_raw_ui_icon_hd6_v1'
                elif fault=='source':asset['source_resource']=6
                elif fault=='metadata':asset['source_metadata_sha256']='0'*64
                elif fault=='alpha':asset['alpha_contract']='opaque'
                elif fault=='master-proof':asset['source_sha256']=[]
                elif fault=='duplicate':
                    extra=copy.deepcopy(asset);extra['asset_id']+='.duplicate';bad['assets'].append(extra)
                path.write_text(json.dumps(bad))
                with self.subTest(fault=fault),self.assertRaises(ValueError):
                    validator.validate(False,path)

    def test_embedded_ui_master_and_png_hashes_are_verified(self):
        base=validator.validate(False)
        asset=next(a for a in base['assets'] if a['role']=='native_ui_embedded_icon_hd')
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'catalog.json';minimal=copy.deepcopy(base);minimal['assets']=[copy.deepcopy(a) for a in base['assets'] if a['role'] in validator.PROTOTYPE_ROLES]+[copy.deepcopy(asset)]
            path.write_text(json.dumps(minimal));validator.validate(True,path)
            for field in ['sha256','source_sha256']:
                bad=copy.deepcopy(minimal)
                if field=='sha256':bad['assets'][-1][field]='0'*64
                else:bad['assets'][-1][field]=['0'*64]
                path.write_text(json.dumps(bad))
                with self.subTest(field=field),self.assertRaisesRegex(ValueError,'(?i)sha|digest'):
                    validator.validate(True,path)


if __name__ == "__main__":
    unittest.main()
