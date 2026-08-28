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
