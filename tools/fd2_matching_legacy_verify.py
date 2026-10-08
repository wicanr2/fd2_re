#!/usr/bin/env python3
"""連結歷史 Watcom OMF 候選並逐位元組比對固定版 FD2 函式。

OMF32 經有雜湊的 objconv 轉成 COFF32，再由 GNU ld 解析明示原版位址。
不遮蔽差異；同一個函式大小也不能作為匹配證明。
"""

import argparse
import hashlib
import json
from pathlib import Path
import re
import struct

from fd2_matching_pilot import LINK_SYMBOLS, checked, linked_text, verify_bindings


ADDRESSES = {"R": 0x4E893, "P": 0x4E893, "O": 0x145CD, "I": 0x1B8E7}
OBJCONV_SHA256 = "5dc2d30a8bba85cff9717408f1a7b16128dbb835cebbd46f40268417ea32b6c1"


def text_size(path):
    data = path.read_bytes()
    machine, count = struct.unpack_from("<HH", data)
    if machine != 0x14C or struct.unpack_from("<H", data, 16)[0]:
        raise ValueError("轉換結果不是 i386 COFF")
    code_size = None
    for index in range(count):
        start = 20 + index * 40
        name = data[start:start + 8].rstrip(b"\0")
        size = struct.unpack_from("<I", data, start + 16)[0]
        if name == b"_TEXT" and code_size is None:
            code_size = size
        elif name in (b"CONST", b"CONST2", b"_DATA", b"_BSS") and size == 0:
            pass
        else:
            raise ValueError(f"未審查的 COFF 區段 {name!r} 或資料")
    if code_size is None:
        raise ValueError("缺少唯一 _TEXT")
    return code_size


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--objects", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--converter", type=Path, required=True)
    parser.add_argument("--evidence", type=Path, required=True)
    args = parser.parse_args()
    if hashlib.sha256(args.converter.read_bytes()).hexdigest() != OBJCONV_SHA256:
        raise ValueError("物件轉換器雜湊與已確認建置不符")
    evidence = json.loads(args.evidence.read_text(encoding="utf-8"))
    verify_bindings(evidence)
    compile_report = json.loads((args.objects / "compile-report.json").read_text())
    args.output.mkdir(parents=True, exist_ok=True)
    trials = []
    for trial in compile_report["trials"]:
        if not trial["compiled"]:
            continue
        original_obj = args.objects / (trial["stem"] + ".OBJ")
        if hashlib.sha256(original_obj.read_bytes()).hexdigest() != trial["object_sha256"]:
            raise ValueError("原始 compiler 物件與收據不符")
        out = args.output / trial["stem"]
        out.mkdir(exist_ok=True)
        omf_dis = out / "original-omf.dis"
        checked(["wdis", "-l=" + str(omf_dis), str(original_obj)])
        assembly = omf_dis.read_text(encoding="utf-8")
        routines = re.findall(r"Routine Size: (\d+) bytes", assembly)
        symbols = re.findall(r"^0000\s+([_A-Za-z][_A-Za-z0-9]*):", assembly, re.M)
        if len(routines) != 1 or len(symbols) != 1 or "No disassembly errors" not in assembly:
            raise ValueError("OMF 並非唯一可完整讀取的函式")
        coff = out / "candidate.cof"
        conversion = checked([str(args.converter), "-fcoff", str(original_obj), str(coff)])
        size = text_size(coff)
        if size != int(routines[0]):
            raise ValueError("轉換器改變了原始 OMF 程式碼大小")
        address = ADDRESSES[trial["case"]]
        script = out / "link.ld"
        definitions = "\n".join(f"{key} = {value:#x};" for key, value in LINK_SYMBOLS.items())
        # 名稱是候選 compiler 的裝飾形式；原版的定位仍採 IDA 線性位址。
        # 每種 memmove 名稱都綁至原始 call 指令證實的同一個 callee。
        definitions += "\nmemmove = 0x373c4; memmove_ = 0x373c4;"
        script.write_text(definitions + f"\nSECTIONS {{ .text {address:#x} : SUBALIGN(1) {{ *(_TEXT) }} "
                          "/DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(_BSS) *(.depend) *(.reloc) } }\n", encoding="ascii")
        exe = out / "linked.exe"
        checked(["ld", "-mi386pe", "--image-base", "0", "--section-alignment", "1", "--file-alignment", "1",
                 "--no-insert-timestamp", "-T", str(script), "-e", symbols[0], "-o", str(exe), str(coff)])
        code = linked_text(exe, address, size)
        (out / "candidate.bin").write_bytes(code)
        expected = bytes.fromhex(next(f["loaded_bytes"] for f in evidence["functions"]
                                     if int(f["inventory"]["start"], 16) == address))
        first_difference = next((index for index, pair in enumerate(zip(code, expected))
                                 if pair[0] != pair[1]), min(len(code), len(expected)))
        trials.append({**trial, "address": hex(address), "original_size": len(expected), "candidate_size": size,
                       "exact_interval_bytes": code == expected, "first_difference_offset": None if code == expected else first_difference,
                       "candidate_code_sha256": hashlib.sha256(code).hexdigest(),
                       "conversion_diagnostics": conversion.strip(), "candidate_symbol": symbols[0]})
    report = {"schema_version": 1, "input": evidence["input"], "compile_report": compile_report,
              "converter_sha256": OBJCONV_SHA256,
              "driver_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              "comparison": "all linked code bytes at original linear address; no masking or instruction patches",
              "trials": trials, "matched_addresses": sorted({t["address"] for t in trials if t["exact_interval_bytes"]})}
    (args.output / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"compiler": compile_report["compiler_sha256"], "linked": len(trials),
                      "matched_addresses": report["matched_addresses"]}), flush=True)


if __name__ == "__main__":
    main()
