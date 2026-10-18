#!/usr/bin/env python3
"""Docker內用實際分離素材驗地形製圖工具的拒收及原子輸出。"""

import argparse
import json
import shutil
import subprocess
import tempfile
from pathlib import Path

from PIL import Image

from build_fd2_hd_terrain_object import build
from verify_fd2_terrain_sources import sha


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ["binary", "pack", "maps", "sources", "recipe", "master", "report"]:
        parser.add_argument("--" + name, type=Path, required=True)
    args = parser.parse_args()
    if args.report.exists():
        raise ValueError("不得覆寫既有拒收收據")
    results = []
    original_mask = args.pack / "tilesets/fdshap/map_32/tile_0000/mask.png"
    original_hash = sha(original_mask)
    with tempfile.TemporaryDirectory(prefix="terrain-contract-", dir=args.report.parent) as temporary:
        root = Path(temporary)
        fixture = root / "pack"
        fixture.mkdir()
        (fixture / "palette").symlink_to(args.pack / "palette", target_is_directory=True)
        banks = fixture / "tilesets/fdshap"
        banks.mkdir(parents=True)
        for index in range(32):
            (banks / f"map_{index:02d}").symlink_to(args.pack / f"tilesets/fdshap/map_{index:02d}", target_is_directory=True)
        last = banks / "map_32"
        last.mkdir()
        actual = args.pack / "tilesets/fdshap/map_32"
        for source in actual.iterdir():
            if source.name != "tile_0000":
                (last / source.name).symlink_to(source, target_is_directory=source.is_dir())
        bad = last / "tile_0000"
        bad.mkdir()
        for name in ["frame.png", "remap_mask.png"]:
            (bad / name).symlink_to(actual / "tile_0000" / name)
        Image.new("L", (23, 24), 255).save(bad / "mask.png")
        out = root / "late-output"
        run = subprocess.run([str(args.binary), "-pack", str(fixture), "-maps", str(args.maps),
                              "-out", str(out)], capture_output=True, text=True, timeout=120)
        if run.returncode == 0 or "mask PNG must be 24x24" not in run.stderr:
            raise AssertionError(f"末張圖庫錯誤未拒收：{run.returncode}, {run.stderr}")
        if out.exists() or list(root.glob(".terrain-art-stage-*")):
            raise AssertionError("預檢失敗仍留下半套輸出")
        results.append({"case": "late-map32-invalid-mask", "status": "PASS", "published": False})

        recipe = json.loads(args.recipe.read_text())
        recipe.pop("foreground_overrides")
        changed = root / "no-foreground-override.json"
        changed.write_text(json.dumps(recipe))
        out = root / "invalid-foreground-output"
        try:
            build(args.pack, args.maps, args.sources, changed, args.master, out)
        except ValueError as error:
            if "不能直接共用新母稿" not in str(error):
                raise
        else:
            raise AssertionError("不同前景被當成base切片")
        if out.exists() or list(root.glob(".terrain-object-stage-*")):
            raise AssertionError("前景拒收仍留下半套輸出")
        results.append({"case": "different-foreground-requires-own-source", "status": "PASS", "published": False})

        out = root / "existing-output"
        out.mkdir()
        marker = out / "marker"
        marker.write_bytes(b"preserve")
        try:
            build(args.pack, args.maps, args.sources, args.recipe, args.master, out)
        except ValueError as error:
            if "輸出須" not in str(error):
                raise
        else:
            raise AssertionError("既有目錄未被保護")
        if marker.read_bytes() != b"preserve" or [p.name for p in out.iterdir()] != ["marker"]:
            raise AssertionError("既有目錄被改寫")
        results.append({"case": "existing-art-output-preserved", "status": "PASS"})
    if sha(original_mask) != original_hash:
        raise AssertionError("原始輸入被修改")
    report = {"status": "PASS", "cases": results, "original_input_unchanged": True,
              "binary_sha256": sha(args.binary), "source_manifest_sha256": sha(args.sources / "manifest.json"),
              "test_tool_sha256": sha(Path(__file__)), "fixture_cleaned": True}
    with args.report.open("x") as output:
        output.write(json.dumps(report, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps(report, ensure_ascii=False))


if __name__ == "__main__":
    main()
