"""容器內驅動 dosgolem 的 FD2 原版 oracle：依宣告式控制序列逐步推進。

每一步寫入 run-dir 的 control.json，再等 current.json 的 control_seq 追上，
不用固定 sleep 猜。控制序列是 JSONL，每行一個命令物件。

## 開環命令（送鍵）

``{"key": "enter", "steps": 3000000}``；key 為 up/down/left/right/enter/esc
或空字串（只前進不送鍵）。三個選用欄位：

* ``repeat``：本行重複 N 次。
* ``gate``：目前只有 ``kbd_empty``。BIOS 環形緩衝只有 15 格，固定速率送鍵在
  遊戲消化不及時會把它撐爆；設了這個閘門，該格若還有未取走的鍵就改送空鍵，
  等於「玩家看到畫面還沒吃掉上一次按鍵就不再按」。
* ``until``：目前只有 ``units_present``。單位陣列一有內容就結束整份計畫。

## 閉環命令（讀狀態再決定送什麼）

一整場戰鬥用方向鍵次數寫死是寫不出來的：游標起點每回合不同，走位還會被地形
與可達範圍改變。這兩個命令改成看 `current.json` 的實際狀態決定下一鍵。

* ``{"goto": [x, y]}``：把地圖游標移到該格。每送一鍵就重讀 `view.cursor_x`／
  `cursor_y` 確認有動；連續不動視為撞邊界或該方向走不了，直接放棄並回報，
  不會無聲地把後面的按鍵送到錯的地方。
* ``{"engage": [x, y]}``：選取該格的我方單位，推進到貼著敵人的格，貼上就攻擊。
  候選格由「與某敵人相鄰且未被佔用」產生，離單位近的先試；移動有沒有生效看
  單位座標變了沒，不看送了幾個鍵。
* ``{"sweep_round": true}``：把這一回合所有未行動的我方單位依序接戰。
* ``{"sweep_battle": true, "rounds": 30}``：一路打到敵方全滅。
* ``{"await": "round>=2"}``：反覆只前進不送鍵，直到條件成立。可用變數：
  ``round``（`view.round`）、``enemy_alive``／``ally_alive``（依 camp 數存活
  單位）、``cursor_x``／``cursor_y``、``steps``。比較運算子 ``>= <= == != > <``。

兩者都吃 ``steps``（每一格的指令數）與 ``max``（最多幾格，逾時即失敗）。
失敗一律非零離開；盲目繼續會產生「看起來跑完了但走錯路」的收據。
"""

import json
import os
import re
import sys
import time

RUN = "/out"
PLAN = "/plan.jsonl"
STEP_TIMEOUT = float(os.environ.get("FD2_ORACLE_STEP_TIMEOUT", "180"))

# 陣營編碼取自實際收據：第一關我方單位是 camp 2、敵方是 camp 0。
ALLY_CAMP = 2
ENEMY_CAMP = 0

CONDITION = re.compile(r"^\s*([a-z_]+)\s*(>=|<=|==|!=|>|<)\s*(-?\d+)\s*$")


def state():
    with open(os.path.join(RUN, "current.json"), encoding="utf-8") as handle:
        return json.load(handle)


def alive(current, camp):
    return sum(1 for u in current.get("units", [])
               if u.get("hp", 0) > 0 and u.get("camp") == camp)


def measure(current, name):
    view = current.get("view", {}) or {}
    if name == "round":
        return int(view.get("round", 0))
    if name == "cursor_x":
        return int(view.get("cursor_x", -1))
    if name == "cursor_y":
        return int(view.get("cursor_y", -1))
    if name == "enemy_alive":
        return alive(current, ENEMY_CAMP)
    if name == "ally_alive":
        return alive(current, ALLY_CAMP)
    if name == "steps":
        return int(current.get("steps", 0))
    raise SystemExit(f"未知的狀態變數：{name}")


def holds(current, expression):
    match = CONDITION.match(expression)
    if not match:
        raise SystemExit(f"看不懂的條件：{expression!r}")
    name, operator, want = match.group(1), match.group(2), int(match.group(3))
    got = measure(current, name)
    return {
        ">=": got >= want, "<=": got <= want, "==": got == want,
        "!=": got != want, ">": got > want, "<": got < want,
    }[operator], got


def reached(name, current):
    if name == "units_present":
        return bool(current.get("units"))
    raise SystemExit(f"未知的 until 條件：{name}")


def send(key, steps):
    """送一格控制並等它被消化完，回傳消化後的狀態。"""
    current = state()
    seq = current["control_seq"] + 1
    body = json.dumps({"seq": seq, "key": key, "steps": int(steps)})
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
            return seq, current
        time.sleep(poll)
        poll = min(poll * 2, 0.02)
    print(f"控制序列 {seq} 等待逾時", file=sys.stderr)
    raise SystemExit(5)


