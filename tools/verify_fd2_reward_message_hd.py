#!/usr/bin/env python3
"""在既有Go Docker驗戰後成功訊息HD；必要測試缺項或SKIP即拒收。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    "hud": {"TestNativeMessageCloseRedrawsHUDAfterRestoreDraw", "TestInputAuditTracksDeathRewardMessageOwner"},
    "unit": {"TestModernHDRewardPanelCoverage", "TestModernHDRewardTextAndNativeAssets"},
    "game": {"TestModernHDRewardCompleteGameGPU"},
    "regression": {"TestModernHDPreparationStableFramesRestoreNativeAssets",
                   "TestModernHDTimedDialogueCoverageMatchesNativeSnapshots",
                   "TestModernHDTimedDialogueFreezeAndAtomicRejection",
                   "TestModernHDPlayer9FrozenInputMatchesSnapshot",
                   "TestNativeDeathRewardFullInventoryNoAndEscapeRemainAtomic",
                   "TestNativeDeathRewardDiscardCommitsOnlyAfterItemPanelClose",
                   "TestNativeDeathRewardSelectorCancelAndSequentialQueue",
                   "TestNativeDeathRewardMissingProvenanceFailsClosedWithoutMutation"},
}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--kind", choices=REQUIRED, required=True)
    for key in ["catalog", "pack", "out"]:
        p.add_argument("--"+key, type=Path, required=True)
    p.add_argument("--timeout", type=int, default=600)
    a = p.parse_args()
    if not Path("/.dockerenv").exists() or not 60 <= a.timeout <= 1500 or not a.out.parent.is_dir():
        raise ValueError("Docker內使用有界預算與既有輸出根")
    if any(Path(str(a.out)+s).exists() for s in [".gate.json", ".log", ".memory.json", "-capture"]):
        raise ValueError("不覆寫收據")
    env = os.environ.copy()
    env.update(FD2_MUTE="1", FD2_HD_REWARD_GPU="1", FD2_TEST_REWARD_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_REWARD_PACK=str(a.pack.resolve()), FD2_TEST_REWARD_OUT=str(a.out)+"-capture")
    command = [sys.executable, str(ROOT/"tools/fd2_go_test_memory_probe.py"), "--out", str(a.out),
               "--timeout", str(a.timeout), "--", "with-xvfb", "go", "test", "-json", "-count=1",
               f"-timeout={a.timeout-30}s", "./cmd/fd2", "-run", "^("+"|".join(sorted(REQUIRED[a.kind]))+")$"]
    r = subprocess.run(command, cwd=ROOT/"remake", env=env)
    if r.returncode:
        raise SystemExit(r.returncode)
    memory = json.loads(Path(str(a.out)+".memory.json").read_text())
    if memory["returncode"] or memory["timeout"] or memory["after"]["memory.events"]["oom"] or memory["after"]["memory.events"]["oom_kill"]:
        raise ValueError("失敗／逾時／OOM")
    events = [json.loads(x) for x in Path(str(a.out)+".log").read_text().splitlines() if x.startswith("{")]
    passed = {e.get("Test") for e in events if e.get("Action") == "pass"}
    if not REQUIRED[a.kind] <= passed or any(e.get("Action") in {"fail", "skip"} for e in events):
        raise ValueError("必要測試缺項／失敗／SKIP")
    if a.kind == "game":
        receipt = json.loads(Path(str(a.out)+"-capture/receipt.json").read_text())
        if receipt["status"] != "PASS-REWARD-HD-RUNTIME-E1" or receipt["cases"] != 2 or receipt["game_frames"] != 26 or receipt["f2_round_trips"] != 26:
            raise ValueError("完整兩種成功訊息缺項")
    report = dict(status="PASS", kind=a.kind, required=sorted(REQUIRED[a.kind]),
                  catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(), pack=str(a.pack.resolve()),
                  scope="受控物品／金錢成功訊息；不是正常玩家或原版對拍證據")
    Path(str(a.out)+".gate.json").write_text(json.dumps(report, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
