import hashlib
import json
from pathlib import Path
import tempfile
import unittest

from validate_separated_asset_pack import validate
from generate_separated_asset_manifest import build_coverage_summary


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
            "schema_version": 2,
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
            "source_resources": [],
            "relationships": [],
            "generated_by": {"tool": "test", "version": "1"},
        }
        raw = root / "raw" / "FDOTHER" / "FDOTHER_005.bin"
        raw.parent.mkdir(parents=True)
        raw.write_bytes(b"raw")
        raw_asset = {
            "asset_id": "metadata/raw/fdother/fdother_005.bin",
            "kind": "metadata",
            "path": "raw/FDOTHER/FDOTHER_005.bin",
            "sha256": hashlib.sha256(raw.read_bytes()).hexdigest(),
            "source_file": source.name,
            "source_resource": 5,
            "status": "intentionally_raw",
            "evidence": "confirmed",
        }
        manifest["assets"].append(raw_asset)
        manifest["source_resources"] = [{
            "source_file": source.name,
            "source_resource": 5,
            "raw_asset_id": raw_asset["asset_id"],
            "raw_bytes": raw.stat().st_size,
            "raw_sha256": raw_asset["sha256"],
            "disposition": "standardized",
            "output_asset_ids": ["ui/frame"],
            "runtime_catalog_refs": [],
            "reason_code": "standard_output_assets",
        }]
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

    def test_rejects_changed_raw_resource(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            (root / "raw" / "FDOTHER" / "FDOTHER_005.bin").write_bytes(b"bad")
            self.assertTrue(any("raw identity" in error for error in validate(manifest)))

    def test_blocked_asset_does_not_invent_output_hash(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            doc = json.loads(manifest.read_text())
            doc["assets"][0].update({"status": "blocked", "path": "blocked/ui-frame"})
            doc["assets"][0].pop("sha256")
            doc["source_resources"][0].update({
                "disposition": "blocked",
                "reason_code": "decoder_blocked",
            })
            manifest.write_text(json.dumps(doc), encoding="utf-8")
            self.assertEqual(validate(manifest), [])

    def test_rejects_missing_or_forged_source_resource_ledger(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            doc = json.loads(manifest.read_text())
            valid_doc = json.loads(manifest.read_text())
            doc["source_resources"] = []
            manifest.write_text(json.dumps(doc), encoding="utf-8")
            self.assertTrue(any("缺少 1 筆" in error for error in validate(manifest)))

            doc = valid_doc
            doc["source_resources"][0]["output_asset_ids"] = ["metadata/raw/fdother/fdother_005.bin"]
            manifest.write_text(json.dumps(doc), encoding="utf-8")
            self.assertTrue(any("無法證明關聯" in error for error in validate(manifest)))

    def test_coverage_summary_is_bound_to_full_manifest(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            manifest = self.make_pack(root)
            doc = json.loads(manifest.read_text())
            summary = root / "coverage.json"
            summary.write_text(json.dumps(build_coverage_summary(
                doc, hashlib.sha256(manifest.read_bytes()).hexdigest(),
            )), encoding="utf-8")
            self.assertEqual(validate(manifest, coverage_summary=summary), [])
            changed = json.loads(summary.read_text())
            changed["dispositions"]["unknown"] = 1
            summary.write_text(json.dumps(changed), encoding="utf-8")
            self.assertTrue(any("摘要與 manifest 不符" in error for error in validate(
                manifest, coverage_summary=summary,
            )))


if __name__ == "__main__":
    unittest.main()
