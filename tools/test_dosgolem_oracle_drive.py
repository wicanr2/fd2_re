#!/usr/bin/env python3
"""dosgolem_oracle_drive.py 純狀態邏輯的測試。

閉環驅動的決策全在這幾支純函式裡：誰還沒行動、往哪一格走、條件成立了沒。
它們錯了不會噴錯，只會安靜地把按鍵送到錯的地方，最後產生一份「跑完了但走錯路」
的收據。所以逐條驗兩個方向，別只驗好路徑。

執行：python3 -m unittest discover -s tools -p 'test_dosgolem_oracle_drive.py'
"""

import importlib.util
import json
import os
import pathlib
import shutil
import tempfile
import unittest
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parent


def load():
    spec = importlib.util.spec_from_file_location(
        "fd2_oracle_drive", ROOT / "dosgolem_oracle_drive.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


drive = load()


def unit(x, y, camp, hp=10, acted=False, identity=None):
    raw = ["00"] * 80
    raw[5] = "80" if acted else "00"
    made = {"x": x, "y": y, "camp": camp, "hp": hp, "raw_hex": "".join(raw)}
    if identity is not None:
        made["identity"] = identity
    return made


class ModuleIntegrity(unittest.TestCase):
    """每個被呼叫的模組級名稱都要存在。

    這些函式大多要有 oracle 在跑才叫得動，單元測試碰不到它們的路徑；用整段字串
    取代改檔時，很容易把相鄰的函式定義一起截掉，而錯誤要等實跑十分鐘後才以
    NameError 現形。靜態掃一遍便宜得多。
    """

    def test_every_called_name_resolves(self):
        import ast
        import builtins
        source = (ROOT / "dosgolem_oracle_drive.py").read_text(encoding="utf-8")
        tree = ast.parse(source)
        defined = {node.name for node in ast.walk(tree)
                   if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef))}
        defined |= {target.id for node in ast.walk(tree)
                    if isinstance(node, ast.Assign)
                    for target in node.targets if isinstance(target, ast.Name)}
        local = set()
        for node in ast.walk(tree):
            if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
                local |= {a.arg for a in node.args.args}
                local |= {n.id for n in ast.walk(node)
                          if isinstance(n, ast.Name) and isinstance(n.ctx, ast.Store)}
        known = defined | local | set(dir(builtins)) | {
            "json", "os", "re", "sys", "time", "collections"}
        missing = sorted({node.func.id for node in ast.walk(tree)
                          if isinstance(node, ast.Call)
                          and isinstance(node.func, ast.Name)
                          and node.func.id not in known})
        self.assertEqual(missing, [], f"呼叫了不存在的名稱：{missing}")


class NestedCommandOptions(unittest.TestCase):
    """文件化的閉環命令參數位於命令名稱之下，不能靜默退回預設值。"""

    def test_nested_probe_options_override_and_keep_top_level_defaults(self):
        options = drive.command_options({
            "town_probe": {"moves": ["left", "left"], "probe_settle": 7},
            "steps": 123,
            "probe_settle": 4,
        }, "town_probe")
        self.assertEqual(options["moves"], ["left", "left"])
        self.assertEqual(options["probe_settle"], 7)
        self.assertEqual(options["steps"], 123)

    def test_boolean_command_preserves_legacy_top_level_options(self):
        options = drive.command_options({
            "shop_probe": True, "move": "left", "slots": 3,
        }, "shop_probe")
        self.assertEqual(options["move"], "left")
        self.assertEqual(options["slots"], 3)


class ActedFlag(unittest.TestCase):
    """record `+5` bit7 是「本回合已行動」。只看它，不看別的 byte。"""

    def test_bit7_both_directions(self):
        self.assertFalse(drive.acted(unit(1, 1, 2)))
        self.assertTrue(drive.acted(unit(1, 1, 2, acted=True)))

    def test_other_bits_in_the_same_byte_do_not_count(self):
        u = unit(1, 1, 2)
        for value, want in (("7f", False), ("80", True), ("ff", True), ("01", False)):
            raw = list(u["raw_hex"])
            raw[10:12] = list(value)
            u["raw_hex"] = "".join(raw)
            self.assertEqual(drive.acted(u), want, value)

    def test_short_record_is_not_treated_as_acted(self):
        self.assertFalse(drive.acted({"raw_hex": "00"}))
        self.assertFalse(drive.acted({}))


class Sides(unittest.TestCase):
    def setUp(self):
        self.current = {"units": [
            unit(1, 1, drive.ALLY_CAMP), unit(2, 2, drive.ALLY_CAMP, hp=0),
            unit(3, 3, drive.ENEMY_CAMP), unit(4, 4, drive.ENEMY_CAMP),
            unit(5, 5, drive.ENEMY_CAMP, hp=0),
        ]}

    def test_dead_units_are_excluded(self):
        self.assertEqual(len(drive.side(self.current, drive.ALLY_CAMP)), 1)
        self.assertEqual(len(drive.side(self.current, drive.ENEMY_CAMP)), 2)

    def test_unit_at_finds_only_living_units(self):
        self.assertIsNotNone(drive.unit_at(self.current, 1, 1))
        self.assertIsNone(drive.unit_at(self.current, 2, 2))
        self.assertIsNone(drive.unit_at(self.current, 9, 9))


