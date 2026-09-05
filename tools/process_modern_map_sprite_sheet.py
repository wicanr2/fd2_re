#!/usr/bin/env python3
"""將 3×4 現代人物母稿轉成 FDICON 12 幀二值透明 PNG。"""

import argparse
from collections import deque
from pathlib import Path

from PIL import Image


def parse_chroma_key(value: str | None) -> tuple[int, int, int] | None:
    if value is None:
        return None
    value = value.removeprefix("#")
    if len(value) != 6:
        raise ValueError("--chroma-key must use #RRGGBB")
    try:
        return tuple(int(value[index:index + 2], 16) for index in (0, 2, 4))
    except ValueError as exc:
        raise ValueError("--chroma-key must use #RRGGBB") from exc


def transparent_background(
    cell: Image.Image,
    chroma_key: tuple[int, int, int] | None = None,
    chroma_tolerance: int = 0,
) -> Image.Image:
    """只移除與格子邊界連通的背景，保留人物輪廓內部的同色區域。"""
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
    tolerance_squared = chroma_tolerance * chroma_tolerance
    dominant_channel = None
    if chroma_key is not None:
        key_max = max(chroma_key)
        key_index = chroma_key.index(key_max)
        key_others = [value for index, value in enumerate(chroma_key) if index != key_index]
        if key_max >= 192 and key_max >= max(key_others) * 2:
            dominant_channel = key_index
    for y in range(height):
        for x in range(width):
            red, green, blue = pixels[x, y]
            if chroma_key is None:
                candidate[y][x] = (
                    min(red, green, blue) >= 220
                    and max(red, green, blue) - min(red, green, blue) <= 5
                )
            else:
                color = (red, green, blue)
                distance_match = sum(
                    (value - key_value) ** 2
                    for value, key_value in zip(color, chroma_key)
                ) <= tolerance_squared
                spill_match = False
                if dominant_channel is not None:
                    dominant = color[dominant_channel]
                    others = [value for index, value in enumerate(color) if index != dominant_channel]
                    saturation = (dominant - min(color)) / dominant if dominant else 0
                    spill_match = (
                        dominant >= 20
                        and dominant >= max(others) * 1.3
                        and saturation >= 0.55
                    )
                candidate[y][x] = distance_match or spill_match

    if chroma_key is not None:
        # 明示綠幕的前提是角色不得使用 key 色；輪廓內的腋下、腿間等封閉空隙
        # 仍是背景，故所有命中色域的像素都必須移除。
        seen = candidate
    else:
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
    parser.add_argument(
        "--chroma-key",
        metavar="#RRGGBB",
        help="只移除與邊界連通、落在指定 RGB 色差容差內的綠幕背景",
    )
    parser.add_argument(
        "--chroma-tolerance",
        type=int,
        default=0,
        help="RGB 歐幾里得距離容差，範圍 0..441；只可搭配 --chroma-key",
    )
    parser.add_argument(
        "--repeat-frame",
        action="append",
        default=[],
        metavar="DEST:SOURCE",
        help="原版明確重複幀時，將 SOURCE 的輸出位元組複製到 DEST",
    )
    args = parser.parse_args()
    if not 0 <= args.group < 96:
        raise SystemExit("group must be in range 0..95")
    if not 0 <= args.chroma_tolerance <= 441:
        raise SystemExit("--chroma-tolerance must be in range 0..441")
    if args.chroma_tolerance and not args.chroma_key:
        raise SystemExit("--chroma-tolerance requires --chroma-key")
    try:
        chroma_key = parse_chroma_key(args.chroma_key)
    except ValueError as exc:
        raise SystemExit(str(exc)) from exc
    source = Image.open(args.source)
    source = source.convert("RGBA") if "A" in source.getbands() else source.convert("RGB")
    args.public.mkdir(parents=True, exist_ok=True)
    args.private.mkdir(parents=True, exist_ok=True)
    master_name = f"fdicon-{args.group:03d}-style-a-v1-master.png"
    master = transparent_background(source, chroma_key, args.chroma_tolerance)
    master.save(args.public / master_name, optimize=False)
    master.save(args.private / master_name, optimize=False)

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
            sprite = keep_largest_component(
                transparent_background(cell, chroma_key, args.chroma_tolerance)
            )
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

    for repeat in args.repeat_frame:
        try:
            destination_text, source_text = repeat.split(":", 1)
            destination, source_frame = int(destination_text), int(source_text)
        except ValueError as exc:
            raise SystemExit(f"invalid --repeat-frame {repeat!r}; want DEST:SOURCE") from exc
        if destination == source_frame or destination not in range(12) or source_frame not in range(12):
            raise SystemExit(f"invalid --repeat-frame {repeat!r}; frames must be distinct and in 0..11")
        source_name = f"fdicon-{args.group:03d}-style-a-f{source_frame:02d}.png"
        destination_name = f"fdicon-{args.group:03d}-style-a-f{destination:02d}.png"
        for root in (args.public, args.private):
            (root / destination_name).write_bytes((root / source_name).read_bytes())
        print(f"repeat {destination:02d} <- {source_frame:02d}")


if __name__ == "__main__":
    main()
