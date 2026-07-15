#!/usr/bin/env python3
"""Export conservative FDTXT-string-to-editable-story mappings.

The original FDTXT format contains an offset table followed by strings.  One
offset-table string can contain multiple dialogue utterances, while the remake
story JSON stores one utterance per ``scenes[].lines[]`` element.  This tool
creates a *count-aligned* manifest only when the total number of logical
utterances in one raw FDTXT resource is exactly equal to the number of lines
in a story JSON that declares that resource in ``source_dat``.

It deliberately does not use fuzzy text matching, speaker IDs, or a guessed
chapter relationship.  Therefore a mismatch never creates a partial mapping:
it is reported in ``diagnostics`` for review instead.

Examples:

    python3 tools/export_story_index_map.py \
        extracted/raw/FDTXT remake/assets/story /tmp/story-index-map.json

    python3 tools/export_story_index_map.py raw/FDTXT story out.json --indent 2

The output has one ``script_mappings`` entry for each count-aligned story
file.  A mapping's ``targets`` is normally one scene, but remains a list so a
raw string which naturally crosses a scene boundary remains lossless.  Script
paths are relative to the supplied story directory.
"""

from __future__ import annotations

import argparse
import json
import struct
import sys
from collections import defaultdict
from pathlib import Path
from typing import Any


OPEN_GLYPH = 557
CONTROL_MIN = 0xFF00
STRING_END = 0xFFFF


def parse_fdtxt_strings(path: Path) -> list[list[int]]:
    """Return raw 16-bit words for every FDTXT offset-table string.

    The returned words exclude the terminating 0xFFFF.  Validation is kept
    strict: a malformed offset table raises ValueError instead of silently
    manufacturing a different string layout.
    """

    data = path.read_bytes()
    if len(data) < 2:
        raise ValueError("file is shorter than the first offset")
    first_offset = struct.unpack_from("<H", data, 0)[0]
    if first_offset == 0 or first_offset % 2:
        raise ValueError(f"invalid first offset 0x{first_offset:04x}")
    count = first_offset // 2
    if first_offset > len(data):
        raise ValueError(
            f"first offset 0x{first_offset:04x} exceeds file length {len(data)}"
        )
    offsets = [struct.unpack_from("<H", data, index * 2)[0] for index in range(count)]
    if any(offset < first_offset or offset > len(data) for offset in offsets):
        raise ValueError("offset table contains an out-of-range offset")
    if offsets != sorted(offsets):
        raise ValueError("offset table is not monotonic")

    strings: list[list[int]] = []
    for index, start in enumerate(offsets):
        end = offsets[index + 1] if index + 1 < count else len(data)
        if (end - start) % 2:
            raise ValueError(f"string {index} has an odd byte length")
        words: list[int] = []
        for pos in range(start, end, 2):
            word = struct.unpack_from("<H", data, pos)[0]
            if word == STRING_END:
                break
            words.append(word)
        strings.append(words)
    return strings


def count_logical_utterances(words: list[int]) -> int:
    """Count dialogue starts, joining FFxx-wrapped visual rows into one line.

    In a dialogue stream a new utterance starts at a non-control chunk whose
    second word is the opening quote glyph (557).  Subsequent chunks until the
    next such start are page/wrap fragments of that same utterance.  This is a
    structural count only; it intentionally does not attach a character name
    to the leading operand because some resources reuse that operand by scene.
    """

    starts = 0
    chunk: list[int] = []
    for word in [*words, STRING_END]:
        if word >= CONTROL_MIN:
            if len(chunk) >= 2 and chunk[1] == OPEN_GLYPH:
                starts += 1
            chunk = []
        else:
            chunk.append(word)
    return starts


def load_story(path: Path, story_root: Path) -> dict[str, Any]:
    """Load the minimal story shape used by the conservative mapper."""

    data = json.loads(path.read_text(encoding="utf-8"))
    source_dat = data.get("source_dat")
    scenes = data.get("scenes")
    if not isinstance(source_dat, str) or not source_dat:
        raise ValueError("missing non-empty source_dat")
    if not isinstance(scenes, list):
        raise ValueError("missing scenes array")

    lines: list[dict[str, Any]] = []
    for scene_index, scene in enumerate(scenes):
        if not isinstance(scene, dict):
            raise ValueError(f"scene {scene_index} is not an object")
        # Some committed story files use a null label for an intentionally
        # unnamed scene.  scene_index remains the unambiguous locator, so do
        # not reject otherwise valid dialogue merely because presentation
        # metadata is absent.
        label = scene.get("label")
        scene_lines = scene.get("lines")
        if label is not None and not isinstance(label, str):
            raise ValueError(f"scene {scene_index} has a non-string label")
        if not isinstance(scene_lines, list):
            raise ValueError(f"scene {scene_index} lacks lines")
        for line_index, line in enumerate(scene_lines):
            if not isinstance(line, dict) or not isinstance(line.get("text"), str):
                raise ValueError(f"scene {scene_index} line {line_index} lacks text")
            lines.append(
                {
                    "scene": label,
                    "scene_index": scene_index,
                    "line": line_index,
                }
            )
    return {
        "script": path.relative_to(story_root).as_posix(),
        "source_dat": source_dat,
        "line_refs": lines,
    }


