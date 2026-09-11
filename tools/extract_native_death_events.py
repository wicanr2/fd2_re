#!/usr/bin/env python3
"""把死亡效果型態 2 用到的全域事件處理器轉寫成可編輯的型別化動作。僅限 Docker。

死亡效果（FDFIELD b22..b24 → runtime +0x31..+0x33）由 0x1B6B7 收集、0x1AA1D
逐筆分派：型態 0 物品、型態 1 金錢、型態 3 顯示章節戰場文字庫第 N 句、型態 2 在
delay(200) 之後呼叫全域事件表 0x51B91 的第 id 項，參數是擊殺者的索引。

本工具的轉寫是人工逐一讀出來的（下方 EVENTS），但每個動作都宣告它對應的原始
指令範圍，工具會重新反組譯、以動作專屬的樣式核對參數；另外從處理器入口沿著跳躍
走到 ret，檢查每一條非序言／收尾的指令都屬於某個動作或條件。漏轉寫或抄錯都會
直接失敗，不會產生一份看起來完整的資料。

用法（容器內）：
  python3 tools/extract_native_death_events.py --exe /orig/FD2.EXE --check
  python3 tools/extract_native_death_events.py --exe /orig/FD2.EXE --write
"""

import argparse
import hashlib
import json
import re
import struct
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / "tools"))
from capstone import CS_ARCH_X86, CS_MODE_32, Cs  # noqa: E402
from disasm_le import build_fixups, load_code  # noqa: E402
from le_xref import parse_le  # noqa: E402

OUTPUT = ROOT / "remake/assets/data/native_death_events.json"
REFERENCE = ROOT / "docs/data/fd2-reference-files.json"
EVENT_TABLE = 0x51B91
EVENT_COUNT = 90

# 全域位址（fixup 目標，LE linear）。
UNITS = 0x53A45        # runtime record 陣列指標，每筆 0x50 bytes
TEXT_BANK = 0x53A79    # 章節戰場文字庫（FDTXT 資源 = [0x53C03] + 1）
STATE = 0x53AD5        # battle-local 狀態表指標（重製端 NativeEventState）
CONTROL = 0x53A55      # 回合事件控制列指標（+3 起每 3 bytes 一列）
ROUND = 0x53BEF        # 回合計數
RANGE = 0x51A83        # 地圖範圍 overlay 選擇值
GATE = 0x53AFA         # 0x10B4E 的 raw placement gate
EXP = 0x53EC8          # 本次行動累積的經驗


def H(value):
    return f"0x{value:x}"


# ---- 轉寫 ----
# 每個動作的 ranges 是 [起, 迄) 的 LE linear 範圍；跨共用尾段時給多段。
# when 是條件：state_eq／state_ne（狀態表索引, 值）、round_lt（回合上限）、
# any_active（單位索引區間，含兩端）。條件在執行到該動作時才判斷，與原版
# 在同一時間點讀取狀態的順序一致。
def op(kind, ranges, **args):
    return {"op": kind, "ranges": ranges, **args}


