#!/usr/bin/env python3
"""核對第27章選擇性鎖血的雙端增援診斷；不改正式章台帳。"""
import argparse
import base64
import hashlib
import json
import re
from pathlib import Path
import fd2save

EXE = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
BASE_SAVE = "89a8d7a93f8baae11e2315f9d06c9a979ef5d6d608ee1558ec42ffb198cdbbd8"
BOOST_SAVE = "0eb3de0a84fd2907f28d5112380511a2e05d13423b0b04a66db5b96c9b9836ea"


def read(path):
    return json.loads(path.read_text())


def lines(path):
    return [json.loads(line) for line in path.read_text().splitlines() if line.strip()]


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--original", type=Path, required=True)
    p.add_argument("--remake", type=Path, required=True)
    p.add_argument("--baseline", type=Path, required=True)
    p.add_argument("--unboosted", type=Path, required=True)
    p.add_argument("--source-slot", type=Path, required=True)
    p.add_argument("--slot", type=Path, required=True)
    p.add_argument("--manifest", type=Path, required=True)
    p.add_argument("--output", type=Path, required=True)
    a = p.parse_args()
    require(Path("/.dockerenv").is_file(), "只在Docker執行")
    runner = read(a.original / "runner.json")
    settings = read(a.remake / "replay-settings.json")
    manifest = read(a.manifest)
    require(manifest["base_sha256"] == BASE_SAVE and manifest["output_sha256"] == BOOST_SAVE, "非本次已揭露的固定SAV pair")
    require(manifest["base_chapter"] == manifest["target_chapter"] == 26 and manifest["base_slot"] == manifest["out_slot"] == 0 and manifest["seed"] == 4, "AP槽章號／槽號／seed不同")
    original_slot = Path(__file__).resolve().parents[1] / runner["state_directory"] / "FD2.SAV"
    require(digest(original_slot) == digest(a.slot), "原版覆蓋層不是同一AP槽")
    require(digest(a.slot) == manifest["output_sha256"] and digest(a.source_slot) == manifest["base_sha256"], "AP槽來源雜湊不同")
    source, boosted = fd2save.decode(a.source_slot.read_bytes()), fd2save.decode(a.slot.read_bytes())
    require(len(source) == len(boosted) == 22987, "AP槽長度不同")
    start_slot, end_slot = fd2save.slot_bounds(0)
    allowed = {22983,22984} # 已有fdsave校驗碼欄；建構器另驗保存格式。
    for index in range(27):
        for offset in (0x37,0x48):
            at = start_slot + index*80 + offset
            before = int.from_bytes(source[at:at+2],"little",signed=True)
            after = int.from_bytes(boosted[at:at+2],"little",signed=True)
            require(after == before+1000, f"名冊{index}的AP強化不是1000")
            allowed.update((at,at+1))
    slot_diffs = [i for i,(x,y) in enumerate(zip(source,boosted)) if x!=y]
    require(set(slot_diffs) <= allowed, "AP槽有未授權欄位變更")
    require(digest(a.remake/"FD2.SAV") == digest(a.slot), "診斷途中改寫了來源保存")
    require(runner["original_fd2_exe_sha256"] == EXE and runner["allow_hp_lock_control"], "原版來源／政策不符")
    require(runner["dosgolem_tracked_dirty_files"] == 0 and runner["dosgolem_untracked_files"] == 0, "執行器來源未凍結")
    require(settings["modified_path_hp_lock"] and settings["go_seed"] == 4, "重製測試政策不符")
    actions = lines(a.original / "actions.jsonl")
    remake_log = a.remake / "checkpoints.jsonl"
    checkpoints = lines(remake_log)
    marks = {x["label"]: x for x in actions if x["kind"] == "mark"}
    def original(seq):
        cp = read(a.original / f"checkpoint-{seq:04d}.json")
        require(cp["exe_sha256"] == EXE, "checkpoint EXE來源不同")
        return cp
    def remake(seq, kind):
        found = [x for x in checkpoints if x["oracle_seq"] == seq and x["kind"] == kind]
        require(len(found) == 1, f"重製seq{seq}／{kind}缺唯一收據")
        return found[0]
    # 鎖血前的所有控制邊界保持。新增旗標metadata不參與guest狀態比較。
    start = marks["battle_start"]["seq"]
    prefix = []
    boosted_boot_register_differences = []
    for seq in range(start + 1):
        old = read(a.baseline / f"checkpoint-{seq:04d}.json")
        current = read(a.unboosted / f"checkpoint-{seq:04d}.json")
        for key in ("eip", "steps", "view", "units", "registers"):
            require(old.get(key) == current.get(key), f"啟動前綴seq{seq}欄位{key}改變")
        png = f"checkpoint-{seq:04d}.png"
        require(digest(a.baseline/png) == digest(a.unboosted/png), f"啟動前綴seq{seq}畫面改變")
        if seq <= 16: # LOAD提交之前，AP槽也應保留相同guest狀態。
            fresh = original(seq)
            for key in ("eip","steps","view","units"):
                require(old.get(key)==fresh.get(key), f"AP槽BOOT前綴seq{seq}改變")
            if old["registers"] != fresh["registers"]:
                boosted_boot_register_differences.append(dict(seq=seq,before=old["registers"],after=fresh["registers"],level="unknown; different SAV input, not the disabled-tool regression"))
        prefix.append(seq)
    lock_actions = [x for x in actions if x["kind"] == "hp_lock"]
    require(len(lock_actions) == 3, "鎖血階段數量不同")
    expected_indices = [[0,24,25,26], [0,24,25,26], [0,26]]
    for action, indices in zip(lock_actions, expected_indices):
        require([x["index"] for x in action["captured"]] == indices, "鎖血綁定順序不同")
        cp = original(action["seq"])
        rc = remake(action["seq"], "hp_lock")
        require(cp["hp_lock_control"]["active"] == action["captured"] == rc["hp_locks"], "雙端鎖血值不同")
    require([x["hp"] for x in lock_actions[0]["captured"]] == [30420,3200,3200,3200], "初始捕獲不同")
    require([x["hp"] for x in lock_actions[1]["captured"]] == [30420,1,1,3200], "HP1政策不同")
    clears = [x for x in actions if x["kind"] == "force_enemy_clear"]
    require(len(clears) == 1 and clears[0]["cleared"] > 0 and clears[0]["preserved_indices"] == [24,25,26], "保護清敵政策不同")
    clear = remake(clears[0]["seq"], "force_enemy_clear")
    require(f"cleared={clears[0]['cleared']}（" in clear.get("note", ""), "重製清敵筆數不同")
    wave = marks["round_6"]["seq"]
    ow, rw = original(wave), remake(wave, "mark:round_6")
    require(len(ow["units"]) == rw["runtime_unit_count"] == 75, "前波十二筆未實際追加")
    require(rw["runtime_groups"].get("1") == rw["runtime_groups"].get("2") == 6, "重製前波群組未實際追加十二筆")
    final = marks["captain_reinforcements_arrived"]["seq"]
    oc, rc = original(final), remake(final,"mark:captain_reinforcements_arrived")
    require(len(oc["units"]) == rc["runtime_unit_count"] == 96, "三路二十一筆未實際追加")
    state = rc["event_state"]
    state = list(base64.b64decode(state)) if isinstance(state,str) else state
    require(state[16] == 2, "重製死亡事件計數不是2")
    ou = {x["index"]:x for x in oc["units"]}
    ru = {x["native_index"]:x for x in rc["units"]}
    require(ou[24]["hp"] == ou[25]["hp"] == 0 and 24 not in ru and 25 not in ru, "前兩名隊長未死亡")
    require(ou[26]["hp"] == ru[26]["hp"] == 3200 and ou[0]["hp"] == ru[0]["hp"] == 30420, "第三名隊長或索爾未保持存活")
    require(sum(x["camp"]==0 and x["hp"]>0 for x in ou.values()) == sum(x["camp"]==0 for x in ru.values()) == 22, "增援後敵軍數不同")
    appended = []
    for index in range(75,96):
        o,r = ou[index],ru[index]
        raw = bytes.fromhex(o["raw_hex"])
        group = 3 + (index-75)//7
        require(r["group"] == group and raw[8] == r["record_byte8"], f"新增record{index}來源群組不同")
        for key in ("camp","x","y","hp"):
            require(o[key] == r[key],f"新增record{index}的{key}不同")
        appended.append(dict(index=index,group=group,record_byte8=raw[8],x=o["x"],y=o["y"],hp=o["hp"]))
    trace_path = a.original / "eip-trace.jsonl"
    trace = lines(trace_path)
    death = [x for x in trace if x["eip"] == "0x358EA"]
    increments = [x for x in trace if x["eip"] == "0x35992"]
    staging = [x for x in trace if x["eip"] == "0x35822"]
    require(len(death) == len(increments) == 2 and len(staging) == 5, "死亡或staging入口數不同")
    groups = [(int(x["stack"][3],0),int(x["stack"][1],0),int(x["stack"][2],0),x["unit_count"]) for x in staging]
    require(groups == [(1,3,27,63),(2,15,27,69),(3,9,44,75),(4,0,9,82),(5,17,9,89)], "實際staging引數或追加順序不同")
    require(death[0]["step"] < increments[0]["step"] < death[1]["step"] < staging[2]["step"] < increments[1]["step"], "死亡→三路增援時序不同")
    require(not any("divergence" in x.get("note","") or x["kind"] == "runtime_error" for x in checkpoints), "重製流程有分岔")
    test_log = (a.remake/"test.log").read_text()
    ai = re.search(r"AI 入口比較：原版 (\d+) 筆、已消費 (\d+) 筆、順序分岔 (\d+) 筆",test_log)
    require(ai is not None and ai[1] == ai[2] and int(ai[3]) == 0, "AI重播未完整消費或順序分岔")
    require("--- PASS: TestChapterParityReplay" in test_log and "--- FAIL:" not in test_log, "重製實測未通過")
    attacks = [x for x in actions if x["kind"] == "attack"]
    require(len(attacks) == 2, "正常攻擊次數不同")
    report = dict(schema_version=1,kind="fd2_ch27_captain_reinforcement_resample",date="2026-10-16",status="passed",evidence_level="RUNTIME-E1 modified-path diagnostic",formal_chapter_receipt_unchanged="docs/data/ui-traces/parity-ch27.json",chapter_ledger_unchanged="24/30",original_runner=runner,remake_settings=settings,prefix_boundaries_unchanged=len(prefix),lock_actions=lock_actions,force_clear=clears[0],staging=[dict(group=g,x=x,y=y,unit_count_before=n) for g,x,y,n in groups],appended_records=appended,original_counter_evidence="既有IDA358EA分支與兩次35992增量路徑；沒有直接擷取原版state16 byte，不冒稱直接記憶體讀值",remake_state16=state[16],last_captain_hp=3200,saul_hp=30420,final_enemy_alive=22,trace_sha256=digest(trace_path),actions_sha256=digest(a.original/"actions.jsonl"),remake_checkpoints_sha256=digest(remake_log),limits=["章內週期鎖血與HP1／清敵皆明示修改；不證明傷害、自然存活、AI選目標或PLAYER-E2","畫面僅參考，未要求逐frame相同","不驗第三名隊長死亡臂或重新保存整章"])
    report["slot_policy"] = dict(base_sha256=digest(a.source_slot), output_sha256=digest(a.slot), base_ap_boost=1000, roster_count=27, decoded_differing_offsets=slot_diffs, manifest_sha256=digest(a.manifest), original_input_sha256=digest(original_slot), remake_input_sha256=digest(a.remake/"FD2.SAV"))
    report["unboosted_source_regression"] = str(a.unboosted)
    report["boosted_boot_register_differences"] = boosted_boot_register_differences
    report["advisory_view"] = dict(original=oc["view"],remake_cursor=rc["cursor"],remake_camera=rc["camera"],scope="三路援軍完成的游標／鏡頭交回點不同；流程與新增單位驗證分開，未宣稱畫面或全部view一致")
    root = Path(__file__).resolve().parents[1]
    report["verification_sources"] = [dict(path=name,sha256=digest(root/name)) for name in (
        "tools/dosgolem_oracle_drive.py", "tools/dosgolem_oracle_container.sh",
        "tools/dosgolem_oracle.sh", "tools/verify_ch27_reinforcement_resample.py",
        "docs/data/parity-plans/ch27-captain-reinforcement-lock-r3.jsonl",
        "remake/cmd/fd2/chapter_parity_replay_test.go",
        "remake/cmd/fd2/chapter_parity_hp_lock_test.go",
        "remake/cmd/fd2/native_chapter_one_play_test.go")]
    report["remake_production_base_commit"] = "eb3cde31147d7dc06713d05e9423f699713c1a4b"
    report["ai_replay"] = dict(original_entries=int(ai[1]),consumed=int(ai[2]),order_divergences=int(ai[3]))
    report["advisory_frames"] = dict(original=str(a.original/f"checkpoint-{final:04d}.png"), original_sha256=digest(a.original/f"checkpoint-{final:04d}.png"),remake=str(a.remake/rc["frame"]),remake_sha256=digest(a.remake/rc["frame"]),comparison="相近狀態；新增單位相同，終點view不同。畫面參考，無逐幀／像素閘門")
    report["advisory_view"]["followup_issue"] = 224
    a.output.write_text(json.dumps(report,ensure_ascii=False,indent=2)+"\n")
    print(f"passed: original/remake 12+21 reinforcements; last captain alive; {a.output}")


if __name__ == "__main__":
    main()
