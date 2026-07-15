#!/usr/bin/env python3
import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))
import dump_chapter_beats as beats


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


if __name__ == "__main__":
    unittest.main()
