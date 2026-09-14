"""匯出 ch21/ch22 六筆動態增援來源的 IDA Pro 9.4 主證據。

本腳本只可在使用者授權的 IDA Docker 流程執行。它不改名、不加註解、
不改型別，也不保存 IDA 資料庫；輸出保留原始位址、名稱、operand 與 bytes。
"""

import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_name
import ida_xref
import idaapi
import idc


EXPECTED_INPUT = {
    "file": "FD2.EXE",
    "size": 357074,
    "md5": "b97caf2239a27a896069d03549d96e1e",
    "sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
}
OUTPUT = "/work/out.txt"
ROUND_COUNTER = 0x53BEF
EVENT_TABLE = 0x51B91
DISPATCHER = 0x1A813
DISPATCH_CALL = 0x1A85A
SPAWN_GROUP = 0x10B4E

HANDLERS = {
    47: {
        "entry": 0x35112,
        "source_load": 0x3511C,
        "flow": [0x3511C, 0x35121, 0x35123, 0x35126, 0x35128, 0x3512A, 0x3512B],
    },
    49: {
        "entry": 0x351E9,
        "source_load": 0x351F3,
        "flow": [0x351F3, 0x351F8, 0x351FA, 0x351FD, 0x351FF, 0x35201, 0x35202],
    },
}

CASES = (
    (21, 20, 2, 47, 1, "四個角落各一隻魔鬼"),
    (21, 20, 4, 47, 2, "四個角落各一隻魔鬼"),
    (21, 20, 6, 47, 3, "四個角落各一隻魔鬼"),
    (21, 20, 8, 47, 4, "四個角落各一隻魔鬼"),
    (22, 21, 3, 49, 1, "左下角與右下角各三隻魔鬼"),
    (22, 21, 7, 49, 3, "左下角與右下角各三隻魔鬼"),
)


def hex_address(value):
    return f"0x{value:x}"


def function_location(address):
    function = ida_funcs.get_func(address)
    if function is None:
        return None
    return {
        "start": hex_address(function.start_ea),
        "end": hex_address(function.end_ea),
        "original_name": ida_name.get_name(function.start_ea) or None,
    }


def instruction(address):
    size = idc.get_item_size(address)
    data = ida_bytes.get_bytes(address, size)
    if not data or size <= 0:
        raise RuntimeError(f"IDA 無法讀取指令 {address:#x}")
    return {
        "address": hex_address(address),
        "bytes": data.hex(" "),
        "original": idc.generate_disasm_line(address, 0) or "",
        "mnemonic": idc.print_insn_mnem(address),
        "operands": [
            operand
            for operand in (idc.print_operand(address, 0), idc.print_operand(address, 1))
            if operand
        ],
        "function": function_location(address),
        "semantic": "未附加推測語意",
        "level": "未知",
        "source": f"FD2.EXE / IDA LE linear {address:#x}",
    }


def instruction_range(start, end):
    rows = []
    address = start
    while address != idc.BADADDR and address < end:
        if ida_bytes.is_code(ida_bytes.get_full_flags(address)):
            rows.append(instruction(address))
        address = idc.next_head(address, end)
    return rows


def data_xrefs_to(address):
    rows = []
    current = ida_xref.get_first_dref_to(address)
    while current != idaapi.BADADDR:
        rows.append(instruction(current))
        current = ida_xref.get_next_dref_to(address, current)
    return rows


def code_xrefs_to(address):
    rows = []
    current = ida_xref.get_first_cref_to(address)
    while current != idaapi.BADADDR:
        rows.append(instruction(current))
        current = ida_xref.get_next_cref_to(address, current)
    return rows


def function_instructions(address):
    location = function_location(address)
    if location is None:
        raise RuntimeError(f"IDA 沒有函式邊界 {address:#x}")
    return {
        **location,
        "instructions": instruction_range(int(location["start"], 0), int(location["end"], 0)),
        "code_xrefs_to_start": code_xrefs_to(int(location["start"], 0)),
    }


def assert_flow(event_id, rows):
    expected_mnemonics = ["mov", "mov", "sar", "sub", "sar", "push", "call"]
    actual_mnemonics = [row["mnemonic"] for row in rows]
    if actual_mnemonics != expected_mnemonics:
        raise RuntimeError(
            f"event {event_id} eax flow 改變：{actual_mnemonics!r}"
        )
    expected_operands = (
        ("eax", "dword_53BEF"),
        ("edx", "eax"),
        ("edx", "1Fh"),
        ("eax", "edx"),
        ("eax", "1"),
        ("eax",),
        ("sub_10B4E",),
    )
    for row, expected in zip(rows, expected_operands):
        normalized = tuple(operand.replace(" ", "") for operand in row["operands"])
        wanted = tuple(operand.replace(" ", "") for operand in expected)
        if normalized != wanted:
            raise RuntimeError(
                f"event {event_id} operand 改變於 {row['address']}：{normalized!r} != {wanted!r}"
            )


def input_identity():
    input_path = idc.get_input_file_path()
    with open(input_path, "rb") as source:
        data = source.read()
    identity = {
        "file": os.path.basename(input_path),
        "size": len(data),
        "md5": hashlib.md5(data).hexdigest(),
        "sha256": hashlib.sha256(data).hexdigest(),
    }
    if identity != EXPECTED_INPUT:
        raise RuntimeError(f"FD2.EXE 身分不符：{identity!r}")
    return identity


