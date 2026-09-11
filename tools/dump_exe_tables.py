#!/usr/bin/env python3
"""炎龍騎士團2 — 從 FD2.EXE 傾印內嵌資料表。

第 2 輪逆向工程成果。當前手上的 FD2.EXE(357074 B, 1998 重打包)經錨定特徵比對，
資料表佈局完全對應青衫攻略所稱的「舊版」offset(8/9 錨定特徵命中於文件精確位置)。
本工具依這些 offset + 結構傾印各表成 JSON / CSV，並對關鍵表做數值自驗。

用法:
    python3 dump_exe_tables.py <FD2.EXE> [輸出目錄]

不依賴第三方套件。
"""
import sys
import os
import json
import struct

# 錨定特徵(用於確認版本對齊;偵測不到就警告)
ANCHORS = {
    "item":   (0x540AC, bytes.fromhex("0B010A005F00")),
    "shop":   (0x56190, bytes.fromhex("808184A5FF")),
    "spell":  (0x557FD, bytes.fromhex("32005A05")),
    "char":   (0x55BA1, bytes.fromhex("0101012A")),
    "growth": (0x55EA1, bytes.fromhex("06080406")),
    "learn":  (0x564B3, bytes.fromhex("05110901")),
    "resist": (0x51D96, bytes.fromhex("0A0000000A000000")),
    "crit":   (0x5219B, bytes.fromhex("05030305")),
    "unit":   (0x558F9, bytes.fromhex("010212000005")),
}

CLASS_NAMES = ["龍","劍士","戰士","騎士","弓兵","法師","僧侶","盜賊","武者","劍聖",
    "聖戰士","聖騎士","狙擊手","大法師","祭師","龍劍士","鬥士","英雄","魔戰士","龍騎士",
    "神射手","召喚師","聖者","忍者","武聖","機兵","？？？","　　","？？？"]


def u8(d, o):  return d[o]
def u16(d, o): return struct.unpack_from("<H", d, o)[0]


def dump_growth(d):
    """升級成長表:每人物 11B。AP0 AP1 DP0 DP1 DX0 DX1 HP0 HP1 MP0 MP1 MGidx。
    傾印到下一張表(shop @0x56190)之前。"""
    base = ANCHORS["growth"][0]
    rows = []
    o = base
    while o + 11 <= 0x56190:
        r = d[o:o + 11]
        rows.append({
            "idx": len(rows), "off": hex(o),
            "ap": [r[0], r[1]], "dp": [r[2], r[3]], "dx": [r[4], r[5]],
            "hp": [r[6], r[7]], "mp": [r[8], r[9]], "learn_idx": r[10],
            "raw": r.hex(),
        })
        o += 11
    return rows


def dump_command_learn(d, growth):
    """升級學招表: learn_idx×12B = 最多六組 (required_level, command_id)。

    `0x1e292` 取 growth row 的 learn_idx，經 `0x4e4a2` 取得
    0x626b3 + idx*12，逐 pair 比較 runtime level，命中即呼叫 0x1d79c OR
    command bit。FF/FF 是 unused pair，保留 raw 以避免憑空延長表。
    """
    used = [row["learn_idx"] for row in growth if row["learn_idx"] != 0xFF]
    count = max(used) + 1 if used else 0
    base = ANCHORS["learn"][0]
    rows = []
    for idx in range(count):
        o = base + idx * 12
        raw = d[o:o + 12]
        pairs = []
        for pos in range(0, 12, 2):
            level, command_id = raw[pos], raw[pos + 1]
            if level == 0xFF and command_id == 0xFF:
                break
            pairs.append({"required_level": level, "command_id": command_id})
        rows.append({"idx": idx, "off": hex(o), "entries": pairs, "raw": raw.hex()})
    return rows


