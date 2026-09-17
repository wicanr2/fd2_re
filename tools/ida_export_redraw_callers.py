"""匯出 FD2 戰場重繪 `0x11CAC` 的呼叫點與 HUD 閘門寫入端（#40 盤點用）。

本腳本只供使用者授權的 IDA Pro Docker 流程使用（`tools/ida.sh`）。輸出只寫到
``FD2_IDA_OUTPUT``。內容是位址導向的純文字，不含原版位元組以外的推測名稱：

- `0x11CAC` 的每個直接呼叫點：所屬函式邊界、前三條指令（推入參數在這裡）。
- 閘 A `[0x51AAB]`、閘 B `[0x51AAC]`、`[0x51A83]` 的每個資料參照：IDA 標的
  讀／寫型別、所屬函式與指令。
- `0x1ACF3`（HUD 小窗）與 `0x1AD2A`（anchor）的直接呼叫點。

語意分類不在這裡做；分類與推論等級寫進 `56`／`58`。
"""

import hashlib
import os

import ida_auto
import ida_funcs
import ida_nalt
import ida_xref
import idaapi
import idc

REDRAW = 0x11CAC
HUD_TARGETS = (0x1ACF3, 0x1AD2A)
GLOBALS = ((0x51AAB, "gate_a"), (0x51AAC, "gate_b"), (0x51A83, "range_mode"))


def function_bounds(address):
    function = ida_funcs.get_func(address)
    if function is None:
        return "<none>"
    return f"{function.start_ea:#x}..{function.end_ea:#x}"


def code_xrefs_to(address):
    current = ida_xref.get_first_cref_to(address)
    while current != idaapi.BADADDR:
        yield current
        current = ida_xref.get_next_cref_to(address, current)


def data_xrefs_to(address):
    xref = ida_xref.xrefblk_t()
    ok = xref.first_to(address, ida_xref.XREF_DATA)
    while ok:
        yield xref.frm, xref.type
        ok = xref.next_to()


def previous_lines(address, count):
    lines = []
    current = address
    for _ in range(count):
        current = idc.prev_head(current, 0)
        if current == idaapi.BADADDR:
            break
        lines.append(f"{current:#x} {idc.generate_disasm_line(current, 0) or ''}")
    return list(reversed(lines))


def input_sha256():
    path = ida_nalt.get_input_file_path()
    try:
        with open(path, "rb") as handle:
            return hashlib.sha256(handle.read()).hexdigest()
    except OSError:
        return "<unreadable>"


def main():
    output = os.environ.get("FD2_IDA_OUTPUT", "/work/fd2-redraw-callers.txt")
    ida_auto.auto_wait()
    xref_names = {
        ida_xref.dr_R: "read",
        ida_xref.dr_W: "write",
        ida_xref.dr_O: "offset",
    }
    with open(output, "w", encoding="utf-8") as out:
        out.write("# FD2.EXE IDA Pro 9.4 export: 0x11CAC callers and HUD gate xrefs\n")
        out.write(f"# input_sha256={input_sha256()} address_space=IDA LE linear\n")
        calls = list(code_xrefs_to(REDRAW))
        out.write(f"=== redraw {REDRAW:#x} direct calls={len(calls)} ===\n")
        for site in calls:
            line = idc.generate_disasm_line(site, 0) or ""
            out.write(f"call={site:#x} function={function_bounds(site)} instruction={line}\n")
            for previous in previous_lines(site, 3):
                out.write(f"    {previous}\n")
        for target in HUD_TARGETS:
            sites = list(code_xrefs_to(target))
            out.write(f"=== hud {target:#x} direct calls={len(sites)} ===\n")
            for site in sites:
                line = idc.generate_disasm_line(site, 0) or ""
                out.write(f"call={site:#x} function={function_bounds(site)} instruction={line}\n")
        for address, label in GLOBALS:
            refs = list(data_xrefs_to(address))
            out.write(f"=== global {address:#x} {label} data xrefs={len(refs)} ===\n")
            for site, kind in refs:
                line = idc.generate_disasm_line(site, 0) or ""
                name = xref_names.get(kind, f"type{kind}")
                out.write(f"xref={site:#x} kind={name} function={function_bounds(site)} instruction={line}\n")
    idc.qexit(0)


if __name__ == "__main__":
    main()