EVENTS = [
    {"id": 4, "handler": 0x343E2, "ops": [
        op("record_bytes", [(0x343EC, 0x343FA)], unit=13, writes=[[6, 1, 1]]),
        op("dialogue", [(0x343FA, 0x34421)], text=7),
    ]},
    {"id": 5, "handler": 0x34D68, "ops": [
        op("spawn_group", [(0x34BEC, 0x34BF6)], group=1, gate=0),
        op("dialogue", [(0x34BF6, 0x34C1D)], text=1),
    ]},
    {"id": 12, "handler": 0x34594, "guards": [
        op("guard_state", [(0x3459E, 0x345AB)], index=0x10, equals=0),
    ], "ops": [
        op("ai_mode_range", [(0x345AB, 0x345B9)], first=0x18, last=0x1B, mode=7,
           when={"state_eq": [0x10, 0]}),
        op("dialogue", [(0x345B9, 0x345E0)], text=3, when={"state_eq": [0x10, 0]}),
        op("state_set", [(0x345E0, 0x345E9)], index=0x10, value=1,
           when={"state_eq": [0x10, 0]}),
    ]},
    {"id": 19, "handler": 0x34716, "guards": [
        op("guard_any_active", [(0x34724, 0x34728), (0x3475D, 0x34786)], first=7, last=0x24),
    ], "ops": [
        op("ai_mode_range", [(0x34728, 0x34736)], first=7, last=0x24, mode=7),
        op("dialogue", [(0x34736, 0x3475D)], text=8),
        op("dialogue", [(0x34786, 0x347AC)], text=0xB, style_register=True,
           when={"any_active": [7, 0x24]}),
    ]},
    {"id": 23, "handler": 0x34844, "guards": [
        op("guard_round", [(0x34883, 0x34890)], below=0xF),
    ], "ops": [
        op("ai_mode_range", [(0x3484E, 0x3485C)], first=8, last=0x1C, mode=0),
        op("dialogue", [(0x3485C, 0x34883)], text=4),
        op("spawn_group", [(0x34890, 0x3489A)], group=2, gate=0, when={"round_lt": 0xF}),
        op("pan", [(0x3489A, 0x348A6)], x=5, y=0x11, when={"round_lt": 0xF}),
        op("acting", [(0x348A6, 0x348B0)], resource=0x19, when={"round_lt": 0xF}),
        op("dialogue", [(0x348B0, 0x348D7)], text=5, when={"round_lt": 0xF}),
        op("pan", [(0x348D7, 0x348E3)], x=5, y=0x11, when={"round_lt": 0xF}),
        op("acting", [(0x348E3, 0x348ED)], resource=0x1A, when={"round_lt": 0xF}),
        op("mark_inactive", [(0x348ED, 0x348F7)], unit=0x21, when={"round_lt": 0xF}),
        op("range_one", [(0x35C18, 0x35C22)], when={"round_lt": 0xF}),
    ]},
    {"id": 24, "handler": 0x348FC, "ops": [
        op("dialogue", [(0x34906, 0x3491F), (0x34C0F, 0x34C1D)], text=3),
    ]},
    {"id": 29, "handler": 0x34A3C, "ops": [
        op("dialogue", [(0x34A46, 0x34A6D)], text=2),
        op("ai_byte_and_range", [(0x34A6D, 0x34A79), (0x34A19, 0x34A3A)],
           first=0xA, last=0x1B, mask=0x80),
    ]},
    {"id": 30, "handler": 0x34A7A, "ops": [
        op("ai_byte_set_range", [(0x34A85, 0x34AA7)], first=0xC, last=0x21, value=0),
        op("control_turn", [(0x34AA7, 0x34AB7)], slot=0, delta=1),
        op("control_turn", [(0x34AB7, 0x34AC8)], slot=1, delta=2),
        op("record_bytes", [(0x34AC8, 0x34AF0)], unit=11, writes=[
            [5, 0, 1], [6, 1, 1], [7, 6, 1], [8, 6, 1], [0x31, 0xFF, 1], [0x34, 0x80, 1],
            [0x40, 1, 2]]),
        op("dialogue", [(0x34AF0, 0x34B17)], text=2),
        op("spawn_group", [(0x34B17, 0x34B21)], group=1, gate=0),
        op("dialogue", [(0x34B21, 0x34B48)], text=3),
        op("exp_cancel", [(0x34B48, 0x34B52)]),
        op("state_set", [(0x34B52, 0x34B5B)], index=0x10, value=2),
    ]},
    {"id": 34, "handler": 0x34C6C, "ops": [
        op("dialogue", [(0x34906, 0x3491F), (0x34C0F, 0x34C1D)], text=3),
    ]},
    {"id": 37, "handler": 0x34CCC, "ops": [
        op("dialogue", [(0x34CD6, 0x34CFD)], text=1),
        op("pan", [(0x34CFD, 0x34D09)], x=0xF, y=0x22),
        op("spawn_group", [(0x34D09, 0x34D21)], group=3, gate=1),
        op("acting", [(0x34D21, 0x34D2B)], resource=0x2B),
        op("reset_pose", [(0x34D2B, 0x34D30)]),
        op("pan", [(0x34D30, 0x34D3C)], x=0, y=0x1A),
        op("spawn_group", [(0x34D3C, 0x34D54)], group=4, gate=1),
        op("acting", [(0x34D54, 0x34D5E)], resource=0x2C),
        op("reset_pose", [(0x34D5E, 0x34D63)]),
        op("range_one", [(0x35C18, 0x35C22)]),
    ]},
    {"id": 39, "handler": 0x34F74, "ops": [
        op("reward", [(0x34F83, 0x34F8D), (0x34F8D, 0x34F9E)], rodata=0x52742),
        op("dialogue", [(0x34F9E, 0x34FC5)], text=0xB),
    ]},
    {"id": 41, "handler": 0x34FF0, "ops": [
        op("dialogue", [(0x35009, 0x35030)], text=3),
        op("reward", [(0x34FFF, 0x35009), (0x35030, 0x35041)], rodata=0x52745),
        op("dialogue", [(0x35041, 0x3505A), (0x34FB7, 0x34FC5)], text=4),
    ]},
    {"id": 51, "handler": 0x3529A, "ops": [
        op("reward", [(0x352A9, 0x352C4)], rodata=0x52748),
        op("dialogue", [(0x352C4, 0x352DD), (0x34FB7, 0x34FC5)], text=3),
    ]},
    {"id": 53, "handler": 0x35321, "ops": [
        op("dialogue", [(0x3532B, 0x35352)], text=5),
        op("clear_hp_from", [(0x35352, 0x3535C)], first=0x12),
    ]},
    {"id": 64, "handler": 0x358EA, "guards": [
        op("guard_state", [(0x358F4, 0x35902)], index=0x10, equals=1),
        op("guard_state", [(0x3595D, 0x35962)], index=0x10, equals=2, reuse_register=True),
    ], "ops": [
        op("dialogue", [(0x35902, 0x35927)], text=1, text_register=True, style_register=True,
           when={"state_eq": [0x10, 1]}),
        op("staging", [(0x35927, 0x35935)], x=9, y=0x2C, group=3, when={"state_eq": [0x10, 1]}),
        op("staging", [(0x35935, 0x35943)], x=0, y=9, group=4, when={"state_eq": [0x10, 1]}),
        op("staging", [(0x35943, 0x35951)], x=0x11, y=9, group=5, when={"state_eq": [0x10, 1]}),
        op("range_one", [(0x35951, 0x3595B)], when={"state_eq": [0x10, 1]}),
        op("dialogue", [(0x35962, 0x35988)], text=2, text_register=True,
           when={"state_eq": [0x10, 2]}),
        op("clear_hp_from", [(0x35988, 0x35992)], first=0x10, when={"state_eq": [0x10, 2]}),
        op("state_inc", [(0x35992, 0x3599A)], index=0x10),
    ]},
    {"id": 67, "handler": 0x35A2F, "ops": [
        op("control_turn", [(0x35A39, 0x35A47)], slot=1, delta=0),
    ]},
    {"id": 71, "handler": 0x35B6B, "guards": [
        op("guard_state", [(0x35B75, 0x35B80)], index=0x13, not_equals=0),
    ], "ops": [
        op("dialogue", [(0x35B80, 0x35BA7)], text=2, when={"state_ne": [0x13, 0]}),
        op("clear_hp_from", [(0x35BA7, 0x35BB1)], first=0x14, when={"state_ne": [0x13, 0]}),
        op("state_inc", [(0x35BB1, 0x35BB9)], index=0x13),
    ]},
    {"id": 72, "handler": 0x35BF2, "ops": [
        op("staging", [(0x35BFC, 0x35C0A)], x=4, y=0x23, group=2),
        op("staging", [(0x35C0A, 0x35C18)], x=0xE, y=0x23, group=3),
        op("range_one", [(0x35C18, 0x35C22)]),
    ]},
    {"id": 73, "handler": 0x35C23, "ops": [
        op("state_set", [(0x35AAE, 0x35AB7)], index=0x12, value=1),
    ]},
    {"id": 77, "handler": 0x35EBE, "ops": [
        op("state_set", [(0x35EC8, 0x35ED1)], index=0x13, value=1),
    ]},
    {"id": 78, "handler": 0x35ED2, "ops": [
        op("state_set", [(0x35EDC, 0x35EE5)], index=0x14, value=1),
    ]},
    {"id": 81, "handler": 0x35F6F, "ops": [
        op("state_inc", [(0x35F79, 0x35F81)], index=0x10),
        op("control_turn", [(0x35F81, 0x35F91)], slot=0, delta=1),
    ]},
    {"id": 83, "handler": 0x36088, "ops": [
        op("dialogue", [(0x36092, 0x360B9)], text=8),
        op("clear_hp_from", [(0x360B9, 0x360BB), (0x35354, 0x3535C)], first=0x14),
    ]},
]

