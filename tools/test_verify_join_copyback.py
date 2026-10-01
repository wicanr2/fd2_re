import json
from pathlib import Path
import tempfile
import unittest

import fd2save
import verify_join_copyback as verifier


class JoinCopybackVerificationTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.original, self.state, self.builder = [self.root / name for name in ("original", "state", "builder")]
        for directory in (self.original, self.state, self.builder):
            directory.mkdir()
        self.base = self.root / "base.SAV"
        plain = bytearray(fd2save.FILE_SIZE)
        start, _ = fd2save.slot_bounds(0)
        plain[start + fd2save.ROSTER_SIZE:start + fd2save.ROSTER_SIZE + 2] = bytes([1, 1])
        self.base.write_bytes(fd2save.encode(plain))
        plain[start + fd2save.ROSTER_SIZE:start + fd2save.ROSTER_SIZE + 2] = bytes([2, 2])
        plain[start + fd2save.UNIT_SIZE + 8] = 8
        plain[start + fd2save.UNIT_SIZE + 0x16:start + fd2save.UNIT_SIZE + 0x1a] = bytes([128, 255, 128, 255])
        for directory in (self.state, self.builder):
            (directory / "FD2.SAV").write_bytes(fd2save.encode(plain))
        self.write(self.original / "runner.json", {
            "runner": "dosgolem apps/fd2/cmd/oracle", "dosgolem_tracked_dirty_files": 0,
            "dosgolem_commit": "synthetic_unit_fixture", "original_fd2_exe_sha256": verifier.EXE_SHA,
            "force_enemy_clear_declared": True, "lock_ally_hp": False, "control_plan": verifier.PLAN,
        })
        self.write(self.original / "checkpoint-0000.json", {"view": {"rng_word": 22661}})
        self.write(self.original / "current.json", {"exe_sha256": verifier.EXE_SHA,
                   "state_injections": ["force-enemy-clear synthetic fixture"]})
        self.actions = [
            {"kind": "mark", "label": "town_loaded", "seq": 1},
            {"kind": "mark", "label": "battle_start", "seq": 2},
            {"kind": "force_enemy_clear", "seq": 3, "modified_path": True, "cleared": 10},
            {"kind": "mark", "label": "town_after_battle", "seq": 4},
            {"kind": "town_save", "slot": 0, "seq": 5, "save_sha256": verifier.sha(self.state / "FD2.SAV")},
        ]
        self.trace = [
            {"eip": "0x10F48", "eax": "0x4", "edx": "0x1000", "edi": "0x1008", "ebx": "0xff", "step": 20},
            {"eip": "0x10F48", "eax": "0x5", "edx": "0x1000", "edi": "0x100a", "ebx": "0xff", "step": 30},
            {"eip": "0x230D9", "stack": ["0x8"], "step": 40},
            {"eip": "0x230E1", "step": 50},
            {"eip": "0x11576", "eax": "0x8", "edx": "0x8", "edi": "0x1000", "ebx": "0x2000",
             "stack": ["0x2000", "0x1000", "0x50"], "step": 60},
        ]
        self.manifest = {
            "base_sha256": verifier.sha(self.base), "output_sha256": verifier.sha(self.builder / "FD2.SAV"),
            "evidence_sources": {verifier.MAP_SOURCE: verifier.sha(verifier.ROOT / "remake/assets" / verifier.MAP_SOURCE)},
            "seed": 2, "level_steps": None,
            "applied": [{"op": "join", "char_id": 8, "result": "appended"},
                        {"op": "sync_join_item_tail", "char_id": 8, "source": {"addr": "0x230e1"}}],
        }

    @staticmethod
    def write(path, data):
        path.write_text(json.dumps(data), encoding="utf-8")

    def result(self):
        for name, rows in (("actions.jsonl", self.actions), ("eip-trace.jsonl", self.trace)):
            (self.original / name).write_text("\n".join(json.dumps(row) for row in rows) + "\n")
        self.write(self.builder / "manifest.json", self.manifest)
        return verifier.verify(self.original, self.state, self.builder, self.base)

    def test_complete_synthetic_contract_with_null_growth_steps(self):
        self.assertEqual(self.result()["status"], "passed")

    def test_ff_without_both_writers_cannot_pass(self):
        self.trace.pop(0)
        self.assertFalse(self.result()["gates"]["writer_join_copy_witness"])

    def test_copy_from_another_runtime_record_cannot_pass(self):
        self.trace[-1]["edi"] = "0x3000"
        self.assertFalse(self.result()["gates"]["writer_join_copy_witness"])

    def test_partial_copy_cannot_pass(self):
        self.trace[-1]["stack"][2] = "0x10"
        self.assertFalse(self.result()["gates"]["writer_join_copy_witness"])

    def test_sync_before_join_cannot_pass(self):
        self.manifest["applied"].reverse()
        self.assertFalse(self.result()["gates"]["builder_join_before_sync"])

    def test_missing_sync_cannot_pass(self):
        self.manifest["applied"].pop()
        self.assertFalse(self.result()["gates"]["builder_join_before_sync"])

    def test_missing_post_endpoint_cannot_pass(self):
        self.actions = [row for row in self.actions if row.get("label") != "town_after_battle"]
        self.assertFalse(self.result()["gates"]["planned_load_battle_post_save"])

    def test_modified_path_must_be_declared(self):
        runner = verifier.load(self.original / "runner.json")
        runner["force_enemy_clear_declared"] = False
        self.write(self.original / "runner.json", runner)
        self.assertFalse(self.result()["gates"]["declared_modified_path"])

    def test_missing_growth_metadata_cannot_pass(self):
        del self.manifest["level_steps"]
        self.assertFalse(self.result()["gates"]["builder_no_growth_draws"])

    def test_tail_difference_cannot_pass_even_with_valid_checksum(self):
        plain = bytearray(fd2save.decode((self.builder / "FD2.SAV").read_bytes()))
        plain[fd2save.SLOT_OFFSET + fd2save.UNIT_SIZE + 0x17] = 0
        (self.builder / "FD2.SAV").write_bytes(fd2save.encode(plain))
        self.manifest["output_sha256"] = verifier.sha(self.builder / "FD2.SAV")
        self.assertFalse(self.result()["gates"]["builder_save_tail"])


if __name__ == "__main__":
    unittest.main()
