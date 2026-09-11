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
* ``{"town_probe": {"moves": [...]}}``：在戰間城鎮沿方向序列切建築，每步 enter 看
  進到哪裡，進了選單就 esc 退回。城鎮是五格循環而不是走動畫面：left 遞增、
  right 遞減，up 與 down 不動。
* ``{"town_save": true}``：在 0 號酒店存檔，建立下一關的續跑點。判準是覆蓋層裡
  ``FD2.SAV`` 的內容雜湊變了沒，不是畫面。
* ``{"sweep_round": true}``：把這一回合所有未行動的我方單位依序接戰。
* ``{"sweep_battle": true, "rounds": 30}``：一路打到敵方全滅。
* ``{"await": "round>=2"}``：反覆只前進不送鍵，直到條件成立。可用變數：
  ``round``（`view.round`）、``enemy_alive``／``ally_alive``（依 camp 數存活
  單位）、``cursor_x``／``cursor_y``、``steps``。比較運算子 ``>= <= == != > <``。

兩者都吃 ``steps``（每一格的指令數）與 ``max``（最多幾格，逾時即失敗）。
失敗一律非零離開；盲目繼續會產生「看起來跑完了但走錯路」的收據。
"""

import hashlib
import json
import os
import re
import sys
import time

RUN = "/out"
PLAN = "/plan.jsonl"
STEP_TIMEOUT = float(os.environ.get("FD2_ORACLE_STEP_TIMEOUT", "180"))

# 陣營編碼共三種，取自重製端 `native_continue_runtime_units.go` 對 raw `+6` 的
# 分派，與實際收據一致：0 敵方、1 友軍、2 我方。
#
# **camp 1 是友軍不是敵人**：AI 控制的盟友，第二關有六個。它們會自己打，敵方 HP
# 因此有時在我方沒出手的回合也會掉。「敵方全滅」只算 camp 0 是對的——把 camp 1
# 算進去會永遠打不完，漏掉 camp 0 的某些單位則會提早收工。
#
# 下面兩個常數的名字沿用既有序列檔裡的 `ally_alive`，指的是**我方**（camp 2），
# 不是 camp 1 的友軍。
ALLY_CAMP = 2
ENEMY_CAMP = 0
FRIENDLY_CAMP = 1

# 戰場的單位陣列基底。戰鬥中途的過場（第一關第 3 回合哈諾與哈瓦特加入）會把
# 指標換掉，那時 `units` 讀出來是垃圾——camp 會是 63／34／54 這種值，x／y 也
# 超出地圖。基底一變就不能再信任 units，否則會挑到不存在的單位並把按鍵送去
# 錯的地方。第一次進入戰場時記下來當閘門。
BATTLE_UNIT_BASE = None
# 戰場的回合數只會遞增。過場期間 `view` 也讀的是別的記憶體，`round` 會掉回 1；
# 這比看 units 內容可靠得多——垃圾資料的 camp 與座標可以剛好落在合法範圍，
# 回合倒退不會。
MAX_ROUND_SEEN = 0

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
        f"steps={current['steps']} ui={ui_mode(current)} "
        f"cursor=({view.get('cursor_x')},{view.get('cursor_y')}) "
        f"round={view.get('round')} ally={alive(current, ALLY_CAMP)} "
        f"enemy={alive(current, ENEMY_CAMP)}{note}",
        flush=True,
    )


def settle(steps, count):
    """只前進不送鍵，讓動畫或演出跑完。"""
    for _ in range(count):
        seq, current = send("", steps)
    return current


def do_goto(command):
    """把地圖游標移到指定格。

    先看 `input_chain` 判斷現在是哪一個介面在收鍵：只有地圖游標與目標選擇這兩種
    模式下，方向鍵才會移動游標。在指令環或系統選單裡送方向鍵是在選選項，游標當然
    不會動——那不是「卡住」，是模式不對。系統選單用 esc 退得掉；指令環要選一項才
    離得開，這裡不替它決定選哪一項，直接回報讓呼叫者處理。
    """
    target_x, target_y = command["goto"]
    steps = int(command.get("steps", 2_000_000))
    budget = int(command.get("max", 80))
    escapes_left = int(command.get("escape_retries", 2))
    for _ in range(budget):
        current = state()
        mode = ui_mode(current)
        if mode not in CURSOR_MODES:
            if mode == "ring":
                # 這裡不能 esc：指令環上的 esc 會取消行動。停在指令環代表上一個
                # 單位沒收乾淨，回報比默默把它的移動抹掉好。
                print(f"goto ({target_x},{target_y})：介面停在指令環，中止",
                      file=sys.stderr)
                return False
            if mode in ESCAPABLE - CURSOR_MODES and escapes_left > 0:
                escapes_left -= 1
                seq, current = send("esc", max(steps, 3_000_000))
                report(seq, "esc", current, f" goto=leave-{mode}")
                continue
            if mode == "unknown":
                # 演出或過場還在跑，等它收完再看一次。
                seq, current = send("", steps)
                report(seq, "", current, " goto=wait")
                continue
            print(f"goto ({target_x},{target_y})：目前介面是 {mode}，方向鍵不會移動"
                  f"游標", file=sys.stderr)
            return False
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
        report(seq, key, current, f" goto=({target_x},{target_y})")
    print(f"goto ({target_x},{target_y}) 用完 {budget} 格仍未到位", file=sys.stderr)
    return False


def do_await(command):
    """只前進不送鍵直到條件成立；遇到對白會送 enter 推它，否則永遠等不到。"""
    expression = command["await"]
    steps = int(command.get("steps", 10_000_000))
    budget = int(command.get("max", 120))
    key = command.get("key", "")
    guard = command.get("abort_if")
    for _ in range(budget):
        current = state()
        # 守衛只在戰場上才算數。離開戰場之後 `units` 是垃圾，`ally_alive` 會讀成 0
        # ——第一關實際打完進了戰後城鎮，卻被守衛判成「我方全滅」。
        if guard and in_battle(current) and holds(current, guard)[0]:
            print(f"await {expression} 中止：{guard} 成立", file=sys.stderr)
            return False
        if guard and not in_battle(current):
            print(f"await {expression}：已離開戰場，視為這一場結束", flush=True)
            return True
        ok, got = holds(current, expression)
        if ok:
            print(f"await {expression} 成立（實測 {got}）", flush=True)
            return True
        send_key = "" if (key and current.get("kbd_pending", 0) > 0) else key
        # 對白不會自己走完。等回合推進的時候中間常常插一段升級訊息或事件台詞，
        # 只送空鍵會等到預算用完——實測第一關第 3 回合就卡在這裡 300 格。
        if ui_mode(current) == "dialogue" and current.get("kbd_pending", 0) == 0:
            send_key = "enter"
        seq, current = send(send_key, steps)
        report(seq, send_key, current, f" await={expression}")
    print(f"await {expression} 在 {budget} 格內未成立", file=sys.stderr)
    return False


# 已知的陣營編碼。過場期間 `units` 是垃圾，camp 會出現 63／34／54 這種值。
KNOWN_CAMPS = {0, 1, 2, 3}
# 地圖上限（第一關 map0 之外的圖也在這個量級內）。垃圾資料的 x／y 會遠超過。
MAP_LIMIT_X, MAP_LIMIT_Y = 31, 63


def plausible_battle(current):
    """`units` 看起來像不像一組真的戰場單位。

    過場會把單位陣列指標換掉，而**狀態欄位本身不會告訴你它失效了**：它照樣是
    合法 JSON、照樣有 12 筆單位。只有內容看得出來——camp 跑出 63／34／54，
    x／y 超出地圖，hp 是 16191 這種數量級。
    """
    units = [u for u in current.get("units", []) if u.get("hp", 0) > 0]
    if not units:
        return False
    if any(u.get("camp") not in KNOWN_CAMPS for u in units):
        return False
    if any(u.get("x", 999) > MAP_LIMIT_X or u.get("y", 999) > MAP_LIMIT_Y for u in units):
        return False
    return any(u.get("camp") == ALLY_CAMP for u in units)


def in_battle(current):
    """在不在戰場。三個判準疊起來，任一不過就當作不在。

    1. 回合數不得倒退。過場期間 `view` 讀的是別的記憶體，`round` 會掉回 1——
       實測第 3 回合的過場就是這樣，而 units 的 camp 與座標剛好都落在合法範圍，
       只看內容會誤判成「還在戰場」，然後把游標往 (8,42) 這種地方送。
    2. 單位陣列基底與已知的相同就是同一場。
    3. 基底變了要看內容：過場後 spawn 新單位會重配置陣列，基底本來就會變。
    """
    global BATTLE_UNIT_BASE, MAX_ROUND_SEEN
    round_now = int((current.get("view") or {}).get("round", 0))
    if round_now < MAX_ROUND_SEEN:
        return False
    here = (BATTLE_UNIT_BASE is not None
            and current.get("unit_base") == BATTLE_UNIT_BASE)
    if not here and plausible_battle(current):
        BATTLE_UNIT_BASE = current.get("unit_base")
        here = True
    if here:
        MAX_ROUND_SEEN = max(MAX_ROUND_SEEN, round_now)
    return here


def resume_battle(steps, budget, confirm=3):
    """把戰鬥中途的過場推完，回到戰場；推不回來就回報 False。

    兩件事都是實測換來的：

    **狀態在過場期間會抖動。** 同一段過場裡連續取樣，`round` 會在 1 和 3 之間跳、
    `units` 會在合理與垃圾之間跳——oracle 讀的那幾個位址在過場期間被複用。單次
    取樣判不準，所以要連續 `confirm` 次都在戰場才採信，中間送空鍵讓遊戲往前走。

    **budget 要小，而且不要一直送 enter。** 過場播動畫時 `kbd_reads` 根本不動，
    送進去的鍵停在緩衝區沒人取；盲送 enter 只會在動畫結束的瞬間一次全部灌進去，
    穿過任何選單。2026-09-10 第一次實作給了 200 格，我方全滅之後那些 enter 一路
    推過戰敗畫面、標題選單、開場動畫，重新開了一局，log 最後看起來像「回到第一關
    第 1 回合、敵方 0 隻」——收據還在，人卻要盯著 checkpoint 才看得出來跑錯了。
    所以這裡以等待為主：每三格才送一次 enter，而且緩衝區還有鍵就不送。
    """
    stable = 0
    for index in range(budget):
        current = state()
        if in_battle(current):
            stable += 1
            if stable >= confirm:
                return True
            seq, current = send("", steps)
            report(seq, "", current, f" cutscene-confirm{stable}")
            continue
        stable = 0
        # 認得出是對白就直接推，不必靠「每三格送一次」的節流猜。
        if ui_mode(current) == "dialogue" and current.get("kbd_pending", 0) == 0:
            seq, current = send("enter", steps)
            report(seq, "enter", current, " cutscene-dialogue")
            continue
        key = "enter" if (index % 3 == 2 and current.get("kbd_pending", 0) == 0) else ""
        seq, current = send(key, steps)
        report(seq, key, current, " cutscene")
    return False


# 介面模式的特徵位址，取自 oracle 快照的 `input_chain`（等鍵盤時堆疊上的返回
# 位址）。這四種介面在 eip、view 與 units 上完全一樣——overlay selector 恆為 1
# ——只有這條鏈分得出來。方向鍵在四種狀態下的意義不同，分不出來就只能猜，猜錯
# 會把方向鍵送進選單，然後游標「莫名其妙不動」。
#
# `0x16FAE` 落在 FD2 已知的系統選單 handler `0x16F55` 那支函式裡，`0x18EEF` 落在
# action chooser `0x18D8C` 同一段；與專案既有的反組譯結論一致。
UI_MODES = (
    ("shop", "0x2D7D1"),      # 商店（店員對話與買賣圖示）：esc 退得掉
    ("town", "0x2CE08"),      # 戰間城鎮：五個建築排成一圈，left 遞增／right 遞減，
                              # up 與 down 無效，enter 進入目前那一棟
    ("grid", "0x1BC8E"),      # 指令 grid（六格圖示，`0x1BBDC` 那組 chooser）
    ("status", "0x1BA37"),    # 單位狀態面板（能力值與裝備）：esc 退得掉
    ("ring", "0x18EEF"),      # 指令環：↑攻擊／←法術／→物品／↓待機
    ("system", "0x16FAE"),    # 空地上按 enter 開的系統選單（含 END）
    ("target", "0x117AE"),    # 選取之後的移動格／攻擊目標選擇
    ("cursor", "0x117F8"),    # 地圖游標自由移動
    # 選取的那一格是過渡狀態：範圍已經畫好、但還沒收到第一個鍵，鏈長得不一樣。
    # 少了這一條，「等它進 target」會永遠等不到——那個迴圈要收到一個鍵才會轉成
    # 0x117AE，而等待送的是空鍵。順序放最後，才不會蓋掉 ring 的 0x18EEF。
    ("target", "0x18978"),
)
# 對白等待（升級訊息、事件台詞）不只一支 handler：實測看到 `0x1E44E`、`0x1E5A8`
# 與 `0x1E464`，共同前綴是 `0x16039`。用範圍比對而不是單一位址，否則同一種畫面
# 會有一部分掉進 unknown，而 unknown 的處置是「等」——對白等不出結果，要 enter。
# 下框對白（一般台詞、升級訊息）走 `0x16039`＋`0x1E4xx`；上框對白（說話者頭像在
# 右，例如第一關第 3 回合哈諾加入的「老爸！老爸！」）走 `0x164C4`＋事件自己的
# native_source（`0x3424D` 落在 join_party 的 `0x341E8` 附近）。兩者共用 `0x16CF8`
# 與 `0x16D05` 的框繪製。四個標記都實測與其他介面零衝突。
DIALOGUE_MARKERS = ("0x16039", "0x164C4", "0x16CF8", "0x16D05")
DIALOGUE_RANGE = (0x1E400, 0x1E5FF)

# 方向鍵會移動地圖游標的模式。其餘模式送方向鍵是在選選項，不會動游標。
CURSOR_MODES = {"cursor", "target"}
# esc 退得掉的介面。指令環也在裡面：回合改用系統選單的 END 結束之後，指令環不再
# 需要「選一項」——打不到人的單位就退出來，讓它這一回合停在原地。退出會取消這次
# 移動，單位回到原位；這一回合本來就不動它，沒有損失。
ESCAPABLE = {"system", "status", "grid", "target", "ring", "shop"}


def ui_mode(current):
    chain = current.get("input_chain") or []
    for name, marker in UI_MODES:
        if marker in chain:
            return name
    if any(marker in chain for marker in DIALOGUE_MARKERS):
        return "dialogue"
    for address in chain:
        try:
            value = int(address, 16)
        except (TypeError, ValueError):
            continue
        if DIALOGUE_RANGE[0] <= value <= DIALOGUE_RANGE[1]:
            return "dialogue"
    return "unknown"


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


# 每個單位實際走得成功的距離。移動力連同地形成本在狀態層看不到，只能從「哪一格
# 走成了」學回來。學到之後下一次先試那個距離，省掉從移動力上限一路往下試的四五次
# 試探——每次試探要送十幾個鍵、跑三千萬指令，第二關第 1 回合光在這上面就花了兩億。
MOVE_SPAN = {}


def unit_key(unit):
    """一輪之內辨識同一個我方單位用的鍵。

    用 identity（record `+8`）不用座標：推進過的單位座標就變了，拿行動前的座標
    比對等於什麼都沒記，同一個單位會被一直重選到預算用完。identity 缺席時才退回
    座標——那時至少擋得住「原地沒動又被選中」。
    """
    identity = unit.get("identity")
    if identity is None:
        return ("cell", unit.get("x"), unit.get("y"))
    return ("identity", identity)


def engage_targets(current, origin, typical_move=6, hint=None):
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
    goal = (nearest["x"], nearest["y"])
    here = distance(origin, goal)

    # 逼近格不沿單一直線。直線被地形擋住時那幾格會十試十敗——第二關 (23,16) 的
    # 單位往西整列都走不了，一路試到 (13,16) 全被拒絕，整個回合白花三億指令。
    # 改成掃 origin 周圍移動力內的格，留下比原地更靠近敵人的。
    reachable = []
    for dx in range(-typical_move, typical_move + 1):
        for dy in range(-typical_move, typical_move + 1):
            cell = (origin[0] + dx, origin[1] + dy)
            if cell[0] < 0 or cell[1] < 0 or cell in occupied or cell in seen:
                continue
            span = distance(cell, origin)
            if span == 0 or span > typical_move:
                continue
            gain = distance(cell, goal)
            if gain >= here:
                continue
            reachable.append((span, gain, cell))

    # 排序要同時涵蓋「走多遠」與「走得到沒」。實際移動力連同地形成本從狀態層看
    # 不出來（第二關實測只走得了三格，而 typical_move 是 6），所以按距離分桶、
    # 每桶只取最靠近敵人的幾格：先試最遠的一桶，走不到就退一格再試。全押在同一
    # 個距離上，猜錯就整輪報銷。
    buckets = {}
    for span, gain, cell in reachable:
        buckets.setdefault(span, []).append((gain, cell))
    if hint:
        # 這個單位上次走成了幾格就從幾格開始試，再往兩邊擴。
        order = sorted(buckets, key=lambda span: (abs(span - hint), -span))
    else:
        order = sorted(buckets, reverse=True)
    approach = []
    for span in order:
        for _, cell in sorted(buckets[span])[:2]:
            approach.append(cell)
    return cells + approach


def wait_mode(wanted, steps, budget=12):
    """只前進不送鍵，等介面變成 wanted 之一。回傳實際模式。"""
    mode = ui_mode(state())
    for _ in range(budget):
        if mode in wanted:
            return mode
        seq, current = send("", steps)
        mode = ui_mode(current)
        report(seq, "", current, f" wait-mode{sorted(wanted)}")
    return mode


def stand_by(steps, note=""):
    """在指令環上結束這個單位的行動。

    四向的預設佈局是 ↑0 攻擊／←1 法術／→2 物品／↓3 待機（`0x18D8C` 的 switch）。
    但可選項會隨單位而變——實測第一關第 3 回合加入的哈瓦特按 ↓ 開的是指令 grid，
    不是待機。所以每個方向送完都檢查介面：開了 grid 或狀態面板就 esc 退回來換
    下一個方向，回到地圖游標才算行動真的結束。

    只移動不待機的單位 record `+5` bit7 不會設，掃描下一輪又會選到它，回合永遠
    推不掉——所以這一步不能省。
    """
    for key in ("down", "right", "left"):
        mode = wait_mode({"ring"}, steps)
        if mode != "ring":
            if mode in {"cursor", "dialogue"}:
                return state()      # 行動已經結束了
            print(f"stand_by：介面是 {mode} 不是指令環，不送待機鍵", file=sys.stderr)
            return state()
        seq, current = send(key, steps)
        report(seq, key, current, f" standby{note}")
        seq, current = send("enter", max(steps, 5_000_000))
        report(seq, "enter", current, f" standby{note}")
        current = settle(steps, 4)
        mode = ui_mode(current)
        if mode == "ring":
            # 這一項不是待機（或按了沒生效），換下一個方向。**不要 esc**——
            # 指令環上的 esc 是取消行動，會把單位送回移動前那一格。
            continue
        if mode in ESCAPABLE - CURSOR_MODES:
            # 開的是 grid 或狀態面板這類子面板：esc 只是退回指令環，行動還在。
            seq, current = send("esc", max(steps, 3_000_000))
            report(seq, "esc", current, f" standby{note}=wrong-option({mode})")
            continue
        return current
    return state()

def ensure_cursor_mode(steps, budget=60):
    """把介面退回「地圖游標自由移動」，或等到它回來。

    sweep 每挑一個單位就要從自由移動狀態開始。上一個單位收尾之後介面可能停在
    目標選擇或系統選單——那時方向鍵移不動地圖游標，goto 會一路送到用完預算。
    esc 退得掉這兩種；指令環要選一項才離得開，不在這裡替它決定。

    `unknown` 多半不是壞掉，是**敵方回合**：AI 在走位與演出，遊戲根本沒在等鍵盤，
    堆疊上自然沒有輸入路徑。那只能等，而且要等得夠久——預算給小了會在換手的
    當下判成「退不回地圖游標」。
    """
    for _ in range(budget):
        current = state()
        mode = ui_mode(current)
        if mode == "cursor":
            return True
        if mode == "ring":
            # 指令環上的 esc 是取消行動——單位會退回移動前那一格。要回到地圖游標
            # 就得把這個單位的行動好好結束掉，不能一路 esc 退回去。
            current = stand_by(steps, "=back-to-cursor")
            if ui_mode(current) == "ring":
                print("ensure_cursor_mode：指令環上找不到待機那一項",
                      file=sys.stderr)
                return False
            continue
        if mode in ESCAPABLE:
            seq, current = send("esc", max(steps, 3_000_000))
            report(seq, "esc", current, f" back-to-cursor(from {mode})")
            continue
        if mode == "dialogue":
            # 升級訊息與事件台詞。等它不會自己走，要送 enter；但緩衝區還有鍵就
            # 別再送，否則多的鍵會被下一個介面吃掉。
            key = "" if current.get("kbd_pending", 0) > 0 else "enter"
            seq, current = send(key, max(steps, 3_000_000))
            report(seq, key, current, " back-to-cursor(dialogue)")
            continue
        seq, current = send("", steps)
        report(seq, "", current, f" back-to-cursor(from {mode})")
    return ui_mode(state()) == "cursor"


def do_engage(command):
    """選取一個我方單位，推進到能打的位置，打得到就打，打不到就待機。

    每一步都用 `input_chain` 的介面模式驗證，不用「座標變了沒」推測：

    - 選取成功 → 介面從 cursor 變成 target（選移動格）。
    - 移動成功 → 介面變成 ring（原版移動完就開指令環）。移動被拒絕時介面會留在
      target，這比比對座標可靠——座標比對分不出「沒走成」與「走到別處」。
    - ring 上：攻擊是直接 enter（預設就是攻擊，確認後游標自動跳到可打的敵人），
      待機是 down 再 enter（↑0 攻擊／←1 法術／→2 物品／↓3 待機）。
    """
    ux, uy = command["engage"]
    steps = int(command.get("steps", 2_000_000))
    # 分桶之後每個距離兩格，移動力 6 就是 12 格；預算少於這個數就試不到近的那幾桶，
    # 而走不動的單位偏偏只有近格走得到。
    tries = int(command.get("tries", 14))
    reach = int(command.get("reach", 2))
    if not resume_battle(int(command.get("cutscene_steps", 5_000_000)),
                         int(command.get("cutscene_max", 30))):
        print("engage：目前不在戰場（過場推不回來）", file=sys.stderr)
        return False
    if not ensure_cursor_mode(max(steps, int(command.get("cursor_steps", 5_000_000))),
                              int(command.get("cursor_wait", 80))):
        print(f"engage ({ux},{uy})：介面退不回地圖游標（目前 {ui_mode(state())}）",
              file=sys.stderr)
        return False
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
    started_round = measure(current, "round")
    if not do_goto({"goto": [ux, uy], "steps": steps, "max": command.get("max", 80)}):
        return False

    seq, current = send("enter", steps)
    report(seq, "enter", current, " engage=select")
    mode = wait_mode({"target", "ring", "system"}, steps)
    if mode == "system":
        # 這一格沒有可選單位（走到之後被打死、或本來就選不了），enter 開了系統
        # 選單。退出去，把這個單位交給下一輪。
        seq, current = send("esc", max(steps, 3_000_000))
        report(seq, "esc", current, " engage=select-missed")
        print(f"engage ({ux},{uy}) 選取時開的是系統選單，這一格不是可選單位", flush=True)
        return False
    if mode != "target":
        print(f"engage ({ux},{uy}) 選取之後介面是 {mode}，不是移動格選擇",
              file=sys.stderr)
        return False

    moved_to = (ux, uy)
    standing = [e for e in side(current, ENEMY_CAMP)
                if distance((e["x"], e["y"]), (ux, uy)) <= reach]
    candidates = [] if standing else engage_targets(
        current, (ux, uy), int(command.get("typical_move", 6)),
        MOVE_SPAN.get(unit_key(unit)))[:tries]
    if standing:
        # 已經站在射程內就原地確認，不必再走。
        seq, current = send("enter", max(steps, 5_000_000))
        report(seq, "enter", current, " engage=stay-in-reach")
    for cell in candidates:
        if not do_goto({"goto": list(cell), "steps": steps, "max": 80}):
            continue
        seq, current = send("enter", max(steps, 5_000_000))
        report(seq, "enter", current, f" engage=move->{cell}")
        if wait_mode({"ring"}, steps) == "ring":
            moved_to = cell
            MOVE_SPAN[unit_key(unit)] = distance(cell, (ux, uy))
            break
        print(f"  移動到 {cell} 被拒絕（介面沒進指令環），換下一個候選格", flush=True)
    mode = wait_mode({"ring"}, steps)
    if mode != "ring":
        print(f"engage ({ux},{uy}) 走完之後介面是 {mode}，指令環沒開", file=sys.stderr)
        return False

    in_reach = [e for e in side(current, ENEMY_CAMP)
                if distance((e["x"], e["y"]), moved_to) <= reach]
    if in_reach:
        before_hp = sum(e.get("hp", 0) for e in side(state(), ENEMY_CAMP))
        before_count = len(side(state(), ENEMY_CAMP))
        seq, current = send("enter", max(steps, 5_000_000))
        report(seq, "enter", current, " engage=ring-attack")
        wait_mode({"target"}, steps)
        seq, current = send("enter", max(steps, 10_000_000))
        report(seq, "enter", current, " engage=strike")
        current = settle(int(command.get("strike_steps", 10_000_000)),
                         int(command.get("strike_settle", 14)))
        if not in_battle(current):
            print(f"engage {moved_to} 攻擊後離開戰場（可能敵方全滅或觸發過場）",
                  flush=True)
            return True
        after = side(current, ENEMY_CAMP)
        print(f"engage：{moved_to} 敵方總 HP {before_hp}→"
              f"{sum(e.get('hp', 0) for e in after)}、存活 {before_count}→{len(after)}",
              flush=True)
    else:
        # 打不到人就待機。**不能用 esc**：esc 在指令環上是「取消這次行動」，原版
        # 會把單位送回移動前那一格——實測走到 (17,15) 開了指令環，送 esc 之後單位
        # 回到 (20,14)，推進整個作廢。第一關看不出來，因為那裡每個單位走一步就
        # 接敵，幾乎不會走到這個分支。
        print(f"engage 走到 {moved_to} 但射程 {reach} 內沒有敵人，改待機",
              flush=True)
        current = stand_by(steps, f"={moved_to}")
        if ui_mode(current) == "ring":
            print(f"engage {moved_to} 待機失敗，指令環上找不到待機那一項",
                  file=sys.stderr)
            return False

    # 收尾只做清理：把介面退回地圖游標，讓下一個單位從已知狀態開始。
    #
    # 不再檢查 record `+5` bit7。它在好幾種情況下都不成立：本回合最後一個單位行動完
    # 會立刻換手、新回合把整批清零；攻擊接的升級對白結束前還沒寫上去；打不到人而
    # 退出指令環的單位根本沒行動過。改由 sweep 自己記「這一輪處理過誰」。
    # 收尾預算要夠一段演出加一輪敵方回合。第二關第 3 回合一次攻擊觸發了增援
    # （敵方 9→15），演出期間介面一直是 unknown；預算 30 格等不完就判成「收尾
    # 之後介面停在 unknown」而中止，其實只是還沒播完。
    for _ in range(int(command.get("finish_wait", 90))):
        current = state()
        if measure(current, "round") > started_round:
            print(f"engage {moved_to} 完成，回合已由 {started_round} 推進到 "
                  f"{measure(current, 'round')}", flush=True)
            return True
        mode = ui_mode(current)
        if mode == "cursor":
            return True
        if mode == "dialogue":
            key = "" if current.get("kbd_pending", 0) > 0 else "enter"
            seq, current = send(key, max(steps, 3_000_000))
            report(seq, key, current, " engage=finish-dialogue")
            continue
        if mode == "ring":
            # 收尾階段不該還停在指令環——這裡的 esc 會取消整次行動，把單位送回
            # 移動前那一格。停在這代表前面沒收乾淨，回報比清掉好。
            print(f"engage {moved_to} 收尾時仍停在指令環，行動沒有結束",
                  file=sys.stderr)
            return False
        if mode in ESCAPABLE:
            seq, current = send("esc", max(steps, 3_000_000))
            report(seq, "esc", current, f" engage=finish-{mode}")
            continue
        seq, current = send("", max(steps, 15_000_000))
        report(seq, "", current, " engage=finish-wait")
    print(f"engage {moved_to} 收尾之後介面停在 {ui_mode(state())}", file=sys.stderr)
    return False


def empty_cell(current, near):
    """找一個沒有單位、離 near 最近的格，用來開系統選單。"""
    taken = occupied_cells(current)
    best, best_span = None, 1 << 30
    for dx in range(-6, 7):
        for dy in range(-6, 7):
            cell = (near[0] + dx, near[1] + dy)
            if cell[0] < 0 or cell[1] < 0 or cell in taken:
                continue
            span = abs(dx) + abs(dy)
            if span and span < best_span:
                best, best_span = cell, span
    return best


def end_turn(command):
    """開系統選單選 END 結束我方回合。

    單位各自待機之後原版有時會自己換手，但「有時」不夠：打不到人而待機的單位、
    推進到一半的單位，換手條件不一定成立，等回合推進會空等到預算用完。END 是既有
    收據走過的路徑（`ch01-phase-banner.jsonl`：在空地開面板、下三次、確認、再確認
    YES），用它收尾不必去猜換手條件。
    """
    steps = int(command.get("steps", 2_000_000))
    if not ensure_cursor_mode(max(steps, 5_000_000), int(command.get("cursor_wait", 80))):
        print("end_turn：介面退不回地圖游標", file=sys.stderr)
        return False
    current = state()
    view = current.get("view", {}) or {}
    cell = empty_cell(current, (int(view.get("cursor_x", 0)), int(view.get("cursor_y", 0))))
    if cell is None:
        print("end_turn：附近找不到空地開系統選單", file=sys.stderr)
        return False
    if not do_goto({"goto": list(cell), "steps": steps, "max": 40}):
        return False
    seq, current = send("enter", max(steps, 5_000_000))
    report(seq, "enter", current, " end-turn=open")
    if wait_mode({"system"}, steps) != "system":
        print(f"end_turn：空地上 enter 開的不是系統選單（{ui_mode(state())}）",
              file=sys.stderr)
        return False
    for _ in range(3):
        seq, current = send("down", steps)
        report(seq, "down", current, " end-turn=pick-END")
    seq, current = send("enter", max(steps, 5_000_000))
    report(seq, "enter", current, " end-turn=confirm")
    settle(steps, 4)
    seq, current = send("enter", max(steps, 5_000_000))
    report(seq, "enter", current, " end-turn=yes")
    return True




def do_town_probe(command):
    """在戰間城鎮沿著給定的方向序列切建築，每一步按 enter 看進到哪裡。

    城鎮不是走動畫面。五棟建築排成一圈，`left` 把編號加一、`right` 減一、超出
    0..4 就繞回去，`up` 與 `down` 完全無效——先前「四個方向各走十幾步都留在城鎮」
    正是因為半數按鍵根本沒有意義，另外半數在原地繞圈。編號順序由原版決定：
    0 酒店、1 武器店、2 出口（出戰整備，通往下一關）、3 道具店、4 教會（存檔）。

    城鎮沒有可讀座標——`view` 的 cursor 是戰場用的，在城鎮不動——所以位置只能靠
    畫面右下角那塊標籤判讀。每一步都回報當時的介面，進了商店那類選單就 esc 退回
    城鎮，不會把後面的按鍵送進選單裡。

    這是探索用的：一次執行問出周圍有什麼，免得為了找一個入口重跑七十幾億指令。
    """
    moves = command.get("moves") or ["down", "right", "up", "left"]
    steps = int(command.get("steps", 8_000_000))
    found = []
    for index, move in enumerate(moves):
        seq, current = send(move, steps)
        report(seq, move, current, f" town-probe[{index}]")
        seq, current = send("enter", max(steps, 10_000_000))
        report(seq, "enter", current, f" town-probe[{index}]")
        current = settle(steps, int(command.get("probe_settle", 5)))
        mode = ui_mode(current)
        if mode != "town":
            found.append((index, move, mode))
            print(f"town_probe[{index}] 往 {move} 之後 enter → {mode}", flush=True)
            for _ in range(int(command.get("escape_max", 6))):
                if ui_mode(state()) == "town":
                    break
                seq, current = send("esc", max(steps, 5_000_000))
                report(seq, "esc", current, f" town-probe[{index}]=leave")
                settle(steps, 3)
    print(f"town_probe 完成：{found or '每一步都還在城鎮的建築選擇上'}", flush=True)
    return True






STATE_DIR = os.environ.get("FD2_ORACLE_STATE", "")


def saved_files():
    """可寫覆蓋層目前有哪些檔。存檔成不成功看這裡，不用猜畫面。"""
    if not STATE_DIR or not os.path.isdir(STATE_DIR):
        return set()
    return {name.upper() for name in os.listdir(STATE_DIR)}


def save_fingerprint():
    """覆蓋層裡每個檔的大小與內容雜湊。

    第二關以後覆蓋層本來就有一份 FD2.SAV（是上一關的續跑點載進來的），所以
    「多出一個檔」這個判準只在第一次成立。之後要看的是**內容變了沒**。
    """
    if not STATE_DIR or not os.path.isdir(STATE_DIR):
        return {}
    marks = {}
    for name in os.listdir(STATE_DIR):
        path = os.path.join(STATE_DIR, name)
        if not os.path.isfile(path):
            continue
        with open(path, "rb") as handle:
            marks[name.upper()] = (os.path.getsize(path),
                                   hashlib.sha256(handle.read()).hexdigest())
    return marks


def do_town_save(command):
    """在城鎮的酒店存檔，建立下一關的續跑點。

    打完一關進城鎮之後跑這個：`enter` 進目前那一棟（載入或戰後都停在 0 號酒店）、
    `right` 一次切到第二個圖示、`enter`、再 `enter` 確認，畫面回「記錄儲存完畢！」。
    判準是覆蓋層裡 `FD2.SAV` 的內容雜湊變了沒——第二關以後覆蓋層本來就有一份
    （上一關的續跑點載進來的），「多出一個檔」不再成立。

    存不成功就失敗收場。默默往下走會讓下一輪從舊存檔起跑，而 log 看起來完全正常。
    """
    steps = int(command.get("steps", 10_000_000))
    before = save_fingerprint()
    print(f"town_save：存檔前 {sorted(before) or '（空）'}", flush=True)
    for note, key in (("open", "enter"), ("slot", "right"),
                      ("pick", "enter"), ("confirm", "enter")):
        seq, current = send(key, max(steps, 20_000_000))
        report(seq, key, current, f" town-save={note}")
        settle(steps, int(command.get("settle", 4)))
    after = save_fingerprint()
    changed = [name for name, mark in after.items() if before.get(name) != mark]
    if "FD2.SAV" not in changed:
        print(f"town_save：FD2.SAV 沒有變（變的是 {changed or '（沒有）'}）",
              file=sys.stderr)
        return False
    print(f"town_save：FD2.SAV 已更新，大小 {after['FD2.SAV'][0]}", flush=True)
    # 退回城鎮，讓後面的建築切換從已知狀態開始。
    for _ in range(int(command.get("escape_max", 4))):
        if ui_mode(state()) == "town":
            break
        seq, current = send("esc", max(steps, 10_000_000))
        report(seq, "esc", current, " town-save=leave")
        settle(steps, 3)
    return True


def do_shop_probe(command):
    """進到店家（酒店／教會／商店）之後逐項按下去，看哪一項會寫出存檔。

    FD2 的存檔在酒店。店內是一排功能圖示，方向鍵移動選擇、enter 確認，但選中的是
    哪一個從狀態層看不出來——所以逐項試，用「可寫覆蓋層多了什麼檔」當判準：那是
    檔案系統層的事實，比讀畫面可靠。
    """
    steps = int(command.get("steps", 8_000_000))
    move = command.get("move", "right")
    before = saved_files()
    print(f"shop_probe：起始覆蓋層檔案 {sorted(before) or '（空）'}", flush=True)
    for index in range(int(command.get("slots", 4))):
        if index:
            seq, current = send(move, steps)
            report(seq, move, current, f" shop-probe[{index}]")
        seq, current = send("enter", max(steps, 10_000_000))
        report(seq, "enter", current, f" shop-probe[{index}]")
        current = settle(steps, int(command.get("probe_settle", 6)))
        # 選了一項之後常接一段確認對白（「要住宿嗎」「要存檔嗎」）。先前一律 esc
        # 取消，所以永遠看不到它到底會不會寫檔。confirm 打開就按到底。
        if command.get("confirm", True):
            for _ in range(int(command.get("confirm_max", 4))):
                if ui_mode(state()) != "dialogue":
                    break
                seq, current = send("enter", max(steps, 8_000_000))
                report(seq, "enter", current, f" shop-probe[{index}]=confirm")
                settle(steps, 3)
        now = saved_files()
        if now - before:
            print(f"shop_probe[{index}]：覆蓋層多了 {sorted(now - before)}", flush=True)
            before = now
        else:
            print(f"shop_probe[{index}]：介面 {ui_mode(current)}，覆蓋層沒有新檔",
                  flush=True)
        # 退回店家主畫面再試下一項。
        for _ in range(int(command.get("escape_max", 4))):
            mode = ui_mode(state())
            if mode in {"shop", "town"}:
                break
            seq, current = send("esc", max(steps, 5_000_000))
            report(seq, "esc", current, f" shop-probe[{index}]=leave")
    return True




def do_sweep_round(command):
    """這一回合：讓打得到人的我方單位各打一次，然後用系統選單結束回合。

    打不到的單位不動它——推進之後要待機才算行動結束，而待機要在六格 command grid
    上選對格子，那從狀態層看不出來。

    挑單位不靠 record `+5` bit7：它在換手、升級對白與「退出指令環沒行動」這幾種
    情況下都不成立，會讓同一個單位被反覆選中。驅動端自己記得這一輪處理過誰，
    而且記的是 identity 不是座標——推進過的單位座標就變了，拿座標比對等於沒記。
    """
    steps = int(command.get("steps", 2_000_000))
    reach = int(command.get("reach", 2))
    span = int(command.get("typical_move", 6))
    handled = set()
    for _ in range(int(command.get("max_units", 12))):
        if not resume_battle(int(command.get("cutscene_steps", 5_000_000)),
                             int(command.get("cutscene_max", 30))):
            print("sweep_round：離開戰場且推不回來，中止", file=sys.stderr)
            return False
        current = state()
        enemies = [(e["x"], e["y"]) for e in side(current, ENEMY_CAMP)]
        if not enemies:
            print("sweep_round：敵方已全滅", flush=True)
            return True
        pending = [u for u in side(current, ALLY_CAMP)
                   if not acted(u) and unit_key(u) not in handled]
        # 只挑走得到敵人旁邊的：貼敵格離它不超過一般移動力加射程。
        reachable = [u for u in pending
                     if min(distance((u["x"], u["y"]), e) for e in enemies) <= span + reach]
        if not reachable and pending and command.get("advance", True):
            # 一個都走不到：改成朝敵人推進。原版的關卡常把兩軍擺在地圖兩端——
            # 第二關我方在東側 (20..24, 13..16)、敵方在西側，相距十幾格——只處理
            # 「走得到」的話每一回合都是原地結束回合，戰鬥永遠不會開始。
            # `engage_targets` 的第二段本來就會產生沿路逼近的落腳格，這裡只是
            # 別把單位先擋在門外。
            reachable = pending
        if not reachable:
            break
        reachable.sort(key=lambda u: (min(distance((u["x"], u["y"]), e) for e in enemies),
                                      u["x"], u["y"]))
        pick = reachable[0]
        handled.add(unit_key(pick))
        inner = dict(command)
        inner.pop("sweep_round", None)
        inner["engage"] = [pick["x"], pick["y"]]
        if not do_engage(inner):
            return False
    print(f"sweep_round：這一回合處理了 {len(handled)} 個單位，結束回合",
          flush=True)
    return end_turn(command)


def do_sweep_battle(command):
    """一路打到敵方全滅：每回合掃完我方單位、送 END 收尾，再等回合數推進。

    `do_sweep_round` 收尾一定送 END。全員行動完原版有時會自己換手，但「有時」
    不夠——打不到人而待機的單位、以及推進到一半的單位，換手條件不一定成立，
    那時 `await` 會空等到預算用完。
    """
    global BATTLE_UNIT_BASE, MAX_ROUND_SEEN
    steps = int(command.get("steps", 2_000_000))
    rounds = int(command.get("rounds", 30))
    BATTLE_UNIT_BASE = state().get("unit_base")
    MAX_ROUND_SEEN = int((state().get("view") or {}).get("round", 0))
    print(f"sweep_battle：戰場單位陣列基底 {BATTLE_UNIT_BASE:#x}", flush=True)
    for index in range(rounds):
        current = state()
        if in_battle(current) and not side(current, ALLY_CAMP):
            # 我方全滅。原版接著跑戰敗流程，回合數再也不會推進——不先認出來的話
            # 後面的 await 會空等三百格，log 最後看起來只是「等換手等不到」。
            print(f"sweep_battle：第 {measure(current, 'round')} 回合我方全滅，戰敗",
                  file=sys.stderr)
            return False
        if not in_battle(current):
            print(f"sweep_battle：第 {index} 輪之前已離開戰場，收工", flush=True)
            return True
        if not side(current, ENEMY_CAMP):
            # 增援是在回合中途登場的，打完最後一個敵人的那一瞬間場上可能真的是空
            # 的，下一格就冒出六個。提早收工會把後面的城鎮動作送進戰場，而 log
            # 看起來像正常結束。多等幾格再確認一次。
            current = settle(int(command.get("turn_steps", 10_000_000)),
                             int(command.get("clear_settle", 8)))
            if not in_battle(current):
                print(f"sweep_battle：敵方清空後離開戰場（第 {index} 輪）",
                      flush=True)
                return True
            if side(current, ENEMY_CAMP):
                print("sweep_battle：敵方看似全滅，等了幾格之後又有單位在場"
                      "（增援），繼續打", flush=True)
                continue
            print(f"sweep_battle：敵方全滅（第 {index} 輪之前）", flush=True)
            return True
        before = measure(current, "round")
        inner = dict(command)
        inner.pop("sweep_battle", None)
        if not do_sweep_round(inner):
            return False
        print(f"sweep_battle：第 {before} 回合我方行動完畢，等換手", flush=True)
        if not do_await({"await": f"round>={before + 1}",
                         "abort_if": "ally_alive<=0",
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
        if "shop_probe" in command:
            if not do_shop_probe(command):
                return 12
            continue
        if "town_probe" in command:
            if not do_town_probe(command):
                return 11
            continue
        if "town_save" in command:
            if not do_town_save(command):
                return 13
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
