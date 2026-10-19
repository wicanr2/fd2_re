#!/usr/bin/env python3
"""原版來源盤點的拒收測試；合法包由 FD2_TEST_ORIGINAL_PACK 指定。"""

import hashlib
import json
import os
from pathlib import Path
import tempfile
import unittest

import audit_fd2_battle_animation_hd as audit


@unittest.skipUnless(os.environ.get("FD2_TEST_ORIGINAL_PACK"), "requires an explicit legal original pack")
class SourceInventoryGuardTests(unittest.TestCase):
    def setUp(self):
        self.source = Path(os.environ["FD2_TEST_ORIGINAL_PACK"]).resolve()
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.pack = Path(self.temp.name)
        for source in self.source.iterdir():
            if source.name != "manifest.json":
                (self.pack / source.name).symlink_to(source, target_is_directory=source.is_dir())
        self.manifest = json.loads((self.source / "manifest.json").read_text())
        self.catalog = audit.REPO / "remake/assets/themes/modern/catalog.json"

    def run_audit(self):
        (self.pack / "manifest.json").write_text(json.dumps(self.manifest))
        return audit.audit(self.pack, self.catalog, 30)

    def test_wrong_fixed_source_rejected(self):
        next(x for x in self.manifest["source_set"] if x["file"] == "FIGANI.DAT")["sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "來源指紋不符"):
            self.run_audit()

    def test_wrong_image_hash_rejected(self):
        next(x for x in self.manifest["assets"] if x["asset_id"] == "battle_animation/animations/figani_000/figani_000_f00.png")["sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "資產雜湊不符"):
            self.run_audit()

    def test_reblessed_raw_still_rejected_by_original_archive_hash(self):
        # 只替換隔離 fixture 的 raw0，保留原版包唯讀。即使 manifest 的兩個
        # raw 雜湊一起改成新值，也必須被固定原版容器雜湊拒絕。
        (self.pack / "raw").unlink()
        raw_dir = self.pack / "raw" / "FIGANI"
        raw_dir.mkdir(parents=True)
        for source in (self.source / "raw" / "FIGANI").iterdir():
            (raw_dir / source.name).symlink_to(source)
        target = raw_dir / "FIGANI_000.bin"
        data = bytearray(target.read_bytes())
        data[-1] ^= 1
        target.unlink()
        target.write_bytes(data)
        digest = hashlib.sha256(data).hexdigest()
        next(x for x in self.manifest["source_resources"] if x["source_file"] == "FIGANI.DAT" and x["source_resource"] == 0)["raw_sha256"] = digest
        next(x for x in self.manifest["assets"] if x["asset_id"] == "metadata/raw/figani/figani_000.bin")["sha256"] = digest
        with self.assertRaisesRegex(ValueError, "重組後不等於固定"):
            self.run_audit()


if __name__ == "__main__":
    unittest.main()
