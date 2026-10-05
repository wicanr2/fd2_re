#!/usr/bin/env python3
"""#35：只補驗既有第4、5章確認框四點，不重建整章通過聲明。

在 fd2-assets-local:20260829-sfx 容器中執行：
  python3 tools/fd2_confirmation_pulse_receipts.py --run work/confirmation-phase-four-points-r4
"""

import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image, ImageChops


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run", required=True, type=Path)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    run = args.run.resolve()
    run.relative_to(root / "work")
    records = json.loads((run / "validation.json").read_text())
    expected = {(4, 38, "departure_prompt"), (4, 1105, "town_enter"),
                (5, 38, "departure_prompt"), (5, 1335, "town_enter")}
    assert len(records) == 4 and {(r["chapter"], r["seq"], r["kind"]) for r in records} == expected
    prepared = {}
    evidence_path = root / "docs/data/ida/fd2_confirmation_pulse_20261005.json"
    evidence = json.loads(evidence_path.read_text())
    evidence.setdefault("historical_receipts", {})
    for record in records:
        assert record["gpu_rgba_diff"] == [0, 0]
        chapter = record["chapter"]
        receipt_path = root / f"docs/data/ui-traces/parity-ch{chapter:02}.json"
        if chapter not in prepared:
            receipt = json.loads(receipt_path.read_text())
            prepared[chapter] = (receipt_path, receipt)
            evidence["historical_receipts"].setdefault(str(receipt_path.relative_to(root)), {
                "sha256": sha(receipt_path), "points": []})
        receipt_path, receipt = prepared[chapter]
        matches = [p for p in receipt["gates"]["frames"]["points"]
                   if p["seq"] == record["seq"] and p["kind"] == record["kind"]]
        assert len(matches) == 1
        point = matches[0]
        original = Path(record["original"])
        assert sha(original) == point["oracle_sha256"]
        candidates = [run / f"ch{chapter:02}-{record['kind']}-{record['seq']}-p{i}.png" for i in range(2)]
        with Image.open(original) as image:
            original_rgb = image.convert("RGB")
        diffs = []
        for candidate in candidates:
            with Image.open(candidate) as image:
                assert image.size == (320, 200)
                delta = ImageChops.difference(original_rgb, image.convert("RGB"))
                diffs.append(sum(pixel != (0, 0, 0) for pixel in delta.getdata()))
        assert diffs == record["diff_pixels"] == [60, 0]
        history = evidence["historical_receipts"][str(receipt_path.relative_to(root))]["points"]
        if not any(old["seq"] == point["seq"] for old in history):
            history.append(dict(point))
        point.update(remake=candidates[1].name, phases=2, diff_pixels=0, box=[], ok=True,
                     remake_sha256=sha(candidates[1]),
                     verification_source="docs/data/ida/fd2_confirmation_pulse_20261005.json",
                     verification_run=str(run.relative_to(root)),
                     verification_scope="正常LOAD確認框合法兩相位完整RGB與GPU；戰後點只比較排版，未重跑整章")
        receipt.setdefault("followup_verifications", {})["issue35"] = {
            "date": "2026-10-05", "source": "docs/data/ida/fd2_confirmation_pulse_20261005.json",
            "scope": "只更新兩個確認框畫面點；其他原版、行為、節點、交易與畫面結論保持歷史來源"}
    for receipt_path, receipt in prepared.values():
        receipt_path.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
    evidence_path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n")
    print("#35四個完整RGB點均0；八個GPU圖均0；原始四點與章收據雜湊已保存")


if __name__ == "__main__":
    main()
