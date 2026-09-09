"""Docker 內以普通 START／Enter 擷取王宮前兩句，含逐字過程。"""
import argparse
import json
import os
import pathlib
import signal
import subprocess
import time

parser = argparse.ArgumentParser()
parser.add_argument("appimage")
parser.add_argument("output")
parser.add_argument("--prefix-pages", type=int, default=0)
args = parser.parse_args()
if not 0 <= args.prefix_pages <= 120:
    parser.error("逐頁擷取上限為120")
root = pathlib.Path(args.output)
root.mkdir(parents=True, exist_ok=True)
assert root.stat().st_uid == os.getuid()
log = (root / "remake-app.log").open("wb")
env = dict(os.environ, FD2_CUTSCENE_LOG="1", FD2_INPUT_AUDIT_LOG=str(root / "remake-state.jsonl"))
display_deadline = time.monotonic() + 10
while subprocess.run(["xdotool", "getdisplaygeometry"], capture_output=True).returncode:
    if time.monotonic() >= display_deadline:
        raise RuntimeError("驗證環境 X11 顯示器尚未就緒")
    time.sleep(.05)
p = subprocess.Popen([args.appimage, "--appimage-extract-and-run"], stdout=log, stderr=log,
                     env=env, start_new_session=True)
events = []

def wait_for(fn, seconds):
    until = time.monotonic() + seconds
    while time.monotonic() < until:
        if p.poll() is not None:
            raise RuntimeError(f"AppImage 提前退出：{p.returncode}")
        value = fn()
        if value:
            return value
        time.sleep(.025)
    raise RuntimeError("等待正常玩家狀態逾時")

def windows():
    r = subprocess.run(["xdotool", "search", "--onlyvisible", "--name", "FD2"], capture_output=True, text=True)
    return r.stdout.split() if r.returncode == 0 else None

try:
    wid = wait_for(windows, 30)[-1]
    subprocess.run(["xdotool", "windowfocus", wid], check=True)
    def key():
        events.append({"time": time.time(), "key": "Return"})
        subprocess.run(["xdotool", "keydown", "Return"], check=True)
        time.sleep(.08)
        subprocess.run(["xdotool", "keyup", "Return"], check=True)

    def shot(name):
        subprocess.run(["import", "-window", wid, str(root / name)], check=True, timeout=10)

    next_key = time.monotonic() + 4
    until = time.monotonic() + 120
    while "op=dialog source=0x32382" not in (root / "remake-app.log").read_text(errors="replace"):
        if p.poll() is not None or time.monotonic() > until:
            raise RuntimeError("正常 START 未到王宮第一句")
        if time.monotonic() >= next_key:
            key()
            next_key = time.monotonic() + 3
        time.sleep(.025)
    if args.prefix_pages:
        def latest_state():
            path = root / "remake-state.jsonl"
            if not path.exists():
                return None
            lines = path.read_text().splitlines()
            if not lines:
                return None
            for line in reversed(lines):
                try:
                    return json.loads(line)
                except json.JSONDecodeError:
                    continue
            return None

        last_frame = -120
        pages = []
        for page in range(args.prefix_pages):
            def ready():
                state = latest_state()
                if state and state.get("error"):
                    raise RuntimeError(state["error"])
                return state if state and state["frame"] >= last_frame + 60 and (
                    state["dialogues"] > 0 or state.get("node", "").startswith("battle_")) else None
            state = wait_for(ready, 60)
            if state.get("node", "").startswith("battle_"):
                break
            time.sleep(1.25)
            name = f"dialogue-{page:03d}.png"
            shot(name)
            pages.append({"index": page, "file": name, "state": latest_state()})
            (root / "pages.json").write_text(json.dumps(pages, ensure_ascii=False, indent=2) + "\n")
            if page + 1 < args.prefix_pages:
                key()
                last_frame = latest_state()["frame"]
        raise SystemExit(0)
    for i in range(30):
        shot(f"first-flow-{i:02d}.png")
        time.sleep(.025)
    time.sleep(2)
    shot("remake-first.png")
    key()
    wait_for(lambda: "line=1 count=" in (root / "remake-app.log").read_text(errors="replace"), 10)
    for i in range(30):
        shot(f"second-flow-{i:02d}.png")
        time.sleep(.025)
    time.sleep(2)
    shot("remake-second.png")
finally:
    (root / "remake-inputs.json").write_text(json.dumps(events, indent=2) + "\n")
    if p.poll() is None:
        os.killpg(p.pid, signal.SIGTERM)
        try:
            p.wait(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(p.pid, signal.SIGKILL)
            p.wait()
    log.close()
