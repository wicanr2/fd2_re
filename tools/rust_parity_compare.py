#!/usr/bin/env python3
"""R4 階段 3：比對 Rust 與凍結 Go 版第一章章重播的輸出（docs/goal/120「驗證」第 2 層）。

用法：tools/rust_parity_compare.py --rust <Rust parity-out> --go <Go parity-out> [--receipt-rust R --receipt-go G]

- checkpoints.jsonl：逐位元組比對；不同時逐筆逐欄列出差異（單位依 native_index 對齊）。
- FD2.SAV：逐位元組比對；不同時列出差異 byte 數與前幾個偏移。
- hotel-save.json：逐位元組比對；不同時只在 fd2-rules serde 子樹（party_roster）套用 F 組的
  NORMALIZATIONS（remake-rs/crates/fd2-game/src/game/save/golden_save_test.rs）再比，結果分開報告。
- 給了兩份 verify_chapter_parity.py 收據時，比對 status 與各閘門的 ok。

逐位元組比對失敗時結束碼為 1；只在正規化後相同的 hotel-save.json 不算失敗，但會另列。
"""
from __future__ import annotations

import argparse
import hashlib
import json
import sys
from pathlib import Path

NORMALIZATIONS = [
    "battle 子樹：null 與空陣列、空物件、空字串相同",
    "battle 子樹：數字只比數值",
    "battle 子樹：物件鍵順序",
    "party_join_order：null 與 [] 相同",
]
BATTLE_SUBTREES = ("party_roster",)


