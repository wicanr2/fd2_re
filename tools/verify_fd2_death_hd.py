#!/usr/bin/env python3
"""在維護中的Go Docker驗死亡動畫背景；缺項、SKIP、逾時或OOM即拒收。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

from verify_fd2_player9_map_hd import check_real_runtime

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    "native": {"TestNativeDeathStageUsesSnapshotSeed", "TestNativeDeathStageColdFailureIsAtomic", "TestModernHDDeathStageOmitsSteadyHUDUnit"},
    "unit": {"TestModernHDDeathStageSourceAndReject"},
    "game": {"TestModernHDDeathStageGameGPU"},
    "regression": {
        "TestModernHDDeathScheduleAndAtomicPreflight", "TestModernHDDeathAssetLoader",
        "TestModernHDDeathRetainsAuxiliarySeedWithoutSteadyWriter",
        "TestNativeUnitDeathRedrawsHUDOnlyInFinalSteadyFrame",
        "TestNativeCh28PostPresentPrecomputesExactScheduleAndCommits",
        "TestNativeCh28PostPresentMissingLMIRejectsBeforeMutation",
        "TestNativeCh28PostPresentFailureRollsBackStateAndBuffers",
        "TestNativeActionDeathPresentationRetainsSettlementAndContinuesOnce",
        "TestNativeActionDeathPresentationRejectsBeforeInactiveWrite",
        "TestNativeCommandDamageDeathUsesWriterFlagWithoutPhysicalEntry",
        "TestNativeActionDeathPresentationSkipsOffscreenAndOldInactive",
        "TestNativeTransientDeathOwnerRunsAfterMessageWithoutRewards",
        "TestNativeTransientDeathPreflightRejectsBeforeHPTransaction",
        "TestNativeTransientDeathPreviewMatchesFocusWithoutPublishingOrTracing",
    },
}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--kind", choices=REQUIRED, required=True)
    for key in ("catalog", "pack", "out"):
        p.add_argument("--"+key, type=Path, required=True)
    p.add_argument("--timeout", type=int, default=600)
    p.add_argument("--require-real-runtime", action="store_true")
    a = p.parse_args()
    if (not Path("/.dockerenv").exists() or not 60 <= a.timeout <= 1500
            or not a.out.parent.is_dir() or not a.catalog.is_file() or not a.pack.is_dir()):
        raise ValueError("Docker內使用既有目錄、明示圖包與有界預算")
    if any(Path(str(a.out)+s).exists() for s in (".gate.json", ".log", ".memory.json", "-capture")):
        raise ValueError("不覆寫既有收據")
    env = os.environ.copy()
    env.update(FD2_MUTE="1", FD2_TEST_UNIT_PRESENT_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_UNIT_PRESENT_PACK=str(a.pack.resolve()), FD2_TEST_DEATH_STAGE_OUT=str(a.out)+"-capture")
    env.pop("FD2_HD_DEATH_STAGE_GPU", None)
    if a.kind == "game": env["FD2_HD_DEATH_STAGE_GPU"] = "1"
    physical = check_real_runtime(a.pack, json.loads(a.catalog.read_text())) if a.require_real_runtime else None
    command = [sys.executable, str(ROOT/"tools/fd2_go_test_memory_probe.py"), "--out", str(a.out),
               "--timeout", str(a.timeout), "--", "with-xvfb", "go", "test", "-json", "-count=1",
               f"-timeout={a.timeout-30}s", "./cmd/fd2", "-run", "^("+"|".join(sorted(REQUIRED[a.kind]))+")$"]
    result = subprocess.run(command, cwd=ROOT/"remake", env=env)
    if result.returncode: raise SystemExit(result.returncode)
    memory = json.loads(Path(str(a.out)+".memory.json").read_text())
    if memory["returncode"] or memory["timeout"] or memory["after"]["memory.events"]["oom"] or memory["after"]["memory.events"]["oom_kill"]:
        raise ValueError("失敗／逾時／OOM")
    events = [json.loads(x) for x in Path(str(a.out)+".log").read_text().splitlines() if x.startswith("{")]
    passed = {e.get("Test") for e in events if e.get("Action") == "pass"}
    if not REQUIRED[a.kind] <= passed or any(e.get("Action") in {"fail", "skip"} for e in events):
        raise ValueError("必要測試缺項／失敗／SKIP")
    if a.kind == "game":
        d = json.loads(Path(str(a.out)+"-capture/receipt.json").read_text())
        if (d.get("status") != "PASS-DEATH-STAGE-HD-RUNTIME-E1" or d.get("cases") != 3
                or d.get("map_index") != 23 or d.get("native_seed") != 7 or d.get("fixed_clock_unix") != 1000
                or d.get("normal_player_verified") is not False or len(d.get("rows", [])) != 78
                or len(d.get("returns", [])) != 3 or d.get("f2_round_trips") != 24):
            raise ValueError("三caller完整畫面與F2缺項")
        for kind in ("script", "action", "status"):
            rows = [r for r in d["rows"] if r.get("kind") == kind]
            tail = [r for r in d["returns"] if r.get("kind") == kind]
            if len(rows) != 26 or {r.get("frame") for r in rows} != set(range(26)) or len(tail) != 1:
                raise ValueError("死亡26拍或返回缺項")
            if any(r.get("native_control_unchanged") is not True or r.get("visible_stage_pixels", 0) <= 0 for r in rows):
                raise ValueError("控制組或非空背景缺項")
            if tail[0].get("continuations") != 1 or tail[0].get("f2_roundtrip") is not True:
                raise ValueError("一次續行或正常HD返回缺項")
    report = dict(status="PASS", kind=a.kind, required=sorted(REQUIRED[a.kind]),
                  catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(), pack=str(a.pack.resolve()),
                  independent_runtime_files=physical, verifier_sha256=hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
                  scope="受控死亡動畫RUNTIME-E1；不是新原版oracle、正常玩家或平台驗收")
    Path(str(a.out)+".gate.json").write_text(json.dumps(report, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
