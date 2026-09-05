#!/usr/bin/env python3
"""將 3×1 現代靜態人物母稿轉成 FDICON 三張二值透明相位。"""

import argparse
from pathlib import Path

from PIL import Image

from process_modern_map_sprite_sheet import keep_largest_component, transparent_background


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--group", type=int, required=True)
    parser.add_argument("--public", type=Path, required=True)
    parser.add_argument("--private", type=Path, required=True)
    args = parser.parse_args()
    if args.group not in {48, 66, 67, 74, 75}:
        raise SystemExit("static three-phase contract is proven only for groups 48, 66, 67, 74, and 75")

    source = Image.open(args.source)
    source = source.convert("RGBA") if "A" in source.getbands() else source.convert("RGB")
    args.public.mkdir(parents=True, exist_ok=True)
    args.private.mkdir(parents=True, exist_ok=True)
    master_name = f"fdicon-{args.group:03d}-style-a-v1-master.png"
    source.save(args.public / master_name, optimize=False)
    source.save(args.private / master_name, optimize=False)

    boundaries = [round(index * source.width / 3) for index in range(4)]
    for frame in range(3):
        cell = source.crop((boundaries[frame], 0, boundaries[frame + 1], source.height))
        sprite = keep_largest_component(transparent_background(cell))
        box = sprite.getbbox()
        if box is None:
            raise SystemExit(f"frame {frame}: empty sprite")
        sprite = sprite.crop(box)
        scale = min(22 / sprite.width, 22 / sprite.height)
        size = (max(1, round(sprite.width * scale)), max(1, round(sprite.height * scale)))
        sprite = sprite.resize(size, Image.Resampling.NEAREST)
        frame_image = Image.new("RGBA", (24, 24))
        frame_image.alpha_composite(sprite, ((24 - size[0]) // 2, 23 - size[1]))
        if not set(frame_image.getchannel("A").getdata()).issubset({0, 255}):
            raise SystemExit(f"frame {frame}: non-binary alpha")
        name = f"fdicon-{args.group:03d}-style-a-f{frame:02d}.png"
        frame_image.save(args.public / name, optimize=False)
        frame_image.save(args.private / name, optimize=False)
        print(frame, box, size, name)


if __name__ == "__main__":
    main()
