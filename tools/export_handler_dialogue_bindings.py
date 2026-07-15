#!/usr/bin/env python3
"""Export conservative handler-dialogue binding skeletons.

This bridges the immutable EXE handler exports to editable story JSON without
guessing text.  A dialog call receives a ``dialogue_contexts`` entry only if
its active FDTXT resource has *exactly one* ``count_aligned`` script mapping
in the supplied story-index manifest.  The only implicit resource is the
nonzero handler chapter's initial ``FDTXT_<chapter>``.  ``loadch`` selects a
map/chapter, not proven FDTXT data; after every ``loadch`` the FDTXT context is
therefore unresolved until a future explicit resource mapping is supplied.

The generated files are HandlerBinding JSON skeletons: they contain dialogue
contexts and an empty ``overrides`` object, leaving pan/acting/other native
operations for evidence-backed RE.  No original text or binary data is copied.
Calls with missing, ambiguous, out-of-range, multi-scene, or post-``loadch``
contexts are recorded in ``_diagnostics.json`` and intentionally receive no
context.  Chapter zero is also deliberately not treated as FDTXT_000: its
prologue handler has direct evidence that map/load text numbering differs.

Existing generated targets are never overwritten.  This makes it safe to use
an output directory that later contains hand-edited bindings: move/delete the
specific generated files deliberately before regenerating them.

Example:

    python3 tools/export_handler_dialogue_bindings.py \
        remake/assets/cutscenes/handlers \
        remake/assets/cutscenes/dialogue-index/count-aligned.json \
        /tmp/fd2-dialogue-bindings
"""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path
from typing import Any


SCHEMA_VERSION = 1
DIAGNOSTICS_NAME = "_diagnostics.json"


def is_nonnegative_int(value: Any) -> bool:
    """Return true for JSON integers usable as chapter/string indices."""

    return isinstance(value, int) and not isinstance(value, bool) and value >= 0


def source_dat_for_chapter(chapter: int) -> str:
    """Format the FDTXT resource name used by an immediate chapter number."""

    return f"FDTXT_{chapter:03d}"


def load_json(path: Path) -> Any:
    """Read JSON with a contextual error for command-line users."""

    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise ValueError(f"cannot read {path}: {exc}") from exc


def load_unique_mappings(path: Path) -> dict[str, dict[str, Any]]:
    """Return source_dat -> sole count-aligned mapping; retain ambiguity.

    The empty dict sentinel means that a source is represented in the manifest
    but cannot be selected because zero or multiple count-aligned scripts were
    supplied.  Malformed mapping records fail before output is created.
    """

    manifest = load_json(path)
    if not isinstance(manifest, dict) or manifest.get("schema_version") != SCHEMA_VERSION:
        raise ValueError(f"story index map {path} must have schema_version={SCHEMA_VERSION}")
    if manifest.get("mapping_kind") != "count_aligned_only":
        raise ValueError(f"story index map {path} is not count_aligned_only")
    resources = manifest.get("resources")
    if not isinstance(resources, list):
        raise ValueError(f"story index map {path} lacks resources array")

    result: dict[str, dict[str, Any]] = {}
    for resource in resources:
        if not isinstance(resource, dict):
            raise ValueError(f"story index map {path} contains non-object resource")
        source_dat = resource.get("source_dat")
        mappings = resource.get("script_mappings")
        if not isinstance(source_dat, str) or not source_dat or not isinstance(mappings, list):
            raise ValueError(f"story index map {path} has invalid resource")
        candidates = [
            item
            for item in mappings
            if isinstance(item, dict) and item.get("status") == "count_aligned"
        ]
        # An invalid candidate should never turn into a seemingly valid binding.
        if len(candidates) == 1:
            candidate = candidates[0]
            if candidate.get("source_dat") != source_dat or not isinstance(candidate.get("script"), str):
                raise ValueError(f"story index map {path} has invalid mapping for {source_dat}")
            if not isinstance(candidate.get("mappings"), list):
                raise ValueError(f"story index map {path} mapping {source_dat} lacks mappings")
            result[source_dat] = candidate
        else:
            result[source_dat] = {}
    return result


def relative_reference(target: Path, output_dir: Path) -> str:
    """Return a portable binding-relative file reference."""

    return os.path.relpath(target.resolve(), output_dir.resolve()).replace(os.sep, "/")


