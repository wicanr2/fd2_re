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
    parser.add_argument("--additional-restored", type=Path, action="append", default=[])
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    raw = args.original.read_bytes()
    image = json.loads(args.evidence.read_text(encoding="utf-8"))
    report_dirs = [args.restored, *args.additional_restored]
    reports = [(directory, json.loads((directory / "report.json").read_text(encoding="utf-8")))
               for directory in report_dirs]
    identity = {"file": "FD2.EXE", "size": len(raw), "md5": hashlib.md5(raw).hexdigest(),
                "sha256": hashlib.sha256(raw).hexdigest()}
    if identity != image["input"] or any(identity != report["input"] for _, report in reports):
        raise ValueError("原版、IDA 與 C 收據不是同一個固定版本")
    if any(report["evidence_sha256"] != sha(args.evidence) for _, report in reports):
        raise ValueError("C 收據未綁定此 IDA 證據")
    meta = parse_le(raw)
    fixups = parse_fixups(raw, meta)
    by_address = {f["inventory"]["start"]: f for f in image["functions"]}
    for directory, report in reports:
        for trial in report['trials']:
            for part in trial.get('function_intervals', []):
                if part.get('kind') == 'unowned_code':
                    if report.get('source_key') != 'game_event_layout':
                        raise ValueError('來源未登記清冊外區間')
                    from fd2_matching_intervals import unowned_interval
                    extra = unowned_interval(int(part['address'], 16), image, raw, args.evidence.parent)
                    if part.get('evidence_sha256') != extra['evidence_sha256']:
                        raise ValueError('清冊外區間收據沒有綁定證據')
                    by_address[part['address']] = extra
    selected = {}
    for directory, report in reports:
        for trial in report["trials"]:
            if trial["exact_interval_bytes"]:
                compiled = (directory / trial["stem"] / "candidate.bin").read_bytes()
                if hashlib.sha256(compiled).hexdigest() != trial["code_sha256"]:
                    raise ValueError("C 程式碼與連結收據不符")
                intervals = trial.get("function_intervals", [{"address": trial["address"], "offset": 0, "size": len(compiled)}])
                addresses = trial.get("addresses", [trial["address"]])
                if [part["address"] for part in intervals] != addresses:
                    raise ValueError("已匹配函式清單與區間不符")
                if len(intervals) > 1 and not trial.get("exact_entry_offsets"):
                    raise ValueError("多函式來源未證明各入口位置相同")
                sparse = trial.get('layout') == 'sparse_functions'
                compound = trial.get('layout') == 'compound_entries'
                if compound:
                    from fd2_matching_game_restore import validate_compound_layout
                    macro = next((flag[2:] for flag in trial['flags'] if flag.startswith('-d')),None)
                    if len(intervals)!=1 or addresses!=[trial['address']] or not trial.get('exact_entry_offsets'):
                        raise ValueError('複合入口必須只涵蓋一個原始owner')
                    validate_compound_layout(macro,report.get('source_key'),by_address[trial['address']],trial.get('compound_layout',{}))
                if sparse:
                    from fd2_matching_game_restore import SPARSE_GROUPS, SOURCES
                    macro = next((flag[2:] for flag in trial['flags'] if flag.startswith('-d')), None)
                    source_key = report.get('source_key')
                    if source_key not in SOURCES or macro not in SOURCES[source_key][1] or addresses != [hex(a) for a in SPARSE_GROUPS.get(macro, ())]:
                        raise ValueError('非連續函式來源未登錄')
                    fragments = trial.get('sparse_layout', {}).get('fragments', [])
                    if len(fragments) != len(intervals) or any(f['address'] != p['address'] or f['compiler_offset'] != p['offset'] or f['size'] != p['size'] for f, p in zip(fragments, intervals)):
                        raise ValueError('非連續函式布局與完整區間不符')
                cursor = 0
                for part in intervals:
                    address = part["address"]
                    function = by_address[address]
                    start, end = int(address, 16), int(function["inventory"]["end"], 16)
                    if part["offset"] != cursor or (not sparse and start != int(trial["address"], 16) + cursor) or part["size"] != end - start:
                        raise ValueError("已匹配區間不連續或不符合原始邊界")
                    code = compiled[cursor:cursor + part["size"]]
                    previous = selected.get(address)
                    if previous and previous[3] != code:
                        raise ValueError("同一位址的已匹配 C 來源相互矛盾")
                    selected.setdefault(address, (directory, trial, part, code))
                    cursor += part["size"]
                if cursor != len(compiled):
                    raise ValueError("多函式區間未覆蓋完整編譯碼")
    rebuilt = bytearray(raw)
    source_spans = []
    written_offsets = set()
    for address, (directory, trial, part, compiled) in selected.items():
        function = by_address[address]
        start, end = int(address, 16), int(function["inventory"]["end"], 16)
        instructions = [i for chunk in function["chunks"] for i in chunk["instructions"]
                        if start <= int(i["ida_linear_address"], 16) < end]
        if len(trial.get("addresses", [address])) == 1 and len(instructions) != sum(len(chunk["instructions"]) for chunk in function["chunks"]):
            raise ValueError("單函式來源不能略過共用尾段")
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
            target_offsets = set(range(offset, offset + len(code)))
            if written_offsets & target_offsets:
                raise ValueError("已匹配 C 區間彼此重疊")
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
            written_offsets.update(target_offsets)
            cursor += len(code)
        if cursor != len(compiled):
            raise ValueError("候選沒有被完整寫入")
        source_spans.append({"ida_linear_address": address, "source_kind": "matched_c",
                             "size": len(compiled), "compiler_flags": trial["flags"],
                             "original_classification": function["inventory"]["classification"]})
        source_report = next(report for source_directory, report in reports if source_directory == directory)
        source_spans[-1].update(source_report_directory=str(directory),
                               source_report_sha256=sha(directory / "report.json"))
        for key in ("compiler_version", "compiler_inputs", "source_key", "source_path", "source_sha256", "compiler_frontend", "cpp_wrapper"):
            if key in source_report:
                source_spans[-1][key] = source_report[key]
        if function['inventory'].get('kind') == 'unowned_code':
            source_spans[-1]['source_kind'] = 'matched_c_unowned_code'
        if len(trial.get("addresses", [])) > 1:
            source_spans[-1].update(compiled_group=trial["address"], group_code_sha256=trial["code_sha256"],
                                    group_offset=part["offset"])
            if trial.get('layout') == 'sparse_functions':
                source_spans[-1].update(group_layout='sparse_functions',
                    sparse_transform_sha256=trial['sparse_transform_sha256'])
        if trial.get('layout') == 'compound_entries':
            source_spans[-1].update(group_layout='compound_entries',
                compiler_entry_addresses=[f['address'] for f in trial['compound_layout']['fragments']],
                sparse_transform_sha256=trial['sparse_transform_sha256'])
    args.output.mkdir(parents=True, exist_ok=True)
    output = args.output / "FD2.EXE"
    output.write_bytes(rebuilt)
    matched = {address for address in selected if by_address[address]['inventory'].get('kind') != 'unowned_code'}
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
              "matched_unowned_code_bytes": sum(s['size'] for s in source_spans if s['source_kind'] == 'matched_c_unowned_code'),
              "source_report_sha256": sha(args.restored / "report.json"),
              "source_reports": [{"path": str(directory), "sha256": sha(directory / "report.json")}
                                 for directory in report_dirs],
              "export_sha256": sha(args.evidence), "driver_sha256": sha(Path(__file__)),
              "rights": "完整 EXE 僅本機研究使用，不加入 Git 或公開發行"}
    (args.output / "bootstrap-receipt.json").write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"whole_file_equal": result["whole_file_equal"], "output_sha256": result["output_sha256"],
                      "decompilation_complete": False, "counts": counts}, ensure_ascii=False), flush=True)


if __name__ == "__main__":
    main()