def dump_unit(d):
    """敵/友單位每級成長:10B = RA CL HP MP AP DP DX MV EX。傾印到 char 表(0x55BA1)前。"""
    base = ANCHORS["unit"][0]
    rows = []
    o = base
    while o + 10 <= 0x55BA1:
        r = d[o:o + 10]
        # 結構:RA(1) CL(1) HP(2 LE) MP(1) AP(1) DP(1) DX(1) MV(1) EX(1) = 10B
        rows.append({
            "idx": len(rows), "off": hex(o), "race": r[0], "cls": r[1],
            "cls_name": CLASS_NAMES[r[1]] if r[1] < len(CLASS_NAMES) else "?",
            "hp": u16(d, o + 2), "mp": r[4], "ap": r[5], "dp": r[6], "dx": r[7],
            "mv": r[8], "ex": r[9], "raw": r.hex(),
        })
        o += 10
    return rows


def dump_character_defaults(d):
    """人物出場屬性:32×24B = RA CL LV HP MP MV CMD×4 IT×6 AP DP DX。

    這張表是 JOIN 建立命名角色時的 authoritative default。特別是索菲亞 index11
    的 IT[2]=0xD1 黃金徽章；FDFIELD 場景用的索菲亞 record 本身沒有這件物品。
    """
    base = ANCHORS["char"][0]
    rows = []
    for i in range(32):
        o = base + i * 24
        r = d[o:o + 24]
        rows.append({
            "idx": i, "off": hex(o), "race": r[0], "cls": r[1], "lv": r[2],
            "hp": u16(d, o + 3), "mp": u16(d, o + 5), "mv": r[7],
            # Constructor 0x10f7f copies these four bytes to runtime unit+0x1a..+0x1d.
            # They are the initial portion of the five-byte command bitset, not a spell table.
            "initial_command_mask": list(r[8:12]),
            "inventory": [item for item in r[12:18] if item != 0xFF],
            "inventory_slots": list(r[12:18]) + [0xFF, 0xFF],
            "ap": u16(d, o + 18), "dp": u16(d, o + 20), "dx": u16(d, o + 22),
            "raw": r.hex(),
        })
    return rows


def dump_spell(d):
    """法術功效:7B = DA DA HT DS RN MP WH。0x00..0x23 + 召喚 0x20..0x23 → 取 0x24 筆。"""
    base = ANCHORS["spell"][0]
    rows = []
    for i in range(0x24):
        o = base + i * 7
        r = d[o:o + 7]
        rows.append({
            "id": i, "off": hex(o), "dmg": u16(d, o), "hit": r[2],
            "dist": r[3], "range": r[4], "mp": r[5], "target": r[6], "raw": r.hex(),
        })
    return rows


def dump_item(d):
    """攻略/normalized 物品視圖，從 file 0x540ac 起每 23B 一筆。

    注意：這個視圖每筆的 byte 0 是前導 byte，並不是 0x4e56c 回傳的
    runtime row byte 0。原生 effect row 從 file 0x540ad 起，請使用
    dump_native_item_effect_rows；兩種視圖不得混用欄位 offset。
    """
    base = ANCHORS["item"][0]
    rows = []
    for i in range(0xD7):  # 攻略列至 0xD6
        o = base + i * 23
        if o + 23 > len(d):
            break
        r = d[o:o + 23]
        rows.append({
            "id": i, "off": hex(o), "type": r[1],
            "ap": u16(d, o + 2), "hit": u16(d, o + 4),
            "dp": u16(d, o + 6), "ev": u16(d, o + 8),
            "atk_attr": r[10], "atk_rate": r[11],
            "range": [r[12], r[13]], "K": list(r[14:20]),
            "price": u16(d, o + 20), "raw": r.hex(),
        })
    return rows


def dump_native_item_effect_rows(d, count=0xD7):
    """0x4e56c 的 raw runtime item-row 視圖（已知 215 筆前綴）。

    0x4e56c 回傳 linear 0x602ad + item*0x17；在目前錨定 EXE 中對應
    file 0x540ad + item*0x17。它比 dump_item 的 normalized/攻略視圖
    向後一 byte，因此每一筆 raw row 的最後一 byte 會跨入下一筆
    normalized row 的前導 byte。count=0xd7 只代表目前可追蹤的已知
    item ID 前綴，不宣稱 native table 在此結束。
    """
    file_base = ANCHORS["item"][0] + 1
    linear_base = 0x602AD
    stride = 0x17
    rows = []
    for i in range(count):
        o = file_base + i * stride
        raw = d[o:o + stride]
        if len(raw) != stride:
            break
        rows.append({
            "id": i,
            "off": hex(o),
            "linear": hex(linear_base + i * stride),
            "raw": raw.hex(),
        })
    return rows


