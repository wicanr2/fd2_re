#!/usr/bin/env python3
"""產生 README 用、不可逆重組原版素材的低解析總覽圖。

輸入必須是玩家自備原版所解出的本機素材；輸出刻意降低解析度、只取每組一幀，
不能取代私人素材包，也不應被遊戲執行期當成資產來源。
"""

from __future__ import annotations

from pathlib import Path
import json
import re

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "remake" / "assets"
OUT = ROOT / "docs" / "figures"
BG = "#11151d"
FG = "#f3e2ad"
MUTED = "#aeb8c7"


def font(size: int) -> ImageFont.ImageFont:
    return ImageFont.load_default(size=size)


def title(canvas: Image.Image, text: str, note: str) -> int:
    draw = ImageDraw.Draw(canvas)
    draw.text((20, 14), text, font=font(24), fill=FG)
    draw.text((20, 46), note, font=font(14), fill=MUTED)
    return 76


def numeric(path: Path) -> int:
    match = re.search(r"(\d+)", path.stem if path.is_file() else path.name)
    if match is None:
        raise ValueError(f"檔名缺少編號：{path}")
    return int(match.group(1))


def render_map(path: Path) -> Image.Image:
    data = json.loads((path.parent / "map.json").read_text(encoding="utf-8"))
    tileset = Image.open(path).convert("RGB")
    tw, th, cols = data["tileW"], data["tileH"], data["cols"]
    image = Image.new("RGB", (data["w"] * tw, data["h"] * th))
    for index, tile in enumerate(data["tiles"]):
        sx, sy = (tile % cols) * tw, (tile // cols) * th
        crop = tileset.crop((sx, sy, sx + tw, sy + th))
        image.paste(crop, ((index % data["w"]) * tw, (index // data["w"]) * th))
    return image


def maps() -> None:
    paths = sorted((ASSETS / "maps").glob("map*/tileset.png"), key=lambda p: numeric(p.parent))
    if len(paths) != 33:
        raise RuntimeError(f"預期 33 張戰場圖，實際 {len(paths)} 張")
    cols, thumb_w, thumb_h, gap = 6, 96, 72, 12
    rows = (len(paths) + cols - 1) // cols
    canvas = Image.new("RGB", (cols * (thumb_w + gap) + 28, 76 + rows * 98 + 16), BG)
    y0 = title(canvas, "ORIGINAL BATTLEFIELDS / 33 MAPS", "LOW-RES RESEARCH INDEX / FULL ASSETS REMAIN PRIVATE")
    draw = ImageDraw.Draw(canvas)
    for i, path in enumerate(paths):
        x = 16 + (i % cols) * (thumb_w + gap)
        y = y0 + (i // cols) * 98
        image = render_map(path).resize((thumb_w, thumb_h), Image.Resampling.BOX)
        canvas.paste(image, (x, y))
        draw.text((x, y + thumb_h + 3), f"MAP {numeric(path.parent):02d}", font=font(13), fill=FG)
    canvas.save(OUT / "original-battlefields-atlas-public.png", optimize=True)


def grouped(kind: str, pattern: str, expected: int, output: str, heading: str,
            source_size: int, sample_size: int) -> None:
    paths = sorted((ASSETS / kind).glob(pattern), key=numeric)
    if len(paths) != expected:
        raise RuntimeError(f"{kind} 預期 {expected} 組，實際 {len(paths)} 組")
    cols, cell, gap = 12, 48, 7
    rows = (len(paths) + cols - 1) // cols
    canvas = Image.new("RGB", (cols * (cell + gap) + 28, 76 + rows * 68 + 16), BG)
    y0 = title(canvas, heading, "ONE DOWNSAMPLED FRAME PER GROUP / NOT A RUNTIME ASSET SHEET")
    draw = ImageDraw.Draw(canvas)
    for i, path in enumerate(paths):
        x = 16 + (i % cols) * (cell + gap)
        y = y0 + (i // cols) * 68
        image = Image.open(path).convert("RGB")
        image = image.resize((sample_size, sample_size), Image.Resampling.BOX)
        image = image.resize((source_size, source_size), Image.Resampling.NEAREST)
        canvas.paste(image, (x + (cell - source_size) // 2, y))
        draw.text((x + 8, y + source_size + 3), f"{numeric(path):03d}", font=font(12), fill=FG)
    canvas.save(OUT / output, optimize=True)


def main() -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    maps()
    grouped("sprites", "fig_*_f00.png", 96, "original-sprites-atlas-public.png",
            "ORIGINAL BATTLE SPRITES / 96 GROUPS", 36, 12)
    grouped("portraits", "DATO_*_m0.png", 96, "original-portraits-atlas-public.png",
            "ORIGINAL DIALOGUE PORTRAITS / 96 GROUPS", 40, 20)


if __name__ == "__main__":
    main()
