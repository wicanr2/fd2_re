import importlib.util
import struct
import tempfile
import unittest
from pathlib import Path


SPEC = importlib.util.spec_from_file_location("music_catalog", Path(__file__).with_name("generate_music_catalog.py"))
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(MODULE)


def page(packet: bytes, granule: int) -> bytes:
    lacing = []
    remain = len(packet)
    while remain >= 255:
        lacing.append(255)
        remain -= 255
    lacing.append(remain)
    header = bytearray(b"OggS\x00\x00")
    header += struct.pack("<QIIIB", granule, 1, 0, 0, len(lacing))
    return bytes(header) + bytes(lacing) + packet


def fake_vorbis(path: Path, sample_rate: int = 48000, samples: int = 96000, comments: bytes = b"") -> None:
    identification = b"\x01vorbis" + struct.pack("<IBI", 0, 2, sample_rate) + bytes(14)
    comment = b"\x03vorbis" + comments
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(page(identification, 0) + page(comment, samples))


class MusicCatalogTest(unittest.TestCase):
    def test_inspect_vorbis_records_exact_geometry(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "track.ogg"
            fake_vorbis(path)
            got = MODULE.inspect_vorbis(path)
            self.assertEqual(got["channels"], 2)
            self.assertEqual(got["sample_rate"], 48000)
            self.assertEqual(got["pcm_samples"], 96000)
            self.assertEqual(got["duration_ms"], 2000)
            self.assertFalse(got["has_loop_tags"])

    def test_build_catalog_requires_every_profile_and_rejects_loop_tags(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            for resource in MODULE.TRACKS:
                for profile in MODULE.PROFILES:
                    fake_vorbis(root / f"music_{profile}" / f"FDMUS_{resource:03d}.ogg")
            catalog = MODULE.build_catalog(root)
            self.assertEqual(len(catalog["tracks"]), 15)
            bad = root / "music_fm" / "FDMUS_001.ogg"
            fake_vorbis(bad, comments=b"LOOPSTART=0")
            with self.assertRaises(MODULE.CatalogError):
                MODULE.build_catalog(root)

    def test_invalid_ogg_fails_closed(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bad.ogg"
            path.write_bytes(b"not ogg")
            with self.assertRaises(MODULE.CatalogError):
                MODULE.inspect_vorbis(path)


if __name__ == "__main__":
    unittest.main()
