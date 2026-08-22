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

    def test_parse_anim_rejects_zero_delay(self):
        data = bytearray(8 + 4 + 13)
        data[0] = 1
        struct.pack_into("<I", data, 8, 12)
        struct.pack_into("<HH", data, 21, 1, 1)

        self.assertEqual(decode_figani.parse_anim(bytes(data)), [])


if __name__ == "__main__":
    unittest.main()
