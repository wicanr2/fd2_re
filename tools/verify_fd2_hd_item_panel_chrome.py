#!/usr/bin/env python3
"""獨立核對三區物品／狀態面板的完整PNG、原生裁切與舊邊帶清除。"""
import argparse
import copy
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops, ImageFont, ImageDraw
from verify_fd2_hd_transient_expiry import chrome_profile, require

FULL = ((5, 7, 91, 93), (92, 7, 315, 93), (5, 94, 315, 196))


def region(sx, sy, dx, dy, w, h, enabled=True):
    return dict(Enabled=enabled, SourceX=sx, SourceY=sy, DestX=dx, DestY=dy, Width=w, Height=h)


def native_regions(frame, bottom_step=None):
    require(type(frame) is int and 0 <= frame <= 11, "原生面板位置錯誤")
    left = 5 if frame < 6 else 101-16*frame
    result = [region(5-min(left, 0), 7, max(left, 0), 7, 86+min(left, 0), 86)]
    if frame <= 8:
        upper = 7 if frame < 3 else 55-16*frame
        result.append(region(92, 7-min(upper, 0), 92, max(upper, 0), 223, 86+min(upper, 0)))
    else:
        result.append(region(0, 0, 0, 0, 0, 0, False))
    if frame < 6 or bottom_step is not None:
        step = frame if bottom_step is None else bottom_step
        require(type(step) is int and 0 <= step <= 6, "下方面板位置錯誤")
        y = 94+16*step
        result.append(region(5, 94, 5, y, 310, min(102, 200-y)))
    else:
        result.append(region(0, 0, 0, 0, 0, 0, False))
    return result


def panel_rectangles(regions):
    for r in regions:
        if not r["Enabled"] or r["Width"] <= 0 or r["Height"] <= 0:
            continue
        bounds = next((b for b in FULL if b[0] <= r["SourceX"] and b[1] <= r["SourceY"] and
                       r["SourceX"]+r["Width"] <= b[2] and r["SourceY"]+r["Height"] <= b[3]), None)
        require(bounds is not None, "面板來源範圍錯誤")
        ox, oy = r["DestX"]-r["SourceX"]+bounds[0], r["DestY"]-r["SourceY"]+bounds[1]
        art_h = 99 if bounds[1] == 94 else bounds[3]-bounds[1]
        visible = (max(0, r["DestX"]*6), max(0, r["DestY"]*6),
                   min(1920, (r["DestX"]+r["Width"])*6), min(1200, (r["DestY"]+r["Height"])*6, (oy+art_h)*6))
        yield ox*6, oy*6, (bounds[2]-bounds[0])*6, art_h*6, visible


def border_pixels(regions):
    for ox, oy, w, h, (x0, y0, x1, y1) in panel_rectangles(regions):
        for y in range(y0, y1):
            sy = y-oy
            intervals = [(x0, x1)] if sy < 18 or sy >= h-18 else [(x0, min(x1, ox+18)), (max(x0, ox+w-18), x1)]
            for start, end in intervals:
                for x in range(start, end):
                    yield x, y, x-ox, sy, w, h


def border_count(regions):
    count = 0
    for ox, oy, w, h, (x0, y0, x1, y1) in panel_rectangles(regions):
        inner_w = max(0, min(x1, ox+w-18)-max(x0, ox+18))
        inner_h = max(0, min(y1, oy+h-18)-max(y0, oy+18))
        count += max(0, x1-x0)*max(0, y1-y0)-inner_w*inner_h
    return count


def color_at(source, x, y, w, h):
    if (x < 3 or x >= w-3) and (y < 3 or y >= h-3):
        corner = int(x >= w-3)+2*int(y >= h-3)
        return source["corners"][corner][(y if y < 3 else y-h+3)*3+(x if x < 3 else x-w+3)]
    if y < 3: return source["edges"][0][y]
    if x >= w-3: return source["edges"][1][x-w+3]
    if y >= h-3: return source["edges"][2][y-h+3]
    if x < 3: return source["edges"][3][x]
    return source["background"]


def check_picture(picture, source, regions):
    require(picture.size == (1920, 1200), "面板PNG不是完整HD畫布")
    pixels = picture.convert("RGBA").load()
    count = 0
    for x, y, sx, sy, w, h in border_pixels(regions):
        require(all(abs(a-b) <= 1 for a, b in zip(pixels[x, y], color_at(source, sx, sy, w, h))), f"面板細框或舊邊帶不同：{x},{y}")
        count += 1
    require(count == border_count(regions), "框線核對範圍不足")
    return count


