#!/usr/bin/env python3
"""Issue #29：固定正常 END 影格的完整索引及 RGB 比較。只在 Docker 執行。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def verify(original, prefix):
    runner = json.loads((original / "runner.json").read_text())
    checkpoint = json.loads((original / "checkpoint-0196.json").read_text())
    expected = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
    if checkpoint["exe_sha256"] != expected or runner["original_fd2_exe_sha256"] != expected:
        raise ValueError("原版 EXE 雜湊不符")
    if checkpoint["state_injections"]:
        raise ValueError("固定普通 END 收據含章內注入")
    trace = [json.loads(line) for line in (original / "eip-trace.jsonl").read_text().splitlines()]
    trace = [row for row in trace if row["control_seq"] == 197]
    metadata = {row["index"]: row for row in map(json.loads, (original / "frames/frames.jsonl").read_text().splitlines())}
    returns = ("0x1A32F", "0x1A3EF", "0x1A4B8")
    frame_indices = (925, 926, 927)
    sources = []
    images = []
    for index, ret in zip(frame_indices, returns):
        row = metadata[index]
        if not any(t["eip"] == "0x17AA9" and t["stack"][0] == ret and t["step"] == row["step"] for t in trace):
            raise ValueError(f"影格 {index} 缺少原版 caller {ret}")
        path = original / "frames" / row["file"]
        img = Image.open(path)
        if img.mode != "P" or img.size != (320, 200) or hashlib.sha256(img.tobytes()).hexdigest() != row["indexed_sha256"]:
            raise ValueError("原版索引影格形態／雜湊不符")
        sources.append({"index": index, "return": ret, "step": row["step"], "file": str(path), "sha256": sha(path), "indexed_sha256": row["indexed_sha256"]})
        images.append(img)
    candidates = []
    for terrain in range(4):
        for idle in range(3):
            result = {"terrain_phase": terrain, "idle_cycle": idle, "frames": []}
            for stage, (source, target) in enumerate(zip(sources, images)):
                path = Path(f"{prefix}-t{terrain}-i{idle}-f{stage}.png")
                actual = Image.open(path)
                if actual.mode != "P" or actual.size != target.size:
                    raise ValueError("重製索引影格形態不符")
                result["frames"].append({"original_index": source["index"], "remake_file": str(path), "remake_sha256": sha(path),
                    "indexed_diff_pixels": sum(a != b for a, b in zip(target.tobytes(), actual.tobytes())),
                    "rgb_diff_pixels": sum(a != b for a, b in zip(target.convert("RGB").getdata(), actual.convert("RGB").getdata()))})
            candidates.append(result)
    passed = [c for c in candidates if all(f["indexed_diff_pixels"] == f["rgb_diff_pixels"] == 0 for f in c["frames"])]
    cue = [row for row in trace if row["eip"] == "0x1A3DF"]
    if len(cue) != 1 or cue[0]["stack"][1:3] != ["0x4", "0x1"]:
        raise ValueError("正常三候選 END 缺單次 sample4")
    return {"schema_version": 1, "kind": "fd2_end_recovery_full_frame_parity", "status": "passed" if passed else "failed",
        "original_runner": "dosgolem", "runner_sha256": sha(original / "runner.json"), "original_exe_sha256": expected,
        "trace_sha256": sha(original / "eip-trace.jsonl"), "checkpoint_sha256": sha(original / "checkpoint-0196.json"),
        "runtime_probe_sha256": sha(Path(f"{prefix}.json")), "original_frames": sources,
        "frame_size": [320, 200], "mask": None, "phase_method": "同一 terrain/idle 相位組合比較全部三張；合法狀態列舉，沒有同步兩側時間",
        "passing_phases": [{"terrain_phase": c["terrain_phase"], "idle_cycle": c["idle_cycle"]} for c in passed], "candidates": candidates,
        "sample": {"index": 4, "loop_count": 1, "call_count": len(cue)},
        "limit": "原版固定建構槽正常 BIOS END；重製正常 LOAD/出戰後匯入 seq196 raw 狀態的 E1 夾具。原版像素只作比較目標。未宣稱全流程 E2、wall-clock 或人耳音色確認。"}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--prefix", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    receipt = verify(args.original, args.prefix)
    args.output.write_text(json.dumps(receipt, ensure_ascii=False, indent=2) + "\n")
    print(receipt["status"], receipt["passing_phases"])
    if receipt["status"] != "passed":
        raise SystemExit(1)


if __name__ == "__main__":
    main()
