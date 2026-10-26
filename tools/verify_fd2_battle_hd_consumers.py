#!/usr/bin/env python3
"""在 Docker 獨立核對正式 owner 匯出的戰鬥 HD 圖源；不代替 GPU／玩家驗證。"""

import argparse
from collections import Counter
from copy import deepcopy
import hashlib
import json
from pathlib import Path

from PIL import Image
from audit_fd2_player9_map_source import audit as audit_lmi_sources


ROOT = Path(__file__).resolve().parents[1]
EXPECTED_CASES = ({f"ordinary/{i}" for i in range(10)} |
                  {f"split/{i}" for i in (24, 29, 30)} |
                  {f"compound/{i}" for i in range(32, 36)} |
                  {f"physical/{variant}/{side}" for variant in ("shared", "separate") for side in (0, 2)})
ROLES = {"FIGANI.DAT": "battle_animation_hd_set",
         "FDOTHER.DAT": "battle_effect_hd_set",
         "BG.DAT": "battle_surface_hd", "TAI.DAT": "battle_surface_hd"}


def digest(data):
    return hashlib.sha256(data).hexdigest()


def require(value, message):
    if not value:
        raise ValueError(message)


def source_tree():
    files = sorted((p for directory in ("cmd", "internal")
                    for p in (ROOT / "remake" / directory).rglob("*.go")),
                   key=lambda p: p.relative_to(ROOT / "remake").as_posix())
    encoded = "".join(f"{p.relative_to(ROOT / 'remake').as_posix()}\0{digest(p.read_bytes())}\n"
                      for p in files).encode()
    return digest(encoded), len(files)


