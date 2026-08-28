import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from generate_separated_asset_manifest import build_manifest
from validate_separated_asset_pack import validate


class GenerateManifestTest(unittest.TestCase):
    def make_fixture(self, root: Path) -> tuple[Path, Path, Path]:
        pack = root / "fd2-original-test"
        (pack / "images").mkdir(parents=True)
        (pack / "raw" / "FIGANI").mkdir(parents=True)
        (pack / "raw" / "FDMUS").mkdir(parents=True)
        (pack / "images" / "FIGANI_FIGANI_003.png").write_bytes(b"png")
        (pack / "animations" / "FIGANI_003").mkdir(parents=True)
        (pack / "animations" / "FIGANI_003" / "frame_000.png").write_bytes(b"frame")
        (pack / "raw" / "FIGANI" / "FIGANI_003.bin").write_bytes(b"raw")
        (pack / "raw" / "FDMUS" / "FDMUS_000.bin").write_bytes(b"music-raw")
        (pack / "music").mkdir()
        (pack / "music" / "FDMUS_000.mid").write_bytes(b"midi")
        (pack / "palette").mkdir()
        (pack / "palette" / "fdother_000.json").write_text("{}")
        (pack / "ui" / "action_cells").mkdir(parents=True)
        (pack / "ui" / "action_cells" / "cell_000.png").write_bytes(b"cell")
        original = root / "original"
        original.mkdir()
        source = original / "FIGANI.DAT"
        source.write_bytes(b"original")
        music_source = original / "FDMUS.DAT"
        music_source.write_bytes(b"music-original")
        ui_source = original / "FDOTHER.DAT"
        ui_source.write_bytes(b"ui-original")
        ref = root / "reference.json"
        data = source.read_bytes()
        music_data = music_source.read_bytes()
        ui_data = ui_source.read_bytes()
        ref.write_text(json.dumps({"files": [{
            "file": "FIGANI.DAT", "size": len(data),
            "md5": hashlib.md5(data).hexdigest(),
            "sha256": hashlib.sha256(data).hexdigest(),
        }, {
            "file": "FDMUS.DAT", "size": len(music_data),
            "md5": hashlib.md5(music_data).hexdigest(),
            "sha256": hashlib.sha256(music_data).hexdigest(),
        }, {
            "file": "FDOTHER.DAT", "size": len(ui_data),
            "md5": hashlib.md5(ui_data).hexdigest(),
            "sha256": hashlib.sha256(ui_data).hexdigest(),
        }]}), encoding="utf-8")
        return pack, ref, original

    def test_stable_assets_and_explicit_raw_inventory(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, ref, original = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, ref, original)
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            self.assertEqual(validate(manifest_path, original), [])
            ids = {asset["asset_id"] for asset in manifest["assets"]}
            self.assertIn("battle_animation/animations/figani_003/frame_000.png", ids)
            self.assertEqual(manifest["assets"][0]["source_file"], "FIGANI.DAT")
            raw = [asset for asset in manifest["assets"] if asset["path"].startswith("raw/")]
            self.assertEqual(len(raw), 2)
            self.assertTrue(all(item["status"] == "intentionally_raw" for item in raw))
            self.assertTrue(all(item["kind"] == "metadata" for item in raw))
            midi = next(item for item in manifest["assets"] if item["path"].endswith(".mid"))
            self.assertEqual(midi["kind"], "music")
            self.assertEqual(midi["status"], "blocked")
            self.assertEqual(midi["source_file"], "FDMUS.DAT")
            palette = next(item for item in manifest["assets"] if item["path"].startswith("palette/"))
            self.assertEqual((palette["source_file"], palette["source_resource"]), ("FDOTHER.DAT", 0))
            cell = next(item for item in manifest["assets"] if item["path"].startswith("ui/action_cells/"))
            self.assertEqual((cell["source_file"], cell["source_resource"], cell["source_frame"]), ("FDOTHER.DAT", 2, 0))

    def test_source_hash_mismatch_fails_closed(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, ref, original = self.make_fixture(Path(temp))
            (original / "FIGANI.DAT").write_bytes(b"changed")
            with self.assertRaises(ValueError):
                build_manifest(pack, ref, original)


if __name__ == "__main__":
    unittest.main()
