#!/usr/bin/env python3

import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

from PIL import Image

from process_modern_map_sprite_sheet import transparent_background


class TransparentBackgroundTest(unittest.TestCase):
    def test_chroma_removes_enclosed_background_holes(self) -> None:
        image = Image.new("RGB", (7, 7), (10, 245, 12))
        for y in range(1, 6):
            for x in range(1, 6):
                image.putpixel((x, y), (20, 30, 40))
        image.putpixel((3, 3), (10, 245, 12))

        result = transparent_background(image, (0, 255, 0), 32)

        self.assertEqual(result.getpixel((0, 0))[3], 0)
        self.assertEqual(result.getpixel((3, 3))[3], 0)
        self.assertEqual(result.getpixel((2, 2)), (20, 30, 40, 255))

    def test_chroma_tolerance_keeps_distant_teal(self) -> None:
        image = Image.new("RGB", (3, 3), (30, 230, 15))
        image.putpixel((1, 1), (0, 90, 100))

        result = transparent_background(image, (0, 255, 0), 80)

        self.assertEqual(result.getpixel((0, 0))[3], 0)
        self.assertEqual(result.getpixel((1, 1)), (0, 90, 100, 255))

    def test_chroma_spill_removes_dominant_green_but_keeps_gray_green(self) -> None:
        image = Image.new("RGB", (5, 1), (24, 64, 9))
        image.putpixel((1, 0), (50, 68, 42))
        image.putpixel((2, 0), (20, 30, 40))
        image.putpixel((3, 0), (34, 52, 1))
        image.putpixel((4, 0), (0, 38, 14))

        result = transparent_background(image, (0, 255, 0), 120)

        self.assertEqual(result.getpixel((0, 0))[3], 0)
        self.assertEqual(result.getpixel((1, 0)), (50, 68, 42, 255))
        self.assertEqual(result.getpixel((2, 0)), (20, 30, 40, 255))
        self.assertEqual(result.getpixel((3, 0))[3], 0)
        self.assertEqual(result.getpixel((4, 0))[3], 0)

    def test_existing_alpha_is_preserved_as_binary_mask(self) -> None:
        image = Image.new("RGBA", (2, 1), (1, 2, 3, 0))
        image.putpixel((1, 0), (4, 5, 6, 128))

        result = transparent_background(image, (0, 255, 0), 80)

        self.assertEqual(result.getpixel((0, 0)), (0, 0, 0, 0))
        self.assertEqual(result.getpixel((1, 0)), (4, 5, 6, 255))

    def test_existing_neutral_background_behavior_remains(self) -> None:
        image = Image.new("RGB", (3, 3), (255, 255, 255))
        image.putpixel((1, 1), (255, 255, 255))
        image.putpixel((1, 0), (0, 0, 0))
        image.putpixel((0, 1), (0, 0, 0))
        image.putpixel((2, 1), (0, 0, 0))
        image.putpixel((1, 2), (0, 0, 0))

        result = transparent_background(image)

        self.assertEqual(result.getpixel((0, 0))[3], 0)
        self.assertEqual(result.getpixel((1, 1)), (255, 255, 255, 255))


class ProcessSheetCLITest(unittest.TestCase):
    def test_cli_writes_twelve_binary_alpha_frames_and_exact_repeat(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source_path = root / "source.png"
            public = root / "public"
            private = root / "private"
            source = Image.new("RGB", (30, 40), (0, 255, 0))
            for row in range(4):
                for column in range(3):
                    left = column * 10 + 2
                    top = row * 10 + 2
                    color = (120 + row * 10, 20 + column * 10, 80)
                    for y in range(top, top + 6):
                        for x in range(left, left + 6):
                            source.putpixel((x, y), color)
            source.save(source_path)

            subprocess.run(
                [
                    sys.executable,
                    str(Path(__file__).with_name("process_modern_map_sprite_sheet.py")),
                    "--source", str(source_path),
                    "--group", "60",
                    "--public", str(public),
                    "--private", str(private),
                    "--chroma-key", "#00ff00",
                    "--chroma-tolerance", "8",
                    "--repeat-frame", "2:1",
                ],
                check=True,
                capture_output=True,
                text=True,
            )

            for output_root in (public, private):
                frames = sorted(output_root.glob("fdicon-060-style-a-f*.png"))
                self.assertEqual(len(frames), 12)
                master = Image.open(output_root / "fdicon-060-style-a-v1-master.png")
                self.assertEqual(master.mode, "RGBA")
                self.assertIn(0, master.getchannel("A").getdata())
                for frame in frames:
                    image = Image.open(frame)
                    self.assertEqual(image.mode, "RGBA")
                    self.assertEqual(image.size, (24, 24))
                    self.assertLessEqual(set(image.getchannel("A").getdata()), {0, 255})
                self.assertEqual(
                    (output_root / "fdicon-060-style-a-f02.png").read_bytes(),
                    (output_root / "fdicon-060-style-a-f01.png").read_bytes(),
                )
            for path in public.iterdir():
                self.assertEqual(path.read_bytes(), (private / path.name).read_bytes())

    def test_static_cli_accepts_chroma_and_writes_three_frames(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source_path = root / "source.png"
            public = root / "public"
            private = root / "private"
            source = Image.new("RGB", (30, 10), (245, 245, 245))
            for column in range(3):
                for y in range(2, 9):
                    for x in range(column * 10 + 2, column * 10 + 8):
                        source.putpixel((x, y), (30, 50 + column * 20, 120))
            source.save(source_path)

            subprocess.run(
                [
                    sys.executable,
                    str(Path(__file__).with_name("process_modern_static_map_sprite_strip.py")),
                    "--source", str(source_path),
                    "--group", "48",
                    "--public", str(public),
                    "--private", str(private),
                    "--chroma-key", "#f5f5f5",
                    "--chroma-tolerance", "16",
                ],
                check=True,
                capture_output=True,
                text=True,
            )

            for output_root in (public, private):
                frames = sorted(output_root.glob("fdicon-048-style-a-f*.png"))
                self.assertEqual(len(frames), 3)
                for frame in frames:
                    image = Image.open(frame)
                    self.assertEqual(image.mode, "RGBA")
                    self.assertEqual(image.size, (24, 24))
                    self.assertLessEqual(set(image.getchannel("A").getdata()), {0, 255})
            for path in public.iterdir():
                self.assertEqual(path.read_bytes(), (private / path.name).read_bytes())


if __name__ == "__main__":
    unittest.main()
