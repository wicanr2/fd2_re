#!/usr/bin/env python3
"""在Linux容器以兩次/proc唯讀取樣記錄指定遊戲的執行緒CPU時間。"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import time


def snapshot(pid):
    rows = {}
    for task in Path(f'/proc/{pid}/task').iterdir():
        try:
            rest = (task/'stat').read_text().rsplit(')', 1)[1].split()
            rows[task.name] = {'name': (task/'comm').read_text().strip(),
                               'ticks': int(rest[11])+int(rest[12])}
        except (FileNotFoundError, ProcessLookupError):
            continue
    return rows


def latest(path):
    if path is None or not path.exists():
        return {}
    with path.open('rb') as stream:
        stream.seek(max(0, path.stat().st_size-262144))
        lines = stream.read().splitlines()
    for line in reversed(lines):
        try:
            return json.loads(line)
        except ValueError:
            continue
    return {}


def cpu_stat():
    path = Path('/sys/fs/cgroup/cpu.stat')
    return {line.split()[0]: int(line.split()[1]) for line in path.read_text().splitlines()}


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--binary-name', required=True)
    p.add_argument('--seconds', type=float, default=2)
    p.add_argument('--out', type=Path, required=True)
    p.add_argument('--audit', type=Path, help='等待目前HD物理body／counter，前後同owner才標記正式取樣')
    p.add_argument('--wait-seconds', type=float, default=180)
    args = p.parse_args()
    if args.out.exists() or not 0 < args.seconds <= 10:
        p.error('輸出須為全新路徑，取樣時間須大於0且不超過10秒')
    if not 0 < args.wait_seconds <= 600:
        p.error('等待須大於0且不超過600秒')
    if args.audit:
        deadline = time.monotonic()+args.wait_seconds
        while True:
            state = latest(args.audit)
            if (state.get('battle_scene') is True and state.get('hd_battle') is True
                and state.get('battle_owner') and state.get('battle_phase') in ('body', 'counter')):
                break
            if time.monotonic() >= deadline:
                raise SystemExit('等待HD物理攻防逾時')
            time.sleep(.1)
    matches = []
    for directory in Path('/proc').iterdir():
        if not directory.name.isdigit():
            continue
        try:
            command = (directory/'cmdline').read_bytes().split(b'\0')[0].decode()
        except (FileNotFoundError, PermissionError, ProcessLookupError):
            continue
        if Path(command).name == args.binary_name:
            matches.append(int(directory.name))
    if len(matches) != 1:
        raise SystemExit(f'遊戲程序數量須為1，實際為{matches}')
    pid = matches[0]
    started = time.monotonic()
    audit_before = latest(args.audit)
    cpu_before = cpu_stat()
    before = snapshot(pid)
    time.sleep(args.seconds)
    after = snapshot(pid)
    cpu_after = cpu_stat()
    audit_after = latest(args.audit)
    elapsed = time.monotonic()-started
    ticks_per_second = os.sysconf('SC_CLK_TCK')
    threads = [{'tid': int(tid), 'name': row['name'], 'ticks': row['ticks']-before[tid]['ticks'],
                'cpu_percent': (row['ticks']-before[tid]['ticks'])/ticks_per_second/elapsed*100}
               for tid,row in after.items() if tid in before and row['name'] == before[tid]['name']]
    threads.sort(key=lambda row: (-row['ticks'], row['tid']))
    result = {'status': 'THREAD-CPU-OBSERVATION', 'pid': pid, 'binary_name': args.binary_name,
              'seconds': elapsed, 'clock_ticks_per_second': ticks_per_second, 'threads': threads,
              'sum_thread_cpu_percent': sum(row['cpu_percent'] for row in threads),
              'ended_threads': sorted(set(before)-set(after)), 'new_threads': sorted(set(after)-set(before)),
              'cgroup_cpu_delta': {key: value-cpu_before[key] for key,value in cpu_after.items()},
              'same_hd_physical_owner': bool(args.audit) and bool(audit_before.get('battle_owner'))
                  and audit_before.get('battle_owner') == audit_after.get('battle_owner')
                  and all(state.get('battle_scene') is True and state.get('hd_battle') is True
                          and state.get('battle_phase') in ('body', 'counter')
                          for state in (audit_before, audit_after)),
              'audit_boundaries': [{key: state.get(key) for key in ('frame','battle_owner','battle_phase','hd_battle','battle_scene')}
                                   for state in (audit_before,audit_after)],
              'tool_sha256': hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              'scope': '容器內同一程序仍存活的執行緒增量；名稱只作導覽，不據此推論函式或GPU硬體使用。'}
    args.out.write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps(result,ensure_ascii=False))


if __name__ == '__main__':
    main()
