#!/usr/bin/env python3
"""核對已保存GUI失敗紀錄中的介面釋放、後續回合與最後錯誤，不從黑圖判卡死。"""
import argparse
import hashlib
import json
import os
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def audit_progress(audit_path, log_path, snapshot_frame):
    if type(snapshot_frame) is not int or snapshot_frame < 0:
        raise ValueError('snapshot-frame須是非負整數')
    rows = [json.loads(line) for line in audit_path.read_text().splitlines() if line.strip()]
    if not rows or any(not isinstance(row, dict) or type(row.get('frame')) is not int for row in rows):
        raise ValueError('需要具整數frame的完整audit')
    if any(left['frame'] >= right['frame'] for left, right in zip(rows, rows[1:])):
        raise ValueError('audit frame須嚴格遞增')
    snapshot = next((row for row in rows if row['frame'] == snapshot_frame), None)
    if snapshot is None or snapshot.get('class_ui_job') is not True or type(snapshot.get('native_round')) is not int:
        raise ValueError('明示snapshot須記錄活躍class_ui_job及native_round')
    released = next((row for row in rows if row['frame'] > snapshot_frame
                     and row.get('class_ui_job') is False), None)
    if released is None:
        raise ValueError('缺少後續明示job釋放，不能反證持續等待')
    advanced = next((row for row in rows if row['frame'] > released['frame']
                     and type(row.get('native_round')) is int
                     and row['native_round'] > snapshot['native_round']), None)
    if advanced is None:
        raise ValueError('缺少釋放後回合前進，不擴稱後續流程已執行')
    final = rows[-1]
    error = final.get('error')
    log = log_path.read_text()
    if (not isinstance(error, str) or not error
            or 'Traceback (most recent call last):' not in log
            or not log.rstrip().endswith('RuntimeError: '+error)):
        raise ValueError('最後audit錯誤須與完整GUI失敗log一致')
    keys = ['frame', 'native_round', 'turn', 'class_ui_job', 'transient_ui', 'ai_busy', 'error']
    selected = lambda row: {key: row.get(key) for key in keys}
    return {
        'status': 'PROVEN-RECORDED-UI-PROGRESS',
        'audit': {'file': str(audit_path), 'sha256': digest(audit_path), 'rows': len(rows)},
        'log': {'file': str(log_path), 'sha256': digest(log_path)},
        'snapshot': selected(snapshot), 'first_released': selected(released),
        'later_round': selected(advanced), 'final_record': selected(final),
        'conclusion': '所指中途job欄位已釋放且回合繼續；此份原紀錄不支持持續卡死，最後失敗原因依原始error。',
        'scope': '只核對已保存的單輪紀錄；不從未綁定frame的黑圖判定其來源，不證整輪成功、通用生命週期修復或原版parity。',
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--audit', type=Path, required=True)
    parser.add_argument('--log', type=Path, required=True)
    parser.add_argument('--snapshot-frame', type=int, required=True)
    parser.add_argument('--report', type=Path, required=True)
    args = parser.parse_args()
    out = args.report.resolve()
    if (not Path('/.dockerenv').exists() or out.exists() or not out.is_relative_to(ROOT)
            or out.parent.stat().st_uid != os.getuid()):
        raise SystemExit('在Docker使用本人擁有工作區的新報告路徑')
    try:
        result = audit_progress(args.audit, args.log, args.snapshot_frame)
    except (ValueError, OSError) as error:
        raise SystemExit(str(error)) from error
    result['tool_sha256'] = digest(Path(__file__))
    out.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps({key: result[key] for key in ['status', 'snapshot', 'first_released', 'later_round', 'final_record']}, ensure_ascii=False))


if __name__ == '__main__':
    main()
