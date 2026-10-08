#!/usr/bin/env python3
"""組合已匹配 C 切片與本機原版其餘區間，建立全檔比較基準。

保留的遊戲機器碼明示為尚未還原。全檔 SHA 相同仍不能宣稱完成 C 反編譯。
輸出含原版 EXE 的其餘 bytes，僅保存本機 work/，不加入公開庫。
"""

import argparse
import hashlib
import json
from pathlib import Path
import struct

from le_xref import parse_le, parse_fixups


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--original", type=Path, required=True)
    parser.add_argument("--evidence", type=Path, required=True)
    parser.add_argument("--restored", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    raw = args.original.read_bytes()
    image = json.loads(args.evidence.read_text(encoding="utf-8"))
    report = json.loads((args.restored / "report.json").read_text(encoding="utf-8"))
    identity = {"file": "FD2.EXE", "size": len(raw), "md5": hashlib.md5(raw).hexdigest(),
                "sha256": hashlib.sha256(raw).hexdigest()}
    if identity != image["input"] or identity != report["input"]:
        raise ValueError("原版、IDA 與 C 收據不是同一個固定版本")
    if report["evidence_sha256"] != sha(args.evidence):
        raise ValueError("C 收據未綁定此 IDA 證據")
    meta = parse_le(raw)
    fixups = parse_fixups(raw, meta)
    by_address = {f["inventory"]["start"]: f for f in image["functions"]}
    selected = {}
    for trial in report["trials"]:
        if trial["exact_interval_bytes"]:
            selected.setdefault(trial["address"], trial)
    rebuilt = bytearray(raw)
    source_spans = []
    for address, trial in selected.items():
        function = by_address[address]
        instructions = [i for chunk in function["chunks"] for i in chunk["instructions"]]
        compiled = (args.restored / trial["stem"] / "candidate.bin").read_bytes()
        if hashlib.sha256(compiled).hexdigest() != trial["code_sha256"]:
            raise ValueError("C 程式碼與連結收據不符")
        if compiled.hex() != "".join(i["loaded_bytes"] for i in instructions):
            raise ValueError("C 程式碼沒有完整匹配原始 IDA 指令")
        cursor = 0
        for instruction in instructions:
            offset = int(instruction["file_offset"], 16)
            file_bytes = bytes.fromhex(instruction["file_bytes"])
            loaded = bytes.fromhex(instruction["loaded_bytes"])
            code = bytearray(compiled[cursor:cursor + len(loaded)])
            if raw[offset:offset + len(file_bytes)] != file_bytes:
                raise ValueError("原始指令來源不符")
            for location, target in fixups.items():
                if not offset <= location < offset + len(code):
                    continue
                relative = location - offset
                if relative + 4 > len(code) or struct.unpack_from("<I", code, relative)[0] != target:
                    raise ValueError("本基準只允許已核對的完整32位元內部重定位")
                objects = [ob for ob in meta["objs"] if ob["base"] <= target < ob["base"] + ob["vsize"]]
                if len(objects) != 1:
                    raise ValueError("重定位目標沒有唯一 LE object")
                object_offset = target - objects[0]["base"]
                if struct.unpack_from("<I", file_bytes, relative)[0] != object_offset:
                    raise ValueError("LE object-relative 位址與原檔不符")
                struct.pack_into("<I", code, relative, object_offset)
            if bytes(code) != file_bytes:
                raise ValueError("解除已知 LE 重定位後仍有位元組差異")
            rebuilt[offset:offset + len(code)] = code
            cursor += len(code)
        if cursor != len(compiled):
            raise ValueError("候選沒有被完整寫入")
        source_spans.append({"ida_linear_address": address, "source_kind": "matched_c",
                             "size": len(compiled), "compiler_flags": trial["flags"],
                             "original_classification": function["inventory"]["classification"]})
    args.output.mkdir(parents=True, exist_ok=True)
    output = args.output / "FD2.EXE"
    output.write_bytes(rebuilt)
    matched = set(selected)
    ledger = [{"ida_linear_address": f["inventory"]["start"],
               "size": f["inventory"]["size"],
               "source_kind": "matched_c" if f["inventory"]["start"] in matched else
                              "retained_library" if f["inventory"]["classification"]["value"] == "runtime" else "unrestored_original_code",
               "original_classification": f["inventory"]["classification"]}
              for f in image["functions"]]
    counts = {kind: sum(r["source_kind"] == kind for r in ledger)
              for kind in ("matched_c", "retained_library", "unrestored_original_code")}
    result = {"schema_version": 1, "input": identity, "output_sha256": sha(output),
              "whole_file_equal": bytes(rebuilt) == raw,
              "decompilation_complete": False,
              "reason": "其餘遊戲／未分類機器碼來自原版，尚未以 C 還原；全檔相同只證明組合基準有效",
              "counts": counts, "restored_spans": source_spans, "functions": ledger,
              "source_report_sha256": sha(args.restored / "report.json"),
              "export_sha256": sha(args.evidence), "driver_sha256": sha(Path(__file__)),
              "rights": "完整 EXE 僅本機研究使用，不加入 Git 或公開發行"}
    (args.output / "bootstrap-receipt.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"whole_file_equal": result["whole_file_equal"], "output_sha256": result["output_sha256"],
                      "decompilation_complete": False, "counts": counts}, ensure_ascii=False), flush=True)


if __name__ == "__main__":
    main()
