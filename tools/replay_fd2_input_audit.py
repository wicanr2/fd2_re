#!/usr/bin/env python3
"""在 Docker／X11 內記錄普通鍵盤操作；不注入 FD2 遊戲狀態。"""

import argparse
import json
import pathlib
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--window", required=True)
    parser.add_argument("--steps", type=pathlib.Path, required=True)
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    if not pathlib.Path("/.dockerenv").exists():
        raise SystemExit("依專案規則，只能在 Docker 容器內執行。")
    steps = json.loads(args.steps.read_text())
    args.output.mkdir(parents=True, exist_ok=True)
    events = []
    started = time.monotonic()
    subprocess.run(["xdotool", "windowfocus", args.window], check=True)
    for index, step in enumerate(steps):
        key = step.get("key")
        event = {"step": index, "requested": step, "seconds": time.monotonic() - started}
        if key:
            subprocess.run(["xdotool", "keydown", key], check=True)
            try:
                time.sleep(float(step.get("hold", .1)))
            finally:
                subprocess.run(["xdotool", "keyup", key], check=True)
        time.sleep(float(step.get("wait", 1)))
        if step.get("shot"):
            name = step["shot"]
            if pathlib.Path(name).name != name or not name.endswith(".png"):
                raise SystemExit("截圖名稱必須是單一 PNG 檔名。")
            subprocess.run(["import", "-window", args.window, str(args.output / name)], check=True)
        event["completed_seconds"] = time.monotonic() - started
        events.append(event)
        (args.output / "inputs.json").write_text(json.dumps(events, ensure_ascii=False, indent=2) + "\n")


if __name__ == "__main__":
    main()
