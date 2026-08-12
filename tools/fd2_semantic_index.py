#!/usr/bin/env python3
"""讀取並驗證 FD2 非破壞性位址語意索引。"""

from __future__ import annotations

import json
from pathlib import Path


CONFIDENCE_LEVELS = {"已證實", "強推論", "假說", "未知"}
CLASSIFICATIONS = {"product", "runtime", "driver", "unknown"}
INPUT_FIELDS = ("file", "size", "md5", "sha256")


def _classification_counts(entries_by_address: dict[int, list[dict]]) -> dict[str, int]:
    result: dict[str, int] = {}
    for annotations in entries_by_address.values():
        value = annotations[0]["classification"]
        result[value] = result.get(value, 0) + 1
    return result


def _validate_input_record(record: object, label: str) -> dict:
    if not isinstance(record, dict):
        raise ValueError(f"{label} must be an object")
    for field in INPUT_FIELDS:
        if field not in record:
            raise ValueError(f"{label}.{field} is required")
    if not isinstance(record["file"], str) or not record["file"]:
        raise ValueError(f"{label}.file must be a non-empty string")
    if not isinstance(record["size"], int) or record["size"] <= 0:
        raise ValueError(f"{label}.size must be a positive integer")
    for field, length in (("md5", 32), ("sha256", 64)):
        value = record[field]
        if (
            not isinstance(value, str)
            or len(value) != length
            or any(char not in "0123456789abcdef" for char in value)
        ):
            raise ValueError(f"{label}.{field} must be lowercase hexadecimal")
    return record


def validate_input_identity(index_input: dict, actual_input: dict) -> None:
    """拒絕把一個版本的位址語意套到另一個執行檔。"""

    expected = _validate_input_record(index_input, "semantic index input")
    actual = _validate_input_record(actual_input, "actual input")
    mismatches = [
        field
        for field in INPUT_FIELDS
        if expected[field] != actual[field]
    ]
    if mismatches:
        details = ", ".join(
            f"{field}: expected {expected[field]!r}, got {actual[field]!r}"
            for field in mismatches
        )
        raise ValueError(f"semantic index input mismatch ({details})")


def load_semantic_index(path: str | Path, repo_root: str | Path | None = None):
    """回傳 ``(document, entries_by_address)``，並完整驗證固定契約。"""

    index_path = Path(path)
    with index_path.open(encoding="utf-8") as source:
        document = json.load(source)
    if document.get("schema_version") != 1:
        raise ValueError("semantic index schema_version must be 1")
    _validate_input_record(document.get("input"), "semantic index input")
    if document.get("address_space") != "IDA flat-loader linear address":
        raise ValueError("semantic index address_space is missing or unsupported")

    entries = document.get("entries")
    if not isinstance(entries, list):
        raise ValueError("semantic index entries must be an array")
    by_address: dict[int, list[dict]] = {}
    root = Path(repo_root).resolve() if repo_root is not None else None
    for position, entry in enumerate(entries):
        label = f"semantic index entries[{position}]"
        if not isinstance(entry, dict):
            raise ValueError(f"{label} must be an object")
        raw_address = entry.get("address")
        if not isinstance(raw_address, str):
            raise ValueError(f"{label}.address must be a hexadecimal string")
        try:
            address = int(raw_address, 0)
        except ValueError as error:
            raise ValueError(f"{label}.address is invalid") from error
        if address < 0 or raw_address != hex(address):
            raise ValueError(f"{label}.address must use canonical lowercase hex")
        if address in by_address:
            raise ValueError(f"duplicate semantic address {raw_address}")
        if entry.get("classification") not in CLASSIFICATIONS:
            raise ValueError(f"{label}.classification is missing or unsupported")
        if entry.get("confidence") not in CONFIDENCE_LEVELS:
            raise ValueError(f"{label}.confidence is missing or unsupported")
        for field in ("semantic", "scope"):
            if not isinstance(entry.get(field), str) or not entry[field].strip():
                raise ValueError(f"{label}.{field} must be a non-empty string")
        evidence = entry.get("evidence")
        if (
            not isinstance(evidence, list)
            or not evidence
            or any(not isinstance(item, str) or not item for item in evidence)
            or len(evidence) != len(set(evidence))
        ):
            raise ValueError(f"{label}.evidence must be a unique non-empty string array")
        if root is not None:
            for relative in evidence:
                target = (root / relative).resolve()
                try:
                    target.relative_to(root)
                except ValueError as error:
                    raise ValueError(f"{label}.evidence escapes the repository: {relative}") from error
                if not target.is_file():
                    raise ValueError(f"{label}.evidence does not exist: {relative}")
        by_address[address] = [entry]
    return document, by_address


def build_compact_report(inventory: dict) -> dict:
    """由完整 IDA 匯出建立受版控的精簡函式清冊。"""

    if inventory.get("schema_version") != 1:
        raise ValueError("function inventory schema_version must be 1")
    functions = inventory.get("functions")
    if not isinstance(functions, list):
        raise ValueError("function inventory functions must be an array")
    if inventory.get("function_count") != len(functions):
        raise ValueError("function inventory count does not match functions")

    starts = set()
    compact_functions = []
    counts: dict[str, int] = {}
    for position, function in enumerate(functions):
        label = f"function inventory functions[{position}]"
        try:
            start = int(function["start"], 0)
            end = int(function["end"], 0)
        except (KeyError, TypeError, ValueError) as error:
            raise ValueError(f"{label} has invalid bounds") from error
        if start in starts or end <= start or function.get("size") != end - start:
            raise ValueError(f"{label} has duplicate or inconsistent bounds")
        starts.add(start)
        classification = function.get("classification")
        if not isinstance(classification, dict) or classification.get("value") not in CLASSIFICATIONS:
            raise ValueError(f"{label} has invalid classification")
        value = classification["value"]
        counts[value] = counts.get(value, 0) + 1
        callers = sorted({
            item["caller_function"]["start"]
            for item in function.get("code_xrefs_to_start", [])
            if isinstance(item, dict) and isinstance(item.get("caller_function"), dict)
        }, key=lambda address: int(address, 0))
        compact_functions.append({
            "start": function["start"],
            "end": function["end"],
            "size": function["size"],
            "ida_analysis_name": function.get("ida_analysis_name"),
            "ida_function_flags": function.get("ida_function_flags", []),
            "direct_caller_functions": callers,
            "direct_code_xref_count": len(function.get("code_xrefs_to_start", [])),
            "direct_data_xref_count": len(function.get("data_xrefs_to_start", [])),
            "classification": classification,
            "semantic_annotations": function.get("semantic_annotations", []),
        })
    if counts != inventory.get("classification_counts"):
        raise ValueError("function inventory classification counts are inconsistent")

    return {
        "schema_version": 1,
        "tool": inventory.get("tool"),
        "input": inventory.get("input"),
        "imagebase": inventory.get("imagebase"),
        "function_count": len(compact_functions),
        "classification_counts": counts,
        "semantic_annotation_count": inventory.get("semantic_annotation_count"),
        "classification_note": inventory.get("classification_note"),
        "detail_note": (
            "每筆保留 IDA 函式邊界、分析名稱、旗標、直接 caller 函式與分級語意；"
            "逐 call-site 交叉參照由同一匯出器重生，不在此精簡版重複保存"
        ),
        "functions": compact_functions,
    }
