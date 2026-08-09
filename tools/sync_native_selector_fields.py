#!/usr/bin/env python3
"""Synchronize proven native constructor fields into map unit assets.

The editable map assets contain manual corrections which a full export_units.py
regeneration must not overwrite.  FDFIELD construction at 0x10d7f..0x10efc
does, however, close three fields for every scripted roster entry:

* roster b1 -> 0x11019 raw key -> runtime unit+2 cache slot
* roster b0 -> runtime unit+6
* roster b1 -> runtime unit+7 / unit+8 raw visual selector bytes
* roster b13..b16 -> runtime unit+0x1a..+0x1d command-mask bytes
* the bounded b1-selected constructor record -> runtime +0x1f/+0x20
* constructor formulas -> runtime max HP +0x42 and max MP +0x46
* roster b17/b18/b19 -> runtime +0x34/+0x35/+0x36

This tool preserves every existing asset field and updates only
the fields above. The optional native table input adds the exact b1-selected
constructor record, independently proven raw race/class bytes and runtime
``+0x42/+0x46``. Keeping the selected record beside each editable unit makes
the constructor projection reproducible without consulting FD2.EXE at runtime.

Optional constructor provenance can be synchronized without touching unrelated
manual asset fields:

  python3 tools/sync_native_selector_fields.py extracted/raw remake/assets/maps \
    --native-tables docs/data/exe_tables/native_unit_tables.json --check

Usage:
  python3 tools/sync_native_selector_fields.py extracted/raw remake/assets/maps --check
  python3 tools/sync_native_selector_fields.py extracted/raw remake/assets/maps --write
"""
import argparse
import json
from pathlib import Path
import sys

import parse_field
import export_units


REFERENCE_FILES = Path(__file__).resolve().parent.parent / "docs/data/fd2-reference-files.json"


FIELDS = (
    "map_selector_key",
    "native_record_byte6",
    "battle_fig",
    "native_record_byte8",
    "native_record_byte3d",
    "native_record_death_effect",
    "native_source_byte3",
    "native_source_byte20",
    "native_source_byte25",
    "native_position_record",
    "initial_command_mask",
    "native_record_byte34",
    "native_record_byte35",
    "native_record_byte36",
)


def validate_native_tables(native_tables, reference_files):
    manifest = json.loads(reference_files.read_text(encoding="utf-8"))
    reference = next(
        (item for item in manifest.get("files", []) if item.get("file") == "FD2.EXE"),
        None,
    )
    if reference is None:
        raise ValueError(f"{reference_files}: missing FD2.EXE reference")
    fields = (
        ("source_size", "size"),
        ("source_md5", "md5"),
        ("source_sha256", "sha256"),
    )
    if native_tables.get("source") != "FD2.EXE":
        raise ValueError("native constructor tables have an invalid source")
    for table_field, reference_field in fields:
        if native_tables.get(table_field) != reference.get(reference_field):
            raise ValueError(
                f"native constructor tables {table_field}="
                f"{native_tables.get(table_field)!r}, expected "
                f"{reference.get(reference_field)!r}"
            )


def expected_units(raw, map_index, native_tables=None):
    expected = []
    parsed = parse_field.parse_map(str(raw), map_index)
    positions = parsed["positions"]
    for index, unit in enumerate(parsed["units"]):
        raw_unit_key = unit["raw_unit_key"]
        if index >= len(positions):
            raise ValueError(
                f"map{map_index}: unit {index} has no six-byte position record"
            )
        position = positions[index]
        item = {
            "map_selector_key": unit["native_map_selector_key"],
            "native_record_byte6": unit["native_record_byte6"],
            "battle_fig": raw_unit_key,
            # Runtime +8 receives the same raw b1 byte, but only player
            # persistent records have the separate identity lookup contract.
            "native_record_byte8": raw_unit_key,
            "native_record_byte3d": unit["native_record_byte3d"],
            "native_record_death_effect": unit["native_record_death_effect"],
            "native_source_byte3": unit["native_source_byte3"],
            "native_source_byte20": unit["native_source_byte20"],
            "native_source_byte25": unit["native_source_byte25"],
            "native_position_record": {
                "x_word": position[0],
                "y_word": position[1],
                "raw_key": position[2],
            },
            "initial_command_mask": unit["initial_command_mask"],
            "native_record_byte34": unit["native_record_byte34"],
            "native_record_byte35": unit["native_record_byte35"],
            "native_record_byte36": unit["native_record_byte36"],
        }
        if native_tables is not None:
            constructor = export_units.native_constructor_for_raw_unit_key(
                native_tables, raw_unit_key
            )
            if constructor is not None:
                item["native_constructor"] = constructor
                item["native_record_race"] = constructor["record"][0]
                item["native_record_class"] = constructor["record"][1]
            word42 = export_units.native_record_word42_for_raw_unit_key(
                native_tables, raw_unit_key, unit["lv"]
            )
            # Unsupported selector/table provenance remains absent.  The
            # runtime loader therefore keeps native predicates fail-closed.
            if word42 is not None:
                item["native_record_word42"] = word42
            word46 = export_units.native_record_word46_for_raw_unit_key(
                native_tables, raw_unit_key, unit["lv"]
            )
            if word46 is not None:
                item["native_record_word46"] = word46
        expected.append(item)
    return expected


