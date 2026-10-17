"""發行工具的權利／來源邊界與跨平台 ZIP 回歸，不是遊戲 parity。"""
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock
import zipfile

SCRIPT = Path(__file__).resolve().parents[1] / "remake/packaging/release.py"
SPEC = importlib.util.spec_from_file_location("fd2_release", SCRIPT)
release = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(release)


class ReleasePackaging(unittest.TestCase):
    def setUp(self):
        # 這些單元測試驗來源比較與拒收；實際 Git HEAD 由原生流水線驗證。
        patch = mock.patch.object(release, "commit", return_value="1" * 40)
        patch.start()
        self.addCleanup(patch.stop)

    def test_zip_unicode_names_and_executable_permissions_survive(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            source = root / "炎龍騎士團2"
            source.mkdir()
            exe = source / "啟動"
            exe.write_bytes(b"test executable")
            exe.chmod(0o755)
            archive = root / "test.zip"
            release.make_zip(source, archive)
            with zipfile.ZipFile(archive) as zipped:
                item = zipped.getinfo("炎龍騎士團2/啟動")
                self.assertTrue(item.flag_bits & 0x800)
                self.assertEqual((item.external_attr >> 16) & 0o777, 0o755)
                self.assertEqual(zipped.read(item), exe.read_bytes())
            with self.assertRaisesRegex(ValueError, "不覆寫"):
                release.make_zip(source, archive)

    def test_invalid_or_mismatched_versions_are_rejected(self):
        with tempfile.TemporaryDirectory() as temporary, mock.patch.object(
                release, "ROOT", Path(temporary)), mock.patch.dict(
                    release.os.environ, {"GITHUB_REF": ""}):
            path = Path(temporary) / "VERSION"
            for invalid in ("1.0.0", "v1.0.0-20261017",
                            "v.1.0.0-20260230", "v.1.0.0-20261017-r2",
                            "v.１.0.0-20261017"):
                path.write_text(invalid)
                with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                    release.version()
            path.write_text("v.1.0.0-20261017")
            with mock.patch.dict(release.os.environ, {"GITHUB_REF": "refs/tags/v.1.0.1-20261017"}):
                with self.assertRaisesRegex(ValueError, "tag"):
                    release.version()

    def fixture(self, root):
        for platform in release.PLATFORMS:
            directory = root / platform
            directory.mkdir()
            suffix = ".AppImage" if platform.startswith("linux") else ".zip"
            name = f"FD2-{release.version()}-{platform}{suffix}"
            package = directory / name
            package.write_bytes(platform.encode())
            release.write_json(directory / f"{platform}-smoke.json", {
                "version": release.version(), "source_commit": release.commit(),
                "platform": platform, "status": "passed", "distribution": "engine-only",
                "package": name, "size": package.stat().st_size, "sha256": release.sha(package)})

    def test_collect_requires_three_same_source_packages(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            downloads = root / "downloads"
            downloads.mkdir()
            self.fixture(downloads)
            release.collect(downloads, root / "out")
            manifest = json.loads((root / "out/SHA256SUMS.json").read_text())
            self.assertEqual(len(manifest["files"]), 3)
            self.assertEqual(manifest["source_commit"], release.commit())
            for row in manifest["files"]:
                self.assertEqual(release.sha(root / "out" / row["file"]), row["sha256"])

    def test_collect_rejects_modified_package_after_attestation(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.fixture(root)
            next(root.rglob("*.AppImage")).write_bytes(b"unapproved replacement")
            with self.assertRaisesRegex(ValueError, "雜湊"):
                release.collect(root, root / "out")

    def test_collect_rejects_foreign_source_commit(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.fixture(root)
            receipt = next(root.rglob("linux-x86_64-smoke.json"))
            data = json.loads(receipt.read_text())
            data["source_commit"] = "0" * 40
            release.write_json(receipt, data)
            with self.assertRaisesRegex(ValueError, "來源"):
                release.collect(root, root / "out")

    def test_collect_rejects_missing_native_receipt(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            self.fixture(root)
            next(root.rglob("macos-universal-smoke.json")).unlink()
            with self.assertRaisesRegex(ValueError, "macos-universal"):
                release.collect(root, root / "out")


if __name__ == "__main__":
    unittest.main()
