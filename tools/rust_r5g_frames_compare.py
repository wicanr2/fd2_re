#!/usr/bin/env python3
"""R5 階段 5（docs/goal/121，r5g）：GPU 模式與 CPU 模式的畫格比對與時間統計。

用法：
  rust_r5g_frames_compare.py frames --cpu <frames.json> --gpu <frames.json> --out <compare.json>
      fd2-game 畫格擷取（opening／chapter 模式）的兩份 frames.json：擷取次數、序列雜湊、每一列
      （capture、count、update_frame、rgba_sha256、node）、gaps 與 stopped 都要相同；GPU 那份要有
      renderer.mode == gpu-display-list。不同時列出第一個不同的擷取。結束碼：相同 0，否則 1。
  rust_r5g_frames_compare.py jsonl --cpu <frames.jsonl> --gpu <frames.jsonl> --out <compare.json>
      fd2_frames／外殼的 frames.jsonl：逐行相同。結束碼同上。
  rust_r5g_frames_compare.py summary --app-summary <summary.json> --want gpu
      外殼 summary.json 的 renderer.used 要是 --want。結束碼同上。
  rust_r5g_frames_compare.py gocheck --compare <compare.json>
      tools/rust_r5_frames_compare.py 的輸出：ok、擷取次數與 Go 黃金相同、沒有停點。結束碼同上。
  rust_r5g_frames_compare.py timing --out <timing.json> 標籤=<timing.jsonl> ...
      每份 timing.jsonl 的 update_ms、draw_ms、record_ms、gpu_ms、interval_ms 的筆數、平均、中位數、p95、最大值。
"""
import argparse
import json
import statistics
import sys


def expand(doc):
    """rows（連續相同雜湊合併成 count）展開成每次擷取一筆。"""
    out = []
    for row in doc["rows"]:
        for i in range(row["count"]):
            out.append((row["capture"] + i, row["rgba_sha256"]))
    return out


