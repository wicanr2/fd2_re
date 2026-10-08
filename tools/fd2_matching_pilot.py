#!/usr/bin/env python3
"""在 Docker 內重跑 FD2 三函式的純 C 編譯與完整位元組比較。

使用 Watcom COFF 輸出與 GNU ld，將外部符號解析到 IDA 已匯出的原版
線性位址。比較不遮蔽重定位、暫存器、指令或跳躍差異。
證據入口：docs/data/ida/fd2_matching_pilot_20261008.json。
"""

import argparse
import hashlib
import itertools
import json
from pathlib import Path
import struct
import subprocess


CASES = (
    ("rng", "FD2_PILOT_RNG", "sub_4E893", 0x4E893, ("-s",)),
    ("rng_promoted", "FD2_PILOT_RNG", "sub_4E893", 0x4E893, ("-s", "-dFD2_RNG_PROMOTED")),
    ("occupancy", "FD2_PILOT_OCCUPANCY", "_sub_145CD", 0x145CD, ()),
    ("inventory", "FD2_PILOT_INVENTORY", "_sub_1B8E7", 0x1B8E7, ()),
)
PROFILES = {
    "size": ("-os",),
    "speed": ("-ot",),
    "loop_size": ("-ol", "-os"),
    "loop_speed": ("-ol", "-ot"),
    "pipeline_size": ("-obmiler", "-os"),
    "pipeline_speed": ("-obmiler", "-ot"),
}
LINK_SYMBOLS = {
    "word_627B8": 0x627B8,
    "dword_53A45": 0x53A45,
    "dword_53BEB": 0x53BEB,
    "__CHK": 0x36CD7,
    "_sub_14625": 0x14625,
    "_memmove": 0x373C4,
}


def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def checked(command):
    result = subprocess.run(command, capture_output=True, text=True, timeout=15)
    if result.returncode:
        raise RuntimeError(f"{command!r}: {result.stdout}\n{result.stderr}")
    return result.stdout + result.stderr


def coff_text_size(path):
    data = path.read_bytes()
    machine, count = struct.unpack_from("<HH", data)
    optional_size = struct.unpack_from("<H", data, 16)[0]
    if machine != 0x14C or optional_size:
        raise ValueError("候選物件不是無 optional header 的 i386 COFF")
    symbol_offset, symbol_count = struct.unpack_from("<II", data, 8)
    strings_offset = symbol_offset + symbol_count * 18
    strings_size = struct.unpack_from("<I", data, strings_offset)[0]
    if strings_size < 4 or strings_offset + strings_size > len(data):
        raise ValueError("COFF 字串表遭截斷")
    sections = []
    for index in range(count):
        start = 20 + optional_size + 40 * index
        name = data[start:start + 8].rstrip(b"\0")
        if name.startswith(b"/"):
            index = int(name[1:])
            if not 4 <= index < strings_size:
                raise ValueError("COFF 區段名稱的字串表索引越界")
            end = data.find(b"\0", strings_offset + index, strings_offset + strings_size)
            if end < 0:
                raise ValueError("COFF 區段名稱沒有結尾")
            name = data[strings_offset + index:end]
        size = struct.unpack_from("<I", data, start + 16)[0]
        if name == b"_TEXT":
            sections.append(size)
        elif name in (b"CONST", b"CONST2", b"_DATA") and size:
            raise ValueError("三函式候選出現未審查的附加資料")
        elif name not in (b"CONST", b"CONST2", b"_DATA", b".depend"):
            raise ValueError(f"未審查的 COFF 區段 {name!r}")
    if len(sections) != 1:
        raise ValueError("COFF 必須有唯一 _TEXT")
    return sections[0]