# 0x1AA1D 的兩個分派點：型態 2 先 delay(200) 再查表，型態 3 直接顯示文字庫第 N 句。
DISPATCH = {
    "event": {"ranges": [(0x1AC05, 0x1AC24)], "delay_ms": 200},
    "text": {"ranges": [(0x1AC26, 0x1AC51)]},
}


# ---- 反組譯 ----
class Image:
    def __init__(self, exe: Path):
        self.raw = exe.read_bytes()
        self.meta = parse_le(self.raw)
        self.code, self.base = load_code(self.raw, self.meta)
        self.fixups = build_fixups(self.raw, self.meta)
        self.cs = Cs(CS_ARCH_X86, CS_MODE_32)

    def insns(self, start, end):
        out = []
        off = start - self.base
        for insn in self.cs.disasm(self.code[off:off + (end - start)], start):
            if insn.address >= end:
                break
            target = next((self.fixups[a] for a in range(insn.address, insn.address + insn.size)
                           if a in self.fixups), None)
            out.append((insn.address, insn.mnemonic, insn.op_str, target, insn.size))
        if not out or out[-1][0] + out[-1][4] != end:
            raise SystemExit(f"範圍 {H(start)}..{H(end)} 沒有落在指令邊界上")
        return out

    def data(self, linear, length):
        obj = next(o for o in self.meta["objs"]
                   if o["base"] <= linear and linear + length <= o["base"] + o["vsize"])
        foff = self.meta["data_off"] + (obj["first"] - 1) * self.meta["page_size"] + (linear - obj["base"])
        return self.raw[foff:foff + length]

    def event_table(self):
        raw = self.data(EVENT_TABLE, EVENT_COUNT * 4)
        # 表項是 code object 內的偏移；code object 的 base 就是 0x10000。
        return [struct.unpack_from("<I", raw, i * 4)[0] + self.base for i in range(EVENT_COUNT)]


