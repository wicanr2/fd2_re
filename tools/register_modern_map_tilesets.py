#!/usr/bin/env python3
"""將全戰役現代地圖圖集以實際幾何與雜湊登錄到主題清冊。"""

from __future__ import annotations

import argparse
import hashlib
import json
import struct
from pathlib import Path


def png_size(path: Path) -> tuple[int, int]:
    raw = path.read_bytes()[:24]
    if len(raw) != 24 or raw[:8] != b"\x89PNG\r\n\x1a\n" or raw[12:16] != b"IHDR":
        raise ValueError(f"不是 PNG：{path}")
    return struct.unpack(">II", raw[16:24])


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, required=True)
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--asset-root", type=Path, required=True)
    args = parser.parse_args()

    document = json.loads(args.catalog.read_text(encoding="utf-8"))
    retained = [asset for asset in document["assets"] if asset.get("role") != "map_tileset_set"]
    entries = []
    for map_id in range(33):
        map_dir = args.maps_root / f"map{map_id}"
        map_data = json.loads((map_dir / "map.json").read_text(encoding="utf-8"))
        filename = f"map{map_id}-tileset-style-a.png"
        asset = args.asset_root / filename
        width, height = png_size(asset)
        tile_width = map_data["tileW"]
        tile_height = map_data["tileH"]
        columns = map_data.get("cols") or width // tile_width
        if width % tile_width or height % tile_height:
            raise SystemExit(f"map{map_id} 圖集不能整除圖塊尺寸")
        tile_count = (height // tile_height) * columns
        if max(map_data["tiles"]) >= tile_count:
            raise SystemExit(f"map{map_id} 使用超出圖集的 tile ID")
        entries.append({
            "asset_id": f"modern.map{map_id}.tileset.style_a",
            "role": "map_tileset_set",
            "status": "runtime_candidate",
            "file": filename,
            "width": width,
            "height": height,
            "sha256": hashlib.sha256(asset.read_bytes()).hexdigest(),
            "source_refs": [
                f"remake/assets/maps/map{map_id}/map.json",
                f"remake/assets/maps/map{map_id}/tileset.png",
                "asset:modern.ch01.battlefield.style_a",
            ],
            "consumer_contract": "map_tileset_indexed_geometry_v1",
            "map_id": map_id,
            "tile_width": tile_width,
            "tile_height": tile_height,
            "columns": columns,
            "tile_count": tile_count,
        })

    insert_at = next(
        index + 1 for index, asset in enumerate(retained)
        if asset.get("asset_id") == "modern.ch01.battlefield.style_a"
    )
    document["assets"] = retained[:insert_at] + entries + retained[insert_at:]
    args.catalog.write_text(
        json.dumps(document, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )
    print(f"registered {len(entries)} modern map tilesets")


if __name__ == "__main__":
    main()
