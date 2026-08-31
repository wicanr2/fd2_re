#!/usr/bin/env python3

import json
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class LocaleEntitiesTest(unittest.TestCase):
    def test_generated_catalogs_match_checked_in_files(self):
        for locale in ("zh-Hant", "zh-Hans", "ja", "en"):
            with tempfile.TemporaryDirectory() as directory:
                output = Path(directory) / "entities.json"
                subprocess.run(
                    [
                        "python3", str(ROOT / "tools/build_locale_entities.py"),
                        "--campaign", str(ROOT / "remake/assets/scenarios/campaign_full.json"),
                        "--content", str(ROOT / f"remake/assets/locales/{locale}/content.json"),
                        "--overrides", str(ROOT / "remake/assets/locales/entity-overrides.json"),
                        "--characters", str(ROOT / "docs/data/localization/party-character-names.json"),
                        "--item-source", str(ROOT / "docs/data/localization/item-names-zh-Hant.json"),
                        "--item-supplements", str(ROOT / "docs/data/localization/item-name-supplements.json"),
                        "--output", str(output),
                    ],
                    check=True,
                )
                self.assertEqual(
                    json.loads(output.read_text(encoding="utf-8")),
                    json.loads((ROOT / f"remake/assets/locales/{locale}/entities.json").read_text(encoding="utf-8")),
                )


if __name__ == "__main__":
    unittest.main()
