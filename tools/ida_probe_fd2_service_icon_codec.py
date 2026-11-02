"""核對設施服務圖示的原始呼叫與像素 primitive，保留位址及分級註記。"""
import hashlib
import json
import os
import sys
from pathlib import Path

import ida_auto
import ida_bytes
import ida_funcs
import ida_kernwin
import ida_nalt
import idautils
import idc

sys.path.insert(0, "/repo/tools")
from fd2_semantic_index import load_semantic_index

ida_auto.auto_wait()
raw = Path("/input/FD2.EXE").read_bytes()
identity = dict(file="FD2.EXE", size=len(raw), md5=hashlib.md5(raw).hexdigest(),
                sha256=hashlib.sha256(raw).hexdigest())
assert identity == dict(file="FD2.EXE", size=357074,
                        md5="b97caf2239a27a896069d03549d96e1e",
                        sha256="222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f")
database_sha = bytes(ida_nalt.retrieve_input_file_sha256()).hex()
assert database_sha == identity["sha256"]
_, annotations = load_semantic_index("/repo/docs/data/ida/fd2_semantic_index.json",
                                    repo_root="/repo", include_instructions=True)

def annotated(ea):
    rows = annotations.get(ea, [])
    confirmed = any(a.get("confidence") == "已證實" for a in rows)
    return dict(address=hex(ea), semantic_annotations=rows,
                semantic_level="已證實" if confirmed else "未知",
                unconfirmed_warning=not confirmed,
                source="FD2.EXE IDA LE linear " + hex(ea))

def instruction(ea):
    result = annotated(ea)
    result.update(bytes=(ida_bytes.get_bytes(ea, idc.get_item_size(ea)) or b"").hex(),
                  original_disassembly=idc.generate_disasm_line(ea, 0))
    return result

targets = [int(x, 0) for x in os.environ.get("FD2_IDA_ADDRESSES",
            "0x2d669 0x2d85f 0x2d9fe 0x1685c 0x4e8af 0x4e916 0x4e9bb 0x4e9e4 0x4e9ff").split()]
functions = []
for ea in targets:
    f = ida_funcs.get_func(ea)
    if f is None:
        functions.append(dict(target=hex(ea), function=None))
        continue
    result = annotated(f.start_ea)
    result.update(target=hex(ea), original_name=idc.get_func_name(f.start_ea),
                  start=hex(f.start_ea), end=hex(f.end_ea),
                  instructions=[instruction(x) for x in idautils.FuncItems(f.start_ea)],
                  callers=[instruction(x) for x in idautils.CodeRefsTo(f.start_ea, False)])
    functions.append(result)
report = dict(schema="fd2-service-icon-codec-ida-v1", input=identity,
              database_input_sha256=database_sha, ida_version=ida_kernwin.get_kernel_version(),
              address_space="IDA LE linear, image base0, code segment begins0x10000",
              semantic_policy="original names/operands/bytes retained; annotations from reviewed semantic index",
              functions=functions)
Path(os.environ.get("FD2_IDA_OUTPUT", "/out/service-codec-ida.json")).write_text(
    json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
idc.qexit(0)