def cmd_frames(a):
    cpu = json.load(open(a.cpu, encoding="utf-8"))
    gpu = json.load(open(a.gpu, encoding="utf-8"))
    problems = []
    renderer = gpu.get("renderer") or {}
    if renderer.get("mode") != "gpu-display-list":
        problems.append(f"GPU 那份沒有 renderer.mode=gpu-display-list：{renderer}")
    if "renderer" in cpu:
        problems.append("CPU 那份不應有 renderer 欄")
    for name, doc in (("CPU", cpu), ("GPU", gpu)):
        if doc.get("captures") != a.expected_captures or doc.get("stopped") is not None:
            problems.append(f"{name} 未完成 {a.expected_captures} 次擷取")
        if sum(row["count"] for row in doc["rows"]) != a.expected_captures:
            problems.append(f"{name} 的畫格列數與擷取次數不符")
        if a.expected_sequence and doc.get("sequence_sha256") != a.expected_sequence:
            problems.append(f"{name} 的畫格序列與階段 3 已驗證結果不同")
    for key in ("mode", "captures", "sequence_sha256", "changed_frames", "distinct_frames", "virtual_frames",
                "uncaptured_gaps", "uncaptured_update", "clock", "stopped"):
        if cpu.get(key) != gpu.get(key):
            problems.append(f"{key} 不同：CPU {cpu.get(key)!r}，GPU {gpu.get(key)!r}")
    if cpu["rows"] != gpu["rows"]:
        ec, eg = expand(cpu), expand(gpu)
        first = next((i for i, (x, y) in enumerate(zip(ec, eg)) if x != y), min(len(ec), len(eg)))
        problems.append(f"rows 不同，第一個不同的擷取是 {first}")
    if cpu.get("gaps") != gpu.get("gaps"):
        problems.append("gaps 不同")
    result = {
        "schema": "fd2_r5g_frames_compare/1",
        "cpu": a.cpu, "gpu": a.gpu,
        "captures": cpu.get("captures"), "sequence_sha256": cpu.get("sequence_sha256"),
        "gpu_sequence_sha256": gpu.get("sequence_sha256"),
        "renderer": renderer, "equal": not problems, "problems": problems,
    }
    json.dump(result, open(a.out, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
    print(f"擷取 {cpu.get('captures')} 次；CPU 序列 {cpu.get('sequence_sha256')}；GPU 序列 {gpu.get('sequence_sha256')}")
    for p in problems:
        print("  不同：" + p)
    print("相同" if not problems else "不同")
    return 0 if not problems else 1


def cmd_jsonl(a):
    cpu = open(a.cpu, encoding="utf-8").read().splitlines()
    gpu = open(a.gpu, encoding="utf-8").read().splitlines()
    first = next((i for i, (x, y) in enumerate(zip(cpu, gpu)) if x != y), None)
    equal = len(cpu) == len(gpu) and first is None
    if first is None and len(cpu) != len(gpu):
        first = min(len(cpu), len(gpu))
    result = {"schema": "fd2_r5g_jsonl_compare/1", "cpu": a.cpu, "gpu": a.gpu, "cpu_lines": len(cpu),
              "gpu_lines": len(gpu), "equal": equal, "first_diff_line": first}
    json.dump(result, open(a.out, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
    print(f"CPU {len(cpu)} 行、GPU {len(gpu)} 行：" + ("逐行相同" if equal else f"第 {first} 行起不同"))
    return 0 if equal else 1


def cmd_summary(a):
    doc = json.load(open(a.app_summary, encoding="utf-8"))
    used = (doc.get("renderer") or {}).get("used")
    print(f"外殼 renderer：{doc.get('renderer')}")
    return 0 if used == a.want else 1


def cmd_gocheck(a):
    doc = json.load(open(a.compare, encoding="utf-8"))
    ok = bool(doc.get("ok")) and doc.get("go_captures") == doc.get("rust_captures") and not doc.get("rust_stopped")
    print(f"{a.compare}：與 Go 黃金 {doc.get('equal')}／{doc.get('go_captures')} 相同，"
          f"Rust 擷取 {doc.get('rust_captures')}：" + ("整段相同" if ok else "不同"))
    return 0 if ok else 1


def stats(values):
    v = sorted(x for x in values if x is not None)
    if not v:
        return None
    p95 = v[min(len(v) - 1, int(round(0.95 * (len(v) - 1))))]
    return {"n": len(v), "mean": round(statistics.fmean(v), 3), "median": round(statistics.median(v), 3),
            "p95": round(p95, 3), "max": round(v[-1], 3)}


def cmd_timing(a):
    out = {"schema": "fd2_r5g_timing/1", "unit": "ms", "series": {}}
    for spec in a.files:
        label, path = spec.split("=", 1)
        rows = [json.loads(line) for line in open(path, encoding="utf-8") if line.strip()]
        series = {"file": path, "frames": len(rows),
                  "renderer": sorted({r.get("renderer") for r in rows if r.get("renderer")})}
        for key in ("update_ms", "draw_ms", "record_ms", "gpu_ms", "interval_ms"):
            s = stats(r.get(key) for r in rows)
            if s:
                series[key] = s
        out["series"][label] = series
        brief = ", ".join(f"{k} 中位 {series[k]['median']} p95 {series[k]['p95']}"
                          for k in ("draw_ms", "gpu_ms", "interval_ms") if k in series)
        print(f"{label}：{len(rows)} 幀；{brief}")
    json.dump(out, open(a.out, "w", encoding="utf-8"), ensure_ascii=False, indent=2)
    return 0


def main():
    p = argparse.ArgumentParser()
    sub = p.add_subparsers(dest="cmd", required=True)
    f = sub.add_parser("frames")
    f.add_argument("--cpu", required=True)
    f.add_argument("--gpu", required=True)
    f.add_argument("--out", required=True)
    f.add_argument("--expected-captures", type=int, required=True)
    f.add_argument("--expected-sequence")
    j = sub.add_parser("jsonl")
    j.add_argument("--cpu", required=True)
    j.add_argument("--gpu", required=True)
    j.add_argument("--out", required=True)
    s = sub.add_parser("summary")
    s.add_argument("--app-summary", required=True)
    s.add_argument("--want", required=True)
    g = sub.add_parser("gocheck")
    g.add_argument("--compare", required=True)
    t = sub.add_parser("timing")
    t.add_argument("--out", required=True)
    t.add_argument("files", nargs="+")
    a = p.parse_args()
    cmds = {"frames": cmd_frames, "jsonl": cmd_jsonl, "summary": cmd_summary, "gocheck": cmd_gocheck,
            "timing": cmd_timing}
    return cmds[a.cmd](a)


if __name__ == "__main__":
    sys.exit(main())
