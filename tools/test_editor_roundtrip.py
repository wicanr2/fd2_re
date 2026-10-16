#!/usr/bin/env python3
"""legacy importer 與 deterministic writer 的最小實際資料回歸。"""

import json
import copy
import tempfile
import unittest
from pathlib import Path

from import_editor_legacy import import_legacy, write_canonical
from validate_editor_documents import validate_documents


ROOT = Path(__file__).resolve().parents[1]


class EditorRoundTripTests(unittest.TestCase):
    def test_campaign_reimport_preserves_existing_ids_after_delete_and_insert(self):
        source = "legacy/campaign_full.json"
        raw = {"nodes": {"start": {"type": "story", "next": "end"},
                         "removed": {"type": "story"}, "end": {"type": "ending"}}}
        previous, _ = import_legacy(raw, source, "campaign")
        old_ids = {n["extensions"]["legacy_key"]: n["node_id"] for n in previous["nodes"]}
        for insert in (False, True):
            nodes = {"extra": {"type": "story", "next": "start"}} if insert else {}
            nodes.update({"start": raw["nodes"]["start"], "end": raw["nodes"]["end"]})
            document, _ = import_legacy({"nodes": nodes}, source, "campaign", previous_document=previous)
            by_key = {n["extensions"]["legacy_key"]: n for n in document["nodes"]}
            for key in ("start", "end"):
                self.assertEqual(by_key[key]["node_id"], old_ids[key])
            self.assertEqual(by_key["start"]["next"], old_ids["end"])
            self.assertNotIn("removed", by_key)
            if insert:
                self.assertEqual(by_key["extra"]["next"], old_ids["start"])
            validate_documents([document], set())

    def test_campaign_id_preservation_rejects_wrong_source_and_ambiguous_ids(self):
        raw = {"nodes": {"a": {"type": "story"}, "b": {"type": "ending"}}}
        previous, _ = import_legacy(raw, "legacy/campaign_full.json", "campaign")
        for mutation in ("source", "key", "id", "kind"):
            bad = copy.deepcopy(previous)
            if mutation == "source":
                bad["document_id"] = "another/document"
            elif mutation == "key":
                bad["nodes"][1]["extensions"]["legacy_key"] = "a"
            elif mutation == "id":
                bad["nodes"][1]["node_id"] = bad["nodes"][0]["node_id"]
            else:
                bad["kind"] = "scenario"
            with self.assertRaises(ValueError):
                import_legacy(raw, "legacy/campaign_full.json", "campaign", previous_document=bad)
        with self.assertRaises(ValueError):
            import_legacy({}, "legacy/ch01.json", "scenario", previous_document=previous)

    def _roundtrip(self, raw, source, kind):
        document, diagnostics = import_legacy(raw, source, kind)
        self.assertIsInstance(diagnostics, list)
        first = write_canonical(document)
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp) / "canonical.json"
            target.write_text(first, encoding="utf-8")
            loaded = json.loads(target.read_text(encoding="utf-8"))
            second = write_canonical(loaded)
        self.assertEqual(first, second)
        self.assertEqual(document["source"], loaded["source"])
        self.assertEqual(document["extensions"], loaded["extensions"])
        return document

    def test_campaign_nodes_object_and_unknown_preserved(self):
        raw = {"title": "顯示文字", "start": "n", "flags": {"x": True}, "nodes": {"n": {"type": "battle", "scenario": "ch01", "next": "end", "future": {"v": 1}}, "end": {"type": "ending"}}}
        doc = self._roundtrip(raw, "legacy/campaign_full.json", "campaign")
        self.assertEqual(doc["extensions"]["legacy"]["title"], "顯示文字")
        self.assertEqual(doc["extensions"]["legacy"]["flags"], {"x": True})
        self.assertEqual(doc["nodes"][0]["extensions"]["legacy"]["future"], {"v": 1})
        self.assertTrue(doc["nodes"][0]["node_id"].endswith("/n/0"))
        self.assertEqual(doc["nodes"][0]["next"], doc["nodes"][1]["node_id"])

        real_path = ROOT / "remake/assets/scenarios/campaign_full.json"
        real, diagnostics = import_legacy(
            json.loads(real_path.read_text(encoding="utf-8")),
            "remake/assets/scenarios/campaign_full.json",
            "campaign",
        )
        self.assertGreater(len(real["nodes"]), 30)
        self.assertFalse([item for item in diagnostics if item["severity"] == "error"])
        validate_documents([real], set())

    def test_real_ch01_scenario_and_story(self):
        scenario_path = ROOT / "remake/assets/scenarios/ch01.json"
        story_path = ROOT / "remake/assets/story/ch01.json"
        scenario = json.loads(scenario_path.read_text(encoding="utf-8"))
        story = json.loads(story_path.read_text(encoding="utf-8"))
        sdoc = self._roundtrip(scenario, "remake/assets/scenarios/ch01.json", "scenario")
        tdoc = self._roundtrip(story, "remake/assets/story/ch01.json", "story")
        self.assertGreater(len(sdoc["units"]), 0); self.assertGreater(len(tdoc["scenes"]), 0)
        self.assertGreater(len(sdoc["events"][0]["actions"]), 0)
        self.assertEqual(tdoc["scenes"][0]["lines"][0]["speaker"], "character/native-0")
        validate_documents([sdoc, tdoc], set())
        ids_before = [line["line_id"] for scene in tdoc["scenes"] for line in scene["lines"]]
        story["scenes"][0]["lines"][0]["speaker_name"] = "改名不應影響 ID"
        changed, _ = import_legacy(story, "remake/assets/story/ch01.json", "story")
        self.assertEqual(ids_before, [line["line_id"] for scene in changed["scenes"] for line in scene["lines"]])

    def test_generated_afm_metadata_is_preserved(self):
        path = ROOT / "remake/generated-assets/fd2-original-b97caf22/animations/FIGANI_004/animation.json"
        raw = json.loads(path.read_text(encoding="utf-8"))
        doc, diagnostics = import_legacy(raw, str(path), "animation")
        self.assertEqual(doc, raw)
        self.assertEqual(diagnostics, [])
        self.assertEqual(write_canonical(doc), write_canonical(json.loads(write_canonical(raw))))


if __name__ == "__main__": unittest.main()
