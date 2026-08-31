#!/usr/bin/env python3
"""依公開 catalog 的 allow-list 建立私人現代主題素材包。"""

import argparse
import hashlib
import json
import shutil
from pathlib import Path


def digest(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, required=True)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--destination", type=Path, required=True)
    args = parser.parse_args()

    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    if catalog.get("theme_id") != "modern-handpainted-a":
        raise SystemExit("unexpected theme_id")

    names: set[str] = set()
    runtime_hashes: dict[str, str] = {}
    for asset in catalog.get("assets", []):
        if "file" in asset:
            names.add(asset["file"])
            runtime_hashes[asset["file"]] = asset["sha256"]
        if "master_file" in asset:
            names.add(asset["master_file"])
        for name, sha256 in zip(asset.get("files", []), asset.get("frame_sha256", [])):
            names.add(name)
            runtime_hashes[name] = sha256

    args.destination.mkdir(parents=True, exist_ok=True)
    records = []
    for name in sorted(names):
        if Path(name).name != name:
            raise SystemExit(f"unsafe catalog filename: {name}")
        source = args.source / name
        if not source.is_file():
            raise SystemExit(f"missing catalog file: {source}")
        actual = digest(source)
        expected = runtime_hashes.get(name)
        if expected and actual != expected:
            raise SystemExit(f"digest mismatch: {name}: {actual} != {expected}")
        target = args.destination / name
        shutil.copyfile(source, target)
        records.append({"file": name, "sha256": actual, "size": source.stat().st_size})

    shutil.copyfile(args.catalog, args.destination / "catalog.json")
    manifest = {
        "schema_version": 1,
        "theme_id": catalog["theme_id"],
        "source_catalog": "fd2_re/remake/assets/themes/modern/catalog.json",
        "file_count": len(records),
        "files": records,
    }
    (args.destination / "private-manifest.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    print(f"synced {len(records)} catalog-listed file(s)")


if __name__ == "__main__":
    main()
