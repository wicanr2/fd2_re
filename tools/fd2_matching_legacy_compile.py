#!/usr/bin/env python3
"""在 Docker 內用本機歷史 Watcom 編譯三函式候選。

compiler 來源唯讀；此工具只保存自己的 C 候選、OMF、旗標與 compiler
版本收據。DOSBox 僅執行開發工具，不產生 FD2 原版遊戲對拍收據。
目標入口：docs/goal/118-goal-fd2-matching-decompilation-20261008.md。
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess


COMPILERS = {
    "9.01": ("/wc901", "99d79830e8bf2bf04582215583cd6c276e21226d86a9bdea197a99abb75eabe0"),
    "9.5": ("/wc95", "3fe098187af3ed4bccbf184b0f0d1a21a18cf976fe72c6178f19b8eea8b13e78"),
    "10.0a": ("/wc10a", "c3666de94f6fa6800f452dae8acf45505ecdb62f0ade2cc27cc86c2d9e8e2b6b"),
}
CODEGEN_901_SHA256 = "4d82d8bb5623539337b6667808ee5e1ece99047fe009d35c852108afa2e84cb2"
CASES = {
    "R": ("FD2_PILOT_RNG", "-s"),
    "P": ("FD2_PILOT_RNG", "-s -dFD2_RNG_PROMOTED"),
    "O": ("FD2_PILOT_OCCUPANCY", ""),
    "I": ("FD2_PILOT_INVENTORY", ""),
}


def run(version, output, source):
    compiler_dir, expected_sha = COMPILERS[version]
    compiler = next(p for p in Path(compiler_dir).iterdir()
                    if p.name.lower() == "wcc386.exe")
    if hashlib.sha256(compiler.read_bytes()).hexdigest() != expected_sha:
        raise ValueError("歷史 compiler 與已確認 SHA-256 不符")
    if version == "9.01" and hashlib.sha256(
            (Path(compiler_dir) / "386wcgl.exe").read_bytes()).hexdigest() != CODEGEN_901_SHA256:
        raise ValueError("Watcom 9.01 產碼程式與已確認 SHA-256 不符")
    output.mkdir(parents=True, exist_ok=True)
    if output.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄不屬於目前使用者")
    # DOS 原廠 lexer 不需讀取 UTF-8 註解；C 識別字、常數與運算式保持原文。
    ascii_source = source.read_text(encoding="utf-8").encode("ascii", "ignore")
    (output / "PILOT.C").write_bytes(ascii_source)
    configuration = [
        "[sdl]", "fullscreen=false", "output=surface", "[dosbox]", "memsize=31",
        "[cpu]", "core=normal", "cycles=max 90% limit 100000", "[mixer]", "nosound=true",
        "[midi]", "mpu401=none", "mididevice=none", "[autoexec]",
        f"mount c {output}", f"mount r {compiler_dir}", "c:",
        "set WATCOM=R:", "set PATH=R:\\;Z:\\",
    ]
    records = []
    for case, (macro, extra) in CASES.items():
        for cpu in ((3, 4) if version == "9.01" else (3, 4, 5)):
            for profile in ("s", "t"):
                stem = f"{case}{cpu}{profile.upper()}"
                flags = f"-mf -{cpu}s -o{profile} {extra} -d{macro}"
                command = f"R:\\WCC386.EXE {flags} -fo={stem}.OBJ PILOT.C > {stem}.TXT"
                if len(command) >= 126:
                    raise ValueError("DOS 命令行超出有界長度")
                configuration.append(command)
                records.append({"case": case, "flags": flags.split(), "stem": stem})
    configuration.append("exit")
    config = output / "DOSBOX.CONF"
    config.write_text("\n".join(configuration) + "\n", encoding="ascii")
    result = subprocess.run(["dosbox", "-conf", str(config), "-noconsole"],
                            capture_output=True, text=True, timeout=180)
    (output / "runner.log").write_text(result.stdout + result.stderr, encoding="utf-8")
    if result.returncode:
        raise ValueError("DOS compiler 執行器未正常退出")
    for record in records:
        log = output / (record["stem"] + ".TXT")
        obj = output / (record["stem"] + ".OBJ")
        message = log.read_text(encoding="cp437") if log.exists() else ""
        match = re.search(r"Optimizing Compiler\s+Version\s+([^\r\n]+)", message)
        record["compiler_version"] = match.group(1).strip() if match else None
        record["compiled"] = obj.is_file() and "0 errors" in message
        record["diagnostics"] = [line for line in message.splitlines() if "Error" in line or "Code size:" in line]
        if record["compiled"]:
            record["object_sha256"] = hashlib.sha256(obj.read_bytes()).hexdigest()
            record["object_size"] = obj.stat().st_size
    report = {
        "schema_version": 1, "compiler_mount": compiler_dir, "compiler_sha256": expected_sha,
        "source_sha256": hashlib.sha256(source.read_bytes()).hexdigest(),
        "ascii_source_sha256": hashlib.sha256(ascii_source).hexdigest(),
        "runner": "DOSBox 0.74-3; compiler execution only; no FD2 game receipt",
        "driver_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(), "trials": records,
    }
    (output / "compile-report.json").write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"version": version, "compiled": sum(r["compiled"] for r in records),
                      "total": len(records), "diagnostics": [r for r in records if not r["compiled"]]}, ensure_ascii=False), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--versions", choices=COMPILERS, nargs="+", default=list(COMPILERS))
    args = parser.parse_args()
    for version in args.versions:
        run(version, args.output / version, args.source)


if __name__ == "__main__":
    main()
