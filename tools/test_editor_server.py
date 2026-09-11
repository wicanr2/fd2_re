#!/usr/bin/env python3
"""編輯器檔案橋的兩件事：寫回不改格式，以及路徑白名單擋得住。

**格式保真**不是美觀問題。受版控的 JSON 縮排不一致（story 1 空格、scenario 2、
`map.json` 單行且完全緊湊、`map0_units.json` 連結尾換行都沒有），統一格式會讓每次
存檔都產生整份 diff——真正改到的那一行就埋在裡面，審查等於沒做。

這裡直接測純函式（讀進來再 dump 出去要等於原檔位元組），不經過 HTTP，所以測試本身
一個字都不會寫進受版控的檔案。
"""

import importlib.util
import json
import pathlib
import sys
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location(
    "editor_serve", ROOT / "tools" / "editor" / "serve.py")
serve = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(serve)

# 四種不同的格式，每一種都要能原樣走完一圈。
SAMPLES = [
    "remake/assets/story/ch01.json",              # 1 空格縮排
    "remake/assets/scenarios/ch01.json",          # 2 空格縮排
    "remake/assets/scenarios/campaign_full.json", # 1 空格縮排、299 個節點
    "remake/assets/maps/map0/map.json",           # 單行、完全緊湊
    "remake/assets/maps/map0/map0_units.json",    # 單行、沒有結尾換行
]


class FormatFidelity(unittest.TestCase):
    def test_round_trip_reproduces_the_file_byte_for_byte(self):
        for name in SAMPLES:
            with self.subTest(name):
                path = ROOT / name
                original = path.read_text(encoding="utf-8")
                again = serve.dump(json.loads(original), serve.style_of(path))
                self.assertEqual(again, original, f"{name} 原樣存回之後變了")

    def test_shipped_json_is_either_faithful_or_refused(self):
        """每一份要嘛能原樣走完一圈，要嘛是 server 會拒寫的手工排版檔。

        重點不是「全部都行」——`cutscenes/acting/*.json` 把
        `{ "slot": 34, "pose": 2 }` 寫在同一行，程式化的 dump 重現不了。重點是
        **沒有第三種**：既保不住格式又寫得進去的檔案，寫一次就整份重排。
        """
        faithful = refused = 0
        for path in sorted((ROOT / "remake/assets").rglob("*.json")):
            if "editor-canonical" in path.parts:
                continue
            original = path.read_text(encoding="utf-8")
            try:
                value = json.loads(original)
            except json.JSONDecodeError:
                continue
            if serve.dump(value, serve.style_of(path)) == original:
                faithful += 1
            else:
                refused += 1
        self.assertGreater(faithful, 50, f"能保真的只有 {faithful} 份，樣本太少")
        self.assertGreater(refused, 0, "一份手工排版的都沒找到：拒寫那條路徑沒被走過")

    def test_detects_each_style_separately(self):
        """反對照：三種風格要真的被分開，不是剛好都用同一組設定。"""
        styles = {name: serve.style_of(ROOT / name) for name in SAMPLES}
        indents = {style[0] for style in styles.values()}
        self.assertIn(1, indents)
        self.assertIn(2, indents)
        self.assertIn(None, indents, "沒有偵測到單行檔")
        trailing = {style[2] for style in styles.values()}
        self.assertEqual(trailing, {True, False}, "沒有偵測到缺結尾換行的檔案")


class PathAllowList(unittest.TestCase):
    def test_accepts_assets_json(self):
        got = serve.resolve("remake/assets/story/ch01.json", serve.WRITABLE_SUFFIXES)
        self.assertIsNotNone(got)

    def test_rejects_outside_assets(self):
        """反對照：白名單外的受版控檔案不能寫。"""
        self.assertIsNone(serve.resolve("docs/data/fd2-worklist.json", serve.WRITABLE_SUFFIXES))
        self.assertIsNone(serve.resolve("tools/editor/serve.py", serve.WRITABLE_SUFFIXES))

    def test_rejects_traversal(self):
        for raw in ("remake/assets/../../etc/passwd.json",
                    "../../../etc/passwd.json",
                    "/etc/passwd.json",
                    "remake/assets/../docs/data/fd2-worklist.json"):
            self.assertIsNone(serve.resolve(raw, serve.WRITABLE_SUFFIXES), raw)

    def test_png_is_readable_but_not_writable(self):
        png = "remake/assets/maps/map0/tileset.png"
        self.assertIsNotNone(serve.resolve(png, serve.READABLE_SUFFIXES))
        self.assertIsNone(serve.resolve(png, serve.WRITABLE_SUFFIXES))

    def test_rejects_other_suffixes(self):
        for raw in ("remake/assets/maps/map0/map.dat",
                    "remake/assets/notes.md",
                    "remake/assets/run.sh"):
            self.assertIsNone(serve.resolve(raw, serve.READABLE_SUFFIXES), raw)

    def test_rejects_empty(self):
        self.assertIsNone(serve.resolve("", serve.READABLE_SUFFIXES))
        self.assertIsNone(serve.resolve(None, serve.READABLE_SUFFIXES))


if __name__ == "__main__":
    sys.exit(0 if unittest.main(exit=False).result.wasSuccessful() else 1)
