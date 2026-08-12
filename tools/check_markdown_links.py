#!/usr/bin/env python3
"""唯讀檢查儲存庫 Markdown 的本地連結與圖片目標。"""

from pathlib import Path
from urllib.parse import unquote, urlsplit
import argparse
import re
import sys


INLINE_LINK = re.compile(r"!?\[[^\]]*\]\(([^)]+)\)")
REFERENCE_LINK = re.compile(r"^\s*\[[^\]]+\]:\s*(\S+)")


def iter_targets(source: Path):
    in_fence = False
    for line_number, line in enumerate(
        source.read_text(encoding="utf-8", errors="replace").splitlines(), 1
    ):
        stripped = line.lstrip()
        if stripped.startswith("```") or stripped.startswith("~~~"):
            in_fence = not in_fence
            continue
        if in_fence:
            continue
        for match in INLINE_LINK.finditer(line):
            yield line_number, match.group(1)
        definition = REFERENCE_LINK.match(line)
        if definition:
            yield line_number, definition.group(1)


def local_path(raw_target: str):
    target = raw_target.strip()
    if target.startswith("<") and ">" in target:
        target = target[1 : target.index(">")]
    else:
        target = target.split(maxsplit=1)[0]
    target = unquote(target)
    parsed = urlsplit(target)
    if not target or target.startswith("#") or parsed.scheme or parsed.netloc:
        return None
    if not parsed.path or any(token in parsed.path for token in ("*", "{", "}")):
        return None
    return parsed.path


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("root", nargs="?", default=".")
    args = parser.parse_args()
    root = Path(args.root).resolve()
    checked = 0
    broken = []

    for source in sorted(root.rglob("*.md")):
        for line_number, raw_target in iter_targets(source):
            path_text = local_path(raw_target)
            if path_text is None:
                continue
            candidate = (
                root / path_text.lstrip("/")
                if path_text.startswith("/")
                else source.parent / path_text
            )
            checked += 1
            if not candidate.exists():
                broken.append(
                    (source.relative_to(root), line_number, raw_target, candidate)
                )

    print(f"checked_local_targets={checked}")
    print(f"broken_targets={len(broken)}")
    for source, line_number, target, candidate in broken:
        print(f"{source}:{line_number}: {target} -> {candidate}")
    return 1 if broken else 0


if __name__ == "__main__":
    sys.exit(main())
