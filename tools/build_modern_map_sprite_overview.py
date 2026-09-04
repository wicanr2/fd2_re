#!/usr/bin/env python3
"""建立可公開的原版／現代地圖人物壓平總攬。"""

import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
ORIGINAL = ROOT / "remake/assets/sprites"
MODERN = ROOT / "remake/generated-assets/modern-theme-prototypes"
OUTPUT = ROOT / "docs/figures/modern-map-sprites-overview.png"
CATALOG = ROOT / "remake/assets/themes/modern/catalog.json"
LABELS = dict((
    (0, "Sol base selector"),
    (1, "Hano base selector"),
    (2, "Tino base selector"),
    (3, "Harvat base selector"),
    (4, "Ares base selector"),
    (5, "Lorna base selector"),
    (6, "Raiden base selector"),
    (7, "Lancelot map projection"),
    (8, "Celia base selector"),
    (9, "Yuni base selector"),
    (10, "Marlin base selector"),
    (11, "Sophia base selector"),
    (12, "Kelly base selector"),
    (13, "Beckway base selector"),
    (14, "Shan base selector"),
    (15, "character-table projection; canonical name unresolved"),
    (16, "Kylas base selector"),
    (17, "Miasdord base selector"),
    (18, "Mitty base selector"),
    (19, "Rodman base selector"),
    (20, "Sarah base selector"),
    (21, "Jonah base selector"),
    (22, "identity unresolved; selector-only projection"),
    (40, "Celia promoted selector"),
    (41, "Yuni default promoted selector"),
    (42, "Marlin default promoted selector"),
    (43, "Sophia default promoted selector"),
    (44, "Kelly promoted selector; original art duplicates selector 12"),
    (45, "Beckway default promoted selector"),
    (46, "Shan default promoted selector"),
    (47, "identity unresolved; promoted selector 15 projection"),
    (49, "identity unresolved; promoted selector 17 exact art reuse"),
    (50, "Sol optional promotion; item 89"),
    (51, "Hano optional promotion; item 93"),
    (68, "late selector sample"),
    (76, "chapter 3 helmeted soldier"),
    (77, "chapter 3 armored captain"),
    (78, "chapters 8, 9, 18 broad-shouldered enemy swordsman"),
    (80, "anonymous enemy heavy-armored unit"),
    (82, "chapters 7-9 enemy swordsman"),
    (83, "chapters 6-9, 18-19 horned enemy swordsman"),
    (85, "chapters 17-22 bucket-helmet enemy swordsman"),
    (86, "chapters 24-26, 32 helmeted fortress guard"),
    (88, "chapters 10, 12-16, 22 enemy swordsman"),
    (90, "chapters 4-9 hooded enemy swordsman"),
    (91, "anonymous hooded enemy unit"),
))


def catalog_groups() -> list[tuple[int, str]]:
    data = json.loads(CATALOG.read_text(encoding="utf-8"))
    groups = sorted(
        asset["source_group"]
        for asset in data["assets"]
        if asset.get("role") == "map_sprite_set"
    )
    if len(groups) != len(set(groups)):
        raise SystemExit("duplicate map sprite source_group in modern catalog")
    return [(group, LABELS.get(group, "selector-only projection")) for group in groups]


def load_strip(group: int, modern: bool) -> Image.Image:
    strip = Image.new("RGBA", (12 * 72, 72), (0, 0, 0, 0))
    for frame in range(12):
        if modern:
            name = f"fdicon-{group:03d}-style-a-f{frame:02d}.png"
            path = MODERN / name
        else:
            path = ORIGINAL / f"fig_{group:03d}_f{frame:02d}.png"
        if not path.is_file():
            raise SystemExit(f"missing sprite frame: {path}")
        image = Image.open(path).convert("RGBA")
        if image.size != (24, 24):
            raise SystemExit(f"unexpected frame geometry: {path}: {image.size}")
        strip.alpha_composite(image.resize((72, 72), Image.Resampling.NEAREST), (frame * 72, 0))
    return strip


def main() -> None:
    groups = catalog_groups()
    width, height = 1040, 72 + len(groups) * 168 + 32
    canvas = Image.new("RGB", (width, height), "#08152b")
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default()
    draw.text((24, 18), "FD2 MAP SPRITES / ORIGINAL AND MODERN RUNTIME CANDIDATES", fill="#ffe5a0", font=font)
    draw.text((24, 38), "12 frames per selector: 4 directions x 3 walk cycles", fill="#aab8d0", font=font)
    y = 72
    for group, label in groups:
        draw.text((24, y + 6), f"{group:03d} original", fill="#d9e2f2", font=font)
        draw.text((24, y + 28), label, fill="#7f93b4", font=font)
        original = load_strip(group, False)
        canvas.paste(original, (152, y), original)
        y += 80
        draw.text((24, y + 6), f"{group:03d} modern", fill="#d9e2f2", font=font)
        modern = load_strip(group, True)
        canvas.paste(modern, (152, y), modern)
        y += 88
    draw.text((24, height - 24), "Flattened overview only; private masters and individual runtime PNG files are not embedded.", fill="#7f93b4", font=font)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(OUTPUT, optimize=False)
    print(OUTPUT)


if __name__ == "__main__":
    main()
