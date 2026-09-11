#!/usr/bin/env python3
"""戰場編輯器的移動成本換算要和匯出管線一致。

編輯器畫完 tile 之後自己重算 `cost`（不沿用舊值），所以它必須和
`tools/export_engine_assets.py` 用同一張表、同一條規則。兩邊各有一份常數，任何
一邊被單獨改到，地圖在編輯器裡看起來會通、在遊戲裡卻走不過去——而那要玩到那一格
才會發作。

這支測試從三個地方各取一次同一條規則再互相對照：

1. `tools/editor/battlefield.html` 裡的 JS 常數（用正則抽出來，不是複製一份）；
2. `tools/export_engine_assets.py` 的 `MOVE_CODE_TO_WALK_COST`；
3. 實際受版控的 `map.json`——用規則重算每一格，和檔案裡已經匯出的 `cost` 比對。

第 3 點是真正的正對照：前兩點只證明兩份常數長得一樣，就算兩邊一起寫錯也會通過。
"""

import json
import pathlib
import re
import sys
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
EDITOR = ROOT / "tools" / "editor" / "battlefield.html"
EXPORTER = ROOT / "tools" / "export_engine_assets.py"
MAPS = ROOT / "remake" / "assets" / "maps"


def editor_cost_table():
    """從編輯器的 HTML 抽出那張表。抽不到就是它被改名或搬走了，要失敗。"""
    text = EDITOR.read_text(encoding="utf-8")
    match = re.search(r"const MOVE_CODE_TO_WALK_COST = \{([^}]*)\}", text)
    if not match:
        raise AssertionError("battlefield.html 裡找不到 MOVE_CODE_TO_WALK_COST")
    return {int(k): int(v) for k, v in re.findall(r"(\d+)\s*:\s*(\d+)", match.group(1))}


def exporter_cost_table():
    namespace = {}
    text = EXPORTER.read_text(encoding="utf-8")
    match = re.search(r"MOVE_CODE_TO_WALK_COST = \{([^}]*)\}", text)
    if not match:
        raise AssertionError("export_engine_assets.py 裡找不到 MOVE_CODE_TO_WALK_COST")
    blocked = int(re.search(r"BLOCKED_COST = (\d+)", text).group(1))
    namespace["BLOCKED_COST"] = blocked
    for code, value in re.findall(r"(\d+)\s*:\s*(\w+)", match.group(1)):
        namespace.setdefault("table", {})[int(code)] = (
            blocked if value == "BLOCKED_COST" else int(value))
    return namespace["table"]


def cost_for_tile(index, control, table):
    """編輯器的 costForTile：地形表比 tile 數少時保守回 1。"""
    at = index * 4 + 1
    if not control or at >= len(control):
        return 1
    return table.get(control[at], 1)


class TerrainCostRules(unittest.TestCase):
    def setUp(self):
        self.editor = editor_cost_table()
        self.exporter = exporter_cost_table()

    def test_editor_and_exporter_agree(self):
        self.assertEqual(self.editor, self.exporter,
                         "編輯器與匯出管線的移動成本表不一致")

    def test_blocked_code_is_not_walkable(self):
        """正對照：表本身要真的擋得住路，不是一張全 1 的表也能通過上一條。"""
        self.assertGreater(self.editor[1], 20, "代碼 1（不可移動）要遠大於任何 MV")
        self.assertGreater(self.editor[5], 20, "代碼 5（不可移動）要遠大於任何 MV")
        self.assertEqual(self.editor[0], 1, "代碼 0（正常）應為 1")
        self.assertEqual(self.editor[4], 2, "代碼 4（沼澤）應為 2")

    def test_rule_reproduces_every_shipped_map(self):
        """用規則重算受版控地圖的每一格，要和匯出的 cost 逐格相同。"""
        checked = 0
        for map_json in sorted(MAPS.glob("map*/map.json")):
            data = json.loads(map_json.read_text(encoding="utf-8"))
            cost, tiles = data.get("cost"), data.get("tiles")
            control = data.get("native_terrain_control")
            if not cost or not tiles:
                continue
            checked += 1
            for at, tile in enumerate(tiles):
                want = cost[at]
                got = cost_for_tile(tile, control, self.editor)
                self.assertEqual(got, want,
                                 f"{map_json.parent.name} 第 {at} 格 tile={tile}："
                                 f"規則算出 {got}，匯出的是 {want}")
        self.assertGreater(checked, 0, "一張帶 cost 的地圖都沒讀到")


if __name__ == "__main__":
    sys.exit(0 if unittest.main(exit=False).result.wasSuccessful() else 1)
