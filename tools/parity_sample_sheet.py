#!/usr/bin/env python3
"""把一章對拍收據的畫面點抽樣成一張「原版｜重製｜差異」對照表。

收據（`tools/verify_chapter_parity.py` 寫的 `parity-chNN.json`）只有數字；這張圖讓人
不用跑工具就看得到同一個 checkpoint 兩側長什麼樣、差在哪幾個像素。每一列三格：
原版 checkpoint PNG、重製 remake-NNNN-pK.png（verifier 取最小差的那一張變體）、差異
遮罩（不同的像素畫白、其餘畫黑，並框出收據的 box）。格上標 seq、kind、diff_pixels 與
兩側 sha256 前 8 碼；index JSON 記每一列對到收據哪一點，讓人能從圖回查收據。

抽樣規則（固定，不挑好看的）：每種 kind 至少一點（取該 kind 第一個）、`diff_pixels>0`
的全部、`--include-seq` 指定的點；依 seq 排序。原生 320×200 不縮放。

用法（在 fd2-assets-local 容器內，它有 Pillow）：
  parity_sample_sheet.py --receipt docs/data/ui-traces/parity-ch07.json \\
      --oracle <dosgolem run 目錄> --remake <重製側輸出目錄> \\
      --out docs/figures/parity-ch07-samples.png \\
      --index docs/data/ui-traces/parity-ch07-samples.json [--include-seq 123 456] [--max-bytes 1500000]

一張超過 --max-bytes 就依列數平均切成 -p1、-p2…（每張各自受上限，index 的每一列記在哪一張）。
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import sys
from pathlib import Path

try:
    from PIL import Image, ImageDraw
except ImportError:  # pragma: no cover - 容器外沒有 Pillow
    Image = None
    ImageDraw = None

FRAME_W, FRAME_H = 320, 200
LABEL_H = 14
GAP = 4
MARGIN = 6


def sha256_of(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def select_points(points: list[dict], include_seq: set[int]) -> list[dict]:
    """固定抽樣規則：每種 kind 第一點、所有 diff>0、指定 seq；依 seq 排序去重。"""
    chosen: dict[int, dict] = {}
    seen_kind: set[str] = set()
    for point in points:
        kind = point.get("kind", "")
        if kind not in seen_kind:
            seen_kind.add(kind)
            chosen[point["seq"]] = point
        if point.get("diff_pixels", 0) > 0 or point["seq"] in include_seq:
            chosen[point["seq"]] = point
    return [chosen[seq] for seq in sorted(chosen)]


def diff_mask(a: "Image.Image", b: "Image.Image") -> "Image.Image":
    mask = Image.new("RGB", (FRAME_W, FRAME_H), (0, 0, 0))
    pa, pb, pm = a.load(), b.load(), mask.load()
    for y in range(FRAME_H):
        for x in range(FRAME_W):
            if pa[x, y] != pb[x, y]:
                pm[x, y] = (255, 255, 255)
    return mask


def build_sheet(receipt: dict, oracle_dir: Path, remake_dir: Path, points: list[dict],
                oracle_label: str, remake_label: str) -> tuple["Image.Image", list[dict]]:
    columns = 3
    row_h = LABEL_H + FRAME_H + GAP
    width = MARGIN * 2 + columns * FRAME_W + (columns - 1) * GAP
    header_h = LABEL_H * 2
    height = MARGIN * 2 + header_h + len(points) * row_h
    sheet = Image.new("RGB", (width, height), (24, 24, 24))
    draw = ImageDraw.Draw(sheet)
    chapter = receipt.get("chapter")
    original = receipt.get("original", {})
    draw.text((MARGIN, MARGIN),
              f"parity ch{chapter:02d}  status={receipt.get('status')}  original={oracle_label}"
              f" (dosgolem {str(original.get('dosgolem_commit', ''))[:8]})  remake={remake_label}  "
              f"{len(points)} of {len(receipt['gates']['frames']['points'])} frame points",
              fill=(230, 230, 230))
    draw.text((MARGIN, MARGIN + LABEL_H), "columns: original checkpoint | remake frame (min-diff variant) | diff mask (white = differs, box = receipt box)",
              fill=(160, 160, 160))
    index_rows = []
    for row, point in enumerate(points):
        top = MARGIN + header_h + row * row_h
        oracle_png = oracle_dir / point["oracle"]
        remake_png = remake_dir / point["remake"]
        a = Image.open(oracle_png).convert("RGB")
        b = Image.open(remake_png).convert("RGB")
        if a.size != (FRAME_W, FRAME_H) or b.size != (FRAME_W, FRAME_H):
            raise ValueError(f"seq {point['seq']}：畫面尺寸不是 320×200：{a.size} vs {b.size}")
        oracle_sha, remake_sha = sha256_of(oracle_png), sha256_of(remake_png)
        if oracle_sha != point.get("oracle_sha256") or remake_sha != point.get("remake_sha256"):
            raise ValueError(f"seq {point['seq']}：run 目錄裡的 PNG 與收據記的 sha256 不同，這不是收據那一輪的圖")
        mask = diff_mask(a, b)
        box = point.get("box") or []
        if len(box) == 4:
            ImageDraw.Draw(mask).rectangle([box[0] - 1, box[1] - 1, box[2] + 1, box[3] + 1], outline=(255, 64, 64))
        diff = point.get("diff_pixels", 0)
        label = (f"seq {point['seq']}  {point['kind']}  diff={diff}px  "
                 f"orig {oracle_sha[:8]}  remake {remake_sha[:8]}  "
                 f"{'ok' if point.get('ok') else 'over budget'}  variants={point.get('phases', 1)}")
        draw.text((MARGIN, top), label, fill=(230, 230, 230) if diff == 0 else (255, 200, 120))
        for col, image in enumerate((a, b, mask)):
            sheet.paste(image, (MARGIN + col * (FRAME_W + GAP), top + LABEL_H))
        index_rows.append({
            "row": row, "seq": point["seq"], "kind": point["kind"], "diff_pixels": diff,
            "box": box, "ok": point.get("ok"), "phases": point.get("phases"),
            "oracle": point["oracle"], "oracle_sha256": oracle_sha,
            "remake": point["remake"], "remake_sha256": remake_sha,
        })
    return sheet, index_rows


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--receipt", required=True)
    parser.add_argument("--oracle", required=True)
    parser.add_argument("--remake", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--index", required=True)
    parser.add_argument("--include-seq", type=int, nargs="*", default=[])
    parser.add_argument("--oracle-label", default="", help="標在表頭的原版側 run 名（容器內路徑看不出來）")
    parser.add_argument("--remake-label", default="", help="標在表頭的重製側 run 名")
    parser.add_argument("--max-bytes", type=int, default=1_500_000)
    args = parser.parse_args()
    if Image is None:
        print("需要 Pillow：在 fd2-assets-local 容器內跑", file=sys.stderr)
        return 2
    receipt_path = Path(args.receipt)
    receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
    points = receipt["gates"]["frames"]["points"]
    chosen = select_points(points, set(args.include_seq))
    if not chosen:
        print("收據沒有畫面點", file=sys.stderr)
        return 1
    oracle_label = args.oracle_label or Path(args.oracle).name
    remake_label = args.remake_label or Path(args.remake).name
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    # 一張放不進 --max-bytes 時依列數平均切成幾張（-p1、-p2…），每張各自受上限；抽樣規則
    # 不因此縮水（第七章 91 列一張要 4 MB 以上）。切分是無損的，index 記每一列在哪一張。
    parts = 1
    while True:
        chunk = -(-len(chosen) // parts)
        groups = [chosen[i:i + chunk] for i in range(0, len(chosen), chunk)]
        sheets = []
        for group in groups:
            sheet, rows = build_sheet(receipt, Path(args.oracle), Path(args.remake), group,
                                      oracle_label, remake_label)
            sheets.append((sheet, rows))
        paths = [out] if len(groups) == 1 else [
            out.with_name(f"{out.stem}-p{i + 1}{out.suffix}") for i in range(len(groups))]
        for path, (sheet, _) in zip(paths, sheets):
            sheet.save(path, optimize=True)
        sizes = [path.stat().st_size for path in paths]
        if max(sizes) <= args.max_bytes:
            break
        for path in paths:
            path.unlink()
        parts += 1
        if parts > len(chosen):
            print(f"對照表單列就超過上限 {args.max_bytes}；提高 --max-bytes", file=sys.stderr)
            return 3
    rows = []
    for part, (path, (_, part_rows)) in enumerate(zip(paths, sheets)):
        for row in part_rows:
            row["sheet"] = str(path)
            row["row"] = len(rows)
            rows.append(row)
    index = {
        "schema_version": 1,
        "kind": "fd2_parity_sample_sheet",
        "chapter": receipt.get("chapter"),
        "generated_at": dt.datetime.now().astimezone().isoformat(timespec="seconds"),
        "receipt": str(receipt_path),
        "receipt_sha256": sha256_of(receipt_path),
        "sheet": str(out) if len(paths) == 1 else [str(path) for path in paths],
        "sheet_sha256": sha256_of(out) if len(paths) == 1 else [sha256_of(path) for path in paths],
        "sheet_bytes": sizes[0] if len(paths) == 1 else sizes,
        "oracle_run": oracle_label,
        "remake_run": remake_label,
        "selection_rule": "每種 kind 第一點、所有 diff_pixels>0、--include-seq 指定；依 seq 排序",
        "include_seq": sorted(set(args.include_seq)),
        "rows": rows,
    }
    Path(args.index).write_text(json.dumps(index, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"{', '.join(str(p) for p in paths)}：{len(rows)} 列、{sizes} bytes；index {args.index}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
