#!/usr/bin/env python3
"""炎龍騎士團2 — 傾印 FDSHAP 地形控制表(全 33 tileset)成 JSON。

背景(見 docs/knowledge-base/01-container-and-asset-formats.md §5、
references/text/modify2.md「修改：FDSHAP.DAT」):

`FDSHAP.DAT`(解包後 67 資源)以「大資源(tileset) / 小資源(地形控制表)」交替成對:
    資源 2N   = tileset N 的圖塊庫(標頭 tw,th,count + offset 表 + bg-RLE 圖塊)
    資源 2N+1 = tileset N 的地形控制表,**per-tile**、每格 4 byte:
        byte0  寶箱資訊  bit5(0x20)=寶箱  bit6(0x40)=隱藏物品
        byte1  移動資訊(代碼 0-5,見下)
        byte2-3 戰鬥背景編號 uint16 LE

地形控制表大小 = 該資源檔案長度 / 4,**不是固定 300**(實測 96~400 格不等,依 tileset
而異;0x2422E 這個攻略提到的偏移只對應「某一個特定 tileset」,不可套用到全部)。
表格長度恆 >= 對應 tileset 的 tile count(多的格是保留,tile index 超出 count 者不會被
地圖引用)。

移動資訊代碼語意(青衫攻略 modify2.md 靜態定義 + references/text/notes.md 玩家攻略
「地形移動力/攻防影響」表交叉驗證 — AP/DP 修正數值完全吻合,見下方 MOVE_CODE_MEANING):
    0 = 正常狀態  (AP+5%,DP+0)
    1 = 不可移動  (AP+0,DP+0)
    2 = 森林類·僅騎兵減速 (AP-5%,DP+10%) — 對應 notes.md「森林」(步行-1=同一般,騎兵-2,飛行-1)
    3 = 森林類·全體減速   (AP-5%,DP+10%) — AP/DP 與 2 相同,notes.md 未拆分出對應地形名,存疑
    4 = 沼澤類·全體減速   (AP-5%,DP-5%) — 對應 notes.md「沼澤」(步行-2,騎兵-3,飛行-1)
    5 = 不可移動  (AP+0,DP+0)

本工具只傾印 byte0/byte1/byte2-3 原始值 + 語意標註,**不下 MoveCost 數字結論**——
那是 remake 端(tools/export_engine_assets.py)依 MOVE_CODE_TO_WALK_COST 換算的事,
留在這裡的是可交叉核對的原始表。

用法:
    python3 dump_terrain_table.py <raw解包根> <out.json>
        raw 解包根需含 raw/FDSHAP/(unpack_dat.py 產出)
"""
import sys
import os
import glob
import json
import struct

MOVE_CODE_MEANING = {
    0: "正常(AP+5%,DP+0)",
    1: "不可移動(AP+0,DP+0)",
    2: "森林類·僅騎兵減速(AP-5%,DP+10%)",
    3: "森林類·全體減速(AP-5%,DP+10%)[地形名存疑]",
    4: "沼澤類·全體減速(AP-5%,DP-5%)",
    5: "不可移動(AP+0,DP+0)",
}


def dump_one(terrain_bin: bytes, tile_count_hint: int | None = None):
    n = len(terrain_bin) // 4
    rows = []
    for i in range(n):
        b0, b1, b2, b3 = terrain_bin[i * 4:i * 4 + 4]
        rows.append({
            "idx": i,
            "chest": bool(b0 & 0x20),
            "hidden_item": bool(b0 & 0x40),
            "chest_byte_raw": b0,
            "move_code": b1,
            "move_meaning": MOVE_CODE_MEANING.get(b1, "未知代碼"),
            "bg_id": b2 | (b3 << 8),
        })
    return {"rows": n, "tile_count_hint": tile_count_hint, "entries": rows}


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    raw, out = argv[1], argv[2]
    shp = sorted(glob.glob(os.path.join(raw, "FDSHAP", "*.bin")),
                 key=lambda p: int(os.path.basename(p).split("_")[1].split(".")[0]))
    tilesets = []
    for m in range(len(shp) // 2):
        big, small = shp[m * 2], shp[m * 2 + 1]
        bd = open(big, "rb").read()
        cnt = struct.unpack_from("<H", bd, 4)[0] if len(bd) >= 6 else None
        td = open(small, "rb").read()
        entry = dump_one(td, tile_count_hint=cnt)
        entry["map_tileset_idx"] = m
        entry["tileset_res"] = os.path.basename(big)
        entry["terrain_res"] = os.path.basename(small)
        tilesets.append(entry)
    json.dump({"tilesets": tilesets, "move_code_meaning": MOVE_CODE_MEANING},
              open(out, "w", encoding="utf-8"), ensure_ascii=False, indent=1)
    print(f"{len(tilesets)} 個 tileset 地形控制表 -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
