#!/usr/bin/env python3
"""fd2_worklist_issues.py 的同步計畫測試；不碰網路。

每一條規則都驗到「會做事」與「不會多做事」兩邊：沒有差異時計畫必須是空的，否則
每次同步都會改一輪 issue，真正的變化就淹沒在雜訊裡。

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


class PlanActions(unittest.TestCase):
    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.root = Path(self.dir)
        self.addCleanup(shutil.rmtree, self.dir)
        self.addCleanup(restore_modules)
        (self.root / "src").mkdir()
        (self.root / "src" / "gap.go").write_text("// 還沒接上\n", encoding="utf-8")
        _, self.tool = load_modules(self.root)
        self.data = {
            "layers": {"runtime": "還沒接進正式執行期", "re": "原版證據還沒閉合"},
            "categories": {"工作": "要做的", "缺陷": "做錯的", "RE待解": "沒閉合的"},
            "verify_kinds": {"manual": "要人判", "present": "自承還在"},
            "items": [
                {"id": "alpha", "layer": "runtime", "category": "缺陷", "title": "甲",
                 "body": "內文", "verify": {"kind": "present", "paths": ["src"],
                                            "pattern": "還沒接上"}},
                {"id": "beta", "layer": "re", "category": "RE待解", "title": "乙",
                 "verify": {"kind": "manual"}},
            ],
        }

    def mirror(self, number, item, state="OPEN", extra_labels=()):
        """一個與條目完全同步的既有 issue。"""
        want = self.tool.desired_issue(item, self.data)
        return {
            "number": number, "title": want["title"], "body": want["body"], "state": state,
            "labels": [{"name": name} for name in sorted(want["labels"]) + list(extra_labels)],
        }

    def test_creates_every_item_without_an_issue(self):
        actions = self.tool.plan_actions(self.data, [])
        self.assertEqual([a["op"] for a in actions], ["create", "create"])
        alpha = actions[0]
        self.assertTrue(alpha["body"].startswith("<!-- fd2-worklist: alpha -->"))
        self.assertEqual(alpha["labels"], {"worklist", "分層:runtime", "類別:缺陷"})
        self.assertIn("verify:要人判", actions[1]["labels"])

    def test_in_sync_plans_nothing(self):
        for index, item in enumerate(self.data["items"], start=1):
            item["issue"] = index
        issues = [self.mirror(1, self.data["items"][0]), self.mirror(2, self.data["items"][1])]
        self.assertEqual(self.tool.plan_actions(self.data, issues), [])

    def test_records_number_missing_from_json(self):
        issues = [self.mirror(7, self.data["items"][0]), self.mirror(8, self.data["items"][1])]
        actions = self.tool.plan_actions(self.data, issues)
        self.assertEqual([(a["op"], a["id"], a["number"]) for a in actions],
                         [("record", "alpha", 7), ("record", "beta", 8)])

    def test_updates_changed_title_body_and_labels_but_keeps_human_labels(self):
        self.data["items"][0]["issue"] = 3
        self.data["items"][1]["issue"] = 4
        stale = self.mirror(3, self.data["items"][0], extra_labels=("help wanted", "分層:re"))
        stale["title"] = "舊標題"
        stale["body"] = stale["body"].replace("內文", "舊內文")
        issues = [stale, self.mirror(4, self.data["items"][1])]
        actions = self.tool.plan_actions(self.data, issues)
        self.assertEqual(len(actions), 1)
        update = actions[0]
        self.assertEqual(update["op"], "update")
        self.assertEqual(update["title"], "甲")
        self.assertIn("內文", update["body"])
        self.assertEqual(update["remove_labels"], ["分層:re"])
        self.assertNotIn("add_labels", update)

    def test_reopens_closed_issue_whose_item_is_still_listed(self):
        self.data["items"][0]["issue"] = 5
        self.data["items"][1]["issue"] = 6
        issues = [self.mirror(5, self.data["items"][0], state="CLOSED"),
                  self.mirror(6, self.data["items"][1])]
        actions = self.tool.plan_actions(self.data, issues)
        self.assertEqual([(a["op"], a["number"]) for a in actions], [("reopen", 5)])

    def test_closes_issue_whose_item_was_removed(self):
        removed = self.data["items"].pop(0)
        self.data["items"][0]["issue"] = 2
        issues = [self.mirror(1, removed), self.mirror(2, self.data["items"][0])]
        actions = self.tool.plan_actions(self.data, issues, head="abc1234")
        self.assertEqual([(a["op"], a["number"]) for a in actions], [("close", 1)])
        self.assertIn("abc1234", actions[0]["comment"])
        # 已經關掉的就不再動它。
        issues[0]["state"] = "CLOSED"
        self.assertEqual(self.tool.plan_actions(self.data, issues), [])

    def test_stale_verify_adds_label_instead_of_closing(self):
        self.data["items"][0]["issue"] = 1
        self.data["items"][1]["issue"] = 2
        issues = [self.mirror(1, self.data["items"][0]), self.mirror(2, self.data["items"][1])]
        (self.root / "src" / "gap.go").write_text("// 已接上\n", encoding="utf-8")
        actions = self.tool.plan_actions(self.data, issues)
        self.assertEqual(len(actions), 1)
        self.assertEqual(actions[0]["op"], "update")
        self.assertEqual(actions[0]["add_labels"], ["verify:可能已完成"])
        self.assertIn("可能已完成", actions[0]["body"])

    def test_ignores_issues_without_marker(self):
        stray = {"number": 9, "title": "人手開的", "body": "沒有標記", "state": "OPEN",
                 "labels": [{"name": "worklist"}]}
        actions = self.tool.plan_actions(self.data, [stray])
        self.assertEqual([a["op"] for a in actions], ["create", "create"])

    def test_rejects_two_issues_for_one_item(self):
        issues = [self.mirror(1, self.data["items"][0]), self.mirror(2, self.data["items"][0])]
        with self.assertRaises(SystemExit):
            self.tool.plan_actions(self.data, issues)

    def test_evidence_links_to_main_and_keeps_section(self):
        self.assertEqual(
            self.tool.evidence_link("docs/a.md §3"),
            "[`docs/a.md`](https://github.com/wicanr2/fd2_re/blob/main/docs/a.md) §3")


class RealWorklistLabels(unittest.TestCase):
    def test_every_label_the_real_worklist_needs_is_defined(self):
        worklist, tool = load_modules(Path(__file__).resolve().parent.parent)
        self.addCleanup(restore_modules)
        data = worklist.load()
        defined = tool.labels_needed(data)
        for item in data["items"]:
            with self.subTest(item["id"]):
                self.assertIn("category", item, "每一條都要標類別")
                for label in tool.desired_issue(item, data)["labels"]:
                    self.assertIn(label, defined)


if __name__ == "__main__":
    unittest.main()
