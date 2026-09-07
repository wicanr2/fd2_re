#!/usr/bin/env python3
"""只複製 Git 追蹤的 remake/assets，避免公開封包混入忽略的原版衍生素材。"""

from __future__ import annotations

import argparse
import pathlib
import shutil
import subprocess


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", type=pathlib.Path, required=True)
    parser.add_argument("--destination", type=pathlib.Path, required=True)
    args = parser.parse_args()

    repo = args.repo.resolve()
    destination = args.destination.resolve()
    raw = subprocess.check_output(
        ["git", "-C", str(repo), "ls-files", "-z", "--", "remake/assets"],
    )
    sources = [pathlib.PurePosixPath(item.decode("utf-8")) for item in raw.split(b"\0") if item]
    if not sources:
        raise SystemExit("找不到 Git 追蹤的 remake/assets")

    destination.mkdir(parents=True, exist_ok=True)
    for relative in sources:
        if relative.is_absolute() or ".." in relative.parts or relative.parts[:2] != ("remake", "assets"):
            raise SystemExit(f"拒絕不安全路徑：{relative}")
        source = repo.joinpath(*relative.parts)
        target = destination.joinpath(*relative.parts[2:])
        if not source.is_file():
            raise SystemExit(f"追蹤資產不存在：{relative}")
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(source, target)

    required = (
        destination / "scenarios" / "campaign_full.json",
        destination / "cutscenes" / "bindings" / "ch00_pre.json",
        destination / "story" / "ch00_palace.json",
        destination / "music_catalog.json",
    )
    missing = [str(path) for path in required if not path.is_file()]
    if missing:
        raise SystemExit("公開資產清冊缺少正式啟動資料：" + ", ".join(missing))
    print(f"已複製 {len(sources)} 個 Git 追蹤資產到 {destination}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