def sha(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def norm(v):
    if v is None or v == [] or v == {} or v == "":
        return None
    if isinstance(v, bool):
        return v
    if isinstance(v, (int, float)):
        return float(v)
    if isinstance(v, list):
        return [norm(x) for x in v]
    if isinstance(v, dict):
        return {k: norm(x) for k, x in sorted(v.items())}
    return v


def first_diff(path: str, a, b) -> str:
    if isinstance(a, dict) and isinstance(b, dict):
        for k in list(a) + [k for k in b if k not in a]:
            if a.get(k) != b.get(k):
                return first_diff(f"{path}.{k}", a.get(k), b.get(k))
    if isinstance(a, list) and isinstance(b, list) and len(a) == len(b):
        for i, (x, y) in enumerate(zip(a, b)):
            if x != y:
                return first_diff(f"{path}[{i}]", x, y)
    return f"{path}：Rust {json.dumps(a, ensure_ascii=False)[:200]} ≠ Go {json.dumps(b, ensure_ascii=False)[:200]}"


def compare_checkpoints(rust: Path, go: Path, out: list[str]) -> bool:
    r, g = rust.read_bytes(), go.read_bytes()
    out.append(f"checkpoints.jsonl：Rust {hashlib.sha256(r).hexdigest()}（{r.count(b'\n')} 筆）"
               f"，Go {hashlib.sha256(g).hexdigest()}（{g.count(b'\n')} 筆）")
    if r == g:
        out.append("  逐位元組相同")
        return True
    rl, gl = r.decode().splitlines(), g.decode().splitlines()
    shown = 0
    for i in range(max(len(rl), len(gl))):
        if i < len(rl) and i < len(gl) and rl[i] == gl[i]:
            continue
        if shown >= 40:
            out.append("  ……其餘省略")
            break
        shown += 1
        if i >= len(rl) or i >= len(gl):
            side = "Rust 缺" if i >= len(rl) else "Go 缺"
            other = gl[i] if i >= len(rl) else rl[i]
            out.append(f"  #{i} {side}：{other[:300]}")
            continue
        a, b = json.loads(rl[i]), json.loads(gl[i])
        head = f"  #{i} {b.get('kind')}(seq {b.get('oracle_seq')})"
        diffs = []
        for k in list(b) + [k for k in a if k not in b]:
            if k == "units":
                continue
            if a.get(k, "<缺>") != b.get(k, "<缺>"):
                diffs.append(f"{k}: Rust {json.dumps(a.get(k, '<缺>'), ensure_ascii=False)} Go {json.dumps(b.get(k, '<缺>'), ensure_ascii=False)}")
        ua = {u["native_index"]: u for u in a.get("units") or []}
        ub = {u["native_index"]: u for u in b.get("units") or []}
        for idx in sorted(set(ua) | set(ub)):
            if ua.get(idx) != ub.get(idx):
                diffs.append(f"units[{idx}]: Rust {json.dumps(ua.get(idx))} Go {json.dumps(ub.get(idx))}")
        if not diffs and rl[i] != gl[i]:
            diffs.append("欄位值相同、位元組不同（欄位順序或數字格式）")
        out.append(head)
        out.extend("    " + d for d in diffs[:20])
    return False


def compare_save(rust: Path, go: Path, out: list[str]) -> bool:
    r, g = rust.read_bytes(), go.read_bytes()
    out.append(f"FD2.SAV：Rust {hashlib.sha256(r).hexdigest()}，Go {hashlib.sha256(g).hexdigest()}")
    if r == g:
        out.append("  逐位元組相同")
        return True
    diff = [i for i in range(min(len(r), len(g))) if r[i] != g[i]]
    out.append(f"  大小 Rust {len(r)}／Go {len(g)}；不同 byte {len(diff)} 個；前幾個偏移 {[hex(i) for i in diff[:16]]}")
    return False


def compare_hotel(rust: Path, go: Path, out: list[str]) -> tuple[bool, bool]:
    r, g = rust.read_bytes(), go.read_bytes()
    out.append(f"hotel-save.json：Rust {hashlib.sha256(r).hexdigest()}，Go {hashlib.sha256(g).hexdigest()}")
    if r == g:
        out.append("  逐位元組相同")
        return True, True
    a, b = json.loads(r), json.loads(g)
    problems = []
    for k in list(b) + [k for k in a if k not in b]:
        x, y = a.get(k, "<缺>"), b.get(k, "<缺>")
        if k in BATTLE_SUBTREES:
            if norm(x) != norm(y):
                problems.append(first_diff(k, norm(x), norm(y)))
        elif k == "party_join_order":
            if (x or None) != (y or None):
                problems.append(first_diff(k, x, y))
        elif x != y:
            problems.append(first_diff(k, x, y))
    if problems:
        out.append("  正規化後仍不同：")
        out.extend("    " + p for p in problems[:20])
        return False, False
    out.append("  位元組不同；只在 fd2-rules serde 子樹（party_roster）以 NORMALIZATIONS 比對後相同：")
    out.extend("    - " + n for n in NORMALIZATIONS)
    # 拿掉 battle 子樹後外層是否逐值相同（位元組差異只在子樹）
    return False, True


def receipt_summary(path: Path) -> dict:
    d = json.loads(path.read_text(encoding="utf-8"))
    gates = d.get("gates", {})
    return {"status": d.get("status"), **{f"gates.{k}.ok": v.get("ok") for k, v in gates.items() if isinstance(v, dict)},
            "save.status": gates.get("transaction", {}).get("save", {}).get("status")}


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--rust", type=Path, required=True)
    p.add_argument("--go", type=Path, required=True)
    p.add_argument("--receipt-rust", type=Path)
    p.add_argument("--receipt-go", type=Path)
    args = p.parse_args()
    out: list[str] = []
    ok = compare_checkpoints(args.rust / "checkpoints.jsonl", args.go / "checkpoints.jsonl", out)
    ok &= compare_save(args.rust / "FD2.SAV", args.go / "FD2.SAV", out)
    hotel_bytes, hotel_norm = compare_hotel(args.rust / "hotel-save.json", args.go / "hotel-save.json", out)
    ok &= hotel_norm
    if args.receipt_rust and args.receipt_go:
        a, b = receipt_summary(args.receipt_rust), receipt_summary(args.receipt_go)
        out.append(f"verify_chapter_parity.py：Rust {a}")
        out.append(f"verify_chapter_parity.py：Go   {b}")
        same = a == b
        out.append("  結論相同" if same else "  結論不同")
        ok &= same
    print("\n".join(out))
    print("結果：" + ("通過" if ok else "失敗") + ("" if hotel_bytes else "（hotel-save.json 依 NORMALIZATIONS）" if hotel_norm else ""))
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
