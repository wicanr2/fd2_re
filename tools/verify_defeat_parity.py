#!/usr/bin/env python3
"""核對第十二／十三章有限敗北呈現；在 fd2-assets-local:20260829-sfx 容器內執行。

依 0x22EC5／0x22EE6 原始 call-site 對應下一個 0x17AA9 擷取，不能挑圖或放寬像素門檻。
範圍只含兩段提示、固定順序、完整標題返回與存檔邊界，不宣稱自然難度或硬體時鐘一致。
"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image

EXE_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def lines(path):
    return [json.loads(line) for line in Path(path).read_text().splitlines() if line.strip()]


def verify(original, remake, plan, slot_manifest, chapter=12):
    runner = json.loads((original / "runner.json").read_text())
    current = json.loads((original / "current.json").read_text())
    presentation = json.loads((remake / "defeat-presentation.json").read_text())
    trace = lines(original / "eip-trace.jsonl")
    captured = lines(original / "frames/frames.jsonl")
    sequence = ["0x22E5C", "0x22EC5", "0x22EE6", "0x25DFB", "0x1F894"]
    if chapter == 13:
        sequence = ["0x207EC", "0x20815"] + sequence
    cursor = -1
    selected = {}
    for address in sequence:
        candidates = [row for row in trace if row["eip"] == address and row["step"] > cursor]
        if not candidates:
            raise ValueError(f"缺少原版順序錨點 {address}")
        selected[address] = candidates[0]
        cursor = candidates[0]["step"]
    gates = {
        "clean_dosgolem": runner["dosgolem_tracked_dirty_files"] == 0 and runner["dosgolem_untracked_files"] == 0 and runner["dosgolem_commit"] != "unknown",
        "fixed_exe": runner["original_fd2_exe_sha256"] == EXE_SHA256 and current["exe_sha256"] == EXE_SHA256,
        "no_periodic_injection": not runner["lock_ally_hp"] and not runner["force_enemy_clear_declared"] and not current["state_injections"],
        "normal_keyboard": current["input_kind"] == "normal BIOS keys",
        "original_title_menu": "0x1FE60" in current["input_chain"] and "0x25ECD" in current["input_chain"],
        "preserved_edi_zero": selected["0x25DFB"]["edi"] == "0x0" and selected["0x1F894"]["edi"] == "0x0",
        "remake_return": presentation["status"] == "passed" and presentation["result"] == "lose" and presentation["title_selection"] == 0,
        "remake_defeat_record": presentation["protected_record"] == (59 if chapter == 13 else 14) and presentation["protected_raw_byte5"] & 1 != 0,
        "no_skip": not presentation["input_can_skip_defeat"],
        "remake_save_unchanged": presentation["save_unchanged"],
    }
    if chapter == 13:
        gates["original_second_branch_only"] = not any(row["eip"] == "0x207A4" for row in trace)
        gates["original_result_dialogue"] = selected["0x20815"]["stack"][1] == "0x2"
        gates["remake_second_branch_only"] = presentation["native_result_rules"] == ["late_record59_inactive"] and presentation["native_round"] == 6 and bool(presentation["initial_allies_active_records"])
    compared = []
    for ticks, address in [(9, "0x22EC5"), (36, "0x22EE6")]:
        anchor = selected[address]["step"]
        candidates = [row for row in captured if row["step"] > anchor and row["eip"] == "0x17AA9"]
        if not candidates:
            raise ValueError(f"{address} 後缺少原版幀，不能改選其他畫面")
        frame = min(candidates, key=lambda row: row["step"])
        if frame["step"] - anchor > 20:
            raise ValueError(f"{address} 與下一個 wait 入口間距不符：{frame['step']-anchor}")
        rows = [row for row in presentation["frames"] if row["hold_bios_ticks"] == ticks]
        if len(rows) != 1:
            raise ValueError(f"重製 {ticks} ticks 幀數不符")
        own = rows[0]
        original_png = original / "frames" / frame["file"]
        remake_png = remake / own["file"]
        with Image.open(original_png) as oi, Image.open(remake_png) as ri:
            if oi.size != (320, 200) or ri.size != (320, 200):
                raise ValueError("兩側必須為原生320×200，不能縮放或裁切")
            a, b = oi.convert("RGBA"), ri.convert("RGBA")
            difference = sum(x != y for x, y in zip(a.getdata(), b.getdata()))
        same_indices = frame["indexed_sha256"] == own["indexed_sha256"]
        gates[f"frame_{ticks}_exact"] = difference == 0 and same_indices
        compared.append({
            "hold_bios_ticks": ticks, "original_call_site": address,
            "original_frame": frame["file"], "original_step": frame["step"], "original_control_seq": selected[address]["control_seq"],
            "original_png_sha256": sha(original_png), "remake_frame": own["file"], "remake_png_sha256": sha(remake_png),
            "original_indexed_sha256": frame["indexed_sha256"], "remake_indexed_sha256": own["indexed_sha256"],
            "different_pixels": difference, "same_indices": same_indices,
        })
    manifest = json.loads(slot_manifest.read_text())
    slot = slot_manifest.parent / "FD2.SAV"
    gates["same_initial_slot"] = sha(slot) == presentation["slot_sha256"]
    replay = json.loads((remake / "replay-settings.json").read_text())
    source_paths = ["remake/cmd/fd2/main.go", "remake/cmd/fd2/native_defeat.go", "remake/cmd/fd2/native_defeat_parity_test.go", "remake/cmd/fd2/battle_protection.go", "remake/internal/battle/native_ai_runtime.go", "remake/internal/battle/native_ai_physical_score.go", "tools/dosgolem_oracle_drive.py", "tools/verify_defeat_parity.py"]
    evidence = ["docs/data/ida/fd2_pending_code1_return_title_20261001.json", "docs/data/ida/fd2_ai_physical_target_terrain_20261001.json"]
    if chapter == 13:
        source_paths += ["remake/cmd/fd2/native_chapter_result.go", "remake/internal/battle/native_result.go", "remake/assets/scenarios/ch13.json", "remake/assets/scenarios/campaign_full.json"]
        evidence.append("docs/data/ida/fd2_ch13_result_conditions_20261001.json")
    return {
        "schema_version": 1, "kind": "fd2_finite_defeat_parity_receipt", "chapter": chapter,
        "status": "passed" if all(gates.values()) else "failed", "gates": gates,
        "evidence": evidence, "spec": "docs/knowledge-base/56-fd2-remake-sdd.md " + ("第十三章逐條結果與對白 READY" if chapter == 13 else "§5.2.1／5.2.2／5.2.3"),
        "original": {"runner": runner["runner"], "dosgolem_commit": runner["dosgolem_commit"], "tracked_dirty": runner["dosgolem_tracked_dirty_files"],
            "normal_player_path_verified": current["normal_player_path_verified"], "state_injections": current["state_injections"],
            "exe_sha256": EXE_SHA256, "address_space": current["address_space"], "plan": str(plan), "plan_sha256": sha(plan),
            "call_sequence": [{"address": address, "step": selected[address]["step"], "control_seq": selected[address]["control_seq"]} for address in sequence]},
        "slot": {"sha256": sha(slot), "manifest_sha256": sha(slot_manifest), "policy": "114 建構槽；非自然難度／傷害／存活證據", "assumptions": manifest.get("assumptions", [])},
        "random_control": {"original_initial_word": json.loads((original / "checkpoint-0000.json").read_text())["view"]["rng_word"],
            "remake": replay, "method": "同一建構槽／等價玩家動作；每個 AI／攻擊／成長決策點承接受控原版 RNG word，不挑選重擲結果"},
        "frames": compared, "remake": presentation, "source_sha256": {path: sha(path) for path in source_paths},
        "limits": ["僅閉合兩段敗北提示與完整標題返回控制流；完整標題逐幀 E2 未由本收據宣稱", "BIOS tick 採既有規格近似；2ms DAC 每步在60Hz呈現，不宣稱逐週期／同硬體 wall-clock", "建構槽政策114只支持節點／介面／存檔驗證，不支持自然難度、傷害或存活 parity"],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--remake", type=Path, required=True)
    parser.add_argument("--plan", type=Path, required=True)
    parser.add_argument("--slot-manifest", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--chapter", type=int, choices=[12, 13], default=12)
    args = parser.parse_args()
    report = verify(args.original, args.remake, args.plan, args.slot_manifest, args.chapter)
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "gates": report["gates"], "frames": report["frames"]}, ensure_ascii=False))
    return 0 if report["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
