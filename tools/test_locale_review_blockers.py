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

    def test_manifest_has_exactly_eleven_unique_entries(self):
        self.assertEqual(len(self.manifest["blockers"]), 11)
        self.assertEqual(len(self.allowed), 11)

    def test_blockers_exist_and_match_traditional_source(self):
        for row in self.manifest["blockers"]:
            with self.subTest(string_id=row["string_id"]):
                self.assertIn(row["string_id"], self.source)
                self.assertEqual(row["source_text"], self.source[row["string_id"]]["text"])
                self.assertTrue(row["reason_code"])
                self.assertTrue(row["reason_zh_hant"])
                self.assertEqual(set(row["locales"]), set(LOCALES))

    def test_listed_blockers_remain_fail_closed(self):
        """清冊項目必須存在，且不得被狀態欄誤升格為已審校。"""
        for locale, entries in self.translations.items():
            for string_id in self.allowed:
                with self.subTest(locale=locale, string_id=string_id):
                    self.assertIn(string_id, entries)
                    self.assertIn(
                        entries[string_id]["status"], {"machine_draft", "blocked"}
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