def linked_text(path, expected_address, expected_size):
    data = path.read_bytes()
    pe = struct.unpack_from("<I", data, 0x3C)[0]
    if data[:2] != b"MZ" or data[pe:pe + 4] != b"PE\0\0":
        raise ValueError("連結器沒有產生有效 PE")
    machine, count = struct.unpack_from("<HH", data, pe + 4)
    optional_size = struct.unpack_from("<H", data, pe + 20)[0]
    optional = pe + 24
    magic = struct.unpack_from("<H", data, optional)[0]
    entry = struct.unpack_from("<I", data, optional + 16)[0]
    image_base = struct.unpack_from("<I", data, optional + 28)[0]
    if machine != 0x14C or magic != 0x10B or image_base or entry != expected_address:
        raise ValueError("PE 架構、image base 或函式進入位址不符")
    sections = []
    for index in range(count):
        start = optional + optional_size + 40 * index
        name = data[start:start + 8].rstrip(b"\0")
        size, address, raw_size, raw_offset = struct.unpack_from("<IIII", data, start + 8)
        if name == b".text":
            if address != expected_address or size != expected_size or raw_size < size:
                raise ValueError("連結後程式碼位址或完整長度不符")
            code = data[raw_offset:raw_offset + size]
            if len(code) != size:
                raise ValueError("PE 程式碼遭截斷")
            sections.append(code)
        else:
            raise ValueError(f"未審查的 PE 區段 {name!r}")
    if len(sections) != 1:
        raise ValueError("PE 必須有唯一 .text")
    return sections[0]