def text(insn):
    return f"{insn[1]} {insn[2]}".strip()


def mem(insn, linear):
    return insn[3] == linear


def imm(value):
    return H(value) if value >= 10 else str(value)


def expect_seq(insns, patterns, where):
    """逐條比對：pattern 是字串（完全相等）或 (字串, 全域位址) 或 callable。"""
    body = [i for i in insns if i[1] != "jmp"]
    if len(body) != len(patterns):
        raise SystemExit(f"{where}：指令數 {len(body)}，樣式 {len(patterns)}\n" +
                         "\n".join(text(i) for i in body))
    for insn, pattern in zip(body, patterns):
        if isinstance(pattern, tuple):
            want, linear = pattern
            ok = text(insn) == want and mem(insn, linear)
        elif callable(pattern):
            ok = pattern(insn)
        else:
            ok = text(insn) == pattern
        if not ok:
            raise SystemExit(f"{where}：{H(insn[0])} 是「{text(insn)}」，不符 {pattern!r}")


def dialogue_patterns(o):
    style = (lambda i: text(i) in ("push 1", "push eax")) if o.get("style_register") else "push 1"
    text_push = "push eax" if o.get("text_register") else f"push {imm(o['text'])}"
    return [style, "push 0x13", "push 0x4a", "push 0x4c", "push 0xcd", "push 0x140",
            "push 0xa0000", text_push, ("push dword ptr [0x3a79]", TEXT_BANK),
            "call 0x15f84", "add esp, 0x24"]


