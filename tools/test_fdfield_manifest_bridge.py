import hashlib
import json
import struct
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from generate_separated_asset_manifest import build_manifest
from validate_separated_asset_pack import validate


class FDFIELDManifestBridgeTest(unittest.TestCase):
    def make_fixture(self, root: Path) -> tuple[Path, Path, Path]:
        pack = root / "pack"
        raw = struct.pack("<HHHBBHBB", 2, 1, 3, 4, 5, 6, 7, 8)
        raw_path = pack / "raw" / "FDFIELD" / "FDFIELD_069.bin"
        raw_path.parent.mkdir(parents=True)
        raw_path.write_bytes(raw)

        source_data = b"synthetic-fdfield"
        source = {
            "file": "FDFIELD.DAT",
            "size": len(source_data),
            "md5": hashlib.md5(source_data).hexdigest(),
            "sha256": hashlib.sha256(source_data).hexdigest(),
        }
        reference = root / "reference.json"
        reference.write_text(json.dumps({"files": [source]}), encoding="utf-8")

        assets = root / "runtime-assets"
        map_path = assets / "maps" / "map23" / "map.json"
        map_path.parent.mkdir(parents=True)
        map_path.write_text(json.dumps({
            "w": 2,
            "h": 1,
            "tiles": [3, 6],
            "native_composition_event_bytes": [4, 7],
            "native_tile_blit_modes": [5, 8],
        }), encoding="utf-8")
        catalog = {
            "schema_version": 1,
            "kind": "fd2_fdfield_runtime_catalog",
            "source": source,
            "resources": [{
                "resource_index": 69,
                "map_id": "map23",
                "path": "maps/map23/map.json",
                "file_bytes": map_path.stat().st_size,
                "file_sha256": hashlib.sha256(map_path.read_bytes()).hexdigest(),
                "raw_bytes": len(raw),
                "raw_sha256": hashlib.sha256(raw).hexdigest(),
                "evidence_level": "confirmed",
                "evidence": "docs/data/ida/test.txt",
            }],
        }
        catalog_path = assets / "maps" / "fdfield_catalog.json"
        catalog_path.write_text(json.dumps(catalog), encoding="utf-8")
        return pack, reference, assets

    def test_catalog_marks_exact_raw_resource_standardized(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, reference, runtime_assets_root=assets)
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            self.assertEqual(validate(manifest_path, runtime_assets=assets), [])
            entry = manifest["source_resources"][0]
            self.assertEqual(entry["disposition"], "standardized")
            self.assertEqual(entry["runtime_catalog_refs"], ["fdfield/FDFIELD_069"])
            self.assertEqual(manifest["runtime_catalogs"]["fdfield"]["resources"], 1)

    def test_generator_rejects_catalog_raw_identity_mismatch(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            (pack / "raw" / "FDFIELD" / "FDFIELD_069.bin").write_bytes(b"different")
            with self.assertRaisesRegex(ValueError, "raw identity"):
                build_manifest(pack, reference, runtime_assets_root=assets)

    def test_validator_rejects_forged_ledger_identity(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, reference, runtime_assets_root=assets)
            raw_path = pack / "raw" / "FDFIELD" / "FDFIELD_069.bin"
            raw_path.write_bytes(b"different")
            changed_hash = hashlib.sha256(raw_path.read_bytes()).hexdigest()
            raw_asset = next(item for item in manifest["assets"] if item["status"] == "intentionally_raw")
            raw_asset["sha256"] = changed_hash
            ledger = manifest["source_resources"][0]
            ledger["raw_bytes"] = raw_path.stat().st_size
            ledger["raw_sha256"] = changed_hash
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            errors = validate(manifest_path, runtime_assets=assets)
            self.assertTrue(any("runtime_catalog_refs raw identity" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