def handler_files(handlers_dir: Path) -> list[Path]:
    """Return only the per-handler JSON exports, in stable filename order."""

    files = sorted(path for path in handlers_dir.glob("ch??_*.json") if path.is_file())
    if not files:
        raise ValueError(f"no chNN_pre.json/chNN_post.json files under {handlers_dir}")
    return files


def dialog_diagnostic(
    beat: dict[str, Any], source_dat: str | None, kind: str, message: str
) -> dict[str, Any]:
    """Build one audit-friendly skipped-dialog record."""

    source = beat.get("source")
    addr = source.get("addr") if isinstance(source, dict) else None
    result: dict[str, Any] = {
        "kind": kind,
        "source_addr": addr,
        "source_dat": source_dat,
        "text_index": beat.get("text_index"),
        "message": message,
    }
    return result


def mapping_targets(mapping: dict[str, Any], string_index: int) -> list[Any] | None:
    """Return the exact target list for a raw string, or None when invalid."""

    mappings = mapping.get("mappings")
    if not isinstance(mappings, list) or string_index >= len(mappings):
        return None
    item = mappings[string_index]
    if not isinstance(item, dict) or item.get("string_index") != string_index:
        return None
    targets = item.get("targets")
    return targets if isinstance(targets, list) else None


def export_handler(
    handler_path: Path,
    output_dir: Path,
    map_path: Path,
    unique_mappings: dict[str, dict[str, Any]],
) -> tuple[dict[str, Any], dict[str, Any]]:
    """Create one binding and its complete diagnostic record without writing."""

    handler = load_json(handler_path)
    if not isinstance(handler, dict):
        raise ValueError(f"handler {handler_path} is not an object")
    chapter = handler.get("chapter")
    beats = handler.get("beats")
    if not is_nonnegative_int(chapter) or not isinstance(beats, list):
        raise ValueError(f"handler {handler_path} lacks non-negative chapter or beats array")

    # The common handler chapter is sufficient evidence only before a loadch.
    # ch00 is exceptional: its handler's map loads and text resources are known
    # not to share an index, so FDTXT_000 would be a fabricated context.
    initial_source: str | None = source_dat_for_chapter(chapter) if chapter != 0 else None
    current_source = initial_source
    context_reason: str | None = (
        None
        if current_source is not None
        else "chapter zero is not an implicit FDTXT resource; explicit mapping required"
    )
    context_origin: str | None = None
    contexts: dict[str, dict[str, str]] = {}
    skipped: list[dict[str, Any]] = []
    loadch_events: list[dict[str, Any]] = []

    for beat_index, beat in enumerate(beats):
        if not isinstance(beat, dict):
            raise ValueError(f"handler {handler_path} beat {beat_index} is not an object")
        op = beat.get("op")
        if op == "loadch":
            immediate = beat.get("chapter")
            source = beat.get("source")
            addr = source.get("addr") if isinstance(source, dict) else None
            # loadch's immediate argument identifies the map/chapter being
            # loaded, but it does not prove the FDTXT table used by later
            # dialog calls.  ch00 demonstrates the two can differ by one or
            # more.  Reset rather than manufacture FDTXT_NNN from it.
            current_source = None
            context_origin = addr if isinstance(addr, str) else None
            context_reason = "loadch does not prove a FDTXT resource; explicit mapping required"
            loadch_events.append(
                {
                    "source_addr": addr,
                    "chapter": immediate if is_nonnegative_int(immediate) else None,
                    "kind": "fdtxt_context_unresolved",
                    "message": context_reason,
                }
            )
            continue
        if op != "dialog":
            continue

        source = beat.get("source")
        addr = source.get("addr") if isinstance(source, dict) else None
        text_index = beat.get("text_index")
        if not isinstance(addr, str) or not addr:
            skipped.append(dialog_diagnostic(beat, current_source, "missing_source_addr", "dialog has no source.addr"))
            continue
        if not is_nonnegative_int(text_index):
            skipped.append(dialog_diagnostic(beat, current_source, "invalid_text_index", "dialog text_index is not a non-negative immediate integer"))
            continue
        if current_source is None:
            item = dialog_diagnostic(beat, None, "unproven_fdtxt_context", context_reason or "no proven active FDTXT resource")
            if context_origin is not None:
                item["context_origin_addr"] = context_origin
            skipped.append(item)
            continue
        mapping = unique_mappings.get(current_source)
        if mapping is None:
            skipped.append(dialog_diagnostic(beat, current_source, "unmapped_source_dat", "source has no count-aligned script mapping"))
            continue
        if not mapping:
            skipped.append(dialog_diagnostic(beat, current_source, "ambiguous_source_dat", "source does not have exactly one count-aligned script mapping"))
            continue
        targets = mapping_targets(mapping, text_index)
        if targets is None:
            skipped.append(dialog_diagnostic(beat, current_source, "out_of_range_text_index", "string index is absent or malformed in the selected mapping"))
            continue
        if len(targets) != 1:
            skipped.append(dialog_diagnostic(beat, current_source, "multi_scene_target", "one dialog string crosses scenes; runtime adapter required"))
            continue
        script = mapping["script"]
        if addr in contexts:
            raise ValueError(f"handler {handler_path} repeats dialog source address {addr}")
        contexts[addr] = {"source_dat": current_source, "script": script}

    binding = {
        "schema_version": SCHEMA_VERSION,
        "handler_script": relative_reference(handler_path, output_dir),
        "story_index_map": relative_reference(map_path, output_dir),
        "dialogue_contexts": contexts,
        "overrides": {},
    }
    report = {
        "handler_file": handler_path.name,
        "handler": handler.get("handler"),
        "chapter": chapter,
        "initial_source_dat": initial_source,
        "initial_context_kind": "handler_chapter" if chapter != 0 else "chapter_zero_requires_explicit_mapping",
        "dialogue_context_count": len(contexts),
        "skipped_dialogs": skipped,
        "loadch_events": loadch_events,
    }
    return binding, report


