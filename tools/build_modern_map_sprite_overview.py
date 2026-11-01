#!/usr/bin/env python3
"""建立可公開的原版／現代地圖人物壓平總攬。"""

import argparse
import hashlib
import json
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
ORIGINAL = ROOT / "remake/generated-assets/fd2-original-b97caf22/sprites/fdicon"
ORIGINAL_PALETTE = ROOT / "remake/generated-assets/fd2-original-b97caf22/palette/fdother_000.json"
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
    (48, "three-phase static blue-haired human with red cloak; prologue father king via map32 and DATO048; no fabricated walk"),
    (49, "identity unresolved; promoted selector 17 exact art reuse"),
    (50, "Sol optional promotion; item 89"),
    (51, "Hano optional promotion; item 93"),
    (52, "Yuni special promotion; item 90; f08 repeats f06"),
    (53, "Hawate optional promotion; item 93"),
    (54, "Ares optional promotion; item 205"),
    (55, "native identity5 optional promotion; item205; own DATO005"),
    (56, "native identity6 optional promotion; item205; own DATO006"),
    (58, "identity unresolved; exact visual reuse of selector 40"),
    (59, "identity unresolved; exact visual reuse of selector 41"),
    (60, "optional promotion; item 88; exact visual reuse of selector 42"),
    (61, "class 22 target; exact visual reuse of selector 43"),
    (62, "optional promotion; item 92; exact visual reuse of selector 44"),
    (63, "native identity13 optional promotion; item88; own DATO013"),
    (64, "optional promotion; item 88; exact visual reuse of selector 46"),
    (65, "optional promotion; item 91; exact visual reuse of selector 47"),
    (66, "three-phase static figure; identity unresolved"),
    (67, "three-phase static figure; identity unresolved"),
    (68, "late selector sample"),
    (69, "identity unresolved; cross-shield heavy armor"),
    (70, "map5 ally, metal eyeguard and exposed lower human face; diagonal sword"),
    (71, "identity unresolved; horned blue-gray heavy armor"),
    (72, "identity unresolved; blue-haired backpack traveler"),
    (73, "map24 brown dragon-headed BIPED with sword and cross shield; not a centaur"),
    (74, "three-phase static purple-hat spellcaster; identity unresolved"),
    (75, "three-phase static red-haired low-profile figure; identity unresolved"),
    (76, "chapter 3 helmeted soldier"),
    (77, "map2 T-visor armored soldier with sword; no round shield"),
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
    (89, "map17 red-covered metal faceplate, exposed neck; no side speaker grilles"),
    (125, "map24 mature red-robed figure; DATO125 un-bearded angular face"),
    (133, "map1 brown-haired girl with blunt straight fringe; no ponytail"),
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


def load_original_palette(path: Path) -> list[int]:
    data = json.loads(path.read_text(encoding="utf-8"))
    source = data.get("source", {})
    components = data.get("dac_6bit_components")
    if data.get("asset_id") != "palette/fdother_000":
        raise SystemExit(f"unexpected original palette asset: {path}")
    if source.get("file") != "FDOTHER.DAT" or source.get("resource") != 0:
        raise SystemExit(f"unexpected original palette source: {path}")
    if not isinstance(components, list) or len(components) != 768:
        raise SystemExit(f"unexpected original palette length: {path}")
    if any(not isinstance(value, int) or value < 0 or value > 63 for value in components):
        raise SystemExit(f"invalid VGA DAC component: {path}")
    return [(value << 2) | (value >> 4) for value in components]


def load_strip(
    group: int,
    modern: bool,
    frame_count: int,
    original_root: Path,
    original_palette: list[int],
    modern_files: list[str] | None = None,
) -> Image.Image:
    strip = Image.new("RGBA", (12 * 72, 72), (0, 0, 0, 0))
    for frame in range(12):
        if frame >= frame_count:
            continue
        if modern:
            if modern_files is None or len(modern_files) != frame_count:
                raise SystemExit("modern frames require the current catalog file list")
            name = modern_files[frame]
            path = MODERN / name
        else:
            frame_root = original_root / f"sprite_{group * 12 + frame:04d}"
            path = frame_root / "frame.png"
        if not path.is_file():
            raise SystemExit(f"missing sprite frame: {path}")
        if not modern:
            indexed = Image.open(path)
            if indexed.mode != "P":
                raise SystemExit(f"original sprite is not indexed: {path}: {indexed.mode}")
            indexed.putpalette(original_palette)
            image = indexed.convert("RGBA")
            mask_path = frame_root / "mask.png"
            if not mask_path.is_file():
                raise SystemExit(f"missing sprite mask: {mask_path}")
            image.putalpha(Image.open(mask_path).convert("L"))
        else:
            image = Image.open(path).convert("RGBA")
        if image.size != (24, 24):
            raise SystemExit(f"unexpected frame geometry: {path}: {image.size}")
        strip.alpha_composite(image.resize((72, 72), Image.Resampling.NEAREST), (frame * 72, 0))
    return strip


