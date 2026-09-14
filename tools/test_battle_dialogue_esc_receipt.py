import hashlib
import json
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]
RECEIPT = ROOT / "docs/data/ui-traces/battle-dialogue-esc-vs-enter.json"


class BattleDialogueEscReceipt(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.receipt = json.loads(RECEIPT.read_text(encoding="utf-8"))

    def test_fixed_executable_and_clean_dosgolem_runner(self):
        self.assertEqual(self.receipt["original_runner"], "dosgolem apps/fd2/cmd/oracle")
        self.assertTrue(self.receipt["runner_clean"])
        self.assertEqual(
            self.receipt["executable"]["sha256"],
            "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f",
        )
        self.assertEqual(self.receipt["method"]["state_injections"], [])

    def test_three_arm_plans_are_versioned_and_hash_locked(self):
        for case in self.receipt["cases"]:
            self.assertEqual(set(case["plans"]), {"esc", "enter", "none"})
            for plan in case["plans"].values():
                path = ROOT / plan["file"]
                self.assertTrue(path.is_file(), path)
                self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), plan["sha256"])

    def test_each_case_proves_input_consumption_and_later_divergence(self):
        self.assertEqual(
            {case["name"] for case in self.receipt["cases"]},
            {"battle_event_dialogue", "turn_start_dialogue"},
        )
        for case in self.receipt["cases"]:
            start_reads = case["probe_gate"]["kbd_reads"]
            self.assertEqual(case["probe_gate"]["kbd_pending"], 0)
            self.assertEqual(case["esc"], case["enter"])
            self.assertEqual(len(case["esc"]), len(case["none"]))
            self.assertGreaterEqual(len(case["esc"]), 4)
            for accepted, idle in zip(case["esc"], case["none"]):
                self.assertEqual(accepted["kbd_reads"], start_reads + 1)
                self.assertEqual(idle["kbd_reads"], start_reads)
            self.assertNotEqual(case["esc"][-1]["png_sha256"],
                                case["none"][-1]["png_sha256"])


if __name__ == "__main__":
    unittest.main()
