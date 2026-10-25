#!/usr/bin/env python3
"""在既有Go Docker驗戰後成功訊息HD；必要測試缺項或SKIP即拒收。"""
import argparse
import hashlib
import json
import os
import re
from pathlib import Path
import subprocess
import sys

from verify_fd2_player9_map_hd import check_real_runtime

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = {
    "full-unit": {"TestModernHDFullRewardQuestion", "TestModernHDFullRewardGlyphClears", "TestModernHDFullRewardSource", "TestModernHDFullRewardChromeRejectsBeforeQueue"},
    "full-game": {"TestModernHDFullRewardGameGPU"},
    "stage-unit": {"TestModernHDRewardStageSource"},
    "stage-game": {"TestModernHDRewardStageGameGPU"},
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
    p.add_argument("--require-real-runtime", action="store_true")
    a = p.parse_args()
    if not Path("/.dockerenv").exists() or not 60 <= a.timeout <= 1500 or not a.out.parent.is_dir() or not a.catalog.is_file() or not a.pack.is_dir():
        raise ValueError("Docker內使用有界預算與既有輸出根")
    if any(Path(str(a.out)+s).exists() for s in [".gate.json", ".log", ".memory.json", "-capture"]):
        raise ValueError("不覆寫收據")
    env = os.environ.copy()
    env.pop("FD2_HD_REWARD_STAGE_GPU", None)
    env.pop("FD2_HD_FULL_REWARD_GPU", None)
    env.update(FD2_MUTE="1", FD2_HD_REWARD_GPU="1", FD2_TEST_REWARD_CATALOG=str(a.catalog.resolve()),
               FD2_TEST_REWARD_PACK=str(a.pack.resolve()), FD2_TEST_REWARD_OUT=str(a.out)+"-capture")
    env.update(FD2_TEST_UNIT_PRESENT_CATALOG=str(a.catalog.resolve()), FD2_TEST_UNIT_PRESENT_PACK=str(a.pack.resolve()))
    if a.kind == "stage-game": env["FD2_HD_REWARD_STAGE_GPU"] = "1"
    if a.kind == "full-game": env["FD2_HD_FULL_REWARD_GPU"] = "1"
    physical = check_real_runtime(a.pack, json.loads(a.catalog.read_text())) if a.require_real_runtime else None
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
    if a.kind == "full-game":
        d = json.loads(Path(str(a.out)+"-capture/receipt.json").read_text())
        if (d.get("status") != "PASS-FULL-REWARD-HD-RUNTIME-E1" or d.get("cases") != 5
                or d.get("map_index") != 23 or d.get("native_seed") != 7 or d.get("fixed_clock_unix") != 1000
                or d.get("normal_player_verified") is not False or d.get("continuations") != 5
                or d.get("f2_boundaries", 0) < 30 or d.get("dialogue_chrome") is not True
                or d.get("legacy_chrome_rejected") is not True or d.get("chrome_pixels", 0) <= 0):
            raise ValueError("滿背包五分支或F2邊界缺項")
        for case in range(5):
            rows = [r for r in d.get("rows", []) if r.get("case") == case]
            phases = {r.get("phase") for r in rows}
            required = {"yes-no", "return", "job-0", "job-2", "job-3"}
            required |= {"item-open", "item-choice", "item-close"} if case in (2, 3) else {"job-4", "response-hold"}
            if case == 3: required |= {"job-5", "job-6", "response-hold"}
            if not required <= phases or any(r.get("native_control_unchanged") is not True for r in rows):
                raise ValueError("滿背包分支轉場或原圖控制組缺項")
            if any(not re.fullmatch("[0-9a-f]{64}", r.get(k, "")) for r in rows for k in ("world_sha256", "final_sha256")):
                raise ValueError("滿背包畫布雜湊缺項")
    if a.kind == "stage-game":
        capture = Path(str(a.out)+"-capture")
        d = json.loads((capture/"receipt.json").read_text())
        if (d.get("status")!="PASS-REWARD-STAGE-HD-RUNTIME-E1" or d.get("cases")!=2 or d.get("map_index")!=23
                or d.get("game_frames")!=28 or d.get("f2_round_trips")!=28 or d.get("native_seed")!=7
                or d.get("fixed_clock_unix")!=1000 or d.get("rgba_tolerance")!=2 or d.get("normal_player_verified") is not False
                or len(d.get("rows",[]))!=26 or len(d.get("returns",[]))!=2 or len(d.get("sources",[]))!=2
                or len(list(capture.glob("*.png")))!=12):
            raise ValueError("特殊背景兩類完整訊息、返回或來源缺項")
        for kind,value in [(0,192),(1,123456)]:
            rows=[r for r in d["rows"] if r.get("kind")==kind]
            tail=[r for r in d["returns"] if r.get("kind")==kind]
            sources=[r for r in d["sources"] if r.get("kind")==kind]
            if len(rows)!=13 or {r.get("phase_index") for r in rows}!=set(range(13)) or len(tail)!=1 or len(sources)!=1:
                raise ValueError("六開／等待／五收／還原或返回缺項")
            if any(r.get("value")!=value or r.get("native_seed")!=7 or r.get("f2_roundtrip") is not True or r.get("native_control_unchanged") is not True for r in rows+tail):
                raise ValueError("來源、F2、seed或原圖控制組不同")
            if any(r.get("visible_stage_pixels",0)<=0 or r.get("outside_hd_pixels",0)<=0 for r in rows):
                raise ValueError("特殊背景或框外HD核對為空")
            if any(not re.fullmatch("[0-9a-f]{64}",r.get(k,"")) for r in rows for k in ["final_sha256","world_sha256","native_vga_sha256"]):
                raise ValueError("完整圖面或原生VGA雜湊缺項")
            if tail[0].get("gold")!=123+(value if kind==1 else 0) or tail[0].get("continuations")!=1 or not re.fullmatch("[0-9a-f]{64}",tail[0].get("final_sha256","")):
                raise ValueError("關框後結算、一次續行或HD返回缺項")
            if sources[0].get("map_index")!=23 or any(not re.fullmatch("[0-9a-f]{64}",sources[0].get(k,"")) for k in ["scene_sha256","staging_sha256"]):
                raise ValueError("密封場景或同次stage來源缺項")
    report = dict(status="PASS", kind=a.kind, required=sorted(REQUIRED[a.kind]),
                  catalog_sha256=hashlib.sha256(a.catalog.read_bytes()).hexdigest(), pack=str(a.pack.resolve()),
                  scope="受控物品／金錢成功訊息；不是正常玩家或原版對拍證據")
    report["independent_runtime_files"] = physical
    report["verifier_sha256"] = hashlib.sha256(Path(__file__).read_bytes()).hexdigest()
    Path(str(a.out)+".gate.json").write_text(json.dumps(report, ensure_ascii=False, indent=2)+"\n")
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
