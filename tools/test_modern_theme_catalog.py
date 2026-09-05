#!/usr/bin/env python3
"""Deterministic tests for the modern-theme prototype catalog."""
from __future__ import annotations

import unittest

import validate_modern_theme_catalog as validator


class ModernThemeCatalogTests(unittest.TestCase):
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
        self.assertEqual(sprite_groups, {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70, 71, 72, 73, 76, 77, 78, 80, 82, 83, 85, 86, 88, 90, 91})
        for group in (48, 66, 67):
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


if __name__ == "__main__":
    unittest.main()