def check_op(image, event_id, o):
    where = f"事件 {event_id} {o['op']}"
    insns = [i for start, end in o["ranges"] for i in image.insns(start, end)]
    kind = o["op"]
    if kind == "dialogue":
        expect_seq(insns, dialogue_patterns(o), where)
    elif kind == "ai_mode_range":
        expect_seq(insns, [f"push {imm(o['mode'])}", f"push {imm(o['last'])}",
                           f"push {imm(o['first'])}", "call 0x3419c", "add esp, 0xc"], where)
    elif kind == "staging":
        expect_seq(insns, [f"push {imm(o['group'])}", f"push {imm(o['y'])}",
                           f"push {imm(o['x'])}", "call 0x35822", "add esp, 0xc"], where)
    elif kind == "pan":
        expect_seq(insns, [f"push {imm(o['y'])}", f"push {imm(o['x'])}",
                           "call 0x135dd", "add esp, 8"], where)
    elif kind == "acting":
        expect_seq(insns, [f"push {imm(o['resource'])}", "call 0x1366a", "add esp, 4"], where)
    elif kind == "spawn_group":
        body = [f"push {imm(o['group'])}", "call 0x10b4e", "add esp, 4"]
        if o["gate"]:
            body = [("mov byte ptr [0x3afa], 1", GATE), *body, ("mov byte ptr [0x3afa], 0", GATE)]
        expect_seq(insns, body, where)
    elif kind == "reset_pose":
        expect_seq(insns, ["call 0x134e4"], where)
    elif kind == "range_one":
        expect_seq(insns, [("mov dword ptr [0x1a83], 1", RANGE)], where)
    elif kind == "mark_inactive":
        expect_seq(insns, [f"push {imm(o['unit'])}", "call 0x32975", "add esp, 4"], where)
    elif kind == "clear_hp_from":
        expect_seq(insns, [f"push {imm(o['first'])}", "call 0x35bba", "add esp, 4"], where)
    elif kind == "exp_cancel":
        expect_seq(insns, [("mov dword ptr [0x3ec8], 0", EXP)], where)
    elif kind in ("state_set", "state_inc"):
        store = (f"mov byte ptr [eax + {imm(o['index'])}], {imm(o['value'])}" if kind == "state_set"
                 else f"inc byte ptr [eax + {imm(o['index'])}]")
        expect_seq(insns, [("mov eax, dword ptr [0x3ad5]", STATE), store], where)
    elif kind == "control_turn":
        offset = 3 + 3 * o["slot"]
        load_round = ("mov dl, byte ptr [0x3bef]", ROUND)
        load_row = ("mov eax, dword ptr [0x3a55]", CONTROL)
        store = f"mov byte ptr [eax + {offset}], dl"
        if o["delta"] == 0:
            expect_seq(insns, [load_row, load_round, store], where)
        elif o["delta"] == 1:
            expect_seq(insns, [load_round, "inc dl", load_row, store], where)
        else:
            expect_seq(insns, [load_round, f"add dl, {o['delta']}", load_row, store], where)
    elif kind == "record_bytes":
        body = [("mov eax, dword ptr [0x3a45]", UNITS), f"add eax, {H(o['unit'] * 0x50)}"]
        for offset, value, width in o["writes"]:
            size = "byte" if width == 1 else "word"
            body.append(f"mov {size} ptr [eax + {imm(offset)}], {imm(value)}")
        expect_seq(insns, body, where)
    elif kind == "ai_byte_set_range":
        expect_seq(insns, [
            f"mov edx, {imm(o['first'])}", "mov eax, edx", "shl eax, 2",
            "lea ebx, [edx + eax]", "shl ebx, 4", ("mov eax, dword ptr [0x3a45]", UNITS),
            f"mov byte ptr [ebx + eax + 0x34], {imm(o['value'])}", "inc edx",
            f"cmp edx, {imm(o['last'] + 1)}", lambda i: i[1] == "jl"], where)
    elif kind == "ai_byte_and_range":
        count = o["last"] - o["first"] + 1
        expect_seq(insns, [
            "push dword ptr [esp + 4]", "call 0x34a0e", "add esp, 4",
            "xor edx, edx", f"lea ebx, [edx + {imm(o['first'])}]", "mov eax, ebx", "shl eax, 2",
            "add ebx, eax", "shl ebx, 4", ("mov eax, dword ptr [0x3a45]", UNITS),
            f"and byte ptr [ebx + eax + 0x34], {H(o['mask'])}", "inc edx",
            f"cmp edx, {imm(count)}", lambda i: i[1] == "jl"], where)
    elif kind == "reward":
        expect_seq(insns, [
            "mov edi, esp", (f"mov esi, {H(o['rodata'] - 0x50000)}", o["rodata"]),
            "movsw word ptr es:[edi], word ptr [esi]", "movsb byte ptr es:[edi], byte ptr [esi]",
            "mov eax, esp", "push eax", "push 1", "push dword ptr [esp + 0x18]",
            "call 0x1aa1d", "add esp, 0xc"], where)
        raw = image.data(o["rodata"], 3)
        o["kind"], o["value"] = raw[0], raw[1] | (raw[2] << 8)
    elif kind == "guard_state":
        if "equals" in o:
            compare = ("test eax, eax" if o["equals"] == 0 else f"cmp eax, {imm(o['equals'])}")
            if o.get("reuse_register"):
                # 0x3595D：eax 仍是同一個處理器前面讀出的狀態值，不重讀。
                expect_seq(insns, [compare, lambda i: i[1] == "jne"], where)
            else:
                expect_seq(insns, [("mov eax, dword ptr [0x3ad5]", STATE),
                                   f"movzx eax, byte ptr [eax + {imm(o['index'])}]",
                                   compare, lambda i: i[1] == "jne"], where)
        else:
            expect_seq(insns, [("mov eax, dword ptr [0x3ad5]", STATE),
                               f"cmp byte ptr [eax + {imm(o['index'])}], {imm(o['not_equals'])}",
                               lambda i: i[1] == "je"], where)
    elif kind == "guard_round":
        expect_seq(insns, [(f"cmp dword ptr [0x3bef], {imm(o['below'])}", ROUND),
                           lambda i: i[1] == "jge"], where)
    elif kind == "guard_any_active":
        expect_seq(insns, [
            "mov byte ptr [esp], 0", f"mov ebx, {imm(o['first'])}", "inc ebx",
            f"cmp ebx, {imm(o['last'] + 1)}", lambda i: i[1] == "jge", "push ebx",
            "call 0x3453e", "add esp, 4", "test eax, eax", lambda i: i[1] == "jne",
            "mov byte ptr [esp], 1", "movzx eax, byte ptr [esp]", "cmp eax, 1",
            lambda i: i[1] == "jne"], where)
    else:
        raise SystemExit(f"{where}：未知的動作種類")
    return [H(start) for start, _ in o["ranges"]]


