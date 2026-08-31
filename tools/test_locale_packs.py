#!/usr/bin/env python3
"""Deterministic contract tests for the checked-in FD2 locale slice."""
from __future__ import annotations

import json
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


if __name__ == "__main__":
    unittest.main()
