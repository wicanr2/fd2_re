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
   nodes      已抽樣原版動作／介面家族相同；跨時序點另列未比較
  transaction 每點金幣相同；酒店存檔兩側 SHA-256（重製側尚未寫原版槽格式時標 blocked）
  frames     每張有畫面的點：320×200 逐像素 RGB 差異 ≤ pixel-budget；有畫面的點數 ≥ min-frames
"""

from __future__ import annotations

import argparse
from collections import Counter
import datetime as dt
import hashlib
import json
import os
import re
import sys
from pathlib import Path

from dosgolem_oracle_drive import ui_mode as oracle_ui_mode

try:
    from PIL import Image, ImageChops
except ImportError:  # pragma: no cover - 容器外沒有 Pillow
    Image = None

ROOT = Path(__file__).resolve().parents[1]


def palette_cycle_metadata(cp: dict) -> dict:
    """保留已分級的候選來源；中間writer不能標成完整相位。"""
    if "palette_cycle_write" in cp:
        writer = cp["palette_cycle_write"]
        if (isinstance(writer, dict) and writer.get("eip") in ("0x4E01F", "0x4E014")
                and type(writer.get("phase")) is int and 0 <= writer["phase"] < 16
                and type(writer.get("completed_entries")) is int
                and 1 <= writer["completed_entries"] <= 16
                and (writer["eip"] != "0x4E014" or writer["completed_entries"] < 16)
                and writer["phase"] == cp.get("palette_cycle_phase")):
            return {"palette_cycle_phase": writer["phase"], "palette_cycle_write": writer,
                    "palette_cycle_source": "原版EIP／暫存器證實完整triplet寫入進度，16色全部匹配新舊raw窗口；僅測試候選"}
        return {"palette_cycle_source": "未知writer metadata；不得宣稱完整相位"}
    if "palette_cycle_phase" in cp:
        return {"palette_cycle_phase": cp["palette_cycle_phase"],
                "palette_cycle_source": "原版索引PNG的DAC E0..EF完整匹配已證實raw窗口；僅測試候選"}
    return {}


def read_jsonl(path: Path) -> list[dict]:
    out = []
    with path.open(encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if line and not line.startswith("#"):
                out.append(json.loads(line))
    return out


def stage_trace_metadata(root: Path, checkpoint: dict, recorded: object) -> dict:
    """獨立核對測試承接的列偏移；只讀raw trace，不讀影像。"""
    source = "docs/data/ida/fd2_ch24_stage_runtime_20261003.json"
    try:
        if type(recorded) is not int or not 0 <= recorded < 192:
            raise ValueError("重製列偏移超界")
        if checkpoint.get("exe_sha256") != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f":
            raise ValueError("原版EXE錯誤")
        if not 0 <= int(checkpoint["eip"], 16) <= 0xffffffff:
            raise ValueError("原版checkpoint EIP無效")
        # #155: work-buffer rotation cannot invalidate a completed VGA publication.
        window = json.loads((root / "runner.json").read_text())["eip_trace_window"]
        trace = read_jsonl(root / "eip-trace.jsonl")
        if window["from_step"] != 0 or window["to_step"] != 0 or len(trace) >= window["max_entries"]:
            raise ValueError("追蹤不是完整且未截斷")
        # #153: deferred PNG uses the first completed return of the same viewport copy.
        frame_steps = checkpoint["steps"]
        deferred = checkpoint.get("frame_deferred", False)
        if type(deferred) is not bool:
            raise ValueError("延後畫面旗標無效")
        chain = checkpoint.get("input_chain", [])
        if deferred and chain and chain[0] == "0x11ED9":
            if len(chain) < 2:
                raise ValueError("延後畫面的返回鏈缺失")
            if chain[1] != "0x12103":  # stage-to-work copy preserves the published VGA.
                if not 0x373C4 <= int(checkpoint["eip"], 16) < 0x37416:
                    raise ValueError("延後畫面不在已證實memcpy內")
                completed = next((row for row in trace if row["step"] > frame_steps), None)
                if completed is None:
                    raise ValueError("延後畫面缺少copy完成邊界")
                stack = completed["stack"]
                if (completed["eip"] != "0x11EED" or len(stack) < 7 or stack[0] != chain[1] or
                        [stack[i] for i in (1, 2, 4, 5, 6)] != ["0xA0504", "0x140", "0x1C8", "0x138", "0xC0"]):
                    raise ValueError("延後畫面的完整viewport返回未知")
                frame_steps = completed["step"]
        offset, presented, source_buffer, previous, rotations, presents = None, None, None, 0, 0, 0
        for entry in trace:
            if entry["step"] < previous:
                raise ValueError("追蹤順序錯誤")
            previous = entry["step"]
            if entry["step"] > frame_steps:
                break
            if entry["eip"] == "0x11EED" and offset is not None:
                stack = entry["stack"]
                if len(stack) < 7:
                    raise ValueError("copy返回參數缺失")
                full_viewport = [stack[i] for i in (1, 2, 4, 5, 6)] == ["0xA0504", "0x140", "0x1C8", "0x138", "0xC0"]
                if not full_viewport:
                    if stack[0] == "0x11D3B":
                        raise ValueError("VGA copy參數未知")
                    continue
                if source_buffer is None and stack[0] != "0x11D3B":
                    raise ValueError("VGA work來源缺失")
                pointer = int(stack[3], 0)
                if not 0 < pointer <= 0xffffffff:
                    raise ValueError("VGA source無效")
                # #150: only a completed native viewport copy can rebind the source.
                if stack[0] != "0x11D3B" and source_buffer != stack[3]:
                    presented = None
                    continue
                source_buffer, presented = stack[3], offset
                presents += 1
                continue
            if entry["eip"] != "0x24D48":
                continue
            latch = int(entry["eax"], 16)
            stack = entry["stack"]
            if not 1 <= latch <= 14 or stack[3] != "0x0":
                raise ValueError("旋轉參數未知")
            if stack[2] == "0x10842":
                offset = 0
                presented, source_buffer = None, None
            elif stack[2] != "0x120B6":
                raise ValueError("旋轉caller未知")
            if offset is None:
                raise ValueError("缺loader")
            offset = (offset + latch) % 192
            rotations += 1
        if presented is None:
            raise ValueError("缺完整VGA發布")
        return {"stage_row_offset": recorded, "original_stage_row_offset": presented,
                "original_stage_work_offset": offset, "stage_trace_presents": presents,
                "stage_trace_rotations": rotations, "stage_phase_match": presented == recorded,
                "stage_state_steps": checkpoint["steps"], "stage_frame_steps": frame_steps,
                "stage_phase_source": source}
    except (OSError, ValueError, KeyError, IndexError, TypeError) as exc:
        return {"stage_phase_match": False, "stage_phase_error": str(exc), "stage_phase_source": source}


def plan_completion(plan: list[dict], actions: list[dict], remake_cps: list[dict] | None = None) -> dict:
    """截短或仍在執行的原版收據不得冒充完整章工作單元。"""
    required = Counter()
    for step in plan:
        copies = max(0, int(step.get("repeat", 1)))
        if step.get("mark"):
            required["mark:" + step["mark"]] += copies
        for kind in ("force_enemy_clear", "shop_sell", "shop_buy", "town_save", "preparation_save", "secret_shop"):
            if step.get(kind):
                required[kind] += copies
    observed = Counter(
        "mark:" + a.get("label", "") if a.get("kind") == "mark" else a.get("kind", "")
        for a in actions
    )
    missing = dict(required - observed)
    remake_missing = {}
    if remake_cps is not None:
        observed_remake = Counter(
            "mark:" + cp.get("kind", "") if "mark:" + cp.get("kind", "") in required
            else cp.get("kind", "") for cp in remake_cps
        )
        remake_missing = dict(required - observed_remake)
    return {"ok": not missing and not remake_missing, "required": dict(required),
            "missing": missing, "remake_missing": remake_missing}


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
    # 任一 RGB 通道非零就算一個不同像素；以 Pillow 執行整張比較，
    # 保留逐像素等值判定與含端點的 bbox，不改門檻或排除任何區域。
    red, green, blue = ImageChops.difference(a, b).split()
    different = ImageChops.lighter(ImageChops.lighter(red, green), blue)
    diff = 320 * 200 - different.histogram()[0]
    box = different.getbbox()
    return diff, [box[0], box[1], box[2] - 1, box[3] - 1] if box else []


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


def preparation_save_proof(action: dict, run: Path) -> bool:
    """#125：語意動作不能取代原版writer收據；核對同一保存前後的DOS成功bytes。"""
    before_seq, seq = action.get("before_seq"), action.get("seq")
    if not isinstance(before_seq, int) or not isinstance(seq, int) or before_seq >= seq:
        return False
    before, after = oracle_checkpoint(run, before_seq), oracle_checkpoint(run, seq)
    expected = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
    if (not before or not after or before.get("exe_sha256") != expected or
            after.get("exe_sha256") != expected or oracle_ui_mode(before) != "record_question" or
            oracle_ui_mode(after) != "record_slots"):
        return False

    def successful_bytes(cp):
        return sum(int(call.get("written_bytes", 0)) for call in cp.get("dos_file_calls", [])
                   if call.get("op") == "write" and call.get("handled") is True and
                   not call.get("carry") and str(call.get("path", "")).replace("\\", "/").rsplit("/", 1)[-1].upper() == "FD2.SAV")

    return successful_bytes(after) - successful_bytes(before) == 22987


