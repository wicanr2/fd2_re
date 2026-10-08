"""以 IDA Pro 9.4 唯讀匯出 matching 試驗的三個既有函式。

僅在專案授權的 Docker 內執行；一次性資料庫不寫回正式資料庫。
證據入口：docs/data/ida/fd2_matching_pilot_20261008.json。
"""

import hashlib
import json
import os
from pathlib import Path

import ida_auto
import ida_bytes
import ida_funcs
import ida_hexrays
import ida_loader
import ida_nalt
import idaapi
import idautils
import idc


def main():
    repo = Path(os.environ.get("FD2_MATCHING_REPO", "/repo"))
    input_path = Path(ida_nalt.get_input_file_path())
    original = input_path.read_bytes()
    reference = next(
        item for item in json.loads(
            (repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8")
        )["files"] if item["file"] == "FD2.EXE"
    )
    identity = {
        "file": "FD2.EXE", "size": len(original),
        "md5": hashlib.md5(original).hexdigest(),
        "sha256": hashlib.sha256(original).hexdigest(),
    }
    if identity != reference:
        raise RuntimeError("原版雜湊與專案基準不符")
    inventory = json.loads(
        (repo / "docs/data/ida/fd2_function_inventory.json").read_text(encoding="utf-8")
    )
    if inventory["input"] != identity:
        raise RuntimeError("函式清冊與原版版本不符")
    ida_auto.auto_wait()
    if not ida_hexrays.init_hexrays_plugin():
        raise RuntimeError("Hex-Rays 無法使用")
    records = []
    for address in (0x4E893, 0x145CD, 0x1B8E7):
        fn = ida_funcs.get_func(address)
        known = next(item for item in inventory["functions"]
                     if int(item["start"], 16) == address)
        if fn is None or fn.start_ea != address or fn.end_ea != int(known["end"], 16):
            raise RuntimeError(f"{address:#x} 函式邊界與既有 IDA 清冊不符")
        instructions = []
        for ea in idautils.FuncItems(address):
            if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
                continue
            size = idc.get_item_size(ea)
            file_offset = ida_loader.get_fileregion_offset(ea)
            instructions.append({
                "ida_linear_address": hex(ea),
                "file_offset": None if file_offset < 0 else hex(file_offset),
                "loaded_bytes": ida_bytes.get_bytes(ea, size).hex(),
                "file_bytes": None if file_offset < 0 else original[file_offset:file_offset + size].hex(),
                "instruction": idc.generate_disasm_line(ea, 0),
            })
        records.append({
            "inventory": known,
            "ida_analysis_name": idc.get_func_name(address),
            "loaded_bytes": ida_bytes.get_bytes(address, fn.end_ea - address).hex(),
            "instructions": instructions,
            "decompiler_navigation_only": str(ida_hexrays.decompile(address)),
            "direct_callers": [hex(x.frm) for x in idautils.XrefsTo(address) if x.iscode],
            "semantic_warning": "未分級的反編譯器名稱只作導覽，不作原版語意證據。",
        })
        if address == 0x145CD:
            tail_start, tail_end = 0x145C9, 0x145CD
            tail = []
            ea = tail_start
            while ea < tail_end:
                size = idc.get_item_size(ea)
                offset = ida_loader.get_fileregion_offset(ea)
                tail.append({
                    "ida_linear_address": hex(ea), "file_offset": hex(offset),
                    "loaded_bytes": ida_bytes.get_bytes(ea, size).hex(),
                    "file_bytes": original[offset:offset + size].hex(),
                    "instruction": idc.generate_disasm_line(ea, 0),
                })
                ea += size
            records[-1]["external_branch_blocks"] = [{
                "ida_linear_start": hex(tail_start), "ida_linear_end": hex(tail_end),
                "ida_owner_name": idc.get_func_name(tail_start),
                "confidence": "已證實",
                "evidence": "0x145F2 的直接 jge 指令與以下四條原始指令",
                "scope": "函式區間外的四位元組返回尾端；保留既有函式邊界，不重新歸屬",
                "instructions": tail,
            }]
    result = {
        "schema_version": 1,
        "input": identity,
        "tool": {"name": "IDA Pro", "version": idaapi.get_kernel_version(),
                 "address_space": "IDA LE loader linear address; file offsets explicitly separate"},
        "exporter_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
        "function_inventory_sha256": hashlib.sha256(
            (repo / "docs/data/ida/fd2_function_inventory.json").read_bytes()
        ).hexdigest(),
        "functions": records,
    }
    output = Path(os.environ["FD2_MATCHING_IDA_OUTPUT"])
    output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    idc.qexit(0)


if __name__ == "__main__":
    main()
