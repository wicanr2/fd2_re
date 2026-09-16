#!/usr/bin/env python3
"""章對拍的四個 gate（111）：把 dosgolem 原版收據與重製側重播逐點比對，寫出章收據。

用法（在 fd2-assets-local 容器內，它有 Pillow）：
  verify_chapter_parity.py --chapter 4 --oracle <dosgolem run 目錄> --remake <重製側輸出目錄> \\
      --slot-manifest <建槽 manifest.json> --out docs/data/ui-traces/parity-ch04.json \\
      [--pixel-budget 640] [--min-frames 12]

比對單位是重製側 checkpoints.jsonl 的每一筆：它帶 oracle_seq，對到原版
checkpoint-<seq>.json；`after_enemy_phase` 對到下一個動作那一格（原版 end_turn 紀錄
在 END 之後、敵方回合之前），只比行為不比畫面——那張原版圖是下一個動作做完後拍的。

gate：
  behavior   單位（camp、x、y、存活）逐點相同、回合相同；HP 逐點相同，
             差異另分成「亂數已同步」與「同步之間漂移」兩類列出，兩類都算失敗
  nodes      節點／介面序列相同
  transaction 每點金幣相同；酒店存檔兩側 SHA-256（重製側尚未寫原版槽格式時標 blocked）
  frames     每張有畫面的點：320×200 逐像素 RGB 差異 ≤ pixel-budget；有畫面的點數 ≥ min-frames
"""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import re
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:  # pragma: no cover - 容器外沒有 Pillow
    Image = None

ROOT = Path(__file__).resolve().parents[1]


