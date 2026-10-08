#!/usr/bin/env python3
"""重跑五個已匹配 C 切片；compile 在 DOS 工具映像，link 在 FD2 工具映像。

這些 raw 函式的原版用途分類仍未知；機器碼匹配不提升其玩法語意。
全檔驗收契約：docs/goal/118-goal-fd2-matching-decompilation-20261008.md。
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess


COMPILER_SHA = "c3666de94f6fa6800f452dae8acf45505ecdb62f0ade2cc27cc86c2d9e8e2b6b"
INPUTS = {
    "WCC386.EXE": COMPILER_SHA,
    "W32RUN.EXE": "d9028582ce7d2d7b0990d677e1947c10e0b0deaefb32d78319fbd2cea603cfcf",
    "DOS4GW.EXE": "dd9f4f342533f99570475b62e53231a468de2e3d83e4fc31e80c27d3a7d6b49c",
}
CASES = (("H13", 0x35EBE, True), ("H14", 0x35ED2, True),
         ("S58", 0x3615E, False), ("S5C", 0x3616E, False), ("G54", 0x368FA, False))
BINDINGS = {"dword_53AD5": 0x53AD5, "off_52758": 0x52758,
            "off_5275C": 0x5275C, "dword_52754": 0x52754, "__CHK": 0x36CD7}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def compile_stage(args):
    for name, expected in INPUTS.items():
        if sha(args.compiler / name) != expected:
            raise ValueError(f"compiler 輸入 {name} 的雜湊不符")
    args.output.mkdir(parents=True, exist_ok=True)
    if args.output.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄擁有權不符")
    source = args.repo / "tools/fd2_matching_raw_helpers.c"
    (args.output / "RAW.C").write_bytes(source.read_text(encoding="utf-8").encode("ascii", "ignore"))
    lines = ["[sdl]", "output=surface", "[dosbox]", "memsize=31", "[cpu]", "core=normal",
             "cycles=max 90% limit 100000", "[mixer]", "nosound=true", "[midi]", "mpu401=none",
             "mididevice=none", "[autoexec]", f"mount c {args.output}", f"mount r {args.compiler}",
             "c:", "set PATH=R:\\;Z:\\"]
    trials = []
    for macro, address, check in CASES:
        for opt in ("-os", "-ot"):
            stem = macro + opt[-1].upper()
            flags = ["-mf", "-3s", opt] + ([] if check else ["-s"]) + ["-d" + macro]
            lines.append("R:\\WCC386.EXE " + " ".join(flags) + f" -fo={stem}.OBJ RAW.C > {stem}.TXT")
            trials.append({"stem": stem, "address": hex(address), "flags": flags})
    lines.append("exit")
    config = args.output / "RUN.CONF"
    config.write_text("\n".join(lines) + "\n", encoding="ascii")
    runner = subprocess.run(["dosbox", "-conf", str(config), "-noconsole"],
                            capture_output=True, text=True, timeout=90)
    (args.output / "runner.log").write_text(runner.stdout + runner.stderr, encoding="utf-8")
    if runner.returncode:
        raise ValueError("DOS compiler 執行器失敗")
    for trial in trials:
        obj = args.output / (trial["stem"] + ".OBJ")
        log = (args.output / (trial["stem"] + ".TXT")).read_text(encoding="cp437")
        trial["compiled"] = obj.is_file() and bool(re.search(r"\b0 errors\b", log)) and "Error!" not in log
        if not trial["compiled"]:
            raise ValueError(f"{trial['stem']} 未成功編譯：{log}")
        trial["object_sha256"] = sha(obj)
    report = {"schema_version": 1, "compiler_sha256": COMPILER_SHA,
              "source_sha256": sha(source), "driver_sha256": sha(Path(__file__)), "trials": trials}
    (args.output / "compile-report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print("十個 C 候選已編譯，輸入及物件雜湊已保存。", flush=True)


def link_stage(args):
    from fd2_matching_pilot import checked, linked_text
    from fd2_matching_legacy_verify import OBJCONV_SHA256, text_size
    if sha(args.converter) != OBJCONV_SHA256:
        raise ValueError("物件轉換器雜湊不符")
    image = json.loads(args.evidence.read_text(encoding="utf-8"))
    reference = next(x for x in json.loads((args.repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8"))["files"]
                     if x["file"] == "FD2.EXE")
    original = args.original.read_bytes()
    identity = {"file": "FD2.EXE", "size": len(original), "md5": hashlib.md5(original).hexdigest(),
                "sha256": hashlib.sha256(original).hexdigest()}
    if identity != reference or image["input"] != reference:
        raise ValueError("原版或 IDA 證據版本不符")
    report = json.loads((args.objects / "compile-report.json").read_text(encoding="utf-8"))
    if report["source_sha256"] != sha(args.repo / "tools/fd2_matching_raw_helpers.c"):
        raise ValueError("C 來源與編譯收據不符")
    args.output.mkdir(parents=True, exist_ok=True)
    results = []
    for trial in report["trials"]:
        obj = args.objects / (trial["stem"] + ".OBJ")
        if sha(obj) != trial["object_sha256"]:
            raise ValueError("物件與 compiler 收據不符")
        out = args.output / trial["stem"]
        out.mkdir(exist_ok=True)
        coff, dis = out / "candidate.cof", out / "candidate.dis"
        checked([str(args.converter), "-fcoff", str(obj), str(coff)])
        checked(["wdis", "-l=" + str(dis), str(obj)])
        symbols = re.findall(r"^0000\s+([_A-Za-z][_A-Za-z0-9]*):", dis.read_text(), re.M)
        if len(symbols) != 1:
            raise ValueError("候選須有唯一函式")
        address, size = int(trial["address"], 16), text_size(coff)
        function = next(f for f in image["functions"] if int(f["inventory"]["start"], 16) == address)
        instructions = [i for chunk in function["chunks"] for i in chunk["instructions"]]
        expected = bytearray()
        for instruction in instructions:
            offset = int(instruction["file_offset"], 16)
            raw = bytes.fromhex(instruction["file_bytes"])
            if original[offset:offset + len(raw)] != raw:
                raise ValueError("原始指令與目前原檔不符")
            expected.extend(bytes.fromhex(instruction["loaded_bytes"]))
        if len(expected) != function["inventory"]["size"]:
            raise ValueError("此切片不是完整連續區間")
        script = out / "link.ld"
        script.write_text("\n".join(f"{key} = {value:#x};" for key, value in BINDINGS.items()) +
                          f"\nSECTIONS {{ .text {address:#x} : SUBALIGN(1) {{ *(_TEXT) }} "
                          "/DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(_BSS) *(.depend) *(.reloc) } }\n", encoding="ascii")
        exe = out / "linked.exe"
        checked(["ld", "-mi386pe", "--image-base", "0", "--section-alignment", "1", "--file-alignment", "1",
                 "--no-insert-timestamp", "-T", str(script), "-e", symbols[0], "-o", str(exe), str(coff)])
        code = linked_text(exe, address, size)
        (out / "candidate.bin").write_bytes(code)
        results.append({**trial, "candidate_size": size, "original_size": len(expected),
                        "exact_interval_bytes": code == expected, "code_sha256": hashlib.sha256(code).hexdigest(),
                        "original_classification": function["inventory"]["classification"],
                        "source_kind": "C candidate; machine match does not infer original authored language"})
    result = {"schema_version": 1, "input": reference, "source_sha256": report["source_sha256"],
              "compiler_sha256": COMPILER_SHA, "converter_sha256": OBJCONV_SHA256,
              "driver_sha256": sha(Path(__file__)), "evidence_sha256": sha(args.evidence),
              "trials": results, "matched_addresses": sorted({r["address"] for r in results if r["exact_interval_bytes"]})}
    (args.output / "report.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("逐位元組匹配：" + ", ".join(result["matched_addresses"]), flush=True)


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
