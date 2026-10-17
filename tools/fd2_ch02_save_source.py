#!/usr/bin/env python3
"""核對第2章希莉亞九個保存差異的唯讀來源鏈；不放行保存差異。"""

import argparse
import hashlib
import json
from pathlib import Path

import fd2save


EXE_SHA = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
COMMIT = "f5be010ac1aa5b8f7ab6db53d1507c9f8dcfd059"


def read(path):
    return json.loads(path.read_text(encoding="utf-8"))


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def require(condition, message):
    if not condition:
        raise ValueError(message)


def verify(args):
    original = args.original.read_bytes()
    require(digest(args.original) == EXE_SHA, "原始EXE雜湊不符")
    image = read(args.ida_image)
    require(image["input"]["sha256"] == EXE_SHA and image["tool"]["version"] == "9.4",
            "不是固定版本的IDA9.4證據")
    functions = {f["inventory"]["start"].lower(): f for f in image["functions"]}
    evidence = []
    # 沿已閉合的配置、建構、複製與保存函式，只覆核相關直接指令。
    intervals = [(0x1088D, 0x109CE, 0x10B4E), (0x10C50, 0x10C50, 0x11019),
                 (0x112A5, 0x112A5, 0x1145A), (0x11506, 0x11506, 0x115B6),
                 (0x30012, 0x30012, 0x301F4)]
    for start, low, high in intervals:
        function = functions[hex(start)]
        rows = [i for c in function["chunks"] for i in c["instructions"]
                if low <= int(i["ida_linear_address"], 16) < high]
        require(rows, "缺少原始函式指令")
        for row in rows:
            offset = int(row["file_offset"], 16)
            raw = bytes.fromhex(row["file_bytes"])
            require(original[offset:offset + len(raw)] == raw, "IDA指令與原始file bytes不同")
        evidence.append({"original_name": function["ida_analysis_name"],
                         "ida_linear_start": hex(low), "ida_linear_end": hex(high),
                         "instructions_checked": len(rows)})

    runner = read(args.oracle / "runner.json")
    require(runner["dosgolem_commit"] == COMMIT, "不是已固定原版執行器")
    require(runner["dosgolem_tracked_dirty_files"] == 0 and runner["heap_profile"] == "native"
            and runner["near_heap_policy"]["status"] == "reviewed", "原版配置來源未受控")
    before = read(args.oracle / "checkpoint-0459.json")
    after = read(args.oracle / "checkpoint-0460.json")
    require(before["unit_base"] == after["unit_base"] and len(before["units"]) == 27,
            "建構前記憶體基底或有效筆數不符")
    unit = after["units"][27]
    require(unit["identity"] == 8 and unit["camp"] == 2, "不是希莉亞場上新紀錄")
    raw = bytes.fromhex(unit["raw_hex"])
    base = after["unit_base"] + 27 * 80
    watched = set(range(base + 0x28, base + 0x31))
    observations = [json.loads(l) for l in (args.oracle / "memory-change.jsonl").read_text().splitlines()]
    require(observations and observations[0]["kind"] == "baseline", "缺少建構前基線")
    baseline = observations[0]
    require(baseline["step"] <= before["steps"] and baseline["memory_valid"], "觀察基線過晚或無效")
    values = {int(s["address"], 16): s["before"] for s in baseline["samples"]}
    require(watched <= values.keys(), "九個位元組未全部觀察")
    require(bytes(values[a] for a in sorted(watched)) == raw[0x28:0x31], "建構前後九byte不同")
    changes = [o for o in observations if o["kind"] == "change"]
    require(all(o["memory_valid"] for o in observations), "記憶體觀察失效")
    require(not any(int(s["address"], 16) in watched for o in changes for s in o["samples"]),
            "觀察到九byte被寫入，必須重新審查來源")
    controls = []
    for offset, writer, expected in [(6, 0x10EF5, 2), (8, 0x10EFF, 8), (0x40, 0x10FE9, 44)]:
        hits = [o for o in changes if int(o["eip_before"], 16) == writer
                and any(int(s["address"], 16) == base + offset and s["after"] == expected
                        for s in o["samples"])]
        require(len(hits) == 1, f"缺少正對照writer {writer:#x}")
        controls.append(hits[0])
    trace = [json.loads(l) for l in (args.oracle / "eip-trace.jsonl").read_text().splitlines()]
    entries = [o for o in trace if int(o["eip"], 16) == 0x10C50]
    require(len(entries) == 1, "有界窗口沒有唯一人物建構入口")

    saved = fd2save.decode(args.oracle_save.read_bytes())
    record_offset = fd2save.SLOT_OFFSET + 5 * fd2save.UNIT_SIZE
    require(saved[record_offset + 8] == 8, "保存名冊第5筆不是希莉亞")
    require(saved[record_offset + 0x28:record_offset + 0x31] == raw[0x28:0x31],
            "場上到保存九byte未原樣保留")
    remake = fd2save.decode(args.remake_save.read_bytes())
    difference_offsets = [i for i, (a, b) in enumerate(zip(saved, remake)) if a != b]
    expected_offsets = list(range(record_offset + 0x28, record_offset + 0x31)) + [0x59C7, 0x59C8]
    require(len(saved) == len(remake) and difference_offsets == expected_offsets,
            "差異不只九byte及兩檢查碼")
    # 僅在局部副本上作因果核對；不寫回任一原版或正式重製SAV。
    projected = bytearray(remake)
    projected[record_offset + 0x28:record_offset + 0x31] = raw[0x28:0x31]
    require(fd2save.encode(projected) == args.oracle_save.read_bytes(),
            "九byte不足以解釋其餘檢查碼差異")
    linked = read(args.match / "report.json")
    require(linked["input"]["sha256"] == EXE_SHA and linked["evidence_sha256"] == digest(args.ida_image),
            "匹配報告來源不符")
    candidates = [t for t in linked["trials"] if t["address"] == "0x112a5"]
    require(len(candidates) == 9, "JOIN C候選矩陣不完整")
    return {
        "schema_version": 1, "kind": "fd2_ch02_save_source", "status": "passed",
        "input": image["input"], "tool": image["tool"], "dosgolem_commit": COMMIT,
        "verifier_sha256": digest(Path(__file__)), "ida_image_sha256": digest(args.ida_image),
        "runner_sha256": digest(args.oracle / "runner.json"),
        "memory_trace_sha256": digest(args.oracle / "memory-change.jsonl"),
        "eip_trace_sha256": digest(args.oracle / "eip-trace.jsonl"),
        "original_instruction_ranges": evidence, "record_base": hex(base),
        "baseline": baseline, "positive_controls": controls, "constructor_entry": entries[0],
        "raw28_30_hex": raw[0x28:0x31].hex(), "writes_to_nine_bytes": 0,
        "oracle_save_sha256": digest(args.oracle_save), "matching_report_sha256": digest(args.match / "report.json"),
        "remake_save_sha256": digest(args.remake_save), "all_other_plain_bytes_equal": True,
        "nine_byte_local_projection_reproduces_original_save": True,
        "join_c_candidates": candidates,
        "conclusion": "九byte在場上紀錄建構前已存在，10C50保留它們；其後整筆copy到名冊並保存。",
        "confidence": "本份收據的建構前後保留與保存傳遞已證實；更早的配置區內容producer仍未知。",
        "limitations": ["112A5完整C候選是否匹配依逐筆結果，不能以近似C證明原版語意。",
                        "本檢查不接受保存差異，也不證明九byte在所有玩法中都未使用。",
                        "觀察者只報值改變；同值寫入依IDA直接指令與建構範圍另行審查。"]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for flag in ("original", "ida-image", "oracle", "oracle-save", "remake-save", "match", "out"):
        parser.add_argument("--" + flag, type=Path, required=True)
    args = parser.parse_args()
    require(Path("/.dockerenv").exists(), "本工具只在Docker執行")
    report = verify(args)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(report["status"], report["raw28_30_hex"], "完整JOIN C匹配", sum(t["exact_interval_bytes"] for t in report["join_c_candidates"]))


if __name__ == "__main__":
    main()
