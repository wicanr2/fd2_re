#!/usr/bin/env python3
"""依概念稿色彩重製既有地圖圖集，同時保持每格索引與透明度。"""

from __future__ import annotations

import argparse
import colorsys
from pathlib import Path

from PIL import Image, ImageEnhance, ImageFilter


def family(rgb: tuple[int, int, int]) -> str:
    r, g, b = (value / 255 for value in rgb)
    h, s, v = colorsys.rgb_to_hsv(r, g, b)
    if s < 0.12:
        return "neutral"
    if 0.19 <= h < 0.48:
        return "green"
    if 0.48 <= h < 0.72:
        return "blue"
    if h < 0.19 or h >= 0.94:
        return "earth"
    return "accent"


def concept_palette(concept: Image.Image) -> dict[str, list[tuple[int, int, int]]]:
    sample = concept.convert("RGB")
    sample.thumbnail((512, 512), Image.Resampling.LANCZOS)
    quantized = sample.quantize(colors=64, method=Image.Quantize.MEDIANCUT)
    palette = quantized.getpalette()
    counts = quantized.getcolors() or []
    groups: dict[str, list[tuple[int, int, int]]] = {
        "neutral": [], "green": [], "blue": [], "earth": [], "accent": []
    }
    for _, index in sorted(counts, reverse=True):
        rgb = tuple(palette[index * 3:index * 3 + 3])
        groups[family(rgb)].append(rgb)
    all_colors = [color for values in groups.values() for color in values]
    for name, values in groups.items():
        if not values:
            groups[name] = all_colors
    return groups


def nearest_by_lightness(rgb: tuple[int, int, int], candidates: list[tuple[int, int, int]]) -> tuple[int, int, int]:
    target = sum(rgb) / 3
    return min(candidates, key=lambda color: abs(sum(color) / 3 - target))


def repaint(source: Image.Image, concept: Image.Image, tile_width: int, tile_height: int) -> Image.Image:
    source = source.convert("RGBA")
    groups = concept_palette(concept)
    out = Image.new("RGBA", source.size)
    for top in range(0, source.height, tile_height):
        for left in range(0, source.width, tile_width):
            tile = source.crop((left, top, left + tile_width, top + tile_height))
            softened = tile.resize(
                (tile_width * 4, tile_height * 4), Image.Resampling.NEAREST
            ).filter(ImageFilter.GaussianBlur(1.15)).resize(
                (tile_width, tile_height), Image.Resampling.LANCZOS
            )
            softened = ImageEnhance.Contrast(softened).enhance(1.12)
            pixels = []
            for original, smooth in zip(tile.getdata(), softened.getdata()):
                if original[3] == 0:
                    pixels.append((0, 0, 0, 0))
                    continue
                # 原版用純黑作洞窟／畫布外區域；概念稿的中性色不可把這些空區
                # 提亮成灰色，否則吊橋與空中要塞會出現大片假地板。
                if max(original[:3]) <= 12:
                    pixels.append((0, 0, 0, original[3]))
                    continue
                base = nearest_by_lightness(original[:3], groups[family(original[:3])])
                # 保留原圖明暗與概念稿色相；少量平滑色避免退化成單純換色。
                luminance = max(0.55, min(1.35, (sum(smooth[:3]) + 1) / (sum(original[:3]) + 1)))
                color = tuple(max(0, min(255, round(channel * luminance))) for channel in base)
                pixels.append((*color, original[3]))
            painted = Image.new("RGBA", tile.size)
            painted.putdata(pixels)
            out.paste(painted, (left, top))
    return out


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--concept", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--tile-width", type=int, default=24)
    parser.add_argument("--tile-height", type=int, default=24)
    args = parser.parse_args()
    source = Image.open(args.source)
    if source.width % args.tile_width or source.height % args.tile_height:
        raise SystemExit("來源圖集尺寸不能整除圖塊尺寸")
    result = repaint(source, Image.open(args.concept), args.tile_width, args.tile_height)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    result.save(args.output, optimize=True)
    print(f"wrote {args.output} {result.width}x{result.height}")


if __name__ == "__main__":
    main()
