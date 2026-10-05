#!/usr/bin/env python3
"""獨立核對 #177 的完整原版／正式 GPU 收據，不搜尋候選或遮罩像素。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--first", type=Path, required=True)
    parser.add_argument("--second", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--figure", type=Path)
    parser.add_argument("--go-log", type=Path)
    args = parser.parse_args()
    metadata = [json.loads(x) for x in (args.original / "frames/frames.jsonl").read_text().splitlines()]
    originals = {row["step"]: row for row in metadata}
    first = json.loads((args.first / "validation.json").read_text())
    second = json.loads((args.second / "validation.json").read_text())
    assert first == second, "兩輪規則、輸入及畫面收據不同"
    assert first["ticks"] == 1561 and first["random_reads"] == 59 and first["cancel_restored_town"]
    assert len(first["frames"]) == len(originals) == 88
    results, representatives = [], {}
    for row in first["frames"]:
        source = args.original / "frames" / originals[row["step"]]["file"]
        one, two = args.first / row["file"], args.second / row["file"]
        original = Image.open(source)
        assert original.mode == "P" and original.size == (320, 200)
        assert hashlib.sha256(original.tobytes()).hexdigest() == row["indexed_sha256"]
        assert row["indexed_sha256"] == originals[row["step"]]["indexed_sha256"]
        a, b = Image.open(one).convert("RGBA"), Image.open(two).convert("RGBA")
        assert a.size == b.size == (640, 400)
        expected = original.convert("RGBA").resize((640, 400), Image.Resampling.NEAREST)
        assert a.tobytes() == expected.tobytes(), "完整RGBA存在差異"
        assert a.tobytes() == b.tobytes() and sha(one) == sha(two)
        assert row["mouth_frame"] in (0, 3) and row["gpu_rgba_diff_pixels"] == 0
        results.append({"step": row["step"], "mouth_frame": row["mouth_frame"], "original": str(source), "original_sha256": sha(source), "remake": str(one), "second_remake": str(two), "remake_sha256": sha(one), "indexed_sha256": row["indexed_sha256"], "rgba_diff_pixels": 0})
        representatives.setdefault(row["mouth_frame"], (original.convert("RGB"), a.convert("RGB").resize((320, 200), Image.Resampling.NEAREST)))
    assert set(representatives) == {0, 3}
    report = {"kind": "fd2_confirmation_mouth_independent_receipt", "issue": 177, "complete_ticks": 1561, "random_reads": 59, "frames": results, "indexed_and_full_rgba_verified": True, "two_runs_equal": True, "scope": "建構第4章槽，正常LOAD與正式Update/Draw；受控原版亂數餘數及合格BIOS輸入，不注入mouth/frame/countdown；不增加整章PLAYER-E2"}
    if args.go_log:
        counts = {"packages_pass": 0, "top_tests_pass": 0, "all_test_cases_pass": 0, "test_cases_skip": 0, "packages_without_tests": 0, "failures": 0}
        for line in args.go_log.read_text().splitlines():
            try:
                event = json.loads(line)
            except json.JSONDecodeError:
                continue
            if event.get("Action") == "fail":
                counts["failures"] += 1
            if event.get("Action") == "pass":
                if "Test" not in event:
                    counts["packages_pass"] += 1
                else:
                    counts["all_test_cases_pass"] += 1
                    counts["top_tests_pass"] += "/" not in event["Test"]
            if event.get("Action") == "skip":
                counts["test_cases_skip" if "Test" in event else "packages_without_tests"] += 1
        assert counts["failures"] == 0 and counts["packages_pass"] == 19
        report["go_suite"] = {"log": str(args.go_log), "sha256": sha(args.go_log), **counts}
    if args.figure:
        canvas = Image.new("RGB", (640, 400))
        for index, mouth in enumerate((0, 3)):
            for column, image in enumerate(representatives[mouth]):
                canvas.paste(image, (column * 320, index * 200))
        canvas.save(args.figure)
        report["figure"] = {"path": str(args.figure), "sha256": sha(args.figure), "layout": "左原版、右正式GPU；上frame0、下frame3。有限總攬，非可回填完整素材。"}
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print("1561次完整刻度；88張完整索引／RGBA零差異；兩輪PNG雜湊相同")


if __name__ == "__main__":
    main()
