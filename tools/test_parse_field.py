#!/usr/bin/env python3
"""FDFIELD 三位元組後處理列的相依性零回歸測試。"""

import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))
import parse_field


class NativePostResolutionSourceTest(unittest.TestCase):
    def test_map_selector_uses_fdf_field_b1_not_camp_b0(self):
        record = bytearray(26)
        record[0] = 2
        record[1] = 0x60
        parsed = parse_field.native_map_selector_key(record)
        self.assertEqual(parsed, 0x60)

    def test_treasure_types_do_not_turn_events_into_items(self):
        self.assertEqual(parse_field.native_reward_kind(0), "item")
        self.assertEqual(parse_field.native_reward_kind(1), "gold")
        self.assertEqual(parse_field.native_reward_kind(2), "event")
        self.assertEqual(parse_field.native_reward_kind(0xFE), "event")

    def test_death_effect_is_type_plus_u16_and_excludes_b25(self):
        record = bytearray(26)
        record[22:26] = bytes((2, 0x34, 0x12, 0xAB))
        self.assertEqual(
            parse_field.native_death_effect(record),
            {"type": 2, "value": 0x1234},
        )

    def test_ff_death_effect_is_absent(self):
        record = bytearray(26)
        record[22] = 0xFF
        self.assertIsNone(parse_field.native_death_effect(record))

    def test_dormant_turn_rows_are_preserved_but_not_enabled(self):
        control = bytearray(3 + 16 * 3)
        for slot in range(16):
            control[3 + slot * 3:6 + slot * 3] = bytes((0xFF, 0xFF, 0xFF))
        control[3:9] = bytes((0xFF, 63, 0, 4, 12, 2))
        rows = parse_field.native_turn_event_controls(control)
        self.assertEqual(rows[0], {"turn": 0xFF, "event_id": 63, "raw_camp": 0})
        self.assertEqual(rows[1], {"turn": 4, "event_id": 12, "raw_camp": 2})
        self.assertEqual(
            parse_field.enabled_turn_events(rows),
            [{"turn": 4, "event_id": 12, "camp": "special"}],
        )


if __name__ == "__main__":
    unittest.main()