class EngageTargets(unittest.TestCase):
    """候選落腳格：貼敵格優先、超出移動力的不試、逼近格由遠而近。"""

    def setUp(self):
        self.current = {"units": [
            unit(7, 14, drive.ALLY_CAMP, hp=42),
            unit(5, 18, drive.ALLY_CAMP, hp=48, acted=True),
            unit(3, 18, drive.ENEMY_CAMP, hp=28),
            unit(5, 19, drive.ENEMY_CAMP, hp=8),
        ]}

    def test_occupied_cells_are_never_offered(self):
        cells = drive.engage_targets(self.current, (7, 14))
        for taken in ((7, 14), (5, 18), (3, 18), (5, 19)):
            self.assertNotIn(taken, cells)

    def test_adjacent_cells_beyond_the_move_budget_are_dropped(self):
        near = drive.engage_targets(self.current, (7, 14), typical_move=6)
        far = drive.engage_targets(self.current, (7, 14), typical_move=2)
        self.assertIn((6, 19), near)          # 距離 6，移動力 6 時要試
        self.assertNotIn((6, 19), far)        # 移動力 2 時不該浪費試探

    def test_approach_cells_go_from_far_to_near(self):
        cells = drive.engage_targets(self.current, (7, 14), typical_move=6)
        approach = [c for c in cells if drive.distance(c, (7, 14)) <= 6
                    and c not in {(6, 19)}]
        spans = [drive.distance(c, (7, 14)) for c in approach]
        self.assertEqual(spans, sorted(spans, reverse=True), approach)

    def test_no_enemies_means_no_candidates(self):
        current = {"units": [unit(7, 14, drive.ALLY_CAMP)]}
        self.assertEqual(drive.engage_targets(current, (7, 14)), [])


class BattleGate(unittest.TestCase):
    """戰場閘門：單位陣列基底一變，units 就不能信。"""

    def tearDown(self):
        drive.BATTLE_UNIT_BASE = None
        drive.MAX_ROUND_SEEN = 0

    def test_without_a_known_base_the_content_decides(self):
        drive.BATTLE_UNIT_BASE = None
        self.assertFalse(drive.in_battle({"unit_base": 0x1043D0}))
        self.assertTrue(drive.in_battle({"unit_base": 0x1043D0, "units": [
            unit(7, 14, drive.ALLY_CAMP, hp=42), unit(3, 18, drive.ENEMY_CAMP, hp=28)]}))

    def test_matching_base_is_in_battle(self):
        drive.BATTLE_UNIT_BASE = 0x1765F0
        self.assertTrue(drive.in_battle({"unit_base": 0x1765F0}))

    def test_garbage_units_are_not_a_battle_even_with_a_new_base(self):
        drive.BATTLE_UNIT_BASE = 0x1765F0
        # 實測第一關第 3 回合哈諾加入的過場：camp 跑出 63／34／54、座標超出地圖。
        garbage = {"unit_base": 0x1043D0, "units": [
            {"x": 63, "y": 39, "hp": 16139, "camp": 63, "raw_hex": "00" * 80},
            {"x": 35, "y": 0, "hp": 14848, "camp": 34, "raw_hex": "00" * 80},
        ]}
        self.assertFalse(drive.in_battle(garbage))
        self.assertEqual(drive.BATTLE_UNIT_BASE, 0x1765F0, "垃圾不該被採納成新基底")

    def test_a_new_base_with_sane_units_is_adopted(self):
        """過場後 spawn 新單位會重配置陣列，基底本來就會變。"""
        drive.BATTLE_UNIT_BASE = 0x1765F0
        fresh = {"unit_base": 0x175D30, "units": [
            unit(7, 14, drive.ALLY_CAMP, hp=42), unit(3, 18, drive.ENEMY_CAMP, hp=28)]}
        self.assertTrue(drive.in_battle(fresh))
        self.assertEqual(drive.BATTLE_UNIT_BASE, 0x175D30)

    def test_units_without_any_ally_are_not_a_battle(self):
        drive.BATTLE_UNIT_BASE = None
        self.assertFalse(drive.in_battle({"unit_base": 1, "units": [
            unit(3, 18, drive.ENEMY_CAMP, hp=28)]}))
        self.assertFalse(drive.in_battle({"unit_base": 1, "units": []}))


class RoundMonotonicity(unittest.TestCase):
    """回合倒退就是「view 也不可信」，比看 units 內容可靠。"""

    def tearDown(self):
        drive.BATTLE_UNIT_BASE = None
        drive.MAX_ROUND_SEEN = 0

    def battle(self, base, round_no, units=None):
        return {"unit_base": base, "view": {"round": round_no},
                "units": units if units is not None else [
                    unit(7, 14, drive.ALLY_CAMP, hp=42),
                    unit(3, 18, drive.ENEMY_CAMP, hp=28)]}

    def test_round_going_backwards_is_never_in_battle(self):
        drive.BATTLE_UNIT_BASE = 0x1765F0
        self.assertTrue(drive.in_battle(self.battle(0x1765F0, 3)))
        self.assertEqual(drive.MAX_ROUND_SEEN, 3)
        # 過場：units 內容剛好合法，但回合掉回 1。
        self.assertFalse(drive.in_battle(self.battle(0x1765F0, 1)))

    def test_round_staying_or_advancing_is_in_battle(self):
        drive.BATTLE_UNIT_BASE = 0x1765F0
        drive.in_battle(self.battle(0x1765F0, 3))
        self.assertTrue(drive.in_battle(self.battle(0x1765F0, 3)))
        self.assertTrue(drive.in_battle(self.battle(0x1765F0, 4)))
        self.assertEqual(drive.MAX_ROUND_SEEN, 4)


