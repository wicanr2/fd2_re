#!/usr/bin/env python3
"""將 3×4 現代人物母稿轉成 FDICON 12 幀二值透明 PNG。"""

import argparse
from collections import deque
from pathlib import Path

from PIL import Image


def transparent_background(cell: Image.Image) -> Image.Image:
    """只移除與格子邊界連通的高亮中性背景，保留人物內部白色區域。"""
    if "A" in cell.getbands() and cell.getchannel("A").getextrema()[0] == 0:
        rgba = cell.convert("RGBA")
        output = Image.new("RGBA", rgba.size)
        source = rgba.load()
        target = output.load()
        for y in range(rgba.height):
            for x in range(rgba.width):
                red, green, blue, alpha = source[x, y]
                target[x, y] = (red, green, blue, 255) if alpha else (0, 0, 0, 0)
        return output
    cell = cell.convert("RGB")
    width, height = cell.size
    pixels = cell.load()
    candidate = [[False] * width for _ in range(height)]
    for y in range(height):
        for x in range(width):
            red, green, blue = pixels[x, y]
            candidate[y][x] = min(red, green, blue) >= 220 and max(red, green, blue) - min(red, green, blue) <= 5

    queue: deque[tuple[int, int]] = deque()
    seen = [[False] * width for _ in range(height)]
    for x in range(width):
        for y in (0, height - 1):
            if candidate[y][x] and not seen[y][x]:
                seen[y][x] = True
                queue.append((x, y))
    for y in range(height):
        for x in (0, width - 1):
            if candidate[y][x] and not seen[y][x]:
                seen[y][x] = True
                queue.append((x, y))
    while queue:
        x, y = queue.popleft()
        for next_x, next_y in ((x - 1, y), (x + 1, y), (x, y - 1), (x, y + 1)):
            if (
                0 <= next_x < width
                and 0 <= next_y < height
                and candidate[next_y][next_x]
                and not seen[next_y][next_x]
            ):
                seen[next_y][next_x] = True
                queue.append((next_x, next_y))

    output = Image.new("RGBA", cell.size)
    target = output.load()
    for y in range(height):
        for x in range(width):
            red, green, blue = pixels[x, y]
            target[x, y] = (0, 0, 0, 0) if seen[y][x] else (red, green, blue, 255)
    return output


def keep_largest_component(image: Image.Image) -> Image.Image:
    width, height = image.size
    alpha = image.getchannel("A")
    seen: set[tuple[int, int]] = set()
    components: list[list[tuple[int, int]]] = []
    for y in range(height):
        for x in range(width):
            if alpha.getpixel((x, y)) == 0 or (x, y) in seen:
                continue
            queue = deque([(x, y)])
            seen.add((x, y))
            component: list[tuple[int, int]] = []
            while queue:
                pixel_x, pixel_y = queue.popleft()
                component.append((pixel_x, pixel_y))
                for next_x, next_y in (
                    (pixel_x - 1, pixel_y),
                    (pixel_x + 1, pixel_y),
                    (pixel_x, pixel_y - 1),
                    (pixel_x, pixel_y + 1),
                ):
                    if (
                        0 <= next_x < width
                        and 0 <= next_y < height
                        and alpha.getpixel((next_x, next_y))
                        and (next_x, next_y) not in seen
                    ):
                        seen.add((next_x, next_y))
                        queue.append((next_x, next_y))
            components.append(component)
    if not components:
        raise ValueError("empty sprite cell after background removal")
    source = image.load()
    output = Image.new("RGBA", image.size)
    target = output.load()
    for x, y in max(components, key=len):
        target[x, y] = source[x, y]
    return output


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--group", type=int, required=True)
    parser.add_argument("--public", type=Path, required=True)
    parser.add_argument("--private", type=Path, required=True)
    args = parser.parse_args()
    if not 0 <= args.group < 96:
        raise SystemExit("group must be in range 0..95")
    source = Image.open(args.source)
    source = source.convert("RGBA") if "A" in source.getbands() else source.convert("RGB")
    args.public.mkdir(parents=True, exist_ok=True)
    args.private.mkdir(parents=True, exist_ok=True)
    master_name = f"fdicon-{args.group:03d}-style-a-v1-master.png"
    source.save(args.public / master_name, optimize=False)
    source.save(args.private / master_name, optimize=False)

    boundaries_x = [round(index * source.width / 3) for index in range(4)]
    boundaries_y = [round(index * source.height / 4) for index in range(5)]
    for row in range(4):
        for column in range(3):
            frame = row * 3 + column
            cell = source.crop(
                (
                    boundaries_x[column],
                    boundaries_y[row],
                    boundaries_x[column + 1],
                    boundaries_y[row + 1],
                )
            )
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