def verify_bindings(evidence):
    observed = {}
    aliases = {"sub_36CD7": "__CHK", "sub_14625": "_sub_14625", "memmove": "_memmove"}
    for function in evidence["functions"]:
        for instruction in function["instructions"]:
            code = bytes.fromhex(instruction["loaded_bytes"])
            text = instruction["instruction"]
            for name in ("word_627B8", "dword_53A45", "dword_53BEB"):
                if name in text:
                    value = int.from_bytes(code[-4:], "little")
                    if value != LINK_SYMBOLS[name]:
                        raise ValueError("外部符號位址與 IDA 原始指令不符")
                    observed[name] = value
            if code[0] == 0xE8 and len(code) == 5:
                for name, symbol in aliases.items():
                    if name in text:
                        value = int(instruction["ida_linear_address"], 16) + 5 + int.from_bytes(code[1:], "little", signed=True)
                        if value != LINK_SYMBOLS[symbol]:
                            raise ValueError("外部符號位址與 IDA 原始指令不符")
                        observed[symbol] = value
    if observed != LINK_SYMBOLS:
        raise ValueError("外部符號位址與 IDA 原始指令不符")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path("/repo"))
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--original", type=Path, default=Path("/input/FD2.EXE"))
    args = parser.parse_args()
    evidence = json.loads(args.evidence.read_text(encoding="utf-8"))
    reference = next(item for item in json.loads(
        (args.repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8")
    )["files"] if item["file"] == "FD2.EXE")
    original = args.original.read_bytes()
    identity = {"file": "FD2.EXE", "size": len(original),
                "md5": hashlib.md5(original).hexdigest(), "sha256": hashlib.sha256(original).hexdigest()}
    if identity != reference or evidence["input"] != reference:
        raise ValueError("原版或 IDA 證據的固定雜湊不符")
    if sha256(args.repo / "tools/ida_export_matching_pilot.py") != evidence["exporter_sha256"]:
        raise ValueError("IDA 證據與目前匯出器不符")
    if sha256(args.repo / "docs/data/ida/fd2_function_inventory.json") != evidence["function_inventory_sha256"]:
        raise ValueError("IDA 證據與目前函式清冊不符")
    for function in evidence["functions"]:
        reconstructed = bytearray()
        for instruction in function["instructions"]:
            offset = int(instruction["file_offset"], 16)
            raw = bytes.fromhex(instruction["file_bytes"])
            if original[offset:offset + len(raw)] != raw:
                raise ValueError("IDA 的原始檔位元組與目前原版不符")
            reconstructed += bytes.fromhex(instruction["loaded_bytes"])
        if bytes(reconstructed) != bytes.fromhex(function["loaded_bytes"]):
            raise ValueError("IDA 函式的逐指令覆蓋不完整")
    verify_bindings(evidence)
    args.output.mkdir(parents=True, exist_ok=True)
    source = args.repo / "tools/fd2_matching_pilot.c"
    compiler = subprocess.run(["wcc386"], capture_output=True, text=True, timeout=5)
    report = {
        "schema_version": 1, "input": identity,
        "scope": "three-function pure-C candidate experiment; no production runtime or full EXE match",
        "compiler": "\n".join((compiler.stdout + compiler.stderr).splitlines()[:6]),
        "compiler_sha256": sha256(Path("/opt/watcom/binl64/wcc386")),
        "linker": checked(["ld", "--version"]).splitlines()[0],
        "linker_sha256": sha256(Path("/usr/bin/ld")),
        "source_sha256": sha256(source),
        "driver_sha256": sha256(Path(__file__)),
        "evidence_sha256": sha256(args.evidence),
        "link_symbols": {key: hex(value) for key, value in LINK_SYMBOLS.items()},
        "comparison": "complete linked function bytes at original IDA linear address; no masks or patches",
        "original_context": "0x145CD contiguous IDA interval is 88 bytes and branches to a separately recorded 4-byte return tail at 0x145C9; isolated compilation does not recreate the original translation-unit tail layout",
        "trials": [],
    }
    for cpu, (profile, options), (name, macro, symbol, address, extra) in itertools.product(
            ("-3s", "-4s", "-5s"), PROFILES.items(), CASES):
        trial = args.output / f"{name}-{cpu[1:]}-{profile}"
        trial.mkdir(exist_ok=True)
        obj = trial / "candidate.obj"
        exe = trial / "linked.exe"
        flags = ("-q", "-bt=dos", "-mf", cpu, *options, *extra, "-eoc", "-zld", "-d" + macro)
        checked(["wcc386", *flags, "-fo=" + str(obj), str(source)])
        size = coff_text_size(obj)
        script = trial / "link.ld"
        definitions = "\n".join(f"{key} = {value:#x};" for key, value in LINK_SYMBOLS.items())
        script.write_text(definitions + f"\nSECTIONS {{ .text {address:#x} : SUBALIGN(1) {{ *(_TEXT) }} "
                          "/DISCARD/ : { *(CONST) *(CONST2) *(_DATA) *(.depend) *(.reloc) } }\n", encoding="ascii")
        checked(["ld", "-mi386pe", "--image-base", "0", "--section-alignment", "1", "--file-alignment", "1",
                 "--no-insert-timestamp", "-T", str(script), "-e", symbol, "-o", str(exe), str(obj)])
        code = linked_text(exe, address, size)
        (trial / "candidate.bin").write_bytes(code)
        checked(["wdis", "-l=" + str(trial / "candidate.dis"), str(obj)])
        target = next(f for f in evidence["functions"] if int(f["inventory"]["start"], 16) == address)
        expected = bytes.fromhex(target["loaded_bytes"])
        prefix = next((i for i, pair in enumerate(zip(code, expected)) if pair[0] != pair[1]), min(len(code), len(expected)))
        report["trials"].append({
            "case": name, "address": hex(address), "cpu": cpu, "profile": profile,
            "compiler_flags": list(flags), "original_size": len(expected), "candidate_size": len(code),
            "exact_function_bytes": code == expected,
            "first_difference_offset": None if code == expected else prefix,
            "candidate_code_sha256": hashlib.sha256(code).hexdigest(),
            "original_loaded_code_sha256": hashlib.sha256(expected).hexdigest(),
            "artifact_directory": trial.name,
        })
    report["trial_count"] = len(report["trials"])
    report["matched_addresses"] = sorted({t["address"] for t in report["trials"] if t["exact_function_bytes"]})
    (args.output / "report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    for address in ("0x4e893", "0x145cd", "0x1b8e7"):
        trials = [t for t in report["trials"] if t["address"] == address]
        print(json.dumps({"address": address, "trials": len(trials),
                          "original_size": trials[0]["original_size"],
                          "candidate_sizes": sorted({t["candidate_size"] for t in trials}),
                          "exact_matches": sum(t["exact_function_bytes"] for t in trials)}, ensure_ascii=False), flush=True)


if __name__ == "__main__":
    main()
