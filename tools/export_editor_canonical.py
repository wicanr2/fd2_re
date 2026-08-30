#!/usr/bin/env python3
"""批次建立 FD2 編輯器 canonical 文件。

此工具只讀取 legacy JSON 與已產生的動畫 metadata，將輸出寫入獨立目錄；
不會覆蓋 ``remake/assets/scenarios`` 或 ``remake/assets/story``。每次輸出
都使用固定排序、來源相對路徑與既有 legacy importer，因此可重複產生相同
內容，且未知欄位仍由 importer 保存在 ``extensions.legacy``。
"""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
from pathlib import Path
from typing import Any

from import_editor_legacy import IMPORTER_VERSION, import_legacy, write_canonical


EXPORTER_VERSION = "fd2-editor-canonical-exporter/1.0"


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _relative_source(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def _write_one(source_path: Path, source_name: str, destination: Path, kind: str) -> tuple[dict[str, Any], list[dict[str, str]]]:
    raw = json.loads(source_path.read_text(encoding="utf-8"))
    document, diagnostics = import_legacy(raw, source_name, kind)
    destination.parent.mkdir(parents=True, exist_ok=True)
    write_canonical(document, destination)
    return {
        "kind": kind,
        "source": source_name,
        "output": destination.as_posix(),
        "document_id": document["document_id"],
        "sha256": _sha256(destination),
        "diagnostics": len(diagnostics),
    }, diagnostics


def _assert_safe_output(output: Path, root: Path) -> None:
    forbidden = {
        (root / "remake" / "assets" / "scenarios").resolve(),
        (root / "remake" / "assets" / "story").resolve(),
    }
    resolved = output.resolve()
    if resolved in forbidden:
        raise ValueError("canonical 輸出目錄不可覆蓋 legacy scenarios/story 目錄")


def _candidate(value: Any, source: str, field: str, index: int) -> dict[str, Any]:
    return {
        "value": value,
        "source": source,
        "field": field,
        "index": index,
        "evidence": "direct legacy field",
    }


def _append_candidate(bucket: list[dict[str, Any]], value: Any, source: str, field: str, index: int) -> None:
    if isinstance(value, (str, int)) and value not in ("", -1):
        item = _candidate(value, source, field, index)
        if item not in bucket:
            bucket.append(item)


def build_character_identity_catalog(root: str | Path) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    """從 legacy party 與 story speaker 的直接欄位建立身份候選，不做猜測合併。"""
    root = Path(root).resolve()
    characters: dict[int, dict[str, Any]] = {}
    diagnostics: list[dict[str, Any]] = []

    def entry(native_identity: int) -> dict[str, Any]:
        return characters.setdefault(native_identity, {
            "character_id": f"character/native-{native_identity}",
            "native_identity": native_identity,
            "display_name_candidates": [],
            "portrait_selector_candidates": [],
            "map_sprite_selector_candidates": [],
            "battle_animation_selector_candidates": [],
            "sources": [],
        })

    for path in sorted((root / "remake" / "assets" / "scenarios").glob("ch*.json")):
        raw = json.loads(path.read_text(encoding="utf-8"))
        source = _relative_source(path, root)
        for index, unit in enumerate(raw.get("party", []) if isinstance(raw, dict) else []):
            if not isinstance(unit, dict) or not isinstance(unit.get("native_identity"), int) or unit["native_identity"] < 0:
                continue
            identity = entry(unit["native_identity"])
            identity["sources"].append({"path": source, "field": "party", "index": index})
            _append_candidate(identity["display_name_candidates"], unit.get("name"), source, "party.name", index)
            _append_candidate(identity["portrait_selector_candidates"], unit.get("portrait"), source, "party.portrait", index)
            # `fig` is preserved as a raw field candidate; this catalog does not
            # assert whether the native consumer is map or battle presentation.
            _append_candidate(identity["map_sprite_selector_candidates"], unit.get("fig"), source, "party.fig", index)

    for path in sorted((root / "remake" / "assets" / "story").glob("*.json")):
        raw = json.loads(path.read_text(encoding="utf-8"))
        source = _relative_source(path, root)
        for scene_index, scene in enumerate(raw.get("scenes", []) if isinstance(raw, dict) else []):
            for line_index, line in enumerate(scene.get("lines", []) if isinstance(scene, dict) else []):
                if not isinstance(line, dict) or not isinstance(line.get("speaker"), int) or line["speaker"] < 0:
                    continue
                identity = entry(line["speaker"])
                identity["sources"].append({"path": source, "field": "scenes.lines", "index": [scene_index, line_index]})
                _append_candidate(identity["display_name_candidates"], line.get("speaker_name"), source, "lines.speaker_name", line_index)

    for identity in characters.values():
        for field in ("display_name_candidates", "portrait_selector_candidates", "map_sprite_selector_candidates", "battle_animation_selector_candidates"):
            values = {json.dumps(item["value"], ensure_ascii=False, sort_keys=True) for item in identity[field]}
            if len(values) > 1:
                diagnostics.append({
                    "code": "conflicting_identity_candidates",
                    "character_id": identity["character_id"],
                    "field": field,
                    "values": sorted(values),
                    "severity": "error",
                    "message": "同一身份有多個直接來源候選；保留全部候選，不選定語意值",
                })

    catalog = {
        "schema_version": 1,
        "kind": "character_identity_catalog",
        "source": {"root": "repository-relative", "evidence": "direct legacy fields"},
        "characters": [characters[key] for key in sorted(characters)],
        "diagnostics": diagnostics,
    }
    return catalog, diagnostics


def validate_character_identity_catalog(catalog: dict[str, Any]) -> None:
    """以標準函式庫執行角色清冊的嚴格結構閘門。

    這不是把推測升格為語意；它只確保匯出資料符合
    ``fd2-character-identity-catalog.schema.json``，且每筆候選仍帶有
    可回查的來源、欄位、索引與證據等級。
    """
    required = {"schema_version", "kind", "source", "characters", "diagnostics"}
    if set(catalog) != required:
        raise ValueError("character identity catalog 頂層欄位不符合 schema")
    if catalog["schema_version"] != 1 or catalog["kind"] != "character_identity_catalog":
        raise ValueError("character identity catalog schema_version/kind 無效")
    if catalog["source"] != {"root": "repository-relative", "evidence": "direct legacy fields"}:
        raise ValueError("character identity catalog source 無效")
    if not isinstance(catalog["characters"], list) or not isinstance(catalog["diagnostics"], list):
        raise ValueError("character identity catalog characters/diagnostics 必須是陣列")

    candidate_fields = (
        "display_name_candidates",
        "portrait_selector_candidates",
        "map_sprite_selector_candidates",
        "battle_animation_selector_candidates",
    )
    character_keys = {"character_id", "native_identity", *candidate_fields, "sources"}
    candidate_keys = {"value", "source", "field", "index", "evidence"}
    source_keys = {"path", "field", "index"}
    diagnostic_keys = {"code", "character_id", "field", "values", "severity", "message"}

    character_ids: set[str] = set()
    for character in catalog["characters"]:
        if not isinstance(character, dict) or set(character) != character_keys:
            raise ValueError("character identity catalog character 欄位不完整")
        character_id = character["character_id"]
        native_identity = character["native_identity"]
        if (
            not isinstance(character_id, str)
            or not character_id.startswith("character/native-")
            or not isinstance(native_identity, int)
            or isinstance(native_identity, bool)
            or native_identity < 0
            or character_id != f"character/native-{native_identity}"
            or character_id in character_ids
        ):
            raise ValueError("character identity catalog character_id/native_identity 無效或重複")
        character_ids.add(character_id)
        for field in candidate_fields:
            if not isinstance(character[field], list):
                raise ValueError(f"{field} 必須是陣列")
            for candidate in character[field]:
                if not isinstance(candidate, dict) or set(candidate) != candidate_keys:
                    raise ValueError(f"{field} 候選欄位不完整")
                value = candidate["value"]
                if not (
                    (isinstance(value, str) and value)
                    or (isinstance(value, int) and not isinstance(value, bool))
                ):
                    raise ValueError(f"{field} 候選 value 無效")
                if (
                    not isinstance(candidate["source"], str)
                    or not candidate["source"]
                    or candidate["source"].startswith("/")
                    or (len(candidate["source"]) >= 2 and candidate["source"][1] == ":")
                    or not isinstance(candidate["field"], str)
                    or not candidate["field"]
                    or not isinstance(candidate["index"], int)
                    or isinstance(candidate["index"], bool)
                    or candidate["index"] < 0
                    or candidate["evidence"] != "direct legacy field"
                ):
                    raise ValueError(f"{field} 候選 provenance 無效")
        if not isinstance(character["sources"], list):
            raise ValueError("character sources 必須是陣列")
        for source in character["sources"]:
            if not isinstance(source, dict) or set(source) != source_keys:
                raise ValueError("character source provenance 欄位不完整")
            index = source["index"]
            valid_index = (
                isinstance(index, int)
                and not isinstance(index, bool)
                and index >= 0
            ) or (
                isinstance(index, list)
                and len(index) == 2
                and all(isinstance(item, int) and not isinstance(item, bool) and item >= 0 for item in index)
            )
            if (
                not isinstance(source["path"], str)
                or not source["path"]
                or source["path"].startswith("/")
                or (len(source["path"]) >= 2 and source["path"][1] == ":")
                or not isinstance(source["field"], str)
                or not source["field"]
                or not valid_index
            ):
                raise ValueError("character source provenance 無效")

    for diagnostic in catalog["diagnostics"]:
        if not isinstance(diagnostic, dict) or set(diagnostic) != diagnostic_keys:
            raise ValueError("character identity diagnostic 欄位不完整")
        if (
            diagnostic["code"] != "conflicting_identity_candidates"
            or diagnostic["character_id"] not in character_ids
            or not isinstance(diagnostic["field"], str)
            or not diagnostic["field"]
            or not isinstance(diagnostic["values"], list)
            or len(diagnostic["values"]) < 2
            or not all(isinstance(value, str) for value in diagnostic["values"])
            or diagnostic["severity"] != "error"
            or not isinstance(diagnostic["message"], str)
            or not diagnostic["message"]
        ):
            raise ValueError("character identity diagnostic 無效")


def validate_canonical_bundle(output: str | Path) -> None:
    """驗證批次文件唯一性、campaign 轉場與 story native identity 引用。"""
    output = Path(output)
    summary = json.loads((output / "bundle-summary.json").read_text(encoding="utf-8"))
    catalog = json.loads((output / "character-identity.json").read_text(encoding="utf-8"))
    validate_character_identity_catalog(catalog)
    documents = []
    for item in summary["documents"]:
        document = json.loads((output / item["output"]).read_text(encoding="utf-8"))
        documents.append(document)
    document_ids = [document.get("document_id") for document in documents]
    if len(document_ids) != len(set(document_ids)):
        raise ValueError("canonical 批次含重複 document_id")
    catalog_ids = {item["character_id"] for item in catalog.get("characters", [])}
    for document in documents:
        if document.get("kind") == "campaign":
            node_ids = {node["node_id"] for node in document.get("nodes", [])}
            for node in document.get("nodes", []):
                for field in ("next", "on_win", "on_lose"):
                    target = node.get(field)
                    if target is not None and target not in node_ids:
                        raise ValueError(f"canonical campaign {field} 斷裂：{target}")
        if document.get("kind") == "story":
            for scene in document.get("scenes", []):
                for line in scene.get("lines", []):
                    speaker = line.get("speaker")
                    if isinstance(speaker, str) and speaker.startswith("character/native-") and speaker not in catalog_ids:
                        raise ValueError(f"canonical story speaker 缺少身份：{speaker}")


def build_canonical_bundle(
    root: str | Path,
    output: str | Path,
    *,
    include_animations: bool = True,
    clean: bool = False,
) -> dict[str, Any]:
    """建立一份完整批次，回傳可機讀摘要。

    ``root`` 是儲存庫根目錄；``output`` 必須是獨立的 canonical 目錄。
    ``campaign_full.json`` 是唯一正式 campaign 輸出，避免把舊的短版 campaign
    當成第二份戰役身份。
    """
    root = Path(root).resolve()
    output = Path(output).resolve()
    _assert_safe_output(output, root)
    if clean and output.exists():
        shutil.rmtree(output)
    output.mkdir(parents=True, exist_ok=True)

    sources: list[tuple[str, Path, str, Path]] = []
    scenario_root = root / "remake" / "assets" / "scenarios"
    story_root = root / "remake" / "assets" / "story"
    campaign = scenario_root / "campaign_full.json"
    if not campaign.is_file():
        raise FileNotFoundError(campaign)
    sources.append(("campaign", campaign, "campaign", output / "campaign" / "campaign_full.json"))

    scenarios = sorted(
        (path for path in scenario_root.glob("ch*.json") if path.name[2:-5].isdigit()),
        key=lambda path: path.name,
    )
    stories = sorted(story_root.glob("ch*.json"), key=lambda path: path.name)
    for path in scenarios:
        sources.append(("scenario", path, "scenario", output / "scenarios" / path.name))
    for path in stories:
        sources.append(("story", path, "story", output / "story" / path.name))

    if include_animations:
        animation_root = root / "remake" / "generated-assets"
        for path in sorted(animation_root.glob("*/animations/*/animation.json")):
            relative = _relative_source(path, root)
            output_name = f"{path.parent.name.lower()}.json"
            sources.append(("animation", path, "animation", output / "animations" / output_name))

    entries: list[dict[str, Any]] = []
    diagnostics: list[dict[str, Any]] = []
    for kind, source_path, importer_kind, destination in sources:
        source_name = _relative_source(source_path, root)
        entry, item_diagnostics = _write_one(source_path, source_name, destination, importer_kind)
        entry["output"] = destination.relative_to(output).as_posix()
        entries.append(entry)
        diagnostics.append({
            "kind": kind,
            "source": source_name,
            "items": item_diagnostics,
        })

    catalog, identity_diagnostics = build_character_identity_catalog(root)
    (output / "character-identity.json").write_text(
        json.dumps(catalog, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )

    summary: dict[str, Any] = {
        "schema_version": 1,
        "exporter_version": EXPORTER_VERSION,
        "importer_version": IMPORTER_VERSION,
        "source_root": "repository-relative",
        "include_animations": include_animations,
        "counts": {
            "campaign": sum(entry["kind"] == "campaign" for entry in entries),
            "scenario": sum(entry["kind"] == "scenario" for entry in entries),
            "story": sum(entry["kind"] == "story" for entry in entries),
            "animation": sum(entry["kind"] == "animation" for entry in entries),
        },
        "documents": entries,
        "diagnostics": diagnostics,
        "identity_diagnostics": len(identity_diagnostics),
    }
    (output / "bundle-summary.json").write_text(
        json.dumps(summary, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )
    validate_canonical_bundle(output)
    return summary


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--without-animations", action="store_true")
    parser.add_argument("--clean", action="store_true", help="只清除指定的 canonical 輸出目錄")
    args = parser.parse_args()
    summary = build_canonical_bundle(
        args.root,
        args.output,
        include_animations=not args.without_animations,
        clean=args.clean,
    )
    print(json.dumps(summary["counts"], ensure_ascii=False, sort_keys=True))


if __name__ == "__main__":
    main()