class UIMode(unittest.TestCase):
    """介面模式由 input_chain 的特徵位址決定——那是唯一分得出四種介面的訊號。"""

    def chain(self, *addrs):
        return {"input_chain": list(addrs)}

    def test_each_interface_has_its_own_marker(self):
        for addrs, want in (
            (("0x11CED", "0x11AE4", "0x117F8", "0x25DD3"), "cursor"),
            (("0x11CED", "0x12DD9", "0x117AE", "0x18C5D"), "target"),
            (("0x17927", "0x17815", "0x18EEF", "0x25B3D"), "ring"),
            (("0x17927", "0x17815", "0x16FAE", "0x118C6"), "system"),
        ):
            self.assertEqual(drive.ui_mode(self.chain(*addrs)), want, addrs)

    def test_dialogue_covers_every_handler_variant(self):
        """升級訊息與事件台詞不只一支 handler，實測看到三個位址。"""
        for variant in ("0x1E44E", "0x1E5A8", "0x1E464"):
            self.assertEqual(drive.ui_mode(self.chain("0x16039", variant)), "dialogue")
        # 只有共同前綴也算——單一位址比對會讓同一種畫面有一部分掉進 unknown，
        # 而 unknown 的處置是等，對白等不出結果。
        self.assertEqual(drive.ui_mode(self.chain("0x16CF8", "0x16039")), "dialogue")

    def test_upper_frame_dialogue_counts_too(self):
        """說話者頭像在右的上框對白是另一支 handler（哈諾加入的那段）。"""
        self.assertEqual(
            drive.ui_mode(self.chain("0x164C4", "0x3424D", "0x1A4CC", "0x135CA")),
            "dialogue")
        self.assertEqual(drive.ui_mode(self.chain("0x16D05", "0x164C4")), "dialogue")

    def test_dialogue_marker_is_disjoint_from_the_others(self):
        """實測 98 個對白檢查點都不含其他四個標記，反之亦然。"""
        for other in ("0x117F8", "0x117AE", "0x18EEF", "0x16FAE"):
            self.assertNotEqual(other, "0x1E44E")
            self.assertNotEqual(drive.ui_mode(self.chain(other)), "dialogue")

    def test_the_moment_of_selection_counts_as_target(self):
        """選取那一格還沒收到鍵，鏈是 0x18BF4／0x18978；漏了它就會永遠等不到。"""
        self.assertEqual(
            drive.ui_mode(self.chain("0x12225", "0x18BF4", "0x18978", "0x394B0")),
            "target")

    def test_ring_wins_over_the_selection_marker(self):
        """指令環的鏈不含 0x18978，但順序仍要讓 0x18EEF 先判。"""
        self.assertEqual(
            drive.ui_mode(self.chain("0x17927", "0x17815", "0x18EEF", "0x18978")),
            "ring")

    def test_ring_and_system_share_a_menu_loop_but_stay_distinct(self):
        """兩者的鏈都含 0x17927／0x17815，差別只在第三項。"""
        self.assertNotEqual(
            drive.ui_mode(self.chain("0x17927", "0x17815", "0x18EEF")),
            drive.ui_mode(self.chain("0x17927", "0x17815", "0x16FAE")))

    def test_no_chain_or_no_marker_is_unknown(self):
        self.assertEqual(drive.ui_mode({}), "unknown")
        self.assertEqual(drive.ui_mode(self.chain("0x12211", "0x45D91")), "unknown")

    def test_town_and_shop_have_their_own_markers(self):
        """戰間城鎮上 enter 會進入目前選到的那一棟建築（例如商店）。"""
        self.assertEqual(
            drive.ui_mode(self.chain("0x1647C", "0x37391", "0x2CFFE", "0x2CE08")),
            "town")
        self.assertEqual(
            drive.ui_mode(self.chain("0x4E9FC", "0x2DA83", "0x2D947", "0x2D7D1")),
            "shop")
        # 商店 esc 退得掉；城鎮本身不是選單，退不出去也不該去退它。
        self.assertIn("shop", drive.ESCAPABLE)
        self.assertNotIn("town", drive.ESCAPABLE)

    def test_status_panel_has_its_own_marker(self):
        """單位狀態面板（能力值與裝備）：esc 退得掉，不是卡住。"""
        self.assertEqual(
            drive.ui_mode(self.chain("0x16D05", "0x1BA37", "0x1B961", "0x1BCE6")),
            "status")
        self.assertNotIn("status", drive.CURSOR_MODES)

    def test_dialogue_is_not_a_cursor_mode(self):
        """對白不吃方向鍵，也不會自己走完——每個等待迴圈都要送 enter 推它。"""
        self.assertNotIn("dialogue", drive.CURSOR_MODES)

    def test_only_cursor_and_target_move_the_map_cursor(self):
        self.assertEqual(drive.CURSOR_MODES, {"cursor", "target"})