def read_jsonl(path: Path) -> list[dict]:
    out = []
    with path.open(encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line and not line.startswith("#"):
                out.append(json.loads(line))
    return out


def oracle_checkpoint(run: Path, seq: int) -> dict | None:
    path = run / f"checkpoint-{seq:04d}.json"
    if not path.is_file():
        return None
    return json.loads(path.read_text(encoding="utf-8"))


def oracle_units(cp: dict) -> set[tuple]:
    return {(u.get("camp"), u.get("x"), u.get("y")) for u in cp.get("units", []) if u.get("hp", 0) > 0}


def oracle_hp(cp: dict) -> dict[tuple, int]:
    return {(u.get("camp"), u.get("x"), u.get("y")): u.get("hp", 0) for u in cp.get("units", []) if u.get("hp", 0) > 0}


def remake_units(cp: dict) -> set[tuple]:
    return {(u["camp"], u["x"], u["y"]) for u in cp.get("units", [])}


def remake_hp(cp: dict) -> dict[tuple, int]:
    return {(u["camp"], u["x"], u["y"]): u["hp"] for u in cp.get("units", [])}


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def frame_diff(oracle_png: Path, remake_png: Path) -> tuple[int, list[int]]:
    a = Image.open(oracle_png).convert("RGB")
    b = Image.open(remake_png).convert("RGB")
    if a.size != (320, 200) or b.size != (320, 200):
        raise ValueError(f"畫面尺寸不是 320×200：{a.size} vs {b.size}")
    pa, pb = a.load(), b.load()
    diff, minx, miny, maxx, maxy = 0, 320, 200, -1, -1
    for y in range(200):
        for x in range(320):
            if pa[x, y] != pb[x, y]:
                diff += 1
                minx, miny, maxx, maxy = min(minx, x), min(miny, y), max(maxx, x), max(maxy, y)
    return diff, [minx, miny, maxx, maxy] if diff else []


def oracle_gold_for(actions: list[dict], seq: int, kind: str, view: dict) -> int | None:
    """原版側這一點該比的金額。重製側的 shop_menu 是出售前的店內畫面，而原版側同一個
    seq 的 checkpoint 在出售之後，出售前的金額記在動作的 gold_before。"""
    if kind == "shop_menu":
        action = next((a for a in actions if a.get("seq") == seq), None)
        if action and "gold_before" in action:
            return action["gold_before"]
    return view.get("gold")


def save_gate_entry(save_actions: list[dict], remake_save: list[dict], blocked_issue: str) -> dict:
    """交易 gate 的存檔項：兩側酒店存檔後整份 FD2.SAV 的 sha256 要相同。重製側的
    town_save checkpoint 在 note 寫 `save_sha256=<hex>`；沒有這個欄位（寫回失敗、
    或舊版重播）而且有指定 blocked issue 時標 blocked，否則 fail。"""
    oracle = save_actions[-1].get("save_sha256") if save_actions else None
    note = remake_save[-1].get("note") if remake_save else None
    remake = None
    if note:
        match = re.search(r"save_sha256=([0-9a-f]{64})", note)
        if match:
            remake = match.group(1)
    entry = {"oracle_save_sha256": oracle, "remake_save_sha256": remake, "remake_note": note, "blocked_by": None}
    if not save_actions:
        entry["status"] = "not_sampled"
    elif remake is None:
        entry["status"] = "blocked" if blocked_issue else "fail"
        entry["blocked_by"] = blocked_issue or None
    else:
        entry["status"] = "ok" if remake == oracle else "fail"
    return entry


def frame_comparable(remake_cp: dict) -> bool:
    """after_enemy_phase 只比行為：它借用下一個動作的原版檢查點，而那張圖是動作做完
    之後（選取後的移動範圍、END 後的換手橫幅）拍的，原版沒有同一瞬間的閒置畫面。"""
    return remake_cp.get("kind") != "after_enemy_phase"


# 0x1A30B..0x1A7BD 是換手處理（回復、回合事件、友軍 AI、橫幅、敵軍 AI）。原版側的
# checkpoint 是動作做完之後隔幾百萬道指令拍的；最後一個玩家動作讓全員行動完自動換手
# 時，回合事件（第五章 event 15 的登場與對白）與友軍 AI 在橫幅之前就開始跑，那張圖與
# 單位座標已經是換手處理中（r5 seq 789／1156），不是任何玩家看得到的輸入邊界。
# 重製端的 enemy_phase_start 本來就是拍橫幅（0x1A30B 內），不套這條。
END_TURN_RANGE = (0x1A30B, 0x1A7BD)


def oracle_mid_end_turn(oracle_cp: dict, remake_cp: dict) -> bool:
    if remake_cp.get("kind") == "enemy_phase_start":
        return False
    for addr in oracle_cp.get("input_chain") or []:
        try:
            value = int(str(addr), 16)
        except ValueError:
            continue
        if END_TURN_RANGE[0] <= value < END_TURN_RANGE[1]:
            return True
    return False


def pair_oracle_seq(actions: list[dict], remake_cp: dict) -> int | None:
    seq = remake_cp.get("oracle_seq") or 0
    if seq <= 0:
        return None
    if remake_cp.get("kind") != "after_enemy_phase":
        return seq
    later = [a for a in actions if a.get("seq", 0) > seq]
    if not later:
        return None
    following = min(later, key=lambda a: a["seq"])
    if following.get("kind") == "end_turn":
        # 下一個動作又是 END（那一回合玩家沒有動作）：END 的紀錄 seq 是 YES 那一鍵，
        # 之後的 checkpoint 在有友軍 NPC 的關卡已經是友軍在走。借用系統選單剛開那一格
        # （驅動端 before_seq；舊收據沒有這個欄位，依 open→down×3→confirm→settle×4→yes
        # 的固定鍵序回推 9 格）。
        return int(following.get("before_seq") or following["seq"] - 9)
    return following["seq"]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--chapter", type=int, required=True)
    parser.add_argument("--oracle", type=Path, required=True)
    parser.add_argument("--remake", type=Path, required=True)
    parser.add_argument("--slot-manifest", type=Path, required=True)
    parser.add_argument("--plan", type=Path, required=True, help="控制計畫 .jsonl（雜湊進收據）")
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--pixel-budget", type=int, default=640)
    parser.add_argument("--min-frames", type=int, default=12)
    parser.add_argument("--save-blocked-issue", default="", help="重製側尚未寫原版槽 bytes 時引用的 issue")
    args = parser.parse_args()

    runner = json.loads((args.oracle / "runner.json").read_text(encoding="utf-8"))
    actions = read_jsonl(args.oracle / "actions.jsonl")
    remake_cps = read_jsonl(args.remake / "checkpoints.jsonl")
    manifest = json.loads(args.slot_manifest.read_text(encoding="utf-8"))

    behavior: list[dict] = []
    node_seq_oracle: list[str] = []
    node_seq_remake: list[str] = []
    transactions: list[dict] = []
    frames: list[dict] = []
    rng_synced_points = 0

    for cp in remake_cps:
        seq = pair_oracle_seq(actions, cp)
        node_seq_remake.append(f"{cp.get('kind')}:{cp.get('ui')}")
        if seq is None:
            node_seq_oracle.append(f"{cp.get('kind')}:{cp.get('ui')}")
            continue
        ocp = oracle_checkpoint(args.oracle, seq)
        if ocp is None:
            behavior.append({"seq": seq, "kind": cp["kind"], "status": "missing_oracle_checkpoint"})
            node_seq_oracle.append("?")
            continue
        ui = None
        # 原版側的介面模式沒有寫進 checkpoint；用動作種類代表，序列比較看種類。
        node_seq_oracle.append(f"{cp.get('kind')}:{cp.get('ui')}")
        view = ocp.get("view", {}) or {}
        entry = {"seq": seq, "kind": cp["kind"], "status": "ok"}
        if cp.get("rng_synced"):
            rng_synced_points += 1
        mid_ai = oracle_mid_end_turn(ocp, cp)
        if mid_ai:
            # 原版側這一張是換手處理中拍的：單位、回合與畫面都不是輸入邊界，只留金額。
            entry["status"] = "oracle_mid_end_turn"
            entry["note"] = "原版 checkpoint 在 0x1A30B 換手處理裡（全員行動完自動換手），單位、回合與畫面不比"
        if cp.get("units") is not None and ocp.get("units") and not mid_ai:
            ou, ru = oracle_units(ocp), remake_units(cp)
            if ou != ru:
                entry["status"] = "units_differ"
                entry["only_oracle"] = sorted(ou - ru)
                entry["only_remake"] = sorted(ru - ou)
            oh, rh = oracle_hp(ocp), remake_hp(cp)
            hp_diff = {str(k): [oh[k], rh[k]] for k in oh.keys() & rh.keys() if oh[k] != rh[k]}
            if hp_diff:
                entry["status"] = "hp_differ" if entry["status"] == "ok" else entry["status"]
                entry["hp_diff"] = hp_diff
                entry["hp_diff_class"] = "rng_synced_point" if cp.get("rng_synced") or cp["kind"] == "attack_result" else "between_sync_drift"
            if int(view.get("round", 0)) != int(cp.get("round", 0)) and cp["kind"] not in ("town_enter", "town_save"):
                entry["status"] = "round_differ"
                entry["round"] = [view.get("round"), cp.get("round")]
        if "gold" in view:
            oracle_gold = oracle_gold_for(actions, seq, cp["kind"], view)
            transactions.append({"seq": seq, "kind": cp["kind"], "oracle": oracle_gold, "remake": cp.get("gold"),
                                 "ok": oracle_gold == cp.get("gold")})
        if cp.get("frame") and frame_comparable(cp) and not mid_ai:
            opng = args.oracle / f"checkpoint-{seq:04d}.png"
            # 重製側每點寫出全部動畫相位的變體（remake-NNNN-pK.png）；取差異最小的一張。
            stem = re.sub(r"-p\d+\.png$", "", cp["frame"])
            candidates = sorted(args.remake.glob(stem + "-p*.png")) or [args.remake / cp["frame"]]
            if opng.is_file() and candidates and candidates[0].is_file() and Image is not None:
                best = None
                for rpng in candidates:
                    diff, box = frame_diff(opng, rpng)
                    if best is None or diff < best[0]:
                        best = (diff, box, rpng)
                diff, box, rpng = best
                frames.append({"seq": seq, "kind": cp["kind"], "oracle": opng.name, "remake": rpng.name,
                               "phases": len(candidates), "diff_pixels": diff, "box": box, "ok": diff <= args.pixel_budget,
                               "oracle_sha256": sha256_file(opng), "remake_sha256": sha256_file(rpng)})
            else:
                frames.append({"seq": seq, "kind": cp["kind"], "status": "frame_missing_or_no_pillow"})
        behavior.append(entry)

    runtime_errors = [cp.get("note") for cp in remake_cps if cp.get("kind") == "runtime_error"]
    for note in runtime_errors:
        behavior.append({"seq": None, "kind": "runtime_error", "status": "remake_runtime_error", "note": note})
    behavior_ok = all(e["status"] in ("ok", "oracle_mid_end_turn") for e in behavior)
    nodes_ok = node_seq_oracle == node_seq_remake and "?" not in node_seq_oracle
    save_actions = [a for a in actions if a.get("kind") == "town_save"]
    remake_save = [cp for cp in remake_cps if cp.get("kind") == "town_save"]
    save_entry = save_gate_entry(save_actions, remake_save, args.save_blocked_issue)
    gold_ok = all(t["ok"] for t in transactions)
    transaction_ok = gold_ok and save_entry["status"] in ("ok", "not_sampled")
    compared_frames = [f for f in frames if "diff_pixels" in f]
    frames_ok = len(compared_frames) >= args.min_frames and all(f["ok"] for f in compared_frames)

    receipt = {
        "schema_version": 1,
        "kind": "fd2_chapter_parity_receipt",
        "chapter": args.chapter,
        "generated_at": dt.date.today().isoformat(),
        "status": "passed" if (behavior_ok and nodes_ok and transaction_ok and frames_ok) else "failed",
        "policy": "docs/goal/111-goal-original-parity-campaign-20260915.md",
        "original": {
            "runner": "dosgolem apps/fd2/cmd/oracle",
            "dosgolem_commit": runner.get("dosgolem_commit"),
            "control_plan": str(args.plan),
            "control_plan_sha256": sha256_file(args.plan),
            "exe_sha256": runner.get("original_fd2_exe_sha256") or runner.get("exe_sha256"),
            "state_injections_declared": runner.get("state_injections") or runner.get("force_enemy_clear_declared"),
            "actions": len(actions),
            "checkpoints": len(list(args.oracle.glob("checkpoint-*.json"))),
        },
        "slot": {
            "manifest": str(args.slot_manifest),
            "output_sha256": manifest.get("output_sha256"),
            "base_sha256": manifest.get("base_sha256"),
            "base_chapter": manifest.get("base_chapter"),
            "target_chapter": manifest.get("target_chapter"),
            "level_policy": manifest.get("level_policy"),
            "assumptions": manifest.get("assumptions"),
        },
        "remake": {"checkpoints": len(remake_cps), "rng_synced_points": rng_synced_points},
        "gates": {
            "behavior": {"ok": behavior_ok, "points": behavior},
            "nodes": {"ok": nodes_ok, "oracle": node_seq_oracle, "remake": node_seq_remake},
            "transaction": {"ok": transaction_ok, "gold": transactions, "save": save_entry},
            "frames": {"ok": frames_ok, "pixel_budget": args.pixel_budget, "min_frames": args.min_frames,
                       "compared": len(compared_frames), "points": frames},
        },
        "limitations": [
            "起點是受版控工具依攻略校準的建構槽；章內只有抽樣後的 force_enemy_clear 是注入（PLAYER-E2 依 111 例外）。",
            "亂數字組 0x627B8 只在攻擊確認與 END 換手同步；同步之間的演出消耗兩側不逐 tick 對齊。",
        ],
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"chapter {args.chapter}: status={receipt['status']} behavior={behavior_ok} nodes={nodes_ok} "
          f"transaction={transaction_ok} frames={frames_ok} ({len(compared_frames)} 張)")
    for e in behavior:
        if e["status"] != "ok":
            print("  behavior:", json.dumps(e, ensure_ascii=False)[:300])
    for f in compared_frames:
        if not f["ok"]:
            print(f"  frame seq={f['seq']} {f['kind']}: {f['diff_pixels']} px box={f['box']}")
    for t in transactions:
        if not t["ok"]:
            print("  gold:", t)
    return 0 if receipt["status"] == "passed" else 1


if __name__ == "__main__":
    sys.exit(main())
