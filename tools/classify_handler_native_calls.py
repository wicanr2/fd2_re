#!/usr/bin/env python3
"""將既有 handler JSON 中已分級的 native call 非破壞性改標。"""

import argparse
import json
import re
from collections import Counter
from pathlib import Path

import dump_chapter_beats
from export_handler_scripts import NATIVE_EVIDENCE, NATIVE_SEMANTICS, UNRESOLVED_NATIVE


CLASSIFIED = {
    hex(address): semantic
    for address, (semantic, _arity) in dump_chapter_beats.PRIM.items()
    if semantic.startswith("native_")
}


def walk(beats):
    for beat in beats:
        yield beat
        yield from walk(beat.get("then", []))
        yield from walk(beat.get("else", []))


def classify_document(document):
    changed = 0
    counts = Counter()
    unresolved_counts = Counter()
    for beat in walk(document.get("beats", [])):
        if beat.get("op") == "native_call":
            counts[beat.get("native_target")] += 1
            continue
        if beat.get("op") == "unresolved_native_call":
            unresolved_counts[beat.get("native_target")] += 1
            continue
        if beat.get("op") != "unknown":
            continue
        target = beat.get("native_target")
        semantic = CLASSIFIED.get(target)
        if semantic is None:
            unresolved = UNRESOLVED_NATIVE.get(target)
            if unresolved is not None:
                beat["op"] = "unresolved_native_call"
                beat["native_semantic"] = unresolved["semantic"]
                beat["native_confidence"] = unresolved["confidence"]
                beat["native_evidence"] = list(unresolved["evidence"])
                changed += 1
                unresolved_counts[target] += 1
            continue
        beat["op"] = "native_call"
        beat["native_semantic"] = NATIVE_SEMANTICS.get(target, semantic)
        beat["native_confidence"] = "已證實"
        beat["native_evidence"] = list(NATIVE_EVIDENCE[target])
        changed += 1
        counts[target] += 1
    diagnostics = document.setdefault("diagnostics", {})
    unknown = sum(1 for beat in walk(document.get("beats", [])) if beat.get("op") == "unknown")
    diagnostics["unknown_ops"] = unknown
    diagnostics["classified_native_ops"] = sum(counts.values())
    diagnostics["unresolved_native_ops"] = sum(unresolved_counts.values())
    return changed, unknown, counts, unresolved_counts


def render_minimal_changes(original, document):
    """只改目標 beat 與 diagnostics，保留人工 JSON 排版。"""

    replacements = []
    for target, semantic in CLASSIFIED.items():
        pattern = re.compile(
            r'(?P<indent>^[ \t]*)"op": "unknown",(?P<body>\n'
            r'(?P=indent)"native_target": "' + re.escape(target) + r'",)',
            re.MULTILINE,
        )
        evidence = json.dumps(NATIVE_EVIDENCE[target], ensure_ascii=False)
        original, count = pattern.subn(
            lambda match: (
                f'{match.group("indent")}"op": "native_call",'
                f'{match.group("body")}\n'
                f'{match.group("indent")}"native_semantic": "{NATIVE_SEMANTICS.get(target, semantic)}",\n'
                f'{match.group("indent")}"native_confidence": "已證實",\n'
                f'{match.group("indent")}"native_evidence": {evidence},'
            ),
            original,
        )
        replacements.append(count)
    for target, unresolved in UNRESOLVED_NATIVE.items():
        pattern = re.compile(
            r'(?P<indent>^[ \t]*)"op": "unknown",(?P<body>\n'
            r'(?P=indent)"native_target": "' + re.escape(target) + r'",)',
            re.MULTILINE,
        )
        evidence = json.dumps(unresolved["evidence"], ensure_ascii=False)
        original, count = pattern.subn(
            lambda match: (
                f'{match.group("indent")}"op": "unresolved_native_call",'
                f'{match.group("body")}\n'
                f'{match.group("indent")}"native_semantic": "{unresolved["semantic"]}",\n'
                f'{match.group("indent")}"native_confidence": "{unresolved["confidence"]}",\n'
                f'{match.group("indent")}"native_evidence": {evidence},'
            ),
            original,
        )
        replacements.append(count)
    diagnostics = document["diagnostics"]
    pattern = re.compile(
        r'(?P<indent>^[ \t]*)"unknown_ops": \d+'
        r'(?:,\n(?P=indent)"classified_native_ops": \d+)?'
        r'(?:,\n(?P=indent)"unresolved_native_ops": \d+)?',
        re.MULTILINE,
    )
    replacement = (
        f'\\g<indent>"unknown_ops": {diagnostics["unknown_ops"]},\n'
        f'\\g<indent>"classified_native_ops": {diagnostics["classified_native_ops"]},\n'
        f'\\g<indent>"unresolved_native_ops": {diagnostics["unresolved_native_ops"]}'
    )
    original, diagnostic_count = pattern.subn(replacement, original, count=1)
    if diagnostic_count != 1:
        raise ValueError("handler diagnostics block was not found exactly once")
    return original, sum(replacements)


