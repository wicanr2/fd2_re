"""匯出 #166 物理演出 caller 與雙背景捲動，保留原始定位及證據分級。

入口：docs/data/ida/fd2_physical_background_selection_20261004.json。
只在合法授權的 IDA Pro 9.4 Docker 內對一次性資料庫執行。
"""
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
canonical_path = "docs/data/ida/fd2_physical_background_selection_20261004.json"
with open("/src/" + canonical_path, encoding="utf-8") as source:
    canonical = json.load(source)
result = {
    "schema_version": 1,
    "input": "FD2.EXE", "size": len(raw), "md5": hashlib.md5(raw).hexdigest(),
    "sha256": hashlib.sha256(raw).hexdigest(), "tool": "IDA Pro " + idaapi.get_kernel_version(),
    "address_space": "IDA LE linear", "functions": [],
}


def annotations_at(ea):
    annotations = []
    for claim in (canonical.get("physical_scroll_evidence", {}).get("claims", []) +
                  canonical.get("physical_tail_evidence", {}).get("claims", [])):
        assert claim["level"] in ("已證實", "強推論", "假說", "未知")
        for location in claim["original"]:
            bounds = [int(part, 16) for part in location.split("..")]
            if bounds[0] <= ea <= bounds[-1]:
                annotations.append({"semantic": claim["semantic"], "level": claim["level"],
                                    "source": [canonical_path] + claim["source"]})
                break
    return annotations


for target in (0x29164, 0x2939D, 0x29C90, 0x29DED, 0x2BC9A):
    fn = ida_funcs.get_func(target)
    assert fn is not None and fn.start_ea == target
    rows = []
    for ea in idautils.FuncItems(target):
        if not ida_bytes.is_code(ida_bytes.get_full_flags(ea)):
            continue
        annotations = annotations_at(ea)
        level = "已證實" if annotations and all(a["level"] == "已證實" for a in annotations) else "未知"
        rows.append({
            "address": hex(ea), "bytes": ida_bytes.get_bytes(ea, idc.get_item_size(ea)).hex(),
            "original": idc.generate_disasm_line(ea, 0), "semantic": "", "level": level,
            "warning": "" if level == "已證實" else "未審查語意；原始指令僅作定位",
            "source": ["tools/ida_probe_physical_presentation.py"], "semantic_annotations": annotations,
        })
    result["functions"].append({
        "address": hex(target), "end": hex(fn.end_ea), "original_name": idc.get_func_name(target),
        "instructions": rows, "semantic": "", "level": "未知",
        "warning": "只閉合已審查的 caller-specific claims", "source": ["tools/ida_probe_physical_presentation.py"],
        "callers": [{"address": hex(x.frm), "original": idc.generate_disasm_line(x.frm, 0),
                     "bytes": ida_bytes.get_bytes(x.frm, idc.get_item_size(x.frm)).hex(),
                     "level": "未知", "warning": "呼叫定位；語意另見 canonical claims",
                     "source": ["tools/ida_probe_physical_presentation.py"]}
                    for x in idautils.XrefsTo(target) if x.iscode],
    })
result["tables"] = []
for ea, size in ((0x5255F, 24), (0x52577, 24), (0x525D6, 6)):
    annotations = annotations_at(ea)
    confirmed = bool(annotations) and all(a["level"] == "已證實" for a in annotations)
    result["tables"].append({
        "address": hex(ea), "original_name": idc.get_name(ea),
        "bytes": ida_bytes.get_bytes(ea, size).hex(),
        "semantic_annotations": annotations,
        "level": "已證實" if confirmed else "未知",
        "warning": "" if confirmed else "未審查資料表語意",
        "source": ["tools/ida_probe_physical_presentation.py"],
    })
with open(os.environ["FD2_IDA_OUTPUT"], "w", encoding="utf-8") as output:
    json.dump(result, output, ensure_ascii=False, indent=2)
    output.write("\n")
idc.qexit(0)
