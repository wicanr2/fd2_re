#!/usr/bin/env python3
"""由台帳與章收據產生「逐章對拍進度」區塊，寫進 README.md 與 docs/REMAKE-STATUS.md。

數字全部從 docs/data/parity-campaign-progress.json 與 docs/data/ui-traces/parity-chNN.json
讀出來，文件裡不留手抄統計（手抄的那種下一章跑完就會漂走）。

用法：
  tools/render_parity_progress.py            重寫兩份文件的產生區塊
  tools/render_parity_progress.py --check    只檢查區塊是不是已經和資料一致（CI／提交前用）

區塊以 BEGIN／END 註解標出，和 tools/fd2_worklist.py 的作法一樣；區塊外的敘述由人維護。
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LEDGER = ROOT / "docs/data/parity-campaign-progress.json"
RECEIPTS = ROOT / "docs/data/ui-traces"
FIGURES = ROOT / "docs/figures"

BEGIN = "<!-- BEGIN tools/render_parity_progress.py render；不要手改這一段 -->"
END = "<!-- END tools/render_parity_progress.py render -->"

GATE_ORDER = ["behavior", "nodes", "transaction", "frames"]


def load_ledger() -> dict:
    return json.loads(LEDGER.read_text(encoding="utf-8"))


def receipt_stats(chapter: int) -> dict | None:
    path = RECEIPTS / f"parity-ch{chapter:02d}.json"
    if not path.exists():
        return None
    receipt = json.loads(path.read_text(encoding="utf-8"))
    gates = receipt.get("gates", {})
    frames = gates.get("frames", {}).get("points", [])
    diffs = [int(point.get("diff_pixels", 0)) for point in frames]
    behavior = gates.get("behavior", {}).get("points", [])
    save = gates.get("transaction", {}).get("save", {})
    return {
        "receipt": path.relative_to(ROOT).as_posix(),
        "actions": receipt.get("original", {}).get("actions"),
        "behavior_points": len(behavior),
        "frame_points": len(frames),
        "frame_zero": sum(1 for value in diffs if value == 0),
        "frame_max": max(diffs) if diffs else 0,
        "gates": {name: bool(gates.get(name, {}).get("ok")) for name in GATE_ORDER},
        # 交易 gate 已經涵蓋存檔，這裡另外留一個旗標讓表格可以直接說「整檔相同」。
        "save_match": bool(save) and save.get("oracle_save_sha256") == save.get("remake_save_sha256"),
    }


def sample_sheets(chapter: int) -> list[str]:
    pattern = f"parity-ch{chapter:02d}-samples"
    pages = sorted(
        path.relative_to(ROOT).as_posix()
        for path in FIGURES.glob(f"{pattern}*.png")
    )
    return pages


def doc_link(path: str, style: str) -> str:
    """README 在儲存庫根、REMAKE-STATUS 在 docs/ 底下，同一個相對路徑要換前綴。"""
    if style == "readme":
        return path
    return path[len("docs/"):] if path.startswith("docs/") else path


def render_rows(ledger: dict, style: str) -> tuple[list[str], list[int], list[int]]:
    rows: list[str] = []
    passed: list[int] = []
    pending: list[int] = []
    for entry in ledger["chapters"]:
        chapter = entry["chapter"]
        if entry.get("status") != "passed":
            pending.append(chapter)
            continue
        passed.append(chapter)
        stats = receipt_stats(chapter)
        if stats is None:
            rows.append(f"| 第 {chapter} 章 | 通過 | 收據檔缺失 | — | — | — | — | — |")
            continue
        gates = "、".join(
            name for name, ok in zip(("行為", "節點", "交易", "畫面"), stats["gates"].values()) if ok
        )
        if not all(stats["gates"].values()):
            gates += "（其餘未過）"
        sheets = sample_sheets(chapter)
        sheet_cell = "—"
        if sheets:
            first = sheets[0]
            sheet_cell = f"[{len(sheets)} 張]({doc_link(first, style)})"
        rows.append(
            "| 第 {chapter} 章 | {gates} | {actions} | {behavior} | {frames}（{zero} 張逐像素相同） | "
            "{maximum} px | {save} | [收據]({receipt}) ／ {sheet} |".format(
                chapter=chapter,
                gates=gates,
                actions=stats["actions"] if stats["actions"] is not None else "—",
                behavior=stats["behavior_points"],
                frames=stats["frame_points"],
                zero=stats["frame_zero"],
                maximum=stats["frame_max"],
                save="整檔相同" if stats["save_match"] else "未比",
                receipt=doc_link(stats["receipt"], style),
                sheet=sheet_cell,
            )
        )
    return rows, passed, pending


def render_block(ledger: dict, style: str) -> str:
    rows, passed, pending = render_rows(ledger, style)
    updated = ledger.get("updated", "")
    lines = [BEGIN, ""]
    if passed:
        if len(passed) == 1:
            span = f"第 {passed[0]} 章"
        elif passed[-1] - passed[0] == len(passed) - 1:
            span = f"第 {passed[0]}～{passed[-1]} 章"
        else:
            span = "、".join(f"第 {chapter} 章" for chapter in passed)
        lines.append(
            f"依 [111]({'docs/goal' if style == 'readme' else 'goal'}/111-goal-original-parity-campaign-20260915.md)"
            f" 的四個 gate（行為、節點、交易、畫面）逐章對拍，{span}已通過（台帳更新日 {updated}）。"
        )
    else:
        lines.append(f"逐章對拍還沒有通過的章（台帳更新日 {updated}）。")
    lines.append("")
    lines.append("| 章 | 通過的 gate | 原版動作 | 行為比較點 | 畫面比較點 | 最大畫面差異 | 酒店存檔 | 證據 |")
    lines.append("|---|---|---|---|---|---|---|---|")
    lines.extend(rows)
    lines.append("")
    if pending:
        head = "、".join(f"第 {chapter} 章" for chapter in pending[:3])
        lines.append(
            f"其餘各章（{head}…共 {len(pending)} 章）還沒跑這套逐章對拍；"
            "第 1～3 章另有更早的單點證據，不列在這張表裡。"
        )
        lines.append("")
    lines.append(
        "每章的建構槽政策、抽樣範圍與限制寫在台帳 "
        f"[`parity-campaign-progress.json`]({'docs/data' if style == 'readme' else 'data'}/parity-campaign-progress.json) "
        "的 `limitations`；這張表由 `tools/render_parity_progress.py` 依台帳與收據產生。"
    )
    lines.append("")
    lines.append(END)
    return "\n".join(lines)


def write_block(path: Path, block: str, check: bool) -> bool:
    text = path.read_text(encoding="utf-8")
    if BEGIN not in text or END not in text:
        raise SystemExit(f"{path} 找不到產生區塊的標記")
    head, rest = text.split(BEGIN, 1)
    _, tail = rest.split(END, 1)
    updated = head + block + tail
    if updated == text:
        return False
    if not check:
        path.write_text(updated, encoding="utf-8")
    return True


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--check", action="store_true", help="只檢查，不寫檔")
    args = parser.parse_args()
    ledger = load_ledger()
    stale = []
    for path, style in ((ROOT / "README.md", "readme"), (ROOT / "docs/REMAKE-STATUS.md", "status")):
        if write_block(path, render_block(ledger, style), args.check):
            stale.append(path.relative_to(ROOT).as_posix())
    if args.check:
        if stale:
            raise SystemExit("產生區塊與台帳不一致：" + "、".join(stale))
        print("產生區塊與台帳一致")
        return
    print("已重寫產生區塊：" + ("、".join(stale) if stale else "（內容沒有變）"))


if __name__ == "__main__":
    main()
