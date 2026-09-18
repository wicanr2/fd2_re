#!/usr/bin/env python3
"""render_parity_progress.py 的區塊產生測試；不碰網路，也不改儲存庫檔案。

執行：python3 -m unittest discover -s tools -p 'test_render_parity_progress.py'
"""

import importlib
import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
render = importlib.import_module("render_parity_progress")


LEDGER = {
    "updated": "2026-09-18",
    "chapters": [
        {"chapter": 4, "status": "passed"},
        {"chapter": 5, "status": "todo"},
    ],
}


class RenderBlockTest(unittest.TestCase):
    def setUp(self):
        self.stats = {
            "receipt": "docs/data/ui-traces/parity-ch04.json",
            "actions": 62,
            "behavior_points": 63,
            "frame_points": 55,
            "frame_zero": 38,
            "frame_max": 201,
            "gates": {"behavior": True, "nodes": True, "transaction": True, "frames": True},
            "save_match": True,
        }
        self._receipt_stats = render.receipt_stats
        self._sample_sheets = render.sample_sheets
        render.receipt_stats = lambda chapter: self.stats if chapter == 4 else None
        render.sample_sheets = lambda chapter: ["docs/figures/parity-ch04-samples-p1.png"]

    def tearDown(self):
        render.receipt_stats = self._receipt_stats
        render.sample_sheets = self._sample_sheets

    def test_readme_and_status_use_their_own_relative_paths(self):
        readme = render.render_block(LEDGER, "readme")
        status = render.render_block(LEDGER, "status")
        self.assertIn("(docs/data/ui-traces/parity-ch04.json)", readme)
        self.assertIn("(docs/figures/parity-ch04-samples-p1.png)", readme)
        self.assertIn("(data/ui-traces/parity-ch04.json)", status)
        self.assertIn("(figures/parity-ch04-samples-p1.png)", status)

    def test_numbers_come_from_the_receipt(self):
        block = render.render_block(LEDGER, "readme")
        self.assertIn("| 第 4 章 | 行為、節點、交易、畫面 | 62 | 63 | 55（38 張逐像素相同） | 201 px | 整檔相同 |", block)
        self.assertIn("台帳更新日 2026-09-18", block)

    def test_pending_chapters_are_named_not_counted_as_passed(self):
        block = render.render_block(LEDGER, "readme")
        self.assertIn("第 4 章已通過", block)
        self.assertIn("共 1 章）還沒跑這套逐章對拍", block)
        self.assertNotIn("| 第 5 章 |", block)

    def test_failed_gate_is_marked_instead_of_silently_listed(self):
        self.stats["gates"]["frames"] = False
        block = render.render_block(LEDGER, "readme")
        self.assertIn("（其餘未過）", block)

    def test_missing_receipt_does_not_invent_numbers(self):
        render.receipt_stats = lambda chapter: None
        block = render.render_block(LEDGER, "readme")
        self.assertIn("收據檔缺失", block)


class WriteBlockTest(unittest.TestCase):
    def test_check_mode_reports_without_writing(self):
        import tempfile

        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "doc.md"
            path.write_text(f"head\n{render.BEGIN}\nold\n{render.END}\ntail\n", encoding="utf-8")
            block = f"{render.BEGIN}\nnew\n{render.END}"
            self.assertTrue(render.write_block(path, block, check=True))
            self.assertIn("old", path.read_text(encoding="utf-8"))
            self.assertTrue(render.write_block(path, block, check=False))
            self.assertIn("new", path.read_text(encoding="utf-8"))
            self.assertFalse(render.write_block(path, block, check=True))

    def test_missing_markers_fail_closed(self):
        import tempfile

        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "doc.md"
            path.write_text("沒有標記\n", encoding="utf-8")
            with self.assertRaises(SystemExit):
                render.write_block(path, "x", check=True)


class LedgerContractTest(unittest.TestCase):
    def test_repository_ledger_has_the_fields_the_tool_reads(self):
        ledger = json.loads(render.LEDGER.read_text(encoding="utf-8"))
        self.assertIn("updated", ledger)
        for entry in ledger["chapters"]:
            self.assertIn("chapter", entry)
            self.assertIn("status", entry)


if __name__ == "__main__":
    unittest.main()
