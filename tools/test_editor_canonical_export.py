"""全戰役 canonical 批次匯出的不覆蓋與確定性測試。"""

from __future__ import annotations

import hashlib
import json
import tempfile
import unittest
from pathlib import Path

from export_editor_canonical import (
    build_canonical_bundle,
    build_character_identity_catalog,
    validate_character_identity_catalog,
)


class CanonicalExportTest(unittest.TestCase):
    root = Path(__file__).parents[1]

    def test_full_campaign_batch_is_deterministic_and_keeps_legacy(self):
        legacy = self.root / "remake" / "assets" / "scenarios" / "campaign_full.json"
        before = hashlib.sha256(legacy.read_bytes()).hexdigest()
        with tempfile.TemporaryDirectory() as first_dir, tempfile.TemporaryDirectory() as second_dir:
            # 受版控測試不得依賴被 .gitignore 排除的玩家私有動畫匯出；
            # 動畫批次由顯式實檔驗證處理，乾淨 clone 仍要可重跑核心契約。
            first = build_canonical_bundle(self.root, first_dir, include_animations=False, clean=True)
            second = build_canonical_bundle(self.root, second_dir, include_animations=False, clean=True)
            self.assertEqual(first["counts"], {"animation": 0, "campaign": 1, "scenario": 30, "story": 35})
            self.assertEqual(first["counts"], second["counts"])
            self.assertEqual(
                (Path(first_dir) / "bundle-summary.json").read_bytes(),
                (Path(second_dir) / "bundle-summary.json").read_bytes(),
            )
            for entry in first["documents"]:
                relative = Path(entry["output"])
                left = Path(first_dir) / relative
                right = Path(second_dir) / relative
                self.assertEqual(left.read_bytes(), right.read_bytes(), relative)
                document = json.loads(left.read_text(encoding="utf-8"))
                self.assertEqual(document["schema_version"], 1)
                self.assertIn("document_id", document)
                self.assertIn("source", document)
                self.assertIn("extensions", document)
            catalog = json.loads((Path(first_dir) / "character-identity.json").read_text(encoding="utf-8"))
            validate_character_identity_catalog(catalog)
            self.assertTrue(catalog["characters"])
            sol = next(item for item in catalog["characters"] if item["native_identity"] == 0)
            self.assertEqual(sol["character_id"], "character/native-0")
            self.assertTrue(sol["portrait_selector_candidates"])
            self.assertTrue(sol["map_sprite_selector_candidates"])
            self.assertEqual(sol["battle_animation_selector_candidates"], [])
        self.assertEqual(hashlib.sha256(legacy.read_bytes()).hexdigest(), before)

    def test_legacy_directories_are_rejected(self):
        with self.assertRaises(ValueError):
            build_canonical_bundle(self.root, self.root / "remake" / "assets" / "story")

    def test_identity_conflict_is_diagnostic_and_does_not_select_value(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "remake/assets/scenarios").mkdir(parents=True)
            (root / "remake/assets/story").mkdir(parents=True)
            (root / "remake/assets/scenarios/campaign_full.json").write_text("{}", encoding="utf-8")
            (root / "remake/assets/scenarios/ch01.json").write_text(json.dumps({"party": [{"native_identity": 3, "name": "甲", "portrait": 3, "fig": 3}]}), encoding="utf-8")
            (root / "remake/assets/scenarios/ch02.json").write_text(json.dumps({"party": [{"native_identity": 3, "name": "乙", "portrait": 3, "fig": 3}]}), encoding="utf-8")
            (root / "remake/assets/story/ch01.json").write_text(json.dumps({"scenes": []}), encoding="utf-8")
            catalog, diagnostics = build_character_identity_catalog(root)
            self.assertEqual(len(diagnostics), 1)
            self.assertEqual(diagnostics[0]["severity"], "error")
            character = next(item for item in catalog["characters"] if item["native_identity"] == 3)
            self.assertEqual(len(character["display_name_candidates"]), 2)

    def test_identity_schema_rejects_unprovenanced_extension(self):
        catalog, _ = build_character_identity_catalog(self.root)
        catalog["characters"][0]["guessed_role"] = "hero"
        with self.assertRaisesRegex(ValueError, "character 欄位不完整"):
            validate_character_identity_catalog(catalog)


if __name__ == "__main__":
    unittest.main()
