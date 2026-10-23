#!/usr/bin/env python3
"""Validate the tracked modern-theme catalog and optional private originals."""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import struct
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
CATALOG = ROOT / "remake/assets/themes/modern/catalog.json"
ID_RE = re.compile(r"^[a-z0-9]+(?:[._-][a-z0-9]+)+$")
SHA_RE = re.compile(r"^[0-9a-f]{64}$")
SOURCE_ASSET_RE = re.compile(r"^asset:[A-Za-z0-9][A-Za-z0-9._-]+$")
PROTOTYPE_ROLES = {"portrait_concept", "battlefield_concept", "battle_hud_concept"}
ROLES = PROTOTYPE_ROLES | {
    "story_portrait_frame", "story_portrait_hd_frame", "map_sprite_set", "map_tileset_set",
    "map_sprite_hd_set", "battlefield_tileset_concept", "battle_hud_panel", "battle_action_icon_set",
    "native_terrain_hd_set", "terrain_hd_master", "battle_animation_hd_set", "battle_effect_hd_set", "map_overlay_hd_set",
    "facility_scene_hd", "battle_surface_hd",
}
STATUSES = {"prototype", "concept", "approved", "runtime_candidate", "runtime_ready"}
# Fixed FDICON.B24 source-write inventory, independently checked by
# audit_modern_fdicon_bank.py and fdicon.NativeSourceGroupHasPixels.
NATIVE_FDICON_GROUP_COUNT = 140
NO_SOURCE_WRITE_SELECTORS = {39, 57, 104, 105, 121, 127, 128, 129, 130, 131, 132, 136, 137, 138, 139}
VISIBLE_FDICON_GROUPS = set(range(NATIVE_FDICON_GROUP_COUNT)) - NO_SOURCE_WRITE_SELECTORS
SPRITE_ROLES = {"map_sprite_set", "map_sprite_hd_set"}
MULTI_FILE_ROLES = SPRITE_ROLES | {"battle_action_icon_set", "native_terrain_hd_set", "battle_animation_hd_set", "battle_effect_hd_set", "map_overlay_hd_set"}
ORDINARY_EFFECT_RESOURCES = {18,19,20,21,22,23,24,25,26,27,28,30,32,33,37,38,39,43}


def png_size(path: Path) -> tuple[int, int]:
    raw = path.read_bytes()[:24]
    if len(raw) != 24 or raw[:8] != b"\x89PNG\r\n\x1a\n" or raw[12:16] != b"IHDR":
        raise ValueError(f"{path}: not a PNG")
    return struct.unpack(">II", raw[16:24])


