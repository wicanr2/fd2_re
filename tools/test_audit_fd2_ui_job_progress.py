#!/usr/bin/env python3
"""防止局部或矛盾紀錄被提升為介面生命週期已恢復的證明。"""
import json
from pathlib import Path
import tempfile
import unittest

from audit_fd2_ui_job_progress import audit_progress


class UIProgressAuditTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.audit = Path(self.directory.name)/'audit.jsonl'
        self.log = Path(self.directory.name)/'gui.log'
        self.rows = [
            {'frame': 7636, 'native_round': 4, 'class_ui_job': True, 'error': ''},
            {'frame': 7638, 'native_round': 4, 'class_ui_job': False, 'error': ''},
            {'frame': 8734, 'native_round': 5, 'class_ui_job': False, 'error': ''},
            {'frame': 10078, 'native_round': 5, 'class_ui_job': False, 'error': 'missing portrait116'},
        ]
        self.log.write_text('Traceback (most recent call last):\nRuntimeError: missing portrait116\n')

    def run_audit(self):
        self.audit.write_text(''.join(json.dumps(row)+'\n' for row in self.rows))
        before = self.audit.read_bytes(), self.log.read_bytes()
        try:
            return audit_progress(self.audit, self.log, 7636)
        finally:
            self.assertEqual(before, (self.audit.read_bytes(), self.log.read_bytes()))

    def test_proves_release_later_round_and_actual_final_error(self):
        report = self.run_audit()
        self.assertEqual(report['first_released']['frame'], 7638)
        self.assertEqual(report['later_round']['frame'], 8734)
        self.assertEqual(report['final_record']['error'], 'missing portrait116')

    def test_rejects_no_release(self):
        for row in self.rows:
            row['class_ui_job'] = True
        with self.assertRaisesRegex(ValueError, '缺少後續明示job釋放'):
            self.run_audit()

    def test_rejects_missing_release_field(self):
        for row in self.rows[1:]:
            del row['class_ui_job']
        with self.assertRaisesRegex(ValueError, '缺少後續明示job釋放'):
            self.run_audit()

    def test_rejects_no_later_round(self):
        for row in self.rows:
            row['native_round'] = 4
        with self.assertRaisesRegex(ValueError, '缺少釋放後回合前進'):
            self.run_audit()

    def test_rejects_wrong_final_log(self):
        self.log.write_text('Traceback (most recent call last):\nRuntimeError: different\n')
        with self.assertRaisesRegex(ValueError, '最後audit錯誤'):
            self.run_audit()

    def test_rejects_reordered_frames(self):
        self.rows[1], self.rows[2] = self.rows[2], self.rows[1]
        with self.assertRaisesRegex(ValueError, 'frame須嚴格遞增'):
            self.run_audit()

    def test_rejects_truncated_snapshot(self):
        self.rows = self.rows[:1]
        with self.assertRaisesRegex(ValueError, '缺少後續明示job釋放'):
            self.run_audit()


if __name__ == '__main__':
    unittest.main()
