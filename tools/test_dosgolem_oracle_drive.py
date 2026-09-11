#!/usr/bin/env python3
"""dosgolem_oracle_drive.py 純狀態邏輯的測試。

閉環驅動的決策全在這幾支純函式裡：誰還沒行動、往哪一格走、條件成立了沒。
它們錯了不會噴錯，只會安靜地把按鍵送到錯的地方，最後產生一份「跑完了但走錯路」
的收據。所以逐條驗兩個方向，別只驗好路徑。

執行：python3 -m unittest discover -s tools -p 'test_dosgolem_oracle_drive.py'
"""

import importlib.util
import os
import pathlib
import shutil
import tempfile
import unittest

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
