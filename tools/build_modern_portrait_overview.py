#!/usr/bin/env python3
"""由現代主題 catalog 產生可公開的角色頭像總攬圖。"""

import argparse
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, required=True)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    portraits = [
        asset
        for asset in catalog["assets"]
        if asset.get("role") == "story_portrait_frame" and asset.get("frame") == 0
    ]
    portraits.sort(key=lambda asset: asset["speaker_id"])
    if not portraits:
        raise SystemExit("no portrait frames")

    columns, cell_w, cell_h = 7, 132, 126
    rows = (len(portraits) + columns - 1) // columns
    canvas = Image.new("RGB", (columns * cell_w, 52 + rows * cell_h), "#07142d")
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default()
    draw.text((16, 12), "FD2 modern hand-painted portraits", fill="#f2d58b", font=font)
    draw.text((16, 30), f"registered runtime candidates: {len(portraits)}", fill="#b9c9e8", font=font)

    for index, asset in enumerate(portraits):
        x = (index % columns) * cell_w
        y = 52 + (index // columns) * cell_h
        image = Image.open(args.source / asset["file"]).convert("RGB").resize((96, 96))
        canvas.paste(image, (x + 18, y + 4))
        slug = asset["asset_id"].split(".")[1]
        label = f'{asset["speaker_id"]:02d} {slug}'
        draw.text((x + 18, y + 104), label, fill="#ffffff", font=font)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(args.output, optimize=True)
    print(f"wrote {args.output} with {len(portraits)} portrait(s)")


if __name__ == "__main__":
    main()
