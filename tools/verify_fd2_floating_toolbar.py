#!/usr/bin/env python3
"""在維護中的 Go Docker 驗證浮出列與歷史固定列，拒收缺項、SKIP、逾時及 OOM。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
TESTS = {
    "unit": {"TestFloatingToolbarGeometryAndOwnership", "TestFloatingToolbarLocaleTextBounds"},
    "floating": {"TestFloatingToolbarRealInputGPU"},
    "fixed": {"TestPlayerThemeToolbarRealInputGPU"},
    "poster-unit": {"TestPlayerPosterApprovedPixelsAndAtomicSourceRejection", "TestPlayerPosterGeometryInputAndBlocking", "TestPlayerPosterMaskFadeAndSourceRemainIndependent", "TestPlayerPosterPreferencesPreserveOtherSettingsAndRejectAtomically", "TestPlayerPosterMenuLabelsFitAllLocales"},
    "poster-gpu": {"TestPlayerPosterCompleteGameGPU"},
    "poster-input": {"TestPlayerPosterToolbarRealInputGPU"},
    "poster-terrain": {"TestPlayerPosterTerrainBackgroundGPU"},
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--kind", choices=TESTS, required=True)
    for name in ("catalog", "pack", "out"):
        p.add_argument("--" + name, type=Path, required=True)
    p.add_argument("--timeout", type=int, default=600)
    a = p.parse_args()
    if not Path("/.dockerenv").exists() or not 60 <= a.timeout <= 1200:
        raise ValueError("需在 Go Docker 內執行並限制時間")
    if not a.out.parent.is_dir() or not a.catalog.is_file() or not a.pack.is_dir():
        raise ValueError("必須使用既有輸入與輸出父目錄")
    if any(Path(str(a.out) + s).exists() for s in (".gate.json", ".log", ".memory.json", "-capture", ".xtest")):
        raise ValueError("不覆寫既有收據")
    env = os.environ.copy()
    env.update(FD2_MUTE="1", FD2_TEST_FACILITY_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_FACILITY_PACK=str(a.pack.resolve()), FD2_TEST_TOOLBAR_OUT=str(a.out) + "-capture")
    env["WITH_XVFB_SCREEN"] = "1920x1272x24" if a.kind == "fixed" else "1920x1200x24"
    for key in ("FD2_TOOLBAR_INPUT_GPU", "FD2_FLOATING_TOOLBAR_GPU", "FD2_POSTER_GPU", "FD2_POSTER_INPUT_GPU", "FD2_POSTER_TERRAIN_GPU"):
        env.pop(key, None)
    if a.kind in {"floating", "fixed", "poster-input"}:
        driver = Path(str(a.out) + ".xtest")
        subprocess.run(["cc", "-O2", "-Wall", "-Wextra", "-Werror", str(ROOT / "tools/docker/fd2-xtest-input.c"),
                        "-lX11", "-lXtst", "-o", str(driver)], check=True, timeout=30)
        env["FD2_TEST_XTEST_INPUT"] = str(driver.resolve())
        env[{"floating": "FD2_FLOATING_TOOLBAR_GPU", "fixed": "FD2_TOOLBAR_INPUT_GPU", "poster-input": "FD2_POSTER_INPUT_GPU"}[a.kind]] = "1"
    if a.kind == "poster-gpu":
        env["FD2_POSTER_GPU"] = "1"
    if a.kind == "poster-terrain":
        env["FD2_POSTER_TERRAIN_GPU"] = "1"
    command = [sys.executable, str(ROOT / "tools/fd2_go_test_memory_probe.py"), "--out", str(a.out),
               "--timeout", str(a.timeout), "--", "with-xvfb", "go", "test", "-json", "-count=1",
               f"-timeout={a.timeout - 30}s", "./cmd/fd2", "-run", "^(" + "|".join(sorted(TESTS[a.kind])) + ")$"]
    subprocess.run(command, cwd=ROOT / "remake", env=env, check=True, timeout=a.timeout + 30)
    memory = json.loads(Path(str(a.out) + ".memory.json").read_text())
    if memory["returncode"] or memory["timeout"] or any(memory["memory_events_delta"].get(k, 0) for k in ("oom", "oom_kill")):
        raise ValueError("失敗、逾時或 OOM")
    events = [json.loads(l) for l in Path(str(a.out) + ".log").read_text().splitlines() if l.startswith("{")]
    passed = {e["Test"] for e in events if e.get("Action") == "pass" and e.get("Test") and "/" not in e["Test"]}
    if any(e.get("Action") in ("skip", "fail") for e in events) or passed != TESTS[a.kind]:
        raise ValueError(f"測試缺項或拒收：{passed}")
    result = {"status": "PASS-FLOATING-TOOLBAR-GATE", "kind": a.kind, "passed": sorted(passed),
              "tool_sha256": sha(Path(__file__)), "catalog_sha256": sha(a.catalog), "pack": str(a.pack),
              "memory": memory, "scope": "受控重製 GUI 與幾何；非原版對拍、完整正常戰役或平台驗收"}
    Path(str(a.out) + ".gate.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": result["status"], "kind": a.kind, "passed": sorted(passed)}))


if __name__ == "__main__":
    main()
