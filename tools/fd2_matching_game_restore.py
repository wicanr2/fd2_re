#!/usr/bin/env python3
"""編譯及驗證主程式區的 C 候選，所有完整區間逐位元組比較。"""

import argparse
import hashlib
import itertools
import json
import os
from pathlib import Path
import re
import subprocess

from fd2_matching_c_restore import INPUTS


CASES = (("CLEAR3", 0x134E4), ("REDRAW", 0x127A9), ("CYCLE", 0x1F525), ("MASK", 0x146A7),
         ("ROWCOPY", 0x11EB0), ("RELEASE", 0x15E71), ("CONDITIONAL_FREE", 0x1A7F1),
         ("CACHE_INIT", 0x1D4CB), ("CONDITIONAL_INIT", 0x1A7BD), ("INVENTORY", 0x1B8E7),
         ("SET_BIT7", 0x13512), ("CLEAR_BIT7", 0x13536), ("SLOT_BYTE", 0x1B722),
         ("SET_BYTE5", 0x32975), ("GET_BIT0", 0x3453E), ("RANGE_LOW_BYTE", 0x3419C),
         ("COPY_WORDS", 0x25089), ("COMPARE_VALUE", 0x2EF8F))
RECORD_CASES = (("CURSOR_Y_DEC", 0x11B48), ("CURSOR_Y_INC", 0x11B9B),
                ("CURSOR_X_INC", 0x11BFA), ("CURSOR_X_DEC", 0x11C59),
                ("FIND_RECORD", 0x12C0D), ("SUM_FIELD", 0x15DA2),
                ("DERIVED_WORDS", 0x1145A), ("BLIT_CELL", 0x126F7),
                ("READ_INFO", 0x12E38), ("CONDITIONAL_INFO", 0x13A44))
BINDINGS = {"dword_53A45": 0x53A45, "dword_53BEB": 0x53BEB,
            "dword_53AC1": 0x53AC1, "dword_53A51": 0x53A51, "__CHK": 0x36CD7,
            "sub_375B2": 0x375B2, "sub_3453E": 0x3453E, "sub_127E0": 0x127E0,
            "sub_129EC": 0x129EC, "sub_11D40": 0x11D40, "memmove": 0x373C4,
            "sub_4E92C": 0x4E92C, "_free": 0x37416, "byte_53AF9": 0x53AF9,
            "dword_53B0F": 0x53B0F, "dword_53B13": 0x53B13,
            "aFdotherDat": 0x51A4D, "sub_111BA": 0x111BA,
            "dword_53BF7": 0x53BF7, "dword_53BFB": 0x53BFB}
BINDINGS.update({"dword_53AB1": 0x53AB1, "dword_53AB5": 0x53AB5,
                 "dword_53AB9": 0x53AB9, "dword_53ABD": 0x53ABD,
                 "dword_53AA9": 0x53AA9, "dword_53AAD": 0x53AAD,
                 "dword_53AC5": 0x53AC5, "dword_51A83": 0x51A83,
                 "dword_51A87": 0x51A87, "dword_51A8B": 0x51A8B,
                 "dword_51A8F": 0x51A8F, "dword_53A49": 0x53A49,
                 "dword_53A4D": 0x53A4D, "dword_53A55": 0x53A55,
                 "dword_53A69": 0x53A69, "sub_11CAC": 0x11CAC,
                 "sub_4E56C": 0x4E56C, "sub_4DEDA": 0x4DEDA,
                 "_sub_4E56C": 0x4E56C, "_sub_4DEDA": 0x4DEDA,
                 "sub_12E38": 0x12E38})
