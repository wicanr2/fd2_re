#!/usr/bin/env python3
"""以固定原版、IDA 匯出與失敗狀態核對清敵引發的死亡收尾堆疊覆寫。

只讀分析工具，不執行原版、不修改遊戲。契約與推論等級見
docs/data/ida/fd2_turn_event54_20261015.json postbattle_clear_viewport_spec。
在 fd2-cap-local 容器執行，原版與研究輸入唯讀掛載。
"""
import argparse
import hashlib
import json
import struct
from pathlib import Path

import capstone
from capstone import CS_ARCH_X86, CS_MODE_32, Cs
from disasm_le import build_fixups, load_code
from le_xref import parse_le


EXE_SHA = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
IDA_SHA = "cdaa0581077eabe9ce94c9afbaef01598fea5178eb3008d1441d6f53082248d5"


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def audit(exe, ida, checkpoint, failure):
    raw = exe.read_bytes()
    if len(raw) != 357074 or hashlib.sha256(raw).hexdigest() != EXE_SHA:
        raise ValueError("原版 EXE 身分不符")
    if digest(ida) != IDA_SHA:
        raise ValueError("IDA 主匯出身分不符")
    exported = json.loads(ida.read_text())
    function = next(f for f in exported["functions"] if f["inventory"]["start"] == "0x1db65")
    rows = [row for chunk in function["chunks"] for row in chunk["instructions"]]
    meta = parse_le(raw)
    code, base = load_code(raw, meta)
    loaded = bytearray(code)
    for address, value in build_fixups(raw, meta).items():
        struct.pack_into("<I", loaded, address - base, value)
    for row in rows:
        address, offset = int(row["ida_linear_address"], 16), int(row["file_offset"], 16)
        original, relocated = bytes.fromhex(row["file_bytes"]), bytes.fromhex(row["loaded_bytes"])
        if raw[offset:offset + len(original)] != original or loaded[address-base:address-base+len(relocated)] != relocated:
            raise ValueError(f"IDA／原始 bytes 不符：{address:#x}")
    decoded = list(Cs(CS_ARCH_X86, CS_MODE_32).disasm(bytes(loaded[0x1DB65-base:0x1DEBE-base]), 0x1DB65))
    if [(i.address, bytes(i.bytes).hex()) for i in decoded] != [
            (int(r["ida_linear_address"], 16), r["loaded_bytes"]) for r in rows]:
        raise ValueError("Capstone 與完整 IDA 指令邊界不符")
    required = {0x1DB73: "83ec7c", 0x1DC29: "890cac", 0x1DD61: "89442478",
                0x1DEB6: "83c47c", 0x1DEB9: "5d", 0x1DEBA: "5f",
                0x1DEBB: "5e", 0x1DEBC: "5b", 0x1DEBD: "c3"}
    by_address = {int(row["ida_linear_address"], 16): row for row in rows}
    if any(by_address[a]["loaded_bytes"] != value for a, value in required.items()):
        raise ValueError("堆疊 writer／consumer 指令不符")
    obj = next(o for o in meta["objs"] if o["base"] <= 0x51A87 < o["base"] + o["vsize"])
    offset = meta["data_off"] + (obj["first"] - 1) * meta["page_size"] + 0x51A87 - obj["base"]
    width, height = struct.unpack_from("<II", raw, offset)
    if (width, height) != (13, 8):
        raise ValueError("固定視野常數不符")
    state = json.loads(checkpoint.read_text())
    failed, _ = json.JSONDecoder().raw_decode(failure.read_text().lstrip())
    if state["exe_sha256"] != EXE_SHA or failed["exe_sha256"] != EXE_SHA:
        raise ValueError("原版執行收據的 EXE 身分不符")
    cx, cy = state["view"]["camera_x"], state["view"]["camera_y"]
    selected = []
    for unit in state["units"]:
        record = bytes.fromhex(unit["raw_hex"])
        if len(record) != 80:
            raise ValueError("單位記錄長度不符")
        x, y = record[:2]
        if not record[5] & 1 and struct.unpack_from("<H", record, 0x40)[0] == 0:
            if cx-1 <= x <= cx+width and cy-1 <= y <= cy+height+1:
                selected.append({"unit_index": unit["index"], "x": x, "y": y})
    if len(selected) <= 35:
        raise ValueError("本收據沒有覆寫到返回位址")
    observed = [("EBP", 31, failed["registers"][5]), ("EDI", 32, failed["registers"][7]),
                ("ESI", 33, failed["registers"][6]), ("EBX", 34, failed["registers"][3]),
                ("return EIP", 35, int(failed["stop_eip"], 16))]
    bases, matches = [], []
    for name, index, value in observed:
        unit = selected[index]
        relative = 0x8088 + (unit["x"]-1-cx)*24 + (unit["y"]-1-cy)*10944 - 0xAB0
        bases.append(value-relative)
        matches.append({"consumer": name, "array_index": index, "stack_offset": hex(index*4),
                        **unit, "observed_value": hex(value), "derived_canvas_base": hex(value-relative)})
    if len(set(bases)) != 1:
        raise ValueError("五項覆寫值無法還原為同一畫面基底")
    allocations = [event for event in failed["near_heap_execution"]["events"]
                   if event["kind"] == "allocation" and event["pointer"] == bases[0]
                   and event["parameter"] == 0x25680 and event["allocation_pointer_valid"]]
    if not allocations:
        raise ValueError("還原的畫面基底沒有對應原生配置紀錄")
    meanings = {0x1DB73: "保留124-byte局部堆疊", 0x1DC29: "以EBP索引寫入畫面指標",
                0x1DD61: "將malloc返回指標存於局部偏移0x78", 0x1DEB6: "釋放124-byte局部堆疊",
                0x1DEB9: "恢復EBP", 0x1DEBA: "恢復EDI", 0x1DEBB: "恢復ESI",
                0x1DEBC: "恢復EBX", 0x1DEBD: "由堆疊取得返回EIP"}
    critical = [{**by_address[a], "original_function": "sub_1DB65",
                 "attached_semantic": meanings[a], "confidence": "已證實",
                 "evidence": f"固定EXE SHA256 {EXE_SHA}／IDA匯出 {IDA_SHA}；原始bytes及消費端核對"}
                for a in required]
    return {"schema_version": 1, "kind": "fd2_clear_viewport_stack_audit",
            "confidence": "強推論：原始 writer／consumer 已證實，五項覆寫與原生配置相符；未取得實際RET指令尾端。",
            "input": {"exe_sha256": EXE_SHA, "ida_export_sha256": IDA_SHA,
                      "checkpoint_sha256": digest(checkpoint), "failure_sha256": digest(failure)},
            "warning": "⚠ 崩潰關聯列強推論；原始指令效果已證實，不推定作者的C宣告。",
            "tools": {"primary": exported["tool"], "cross_check": {
                          "name": "Capstone", "version": capstone.__version__, "mode": "32-bit x86"},
                      "audit_sha256": digest(Path(__file__))},
            "original_function": {"name": "sub_1DB65", "address_space": "IDA LE loader linear",
                                  "start": "0x1DB65", "end_exclusive": "0x1DEBE", "bytes": 857,
                                  "instructions_verified": len(rows),
                                  "critical_original_instructions": critical},
            "frame": {"local_bytes": 124, "pointer_slots": 30, "malloc_pointer_offset": "0x78",
                      "saved_register_offsets": ["0x7C", "0x80", "0x84", "0x88"], "return_offset": "0x8C"},
            "viewport": {"camera_x": cx, "camera_y": cy, "width_constant": width, "height_constant": height,
                         "constant_address_space": "FD2.EXE LE linear, file-backed initial values"},
            "selected_count": len(selected), "selected": selected, "overwrite_matches": matches,
            "matching_native_allocations": allocations,
            "limitations": ["只分析指定失敗，不將未執行的局部模型當成正常玩家收據。",
                            "清敵注入同時新增多個HP0角色；不可推定正常攻擊一定觸發相同溢位。"]}


def main():
    if not Path("/.dockerenv").is_file():
        raise SystemExit("本工具只在 Docker 執行")
    parser = argparse.ArgumentParser(description=__doc__)
    for key in ("exe", "ida", "checkpoint", "failure", "out"):
        parser.add_argument("--" + key, type=Path, required=True)
    args = parser.parse_args()
    if args.out.exists():
        raise SystemExit("輸出已存在，請使用新的明確路徑")
    result = audit(args.exe, args.ida, args.checkpoint, args.failure)
    args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(f"完整857 bytes已核對；{result['selected_count']}個候選超過30格，五項覆寫對上同一原生配置。")


if __name__ == "__main__":
    main()