class Conditions(unittest.TestCase):
    def setUp(self):
        self.current = {
            "view": {"round": 3, "cursor_x": 5, "cursor_y": 19},
            "steps": 1234,
            "units": [unit(1, 1, drive.ALLY_CAMP), unit(2, 2, drive.ENEMY_CAMP),
                      unit(3, 3, drive.ENEMY_CAMP)],
        }

    def test_measure_reads_each_variable(self):
        for name, want in (("round", 3), ("cursor_x", 5), ("cursor_y", 19),
                           ("ally_alive", 1), ("enemy_alive", 2), ("steps", 1234)):
            self.assertEqual(drive.measure(self.current, name), want, name)

    def test_every_operator_both_directions(self):
        for expression, want in (
            ("round>=3", True), ("round>=4", False),
            ("round<=3", True), ("round<=2", False),
            ("round==3", True), ("round==4", False),
            ("round!=4", True), ("round!=3", False),
            ("round>2", True), ("round>3", False),
            ("round<4", True), ("round<3", False),
            ("enemy_alive<=0", False), ("enemy_alive<=2", True),
        ):
            ok, _ = drive.holds(self.current, expression)
            self.assertEqual(ok, want, expression)

    def test_ally_wipeout_is_expressible_as_a_guard(self):
        """我方全滅之後回合再也不會推進，等待迴圈要認得出來而不是空等。"""
        wiped = {"view": {"round": 5}, "units": [unit(3, 3, drive.ENEMY_CAMP)]}
        self.assertTrue(drive.holds(wiped, "ally_alive<=0")[0])
        alive = {"view": {"round": 5}, "units": [unit(1, 1, drive.ALLY_CAMP)]}
        self.assertFalse(drive.holds(alive, "ally_alive<=0")[0])

    def test_unknown_variable_and_syntax_both_stop_the_run(self):
        with self.assertRaises(SystemExit):
            drive.holds(self.current, "morale>=1")
        with self.assertRaises(SystemExit):
            drive.holds(self.current, "round ~ 3")
        with self.assertRaises(SystemExit):
            drive.measure(self.current, "morale")


class ModifiedPathControls(unittest.TestCase):
    def snapshot(self, mode, enemies=0, injections=None):
        chains = {
            "dialogue": ["0x16039"],
            "town": ["0x2CE08"],
            "unknown": [],
        }
        units = [unit(i, 1, drive.ENEMY_CAMP) for i in range(enemies)]
        return {"input_chain": chains[mode], "kbd_pending": 0,
                "eip": "0x10000", "steps": 1, "view": {}, "units": units,
                "state_injections": injections or []}

    def test_await_ui_only_confirms_dialogue_and_stops_before_town_input(self):
        states = iter([self.snapshot("unknown"), self.snapshot("dialogue"),
                       self.snapshot("town")])
        sent = []

        def fake_send(key, steps, **control):
            sent.append((key, steps, control))
            return len(sent), self.snapshot("unknown")

        with mock.patch.object(drive, "state", side_effect=lambda: next(states)), \
                mock.patch.object(drive, "send", side_effect=fake_send), \
                mock.patch.object(drive, "report"):
            self.assertTrue(drive.do_await_ui(
                {"await_ui": "town", "steps": 123, "max": 3}))
        self.assertEqual(sent, [("", 123, {}), ("enter", 123, {})])

    def test_force_clear_requires_oracle_disclosure(self):
        before = self.snapshot("unknown", enemies=2)
        after = self.snapshot(
            "unknown", injections=["force-enemy-clear：執行 1 次、寫入 2 筆"])
        sent = []

        def fake_send(key, steps, **control):
            sent.append((key, steps, control))
            return 1, after

        with mock.patch.object(drive, "state", return_value=before), \
                mock.patch.object(drive, "send", side_effect=fake_send), \
                mock.patch.object(drive, "report"):
            self.assertTrue(drive.do_force_enemy_clear(
                {"force_enemy_clear": True, "steps": 456, "end_turn": False}))
        self.assertEqual(sent, [("", 456, {"force_enemy_clear": True})])

    def test_force_clear_fails_closed_without_disclosure(self):
        before = self.snapshot("unknown", enemies=1)
        after = self.snapshot("unknown")
        with mock.patch.object(drive, "state", return_value=before), \
                mock.patch.object(drive, "send", return_value=(1, after)), \
                mock.patch.object(drive, "report"):
            self.assertFalse(drive.do_force_enemy_clear(
                {"force_enemy_clear": True, "end_turn": False}))


class MoveUnit(unittest.TestCase):
    """move_unit：候選格依序試，被拒絕就換下一格；全被拒絕且 stay_if_blocked 才原地待機。"""

    def snapshot(self, x, y, round_=3):
        unit = {"index": 2, "camp": drive.ALLY_CAMP, "x": x, "y": y, "hp": 289,
                "raw_hex": "00" * 5 + "00" + "02" + "00" * 0x49}
        return {"input_chain": ["0x1a4e2"], "kbd_pending": 0, "eip": "0x1a4e2", "steps": 1,
                "view": {"round": round_, "cursor_x": x, "cursor_y": y}, "units": [unit]}

    def run_move(self, spec, modes, positions):
        """modes：wait_mode 依序回傳的介面；positions：state() 依序回傳的單位座標。"""
        # state() 的呼叫次數依路徑不同，最後一個快照重複給到結束（回合已推進）。
        def next_state(seq=iter(positions), last=[None]):
            try:
                last[0] = next(seq)
            except StopIteration:
                pass
            return last[0]
        logged = []
        modes = iter(modes)
        with mock.patch.object(drive, "resume_battle", return_value=True), \
                mock.patch.object(drive, "ensure_cursor_mode", return_value=True), \
                mock.patch.object(drive, "do_goto", return_value=True), \
                mock.patch.object(drive, "send", return_value=(1, self.snapshot(12, 17))), \
                mock.patch.object(drive, "report"), \
                mock.patch.object(drive, "wait_mode", side_effect=lambda wanted, steps, budget=12: next(modes)), \
                mock.patch.object(drive, "stand_by", return_value=self.snapshot(0, 0, round_=4)), \
                mock.patch.object(drive, "log_action", side_effect=lambda kind, current=None, **f: logged.append((kind, f))), \
                mock.patch.object(drive, "state", side_effect=next_state):
            ok = drive.do_move_unit({"move_unit": spec, "steps": 1})
        return ok, logged

    def test_second_candidate_is_taken_when_the_first_is_rejected(self):
        ok, logged = self.run_move(
            {"from_index": 2, "to_any": [[12, 15], [13, 15]]},
            modes=["target", "target", "ring"],
            positions=[self.snapshot(12, 17), self.snapshot(12, 17), self.snapshot(13, 15),
                       self.snapshot(13, 15), self.snapshot(13, 15, round_=4)])
        self.assertTrue(ok)
        kinds = [k for k, _ in logged]
        self.assertEqual(kinds, ["select", "move", "wait"])
        self.assertEqual(logged[1][1]["to"], [13, 15])

    def test_all_candidates_rejected_fails_unless_stay_if_blocked(self):
        ok, logged = self.run_move(
            {"from": [12, 17], "to_any": [[12, 15]]},
            modes=["target", "target"],
            positions=[self.snapshot(12, 17), self.snapshot(12, 17), self.snapshot(12, 17)])
        self.assertFalse(ok)
        ok, logged = self.run_move(
            {"from": [12, 17], "to_any": [[12, 15]], "stay_if_blocked": True},
            modes=["target", "target", "ring"],
            positions=[self.snapshot(12, 17), self.snapshot(12, 17), self.snapshot(12, 17),
                       self.snapshot(12, 17), self.snapshot(12, 17, round_=4)])
        self.assertTrue(ok)
        self.assertEqual([k for k, _ in logged], ["select", "stay", "wait"])


