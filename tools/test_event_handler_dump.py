#!/usr/bin/env python3
"""固定原版章節結果清冊回歸；僅在 fd2-cap-local 容器執行。"""

import contextlib
import io
import json
import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import event_handler_dump


class EventHandlerDumpTest(unittest.TestCase):
    def test_ch13_dynamic_scan_does_not_inherit_stack_probe_argument(self):
        root = Path(__file__).resolve().parents[1]
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(event_handler_dump.main([
                "event_handler_dump.py",
                str(root / "org_game/炎龍騎士團/FLAME2/FD2.EXE"), "json",
            ]), 0)
        rows = json.loads(output.getvalue())
        self.assertEqual(len(rows), 30)
        row = next(row for row in rows if row["chapter"] == 12)
        # IDA 0x207DE push3Bh；0x20786..0x2078A 是 eax=ebx+15，
        # 原始入口 push30h 只交給 sub_36CD7，不能當作 runtime48。
        self.assertEqual(row["trigger_units_flag"], [59])
        self.assertEqual(row["extra_conditions"], ["raw_record_byte5_bit0"])
        self.assertEqual(row["result_codes"], [1])


if __name__ == "__main__":
    unittest.main()
