#!/usr/bin/env python3
"""由 dosgolem／重製擷取與 parity 報告產生 storyBG 對白 E1 收據。"""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path


EXE_SIZE = 357074
EXE_MD5 = "b97caf2239a27a896069d03549d96e1e"
EXE_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
SHA_KEYS = ("original_capture_sha256", "remake_capture_sha256")


def read_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def validate_receipt(receipt: dict) -> None:
    require(receipt.get("schema_version") == 1, "schema_version 必須是 1")
    require(receipt.get("kind") == "fd2_storybg_dialogue_parity_receipt", "kind 不符")
    require(receipt.get("original_runner") == "dosgolem", "original_runner 必須是 dosgolem")
    require(receipt.get("evidence_level") == "RUNTIME-E1", "證據層級不得冒稱 E2")
    exe = receipt.get("executable", {})
    require(exe.get("size") == EXE_SIZE, "FD2.EXE size 不符")
    require(exe.get("md5") == EXE_MD5, "FD2.EXE MD5 不符")
    require(exe.get("sha256") == EXE_SHA256, "FD2.EXE SHA-256 不符")
    provenance = receipt.get("provenance", {})
    require(provenance.get("original_runner") == "dosgolem apps/fd2/cmd/oracle", "原版 runner 不是 dosgolem")
    require(provenance.get("dosgolem_tracked_dirty_files") == 0, "dosgolem 有未提交修改")
    require(provenance.get("state_injections") == [], "正式收據不得有狀態注入")
    for key in ("oracle_plan_sha256", "parity_scenario_sha256"):
        require(isinstance(provenance.get(key), str) and len(provenance[key]) == 64, f"{key} 不是 SHA-256")
    for key in ("dosgolem_commit", "remake_commit"):
        require(isinstance(provenance.get(key), str) and len(provenance[key]) == 40, f"{key} 不是完整 Git commit")
    state = receipt.get("state_alignment", {})
    require(state.get("classification") == "same-state", "狀態必須是 same-state")
    require(state.get("actor_mismatches") == [], "角色配置不一致")
    require(state.get("original", {}).get("camera_grid") == [3, 20], "原版鏡頭不符")
    require(state.get("remake", {}).get("camera_grid") == [3, 20], "重製鏡頭不符")
    require(state.get("original", {}).get("cursor") == [8, 21], "原版焦點不符")
    require(state.get("remake", {}).get("cursor") == [8, 21], "重製焦點不符")
    phase_crosscheck = receipt.get("phase_crosscheck", [])
    require(len(phase_crosscheck) == 6, "必須保存原版兩邊界乘重製三相位的六組比較")
    require(len({row.get("diff_sha256") for row in phase_crosscheck}) == 1, "六組差異遮罩不一致，動畫相位尚未鎖定")
    comparison = receipt.get("image_comparison", {})
    require(comparison.get("remake_normalization") == "nearest_2x", "重製畫面正規化不符")
    require(comparison.get("total_pixels") == 64000, "canonical 畫布不是 320x200")
    regions = comparison.get("regions", {})
    for name in ("dialogue_overlay", "viewport_top_border", "viewport_left_border", "viewport_right_border"):
        require(regions.get(name, {}).get("equal_ratio") == 1, f"{name} 未逐像素一致")
    for key in SHA_KEYS:
        require(isinstance(comparison.get(key), str) and len(comparison[key]) == 64, f"{key} 無效")
    lifecycle = receipt.get("post_input_lifecycle", {})
    require(lifecycle.get("input") == "Enter", "生命週期輸入不是 Enter")
    require(lifecycle.get("original", {}).get("cursor_after") == [7, 5], "原版輸入後焦點不符")
    require(lifecycle.get("original", {}).get("camera_grid_after") == [3, 4], "原版輸入後鏡頭不符")
    require(lifecycle.get("remake", {}).get("regression_test") == "TestStoryBGFirstDialogueEnterAdvancesToKing", "缺少重製生命週期回歸")
    require(receipt.get("limitations", {}).get("player_e2_claimed") is False, "不得冒稱 PLAYER-E2")


