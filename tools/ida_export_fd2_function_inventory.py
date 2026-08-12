"""匯出 FD2.EXE 的非破壞性 IDA 函式清冊。

只可在使用者授權的 IDA Pro Docker 流程執行。輸出保留 IDA 原始函式名稱、
位址邊界、區段與直接交叉參照計數；語意註記只從受版控、已分級的外部索引合併，
不會改名、改型別或寫回 IDA 資料庫。
"""

import hashlib
import json
import os
import sys

import ida_auto
import ida_funcs
import ida_name
import ida_segment
import ida_xref
import idaapi
import idautils
import idc

sys.path.insert(0, os.path.dirname(__file__))
from fd2_semantic_index import load_semantic_index, validate_input_identity


DEFAULT_OUTPUT = "/work/fd2_function_inventory.json"
DEFAULT_SEMANTICS = "/workspace/docs/data/ida/fd2_semantic_index.json"
DEFAULT_REPO_ROOT = "/workspace"


def collect_xrefs(start, first, next_):
    result = []
    current = first(start)
    while current != idaapi.BADADDR:
        result.append(current)
        current = next_(start, current)
    return result


def hex_address(address):
    return hex(address)


def function_location(address):
    function = ida_funcs.get_func(address)
    if function is None:
        return None
    return {
        "start": hex_address(function.start_ea),
        "end": hex_address(function.end_ea),
        "ida_analysis_name": ida_name.get_name(function.start_ea) or None,
    }


def code_xrefs_to(address):
    result = []
    for source in collect_xrefs(
        address, ida_xref.get_first_cref_to, ida_xref.get_next_cref_to
    ):
        result.append({
            "from": hex_address(source),
            "caller_function": function_location(source),
        })
    return result


def data_xrefs_to(address):
    return [
        hex_address(source)
        for source in collect_xrefs(
            address, ida_xref.get_first_dref_to, ida_xref.get_next_dref_to
        )
    ]


def segment(address):
    item = ida_segment.getseg(address)
    if item is None:
        return None
    return {
        "start": hex_address(item.start_ea),
        "end": hex_address(item.end_ea),
        "name": ida_segment.get_segm_name(item),
    }


FUNCTION_FLAGS = (
    ("library", "FUNC_LIB"),
    ("thunk", "FUNC_THUNK"),
    ("no_return", "FUNC_NORET"),
    ("far", "FUNC_FAR"),
    ("static", "FUNC_STATIC"),
    ("frame", "FUNC_FRAME"),
    ("hidden", "FUNC_HIDDEN"),
    ("lumina", "FUNC_LUMINA"),
)


def function_flags(flags):
    return [
        label
        for label, constant_name in FUNCTION_FLAGS
        if flags & getattr(ida_funcs, constant_name, 0)
    ]


def classification_for(flags, annotations):
    if annotations:
        values = {entry["classification"] for entry in annotations}
        if len(values) != 1:
            raise RuntimeError("semantic annotations disagree on classification")
        confidence = min(
            (entry["confidence"] for entry in annotations),
            key=("已證實", "強推論", "假說", "未知").index,
        )
        return {
            "value": values.pop(),
            "confidence": confidence,
            "source": "versioned semantic index",
        }
    if flags & getattr(ida_funcs, "FUNC_LIB", 0):
        return {
            "value": "runtime",
            "confidence": "強推論",
            "source": "IDA FUNC_LIB after configured FLIRT signatures",
        }
    return {
        "value": "unknown",
        "confidence": "未知",
        "source": "not yet classified",
    }


def kernel_version():
    for module in (idaapi, idc):
        getter = getattr(module, "get_kernel_version", None)
        if getter is not None:
            return str(getter())
    sdk = getattr(idaapi, "IDA_SDK_VERSION", None)
    return f"SDK {sdk}" if sdk is not None else "unknown"


def main():
    ida_auto.auto_wait()
    input_path = idc.get_input_file_path()
    output_path = os.environ.get("FD2_IDA_OUTPUT", DEFAULT_OUTPUT)
    semantic_path = os.environ.get("FD2_IDA_SEMANTIC_INDEX", DEFAULT_SEMANTICS)
    repo_root = os.environ.get("FD2_REPO_ROOT", DEFAULT_REPO_ROOT)
    semantic_document, semantics = load_semantic_index(semantic_path, repo_root)

    with open(input_path, "rb") as source:
        input_bytes = source.read()
    input_identity = {
        "file": os.path.basename(input_path),
        "size": len(input_bytes),
        "md5": hashlib.md5(input_bytes).hexdigest(),
        "sha256": hashlib.sha256(input_bytes).hexdigest(),
    }
    validate_input_identity(semantic_document["input"], input_identity)

    functions = []
    for start in idautils.Functions():
        function = ida_funcs.get_func(start)
        if function is None:
            continue
        annotations = semantics.get(function.start_ea, [])
        code_xrefs = code_xrefs_to(function.start_ea)
        data_xrefs = data_xrefs_to(function.start_ea)
        functions.append({
            "start": hex_address(function.start_ea),
            "end": hex_address(function.end_ea),
            "size": function.end_ea - function.start_ea,
            "ida_analysis_name": ida_name.get_name(function.start_ea) or None,
            "segment": segment(function.start_ea),
            "ida_function_flags": function_flags(function.flags),
            "code_xrefs_to_start": code_xrefs,
            "data_xrefs_to_start": data_xrefs,
            "classification": classification_for(function.flags, annotations),
            "semantic_annotations": annotations,
        })

    function_starts = {int(item["start"], 0) for item in functions}
    unmatched_annotations = [
        hex_address(address)
        for address in sorted(set(semantics) - function_starts)
    ]
    if unmatched_annotations:
        raise RuntimeError(
            "semantic annotations do not match IDA function starts: "
            + ", ".join(unmatched_annotations)
        )

    classification_counts = {}
    for function in functions:
        value = function["classification"]["value"]
        classification_counts[value] = classification_counts.get(value, 0) + 1

    report = {
        "schema_version": 1,
        "tool": {
            "name": "IDA Pro",
            "version": kernel_version(),
            "address_space": "IDA flat-loader linear address",
            "annotation_policy": (
                "export only; the script does not rename, retype, comment, or "
                "otherwise mutate the IDA database"
            ),
        },
        "input": input_identity,
        "imagebase": hex_address(idaapi.get_imagebase()),
        "function_count": len(functions),
        "classification_counts": classification_counts,
        "semantic_annotation_count": sum(
            len(function["semantic_annotations"]) for function in functions
        ),
        "classification_note": (
            "product/driver only come from the versioned semantic index; "
            "runtime is a mechanical IDA FUNC_LIB classification after the "
            "configured FLIRT signatures; every other function remains unknown"
        ),
        "functions": functions,
    }
    with open(output_path, "w", encoding="utf-8") as output:
        json.dump(report, output, ensure_ascii=False, indent=2)
        output.write("\n")
    idc.qexit(0)


if __name__ == "__main__":
    main()
