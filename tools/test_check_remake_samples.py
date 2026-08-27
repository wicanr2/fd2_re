from pathlib import Path
import unittest

import check_remake_samples as checker


class CheckRemakeSamplesTest(unittest.TestCase):
    def sample(self, **updates):
        value = {
            "id": "sample-1",
            "layer": "ui",
            "result": "pass",
            "evidence_level": "RUNTIME-E1",
            "remake_checked": True,
            "production_path": True,
            "normal_input": True,
            "route_patch": False,
            "debug_shortcut": False,
            "qualifies": True,
            "artifacts": ["README.md"],
            "limitations": [],
        }
        value.update(updates)
        return value

    def registry(self, samples):
        return {
            "schema_version": 1,
            "confidence_target": 0.95,
            "minimum_total": 60,
            "layer_minimums": checker.LAYERS,
            "samples": samples,
        }

    def test_valid_qualifying_sample_is_counted(self):
        errors, counts, registered = checker.validate(
            self.registry([self.sample()]), Path(__file__).resolve().parents[1]
        )
        self.assertEqual(errors, [])
        self.assertEqual(registered, 1)
        self.assertEqual(counts["ui"], 1)

    def test_debug_sample_cannot_claim_qualification(self):
        errors, counts, _ = checker.validate(
            self.registry([self.sample(debug_shortcut=True)]),
            Path(__file__).resolve().parents[1],
        )
        self.assertTrue(any("qualifies 應為 false" in error for error in errors))
        self.assertEqual(counts["ui"], 0)

    def test_missing_artifact_is_rejected(self):
        errors, _, _ = checker.validate(
            self.registry([self.sample(artifacts=["does-not-exist"])]),
            Path(__file__).resolve().parents[1],
        )
        self.assertTrue(any("artifacts 不存在" in error for error in errors))

    def test_duplicate_id_is_rejected(self):
        errors, _, _ = checker.validate(
            self.registry([self.sample(), self.sample()]),
            Path(__file__).resolve().parents[1],
        )
        self.assertTrue(any("id 重複" in error for error in errors))


if __name__ == "__main__":
    unittest.main()
