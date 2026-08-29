#!/usr/bin/env python3
"""Build the README visual overview from versioned runtime screenshots.

The result deliberately does not copy individual assets from the player-supplied
original data pack.  It only composes already reviewed screenshots in docs/figures.
"""

from pathlib import Path

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
    original_dialogue = Image.open(
        FIGURES / "ch01-dialogue-original-dosbox.png"
    ).convert("RGB").resize((640, 400), Image.Resampling.NEAREST)
    remake_dialogue = Image.open(
        FIGURES / "dialogue-remake-runtime.png"
    ).convert("RGB")
    battle = Image.open(
        FIGURES / "battle-field-ch01-scoped-compare-20260810.png"
    ).convert("RGB")

    canvas = Image.new("RGB", (1360, 790), "#11151d")
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
    canvas.save(OUTPUT, optimize=True)


if __name__ == "__main__":
    main()
