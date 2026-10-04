#!/usr/bin/env python3
"""核對 #32 固定槽酒店 LOAD 的完整影格與返回邊界。

在 fd2-assets-local:20260829-sfx 內執行。原版固定取計畫 r2 的 35、40、44，
重製端只列舉正式 composer 的合法頭像與游標相位，不同步硬體時間。
"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image

EXE_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
SLOT_SHA256 = "ee9e412c0df245672e1644689f922cecb01e1b3abfd97a3d3334081921a963e5"


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def read(path):
    return json.loads(Path(path).read_text())


def verify(original, prefix, slot, plan):
    runner = read(original / "runner.json")
    runtime = read(str(prefix) + ".json")
    final = read(original / "current.json")
    trace = [json.loads(s) for s in (original / "eip-trace.jsonl").read_text().splitlines() if s]
    gates = {
        "canonical_runner": runner["runner"] == "dosgolem apps/fd2/cmd/oracle",
        "clean_oracle": runner["dosgolem_tracked_dirty_files"] == 0 and runner["dosgolem_untracked_files"] == 0,
        "fixed_exe": runner["original_fd2_exe_sha256"] == EXE_SHA256 and final["exe_sha256"] == EXE_SHA256,
        "fixed_slot": sha(slot) == SLOT_SHA256 == runtime["slot_sha256"],
        "ordinary_input": final["input_kind"] == "normal BIOS keys" and not final["state_injections"],
        "no_periodic_injection": not runner["lock_ally_hp"] and not runner["force_enemy_clear_declared"],
        "authored_runtime": runtime["status"] == "passed" and runtime["campaign"] == "canonical",
        "save_unchanged": runtime["save_unchanged"],
        "hotel_then_town": runtime["nodes"] == ["town_ch04", "hotel_ch04", "hotel_ch04", "hotel_ch04", "town_ch04"],
        "selector_two": runtime["selected_service"] == 2,
        "original_return": final["control_seq"] == 49 and "0x2CE08" in final["input_chain"],
    }
    # 原版成功讀取發生在 seq36，ACK 發生在 seq41；避免把重開列表誤當返回。
    gates["original_load_order"] = any(x["eip"] == "0x302BF" and x["control_seq"] == 36 for x in trace) and any(x["eip"] == "0x303C9" and x["control_seq"] == 36 for x in trace)
    gates["original_selector_two"] = any(x["eip"] == "0x2FD33" and x["control_seq"] == 41 and x["esi"] == "0x2" for x in trace)
    frames = []
    for name, seq in [("slots", 35), ("loaded", 40), ("menu", 44)]:
        source = original / f"checkpoint-{seq:04d}.png"
        state = read(original / f"checkpoint-{seq:04d}.json")
        with Image.open(source) as im:
            if im.mode != "P" or im.size != (320, 200):
                raise ValueError("原版必須為完整320×200索引影格")
            indices, rgb = list(im.getdata()), list(im.convert("RGB").getdata())
        variants = []
        for portrait, pulse in [(p, c) for p in range(4) for c in range(4)]:
            path = prefix.parent / (prefix.name + f"-{name}-p{portrait}-c{pulse}.png")
            with Image.open(path) as im:
                if im.mode != "P" or im.size != (320, 200):
                    raise ValueError("重製必須為完整320×200索引影格")
                index_diff = sum(a != b for a, b in zip(indices, im.getdata()))
                rgb_diff = sum(a != b for a, b in zip(rgb, im.convert("RGB").getdata()))
            variants.append({"file": str(path), "sha256": sha(path), "different_indices": index_diff, "different_rgb_pixels": rgb_diff})
        matched = [x["file"] for x in variants if x["different_indices"] == 0 and x["different_rgb_pixels"] == 0]
        gates[name + "_phase_equivalence"] = bool(matched)
        frames.append({"state": name, "original_seq": seq, "original_file": str(source), "original_sha256": sha(source), "original_state_sha256": sha(original / f"checkpoint-{seq:04d}.json"), "original_rng_word": state["view"]["rng_word"], "exact_legal_phases": matched, "all_legal_variants": variants})
    return {
        "schema_version": 1, "kind": "fd2_hotel_load_phase_equivalence", "status": "passed" if all(gates.values()) else "failed",
        "gates": gates, "frames": frames, "runtime": runtime,
        "original_runner": runner, "plan": str(plan), "plan_sha256": sha(plan),
        "trace_sha256": sha(original / "eip-trace.jsonl"), "runtime_receipt_sha256": sha(str(prefix) + ".json"),
        "limits": ["原版沿111建構槽，重製城鎮酒店入口以authored opt0推進；不增加章PLAYER-E2。", "三個固定停點的完整索引／RGB合法相位等價，不證明兩側頭像、游標或硬體時間同步。", "本探針未比較原版傳聞角色選單、內容或gate1提示畫面；gate1拒收依直接bytes及E1交易測試。"],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--remake-prefix", type=Path, required=True)
    parser.add_argument("--slot", type=Path, required=True)
    parser.add_argument("--plan", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    receipt = verify(args.original, args.remake_prefix, args.slot, args.plan)
    args.output.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": receipt["status"], "gates": receipt["gates"]}, ensure_ascii=False))
    raise SystemExit(0 if receipt["status"] == "passed" else 1)


if __name__ == "__main__":
    main()
