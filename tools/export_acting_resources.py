#!/usr/bin/env python3
"""Export FD2 acting resources as editable, deterministic behaviour JSON.

The original executable stays an input outside version control.  This tool
reads its 106-entry acting directory and emits only the decoded frame
semantics used by the remake: duration, special-mode flag, original runtime
slot, and pose.  It deliberately does not write pointers, offsets, or source
bytes to the output.

Example:

    python3 tools/export_acting_resources.py \
      org_game/炎龍騎士團/FLAME2/FD2.EXE \
      remake/assets/cutscenes/acting/map32.json
"""

from __future__ import annotations

import argparse
import json
import struct
from pathlib import Path
from typing import Any


# FD2.EXE file-relative layout, verified against the acting getter's 106-entry
# table.  These are extractor implementation details only: never serialized.
DIRECTORY_OFFSET = 0x565D8
DATA_OFFSET = 0x53E00
RESOURCE_COUNT = 106


def read_u32(data: bytes, offset: int, label: str) -> int:
    if offset < 0 or offset + 4 > len(data):
        raise ValueError(f"{label} u32 at file+0x{offset:x} is outside input")
    return struct.unpack_from("<I", data, offset)[0]


def parse_resource(data: bytes, start: int, resource_id: int) -> list[dict[str, Any]]:
    """Decode one self-describing acting resource without retaining raw data."""

    if start >= len(data):
        raise ValueError(f"resource {resource_id} starts outside input")
    cursor = start
    frame_count = data[cursor]
    cursor += 1
    if frame_count == 0:
        raise ValueError(f"resource {resource_id} has no frames")

    frames: list[dict[str, Any]] = []
    for frame_index in range(frame_count):
        if cursor + 2 > len(data):
            raise ValueError(f"resource {resource_id} frame {frame_index} header is truncated")
        duration_raw, unit_count = data[cursor], data[cursor + 1]
        cursor += 2
        pair_bytes = unit_count * 2
        if cursor + pair_bytes > len(data):
            raise ValueError(f"resource {resource_id} frame {frame_index} units are truncated")
        units = [
            {"slot": data[pair], "pose": data[pair + 1]}
            for pair in range(cursor, cursor + pair_bytes, 2)
        ]
        cursor += pair_bytes
        frame: dict[str, Any] = {"beats": duration_raw & 0x7F, "units": units}
        if duration_raw & 0x80:
            frame["special"] = True
        frames.append(frame)
    return frames


def export_resources(executable: Path) -> dict[str, list[dict[str, Any]]]:
    """Decode all original global acting IDs (0 through 105)."""

    data = executable.read_bytes()
    directory_end = DIRECTORY_OFFSET + RESOURCE_COUNT * 4
    if directory_end > len(data):
        raise ValueError("acting directory is outside input")
    resources: dict[str, list[dict[str, Any]]] = {}
    for resource_id in range(RESOURCE_COUNT):
        relative = read_u32(data, DIRECTORY_OFFSET + resource_id * 4, "acting directory")
        start = DATA_OFFSET + relative
        resources[str(resource_id)] = parse_resource(data, start, resource_id)
    return resources


def render(resources: dict[str, list[dict[str, Any]]]) -> str:
    """Use one canonical representation so reruns produce byte-identical JSON."""

    return json.dumps(
        {"schema_version": 1, "resources": resources},
        ensure_ascii=False,
        indent=2,
    ) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("executable", type=Path, help="local, user-supplied FD2.EXE")
    parser.add_argument("output", type=Path, help="editable acting-resource JSON")
    parser.add_argument(
        "--check",
        action="store_true",
        help="fail unless output already equals the deterministic regenerated content",
    )
    args = parser.parse_args()

    output = render(export_resources(args.executable))
    if args.check:
        if not args.output.is_file() or args.output.read_text(encoding="utf-8") != output:
            raise SystemExit(f"{args.output} is not current; rerun this extractor")
        print(f"verified {RESOURCE_COUNT} acting resources in {args.output}")
        return 0

    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(output, encoding="utf-8")
    print(f"exported {RESOURCE_COUNT} acting resources -> {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
