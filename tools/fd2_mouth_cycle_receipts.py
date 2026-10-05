#!/usr/bin/env python3
"""#38：補驗第六章三個嘴型／DAC點，保留原章的其他來源與結論。

使用 fd2-assets-local:20260829-sfx 的 Pillow；完整命令見
docs/data/ida/fd2_parity_mouth_cycle_20261005.json。
"""

import argparse
import hashlib
import json
import re
from pathlib import Path

from PIL import Image, ImageChops


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("first", "second", "first-remake", "second-remake", "oracle"):
        parser.add_argument("--" + name, type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    for path in vars(args).values():
        path.resolve().relative_to(root / "work")
    reports = [json.loads(p.read_text()) for p in (args.first, args.second)]
    for report in reports:
        assert report["chapter"] == 6 and all(g["ok"] for g in report["gates"].values())
    receipt_path = root / "docs/data/ui-traces/parity-ch06.json"
    evidence_path = root / "docs/data/ida/fd2_parity_mouth_cycle_20261005.json"
    receipt = json.loads(receipt_path.read_text())
    evidence = json.loads(evidence_path.read_text())
    history = evidence["historical_receipts"].setdefault(str(receipt_path.relative_to(root)), {
        "sha256": sha(receipt_path), "points": []})
    points = []
    for seq, kind in ((619, "attack_armed"), (1053, "attack_armed"), (1603, "town_enter")):
        original_points = [p for p in receipt["gates"]["frames"]["points"] if p["seq"] == seq and p["kind"] == kind]
        assert len(original_points) == 1
        original_point = original_points[0]
        matched = []
        for report in reports:
            selected = [p for p in report["gates"]["frames"]["points"] if p["seq"] == seq and p["kind"] == kind]
            assert len(selected) == 1 and selected[0]["diff_pixels"] == 0
            matched.append(selected[0])
        original = args.oracle / matched[0]["oracle"]
        assert sha(original) == original_point["oracle_sha256"]
        with Image.open(original) as image:
            reference = image.convert("RGB")
        for run, point in zip((args.first_remake, args.second_remake), matched):
            assert re.fullmatch(r"remake-\d{4}-p\d+\.png", point["remake"])
            candidate = run / point["remake"]
            assert sha(candidate) == point["remake_sha256"] == matched[0]["remake_sha256"]
            with Image.open(candidate) as image:
                assert image.size == reference.size == (320, 200)
                assert not ImageChops.difference(reference, image.convert("RGB")).getbbox()
        if not any(p["seq"] == seq for p in history["points"]):
            history["points"].append(dict(original_point))
        original_point.update(matched[1])
        original_point.update(verification_source=str(evidence_path.relative_to(root)),
                              verification_run=str(args.second_remake),
                              verification_scope="有限嘴型與完整DAC窗口候選；正式mouth owner仍在#177")
        points.append(dict(original_point))
    receipt.setdefault("followup_verifications", {})["issue38"] = {
        "date": "2026-10-05", "source": str(evidence_path.relative_to(root)),
        "scope": "只補驗seq619／1053／1603；其他原版、行為、節點、交易與畫面結論保持歷史來源"}
    evidence["validation"] = {"points": points, "repeat_winner_sha_equal": True,
                              "report_paths": [str(args.first), str(args.second)],
                              "all_four_gates": True, "player_e2_added": False}
    receipt_path.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
    evidence_path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n")
    print("三點完整RGB為0，兩次勝出PNG雜湊相同；只更新指定點並保存歷史。")


if __name__ == "__main__":
    main()
