#!/usr/bin/env python3
"""fd2_worklist.py 的正反對照測試。

只驗「它印出仍未完成」證明不了機制有在看——空的檔案、寫錯的路徑、永遠為真的
pattern 都會印出一樣的好消息。所以每一條非 manual 的條目都要驗兩個方向：訊號
在的時候報「仍未完成」，把訊號拿掉之後真的開口。

執行：python3 -m unittest discover -s tools -p 'test_fd2_worklist.py'
"""

import importlib
import json
import os
import re
import shutil
import tempfile
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent


def load_tool(root=None, data=None):
    """以指定的根目錄重新載入工具模組。"""
    os.environ["FD2_WORKLIST_ROOT"] = str(root or REPO)
    if data:
        os.environ["FD2_WORKLIST_DATA"] = str(data)
    else:
        os.environ.pop("FD2_WORKLIST_DATA", None)
    module = importlib.import_module("fd2_worklist")
    return importlib.reload(module)


def write(path, text):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


class RealWorklistBothDirections(unittest.TestCase):
    """對真正的 worklist 逐條驗兩個方向。"""

    def setUp(self):
        self.tool = load_tool()
        self.data = self.tool.load()

    def test_every_item_is_open_on_the_current_tree(self):
        for item in self.data["items"]:
            with self.subTest(item["id"]):
                open_, why = self.tool.still_open(item)
                self.assertTrue(open_, f'{item["id"]} 在目前的樹上被判成已完成：{why}')

    def test_present_items_speak_up_when_the_signal_goes_away(self):
        """把自承那幾行刪掉，條目就該說「可能已完成」。"""
        items = [i for i in self.data["items"] if i["verify"]["kind"] == "present"]
        self.assertTrue(items, "worklist 裡沒有 present 條目，這個測試就沒有意義")
        for item in items:
            with self.subTest(item["id"]):
                verify = item["verify"]
                expression = re.compile(verify["pattern"])
                with tempfile.TemporaryDirectory() as raw:
                    root = Path(raw)
                    for target in verify["paths"]:
                        source = REPO / target
                        files = sorted(source.rglob("*")) if source.is_dir() else [source]
                        for path in files:
                            if not self.tool.scannable(path):
                                continue
                            text = path.read_text(encoding="utf-8", errors="ignore")
                            stripped = "\n".join(
                                line for line in text.split("\n")
                                if not expression.search(line)
                            )
                            write(root / path.relative_to(REPO), stripped)
                    tool = load_tool(root)
                    open_, why = tool.still_open(item)
                    self.assertFalse(
                        open_, f'{item["id"]} 的訊號拿掉之後還是說仍未完成：{why}')
        load_tool()

    def test_absent_items_speak_up_when_the_symbol_appears(self):
        """把 pattern 會中的字樣放進產品檔，條目就該說「可能已完成」。"""
        items = [i for i in self.data["items"] if i["verify"]["kind"] == "absent"]
        self.assertTrue(items, "worklist 裡沒有 absent 條目，這個測試就沒有意義")
        samples = {
            "wasm-web-release": "GOOS=js GOARCH=wasm go build ./cmd/fd2\n",
            "android-package": "ebitenmobile bind -target android ./mobile\n",
            "parity-ch02-ch04-original-side": "# 第四關：從第三關的續跑點起跑\n",
            "native-level-cap": "// 0x1E2E0：+7 為 0x1E／0x1F 時等級上限 99，其餘 40\n",
        }
        for item in items:
            with self.subTest(item["id"]):
                self.assertIn(item["id"], samples, "新增 absent 條目時要一起補樣本")
                with tempfile.TemporaryDirectory() as raw:
                    root = Path(raw)
                    target = Path(item["verify"]["paths"][0]) / "sample.sh"
                    write(root / target, samples[item["id"]])
                    tool = load_tool(root)
                    open_, why = tool.still_open(item)
                    self.assertFalse(
                        open_, f'{item["id"]} 在字樣出現之後還是說仍未完成：{why}')
        load_tool()


class VerifyKinds(unittest.TestCase):
    """四種 verify 各自的行為，包含測試檔要被跳過。"""

    def setUp(self):
        self.dir = tempfile.mkdtemp()
        self.root = Path(self.dir)
        self.addCleanup(shutil.rmtree, self.dir)
        self.addCleanup(load_tool)

    def build(self, items, extra_layers=None):
        data = {
            "schema": "fd2-worklist/1",
            "layers": {"runtime": "測試用"} | (extra_layers or {}),
            "verify_kinds": {
                "present": "", "absent": "", "json_len": "", "manual": "",
            },
            "items": items,
        }
        path = self.root / "worklist.json"
        write(path, json.dumps(data, ensure_ascii=False))
        return load_tool(self.root, path)

    def item(self, verify, ident="probe"):
        return {"id": ident, "layer": "runtime", "title": "測試用", "verify": verify}

    def test_manual_always_reports_open(self):
        tool = self.build([self.item({"kind": "manual", "note": "要人判"})])
        open_, why = tool.still_open(tool.load()["items"][0])
        self.assertTrue(open_)
        self.assertIn("要人判", why)

    def test_present_both_directions(self):
        write(self.root / "src/app.go", "// 這一段尚未接線\n")
        tool = self.build([self.item(
            {"kind": "present", "paths": ["src"], "pattern": "尚未接線"})])
        self.assertTrue(tool.still_open(tool.load()["items"][0])[0])
        write(self.root / "src/app.go", "// 已經接好了\n")
        self.assertFalse(tool.still_open(tool.load()["items"][0])[0])

    def test_absent_both_directions(self):
        write(self.root / "src/app.go", "package main\n")
        tool = self.build([self.item(
            {"kind": "absent", "paths": ["src"], "pattern": "NewCompiler"})])
        self.assertTrue(tool.still_open(tool.load()["items"][0])[0])
        write(self.root / "src/app.go", "func NewCompiler() {}\n")
        self.assertFalse(tool.still_open(tool.load()["items"][0])[0])

    def test_test_files_do_not_count(self):
        """測試檔提到還沒接上的東西是常態，掃進來會把真缺口蓋掉。"""
        write(self.root / "src/app_test.go", "func TestNewCompiler() { NewCompiler() }\n")
        write(self.root / "src/test_helper.py", "NewCompiler()\n")
        tool = self.build([self.item(
            {"kind": "absent", "paths": ["src"], "pattern": "NewCompiler"})])
        open_, why = tool.still_open(tool.load()["items"][0])
        self.assertTrue(open_, f"測試檔不該讓 absent 判成已完成：{why}")

    def test_json_len_both_directions(self):
        write(self.root / "data.json", json.dumps({"rows": [1, 2]}))
        tool = self.build([self.item({
            "kind": "json_len", "path": "data.json", "field": "rows", "max": 2})])
        self.assertTrue(tool.still_open(tool.load()["items"][0])[0])
        write(self.root / "data.json", json.dumps({"rows": [1, 2, 3]}))
        self.assertFalse(tool.still_open(tool.load()["items"][0])[0])

    def test_schema_rejects_duplicate_ids(self):
        verify = {"kind": "manual", "note": ""}
        tool = self.build([self.item(verify, "same"), self.item(verify, "same")])
        with self.assertRaises(SystemExit):
            tool.load()

    def test_schema_rejects_undefined_layer(self):
        item = self.item({"kind": "manual", "note": ""})
        item["layer"] = "沒定義的分層"
        tool = self.build([item])
        with self.assertRaises(SystemExit):
            tool.load()


if __name__ == "__main__":
    unittest.main()
