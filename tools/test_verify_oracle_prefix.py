import json
import tempfile
import unittest
from pathlib import Path

from verify_oracle_prefix import EXE_SHA256, verify


class PrefixVerificationTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.prefix, self.full = [Path(self.tmp.name) / name for name in ["prefix", "full"]]
        for directory in [self.prefix, self.full]:
            directory.mkdir()
            self.write(directory / "runner.json", {"dosgolem_commit": "fixed-commit", "original_fd2_exe_sha256": EXE_SHA256,
                       "dosgolem_tracked_dirty_files": 0, "dosgolem_untracked_files": 0})
            (directory / "control-history.jsonl").write_text(json.dumps({"seq": 1, "key": "enter", "steps": 20000000}) + "\n")
            for seq in [0, 1]:
                self.write(directory / f"checkpoint-{seq:04d}.json", {"control_seq": seq, "exe_sha256": EXE_SHA256,
                           "registers": [1, 2, 3, 4, 5, 6, 7, 8], "units": [{"raw_hex": "00000300" + "00" * 76}], "view": {"rng_word": 22661},
                           "state_injections": [], "frame_deferred": False, "frames_deferred_so_far": 0,
                           "address_space": "dosgolem relocated LE linear", "dos_file_calls": [], "eip": "0x36D98",
                           "evidence_restrictions": [], "input_chain": [], "input_kind": "normal BIOS keys",
                           "kbd_pending": 0, "kbd_reads": 0, "normal_player_path_verified": False,
                           "runner": "dosgolem", "schema": 1, "steps": seq * 20000000, "unit_base": 100})

    def write(self, path, data):
        path.write_text(json.dumps(data))

    def mutate(self, filename, change):
        path = self.full / filename
        data = json.loads(path.read_text())
        change(data)
        self.write(path, data)

    def test_same_prefix_ignores_only_async_frame_scheduling(self):
        self.mutate("checkpoint-0001.json", lambda data: data.update(frame_deferred=True, frames_deferred_so_far=100))
        self.assertEqual(verify(self.prefix, self.full)["status"], "passed")

    def test_different_raw_record_cannot_hide_behind_same_rng(self):
        self.mutate("checkpoint-0001.json", lambda data: data.update(units=[{"raw_hex": "00" * 80}]))
        self.assertEqual(verify(self.prefix, self.full)["status"], "failed")

    def test_different_keys_cannot_hide_behind_same_checkpoint(self):
        (self.full / "control-history.jsonl").write_text(json.dumps({"seq": 1, "key": "esc", "steps": 20000000}))
        self.assertEqual(verify(self.prefix, self.full)["status"], "failed")

    def test_missing_checkpoint_and_noncontiguous_prefix_fail(self):
        (self.full / "checkpoint-0001.json").unlink()
        self.assertEqual(verify(self.prefix, self.full)["status"], "failed")
        (self.prefix / "checkpoint-0000.json").unlink()
        self.assertEqual(verify(self.prefix, self.prefix)["status"], "failed")

    def test_same_missing_raw_field_is_not_evidence(self):
        for directory in [self.prefix, self.full]:
            path = directory / "checkpoint-0001.json"
            data = json.loads(path.read_text())
            del data["units"]
            self.write(path, data)
        self.assertEqual(verify(self.prefix, self.full)["status"], "failed")

    def test_same_unit_without_full_raw_bytes_is_not_evidence(self):
        for directory in [self.prefix, self.full]:
            path = directory / "checkpoint-0001.json"
            data = json.loads(path.read_text())
            data["units"] = [{"hp": 320, "raw_hex": "00000300"}]
            self.write(path, data)
        self.assertEqual(verify(self.prefix, self.full)["status"], "failed")

    def test_injection_unknown_exe_and_dirty_source_fail(self):
        for filename, change in [
            ("checkpoint-0001.json", {"state_injections": ["force-enemy-clear"]}),
            ("checkpoint-0001.json", {"exe_sha256": "unknown"}),
            ("runner.json", {"dosgolem_tracked_dirty_files": 1}),
        ]:
            with self.subTest(change=change):
                path = self.full / filename
                saved = path.read_text()
                self.mutate(filename, lambda data: data.update(change))
                self.assertEqual(verify(self.prefix, self.full)["status"], "failed")
                path.write_text(saved)


if __name__ == "__main__":
    unittest.main()