def report(seq, key, current, note=""):
    view = current.get("view", {}) or {}
    print(
        f"seq={seq} key={key!r} eip={current['eip']} "
        f"pending={current.get('kbd_pending')} reads={current.get('kbd_reads')} "
        f"steps={current['steps']} cursor=({view.get('cursor_x')},{view.get('cursor_y')}) "
        f"round={view.get('round')} ally={alive(current, ALLY_CAMP)} "
        f"enemy={alive(current, ENEMY_CAMP)}{note}",
        flush=True,
    )


def do_goto(command):
    target_x, target_y = command["goto"]
    steps = int(command.get("steps", 2_000_000))
    budget = int(command.get("max", 80))
    stuck = 0
    for _ in range(budget):
        current = state()
        view = current.get("view", {}) or {}
        x, y = int(view.get("cursor_x", -1)), int(view.get("cursor_y", -1))
        if (x, y) == (target_x, target_y):
            return True
        # 先走水平再走垂直；哪一軸先動不影響終點，固定順序讓收據可重現。
        if x != target_x:
            key = "right" if target_x > x else "left"
        else:
            key = "down" if target_y > y else "up"
        if current.get("kbd_pending", 0) > 0:
            key = ""
        seq, current = send(key, steps)
        view = current.get("view", {}) or {}
        moved = (int(view.get("cursor_x", -1)), int(view.get("cursor_y", -1))) != (x, y)
        report(seq, key, current, f" goto=({target_x},{target_y})")
        # 送了實鍵卻沒動：撞地圖邊界，或這一格的模式根本不吃方向鍵。
        stuck = 0 if (moved or key == "") else stuck + 1
        if stuck >= 3:
            print(f"goto ({target_x},{target_y}) 卡在 ({x},{y})：連送三次 {key} 游標不動",
                  file=sys.stderr)
            return False
    print(f"goto ({target_x},{target_y}) 用完 {budget} 格仍未到位", file=sys.stderr)
    return False


def do_await(command):
    expression = command["await"]
    steps = int(command.get("steps", 10_000_000))
    budget = int(command.get("max", 120))
    key = command.get("key", "")
    for _ in range(budget):
        current = state()
        ok, got = holds(current, expression)
        if ok:
            print(f"await {expression} 成立（實測 {got}）", flush=True)
            return True
        send_key = "" if (key and current.get("kbd_pending", 0) > 0) else key
        seq, current = send(send_key, steps)
        report(seq, send_key, current, f" await={expression}")
    print(f"await {expression} 在 {budget} 格內未成立", file=sys.stderr)
    return False


def unit_at(current, x, y):
    for u in current.get("units", []):
        if u.get("hp", 0) > 0 and u.get("x") == x and u.get("y") == y:
            return u
    return None


def acted(unit):
    """record `+5` 的 bit7 是「這個單位本回合已行動」（doc 31 §6 已證實）。"""
    raw = unit.get("raw_hex") or ""
    if len(raw) < 12:
        return False
    return bool(int(raw[10:12], 16) & 0x80)


def side(current, camp):
    return [u for u in current.get("units", []) if u.get("hp", 0) > 0 and u.get("camp") == camp]


def distance(a, b):
    return abs(a[0] - b[0]) + abs(a[1] - b[1])


def occupied_cells(current):
    return {(u["x"], u["y"]) for u in current.get("units", []) if u.get("hp", 0) > 0}


def step_path(origin, target):
    """origin 到 target 的逐格路徑（先走 x 再走 y），不含 origin、含 target。"""
    x, y = origin
    path = []
    while (x, y) != tuple(target):
        if x != target[0]:
            x += 1 if target[0] > x else -1
        else:
            y += 1 if target[1] > y else -1
        path.append((x, y))
    return path


