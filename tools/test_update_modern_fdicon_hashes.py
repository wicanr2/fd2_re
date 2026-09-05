#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from PIL import Image

import update_modern_fdicon_hashes as updater


class UpdateModernFdiconHashesTests(unittest.TestCase):
    def make_fixture(self, *, alpha: int = 255, frame_count: int = 12) -> tuple[Path, Path, str]:
        root = Path(tempfile.mkdtemp())
        asset_root = root / "assets"
        asset_root.mkdir()
        files = [f"fdicon-004-style-a-f{i:02d}.png" for i in range(frame_count)]
        for i, name in enumerate(files):
            image = Image.new("RGBA", (24, 24), (i, 2, 3, alpha))
            image.save(asset_root / name)
        catalog = root / "catalog.json"
        catalog.write_text(json.dumps({
            "schema_version": 1,
            "assets": [{
                "asset_id": "modern.fdicon.group_004.style_a",
                "role": "map_sprite_set",
                "files": files,
                "width": 24,
                "height": 24,
                "frame_count": frame_count,
                "consumer_contract": (
                    "fdicon_static_sprite_3x24_v1" if frame_count == 3
                    else "fdicon_map_sprite_12x24_v1"
                ),
                "frame_sha256": ["0" * 64] * 12,
            }],
            "trailing": "must remain",
        }, indent=2) + "\n", encoding="utf-8")
        return catalog, asset_root, hashlib.sha256(catalog.read_bytes()).hexdigest()

    def test_updates_only_target_array(self):
        catalog, asset_root, before = self.make_fixture()
        original = catalog.read_text()
        updater.update(catalog, asset_root, [4])
        updated = catalog.read_text()
        self.assertNotEqual(before, hashlib.sha256(updated.encode()).hexdigest())
        data = json.loads(updated)
        expected = [hashlib.sha256((asset_root / name).read_bytes()).hexdigest()
                    for name in data["assets"][0]["files"]]
        self.assertEqual(data["assets"][0]["frame_sha256"], expected)
        self.assertEqual(data["trailing"], "must remain")
        self.assertEqual(updated.count('"asset_id": "modern.fdicon.group_004.style_a"'), 1)
        old_start = original.index("[", original.index('"frame_sha256"'))
        old_end = original.index("]", old_start) + 1
        new_start = updated.index("[", updated.index('"frame_sha256"'))
        new_end = updated.index("]", new_start) + 1
        self.assertEqual(original[:old_start], updated[:new_start])
        self.assertEqual(original[old_end:], updated[new_end:])

    def test_duplicate_group_fails_without_writing(self):
        catalog, asset_root, before = self.make_fixture()
        with self.assertRaises(ValueError):
            updater.update(catalog, asset_root, [4, 4])
        self.assertEqual(hashlib.sha256(catalog.read_bytes()).hexdigest(), before)

    def test_updates_static_three_phase_group(self):
        catalog, asset_root, _ = self.make_fixture(frame_count=3)
        updater.update(catalog, asset_root, [4])
        data = json.loads(catalog.read_text())
        expected = [hashlib.sha256((asset_root / name).read_bytes()).hexdigest()
                    for name in data["assets"][0]["files"]]
        self.assertEqual(data["assets"][0]["frame_sha256"], expected)

    def test_missing_frame_fails_without_writing(self):
        catalog, asset_root, before = self.make_fixture()
        (asset_root / "fdicon-004-style-a-f11.png").unlink()
        with self.assertRaises(ValueError):
            updater.update(catalog, asset_root, [4])
        self.assertEqual(hashlib.sha256(catalog.read_bytes()).hexdigest(), before)

    def test_non_binary_alpha_fails_without_writing(self):
        catalog, asset_root, before = self.make_fixture()
        image = Image.new("RGBA", (24, 24), (1, 2, 3, 127))
        image.save(asset_root / "fdicon-004-style-a-f11.png")
        with self.assertRaises(ValueError):
            updater.update(catalog, asset_root, [4])
        self.assertEqual(hashlib.sha256(catalog.read_bytes()).hexdigest(), before)


if __name__ == "__main__":
    unittest.main()