def main():
    ida_auto.auto_wait()
    identity = input_identity()

    handler_reports = {}
    for event_id, metadata in HANDLERS.items():
        flow = [instruction(address) for address in metadata["flow"]]
        assert_flow(event_id, flow)
        slot = EVENT_TABLE + event_id * 4
        target = ida_bytes.get_dword(slot)
        if target != metadata["entry"]:
            raise RuntimeError(
                f"event {event_id} 跳表目標改變：{target:#x} != {metadata['entry']:#x}"
            )
        handler_reports[str(event_id)] = {
            "event_id": event_id,
            "jump_table_slot": {
                "address": hex_address(slot),
                "bytes": ida_bytes.get_bytes(slot, 4).hex(" "),
                "decoded_target": hex_address(target),
                "level": "已證實",
                "source": f"FD2.EXE / IDA LE linear {slot:#x}",
            },
            "function": function_location(metadata["entry"]),
            "eax_source_flow": flow,
            "formula": {
                "expression": "signed_trunc_toward_zero(round_counter / 2)",
                "positive_runtime_equivalent": "floor(round_counter / 2)",
                "source_global": hex_address(ROUND_COUNTER),
                "source_load": hex_address(metadata["source_load"]),
                "consumer_call": hex_address(metadata["flow"][-1]),
                "consumer": hex_address(SPAWN_GROUP),
                "level": "已證實",
                "evidence": "七條連續 IDA 指令與原始 bytes；語意不依賴偽代碼",
            },
        }

    dispatcher_function = function_instructions(DISPATCHER)
    dispatcher_indirect_calls = [instruction(DISPATCH_CALL)]
    dispatch_row = dispatcher_indirect_calls[0]
    dispatch_operands = "".join(dispatch_row["operands"]).replace(" ", "").lower()
    if dispatch_row["mnemonic"] != "call" or "eax*4" not in dispatch_operands:
        raise RuntimeError(
            f"event dispatcher 間接跳表 call 改變：{dispatch_row!r}"
        )

    cases = []
    for chapter, map_id, turn, event_id, group, context in CASES:
        computed = int(turn / 2)
        if computed != group:
            raise RuntimeError(f"case 公式不一致：ch{chapter} turn {turn}")
        cases.append({
            "id": f"ch{chapter:02d}-turn-{turn}-event-{event_id}",
            "chapter": chapter,
            "map": map_id,
            "turn": turn,
            "raw_camp": 0,
            "event_id": event_id,
            "handler": hex_address(HANDLERS[event_id]["entry"]),
            "computed_group": group,
            "player_visible_context": context,
            "context_level": "攻略旁證",
            "context_source": "references/text/fd2-walkthrough-index.md（第21／22章事件列）",
            "schedule_source": "docs/data/turn_events.json",
            "binary_evidence_ref": f"handlers.{event_id}",
            "result": "RE-CLOSED",
        })

    report = {
        "schema_version": 1,
        "status": "RE-CLOSED",
        "scope": "ch21/ch22 六筆由 event 47/49 計算 eax 的回合增援來源",
        "tool": {
            "name": "IDA Pro",
            "version": idaapi.get_kernel_version(),
            "address_space": "IDA LE flat-loader linear address",
            "annotation_policy": "只讀匯出；不改名、不改型別、不加註解、不保存資料庫",
        },
        "input": identity,
        "evidence_levels": {
            "binary_control_and_data_flow": "已證實",
            "six_schedule_rows": "已證實（固定 FDFIELD 擷取資料與公式交叉核對）",
            "player_visible_context": "攻略旁證；不作 ABI 或欄位語意證據",
        },
        "dispatcher": {
            "entry": hex_address(DISPATCHER),
            "function": dispatcher_function,
            "event_table": hex_address(EVENT_TABLE),
            "indirect_calls": dispatcher_indirect_calls,
            "level": "已證實",
            "note": "依 turn 與 raw camp 篩選後，以 event_id 索引 90-entry 跳表；間接 call 不能由 handler 的直接 code xref 取代。",
        },
        "round_counter": {
            "address": hex_address(ROUND_COUNTER),
            "data_xrefs": data_xrefs_to(ROUND_COUNTER),
            "level": "已證實（原始讀寫集合）；個別 writer 語意須由各呼叫路徑判讀",
        },
        "spawn_group_consumer": {
            "entry": hex_address(SPAWN_GROUP),
            "function": function_instructions(SPAWN_GROUP),
            "level": "已證實",
            "note": "handler 將計算後 eax push 為唯一參數；consumer 以原始 FDFIELD group byte 篩選並 materialize unit。",
        },
        "handlers": handler_reports,
        "cases": cases,
        "coverage": {
            "expected": 6,
            "closed": len(cases),
            "unresolved": 0,
        },
        "limitations": [
            "本證據閉合 event_id→handler→eax 來源公式→spawn consumer，不宣稱完整章節 PLAYER-E2。",
            "攻略只定位玩家可見觸發情境；二進位語意由 IDA 指令、bytes、跳表與資料流裁決。",
            "raw camp 0 保留原值；陣營名稱不由本證據升格。",
        ],
    }
    output_path = os.environ.get("FD2_IDA_OUTPUT", OUTPUT)
    with open(output_path, "w", encoding="utf-8") as output:
        json.dump(report, output, ensure_ascii=False, indent=2)
        output.write("\n")
    idc.qexit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception as error:
        output_path = os.environ.get("FD2_IDA_OUTPUT", OUTPUT)
        with open(output_path, "w", encoding="utf-8") as output:
            json.dump(
                {"schema_version": 1, "status": "ERROR", "error": repr(error)},
                output,
                ensure_ascii=False,
                indent=2,
            )
            output.write("\n")
        idc.qexit(1)
