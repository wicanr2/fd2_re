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
        self.assertEqual(sprite_groups, {0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 68})


if __name__ == "__main__":
    unittest.main()