class SourceVerifier:
    def __init__(self, original, modern, catalog, preflights=()):
        self.original, self.modern = original, modern
        self.catalog = catalog
        self.preflights = preflights
        self.manifest_path = original / "manifest.json"
        manifest = json.loads(self.manifest_path.read_text())
        reference = json.loads((ROOT / "docs/data/fd2-reference-files.json").read_text())
        for name in ("FD2.EXE", "FIGANI.DAT", "FDOTHER.DAT", "BG.DAT", "TAI.DAT", "FDICON.B24"):
            pinned = next(x for x in reference["files"] if x["file"] == name)
            actual = next(x for x in manifest["source_set"] if x["file"] == name)
            require(pinned == actual, f"原版來源指紋不同：{name}")
        self.entries = {x["path"]: x for x in manifest["assets"]}
        self.native_cache, self.hd_cache, self.file_hashes = {}, {}, {}
        self.lmi_hd_cache, self.mask_hd_cache = {}, {}

    def original_file(self, relative):
        relative = Path(relative)
        require(not relative.is_absolute() and ".." not in relative.parts, "來源路徑越界")
        key = relative.as_posix()
        require(key in self.entries, f"原版檔案未登錄：{key}")
        path = self.original / relative
        if key not in self.file_hashes:
            actual = digest(path.read_bytes())
            require(actual == self.entries[key]["sha256"], f"原版檔案SHA不同：{key}")
            self.file_hashes[key] = actual
        return path

    def native(self, archive, resource, index):
        key = archive, resource, index
        if key not in self.native_cache:
            require(archive in ROLES, f"未知命名空間：{archive}")
            prefix = archive[:-4]
            if archive in ("FIGANI.DAT", "FDOTHER.DAT"):
                relative = Path("animations") / f"{prefix}_{resource:03d}" / "animation.json"
                meta = json.loads(self.original_file(relative).read_text())
                require(meta["source"] == {"file": archive, "resource": resource}, "動畫來源不同")
                require(0 <= index < len(meta["frames"]), f"原版畫格越界：{key}")
                frame = meta["frames"][index]
                pixels = self.original_file(relative.parent / frame["path"])
                mask = self.original_file(relative.parent / frame["mask_path"])
                width, height = frame["width"], frame["height"]
                descriptor = {"X": frame["x"], "Y": frame["y"], "Delay": frame["delay_native"],
                              "Raw4": frame["raw_byte_4"], "Raw5": frame["raw_byte_5"], "Raw7": frame["raw_byte_7"]}
            else:
                require(index == 0, "背景或台座只允許畫格0")
                relative = Path("surfaces") / f"{prefix}_{resource:03d}" / "resource.json"
                meta = json.loads(self.original_file(relative).read_text())
                require(meta["source"]["file"] == archive and meta["source"]["resource"] == resource,
                        "背景來源不同")
                width, height = meta["width"], meta["height"]
                descriptor = {}
                pixels = self.original_file(relative.parent / meta["frame"])
                mask = self.original_file(relative.parent / meta["mask"])
            with Image.open(pixels) as picture, Image.open(mask) as coverage:
                require(picture.mode == "P" and coverage.mode == "L", "原版PNG模式不同")
                require(picture.size == coverage.size == (width, height), "原版PNG尺寸不同")
                self.native_cache[key] = {
                    "width": width, "height": height,
                    "pixels": digest(picture.tobytes()),
                    "pixel_bytes": picture.tobytes(),
                    # figani/separated.go stores a binary 0/1 mask. The
                    # fdother surface adapter retains its 0/255 coverage.
                    "mask": digest(bytes(int(v != 0) for v in coverage.tobytes())
                                   if archive in ("FIGANI.DAT", "FDOTHER.DAT") else coverage.tobytes()),
                    "metadata_sha256": self.file_hashes[relative.as_posix()],
                    "visible_pixels": sum(v != 0 for v in coverage.tobytes()),
                    "descriptor": descriptor,
                }
        return self.native_cache[key]

    def hd(self, asset, index, native):
        key = asset["asset_id"], index
        if key not in self.hd_cache:
            require(asset["source_metadata_sha256"] == native["metadata_sha256"], "HD綁定來源metadata不同")
            if asset["role"] == "battle_surface_hd":
                require(index == 0, "HD背景畫格不同")
                filename, expected = asset["file"], asset["sha256"]
            else:
                require(0 <= index < len(asset["files"]), "HD畫格越界")
                filename, expected = asset["files"][index], asset["frame_sha256"][index]
            relative = Path(filename)
            require(not relative.is_absolute() and ".." not in relative.parts, "HD路徑越界")
            path = self.modern / relative
            require(digest(path.read_bytes()) == expected, f"HD成品SHA不同：{filename}")
            with Image.open(path) as picture:
                require(picture.mode == "RGBA", "HD模式不同")
                require(picture.size == (native["width"] * 6, native["height"] * 6), "HD尺寸不同")
            self.hd_cache[key] = {"file": filename, "sha256": expected}
        return self.hd_cache[key]

    def map_assets(self, catalog):
        overlays, masks = {}, {}
        for asset in catalog["assets"]:
            if asset["role"] == "map_overlay_hd_set":
                require(asset["source_archive"] == "FDOTHER.DAT", "LMI命名空間不同")
                for slot, frame in enumerate(asset["native_lmi_frames"]):
                    key = asset["source_resource"], frame["index"]
                    require(key not in overlays, "LMI同entry重複登錄")
                    overlays[key] = asset, slot, frame
            elif asset["role"] == "map_sprite_hd_set":
                require(asset["source_group"] not in masks, "HD人物group重複登錄")
                masks[asset["source_group"]] = asset
        return overlays, masks

    def hd_lmi(self, layer, native, assets):
        key = layer["Resource"], layer["Entry"]
        require(layer["Kind"] == "LMI1", "不能將LMI1當動畫畫格")
        require(key in assets, f"地圖尾段LMI缺HD登錄：{key}")
        require((layer["Width"], layer["Height"], layer["PixelsSHA256"]) ==
                (native["width"], native["height"], native["pixels_sha256"]), "LMI原始來源不同")
        asset, slot, description = assets[key]
        require(asset["source_metadata_sha256"] == native["metadata_sha256"], "LMI metadata綁定不同")
        require(description == {k: native[k] for k in ("index", "width", "height", "pixels_sha256")},
                "HD登錄的原生LMI描述子不同")
        cache_key = asset["asset_id"], slot
        if cache_key not in self.lmi_hd_cache:
            path = self.modern / asset["files"][slot]
            require(digest(path.read_bytes()) == asset["frame_sha256"][slot], "LMI成品SHA不同")
            with Image.open(path) as im:
                require(im.mode == "RGBA" and im.size == (native["width"] * 6, native["height"] * 6),
                        "LMI成品尺寸或模式不同")
            self.lmi_hd_cache[cache_key] = key

    def hd_mask(self, source, assets):
        group, frame = source["Group"], source["Frame"]
        require(0 <= frame < 3 and group in assets, "人物遮罩缺HD或cycle不同")
        ordinal = group * 12 + frame
        folder = Path("sprites/fdicon") / f"sprite_{ordinal:04d}"
        for name, field in (("frame.png", "PixelsSHA256"), ("mask.png", "MaskSHA256")):
            path = self.original_file(folder / name)
            entry = self.entries[(folder / name).as_posix()]
            require(entry["source_file"] == "FDICON.B24" and entry["source_frame"] == ordinal, "原人物來源或索引不同")
            with Image.open(path) as im:
                require(im.size == (24, 24), "原人物尺寸不同")
                pixels = im.tobytes()
                if name == "mask.png":
                    require(im.mode == "L", "原人物遮罩模式不同")
                    pixels = bytes(int(v != 0) for v in pixels)
                else:
                    require(im.mode == "P", "原人物索引圖模式不同")
                require(digest(pixels) == source[field], "原人物遮罩圖源SHA不同")
        asset = assets[group]
        key = group, frame
        if key not in self.mask_hd_cache:
            path = self.modern / asset["files"][frame]
            require(digest(path.read_bytes()) == asset["frame_sha256"][frame], "人物遮罩HD SHA不同")
            with Image.open(path) as im:
                require(im.mode == "RGBA" and im.size == (144, 144), "人物遮罩HD尺寸不同")
                require(im.getchannel("A").getextrema() == (0, 255), "人物遮罩HD缺透明及核心")
            self.mask_hd_cache[key] = asset["frame_sha256"][frame]

    def verify(self, observed, catalog=None):
        catalog = self.catalog if catalog is None else catalog
        require(observed["status"] == "CONTROLLED-CONSUMER-OBSERVATIONS", "觀測狀態不同")
        tree_sha, tree_count = source_tree()
        require(observed["source_tree_sha256"] == tree_sha and observed["source_file_count"] == tree_count,
                "匯出後Go來源已有變動，須重新匯出")
        for key, path in {"catalog": ROOT / "remake/assets/themes/modern/catalog.json",
                          "reference": ROOT / "docs/data/fd2-reference-files.json",
                          "original_manifest": self.manifest_path}.items():
            require(observed["input_hashes"][key] == digest(path.read_bytes()), f"匯出來源雜湊不同：{key}")
        for key in ("gpu_verified", "normal_player_verified", "original_parity_verified", "full_battle_hd_complete"):
            require(observed.get(key) is False, f"不可提升證據等級：{key}")
        rows = observed["cases"]
        require(len(rows) == len(EXPECTED_CASES) and {r["ID"] for r in rows} == EXPECTED_CASES,
                "正式消費端代表案例缺失或重複")
        overlays, masks = self.map_assets(catalog)
        ranges = {}
        for row in rows:
            for source in row.get("MapSources") or []:
                for layer in source.get("Layers") or []:
                    ranges.setdefault(layer["Resource"], set()).add(layer["Entry"])
        require(ranges, "缺召喚地圖尾段來源")
        observed_lmi_keys = {(k, i) for k, values in ranges.items() for i in values}
        if self.preflights:
            require(len(self.preflights) == 4 and {p["command"] for p in self.preflights} == {32, 33, 34, 35},
                    "完整主題預備收據須含四種召喚")
        for proof in self.preflights:
            require(proof["status"] == "PASS-COMPOUND-MAP-SOURCES" and proof["native_fallback_layers"] == 0,
                    "完整主題預備失敗或有未替換LMI")
            require(all(proof[k] is True for k in ("native_reconstruction", "rng_unchanged", "unit_state_unchanged"))
                    and proof["atomic_rejections"] >= 2, "預備來源或原子拒收證據缺失")
            require(sum(proof["phases"].values()) == proof["map_frames"] and proof["phases"]["map-ramp"] == 41,
                    "預備地圖階段不完整")
            require(sum(proof["sources"].values()) == proof["registered_layers"], "預備LMI計數不同")
            for source, count in proof["sources"].items():
                resource, index = map(int, source.split(":"))
                require(resource in (5, 6) and count > 0, "預備LMI索引不同")
                ranges.setdefault(resource, set()).add(index)
        lmi_audit, _ = audit_lmi_sources(self.original, {k: sorted(v) for k, v in ranges.items()})
        lmi_native = {(bank["resource"], frame["index"]): {**frame, "metadata_sha256": bank["metadata_sha256"]}
                      for bank in lmi_audit["resources"] for frame in bank["frames"]}
        for proof in self.preflights:
            for source in proof["sources"]:
                resource, index = map(int, source.split(":"))
                native = lmi_native[resource, index]
                # The preflight reports an actual producer entry index. Raw
                # reconstruction and HD identity are checked independently;
                # only direct observations carry an extra producer pixel SHA.
                self.hd_lmi({"Kind": "LMI1", "Resource": resource, "Entry": index,
                             "Width": native["width"], "Height": native["height"],
                             "PixelsSHA256": native["pixels_sha256"]}, native, overlays)
        assets = {}
        for a in catalog["assets"]:
            namespace = a.get("source_archive")
            if namespace in ROLES and a.get("role") == ROLES[namespace]:
                k = namespace, a["source_resource"]
                require(k not in assets, f"HD同命名空間重複：{k}")
                assets[k] = a
        usages, cases, empty_sources = set(), [], set()
        for row in rows:
            expected_id = -1 if row["ID"].startswith("physical/") else int(row["ID"].split("/")[1])
            require(row["Command"] == expected_id, "案例指令ID不同")
            require(row["Frames"] > 0 and row["Sources"], f"空白消費端：{row['ID']}")
            require(sum(row["Phases"].values()) == row["Frames"], "演出階段計數不同")
            counts = Counter()
            for source in row["Sources"]:
                key = source["Archive"], source["Resource"], source["Frame"]
                require(source["Count"] > 0 and source["Phase"] in row["Phases"], "來源觀測次數或階段不同")
                native = self.native(*key)
                require((source["Width"], source["Height"]) == (native["width"], native["height"]),
                        f"原生來源尺寸不同：{row['ID']} {key}")
                require(all(source[k] == v for k, v in native["descriptor"].items()),
                        f"原生描述子座標／節拍／raw欄位不同：{row['ID']} {key}")
                shader = source["PackedShader"]
                pixels_sha = native["pixels"]
                if shader:
                    require(shader & 255 == 0xb0 and 2 <= shader >> 8 <= 8, "packed轉色值不同")
                    pixels_sha = digest(bytes(((v + (shader >> 8)) & 7) + 0xb0
                                              for v in native["pixel_bytes"]))
                require(source["PixelsSHA256"] == pixels_sha and source["MaskSHA256"] == native["mask"],
                        f"原生來源像素或遮罩不同：{row['ID']} {key}")
                if key[:2] in assets:
                    self.hd(assets[key[:2]], key[2], native)
                else:
                    require(native["visible_pixels"] == 0, f"HD缺登錄：{row['ID']} {key}")
                    empty_sources.add(key)
                usages.add(key)
                counts[key[0]] += source["Count"]
            cases.append({"id": row["ID"], "constructor": row["Constructor"], "scope": row["Scope"],
                          "frames": row["Frames"], "unbound_battle_frames": row["UnboundFrames"],
                          "phases": row["Phases"], "source_observations_by_archive": dict(counts),
                          "unique_source_bindings": len(row["Sources"]),
                          "source_sequence_sha256": row["SequenceSHA256"]})
            map_sources = row.get("MapSources") or []
            map_count = sum(s["Count"] for s in map_sources)
            require(map_count == row["MapFrames"], "地圖來源次數不同")
            if row["ID"].startswith("compound/"):
                require(map_count > 0 and map_count == row["UnboundFrames"], "召喚尾段仍有未分類來源")
            else:
                require(map_count == 0, "非召喚案例出現未指定地圖來源")
            layer_count, mask_count, map_phases = 0, 0, Counter()
            for source in map_sources:
                require(source["Count"] > 0 and source["Phase"] in row["Phases"], "地圖來源階段不同")
                require(all(len(source[k]) == 64 for k in ("SceneSHA256", "SourceSHA256", "PixelsSHA256")),
                        "地圖來源seal或畫面雜湊缺失")
                require(0 <= source["Mix"] <= 1 and all(0 <= v <= 1 for v in source["Target"]), "地圖混色範圍不同")
                map_phases[source["Phase"]] += source["Count"]
                for order, layer in enumerate(source.get("Layers") or []):
                    require(layer["Order"] == order, "LMI層序不同")
                    self.hd_lmi(layer, lmi_native[(layer["Resource"], layer["Entry"])], overlays)
                    layer_count += source["Count"]
                for mask in source.get("Masks") or []:
                    self.hd_mask(mask, masks)
                    mask_count += source["Count"]
            if map_count:
                require(map_phases["map-ramp"] == 41, "召喚地圖ramp不完整")
            cases[-1].update(map_frames=map_count, map_phases=dict(map_phases), lmi_layer_observations=layer_count,
                             mask_source_observations=mask_count,
                             unclassified_frames=row["UnboundFrames"] - map_count)
        return {"cases": cases, "unique_archive_resource_frames": len(usages),
                "map_tail_lmi_hd_entries": len(self.lmi_hd_cache),
                "map_tail_direct_observed_lmi_entries": len(observed_lmi_keys),
                "map_tail_mask_hd_frames": len(self.mask_hd_cache),
                "map_tail_preflight_commands": [p["command"] for p in self.preflights],
                "map_tail_lmi_sources": {str(k): sorted(v) for k, v in ranges.items()},
                "map_tail_native_lmi_audit": lmi_audit,
                "empty_native_sources_without_hd": [list(k) for k in sorted(empty_sources)],
                "resources_by_archive": {archive: sorted({k[1] for k in usages if k[0] == archive})
                                         for archive in sorted({k[0] for k in usages})},
                "verified_hd_frames": len(self.hd_cache), "verified_original_files": len(self.file_hashes)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--observed", required=True, type=Path)
    parser.add_argument("--original-pack", required=True, type=Path)
    parser.add_argument("--modern-pack", required=True, type=Path)
    parser.add_argument("--catalog", type=Path, default=ROOT / "remake/assets/themes/modern/catalog.json")
    parser.add_argument("--out", required=True, type=Path)
    parser.add_argument("--self-test", action="store_true")
    parser.add_argument("--map-preflight", action="append", default=[], type=Path,
                        help="既有完整主題TestModernCompoundMapAllSources的收據，可重複指定四種召喚")
    args = parser.parse_args()
    observed = json.loads(args.observed.read_text())
    require(digest(args.catalog.read_bytes()) == observed["input_hashes"]["catalog"],
            "核對清冊不是匯出時綁定的正式catalog")
    catalog = json.loads(args.catalog.read_text())
    preflights = [json.loads(p.read_text()) for p in args.map_preflight]
    verifier = SourceVerifier(args.original_pack, args.modern_pack, catalog, preflights)
    result = verifier.verify(observed)
    rejected = []
    if args.self_test:
        for kind in ("wrong-archive", "wrong-resource", "wrong-frame", "wrong-pixels", "wrong-mask", "missing-case", "false-gpu", "missing-hd", "wrong-code", "wrong-command", "wrong-packed", "wrong-descriptor",
                     "missing-map-route", "wrong-lmi-kind", "wrong-lmi-pixels", "wrong-lmi-entry", "missing-lmi-hd", "wrong-map-mask", "missing-map-mask-hd"):
            bad, altered = deepcopy(observed), deepcopy(catalog)
            source = bad["cases"][0]["Sources"][0]
            if kind == "wrong-archive":
                source["Archive"] = "UNKNOWN.DAT"
            elif kind == "wrong-resource":
                source["Resource"] = 99999
            elif kind == "wrong-frame":
                source["Frame"] = 99999
            elif kind == "wrong-pixels":
                source["PixelsSHA256"] = "0" * 64
            elif kind == "wrong-mask":
                source["MaskSHA256"] = "0" * 64
            elif kind == "missing-case":
                bad["cases"].pop()
            elif kind == "false-gpu":
                bad["gpu_verified"] = True
            elif kind == "missing-hd":
                altered["assets"] = [a for a in altered["assets"] if
                                     (a.get("source_archive"), a.get("source_resource")) !=
                                     (source["Archive"], source["Resource"])]
            elif kind == "wrong-code":
                bad["source_tree_sha256"] = "0" * 64
            elif kind == "wrong-command":
                bad["cases"][0]["Command"] = 999
            elif kind == "wrong-packed":
                source["PackedShader"] = 1
            elif kind == "wrong-descriptor":
                source = next(s for r in bad["cases"] for s in r["Sources"] if s["Archive"] == "FIGANI.DAT")
                source["Raw7"] ^= 1
            elif kind == "missing-map-route":
                next(r for r in bad["cases"] if r.get("MapSources"))["MapSources"].pop()
            elif kind.startswith("wrong-lmi") or kind == "missing-lmi-hd":
                layer = next(layer for r in bad["cases"] for s in r.get("MapSources") or [] for layer in s.get("Layers") or [])
                if kind == "wrong-lmi-kind":
                    layer["Kind"] = "animation"
                elif kind == "wrong-lmi-pixels":
                    layer["PixelsSHA256"] = "0" * 64
                elif kind == "wrong-lmi-entry":
                    layer["Entry"] = 99999
                else:
                    altered["assets"] = [a for a in altered["assets"] if not (
                        a["role"] == "map_overlay_hd_set" and a["source_resource"] == layer["Resource"] and
                        any(f["index"] == layer["Entry"] for f in a["native_lmi_frames"]))]
            elif kind in ("wrong-map-mask", "missing-map-mask-hd"):
                mask = next(mask for r in bad["cases"] for s in r.get("MapSources") or [] for mask in s.get("Masks") or [])
                if kind == "wrong-map-mask":
                    mask["PixelsSHA256"] = "0" * 64
                else:
                    altered["assets"] = [a for a in altered["assets"] if not (
                        a["role"] == "map_sprite_hd_set" and a["source_group"] == mask["Group"])]
            try:
                verifier.verify(bad, altered)
            except (ValueError, KeyError, IndexError, FileNotFoundError):
                rejected.append(kind)
            else:
                raise ValueError(f"反例未拒收：{kind}")
    result.update(status="PASS-CONTROLLED-CONSUMER-SOURCES", schema_version=1,
                  scope="受控正式建構器及逐拍owner的archive與召喚地圖尾段LMI／人物遮罩來源核對；未新增GPU或正常玩家收據。",
                  gpu_verified=False, normal_player_verified=False, original_parity_verified=False,
                  full_battle_hd_complete=False,
                  unobserved=observed["unobserved"] + ["non-archive layers and pixels baked into Base are not asset-coverage claims"],
                  rejected_mutants=rejected,
                  inputs={"observed": {"path": str(args.observed), "sha256": digest(args.observed.read_bytes())},
                          "go_source_tree_sha256": observed["source_tree_sha256"],
                          "catalog_sha256": digest(args.catalog.read_bytes()),
                          "map_preflights": [{"path": str(p), "sha256": digest(p.read_bytes())} for p in args.map_preflight],
                          "original_manifest_sha256": digest(verifier.manifest_path.read_bytes()),
                          "verifier_sha256": digest(Path(__file__).read_bytes())})
    args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": result["status"], "cases": len(result["cases"]),
                      "unique_frames": result["unique_archive_resource_frames"], "rejected": rejected}, ensure_ascii=False))


if __name__ == "__main__":
    main()