# ---- 覆蓋：沿控制流走一遍，每條非序言／收尾指令都要有人認領 ----
# 序言是 `push N; call 0x36cd7`（Watcom stack probe，doc59）；N 只在處理器入口
# 那一條算序言，其他位置的 push imm 都是參數，必須被動作認領。
FILLER = re.compile(
    r"^(call 0x36cd7|push (ebx|esi|edi|ebp)|pop (ebx|esi|edi|ebp)|"
    r"sub esp, (0x[0-9a-f]+|[0-9]+)|add esp, (0x[0-9a-f]+|[0-9]+)|ret|jmp 0x[0-9a-f]+)$")
PROBE = re.compile(r"^push (0x[0-9a-f]+|[0-9]+)$")


def reachable(image, entry):
    seen, todo, out = set(), [entry], {}
    while todo:
        addr = todo.pop()
        while addr not in seen:
            seen.add(addr)
            insn = _one(image, addr)
            out[addr] = insn
            mnemonic = insn[1]
            if mnemonic == "ret":
                break
            if mnemonic == "jmp":
                addr = int(insn[2], 16)
                continue
            if mnemonic.startswith("j"):
                todo.append(int(insn[2], 16))
            addr += insn[4]
    return out


def _one(image, addr):
    off = addr - image.base
    insn = next(image.cs.disasm(image.code[off:off + 16], addr))
    target = next((image.fixups[a] for a in range(insn.address, insn.address + insn.size)
                   if a in image.fixups), None)
    return (insn.address, insn.mnemonic, insn.op_str, target, insn.size)


def check_coverage(image, event):
    claimed = set()
    for o in event.get("guards", []) + event["ops"]:
        for start, end in o["ranges"]:
            claimed.update(i[0] for i in image.insns(start, end))
    missing = []
    for addr, insn in sorted(reachable(image, event["handler"]).items()):
        if addr in claimed or FILLER.match(text(insn)):
            continue
        if addr == event["handler"] and PROBE.match(text(insn)):
            continue
        missing.append(f"{H(addr)} {text(insn)}")
    if missing:
        raise SystemExit(f"事件 {event['id']} 有指令沒有被轉寫：\n" + "\n".join(missing))


def battle_usage():
    """哪些戰場（map 0..29）的單位帶這個死亡效果。map31／32 是劇情地圖，不打仗。"""
    usage = {}
    for path in sorted((ROOT / "remake/assets/maps").glob("map*/map*_units.json")):
        asset = json.loads(path.read_text(encoding="utf-8"))
        if asset["map"] > 29:
            continue
        for index, unit in enumerate(asset["units"]):
            effect = unit.get("death_effect")
            if effect and effect["type"] == 2:
                usage.setdefault(effect["value"], []).append({"map": asset["map"], "unit": index})
    return usage


