#!/usr/bin/env python3
"""將舊版編輯 JSON 匯入 FD2 canonical 文件。

此工具只做有證據的欄位映射。無法映射的值不會被丟棄，而是原樣保留在
``extensions.legacy``；穩定身份只由來源路徑、legacy key 與陣列索引組成。
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import re
from pathlib import Path
from typing import Any

IMPORTER_VERSION = "fd2-editor-legacy-importer/1.0"
KINDS = {"campaign", "scenario", "story", "animation"}


def _path(value: str | Path) -> str:
    return Path(value).as_posix()


def _id(kind: str, source: str, key: str, index: int | None = None) -> str:
    """產生不依賴顯示文字的 ID；digest 也避免絕對路徑造成非法字元。"""
    digest = hashlib.sha256(source.encode("utf-8")).hexdigest()[:12]
    token = re.sub(r"[^a-z0-9._/-]+", "-", str(key).lower()).strip("-/.") or "item"
    suffix = f"/{index}" if index is not None else ""
    return f"legacy/{kind}/{digest}/{token}{suffix}"


def _source(source: str) -> dict[str, str]:
    return {"path": _path(source), "importer_version": IMPORTER_VERSION}


def _legacy_ref(kind: str, value: Any) -> str:
    """只按舊檔中可直接觀察的路徑／數值建立跨文件 ID。"""
    token = Path(str(value)).stem.lower()
    token = re.sub(r"[^a-z0-9._/-]+", "-", token).strip("-/. ") or "unknown"
    return f"{kind}/{token}"


def _diag(code: str, path: str, message: str, severity: str = "warning") -> dict[str, str]:
    return {"code": code, "path": path, "message": message, "severity": severity}


def _unknown(raw: dict[str, Any], known: set[str], where: str, diagnostics: list[dict[str, str]]) -> dict[str, Any]:
    extra = {k: copy.deepcopy(v) for k, v in raw.items() if k not in known}
    if extra:
        diagnostics.append(_diag("unmapped_field", where, "欄位已完整保留於 extensions.legacy"))
    return extra


def _base(kind: str, source: str) -> dict[str, Any]:
    return {
        "schema_version": 1,
        "document_id": _id(kind, source, Path(source).stem or kind),
        "kind": kind,
        "source": _source(source),
        "extensions": {},
    }


def _campaign(raw: dict[str, Any], source: str, diagnostics: list[dict[str, str]]) -> dict[str, Any]:
    doc = _base("campaign", source)
    known = {"nodes"}
    legacy = _unknown(raw, known, "$", diagnostics)
    nodes: list[dict[str, Any]] = []
    table = raw.get("nodes", {})
    if isinstance(table, dict):
        entries = list(table.items())
    elif isinstance(table, list):
        entries = [(str(i), item) for i, item in enumerate(table)]
        diagnostics.append(_diag("legacy_nodes_array", "nodes", "legacy nodes 陣列已按索引匯入"))
    else:
        entries = []
        diagnostics.append(_diag("invalid_nodes", "nodes", "nodes 不是物件或陣列", "error"))
    fields = {"type", "map", "scenario", "story", "next", "on_win", "on_lose", "asset_ids"}
    node_ids = {str(key): _id("node", source, key, index) for index, (key, _value) in enumerate(entries)}
    for index, (key, value) in enumerate(entries):
        if not isinstance(value, dict):
            diagnostics.append(_diag("invalid_node", f"nodes.{key}", "節點不是物件", "error"))
            continue
        node = {"node_id": _id("node", source, key, index), "type": value.get("type", "event"), "extensions": {}}
        for old, new in (("map", "map_id"), ("scenario", "scenario_id"), ("story", "story_id")):
            if old in value:
                node[new] = _legacy_ref(old, value[old])
        for field in ("next", "on_win", "on_lose", "asset_ids"):
            if field in value:
                if field != "asset_ids" and isinstance(value[field], str) and value[field] in node_ids:
                    node[field] = node_ids[value[field]]
                else:
                    node[field] = copy.deepcopy(value[field])
                    if field != "asset_ids" and isinstance(value[field], str):
                        diagnostics.append(_diag("unresolved_xref", f"nodes.{key}.{field}", "找不到 legacy 節點目標，保留原值"))
        extra = _unknown(value, fields, f"nodes.{key}", diagnostics)
        if extra:
            node["extensions"]["legacy"] = extra
        node["extensions"]["legacy_key"] = key
        nodes.append(node)
    doc["nodes"] = nodes
    if legacy:
        doc["extensions"]["legacy"] = legacy
    return doc


def _scenario(raw: dict[str, Any], source: str, diagnostics: list[dict[str, str]]) -> dict[str, Any]:
    doc = _base("scenario", source)
    doc["scenario_id"] = _legacy_ref("scenario", Path(source).stem)
    doc["map_id"] = _legacy_ref("map", raw.get("map", "unknown"))
    doc["units"], doc["events"] = [], []
    known_top = {"chapter", "map", "party", "events"}
    legacy = _unknown(raw, known_top, "$", diagnostics)
    for index, value in enumerate(raw.get("party", []) if isinstance(raw.get("party", []), list) else []):
        if not isinstance(value, dict):
            diagnostics.append(_diag("invalid_unit", f"party[{index}]", "單位不是物件", "error"))
            continue
        native_identity = value.get("native_identity")
        if isinstance(native_identity, int) and native_identity >= 0:
            character_id = f"character/native-{native_identity}"
        else:
            character_id = _id("character", source, "party", index)
        unit = {"unit_id": _id("unit", source, "party", index), "character_id": character_id, "extensions": {}}
        for field in ("camp", "x", "y"):
            if field in value and isinstance(value[field], (str, int)):
                unit[field] = value[field]
        extra = _unknown(value, {"camp", "x", "y"}, f"party[{index}]", diagnostics)
        if extra:
            unit["extensions"]["legacy"] = extra
        doc["units"].append(unit)
    for index, value in enumerate(raw.get("events", []) if isinstance(raw.get("events", []), list) else []):
        if not isinstance(value, dict):
            diagnostics.append(_diag("invalid_event", f"events[{index}]", "事件不是物件", "error"))
            continue
        event = {"event_id": _id("event", source, "events", index), "trigger": str(value.get("trigger", "legacy")), "actions": [], "extensions": {}}
        if isinstance(value.get("when"), dict):
            event["when"] = copy.deepcopy(value["when"])
        action_values = value.get("actions", value.get("do", []))
        for ai, action_raw in enumerate(action_values if isinstance(action_values, list) else []):
            action_raw = action_raw if isinstance(action_raw, dict) else {"value": action_raw}
            action = {"action_id": _id("action", source, f"events/{index}", ai), "type": str(action_raw.get("type", "legacy")), "extensions": {}}
            if isinstance(action_raw.get("asset_ids"), list): action["asset_ids"] = copy.deepcopy(action_raw["asset_ids"])
            extra = _unknown(action_raw, {"type", "asset_ids"}, f"events[{index}].actions[{ai}]", diagnostics)
            if extra: action["extensions"]["legacy"] = extra
            event["actions"].append(action)
        extra = _unknown(value, {"trigger", "when", "actions", "do"}, f"events[{index}]", diagnostics)
        if extra: event["extensions"]["legacy"] = extra
        doc["events"].append(event)
    if legacy: doc["extensions"]["legacy"] = legacy
    return doc


def _story(raw: dict[str, Any], source: str, diagnostics: list[dict[str, str]]) -> dict[str, Any]:
    doc = _base("story", source); doc["scenes"] = []
    legacy = _unknown(raw, {"scenes"}, "$", diagnostics)
    for si, scene_raw in enumerate(raw.get("scenes", []) if isinstance(raw.get("scenes", []), list) else []):
        scene_raw = scene_raw if isinstance(scene_raw, dict) else {}
        scene = {"scene_id": _id("scene", source, "scenes", si), "lines": [], "beats": [], "extensions": {}}
        for li, line_raw in enumerate(scene_raw.get("lines", []) if isinstance(scene_raw.get("lines", []), list) else []):
            line_raw = line_raw if isinstance(line_raw, dict) else {}
            raw_speaker = line_raw.get("speaker")
            if isinstance(raw_speaker, int) and raw_speaker >= 0:
                speaker_id = f"character/native-{raw_speaker}"
            else:
                speaker_id = _id("speaker", source, f"scenes/{si}/lines", li)
            line = {"line_id": _id("line", source, f"scenes/{si}/lines", li), "speaker": speaker_id, "text": str(line_raw.get("text", "")), "extensions": {}}
            extra = _unknown(line_raw, {"text"}, f"scenes[{si}].lines[{li}]", diagnostics)
            if extra: line["extensions"]["legacy"] = extra
            scene["lines"].append(line)
        extra = _unknown(scene_raw, {"lines"}, f"scenes[{si}]", diagnostics)
        if extra: scene["extensions"]["legacy"] = extra
        doc["scenes"].append(scene)
    if legacy: doc["extensions"]["legacy"] = legacy
    return doc


def _animation(raw: dict[str, Any], source: str, diagnostics: list[dict[str, str]]) -> dict[str, Any]:
    if raw.get("schema_version") == 1 and raw.get("kind") == "animation":
        return copy.deepcopy(raw)
    doc = _base("animation", source); doc["animation_id"] = _id("animation", source, raw.get("resource", Path(source).stem)); doc["frames"] = []
    header = raw.get("native_header")
    if isinstance(header, dict): doc["native_header"] = copy.deepcopy(header)
    for i, frame_raw in enumerate(raw.get("frames", []) if isinstance(raw.get("frames", []), list) else []):
        frame_raw = frame_raw if isinstance(frame_raw, dict) else {}
        frame = {"frame_id": _id("frame", source, "frames", i), "asset_id": _id("asset", source, "frames", i), "x": int(frame_raw.get("x", 0)), "y": int(frame_raw.get("y", 0)), "extensions": {}}
        for field in ("delay_ms", "delay_native", "width", "height", "path", "mask_path", "raw_byte_4", "raw_byte_5", "raw_byte_7", "anchor", "palette"):
            if field in frame_raw: frame[field] = copy.deepcopy(frame_raw[field])
        extra = _unknown(frame_raw, {"x", "y", "delay_ms", "delay_native", "width", "height", "path", "mask_path", "raw_byte_4", "raw_byte_5", "raw_byte_7", "anchor", "palette"}, f"frames[{i}]", diagnostics)
        if extra: frame["extensions"]["legacy"] = extra
        doc["frames"].append(frame)
    legacy = _unknown(raw, {"resource", "native_header", "frames"}, "$", diagnostics)
    if legacy: doc["extensions"]["legacy"] = legacy
    return doc


def import_legacy(raw: dict[str, Any], source_path: str | Path, kind: str | None = None) -> tuple[dict[str, Any], list[dict[str, str]]]:
    """回傳 ``(canonical_document, machine_readable_diagnostics)``。"""
    if not isinstance(raw, dict): raise TypeError("legacy 文件根節點必須是物件")
    source = _path(source_path); kind = kind or raw.get("kind")
    if kind not in KINDS:
        name = Path(source).name.lower()
        kind = "campaign" if "campaign" in name else "scenario" if "scenario" in name else "story" if "story" in name else "animation" if "animation" in name else None
    if kind not in KINDS: raise ValueError("無法從來源判定文件 kind，請明確指定")
    diagnostics: list[dict[str, str]] = []
    doc = {"campaign": _campaign, "scenario": _scenario, "story": _story, "animation": _animation}[kind](raw, source, diagnostics)
    return doc, diagnostics


import_legacy_document = import_legacy


def write_canonical(document: dict[str, Any], destination: str | Path | None = None) -> str:
    """以排序 key、固定縮排寫出 canonical JSON；不改動輸入物件。"""
    text = json.dumps(document, ensure_ascii=False, indent=2, sort_keys=True, separators=(",", ": ")) + "\n"
    if destination is not None: Path(destination).write_text(text, encoding="utf-8")
    return text


def load_legacy(path: str | Path, kind: str | None = None) -> tuple[dict[str, Any], list[dict[str, str]]]:
    return import_legacy(json.loads(Path(path).read_text(encoding="utf-8")), path, kind)


def load_canonical(path: str | Path) -> dict[str, Any]:
    """讀取 canonical 文件，不重新推導身份或改寫 extensions。"""
    document = json.loads(Path(path).read_text(encoding="utf-8"))
    if not isinstance(document, dict) or document.get("schema_version") != 1:
        raise ValueError("不是 schema_version=1 的 canonical 文件")
    return document


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path); parser.add_argument("output", type=Path)
    parser.add_argument("--kind", choices=sorted(KINDS)); parser.add_argument("--diagnostics", type=Path)
    args = parser.parse_args()
    doc, diagnostics = load_legacy(args.input, args.kind)
    write_canonical(doc, args.output)
    if args.diagnostics: args.diagnostics.write_text(json.dumps(diagnostics, ensure_ascii=False, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    else: print(json.dumps(diagnostics, ensure_ascii=False, sort_keys=True))


if __name__ == "__main__": main()
