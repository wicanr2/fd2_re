#!/usr/bin/env python3
"""以既有 FM／MT-32 OGG 產生可重生、嚴格的 FD2 音樂 catalog。"""

from __future__ import annotations

import argparse
import hashlib
import json
import struct
from pathlib import Path


TRACKS = (1, 3, 4, 6, 8, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19)
FDMUS = {
    "file": "FDMUS.DAT",
    "size": 80367,
    "md5": "4dfa214125edcc4658acbba2e1201a28",
    "sha256": "4105ebde543fe1c497e852728f6bc333bda80edeb7fb3671e487504bee74e998",
}
PROFILES = {
    "fm": {
        "render_pipeline": "FDMUS.DAT/XMI -> MIDI -> SAMPLE.AD/OPL -> libADLMIDI -> Vorbis OGG",
        "provenance_status": "incomplete_legacy_render",
        "rights_note": "原版衍生音訊；僅記錄既有 render，不主張逐樣本或硬體時序一致。",
    },
    "mt32": {
        "render_pipeline": "FDMUS.DAT/XMI -> MIDI -> munt + user-owned Roland ROM -> Vorbis OGG",
        "provenance_status": "incomplete_legacy_render",
        "rights_note": "不得散布、嵌入或在 catalog 記錄 Roland ROM；僅記錄既有 render。",
    },
}


class CatalogError(ValueError):
    pass


def inspect_vorbis(path: Path) -> dict[str, int | str | bool]:
    raw = path.read_bytes()
    offset = 0
    packet = bytearray()
    packets: list[bytes] = []
    last_granule = None
    while offset < len(raw):
        if offset + 27 > len(raw) or raw[offset : offset + 4] != b"OggS":
            raise CatalogError(f"{path}: invalid Ogg page at {offset}")
        if raw[offset + 4] != 0:
            raise CatalogError(f"{path}: unsupported Ogg version")
        segments = raw[offset + 26]
        header_end = offset + 27 + segments
        if header_end > len(raw):
            raise CatalogError(f"{path}: truncated Ogg segment table")
        lacing = raw[offset + 27 : header_end]
        body_end = header_end + sum(lacing)
        if body_end > len(raw):
            raise CatalogError(f"{path}: truncated Ogg page body")
        granule = struct.unpack_from("<Q", raw, offset + 6)[0]
        if granule != 0xFFFFFFFFFFFFFFFF:
            last_granule = granule
        cursor = header_end
        for length in lacing:
            packet.extend(raw[cursor : cursor + length])
            cursor += length
            if length < 255:
                if len(packets) < 3:
                    packets.append(bytes(packet))
                packet.clear()
        offset = body_end
    if offset != len(raw) or packet:
        raise CatalogError(f"{path}: incomplete Ogg packet")
    if not packets or not packets[0].startswith(b"\x01vorbis") or len(packets[0]) < 16:
        raise CatalogError(f"{path}: Vorbis identification packet absent")
    version, channels, sample_rate = struct.unpack_from("<IBI", packets[0], 7)
    if version != 0 or channels != 2 or sample_rate <= 0 or not last_granule:
        raise CatalogError(f"{path}: unsupported Vorbis geometry")
    comments = packets[1] if len(packets) > 1 and packets[1].startswith(b"\x03vorbis") else b""
    upper_comments = comments.upper()
    return {
        "bytes": len(raw),
        "sha256": hashlib.sha256(raw).hexdigest(),
        "codec": "vorbis",
        "channels": channels,
        "sample_rate": sample_rate,
        "pcm_samples": last_granule,
        "duration_ms": (last_granule * 1000 + sample_rate // 2) // sample_rate,
        "has_loop_tags": b"LOOPSTART=" in upper_comments or b"LOOPEND=" in upper_comments,
    }


def build_catalog(assets_root: Path) -> dict:
    tracks = []
    for resource in TRACKS:
        track_id = f"FDMUS_{resource:03d}"
        renders = {}
        for profile in PROFILES:
            rel = Path(f"music_{profile}") / f"{track_id}.ogg"
            geometry = inspect_vorbis(assets_root / rel)
            if geometry.pop("has_loop_tags"):
                raise CatalogError(f"{rel}: unreviewed loop tags are not permitted")
            renders[profile] = {"path": rel.as_posix(), **geometry}
        tracks.append({"track_id": track_id, "resource_index": resource, "renders": renders})
    return {
        "schema": "fd2_music_catalog",
        "schema_version": 1,
        "source": FDMUS,
        "loop": {
            "mode": "whole_file_runtime_repeat",
            "accepted_counts": [0, 1],
            "seam_evidence": "unknown",
            "evidence_level": "strong_inference_e1",
        },
        "profiles": PROFILES,
        "tracks": tracks,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--assets-root", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    catalog = build_catalog(args.assets_root)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(catalog, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
