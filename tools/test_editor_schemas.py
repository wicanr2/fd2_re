"""不依賴第三方套件的編輯器 Schema fixture／邊界測試。"""
import json
import re
import tempfile
import unittest
from pathlib import Path

from validate_editor_documents import validate_documents

ROOT = Path(__file__).parents[1]
SCHEMA_DIR = ROOT / "docs" / "data" / "schema"
ID_RE = re.compile(r"^[a-z0-9][a-z0-9._/-]+$")


def load(name):
    return json.loads((SCHEMA_DIR / name).read_text(encoding="utf-8"))


def check_common(doc, kind):
    for key in ("schema_version", "document_id", "kind", "source", "extensions"):
        if key not in doc:
            raise ValueError(f"缺少 {key}")
    if doc["schema_version"] != 1 or doc["kind"] != kind or not ID_RE.fullmatch(doc["document_id"]):
        raise ValueError("共用欄位無效")
    if not isinstance(doc["extensions"], dict):
        raise ValueError("extensions 必須是物件")


def check_asset_refs(doc, asset_ids):
    def walk(value):
        if isinstance(value, dict):
            for key, child in value.items():
                if key.endswith("asset_id") and child not in asset_ids:
                    raise ValueError(f"不存在的 asset_id：{child}")
                if key == "asset_ids":
                    for item in child:
                        if item not in asset_ids:
                            raise ValueError(f"不存在的 asset_id：{item}")
                walk(child)
        elif isinstance(value, list):
            for child in value:
                walk(child)
    walk(doc)