def engage_targets(current, origin, typical_move=6):
    """候選落腳格，由「最值得試」排到「最不值得試」。

    兩段：先試貼著敵人的格（走到就能打），再試沿路徑逼近的格。移動成本受地形
    影響，狀態層看不到可達範圍集合，所以可達與否一律交給實際送 enter 之後看
    單位座標有沒有變來裁決——猜不到就試，試不成就換下一格。

    逼近格按「與 typical_move 的差」排序而不是由遠到近：第一關的移動力是 5～7，
    從六格外開始試，命中率最高；由最遠開始試會白花好幾次試探。
    """
    occupied = occupied_cells(current)
    enemies = side(current, ENEMY_CAMP)
    if not enemies:
        return []
    cells, seen = [], set()

    # 貼敵格離單位比移動力還遠就不試：每次試探要送十幾個鍵、跑兩千多萬指令，
    # 拿去試一定走不到的格子是純浪費。
    adjacent = {}
    for enemy in enemies:
        for dx, dy in ((1, 0), (-1, 0), (0, 1), (0, -1)):
            cell = (enemy["x"] + dx, enemy["y"] + dy)
            if cell[0] < 0 or cell[1] < 0 or cell in occupied:
                continue
            reach = distance(cell, origin)
            if reach > typical_move:
                continue
            adjacent[cell] = min(adjacent.get(cell, 1 << 30), reach)
    for cell, _ in sorted(adjacent.items(), key=lambda kv: (kv[1], kv[0])):
        cells.append(cell)
        seen.add(cell)

    nearest = min(enemies, key=lambda e: distance((e["x"], e["y"]), origin))
    path = step_path(origin, (nearest["x"], nearest["y"]))[:-1]
    approach = [c for c in path if c not in occupied and c not in seen]
    def rank(cell):
        reach = distance(cell, origin)
        # 移動力內的由遠而近試（走最多算最多）；超出的排到最後當保險。
        return (0, typical_move - reach) if reach <= typical_move else (1, reach)

    approach.sort(key=rank)
    return cells + approach


def settle(steps, count):
    """只前進不送鍵，讓動畫或演出跑完。"""
    for _ in range(count):
        seq, current = send("", steps)
    return current


def stand_by(steps, note=""):
    """在指令環上選「待機」結束這個單位的行動。

    原版四向是 ↑0 攻擊／←1 法術／→2 物品／↓3 待機（`0x18D8C` 的 switch 釘死）。
    只移動不待機的話這個單位的 record `+5` bit7 不會設起來，下一輪掃描又會選到
    它，回合永遠推不掉。
    """
    seq, current = send("down", steps)
    report(seq, "down", current, f" standby{note}")
    seq, current = send("enter", max(steps, 5_000_000))
    report(seq, "enter", current, f" standby{note}")
    return settle(steps, 4)

def do_engage(command):
    """選取一個我方單位，推進到貼著敵人的格，貼上就攻擊。

    每一步都回讀狀態裁決：移動有沒有生效看單位座標變了沒，不看送了幾個鍵。
    盲送方向鍵在多回合戰鬥裡必然錯位——起點每回合不同，地形還會改變成本。
    """
    ux, uy = command["engage"]
    steps = int(command.get("steps", 2_000_000))
    tries = int(command.get("tries", 10))
    current = state()
    unit = unit_at(current, ux, uy)
    if unit is None:
        print(f"engage ({ux},{uy}) 沒有存活單位", file=sys.stderr)
        return False
    if unit.get("camp") != ALLY_CAMP:
        print(f"engage ({ux},{uy}) 不是我方單位（camp={unit.get('camp')}）", file=sys.stderr)
        return False
    if acted(unit):
        print(f"engage ({ux},{uy}) 本回合已行動，跳過", flush=True)
        return True
    if not do_goto({"goto": [ux, uy], "steps": steps, "max": command.get("max", 80)}):
        return False
    seq, current = send("enter", steps)
    report(seq, "enter", current, " engage=select")

    moved_to = (ux, uy)
    for cell in engage_targets(current, (ux, uy),
                               int(command.get("typical_move", 6)))[:tries]:
        if not do_goto({"goto": list(cell), "steps": steps, "max": 80}):
            continue
        seq, current = send("enter", max(steps, 5_000_000))
        report(seq, "enter", current, f" engage=move->{cell}")
        current = settle(steps, int(command.get("move_settle", 8)))
        if unit_at(current, cell[0], cell[1]) is not None and unit_at(current, ux, uy) is None:
            moved_to = cell
            break
        print(f"  移動到 {cell} 沒有生效，換下一個候選格", flush=True)
    if moved_to == (ux, uy):
        # 走不動也要把這個單位的行動結束掉，否則 record `+5` bit7 不設，
        # sweep 下一輪又選到同一個單位，回合永遠推不掉。
        print(f"engage ({ux},{uy}) 所有候選格都到不了，改原地待機", flush=True)
        if not do_goto({"goto": [ux, uy], "steps": steps, "max": 80}):
            return False
        seq, current = send("enter", max(steps, 5_000_000))
        report(seq, "enter", current, " engage=stay")
        current = settle(steps, int(command.get("move_settle", 8)))
        current = stand_by(steps, "=stay")
        unit = unit_at(current, ux, uy)
        if unit is None or not acted(unit):
            print(f"engage ({ux},{uy}) 原地待機之後仍未標記已行動", file=sys.stderr)
            return False
        return True

    # 射程不是恆等於 1：亞雷斯（騎士）的 atk_max 是 2，實測指令環會把游標自動
    # 跳到距離 2 的敵人並命中。所以這裡用 reach（預設 2）判斷值不值得開指令環，
    # 真正能打到誰由原版自己選。
    reach = int(command.get("reach", 2))
    in_reach = [e for e in side(current, ENEMY_CAMP)
                if distance((e["x"], e["y"]), moved_to) <= reach]
    if not in_reach:
        print(f"engage 走到 {moved_to} 但射程 {reach} 內沒有敵人，待機結束行動", flush=True)
        current = stand_by(steps, "=advance")
        unit = unit_at(current, moved_to[0], moved_to[1])
        if unit is None or not acted(unit):
            print(f"engage 推進到 {moved_to} 之後仍未標記已行動", file=sys.stderr)
            return False
        return True
    # 指令環開啟時預設就是攻擊，確認後游標自動跳到可攻擊的敵人，所以是 enter、enter。
    seq, current = send("enter", max(steps, 5_000_000))
    report(seq, "enter", current, " engage=ring")
    current = settle(steps, int(command.get("ring_settle", 4)))
    before = {(e["x"], e["y"]): e.get("hp") for e in side(current, ENEMY_CAMP)}
    seq, current = send("enter", max(steps, 10_000_000))
    report(seq, "enter", current, " engage=strike")
    current = settle(int(command.get("strike_steps", 10_000_000)),
                     int(command.get("strike_settle", 14)))
    after = {(e["x"], e["y"]): e.get("hp") for e in side(current, ENEMY_CAMP)}
    hurt = [c for c, hp in before.items() if after.get(c, 0) != hp]
    print(f"engage 完成：{moved_to} 攻擊後改變的敵人格 {hurt}"
          f"（敵方存活 {len(after)}）", flush=True)
    return True

