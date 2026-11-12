#!/usr/bin/env python3
"""Deterministic contract tests for the checked-in FD2 locale slice."""
from __future__ import annotations

import json
import re
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))
import validate_locale_packs as validator
OFFICIAL = [ROOT / "remake" / "assets" / "locales" / locale / "pack.json" for locale in ("zh-Hant", "zh-Hans", "ja", "en")]


class LocalePackTests(unittest.TestCase):
    def test_official_packs_have_one_complete_consistent_slice(self):
        packs = validator.validate(OFFICIAL)
        self.assertEqual({p["locale"] for p in packs}, validator.OFFICIAL)
        keys = set(packs[0]["entries"])
        self.assertTrue(keys)
        self.assertTrue(all(set(pack["entries"]) == keys for pack in packs))
        self.assertTrue(all(all(entry["text"] for entry in p["entries"].values()) for p in packs))

    def test_community_pack_may_fallback_and_only_override_known_keys(self):
        payload = json.loads(OFFICIAL[0].read_text(encoding="utf-8"))
        payload["pack_id"] = "community-demo"
        payload["locale"] = "zh-Hant-TW"
        payload["kind"] = "community"
        payload["base_locale"] = "zh-Hant"
        payload["entries"] = {next(iter(payload["entries"])): next(iter(payload["entries"].values()))}
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "community.json"
            path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
            packs = validator.validate(OFFICIAL + [path])
        self.assertEqual(packs[-1]["base_locale"], "zh-Hant")

    def test_official_missing_key_fails_closed(self):
        payload = json.loads(OFFICIAL[1].read_text(encoding="utf-8"))
        payload["entries"].pop(next(iter(payload["entries"])))
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "broken.json"
            path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "key set mismatch"):
                validator.validate([OFFICIAL[0], path, OFFICIAL[2], OFFICIAL[3]])

    def test_variable_signature_mismatch_fails_closed(self):
        payload = json.loads(OFFICIAL[2].read_text(encoding="utf-8"))
        key = "battle.attack.hit"
        payload["entries"][key]["variables"] = ["%s", "%d", "%d"]
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "broken.json"
            path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "variable signature mismatch"):
                validator.validate([OFFICIAL[0], OFFICIAL[1], path, OFFICIAL[3]])

    def test_community_cannot_add_campaign_or_handler_structure(self):
        payload = json.loads(OFFICIAL[0].read_text(encoding="utf-8"))
        payload.update({"pack_id": "community-bad", "locale": "en-GB", "kind": "community", "base_locale": "en", "handler": "battle"})
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "bad.json"
            path.write_text(json.dumps(payload, ensure_ascii=False), encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "invalid fields"):
                validator.validate_one(path)


    def test_approved_sources_match_engine_contract(self):
        # validate_locale_packs.APPROVED_SOURCE 與引擎 officialLocaleContract 各自手動
        # 維護；只改一邊時，語系檔會在驗證工具被拒，或引擎接受未核准的來源。
        source = (ROOT / "remake" / "cmd" / "fd2" / "locale.go").read_text(encoding="utf-8")
        block = re.search(r"var officialLocaleContract = map\[string\]localeEntryContract\{\n(.*?)\n\}", source, re.S)
        self.assertIsNotNone(block)
        engine = dict(re.findall(r'^\t"([^"]+)":\s*\{\[\]string\{[^}]*\}, "([^"]+)"\},$', block.group(1), re.M))
        self.assertEqual(len(engine), block.group(1).count("\n") + 1, "locale.go 有無法解析的契約列")
        info = (ROOT / "remake" / "cmd" / "fd2" / "native_system_info_locale.go").read_text(encoding="utf-8")
        captions = re.search(r"systemInfoCaptionKeys = \[8\]string\{([^}]*)\}", info)
        bounds = re.search(r"for index := (\d+); index < (\d+); index\+\+", info)
        self.assertIsNotNone(captions)
        self.assertIsNotNone(bounds)
        self.assertEqual(tuple(re.findall(r'"([^"]+)"', captions.group(1))), validator.SYSTEM_INFO_CAPTION_KEYS)
        self.assertEqual(range(int(bounds.group(1)), int(bounds.group(2))), validator.SYSTEM_INFO_CONDITION_RANGE)
        for key in validator.SYSTEM_INFO_CAPTION_KEYS:
            engine[f"battle.info.{key}"] = f"FDOTHER_005/system-info/{key}"
        for index in validator.SYSTEM_INFO_CONDITION_RANGE:
            engine[f"battle.info.condition.{index:04d}"] = f"FDTXT_000/string_{index:04d}"
        self.assertEqual(engine, validator.APPROVED_SOURCE)

if __name__ == "__main__":
    unittest.main()
