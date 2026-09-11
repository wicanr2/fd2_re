#!/usr/bin/env python3
"""身份候選名稱的分類判準要分得開三種情況，而且不能把真的衝突放行。

一個身份有多個候選名稱不一定是判錯：原版就有場景相依的稱呼（索爾／索爾(少年)、
刺客／蘭斯洛特）與共用 sprite 的雜兵編號（強盜 B/C/L/M/N）。逼人「選一個」等於把
場景相依的稱呼壓成單一值，資料反而變差。

判準放行得太寬會讓真的衝突靜靜地降級成 note，所以這裡兩個方向都驗。
"""

import importlib.util
import pathlib
import sys
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location(
    "export_editor_canonical", ROOT / "tools" / "export_editor_canonical.py")
exporter = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(exporter)


def identity(pairs):
    """pairs: [(名稱, 來源章節), …] → 一個只帶 display_name_candidates 的身份。"""
    return {
        "character_id": "character/native-test",
        "display_name_candidates": [
            {"value": name, "source": f"remake/assets/scenarios/ch{chapter:02d}.json",
             "field": "party.name", "index": 0, "evidence": "direct legacy field"}
            for name, chapter in pairs
        ],
    }


class Classification(unittest.TestCase):
    def classify(self, pairs):
        return exporter._classify_multi_candidate(identity(pairs), "display_name_candidates")

    def test_disjoint_chapters_are_scene_dependent(self):
        """索爾 0–31、索爾(少年) 32–33：章節互不重疊。"""
        got = self.classify([("索爾", 1), ("索爾", 31), ("索爾(少年)", 32), ("索爾(少年)", 33)])
        self.assertEqual(got["code"], "scene_dependent_name")
        self.assertEqual(got["severity"], "note")
        self.assertIn("chapters", got)

    def test_reveal_name_is_scene_dependent(self):
        """刺客只在第 3 章、蘭斯洛特在 18 章之後：劇情揭露前後的稱呼。"""
        got = self.classify([("刺客", 3), ("蘭斯洛特", 18), ("蘭斯洛特", 30)])
        self.assertEqual(got["code"], "scene_dependent_name")

    def test_shared_prefix_in_one_chapter_is_extras(self):
        """強盜 B/C/L 全在第 2 章：多個雜兵共用同一個身份。"""
        got = self.classify([("強盜B", 2), ("強盜C", 2), ("強盜L", 2)])
        self.assertEqual(got["code"], "shared_extra_identity")
        self.assertEqual(got["severity"], "note")
        self.assertEqual(got["shared_prefix"], "強盜")

    def test_same_chapter_different_names_stays_an_error(self):
        """反對照：同一章裡兩個沒有關係的名字，那才是真的衝突。"""
        got = self.classify([("索爾", 5), ("悠妮", 5)])
        self.assertEqual(got["code"], "conflicting_identity_candidates")
        self.assertEqual(got["severity"], "error")

    def test_short_shared_prefix_is_not_extras(self):
        """反對照：哈瓦特與哈諾共同前綴只有一個字、剩餘兩個字，不能併成雜兵。"""
        got = self.classify([("哈瓦特", 5), ("哈諾", 5)])
        self.assertEqual(got["code"], "conflicting_identity_candidates")

    def test_overlapping_chapters_are_not_scene_dependent(self):
        """反對照：章節有重疊就不是場景相依，即使大部分章節分得開。"""
        got = self.classify([("甲名", 1), ("甲名", 5), ("乙名", 5), ("乙名", 9)])
        self.assertEqual(got["code"], "conflicting_identity_candidates")

    def test_missing_chapter_information_stays_an_error(self):
        """反對照：來源看不出章節時不能當成不重疊——那是資訊不足，不是證據。"""
        blank = {
            "character_id": "character/native-test",
            "display_name_candidates": [
                {"value": "甲名", "source": "remake/assets/data/item.json",
                 "field": "party.name", "index": 0, "evidence": "direct legacy field"},
                {"value": "乙名", "source": "remake/assets/data/item.json",
                 "field": "party.name", "index": 1, "evidence": "direct legacy field"},
            ],
        }
        got = exporter._classify_multi_candidate(blank, "display_name_candidates")
        self.assertEqual(got["code"], "conflicting_identity_candidates")


class ShippedCatalog(unittest.TestCase):
    """實際的 bundle 走完同一條判準之後不該再留下 error。"""

    def test_no_unclassified_conflicts_remain(self):
        import json
        catalog = json.loads((ROOT / "remake/assets/editor-canonical/character-identity.json")
                             .read_text(encoding="utf-8"))
        errors = [d for d in catalog["diagnostics"] if d["severity"] == "error"]
        self.assertEqual(errors, [], "還有沒分類的候選衝突")
        self.assertTrue(catalog["diagnostics"], "一筆診斷都沒有：判準可能把東西吃掉了")
        for diagnostic in catalog["diagnostics"]:
            self.assertIn("chapters", diagnostic, "分類要附上章節依據")


if __name__ == "__main__":
    sys.exit(0 if unittest.main(exit=False).result.wasSuccessful() else 1)
