#!/usr/bin/env python3
"""驗證 #23 的有界 dosgolem writer→copy-back→建槽尾格，不外推整筆存檔或 E2。"""
import argparse
import hashlib
import json
from pathlib import Path

import fd2save

ROOT = Path(__file__).resolve().parents[1]
EXE_SHA = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
MAP_SOURCE = "maps/map1/map1_units.json"
PLAN = "docs/data/parity-plans/ch02-join-copyback.jsonl"


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def load(path):
    return json.loads(Path(path).read_text(encoding="utf-8"))


def jsonl(path):
    return [json.loads(line) for line in Path(path).read_text().splitlines() if line.strip()]


def slot_record(path, identity):
    plain = fd2save.decode(Path(path).read_bytes())
    start, _ = fd2save.slot_bounds(0)
    chapter, count = plain[start + fd2save.ROSTER_SIZE:start + fd2save.ROSTER_SIZE + 2]
    if count > fd2save.ROSTER_UNITS:
        raise ValueError("原生槽名冊數越界")
    records = [plain[start + i * fd2save.UNIT_SIZE:start + (i + 1) * fd2save.UNIT_SIZE] for i in range(count)]
    matches = [(i, row) for i, row in enumerate(records) if row[8] == identity]
    if len(matches) != 1:
        raise ValueError(f"identity{identity} 的原生記錄不是唯一一筆")
    return chapter, count, matches[0]


def writer_copy_witness(trace, expected):
    """以同一 runtime 指標串接兩個 writer 和完整 copy；不靠輸出 ff 猜來源。"""
    for copy in trace:
        if copy.get("eip", "").lower() != "0x11576" or int(copy["eax"], 16) != 8 or int(copy["edx"], 16) != 8:
            continue
        base = int(copy["edi"], 16)
        if int(copy["stack"][2], 16) != 0x50 or int(copy["stack"][1], 16) != base or int(copy["stack"][0], 16) != int(copy["ebx"], 16):
            continue
        writes = []
        for index, item in zip((4, 5), expected):
            candidates = [row for row in trace if row.get("eip", "").lower() == "0x10f48"
                          and int(row["edx"], 16) == base and int(row["eax"], 16) == index
                          and int(row["edi"], 16) + 0x0f == base + 0x0f + index * 2
                          and int(row["ebx"], 16) & 255 == item and row["step"] < copy["step"]]
            if len(candidates) != 1:
                break
            writes.append(candidates[0])
        joins = [row for row in trace if row.get("eip", "").lower() == "0x230d9"
                 and int(row["stack"][0], 16) == 8 and row["step"] < copy["step"]]
        syncs = [row for row in trace if row.get("eip", "").lower() == "0x230e1"
                 and row["step"] < copy["step"]]
        if len(writes) == 2 and len(joins) == 1 and len(syncs) == 1 and joins[0]["step"] < syncs[0]["step"]:
            return {"address_space": "dosgolem protected-mode linear memory",
                    "runtime_address": hex(base), "persistent_address": copy["ebx"],
                    "writer_steps": [row["step"] for row in writes], "join_step": joins[0]["step"],
                    "sync_step": syncs[0]["step"], "copy_step": copy["step"],
                    "copy_rng_word": copy.get("rng_word")}
    return None