def build_receipt(args: argparse.Namespace) -> dict:
    original_dir = args.original_dir.resolve()
    parity_path = args.parity_report.resolve()
    remake_png = args.remake_png.resolve()
    remake_state_path = args.remake_state.resolve()
    oracle_plan = args.oracle_plan.resolve()
    parity_scenario = args.parity_scenario.resolve()

    parity = read_json(parity_path)
    runner = read_json(original_dir / "runner.json")
    before = read_json(original_dir / "checkpoint-0012.json")
    post = read_json(original_dir / "checkpoint-0014.json")
    remake = read_json(remake_state_path)

    exe = parity.get("executable", {})
    require((exe.get("size"), exe.get("md5"), exe.get("sha256")) == (EXE_SIZE, EXE_MD5, EXE_SHA256), "parity 的 FD2.EXE 身分不符")
    require(parity.get("state") == "same-state" and parity.get("original_runner") == "dosgolem", "parity 未宣告 dosgolem same-state")
    require(parity.get("input_sha256") == sha256(parity_scenario), "parity scenario 雜湊不符")
    require(parity.get("original_capture_sha256") == sha256(original_dir / "checkpoint-0012.png"), "原版 PNG 雜湊不符")
    require(parity.get("remake_capture_sha256") == sha256(remake_png), "重製 PNG 雜湊不符")
    require(runner.get("dosgolem_commit") == parity.get("dosgolem_commit"), "dosgolem commit 不一致")
    require(runner.get("dosgolem_tracked_dirty_files") == 0 and not runner.get("lock_ally_hp"), "oracle 來源不乾淨或使用作弊")
    require(before.get("input_kind") == "normal BIOS keys" and before.get("state_injections") == [], "原版擷取不是未注入的正常輸入")
    require(before.get("view", {}).get("camera_x") == 3 and before.get("view", {}).get("camera_y") == 20, "原版第一句鏡頭不符")
    require(before.get("view", {}).get("cursor_x") == 8 and before.get("view", {}).get("cursor_y") == 21, "原版第一句焦點不符")
    require(post.get("input_kind") == "normal BIOS keys" and post.get("state_injections") == [], "原版第二次輸入不是正常路徑")
    require(post.get("kbd_reads") == before.get("kbd_reads") + 1, "Enter 沒有跨越一個 BIOS 讀鍵邊界")
    require(post.get("view", {}).get("camera_x") == 3 and post.get("view", {}).get("camera_y") == 4, "原版輸入後鏡頭不符")
    require(post.get("view", {}).get("cursor_x") == 7 and post.get("view", {}).get("cursor_y") == 5, "原版輸入後焦點不符")

    story = remake.get("story", {})
    dialogue = remake.get("dialogue", {})
    require(remake.get("campaign_node") == "story_ch00_handler", "重製節點不符")
    require(story.get("camera_grid") == [3, 20] and remake.get("cursor") == [8, 21], "重製第一句鏡頭／焦點不符")
    require((story.get("beat_index"), story.get("beat_op"), story.get("beat_source")) == (4, "dialog", "0x32382"), "重製 beat 身分不符")
    require((dialogue.get("speaker"), dialogue.get("upper"), dialogue.get("native"), dialogue.get("source_dat"), dialogue.get("string_index"), dialogue.get("utterance")) == (0, False, True, "FDTXT_033", 0, 0), "重製對白身分不符")

    actor_mismatches = []
    original_units = before.get("units", [])
    remake_actors = story.get("actors", [])
    require(len(original_units) == len(remake_actors) == 21, "角色數不是 21")
    for unit, actor in zip(original_units, remake_actors):
        observed = (unit.get("index"), unit.get("fig"), unit.get("x"), unit.get("y"))
        expected = (actor.get("slot"), actor.get("fig"), actor.get("x"), actor.get("y"))
        if observed != expected:
            actor_mismatches.append({"original": observed, "remake": expected})

    frame_sequence = []
    for seq in range(12, 17):
        png_path = original_dir / f"checkpoint-{seq:04d}.png"
        state_path = original_dir / f"checkpoint-{seq:04d}.json"
        state_doc = read_json(state_path)
        frame_sequence.append({
            "control_seq": seq,
            "steps": state_doc["steps"],
            "input_kind": state_doc["input_kind"],
            "keyboard_reads": state_doc["kbd_reads"],
            "camera_grid": [state_doc["view"]["camera_x"], state_doc["view"]["camera_y"]],
            "cursor": [state_doc["view"]["cursor_x"], state_doc["view"]["cursor_y"]],
            "png_sha256": sha256(png_path),
            "state_sha256": sha256(state_path),
        })

    phase_crosscheck = []
    for original_seq in (12, 13):
        for remake_frame in (362, 370, 378):
            phase_report_path = original_dir / f"phase-o{original_seq}-r{remake_frame}.json"
            phase_diff_path = original_dir / f"phase-o{original_seq}-r{remake_frame}.png"
            phase_report = read_json(phase_report_path)
            result = phase_report["comparison"]
            require(phase_report.get("state") == "same-state" and phase_report.get("original_runner") == "dosgolem", "相位交叉比較不是 dosgolem same-state")
            phase_crosscheck.append({
                "original_control_seq": original_seq,
                "remake_frame": remake_frame,
                "remake_sprite_phase": (remake_frame // 8) % 3,
                "equal_pixels": result["equal_pixels"],
                "total_pixels": result["total_pixels"],
                "mean_abs_rgb": result["mean_abs_rgb"],
                "diff_sha256": sha256(phase_diff_path),
            })
    require(len({row["diff_sha256"] for row in phase_crosscheck}) == 1, "六組差異遮罩不一致")

    comparison = parity["comparison"]
    receipt = {
        "schema_version": 1,
        "kind": "fd2_storybg_dialogue_parity_receipt",
        "original_runner": "dosgolem",
        "scenario": "storybg-audience-first-dialogue",
        "generated_date": args.generated_date,
        "evidence_level": "RUNTIME-E1",
        "executable": exe,
        "provenance": {
            "original_runner": runner["runner"],
            "dosgolem_commit": parity["dosgolem_commit"],
            "dosgolem_tracked_dirty_files": runner["dosgolem_tracked_dirty_files"],
            "remake_commit": parity["remake_commit"],
            "oracle_plan": "docs/data/ui-traces/storybg-dialogue-original-input-e1.jsonl",
            "oracle_plan_sha256": sha256(oracle_plan),
            "parity_scenario": "docs/data/ui-traces/storybg-dialogue-parity-input-e1.json",
            "parity_scenario_sha256": sha256(parity_scenario),
            "state_injections": before["state_injections"],
            "original_control_boundary": 12,
            "remake_frame": remake["frame"],
            "remake_capture_environment": {
                "docker_image": "fd2-go-test-local:20260909",
                "campaign": "assets/scenarios/campaign_full.json",
                "asset_pack": "remake/generated-assets/fd2-original-b97caf22",
                "shot_deterministic": True,
                "shot_frame": remake["frame"],
                "sprite_phase_formula": "(frame / 8) % 3",
                "sprite_phase": (remake["frame"] // 8) % 3,
                "output_canvas": [640, 400],
                "canonical_canvas": [320, 200],
                "normalization": parity["remake_normalization"],
            },
            "comparison_tool": "dosgolem apps/fd2/cmd/parity",
        },
        "state_alignment": {
            "classification": "same-state",
            "original": {
                "control_seq": before["control_seq"],
                "steps": before["steps"],
                "camera_grid": [before["view"]["camera_x"], before["view"]["camera_y"]],
                "cursor": [before["view"]["cursor_x"], before["view"]["cursor_y"]],
                "keyboard_reads": before["kbd_reads"],
            },
            "remake": {
                "frame": remake["frame"],
                "campaign_node": remake["campaign_node"],
                "camera_grid": story["camera_grid"],
                "cursor": remake["cursor"],
                "beat_index": story["beat_index"],
                "beat_op": story["beat_op"],
                "beat_source": story["beat_source"],
                "dialogue": dialogue,
            },
            "actor_count": len(original_units),
            "actor_mismatches": actor_mismatches,
        },
        "frame_sequence": frame_sequence,
        "phase_crosscheck": phase_crosscheck,
        "image_comparison": {
            "original_capture": parity["original_capture"],
            "original_capture_sha256": parity["original_capture_sha256"],
            "remake_capture": parity["remake_capture"],
            "remake_capture_sha256": parity["remake_capture_sha256"],
            "remake_normalization": parity["remake_normalization"],
            "equal_pixels": comparison["equal_pixels"],
            "total_pixels": comparison["total_pixels"],
            "equal_ratio": comparison["equal_ratio"],
            "mean_abs_rgb": comparison["mean_abs_rgb"],
            "diff_box": comparison.get("diff_box"),
            "regions": parity["regions"],
        },
        "post_input_lifecycle": {
            "input": "Enter",
            "original": {
                "control_seq": post["control_seq"],
                "steps": post["steps"],
                "keyboard_reads_before": before["kbd_reads"],
                "keyboard_reads_after": post["kbd_reads"],
                "camera_grid_after": [post["view"]["camera_x"], post["view"]["camera_y"]],
                "cursor_after": [post["view"]["cursor_x"], post["view"]["cursor_y"]],
            },
            "remake": {
                "input_owner": "handleNativeStoryInput(nativeStoryInput{enter: true})",
                "regression_test": "TestStoryBGFirstDialogueEnterAdvancesToKing",
                "expected_next_camera_grid": [3, 4],
                "expected_next_cursor": [7, 5],
            },
        },
        "conclusions": {
            "confirmed": [
                "storyBG 第一段對白與重製端位於相同節點、角色配置、鏡頭、焦點、對白來源及動畫相位。",
                "對白框與上、左、右視窗邊界逐像素一致。",
                "Enter 經正常讀鍵邊界將原版焦點推進至下一位說話者；重製端由 production 輸入 owner 的回歸測試固定同一生命週期。",
            ],
            "known_difference": "全畫面差異只落在 y=21..111 的上半部人物 sprite，未進入 y=112..199 的對白 overlay；原版兩個穩定邊界乘重製三個 sprite 相位的六組差異遮罩完全相同，因此不是以不同動畫時點硬湊出的差異。",
        },
        "limitations": {
            "player_e2_claimed": False,
            "reason": "原版是未修改 normal START 玩家路徑；重製證據使用決定性截圖鉤子並由 runtime 回歸補上輸入後生命週期，因此只宣告 RUNTIME-E1，不外推完整 PLAYER-E2。",
            "not_claimed": ["全戰役 storyBG 逐幀一致", "所有說話者與上下框分支已逐場對拍", "上半部人物動畫逐像素一致"],
        },
    }
    validate_receipt(receipt)
    return receipt


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--original-dir", type=Path, required=True)
    parser.add_argument("--remake-png", type=Path, required=True)
    parser.add_argument("--remake-state", type=Path, required=True)
    parser.add_argument("--parity-report", type=Path, required=True)
    parser.add_argument("--oracle-plan", type=Path, required=True)
    parser.add_argument("--parity-scenario", type=Path, required=True)
    parser.add_argument("--generated-date", required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    receipt = build_receipt(args)
    args.output.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
