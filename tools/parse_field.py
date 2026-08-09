#!/usr/bin/env python3
"""炎龍騎士團2 — FDFIELD 戰場定義解析(構成 / 控制 / 出場 三段)。

FDFIELD.DAT 每 3 資源 = 一張戰場:
  資源 3N+0 構成: u16 W, u16 H, 每格 (u16 地形索引, u16 事件/寶箱)
  資源 3N+1 控制: u8 地圖編號, u8 己方可出場數, u8 敵友出場總數,
                  回合事件[16]×3B(回合 u8, 全域事件id u8, 陣營 u8:0敵/1友/2特殊),
                  格子事件[16]×2B(全域事件id u8, selector u8),
                  寶箱[16]×3B(型態u8:0物品/1金錢, 內容u16),
                  出場人物[敵友總數]×26B(陣營,肖像,種族,職業,等級,物品×8,
                  initial-command-mask×4,其餘未解 bytes×4,group(波次),drop×4)
    註:單位 b21=group(出場波次,turn_events 觸發);增援單位的座標在 3N+2 出場位置段(地圖角落)。
    事件腳本「在 FDFIELD.DAT 資料」(非 EXE);EXE handler 只管勝負判定。remake 重新設計可擴充 DSL,不照搬本格式。
  資源 3N+2 出場位置: u16 人數, 每組 (u16 X, u16 Y, u16 肖像;0=己方)

用法:
    python3 parse_field.py <raw解包根> <map編號>        # 印該圖定義(JSON 摘要)
    python3 parse_field.py --all <raw解包根> <out.json>  # 全 33 圖 metadata → JSON
"""
import sys
import os
import json
import struct
import glob


def native_reward_kind(native_type):
    """只回傳 0x190ac 已證實的分支；其他型態保持事件。"""
    return "item" if native_type == 0 else "gold" if native_type == 1 else "event"


def native_death_effect(record):
    """解碼 0x10fa8..0x10fb2 的三位元組來源，不把 b25 併入。"""
    if len(record) < 26 or record[22] == 0xFF:
        return None
    return {"type": record[22], "value": record[23] | (record[24] << 8)}


def native_map_selector_key(record):
    """Return FDFIELD b1, the proven raw key passed to ``0x11019``."""
    if len(record) < 2:
        raise ValueError("FDFIELD roster record is shorter than b1")
    return record[1]


def native_turn_event_controls(control):
    """保留完整 16 列；turn=0xff 是原始休眠值，不是第 255 回合。"""
    return [
        {
            "turn": control[3 + slot * 3],
            "event_id": control[3 + slot * 3 + 1],
            "raw_camp": control[3 + slot * 3 + 2],
        }
        for slot in range(16)
    ]


def enabled_turn_events(controls):
    """舊版可執行摘要；休眠列只留在 turn_event_controls。"""
    return [
        {
            "turn": row["turn"],
            "event_id": row["event_id"],
            "camp": ["enemy", "ally", "special"][row["raw_camp"]]
            if row["raw_camp"] < 3
            else row["raw_camp"],
        }
        for row in controls
        if row["turn"] != 0xFF
    ]


