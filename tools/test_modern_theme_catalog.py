#!/usr/bin/env python3
"""Deterministic tests for the modern-theme prototype catalog."""
from __future__ import annotations

import copy
import hashlib
import json
from pathlib import Path
import tempfile
import unittest

import validate_modern_theme_catalog as validator


class ModernThemeCatalogTests(unittest.TestCase):
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
            good=copy.deepcopy(base);good['assets'].append(asset)
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
        candidate = next(asset for asset in catalog["assets"] if asset["role"] == "story_portrait_frame")
        self.assertEqual(
            (candidate["width"], candidate["height"], candidate["speaker_id"], candidate["frame"]),
            (80, 80, 0, 0),
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
            (1, 12, "binary", "fdicon-001-style-a-v1-master.png"),
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
        selector40 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 40)
        selector58 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 58)
        self.assertEqual(selector58["frame_sha256"], selector40["frame_sha256"])
        selector41 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 41)
        selector59 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 59)
        self.assertEqual(selector59["frame_sha256"], selector41["frame_sha256"])
        selector42 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 42)
        selector60 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 60)
        self.assertEqual(selector60["frame_sha256"], selector42["frame_sha256"])
        selector43 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 43)
        selector61 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 61)
        self.assertEqual(selector61["frame_sha256"], selector43["frame_sha256"])
        selector44 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 44)
        selector62 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 62)
        self.assertEqual(selector62["frame_sha256"], selector44["frame_sha256"])
        selector46 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 46)
        selector64 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 64)
        self.assertEqual(selector64["frame_sha256"], selector46["frame_sha256"])
        selector47 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 47)
        selector65 = next(asset for asset in catalog["assets"] if asset.get("source_group") == 65)
        self.assertEqual(selector65["frame_sha256"], selector47["frame_sha256"])
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


if __name__ == "__main__":
    unittest.main()
