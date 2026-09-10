"""語系翻譯阻擋清冊的結構與來源一致性測試。"""

import json
import os
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]
BLOCKER_PATH = ROOT / "docs/data/localization/review-blockers.json"
LOCALES = ("zh-Hans", "en", "ja")


def load_entries(locale):
    with (ROOT / f"remake/assets/locales/{locale}/content.json").open(
        encoding="utf-8"
    ) as handle:
        return {entry["string_id"]: entry for entry in json.load(handle)["entries"]}


class LocaleReviewBlockerTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        with BLOCKER_PATH.open(encoding="utf-8") as handle:
            cls.manifest = json.load(handle)
        cls.source = load_entries("zh-Hant")
        cls.translations = {locale: load_entries(locale) for locale in LOCALES}
        cls.allowed = {row["string_id"] for row in cls.manifest["blockers"]}

    def test_manifest_counts_match_the_two_documented_categories(self):
        """清冊不得無聲長大：兩類阻擋各自的筆數釘死，且沒有重複 string_id。"""
        self.assertEqual(len(self.allowed), len(self.manifest["blockers"]))
        by_role = {}
        for row in self.manifest["blockers"]:
            role = self.source[row["string_id"]]["role"]
            by_role.setdefault(role, []).append(row["reason_code"])
        self.assertEqual(sorted(by_role), ["character_name", "dialogue"])
        # 對話本文：語意或專名邊界未閉合。原本還有一筆「來源截斷」，
        # 2026-09-10 以場景證據推翻：那個「我..」是被下一行打斷，不是缺字。
        self.assertEqual(len(by_role["dialogue"]), 4)
        # 說話者欄位：來源只剩單一字模，身分未閉合。
        self.assertEqual(len(by_role["character_name"]), 170)
        self.assertEqual(
            set(by_role["character_name"]), {"unresolved_speaker_identity"})

    def test_speaker_fragments_are_single_glyph_and_stay_blocked(self):
        """說話者碎片的判準要自己成立：來源真的只有一個字，而且沒被升格。"""
        for row in self.manifest["blockers"]:
            if self.source[row["string_id"]]["role"] != "character_name":
                continue
            with self.subTest(string_id=row["string_id"]):
                self.assertEqual(len(row["source_text"]), 1)
                for locale in row["locales"]:
                    entry = self.translations[locale][row["string_id"]]
                    self.assertEqual(entry["status"], "blocked")
                    # 譯文欄位改成「身分未知」，不留機器初稿捏造的名字
                    # （原本是 `惡`→"Shit."、`米`→"Rice" 這種東西）。來源字模
                    # 留在清冊的 source_text，追溯得回去。
                    self.assertEqual(entry["text"], "?")

    def test_blockers_exist_and_match_traditional_source(self):
        for row in self.manifest["blockers"]:
            with self.subTest(string_id=row["string_id"]):
                self.assertIn(row["string_id"], self.source)
                self.assertEqual(row["source_text"], self.source[row["string_id"]]["text"])
                self.assertTrue(row["reason_code"])
                self.assertTrue(row["reason_zh_hant"])
                # 阻擋範圍逐列宣告：對話本文三語都擋，說話者碎片只擋 en／ja
                # （簡中是同一個字模的直接轉換，破的是來源身分不是譯文）。
                self.assertTrue(row["locales"])
                self.assertTrue(set(row["locales"]) <= set(LOCALES))

    def test_listed_blockers_remain_fail_closed(self):
        """清冊項目必須存在，且不得被狀態欄誤升格為已審校。"""
        for row in self.manifest["blockers"]:
            for locale in row["locales"]:
                string_id = row["string_id"]
                with self.subTest(locale=locale, string_id=string_id):
                    self.assertIn(string_id, self.translations[locale])
                    self.assertIn(
                        self.translations[locale][string_id]["status"],
                        {"machine_draft", "blocked"},
                    )

    @unittest.skipUnless(
        os.environ.get("FD2_STRICT_LOCALE_REVIEW") == "1",
        "完整對話驗收時以 FD2_STRICT_LOCALE_REVIEW=1 啟用",
    )
    def test_strict_mode_has_no_unlisted_dialogue_drafts(self):
        for locale, entries in self.translations.items():
            remaining = [
                string_id
                for string_id, entry in entries.items()
                if entry.get("status") in {"machine_draft", "blocked"}
                and entry.get("role") in {"dialogue", "dialogue_or_system"}
                and string_id not in self.allowed
            ]
            self.assertEqual(remaining, [], f"{locale} has unlisted drafts: {remaining}")


if __name__ == "__main__":
    unittest.main()
