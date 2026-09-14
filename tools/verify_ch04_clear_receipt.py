#!/usr/bin/env python3
"""驗證第四關修改路徑閉環收據；不把受控清場提升成 PLAYER-E2。"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path


HEX256 = re.compile(r"^[0-9a-f]{64}$")


def digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(1024 * 1024), b""):
            value.update(block)
    return value.hexdigest()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise SystemExit(f"第四關閉環收據驗證失敗：{message}")


def alive(checkpoint: dict, camp: int) -> int:
    return sum(1 for unit in checkpoint.get("units", [])
               if unit.get("camp") == camp and unit.get("hp", 0) > 0)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("receipt", type=Path)
    parser.add_argument("--artifact-root", type=Path)
    parser.add_argument("--require-artifacts", action="store_true")
    args = parser.parse_args()

    receipt_path = args.receipt.resolve()
    repo = receipt_path.parents[2]
    receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
    require(receipt.get("schema_version") == 1, "schema_version 不是 1")
    require(receipt.get("kind") == "fd2_ch04_modified_path_closure_receipt",
            "kind 不符")

    inputs = receipt["inputs"]
    plan_path = repo / inputs["plan"]
    require(plan_path.is_file(), f"找不到控制序列 {plan_path}")
    require(digest(plan_path) == inputs["plan_sha256"], "控制序列雜湊不符")
    plan = [json.loads(line) for line in plan_path.read_text(encoding="utf-8").splitlines()
            if line.strip() and not line.startswith("#")]
    require(any(item.get("force_enemy_clear") is True for item in plan),
            "控制序列沒有 force_enemy_clear")
    require(any(item.get("await_ui") == "town" for item in plan),
            "控制序列沒有 await_ui=town")
    require(any(item.get("town_save") is True for item in plan),
            "控制序列沒有 town_save")

    manifest = json.loads((repo / inputs["reference_manifest"]).read_text(encoding="utf-8"))
    exe = next(item for item in manifest["files"] if item["file"] == "FD2.EXE")
    require(exe["size"] == inputs["fd2_exe_size"], "FD2.EXE 大小不符")
    require(exe["md5"] == inputs["fd2_exe_md5"], "FD2.EXE MD5 不符")
    require(exe["sha256"] == inputs["fd2_exe_sha256"], "FD2.EXE SHA-256 不符")

    evidence = receipt["evidence"]
    require(evidence.get("level") == "MODIFIED-PATH", "證據未標修改路徑")
    restrictions = set(evidence.get("does_not_prove", []))
    require("一般玩家路徑 PLAYER-E2" in restrictions, "缺 PLAYER-E2 禁止宣稱")
    require("傷害公式" in restrictions and "未修改的戰鬥勝利條件" in restrictions,
            "缺戰鬥規則禁止宣稱")
    require(receipt["outputs"].get("town_save_changed") is True,
            "沒有城鎮存檔變更聲明")
    require(inputs["source_fd2_sav_sha256"] != receipt["outputs"]["result_fd2_sav_sha256"],
            "起訖 FD2.SAV 雜湊相同")

    for section in (receipt["runner"], receipt["execution"], receipt["outputs"]):
        for key, value in section.items():
            if key.endswith("sha256"):
                require(isinstance(value, str) and HEX256.fullmatch(value) is not None,
                        f"{key} 不是 SHA-256")
    for checkpoint in receipt["checkpoints"]:
        require(HEX256.fullmatch(checkpoint["sha256"]) is not None,
                f"checkpoint {checkpoint['seq']} 雜湊無效")

    root = args.artifact_root
    if root is None:
        candidate = repo / receipt["outputs"]["local_receipt_root"]
        if candidate.is_dir():
            root = candidate
    if root is None:
        require(not args.require_artifacts, "要求驗證本機產物，但沒有 artifact root")
        print("第四關閉環收據靜態契約通過（未提供本機產物）")
        return 0

    root = root.resolve()
    run = root / "run"
    result_state = root / "state"
    source_state = repo / inputs["source_state"]
    require(run.is_dir() and result_state.is_dir() and source_state.is_dir(),
            "本機 run／state／source-state 不完整")

    checks = {
        run / "runner.json": receipt["runner"]["sha256"],
        run / "control-history.jsonl": receipt["execution"]["control_history_sha256"],
        run / "driver.log": receipt["execution"]["driver_log_sha256"],
        run / "current.json": receipt["outputs"]["current_checkpoint_sha256"],
        source_state / "FD2.SAV": inputs["source_fd2_sav_sha256"],
        source_state / "FD2.TMP": inputs["source_fd2_tmp_sha256"],
        result_state / "FD2.SAV": receipt["outputs"]["result_fd2_sav_sha256"],
        result_state / "FD2.TMP": receipt["outputs"]["result_fd2_tmp_sha256"],
    }
    for checkpoint in receipt["checkpoints"]:
        checks[run / f"checkpoint-{checkpoint['seq']:04d}.json"] = checkpoint["sha256"]
    for path, expected in checks.items():
        require(path.is_file(), f"找不到產物 {path}")
        require(digest(path) == expected, f"產物雜湊不符：{path}")

    runner = json.loads((run / "runner.json").read_text(encoding="utf-8"))
    require(runner["dosgolem_commit"] == receipt["runner"]["commit"],
            "dosgolem commit 不符")
    require(runner["dosgolem_tracked_dirty_files"] == 0, "dosgolem 有追蹤中修改")
    require(runner["force_enemy_clear_declared"] is True and runner["lock_ally_hp"] is True,
            "runner 未宣告兩項修改路徑")
    require(runner["original_fd2_exe_sha256"] == inputs["fd2_exe_sha256"],
            "runner 原版 EXE 雜湊不符")

    before = json.loads((run / "checkpoint-0052.json").read_text(encoding="utf-8"))
    after = json.loads((run / "checkpoint-0053.json").read_text(encoding="utf-8"))
    require(alive(before, 0) == 17 and alive(before, 2) == 6,
            "清場前單位數不是 camp0=17、camp2=6")
    require(alive(after, 0) == 0, "清場後仍有 camp 0 單位存活")
    require(any("force-enemy-clear" in item and "17 筆" in item
                for item in after["state_injections"]), "checkpoint 0053 缺清場筆數")

    for path in sorted(run.glob("checkpoint-*.json")):
        checkpoint = json.loads(path.read_text(encoding="utf-8"))
        require(checkpoint.get("exe_sha256") == inputs["fd2_exe_sha256"],
                f"{path.name} 缺原版 EXE 雜湊")
        require(checkpoint.get("state_injections"), f"{path.name} 缺 state_injections")
        require(checkpoint.get("normal_player_path_verified") is False,
                f"{path.name} 未降級一般玩家路徑")

    driver_log = (run / "driver.log").read_text(encoding="utf-8")
    for phrase in ("await enemy_alive>=1 成立（實測 17）",
                   "force-enemy-clear=17->0（修改路徑）",
                   "await_ui=town 成立",
                   "town_save：FD2.SAV 已更新，大小 22987"):
        require(phrase in driver_log, f"driver.log 缺少：{phrase}")
    require((result_state / "FD2.SAV").stat().st_size ==
            receipt["outputs"]["result_fd2_sav_size"], "結果存檔大小不符")

    print("第四關修改路徑閉環收據與本機產物全部通過")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
