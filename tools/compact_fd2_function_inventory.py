#!/usr/bin/env python3
"""把一次性 IDA 全量匯出縮成可版控的 FD2 函式清冊。"""

import argparse
import hashlib
import json
import os
import tempfile
from pathlib import Path

from fd2_semantic_index import build_compact_report, load_semantic_index, refresh_compact_annotations


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input")
    parser.add_argument("output")
    parser.add_argument("--refresh-semantic-index", type=Path)
    args = parser.parse_args()

    with open(args.input, encoding="utf-8") as source:
        inventory = json.load(source)
    if args.refresh_semantic_index:
        index, annotations = load_semantic_index(args.refresh_semantic_index, Path(__file__).resolve().parents[1])
        report = refresh_compact_annotations(inventory, index["input"], annotations)
        metadata = {key: value for key, value in inventory.items()
                    if key not in ("functions", "classification_counts", "semantic_annotation_count", "annotation_refresh")}
        metadata["functions"] = [{key: value for key, value in function.items()
                                  if key not in ("classification", "semantic_annotations")}
                                 for function in inventory["functions"]]
        metadata_sha = hashlib.sha256(json.dumps(metadata, ensure_ascii=False, sort_keys=True).encode()).hexdigest()
        previous = inventory.get("annotation_refresh", {})
        source_sha = previous.get("input_inventory_sha256") if previous.get("source_metadata_sha256") == metadata_sha else None
        report["annotation_refresh"] = {
            "method": "offline versioned semantic-index binding; original IDA metadata retained",
            "input_inventory_sha256": source_sha or hashlib.sha256(Path(args.input).read_bytes()).hexdigest(),
            "source_metadata_sha256": metadata_sha,
            "semantic_index_sha256": hashlib.sha256(args.refresh_semantic_index.read_bytes()).hexdigest(),
            "ida_rerun": False,
        }
    else:
        report = build_compact_report(inventory)
    output = Path(args.output)
    output.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary = tempfile.mkstemp(prefix=output.name + ".", dir=output.parent)
    try:
        with os.fdopen(descriptor, "w", encoding="utf-8") as target:
            json.dump(report, target, ensure_ascii=False, indent=2)
            target.write("\n")
        os.replace(temporary, output)
    except BaseException:
        try:
            os.unlink(temporary)
        except FileNotFoundError:
            pass
        raise
    print(
        f"functions={report['function_count']} "
        f"classifications={report['classification_counts']} output={output}"
    )


if __name__ == "__main__":
    main()
