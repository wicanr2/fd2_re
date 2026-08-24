#!/usr/bin/env python3
"""FIGANI 工具的最小格式回歸。"""

import struct
import unittest

import decode_figani


class DecodeFiganiTest(unittest.TestCase):
    def test_parse_anim_keeps_byte_header_and_raw_frame_markers_separate(self):
        data = bytearray(8 + 4 + 13 + 2)
        data[0:5] = bytes((1, 7, 8, 0, 9))
        struct.pack_into("<I", data, 8, 12)
        struct.pack_into("<hh", data, 12, -3, 4)
        data[16:20] = bytes((1, 2, 6, 7))
        struct.pack_into("<HH", data, 21, 1, 1)
        data[25:27] = bytes((0, 5))

        frames = decode_figani.parse_anim(bytes(data))

        self.assertEqual(frames, [(-3, 4, 1, 1, 1, 2, 6, 7, bytes((0, 5)))])

    def test_direct_indexed_frame_may_have_zero_delay(self):
        header = bytearray(12)
        header[0] = 1
        struct.pack_into("<I", header, 8, 12)
        frame = struct.pack("<hhBBBBBHH", 3, 4, 0, 0, 0, 0, 0, 1, 1) + b"\x00\x07"

        frames = decode_figani.parse_anim(bytes(header) + frame)

        self.assertEqual(len(frames), 1)
        self.assertEqual(frames[0][0:8], (3, 4, 1, 1, 0, 0, 0, 0))


if __name__ == "__main__":
    unittest.main()
