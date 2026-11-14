"""列出 FD2.EXE 中參照指定資料位址的每一條指令（IDA 9.4 批次腳本）。

用途：找某個全域變數的寫入端與讀取端，例如經驗累計 `[0x53EC8]`。每一筆保留原始
線性位址、IDA 的 xref 型別、所在函式邊界、指令文字與原始 bytes，不改名、不加語意。

IDA 的資料 xref 只涵蓋直接定址的運算元；經暫存器或相對基底的間接讀寫不會出現，
不能用「這裡沒列」證明沒有寫入端。

用法（經 tools/ida.sh）：
    tools/ida.sh ida_export_data_writers.py <輸出 JSON> 53EC8 [其他位址…]
"""

import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_lines
import ida_xref
import idaapi
import idc

XREF_TYPES = {
    ida_xref.dr_O: "offset",
    ida_xref.dr_W: "write",
    ida_xref.dr_R: "read",
    ida_xref.dr_T: "text",
    ida_xref.dr_I: "info",
}


def func_info(ea):
    fn = ida_funcs.get_func(ea)
    if not fn:
        return None
    return {"start": hex(fn.start_ea), "end": hex(fn.end_ea), "name": idc.get_func_name(fn.start_ea) or None}


def insn(ea):
    size = idc.get_item_size(ea)
    return {
        "address": hex(ea),
        "text": ida_lines.tag_remove(idc.generate_disasm_line(ea, 0) or ""),
        "bytes": ida_bytes.get_bytes(ea, size).hex() if size > 0 else "",
    }


def main():
    ida_auto.auto_wait()
    input_path = idc.get_input_file_path()
    with open(input_path, "rb") as source:
        raw = source.read()
    targets = [int(a, 16) for a in os.environ.get("FD2_IDA_ADDRESSES", "").split()]
    report = {
        "tool": "tools/ida_export_data_writers.py",
        "ida_version": idaapi.get_kernel_version(),
        "address_space": "IDA linear address (LE object base)",
        "input_file": os.path.basename(input_path),
        "input_size": len(raw),
        "input_md5": hashlib.md5(raw).hexdigest(),
        "input_sha256": hashlib.sha256(raw).hexdigest(),
        "limitation": "只含 IDA 直接資料 xref；間接定址的讀寫不在內",
        "targets": [],
    }
    for target in targets:
        refs = []
        x = ida_xref.get_first_dref_to(target)
        while x != idc.BADADDR:
            refs.append(x)
            x = ida_xref.get_next_dref_to(target, x)
        entries = []
        for ea in sorted(set(refs)):
            kinds = []
            xb = ida_xref.xrefblk_t()
            ok = xb.first_to(target, ida_xref.XREF_DATA)
            while ok:
                if xb.frm == ea:
                    kinds.append(XREF_TYPES.get(xb.type, str(xb.type)))
                ok = xb.next_to()
            entry = insn(ea)
            entry["xref_types"] = kinds
            entry["function"] = func_info(ea)
            entries.append(entry)
        report["targets"].append({"address": hex(target), "references": entries})
    with open(os.environ["FD2_IDA_OUTPUT"], "w", encoding="utf-8") as out:
        json.dump(report, out, ensure_ascii=False, indent=2)
        out.write("\n")
    idc.qexit(0)


if __name__ == "__main__":
    main()