def validate(verify_private: bool, catalog_path: Path | None = None) -> dict:
    data = json.loads((catalog_path or CATALOG).read_text(encoding="utf-8"))
    if set(data) != {"schema_version", "theme_id", "status", "private_root", "assets"}:
        raise ValueError("catalog fields mismatch")
    if data["schema_version"] != 1 or data["status"] != "prototype":
        raise ValueError("unsupported catalog version or status")
    if not ID_RE.fullmatch(data["theme_id"]):
        raise ValueError("invalid theme_id")
    private_root = Path(data["private_root"])
    if private_root.is_absolute() or ".." in private_root.parts:
        raise ValueError("private_root must be a safe repository-relative path")
    assets = data["assets"]
    if not isinstance(assets, list) or not assets:
        raise ValueError("assets must be a non-empty list")
    ids: set[str] = set()
    roles: set[str] = set()
    portrait_speakers, hd_speakers = set(), set()
    parent_sprites, hd_sprites = {}, {}
    terrain_frames = set()
    battle_resources = set()
    facility_resources = set()
    surface_resources = set()
    for entry in assets:
        required = {"asset_id", "role", "status", "file", "width", "height", "sha256", "source_refs"}
        candidate = {"master_file", "consumer_contract", "speaker_id", "frame", "mouth_state"}
        sprite_set = {
            "asset_id", "role", "status", "files", "width", "height", "frame_count",
            "frame_sha256", "source_group", "consumer_contract", "alpha_contract",
            "cycle_policy", "source_refs",
        }
        map_tileset = required | {
            "consumer_contract", "map_id", "tile_width", "tile_height", "columns", "tile_count"
        }
        hud_panel = required | {"master_file", "consumer_contract"}
        action_icons = {
            "asset_id", "role", "status", "master_file", "files", "width", "height",
            "frame_count", "frame_sha256", "semantics", "source_refs", "consumer_contract",
        }
        if not isinstance(entry, dict):
            raise ValueError("asset entry must be an object")
        if entry.get("role") == "battle_surface_hd":
            expected = required | {"master_file", "consumer_contract", "source_archive", "source_resource", "source_metadata_sha256", "source_files", "source_sha256", "native_frames", "alpha_contract", "layout_policy"}
        elif entry.get("role") == "facility_scene_hd":
            expected = required | {"master_file", "consumer_contract", "source_archive", "source_resource", "source_metadata_sha256", "source_files", "source_sha256"}
        elif entry.get("role") in {"story_portrait_frame", "story_portrait_hd_frame"}:
            expected = required | candidate
        elif entry.get("role") == "map_sprite_set":
            expected = sprite_set | ({"master_file"} if "master_file" in entry else set())
        elif entry.get("role") == "map_sprite_hd_set":
            expected = sprite_set | {"layout_policy", "master_file", "frame_overrides", "source_files", "source_sha256"}
        elif entry.get("role") == "map_tileset_set":
            expected = map_tileset
        elif entry.get("role") == "battle_hud_panel":
            expected = hud_panel
        elif entry.get("role") == "battle_action_icon_set":
            expected = action_icons
        elif entry.get("role") == "native_terrain_hd_set":
            expected = sprite_set - {"source_group"} | {"map_id", "frame_ids", "semantics"}
        elif entry.get("role") in {"battle_animation_hd_set", "battle_effect_hd_set"}:
            expected = sprite_set - {"source_group"} | {"source_archive", "source_resource", "source_metadata_sha256",
                "native_header", "native_frames", "layout_policy", "source_files", "source_sha256"}
        elif entry.get("role") == "map_overlay_hd_set":
            expected = sprite_set - {"source_group"} | {"source_archive", "source_resource", "source_metadata_sha256",
                "native_lmi_frames", "layout_policy", "source_files", "source_sha256"}
        else:
            expected = required
        if set(entry) != expected:
            raise ValueError("asset fields mismatch")
        asset_id = entry["asset_id"]
        if not isinstance(asset_id, str) or not ID_RE.fullmatch(asset_id) or asset_id in ids:
            raise ValueError(f"invalid or duplicate asset_id: {asset_id!r}")
        ids.add(asset_id)
        # Character designs have distinct asset IDs. Keep approved character
        # sheets alongside the original Sol concept without replacing it.
        if entry["role"] not in ROLES or (entry["role"] in PROTOTYPE_ROLES - {"portrait_concept"} and entry["role"] in roles):
            raise ValueError(f"invalid or duplicate role: {entry['role']!r}")
        roles.add(entry["role"])
        if entry["status"] not in STATUSES:
            raise ValueError(f"invalid status: {entry['status']!r}")
        if entry["status"] == "prototype" and entry["role"] != "map_sprite_set":
            raise ValueError(f"prototype status is restricted to sprite sets: {asset_id}")
        if "master_file" in entry:
            master = Path(entry["master_file"])
            if master.is_absolute() or len(master.parts) != 1 or master.suffix.lower() != ".png":
                raise ValueError(f"unsafe private master file: {entry['master_file']!r}")
            if verify_private and not (ROOT / private_root / master).is_file():
                raise ValueError(f"private master asset missing: {ROOT / private_root / master}")
        if entry["role"] == "story_portrait_frame":
            if entry["consumer_contract"] != "native_story_dialogue_rgba_overlay_v1":
                raise ValueError(f"invalid consumer contract: {asset_id}")
            if not isinstance(entry["speaker_id"], int) or entry["speaker_id"] < 0:
                raise ValueError(f"invalid speaker identity: {asset_id}")
            if entry["frame"] not in range(4) or entry["mouth_state"] not in {"closed", "open"}:
                raise ValueError(f"invalid portrait frame identity: {asset_id}")
            portrait_speakers.add(entry['speaker_id'])
        if entry['role'] == 'battle_surface_hd':
            source = (entry['source_archive'], entry['source_resource'])
            frames = entry['native_frames']
            if (source[0] not in {'BG.DAT', 'TAI.DAT'} or type(source[1]) is not int or source[1] not in range(56)
                    or source in surface_resources or entry['consumer_contract'] != 'native_battle_surface_rgba_hd6_v1'
                    or entry['status'] not in {'runtime_candidate', 'runtime_ready'} or entry['alpha_contract'] != 'coverage'
                    or entry['layout_policy'] != 'native_surface_hd6' or not SHA_RE.fullmatch(entry['source_metadata_sha256'])
                    or not isinstance(frames, list) or len(frames) != 1 or not isinstance(frames[0], dict)):
                raise ValueError(f'invalid battle surface contract: {asset_id}')
            frame = frames[0]
            if (set(frame) != {'x', 'y', 'width', 'height', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7'}
                    or any(type(v) is not int for v in frame.values())
                    or not 1 <= frame['width'] <= 320 or not 1 <= frame['height'] <= 200
                    or any(frame[k] != 0 for k in ['x', 'y', 'delay_native', 'raw_byte_4', 'raw_byte_5', 'raw_byte_7'])
                    or (entry['width'], entry['height']) != (frame['width']*6, frame['height']*6)
                    or entry['source_files'] != [entry['master_file']] or len(entry['source_sha256']) != 1
                    or not SHA_RE.fullmatch(entry['source_sha256'][0])):
                raise ValueError(f'invalid battle surface geometry or master: {asset_id}')
            surface_resources.add(source)
            if verify_private and hashlib.sha256((ROOT/private_root/entry['master_file']).read_bytes()).hexdigest() != entry['source_sha256'][0]:
                raise ValueError(f'battle surface master sha256 mismatch: {asset_id}')
        if entry['role'] == 'facility_scene_hd':
            resource = entry['source_resource']
            if (resource not in {11, 61, 62, 12, 29, 63} or resource in facility_resources
                    or entry['source_archive'] != 'FDOTHER.DAT'
                    or entry['consumer_contract'] != 'native_facility_scene_hd6_v1'
                    or (entry['width'], entry['height']) != (1920, 1200)
                    or entry['status'] not in {'runtime_candidate', 'runtime_ready'}
                    or not SHA_RE.fullmatch(entry['source_metadata_sha256'])
                    or entry['source_files'] != [entry['master_file']]
                    or len(entry['source_sha256']) != 1 or not SHA_RE.fullmatch(entry['source_sha256'][0])):
                raise ValueError(f'invalid facility scene contract: {asset_id}')
            facility_resources.add(resource)
            if verify_private:
                source = ROOT / private_root / entry['master_file']
                if hashlib.sha256(source.read_bytes()).hexdigest() != entry['source_sha256'][0]:
                    raise ValueError(f'facility master sha256 mismatch: {asset_id}')
        if entry['role'] == 'story_portrait_hd_frame':
            if (entry['consumer_contract'] != 'native_story_dialogue_rgba_overlay_hd6_v1'
                    or (entry['width'], entry['height'], entry['frame'], entry['mouth_state'])
                    != (480, 480, 0, 'closed')):
                raise ValueError(f'invalid HD portrait contract: {asset_id}')
            speaker = entry['speaker_id']
            if not isinstance(speaker, int) or speaker < 0 or speaker in hd_speakers:
                raise ValueError(f'invalid or duplicate HD speaker: {asset_id}')
            hd_speakers.add(speaker)
        if entry["role"] == "battle_hud_panel":
            if entry["consumer_contract"] != "battle_status_panel_149x42_v1" or (
                    entry["width"], entry["height"]) != (149, 42):
                raise ValueError(f"invalid battle HUD panel contract: {asset_id}")
        if entry["role"] == "battle_action_icon_set":
            if entry["consumer_contract"] != "battle_action_icons_4x28x26_v1" or (
                    entry["width"], entry["height"], entry["frame_count"]) != (28, 26, 4):
                raise ValueError(f"invalid battle action icon contract: {asset_id}")
            if entry["semantics"] != ["attack", "spell", "item", "wait"]:
                raise ValueError(f"invalid battle action icon semantics: {asset_id}")
            if len(entry["files"]) != 4 or len(set(entry["files"])) != 4 or (
                    len(entry["frame_sha256"]) != 4):
                raise ValueError(f"invalid battle action icon files: {asset_id}")
        if entry["role"] == "native_terrain_hd_set":
            count = entry["frame_count"]
            frames, semantics = entry["frame_ids"], entry["semantics"]
            if (entry["status"] not in {"runtime_candidate", "runtime_ready"}
                    or entry["consumer_contract"] != "fdshap_native_hd_sparse_v1"
                    or type(entry["map_id"]) is not int or entry["map_id"] not in range(33)
                    or (entry["width"], entry["height"]) != (144, 144)
                    or entry["alpha_contract"] != "native_mask_nearest_6"
                    or entry["cycle_policy"] != "native_selector"
                    or type(count) is not int or count not in range(1, 1025)):
                raise ValueError(f"invalid native HD terrain contract: {asset_id}")
            if (not isinstance(frames, list) or len(frames) != count
                    or any(type(frame) is not int or frame not in range(1024) for frame in frames)
                    or len(set(frames)) != count
                    or not isinstance(semantics, list) or len(semantics) != count
                    or any(role not in {"terrain", "foreground", "both"} for role in semantics)):
                raise ValueError(f"invalid native HD terrain identity: {asset_id}")
            for frame in frames:
                identity = (entry["map_id"], frame)
                if identity in terrain_frames:
                    raise ValueError(f"duplicate native HD terrain identity: {asset_id}")
                terrain_frames.add(identity)
            files, hashes = entry["files"], entry["frame_sha256"]
            if (not isinstance(files, list) or len(files) != count or len(set(files)) != count
                    or not isinstance(hashes, list) or len(hashes) != count
                    or any(not isinstance(value, str) or not SHA_RE.fullmatch(value) for value in hashes)):
                raise ValueError(f"invalid native HD terrain files: {asset_id}")
            file_paths = [Path(value) for value in files]
        elif entry["role"] == "map_overlay_hd_set":
            resource, count = entry['source_resource'], entry['frame_count']
            indices = (list(range(68,80)) if resource == 5 and count == 12 else
                       list(range(94,104))+[116,117,118] if resource == 5 and count == 13 else
                       list(range(87,114)) if resource == 6 and count == 27 else [])
            if (entry['status'] not in {'runtime_candidate', 'runtime_ready'} or
                    entry['consumer_contract'] != 'native_lmi1_rgba_hd6_v1' or entry['source_archive'] != 'FDOTHER.DAT' or
                    type(resource) is not int or not indices or
                    entry['alpha_contract'] != 'coverage' or entry['cycle_policy'] != 'native_lmi1_entries' or
                    entry['layout_policy'] != 'native_lmi1_rect_hd6' or (entry['width'],entry['height']) != (1920,1200) or
                    type(count) is not int or
                    not isinstance(entry['source_metadata_sha256'],str) or not SHA_RE.fullmatch(entry['source_metadata_sha256'])):
                raise ValueError(f'invalid HD LMI bank: {asset_id}')
            frames = entry['native_lmi_frames']
            if not isinstance(frames,list) or len(frames)!=len(indices):
                raise ValueError(f'invalid HD LMI descriptors: {asset_id}')
            for index, frame in zip(indices,frames):
                key=('LMI1',resource,index)
                if key in battle_resources:
                    raise ValueError(f'duplicate HD LMI entry: {asset_id}')
                battle_resources.add(key)
                if (not isinstance(frame,dict) or set(frame)!={'index','width','height','pixels_sha256'} or
                        type(frame['index']) is not int or frame['index']!=index or
                        any(type(frame[k]) is not int or frame[k] not in range(1,129) for k in ['width','height']) or
                        not isinstance(frame['pixels_sha256'],str) or not SHA_RE.fullmatch(frame['pixels_sha256'])):
                    raise ValueError(f'invalid HD LMI descriptor: {asset_id}')
            files, hashes = entry['files'], entry['frame_sha256']
            sources, source_hashes = entry['source_files'], entry['source_sha256']
            for names, digests, count in [(files,hashes,len(indices)),(sources,source_hashes,None)]:
                if (not isinstance(names,list) or not names or any(not isinstance(v,str) for v in names) or
                        len(set(names))!=len(names) or (count is not None and len(names)!=count) or
                        not isinstance(digests,list) or len(digests)!=len(names) or
                        any(not isinstance(v,str) or not SHA_RE.fullmatch(v) for v in digests)):
                    raise ValueError(f'invalid HD LMI files: {asset_id}')
                for name, digest in zip(names,digests):
                    if Path(name).name!=name or Path(name).suffix!='.png':
                        raise ValueError(f'unsafe HD LMI path: {asset_id}')
                    if verify_private and hashlib.sha256((ROOT/private_root/name).read_bytes()).hexdigest()!=digest:
                        raise ValueError(f'HD LMI sha256 mismatch: {asset_id}')
            file_paths = [Path(v) for v in files]
        elif entry["role"] in {"battle_animation_hd_set", "battle_effect_hd_set"}:
            resource, count = entry['source_resource'], entry['frame_count']
            effect = entry['role'] == 'battle_effect_hd_set'
            source = 'FDOTHER.DAT' if effect else 'FIGANI.DAT'
            consumer = 'native_fdother_compound_rgba_hd6_v1' if effect else 'native_figani_rgba_hd6_v1'
            cycle = 'native_compound_frames' if effect else 'native_descriptor_key_poses'
            if effect and type(resource) is int and resource in ORDINARY_EFFECT_RESOURCES:
                consumer, cycle = 'native_fdother_ordinary_rgba_hd6_v1', 'native_ordinary_frames'
            if effect and resource == 44:
                consumer, cycle = 'native_fdother_command9_ai_rgba_hd6_v1', 'native_command9_ai_frames'
            layout = 'native_descriptor_hd6' if effect else 'contain_native_descriptor_hd6_candidate'
            if (entry['status'] not in {'runtime_candidate', 'runtime_ready'} or
                    entry['consumer_contract'] != consumer or
                    entry['source_archive'] != source or type(resource) is not int or resource not in (ORDINARY_EFFECT_RESOURCES | {44} | set(range(65,69)) if effect else range(409)) or
                    (source,resource) in battle_resources or entry['alpha_contract'] != 'coverage' or
                    entry['cycle_policy'] != cycle or entry['layout_policy'] != layout or
                    (entry['width'], entry['height']) != (1920, 1200) or
                    type(count) is not int or count not in range(1, 256) or
                    not isinstance(entry['source_metadata_sha256'], str) or not SHA_RE.fullmatch(entry['source_metadata_sha256'])):
                raise ValueError(f'invalid HD battle resource: {asset_id}')
            battle_resources.add((source,resource))
            header, frames = entry['native_header'], entry['native_frames']
            if effect and resource == 44 and (count != 31 or header != [0,31,0]):
                raise ValueError(f'invalid command9 AI shape: {asset_id}')
            if (not isinstance(header, list) or len(header) != 3 or any(type(x) is not int or x not in range(256) for x in header)
                    or (header[0] != 0 and header[1] > count) or not isinstance(frames, list) or len(frames) != count):
                raise ValueError(f'invalid HD battle descriptors: {asset_id}')
            for frame in frames:
                if (not isinstance(frame, dict) or set(frame) != {'x','y','width','height','delay_native','raw_byte_4','raw_byte_5','raw_byte_7'}
                        or any(type(v) is not int for v in frame.values()) or
                        frame['x'] not in range(-32768,32768) or frame['y'] not in range(-32768,32768) or
                        frame['width'] not in range(1,1025) or frame['height'] not in range(1,1025) or
                        any(frame[k] not in range(256) for k in ['delay_native','raw_byte_4','raw_byte_5','raw_byte_7'])):
                    raise ValueError(f'invalid HD battle frame: {asset_id}')
            files, hashes = entry['files'], entry['frame_sha256']
            if (not isinstance(files,list) or len(files)!=count or len(set(files))!=count or
                    not isinstance(hashes,list) or len(hashes)!=count or
                    any(not isinstance(h,str) or not SHA_RE.fullmatch(h) for h in hashes)):
                raise ValueError(f'invalid HD battle files: {asset_id}')
            sources, source_hashes = entry['source_files'], entry['source_sha256']
            if (not isinstance(sources,list) or not sources or len(set(sources))!=len(sources) or
                    not isinstance(source_hashes,list) or len(source_hashes)!=len(sources)):
                raise ValueError(f'invalid HD battle masters: {asset_id}')
            for name, expected_hash in zip(sources,source_hashes):
                if (not isinstance(name,str) or Path(name).name!=name or Path(name).suffix!='.png' or
                        not isinstance(expected_hash,str) or not SHA_RE.fullmatch(expected_hash)):
                    raise ValueError(f'unsafe HD battle master: {asset_id}')
                if verify_private and hashlib.sha256((ROOT/private_root/name).read_bytes()).hexdigest()!=expected_hash:
                    raise ValueError(f'HD battle master sha256 mismatch: {asset_id}')
            file_paths = [Path(value) for value in files]
        elif entry["role"] in SPRITE_ROLES | {"battle_action_icon_set"}:
            is_sprite = entry["role"] in SPRITE_ROLES
            is_hd = entry["role"] == "map_sprite_hd_set"
            if is_sprite and (type(entry["source_group"]) is not int or entry["source_group"] not in VISIBLE_FDICON_GROUPS):
                raise ValueError(f"invalid map sprite identity: {asset_id}")
            if is_sprite:
                collection = hd_sprites if is_hd else parent_sprites
                if entry['source_group'] in collection:
                    raise ValueError(f'duplicate map sprite group: {asset_id}')
                collection[entry['source_group']] = entry
                size = 144 if is_hd else 24
                if (entry['width'], entry['height']) != (size, size):
                    raise ValueError(f'invalid map sprite geometry: {asset_id}')
            walk_contract = is_sprite and (
                entry["consumer_contract"] == ("fdicon_hd_map_sprite_12x144_v1" if is_hd else "fdicon_map_sprite_12x24_v1") and
                entry["frame_count"] == 12 and
                entry["cycle_policy"] in {"three_distinct_cycles", "source_exact_repeats"}
            )
            static_contract = is_sprite and (
                entry["consumer_contract"] == ("fdicon_hd_static_sprite_3x144_v1" if is_hd else "fdicon_static_sprite_3x24_v1") and
                entry["frame_count"] == 3 and entry["cycle_policy"] == "static_three_phase"
            )
            if is_sprite and not (walk_contract or static_contract):
                raise ValueError(f"invalid map sprite contract: {asset_id}")
            if is_sprite and entry["alpha_contract"] != ("coverage" if is_hd else "binary"):
                raise ValueError(f"invalid alpha contract: {asset_id}")
            if is_hd:
                if entry['layout_policy'] != 'contain_22px_cell_feet_v1':
                    raise ValueError(f'invalid map sprite layout: {asset_id}')
                sources, source_hashes = entry['source_files'], entry['source_sha256']
                if (not isinstance(sources, list) or not sources or
                        not all(isinstance(name, str) for name in sources) or len(set(sources)) != len(sources) or
                        not isinstance(source_hashes, list) or len(sources) != len(source_hashes)):
                    raise ValueError(f'invalid HD source collection: {asset_id}')
                for name, expected_hash in zip(sources, source_hashes):
                    if (not isinstance(name, str) or Path(name).name != name or Path(name).suffix.lower() != '.png' or
                            not isinstance(expected_hash, str) or not SHA_RE.fullmatch(expected_hash)):
                        raise ValueError(f'invalid HD source provenance: {asset_id}')
                    if verify_private:
                        path = ROOT/private_root/name
                        if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != expected_hash:
                            raise ValueError(f'HD source mismatch: {asset_id}/{name}')
                overrides = entry['frame_overrides']
                if not isinstance(overrides, dict):
                    raise ValueError(f'invalid HD frame sources: {asset_id}')
                for index, name in overrides.items():
                    if index not in {str(i) for i in range(entry['frame_count'])} or name not in sources:
                        raise ValueError(f'invalid HD frame source: {asset_id}')
                if set(sources) != {entry['master_file'], *overrides.values()}:
                    raise ValueError(f'unused or missing HD source: {asset_id}')
            files = entry["files"]
            hashes = entry["frame_sha256"]
            want_count = entry["frame_count"] if is_sprite else 4
            if not isinstance(files, list) or len(files) != want_count or len(set(files)) != want_count:
                raise ValueError(f"invalid map sprite file set: {asset_id}")
            if not isinstance(hashes, list) or len(hashes) != want_count or not all(
                isinstance(value, str) and SHA_RE.fullmatch(value) for value in hashes
            ):
                raise ValueError(f"invalid map sprite hashes: {asset_id}")
            if is_sprite and entry["cycle_policy"] == "three_distinct_cycles" and len(set(hashes)) != 12:
                raise ValueError(f"map sprite cycles are not distinct: {asset_id}")
            if is_sprite and entry["cycle_policy"] == "source_exact_repeats" and len(set(hashes)) == 12:
                raise ValueError(f"map sprite declares source repeats but has none: {asset_id}")
            if is_sprite and entry["cycle_policy"] == "static_three_phase" and len(set(hashes)) != 3:
                raise ValueError(f"static map sprite phases are not distinct: {asset_id}")
            file_paths = [Path(value) for value in files]
        elif entry["role"] == "map_tileset_set":
            if entry["consumer_contract"] != "map_tileset_indexed_geometry_v1":
                raise ValueError(f"invalid map tileset contract: {asset_id}")
            if entry["map_id"] not in range(33) or (
                    entry["tile_width"], entry["tile_height"], entry["columns"], entry["width"]
                    ) != (24, 24, 16, 384):
                raise ValueError(f"invalid map tileset geometry: {asset_id}")
            if entry["height"] <= 0 or entry["height"] % entry["tile_height"] or (
                    entry["tile_count"] != entry["height"] // entry["tile_height"] * entry["columns"]):
                raise ValueError(f"invalid map tileset geometry: {asset_id}")
            file_paths = [Path(entry["file"])]
        else:
            file_paths = [Path(entry["file"])]
        for file_path in file_paths:
            if file_path.is_absolute() or len(file_path.parts) != 1 or file_path.suffix.lower() != ".png":
                raise ValueError(f"unsafe private file: {str(file_path)!r}")
        if not isinstance(entry["width"], int) or entry["width"] <= 0:
            raise ValueError(f"invalid width: {asset_id}")
        if not isinstance(entry["height"], int) or entry["height"] <= 0:
            raise ValueError(f"invalid height: {asset_id}")
        if entry["role"] not in MULTI_FILE_ROLES and (
            not isinstance(entry["sha256"], str) or not SHA_RE.fullmatch(entry["sha256"])
        ):
            raise ValueError(f"invalid sha256: {asset_id}")
        refs = entry["source_refs"]
        if not isinstance(refs, list) or not refs:
            raise ValueError(f"missing source_refs: {asset_id}")
        for ref in refs:
            if not isinstance(ref, str):
                raise ValueError(f"invalid source_ref for {asset_id}: {ref!r}")
            if SOURCE_ASSET_RE.fullmatch(ref):
                continue
            ref_path = Path(ref)
            if ref_path.is_absolute() or ".." in ref_path.parts or not (ROOT / ref_path).is_file():
                raise ValueError(f"invalid source_ref for {asset_id}: {ref!r}")
        if verify_private:
            expected_hashes = entry["frame_sha256"] if entry["role"] in MULTI_FILE_ROLES else [entry["sha256"]]
            for frame_index, (file_path, expected_hash) in enumerate(zip(file_paths, expected_hashes)):
                private_path = ROOT / private_root / file_path
                if not private_path.is_file():
                    raise ValueError(f"private asset missing: {private_path}")
                expected_size = ((entry['native_frames'][frame_index]['width']*6, entry['native_frames'][frame_index]['height']*6)
                                 if entry['role'] in {'battle_animation_hd_set','battle_effect_hd_set'} else (entry["width"],entry["height"]))
                if entry['role']=='map_overlay_hd_set':
                    frame=entry['native_lmi_frames'][frame_index]
                    expected_size=(frame['width']*6,frame['height']*6)
                if png_size(private_path) != expected_size:
                    raise ValueError(f"dimension mismatch: {asset_id}/{file_path.name}")
                digest = hashlib.sha256(private_path.read_bytes()).hexdigest()
                if digest != expected_hash:
                    raise ValueError(f"sha256 mismatch: {asset_id}/{file_path.name}")
                if entry["role"] in {"map_sprite_set", "native_terrain_hd_set"}:
                    from PIL import Image
                    alpha = Image.open(private_path).convert("RGBA").getchannel("A")
                    if not set(alpha.getdata()).issubset({0, 255}):
                        raise ValueError(f"non-binary alpha: {asset_id}/{file_path.name}")
                elif entry['role'] == 'map_overlay_hd_set':
                    from PIL import Image
                    low, high = Image.open(private_path).convert('RGBA').getchannel('A').getextrema()
                    if low != 0 or high == 0:
                        raise ValueError(f'invalid HD LMI alpha: {asset_id}/{file_path.name}')
                elif entry['role'] == 'map_sprite_hd_set':
                    from PIL import Image
                    with Image.open(private_path) as image:
                        alpha = image.convert('RGBA').getchannel('A')
                        if alpha.getextrema() != (0, 255):
                            raise ValueError(f'invalid HD coverage alpha: {asset_id}/{file_path.name}')
                elif entry["role"] in {"battle_action_icon_set", "story_portrait_hd_frame", "facility_scene_hd"}:
                    from PIL import Image
                    alpha = Image.open(private_path).convert("RGBA").getchannel("A")
                    if set(alpha.getdata()) != {255}:
                        raise ValueError(f"non-opaque asset: {asset_id}/{file_path.name}")
    if facility_resources and facility_resources != {11, 61, 62, 12, 29, 63}:
        raise ValueError('facility scene group is incomplete')
    if (('LMI1',6,87) in battle_resources) != (('LMI1',5,94) in battle_resources):
        raise ValueError('player9 effect/result group is incomplete')
    if hd_speakers and hd_speakers != portrait_speakers:
        raise ValueError('HD portrait speaker group is incomplete or orphaned')
    if hd_sprites:
        if set(hd_sprites) != VISIBLE_FDICON_GROUPS or set(parent_sprites) != VISIBLE_FDICON_GROUPS:
            raise ValueError('HD map sprite collection or 24px parents do not cover the complete native bank')
        for group, child in hd_sprites.items():
            parent = parent_sprites[group]
            if child['frame_count'] != parent['frame_count'] or child['cycle_policy'] != parent['cycle_policy']:
                raise ValueError(f'HD map sprite parent contract differs: {group}')
            for i in range(child['frame_count']):
                if str(i) in child['frame_overrides'] and parent['frame_sha256'][i] in parent['frame_sha256'][:i]:
                    raise ValueError(f'HD override conflicts with repeated phase: {group}')
                for j in range(i):
                    if ((child['frame_sha256'][i] == child['frame_sha256'][j]) !=
                            (parent['frame_sha256'][i] == parent['frame_sha256'][j])):
                        raise ValueError(f'HD map sprite parent phase equivalence differs: {group}')
    if not PROTOTYPE_ROLES.issubset(roles):
        raise ValueError(f"required prototype roles missing: {sorted(PROTOTYPE_ROLES - roles)}")
    return data


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--verify-private", action="store_true")
    parser.add_argument('--catalog', type=Path, help='明示候選清冊；未指定時驗正式清冊')
    args = parser.parse_args()
    catalog = validate(args.verify_private, args.catalog)
    print(f"validated {len(catalog['assets'])} modern-theme asset(s)")
