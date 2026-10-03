#!/usr/bin/env python3
"""全域事件轉寫的固定原版指令回歸；在fd2-cap-local內執行。"""

import copy
import os
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import extract_native_death_events as extractor


class NativeDeathExtractorTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.image = extractor.Image(Path(os.environ.get("FD2_EXE", "/orig/FD2.EXE")))
        cls.events = {event["id"]: event for event in extractor.EVENTS}

    def test_ch15_all_three_handlers_match_bytes_and_cover_control_flow(self):
        for event_id in (13, 18, 38):
            with self.subTest(event=event_id):
                event = self.events[event_id]
                for op in event["ops"]:
                    extractor.check_op(self.image, event_id, op)
                extractor.check_coverage(self.image, event)

    def test_ch22_event50_covers_spawn_pan_tick_join_and_shared_dialogue(self):
        event = self.events[50]
        for action in event["ops"]:
            extractor.check_op(self.image, 50, action)
        extractor.check_coverage(self.image, event)

    def test_ch22_event50_rejects_wrong_ticks_join_and_missing_tail(self):
        for index, field, value in [(2, "ticks", 7), (3, "char_id", 21), (4, "text", 1)]:
            action = copy.deepcopy(self.events[50]["ops"][index])
            action[field] = value
            with self.assertRaises(SystemExit):
                extractor.check_op(self.image, 50, action)
        event = copy.deepcopy(self.events[50])
        event["ops"] = event["ops"][:-1]
        with self.assertRaises(SystemExit):
            extractor.check_coverage(self.image, event)

    def test_record_loop_rejects_slot_outside_original_inclusive_range(self):
        op = copy.deepcopy(self.events[13]["ops"][1])
        op["unit"] = 74
        with self.assertRaises(SystemExit):
            extractor.check_op(self.image, 13, op)

    def test_record_loop_rejects_different_write_value(self):
        op = copy.deepcopy(self.events[13]["ops"][1])
        op["writes"] = [[0x35, 1, 1]]
        with self.assertRaises(SystemExit):
            extractor.check_op(self.image, 13, op)

    def test_record_loop_rejects_wrong_inclusive_endpoint(self):
        op = copy.deepcopy(self.events[13]["ops"][1])
        op["record_loop"] = (64, 74, 0x34EC9, 0x34EDF)
        with self.assertRaises(SystemExit):
            extractor.check_op(self.image, 13, op)

    def test_event13_missing_record_loop_is_not_complete(self):
        event = copy.deepcopy(self.events[13])
        event["ops"] = [op for op in event["ops"] if op["op"] != "record_bytes"]
        with self.assertRaises(SystemExit):
            extractor.check_coverage(self.image, event)

    def test_event38_spawn_only_is_not_complete(self):
        event = copy.deepcopy(self.events[38])
        event["ops"] = [event["ops"][0]]
        with self.assertRaises(SystemExit):
            extractor.check_coverage(self.image, event)

    def test_event46_matches_original_and_includes_join(self):
        event = self.events[46]
        for op in event["ops"]:
            extractor.check_op(self.image, 46, op)
        extractor.check_coverage(self.image, event)
        incomplete = copy.deepcopy(event)
        incomplete["ops"] = incomplete["ops"][:-1]
        with self.assertRaises(SystemExit):
            extractor.check_coverage(self.image, incomplete)

    def test_event46_rejects_different_join_identity(self):
        op = copy.deepcopy(self.events[46]["ops"][-1])
        op["char_id"] = 26
        with self.assertRaises(SystemExit):
            extractor.check_op(self.image, 46, op)


if __name__ == "__main__":
    unittest.main()
