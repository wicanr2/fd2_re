#!/usr/bin/env python3
"""由 extract_all.py 的輸出建立可驗證的分離素材清冊。

這個工具不解碼、不修改原始檔，也不把 ``raw/`` 當成正式 runtime 素材。
它只掃描已存在的輸出，將可辨識的輸出列入 ``assets``，其餘每個 raw
resource 同樣列入 ``assets``，但狀態標示為 ``intentionally_raw``，不會冒充
正式 runtime 素材。原始目錄若有提供，會先依 reference manifest 核對大小、MD5、
SHA-256；核對失敗即停止，不產生看似可信的清冊。

用法：
    python3 tools/generate_separated_asset_manifest.py PACK ROOT-REF.json \
        [--original-dir FLAME2] [--output manifest.json]
"""

from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from pathlib import Path


SOURCE_BY_CONTAINER = {
    "ANI": "ANI.DAT", "BG": "BG.DAT", "DATO": "DATO.DAT",
    "FDFIELD": "FDFIELD.DAT", "FDICON": "FDICON.B24", "FDMUS": "FDMUS.DAT",
    "FDOTHER": "FDOTHER.DAT", "FDSHAP": "FDSHAP.DAT", "FDTXT": "FDTXT.DAT",
    "FIGANI": "FIGANI.DAT", "TAI": "TAI.DAT", "TITLE": "TITLE.DAT",
}
CONTAINER_RE = re.compile(r"^(ANI|BG|DATO|FDFIELD|FDMUS|FDOTHER|FDSHAP|FDTXT|FIGANI|TAI|TITLE)_(\d+)\.bin$", re.I)
FRAME_RE = re.compile(r"^frame[_-]?(\d+)", re.I)
HEX = re.compile(r"^[0-9a-f]{64}$")


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def source_entries(reference: Path, original_dir: Path | None) -> tuple[list[dict], list[str]]:
    doc = json.loads(reference.read_text(encoding="utf-8"))
    entries = doc.get("files")
    if not isinstance(entries, list) or not entries:
        raise ValueError("reference manifest 缺少非空 files 陣列")
    errors: list[str] = []
    result: list[dict] = []
    for item in entries:
        if not isinstance(item, dict) or not isinstance(item.get("file"), str):
            raise ValueError("reference manifest 含有無效來源項目")
        result.append({k: item[k] for k in ("file", "size", "md5", "sha256")})
        if original_dir is None:
            continue
        path = original_dir / item["file"]
        if not path.is_file():
            errors.append(f"缺少原始來源：{item['file']}")
            continue
        if path.stat().st_size != item.get("size"):
            errors.append(f"來源大小不符：{item['file']}")
        md5 = hashlib.md5(path.read_bytes()).hexdigest()
        if md5 != item.get("md5"):
            errors.append(f"來源 MD5 不符：{item['file']}")
        if sha256(path) != item.get("sha256"):
            errors.append(f"來源 SHA-256 不符：{item['file']}")
    return result, errors


def parse_raw(path: Path) -> tuple[str | None, int | None]:
    match = CONTAINER_RE.match(path.name)
    if not match:
        return None, None
    return match.group(1).upper(), int(match.group(2))


def classify(path: Path) -> str | None:
    parts = path.parts
    if not parts:
        return None
    top = parts[0].lower()
    suffix = path.suffix.lower()
    if top == "images" and suffix == ".png":
        name = path.stem.upper()
        if name.startswith("FDFIELD_"):
            return "map"
        if name.startswith("FIGANI_"):
            return "battle_animation"
        if name.startswith("BG_"):
            return "background"
        if name.startswith("TITLE_") or name.startswith("FDOTHER_") or name.startswith("TAI_"):
            return "ui"
        if name.startswith("FDSHAP_"):
            return "tileset"
        return "metadata"
    if top == "animations":
        if suffix == ".png":
            return "battle_animation"
        if path.name.lower() == "animation.json":
            return "metadata"
        return None
    if top == "portraits" and suffix == ".png":
        return "portrait"
    if top == "maps":
        return "map" if suffix == ".json" else "tileset" if suffix == ".png" else None
    if top == "fonts":
        return "font" if suffix in {".png", ".json"} else None
    if top == "ui" and suffix == ".png":
        return "ui"
    if top == "palette" and suffix == ".json":
        return "metadata"
    if top == "music":
        return "music" if suffix == ".ogg" else "music" if suffix in {".mid", ".xmi"} else None
    if top in {"audio", "sfx"}:
        return "sfx" if suffix == ".ogg" else "blocked" if suffix not in {".json"} else None
    if top in {"text", "dialogue"} and suffix == ".json":
        return "text"
    return None


