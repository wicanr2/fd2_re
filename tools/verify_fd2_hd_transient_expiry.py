#!/usr/bin/env python3
"""核對狀態到期HD的完整流程收據、實際PNG及四語文字。"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
LOCALES = ("zh-Hant", "zh-Hans", "en", "ja")
KEYS = ("attack", "defence", "speed", "poison", "paralysis", "silence")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def validate(data: dict, directory: Path, images: bool = True) -> dict:
    require(data.get("status") == "PASS-EXPIRY-HD-RUNTIME-E1", "收據狀態錯誤")
    require(data.get("normal_player_verified") is False, "建構槽不得冒稱正常玩家")
    require(data.get("locales") == list(LOCALES), "四語清單不完整")
    require(data.get("counters") == 6 and data.get("game_frames") == 288, "六提示完整鏈不足")
    require(data.get("f2_roundtrips") == 36 and data.get("rejected_sources") == 3, "切換或拒收不足")
    rows = data.get("rows", [])
    require(len(rows) == 288, "逐拍資料不完整")
    packs = {
        locale: json.loads((ROOT / "remake/assets/locales" / locale / "pack.json").read_text())["entries"]
        for locale in LOCALES
    }
    captures, outside, portraits, ink = [], 0, 0, 0
    for i, row in enumerate(rows):
        locale, counter, frame = LOCALES[i // 72], (i % 72) // 12, i % 12
        require((row.get("locale"), row.get("counter"), row.get("frame")) == (locale, counter, frame), "提示順序錯誤")
        expected_f2 = frame == 5 or counter == 0 and frame in (0, 9, 11)
        require(row.get("f2_roundtrip") is expected_f2, "F2抽樣階段不完整或誤列")
        shift = (5 - frame) * 13 if frame < 6 else (frame - 5) * 13
        height = 0 if frame == 11 else min(198 + shift, 200) - (112 + shift)
        expected_outside = 1920 * 1200 - 310 * height * 36
        require(row.get("outside_hd_pixels") == expected_outside, "移動框外HD核對範圍錯誤")
        require(row.get("portrait_source_points", 0) >= (0 if frame == 11 else 100), "HD頭像未核對")
        outside += expected_outside
        portraits += row.get("portrait_source_points", 0)
        if frame == 5:
            require(row.get("actual_text_pixels", 0) >= 100, "文字未到達最終畫布")
            require(row.get("text_runs", 0) > 0, "沒有HD文字")
            ink += row["actual_text_pixels"]
            if locale != "zh-Hant":
                expected = packs[locale]["battle.status.expiry." + KEYS[counter]]["text"]
                require("".join(row.get("visible_text", "").split()) == "".join(expected.split()), "實際文字未切換語系")
        expected_capture = frame == 5 or counter == 0 and frame in (0, 9, 11)
        require(bool(row.get("capture")) == expected_capture, "代表性截圖缺件或額外混入")
        if not expected_capture:
            continue
        name = row["capture"]
        require(name == f"expiry-{locale}-counter{counter}-frame{frame:02d}.png", "截圖路徑錯誤")
        path = directory / name
        if images:
            with Image.open(path) as source:
                require(source.size == (1920, 1200), "截圖不是完整HD畫布")
                digest = hashlib.sha256(source.convert("RGBA").tobytes()).hexdigest()
                require(digest == row.get("final_rgba_sha256"), "PNG與實際最終畫布不同")
            captures.append({"file": name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
    return {"status": "PASS-EXPIRY-HD-INDEPENDENT-CAPTURE-CHECK", "frames": len(rows),
            "outside_hd_pixels": outside, "portrait_source_points": portraits,
            "actual_text_pixels": ink, "captures": captures, "normal_player_verified": False}


def rejection_checks(data: dict, directory: Path) -> int:
    mutations = (
        lambda x: x.update(normal_player_verified=True),
        lambda x: x.update(rejected_sources=0),
        lambda x: x["rows"].pop(),
        lambda x: x["rows"][0].update(f2_roundtrip=False),
        lambda x: x["rows"][0].update(outside_hd_pixels=0),
        lambda x: x["rows"][0].update(portrait_source_points=0),
        lambda x: x["rows"][5].update(actual_text_pixels=0),
        lambda x: x["rows"][149].update(visible_text="增加攻擊力的效果消失了！"),
        lambda x: x["rows"][5].pop("capture"),
    )
    for mutate in mutations:
        candidate = copy.deepcopy(data)
        mutate(candidate)
        try:
            validate(candidate, directory, images=False)
        except ValueError:
            continue
        raise ValueError("反例未被拒收")
    candidate = copy.deepcopy(data)
    candidate["rows"][0]["final_rgba_sha256"] = "0" * 64
    try:
        validate(candidate, directory)
    except ValueError:
        return len(mutations) + 1
    raise ValueError("錯誤PNG雜湊未被拒收")


def overview(directory: Path, output: Path) -> None:
    result = Image.new("RGB", (1920, 1264), "#121a2b")
    font = ImageFont.load_default(size=22)
    draw = ImageDraw.Draw(result)
    for i, locale in enumerate(LOCALES):
        x, y = (i % 2) * 960, (i // 2) * 632
        draw.text((x + 12, y + 2), locale, font=font, fill="white")
        with Image.open(directory / f"expiry-{locale}-counter0-frame05.png") as source:
            result.paste(source.convert("RGB").resize((960, 600), Image.Resampling.LANCZOS), (x, y + 32))
    result.save(output)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("receipt", type=Path)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--overview", type=Path)
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    data = json.loads(args.receipt.read_text())
    report = validate(data, args.receipt.parent)
    report["receipt_sha256"] = hashlib.sha256(args.receipt.read_bytes()).hexdigest()
    if args.self_test:
        report["rejection_checks"] = rejection_checks(data, args.receipt.parent)
    if args.overview:
        overview(args.receipt.parent, args.overview)
        report["overview_sha256"] = hashlib.sha256(args.overview.read_bytes()).hexdigest()
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(f"{report['status']}: {len(report['captures'])} PNG，{report['frames']}拍")


if __name__ == "__main__":
    main()
