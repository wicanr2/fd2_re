"""storyBG 對拍收據的正反驗證。"""

import copy
import json
import unittest
from pathlib import Path

from generate_storybg_dialogue_receipt import validate_receipt


ROOT = Path(__file__).parents[1]
RECEIPT = ROOT / "docs" / "data" / "ui-traces" / "storybg-dialogue-original-vs-remake-e1.json"
SCHEMA = ROOT / "docs" / "data" / "schema" / "fd2-storybg-dialogue-parity-receipt.schema.json"


class StoryBGDialogueReceiptTest(unittest.TestCase):
    def setUp(self):
        self.receipt = json.loads(RECEIPT.read_text(encoding="utf-8"))

    def test_schema_and_receipt_positive(self):
        schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
        self.assertEqual(schema["properties"]["kind"]["const"], self.receipt["kind"])
        self.assertEqual(schema["properties"]["evidence_level"]["const"], "RUNTIME-E1")
        self.assertFalse(schema["additionalProperties"])
        validate_receipt(self.receipt)

    def test_non_dosgolem_original_rejected(self):
        bad = copy.deepcopy(self.receipt)
        bad["original_runner"] = "dosbox-bootstrap"
        with self.assertRaisesRegex(ValueError, "dosgolem"):
            validate_receipt(bad)

    def test_unlocked_animation_phase_rejected(self):
        bad = copy.deepcopy(self.receipt)
        bad["phase_crosscheck"][0]["diff_sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "動畫相位"):
            validate_receipt(bad)

    def test_dialogue_pixel_difference_rejected(self):
        bad = copy.deepcopy(self.receipt)
        bad["image_comparison"]["regions"]["dialogue_overlay"]["equal_ratio"] = 0.999
        with self.assertRaisesRegex(ValueError, "dialogue_overlay"):
            validate_receipt(bad)

    def test_player_e2_overclaim_rejected(self):
        bad = copy.deepcopy(self.receipt)
        bad["limitations"]["player_e2_claimed"] = True
        with self.assertRaisesRegex(ValueError, "PLAYER-E2"):
            validate_receipt(bad)


if __name__ == "__main__":
    unittest.main()
