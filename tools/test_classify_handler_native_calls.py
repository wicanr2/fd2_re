import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from classify_handler_native_calls import classify_document, render_minimal_changes


class ClassifyHandlerNativeCallsTest(unittest.TestCase):
    def test_classifies_only_evidence_indexed_target(self):
        document = {
            "beats": [
                {
                    "op": "unknown", "native_target": "0x11df2",
                    "raw_args": [0, 255, 0], "source": {"addr": "0x100"},
                },
                {
                    "op": "unknown", "native_target": "0x22253",
                    "raw_args": [], "source": {"addr": "0x101"},
                },
            ],
            "diagnostics": {"unknown_ops": 2},
        }
        changed, unknown, counts, unresolved = classify_document(document)
        self.assertEqual(changed, 2)
        self.assertEqual(unknown, 0)
        self.assertEqual(counts["0x11df2"], 1)
        self.assertEqual(document["beats"][0]["op"], "native_call")
        self.assertEqual(document["beats"][0]["native_semantic"], "native_palette_update")
        self.assertEqual(document["beats"][0]["native_confidence"], "已證實")
        self.assertTrue(document["beats"][0]["native_evidence"])
        self.assertEqual(document["beats"][0]["raw_args"], [0, 255, 0])
        self.assertEqual(unresolved["0x22253"], 1)
        self.assertEqual(document["beats"][1]["op"], "unresolved_native_call")

    def test_walks_branches_and_is_idempotent(self):
        document = {
            "beats": [{
                "op": "if", "then": [{
                    "op": "unknown", "native_target": "0x13536", "source": {},
                }], "else": [],
            }],
        }
        self.assertEqual(classify_document(document)[:2], (1, 0))
        snapshot = json.dumps(document, sort_keys=True)
        self.assertEqual(classify_document(document)[:2], (0, 0))
        self.assertEqual(json.dumps(document, sort_keys=True), snapshot)

    def test_minimal_renderer_preserves_unrelated_compact_layout(self):
        original = '''{
  "beats": [
    {
      "op": "unknown",
      "native_target": "0x11df2",
      "raw_args": [0, 255, 0]
    },
    { "op": "direct_record_patch", "units": [{ "slot": 1, "x": 2 }] }
  ],
  "diagnostics": {
    "unknown_ops": 1
  }
}
'''
        document = json.loads(original)
        changed, _, _, _ = classify_document(document)
        rendered, replacements = render_minimal_changes(original, document)
        self.assertEqual(changed, replacements)
        self.assertIn('{ "op": "direct_record_patch", "units": [{ "slot": 1, "x": 2 }] }', rendered)
        self.assertIn('"native_semantic": "native_palette_update"', rendered)
        self.assertIn('"native_confidence": "已證實"', rendered)
        self.assertEqual(json.loads(rendered), document)


if __name__ == "__main__":
    unittest.main()
