#!/usr/bin/env python3
"""驗證 FD2 canonical 編輯文件的穩定身份與跨檔引用。

JSON Schema 負責單檔形狀；本工具補上 Schema 無法表達的唯一性、戰役轉場、
動畫與分離素材 asset_id 關聯。未知遊戲語意不會在此被猜測補值。
"""

import argparse
import json
import re
from pathlib import Path

ID_RE = re.compile(r"^[a-z0-9][a-z0-9._/-]+$")


def _require_id(value, label):
    if not isinstance(value, str) or not ID_RE.fullmatch(value):
        raise ValueError(f"{label} 不是合法穩定 ID：{value!r}")


def _unique(items, key, label):
    seen = set()
    for item in items:
        value = item.get(key)
        _require_id(value, f"{label}.{key}")
        if value in seen:
            raise ValueError(f"重複的 {label}.{key}：{value}")
        seen.add(value)
    return seen


def _walk_asset_refs(value):
    if isinstance(value, dict):
        for key, child in value.items():
            if key.endswith("asset_id") and isinstance(child, str):
                yield child
            elif key == "asset_ids" and isinstance(child, list):
                yield from child
            yield from _walk_asset_refs(child)
    elif isinstance(value, list):
        for child in value:
            yield from _walk_asset_refs(child)


def validate_documents(documents, asset_ids):
    document_ids = set()
    animation_ids = set()
    for document in documents:
        for key in ("schema_version", "document_id", "kind", "source", "extensions"):
            if key not in document:
                raise ValueError(f"文件缺少 {key}")
        if document["schema_version"] != 1 or not isinstance(document["extensions"], dict):
            raise ValueError(f"文件 {document.get('document_id')} 共用契約無效")
        document_id = document["document_id"]
        _require_id(document_id, "document_id")
        if document_id in document_ids:
            raise ValueError(f"重複的 document_id：{document_id}")
        document_ids.add(document_id)
        if document["kind"] == "animation":
            animation_ids.add(document.get("animation_id"))

    for document in documents:
        kind = document["kind"]
        if kind == "campaign":
            nodes = document.get("nodes", [])
            node_ids = _unique(nodes, "node_id", "campaign.nodes")
            for node in nodes:
                for key in ("next", "on_win", "on_lose"):
                    target = node.get(key)
                    if target is not None and target not in node_ids:
                        raise ValueError(f"節點 {node['node_id']} 的 {key} 指向不存在節點：{target}")
        elif kind == "scenario":
            _unique(document.get("units", []), "unit_id", "scenario.units")
            events = document.get("events", [])
            _unique(events, "event_id", "scenario.events")
            actions = [action for event in events for action in event.get("actions", [])]
            _unique(actions, "action_id", "scenario.actions")
        elif kind == "story":
            scenes = document.get("scenes", [])
            _unique(scenes, "scene_id", "story.scenes")
            lines = [line for scene in scenes for line in scene.get("lines", [])]
            _unique(lines, "line_id", "story.lines")
            for line in lines:
                animation_id = line.get("mouth_animation_id")
                if animation_id is not None and animation_id not in animation_ids:
                    raise ValueError(f"台詞 {line['line_id']} 引用不存在動畫：{animation_id}")
        elif kind == "animation":
            _require_id(document.get("animation_id"), "animation_id")
            _unique(document.get("frames", []), "frame_id", "animation.frames")
        else:
            raise ValueError(f"未知編輯文件 kind：{kind}")

        for asset_id in _walk_asset_refs(document):
            _require_id(asset_id, "asset_id")
            if asset_id not in asset_ids:
                raise ValueError(f"文件 {document['document_id']} 引用不存在素材：{asset_id}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", required=True)
    parser.add_argument("documents", nargs="+")
    args = parser.parse_args()
    manifest = json.loads(Path(args.manifest).read_text(encoding="utf-8"))
    asset_ids = {item["asset_id"] for item in manifest.get("assets", [])}
    documents = [json.loads(Path(path).read_text(encoding="utf-8")) for path in args.documents]
    validate_documents(documents, asset_ids)
    print(f"通過：{len(documents)} 份編輯文件，{len(asset_ids)} 個素材 ID")


if __name__ == "__main__":
    main()
