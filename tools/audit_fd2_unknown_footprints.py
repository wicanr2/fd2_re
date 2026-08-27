#!/usr/bin/env python3
"""建立 FD2 unknown 函式的受版控文字足跡清冊。

這個工具只做位址交叉索引，不推導語意、不改函式分類，也不寫回 IDA。
"""

from __future__ import annotations

import argparse
import bisect
import json
import re
from collections import Counter, defaultdict
from pathlib import Path


ADDRESS = re.compile(r"(?i)(?<![0-9a-f])0x[0-9a-f]+(?![0-9a-f])")
TEXT_SUFFIXES = {".md", ".txt", ".json", ".yaml", ".yml"}


def source_kind(path: Path) -> str:
    value = path.as_posix()
    name = path.name.lower()
    if name == "session-handoff-2026-07-06.md":
        return "history_only"
    if value.startswith("docs/knowledge-base/"):
        return "canonical_claim"
    if value.startswith("docs/data/ida/") or any(
        marker in name for marker in ("_ida", "capstone", "disasm", "xref", ".asm")
    ):
        return "direct_artifact"
    if value.startswith("docs/data/"):
        return "raw_only"
    return "other_docs"


def priority(kinds: set[str]) -> str:
    if {"canonical_claim", "direct_artifact"}.issubset(kinds):
        return "review_first"
    if "direct_artifact" in kinds:
        return "review_direct_artifact"
    if "canonical_claim" in kinds:
        return "review_canonical_claim"
    if kinds:
        return "recover_lead_only"
    return "no_versioned_text_footprint"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--inventory", default="docs/data/ida/fd2_function_inventory.json")
    parser.add_argument("--root", default=".")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    root = Path(args.root).resolve()
    inventory_path = (root / args.inventory).resolve()
    output_path = (root / args.output).resolve()
    inventory = json.loads(inventory_path.read_text(encoding="utf-8"))
    unknown = [
        item
        for item in inventory["functions"]
        if item["classification"]["value"] == "unknown"
    ]
    starts = [int(item["start"], 0) for item in unknown]
    by_start = {int(item["start"], 0): item for item in unknown}
    bounds = [(int(item["start"], 0), int(item["end"], 0)) for item in unknown]
    exact_hits: dict[int, dict[str, set[int]]] = defaultdict(lambda: defaultdict(set))
    range_hits: dict[int, dict[str, set[int]]] = defaultdict(lambda: defaultdict(set))

    candidates = []
    for base in (root / "docs", root / "README.md", root / "remake" / "README.md"):
        if base.is_file():
            candidates.append(base)
        elif base.is_dir():
            candidates.extend(path for path in base.rglob("*") if path.is_file())

    scanned = 0
    excluded = {
        inventory_path,
        output_path,
        (root / "docs/data/ida/fd2_unknown_footprints.json").resolve(),
    }
    for path in sorted(set(candidates)):
        if path in excluded or path.suffix.lower() not in TEXT_SUFFIXES:
            continue
        try:
            lines = path.read_text(encoding="utf-8").splitlines()
        except UnicodeDecodeError:
            continue
        scanned += 1
        relative = path.relative_to(root).as_posix()
        for line_number, line in enumerate(lines, 1):
            for token in ADDRESS.findall(line):
                address = int(token, 16)
                if address in by_start:
                    exact_hits[address][relative].add(line_number)
                position = bisect.bisect_right(starts, address) - 1
                if position >= 0 and bounds[position][0] <= address < bounds[position][1]:
                    range_hits[bounds[position][0]][relative].add(line_number)

    entries = []
    priority_counts = Counter()
    for item in unknown:
        start = int(item["start"], 0)
        exact = exact_hits.get(start, {})
        ranged = range_hits.get(start, {})
        files = set(exact) | set(ranged)
        kinds = {source_kind(Path(path)) for path in files}
        review_priority = priority(kinds)
        priority_counts[review_priority] += 1
        entries.append({
            "start": item["start"],
            "end": item["end"],
            "size": item["size"],
            "ida_analysis_name": item["ida_analysis_name"],
            "ida_function_flags": item["ida_function_flags"],
            "direct_caller_functions": item["direct_caller_functions"],
            "direct_code_xref_count": item["direct_code_xref_count"],
            "footprint_source_kinds": sorted(kinds),
            "review_priority": review_priority,
            "review_state": "pending_human_evidence_review",
            "exact_start_hits": [
                {"path": path, "lines": sorted(lines)} for path, lines in sorted(exact.items())
            ],
            "range_hits": [
                {"path": path, "lines": sorted(lines)} for path, lines in sorted(ranged.items())
            ],
        })

    report = {
        "schema_version": 1,
        "input": inventory["input"],
        "address_space": inventory["tool"]["address_space"],
        "inventory_function_count": inventory["function_count"],
        "unknown_function_count": len(unknown),
        "scanned_text_file_count": scanned,
        "exact_start_mentioned_count": sum(bool(exact_hits.get(start)) for start in starts),
        "range_mentioned_count": sum(bool(range_hits.get(start)) for start in starts),
        "review_priority_counts": dict(sorted(priority_counts.items())),
        "policy": (
            "位址命中只代表受版控文字足跡；classification/confidence/semantic "
            "必須由人工回查 caller、consumer、writer、raw bytes 與證據分級後另行加入語意索引"
        ),
        "entries": entries,
    }
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(
        json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    print(
        f"unknown={len(unknown)} exact={report['exact_start_mentioned_count']} "
        f"range={report['range_mentioned_count']} priorities={dict(priority_counts)}"
    )


if __name__ == "__main__":
    main()
