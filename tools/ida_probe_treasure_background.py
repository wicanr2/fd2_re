"""匯出 #53 的取寶提示底圖 caller 與保存，保留原始定位並合併已審查註記。"""
import hashlib
import json
import os

import ida_auto
import ida_bytes
import ida_funcs
import ida_nalt
import idaapi
import idautils
import idc

ida_auto.auto_wait()
with open(os.environ.get("FD2_IDA_INPUT", "/input/FD2.EXE"), "rb") as source:
    raw = source.read()
assert len(raw) == 357074
assert hashlib.sha256(raw).hexdigest() == "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f"
assert ida_nalt.retrieve_input_file_md5().hex() == hashlib.md5(raw).hexdigest()
result = {
    "input": "FD2.EXE", "size": len(raw), "md5": hashlib.md5(raw).hexdigest(),
    "sha256": hashlib.sha256(raw).hexdigest(), "tool": "IDA Pro " + idaapi.get_kernel_version(),
    "address_space": "IDA LE linear", "functions": [],
}
canonical_path = "/src/docs/data/ida/fd2_treasure_background_20261001.json"
canonical = None
if os.path.isfile(canonical_path):
    with open(canonical_path, encoding="utf-8") as source:
        canonical = json.load(source)
    assert canonical["sha256"] == result["sha256"]
    result["claims"] = canonical.get("claims", [])
    for key in ("kind", "date", "scope", "visual", "projection_review"):
        if key in canonical:
            result[key] = canonical[key]


def reviewed_annotations(ea):
    annotations = []
    for claim in (canonical or {}).get("claims", []):
        assert claim["level"] in ("已證實", "強推論", "假說", "未知")
        for location in claim["original"]:
            bounds = [int(part, 16) for part in location.split("..")]
            if bounds[0] <= ea <= bounds[-1]:
                annotations.append({
                    "semantic": claim["semantic"], "level": claim["level"],
                    "source": [canonical_path.replace("/src/", "")] + claim["source"],
                })
                break
    return annotations


for target in (0x1956B, 0x176B4, 0x17643, 0x18D8C):
    fn = ida_funcs.get_func(target)
    assert fn is not None
    rows = []
    for ea in idautils.FuncItems(fn.start_ea):
        if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
            continue
        annotations = reviewed_annotations(ea)
        level = "已證實" if annotations and all(a["level"] == "已證實" for a in annotations) else "未知"
        rows.append({
            "address": hex(ea), "bytes": ida_bytes.get_bytes(ea, idc.get_item_size(ea)).hex(),
            "original": idc.generate_disasm_line(ea, 0), "semantic": "", "level": level,
            "warning": "" if level == "已證實" else "未審查語意；原始指令僅作定位",
            "source": ["tools/ida_probe_treasure_background.py"], "semantic_annotations": annotations,
        })
    result["functions"].append({
        "target": hex(target), "address": hex(fn.start_ea), "end": hex(fn.end_ea),
        "original_name": idc.get_func_name(fn.start_ea), "instructions": rows,
        "semantic": "", "level": "未知", "warning": "只閉合個別已審查的寫入與消費端",
        "source": ["tools/ida_probe_treasure_background.py"],
        "callers": [{"address": hex(x.frm), "original": idc.generate_disasm_line(x.frm, 0),
                     "bytes": ida_bytes.get_bytes(x.frm, idc.get_item_size(x.frm)).hex(),
                     "semantic": "", "level": "未知", "warning": "只保留呼叫定位；語意未在本探針審查",
                     "source": ["tools/ida_probe_treasure_background.py"]}
                    for x in idautils.XrefsTo(fn.start_ea) if x.iscode],
    })
with open(os.environ["FD2_IDA_OUTPUT"], "w", encoding="utf-8") as output:
    json.dump(result, output, ensure_ascii=False, indent=2)
    output.write("\n")
idc.qexit(0)