class TowardCandidates(unittest.TestCase):
    """toward：候選格由地圖成本格與敵我位置估可達集合，依「離目標最近」排序。"""

    def grid(self):
        # 6×8，x=3 的 y=2..4 是牆；其餘成本 1。
        w, h = 6, 8
        cost = [1] * (w * h)
        for y in (2, 3, 4):
            cost[y * w + 3] = 99
        return w, h, cost

    def unit(self, index, x, y, camp, hp=100):
        return {"index": index, "camp": camp, "x": x, "y": y, "hp": hp, "byte5": 0}

    def test_enemy_cells_block_and_their_neighbours_zero_the_budget(self):
        current = {"units": [self.unit(0, 2, 7, drive.ALLY_CAMP), self.unit(9, 2, 4, drive.ENEMY_CAMP)]}
        cells = drive.reachable_cells(self.grid(), current, (2, 7), 4)
        self.assertNotIn((2, 4), cells)          # 敵格不可進
        self.assertEqual(cells[(2, 5)], 0)       # 敵格鄰格：進了預算歸零，可停
        self.assertNotIn((2, 3), cells)          # 穿不過去
        self.assertEqual(cells[(1, 5)], 1)       # 繞旁邊走不受影響
        self.assertEqual(cells[(1, 4)], 0)       # 敵格的左鄰：四步到、預算歸零
        self.assertNotIn((1, 3), cells)          # 五步走不到

    def test_ally_cells_are_passable_but_not_destinations(self):
        current = {"units": [self.unit(0, 2, 7, drive.ALLY_CAMP), self.unit(1, 2, 6, drive.ALLY_CAMP),
                             self.unit(9, 5, 0, drive.ENEMY_CAMP)]}
        cells = drive.reachable_cells(self.grid(), current, (2, 7), 3)
        self.assertNotIn((2, 6), cells)
        self.assertEqual(cells[(2, 5)], 1)

    def test_walls_and_goal_distance_order_the_candidates(self):
        current = {"units": [self.unit(0, 4, 7, drive.ALLY_CAMP), self.unit(9, 5, 0, drive.ENEMY_CAMP)]}
        with mock.patch.object(drive, "load_map_cost_grid", return_value=self.grid()):
            ranked = drive.toward_candidates(
                {"toward": [3, 0], "map": 6, "mv": 3, "max_tries": 3}, current, (4, 7))
        self.assertEqual(ranked, [(3, 5), (4, 4), (3, 6)])
        with mock.patch.object(drive, "load_map_cost_grid", return_value=self.grid()):
            staged = drive.toward_candidates(
                {"toward": [4, 5], "map": 6, "mv": 3, "max_tries": 3, "stop_distance": 1}, current, (4, 7))
        self.assertNotIn((4, 5), staged)
        self.assertEqual(staged[0], (4, 6))

    def test_no_candidate_closer_than_here_when_already_on_goal(self):
        current = {"units": [self.unit(0, 4, 7, drive.ALLY_CAMP), self.unit(9, 5, 0, drive.ENEMY_CAMP)]}
        with mock.patch.object(drive, "load_map_cost_grid", return_value=self.grid()):
            self.assertEqual(drive.toward_candidates({"toward": [4, 7], "map": 6, "mv": 3}, current, (4, 7)), [])


