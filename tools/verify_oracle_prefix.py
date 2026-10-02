#!/usr/bin/env python3
"""核對 dosgolem 原版預算重跑的完整前綴；不改寫輸入、影像或原版狀態。

只排除 PNG 背景輸出的兩個排程欄位，所有 CPU、原始記錄、亂數、DOS 呼叫、
輸入鏈及狀態注入均完整比較。這不是原版／重製畫面驗收的替代品。
"""
import argparse
import hashlib
import json
import re
from pathlib import Path

EXE_SHA256 = "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
ASYNC_FIELDS = {"frame_deferred", "frames_deferred_so_far"}
REQUIRED_FIELDS = {"address_space", "control_seq", "dos_file_calls", "eip", "evidence_restrictions",
                   "exe_sha256", "input_chain", "input_kind", "kbd_pending", "kbd_reads",
                   "normal_player_path_verified", "registers", "runner", "schema", "state_injections",
                   "steps", "unit_base", "units", "view"}


def read_json(path):
    return json.loads(path.read_text())


def controls(path):
    rows = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
    result = {}
    for row in rows:
        seq = row["seq"]
        if seq in result:
            raise ValueError(f"控制序號重複：{seq}")
        result[seq] = row
    return result


def valid_checkpoint(document):
    if not (REQUIRED_FIELDS <= set(document) and document["schema"] == 1 and document["runner"] == "dosgolem" and
            isinstance(document["registers"], list) and len(document["registers"]) == 8 and
            isinstance(document["units"], list) and isinstance(document["view"], dict) and
            "rng_word" in document["view"] and isinstance(document["state_injections"], list)):
        return False
    for unit in document["units"]:
        if not isinstance(unit, dict) or not isinstance(unit.get("raw_hex"), str) or len(unit["raw_hex"]) != 160:
            return False
        try:
            bytes.fromhex(unit["raw_hex"])
        except ValueError:
            return False
    return True


def verify(prefix, full, *, prefix_commit=None, full_commit=None):
    explicit_pair = prefix_commit is not None or full_commit is not None
    if explicit_pair and not all(isinstance(commit, str) and re.fullmatch(r"[0-9a-f]{40}", commit)
                                 for commit in [prefix_commit, full_commit]):
        raise ValueError("升級比較須同時指定完整40位 prefix-commit 與 full-commit")
    checkpoints = sorted(prefix.glob("checkpoint-*.json"))
    if not checkpoints:
        raise ValueError("前綴沒有原版檢查點")
    old_controls = controls(prefix / "control-history.jsonl")
    new_controls = controls(full / "control-history.jsonl")
    old_runner, new_runner = read_json(prefix / "runner.json"), read_json(full / "runner.json")
    commit = old_runner.get("dosgolem_commit")
    new_commit = new_runner.get("dosgolem_commit")
    clean = bool(commit and commit != "unknown" and commit == new_commit)
    if explicit_pair:
        clean = commit == prefix_commit and new_commit == full_commit
    for runner in [old_runner, new_runner]:
        clean &= runner.get("original_fd2_exe_sha256") == EXE_SHA256
        clean &= runner.get("dosgolem_tracked_dirty_files") == 0 and runner.get("dosgolem_untracked_files") == 0
    differences = []
    checked = []
    for path in checkpoints:
        seq = int(path.stem.split("-")[1])
        other = full / path.name
        if not other.is_file():
            differences.append({"seq": seq, "reason": "重跑缺少此檢查點"})
            continue
        before, after = read_json(path), read_json(other)
        if not all(valid_checkpoint(document) for document in [before, after]):
            differences.append({"seq": seq, "reason": "未知或缺欄位的原版檢查點"})
            continue
        if before.get("control_seq") != seq or after.get("control_seq") != seq:
            differences.append({"seq": seq, "reason": "檔名與控制序號不符"})
            continue
        if before.get("exe_sha256") != EXE_SHA256 or after.get("exe_sha256") != EXE_SHA256:
            differences.append({"seq": seq, "reason": "未知執行檔"})
            continue
        fields = sorted(k for k in set(before) | set(after)
                        if k not in ASYNC_FIELDS and (k not in before or k not in after or before[k] != after[k]))
        if seq > 0 and (seq not in old_controls or seq not in new_controls or old_controls[seq] != new_controls[seq]):
            fields.append("control-history")
        if fields:
            differences.append({"seq": seq, "fields": fields})
        checked.append(seq)
    contiguous = checked == list(range(len(checkpoints)))
    return {
        "schema_version": 1, "kind": "fd2_original_oracle_prefix_comparison",
        "status": "passed" if clean and contiguous and not differences else "failed",
        "prefix": str(prefix), "full": str(full), "dosgolem_commit": commit,
        "prefix_dosgolem_commit": commit, "full_dosgolem_commit": new_commit,
        "commit_comparison": "explicit-commit-pair" if explicit_pair else "same-commit",
        "fixed_exe_sha256": EXE_SHA256, "clean_source": clean,
        "prefix_checkpoints": len(checkpoints), "checked": len(checked), "contiguous_from_zero": contiguous,
        "ignored_fields": sorted(ASYNC_FIELDS), "differences": differences,
        "limits": "只驗原版重跑的控制與CPU／記錄前綴；PNG排程欄位不參與，完整RGB另走章驗證器。",
        "verifier_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--prefix", type=Path, required=True)
    parser.add_argument("--full", type=Path, required=True)
    parser.add_argument("--out", type=Path, required=True)
    parser.add_argument("--prefix-commit", help="升級比較時明示前綴來源的完整40位提交")
    parser.add_argument("--full-commit", help="升級比較時明示重跑來源的完整40位提交")
    args = parser.parse_args()
    report = verify(args.prefix, args.full, prefix_commit=args.prefix_commit, full_commit=args.full_commit)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(f"原版前綴：{report['status']}，{report['checked']}/{report['prefix_checkpoints']}檢查點，差異{len(report['differences'])}筆")
    return 0 if report["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