def dump_native_movement_cost_rows(d):
    """0x4e555 selector→20-byte terrain-cost rows.

    The linear table is 0x61646..0x61889. In this anchored executable it maps
    to file 0x55446; 0x6188a begins the separately exported compatibility
    table, proving an exact 29-row boundary.

    對位依據：`18 01 12 00 17` 這串在檔案裡只出現一次，位於 0x55440，而反組譯
    把它放在 linear 0x61640，所以 linear 0x61646 = file 0x55446。早先寫的
    0x55445 讓每一列都早一個位元組，語意整個錯位：地形碼 1（不可移動）變成
    成本 1、碼 2（森林）變成 20，海與房舍可以走、森林反而是牆。
    """
    file_base = 0x55446
    linear_base = 0x61646
    stride = 20
    rows = []
    for selector in range(29):
        o = file_base + selector * stride
        raw = d[o:o + stride]
        if len(raw) != stride:
            break
        rows.append({
            "selector": selector,
            "off": hex(o),
            "linear": hex(linear_base + selector * stride),
            "costs": list(raw),
            "raw": raw.hex(),
        })
    return rows


def dump_resist_crit(d):
    """職業魔抗(4B/職業,由編號01起)+ 暴擊率(1B/職業)。"""
    rb, cb = ANCHORS["resist"][0], ANCHORS["crit"][0]
    rows = []
    for i in range(1, 0x1B):  # 01..1A
        rv = u8(d, rb + (i - 1) * 4)
        cv = u8(d, cb + (i - 1))
        rows.append({
            "cls": i, "name": CLASS_NAMES[i] if i < len(CLASS_NAMES) else "?",
            "resist_pct": (10 - rv) * 10 if rv <= 10 else None,
            "resist_raw": rv, "crit_pct": cv,
        })
    return rows


def dump_class_equip_types(d):
    """原版 0x1c1c3 的 class×item.type 六欄白名單（file 0x55689）。"""
    base, stride = 0x55689, 7
    rows = []
    for cls in range(29):
        o = base + cls * stride
        raw = list(d[o:o + stride])
        if len(raw) != stride:
            break
        # 第一個 byte 是表項常數 1；0x1c1c3 只掃後六個 item.type slots。
        rows.append({"cls": cls, "name": CLASS_NAMES[cls] if cls < len(CLASS_NAMES) else "?",
                     "types": raw[1:], "raw": raw})
    return rows


def write_out(out, name, rows):
    os.makedirs(out, exist_ok=True)
    with open(os.path.join(out, name + ".json"), "w", encoding="utf-8") as f:
        json.dump(rows, f, ensure_ascii=False, indent=1)


# 攻略 modify2 §5 升級成長字面值(舊版同 bytes)— 用於對齊自驗
GROWTH_CHECK = {
    0: "060804060203080c0000ff",  # 索爾
    1: "070a04060103 0a0f0000ff".replace(" ", ""),  # 哈諾
    2: "060904060203070c0000ff",  # 鐵諾
}
# 攻略 §8 敵友單位字面值(前幾筆友軍)
UNIT_CHECK = {
    0: ("士兵", 0x01, 0x02, 18, 0, 5, 2, 1, 4, 30),
    1: ("王國正規軍", 0x01, 0x03, 20, 0, 8, 2, 1, 7, 30),
}


