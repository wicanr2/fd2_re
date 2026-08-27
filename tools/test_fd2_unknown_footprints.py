import json
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
INVENTORY = ROOT / "docs/data/ida/fd2_function_inventory.json"
REPORT = ROOT / "docs/data/ida/fd2_unknown_footprints.json"


class UnknownFootprintsTest(unittest.TestCase):
    def test_checked_in_report_matches_current_unknown_inventory(self):
        inventory = json.loads(INVENTORY.read_text(encoding="utf-8"))
        report = json.loads(REPORT.read_text(encoding="utf-8"))
        unknown = {
            item["start"]
            for item in inventory["functions"]
            if item["classification"]["value"] == "unknown"
        }
        self.assertEqual(report["input"], inventory["input"])
        self.assertEqual(report["unknown_function_count"], len(unknown))
        self.assertEqual({item["start"] for item in report["entries"]}, unknown)
        self.assertEqual(
            sum(report["review_priority_counts"].values()),
            report["unknown_function_count"],
        )

    def test_report_never_treats_itself_as_evidence(self):
        report = json.loads(REPORT.read_text(encoding="utf-8"))
        for entry in report["entries"]:
            paths = {
                hit["path"]
                for field in ("exact_start_hits", "range_hits")
                for hit in entry[field]
            }
            self.assertNotIn("docs/data/ida/fd2_unknown_footprints.json", paths)


if __name__ == "__main__":
    unittest.main()
