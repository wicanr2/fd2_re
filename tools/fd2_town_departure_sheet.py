#!/usr/bin/env python3
"""#39公開壓平總覽：原版indexed＋寫後DAC、正式GPU、完整差異。

Docker/Pillow入口與限制見docs/data/ida/fd2_town_departure_20261005.json。
"""
import argparse
import json
from pathlib import Path

from PIL import Image, ImageChops

parser = argparse.ArgumentParser()
parser.add_argument("--original", type=Path, required=True)
parser.add_argument("--gpu", type=Path, required=True)
parser.add_argument("--out", type=Path, required=True)
args = parser.parse_args()
frames = [json.loads(x) for x in (args.original / "frames/frames.jsonl").read_text().splitlines()]
trace = [json.loads(x) for x in (args.original / "eip-trace.jsonl").read_text().splitlines()]
town = [r for r in trace if r["eip"] == "0x2D25A"]
fade = [r for r in trace if r["eip"] == "0x1F544" and r["stack"][4] == "0x2066E"]
assert len(frames) == 11 and len(town) == 10 and len(fade) == 65
sheet = Image.new("RGB", (960, 600))
for row, (index, record, suffix) in enumerate([(0, town[0], "town-01"), (9, town[9], "town-10"), (10, fade[-1], "map-final")]):
    original = Image.open(args.original / "frames" / frames[index]["file"])
    assert original.mode == "P" and original.size == (320, 200)
    dac = bytes.fromhex(record["map_runtime"]["palette_dac6_hex"])
    assert len(dac) == 768 and max(dac) <= 63
    # 與oracle及正式fdother.ParseVGAPalette相同的6→8位元擴展。
    original.putpalette([(component << 2) | (component >> 4) for component in dac])
    original = original.convert("RGB")
    gpu = Image.open(str(args.gpu) + "-" + suffix + ".png")
    assert gpu.size == (640, 400)
    gpu = gpu.convert("RGB").resize((320, 200), Image.Resampling.NEAREST)
    difference = ImageChops.difference(original, gpu)
    assert difference.getbbox() is None
    for col, image in enumerate([original, gpu, difference]):
        sheet.paste(image, (320 * col, 200 * row))
sheet.save(args.out)
print(f"{args.out}: 三個完整畫布均0差異；原版欄是indexed＋同時點DAC重建。")