def selftest(growth, command_learn, unit, characters, resist, equip):
    ok = True
    for i, exp in GROWTH_CHECK.items():
        got = growth[i]["raw"]
        m = got == exp
        ok &= m
        print(f"  升級成長[{i}] {'✓' if m else '✗ 期望 '+exp} {got}")
    expected_learn0 = [{"required_level": 5, "command_id": 17}, {"required_level": 9, "command_id": 1},
                       {"required_level": 15, "command_id": 26}, {"required_level": 21, "command_id": 2},
                       {"required_level": 26, "command_id": 27}]
    m = len(command_learn) == 20 and command_learn[0]["entries"] == expected_learn0 and command_learn[19]["entries"][-1] == {"required_level": 23, "command_id": 32}
    ok &= m
    print(f"  升級學招表 {'✓' if m else '✗ learn_idx 0/19 anchor mismatch'}")
    for i, exp in UNIT_CHECK.items():
        u = unit[i]
        got = (u["cls_name"], u["race"], u["cls"], u["hp"], u["mp"], u["ap"], u["dp"], u["dx"], u["mv"], u["ex"])
        m = got[1:] == exp[1:]
        ok &= m
        print(f"  敵友[{i}] {'✓' if m else '✗ 期望 '+str(exp)} {got}")
    # JOIN 11（索菲亞）的 EXE default IT[2] 必須是黃金徽章 0xD1。
    sophia_items = characters[11]["inventory"]
    m = sophia_items == [0x36, 0xA7, 0xD1]
    ok &= m
    print(f"  人物[11]索菲亞初始物品 {'✓' if m else '✗ 期望 [54,167,209]'} {sophia_items}")
    # 職業魔抗:法師(05)應 30%、聖騎士(0B)應 10%、僧侶(06)30%
    rd = {r["cls"]: r["resist_pct"] for r in resist}
    for cls, exp in [(0x05, 30), (0x06, 30), (0x0B, 10), (0x0D, 50)]:
        m = rd.get(cls) == exp
        ok &= m
        print(f"  魔抗[職業{cls:#x}] {'✓' if m else '✗ 期望 '+str(exp)} {rd.get(cls)}%")
    # 0x1c1c3 只掃 class record 的後六個 type slots；首 byte 1 是常數。
    for cls, exp in [(0, [0xFF] * 6), (1, [1, 21, 22, 0xFF, 0xFF, 0xFF]),
                     (25, [8, 27, 0xFF, 0xFF, 0xFF, 0xFF])]:
        m = equip[cls]["types"] == exp
        ok &= m
        print(f"  裝備相容[職業{cls}] {'✓' if m else '✗ 期望 '+str(exp)} {equip[cls]['types']}")
    return ok


def main(argv):
    if len(argv) < 2:
        print(__doc__); return 1
    d = open(argv[1], "rb").read()
    out = argv[2] if len(argv) > 2 else "exe_tables"
    print(f"FD2.EXE size = {len(d)} (0x{len(d):x})")

    # 版本對齊檢查
    print("錨定特徵對齊:")
    aligned = True
    for name, (off, pat) in ANCHORS.items():
        hit = d[off:off + len(pat)] == pat
        aligned &= hit
        print(f"  {name:7} @0x{off:x} {'✓' if hit else '✗'}")
    if not aligned:
        print("⚠ 部分錨定特徵未對齊,offset 可能不適用此版本!")

    growth = dump_growth(d)
    command_learn = dump_command_learn(d, growth)
    unit = dump_unit(d)
    characters = dump_character_defaults(d)
    spell = dump_spell(d)
    item = dump_item(d)
    native_item_effect_rows = dump_native_item_effect_rows(d)
    native_movement_cost_rows = dump_native_movement_cost_rows(d)
    rc = dump_resist_crit(d)
    equip = dump_class_equip_types(d)

    for name, rows in [("growth", growth), ("command_learn", command_learn), ("unit", unit), ("character_defaults", characters), ("spell", spell),
                       ("item", item), ("native_item_effect_rows", native_item_effect_rows),
                       ("native_movement_cost_rows", native_movement_cost_rows),
                       ("resist_crit", rc), ("class_equip_types", equip)]:
        write_out(out, name, rows)
        print(f"  -> {name}.json  ({len(rows)} 列)")

    print("數值自驗(對照青衫攻略字面值):")
    ok = selftest(growth, command_learn, unit, characters, rc, equip)
    print("自驗結果:", "全部通過 ✓" if ok else "有不符 ✗")
    return 0 if ok else 2


if __name__ == "__main__":
    sys.exit(main(sys.argv))
