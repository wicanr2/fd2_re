"""共用的分離音樂 catalog／外部 assets root 驗證契約。"""

from __future__ import annotations

import hashlib
import json
from pathlib import Path, PurePosixPath

from generate_music_catalog import TRACKS, inspect_vorbis


PROFILES = ("fm", "mt32")


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def _safe_relative(value: object) -> bool:
    if not isinstance(value, str) or not value or "\\" in value:
        return False
    path = PurePosixPath(value)
    return not path.is_absolute() and ".." not in path.parts


def validate_music_assets(
    assets_root: Path,
    expected_source: dict | None = None,
    expected_catalog_sha256: str | None = None,
) -> tuple[dict | None, list[str]]:
    errors: list[str] = []
    catalog_path = assets_root / "music_catalog.json"
    try:
        raw = catalog_path.read_bytes()
        catalog = json.loads(raw)
    except (OSError, ValueError) as exc:
        return None, [f"無法讀取音樂 catalog：{exc}"]
    catalog_hash = hashlib.sha256(raw).hexdigest()
    if expected_catalog_sha256 is not None and catalog_hash != expected_catalog_sha256:
        errors.append("音樂 catalog SHA-256 不符")
    if catalog.get("schema") != "fd2_music_catalog" or catalog.get("schema_version") != 1:
        errors.append("音樂 catalog schema 必須為 fd2_music_catalog version 1")
    source = catalog.get("source")
    if not isinstance(source, dict):
        errors.append("音樂 catalog source 無效")
        source = {}
    if expected_source is not None:
        wanted = {key: expected_source.get(key) for key in ("file", "size", "md5", "sha256")}
        actual = {key: source.get(key) for key in ("file", "size", "md5", "sha256")}
        if actual != wanted:
            errors.append("音樂 catalog FDMUS source identity 不符")
    loop = catalog.get("loop")
    if not isinstance(loop, dict) or loop != {
        "mode": "whole_file_runtime_repeat",
        "accepted_counts": [0, 1],
        "seam_evidence": "unknown",
        "evidence_level": "strong_inference_e1",
    }:
        errors.append("音樂 catalog loop 契約不符")
    profiles = catalog.get("profiles")
    if not isinstance(profiles, dict) or set(profiles) != set(PROFILES):
        errors.append("音樂 catalog profile 集合不符")
    else:
        for profile in PROFILES:
            metadata = profiles[profile]
            if (
                not isinstance(metadata, dict)
                or not isinstance(metadata.get("render_pipeline"), str)
                or not metadata["render_pipeline"]
                or metadata.get("provenance_status") != "incomplete_legacy_render"
                or not isinstance(metadata.get("rights_note"), str)
                or not metadata["rights_note"]
            ):
                errors.append(f"音樂 catalog profile {profile} metadata 不符")
    tracks = catalog.get("tracks")
    if not isinstance(tracks, list) or len(tracks) != len(TRACKS):
        errors.append("音樂 catalog track 數量不符")
        tracks = []
    for position, resource in enumerate(TRACKS):
        if position >= len(tracks) or not isinstance(tracks[position], dict):
            continue
        track = tracks[position]
        track_id = f"FDMUS_{resource:03d}"
        if track.get("track_id") != track_id or track.get("resource_index") != resource:
            errors.append(f"音樂 catalog track {position} identity 不符")
            continue
        renders = track.get("renders")
        if not isinstance(renders, dict) or set(renders) != set(PROFILES):
            errors.append(f"音樂 catalog {track_id} render 集合不符")
            continue
        for profile in PROFILES:
            render = renders[profile]
            if not isinstance(render, dict):
                errors.append(f"音樂 catalog {track_id}/{profile} render 無效")
                continue
            relative = render.get("path")
            expected_path = f"music_{profile}/{track_id}.ogg"
            if not _safe_relative(relative) or relative != expected_path:
                errors.append(f"音樂 catalog {track_id}/{profile} 路徑不符")
                continue
            output = assets_root / relative
            try:
                info = output.stat()
                digest = _sha256(output)
                geometry = inspect_vorbis(output)
            except (OSError, ValueError) as exc:
                errors.append(f"音樂 render {relative} 無效：{exc}")
                continue
            has_loop_tags = geometry.pop("has_loop_tags")
            actual = {
                "bytes": info.st_size,
                "sha256": digest,
                "codec": "vorbis",
                **geometry,
            }
            expected = {key: render.get(key) for key in actual}
            if actual != expected or has_loop_tags:
                errors.append(f"音樂 render {relative} identity／geometry 不符")
    summary = {
        "kind": "fd2_music_catalog",
        "asset_root": "runtime_assets",
        "catalog_path": "music_catalog.json",
        "catalog_bytes": len(raw),
        "catalog_sha256": catalog_hash,
        "schema_version": 1,
        "source_file": source.get("file"),
        "profiles": len(PROFILES),
        "tracks": len(TRACKS),
        "renders": len(PROFILES) * len(TRACKS),
    }
    return summary, errors
