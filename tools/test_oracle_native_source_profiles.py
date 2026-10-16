#!/usr/bin/env python3
"""用真實來源bytes驗證native來源准入，不以mock雜湊取代證據。"""
import argparse
import ast
import hashlib
import os
import subprocess
import tempfile
from pathlib import Path


def main():
    if not Path("/.dockerenv").is_file():
        raise SystemExit("本工具只在Docker內執行")
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--dosgolem", type=Path, required=True)
    a = p.parse_args()
    root = Path(__file__).resolve().parent.parent
    shell = (root / "tools/dosgolem_oracle_container.sh").read_text()
    code = shell.split("python3 - <<'META'\n", 1)[1].split("\nMETA\n", 1)[0]
    module = ast.parse(code)
    function = next(n for n in module.body if isinstance(n, ast.FunctionDef) and n.name == "near_heap_policy")
    namespace = {"hashlib": hashlib, "os": os, "Path": Path}
    exec(compile(ast.Module(body=[function], type_ignores=[]), "oracle-META", "exec"), namespace)
    policy = namespace["near_heap_policy"]
    paths = ["internal/machine/watcom_runtime.go", "apps/fd2/cmd/oracle/main.go",
             "internal/cpu386/cpu.go", "internal/machine/le_startup.go",
             "internal/machine/le_dma_clock.go", "internal/machine/le_real_irq7.go"]
    helper = "apps/fd2/cmd/oracle/hp_lock.go"
    current = {p: (a.dosgolem / p).read_bytes() for p in paths + [helper]}
    old = {p: subprocess.run(["git", "-C", str(a.dosgolem), "show",
            "fa25f6e7d7f6be399b6563b7df6a55995daba77b:" + p], check=True, capture_output=True).stdout for p in paths[:4]}
    merged = {p: subprocess.run(["git", "-C", str(a.dosgolem), "show",
            "81c85bebdefe2e29ea1d5dc2628b2081d464e927:" + p], check=True, capture_output=True).stdout for p in paths}
    fixed = {p: subprocess.run(["git", "-C", str(a.dosgolem), "show",
            "e26fb41ff68b38bceb89326dccd8fdceeab51666:" + p], check=True, capture_output=True).stdout for p in paths}
    os.environ["FD2_ORACLE_HEAP_PROFILE"] = "native"
    with tempfile.TemporaryDirectory() as temp:
        target = Path(temp)
        def check(sources, accepted):
            for name, raw in sources.items():
                dest = target / name
                dest.parent.mkdir(parents=True, exist_ok=True)
                dest.write_bytes(raw)
            try:
                result = policy(target)
            except SystemExit:
                assert not accepted
            else:
                assert accepted and result["profile"] == "native" and result["status"] == "reviewed"
        check(old, True)
        check(merged, True)
        check(fixed, True)
        check(current, True)
        for name in paths:
            check({**fixed, name: fixed[name] + b"\n"}, False)
        for name in paths + [helper]:
            check({**current, name: current[name] + b"\n"}, False)
        check(current, True)
        (target / helper).unlink()
        check({name: raw for name, raw in current.items() if name != helper}, False)
        check({**fixed, paths[2]: old[paths[2]]}, False)
        check({**fixed, paths[3]: old[paths[3]]}, False)
    print("native source profiles: old/merged/fixed/current accepted; mutations, missing helper and mixed tuples rejected")


if __name__ == "__main__":
    main()
