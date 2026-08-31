#!/usr/bin/env python3

import json
import subprocess
import tempfile
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]


class FDTXTItemNamesTest(unittest.TestCase):
    def test_fixed_resource_round_trip_matches_checked_in_catalog(self):
        resource = ROOT / "remake/generated-assets/fd2-original-b97caf22/text/FDTXT_000/resource.json"
        if not resource.is_file():
            self.skipTest("private separated FDTXT_000 metadata is unavailable")
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / "items.json"
            subprocess.run(
                [
                    "python3", str(ROOT / "tools/export_fdtxt_item_names.py"),
                    "--resource", str(resource),
                    "--output", str(output),
                ],
                check=True,
            )
            self.assertEqual(
                json.loads(output.read_text(encoding="utf-8")),
                json.loads((ROOT / "docs/data/localization/item-names-zh-Hant.json").read_text(encoding="utf-8")),
            )


if __name__ == "__main__":
    unittest.main()