SOURCE_DATE_EPOCH = 315532800  # DOS 可表示的 1980-01-01 UTC。
SOURCES = {
    "game": ("tools/fd2_matching_game_slices.c", tuple(macro for macro, _ in CASES)),
    "record_layout": ("tools/fd2_matching_record_layout.c", ("SET_BIT7", "SLOT_BYTE", "COPY_WORDS")),
    "game_records": ("tools/fd2_matching_game_records.c", tuple(macro for macro, _ in RECORD_CASES)),
}
COSTS = {"balanced": (), "space": ("-os",), "speed": ("-ot",)}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def compile_stage(args):
    for name, digest in INPUTS.items():
        if sha(args.compiler / name) != digest:
            raise ValueError("compiler 組件雜湊不符")
    args.output.mkdir(parents=True, exist_ok=True)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄擁有權不符")
    source_path, source_cases = SOURCES[args.source_key]
    source = args.repo / source_path
    selected_cases = args.cases or source_cases or [macro for macro, _ in CASES]
    if source_cases and not set(selected_cases).issubset(source_cases):
        raise ValueError("所選來源未提供這些候選")
    cases = [(macro, address) for macro, address in (*CASES, *RECORD_CASES) if macro in selected_cases]
    dos_source = args.output / "GAME.C"
    dos_source.write_bytes(source.read_text(encoding="utf-8").encode("ascii", "ignore"))
    os.utime(dos_source, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
    lines = ["[sdl]", "output=surface", "[dosbox]", "memsize=31", "[cpu]", "core=normal",
             "cycles=max 90% limit 100000", "[mixer]", "nosound=true", "[midi]", "mpu401=none",
             "mididevice=none", "[autoexec]", f"mount c {args.output}", f"mount r {args.compiler}",
             "c:", "set PATH=R:\\;Z:\\"]
    commands = []
    trials = []
    for index, ((macro, address), cpu, cost) in enumerate(itertools.product(cases, args.cpus, args.costs)):
        stem = f"G{index:02}"
        flags = ["-mf", "-" + cpu, *COSTS[cost], "-d" + macro]
        commands.append("R:\\WCC386.EXE " + " ".join(flags) + f" -fo={stem}.OBJ GAME.C > {stem}.TXT")
        trials.append({"stem": stem, "address": hex(address), "flags": flags})
    compile_batch = args.output / "BUILD.BAT"
    compile_batch.write_text("\r\n".join(commands) + "\r\n", encoding="ascii")
    lines.extend(["call BUILD.BAT", "exit"])
    config = args.output / "RUN.CONF"
    config.write_text("\n".join(lines) + "\n", encoding="ascii")
    result = subprocess.run(["dosbox", "-conf", str(config), "-noconsole"], capture_output=True, text=True, timeout=90)
    (args.output / "runner.log").write_text(result.stdout + result.stderr, encoding="utf-8")
    if result.returncode or "Exit to error:" in result.stdout + result.stderr:
        raise ValueError("compiler 執行器失敗")
    for trial in trials:
        obj = args.output / (trial["stem"] + ".OBJ")
        text = (args.output / (trial["stem"] + ".TXT")).read_text(encoding="cp437")
        if not obj.is_file() or not re.search(r"\b0 errors\b", text) or "Error!" in text:
            raise ValueError(f"{trial['stem']}: compiler 未成功產生物件")
        trial["object_sha256"] = sha(obj)
    report = {"schema_version": 1, "source_key": args.source_key,
              "source_path": source_path, "source_sha256": sha(source), "compiler_inputs": INPUTS,
              "source_date_epoch": SOURCE_DATE_EPOCH,
              "driver_sha256": sha(Path(__file__)), "trials": trials}
    (args.output / "compile-report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(f"{len(trials)} 個主程式 C 候選已編譯。", flush=True)


def link_stage(args):
    from fd2_matching_pilot import checked, linked_text
    from fd2_matching_legacy_verify import OBJCONV_SHA256, text_size
    if sha(args.converter) != OBJCONV_SHA256:
        raise ValueError("物件轉換器雜湊不符")
    image = json.loads(args.evidence.read_text(encoding="utf-8"))
    original = args.original.read_bytes()
    reference = next(x for x in json.loads((args.repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8"))["files"]
                     if x["file"] == "FD2.EXE")
    identity = {"file": "FD2.EXE", "size": len(original), "md5": hashlib.md5(original).hexdigest(), "sha256": hashlib.sha256(original).hexdigest()}
    if identity != reference or image["input"] != reference:
        raise ValueError("原檔或 IDA 版本不符")
    compile_report = json.loads((args.objects / "compile-report.json").read_text(encoding="utf-8"))
    source_key = compile_report.get("source_key", "game")
    source_path = SOURCES[source_key][0]
    if compile_report.get("source_path", source_path) != source_path:
        raise ValueError("C 來源路徑未登記")
    if compile_report["source_sha256"] != sha(args.repo / source_path):
        raise ValueError("C 來源與編譯收據不符")
    if (args.objects / "GAME.C").read_bytes() != (args.repo / source_path).read_text(encoding="utf-8").encode("ascii", "ignore"):
        raise ValueError("實際 DOS C 來源與登記來源不符")
    args.output.mkdir(parents=True, exist_ok=True)
    results = []
    for trial in compile_report["trials"]:
        obj = args.objects / (trial["stem"] + ".OBJ")
        if sha(obj) != trial["object_sha256"]:
            raise ValueError("原始 compiler 物件與收據不符")
        out = args.output / trial["stem"]
        out.mkdir(exist_ok=True)
        coff, dis = out / "candidate.cof", out / "candidate.dis"
        checked([str(args.converter), "-fcoff", str(obj), str(coff)])
        checked(["wdis", "-l=" + str(dis), str(obj)])
        symbols = re.findall(r"^0000\s+([_A-Za-z][_A-Za-z0-9]*):", dis.read_text(), re.M)
        if len(symbols) != 1:
            raise ValueError("每個候選必須只有一個函式")
        size, address = text_size(coff), int(trial["address"], 16)
        fn = next(f for f in image["functions"] if int(f["inventory"]["start"], 16) == address)
        instructions = [i for c in fn["chunks"] for i in c["instructions"]]
        expected = bytearray()
        for instruction in instructions:
            offset = int(instruction["file_offset"], 16)
            raw = bytes.fromhex(instruction["file_bytes"])
            if original[offset:offset + len(raw)] != raw:
                raise ValueError("原始指令版本不符")
            expected.extend(bytes.fromhex(instruction["loaded_bytes"]))
        if len(expected) != fn["inventory"]["size"]:
            raise ValueError("目標指令不涵蓋完整連續區間")
        script = out / "link.ld"
        definitions = "\n".join(f"{key} = {value:#x};" for key, value in BINDINGS.items())
        script.write_text(definitions + f"\nSECTIONS {{ .text {address:#x} : SUBALIGN(1) {{ *(_TEXT) }} "
                          "/DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(_BSS) *(.depend) *(.reloc) } }\n", encoding="ascii")
        exe = out / "linked.exe"
        checked(["ld", "-mi386pe", "--image-base", "0", "--section-alignment", "1", "--file-alignment", "1",
                 "--no-insert-timestamp", "-T", str(script), "-e", symbols[0], "-o", str(exe), str(coff)])
        code = linked_text(exe, address, size)
        (out / "candidate.bin").write_bytes(code)
        first = next((i for i, pair in enumerate(zip(code, expected)) if pair[0] != pair[1]), min(len(code), len(expected)))
        results.append({**trial, "candidate_size": size, "original_size": len(expected), "exact_interval_bytes": code == expected,
                        "first_difference_offset": None if code == expected else first, "code_sha256": hashlib.sha256(code).hexdigest(),
                        "original_classification": fn["inventory"]["classification"]})
    report = {"schema_version": 1, "input": reference, "source_key": source_key,
              "source_path": source_path, "source_sha256": compile_report["source_sha256"],
              "compile_driver_sha256": compile_report["driver_sha256"],
              "driver_sha256": sha(Path(__file__)), "evidence_sha256": sha(args.evidence), "trials": results,
              "matched_addresses": sorted({r["address"] for r in results if r["exact_interval_bytes"]})}
    (args.output / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"matched_addresses": report["matched_addresses"], "trials": len(results)}), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("stage", choices=("compile", "link"))
    parser.add_argument("--repo", type=Path, default=Path("/repo"))
    parser.add_argument("--compiler", type=Path, default=Path("/wc10a"))
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--objects", type=Path)
    parser.add_argument("--evidence", type=Path)
    parser.add_argument("--original", type=Path, default=Path("/input/FD2.EXE"))
    parser.add_argument("--converter", type=Path, default=Path("/work/objconv"))
    parser.add_argument("--source-key", choices=SOURCES, default="game")
    parser.add_argument("--cases", choices=[macro for macro, _ in (*CASES, *RECORD_CASES)], nargs="+")
    parser.add_argument("--cpus", choices=("3s", "4s", "5s"), nargs="+", default=("3s", "4s", "5s"))
    parser.add_argument("--costs", choices=COSTS, nargs="+", default=tuple(COSTS))
    args = parser.parse_args()
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    compile_stage(args) if args.stage == "compile" else link_stage(args)


if __name__ == "__main__":
    main()
