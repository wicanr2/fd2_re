#!/usr/bin/env python3
"""炎龍騎士團2 — 把一張戰場輸出成 Go/Ebiten 引擎用的資產(tileset 網格 PNG + 地圖 JSON)。

輸出:
  <out>/tileset.png  該 tileset 全部 24×24 圖塊排成網格(cols 欄)
  <out>/map.json     {"w","h","tileW","tileH","cols","tiles":[地形索引...],"cost":[移動成本...]}

引擎(remake/cmd/fd2)讀這兩個檔即可渲染地圖。資產屬遊戲著作權,只在本機,不入庫。

"cost" 陣列(第 8 輪新增,worklist「地形屬性接線」):per-tile 移動成本,由地形控制表的
「移動資訊」代碼(見 tools/dump_terrain_table.py、docs/knowledge-base/01-… §5)換算而來:
    0 正常          -> 1
    1 不可移動      -> BLOCKED_COST(99,遠大於任何 MV,Reachable/Path 天然擋停不用特判)
    2 森林·僅騎兵減速 -> 1(remake 尚無騎兵/步行/飛行兵種分類,見下方限制說明,先不罰步行)
    3 森林·全體減速   -> 1(同上;AP/DP 加成本身也還沒接,只接「可否通行」)
    4 沼澤·全體減速   -> 2(references/text/notes.md 玩家攻略「步行-2」直接對應)
    5 不可移動      -> BLOCKED_COST
    其他未知代碼     -> 1(保守,不無故擋路)
換算依據見 tools/dump_terrain_table.py 檔頭。**限制**:上表只用「步行」單位的移動力扣減,
notes.md 另有騎兵/飛行的差異扣減(如森林騎兵-2),remake Unit 尚無兵種欄位,之後要接的話
需在 battle.Unit 加 MoveType 並讓 MoveCost 依單位種類查不同係數,此處先留唯一步行成本。
若地圖 tile index 超出地形表格數(極少數 tileset 地形表格數 < tile count,如 map24),
cost 保守回退 1(不擋路,寧可錯放不錯擋)。

用法:
    python3 export_engine_assets.py <FDFIELD構成.bin> <FDSHAP_tileset.bin> <palette.bin> <out目錄> [cols] [FDSHAP_terrain.bin]
"""
import sys
import os
import json
import struct
from PIL import Image

sys.path.insert(0, os.path.dirname(__file__))
from render_map import decode_tileset, load_palette

BLOCKED_COST = 99

MOVE_CODE_TO_WALK_COST = {
    0: 1,
    1: BLOCKED_COST,
    2: 1,
    3: 1,
    4: 2,
    5: BLOCKED_COST,
}


def load_terrain_records(terrainp):
    """讀地形控制表(每格 4B)，回傳 (byte0 flags, 步行成本)。"""
    d = open(terrainp, "rb").read()
    n = len(d) // 4
    flags = [d[i * 4] for i in range(n)]
    costs = [MOVE_CODE_TO_WALK_COST.get(d[i * 4 + 1], 1) for i in range(n)]
    return flags, costs


def main(argv):
    if len(argv) < 5:
        print(__doc__); return 1
    fieldp, shapp, palp, out = argv[1], argv[2], argv[3], argv[4]
    cols = int(argv[5]) if len(argv) > 5 else 16
    terrainp = argv[6] if len(argv) > 6 else None
    os.makedirs(out, exist_ok=True)
    pal = load_palette(palp)
    tw, th, tiles = decode_tileset(shapp)
    rows = (len(tiles) + cols - 1) // cols
    sheet = Image.new("P", (cols * tw, rows * th), 0)
    sheet.putpalette(pal)
    for i, px in enumerate(tiles):
        sheet.paste(Image.frombytes("P", (tw, th), px), ((i % cols) * tw, (i // cols) * th))
    sheet.convert("RGB").save(os.path.join(out, "tileset.png"))

    d = open(fieldp, "rb").read()
    w, h = struct.unpack_from("<HH", d, 0)
    tilesidx = [struct.unpack_from("<H", d, 4 + i * 4)[0] for i in range(w * h)]
    event_words = [struct.unpack_from("<H", d, 6 + i * 4)[0] for i in range(w * h)]
    meta = {"w": w, "h": h, "tileW": tw, "tileH": th, "cols": cols, "tiles": tilesidx}
    if terrainp:
        terrain_flags, costs = load_terrain_records(terrainp)
        oob = 0
        cost_arr = []
        treasure_slots = []
        treasure_hidden = []
        for ti, event_word in zip(tilesidx, event_words):
            if 0 <= ti < len(costs):
                cost_arr.append(costs[ti])
                flags = terrain_flags[ti]
                if flags & 0x60:  # 0x20=普通寶箱、0x40=隱藏物品
                    treasure_slots.append(event_word & 0x1F)
                    treasure_hidden.append(bool(flags & 0x40))
                else:
                    treasure_slots.append(-1)
                    treasure_hidden.append(False)
            else:
                cost_arr.append(1)
                treasure_slots.append(-1)
                treasure_hidden.append(False)
                oob += 1
        meta["cost"] = cost_arr
        # 與 tiles/cost 同為 row-major 陣列。slot0 合法，-1 才表示非寶箱格。
        meta["treasure_slots"] = treasure_slots
        meta["treasure_hidden"] = treasure_hidden
        if oob:
            print(f"警告:{oob} 格 tile index 超出地形表範圍({len(costs)} 格),已回退 cost=1")
    json.dump(meta, open(os.path.join(out, "map.json"), "w"), separators=(",", ":"))
    tail = " + cost[]" if terrainp else "(無地形表,未加 cost)"
    print(f"tileset.png ({cols}×{rows} 圖塊) + map.json ({w}×{h}){tail} -> {out}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