def frame_comparable(remake_cp: dict) -> bool:
    """after_enemy_phase 只比行為：它借用下一個動作的原版檢查點，而那張圖是動作做完
    之後（選取後的移動範圍、END 後的換手橫幅）拍的，原版沒有同一瞬間的閒置畫面。"""
    return remake_cp.get("kind") != "after_enemy_phase"


def replay_divergences(checkpoints: list[dict]) -> list[dict]:
    """重播拒絕原版接受的動作時，後續狀態碰巧相同也不得放行。"""
    return [{"seq": cp.get("oracle_seq"), "kind": cp.get("kind"),
             "status": "remake_divergence", "note": cp["note"]}
            for cp in checkpoints if str(cp.get("note", "")).startswith("divergence:")]


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
    if following.get("kind") == "force_enemy_clear":
        # 清敵收據已將敵軍 HP 寫成 0；敵方回合結束的狀態必須比注入前一格。
        # 例如第十四章 END 1022 → round4 閒置 1073 → 清敵 1074。
        # 清敵本身仍另比 following.seq，不可用清敵後狀態遮蔽回合差異。
        return following["seq"] - 1
    if following.get("kind") == "end_turn":
        # 下一個動作又是 END（那一回合玩家沒有動作）：END 的紀錄 seq 是 YES 那一鍵，
        # 之後的 checkpoint 在有友軍 NPC 的關卡已經是友軍在走。借用系統選單剛開那一格
        # （驅動端 before_seq；舊收據沒有這個欄位，依 open→down×3→confirm→settle×4→yes
        # 的固定鍵序回推 9 格）。
        return int(following.get("before_seq") or following["seq"] - 9)
    return following["seq"]