def sync_asset(
    raw,
    asset_path,
    write,
    native_tables=None,
    rewrite_map_selector_key=False,
):
    asset = json.loads(asset_path.read_text(encoding="utf-8"))
    map_index = asset.get("map")
    if not isinstance(map_index, int):
        raise ValueError(f"{asset_path}: missing integer map")
    units = asset.get("units")
    if not isinstance(units, list):
        raise ValueError(f"{asset_path}: missing units list")
    expected = expected_units(raw, map_index, native_tables)
    if len(units) != len(expected):
        raise ValueError(
            f"{asset_path}: asset has {len(units)} units but FDFIELD map {map_index} has {len(expected)}"
        )

    changed = 0
    for index, (unit, native) in enumerate(zip(units, expected)):
        if not isinstance(unit, dict):
            raise ValueError(f"{asset_path}: unit {index} is not an object")
        fields = list(FIELDS)
        legacy_identity = unit.get("native_identity")
        if legacy_identity is not None:
            if legacy_identity != native["native_record_byte8"]:
                raise ValueError(
                    f"{asset_path}: unit {index} legacy native_identity="
                    f"{legacy_identity!r}, expected raw +8 "
                    f"{native['native_record_byte8']!r}"
                )
            if write:
                del unit["native_identity"]
            changed += 1
        if "native_record_race" in native:
            fields.extend((
                "native_constructor",
                "native_record_race",
                "native_record_class",
            ))
        if "native_record_word42" in native:
            fields.append("native_record_word42")
        if "native_record_word46" in native:
            fields.append("native_record_word46")
        for field in fields:
            current = unit.get(field)
            value = native[field]
            if current is not None and current != value:
                if field == "map_selector_key" and rewrite_map_selector_key:
                    unit[field] = value
                    changed += 1
                    continue
                raise ValueError(
                    f"{asset_path}: unit {index} {field}={current!r}, expected native {value!r}"
                )
            if current is None:
                unit[field] = value
                changed += 1

    if write and changed:
        asset_path.write_text(
            json.dumps(asset, ensure_ascii=False, separators=(",", ":")), encoding="utf-8"
        )
    return map_index, changed


def main(argv):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("raw", type=Path)
    parser.add_argument("assets", type=Path)
    parser.add_argument("--native-tables", type=Path,
                        help="raw constructor tables from extract_native_unit_tables.py")
    parser.add_argument(
        "--reference-files",
        type=Path,
        default=REFERENCE_FILES,
        help="hash manifest used to bind native tables to the supported FD2.EXE",
    )
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--write", action="store_true")
    parser.add_argument(
        "--rewrite-map-selector-key",
        action="store_true",
        help=(
            "rewrite only stale map_selector_key values from the proven "
            "FDFIELD b1 -> 0x11019 source; requires --write"
        ),
    )
    args = parser.parse_args(argv)
    if args.rewrite_map_selector_key and not args.write:
        parser.error("--rewrite-map-selector-key requires --write")

    paths = sorted(args.assets.glob("map*/map*_units.json"), key=lambda p: int(p.parent.name[3:]))
    if not paths:
        raise ValueError(f"no map unit assets under {args.assets}")
    native_tables = None
    if args.native_tables is not None:
        native_tables = json.loads(args.native_tables.read_text(encoding="utf-8"))
        validate_native_tables(native_tables, args.reference_files)
    changed = 0
    for path in paths:
        map_index, count = sync_asset(
            args.raw,
            path,
            args.write,
            native_tables,
            args.rewrite_map_selector_key,
        )
        changed += count
        print(f"map{map_index}: {'updated' if args.write else 'verified'} ({count} pending native field migrations)")
    if args.check and changed:
        raise ValueError(f"{changed} native field migrations are pending; run with --write")
    print(f"{len(paths)} map assets {'updated' if args.write else 'verified'}; {changed} fields {'migrated' if args.write else 'pending'}")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except (OSError, ValueError, json.JSONDecodeError) as error:
        print(f"error: {error}", file=sys.stderr)
        sys.exit(1)