def verify(original, state, builder, base):
    original, state, builder, base = map(Path, (original, state, builder, base))
    runner, manifest = load(original / "runner.json"), load(builder / "manifest.json")
    actions, trace = jsonl(original / "actions.jsonl"), jsonl(original / "eip-trace.jsonl")
    initial, final = load(original / "checkpoint-0000.json"), load(original / "current.json")
    field = load(ROOT / "remake/assets" / MAP_SOURCE)["units"][22]
    expected = field["inventory_slots"][6:8]
    if len(expected) != 2 or field["native_record_byte8"] != 8 or field["group"] != 4:
        raise ValueError("map1 第22筆的已驗證來源漂移")
    original_save, builder_save = state / "FD2.SAV", builder / "FD2.SAV"
    oc, on, (oi, orec) = slot_record(original_save, 8)
    bc, bn, (bi, brec) = slot_record(builder_save, 8)
    witness = writer_copy_witness(trace, expected)
    marks = {a.get("label"): a["seq"] for a in actions if a.get("kind") == "mark"}
    cleared = [a for a in actions if a.get("kind") == "force_enemy_clear" and a.get("modified_path") and a.get("cleared", 0) > 0]
    saved = [a for a in actions if a.get("kind") == "town_save" and a.get("slot") == 0]
    applied = manifest.get("applied", [])
    joins = [i for i, a in enumerate(applied) if a.get("op") == "join" and a.get("char_id") == 8 and a.get("result") == "appended"]
    projected = [i for i, a in enumerate(applied) if a.get("op") == "sync_join_item_tail" and a.get("char_id") == 8
                 and a.get("source", {}).get("addr", "").lower() == "0x230e1"]
    tail = bytes(value for item in expected for value in (0x80 if item == 255 else 0, item))
    gates = {
        "original_runner": runner.get("runner") == "dosgolem apps/fd2/cmd/oracle"
            and runner.get("dosgolem_tracked_dirty_files") == 0 and bool(runner.get("dosgolem_commit"))
            and runner.get("original_fd2_exe_sha256") == EXE_SHA and final.get("exe_sha256") == EXE_SHA
            and runner.get("control_plan", "").endswith(PLAN),
        "declared_modified_path": runner.get("force_enemy_clear_declared") is True and not runner.get("lock_ally_hp")
            and bool(cleared) and any("force-enemy-clear" in row for row in final.get("state_injections", [])),
        "planned_load_battle_post_save": all(label in marks for label in ("town_loaded", "battle_start", "town_after_battle"))
            and marks.get("town_loaded", 0) < marks.get("battle_start", 0) < marks.get("town_after_battle", 0)
            and bool(saved) and saved[-1].get("save_sha256") == sha(original_save),
        "writer_join_copy_witness": witness is not None,
        "original_save_tail": oc == 2 and orec[0x16:0x1a] == tail,
        "builder_save_tail": bc == oc and bn == on and bi == oi and brec[0x16:0x1a] == tail,
        "builder_join_before_sync": len(joins) == len(projected) == 1 and joins[0] < projected[0],
        "builder_input_and_output_hashes": manifest.get("base_sha256") == sha(base)
            and manifest.get("output_sha256") == sha(builder_save)
            and manifest.get("evidence_sources", {}).get(MAP_SOURCE) == sha(ROOT / "remake/assets" / MAP_SOURCE),
        "builder_no_growth_draws": manifest.get("seed") == 2 and "level_steps" in manifest
            and manifest["level_steps"] in (None, []),
    }
    return {"schema_version": 1, "kind": "fd2_join_copyback_verification", "issue": 23,
            "status": "passed" if all(gates.values()) else "failed", "gates": gates,
            "original": {"runner": runner["runner"], "dosgolem_commit": runner["dosgolem_commit"],
                         "exe_sha256": EXE_SHA, "plan": PLAN, "plan_sha256": sha(ROOT / PLAN),
                         "initial_rng_word": initial.get("view", {}).get("rng_word"),
                         "save_sha256": sha(original_save), "copy_witness": witness},
            "builder": {"base_sha256": sha(base), "output_sha256": sha(builder_save),
                        "seed": manifest.get("seed"), "level_steps": len(manifest.get("level_steps") or []),
                        "record_index": bi, "tail_hex": brec[0x16:0x1a].hex()},
            "source": {"path": MAP_SOURCE, "sha256": sha(ROOT / "remake/assets" / MAP_SOURCE),
                       "record_index": 22, "raw_byte8": 8, "group": 4, "item_bytes": expected},
            "limitations": ["明示清場修改路徑；只驗證第二章戰後物品尾格，沒有新增 PLAYER-E2。",
                            "工具只投影四個 byte，不能由此宣稱完整 0x50-byte 記錄或整檔存檔一致。",
                            "未比較亂數規則；原版使用固定 dosgolem 虛擬開機條件，建槽 seed2 未執行成長擲骰。"]}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ("original", "state", "builder", "base", "output"):
        p.add_argument("--" + name, required=True, type=Path)
    args = p.parse_args()
    result = verify(args.original, args.state, args.builder, args.base)
    args.output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(result["status"], result["gates"])
    return 0 if result["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