def do_sweep_round(command):
    """把這一回合所有還沒行動的我方單位依序接戰。

    座標不能寫死：第一回合以後每個單位都在上一回合走到的位置。每一輪都重讀
    `units` 找「camp 是我方、record `+5` bit7 還沒設」的單位。
    """
    steps = int(command.get("steps", 2_000_000))
    for _ in range(int(command.get("max_units", 12))):
        current = state()
        pending = [u for u in side(current, ALLY_CAMP) if not acted(u)]
        if not pending:
            print("sweep_round：本回合我方單位都已行動", flush=True)
            return True
        if not side(current, ENEMY_CAMP):
            print("sweep_round：敵方已全滅", flush=True)
            return True
        # 固定挑「離最近敵人最近」的那個單位，順序才可重現。
        enemies = [(e["x"], e["y"]) for e in side(current, ENEMY_CAMP)]
        pending.sort(key=lambda u: (min(distance((u["x"], u["y"]), e) for e in enemies),
                                    u["x"], u["y"]))
        pick = pending[0]
        inner = dict(command)
        inner.pop("sweep_round", None)
        inner["engage"] = [pick["x"], pick["y"]]
        if not do_engage(inner):
            return False
    print("sweep_round：用完單位上限仍有未行動單位", file=sys.stderr)
    return False


def do_sweep_battle(command):
    """一路打到敵方全滅：每回合掃完我方單位，再等回合數推進。

    回合推進不主動按 END——先確認全部行動完之後原版會不會自己換手；`await`
    逾時就代表要另外送結束回合，那時再處理，不要先假設。
    """
    steps = int(command.get("steps", 2_000_000))
    rounds = int(command.get("rounds", 30))
    for index in range(rounds):
        current = state()
        if not side(current, ENEMY_CAMP):
            print(f"sweep_battle：敵方全滅（第 {index} 輪之前）", flush=True)
            return True
        before = measure(current, "round")
        inner = dict(command)
        inner.pop("sweep_battle", None)
        if not do_sweep_round(inner):
            return False
        print(f"sweep_battle：第 {before} 回合我方行動完畢，等換手", flush=True)
        if not do_await({"await": f"round>={before + 1}",
                         "steps": int(command.get("turn_steps", 10_000_000)),
                         "max": int(command.get("turn_max", 200))}):
            return False
    print(f"sweep_battle：{rounds} 回合內未打完", file=sys.stderr)
    return False

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
        if "goto" in command:
            if not do_goto(command):
                return 6
            continue
        if "await" in command:
            if not do_await(command):
                return 7
            continue
        if "engage" in command:
            if not do_engage(command):
                return 8
            continue
        if "sweep_round" in command:
            if not do_sweep_round(command):
                return 9
            continue
        if "sweep_battle" in command:
            if not do_sweep_battle(command):
                return 10
            continue
        key = command.get("key", "")
        if command.get("gate") == "kbd_empty" and current.get("kbd_pending", 0) > 0:
            key = ""
        seq, current = send(key, int(command.get("steps", 3_000_000)))
        report(seq, key, current)
    return 0


if __name__ == "__main__":
    sys.exit(main())
