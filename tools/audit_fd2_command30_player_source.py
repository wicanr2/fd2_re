#!/usr/bin/env python3
"""機械重核蜜蒂的原始學招鏈；不重開函式反組譯，不冒稱正常玩家動態收據。"""
import argparse
import hashlib
import json
from pathlib import Path

from unpack_dat import parse_directory

ROOT = Path(__file__).resolve().parents[1]


def sha(p):
    return hashlib.sha256(p.read_bytes()).hexdigest()


def reference(p):
    return {"file": str(p.relative_to(ROOT)), "sha256": sha(p)}


def audit(original):
    pinned = {r["file"]: r for r in json.loads((ROOT / "docs/data/fd2-reference-files.json").read_text())["files"]}
    inputs = []
    for name in ("FD2.EXE", "FDFIELD.DAT", "FIGANI.DAT", "FDOTHER.DAT"):
        p = original / name
        assert p.stat().st_size == pinned[name]["size"] and sha(p) == pinned[name]["sha256"], name
        assert hashlib.md5(p.read_bytes()).hexdigest() == pinned[name]["md5"], name
        inputs.append(pinned[name])
    exe = (original / "FD2.EXE").read_bytes()
    table_paths = ["docs/data/exe_tables/character_defaults.json", "remake/assets/data/class_change_growth.json",
                   "remake/assets/data/command_learn.json", "remake/assets/spells.json"]
    default, growth, learn, commands = [json.loads((ROOT / p).read_text()) for p in table_paths]
    rows = []
    for kind, row in [("constructor", default[18]), ("growth", growth[18]), ("learn", learn[5]), ("command", commands[30])]:
        offset, raw = int(row["off"], 16), bytes.fromhex(row["raw"])
        assert exe[offset:offset+len(raw)] == raw, kind
        rows.append({"kind": kind, "address_space": "FD2.EXE file offset", "offset": row["off"], "raw_hex": raw.hex()})
    assert growth[18]["learn_idx"] == 5
    assert learn[5]["entries"] == [{"required_level": 2, "command_id": 30}]
    assert commands[30]["mp"] == 24 and commands[30]["dist"] == 0x14
    assert commands[30]["range"] == 0 and commands[30]["target"] == 0
    # This corrects a claim about the entire learn table, without promoting
    # an unused/unknown row to a normal-player route by elimination.
    grants = {n: [{"row": r["idx"], **e} for r in learn for e in r["entries"] if e["command_id"] == n]
              for n in (28, 30, 31)}
    assert not grants[28] and not grants[31] and len(grants[30]) == 1
    field = (original / "FDFIELD.DAT").read_bytes()
    directory = parse_directory(field)
    owners = []
    for m in range(33):
        resource = m*3+1
        offset, size = directory[resource]
        control = field[offset:offset+size]
        roster_path = ROOT / f"remake/assets/maps/map{m}/map{m}_units.json"
        roster = json.loads(roster_path.read_text())["units"]
        assert len(roster) == control[2], m
        start = 3+16*3+16*2+16*3
        for slot, u in enumerate(roster):
            raw = control[start+slot*26:start+(slot+1)*26]
            assert len(raw) == 26 and raw[1] == u["battle_fig"] and raw[0] == u["native_record_byte6"]
            if raw[1] != 18:
                continue
            assert raw[4] == u["lv"] and list(raw[13:17]) == u["initial_command_mask"]
            owners.append({"map": m, "slot": slot, "camp_raw": raw[0], "level": raw[4],
                           "initial_command_mask": list(raw[13:17]), "group": raw[21], "mode_raw": raw[17],
                           "source_resource": resource, "source_resource_sha256": hashlib.sha256(control).hexdigest(),
                           "address_space": "FDFIELD.DAT file offset", "record_offset": hex(offset+start+slot*26),
                           "raw_hex": raw.hex(), "roster": reference(roster_path)})
    assert any(r["map"] == 16 and r["slot"] == 36 and r["camp_raw"] == 2 and r["level"] == 1 for r in owners)
    inventory_path = ROOT / "docs/data/fd2-battle-hd-source-inventory-20261019.json"
    inventory = json.loads(inventory_path.read_text())
    effect = inventory["resources"][56]
    assert effect["resource"] == 56 and effect["native_header"] == {"byte_1": 1, "byte_2": 9, "byte_4": 2}
    assert len(effect["frames"]) == 14
    events = [{"frame": i, "impact": f["raw_byte_4"], "sample": f["raw_byte_5"], "delay": f["delay_native"]}
              for i, f in enumerate(effect["frames"]) if f["raw_byte_4"] or f["raw_byte_5"]]
    catalog_path = ROOT / "remake/assets/themes/modern/catalog.json"
    catalog = json.loads(catalog_path.read_text())
    hd = [a for a in catalog["assets"] if a["role"] == "battle_animation_hd_set" and a["source_resource"] == 56]
    assert len(hd) == 1 and hd[0]["frame_count"] == 14 and hd[0]["source_metadata_sha256"] == effect["metadata_sha256"]
    other = (original / "FDOTHER.DAT").read_bytes()
    sound_offset, sound_size = parse_directory(other)[49]
    sound = other[sound_offset:sound_offset+sound_size]
    samples = parse_directory(sound)
    assert len(samples) == 8 and samples[7] == (len(sound), 0)
    cues = []
    for event in events:
        offset, size = samples[event["sample"]]
        assert size > 0
        cues.append({"frame": event["frame"], "subresource": event["sample"],
                     "address_space": "FDOTHER.DAT file offset", "offset": hex(sound_offset+offset),
                     "byte_count": size, "pcm_sha256": hashlib.sha256(sound[offset:offset+size]).hexdigest()})
    return {"schema_version": 1, "kind": "fd2_command30_player_source_recheck", "status": "PASS-FIXED-SOURCE-RECHECK",
            "source_inputs": inputs, "tool_sha256": sha(Path(__file__)), "tables": [reference(ROOT/p) for p in table_paths],
            "raw_table_rows": rows, "learn_grants": grants, "selector18_original_roster_rows": owners,
            "figani_source_inventory": reference(inventory_path), "effect_resource": 56, "effect_frames": 14,
            "effect_metadata_sha256": effect["metadata_sha256"], "effect_events": events,
            "sound_resource": 49, "sound_resource_sha256": hashlib.sha256(sound).hexdigest(), "sound_cues": cues,
            "catalog": reference(catalog_path), "hd_asset": hd[0]["asset_id"],
            "closed_writer_evidence": "docs/data/ida/fd2_command28_31_reachability_ida.txt: sub_1E292 -> sub_4E4D1 -> sub_4E4A2 -> sub_1D79C; IDA Pro 9.4 linear addresses retained",
            "reopen_reason": "Fixed original bytes contradict the 2026-08-25 claim that no learn row grants ID30. Selector18 uses learn5, level2 grants30, and original map16 slot36 is player camp2 selector18 level1.",
            "evidence_level": {"raw_tables_and_rosters": "confirmed", "normal_level_up_producer": "strong inference from original data and closed writer; remake runtime probe recorded separately"},
            "original_runtime_executed": False, "normal_campaign_verified": False,
            "scope": "Known data offsets and raw resource extraction only; no new function disassembly, original normal-player claim or guessed command28/31 owner."}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--original-dir", required=True, type=Path)
    p.add_argument("--report", required=True, type=Path)
    a = p.parse_args()
    if not Path("/.dockerenv").exists() or not a.report.parent.is_dir() or a.report.exists():
        raise ValueError("Docker、既有父目錄、全新收據路徑為必要條件")
    result = audit(a.original_dir)
    a.report.write_text(json.dumps(result, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps({"status": result["status"], "selector18_roster_rows": len(result["selector18_original_roster_rows"]),
                      "command30_learn_rows": len(result["learn_grants"][30]), "effect_events": result["effect_events"]}))


if __name__ == "__main__":
    main()
