#!/usr/bin/env python3
"""Docker 內量測選定動態片段的亮度變化；不把靜態等待當卡頓。"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess


def measure(path, start, duration, crop, threshold):
    filters = ([f"crop={crop}"] if crop else []) + [
        "scale=320:200:flags=area", "signalstats", "metadata=print:file=-"]
    command = ["ffmpeg", "-v", "error", "-threads", "1", "-ss", str(start),
               "-i", str(path), "-t", str(duration), "-an", "-filter_threads", "1",
               "-vf", ",".join(filters), "-f", "null", "/dev/null"]
    raw = subprocess.check_output(command, text=True)
    times, changes = [], []
    for line in raw.splitlines():
        match = re.search(r"pts_time:([0-9.e+-]+)", line)
        if match:
            times.append(float(match[1]))
        if line.startswith("lavfi.signalstats.YDIF="):
            changes.append(float(line.split("=", 1)[1]))
    if len(times) < 2 or len(times) != len(changes):
        raise ValueError("missing frame measurements")
    # First frame has no preceding frame in this bounded sample.
    changes = changes[1:]
    elapsed = times[-1] - times[0]
    held, longest = 0, 0
    for difference in changes:
        held = held + 1 if difference <= threshold else 0
        longest = max(longest, held)
    return {"start_seconds": start, "duration_seconds": duration, "crop": crop,
            "frames": len(times), "observed_fps": round(len(changes) / elapsed, 3),
            "change_frames_per_second": round(sum(v > threshold for v in changes) / elapsed, 3),
            "low_change_fraction": round(sum(v <= threshold for v in changes) / len(changes), 4),
            "longest_low_change_seconds": round(longest * elapsed / len(changes), 4),
            "threshold_mean_luma_difference": threshold,
            "meaning": "低變化可包含原生動畫停留或讀字等待；此量測不能單獨證明掉幀、遊戲速度或原版時序一致。"}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input", type=Path)
    parser.add_argument("--segment", action="append", required=True, help="start,duration")
    parser.add_argument("--crop", help="w:h:x:y，排除字幕與畫面外區域")
    parser.add_argument("--threshold", type=float, default=.12)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    assert Path("/.dockerenv").exists()
    segments = [tuple(map(float, v.split(","))) for v in args.segment]
    assert all(start >= 0 and duration > 0 for start, duration in segments)
    report = {"schema_version": 1, "kind": "fd2-video-cadence-diagnostic",
              "input": str(args.input),
              "sha256": hashlib.file_digest(args.input.open("rb"), "sha256").hexdigest(),
              "ffmpeg": subprocess.check_output(["ffmpeg", "-version"], text=True).splitlines()[0],
              "segments": [measure(args.input, a, b, args.crop, args.threshold) for a, b in segments]}
    args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
