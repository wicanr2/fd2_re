import io
import json
import struct
import sys
import tempfile
import unittest
from contextlib import redirect_stdout
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import fd2_chapter_slot as tool  # noqa: E402
import fd2save  # noqa: E402


def make_save(records: list[dict], chapter: int, gold: int, slot: int = 0) -> bytes:
    plain = bytearray(fd2save.FILE_SIZE)
    for s in range(fd2save.SLOT_COUNT):
        start, _ = fd2save.slot_bounds(s)
        plain[start + fd2save.ROSTER_SIZE] = 0xFF  # 空槽
    start, _ = fd2save.slot_bounds(slot)
    for index, spec in enumerate(records):
        base = start + index * fd2save.UNIT_SIZE
        plain[base + 6] = 2
        plain[base + 7] = spec["id"]
        plain[base + 8] = spec["id"]
        plain[base + 0x21] = spec.get("level", 1)
        struct.pack_into("<h", plain, base + 0x42, spec.get("max_hp", 30))
        struct.pack_into("<h", plain, base + 0x40, spec.get("hp", spec.get("max_hp", 30)))
        for cell in range(8):
            plain[base + 0x0A + cell * 2] = 0x80
            plain[base + 0x0B + cell * 2] = 0xFF
        for cell, item in enumerate(spec.get("items", [])):
            plain[base + 0x0A + cell * 2] = 0x00
            plain[base + 0x0B + cell * 2] = item
    meta = start + fd2save.ROSTER_SIZE
    plain[meta] = chapter
    plain[meta + 1] = len(records)
    struct.pack_into("<I", plain, meta + 2, gold)
    return fd2save.encode(bytes(plain))


class ChapterSlotToolTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def write(self, name, data):
        path = self.dir / name
        path.write_bytes(data)
        return path

    def test_inspect_reads_verified_fields_only(self):
        path = self.write("a.sav", make_save([{"id": 0, "level": 3, "max_hp": 60, "items": [0x00, 0xC0]}], chapter=3, gold=2000))
        summary = tool.summarize(path, 0)
        self.assertEqual(summary["meta"]["chapter"], 3)
        self.assertEqual(summary["meta"]["gold"], 2000)
        self.assertEqual(summary["meta"]["roster_count"], 1)
        record = summary["records"][0]
        self.assertEqual(record["level"], 3)
        self.assertEqual(record["max_hp"], 60)
        self.assertTrue(record["inventory"].startswith("0000 00c0 80ff"))
        self.assertIn("+0x28..+0x33", record)  # 未證實區段只以偏移顯示

    def test_compare_reports_field_and_membership_differences(self):
        built = self.write("built.sav", make_save(
            [{"id": 0, "level": 2}, {"id": 2, "level": 4}], chapter=3, gold=2000))
        real = self.write("real.sav", make_save(
            [{"id": 0, "level": 3}], chapter=3, gold=2000))
        out = io.StringIO()
        with redirect_stdout(out):
            code = tool.cmd_compare(type("A", (), {"built": built, "real": real, "slot": 0, "strict": True})())
        text = out.getvalue()
        self.assertEqual(code, 1)
        self.assertIn("meta.roster_count: built=2 real=1", text)
        self.assertIn("id8=0.level: built=2 real=3", text)
        self.assertIn("id8=2: 只在 built", text)

    def test_compare_identical_slots(self):
        data = make_save([{"id": 0}], chapter=1, gold=1000)
        a = self.write("a.sav", data)
        b = self.write("b.sav", data)
        out = io.StringIO()
        with redirect_stdout(out):
            code = tool.cmd_compare(type("A", (), {"built": a, "real": b, "slot": 0, "strict": True})())
        self.assertEqual(code, 0)
        self.assertIn("相同", out.getvalue())

    def test_build_rejects_wrong_basename_before_docker(self):
        path = self.write("x.sav", make_save([{"id": 0}], chapter=1, gold=0))
        args = type("A", (), {
            "base": str(path), "out_dir": str(self.dir / "out"), "base_slot": 0, "target": 2,
            "out_slot": -1, "gold": -1, "levels_per_chapter": 0, "level_overrides": "", "seed": 0,
        })()
        with self.assertRaises(SystemExit):
            tool.cmd_build(args)


if __name__ == "__main__":
    unittest.main()
