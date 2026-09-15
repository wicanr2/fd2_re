import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import fd2_parity_progress as tool  # noqa: E402


class ParityProgressTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        root = Path(self.tmp.name)
        (root / "docs" / "data").mkdir(parents=True)
        self.orig_root, self.orig_ledger = tool.ROOT, tool.LEDGER
        tool.ROOT = root
        tool.LEDGER = root / "docs" / "data" / "parity-campaign-progress.json"
        tool.cmd_init(None)

    def tearDown(self):
        tool.ROOT, tool.LEDGER = self.orig_root, self.orig_ledger
        self.tmp.cleanup()

    def test_init_is_thirty_todo_and_not_passed(self):
        data = tool.load()
        self.assertEqual([c["chapter"] for c in data["chapters"]], list(range(1, 31)))
        self.assertTrue(all(c["status"] == "todo" for c in data["chapters"]))
        self.assertFalse(data["all_chapters_passed"])

    def test_passed_requires_receipt_file_with_passed_status(self):
        data = tool.load()
        data["chapters"][3].update({"status": "passed", "receipt": "docs/data/x.json", "verify": "true"})
        self.assertTrue(any("receipt 不存在" in p for p in tool.problems(data)))
        (tool.ROOT / "docs" / "data" / "x.json").write_text(json.dumps({"status": "candidate"}), encoding="utf-8")
        self.assertTrue(any("收據 status" in p for p in tool.problems(data)))
        (tool.ROOT / "docs" / "data" / "x.json").write_text(json.dumps({"status": "passed"}), encoding="utf-8")
        self.assertEqual(tool.problems(data), [])

    def test_all_chapters_passed_only_when_every_chapter_passed(self):
        data = tool.load()
        (tool.ROOT / "docs" / "data" / "r.json").write_text(json.dumps({"status": "passed"}), encoding="utf-8")
        for c in data["chapters"][:29]:
            c.update({"status": "passed", "receipt": "docs/data/r.json", "verify": "true"})
        tool.save(data)
        self.assertFalse(tool.load()["all_chapters_passed"])
        data["chapters"][29].update({"status": "passed", "receipt": "docs/data/r.json", "verify": "true"})
        tool.save(data)
        self.assertTrue(tool.load()["all_chapters_passed"])

    def test_blocked_needs_reason(self):
        data = tool.load()
        data["chapters"][0].update({"status": "blocked", "verify": "true"})
        self.assertTrue(any("blocked 必須寫原因" in p for p in tool.problems(data)))


if __name__ == "__main__":
    unittest.main()
