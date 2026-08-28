import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from validate_separated_asset_pack import validate


class SeparatedAssetPackTest(unittest.TestCase):
    def make_pack(self, root: Path) -> Path:
        original = root / "original"
        original.mkdir()
        source = original / "FDOTHER.DAT"
        source.write_bytes(b"source")
        asset = root / "ui" / "frame.png"
        asset.parent.mkdir()
        asset.write_bytes(b"png")
        manifest = {
            "schema_version": 1,
            "pack_id": "fd2-test",
            "source_set": [{
                "file": source.name,
                "size": source.stat().st_size,
                "md5": hashlib.md5(source.read_bytes()).hexdigest(),
                "sha256": hashlib.sha256(source.read_bytes()).hexdigest(),
            }],
            "assets": [{
                "asset_id": "ui/frame",
                "kind": "ui",
                "path": "ui/frame.png",
                "sha256": hashlib.sha256(asset.read_bytes()).hexdigest(),
                "source_file": source.name,
                "source_resource": 5,
                "status": "exported",
                "evidence": "confirmed",
            }],
            "relationships": [],
            "generated_by": {"tool": "test", "version": "1"},
        }
        path = root / "manifest.json"
        path.write_text(json.dumps(manifest), encoding="utf-8")
        return path

    def test_valid_pack_and_source(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            self.assertEqual(validate(manifest, root / "original"), [])

    def test_rejects_duplicate_id_and_missing_relationship(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            doc = json.loads(manifest.read_text())
            doc["assets"].append(dict(doc["assets"][0]))
            doc["relationships"] = [{"from": "ui/frame", "type": "uses", "to": "missing"}]
            manifest.write_text(json.dumps(doc), encoding="utf-8")
            errors = validate(manifest)
            self.assertTrue(any("重複 asset_id" in error for error in errors))
            self.assertTrue(any("引用不存在" in error for error in errors))

    def test_rejects_parent_path_and_changed_output(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            doc = json.loads(manifest.read_text())
            doc["assets"][0]["path"] = "../outside.png"
            manifest.write_text(json.dumps(doc), encoding="utf-8")
            self.assertTrue(any("安全相對路徑" in error for error in validate(manifest)))


if __name__ == "__main__":
    unittest.main()
