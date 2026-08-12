#!/usr/bin/env python3
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))
import dump_chapter_beats as beats
import export_handler_scripts as handler_scripts


class Insn:
    def __init__(self, address, mnemonic, op_str="", size=1):
        self.address = address
        self.mnemonic = mnemonic
        self.op_str = op_str
        self.size = size


class FakeCG:
    def __init__(self, instructions):
        self.instructions = {ins.address: ins for ins in instructions}

    def _insn(self, address):
        return self.instructions.get(address)


class DumpRangeTest(unittest.TestCase):
    def test_follows_shared_tail_beyond_next_handler_without_fallthrough(self):
        cg = FakeCG([
            Insn(0x100, "push", "1"),
            Insn(0x101, "jmp", "0x200"),
            Insn(0x110, "call", "0xdead"),  # next handler: never fall through
            Insn(0x200, "push", "3"),
            Insn(0x201, "call", "0x15f84"),
            Insn(0x202, "ret"),
        ])
        got = beats.dump_range(cg, 0x100, 0x110, 0x300)
        self.assertEqual([ins.address for ins in got], [0x100, 0x101, 0x200, 0x201, 0x202])

    def test_backward_shared_tail_is_appended_after_local_body(self):
        cg = FakeCG([
            Insn(0x300, "push", "3"),
            Insn(0x301, "jmp", "0x200"),
            Insn(0x200, "push", "0"),
            Insn(0x201, "call", "0x12d7b"),
            Insn(0x202, "ret"),
        ])
        got = beats.dump_range(cg, 0x300, 0x310, 0x400)
        self.assertEqual([ins.address for ins in got], [0x300, 0x301, 0x200, 0x201, 0x202])


class StructureControlFlowTest(unittest.TestCase):
    def test_spawn_exports_exact_call_site_raw_placement_gate(self):
        insns = [
            Insn(0x32E4F, "push", "2"),
            Insn(0x32E50, "call", "0x10b4e"),
            Insn(0x32E60, "push", "3"),
            Insn(0x32E61, "call", "0x10b4e"),
            Insn(0x32E70, "push", "1"),
            Insn(0x32E71, "call", "0x32999"),
        ]
        raw_beats = beats.extract_beats(insns)
        self.assertEqual(
            [beat["raw_placement_gate"] for beat in raw_beats],
            [1, 0, 0],
        )
        authored = handler_scripts.normalize(raw_beats)
        self.assertEqual(
            [beat["raw_placement_gate"] for beat in authored],
            [1, 0, 0],
        )

    def test_10652_is_not_exported_as_a_complete_chapter_background_load(self):
        self.assertEqual(
            beats.PRIM[0x10652],
            ("prepare_chapter_aux_graphics", 0),
        )

    def test_classified_native_call_keeps_narrow_label_and_push_order(self):
        insns = [
            Insn(0x200, "push", "0"),
            Insn(0x201, "push", "0xff"),
            Insn(0x202, "push", "0x3f"),
            Insn(0x203, "call", "0x11df2"),
        ]
        raw_beats = beats.extract_beats(insns)
        self.assertEqual(raw_beats[0]["op"], "native_palette_update")
        self.assertEqual(raw_beats[0]["args"], [0, 0xFF, 0x3F])
        authored = handler_scripts.normalize(raw_beats)
        self.assertEqual(authored, [{
            "op": "native_call",
            "native_semantic": "native_palette_update",
            "native_confidence": "已證實",
            "native_evidence": ["docs/data/fd2_11df2_palette_disasm.txt"],
            "native_target": "0x11df2",
            "raw_args": [0, 0xFF, 0x3F],
            "source": {"addr": "0x203", "target": "0x11df2"},
        }])

    def test_unclassified_high_risk_native_target_remains_unknown(self):
        raw_beats = beats.extract_beats([Insn(0x300, "call", "0x22253")])
        self.assertEqual(raw_beats[0]["op"], "unknown")
        authored = handler_scripts.normalize(raw_beats)[0]
        self.assertEqual(authored["op"], "unresolved_native_call")
        self.assertEqual(authored["native_confidence"], "強推論")

    def test_single_slot_unit_inactive_diamond_keeps_shared_merge_once(self):
        # Deliberately unrelated synthetic addresses: regression proves that the
        # recognizer is based on the original instruction shape, not ch02_post.
        insns = [
            Insn(0x100, "call", "0x11506"),
            Insn(0x101, "push", "6"),
            Insn(0x102, "call", "0x3453e"),
            Insn(0x103, "add", "esp, 4"),
            Insn(0x104, "test", "eax, eax"),
            Insn(0x105, "je", "0x120"),
            Insn(0x106, "push", "6"),
            Insn(0x107, "push", "dword ptr [0x3a79]"),
            Insn(0x108, "call", "0x15f84"),
            Insn(0x109, "jmp", "0x140"),
            Insn(0x120, "call", "0x233c6"),
            Insn(0x121, "push", "7"),
            Insn(0x122, "push", "dword ptr [0x3a79]"),
            Insn(0x123, "call", "0x15f84"),
            Insn(0x124, "push", "2"),
            Insn(0x125, "call", "0x112a5"),
            Insn(0x126, "jmp", "0x140"),
            Insn(0x140, "inc", "dword ptr [0x3c03]"),
            Insn(0x141, "ret"),
        ]

        got = beats.structure_control_flow(insns, beats.extract_beats(insns))

        self.assertEqual([beat["op"] for beat in got],
                         ["sync_party", "if", "increment_chapter"])
        conditional = got[1]
        self.assertEqual(conditional["condition"], {
            "op": "any_unit_inactive",
            "unit_slots": [6],
        })
        self.assertEqual([beat["op"] for beat in conditional["then"]], ["dialog"])
        self.assertEqual(conditional["then"][0]["args"], ["dword ptr [0x3a79]", 6])
        self.assertEqual([beat["op"] for beat in conditional["else"]],
                         ["layout_units", "dialog", "join"])
        self.assertEqual(conditional["else"][1]["args"], ["dword ptr [0x3a79]", 7])
        self.assertEqual(conditional["else"][2]["args"], [2])
        self.assertEqual(sum(beat["op"] == "increment_chapter" for beat in got), 1)


if __name__ == "__main__":
    unittest.main()
