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


CASES = (("CLEAR3", 0x134E4), ("REDRAW", 0x127A9), ("CYCLE", 0x1F525), ("MASK", 0x146A7))
BINDINGS = {"dword_53A45": 0x53A45, "dword_53BEB": 0x53BEB,
            "dword_53AC1": 0x53AC1, "dword_53A51": 0x53A51, "__CHK": 0x36CD7,
            "sub_375B2": 0x375B2, "sub_3453E": 0x3453E, "sub_127E0": 0x127E0,
            "sub_129EC": 0x129EC, "sub_11D40": 0x11D40}
SOURCE_DATE_EPOCH = 315532800  # DOS 可表示的 1980-01-01 UTC。


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def compile_stage(args):
    for name, digest in INPUTS.items():
        if sha(args.compiler / name) != digest:
            raise ValueError("compiler 組件雜湊不符")
    args.output.mkdir(parents=True, exist_ok=True)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄擁有權不符")
    source = args.repo / "tools/fd2_matching_game_slices.c"
    dos_source = args.output / "GAME.C"
    dos_source.write_bytes(source.read_text(encoding="utf-8").encode("ascii", "ignore"))
    os.utime(dos_source, (SOURCE_DATE_EPOCH, SOURCE_DATE_EPOCH))
    lines = ["[sdl]", "output=surface", "[dosbox]", "memsize=31", "[cpu]", "core=normal",
             "cycles=max 90% limit 100000", "[mixer]", "nosound=true", "[midi]", "mpu401=none",
             "mididevice=none", "[autoexec]", f"mount c {args.output}", f"mount r {args.compiler}",
             "c:", "set PATH=R:\\;Z:\\"]
    trials = []
    for index, ((macro, address), cpu, opt) in enumerate(itertools.product(CASES, ("3s", "4s"), ("-os", "-ot"))):
        stem = f"G{index:02}"
        flags = ["-mf", "-" + cpu, opt, "-d" + macro]
        lines.append("R:\\WCC386.EXE " + " ".join(flags) + f" -fo={stem}.OBJ GAME.C > {stem}.TXT")
        trials.append({"stem": stem, "address": hex(address), "flags": flags})
    lines.append("exit")
    config = args.output / "RUN.CONF"
    config.write_text("\n".join(lines) + "\n", encoding="ascii")
    result = subprocess.run(["dosbox", "-conf", str(config), "-noconsole"], capture_output=True, text=True, timeout=90)
    (args.output / "runner.log").write_text(result.stdout + result.stderr, encoding="utf-8")
    if result.returncode:
        raise ValueError("compiler 執行器失敗")
    for trial in trials:
        obj = args.output / (trial["stem"] + ".OBJ")
        text = (args.output / (trial["stem"] + ".TXT")).read_text(encoding="cp437")
        if not obj.is_file() or not re.search(r"\b0 errors\b", text) or "Error!" in text:
            raise ValueError("compiler 未成功產生物件")
        trial["object_sha256"] = sha(obj)
    report = {"schema_version": 1, "source_sha256": sha(source), "compiler_inputs": INPUTS,
              "source_date_epoch": SOURCE_DATE_EPOCH,
              "driver_sha256": sha(Path(__file__)), "trials": trials}
    (args.output / "compile-report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print("十六個主程式 C 候選已編譯。", flush=True)


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
    if compile_report["source_sha256"] != sha(args.repo / "tools/fd2_matching_game_slices.c"):
        raise ValueError("C 來源與編譯收據不符")
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
    report = {"schema_version": 1, "input": reference, "source_sha256": compile_report["source_sha256"],
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
    args = parser.parse_args()
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    compile_stage(args) if args.stage == "compile" else link_stage(args)


if __name__ == "__main__":
    main()