def validate(receipt, images=True, data=None):
    if data is None: data = json.loads(receipt.read_text())
    require(data.get("normal_player_verified") is False and data.get("native_seed") == 7 and data.get("fixed_clock_unix") == 1000, "受控測試條件不同")
    full = data.get("status") == "PASS-FULL-REWARD-HD-RUNTIME-E1"
    require(full or data.get("status") == "PASS-ITEM-PANEL-STATUS-HD-RUNTIME-E1", "錯誤的面板收據")
    source = chrome_profile(dict(data, dialogue_chrome=True), True)
    rows = data.get("rows", [])
    require(len(rows) == (291 if full else 41), "完整流程畫格缺項")
    if full:
        require(data.get("item_panel_chrome") is True and data.get("continuations") == 5, "滿背包細框或續行缺項")
    else:
        expected = ["opening"]*12+["status"]+["transition"]*14+["commands"]+["closing"]*12+["return"]
        require([r.get("phase") for r in rows] == expected and data.get("f2_roundtrips") == 9, "狀態兩頁流程或F2缺項")
        required_pictures = {f"status-{name}.png" for name in ["opening", "opening-last", "status", "transition", "transition-last", "commands", "closing", "closing-last", "return"]}
        require({r.get("capture") for r in rows if r.get("capture")} == required_pictures, "狀態代表畫面缺項")
    require(all(type(r.get("f2_roundtrip")) is bool for r in rows) and sum(r["f2_roundtrip"] for r in rows) == (40 if full else 9), "F2逐筆紀錄不同")
    total, png_points, captures = 0, 0, []
    for row in rows:
        require(row.get("native_control_unchanged") is True, "原生狀態不同")
        phase = row["phase"]
        if full:
            if phase.startswith("item-"):
                step = row.get("item_anim_step")
                require(type(step) is int and 0 <= step <= 11 and type(row.get("item_closing")) is bool, "物品動畫條件缺件")
                require(row["item_closing"] == (phase == "item-close"), "物品開收旗標不同")
                regions = native_regions(step if row["item_closing"] else 11-step)
            else: regions = []
            require(row.get("item_panel_chrome_pixels") == border_count(regions), "物品細框畫格數量不同")
        else:
            frame = row.get("frame")
            if phase == "return": regions = []
            elif phase == "opening": regions = native_regions(11-frame)
            elif phase == "closing": regions = native_regions(frame)
            elif phase == "transition":
                require(type(frame) is int and 0 <= frame <= 13, "狀態頁轉場位置錯誤")
                regions = native_regions(0, frame if frame < 7 else 13-frame)
            else:
                require(frame == 0, "穩定狀態頁位置錯誤")
                regions = native_regions(0)
            require(row.get("regions") == (regions or None), "狀態來源／目的範圍不同")
            require(row.get("chrome_pixels") == border_count(regions), "狀態細框畫格數量不同")
        total += border_count(regions)
        if not images or not row.get("capture"): continue
        name = row["capture"]
        require(Path(name).name == name and name.endswith(".png"), "截圖路徑錯誤")
        path = receipt.parent/name
        with Image.open(path) as picture:
            picture = picture.convert("RGBA")
            require(hashlib.sha256(picture.tobytes()).hexdigest() == row.get("final_sha256"), "PNG與完整畫布雜湊不同")
            if regions: png_points += check_picture(picture, source, regions)
        captures.append(dict(file=name, sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
    require(total > 0 and data.get("item_panel_chrome_pixels" if full else "chrome_pixels") == total, "細框總量不同")
    return dict(status="PASS-ITEM-PANEL-CHROME-INDEPENDENT", frames=len(rows), chrome_pixels=total, png_chrome_pixels=png_points,
                captures=captures, receipt_sha256=hashlib.sha256(receipt.read_bytes()).hexdigest(), normal_player_verified=False)


def rejection_checks(receipt):
    original = json.loads(receipt.read_text())
    validate(receipt, False, original)
    full = original["status"] == "PASS-FULL-REWARD-HD-RUNTIME-E1"
    mutators = [lambda d: d.update(normal_player_verified=True), lambda d: d.update(native_seed=8),
                lambda d: d["rows"][0].update(native_control_unchanged=False), lambda d: d["rows"].pop(),
                lambda d: d["rows"][0].update(f2_roundtrip="true"), lambda d: d["chrome_source"].pop("corners"),
                lambda d: d["chrome_source"]["edges"][0][0].__setitem__(3, 0)]
    if full:
        mutators += [lambda d: d.update(item_panel_chrome=False), lambda d: d["rows"][0].update(item_panel_chrome_pixels=1)]
    else:
        mutators += [lambda d: d.update(f2_roundtrips=0), lambda d: d["rows"][0].update(frame=99)]
    for mutate in mutators:
        changed = copy.deepcopy(original);mutate(changed)
        try: validate(receipt, False, changed)
        except ValueError: continue
        raise ValueError("變造收據未被拒收")
    return len(mutators)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--full-reward", type=Path)
    parser.add_argument("--status", type=Path)
    parser.add_argument("--previous", type=Path, help="舊HD畫面的原生粗框正反對照")
    parser.add_argument("--reject-picture", type=Path, action="append", default=[], help="以真實錯框PNG驗證拒收")
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--overview", type=Path)
    parser.add_argument("--font", type=Path)
    args = parser.parse_args()
    require(args.full_reward or args.status, "缺少收據")
    require(not args.output.exists(), "不覆寫核對結果")
    result = dict(status="PASS-ITEM-PANEL-CHROME-INDEPENDENT", normal_player_verified=False)
    for name, path in (("full_reward", args.full_reward), ("player_status", args.status)):
        if path:
            result[name] = validate(path)
            result[name]["rejection_checks"] = rejection_checks(path)
    if args.previous:
        require(args.full_reward is not None, "舊畫面對照需要滿背包收據")
        source = json.loads(args.full_reward.read_text())["chrome_source"]
        with Image.open(args.previous) as picture:
            try: check_picture(picture, source, native_regions(0))
            except ValueError as error: reason = str(error)
            else: raise ValueError("舊原生粗框未被拒收")
        result["old_native_rejected"] = dict(file=str(args.previous), sha256=hashlib.sha256(args.previous.read_bytes()).hexdigest(), reason=reason)
        with Image.open(args.previous) as a, Image.open(args.full_reward.parent/"case2-item-choice.png") as b:
            a, b = a.convert("RGBA"), b.convert("RGBA")
            # Exclude only the explicitly replaced three-pixel native borders.
            diff = ImageChops.difference(a, b)
            mask = Image.new("L", a.size, 255)
            for x, y, *_ in border_pixels(native_regions(0)): mask.putpixel((x, y), 0)
            require(all(ImageChops.multiply(channel, mask).getbbox() is None for channel in diff.split()), "穩定面板框外與上一版不同")
        result["stable_non_border_pixels_unchanged"] = 1920*1200-border_count(native_regions(0))
    if args.reject_picture:
        source = json.loads((args.full_reward or args.status).read_text())["chrome_source"]
        rejected = []
        for path in args.reject_picture:
            with Image.open(path) as picture:
                try: check_picture(picture, source, native_regions(0))
                except ValueError as error: reason = str(error)
                else: raise ValueError("錯框PNG未被拒收")
            rejected.append(dict(file=str(path), sha256=hashlib.sha256(path.read_bytes()).hexdigest(), reason=reason))
        result["rejected_pictures"] = rejected
    if args.overview:
        require(args.full_reward and args.status and args.font, "總覽需要兩種完整流程與字型")
        entries = [(args.full_reward.parent/"case2-item-choice.png", "物品選擇器"),
                   (args.full_reward.parent/"case2-item-open-step5.png", "物品面板滑入裁切"),
                   (args.status.parent/"status-status.png", "角色狀態頁"),
                   (args.status.parent/"status-commands.png", "角色指令頁")]
        board = Image.new("RGB", (1920, 1264), (8, 15, 29));draw = ImageDraw.Draw(board)
        font = ImageFont.truetype(str(args.font), 24)
        for i, (path, label) in enumerate(entries):
            x, y = i%2*960, i//2*632
            draw.text((x+16, y+3), label, font=font, fill=(240, 244, 255))
            with Image.open(path) as pic: board.paste(pic.convert("RGB").resize((960, 600), Image.Resampling.LANCZOS), (x, y+32))
        board.save(args.overview)
        result["overview_sha256"] = hashlib.sha256(args.overview.read_bytes()).hexdigest()
    args.output.write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({k: v for k, v in result.items() if k not in {"full_reward", "player_status"}}, ensure_ascii=False))


if __name__ == "__main__":
    main()
