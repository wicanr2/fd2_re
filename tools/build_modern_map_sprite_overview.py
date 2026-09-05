#!/usr/bin/env python3
"""建立可公開的原版／現代地圖人物壓平總攬。"""

import argparse
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
    (48, "three-phase static royal figure; no fabricated walk frames"),
    (49, "identity unresolved; promoted selector 17 exact art reuse"),
    (50, "Sol optional promotion; item 89"),
    (51, "Hano optional promotion; item 93"),
    (52, "Yuni special promotion; item 90; f08 repeats f06"),
    (53, "Hawate optional promotion; item 93"),
    (54, "Ares optional promotion; item 205"),
    (55, "identity unresolved; optional promotion; item 205"),
    (56, "identity unresolved; optional promotion; item 205"),
    (58, "identity unresolved; exact visual reuse of selector 40"),
    (59, "identity unresolved; exact visual reuse of selector 41"),
    (60, "optional promotion; item 88; exact visual reuse of selector 42"),
    (61, "class 22 target; exact visual reuse of selector 43"),
    (62, "optional promotion; item 92; exact visual reuse of selector 44"),
    (63, "optional promotion; item 88; identity unresolved"),
    (64, "optional promotion; item 88; exact visual reuse of selector 46"),
    (65, "optional promotion; item 91; exact visual reuse of selector 47"),
    (66, "three-phase static figure; identity unresolved"),
    (67, "three-phase static figure; identity unresolved"),
    (68, "late selector sample"),
    (69, "identity unresolved; cross-shield heavy armor"),
    (70, "identity unresolved; masked blue-cap swordsman"),
    (71, "identity unresolved; horned blue-gray heavy armor"),
    (72, "identity unresolved; blue-haired backpack traveler"),
    (73, "identity unresolved; horse-form spear and shield unit"),
    (74, "three-phase static purple-hat spellcaster; identity unresolved"),
    (75, "three-phase static red-haired low-profile figure; identity unresolved"),
    (76, "chapter 3 helmeted soldier"),
    (77, "chapter 3 armored captain"),
    (78, "chapters 8, 9, 18 broad-shouldered enemy swordsman"),
    (79, "identity unresolved; olive-hood masked swordsman"),
    (80, "anonymous enemy heavy-armored unit"),
    (81, "identity unresolved; horned deep-blue heavy armor"),
    (82, "chapters 7-9 enemy swordsman"),
    (83, "chapters 6-9, 18-19 horned enemy swordsman"),
    (84, "identity unresolved; vivid-red shield heavy armor"),
    (85, "chapters 17-22 bucket-helmet enemy swordsman"),
    (86, "chapters 24-26, 32 helmeted fortress guard"),
    (88, "chapters 10, 12-16, 22 enemy swordsman"),
    (90, "chapters 4-9 hooded enemy swordsman"),
    (91, "anonymous hooded enemy unit"),
))


def catalog_groups() -> list[tuple[int, str, int]]:
    data = json.loads(CATALOG.read_text(encoding="utf-8"))
    assets = sorted(
        (asset["source_group"], asset["frame_count"])
        for asset in data["assets"]
        if asset.get("role") == "map_sprite_set"
    )
    groups = [group for group, _ in assets]
    if len(groups) != len(set(groups)):
        raise SystemExit("duplicate map sprite source_group in modern catalog")
    return [(group, LABELS.get(group, "selector-only projection"), frame_count)
            for group, frame_count in assets]


def load_strip(group: int, modern: bool, frame_count: int, original_root: Path) -> Image.Image:
    strip = Image.new("RGBA", (12 * 72, 72), (0, 0, 0, 0))
    for frame in range(12):
        if frame >= frame_count:
            continue
        if modern:
            name = f"fdicon-{group:03d}-style-a-f{frame:02d}.png"
            path = MODERN / name
        else:
            path = original_root / f"fig_{group:03d}_f{frame:02d}.png"
        if not path.is_file():
            raise SystemExit(f"missing sprite frame: {path}")
        image = Image.open(path).convert("RGBA")
        if image.size != (24, 24):
            raise SystemExit(f"unexpected frame geometry: {path}: {image.size}")
        strip.alpha_composite(image.resize((72, 72), Image.Resampling.NEAREST), (frame * 72, 0))
    return strip


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--original-root",
        type=Path,
        default=ORIGINAL,
        help="原版逐幀參考根目錄；可指向不進 Git 的暫存輸出",
    )
    args = parser.parse_args()
    groups = catalog_groups()
    width, height = 1040, 72 + len(groups) * 168 + 32
    canvas = Image.new("RGB", (width, height), "#08152b")
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default()
    draw.text((24, 18), "FD2 MAP SPRITES / ORIGINAL AND MODERN RUNTIME CANDIDATES", fill="#ffe5a0", font=font)
    draw.text((24, 38), "12-frame walkers and evidence-backed 3-phase static selectors", fill="#aab8d0", font=font)
    y = 72
    for group, label, frame_count in groups:
        draw.text((24, y + 6), f"{group:03d} original", fill="#d9e2f2", font=font)
        draw.text((24, y + 28), label, fill="#7f93b4", font=font)
        original = load_strip(group, False, frame_count, args.original_root)
        canvas.paste(original, (152, y), original)
        y += 80
        draw.text((24, y + 6), f"{group:03d} modern", fill="#d9e2f2", font=font)
        modern = load_strip(group, True, frame_count, args.original_root)
        canvas.paste(modern, (152, y), modern)
        y += 88
    draw.text((24, height - 24), "Flattened overview only; private masters and individual runtime PNG files are not embedded.", fill="#7f93b4", font=font)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(OUTPUT, optimize=False)
    print(OUTPUT)


if __name__ == "__main__":
    main()
