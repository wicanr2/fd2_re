"""匯出全檔匹配需要的 IDA 原始指令，產物只保存本機 work/。

不改名、不改型別、不修函式邊界。既有清冊與語意索引只附加導覽及證據等級。
全檔原始碼範圍：docs/goal/118-goal-fd2-matching-decompilation-20261008.md。
"""

import hashlib
import json
import os
from pathlib import Path
import sys

import ida_auto
import ida_bytes
import ida_funcs
import ida_loader
import ida_nalt
import idaapi
import idautils
import idc

sys.path.insert(0, str(Path(__file__).parent))
from fd2_semantic_index import load_semantic_index, validate_input_identity


def main():
    repo = Path("/repo")
    raw = Path(ida_nalt.get_input_file_path()).read_bytes()
    identity = {"file": "FD2.EXE", "size": len(raw), "md5": hashlib.md5(raw).hexdigest(),
                "sha256": hashlib.sha256(raw).hexdigest()}
    reference = next(x for x in json.loads((repo / "docs/data/fd2-reference-files.json").read_text(encoding="utf-8"))["files"]
                     if x["file"] == "FD2.EXE")
    validate_input_identity(reference, identity)
    inventory_path = repo / "docs/data/ida/fd2_function_inventory.json"
    inventory = json.loads(inventory_path.read_text(encoding="utf-8"))
    validate_input_identity(inventory["input"], identity)
    index_path = repo / "docs/data/ida/fd2_semantic_index.json"
    index, annotations = load_semantic_index(index_path, repo, include_instructions=True)
    validate_input_identity(index["input"], identity)
    ida_auto.auto_wait()
    functions = []
    for known in inventory["functions"]:
        address = int(known["start"], 16)
        fn = ida_funcs.get_func(address)
        if fn is None or fn.start_ea != address:
            raise ValueError(f"既有函式 {address:#x} 未在目前資料庫找到")
        chunks = []
        for start, end in idautils.Chunks(address):
            instructions = []
            ea = start
            while ea < end:
                if ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
                    size = idc.get_item_size(ea)
                    offset = ida_loader.get_fileregion_offset(ea)
                    if offset < 0 or offset + size > len(raw):
                        raise ValueError(f"指令 {ea:#x} 沒有可回查原始檔位元組")
                    instructions.append({
                        "ida_linear_address": hex(ea), "file_offset": hex(offset),
                        "loaded_bytes": ida_bytes.get_bytes(ea, size).hex(),
                        "file_bytes": raw[offset:offset + size].hex(),
                        "instruction": idc.generate_disasm_line(ea, 0),
                        "semantic_annotations": annotations.get(ea, []),
                    })
                ea = idc.next_head(ea, end)
                if ea == idaapi.BADADDR:
                    break
            chunks.append({"ida_linear_start": hex(start), "ida_linear_end": hex(end),
                           "instructions": instructions})
        functions.append({
            "inventory": known, "ida_analysis_name": idc.get_func_name(address),
            "current_ida_linear_end": hex(fn.end_ea),
            "boundary_matches_inventory": fn.end_ea == int(known["end"], 16),
            "semantic_annotations": annotations.get(address, []), "chunks": chunks,
            "semantic_warning": "沒有已證實註記的名稱與操作數只作導覽，不自動提升語意。",
        })
    result = {"schema_version": 1, "input": identity,
              "tool": {"name": "IDA Pro", "version": idaapi.get_kernel_version(),
                       "address_space": "IDA LE loader linear address; file offsets explicitly separate"},
              "exporter_sha256": hashlib.sha256(Path(__file__).read_bytes()).hexdigest(),
              "inventory_sha256": hashlib.sha256(inventory_path.read_bytes()).hexdigest(),
              "semantic_index_sha256": hashlib.sha256(index_path.read_bytes()).hexdigest(),
              "function_count": len(functions), "functions": functions,
              "rights": "完整指令只保存本機 work，不加入公開儲存庫或發行包"}
    Path(os.environ["FD2_MATCHING_IMAGE_OUTPUT"]).write_text(
        json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    idc.qexit(0)


if __name__ == "__main__":
    main()
