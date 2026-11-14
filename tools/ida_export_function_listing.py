"""輸出 FD2.EXE 指定函式的逐條指令與直接呼叫端（IDA 9.4 批次腳本）。

每個位址取所在 IDA 函式（含 tail chunk），逐條列出線性位址、原始 bytes 與指令
文字，再列出直接 code xref 呼叫端。只保留原始定位，不改名、不加語意；反編譯結果
不在這裡，避免把導覽層當證據。

用法（經 tools/ida.sh）：
    tools/ida.sh ida_export_function_listing.py <輸出 JSON> 1C81F [其他位址…]
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
import idautils
import idc


def insn(ea):
    size = idc.get_item_size(ea)
    return {
        "address": hex(ea),
        "bytes": ida_bytes.get_bytes(ea, size).hex() if size > 0 else "",
        "text": ida_lines.tag_remove(idc.generate_disasm_line(ea, 0) or ""),
    }


def callers(start):
    out = []
    x = ida_xref.get_first_cref_to(start)
    while x != idc.BADADDR:
        fn = ida_funcs.get_func(x)
        out.append({"from": hex(x), "function": hex(fn.start_ea) if fn else None})
        x = ida_xref.get_next_cref_to(start, x)
    return out


def main():
    ida_auto.auto_wait()
    input_path = idc.get_input_file_path()
    with open(input_path, "rb") as source:
        raw = source.read()
    report = {
        "tool": "tools/ida_export_function_listing.py",
        "ida_version": idaapi.get_kernel_version(),
        "address_space": "IDA linear address (LE object base)",
        "input_file": os.path.basename(input_path),
        "input_size": len(raw),
        "input_md5": hashlib.md5(raw).hexdigest(),
        "input_sha256": hashlib.sha256(raw).hexdigest(),
        "functions": [],
    }
    for text in os.environ.get("FD2_IDA_ADDRESSES", "").split():
        ea = int(text, 16)
        fn = ida_funcs.get_func(ea)
        if not fn:
            report["functions"].append({"requested": hex(ea), "error": "no IDA function"})
            continue
        chunks = [(c[0], c[1]) for c in idautils.Chunks(fn.start_ea)]
        lines = []
        for start, end in sorted(chunks):
            for head in idautils.Heads(start, end):
                if ida_bytes.is_code(ida_bytes.get_flags(head)):
                    lines.append(insn(head))
        report["functions"].append({
            "requested": hex(ea),
            "start": hex(fn.start_ea),
            "end": hex(fn.end_ea),
            "chunks": [[hex(s), hex(e)] for s, e in sorted(chunks)],
            "callers": callers(fn.start_ea),
            "instructions": lines,
        })
    with open(os.environ["FD2_IDA_OUTPUT"], "w", encoding="utf-8") as out:
        json.dump(report, out, ensure_ascii=False, indent=2)
        out.write("\n")
    idc.qexit(0)


if __name__ == "__main__":
    main()
