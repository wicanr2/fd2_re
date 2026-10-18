#!/usr/bin/env python3
"""將已驗物件候選及選用母稿登錄為稀疏原生HD集合，不加入原版參照圖。"""

import argparse
import hashlib
import json
import os
import shutil
import struct
import tempfile
from pathlib import Path


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ["candidate", "catalog", "asset-root", "master-root"]:
        parser.add_argument("--" + name, type=Path, required=True)
    parser.add_argument('--object', choices=['house','roof-object'], default='house')
    args = parser.parse_args()
    data = json.loads((args.candidate / "manifest.json").read_text())
    if data["verification"]["native_alpha"] != "PASS" or data["verification"]["cut_reconstruction_difference_bytes"] != 0:
        raise ValueError("候選未通過遮罩及重組檢查")
    frames = sorted(data["frames"], key=lambda entry: entry["frame"])
    count, terrain_count = (30,24) if args.object=='house' else (10,8)
    if (len(frames) != count or sum(entry["role"] == "terrain" for entry in frames) != terrain_count
            or sum(entry['role']=='foreground' for entry in frames)!=count-terrain_count
            or len({entry['frame'] for entry in frames})!=count or any(entry["map"] != 0 for entry in frames)):
        raise ValueError("物件必須具有完整且無重複的已授權map0槽位")
    identity='modern.map0.house' if args.object=='house' else 'modern.map0.roof_object'
    master_specs=[("map00-house-hd-v2-master.png", "modern.map0.house.hd.master", data["painted_master_sha256"]),
                  ("map00-foreground0279-hd-master.png", "modern.map0.foreground279.hd.master",
                   next(item["sha256"] for item in data["foreground_sources"] if item["frame"] == 279))] if args.object=='house' else [
                      ('map00-roof-object-hd-v1-master.png',identity+'.hd.master',data['painted_master_sha256'])]
    if args.object=='roof-object' and {entry['frame'] for entry in frames}!={28,29,30,31,32,33,34,35,272,273}:
        raise ValueError('褐頂物件槽位不符已核對範圍')
    copies = []
    for entry in frames:
        source = args.candidate / entry["file"]
        if source.name != entry["file"] or sha(source) != entry["sha256"]:
            raise ValueError("候選檔名或SHA不符")
        copies.append((source, entry["sha256"]))
    masters = []
    for name, master_id, expected in master_specs:
        source = args.master_root / name
        if sha(source) != expected:
            raise ValueError("母稿SHA不符")
        copies.append((source, expected))
        width, height = struct.unpack(">II", source.read_bytes()[16:24])
        masters.append({"asset_id": master_id, "role": "terrain_hd_master", "status": "concept",
                        "file": name, "width": width, "height": height, "sha256": expected,
                        "source_refs": ["docs/data/fd2-modern-hd-battlefield-20261019.json", "asset:modern.ch01.battlefield.style_a"]})
    asset = {"asset_id": identity+'.native_hd', "role": "native_terrain_hd_set", "status": "runtime_candidate",
             "consumer_contract": "fdshap_native_hd_sparse_v1", "map_id": 0, "width": 144, "height": 144,
             "frame_count": len(frames), "frame_ids": [entry["frame"] for entry in frames],
             "files": [entry["file"] for entry in frames], "frame_sha256": [entry["sha256"] for entry in frames],
             "semantics": [entry["role"] for entry in frames], "alpha_contract": "native_mask_nearest_6",
             "cycle_policy": "native_selector", "source_refs": ['asset:'+entry['asset_id'] for entry in masters]+[
                "docs/data/fd2-modern-hd-battlefield-20261019.json"]}
    additions = masters + [asset]
    catalog = json.loads(args.catalog.read_text())
    existing = {entry["asset_id"]: entry for entry in catalog["assets"]}
    for entry in additions:
        if entry["asset_id"] in existing and existing[entry["asset_id"]] != entry:
            raise ValueError("拒絕覆寫不同的清冊登錄")
    for source, expected in copies:
        target = args.asset_root / source.name
        if target.exists() and sha(target) != expected:
            raise ValueError("拒絕覆寫不同的素材")
    for source, _ in copies:
        target = args.asset_root / source.name
        if not target.exists():
            shutil.copyfile(source, target)
    for entry in additions:
        if entry["asset_id"] not in existing:
            catalog["assets"].append(entry)
    with tempfile.NamedTemporaryFile("w", dir=args.catalog.parent, delete=False, encoding="utf-8") as stream:
        stream.write(json.dumps(catalog, ensure_ascii=False, indent=2) + "\n")
        temporary = Path(stream.name)
    os.chmod(temporary, args.catalog.stat().st_mode & 0o777)
    temporary.replace(args.catalog)
    print(json.dumps({"registered_frames": count, "masters": len(masters), "catalog_sha256": sha(args.catalog)}))


if __name__ == "__main__":
    main()
