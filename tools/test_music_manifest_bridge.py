import hashlib
import json
import struct
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from generate_music_catalog import TRACKS, inspect_vorbis
from generate_separated_asset_manifest import build_manifest
from validate_separated_asset_pack import validate


def ogg_page(packet: bytes, granule: int) -> bytes:
    lacing = []
    remaining = len(packet)
    while remaining >= 255:
        lacing.append(255)
        remaining -= 255
    lacing.append(remaining)
    return b"OggS\x00\x00" + struct.pack("<QIIIB", granule, 1, 0, 0, len(lacing)) + bytes(lacing) + packet


def write_vorbis(path: Path, samples: int = 96000) -> None:
    identification = b"\x01vorbis" + struct.pack("<IBI", 0, 2, 48000) + bytes(14)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(ogg_page(identification, 0) + ogg_page(b"\x03vorbis", samples))


class MusicManifestBridgeTest(unittest.TestCase):
    def make_fixture(self, root: Path) -> tuple[Path, Path, Path]:
        pack = root / "pack"
        (pack / "music").mkdir(parents=True)
        (pack / "music" / "FDMUS_001.mid").write_bytes(b"midi")
        source_data = b"synthetic-fdmus"
        source = {
            "file": "FDMUS.DAT",
            "size": len(source_data),
            "md5": hashlib.md5(source_data).hexdigest(),
            "sha256": hashlib.sha256(source_data).hexdigest(),
        }
        reference = root / "reference.json"
        reference.write_text(json.dumps({"files": [source]}), encoding="utf-8")
        assets = root / "runtime-assets"
        tracks = []
        for resource in TRACKS:
            track_id = f"FDMUS_{resource:03d}"
            renders = {}
            for profile in ("fm", "mt32"):
                relative = f"music_{profile}/{track_id}.ogg"
                output = assets / relative
                write_vorbis(output, 96000 + resource)
                geometry = inspect_vorbis(output)
                geometry.pop("has_loop_tags")
                renders[profile] = {
                    "path": relative,
                    "bytes": output.stat().st_size,
                    "sha256": hashlib.sha256(output.read_bytes()).hexdigest(),
                    "codec": "vorbis",
                    **geometry,
                }
            tracks.append({"track_id": track_id, "resource_index": resource, "renders": renders})
        catalog = {
            "schema": "fd2_music_catalog",
            "schema_version": 1,
            "source": source,
            "loop": {
                "mode": "whole_file_runtime_repeat",
                "accepted_counts": [0, 1],
                "seam_evidence": "unknown",
                "evidence_level": "strong_inference_e1",
            },
            "profiles": {
                profile: {
                    "render_pipeline": "synthetic test",
                    "provenance_status": "incomplete_legacy_render",
                    "rights_note": "test only",
                }
                for profile in ("fm", "mt32")
            },
            "tracks": tracks,
        }
        (assets / "music_catalog.json").write_text(
            json.dumps(catalog, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
        )
        return pack, reference, assets

    def test_bridge_validates_external_renders_without_copying_them(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, reference, music_assets_root=assets)
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            self.assertEqual(validate(manifest_path, runtime_assets=assets), [])
            bridge = manifest["runtime_catalogs"]["music"]
            self.assertEqual((bridge["profiles"], bridge["tracks"], bridge["renders"]), (2, 15, 30))
            self.assertFalse(any(path.suffix == ".ogg" for path in pack.rglob("*")))
            midi = next(item for item in manifest["assets"] if item["path"].endswith(".mid"))
            self.assertEqual(midi["status"], "blocked")

    def test_bridge_requires_explicit_runtime_root(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(
                json.dumps(build_manifest(pack, reference, music_assets_root=assets)), encoding="utf-8"
            )
            self.assertTrue(any("明確 runtime assets root" in error for error in validate(manifest_path)))

    def test_bridge_rejects_catalog_and_render_drift(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(
                json.dumps(build_manifest(pack, reference, music_assets_root=assets)), encoding="utf-8"
            )
            (assets / "music_fm" / "FDMUS_001.ogg").write_bytes(b"changed")
            self.assertTrue(any("FDMUS_001.ogg" in error for error in validate(manifest_path, runtime_assets=assets)))

    def test_bridge_rejects_catalog_hash_mutation(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, reference, music_assets_root=assets)
            manifest["runtime_catalogs"]["music"]["catalog_sha256"] = "0" * 64
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            errors = validate(manifest_path, runtime_assets=assets)
            self.assertTrue(any("catalog SHA-256" in error for error in errors))

    def test_bridge_rejects_catalog_path_traversal(self):
        with tempfile.TemporaryDirectory() as temp:
            pack, reference, assets = self.make_fixture(Path(temp))
            manifest = build_manifest(pack, reference, music_assets_root=assets)
            manifest["runtime_catalogs"]["music"]["catalog_path"] = "../music_catalog.json"
            manifest_path = pack / "manifest.json"
            manifest_path.write_text(json.dumps(manifest), encoding="utf-8")
            errors = validate(manifest_path, runtime_assets=assets)
            self.assertTrue(any("固定契約" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
