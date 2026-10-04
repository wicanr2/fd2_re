#!/usr/bin/env python3
"""核對 #163 的正常原版三停點與同源 E1 全畫面；只在 Docker 執行。"""
import argparse
import hashlib
import json
from pathlib import Path

from PIL import Image


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def read(path):
    return json.loads(path.read_text())


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--prefix", type=Path, required=True)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args()
    assert Path("/.dockerenv").exists(), "只在 Docker 執行"
    original = args.original
    checkpoints = [read(original / f"checkpoint-{n:04d}.json") for n in (178, 179, 180, 181)]
    waits = [1447060369, 1447075147, 1447086515]
    returns = ["0x14030", "0x1409B", "0x140F1"]
    for point, step in zip(checkpoints, waits + [waits[-1] + 10000]):
        assert point["steps"] == step and point["runner"] == "dosgolem"
        assert point["exe_sha256"] == "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
        assert not point["state_injections"]
    hp = [c["units"][11]["hp"] for c in checkpoints]
    assert hp == [182, 182, 182, 184], hp
    rows = [json.loads(line) for line in (original / "eip-trace.jsonl").read_text().splitlines()]
    accepted = [r for r in rows if r["eip"] == "0x14018"]
    assert len(accepted) == 1 and accepted[0]["stack"][1] == "0xB"
    for step, ret in zip(waits, returns):
        matches = [r for r in rows if r["step"] == step and r["eip"] == "0x17AA9"]
        assert len(matches) == 1 and matches[0]["stack"][:2] == [ret, "0x1"]
    fixture = read(Path(str(args.prefix) + ".json"))
    assert fixture["checkpoint_sha256"] == digest(original / "checkpoint-0178.json")
    assert fixture["actor"] == 11 and fixture["before_hp"] == 182 and fixture["after_hp"] == 184
    assert fixture["three_real_draws_before_commit"]
    frames = [json.loads(line) for line in (original / "frames/frames.jsonl").read_text().splitlines()]
    selected = [next(r for r in frames if r["step"] == step) for step in waits]
    images = []
    for frame in selected:
        im = Image.open(original / "frames" / frame["file"])
        assert im.mode == "P" and im.size == (320, 200)
        assert hashlib.sha256(im.tobytes()).hexdigest() == frame["indexed_sha256"]
        images.append(im)
    candidates = []
    for terrain in range(4):
        for idle in range(3):
            comparisons = []
            for n, im in enumerate(images):
                path = Path(f"{args.prefix}-t{terrain}-i{idle}-f{n}.png")
                remake = Image.open(path)
                assert remake.mode == "P" and remake.size == (320, 200)
                indexed = sum(a != b for a, b in zip(im.tobytes(), remake.tobytes()))
                rgb = sum(a != b for a, b in zip(im.convert("RGB").getdata(), remake.convert("RGB").getdata()))
                comparisons.append({"frame": n, "original": selected[n]["file"], "remake": str(path), "indexed_different": indexed, "rgb_different": rgb, "remake_sha256": digest(path)})
            candidates.append({"terrain": terrain, "idle": idle, "frames": comparisons})
    passing = [c for c in candidates if all(f["indexed_different"] == f["rgb_different"] == 0 for f in c["frames"])]
    report = {"schema": 1, "issue": 163, "status": "passed" if passing else "failed", "original_runner": "dosgolem", "input_kind": "normal BIOS keys", "state_injections": 0, "accepted": accepted[0], "wait_steps": waits, "hp": hp, "checkpoint_sha256": fixture["checkpoint_sha256"], "trace_sha256": digest(original / "eip-trace.jsonl"), "passing_phases": [{"terrain": c["terrain"], "idle": c["idle"]} for c in passing], "candidates": candidates, "limit": "正常原版回復／同源raw E1夾具；合法相位枚舉，不宣稱時鐘同步或整章PLAYER-E2。"}
    args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": report["status"], "passing_phases": report["passing_phases"], "minimum_different": min(sum(f["indexed_different"] for f in c["frames"]) for c in candidates)}, ensure_ascii=False))
    return 0 if passing else 1


if __name__ == "__main__":
    raise SystemExit(main())
