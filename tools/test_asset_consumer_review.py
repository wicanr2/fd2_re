import copy
import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from review_unknown_asset_consumers import review
from summarize_asset_dispositions import summarize


class AssetConsumerReviewTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.repo = Path(__file__).resolve().parents[1]
        cls.pack = cls.repo / "remake/generated-assets/fd2-original-b97caf22"
        if not (cls.pack / "manifest.json").is_file():
            raise unittest.SkipTest("需要本機私人分離素材包做整合核對")
        cls.manifest = json.loads((cls.pack / "manifest.json").read_text(encoding="utf-8"))

    def test_real_manifest_reviews_all_93_entries(self):
        ledger = review(self.manifest, self.pack, self.repo)
        self.assertEqual(ledger["reviewed_total"], 93)
        self.assertEqual(ledger["outcome_counts"], {
            "confirmed_non_playable_sentinel": 5,
            "no_registered_player_consumer": 9,
            "player_data_consumer_registered": 79,
        })
        summary = summarize(self.manifest, ledger, "docs/data/asset-consumer-review.json")
        self.assertEqual(summary["manifest_unknown_total"], 93)
        self.assertEqual(summary["reviewed_unknown_total"], 93)
        self.assertEqual(summary["unknown_remaining"], 0)
        self.assertEqual(summary["unknown"], [])

    def test_raw_identity_drift_fails_closed(self):
        broken = copy.deepcopy(self.manifest)
        item = next(e for e in broken["source_resources"] if e.get("disposition") == "unknown")
        item["raw_sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "raw SHA-256 漂移"):
            review(broken, self.pack, self.repo)

    def test_review_must_cover_exact_unknown_set(self):
        ledger = review(self.manifest, self.pack, self.repo)
        ledger["entries"].pop()
        with self.assertRaisesRegex(ValueError, "逐筆且唯一覆蓋"):
            summarize(self.manifest, ledger)

    def test_music_sentinel_is_exact(self):
        ledger = review(self.manifest, self.pack, self.repo)
        rows = [e for e in ledger["entries"] if e["source_file"] == "FDMUS.DAT"]
        self.assertEqual({e["source_resource"] for e in rows}, {0, 2, 5, 7, 9})
        self.assertTrue(all(e["raw_hex"] == "20 0d 0a" for e in rows))


if __name__ == "__main__":
    unittest.main()
