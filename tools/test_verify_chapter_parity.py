import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import verify_chapter_parity as vp  # noqa: E402


class PairingAndUnits(unittest.TestCase):
    def test_after_enemy_phase_pairs_with_the_next_action(self):
        actions = [{"kind": "end_turn", "seq": 50}, {"kind": "select", "seq": 80}, {"kind": "move", "seq": 95}]
        self.assertEqual(vp.pair_oracle_seq(actions, {"kind": "after_enemy_phase", "oracle_seq": 50}), 80)
        self.assertEqual(vp.pair_oracle_seq(actions, {"kind": "move", "oracle_seq": 95}), 95)
        self.assertIsNone(vp.pair_oracle_seq(actions, {"kind": "after_enemy_phase", "oracle_seq": 95}))
        self.assertIsNone(vp.pair_oracle_seq(actions, {"kind": "town_loaded", "oracle_seq": 0}))

    def test_dead_oracle_units_are_dropped(self):
        cp = {"units": [{"camp": 0, "x": 1, "y": 2, "hp": 0}, {"camp": 2, "x": 3, "y": 4, "hp": 9}]}
        self.assertEqual(vp.oracle_units(cp), {(2, 3, 4)})
        self.assertEqual(vp.oracle_hp(cp), {(2, 3, 4): 9})

    def test_shop_menu_compares_gold_before_the_sale(self):
        actions = [{"kind": "shop_sell", "seq": 1097, "gold_before": 2000, "gold_after": 2037}]
        self.assertEqual(vp.oracle_gold_for(actions, 1097, "shop_menu", {"gold": 2037}), 2000)
        self.assertEqual(vp.oracle_gold_for(actions, 1097, "shop_sell", {"gold": 2037}), 2037)
        self.assertEqual(vp.oracle_gold_for([], 1097, "shop_menu", {"gold": 2037}), 2037)

    def test_save_gate_compares_whole_file_hashes(self):
        h = "a" * 64
        acts = [{"kind": "town_save", "save_sha256": h}]
        self.assertEqual(vp.save_gate_entry(acts, [{"note": f"save_sha256={h}"}], "")["status"], "ok")
        self.assertEqual(vp.save_gate_entry(acts, [{"note": "save_sha256=" + "b" * 64}], "")["status"], "fail")
        self.assertEqual(vp.save_gate_entry(acts, [{"note": "原版槽寫回失敗：x"}], "24")["status"], "blocked")
        self.assertEqual(vp.save_gate_entry(acts, [{"note": "原版槽寫回失敗：x"}], "")["status"], "fail")
        self.assertEqual(vp.save_gate_entry([], [], "")["status"], "not_sampled")

    def test_remake_units_keep_camp_code(self):
        cp = {"units": [{"camp": 2, "x": 3, "y": 4, "hp": 9, "identity": 0, "acted": 0}]}
        self.assertEqual(vp.remake_units(cp), {(2, 3, 4)})


if __name__ == "__main__":
    unittest.main()
