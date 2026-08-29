import hashlib
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from generate_separated_asset_manifest import build_manifest


class FDOTHERManifestAdmissionTest(unittest.TestCase):
    def make_fixture(self, root: Path) -> tuple[Path, Path]:
        pack = root / "pack"
        (pack / "raw" / "FDOTHER").mkdir(parents=True)
        (pack / "raw" / "FDOTHER" / "FDOTHER_001.bin").write_bytes(b"raw-one")
        (pack / "raw" / "FDOTHER" / "FDOTHER_006.bin").write_bytes(b"raw-six")

        overlay = pack / "sprites" / "fdother_001_range_overlay"
        (overlay / "sprite_0000").mkdir(parents=True)
        (overlay / "bank.json").write_text("{}", encoding="utf-8")
        for name in ("frame.png", "mask.png", "remap.png"):
            (overlay / "sprite_0000" / name).write_bytes(b"not-a-real-png")

        effect = pack / "effects" / "fdother_006_lmi1_opaque"
        (effect / "entry_000").mkdir(parents=True)
        (effect / "bank.json").write_text("{}", encoding="utf-8")
        (effect / "entry_000" / "frame.png").write_bytes(b"also-not-a-real-png")

        # This similarly named path is intentionally outside the approved
        # FDOTHER admission list and must not become a manifest asset.
        unapproved = pack / "effects" / "fdother_007"
        (unapproved / "entry_000").mkdir(parents=True)
        (unapproved / "bank.json").write_text("{}", encoding="utf-8")
        (unapproved / "entry_000" / "frame.png").write_bytes(b"unapproved")

        original = root / "original"
        original.mkdir()
        source = original / "FDOTHER.DAT"
        source.write_bytes(b"synthetic-fdother")
        data = source.read_bytes()
        reference = root / "reference.json"
        reference.write_text(json.dumps({"files": [{
            "file": "FDOTHER.DAT",
            "size": len(data),
            "md5": hashlib.md5(data).hexdigest(),
            "sha256": hashlib.sha256(data).hexdigest(),
        }]}), encoding="utf-8")
        return pack, reference

    def test_only_approved_fdother_paths_enter_manifest(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, reference, pack.parent / "original")

            assets = {
                item["path"]: item
                for item in manifest["assets"]
                if not item["path"].startswith("raw/")
            }
            expected = {
                "sprites/fdother_001_range_overlay/bank.json": ("metadata", 1, None),
                "sprites/fdother_001_range_overlay/sprite_0000/frame.png": ("map_sprite", 1, 0),
                "sprites/fdother_001_range_overlay/sprite_0000/mask.png": ("map_sprite", 1, 0),
                "sprites/fdother_001_range_overlay/sprite_0000/remap.png": ("map_sprite", 1, 0),
                "effects/fdother_006_lmi1_opaque/bank.json": ("metadata", 6, None),
                "effects/fdother_006_lmi1_opaque/entry_000/frame.png": ("battle_animation", 6, 0),
            }
            self.assertEqual(set(assets), set(expected))
            for path, (kind, resource, frame) in expected.items():
                item = assets[path]
                self.assertEqual(
                    (item["kind"], item["source_resource"], item.get("source_frame")),
                    (kind, resource, frame),
                )
                self.assertEqual(item["source_file"], "FDOTHER.DAT")

            ledger = {
                item["source_resource"]: item
                for item in manifest["source_resources"]
            }
            self.assertEqual(set(ledger), {1, 6})
            self.assertEqual(ledger[1]["disposition"], "standardized")
            self.assertEqual(ledger[6]["disposition"], "standardized")
            self.assertTrue(all(item["output_asset_ids"] for item in ledger.values()))


if __name__ == "__main__":
    unittest.main()