def main():
    assert Path("/.dockerenv").exists(), "只允許在 Docker 內執行"
    ap = argparse.ArgumentParser()
    ap.add_argument("--exe", type=Path, required=True)
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true")
    mode.add_argument("--write", action="store_true")
    args = ap.parse_args()

    raw = args.exe.read_bytes()
    reference = json.loads(REFERENCE.read_text(encoding="utf-8"))
    exe = next(f for f in reference["files"] if f["file"] == "FD2.EXE")
    if (len(raw), hashlib.md5(raw).hexdigest(), hashlib.sha256(raw).hexdigest()) != \
            (exe["size"], exe["md5"], exe["sha256"]):
        raise SystemExit("FD2.EXE 與 fd2-reference-files.json 不符")
    image = Image(args.exe)
    table = image.event_table()

    for name, spec in DISPATCH.items():
        insns = [i for s, e in spec["ranges"] for i in image.insns(s, e)]
        if name == "event":
            expect_seq(insns, ["cmp eax, 2", lambda i: i[1] == "jne", "push 0xc8", "call 0x375b2",
                               "add esp, 4", "mov eax, ebx", "push edi",
                               ("call dword ptr [eax*4 + 0x1b91]", EVENT_TABLE), "add esp, 4"],
                       "型態 2 分派")
        else:
            expect_seq(insns, ["cmp eax, 3", lambda i: i[1] == "jne", "push 1", "push 0x13",
                               "push 0x4a", "push 0x4c", "push 0xcd", "push 0x140", "push 0xa0000",
                               "push ebx", ("push dword ptr [0x3a79]", 0x53A79), "call 0x15f84",
                               "add esp, 0x24"], "型態 3 分派")

    usage = battle_usage()
    events = []
    for event in EVENTS:
        if table[event["id"]] != event["handler"]:
            raise SystemExit(f"事件 {event['id']} 的表項是 {H(table[event['id']])}，轉寫寫的是 {H(event['handler'])}")
        for guard in event.get("guards", []):
            check_op(image, event["id"], guard)
        check_coverage(image, event)
        ops = []
        for o in event["ops"]:
            sources = check_op(image, event["id"], o)
            clean = {k: v for k, v in o.items() if k not in ("ranges", "style_register", "text_register")}
            if "rodata" in clean:
                clean["rodata"] = H(clean["rodata"])
            clean["source"] = sources[0]
            if len(sources) > 1:
                clean["tail_sources"] = sources[1:]
            ops.append(clean)
        events.append({"id": event["id"], "handler": H(event["handler"]),
                       "battle_units": usage.get(event["id"], []), "ops": ops})

    missing = sorted(set(usage) - {e["id"] for e in EVENTS})
    if missing:
        raise SystemExit(f"戰場上有型態 2 id 沒有轉寫：{missing}")

    document = {
        "schema_version": 1,
        "kind": "fd2_native_death_events",
        "source": {"file": "FD2.EXE", "size": exe["size"], "md5": exe["md5"], "sha256": exe["sha256"]},
        "generated_by": "tools/extract_native_death_events.py",
        "dispatch": {
            "collector": "0x1b6b7：+5 bit0 未設、+0x31 != 0xff、HP <= 0 的記錄依索引順序收集三個 byte",
            "dispatcher": "0x1aa1d(擊殺者索引, 筆數, 緩衝)",
            "event": {"table": H(EVENT_TABLE), "delay_ms": DISPATCH["event"]["delay_ms"],
                      "argument": "擊殺者索引", "source": H(DISPATCH["event"]["ranges"][0][0])},
            "text": {"bank": "FDTXT 資源 [0x53c03] + 1（0x101dc..0x101f6）",
                     "source": H(DISPATCH["text"]["ranges"][0][0])},
        },
        "events": events,
    }
    encoded = json.dumps(document, ensure_ascii=False, indent=2) + "\n"
    if args.write:
        OUTPUT.write_text(encoded, encoding="utf-8")
        print(f"已寫入 {OUTPUT.relative_to(ROOT)}：{len(events)} 個事件")
    else:
        if OUTPUT.read_text(encoding="utf-8") != encoded:
            raise SystemExit(f"{OUTPUT.relative_to(ROOT)} 與原版轉寫不一致；重跑 --write")
        print(f"{OUTPUT.relative_to(ROOT)} 與原版轉寫一致：{len(events)} 個事件")


if __name__ == "__main__":
    main()
