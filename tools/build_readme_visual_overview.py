#!/usr/bin/env python3
"""Build the README visual overview from versioned runtime screenshots.

The result deliberately does not copy individual assets from the player-supplied
original data pack.  It only composes already reviewed screenshots in docs/figures.
"""

from pathlib import Path
import argparse
import hashlib
import json

from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
FIGURES = ROOT / "docs" / "figures"
OUTPUT = FIGURES / "original-remake-portrait-battle-overview.png"


def font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    # Pillow bundles this font, so the build does not depend on a host font.
    return ImageFont.load_default(size=size)


def centered(draw: ImageDraw.ImageDraw, box: tuple[int, int, int, int], text: str,
             face: ImageFont.FreeTypeFont, fill: str) -> None:
    left, top, right, bottom = box
    bounds = draw.textbbox((0, 0), text, font=face)
    width = bounds[2] - bounds[0]
    height = bounds[3] - bounds[1]
    draw.text(
        (left + (right - left - width) // 2, top + (bottom - top - height) // 2),
        text,
        font=face,
        fill=fill,
    )


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--hd1-feedback', type=Path)
    args = parser.parse_args()
    if args.hd1_feedback:
        runtime_gallery(args.hd1_feedback)
        return
    original_dialogue = Image.open(
        FIGURES / "ch01-dialogue-original-dosbox.png"
    ).convert("RGB").resize((640, 400), Image.Resampling.NEAREST)
    remake_dialogue = Image.open(
        FIGURES / "dialogue-remake-runtime.png"
    ).convert("RGB")
    battle = Image.open(
        FIGURES / "battle-field-ch01-scoped-compare-20260810.png"
    ).convert("RGB")

    canvas = Image.new("RGB", (1360, 832), "#11151d")
    draw = ImageDraw.Draw(canvas)
    title_font = font(25)
    label_font = font(20)
    note_font = font(16)

    centered(draw, (0, 12, 1360, 50), "PORTRAIT SYSTEM / DIALOGUE CONTEXT",
             title_font, "#f3e2ad")
    canvas.paste(original_dialogue, (24, 86))
    canvas.paste(remake_dialogue, (696, 86))
    centered(draw, (24, 52, 664, 84), "ORIGINAL DOSBox CAPTURE", label_font, "#e8edf5")
    centered(draw, (696, 52, 1336, 84), "REMAKE RUNTIME", label_font, "#e8edf5")
    centered(
        draw,
        (0, 492, 1360, 520),
        "Layout examples only: these dialogue captures are not the same scene.",
        note_font,
        "#aeb8c7",
    )

    centered(draw, (0, 528, 1360, 566), "BATTLEFIELD ICONS / SAME-STATE SAMPLE",
             title_font, "#f3e2ad")
    canvas.paste(battle, ((1360 - battle.width) // 2, 572))
    centered(
        draw,
        (0, 796, 1360, 828),
        "40,127 / 104 MB / SHA-256",
        label_font,
        "#f3e2ad",
    )
    canvas.save(OUTPUT, optimize=True)


def runtime_gallery(feedback: Path) -> None:
    """Compose only the explicitly approved runtime captures in the recipe."""
    assert Path('/.dockerenv').exists()
    recipe = json.loads(feedback.read_text())['readme_gallery']
    font_path = Path(recipe['font'])
    assert hashlib.sha256(font_path.read_bytes()).hexdigest() == recipe['font_sha256']
    face = lambda n: ImageFont.truetype(str(font_path), n, index=3)
    canvas = Image.new('RGB', (1920, 1080), '#08121e')
    draw = ImageDraw.Draw(canvas)
    draw.text((36, 20), recipe['title'], font=face(44), fill='#f3e2ad')
    assert len(recipe['captures']) == 6
    for i, row in enumerate(recipe['captures']):
        source = Path(row['path'])
        assert hashlib.sha256(source.read_bytes()).hexdigest() == row['sha256'], source
        picture = Image.open(source).convert('RGB')
        picture.thumbnail((608, 380), Image.Resampling.LANCZOS)
        left, top = 32 + (i % 3) * 624, 104 + (i // 3) * 442
        canvas.paste(picture, (left + (608 - picture.width) // 2, top))
        draw.text((left + 8, top + 387), row['label'], font=face(28), fill='#e8edf5')
    draw.text((36, 1005), recipe['note'], font=face(25), fill='#b8cad9')
    canvas.save(ROOT / recipe['output'], optimize=True)
    title = recipe['title_capture']
    source = Path(title['path'])
    assert hashlib.sha256(source.read_bytes()).hexdigest() == title['sha256']
    Image.open(source).convert('RGB').save(ROOT / title['output'], optimize=True)


if __name__ == "__main__":
    main()