def comparable_action_kind(action: dict, remake_kind: str | None) -> str | None:
    """只沿用已審查的名稱對照；不以 UI 或原版動作順序推定來源。"""
    kind = action.get("label") if action.get("kind") == "mark" else action.get("kind")
    if (isinstance(remake_kind, str) and remake_kind.startswith("mark:") and
            action.get("kind") != "mark"):
        return None
    if (action.get("kind") == "mark" and isinstance(kind, str) and kind and
            remake_kind == "mark:" + kind):
        return remake_kind
    return "attack_armed" if kind == "attack" else kind


def compare_node(actions: list[dict], remake_cp: dict, oracle_cp: dict | None,
                 seq: int | None) -> dict:
    """#117：從原版動作及輸入owner獨立取值；契約見fd2-chapter-node-comparison-contract.json。"""
    kind = remake_cp.get("kind")
    entry = {"seq": seq, "kind": kind, "remake_ui": remake_cp.get("ui")}
    if seq is None:
        return {**entry, "status": "not_comparable", "reason": "衍生重製點沒有原版配對"}
    if oracle_cp is None:
        return {**entry, "status": "missing_oracle_checkpoint"}
    candidates = [a for a in actions if a.get("seq") == seq]
    original_ui = oracle_ui_mode(oracle_cp)
    entry["oracle_ui"] = original_ui
    if kind in ("after_enemy_phase", "shop_menu", "end_turn", "enemy_phase_start", "town_save"):
        return {**entry, "status": "not_comparable", "reason": {
            "after_enemy_phase": "借用下一動作；原版沒有同瞬間閒置畫面",
            "shop_menu": "出售前介面借用出售後的原版checkpoint",
            "end_turn": "END按鍵後有界前進；可能已進換手或戰後對白",
            "enemy_phase_start": "重製換手橫幅與原版END按鍵後checkpoint不同時序",
            "town_save": "SAVE動作在DOS寫入完成點；服務框／確認訊息時序不同",
        }[kind]}
    if not candidates:
        return {**entry, "status": "missing_oracle_action"}
    matches = [a for a in candidates if comparable_action_kind(a, kind) == kind]
    entry["oracle_action_candidates"] = len(candidates)
    entry["oracle_matching_actions"] = len(matches)
    if len(matches) > 1:
        return {**entry, "status": "ambiguous_oracle_action"}
    if not matches and len(candidates) > 1:
        return {**entry, "status": "missing_matching_oracle_action"}
    action = matches[0] if matches else candidates[0]
    original_kind = action.get("label") if action.get("kind") == "mark" else action.get("kind")
    entry["oracle_action_kind"] = action.get("kind")
    if action.get("kind") == "mark":
        entry["oracle_action_label"] = action.get("label")
    entry["oracle_kind"] = original_kind
    comparable_kind = comparable_action_kind(action, kind)
    entry["oracle_comparable_kind"] = comparable_kind
    if oracle_mid_end_turn(oracle_cp, remake_cp):
        return {**entry, "status": "not_comparable", "reason": "原版已在換手處理，並非同一輸入邊界"}
    if original_ui == "unknown":
        return {**entry, "status": "unknown_oracle_ui"}
    # 原版2D7D1為共用服務對話owner；此處不宣稱具體建築一致。
    family = lambda ui: "service" if ui in ("shop", "hotel", "church") else ui
    entry["oracle_family"] = family(original_ui)
    entry["remake_family"] = family(remake_cp.get("ui"))
    entry["status"] = "ok" if (comparable_kind == kind and
                               entry["oracle_family"] == entry["remake_family"]) else "node_differ"
    return entry


