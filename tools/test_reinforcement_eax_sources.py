#!/usr/bin/env python3
"""ch21/ch22 動態增援 IDA 主證據的 schema 與 6/6 coverage 回歸。"""

import json
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
EVIDENCE = ROOT / "docs/data/ida/fd2_reinforcement_eax_sources.json"
TURN_EVENTS = ROOT / "docs/data/turn_events.json"
EXPECTED_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"


class ReinforcementEAXSourcesTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.report = json.loads(EVIDENCE.read_text(encoding="utf-8"))
        cls.turn_events = json.loads(TURN_EVENTS.read_text(encoding="utf-8"))

    def test_fixed_input_and_ida_contract(self):
        self.assertEqual(self.report["schema_version"], 1)
        self.assertEqual(self.report["status"], "RE-CLOSED")
        self.assertEqual(self.report["tool"]["name"], "IDA Pro")
        self.assertEqual(self.report["tool"]["version"], "9.4")
        self.assertEqual(
            self.report["tool"]["address_space"],
            "IDA LE flat-loader linear address",
        )
        self.assertEqual(self.report["input"]["size"], 357074)
        self.assertEqual(self.report["input"]["sha256"], EXPECTED_SHA256)

    def test_six_cases_match_versioned_turn_events(self):
        cases = self.report["cases"]
        self.assertEqual(self.report["coverage"], {
            "expected": 6,
            "closed": 6,
            "unresolved": 0,
        })
        self.assertEqual(len(cases), 6)
        self.assertEqual(len({case["id"] for case in cases}), 6)

        reported = {
            (
                case["chapter"],
                case["map"],
                case["turn"],
                case["event_id"],
                case["computed_group"],
            )
            for case in cases
        }
        scheduled = {
            (
                chapter["chapter"],
                chapter["map"],
                event["turn"],
                event["event_id"],
                event["groups"][0],
            )
            for chapter in self.turn_events
            for event in chapter["turn_events"]
            if event["event_id"] in (47, 49)
        }
        self.assertEqual(reported, scheduled)
        for case in cases:
            self.assertEqual(case["computed_group"], int(case["turn"] / 2))
            self.assertEqual(case["context_level"], "攻略旁證")
            self.assertEqual(case["result"], "RE-CLOSED")

    def test_jump_table_and_eax_flows_preserve_raw_evidence(self):
        expected = {
            "47": {
                "slot": "0x51c4d",
                "slot_bytes": "12 51 03 00",
                "target": "0x35112",
                "addresses": [
                    "0x3511c", "0x35121", "0x35123", "0x35126",
                    "0x35128", "0x3512a", "0x3512b",
                ],
            },
            "49": {
                "slot": "0x51c55",
                "slot_bytes": "e9 51 03 00",
                "target": "0x351e9",
                "addresses": [
                    "0x351f3", "0x351f8", "0x351fa", "0x351fd",
                    "0x351ff", "0x35201", "0x35202",
                ],
            },
        }
        self.assertEqual(set(self.report["handlers"]), set(expected))
        for event_id, wanted in expected.items():
            handler = self.report["handlers"][event_id]
            slot = handler["jump_table_slot"]
            self.assertEqual(slot["address"], wanted["slot"])
            self.assertEqual(slot["bytes"], wanted["slot_bytes"])
            self.assertEqual(slot["decoded_target"], wanted["target"])
            flow = handler["eax_source_flow"]
            self.assertEqual([row["address"] for row in flow], wanted["addresses"])
            self.assertEqual(
                [row["mnemonic"] for row in flow],
                ["mov", "mov", "sar", "sub", "sar", "push", "call"],
            )
            self.assertTrue(all(row["bytes"] for row in flow))
            self.assertEqual(handler["formula"]["source_global"], "0x53bef")
            self.assertEqual(handler["formula"]["consumer"], "0x10b4e")
            self.assertEqual(handler["formula"]["level"], "已證實")

    def test_indirect_dispatch_and_round_counter_writers_are_explicit(self):
        calls = self.report["dispatcher"]["indirect_calls"]
        self.assertEqual(len(calls), 1)
        self.assertEqual(calls[0]["address"], "0x1a85a")
        self.assertEqual(calls[0]["bytes"], "ff 14 85 91 1b 05 00")
        self.assertIn("eax*4", calls[0]["original"])

        xrefs = {
            row["address"]: row
            for row in self.report["round_counter"]["data_xrefs"]
        }
        for address in ("0x103df", "0x1a5b9", "0x2066e", "0x3511c", "0x351f3"):
            self.assertIn(address, xrefs)
            self.assertTrue(xrefs[address]["bytes"])
        self.assertIn("dword_53BEF, 1", xrefs["0x2066e"]["original"])

    def test_spawn_consumer_filters_raw_group_before_constructor(self):
        function = self.report["spawn_group_consumer"]["function"]
        self.assertEqual(function["start"], "0x10b4e")
        self.assertEqual(function["end"], "0x10c50")
        instructions = {row["address"]: row for row in function["instructions"]}
        self.assertEqual(
            instructions["0x10be4"]["original"],
            "movzx   eax, byte ptr [eax+15h]",
        )
        self.assertEqual(instructions["0x10be8"]["original"], "cmp     eax, edi")
        self.assertEqual(instructions["0x10bee"]["original"], "call    sub_10C50")


if __name__ == "__main__":
    unittest.main()
