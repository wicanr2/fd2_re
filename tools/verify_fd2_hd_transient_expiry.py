#!/usr/bin/env python3
"""核對到期／獎勵提示HD的完整流程收據、實際PNG、文字及細框。"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
import itertools
from pathlib import Path

from PIL import Image, ImageChops, ImageDraw, ImageFont, ImageOps

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
    if data.get("status") == "PASS-FULL-REWARD-HD-RUNTIME-E1":
        return validate_full_reward(data, directory, images, require_chrome)
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


def validate_full_reward(data: dict, directory: Path, images: bool, require_chrome: bool) -> dict:
    require(data.get("normal_player_verified") is False, "滿背包建構槽不得冒稱正常玩家")
    require(data.get("cases") == 5 and data.get("continuations") == 5, "滿背包五分支或續行缺項")
    rows = data.get("rows", [])
    require(0 < len(rows) <= 800 and data.get("game_frames") == len(rows), "滿背包完整畫布資料不足")
    require(data.get("native_seed") == 7 and data.get("fixed_clock_unix") == 1000, "滿背包測試條件不同")
    source = chrome_profile(data, require_chrome)
    require(source is not None and data.get("legacy_chrome_rejected") is True, "滿背包新舊框線缺少正反對照")
    captures, total, png_total, f2 = [], 0, 0, 0
    for case in range(5):
        selected = [row for row in rows if row.get("case") == case]
        phases = {row.get("phase") for row in selected}
        needed = {"job-0", "job-2", "job-3", "yes-no", "return"}
        needed |= {"item-open", "item-choice", "item-close"} if case in (2, 3) else {"job-4", "response-hold"}
        if case == 3:
            needed |= {"job-5", "job-6", "response-hold"}
        require(needed <= phases, "滿背包分支缺少必要轉場")
    require(all(type(row.get("case")) is int and 0 <= row["case"] < 5 for row in rows), "滿背包分支索引錯誤")
    for row in rows:
        require(row.get("native_control_unchanged") is True, "滿背包原生狀態不同")
        require(type(row.get("f2_roundtrip")) is bool, "滿背包F2紀錄缺件")
        f2 += row["f2_roundtrip"]
        shift, height = row.get("dialogue_chrome_shift"), row.get("dialogue_chrome_height")
        require((shift, height) in {(0, 0), (0, 86), (13, 75), (26, 62), (39, 49), (52, 36), (65, 23)}, "滿背包框線位移／裁切錯誤")
        if row["phase"].startswith("item-") or row["phase"] == "return":
            require(height == 0, "選擇器或返回殘留對話框")
        if row["phase"] in {"yes-no", "response-hold"}:
            require((shift, height) == (0, 86) and row.get("text_runs", 0) > 0, "滿背包穩定訊息缺少框線或文字")
        require(row.get("dialogue_chrome_pixels") == chrome_count(height*6), "滿背包框線核對範圍錯誤")
        total += row["dialogue_chrome_pixels"]
        if not row.get("capture"):
            continue
        name = f"case{row['case']}-{row['phase']}.png"
        if data.get("item_panel_chrome") and row["phase"] in {"item-open", "item-close"}:
            steps = {0, 5, 8} if row["phase"] == "item-open" else {0, 6, 11}
            require(row["case"] == 2 and type(row.get("item_anim_step")) is int and row["item_anim_step"] in steps, "物品開收截圖位置錯誤")
            name = f"case2-{row['phase']}-step{row['item_anim_step']}.png"
        require(row["capture"] == name, "滿背包截圖路徑錯誤")
        path = directory/name
        if images:
            with Image.open(path) as picture:
                require(picture.size == (1920, 1200), "滿背包截圖不是完整畫布")
                picture = picture.convert("RGBA")
                require(hashlib.sha256(picture.tobytes()).hexdigest() == row.get("final_sha256"), "滿背包PNG與畫布不同")
                if height:
                    png_total += check_chrome_png(picture, source, shift, height*6)
            captures.append({"file": name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
    require(data.get("f2_boundaries") == f2 and f2 >= 30, "滿背包F2邊界不足")
    require(data.get("chrome_pixels") == total and total > 0, "滿背包細框核對總量錯誤")
    expected = {f"case{case}-{phase}.png" for case, phases in ((0, ("yes-no", "response-hold", "return")), (2, ("yes-no", "item-choice", "return")), (3, ("yes-no", "item-choice", "response-hold", "return"))) for phase in phases}
    if data.get("item_panel_chrome"):
        expected |= {f"case2-{phase}-step{step}.png" for phase, steps in (("item-open", (0, 5, 8)), ("item-close", (0, 6, 11))) for step in steps}
    require({row.get("capture") for row in rows if row.get("capture")} == expected, "滿背包代表截圖缺件")
    return {"status": "PASS-FULL-REWARD-HD-INDEPENDENT-CAPTURE-CHECK", "frames": len(rows), "f2_boundaries": f2,
            "captures": captures, "dialogue_chrome_pixels": total, "png_dialogue_chrome_pixels": png_total,
            "normal_player_verified": False}


def rejection_checks(data: dict, directory: Path, require_chrome: bool = False) -> int:
    if data.get("status") == "PASS-FULL-REWARD-HD-RUNTIME-E1":
        mutations = (
            lambda x: x.update(normal_player_verified=True),
            lambda x: x.update(continuations=0),
            lambda x: x.update(f2_boundaries=0),
            lambda x: x["rows"].pop(),
            lambda x: x["rows"][0].update(dialogue_chrome_pixels=0),
            lambda x: x["rows"][0].update(dialogue_chrome_shift=99),
            lambda x: x.update(legacy_chrome_rejected=False),
        )
        for mutate in mutations:
            candidate = copy.deepcopy(data)
            mutate(candidate)
            try:
                validate(candidate, directory, images=False, require_chrome=require_chrome)
            except ValueError:
                continue
            raise ValueError("滿背包反例未被拒收")
        for mutate in (lambda x: x["chrome_source"]["background"].__setitem__(0, (x["chrome_source"]["background"][0]+64) % 256),
                       lambda x: next(row for row in x["rows"] if row.get("capture")).update(final_sha256="0"*64)):
            candidate = copy.deepcopy(data)
            mutate(candidate)
            try:
                validate(candidate, directory, require_chrome=require_chrome)
            except ValueError:
                continue
            raise ValueError("滿背包PNG反例未被拒收")
        return len(mutations)+2
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


def confirmation_rgba(ink: dict) -> tuple[int, ...]:
    require(isinstance(ink, dict) and set(ink) == {"R", "G", "B", "A"}, "選項色盤資料錯誤")
    result = tuple(ink[k] for k in ("R", "G", "B", "A"))
    require(all(type(v) is int and 0 <= v <= 255 for v in result) and result[3] == 255, "選項色盤不是不透明RGBA")
    return result


def check_confirmation_picture(picture: Image.Image, buttons: list) -> tuple[int, int]:
    pixels = picture.load()
    checked, ink_points = 0, 0
    for i, button in enumerate(buttons):
        x0, y0, x1, y1 = button["rect"]
        background, ink, shadow = (confirmation_rgba(button[k]) for k in ("background", "ink", "shadow"))
        visible_ink = []
        later = [b["rect"] for b in buttons[i+1:]]
        for y in range(y0, y1):
            for x in range(x0, x1):
                if any(a <= x < c and b <= y < d for a, b, c, d in later):
                    continue
                actual = pixels[x, y]
                if x0+12 <= x < x1-12 and y0+12 <= y < y1-12:
                    if max(abs(a-b) for a, b in zip(actual[:3], background[:3])) > 3:
                        visible_ink.append((x, y))
                    continue
                dx, dy, height = x-x0, y-y0, y1-y0
                expected = ink if dx < 3 or dx >= 141 or dy < 3 or dy >= height-3 else shadow if dx == 140 or dy == height-4 else background
                require(all(abs(a-b) <= 2 for a, b in zip(actual, expected)), f"HD選項細框或舊格清除錯誤：{x},{y}")
                checked += 1
        overlap = any(max(x0, a) < min(x1, c) and max(y0, b) < min(y1, d) for a, b, c, d in later)
        if not overlap:
            require(len(visible_ink) >= 30, "HD選項文字未抵達最終PNG")
            actual_rect = (min(x for x, y in visible_ink), min(y for x, y in visible_ink), max(x for x, y in visible_ink)+1, max(y for x, y in visible_ink)+1)
            desired = button["ink_rect"]
            require(all(actual_rect[j] >= desired[j]-1 for j in (0, 1)) and all(actual_rect[j] <= desired[j]+1 for j in (2, 3)), "選項實際字形超出量測墨跡")
            require(abs(actual_rect[0]+actual_rect[2]-desired[0]-desired[2]) <= 4 and abs(actual_rect[1]+actual_rect[3]-desired[1]-desired[3]) <= 4, "選項字形置中或pulse位置錯誤")
        ink_points += len(visible_ink)
    require(checked > 0, "HD選項實際PNG核對為空")
    return checked, ink_points


def validate_confirmation(data: dict, directory: Path, images: bool = True) -> dict:
    require(data.get("hd_confirmation") is True, "沒有HD確認選項收據")
    rows = data.get("confirmation_rows", [])
    require(len(rows) == 48 and data.get("confirmation_f2") == 16, "四語選項／開收矩陣或F2不完整")
    expected = {(locale, "steady", -1, choice, pulse) for locale in LOCALES for choice in range(2) for pulse in range(2)}
    expected |= {(locale, phase, index, 0, 0) for locale in LOCALES for phase, indices in (("opening", range(6, 10)), ("closing", range(4))) for index in indices}
    seen, captures, geometry, ink_points = set(), [], 0, 0
    packs = {locale: json.loads((ROOT/"remake/assets/locales"/locale/"pack.json").read_text())["entries"] for locale in LOCALES}
    for row in rows:
        locale, phase, index, choice, pulse = (row.get(k) for k in ("locale", "phase", "index", "choice", "pulse"))
        require(locale in LOCALES and phase in {"steady", "opening", "closing"} and choice in (0, 1) and pulse in (0, 1), "選項狀態欄位錯誤")
        key = (locale, phase, index, choice, pulse) if phase == "steady" else (locale, phase, index, 0, 0)
        require(key in expected and key not in seen, "選項矩陣錯誤或重複")
        seen.add(key)
        require(row.get("full_rgba_pixels") == 1920*1200 and row.get("native_state_unchanged") is True, "完整畫布／原生狀態核對缺件")
        require(row.get("f2_roundtrip") is (phase == "steady"), "四語穩定選項F2不足")
        spread = 16 if phase == "steady" else (index-5)*4 if phase == "opening" else 12-index*4
        height = 96 if phase == "steady" else 120
        buttons = row.get("buttons", [])
        require(len(buttons) == 2, "選項左右兩格缺件")
        for option, button in enumerate(buttons):
            x = (248 + (-spread if option == 0 else spread))*6
            rect = [x, 1008, x+144, 1008+height]
            source_index = 16+option if phase != "steady" else (48, 51)[option] + (pulse if option == choice else 0)
            require(button.get("rect") == rect and button.get("source_index") == source_index, "選項原生格、位置或疊畫順序錯誤")
            require(button.get("pulse") == (pulse if phase == "steady" and option == choice else 0), "選項文字pulse錯誤")
            require(button.get("label") == packs[locale][("common.yes", "common.no")[option]]["text"], "選項文字沒有切換到正式語系")
            ink = button.get("ink_rect", [])
            require(len(ink) == 4 and all(type(v) is int for v in ink) and x+12 <= ink[0] < ink[2] <= x+132 and 1020 <= ink[1] < ink[3] <= rect[3]-12, "選項字形超過文字安全區")
            for k in ("background", "ink", "shadow"):
                confirmation_rgba(button.get(k))
        capture = phase == "steady" and choice == 0 and pulse == 0 or locale == "zh-Hant" and (phase == "opening" and index in (6, 9) or phase == "closing" and index in (0, 3))
        require(bool(row.get("capture")) == capture, "四語選項／開收代表截圖缺件")
        if not capture:
            continue
        name = f"confirmation-{locale}-{phase}{index}-c{choice}-p{pulse}.png"
        require(row["capture"] == name, "選項截圖路徑錯誤")
        path = directory/name
        if images:
            with Image.open(path) as picture:
                require(picture.size == (1920, 1200), "選項不是完整HD畫布")
                picture = picture.convert("RGBA")
                require(hashlib.sha256(picture.tobytes()).hexdigest() == row.get("final_sha256"), "選項PNG與完整畫布不同")
                count, points = check_confirmation_picture(picture, buttons)
                geometry += count
                ink_points += points
            captures.append({"file": name, "sha256": hashlib.sha256(path.read_bytes()).hexdigest()})
    require(seen == expected and (not images or len(captures) == 8), "四語選項矩陣或圖片不足")
    return {"confirmation_samples": 48, "confirmation_f2": 16, "confirmation_cpu_pixels": 48*1920*1200,
            "confirmation_png_geometry_pixels": geometry, "confirmation_png_ink_pixels": ink_points, "confirmation_captures": captures}


def confirmation_rejection_checks(data: dict, directory: Path) -> int:
    mutations = (
        lambda x: x.update(hd_confirmation=False),
        lambda x: x["confirmation_rows"].pop(),
        lambda x: x.update(confirmation_f2=0),
        lambda x: x["confirmation_rows"][0].update(native_state_unchanged=False),
        lambda x: x["confirmation_rows"][0].update(full_rgba_pixels=0),
        lambda x: x["confirmation_rows"][0]["buttons"][0]["rect"].__setitem__(0, 0),
        lambda x: x["confirmation_rows"][0]["buttons"][0].update(label="wrong"),
        lambda x: x["confirmation_rows"][0]["buttons"][0]["ink_rect"].__setitem__(2, 99999),
    )
    for mutate in mutations:
        candidate = copy.deepcopy(data)
        mutate(candidate)
        try:
            validate_confirmation(candidate, directory, images=False)
        except ValueError:
            continue
        raise ValueError("HD選項反例未被拒收")
    for field in ("final_sha256", "background"):
        candidate = copy.deepcopy(data)
        row = next(r for r in candidate["confirmation_rows"] if r.get("capture"))
        if field == "final_sha256":
            row[field] = "0"*64
        else:
            row["buttons"][1][field]["R"] ^= 127
        try:
            validate_confirmation(candidate, directory)
        except ValueError:
            continue
        raise ValueError("HD選項實際PNG反例未被拒收")
    return len(mutations)+2


def check_full_reward_portraits(data: dict, directory: Path, pack: Path) -> dict:
    # The five existing full-reward fixtures use DATO0, at native (8,115).
    require(data.get("status") == "PASS-FULL-REWARD-HD-RUNTIME-E1", "頭像抽驗只適用滿背包五分支收據")
    catalog = json.loads((ROOT/"remake/assets/themes/modern/catalog.json").read_text())
    assets = [a for a in catalog["assets"] if a.get("role") == "story_portrait_hd_frame" and a.get("speaker_id") == 0]
    require(len(assets) == 1, "索爾HD頭像清冊缺件或重複")
    asset = assets[0]
    require(Path(asset["file"]).name == asset["file"], "頭像路徑不安全")
    source = pack/asset["file"]
    require(hashlib.sha256(source.read_bytes()).hexdigest() == asset["sha256"], "頭像圖源SHA不同")
    with Image.open(source) as picture:
        wanted = ImageOps.mirror(picture.convert("RGBA"))
    require(wanted.size == (480, 480), "頭像不是已登錄480圖源")
    points, captures = 0, 0
    for row in data["rows"]:
        if not row.get("capture") or not row["dialogue_chrome_height"]:
            continue
        top = (115+row["dialogue_chrome_shift"])*6
        height = min(480, 1200-top)
        with Image.open(directory/row["capture"]) as picture:
            actual = picture.convert("RGBA").crop((48, top, 528, top+height))
        difference = ImageChops.difference(actual, wanted.crop((0, 0, 480, height)))
        require(max(high for low, high in difference.getextrema()) <= 2, "最終畫面未使用同次HD頭像或鏡射錯誤")
        points += 480*height
        captures += 1
    require(captures >= 5 and points > 0, "頭像實際PNG核對不足")
    return {"portrait_source_points": points, "portrait_captures": captures,
            "portrait_source": {"file": asset["file"], "sha256": asset["sha256"]}}


def overview(directory: Path, output: Path, reward: bool = False, full_reward: bool = False, confirmation: bool = False) -> None:
    result = Image.new("RGB", (1920, 1264), "#121a2b")
    font = ImageFont.load_default(size=22)
    draw = ImageDraw.Draw(result)
    samples = [(locale, f"expiry-{locale}-counter0-frame05.png") for locale in LOCALES]
    if reward:
        samples = [(label, f"reward-kind{kind}-phase{phase:02d}.png") for label, kind, phase in (("ITEM", 0, 6), ("GOLD", 1, 6), ("CLOSING", 1, 9), ("RESTORE", 1, 12))]
    if full_reward:
        samples = [("FULL INVENTORY", "case0-yes-no.png"), ("DECLINE", "case0-response-hold.png"),
                   ("ITEM SELECTOR", "case2-item-choice.png"), ("CANCEL SELECTOR", "case3-response-hold.png")]
    if confirmation:
        samples = [(locale, f"confirmation-{locale}-steady-1-c0-p0.png") for locale in LOCALES]
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
    parser.add_argument("--portrait-pack", type=Path, help="以正式私人包獨立核對滿背包最終頭像")
    parser.add_argument("--require-chrome", action="store_true", help="新細框驗收，舊收據維持預設相容")
    parser.add_argument("--require-confirmation", action="store_true", help="要求四語HD確認選項，舊收據預設相容")
    args = parser.parse_args()
    data = json.loads(args.receipt.read_text())
    report = validate(data, args.receipt.parent, require_chrome=args.require_chrome)
    report["receipt_sha256"] = hashlib.sha256(args.receipt.read_bytes()).hexdigest()
    if args.require_confirmation or data.get("hd_confirmation"):
        report.update(validate_confirmation(data, args.receipt.parent))
        if args.self_test:
            report["confirmation_rejection_checks"] = confirmation_rejection_checks(data, args.receipt.parent)
    if args.portrait_pack:
        report.update(check_full_reward_portraits(data, args.receipt.parent, args.portrait_pack))
    if args.self_test:
        report["rejection_checks"] = rejection_checks(data, args.receipt.parent, args.require_chrome)
    if args.overview:
        overview(args.receipt.parent, args.overview, data.get("status") == "PASS-REWARD-HD-RUNTIME-E1",
                 data.get("status") == "PASS-FULL-REWARD-HD-RUNTIME-E1", data.get("hd_confirmation") is True)
        report["overview_sha256"] = hashlib.sha256(args.overview.read_bytes()).hexdigest()
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    pictures = len(report["captures"]) + len(report.get("confirmation_captures", []))
    print(f"{report['status']}: {pictures} PNG，{report['frames']}主流程畫格")


if __name__ == "__main__":
    main()
