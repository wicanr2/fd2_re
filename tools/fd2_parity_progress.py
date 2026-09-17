#!/usr/bin/env python3
"""全戰役原版一致目標（111）的進度台帳：docs/data/parity-campaign-progress.json。

台帳是資料不是散文：30 章各一筆，`status` 只能是 todo／slot-ready／oracle-done／
passed／blocked；`todo` 以外每一筆都要有 `verify`（可重跑的命令）與 `receipt`。
頂層 `all_chapters_passed` 只由本工具在 30 章都 passed 時寫成 true——
issue #14 的 verify 盯的就是這個欄位。

用法：
  fd2_parity_progress.py init                    建立 30 筆 todo（已存在就拒絕）
  fd2_parity_progress.py verify                  檢查 schema、收據路徑、雜湊格式並回填 all_chapters_passed
  fd2_parity_progress.py set <章> --status … [--receipt … --slot-sha256 … --dosgolem-commit … --limitation …]
  fd2_parity_progress.py render                  印一張 30 章狀態表（給 91／README 引用）
"""

from __future__ import annotations

import argparse
import datetime as dt
import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LEDGER = ROOT / "docs" / "data" / "parity-campaign-progress.json"
STATUSES = ("todo", "slot-ready", "oracle-done", "passed", "blocked")
HEX64 = re.compile(r"^[0-9a-f]{64}$")
HEX40 = re.compile(r"^[0-9a-f]{40}$")


def load() -> dict:
    return json.loads(LEDGER.read_text(encoding="utf-8"))


def save(data: dict) -> None:
    data["updated"] = dt.date.today().isoformat()
    data["all_chapters_passed"] = all(c["status"] == "passed" for c in data["chapters"])
    LEDGER.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def empty_chapter(n: int) -> dict:
    return {
        "chapter": n,
        "status": "todo",
        "receipt": None,
        "slot_manifest": None,
        "slot_sha256": None,
        "dosgolem_commit": None,
        "date": None,
        "verify": None,
        "limitations": [],
        "prior_receipts": [],
    }


def cmd_init(_: argparse.Namespace) -> int:
    if LEDGER.exists():
        raise SystemExit(f"{LEDGER} 已存在；要重建先手動刪掉")
    data = {
        "schema_version": 1,
        "kind": "fd2_parity_campaign_progress",
        "policy": "docs/goal/111-goal-original-parity-campaign-20260915.md",
        "updated": None,
        "all_chapters_passed": False,
        "chapters": [empty_chapter(n) for n in range(1, 31)],
    }
    save(data)
    print(f"已建立 {LEDGER}（30 筆 todo）")
    return 0


def problems(data: dict) -> list[str]:
    out: list[str] = []
    chapters = data.get("chapters", [])
    if [c.get("chapter") for c in chapters] != list(range(1, 31)):
        out.append("chapters 必須是 1..30 各一筆、依序")
    for c in chapters:
        n = c.get("chapter")
        status = c.get("status")
        if status not in STATUSES:
            out.append(f"ch{n}: status={status!r} 不在 {STATUSES}")
            continue
        if status == "todo":
            continue
        if not c.get("verify"):
            out.append(f"ch{n}: {status} 但沒有 verify 命令")
        if status in ("oracle-done", "passed"):
            receipt = c.get("receipt")
            if not receipt or not (ROOT / receipt).is_file():
                out.append(f"ch{n}: {status} 但 receipt 不存在：{receipt}")
        if status != "todo" and c.get("slot_sha256") and not HEX64.fullmatch(c["slot_sha256"]):
            out.append(f"ch{n}: slot_sha256 不是 64 位十六進位")
        if c.get("dosgolem_commit") and not HEX40.fullmatch(c["dosgolem_commit"]):
            out.append(f"ch{n}: dosgolem_commit 不是完整 40 位雜湊")
        if status == "blocked" and not c.get("limitations"):
            out.append(f"ch{n}: blocked 必須寫原因到 limitations")
        if status == "passed":
            policy = c.get("slot_policy")
            # 建構槽政策（升級、seed、戰場狀態、AP／DP／DX 強化）決定原版與重製端 LOAD 的 bytes；
            # 114 起每一章都要寫明，缺了就無法從台帳重建同一個槽。
            if not isinstance(policy, dict) or not {"levels_per_chapter", "seed", "event_states", "boost"} <= policy.keys():
                out.append(f"ch{n}: passed 但 slot_policy 缺 levels_per_chapter／seed／event_states／boost")
        if status == "passed" and c.get("receipt") and (ROOT / c["receipt"]).is_file():
            receipt = json.loads((ROOT / c["receipt"]).read_text(encoding="utf-8"))
            if receipt.get("status") != "passed":
                out.append(f"ch{n}: 台帳 passed 但收據 status={receipt.get('status')!r}")
    return out


def cmd_verify(_: argparse.Namespace) -> int:
    data = load()
    issues = problems(data)
    before = data.get("all_chapters_passed")
    save(data)
    after = data["all_chapters_passed"]
    counts = {s: sum(1 for c in data["chapters"] if c["status"] == s) for s in STATUSES}
    print("狀態：" + "、".join(f"{k}={v}" for k, v in counts.items()))
    print(f"all_chapters_passed={after}（原 {before}）")
    if issues:
        print("\n".join("  ✗ " + i for i in issues))
        return 1
    print("台帳一致")
    return 0


def cmd_set(args: argparse.Namespace) -> int:
    data = load()
    entry = data["chapters"][args.chapter - 1]
    if args.status:
        if args.status not in STATUSES:
            raise SystemExit(f"status 必須是 {STATUSES}")
        entry["status"] = args.status
    for key in ("receipt", "slot_manifest", "slot_sha256", "dosgolem_commit", "verify"):
        value = getattr(args, key)
        if value is not None:
            entry[key] = value
    if args.limitation:
        entry["limitations"] = list(args.limitation)
    if args.prior_receipt:
        entry["prior_receipts"] = list(args.prior_receipt)
    entry["date"] = args.date or dt.date.today().isoformat()
    issues = [i for i in problems(data) if i.startswith(f"ch{args.chapter}:")]
    if issues:
        print("\n".join("  ✗ " + i for i in issues))
        return 1
    save(data)
    print(f"ch{args.chapter:02d} → {entry['status']}")
    return 0


def cmd_render(_: argparse.Namespace) -> int:
    data = load()
    print("| 章 | 狀態 | 收據 | 日期 | 限制 |")
    print("|---|---|---|---|---|")
    for c in data["chapters"]:
        receipt = f"[{Path(c['receipt']).name}](../{c['receipt']})" if c.get("receipt") else ""
        limits = "；".join(c.get("limitations", []))
        print(f"| {c['chapter']:02d} | {c['status']} | {receipt} | {c.get('date') or ''} | {limits} |")
    print(f"\nall_chapters_passed={data['all_chapters_passed']}（更新 {data.get('updated')}）")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("init").set_defaults(func=cmd_init)
    sub.add_parser("verify").set_defaults(func=cmd_verify)
    sub.add_parser("render").set_defaults(func=cmd_render)
    p = sub.add_parser("set")
    p.add_argument("chapter", type=int, choices=range(1, 31), metavar="章")
    p.add_argument("--status", choices=STATUSES)
    p.add_argument("--receipt")
    p.add_argument("--slot-manifest")
    p.add_argument("--slot-sha256")
    p.add_argument("--dosgolem-commit")
    p.add_argument("--verify")
    p.add_argument("--date")
    p.add_argument("--limitation", action="append")
    p.add_argument("--prior-receipt", action="append")
    p.set_defaults(func=cmd_set)
    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
