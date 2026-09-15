#!/usr/bin/env python3
"""建構與比對「第 N 章已通關」的 FD2.SAV 槽。

`build`   在 fd2-go-test-local 容器內跑 remake/cmd/fd2-chapter-slot：由基底存檔套用
          各章戰後 handler 的已證實寫入（JOIN／grant_item／set_chapter）與明示的升級、
          金幣政策，輸出 FD2.SAV 與 manifest.json。
`compare` 把兩份存檔的同一槽逐筆、逐欄位比對（已證實欄位用名字，其餘用偏移），
          給正對照校準用：差異必須能歸因到政策值與實際遊玩值的差。
`inspect` 印一份存檔某槽的隊伍摘要。

欄位名稱只用 fd2save.py／remake/internal/fdsave 已證實的偏移；沒證實的一律以
`+0xNN` 顯示，不取名。
"""

from __future__ import annotations

import argparse
import json
import os
import struct
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import fd2save  # noqa: E402

ROOT = Path(__file__).resolve().parents[1]
GO_IMAGE = os.environ.get("FD2_GO_TEST_IMAGE", "fd2-go-test-local:latest")

# 已證實的 0x50-byte 持續紀錄欄位（remake/internal/fdsave.PersistentRecordView）。
RECORD_FIELDS = [
    ("+5", 0x05, "B"), ("camp", 0x06, "B"), ("key7", 0x07, "B"), ("id8", 0x08, "B"),
    ("+9", 0x09, "B"),
    ("cmd_mask", 0x1A, "5s"), ("race", 0x1F, "B"), ("class", 0x20, "B"), ("level", 0x21, "B"),
    ("transient", 0x22, "6s"), ("+0x34", 0x34, "B"), ("+0x35", 0x35, "B"), ("+0x36", 0x36, "B"),
    ("base_ap", 0x37, "<h"), ("base_dp", 0x39, "<h"), ("mv", 0x3B, "B"), ("exp", 0x3C, "B"),
    ("+0x3d", 0x3D, "B"), ("dx", 0x3E, "<h"), ("hp", 0x40, "<h"), ("max_hp", 0x42, "<h"),
    ("mp", 0x44, "<h"), ("max_mp", 0x46, "<h"), ("ap", 0x48, "<h"), ("dp", 0x4A, "<h"),
    ("hit", 0x4C, "<h"), ("ev", 0x4E, "<h"),
]
INVENTORY_OFFSET, INVENTORY_CELLS = 0x0A, 8
OPAQUE_RANGES = [(0x00, 0x05), (0x28, 0x34)]


def decode_file(path: Path) -> bytes:
    return fd2save.decode(path.read_bytes())


def slot_parts(plain: bytes, slot: int) -> tuple[bytes, bytes]:
    start, end = fd2save.slot_bounds(slot)
    return plain[start:start + fd2save.ROSTER_SIZE], plain[start + fd2save.ROSTER_SIZE:end]


def record_fields(record: bytes) -> dict[str, object]:
    out: dict[str, object] = {}
    for name, offset, fmt in RECORD_FIELDS:
        value = struct.unpack_from(fmt, record, offset)[0]
        out[name] = value.hex() if isinstance(value, bytes) else value
    cells = []
    for i in range(INVENTORY_CELLS):
        flag, item = record[INVENTORY_OFFSET + 2 * i], record[INVENTORY_OFFSET + 2 * i + 1]
        cells.append(f"{flag:02x}{item:02x}")
    out["inventory"] = " ".join(cells)
    for lo, hi in OPAQUE_RANGES:
        out[f"+{lo:#04x}..+{hi - 1:#04x}"] = record[lo:hi].hex()
    return out


def meta_fields(meta: bytes) -> dict[str, object]:
    return {
        "chapter": meta[0],
        "roster_count": meta[1],
        "gold": struct.unpack_from("<I", meta, 2)[0],
        "hud_gate_a": meta[6], "+7": meta[7], "+8": meta[8], "+9": meta[9],
        "+10..+39": meta[10:].hex(),
    }


def summarize(path: Path, slot: int) -> dict[str, object]:
    roster, meta = slot_parts(decode_file(path), slot)
    fields = meta_fields(meta)
    count = int(fields["roster_count"])
    records = [record_fields(roster[i * fd2save.UNIT_SIZE:(i + 1) * fd2save.UNIT_SIZE]) for i in range(count)]
    return {"meta": fields, "records": records}


def cmd_inspect(args: argparse.Namespace) -> int:
    print(json.dumps(summarize(args.save, args.slot), ensure_ascii=False, indent=1))
    return 0


