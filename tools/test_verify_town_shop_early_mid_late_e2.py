#!/usr/bin/env python3
import copy
import json
import unittest
from pathlib import Path

import verify_town_shop_early_mid_late_e2 as verifier


ROOT = Path(__file__).resolve().parents[1]
RECEIPT = ROOT / "docs/data/ui-traces/town-shop-early-mid-late-e2.json"


class TownShopReceiptSchemaTest(unittest.TestCase):
    def test_repository_receipt_schema(self):
        verifier.verify(json.loads(RECEIPT.read_text(encoding="utf-8")))

    def test_rejects_modified_oracle(self):
        receipt = json.loads(RECEIPT.read_text(encoding="utf-8"))
        receipt = copy.deepcopy(receipt)
        receipt["original"]["lock_ally_hp"] = True
        with self.assertRaises(ValueError):
            verifier.verify(receipt)

    def test_rejects_missing_late_fixture(self):
        receipt = json.loads(RECEIPT.read_text(encoding="utf-8"))
        receipt = copy.deepcopy(receipt)
        receipt["fixtures"] = [fixture for fixture in receipt["fixtures"] if fixture["name"] != "late"]
        with self.assertRaises(ValueError):
            verifier.verify(receipt)


if __name__ == "__main__":
    unittest.main()