def render_manifest_minimal_changes(original, summaries):
    document = json.loads(original)
    offset = 0
    rendered = original
    for script in document.get("scripts", []):
        summary = summaries.get((script.get("chapter"), script.get("phase")))
        if summary is None:
            raise ValueError("handler manifest references a missing chapter/phase")
        marker = f'"unknown_ops": {script.get("unknown_ops")}'
        position = rendered.find(marker, offset)
        if position < 0:
            raise ValueError("handler manifest unknown_ops entry was not found in order")
        metrics = re.match(
            r'"unknown_ops": \d+'
            r'(?:,\n[ \t]+"(?:classified_native_ops|unresolved_native_ops)": \d+)*',
            rendered[position:],
        )
        if metrics is None:
            raise ValueError("handler manifest metrics block was not found")
        replacement = (
            f'"unknown_ops": {summary["unknown_ops"]}'
        )
        if summary["classified_native_ops"]:
            replacement += (
                f',\n      "classified_native_ops": '
                f'{summary["classified_native_ops"]}'
            )
        if summary["unresolved_native_ops"]:
            replacement += (
                f',\n      "unresolved_native_ops": '
                f'{summary["unresolved_native_ops"]}'
            )
        rendered = rendered[:position] + replacement + rendered[position + metrics.end():]
        offset = position + len(replacement)
    return rendered


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("handler_dir")
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()

    root = Path(args.handler_dir)
    total_changed = 0
    total_unknown = 0
    totals = Counter()
    unresolved_totals = Counter()
    summaries = {}
    for path in sorted(root.glob("ch??_*.json")):
        original = path.read_text(encoding="utf-8")
        document = json.loads(original)
        changed, unknown, counts, unresolved_counts = classify_document(document)
        total_changed += changed
        total_unknown += unknown
        totals.update(counts)
        unresolved_totals.update(unresolved_counts)
        summaries[(document.get("chapter"), document.get("phase"))] = {
            "unknown_ops": unknown,
            "classified_native_ops": sum(counts.values()),
            "unresolved_native_ops": sum(unresolved_counts.values()),
        }
        if changed and not args.check:
            rendered, replacement_count = render_minimal_changes(original, document)
            if replacement_count != changed:
                raise ValueError(f"{path}: expected {changed} replacements, got {replacement_count}")
            path.write_text(rendered, encoding="utf-8")

    manifest_changed = False
    manifest_path = root / "_manifest.json"
    if manifest_path.is_file():
        manifest_original = manifest_path.read_text(encoding="utf-8")
        manifest = json.loads(manifest_original)
        for script in manifest.get("scripts", []):
            summary = summaries.get((script.get("chapter"), script.get("phase")))
            if summary is None:
                raise ValueError("handler manifest references a missing chapter/phase")
            for key, value in summary.items():
                if key in ("classified_native_ops", "unresolved_native_ops") and value == 0 and key not in script:
                    continue
                if script.get(key) != value:
                    script[key] = value
                    manifest_changed = True
        if manifest_changed and not args.check:
            manifest_path.write_text(
                render_manifest_minimal_changes(manifest_original, summaries),
                encoding="utf-8",
            )
    print(
        f"changed={total_changed} classified_native={sum(totals.values())} "
        f"unresolved_native={sum(unresolved_totals.values())} unknown={total_unknown} "
        f"classified_targets={len(totals)} unresolved_targets={len(unresolved_totals)} "
        f"manifest_changed={str(manifest_changed).lower()}"
    )
    if args.check and (total_changed or manifest_changed):
        raise SystemExit(1)


if __name__ == "__main__":
    main()
