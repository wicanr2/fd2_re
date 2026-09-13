"""匯出 FD2 狀態扣血致死與死亡效果收集的 IDA 9.4 主證據。

只可在使用者授權的 IDA Pro Docker 環境執行。輸出保留原始函式名、LE linear
位址、指令 bytes、operand 與直接 xref；高階語意留給受版控規格分級，不在 IDA
資料庫內做破壞性改名。
"""

import hashlib
import os
import traceback

import ida_auto
import ida_bytes
import ida_funcs
import ida_xref
import idaapi
import idc


EXPECTED_SIZE = 357074
EXPECTED_MD5 = "b97caf2239a27a896069d03549d96e1e"
EXPECTED_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"

TARGETS = (0x1A866, 0x1DB65, 0x1B6B7, 0x1AA1D)
RANGES = (
    ("sub_1A866 +0x25 HP writer、death presenter 與 chapter hook", 0x1A874, 0x1A957),
    ("sub_1A866 六個狀態倒數；inactive gate 在扣減前", 0x1A959, 0x1AA1D),
    ("sub_1DB65 無可見死亡單位時的 +5 writer", 0x1DB76, 0x1DC67),
    ("sub_1DB65 有可見死亡演出後的 +5 writer", 0x1DCDE, 0x1DD52),
    ("sub_1B6B7 死亡效果收集 gate", 0x1B6C4, 0x1B722),
    ("selector 1 phase caller", 0x1A4C5, 0x1A4E6),
    ("selector 0 phase caller", 0x1A552, 0x1A573),
    ("selector 2 phase caller", 0x1A78B, 0x1A7A9),
    ("AI mode 11 行動結算收集順序", 0x15610, 0x15670),
    ("玩家行動結算收集順序", 0x18FBD, 0x19020),
    ("一般 command 行動結算收集順序", 0x1D470, 0x1D4CB),
    ("AI item 行動結算收集順序", 0x21040, 0x21082),
)


def function_label(address):
    function = ida_funcs.get_func(address)
    if function is None:
        return "<none>"
    return (
        f"{function.start_ea:#x}..{function.end_ea:#x} "
        f"raw_name={idc.get_func_name(function.start_ea)}"
    )


def instruction_line(address):
    size = idc.get_item_size(address)
    raw = ida_bytes.get_bytes(address, size) or b""
    disassembly = idc.generate_disasm_line(address, 0) or ""
    return f"{address:#x}  {raw.hex(' '):<35}  {disassembly}"


def dump_range(out, label, start, end):
    out.write(f"\n=== {label} [{start:#x}..{end:#x}) ===\n")
    address = start
    while address != idaapi.BADADDR and address < end:
        if ida_bytes.is_code(ida_bytes.get_full_flags(address)):
            out.write(instruction_line(address) + "\n")
        address = idc.next_head(address, end)


def dump_xrefs(out, target):
    out.write(f"\n=== xrefs to {target:#x}; function={function_label(target)} ===\n")
    source = ida_xref.get_first_cref_to(target)
    while source != idaapi.BADADDR:
        out.write(
            f"{source:#x} caller={function_label(source)} "
            f"instruction={idc.generate_disasm_line(source, 0) or ''}\n"
        )
        source = ida_xref.get_next_cref_to(target, source)


def main():
    ida_auto.auto_wait()
    input_path = idc.get_input_file_path()
    with open(input_path, "rb") as source:
        raw = source.read()
    md5 = hashlib.md5(raw).hexdigest()
    sha256 = hashlib.sha256(raw).hexdigest()
    if len(raw) != EXPECTED_SIZE or md5 != EXPECTED_MD5 or sha256 != EXPECTED_SHA256:
        raise RuntimeError(
            f"unexpected FD2.EXE identity: size={len(raw)} md5={md5} sha256={sha256}"
        )

    output = os.environ.get("FD2_IDA_OUTPUT", "/out/fd2_status_death_ida.txt")
    with open(output, "w", encoding="utf-8") as out:
        out.write("FD2 狀態扣血致死／死亡效果收集：IDA Pro 9.4 非破壞性匯出\n")
        out.write(f"input_file={os.path.basename(input_path)}\n")
        out.write(f"input_size={len(raw)}\n")
        out.write(f"input_md5={md5}\n")
        out.write(f"input_sha256={sha256}\n")
        out.write(f"ida_version={idaapi.get_kernel_version()}\n")
        out.write(f"imagebase={idaapi.get_imagebase():#x}\n")
        out.write("address_space=DOS LE loader linear address\n")
        out.write("inference_contract=raw_name/address/bytes preserved; semantics graded in doc 110\n")
        for target in TARGETS:
            dump_xrefs(out, target)
        for label, start, end in RANGES:
            dump_range(out, label, start, end)
    idc.qexit(0)


if __name__ == "__main__":
    try:
        main()
    except Exception:
        with open("/out/fd2_status_death_ida.error.txt", "w", encoding="utf-8") as out:
            out.write(traceback.format_exc())
        idc.qexit(1)
