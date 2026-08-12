#!/usr/bin/env python3
"""把一次性 IDA 全量匯出縮成可版控的 FD2 函式清冊。"""

import argparse
import json
import os
import tempfile
from pathlib import Path

from fd2_semantic_index import build_compact_report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("input")
    parser.add_argument("output")
    args = parser.parse_args()

    with open(args.input, encoding="utf-8") as source:
        report = build_compact_report(json.load(source))
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
