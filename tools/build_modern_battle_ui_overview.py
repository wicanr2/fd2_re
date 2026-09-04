#!/usr/bin/env python3
"""建立可公開展示、不可直接當 runtime pack 的現代戰場 UI 壓平總覽。"""

from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--asset-root", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    canvas = Image.new("RGB", (900, 360), (10, 18, 32))
    panel = Image.open(args.asset_root / "battle-hud-panel-style-a-149x42.png").convert("RGB")
    panel = panel.resize((596, 168), Image.Resampling.NEAREST)
    canvas.paste(panel, ((canvas.width - panel.width) // 2, 24))

    names = ("attack", "spell", "item", "wait")
    icon_size = (112, 104)
    gap = 44
    total = len(names) * icon_size[0] + (len(names) - 1) * gap
    left = (canvas.width - total) // 2
    for index, name in enumerate(names):
        icon = Image.open(
            args.asset_root / f"battle-action-{name}-style-a-28x26.png"
        ).convert("RGB").resize(icon_size, Image.Resampling.NEAREST)
        canvas.paste(icon, (left + index * (icon_size[0] + gap), 222))

    args.output.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(args.output, optimize=True)
    print(f"wrote {args.output} {canvas.width}x{canvas.height}")


if __name__ == "__main__":
    main()