def cmd_compare(args: argparse.Namespace) -> int:
    left = summarize(args.built, args.slot)
    right = summarize(args.real, args.slot)
    diffs: list[str] = []
    for key, value in left["meta"].items():
        if right["meta"].get(key) != value:
            diffs.append(f"meta.{key}: built={value} real={right['meta'].get(key)}")
    built_by_id = {r["id8"]: r for r in left["records"]}
    real_by_id = {r["id8"]: r for r in right["records"]}
    for identity in sorted(set(built_by_id) | set(real_by_id)):
        if identity not in real_by_id:
            diffs.append(f"id8={identity}: 只在 built（key7={built_by_id[identity]['key7']} level={built_by_id[identity]['level']}）")
            continue
        if identity not in built_by_id:
            diffs.append(f"id8={identity}: 只在 real（key7={real_by_id[identity]['key7']} level={real_by_id[identity]['level']}）")
            continue
        b, r = built_by_id[identity], real_by_id[identity]
        for key in b:
            if b[key] != r[key]:
                diffs.append(f"id8={identity}.{key}: built={b[key]} real={r[key]}")
    if not diffs:
        print("兩槽逐欄位相同")
        return 0
    print("\n".join(diffs))
    print(f"\n共 {len(diffs)} 處差異（要逐條歸因，見 111 的正對照規則）")
    return 1 if args.strict else 0


def cmd_build(args: argparse.Namespace) -> int:
    base = Path(args.base).resolve()
    out_dir = Path(args.out_dir).resolve()
    out_dir.mkdir(parents=True, exist_ok=True)
    if not base.is_file():
        raise SystemExit(f"基底存檔不存在：{base}")
    cache = Path(os.environ.get("FD2_GO_TEST_CACHE", ROOT / "work" / "gocache"))
    cache.mkdir(parents=True, exist_ok=True)
    uid = f"{os.getuid()}:{os.getgid()}"
    inner = [
        "/tmp/fd2-chapter-slot",
        "-base", "/base/FD2.SAV", "-base-slot", str(args.base_slot),
        "-target", str(args.target), "-out-dir", "/out",
        "-out-slot", str(args.out_slot), "-gold", str(args.gold),
        "-levels-per-chapter", str(args.levels_per_chapter), "-seed", str(args.seed),
    ]
    if args.level_overrides:
        inner += ["-level-overrides", args.level_overrides]
    script = (
        "go build -o /tmp/fd2-chapter-slot ./cmd/fd2-chapter-slot && "
        + " ".join(inner)
    )
    command = [
        "docker", "run", "--rm", "--network", "none", "--memory", "4g", "--cpus", "2",
        "--pids-limit", "256", "--log-opt", "max-size=10m", "--log-opt", "max-file=3",
        "-u", uid, "-e", "HOME=/tmp/home", "-e", "GOCACHE=/gocache", "-e", "GOFLAGS=-mod=mod",
        "-v", f"{ROOT}:/src", "-v", f"{base.parent}:/base:ro", "-v", f"{cache}:/gocache",
        "-v", f"{out_dir}:/out", "-w", "/src/remake", GO_IMAGE, "sh", "-c", script,
    ]
    if base.name != "FD2.SAV":
        raise SystemExit("基底檔名必須是 FD2.SAV（容器內以目錄掛載）")
    print(" ".join(command[-3:]))
    result = subprocess.run(command, check=False)
    return result.returncode


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("inspect", help="印一份存檔某槽的隊伍摘要")
    p.add_argument("save", type=Path)
    p.add_argument("--slot", type=int, default=0)
    p.set_defaults(func=cmd_inspect)

    p = sub.add_parser("compare", help="逐欄位比對 built 與 real 的同一槽")
    p.add_argument("built", type=Path)
    p.add_argument("real", type=Path)
    p.add_argument("--slot", type=int, default=0)
    p.add_argument("--strict", action="store_true", help="有差異就以 1 結束")
    p.set_defaults(func=cmd_compare)

    p = sub.add_parser("build", help="在容器內建構第 N 章槽")
    p.add_argument("--base", required=True, help="基底 FD2.SAV 路徑（檔名必須是 FD2.SAV）")
    p.add_argument("--base-slot", type=int, default=0)
    p.add_argument("--target", type=int, required=True, help="目標已通關章節 1..30")
    p.add_argument("--out-dir", required=True)
    p.add_argument("--out-slot", type=int, default=-1)
    p.add_argument("--gold", type=int, default=-1)
    p.add_argument("--levels-per-chapter", type=int, default=0)
    p.add_argument("--level-overrides", default="")
    p.add_argument("--seed", type=int, default=0)
    p.set_defaults(func=cmd_build)

    args = parser.parse_args()
    return args.func(args)


if __name__ == "__main__":
    raise SystemExit(main())
