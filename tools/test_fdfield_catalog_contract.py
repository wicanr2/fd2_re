import hashlib
import json
import struct
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from fdfield_catalog_contract import validate_fdfield_assets


class FdfieldCatalogContractTest(unittest.TestCase):
    def make_fixture(self, root: Path) -> tuple[Path, dict, str]:
        runtime = root / "runtime"
        map_path = runtime / "maps" / "map7" / "map.json"
        map_path.parent.mkdir(parents=True)
        map_document = {
            "map": 7,
            "w": 2,
            "h": 2,
            "tiles": [1, 2, 3, 4],
            "native_composition_event_bytes": [0, 5, 0, 255],
            "native_tile_blit_modes": [0, 1, 0, 2],
        }
        map_bytes = json.dumps(map_document, separators=(",", ":")).encode("utf-8")
        map_path.write_bytes(map_bytes)

        raw = struct.pack("<HH", 2, 2) + b"\x01\x00\x00\x00\x02\x00\x05\x01\x03\x00\x00\x00\x04\x00\xff\x02"
        source_data = b"synthetic FDFIELD source"
        source = {
            "file": "FDFIELD.DAT",
            "size": len(source_data),
            "md5": hashlib.md5(source_data).hexdigest(),
            "sha256": hashlib.sha256(source_data).hexdigest(),
        }
        catalog = {
            "schema_version": 1,
            "kind": "fd2_fdfield_runtime_catalog",
            "source": source,
            "resources": [{
                "resource_index": 7,
                "map_id": "map7",
                "path": "maps/map7/map.json",
                "file_bytes": len(map_bytes),
                "file_sha256": hashlib.sha256(map_bytes).hexdigest(),
                "raw_bytes": len(raw),
                "raw_sha256": hashlib.sha256(raw).hexdigest(),
                "evidence_level": "confirmed",
                "evidence": "docs/data/ida/synthetic.txt",
            }],
        }
        catalog_path = runtime / "maps" / "fdfield_catalog.json"
        catalog_bytes = json.dumps(catalog, separators=(",", ":")).encode("utf-8")
        catalog_path.write_bytes(catalog_bytes)
        return runtime, source, hashlib.sha256(catalog_bytes).hexdigest()

    def test_valid_synthetic_catalog_does_not_need_original_fdf_field_dat(self):
        with tempfile.TemporaryDirectory() as temp:
            runtime, source, catalog_sha256 = self.make_fixture(Path(temp))
            self.assertFalse((runtime / "FDFIELD.DAT").exists())
            bridge, resources, errors = validate_fdfield_assets(
                runtime, source, catalog_sha256,
            )
            self.assertEqual(errors, [])
            self.assertEqual(bridge["resources"], 1)
            self.assertEqual(resources[7]["map_id"], "map7")

    def test_rejects_tampered_map_file(self):
        with tempfile.TemporaryDirectory() as temp:
            runtime, source, catalog_sha256 = self.make_fixture(Path(temp))
            map_path = runtime / "maps" / "map7" / "map.json"
            map_path.write_bytes(map_path.read_bytes() + b"\n")
            _, _, errors = validate_fdfield_assets(runtime, source, catalog_sha256)
            self.assertTrue(any("map JSON SHA-256 不符" in error for error in errors))

    def test_rejects_array_length_mismatch(self):
        with tempfile.TemporaryDirectory() as temp:
            runtime, source, catalog_sha256 = self.make_fixture(Path(temp))
            map_path = runtime / "maps" / "map7" / "map.json"
            document = json.loads(map_path.read_text(encoding="utf-8"))
            document["native_tile_blit_modes"].pop()
            map_path.write_text(json.dumps(document), encoding="utf-8")
            _, _, errors = validate_fdfield_assets(runtime, source, catalog_sha256)
            self.assertTrue(any("組合格陣列長度不符" in error for error in errors))

    def test_rejects_wrong_reconstructed_raw_sha256(self):
        with tempfile.TemporaryDirectory() as temp:
            runtime, source, catalog_sha256 = self.make_fixture(Path(temp))
            catalog_path = runtime / "maps" / "fdfield_catalog.json"
            catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
            catalog["resources"][0]["raw_sha256"] = "0" * 64
            catalog_path.write_text(json.dumps(catalog), encoding="utf-8")
            _, _, errors = validate_fdfield_assets(runtime, source)
            self.assertTrue(any("重建 raw SHA-256 不符" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
