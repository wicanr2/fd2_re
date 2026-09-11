#!/usr/bin/env python3
"""fd2_worklist_issues.py 的解析與快照測試；不碰網路。

權威是 GitHub Issues，快照是從 issue 拉下來的。這裡驗兩件事：格式化之後再解析要
拿回同一個條目（不然 pull 會悄悄改掉內容），以及任何一個讀不懂的 issue 都要讓
整批 pull 失敗（不然快照會少一條而沒有人發現）。

執行：python3 -m unittest discover -s tools -p 'test_fd2_worklist_issues.py'
"""

import importlib
import os
import shutil
import tempfile
import unittest
from pathlib import Path


def load_modules(root):
    os.environ["FD2_WORKLIST_ROOT"] = str(root)
    os.environ.pop("FD2_WORKLIST_DATA", None)
    worklist = importlib.reload(importlib.import_module("fd2_worklist"))
    issues = importlib.reload(importlib.import_module("fd2_worklist_issues"))
    return worklist, issues


def restore_modules():
    os.environ.pop("FD2_WORKLIST_ROOT", None)
    importlib.reload(importlib.import_module("fd2_worklist"))
    importlib.reload(importlib.import_module("fd2_worklist_issues"))


SCHEMA = {
    "schema": "fd2-worklist/1",
    "note": "快照",
    "layers": {"runtime": "還沒接進正式執行期", "re": "原版證據還沒閉合"},
    "categories": {"工作": "要做的", "缺陷": "做錯的", "RE待解": "沒閉合的"},
    "verify_kinds": {"manual": "要人判", "present": "自承還在"},
    "items": [],
}


def issue(number, item, state="OPEN", extra_labels=()):
    return {"number": number, "title": item["title"], "state": state,
            "labels": [{"name": "worklist"}, {"name": f"分層:{item['layer']}"},
                       {"name": f"類別:{item['category']}"}] + [{"name": n} for n in extra_labels]}


class IssueFormat(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.addCleanup(shutil.rmtree, self.dir)
        self.addCleanup(restore_modules)
        self.root = Path(self.dir)
        (self.root / "src").mkdir()
        (self.root / "src" / "gap.go").write_text("// 還沒接上\n", encoding="utf-8")
        _, self.tool = load_modules(self.root)
        self.item = {
            "id": "alpha", "layer": "runtime", "category": "缺陷", "title": "甲",
            "body": "說明第一段。\n\n第二段。", "acceptance": "做完的樣子", "evidence": "docs/a.md §2",
            "verify": {"kind": "present", "paths": ["src"], "pattern": "還沒接上"},
        }

    def as_issue(self, number=7, **kw):
        found = issue(number, self.item, **kw)
        found["body"] = self.tool.format_body(self.item)
        return found

    def test_format_then_parse_round_trips(self):
        parsed, problem = self.tool.parse_issue(self.as_issue(), SCHEMA)
        self.assertIsNone(problem)
        self.assertEqual(parsed, {**self.item, "issue": 7})

    def test_human_edits_to_prose_survive(self):
        found = self.as_issue()
        found["body"] = "人改過的說明。\n\n" + found["body"].split("\n\n", 2)[2]
        parsed, _ = self.tool.parse_issue(found, SCHEMA)
        self.assertEqual(parsed["body"], "人改過的說明。")

    def test_missing_block_or_labels_is_reported(self):
        no_block = self.as_issue()
        no_block["body"] = "只有文字"
        self.assertIn("沒有 fd2-worklist 區塊", self.tool.parse_issue(no_block, SCHEMA)[1])
        bad_json = self.as_issue()
        bad_json["body"] = "```fd2-worklist\n{not json}\n```"
        self.assertIn("不是 JSON", self.tool.parse_issue(bad_json, SCHEMA)[1])
        two_layers = self.as_issue(extra_labels=("分層:re",))
        self.assertIn("恰好一個分層", self.tool.parse_issue(two_layers, SCHEMA)[1])

    def test_unknown_verify_kind_is_rejected(self):
        self.item["verify"] = {"kind": "json_len"}
        self.assertIn("verify", self.tool.parse_issue(self.as_issue(), SCHEMA)[1])

    def test_snapshot_keeps_open_issues_in_number_order(self):
        beta = dict(self.item, id="beta", title="乙")
        second = issue(3, beta)
        second["body"] = self.tool.format_body(beta)
        closed = self.as_issue(number=9, state="CLOSED")
        snapshot = self.tool.build_snapshot(SCHEMA, [self.as_issue(), second, closed])
        self.assertEqual([(i["issue"], i["id"]) for i in snapshot["items"]], [(3, "beta"), (7, "alpha")])
        self.assertEqual(snapshot["layers"], SCHEMA["layers"])

    def test_one_unreadable_issue_fails_the_whole_pull(self):
        broken = self.as_issue(number=8)
        broken["body"] = "壞掉"
        with self.assertRaises(SystemExit):
            self.tool.build_snapshot(SCHEMA, [self.as_issue(), broken])

    def test_duplicate_ids_fail_the_pull(self):
        with self.assertRaises(SystemExit):
            self.tool.build_snapshot(SCHEMA, [self.as_issue(number=1), self.as_issue(number=2)])

    def test_stale_verify_plans_label_and_clears_it_again(self):
        data = dict(SCHEMA, items=[{**self.item, "issue": 7}])
        self.assertEqual(self.tool.plan_label_updates(data, [self.as_issue()]), [])
        (self.root / "src" / "gap.go").write_text("// 已接上\n", encoding="utf-8")
        updates = self.tool.plan_label_updates(data, [self.as_issue()])
        self.assertEqual(updates, [{"number": 7, "id": "alpha", "add": ["verify:可能已完成"], "remove": []}])
        stale = self.as_issue(extra_labels=("verify:可能已完成",))
        (self.root / "src" / "gap.go").write_text("// 還沒接上\n", encoding="utf-8")
        self.assertEqual(self.tool.plan_label_updates(data, [stale])[0]["remove"], ["verify:可能已完成"])

    def test_migration_rewrites_only_old_marker_bodies(self):
        data = dict(SCHEMA, items=[self.item])
        old = issue(4, self.item)
        old["body"] = "<!-- fd2-worklist: alpha -->\n舊的鏡像內文"
        updates = self.tool.plan_migration(data, [old, self.as_issue(number=5)])
        self.assertEqual([u["number"] for u in updates], [4])
        parsed, problem = self.tool.parse_issue({**old, "body": updates[0]["body"]}, SCHEMA)
        self.assertIsNone(problem)
        self.assertEqual(parsed["verify"], self.item["verify"])


if __name__ == "__main__":
    unittest.main()
