import copy
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from fd2_semantic_index import (
    build_compact_report,
    load_semantic_index,
    validate_input_identity,
)


ROOT = Path(__file__).resolve().parents[1]
INDEX = ROOT / "docs/data/ida/fd2_semantic_index.json"
INVENTORY = ROOT / "docs/data/ida/fd2_function_inventory.json"


class SemanticIndexTest(unittest.TestCase):
    def test_checked_in_index_is_well_formed_and_evidence_exists(self):
        document, entries = load_semantic_index(INDEX, ROOT)
        self.assertEqual(document["input"]["file"], "FD2.EXE")
        self.assertGreater(len(entries), 0)
        self.assertIn(0x13FD4, entries)
        self.assertIn(0x2BCE5, entries)

    def test_checked_in_inventory_matches_index_identity_and_annotations(self):
        document, entries = load_semantic_index(INDEX, ROOT)
        inventory = json.loads(INVENTORY.read_text(encoding="utf-8"))
        validate_input_identity(document["input"], inventory["input"])
        self.assertEqual(inventory["function_count"], 1305)
        self.assertEqual(
            inventory["classification_counts"],
            {"product": 58, "runtime": 171, "unknown": 1076},
        )
        self.assertEqual(inventory["semantic_annotation_count"], len(entries))

        functions = {int(item["start"], 0): item for item in inventory["functions"]}
        for address, annotations in entries.items():
            self.assertIn(address, functions, hex(address))
            self.assertEqual(functions[address]["semantic_annotations"], annotations, hex(address))

    def test_checked_in_index_is_sorted_by_linear_address(self):
        document = json.loads(INDEX.read_text(encoding="utf-8"))
        addresses = [int(entry["address"], 0) for entry in document["entries"]]
        self.assertEqual(addresses, sorted(addresses))

    def test_input_identity_rejects_different_hash(self):
        document, _ = load_semantic_index(INDEX, ROOT)
        actual = copy.deepcopy(document["input"])
        actual["md5"] = "0" * 32
        with self.assertRaisesRegex(ValueError, "input mismatch"):
            validate_input_identity(document["input"], actual)

    def test_duplicate_address_is_rejected(self):
        document = json.loads(INDEX.read_text(encoding="utf-8"))
        document["entries"].append(copy.deepcopy(document["entries"][0]))
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "index.json"
            path.write_text(json.dumps(document), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "duplicate semantic address"):
                load_semantic_index(path, ROOT)

    def test_unclassified_entry_is_rejected(self):
        document = json.loads(INDEX.read_text(encoding="utf-8"))
        document["entries"][0].pop("classification")
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "index.json"
            path.write_text(json.dumps(document), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "classification"):
                load_semantic_index(path, ROOT)

    def test_compact_report_keeps_bounds_callers_and_annotations(self):
        inventory = {
            "schema_version": 1,
            "tool": {"name": "IDA Pro"},
            "input": {"file": "FD2.EXE"},
            "imagebase": "0x0",
            "function_count": 1,
            "classification_counts": {"product": 1},
            "semantic_annotation_count": 1,
            "classification_note": "test",
            "functions": [{
                "start": "0x10000",
                "end": "0x10010",
                "size": 16,
                "ida_analysis_name": "sub_10000",
                "ida_function_flags": [],
                "code_xrefs_to_start": [
                    {"from": "0x20001", "caller_function": {"start": "0x20000"}},
                    {"from": "0x20005", "caller_function": {"start": "0x20000"}},
                ],
                "data_xrefs_to_start": ["0x50000"],
                "classification": {
                    "value": "product", "confidence": "已證實", "source": "test",
                },
                "semantic_annotations": [{"semantic": "test"}],
            }],
        }
        report = build_compact_report(inventory)
        self.assertEqual(report["function_count"], 1)
        self.assertEqual(report["functions"][0]["direct_caller_functions"], ["0x20000"])
        self.assertEqual(report["functions"][0]["direct_code_xref_count"], 2)
        self.assertEqual(report["functions"][0]["direct_data_xref_count"], 1)

    def test_compact_report_rejects_inconsistent_bounds(self):
        inventory = {
            "schema_version": 1,
            "function_count": 1,
            "classification_counts": {"unknown": 1},
            "functions": [{
                "start": "0x10000", "end": "0x10010", "size": 15,
                "classification": {"value": "unknown"},
            }],
        }
        with self.assertRaisesRegex(ValueError, "inconsistent bounds"):
            build_compact_report(inventory)


if __name__ == "__main__":
    unittest.main()