def group_targets(line_refs: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """Compress consecutive mapped lines from the same scene into targets."""

    targets: list[dict[str, Any]] = []
    for ref in line_refs:
        if (
            targets
            and targets[-1]["scene_index"] == ref["scene_index"]
            and targets[-1]["lines"][-1] + 1 == ref["line"]
        ):
            targets[-1]["lines"].append(ref["line"])
            continue
        targets.append(
            {
                "scene": ref["scene"],
                "scene_index": ref["scene_index"],
                "lines": [ref["line"]],
            }
        )
    return targets


def build_count_aligned_mapping(
    raw_strings: list[list[int]], story: dict[str, Any]
) -> dict[str, Any]:
    """Map every raw string to its exactly-sized sequential story line range."""

    raw_counts = [count_logical_utterances(words) for words in raw_strings]
    line_refs = story["line_refs"]
    if sum(raw_counts) != len(line_refs):
        raise ValueError("raw utterance count and story line count differ")

    cursor = 0
    mappings: list[dict[str, Any]] = []
    for string_index, utterance_count in enumerate(raw_counts):
        targets = group_targets(line_refs[cursor : cursor + utterance_count])
        mappings.append(
            {
                "string_index": string_index,
                "utterance_count": utterance_count,
                "targets": targets,
            }
        )
        cursor += utterance_count
    if cursor != len(line_refs):  # Defensive; equality above proves this today.
        raise AssertionError("count-aligned cursor did not consume all story lines")
    return {
        "script": story["script"],
        "source_dat": story["source_dat"],
        "status": "count_aligned",
        "raw_utterance_count": sum(raw_counts),
        "story_line_count": len(line_refs),
        "mappings": mappings,
    }


def build_manifest(raw_dir: Path, story_dir: Path) -> dict[str, Any]:
    """Build a deterministic manifest; mismatches are diagnostics only."""

    raw_files = sorted(raw_dir.glob("FDTXT_*.bin"))
    if not raw_files:
        raise ValueError(f"no FDTXT_*.bin files under {raw_dir}")
    story_files = sorted(story_dir.glob("*.json"))
    if not story_files:
        raise ValueError(f"no JSON story files under {story_dir}")

    diagnostics: list[dict[str, Any]] = []
    stories_by_source: dict[str, list[dict[str, Any]]] = defaultdict(list)
    for path in story_files:
        try:
            story = load_story(path, story_dir)
        except (OSError, ValueError, json.JSONDecodeError) as exc:
            diagnostics.append(
                {
                    "kind": "invalid_story_json",
                    "script": path.relative_to(story_dir).as_posix(),
                    "message": str(exc),
                }
            )
            continue
        stories_by_source[story["source_dat"]].append(story)

    resources: list[dict[str, Any]] = []
    seen_sources: set[str] = set()
    for raw_path in raw_files:
        source_dat = raw_path.stem
        seen_sources.add(source_dat)
        try:
            raw_strings = parse_fdtxt_strings(raw_path)
        except (OSError, ValueError) as exc:
            diagnostics.append(
                {
                    "kind": "invalid_raw_fdtxt",
                    "source_dat": source_dat,
                    "raw_file": raw_path.name,
                    "message": str(exc),
                }
            )
            continue

        raw_counts = [count_logical_utterances(words) for words in raw_strings]
        script_mappings: list[dict[str, Any]] = []
        candidates = stories_by_source.get(source_dat, [])
        if not candidates:
            diagnostics.append(
                {
                    "kind": "unreferenced_raw_resource",
                    "source_dat": source_dat,
                    "raw_file": raw_path.name,
                }
            )
        for story in candidates:
            story_count = len(story["line_refs"])
            if sum(raw_counts) != story_count:
                diagnostics.append(
                    {
                        "kind": "utterance_count_mismatch",
                        "source_dat": source_dat,
                        "script": story["script"],
                        "raw_utterance_count": sum(raw_counts),
                        "story_line_count": story_count,
                    }
                )
                continue
            script_mappings.append(build_count_aligned_mapping(raw_strings, story))
        resources.append(
            {
                "source_dat": source_dat,
                "raw_file": raw_path.name,
                "raw_string_count": len(raw_strings),
                "raw_utterance_count": sum(raw_counts),
                "script_mappings": script_mappings,
            }
        )

    for source_dat, stories in sorted(stories_by_source.items()):
        if source_dat not in seen_sources:
            for story in stories:
                diagnostics.append(
                    {
                        "kind": "missing_raw_resource",
                        "source_dat": source_dat,
                        "script": story["script"],
                    }
                )

    return {
        "schema_version": 1,
        "mapping_kind": "count_aligned_only",
        "raw_dir": str(raw_dir),
        "story_dir": str(story_dir),
        "resources": resources,
        "diagnostics": diagnostics,
    }


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("raw_fdtxt_dir", type=Path, help="directory containing FDTXT_NNN.bin")
    parser.add_argument("story_json_dir", type=Path, help="directory containing editable story JSON")
    parser.add_argument("output", type=Path, help="manifest path to write")
    parser.add_argument("--indent", type=int, default=2, help="JSON indent (default: 2)")
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    try:
        manifest = build_manifest(args.raw_fdtxt_dir, args.story_json_dir)
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(
        json.dumps(manifest, ensure_ascii=False, indent=args.indent) + "\n", encoding="utf-8"
    )
    mapped = sum(len(resource["script_mappings"]) for resource in manifest["resources"])
    print(
        f"wrote {args.output}: {mapped} count-aligned script mappings, "
        f"{len(manifest['diagnostics'])} diagnostics"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
