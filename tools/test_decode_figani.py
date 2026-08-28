#!/usr/bin/env python3
"""FIGANI 工具的最小格式回歸。"""

import struct
import tempfile
import unittest
from pathlib import Path
import json

import decode_figani


class DecodeFiganiTest(unittest.TestCase):
    def test_mask_distinguishes_opaque_palette_zero_from_transparent_skip(self):
        pixels, mask = decode_figani.decode_rle_layers(
            bytes([0x80, 0x00, 0xC0, 0x40, 0x00]), 4, 1
        )
        self.assertEqual(pixels, bytes([0, 0, 0, 0]))
        self.assertEqual(mask, bytes([255, 0, 0, 255]))

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

    def test_frames_export_preserves_alpha_mask_and_metadata(self):
        from PIL import Image

        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / "FIGANI_004.bin"
            palette = root / "palette.bin"
            output = root / "out"
            header = bytearray(12)
            header[0:5] = bytes([1, 0, 1, 0, 7])
            struct.pack_into("<I", header, 8, 12)
            frame = struct.pack("<hhBBBBBHH", 3, 4, 1, 2, 6, 7, 0, 2, 1)
            source.write_bytes(bytes(header) + frame + bytes([0x80, 0x00, 0xC0]))
            palette.write_bytes(bytes(768))

            decode_figani.cmd_frames(str(source), str(palette), str(output))

            document = json.loads((output / "animation.json").read_text(encoding="utf-8"))
            self.assertEqual(document["native_header"], {"byte_1": 0, "byte_2": 1, "byte_4": 7})
            self.assertEqual(document["frames"][0]["delay_native"], 6)
            self.assertEqual(document["frames"][0]["raw_byte_5"], 2)
            indexed = Image.open(output / "FIGANI_004_f00.png")
            self.assertEqual(indexed.mode, "P")
            self.assertEqual(list(indexed.getdata()), [0, 0])
            mask = Image.open(output / "FIGANI_004_f00_mask.png")
            self.assertEqual(list(mask.getdata()), [255, 0])


if __name__ == "__main__":
    unittest.main()