class StepInto(unittest.TestCase):
    """step_into：第一個估得到走進指定格的未行動單位踏進去；誰都走不到就不動。"""

    def grid(self):
        w, h = 8, 12
        return w, h, [1] * (w * h)

    def unit(self, index, x, y, camp, mv=4, acted=False):
        raw = bytearray(0x50)
        raw[5] = 0x80 if acted else 0
        raw[0x3b] = mv
        return {"index": index, "camp": camp, "x": x, "y": y, "hp": 100, "byte5": raw[5],
                "raw_hex": raw.hex()}

    def run_step(self, spec, units):
        current = {"input_chain": ["0x1a4e2"], "kbd_pending": 0, "units": units, "view": {"round": 7}}
        moves = []
        with mock.patch.object(drive, "resume_battle", return_value=True), \
                mock.patch.object(drive, "ensure_cursor_mode", return_value=True), \
                mock.patch.object(drive, "load_map_cost_grid", return_value=self.grid()), \
                mock.patch.object(drive, "state", return_value=current), \
                mock.patch.object(drive, "do_move_unit", side_effect=lambda cmd: moves.append(cmd) or True):
            ok = drive.do_step_into({"step_into": spec, "steps": 1})
        return ok, moves

    def test_first_unit_in_reach_steps_in(self):
        units = [self.unit(0, 4, 11, drive.ALLY_CAMP), self.unit(2, 4, 8, drive.ALLY_CAMP, mv=7),
                 self.unit(9, 0, 0, drive.ENEMY_CAMP)]
        ok, moves = self.run_step({"cells": [[4, 2], [3, 2]], "map": 6, "indices": [0, 2]}, units)
        self.assertTrue(ok)
        self.assertEqual(len(moves), 1)
        self.assertEqual(moves[0]["move_unit"]["from_index"], 2)
        self.assertEqual(moves[0]["move_unit"]["to_any"], [[4, 2], [3, 2]])

    def test_acted_units_and_unreachable_cells_are_skipped(self):
        units = [self.unit(2, 4, 8, drive.ALLY_CAMP, mv=7, acted=True), self.unit(0, 4, 11, drive.ALLY_CAMP),
                 self.unit(9, 0, 0, drive.ENEMY_CAMP)]
        ok, moves = self.run_step({"cells": [[4, 2]], "map": 6}, units)
        self.assertTrue(ok)
        self.assertEqual(moves, [])

    def test_unit_already_on_a_cell_is_left_alone(self):
        units = [self.unit(2, 4, 2, drive.ALLY_CAMP, mv=7), self.unit(9, 7, 0, drive.ENEMY_CAMP)]
        ok, moves = self.run_step({"cells": [[4, 2], [3, 2]], "map": 6}, units)
        self.assertTrue(ok)
        self.assertEqual(moves, [])

    def test_enemy_block_and_zoc_are_respected(self):
        # 敵人站在 (4,5)：(4,8) 的單位（MV 7）到 (4,2) 要繞，繞不過鄰格預算歸零。
        units = [self.unit(2, 4, 8, drive.ALLY_CAMP, mv=7), self.unit(9, 4, 5, drive.ENEMY_CAMP)]
        ok, moves = self.run_step({"cells": [[4, 2]], "map": 6}, units)
        self.assertTrue(ok)
        self.assertEqual(moves, [])
        # 敵人在 (7,0)，離路徑遠：走得到。
        units = [self.unit(2, 4, 8, drive.ALLY_CAMP, mv=7), self.unit(9, 7, 0, drive.ENEMY_CAMP)]
        ok, moves = self.run_step({"cells": [[4, 2]], "map": 6}, units)
        self.assertEqual(len(moves), 1)


class DialogueProbe(unittest.TestCase):
    def dialogue(self, pending=0):
        return {"input_chain": ["0x16039"], "kbd_pending": pending,
                "eip": "0x16039", "steps": 1, "view": {}, "units": []}

    def test_probe_uses_fixed_empty_prefix_one_trial_and_fixed_suffix(self):
        sent = []

        def fake_send(key, steps):
            sent.append((key, steps))
            return len(sent), self.dialogue()

        with mock.patch.object(drive, "state", return_value=self.dialogue()), \
                mock.patch.object(drive, "send", side_effect=fake_send), \
                mock.patch.object(drive, "report"):
            with self.assertRaises(drive.DialogueProbeComplete):
                drive.do_dialogue_probe(
                    {"key": "esc", "steps": 123, "before": 2, "after": 3})
        self.assertEqual(sent, [("", 123), ("", 123), ("esc", 123),
                                ("", 123), ("", 123), ("", 123)])

    def test_probe_rejects_non_dialogue_or_pending_keyboard(self):
        for current in ({"input_chain": [], "kbd_pending": 0}, self.dialogue(1)):
            with self.subTest(current=current), \
                    mock.patch.object(drive, "state", return_value=current):
                with self.assertRaises(SystemExit):
                    drive.do_dialogue_probe({"key": "enter", "before": 0})

    def test_probe_chain_filter_distinguishes_event_from_attack_dialogue(self):
        command = {"dialogue_probe": {"chain_contains": ["0x342AB"]}}
        attack = self.dialogue()
        attack["input_chain"] = ["0x16039", "0x1E44E"]
        event = self.dialogue()
        event["input_chain"] = ["0x16CF8", "0x16161", "0x342AB", "0x1A4CC",
                                "0x135CA", "0x1198A", "0x25DD3",
                                "0x45D91", "0x3CB91"]
        with mock.patch.object(drive, "do_dialogue_probe") as probe:
            self.assertFalse(drive.maybe_dialogue_probe(attack, command))
            probe.assert_not_called()
            self.assertTrue(drive.maybe_dialogue_probe(event, command))
            probe.assert_called_once_with(command["dialogue_probe"])

    def test_probe_chain_filter_rejects_invalid_shape(self):
        current = self.dialogue()
        with self.assertRaises(SystemExit):
            drive.maybe_dialogue_probe(
                current, {"dialogue_probe": {"chain_contains": [123]}})