def infer_provenance(path: Path) -> tuple[str | None, int | None, int | None]:
    """回傳來源檔、resource、frame；無法證明時保留 None。"""
    stem = path.stem
    match = re.search(r"(?:^|[_/])(ANI|BG|DATO|FDFIELD|FDMUS|FDOTHER|FDSHAP|FDTXT|FIGANI|TAI|TITLE)[_-](\d+)", stem, re.I)
    resource = int(match.group(2)) if match else None
    container = match.group(1).upper() if match else None
    frame = None
    for part in path.parts:
        frame_match = FRAME_RE.match(part)
        if frame_match:
            frame = int(frame_match.group(1))
            break
    if frame is None:
        frame_match = re.search(r"_f(\d+)(?:_mask)?$", stem, re.I)
        if frame_match:
            frame = int(frame_match.group(1))
    if container:
        return SOURCE_BY_CONTAINER.get(container), resource, frame
    if path.parts and path.parts[0].lower() == "animations":
        match = re.match(r"FIGANI[_-](\d+)", path.parts[1], re.I) if len(path.parts) > 1 else None
        if match:
            return "FIGANI.DAT", int(match.group(1)), frame
    if path.parts and path.parts[0].lower() == "portraits":
        match = re.search(r"DATO[_-](\d+)", stem, re.I)
        if match:
            return "DATO.DAT", int(match.group(1)), frame
    if path.as_posix().lower() == "palette/fdother_000.json":
        return "FDOTHER.DAT", 0, None
    if len(path.parts) == 3 and path.parts[0].lower() == "ui" and path.parts[1].lower() == "action_cells":
        match = re.fullmatch(r"cell_(\d+)\.png", path.name, re.I)
        if match:
            return "FDOTHER.DAT", 2, int(match.group(1))
    return None, resource, frame


def stable_asset_id(kind: str, relative: str) -> str:
    # 路徑是輸出身份的一部分；統一斜線與大小寫以避免平台差異。
    normalized = relative.replace("\\", "/").lower()
    safe = re.sub(r"[^a-z0-9._/-]+", "-", normalized).strip("/")
    return f"{kind}/{safe}"


def build_manifest(pack_root: Path, reference: Path, original_dir: Path | None = None) -> dict:
    sources, source_errors = source_entries(reference, original_dir)
    if source_errors:
        raise ValueError("；".join(source_errors))
    source_names = {entry["file"] for entry in sources}
    assets: list[dict] = []
    for path in sorted(pack_root.rglob("*")):
        if not path.is_file() or "raw" in path.relative_to(pack_root).parts:
            continue
        relative = path.relative_to(pack_root).as_posix()
        if relative in {"manifest.json", "INDEX.md"}:
            continue
        kind = classify(path.relative_to(pack_root))
        if kind is None:
            continue
        source_file, resource, frame = infer_provenance(path.relative_to(pack_root))
        if source_file not in source_names:
            # 沒有可回查來源的輸出不能進入正式清冊；避免 null provenance 被
            # 後續工具誤認為已證實。這類檔案由呼叫端另行診斷即可。
            continue
        item = {
            "asset_id": stable_asset_id(kind, relative),
            "kind": kind,
            "path": relative,
            "sha256": sha256(path),
            "source_file": source_file,
            "status": "blocked" if path.suffix.lower() in {".mid", ".xmi"} else "exported",
            "evidence": "confirmed",
        }
        if resource is not None:
            item["source_resource"] = resource
        if frame is not None:
            item["source_frame"] = frame
        assets.append(item)

    # raw resource 也是 assets[] 的明確記錄，但永遠不是 exported；validator
    # 因此能檢查其路徑安全性，同時不會把它當作 runtime 可消費素材。
    raw_root = pack_root / "raw"
    if raw_root.is_dir():
        for path in sorted(raw_root.rglob("*.bin")):
            relative = path.relative_to(pack_root).as_posix()
            container, resource = parse_raw(path)
            source_file = SOURCE_BY_CONTAINER.get(container or "")
            if source_file not in source_names:
                # 無法與 reference manifest 對應的 raw 不可捏造 source_file。
                continue
            status = "intentionally_raw"
            entry = {
                "asset_id": stable_asset_id("metadata", relative),
                "kind": "metadata",
                "path": relative,
                "sha256": sha256(path),
                "source_file": source_file,
                "source_resource": resource,
                "status": status,
                "evidence": "confirmed",
            }
            assets.append(entry)

    pack_id = pack_root.name
    return {
        "schema_version": 1,
        "pack_id": pack_id,
        "source_set": sources,
        "assets": assets,
        "relationships": [],
        "generated_by": {
            "tool": "generate_separated_asset_manifest.py",
            "version": "1",
        },
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("pack_root", type=Path)
    parser.add_argument("reference", type=Path)
    parser.add_argument("--original-dir", type=Path)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args(argv)
    output = args.output or args.pack_root / "manifest.json"
    try:
        manifest = build_manifest(args.pack_root, args.reference, args.original_dir)
        output.write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"錯誤：無法建立分離素材清冊：{exc}", file=sys.stderr)
        return 1
    raw_count = sum(item["status"] == "intentionally_raw" for item in manifest["assets"])
    print(f"分離素材清冊已建立：{output}（輸出素材 {len(manifest['assets']) - raw_count} 筆，raw 清冊 {raw_count} 筆）")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
