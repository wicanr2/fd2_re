#!/usr/bin/env python3
"""由現代主題 catalog 產生可公開的角色頭像總攬圖。"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", type=Path, required=True)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--font", type=Path, help="以可攜中文字型呈現正式姓名")
    parser.add_argument("--party-only", action="store_true", help="只列32名伙伴，優先使用480像素頭像")
    args = parser.parse_args()

    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    portraits = [
        asset
        for asset in catalog["assets"]
        if asset.get("role") == "story_portrait_frame" and asset.get("frame") == 0
    ]
    portraits.sort(key=lambda asset: asset["speaker_id"])
    if args.party_only:
        portraits = [a for a in portraits if 0 <= a['speaker_id'] < 32]
        if [a['speaker_id'] for a in portraits] != list(range(32)):
            raise SystemExit('伙伴頭像未齊，不產生完整群像')
        hd = {a['speaker_id']: a for a in catalog['assets'] if a['role'] == 'story_portrait_hd_frame'}
        if not all(a['speaker_id'] in hd for a in portraits):
            raise SystemExit('伙伴480像素頭像未齊')
        portraits = [hd[a['speaker_id']] for a in portraits]
    if not portraits:
        raise SystemExit("no portrait frames")

    columns, cell_w, cell_h = (8, 176, 200) if args.party_only else (7, 132, 126)
    rows = (len(portraits) + columns - 1) // columns
    canvas = Image.new("RGB", (columns * cell_w, 52 + rows * cell_h), "#07142d")
    draw = ImageDraw.Draw(canvas)
    font = ImageFont.truetype(str(args.font), 19, index=3) if args.font else ImageFont.load_default()
    title = '炎龍騎士團2｜伙伴HD頭像' if args.font else 'FD2 modern hand-painted portraits'
    draw.text((16, 6), title, fill="#f2d58b", font=font)
    count = f'已登錄{len(portraits)}位｜依各自原版面容重繪' if args.font else f'registered runtime candidates: {len(portraits)}'
    draw.text((16, 29), count, fill="#b9c9e8", font=font)
    entities = json.loads((Path(__file__).resolve().parents[1]/'remake/assets/locales/zh-Hant/entities.json').read_text())

    for index, asset in enumerate(portraits):
        x = (index % columns) * cell_w
        y = 52 + (index // columns) * cell_h
        path = args.source / asset['file']
        if hashlib.sha256(path.read_bytes()).hexdigest() != asset['sha256']:
            raise SystemExit(f'頭像SHA不符：{path.name}')
        side = 144 if args.party_only else 96
        image = Image.open(path).convert("RGB").resize((side, side), Image.Resampling.LANCZOS)
        canvas.paste(image, (x + (cell_w-side)//2, y + 4))
        slug = asset["asset_id"].split(".")[1]
        label = f'{asset["speaker_id"]:02d} {slug}'
        if args.font:
            person = entities['characters'].get(str(asset['speaker_id']))
            label = f'{asset["speaker_id"]:02d} ' + (person['name'] if person else slug)
        draw.text((x + 16, y + side + 12), label, fill="#ffffff", font=font)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    canvas.save(args.output, optimize=True)
    print(f"wrote {args.output} with {len(portraits)} portrait(s)")


if __name__ == "__main__":
    main()
