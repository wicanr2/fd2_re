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
        (pack / "animations" / "FIGANI_003" / "animation.json").write_text("{}", encoding="utf-8")
        (pack / "animations" / "FIGANI_003" / "resource.json").write_text("{}", encoding="utf-8")
        (pack / "animations" / "FDOTHER_018").mkdir(parents=True)
        (pack / "animations" / "FDOTHER_018" / "frame_000.png").write_bytes(b"fdother-frame")
        (pack / "animations" / "FDOTHER_018" / "animation.json").write_text("{}", encoding="utf-8")
        (pack / "animations" / "FDOTHER_018" / "resource.json").write_text("{}", encoding="utf-8")
        (pack / "raw" / "FIGANI" / "FIGANI_003.bin").write_bytes(b"raw")
        (pack / "raw" / "FDMUS" / "FDMUS_000.bin").write_bytes(b"music-raw")
        (pack / "music").mkdir()
        (pack / "music" / "FDMUS_000.mid").write_bytes(b"midi")
        (pack / "palette").mkdir()
        (pack / "palette" / "fdother_000.json").write_text("{}")
        (pack / "ui" / "action_cells").mkdir(parents=True)
        (pack / "ui" / "action_cells" / "cell_000.png").write_bytes(b"cell")
        (pack / "surfaces" / "BG_056").mkdir(parents=True)
        (pack / "surfaces" / "BG_056" / "resource.json").write_text(
            json.dumps({"status": "blocked"}), encoding="utf-8")
        (pack / "text" / "FDTXT_000").mkdir(parents=True)
        (pack / "text" / "FDTXT_000" / "resource.json").write_text(
            json.dumps({"status": "decoded"}), encoding="utf-8")
        (pack / "fonts" / "fdother_004").mkdir(parents=True)
        (pack / "fonts" / "fdother_004" / "atlas.png").write_bytes(b"font-atlas")
        (pack / "fonts" / "fdother_004" / "font.json").write_text("{}", encoding="utf-8")
        original = root / "original"
        original.mkdir()
        source = original / "FIGANI.DAT"
        source.write_bytes(b"original")
        music_source = original / "FDMUS.DAT"
        music_source.write_bytes(b"music-original")
        ui_source = original / "FDOTHER.DAT"
        ui_source.write_bytes(b"ui-original")
        bg_source = original / "BG.DAT"
        bg_source.write_bytes(b"bg-original")
        text_source = original / "FDTXT.DAT"
        text_source.write_bytes(b"text-original")
        ref = root / "reference.json"
        data = source.read_bytes()
        music_data = music_source.read_bytes()
        ui_data = ui_source.read_bytes()
        bg_data = bg_source.read_bytes()
        text_data = text_source.read_bytes()
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
        }, {
            "file": "BG.DAT", "size": len(bg_data),
            "md5": hashlib.md5(bg_data).hexdigest(),
            "sha256": hashlib.sha256(bg_data).hexdigest(),
        }, {
            "file": "FDTXT.DAT", "size": len(text_data),
            "md5": hashlib.md5(text_data).hexdigest(),
            "sha256": hashlib.sha256(text_data).hexdigest(),
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
            self.assertIn("metadata/animations/figani_003/animation.json", ids)
            self.assertIn("metadata/animations/figani_003/resource.json", ids)
            embedded = next(asset for asset in manifest["assets"] if asset["path"] == "animations/FDOTHER_018/animation.json")
            self.assertEqual((embedded["source_file"], embedded["source_resource"]), ("FDOTHER.DAT", 18))
            figani = next(asset for asset in manifest["assets"] if asset["path"] == "animations/FIGANI_003/animation.json")
            self.assertEqual((figani["source_file"], figani["source_resource"]), ("FIGANI.DAT", 3))
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
            blocked_surface = next(item for item in manifest["assets"] if item["path"] == "surfaces/BG_056/resource.json")
            self.assertEqual((blocked_surface["source_file"], blocked_surface["source_resource"], blocked_surface["status"]), ("BG.DAT", 56, "blocked"))
            text = next(item for item in manifest["assets"] if item["path"] == "text/FDTXT_000/resource.json")
            self.assertEqual((text["kind"], text["source_file"], text["source_resource"], text["status"]), ("text", "FDTXT.DAT", 0, "exported"))
            font = next(item for item in manifest["assets"] if item["path"] == "fonts/fdother_004/font.json")
            self.assertEqual((font["kind"], font["source_file"], font["source_resource"]), ("font", "FDOTHER.DAT", 4))

    def test_source_hash_mismatch_fails_closed(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, ref, original = self.make_fixture(Path(temp))
            (original / "FIGANI.DAT").write_bytes(b"changed")
            with self.assertRaises(ValueError):
                build_manifest(pack, ref, original)


if __name__ == "__main__":
    unittest.main()