class CampEncoding(unittest.TestCase):
    """0 敵方、1 友軍、2 我方。友軍算進敵方就永遠打不完。"""

    def test_friendly_camp_counts_as_neither_side(self):
        current = {"view": {"round": 1},
                   "units": [unit(1, 1, drive.ALLY_CAMP, identity=0),
                             unit(2, 2, drive.FRIENDLY_CAMP),
                             unit(3, 3, drive.ENEMY_CAMP)]}
        self.assertEqual(len(drive.side(current, drive.ALLY_CAMP)), 1)
        self.assertEqual(len(drive.side(current, drive.ENEMY_CAMP)), 1)
        self.assertEqual(len(drive.side(current, drive.FRIENDLY_CAMP)), 1)

    def test_clearing_the_enemy_camp_ends_the_battle(self):
        """友軍還在場不影響「敵方全滅」。"""
        current = {"view": {"round": 3},
                   "units": [unit(1, 1, drive.ALLY_CAMP, identity=0),
                             unit(2, 2, drive.FRIENDLY_CAMP)]}
        self.assertTrue(drive.holds(current, "enemy_alive<=0")[0])
        self.assertFalse(drive.holds(current, "ally_alive<=0")[0])


class SaveFingerprint(unittest.TestCase):
    """存檔判準：第二關以後「多出一個檔」不再成立，要看內容變了沒。"""

    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.dir, True)
        self.previous = drive.STATE_DIR
        drive.STATE_DIR = self.dir
        self.addCleanup(setattr, drive, "STATE_DIR", self.previous)

    def write(self, name, body):
        with open(os.path.join(self.dir, name), "wb") as handle:
            handle.write(body)

    def test_same_bytes_same_fingerprint(self):
        self.write("FD2.SAV", b"abc")
        first = drive.save_fingerprint()
        self.assertEqual(first, drive.save_fingerprint())

    def test_rewriting_the_save_changes_the_fingerprint(self):
        self.write("FD2.SAV", b"abc")
        before = drive.save_fingerprint()
        self.write("FD2.SAV", b"xyz")
        self.assertNotEqual(before["FD2.SAV"], drive.save_fingerprint()["FD2.SAV"])

    def test_same_size_different_content_is_still_a_change(self):
        """存檔大小固定，只比大小會把「存了新進度」看成沒變。"""
        self.write("FD2.SAV", b"abc")
        before = drive.save_fingerprint()
        self.write("FD2.SAV", b"abd")
        after = drive.save_fingerprint()
        self.assertEqual(before["FD2.SAV"][0], after["FD2.SAV"][0])
        self.assertNotEqual(before["FD2.SAV"], after["FD2.SAV"])

    def test_missing_state_dir_is_empty_not_an_error(self):
        drive.STATE_DIR = os.path.join(self.dir, "nope")
        self.assertEqual(drive.save_fingerprint(), {})

    def test_successful_save_writes_accepts_only_observed_fd2_save_writes(self):
        current = {"dos_file_calls": [
            {"op": "write", "path": r"C:\\FLAME2\\fd2.sav", "handled": True,
             "carry": False, "written_bytes": 22987},
            {"op": "write", "path": "FD2.TMP", "handled": True,
             "carry": False, "written_bytes": 22987},
            {"op": "write", "path": "FD2.SAV", "handled": True,
             "carry": True, "written_bytes": 0},
            {"op": "open", "path": "FD2.SAV", "handled": True,
             "carry": False},
        ]}
        self.assertEqual(drive.successful_save_writes(current), 1)

    def test_town_save_accepts_identical_bytes_only_with_new_dos_write(self):
        self.write("FD2.SAV", b"same")
        calls = [0]

        def snapshot():
            events = [{"op": "write", "path": "FD2.SAV", "handled": True,
                       "carry": False, "written_bytes": 4}] * calls[0]
            return {"dos_file_calls": events, "input_chain": ["0x2CE08"],
                    "eip": "0x10000", "steps": 1, "view": {}, "units": []}

        def fake_send(key, steps):
            if key == "enter" and calls[0] == 0:
                calls[0] = 1
            return calls[0], snapshot()

        with mock.patch.object(drive, "state", side_effect=snapshot), \
                mock.patch.object(drive, "send", side_effect=fake_send), \
                mock.patch.object(drive, "settle", side_effect=lambda *_: snapshot()), \
                mock.patch.object(drive, "report"):
            self.assertTrue(drive.do_town_save({"town_save": True}))

    def test_town_save_fails_when_identical_bytes_have_no_write_receipt(self):
        self.write("FD2.SAV", b"same")
        snapshot = {"dos_file_calls": [], "input_chain": ["0x2CE08"],
                    "eip": "0x10000", "steps": 1, "view": {}, "units": []}
        with mock.patch.object(drive, "state", return_value=snapshot), \
                mock.patch.object(drive, "send", return_value=(1, snapshot)), \
                mock.patch.object(drive, "settle", return_value=snapshot), \
                mock.patch.object(drive, "report"):
            self.assertFalse(drive.do_town_save({"town_save": True}))

    def test_town_save_moves_to_the_explicit_slot_before_confirming(self):
        self.write("FD2.SAV", b"same")
        keys = []
        writes = [0]

        def snapshot():
            events = [{"op": "write", "path": "FD2.SAV", "handled": True,
                       "carry": False, "written_bytes": 4}] * writes[0]
            return {"dos_file_calls": events, "input_chain": ["0x2CE08"],
                    "eip": "0x10000", "steps": 1, "view": {}, "units": []}

        def fake_send(key, steps):
            keys.append(key)
            if keys == ["enter", "right", "enter", "down", "down", "enter"]:
                writes[0] = 1
            return len(keys), snapshot()

        with mock.patch.object(drive, "state", side_effect=snapshot), \
                mock.patch.object(drive, "send", side_effect=fake_send), \
                mock.patch.object(drive, "settle", side_effect=lambda *_: snapshot()), \
                mock.patch.object(drive, "report"):
            self.assertTrue(drive.do_town_save({"town_save": True, "slot": 2}))
        self.assertEqual(keys[:6],
                         ["enter", "right", "enter", "down", "down", "enter"])


