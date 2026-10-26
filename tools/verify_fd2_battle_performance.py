#!/usr/bin/env python3
"""核對正常GUI效能原始觀測，分開API頻率與實際呼叫計數，不代替平台驗收。"""
import argparse
import hashlib
import json
import math
from pathlib import Path
import statistics


def digest(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def require(condition, message):
    if not condition:
        raise ValueError(message)


def validate_rows(rows):
    previous = 0.0
    require(bool(rows), '缺效能觀測')
    for index, row in enumerate(rows):
        for key in ('elapsed_seconds', 'interval_seconds', 'ebiten_actual_fps', 'ebiten_actual_tps'):
            value = row.get(key)
            require(type(value) in (int, float) and math.isfinite(value) and value >= 0, '頻率／時間值無效')
        interval, elapsed = row['interval_seconds'], row['elapsed_seconds']
        require(interval > 0 and elapsed > previous and abs(elapsed-previous-interval) < 1e-6,
                '時間窗不連續')
        previous = elapsed
        for key in ('updates', 'draws', 'final_draws'):
            require(type(row.get(key)) is int and row[key] >= 0, '呼叫次數無效')
        scenes = row.get('draw_scenes')
        require(isinstance(scenes, dict) and all(type(n) is int and n >= 0 for n in scenes.values()),
                '場景計數無效')
        require(set(scenes) <= {'world-or-ui', 'physical', 'split-command', 'compound-command'}, '未知場景')
        require(sum(scenes.values()) == row['draws'], '場景合計與Draw不同')
        require(type(row.get('final')) is bool and (not row['final'] or index == len(rows)-1), '末段標記不同')


def summarize(rows):
    if not rows:
        return None
    total = sum(row['interval_seconds'] for row in rows)
    return {
        'windows': len(rows), 'seconds': total,
        'update_hz_from_calls': sum(row['updates'] for row in rows)/total,
        'draw_hz_from_calls': sum(row['draws'] for row in rows)/total,
        'final_draw_hz_from_calls': sum(row['final_draws'] for row in rows)/total,
        'actual_tps_median': statistics.median(row['ebiten_actual_tps'] for row in rows),
        'actual_tps_min': min(row['ebiten_actual_tps'] for row in rows),
        'actual_fps_median': statistics.median(row['ebiten_actual_fps'] for row in rows),
        'actual_fps_min': min(row['ebiten_actual_fps'] for row in rows),
        'longest_window_seconds': max(row['interval_seconds'] for row in rows),
    }


def main():
    p = argparse.ArgumentParser(description=__doc__)
    for name in ('gui', 'manifest', 'binary', 'gui-tool', 'out'):
        p.add_argument('--'+name, type=Path, required=True)
    p.add_argument('--self-test', action='store_true')
    args = p.parse_args()
    require(not args.out.exists(), '輸出已存在')
    gui, manifest = [json.loads(path.read_text()) for path in (args.gui, args.manifest)]
    require(gui.get('status') == 'PASS' and manifest.get('status') == 'OBSERVATION-ONLY', '正常GUI未通過')
    require(gui.get('binary_sha256') == manifest.get('binary_sha256') == digest(args.binary), '程式不同')
    require(gui.get('gui_tool_sha256') == manifest.get('gui_tool_sha256') == digest(args.gui_tool), 'GUI工具不同')
    require(all(gui.get(key) is True for key in ('natural_attack_return_checked', 'theme_save_bytes_equal',
                                                'node_boundary_save_checked', 'cold_loaded_units_equal',
                                                'new_process_title_load')), '正常返回或保存冷讀不足')
    require(manifest.get('missing_reports') == [] and len(manifest.get('reports', [])) == 2,
            '須有普通操作及新程序LOAD兩份效能觀測')
    reports, all_rows = [], []
    require(len({item['file'] for item in manifest['reports']}) == 2, '重複效能報告')
    for epoch, item in enumerate(manifest['reports']):
        require(item['file'] == f'performance-{epoch}.jsonl', '效能路徑不同')
        path = args.manifest.parent/item['file']
        require(digest(path) == item['sha256'] and path.stat().st_size == item['bytes'], '原始效能檔不同')
        rows = [json.loads(line) for line in path.read_text().splitlines()]
        validate_rows(rows)
        stable = [row for row in rows if row['elapsed_seconds']-row['interval_seconds'] >= 5 and not row['final']]
        physical = [row for row in stable if row['draws'] > 0 and row['draw_scenes'].get('physical') == row['draws']]
        reports.append({'epoch': epoch, 'file': item['file'], 'sha256': item['sha256'],
                        'all_windows': summarize(rows), 'after_first_five_seconds': summarize(stable),
                        'physical_only_complete_windows': summarize(physical)})
        if epoch == 0:
            require(len(physical) >= 3, '缺代表性物理攻防的連續觀測窗')
        all_rows.extend(rows)
    rejected = []
    if args.self_test:
        for name in ('negative-frequency', 'broken-time-window', 'wrong-scene-count', 'fractional-count'):
            rows = json.loads(json.dumps(all_rows[:1]))
            if name == 'negative-frequency': rows[0]['ebiten_actual_tps'] = -1
            elif name == 'broken-time-window': rows[0]['interval_seconds'] += 2
            elif name == 'wrong-scene-count': rows[0]['draws'] += 1
            else: rows[0]['updates'] = 1.5
            try:
                validate_rows(rows)
            except ValueError:
                rejected.append(name)
            else:
                raise ValueError('效能反例誤放行：'+name)
    result = {'status': 'PASS-PERFORMANCE-OBSERVATION',
              'gui_sha256': digest(args.gui), 'manifest_sha256': digest(args.manifest),
              'binary_sha256': digest(args.binary), 'verifier_sha256': digest(Path(__file__)),
              'epochs': reports, 'rejected_mutants': rejected,
              'platform_acceptance': False,
              'scope': '正常X11鍵盤／HD與F2，Xvfb、llvmpipe、audit與PNG擷取開銷均包含。API FPS可能受虛擬垂直同步影響；呼叫頻率不是GPU完成延遲。只核觀測完整性，不把資料格式通過當流暢度或三平台通過。'}
    args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    print(json.dumps(result, ensure_ascii=False))


if __name__ == '__main__':
    main()
