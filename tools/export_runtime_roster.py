#!/usr/bin/env python3
"""Create an editable runtime-roster projection from an exported units file.

FDFIELD battle exports may retain placeholder rows beyond the original runtime
unit count.  A handler ``loadch`` needs a slot-stable roster, so this utility
copies only a proven prefix while preserving its original slot order.

Example:

    python3 tools/export_runtime_roster.py \
      remake/assets/maps/map32/map32_units.json 21 \
      remake/assets/cutscenes/rosters/map32_runtime.json
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("slot_count", type=int)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    if args.slot_count <= 0:
        parser.error("slot_count must be positive")
    try:
        source = json.loads(args.input.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        parser.error(f"cannot read {args.input}: {exc}")
    units = source.get("units") if isinstance(source, dict) else None
    if not isinstance(units, list) or len(units) < args.slot_count:
        parser.error(f"input has fewer than {args.slot_count} unit rows")
    source["units"] = units[: args.slot_count]
    source["runtime_slot_count"] = args.slot_count
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(source, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"exported {args.slot_count} stable runtime slots -> {args.output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
