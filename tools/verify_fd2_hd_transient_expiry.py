#!/usr/bin/env python3
"""核對到期／獎勵提示HD的完整流程收據、實際PNG、文字及細框。"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
import itertools
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
LOCALES = ("zh-Hant", "zh-Hans", "en", "ja")
KEYS = ("attack", "defence", "speed", "poison", "paralysis", "silence")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def chrome_profile(data: dict, required: bool) -> dict | None:
    if not data.get("dialogue_chrome"):
        require(not required, "沒有HD細框驗證")
        return None
    source = data.get("chrome_source")
    require(isinstance(source, dict), "細框來源缺件")
    colors = [source.get("background")]
    for field, count in (("edges", 3), ("corners", 9)):
        groups = source.get(field)
        require(isinstance(groups, list) and len(groups) == 4, "四邊／四角來源缺件")
        for group in groups:
            require(isinstance(group, list) and len(group) == count, "細框來源尺寸錯誤")
            colors.extend(group)
    for ink in colors:
        require(isinstance(ink, list) and len(ink) == 4 and all(type(v) is int and 0 <= v <= 255 for v in ink) and ink[3] == 255, "細框色盤錯誤")
    return source


def chrome_pixel(source: dict, x: int, y: int) -> tuple[int, ...]:
    if (x < 3 or x >= 1857) and (y < 3 or y >= 513):
        corner = (1 if x >= 1857 else 0) + (2 if y >= 513 else 0)
        return tuple(source["corners"][corner][(y if y < 3 else y - 513)*3 + (x if x < 3 else x - 1857)])
    if y < 3:
        return tuple(source["edges"][0][y])
    if x >= 1857:
        return tuple(source["edges"][1][x - 1857])
    if y >= 513:
        return tuple(source["edges"][2][y - 513])
    if x < 3:
        return tuple(source["edges"][3][x])
    return tuple(source["background"])


def chrome_count(height: int) -> int:
    return 1860*height - 1824*max(0, min(height, 498)-18)


def check_chrome_png(picture: Image.Image, source: dict, shift: int, height: int) -> int:
    pixels = picture.load()
    count = 0
    for y in range(height):
        xs = range(1860) if y < 18 or y >= 498 else itertools.chain(range(18), range(1842, 1860))
        for x in xs:
            actual = pixels[30+x, (112+shift)*6+y]
            expected = chrome_pixel(source, x, y)
            require(all(abs(a-b) <= 1 for a, b in zip(actual, expected)), f"HD細框或舊邊帶清除錯誤：{x},{y}")
            count += 1
    require(count == chrome_count(height), "細框核對範圍不足")
    return count


def validate(data: dict, directory: Path, images: bool = True, require_chrome: bool = False) -> dict:
    if data.get("status") == "PASS-REWARD-HD-RUNTIME-E1":
        return validate_reward(data, directory, images, require_chrome)
    require(data.get("status") == "PASS-EXPIRY-HD-RUNTIME-E1", "收據狀態錯誤")
    require(data.get("normal_player_verified") is False, "建構槽不得冒稱正常玩家")
    require(data.get("locales") == list(LOCALES), "四語清單不完整")
    require(data.get("counters") == 6 and data.get("game_frames") == 288, "六提示完整鏈不足")
    require(data.get("f2_roundtrips") == 36 and data.get("rejected_sources") == 3, "切換或拒收不足")
    rows = data.get("rows", [])
    require(len(rows) == 288, "逐拍資料不完整")
    source = chrome_profile(data, require_chrome)
    if source:
        require(data.get("legacy_chrome_rejected") is True, "舊粗框未被拒收")
    packs = {
        locale: json.loads((ROOT / "remake/assets/locales" / locale / "pack.json").read_text())["entries"]
        for locale in LOCALES
    }
    captures, outside, portraits, ink, chrome, png_chrome = [], 0, 0, 0, 0, 0
    for i, row in enumerate(rows):
        locale, counter, frame = LOCALES[i // 72], (i % 72) // 12, i % 12
        require((row.get("locale"), row.get("counter"), row.get("frame")) == (locale, counter, frame), "提示順序錯誤")
        expected_f2 = frame == 5 or counter == 0 and frame in (0, 9, 11)
        require(row.get("f2_roundtrip") is expected_f2, "F2抽樣階段不完整或誤列")
        shift = (5 - frame) * 13 if frame < 6 else (frame - 5) * 13
        height = 0 if frame == 11 else min(198 + shift, 200) - (112 + shift)
        if source:
            require(row.get("dialogue_chrome_pixels") == chrome_count(height*6), "細框逐拍核對範圍錯誤")
            chrome += row["dialogue_chrome_pixels"]
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
            with Image.open(path) as picture:
                require(picture.size == (1920, 1200), "截圖不是完整HD畫布")
                picture = picture.convert("RGBA")
                digest = hashlib.sha256(picture.tobytes()).hexdigest()
                require(digest == row.get("final_rgba_sha256"), "PNG與實際最終畫布不同")
                if source and height:
                    png_chrome += check_chrome_png(picture, source, shift, height*6)
            captures.append({"file": name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
    if source:
        require(data.get("chrome_pixels") == chrome, "細框總核對範圍錯誤")
    return {"status": "PASS-EXPIRY-HD-INDEPENDENT-CAPTURE-CHECK", "frames": len(rows),
            "outside_hd_pixels": outside, "portrait_source_points": portraits,
            "actual_text_pixels": ink, "captures": captures, "dialogue_chrome_pixels": chrome,
            "png_dialogue_chrome_pixels": png_chrome, "normal_player_verified": False}


def validate_reward(data: dict, directory: Path, images: bool, require_chrome: bool) -> dict:
    require(data.get("normal_player_verified") is False, "建構獎勵槽不得冒稱正常玩家")
    require(data.get("cases") == 2 and data.get("game_frames") == 26 and data.get("f2_round_trips") == 26, "兩條獎勵鏈或F2不足")
    rows = data.get("rows", [])
    require(len(rows) == 26, "獎勵逐拍資料不完整")
    source = chrome_profile(data, require_chrome)
    captures, outside, portraits, chrome, png_chrome = [], 0, 0, 0, 0
    for i, row in enumerate(rows):
        kind, phase = i // 13, i % 13
        require((row.get("kind"), row.get("phase_index")) == (kind, phase), "獎勵提示順序錯誤")
        require(row.get("f2_roundtrip") is True, "獎勵F2不足")
        shift = (5-phase)*13 if phase < 6 else (phase-6)*13
        height = 0 if phase == 12 else min(198+shift, 200)-(112+shift)
        expected_outside = 1920*1200 - 310*height*36
        require(row.get("outside_hd_pixels") == expected_outside, "獎勵框外核對範圍錯誤")
        require(row.get("portrait_source_points", 0) >= (0 if phase == 12 else 100), "獎勵頭像未核對")
        outside += expected_outside
        portraits += row.get("portrait_source_points", 0)
        if source:
            require(row.get("dialogue_chrome_pixels") == chrome_count(height*6), "獎勵細框核對範圍錯誤")
            chrome += row["dialogue_chrome_pixels"]
        expected_capture = phase in (0, 5, 6, 9, 12)
        require(bool(row.get("capture")) == expected_capture, "獎勵截圖缺件")
        if not expected_capture:
            continue
        name = row["capture"]
        require(name == f"reward-kind{kind}-phase{phase:02d}.png", "獎勵截圖路徑錯誤")
        path = directory / name
        if images:
            with Image.open(path) as picture:
                require(picture.size == (1920, 1200), "獎勵截圖不是完整HD畫布")
                picture = picture.convert("RGBA")
                require(hashlib.sha256(picture.tobytes()).hexdigest() == row.get("final_sha256"), "獎勵PNG與最終畫布不同")
                if source and height:
                    png_chrome += check_chrome_png(picture, source, shift, height*6)
            captures.append({"file": name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
    if source:
        require(data.get("chrome_pixels") == chrome, "獎勵細框總核對範圍錯誤")
    return {"status": "PASS-REWARD-HD-INDEPENDENT-CAPTURE-CHECK", "frames": len(rows),
            "outside_hd_pixels": outside, "portrait_source_points": portraits, "captures": captures,
            "dialogue_chrome_pixels": chrome, "png_dialogue_chrome_pixels": png_chrome,
            "normal_player_verified": False}


def rejection_checks(data: dict, directory: Path, require_chrome: bool = False) -> int:
    if data.get("status") == "PASS-REWARD-HD-RUNTIME-E1":
        mutations = (
            lambda x: x.update(normal_player_verified=True),
            lambda x: x.update(f2_round_trips=0),
            lambda x: x["rows"].pop(),
            lambda x: x["rows"][0].update(phase_index=99),
            lambda x: x["rows"][0].update(f2_roundtrip=False),
            lambda x: x["rows"][0].pop("capture"),
        )
        if data.get("dialogue_chrome"):
            mutations += (lambda x: x["rows"][0].update(dialogue_chrome_pixels=0),)
        for mutate in mutations:
            candidate = copy.deepcopy(data)
            mutate(candidate)
            try:
                validate(candidate, directory, images=False, require_chrome=require_chrome)
            except ValueError:
                continue
            raise ValueError("獎勵反例未被拒收")
        candidate = copy.deepcopy(data)
        candidate["rows"][0]["final_sha256"] = "0"*64
        try:
            validate(candidate, directory, require_chrome=require_chrome)
        except ValueError:
            return len(mutations)+1
        raise ValueError("錯獎勵PNG未被拒收")
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
            validate(candidate, directory, images=False, require_chrome=require_chrome)
        except ValueError:
            continue
        raise ValueError("反例未被拒收")
    candidate = copy.deepcopy(data)
    candidate["rows"][0]["final_rgba_sha256"] = "0" * 64
    try:
        validate(candidate, directory, require_chrome=require_chrome)
    except ValueError:
        pass
    else:
        raise ValueError("錯誤PNG雜湊未被拒收")
    count = len(mutations) + 1
    if require_chrome:
        for mutate in (lambda x: x.update(dialogue_chrome=False), lambda x: x["rows"][0].update(dialogue_chrome_pixels=0), lambda x: x.update(legacy_chrome_rejected=False)):
            candidate = copy.deepcopy(data)
            mutate(candidate)
            try:
                validate(candidate, directory, images=False, require_chrome=True)
            except ValueError:
                count += 1
                continue
            raise ValueError("缺HD細框收據未被拒收")
        candidate = copy.deepcopy(data)
        candidate["chrome_source"]["background"][0] ^= 127
        try:
            validate(candidate, directory, require_chrome=True)
        except ValueError:
            count += 1
        else:
            raise ValueError("錯細框顏色未被實際PNG拒收")
    return count


def overview(directory: Path, output: Path, reward: bool = False) -> None:
    result = Image.new("RGB", (1920, 1264), "#121a2b")
    font = ImageFont.load_default(size=22)
    draw = ImageDraw.Draw(result)
    samples = [(locale, f"expiry-{locale}-counter0-frame05.png") for locale in LOCALES]
    if reward:
        samples = [(label, f"reward-kind{kind}-phase{phase:02d}.png") for label, kind, phase in (("ITEM", 0, 6), ("GOLD", 1, 6), ("CLOSING", 1, 9), ("RESTORE", 1, 12))]
    for i, (label, name) in enumerate(samples):
        x, y = (i % 2) * 960, (i // 2) * 632
        draw.text((x + 12, y + 2), label, font=font, fill="white")
        with Image.open(directory / name) as source:
            result.paste(source.convert("RGB").resize((960, 600), Image.Resampling.LANCZOS), (x, y + 32))
    result.save(output)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("receipt", type=Path)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--overview", type=Path)
    parser.add_argument("--self-test", action="store_true")
    parser.add_argument("--require-chrome", action="store_true", help="新細框驗收，舊收據維持預設相容")
    args = parser.parse_args()
    data = json.loads(args.receipt.read_text())
    report = validate(data, args.receipt.parent, require_chrome=args.require_chrome)
    report["receipt_sha256"] = hashlib.sha256(args.receipt.read_bytes()).hexdigest()
    if args.self_test:
        report["rejection_checks"] = rejection_checks(data, args.receipt.parent, args.require_chrome)
    if args.overview:
        overview(args.receipt.parent, args.overview, data.get("status") == "PASS-REWARD-HD-RUNTIME-E1")
        report["overview_sha256"] = hashlib.sha256(args.overview.read_bytes()).hexdigest()
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(f"{report['status']}: {len(report['captures'])} PNG，{report['frames']}拍")


if __name__ == "__main__":
    main()