def parse_map(raw, m):
    fld = sorted(glob.glob(os.path.join(raw, "FDFIELD", "*.bin")))
    comp = open(fld[m * 3], "rb").read()
    ctl = open(fld[m * 3 + 1], "rb").read()
    spw = open(fld[m * 3 + 2], "rb").read()
    w, h = struct.unpack_from("<HH", comp, 0)
    info = {"map": m, "w": w, "h": h,
            "own_deploy": ctl[1], "enemy_ally_total": ctl[2]}
    # 回合事件：完整 raw 列與啟用摘要分開，避免丟掉 handler 後續會改寫的
    # 0xff 休眠列，也避免把它錯當第 255 回合。
    controls = native_turn_event_controls(ctl)
    info["turn_event_controls"] = controls
    info["turn_events"] = enabled_turn_events(controls)
    o = 3
    o += 16*3
    # 0x13a44 以地圖構成 event-word low5 的 1-based slot 查這張表；
    # 第一 byte 寫入 [0x51a8f]，第二 byte 必須等於 caller selector。
    info["field_events"] = [
        {"slot": i, "event_id": ctl[o+i*2], "selector": ctl[o+i*2+1]}
        for i in range(16)
    ]
    o += 16*2
    info["chests"] = []
    for i in range(16):
        t = ctl[o+i*3]; v = struct.unpack_from("<H", ctl, o+i*3+1)[0]
        if t != 0xFF and v != 0:
            # 地圖構成每格第二個 word 的低 5 bit 直接索引這個 slot。
            # slot 0 是合法值（map10 星之眼即 slot0），不可用 truthiness 丟掉。
            # 0x190ac: type0 走 item、type1 走 gold，其餘型態把 value
            # 當作 0x51b91 全域事件 ID。未知型態不可再降成 item。
            kind = native_reward_kind(t)
            info["chests"].append(
                {"slot": i, "type": kind, "native_type": t, "value": v}
            )
    o += 16*3
    units = []
    for k in range(ctl[2]):
        b = ctl[o+k*26:o+(k+1)*26]
        if len(b) < 26:
            break
        death_effect = native_death_effect(b)
        units.append({"camp": ["enemy", "ally", "own"][b[0]] if b[0] < 3 else b[0],
                      # 0x10d7f reads FDFIELD b1 and 0x10ed6 passes that byte
                      # to 0x11019; the returned FDICON cache slot becomes
                      # runtime unit+2. This is the raw map selector source,
                      # not the camp byte.
                      "native_map_selector_key": native_map_selector_key(b),
                      # 0x10ec1/0x10ef5 copy FDFIELD b0 directly to runtime
                      # record +6. Preserve this raw provenance; it is not a
                      # normalized camp synonym in the native record.
                      "native_record_byte6": b[0],
                      # b1 is also copied to runtime +7/+8 and selects the
                      # native constructor tables. Historical exporters called
                      # it portrait, but a universal DATO/identity meaning has
                      # not been closed for both FDFIELD and persistent sources.
                      "raw_unit_key": b[1],
                      "portrait": b[1],  # legacy output alias; not ABI evidence
                      # Historical exporter labels kept only for normalized
                      # compatibility. 0x10c50 does not use b2/b3 as runtime
                      # +0x1f/+0x20: those bytes come from EXE tables. It
                      # copies b2 separately to +0x3d and does not read b3.
                      "race": b[2], "cls": b[3],
                      "native_record_byte3d": b[2],
                      "native_source_byte3": b[3],
                      "lv": b[4],
                      "inventory": [item for item in b[5:13] if item != 0xFF],
                      "inventory_slots": list(b[5:13]),
                      # Constructor 0x10f7f copies exactly b[13:17] to runtime
                      # unit+0x1a..+0x1d, then clears byte +0x1e.  Do not turn
                      # these bits into spell IDs here: they are native command
                      # inventory and individual ID effects remain unresolved.
                      "initial_command_mask": list(b[13:17]),
                      # Constructor 0x10fb6..0x10fc5 copies these exact source
                      # bytes to runtime +0x34/+0x35/+0x36.  Only +0x34 low
                      # nibble is the 0x13a9f dispatch mode; preserve all bits.
                      "native_record_byte34": b[17],
                      "native_record_byte35": b[18],
                      "native_record_byte36": b[19],
                      "native_source_byte20": b[20],
                      "group": b[21],
                      # 0=item、1=gold 已由原攻略確認；2/3 是特殊死亡效果，
                      # 語意未全解前保留原值，不猜成一般掉落物。
                      "death_effect": death_effect,
                      "native_record_death_effect": list(b[22:25]),
                      "native_source_byte25": b[25]})
                      # b21=出場波次 group；b22 + b23..24=runtime +0x31..33；
                      # b25 目前只保存原始來源，不併入效果 payload。
    info["units"] = units
    n = struct.unpack_from("<H", spw, 0)[0]
    info["positions"] = [list(struct.unpack_from("<HHH", spw, 2+k*6)) for k in range(n)]
    return info


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    if argv[1] == "--all":
        raw, out = argv[2], argv[3]
        fld = sorted(
            glob.glob(os.path.join(raw, "FDFIELD", "*.bin")),
            key=lambda p: int(os.path.basename(p).split("_")[1].split(".")[0]),
        )
        maps = []
        for m in range(len(fld)//3):
            try:
                maps.append(parse_map(raw, m))
            except Exception as e:
                maps.append({"map": m, "error": str(e)})
        json.dump(maps, open(out, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
        print(f"{len(maps)} 張戰場 metadata -> {out}")
        return 0
    info = parse_map(argv[1] if False else argv[1], int(argv[2]))
    print(json.dumps(info, ensure_ascii=False, indent=1))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