def node_gate_ok(points: list[dict], completion: dict, checkpoints: list[dict]) -> bool:
    return (completion["ok"] and any(p["status"] == "ok" for p in points) and
            all(p["status"] in ("ok", "not_comparable") for p in points) and
            not replay_divergences(checkpoints) and
            not any(cp.get("kind") == "runtime_error" for cp in checkpoints))


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
    completion = plan_completion(read_jsonl(args.plan), actions, remake_cps)
    manifest = json.loads(args.slot_manifest.read_text(encoding="utf-8"))
    settings_path = args.remake / "replay-settings.json"
    replay_settings = json.loads(settings_path.read_text(encoding="utf-8")) if settings_path.is_file() else None
    boot_checkpoint = oracle_checkpoint(args.oracle, 0) or {}

    behavior: list[dict] = []
    node_seq_oracle: list[str] = []
    node_seq_remake: list[str] = []
    node_points: list[dict] = []
    transactions: list[dict] = []
    frames: list[dict] = []
    rng_synced_points = 0

    for cp in remake_cps:
        seq = pair_oracle_seq(actions, cp)
        ocp = oracle_checkpoint(args.oracle, seq) if seq is not None else None
        node = compare_node(actions, cp, ocp, seq)
        node_points.append(node)
        if node["status"] != "not_comparable":
            node_seq_oracle.append(f"{node.get('oracle_comparable_kind')}:{node.get('oracle_family')}")
            node_seq_remake.append(f"{cp.get('kind')}:{node.get('remake_family')}")
        if seq is None:
            continue
        if ocp is None:
            behavior.append({"seq": seq, "kind": cp["kind"], "status": "missing_oracle_checkpoint"})
            continue
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
                frames[-1].update(palette_cycle_metadata(cp))
                if "stage_row_offset" in cp:
                    frames[-1].update(stage_trace_metadata(args.oracle, ocp, cp["stage_row_offset"]))
                    frames[-1]["ok"] &= frames[-1]["stage_phase_match"]
            else:
                frames.append({"seq": seq, "kind": cp["kind"], "status": "frame_missing_or_no_pillow"})
        behavior.append(entry)

    runtime_errors = [cp.get("note") for cp in remake_cps if cp.get("kind") == "runtime_error"]
    for note in runtime_errors:
        behavior.append({"seq": None, "kind": "runtime_error", "status": "remake_runtime_error", "note": note})
    behavior.extend(replay_divergences(remake_cps))
    behavior_ok = all(e["status"] in ("ok", "oracle_mid_end_turn") for e in behavior)
    nodes_ok = node_gate_ok(node_points, completion, remake_cps)
    save_actions = [a for a in actions if a.get("kind") in ("town_save", "preparation_save")]
    remake_save = [cp for cp in remake_cps if cp.get("kind") in ("town_save", "preparation_save")]
    save_entry = save_gate_entry(save_actions, remake_save, args.save_blocked_issue)
    if any(not preparation_save_proof(a, args.oracle) for a in save_actions
           if a.get("kind") == "preparation_save"):
        save_entry["status"] = "original_save_write_missing"
    if any(completion["missing"].get(k) or completion["remake_missing"].get(k)
           for k in ("town_save", "preparation_save")):
        save_entry["status"] = "planned_save_missing"
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
            "random_control": {
                "initial_checkpoint_rng_word": boot_checkpoint.get("view", {}).get("rng_word"),
                "setup": "dosgolem 決定性全新程序與虛擬時鐘；決策點 RNG word 見動作及 EIP trace",
            },
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
        "remake": {"checkpoints": len(remake_cps), "rng_synced_points": rng_synced_points,
                   "random_control": replay_settings},
        "gates": {
            "behavior": {"ok": behavior_ok, "points": behavior},
            "nodes": {"ok": nodes_ok, "oracle": node_seq_oracle, "remake": node_seq_remake,
                       "plan_completion": completion, "comparison_version": 3,
                       "source": "dosgolem_oracle_drive.ui_mode(input_chain)及原版actions",
                       "contract": "docs/data/fd2-chapter-node-comparison-contract.json",
                       "points": node_points},
            "transaction": {"ok": transaction_ok, "gold": transactions, "save": save_entry},
            "frames": {"ok": frames_ok, "pixel_budget": args.pixel_budget, "min_frames": args.min_frames,
                       "compared": len(compared_frames), "points": frames},
        },
        "limitations": [
            "起點是受版控工具依攻略校準的建構槽；章內只有抽樣後的 force_enemy_clear 是注入（PLAYER-E2 依 111 例外）。",
            "亂數字組 0x627B8 只在攻擊確認與 END 換手同步；同步之間的演出消耗兩側不逐 tick 對齊。",
            "nodes只驗已抽樣原版動作與介面家族；not_comparable點未驗介面。舊comparison_version缺失的nodes=true沒有獨立原版UI判準。",
        ],
    }
    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(f"chapter {args.chapter}: status={receipt['status']} behavior={behavior_ok} nodes={nodes_ok} "
          f"transaction={transaction_ok} frames={frames_ok} ({len(compared_frames)} 張)")
    for e in behavior:
        if e["status"] != "ok":
            print("  behavior:", json.dumps(e, ensure_ascii=False)[:300])
    for node in node_points:
        if node["status"] not in ("ok", "not_comparable"):
            print("  node:", json.dumps(node, ensure_ascii=False))
    for f in compared_frames:
        if not f["ok"]:
            print(f"  frame seq={f['seq']} {f['kind']}: {f['diff_pixels']} px box={f['box']}")
    for t in transactions:
        if not t["ok"]:
            print("  gold:", t)
    return 0 if receipt["status"] == "passed" else 1


if __name__ == "__main__":
    sys.exit(main())