class UnitIdentityKey(unittest.TestCase):
    """一輪之內辨識同一個單位的鍵。"""

    def test_identity_survives_the_unit_moving(self):
        """推進過的單位座標會變；用座標記「處理過誰」等於沒記。"""
        before = unit(20, 14, drive.ALLY_CAMP, identity=9)
        after = unit(14, 16, drive.ALLY_CAMP, identity=9)
        self.assertEqual(drive.unit_key(before), drive.unit_key(after))

    def test_different_units_do_not_collide(self):
        a = unit(20, 14, drive.ALLY_CAMP, identity=9)
        b = unit(20, 14, drive.ALLY_CAMP, identity=30)
        self.assertNotEqual(drive.unit_key(a), drive.unit_key(b))

    def test_missing_identity_falls_back_to_the_cell(self):
        """沒有 identity 時退回座標，至少擋得住原地沒動又被選中。"""
        a = unit(20, 14, drive.ALLY_CAMP)
        self.assertEqual(drive.unit_key(a), ("cell", 20, 14))
        self.assertNotEqual(drive.unit_key(a),
                            drive.unit_key(unit(21, 14, drive.ALLY_CAMP)))


class ApproachWhenNothingIsInRange(unittest.TestCase):
    """兩軍隔著半張地圖時，候選落腳格仍要沿路逼近。"""

    def setUp(self):
        self.current = {"view": {"round": 1},
                        "units": [unit(20, 14, drive.ALLY_CAMP, identity=0),
                                  unit(6, 14, drive.ENEMY_CAMP)]}

    def test_targets_advance_toward_the_nearest_enemy(self):
        cells = drive.engage_targets(self.current, (20, 14), typical_move=6)
        self.assertTrue(cells, "十四格外的敵人也要產生逼近格")
        for cell in cells:
            self.assertLess(drive.distance(cell, (6, 14)),
                            drive.distance((20, 14), (6, 14)),
                            f"{cell} 沒有比原地更靠近敵人")
            self.assertLessEqual(drive.distance(cell, (20, 14)), 6,
                                 f"{cell} 超出一般移動力")

    def test_candidates_cover_several_distances(self):
        """走多遠才走得到是未知的，候選格不能全押在同一個距離上。"""
        cells = drive.engage_targets(self.current, (20, 14), typical_move=6)
        spans = {drive.distance(c, (20, 14)) for c in cells[:12]}
        self.assertGreaterEqual(len(spans), 4, f"只涵蓋了 {sorted(spans)}")
        self.assertEqual(drive.distance(cells[0], (20, 14)), 6,
                         "先試最遠的一桶")

    def test_hint_puts_the_known_distance_first(self):
        """學到「這個單位走得了四格」之後就從四格開始試，不再從上限往下掃。"""
        cells = drive.engage_targets(self.current, (20, 14),
                                     typical_move=6, hint=4)
        self.assertEqual(drive.distance(cells[0], (20, 14)), 4)
        spans = [drive.distance(c, (20, 14)) for c in cells[:6]]
        self.assertEqual(spans[:2], [4, 4], spans)
        self.assertIn(spans[2], (3, 5), f"第二順位要是相鄰距離，實際 {spans}")

    def test_candidates_skip_occupied_cells(self):
        blocked = dict(self.current)
        blocked["units"] = self.current["units"] + [
            unit(14, 14, drive.ALLY_CAMP, identity=9)]
        cells = drive.engage_targets(blocked, (20, 14), typical_move=6)
        self.assertNotIn((14, 14), cells)


if __name__ == "__main__":
    unittest.main()


class ActionLog(unittest.TestCase):
    """語意動作紀錄：重製端重播讀的是這份，不是方向鍵次數。"""

    def setUp(self):
        self.tmp = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.tmp, ignore_errors=True)
        self.old_run = drive.RUN
        drive.RUN = self.tmp
        self.addCleanup(setattr, drive, "RUN", self.old_run)

    def test_records_seq_round_rng_and_fields(self):
        current = {"control_seq": 42, "view": {"round": 3, "rng_word": 0x1234, "gold": 2000}}
        made = drive.log_action("attack", current, frm=[8, 17], target=[8, 16])
        self.assertEqual(made["seq"], 42)
        self.assertEqual(made["round"], 3)
        self.assertEqual(made["rng_word"], 0x1234)
        self.assertEqual(made["gold"], 2000)
        with open(os.path.join(self.tmp, "actions.jsonl"), encoding="utf-8") as handle:
            lines = [json.loads(l) for l in handle]
        self.assertEqual(len(lines), 1)
        self.assertEqual(lines[0]["kind"], "attack")
        self.assertEqual(lines[0]["target"], [8, 16])

    def test_appends_in_order(self):
        current = {"control_seq": 1, "view": {}}
        drive.log_action("select", current, at=[1, 1])
        drive.log_action("end_turn", dict(current, control_seq=2))
        with open(os.path.join(self.tmp, "actions.jsonl"), encoding="utf-8") as handle:
            kinds = [json.loads(l)["kind"] for l in handle]
        self.assertEqual(kinds, ["select", "end_turn"])
