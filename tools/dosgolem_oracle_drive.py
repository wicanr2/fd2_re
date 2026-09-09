"""容器內驅動 dosgolem 的 FD2 原版 oracle：依宣告式控制序列逐步推進。

每一步寫入 run-dir 的 control.json，再等 current.json 的 control_seq 追上，
不用固定 sleep 猜。控制序列是 JSONL，每行 {"key": "...", "steps": N}；
key 為 up/down/left/right/enter/esc 或空字串（只前進不送鍵）。

三個選用欄位讓「一直按到某件事發生」不必展開成幾百行計畫：

* ``repeat``：本行重複 N 次。
* ``gate``：目前只有 ``kbd_empty``。BIOS 環形緩衝只有 15 格，固定速率送鍵在
  遊戲消化不及時會把它撐爆；設了這個閘門，該格若還有未取走的鍵就改送空鍵，
  等於「玩家看到畫面還沒吃掉上一次按鍵就不再按」。
* ``until``：目前只有 ``units_present``。單位陣列一有內容就結束整份計畫，
  用來量「從這裡到戰場要多久」而不必猜要排幾格。
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


def reached(name, current):
    if name == "units_present":
        return bool(current.get("units"))
    raise SystemExit(f"未知的 until 條件：{name}")


def main():
    first = state()
    print(f"checkpoint-0000 eip={first['eip']} view={first['view']}", flush=True)
    if not os.path.exists(PLAN):
        return 0
    with open(PLAN, encoding="utf-8") as handle:
        lines = [l.strip() for l in handle if l.strip() and not l.startswith("#")]
    plan = []
    for line in lines:
        command = json.loads(line)
        plan.extend([command] * int(command.get("repeat", 1)))
    for command in plan:
        current = state()
        until = command.get("until")
        if until and reached(until, current):
            print(f"until={until} 已成立，提前結束", flush=True)
            return 0
        key = command.get("key", "")
        if command.get("gate") == "kbd_empty" and current.get("kbd_pending", 0) > 0:
            key = ""
        seq = current["control_seq"] + 1
        body = json.dumps({
            "seq": seq,
            "key": key,
            "steps": int(command.get("steps", 3_000_000)),
        })
        tmp = os.path.join(RUN, "control.tmp")
        with open(tmp, "w", encoding="utf-8") as handle:
            handle.write(body)
        os.replace(tmp, os.path.join(RUN, "control.json"))
        deadline = time.time() + STEP_TIMEOUT
        # 輪詢間隔由短往長退避。固定 0.1 秒會讓細粒度追蹤的每一步都至少多等
        # 一次輪詢，整段時間由等待而非執行決定。
        poll = 0.0002
        while time.time() < deadline:
            try:
                current = state()
            except (json.JSONDecodeError, FileNotFoundError):
                time.sleep(poll)
                poll = min(poll * 2, 0.02)
                continue
            if current["control_seq"] >= seq:
                print(
                    f"seq={seq} key={key!r} eip={current['eip']} "
                    f"pending={current.get('kbd_pending')} "
                    f"reads={current.get('kbd_reads')} steps={current['steps']}",
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