def build_exports(
    handlers_dir: Path, map_path: Path, output_dir: Path
) -> tuple[list[tuple[Path, dict[str, Any]]], dict[str, Any]]:
    """Plan every output and diagnostics record before touching output_dir."""

    unique_mappings = load_unique_mappings(map_path)
    planned: list[tuple[Path, dict[str, Any]]] = []
    reports: list[dict[str, Any]] = []
    for handler_path in handler_files(handlers_dir):
        binding, report = export_handler(handler_path, output_dir, map_path, unique_mappings)
        planned.append((output_dir / handler_path.name, binding))
        reports.append(report)

    mapped = sum(report["dialogue_context_count"] for report in reports)
    skipped = sum(len(report["skipped_dialogs"]) for report in reports)
    diagnostics = {
        "schema_version": SCHEMA_VERSION,
        "generator": "tools/export_handler_dialogue_bindings.py",
        "mapping_policy": "one_count_aligned_script_per_active_source",
        "handlers_dir": str(handlers_dir),
        "story_index_map": str(map_path),
        "summary": {
            "handler_count": len(reports),
            "dialogue_context_count": mapped,
            "skipped_dialogue_count": skipped,
        },
        "handlers": reports,
    }
    planned.append((output_dir / DIAGNOSTICS_NAME, diagnostics))
    return planned, diagnostics


def write_exports(planned: list[tuple[Path, dict[str, Any]]], indent: int) -> None:
    """Write a fully preflighted plan, refusing to replace any target file."""

    existing = [path for path, _ in planned if path.exists()]
    if existing:
        files = ", ".join(str(path) for path in existing)
        raise FileExistsError(f"refusing to overwrite existing output(s): {files}")
    # The preflight above makes each ordinary write non-destructive.  A failed
    # filesystem write may leave only newly created skeletons, never overwrite
    # authored material; re-run into a fresh directory in that case.
    for path, data in planned:
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(data, ensure_ascii=False, indent=indent) + "\n", encoding="utf-8")


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("handlers_dir", type=Path, help="directory containing chNN_pre/post handler JSON")
    parser.add_argument("story_index_map", type=Path, help="count-aligned.json from export_story_index_map.py")
    parser.add_argument("output_dir", type=Path, help="new or empty directory for generated binding JSON")
    parser.add_argument("--indent", type=int, default=2, help="JSON indent (default: 2)")
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    try:
        planned, diagnostics = build_exports(args.handlers_dir, args.story_index_map, args.output_dir)
        write_exports(planned, args.indent)
    except (OSError, ValueError, FileExistsError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    summary = diagnostics["summary"]
    print(
        f"wrote {summary['handler_count']} binding skeletons to {args.output_dir}: "
        f"{summary['dialogue_context_count']} dialogue contexts, "
        f"{summary['skipped_dialogue_count']} skipped dialogs (see {DIAGNOSTICS_NAME})"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
