#!/usr/bin/env python3
import json
from pathlib import Path
import tempfile
import unittest

import sync_native_selector_fields as sync


class NativeTableVersionTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.manifest = Path(self.directory.name) / "reference.json"
        self.manifest.write_text(
            json.dumps({
                "files": [{
                    "file": "FD2.EXE",
                    "size": 357074,
                    "md5": "b97caf2239a27a896069d03549d96e1e",
                    "sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
                }],
            }),
            encoding="utf-8",
        )
        self.tables = {
            "source": "FD2.EXE",
            "source_size": 357074,
            "source_md5": "b97caf2239a27a896069d03549d96e1e",
            "source_sha256": "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
        }

    def tearDown(self):
        self.directory.cleanup()

    def test_accepts_reference_executable(self):
        sync.validate_native_tables(self.tables, self.manifest)

    def test_rejects_unbound_table_dump(self):
        bad = dict(self.tables)
        del bad["source_md5"]
        with self.assertRaisesRegex(ValueError, "source_md5"):
            sync.validate_native_tables(bad, self.manifest)

    def test_rejects_different_executable_hash(self):
        bad = dict(self.tables)
        bad["source_sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "source_sha256"):
            sync.validate_native_tables(bad, self.manifest)


class MapAssetInvariantTest(unittest.TestCase):
    """不需要原始 FDFIELD 也能驗的不變式：同步工具一次寫入的欄位要一起在。

    `native_constructor` 與 `native_record_race`／`native_record_class` 來自同一筆
    b1 選中的建構記錄，同步工具總是一起寫。只剩前者，代表有別的工具重生了這份
    地圖檔、把後兩者丟掉了——`--check` 需要 extracted/raw 才跑得動，這裡先擋。
    """

    def test_constructor_record_keeps_race_and_class(self):
        maps = Path(__file__).resolve().parent.parent / "remake" / "assets" / "maps"
        assets = sorted(maps.glob("map*/map*_units.json"))
        self.assertTrue(assets, "找不到地圖單位檔")
        missing = []
        for path in assets:
            units = json.loads(path.read_text(encoding="utf-8"))["units"]
            for index, unit in enumerate(units):
                if "native_constructor" not in unit:
                    continue
                for field in ("native_record_race", "native_record_class"):
                    if field not in unit:
                        missing.append(f"{path.parent.name} unit {index} {field}")
        self.assertEqual(missing, [], "有建構記錄卻缺種族／職業；跑 sync_native_selector_fields.py --write")


if __name__ == "__main__":
    unittest.main()
