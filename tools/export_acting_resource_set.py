#!/usr/bin/env python3
"""Export editable acting behaviour from a decoder transcript.

The input is the human-readable output of ``decode_acting.py``.  It contains
only a decoded account of a runtime resource table; this exporter writes an
editable JSON frame set (beat count, special flag, original roster slot and
pose), never original bytes, pointers, or executable data.

Example:

    python3 tools/export_acting_resource_set.py \
      extracted/dosbox_dump/acting_decoded/acting_decoded_throne.txt \
      remake/assets/cutscenes/acting/map32.json
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path


HEADER = re.compile(r"^=== acting id 0x([0-9a-fA-F]+) \(ptr=.*\) ===$")
FRAME = re.compile(
    r"^\s+frame\[(\d+)\] 拍數=(\d+) \[bit7=(0|1)[^]]*\] N=(\d+):(?: (.*))?$"
)
UNIT = re.compile(r"\(unit=(\d+),pose=(\d+)\)")


def parse(path: Path) -> dict[str, list[dict[str, object]]]:
    """Parse complete decoder sections, rejecting truncated/malformed frames."""

    resources: dict[str, list[dict[str, object]]] = {}
    current: int | None = None
    expected: int | None = None
    frames: list[dict[str, object]] = []

    def finish() -> None:
        nonlocal current, expected, frames
        if current is None:
            return
        if expected is None or len(frames) != expected:
            raise ValueError(
                f"acting id {current} has {len(frames)} frames, expected {expected}"
            )
        resources[str(current)] = frames
        current, expected, frames = None, None, []

    for raw_line in path.read_text(encoding="utf-8").splitlines():
        header = HEADER.match(raw_line)
        if header:
            finish()
            current = int(header.group(1), 16)
            continue
        if current is None:
            continue
        if "(指標超出本輪 dump 範圍" in raw_line:
            current, expected, frames = None, None, []
            continue
        if "帧数=" in raw_line:
            matched = re.search(r"帧数=(\d+)", raw_line)
            if matched is None:
                raise ValueError(f"cannot parse frame count: {raw_line!r}")
            expected = int(matched.group(1))
            continue
        frame = FRAME.match(raw_line)
        if frame is None:
            continue
        if expected is None:
            raise ValueError(f"acting id {current} has frame before declared count")
        index, beats, bit7, declared_units, pairs = frame.groups()
        if int(index) != len(frames):
            raise ValueError(f"acting id {current} frame index {index} is not contiguous")
        units = [
            {"slot": int(slot), "pose": int(pose)}
            for slot, pose in UNIT.findall(pairs or "")
        ]
        if len(units) != int(declared_units):
            raise ValueError(
                f"acting id {current} frame {index} has {len(units)} units, expected {declared_units}"
            )
        item: dict[str, object] = {"beats": int(beats), "units": units}
        if bit7 == "1":
            item["special"] = True
        frames.append(item)
    finish()
    if not resources:
        raise ValueError(f"no complete acting resources found in {path}")
    return dict(sorted(resources.items(), key=lambda item: int(item[0])))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path, help="decode_acting.py text transcript")
    parser.add_argument("output", type=Path, help="editable acting-resource JSON")
    args = parser.parse_args()

    result = {"schema_version": 1, "resources": parse(args.input)}
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"exported {len(result['resources'])} acting resources -> {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
