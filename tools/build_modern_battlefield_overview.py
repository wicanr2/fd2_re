#!/usr/bin/env python3
"""從正式 map.json 與現代圖集產生全戰役戰場總覽。"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from PIL import Image, ImageDraw


def render_map(map_dir: Path, tileset_path: Path) -> Image.Image:
    data = json.loads((map_dir / "map.json").read_text(encoding="utf-8"))
    atlas = Image.open(tileset_path).convert("RGBA")
    tw, th = data["tileW"], data["tileH"]
    cols = data.get("cols") or atlas.width // tw
    canvas = Image.new("RGBA", (data["w"] * tw, data["h"] * th))
    for index, tile_id in enumerate(data["tiles"]):
        sx, sy = tile_id % cols * tw, tile_id // cols * th
        tile = atlas.crop((sx, sy, sx + tw, sy + th))
        x, y = index % data["w"] * tw, index // data["w"] * th
        canvas.alpha_composite(tile, (x, y))
    return canvas


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--maps-root", type=Path, required=True)
    parser.add_argument("--asset-root", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    columns, cell_w, cell_h = 5, 260, 210
    rows = (33 + columns - 1) // columns
    sheet = Image.new("RGB", (columns * cell_w, rows * cell_h), (18, 22, 18))
    draw = ImageDraw.Draw(sheet)
    for map_id in range(33):
        rendered = render_map(
            args.maps_root / f"map{map_id}",
            args.asset_root / f"map{map_id}-tileset-style-a.png",
        ).convert("RGB")
        rendered.thumbnail((cell_w - 16, cell_h - 30), Image.Resampling.LANCZOS)
        cell_x = map_id % columns * cell_w
        cell_y = map_id // columns * cell_h
        x = cell_x + (cell_w - rendered.width) // 2
        y = cell_y + 22 + (cell_h - 28 - rendered.height) // 2
        sheet.paste(rendered, (x, y))
        draw.text((cell_x + 8, cell_y + 5), f"map{map_id}", fill=(232, 222, 180))
    args.output.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(args.output, optimize=True)
    print(f"wrote {args.output} {sheet.width}x{sheet.height}")


if __name__ == "__main__":
    main()
