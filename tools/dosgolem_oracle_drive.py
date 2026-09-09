"""容器內驅動 dosgolem 的 FD2 原版 oracle：依宣告式控制序列逐步推進。

每一步寫入 run-dir 的 control.json，再等 current.json 的 control_seq 追上，
不用固定 sleep 猜。控制序列是 JSONL，每行 {"key": "...", "steps": N}；
key 為 up/down/left/right/enter/esc 或空字串（只前進不送鍵）。
"""

import json
import os
import sys
import time

RUN = "/out"
PLAN = "/plan.jsonl"
STEP_TIMEOUT = float(os.environ.get("FD2_ORACLE_STEP_TIMEOUT", "180"))


def state():
    with open(os.path.join(RUN, "current.json"), encoding="utf-8") as handle:
        return json.load(handle)


def main():
    first = state()
    print(f"checkpoint-0000 eip={first['eip']} view={first['view']}", flush=True)
    if not os.path.exists(PLAN):
        return 0
    with open(PLAN, encoding="utf-8") as handle:
        lines = [l.strip() for l in handle if l.strip() and not l.startswith("#")]
    for line in lines:
        command = json.loads(line)
        seq = state()["control_seq"] + 1
        body = json.dumps({
            "seq": seq,
            "key": command.get("key", ""),
            "steps": int(command.get("steps", 3_000_000)),
        })
        tmp = os.path.join(RUN, "control.tmp")
        with open(tmp, "w", encoding="utf-8") as handle:
            handle.write(body)
        os.replace(tmp, os.path.join(RUN, "control.json"))
        until = time.time() + STEP_TIMEOUT
        # 輪詢間隔由短往長退避。固定 0.1 秒會讓細粒度追蹤的每一步都至少多等
        # 一次輪詢，整段時間由等待而非執行決定。
        poll = 0.0002
        while time.time() < until:
            try:
                current = state()
            except (json.JSONDecodeError, FileNotFoundError):
                time.sleep(poll)
                poll = min(poll * 2, 0.02)
                continue
            if current["control_seq"] >= seq:
                print(
                    f"seq={seq} key={command.get('key', '')!r} "
                    f"eip={current['eip']} view={current['view']}",
                    flush=True,
                )
                break
            time.sleep(poll)
            poll = min(poll * 2, 0.02)
        else:
            print(f"控制序列 {seq} 等待逾時", file=sys.stderr)
            return 5
    return 0


sys.exit(main())