class EditorSchemaTest(unittest.TestCase):
    def test_source_resource_coverage_summary(self):
        manifest_schema = load("fd2-separated-asset-pack.schema.json")
        fdfield_schema = load("fd2-fdfield-runtime-catalog.schema.json")
        fdfield_catalog = json.loads((
            ROOT / "remake" / "assets" / "maps" / "fdfield_catalog.json"
        ).read_text(encoding="utf-8"))
        schema = load("fd2-source-resource-coverage-summary.schema.json")
        summary = json.loads((
            ROOT / "docs" / "data" / "fd2-source-resource-coverage-summary.json"
        ).read_text(encoding="utf-8"))
        self.assertEqual(schema["properties"]["kind"]["const"], summary["kind"])
        self.assertEqual(manifest_schema["properties"]["schema_version"]["const"], 2)
        self.assertIn("runtime_catalogs", manifest_schema["properties"])
        self.assertIn("fdfield", manifest_schema["$defs"]["runtime_catalogs"]["properties"])
        self.assertEqual(fdfield_schema["properties"]["kind"]["const"], fdfield_catalog["kind"])
        self.assertEqual(fdfield_catalog["resources"][0]["resource_index"], 69)
        self.assertIn("source_resources", manifest_schema["required"])
        self.assertEqual(summary["manifest_schema_version"], 2)
        self.assertRegex(summary["manifest_sha256"], r"^[0-9a-f]{64}$")
        self.assertEqual(sum(summary["dispositions"].values()), summary["total_resources"])
        self.assertEqual(sum(row["total"] for row in summary["sources"]), summary["total_resources"])
        for row in summary["sources"]:
            self.assertEqual(sum(row["dispositions"].values()), row["total"])
            self.assertEqual(len(row["confirmed_empty_resources"]), row["dispositions"]["confirmed_empty"])
            self.assertEqual(len(row["blocked_resources"]), row["dispositions"]["blocked"])
            self.assertEqual(len(row["unknown_resources"]), row["dispositions"]["unknown"])

    def test_diagnostic_string_inventory_summary(self):
        full_schema = load("fd2-string-inventory.schema.json")
        summary_schema = load("fd2-string-inventory-summary.schema.json")
        summary = json.loads(
            (ROOT / "docs" / "data" / "fd2-string-inventory-summary.json").read_text(encoding="utf-8")
        )
        self.assertEqual(full_schema["properties"]["kind"]["const"], "fd2_string_inventory")
        self.assertEqual(summary_schema["properties"]["kind"]["const"], summary["kind"])
        self.assertEqual(summary["locale"], "zh-Hant")
        self.assertEqual(summary["status"], "diagnostic")
        self.assertRegex(summary["inventory_sha256"], r"^[0-9a-f]{64}$")
        self.assertLessEqual(summary["unique_text_count"], summary["entry_count"])
        self.assertLessEqual(summary["variable_entry_count"], summary["entry_count"])
        for dimension in ("by_id_status", "by_role", "by_confidence"):
            self.assertEqual(sum(summary[dimension].values()), summary["entry_count"])
        self.assertEqual(set(summary["by_role_unique_text"]), set(summary["by_role"]))

        review_schema = load("fd2-string-review.schema.json")
        review = json.loads((ROOT / "docs" / "data" / "fd2-string-review.json").read_text(encoding="utf-8"))
        self.assertEqual(review_schema["properties"]["kind"]["const"], review["kind"])
        self.assertEqual(review["inventory_sha256"], summary["inventory_sha256"])
        counts = {name: len(group["string_ids"]) for name, group in review["dispositions"].items()}
        self.assertEqual(counts, {"player_visible": 30, "internal_diagnostic": 42, "development": 3, "unknown": 4})
        all_ids = [item for group in review["dispositions"].values() for item in group["string_ids"]]
        self.assertEqual(len(all_ids), len(set(all_ids)))

    def test_four_schema_documents_and_extensions(self):
        docs = [
            ("fd2-campaign.schema.json", {"kind": "campaign", "nodes": []}),
            ("fd2-scenario.schema.json", {"kind": "scenario", "scenario_id": "sc-1", "map_id": "map-1", "units": [], "events": []}),
            ("fd2-story.schema.json", {"kind": "story", "scenes": []}),
            ("fd2-animation.schema.json", {"kind": "animation", "animation_id": "anim-1", "frames": [{"frame_id": "f-1", "asset_id": "portrait/a", "delay_ms": 0, "x": 0, "y": 0, "extensions": {}}]}),
        ]
        for filename, body in docs:
            schema = load(filename)
            self.assertEqual(schema["properties"]["schema_version"]["const"], 1)
            doc = {"schema_version": 1, "document_id": "doc-1", "kind": body.pop("kind"), "source": {"evidence": "test"}, "extensions": {"future": {"x": 1}}, **body}
            check_common(doc, schema["properties"]["kind"]["const"])
            check_asset_refs(doc, {"portrait/a"})

    def test_duplicate_and_missing_asset_rejected(self):
        doc = {"schema_version": 1, "document_id": "doc-1", "kind": "animation", "source": {}, "animation_id": "a", "frames": [], "extensions": {}, "asset_ids": ["missing"]}
        with self.assertRaises(ValueError):
            check_asset_refs(doc, set())

    def test_unknown_top_level_requires_extensions_boundary(self):
        schema = load("fd2-story.schema.json")
        self.assertFalse(schema["additionalProperties"])
        self.assertTrue(schema["properties"]["extensions"]["additionalProperties"])

    def test_cross_document_ids_and_references(self):
        common = {"schema_version": 1, "source": {}, "extensions": {}}
        docs = [
            {**common, "document_id": "campaign/main", "kind": "campaign", "nodes": [
                {"node_id": "node/start", "type": "story", "next": "node/end", "extensions": {}},
                {"node_id": "node/end", "type": "ending", "extensions": {}},
            ]},
            {**common, "document_id": "animation/talk", "kind": "animation", "animation_id": "anim/talk", "frames": [
                {"frame_id": "frame/0", "asset_id": "portrait/a", "delay_ms": 0, "x": 0, "y": 0, "extensions": {}},
            ]},
            {**common, "document_id": "story/intro", "kind": "story", "scenes": [
                {"scene_id": "scene/intro", "beats": [], "extensions": {}, "lines": [
                    {"line_id": "line/1", "speaker": "character/sol", "text": "測試", "portrait_asset_id": "portrait/a", "mouth_animation_id": "anim/talk", "extensions": {}},
                ]},
            ]},
        ]
        validate_documents(docs, {"portrait/a"})
        docs[0]["nodes"][0]["next"] = "node/missing"
        with self.assertRaisesRegex(ValueError, "不存在節點"):
            validate_documents(docs, {"portrait/a"})

    def test_duplicate_stable_id_rejected(self):
        doc = {"schema_version": 1, "document_id": "scenario/1", "kind": "scenario", "source": {}, "extensions": {},
               "scenario_id": "scenario/1", "map_id": "map/1", "events": [], "units": [
                   {"unit_id": "unit/1", "character_id": "character/a", "extensions": {}},
                   {"unit_id": "unit/1", "character_id": "character/b", "extensions": {}},
               ]}
        with self.assertRaisesRegex(ValueError, "重複"):
            validate_documents([doc], set())


if __name__ == "__main__":
    unittest.main()
