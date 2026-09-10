#!/usr/bin/env python3
"""dosgolem_oracle_drive.py 純狀態邏輯的測試。

閉環驅動的決策全在這幾支純函式裡：誰還沒行動、往哪一格走、條件成立了沒。
它們錯了不會噴錯，只會安靜地把按鍵送到錯的地方，最後產生一份「跑完了但走錯路」
的收據。所以逐條驗兩個方向，別只驗好路徑。

執行：python3 -m unittest discover -s tools -p 'test_dosgolem_oracle_drive.py'
"""

import importlib.util
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parent


def load():
    spec = importlib.util.spec_from_file_location(
        "fd2_oracle_drive", ROOT / "dosgolem_oracle_drive.py")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


drive = load()


def unit(x, y, camp, hp=10, acted=False):
    raw = ["00"] * 80
    raw[5] = "80" if acted else "00"
    return {"x": x, "y": y, "camp": camp, "hp": hp, "raw_hex": "".join(raw)}


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


class StepPath(unittest.TestCase):
    def test_path_walks_x_then_y_and_ends_on_target(self):
        self.assertEqual(drive.step_path((7, 14), (5, 16)),
                         [(6, 14), (5, 14), (5, 15), (5, 16)])

    def test_path_is_empty_when_already_there(self):
        self.assertEqual(drive.step_path((3, 3), (3, 3)), [])

    def test_每一步只動一格(self):
        path = drive.step_path((0, 0), (4, 3))
        previous = (0, 0)
        for cell in path:
            self.assertEqual(drive.distance(previous, cell), 1, path)
            previous = cell
        self.assertEqual(path[-1], (4, 3))


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

    def test_unknown_variable_and_syntax_both_stop_the_run(self):
        with self.assertRaises(SystemExit):
            drive.holds(self.current, "morale>=1")
        with self.assertRaises(SystemExit):
            drive.holds(self.current, "round ~ 3")
        with self.assertRaises(SystemExit):
            drive.measure(self.current, "morale")


if __name__ == "__main__":
    unittest.main()
