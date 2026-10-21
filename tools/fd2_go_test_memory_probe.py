#!/usr/bin/env python3
"""Run a bounded command inside Docker and preserve cgroup memory evidence.

Usage inside the maintained Go image:
  python3 tools/fd2_go_test_memory_probe.py --out /existing/path/run -- command ...
The parent directory must exist. Existing evidence is never overwritten.
"""

import argparse
import json
import os
from pathlib import Path
import resource
import signal
import subprocess
import time


def memory_snapshot():
    root = Path("/sys/fs/cgroup")
    result = {}
    for name in ("memory.current", "memory.peak", "memory.max", "memory.swap.current",
                 "memory.swap.max"):
        path = root / name
        if path.is_file():
            value = path.read_text().strip()
            result[name] = int(value) if value.isdigit() else value
    for name in ("memory.events", "memory.stat"):
        path = root / name
        if path.is_file():
            result[name] = {k: int(v) for k, v in
                            (line.split() for line in path.read_text().splitlines())}
    return result


def largest_processes():
    found = []
    for path in Path("/proc").iterdir():
        if not path.name.isdigit():
            continue
        try:
            fields = dict(line.split(":", 1) for line in
                          (path / "status").read_text().splitlines() if ":" in line)
            found.append({"pid": int(path.name), "name": fields.get("Name", "").strip(),
                          "rss_kib": int(fields.get("VmRSS", "0 kB").split()[0])})
        except (FileNotFoundError, ProcessLookupError, PermissionError):
            continue
    return sorted(found, key=lambda item: item["rss_kib"], reverse=True)[:8]


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--timeout", type=float, default=900)
    parser.add_argument("command", nargs=argparse.REMAINDER)
    args = parser.parse_args()
    command = args.command[1:] if args.command[:1] == ["--"] else args.command
    if not command or args.timeout <= 0:
        parser.error("a command and positive timeout are required")
    outputs = [Path(str(args.out) + suffix) for suffix in
               (".log", ".memory.json", ".samples.jsonl")]
    if not args.out.parent.is_dir() or any(path.exists() for path in outputs):
        parser.error("output parent must exist and evidence paths must be unused")
    before = memory_snapshot()
    started = time.monotonic()
    timed_out = False
    peak_sample = before
    peak_processes = []
    with outputs[0].open("x") as log, outputs[2].open("x") as samples:
        process = subprocess.Popen(command, stdout=log, stderr=subprocess.STDOUT,
                                   start_new_session=True)
        try:
            while process.poll() is None:
                elapsed = time.monotonic() - started
                sample = memory_snapshot()
                if sample.get("memory.current", 0) >= peak_sample.get("memory.current", 0):
                    peak_sample = sample
                    peak_processes = largest_processes()
                samples.write(json.dumps({"elapsed_seconds": round(elapsed, 3),
                                          "memory": sample}) + "\n")
                samples.flush()
                if elapsed > args.timeout:
                    timed_out = True
                    os.killpg(process.pid, signal.SIGKILL)
                    break
                time.sleep(0.5)
        finally:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGKILL)
            process.wait()
    after = memory_snapshot()
    events_before = before.get("memory.events", {})
    events_delta = {key: value - events_before.get(key, 0)
                    for key, value in after.get("memory.events", {}).items()}
    report = {"schema_version": 1, "command": command, "cwd": os.getcwd(),
              "returncode": process.returncode, "timeout": timed_out,
              "elapsed_seconds": round(time.monotonic() - started, 3),
              "before": before, "after": after, "memory_events_delta": events_delta,
              "largest_sample": peak_sample, "largest_sample_processes": peak_processes,
              "children_maxrss_kib": resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss,
              "log": str(outputs[0]), "samples": str(outputs[2])}
    outputs[1].write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps({"report": str(outputs[1]), "returncode": process.returncode,
                      "timeout": timed_out, "elapsed_seconds": report["elapsed_seconds"],
                      "memory_peak_bytes": after.get("memory.peak"),
                      "memory_events_delta": events_delta}), flush=True)
    return 124 if timed_out else (process.returncode if process.returncode >= 0
                                 else 128 - process.returncode)


if __name__ == "__main__":
    raise SystemExit(main())