def chibi_summary(original_root, original_palette, group_range):
    """依目前清冊畫原版與144／72／48尺寸，不以舊style-a檔名猜來源。"""
    digest = lambda path: hashlib.sha256(path.read_bytes()).hexdigest()
    catalog = json.loads(CATALOG.read_text(encoding="utf-8"))
    parent_rows = [a for a in catalog["assets"] if a.get("role") == "map_sprite_set"]
    hd_rows = [a for a in catalog["assets"] if a.get("role") == "map_sprite_hd_set"]
    parents = {a["source_group"]: a for a in parent_rows}
    hd = {a["source_group"]: a for a in hd_rows}
    if len(parents) != len(parent_rows) or len(hd) != len(hd_rows) or set(parents) != set(hd):
        raise SystemExit("duplicate or mismatched current map-sprite groups")
    groups = sorted(g for g in hd if group_range is None or group_range[0] <= g <= group_range[1])
    if not groups:
        raise SystemExit("empty source group range")
    rows = []
    images = []
    for group in groups:
        p, a = parents[group], hd[group]
        count = a.get("frame_count")
        if count not in (3, 12) or p.get("frame_count") != count or len(a.get("files", [])) != count or len(a.get("frame_sha256", [])) != count:
            raise SystemExit(f"invalid actual source shape for group{group}")
        name = a["files"][0]
        if Path(name).name != name:
            raise SystemExit("unsafe catalog frame name")
        path = MODERN / name
        if not path.is_file() or digest(path) != a["frame_sha256"][0]:
            raise SystemExit(f"current HD source missing or changed: {path}")
        with Image.open(path) as image:
            modern = image.convert("RGBA")
        if modern.size != (144, 144):
            raise SystemExit(f"HD geometry differs: {path}")
        native_root = original_root / f"sprite_{group * 12:04d}"
        with Image.open(native_root / "frame.png") as image:
            if image.mode != "P" or image.size != (24, 24):
                raise SystemExit(f"invalid original frame: {native_root}")
            image.putpalette(original_palette)
            native = image.convert("RGBA")
        with Image.open(native_root / "mask.png") as image:
            native.putalpha(image.convert("L"))
        if not native.getchannel("A").getbbox():
            raise SystemExit(f"empty original source: {native_root}")
        rows.append(dict(group=group, frame=0, frame_count=count, hd_file=name, hd_sha256=digest(path), native_inputs={n:digest(native_root/n) for n in ("frame.png", "mask.png", "remap_mask.png")}))
        images.append((native, modern))
    columns, tile_width, tile_height = 4, 416, 202
    height = 116 + ((len(groups) + columns - 1) // columns) * tile_height + 40
    canvas = Image.new("RGB", (48 + columns * tile_width, height), "#08152b")
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default(size=14)
    title = ImageFont.load_default(size=20)
    draw.text((24, 18), "FD2 / CURRENT ORIGINAL + HD1 MAP SOURCES", fill="#ffe5a0", font=title)
    draw.text((24, 48), f"GROUPS {groups[0]:03d}-{groups[-1]:03d} / {len(groups)} sources / original72 + HD144 / 72 / 48 pixels", fill="#d9e2f2", font=font)
    draw.text((24, 73), "Current catalog files, neutral frame0 only. Source comparison, not runtime or complete human acceptance.", fill="#aab8d0", font=font)
    for index, (row, pair) in enumerate(zip(rows, images)):
        x = 24 + (index % columns) * tile_width
        y = 116 + (index // columns) * tile_height
        group = row["group"]
        label = f"GROUP {group:03d} / {row['frame_count']} phases"
        if group == 67:
            label += " / anonymous*"
        draw.text((x + 12, y + 4), label, fill="#ffe5a0" if group == 67 else "#d9e2f2", font=font)
        native, modern = pair
        native72 = native.resize((72, 72), Image.Resampling.NEAREST)
        canvas.paste(native72, (x + 12, y + 42), native72)
        canvas.paste(modern, (x + 112, y + 28), modern)
        for size, dx, dy in ((72, 274, 40), (48, 354, 52)):
            small = modern.resize((size, size), Image.Resampling.LANCZOS)
            canvas.paste(small, (x + dx, y + dy), small)
            draw.text((x + dx, y + 124), str(size), fill="#aab8d0", font=font)
        draw.text((x + 12, y + 124), "ORIG72", fill="#aab8d0", font=font)
        draw.text((x + 112, y + 180), "HD144", fill="#aab8d0", font=font)
    draw.text((24, height - 28), "*67 uses its own native face under explicit source exception; identity/portrait/appearance unknown. Representative native palette.", fill="#aab8d0", font=font)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(OUTPUT, optimize=False)
    receipt = dict(schema="fd2-hd1-current-source-overview-v1", catalog_sha256=digest(CATALOG), tool_sha256=digest(Path(__file__)), image_sha256=digest(OUTPUT), groups=groups, group_count=len(groups), source_frames=sum(r["frame_count"] for r in rows), shown_frame=0, original_size=72, hd_sizes=[144,72,48], width=canvas.width, height=canvas.height, rows=rows, scope="current source comparison only; not identity binding, runtime capture or complete human acceptance")
    OUTPUT.with_suffix(".json").write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({k:v for k,v in receipt.items() if k != "rows"}, ensure_ascii=False))


def main() -> None:
    global CATALOG, MODERN, OUTPUT
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, default=CATALOG)
    parser.add_argument("--modern-root", type=Path, default=MODERN)
    parser.add_argument("--output", type=Path, default=OUTPUT)
    parser.add_argument("--chibi-summary", action="store_true")
    parser.add_argument("--group-range", nargs=2, type=int)
    parser.add_argument(
        "--original-root",
        type=Path,
        default=ORIGINAL,
        help="原版逐幀參考根目錄；可指向不進 Git 的暫存輸出",
    )
    parser.add_argument(
        "--original-palette",
        type=Path,
        default=ORIGINAL_PALETTE,
        help="雜湊固定 FDOTHER #0 的六位元 VGA DAC JSON",
    )
    args = parser.parse_args()
    CATALOG, MODERN, OUTPUT = args.catalog, args.modern_root, args.output
    original_palette = load_original_palette(args.original_palette)
    if args.chibi_summary:
        chibi_summary(args.original_root, original_palette, args.group_range)
        return
    if args.group_range is not None:
        parser.error("--group-range requires --chibi-summary")
    groups = catalog_groups()
    current_files = {a["source_group"]: a["files"] for a in json.loads(CATALOG.read_text(encoding="utf-8"))["assets"] if a.get("role") == "map_sprite_set"}
    width, height = 1040, 72 + len(groups) * 168 + 32
    canvas = Image.new("RGB", (width, height), "#08152b")
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.load_default()
    draw.text((24, 18), "FD2 MAP SPRITES / ORIGINAL AND MODERN RUNTIME CANDIDATES", fill="#ffe5a0", font=font)
    draw.text((24, 38), "original sprites with FDOTHER #0 representative palette / modern runtime candidates", fill="#aab8d0", font=font)
    y = 72
    for group, label, frame_count in groups:
        draw.text((24, y + 6), f"{group:03d} original", fill="#d9e2f2", font=font)
        draw.text((24, y + 28), label, fill="#7f93b4", font=font)
        original = load_strip(group, False, frame_count, args.original_root, original_palette)
        canvas.paste(original, (152, y), original)
        y += 80
        draw.text((24, y + 6), f"{group:03d} modern", fill="#d9e2f2", font=font)
        modern = load_strip(group, True, frame_count, args.original_root, original_palette, current_files[group])
        canvas.paste(modern, (152, y), modern)
        y += 88
    draw.text((24, height - 24), "Flattened overview only; private masters and individual runtime PNG files are not embedded.", fill="#7f93b4", font=font)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(OUTPUT, optimize=False)
    print(OUTPUT)


if __name__ == "__main__":
    main()
