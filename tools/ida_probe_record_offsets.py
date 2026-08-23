"""Export non-destructive IDA references to selected displacement offsets.

This probe is intentionally semantic-free: it preserves each original
instruction, function boundary, address, and operand displacement. Run it only
inside the authorized IDA Docker image. Environment variables:

  FD2_IDA_RECORD_OFFSETS="22 23 24 25 26 27"
  FD2_IDA_OUTPUT=/work/fd2-record-offsets.txt
"""

import os

import ida_auto
import ida_funcs
import ida_ua
import idaapi
import idautils
import idc


def requested_offsets():
    return {
        int(value, 16)
        for value in os.environ.get(
            "FD2_IDA_RECORD_OFFSETS", "22 23 24 25 26 27"
        ).split()
    }


def main():
    ida_auto.auto_wait()
    wanted = requested_offsets()
    output = os.environ.get("FD2_IDA_OUTPUT", "/work/fd2-record-offsets.txt")
    rows = []
    for function_ea in idautils.Functions():
        function = ida_funcs.get_func(function_ea)
        if function is None:
            continue
        for ea in idautils.FuncItems(function.start_ea):
            instruction = ida_ua.insn_t()
            if ida_ua.decode_insn(instruction, ea) <= 0:
                continue
            matched = []
            for operand_index, operand in enumerate(instruction.ops):
                if operand.type == ida_ua.o_void:
                    break
                displacement = int(operand.addr)
                if operand.type == ida_ua.o_displ and displacement in wanted:
                    matched.append((operand_index, displacement))
            if matched:
                rows.append(
                    (
                        ea,
                        function.start_ea,
                        function.end_ea,
                        matched,
                        idc.generate_disasm_line(ea, 0) or "",
                    )
                )
    with open(output, "w", encoding="utf-8") as stream:
        stream.write("# raw displacement matches; no semantic names inferred\n")
        for ea, start, end, matched, disassembly in rows:
            operands = ",".join(
                f"op{operand_index}=+0x{displacement:x}"
                for operand_index, displacement in matched
            )
            stream.write(
                f"{ea:#x} function={start:#x}..{end:#x} "
                f"{operands} {disassembly}\n"
            )
    idc.qexit(0)


if __name__ == "__main__":
    main()
