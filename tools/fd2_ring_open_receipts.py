#!/usr/bin/env python3
"""以完整畫面核對 #34 的有限開框候選，保存每點來源與所有剩餘差異。"""
import argparse
import hashlib
import json
import re
from pathlib import Path

from PIL import Image, ImageChops


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def rows(path):
    return [json.loads(line) for line in path.read_text().splitlines()]


def difference(a, b):
    delta = ImageChops.difference(a, b)
    if delta.mode == "RGB":
        red, green, blue = delta.split()
        delta = ImageChops.lighter(ImageChops.lighter(red, green), blue)
    return a.width * a.height - delta.histogram()[0], delta.getbbox()


def candidates(root, checkpoint, original):
    stem = re.sub(r"-p\d+\.png$", "", checkpoint["frame"])
    files = sorted(root.glob(stem + "-p*.png"))
    assert files, (root, checkpoint)
    out = []
    for path in files:
        candidate = Image.open(path)
        assert candidate.mode == "P" and candidate.size == original.size == (320, 200)
        rgb, box = difference(original.convert("RGB"), candidate.convert("RGB"))
        indexed, indexed_box = difference(original.convert("L") if original.mode != "P" else Image.frombytes("L", original.size, original.tobytes()), Image.frombytes("L", candidate.size, candidate.tobytes()))
        number = int(re.search(r"-p(\d+)\.png$", path.name).group(1))
        group = [g for g in checkpoint.get("action_overlay_candidates", [])
                 if g["first_variant"] <= number < g["first_variant"] + g["variant_count"]]
        assert len(group) <= 1
        out.append({"file": str(path), "png_sha256": sha(path),
                    "indexed_sha256": hashlib.sha256(candidate.tobytes()).hexdigest(),
                    "diff_pixels": rgb, "box": box, "indexed_diff_pixels": indexed,
                    "indexed_box": indexed_box, "candidate": group[0] if group else None})
    return sorted(out, key=lambda row: (row["diff_pixels"], row["file"]))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--chapter", type=int, required=True)
    parser.add_argument("--oracle", type=Path, required=True)
    parser.add_argument("--before", type=Path, required=True)
    parser.add_argument("--after", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    before = {(p["kind"], p["oracle_seq"]): p for p in rows(args.before / "checkpoints.jsonl") if p.get("frame") and p["kind"] in ("move", "stay")}
    after = {(p["kind"], p["oracle_seq"]): p for p in rows(args.after / "checkpoints.jsonl") if p.get("frame") and p["kind"] in ("move", "stay")}
    assert before.keys() == after.keys()
    points = []
    for key, checkpoint in after.items():
        source = args.oracle / f"checkpoint-{key[1]:04d}.png"
        original = Image.open(source)
        assert original.mode == "P"
        old = candidates(args.before, before[key], original)
        new = candidates(args.after, checkpoint, original)
        assert new[0]["diff_pixels"] <= old[0]["diff_pixels"], key
        assert len(checkpoint["action_overlay_candidates"]) == 6
        unchanged = lambda p: {k: v for k, v in p.items() if k not in ("frame", "indexed_sha256", "palette_cycle_phase", "palette_cycle_write", "action_overlay_candidates")}
        assert unchanged(before[key]) == unchanged(checkpoint), "候選修改玩家狀態"
        points.append({"kind": key[0], "seq": key[1], "source": str(source), "source_sha256": sha(source),
                       "source_indexed_sha256": hashlib.sha256(original.tobytes()).hexdigest(),
                       "before": old[0], "after": new[0], "all_candidates": new})
    report = {"kind": "fd2_ring_open_independent_receipt", "chapter": args.chapter,
              "points": points, "complete_frame_rgb_zero": sum(p["after"]["diff_pixels"] == 0 for p in points),
              "complete_frame_indexed_zero": sum(p["after"]["indexed_diff_pixels"] == 0 for p in points),
              "remaining": [{"seq": p["seq"], "kind": p["kind"], "diff_pixels": p["after"]["diff_pixels"], "indexed_diff_pixels": p["after"]["indexed_diff_pixels"], "box": p["after"]["box"]} for p in points if p["after"]["diff_pixels"] != 0],
              "input_sha256": {str(p): sha(p) for p in (args.oracle / "runner.json", args.oracle / "actions.jsonl", args.before / "checkpoints.jsonl", args.after / "checkpoints.jsonl")},
              "scope": "歷史 dosgolem 原版收據的完整有限候選補驗；不新增章 PLAYER-E2；沒有遮罩或提高預算"}
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({k: v for k, v in report.items() if k in ("chapter", "complete_frame_rgb_zero", "complete_frame_indexed_zero", "remaining")}, ensure_ascii=False))


if __name__ == "__main__":
    main()
