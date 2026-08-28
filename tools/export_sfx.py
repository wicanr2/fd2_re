#!/usr/bin/env python3
"""炎龍騎士團2 (Flame Dragon Knight 2) SFX 導出工具。

FDOTHER.DAT 資源 #31 是巢狀 `LLLLLL` 容器(見 docs/knowledge-base/36-sfx-audio-data.md),
內含 14 個子樣本,格式為 8-bit unsigned mono raw PCM(無檔頭)。本工具解開巢狀容器,
逐個子樣本補上標準 44-byte RIFF/WAV 檔頭,輸出到 remake/assets/sfx/。

取樣率:反組譯未找到 AIL_set_sample_type/set_sample_playback_rate 立即數呼叫點
(見 docs/knowledge-base/36 待辦),沿用文件既有推定值 11025Hz ── 1995 年 AIL 遊戲常見預設值。

第 10 輪新增 `--battle` 模式:戰鬥音效走另一批 FDOTHER.DAT 子資源；其中
`#48/#49/#50/#51/#52/#53/#64/#78` 是動態候選池，`#82`與`#84..#88`及`#90`
分別包含 command 0、3..8
由 `0x2A6BD→0x26152/0x26795/0x274B0` 證實的 actor／handler 音效資源，`#95` 是 0x32999
第 1 次呈現直接使用的固定資源；`#91..#94` 是 commands32..35 的
`0x27FC9` 共用 indexed owner 音效資源（見 docs/knowledge-base/36），
用同一個巢狀 LLLLLL 解包邏輯,輸出到 remake/assets/sfx/battle_<資源號>_<子序>.wav。

用法:
    python3 tools/export_sfx.py                # UI 音效池(資源 #31)→ sfx_NN.wav
    python3 tools/export_sfx.py --battle        # 戰鬥音效匯出集合 → battle_NN_MM.wav
    python3 tools/export_sfx.py --res <idx>      # 導出任意 FDOTHER.DAT 資源號(需為巢狀容器)
    python3 tools/export_sfx.py --separated-pack OUT --source FDOTHER.DAT
                                                # 正式UI／戰鬥音效→分離 OGG pack
"""
import argparse
import hashlib
import json
import os
import re
import shutil
import struct
import subprocess
import sys
import tempfile
import wave
from pathlib import Path

sys.path.insert(0, os.path.dirname(__file__))
from unpack_dat import parse_directory, NotAContainer

FDOTHER_DAT = os.path.join(
    os.path.dirname(__file__), "..", "org_game", "炎龍騎士團", "FLAME2", "FDOTHER.DAT")
SRC = os.path.join(os.path.dirname(__file__), "..", "extracted", "FDOTHER", "FDOTHER_031.bin")
OUT_DIR = os.path.join(os.path.dirname(__file__), "..", "remake", "assets", "sfx")
SAMPLE_RATE = 11025  # 推定值,見 docs/knowledge-base/36-sfx-audio-data.md 待辦

# 正式UI／第一批戰鬥巢狀音效的固定來源與容器形狀。這些值不是由輸出目錄推導，
# 而是匯入前的 provenance gate；換版本的 FDOTHER.DAT 必須先建立新契約。
SEPARATED_SOURCE_NAME = "FDOTHER.DAT"
SEPARATED_SOURCE_SIZE = 3382481
SEPARATED_SOURCE_MD5 = "22f56e5027edc7c766ad34ca4e5aca93"
SEPARATED_SOURCE_SHA256 = "a81b13493725fb70e750c4d9e0dce4e1b57d0df312c4ad4157e6d45171b13bce"
SEPARATED_TOP_RESOURCE_COUNT = 104
SEPARATED_RESOURCES = (31, 80, 82, 83, 84, 85, 86, 87, 88, 90)
# 值為巢狀 container 的非空 sample 數；directory 另有一筆 0-byte 尾哨兵。
SEPARATED_SAMPLE_COUNTS = {
    31: 13,
    80: 16,
    82: 2,
    83: 4,
    84: 3,
    85: 2,
    86: 2,
    87: 4,
    88: 2,
    90: 3,
}
SEPARATED_OGG_QUALITY = 3
SEPARATED_TIMING_EVIDENCE = "hardware-spec_approximation"

# 戰鬥音效候選池:PCM 特徵(值集中 0x80 附近、std 窄)比對確認,見 doc36 第 10 輪。
# 精確「哪個 index 對應哪招」仍是動態值(攻擊資料決定),此處先把整個候選家族導出。
BATTLE_EXPORT_INDICES = [48, 49, 50, 51, 52, 53, 64, 78, 80, 82, 84, 85, 86, 87, 88, 90, 91, 92, 93, 94, 95]


def _parse_directory_strict(data: bytes, label: str):
    """Parse an LLLLLL directory and reject malformed/empty directory headers."""

    if len(data) < 10:
        raise ValueError(f"{label}: LLLLLL container is too short")
    try:
        entries = parse_directory(data)
    except (NotAContainer, IndexError, struct.error) as exc:
        raise ValueError(f"{label}: invalid LLLLLL directory: {exc}") from exc
    if not entries:
        raise ValueError(f"{label}: LLLLLL directory has no entries")
    first = struct.unpack_from("<I", data, 6)[0]
    if first < 10 or first != 6 + 4 * len(entries) or first > len(data):
        raise ValueError(f"{label}: directory bounds are invalid")
    for index, (offset, length) in enumerate(entries):
        if offset < first or length < 0 or offset + length > len(data):
            raise ValueError(f"{label}: entry {index} exceeds container bounds")
    return entries


def _source_identity(data: bytes) -> dict[str, object]:
    """Return the source identity shape used by resource.json."""

    return {
        "name": SEPARATED_SOURCE_NAME,
        "size": len(data),
        "md5": hashlib.md5(data).hexdigest(),
        "sha256": hashlib.sha256(data).hexdigest(),
    }


def _read_and_verify_separated_source(source: Path) -> tuple[bytes, dict[str, object]]:
    """Read a player-provided FDOTHER.DAT and enforce the fixed provenance gate."""

    if source.name != SEPARATED_SOURCE_NAME:
        raise ValueError(f"來源檔名必須是 {SEPARATED_SOURCE_NAME}，收到 {source.name!r}")
    if not source.is_file() or source.is_symlink():
        raise ValueError(f"來源不是一般檔案: {source}")
    data = source.read_bytes()
    identity = _source_identity(data)
    expected = {
        "name": SEPARATED_SOURCE_NAME,
        "size": SEPARATED_SOURCE_SIZE,
        "md5": SEPARATED_SOURCE_MD5,
        "sha256": SEPARATED_SOURCE_SHA256,
    }
    if identity != expected:
        raise ValueError(
            "FDOTHER.DAT provenance 不符: "
            f"size={identity['size']} md5={identity['md5']} sha256={identity['sha256']}"
        )
    return data, identity


def _read_separated_resources(data: bytes) -> dict[int, list[bytes]]:
    """Validate both archive directory levels and return the canonical raw PCM samples."""

    outer_entries = _parse_directory_strict(data, SEPARATED_SOURCE_NAME)
    if len(outer_entries) != SEPARATED_TOP_RESOURCE_COUNT:
        raise ValueError(
            f"{SEPARATED_SOURCE_NAME}: outer resource count={len(outer_entries)}, "
            f"expected {SEPARATED_TOP_RESOURCE_COUNT}"
        )

    result: dict[int, list[bytes]] = {}
    for resource in SEPARATED_RESOURCES:
        offset, length = outer_entries[resource]
        nested_data = data[offset : offset + length]
        expected_samples = SEPARATED_SAMPLE_COUNTS[resource]
        entries = _parse_directory_strict(data=nested_data, label=f"FDOTHER.DAT #{resource}")
        expected_container_count = expected_samples + 1
        if len(entries) != expected_container_count:
            raise ValueError(
                f"FDOTHER.DAT #{resource}: nested count={len(entries)}, "
                f"expected {expected_container_count}"
            )
        tail_index = expected_samples
        for index, (_, sample_length) in enumerate(entries):
            if index < expected_samples and sample_length <= 0:
                raise ValueError(f"FDOTHER.DAT #{resource}: sample {index} is empty")
            if index == tail_index and sample_length != 0:
                raise ValueError(f"FDOTHER.DAT #{resource}: tail entry {index} is not empty")
        result[resource] = [nested_data[offset : offset + length] for offset, length in entries[:expected_samples]]

    if sum(len(samples) for samples in result.values()) != 51:
        raise ValueError("正式分離音效非空 sample 總數不是 51")
    return result


def _find_oggenc() -> str:
    executable = shutil.which("oggenc")
    if executable is None:
        raise RuntimeError("找不到 oggenc；請使用含 vorbis-tools 的 fd2-assets Docker image")
    return executable


def _encoder_metadata(oggenc: str) -> dict[str, object]:
    completed = subprocess.run(
        [oggenc, "--version"], capture_output=True, text=True, check=False
    )
    if completed.returncode != 0:
        detail = (completed.stderr or completed.stdout).strip()
        raise RuntimeError(f"oggenc --version 失敗: {detail or completed.returncode}")
    first_line = (completed.stdout or completed.stderr).splitlines()[0].strip()
    match = re.search(r"oggenc from (.+)$", first_line)
    version = match.group(1) if match else first_line
    if not version:
        raise RuntimeError("oggenc --version 沒有回傳版本")
    return {"name": "oggenc", "version": version, "quality": SEPARATED_OGG_QUALITY}


def _encode_ogg(oggenc: str, pcm: bytes, output: Path) -> None:
    """Encode unsigned-u8 mono PCM with fixed settings and a fixed Ogg serial."""

    command = [
        oggenc,
        "--quiet",
        "--raw",
        "--raw-bits=8",
        "--raw-chan=1",
        f"--raw-rate={SAMPLE_RATE}",
        "--raw-endianness=0",
        f"--quality={SEPARATED_OGG_QUALITY}",
        "--serial=0",
        f"--output={output}",
        "-",
    ]
    completed = subprocess.run(command, input=pcm, capture_output=True, check=False)
    if completed.returncode != 0:
        detail = (completed.stderr or completed.stdout).decode(errors="replace").strip()
        raise RuntimeError(f"oggenc 編碼失敗 ({output.name}): {detail or completed.returncode}")
    try:
        encoded = output.read_bytes()
    except OSError as exc:
        raise RuntimeError(f"oggenc 沒有產生 {output.name}: {exc}") from exc
    if len(encoded) < 4 or encoded[:4] != b"OggS":
        raise RuntimeError(f"oggenc 產生的 {output.name} 不是有效 OGG")


def _probe_ogg(output: Path, expected_samples: int) -> None:
    """Decode one OGG in the tool image and enforce its audio shape."""

    ogginfo = shutil.which("ogginfo")
    oggdec = shutil.which("oggdec")
    if ogginfo is None or oggdec is None:
        raise RuntimeError("找不到 ogginfo/oggdec；無法完成 OGG 解碼驗證")
    info = subprocess.run([ogginfo, str(output)], capture_output=True, text=True, check=False)
    if info.returncode != 0:
        detail = (info.stderr or info.stdout).strip()
        raise RuntimeError(f"OGG probe 失敗 ({output.name}): {detail or info.returncode}")
    if not re.search(r"(?m)^\s*Channels:\s*1\s*$", info.stdout):
        raise RuntimeError(f"OGG {output.name} 不是 mono")
    if not re.search(r"(?m)^\s*Rate:\s*11025\s*$", info.stdout):
        raise RuntimeError(f"OGG {output.name} 不是 11025 Hz")

    decoded = output.with_name(f".{output.stem}.probe.wav")
    try:
        result = subprocess.run(
            [oggdec, "--quiet", f"--output={decoded}", str(output)],
            capture_output=True,
            check=False,
        )
        if result.returncode != 0:
            detail = (result.stderr or result.stdout).decode(errors="replace").strip()
            raise RuntimeError(f"OGG decode 失敗 ({output.name}): {detail or result.returncode}")
        with wave.open(str(decoded), "rb") as wav:
            if wav.getnchannels() != 1 or wav.getframerate() != SAMPLE_RATE:
                raise RuntimeError(f"OGG {output.name} 解碼格式不是 mono/{SAMPLE_RATE} Hz")
            frame_count = wav.getnframes()
            if abs(frame_count - expected_samples) > 1:
                raise RuntimeError(
                    f"OGG {output.name} duration 不符: decoded={frame_count}, source={expected_samples}"
                )
            decoded_pcm = wav.readframes(frame_count)
            if not decoded_pcm or not any(decoded_pcm):
                raise RuntimeError(f"OGG {output.name} 是靜音")
    except wave.Error as exc:
        raise RuntimeError(f"OGG {output.name} 解碼 WAV 無效: {exc}") from exc
    finally:
        try:
            decoded.unlink()
        except FileNotFoundError:
            pass


def _write_json(path: Path, document: dict[str, object]) -> None:
    path.write_text(
        json.dumps(document, ensure_ascii=False, indent=2, sort_keys=True) + "\n",
        encoding="utf-8",
    )


def export_separated_pack(source: Path, output: Path) -> list[Path]:
    """Atomically export the canonical UI and battle sound banks into ``output``."""

    data, identity = _read_and_verify_separated_source(source)
    banks = _read_separated_resources(data)
    oggenc = _find_oggenc()
    encoder = _encoder_metadata(oggenc)

    output = output.expanduser()
    if os.path.lexists(output):
        raise ValueError(f"輸出目錄已存在，為避免覆蓋拒絕寫入: {output}")
    parent = output.parent if output.parent != Path("") else Path(".")
    parent.mkdir(parents=True, exist_ok=True)
    staging = Path(tempfile.mkdtemp(prefix=f".{output.name}.tmp-", dir=parent))
    written: list[Path] = []
    try:
        sfx_root = staging / "sfx"
        sfx_root.mkdir()
        for resource in SEPARATED_RESOURCES:
            bank_root = sfx_root / f"FDOTHER_{resource:03d}"
            bank_root.mkdir()
            samples_metadata: list[dict[str, object]] = []
            for subresource, pcm in enumerate(banks[resource]):
                sample_name = f"sample_{subresource:03d}.ogg"
                sample_path = bank_root / sample_name
                _encode_ogg(oggenc, pcm, sample_path)
                _probe_ogg(sample_path, len(pcm))
                samples_metadata.append(
                    {
                        "subresource": subresource,
                        "source_byte_count": len(pcm),
                        "source_pcm_sha256": hashlib.sha256(pcm).hexdigest(),
                        "path": sample_name,
                        "cue_evidence": "typed_schedule",
                    }
                )
                written.append(output / "sfx" / f"FDOTHER_{resource:03d}" / sample_name)

            document = {
                "schema_version": 1,
                "kind": "fd2_pcm_sound_bank",
                "asset_id": f"sfx/FDOTHER_{resource:03d}",
                "status": "converted",
                "source": identity,
                "resource": resource,
                "container_count": SEPARATED_SAMPLE_COUNTS[resource] + 1,
                "zero_length_tail_index": SEPARATED_SAMPLE_COUNTS[resource],
                "sample_rate": SAMPLE_RATE,
                "channels": 1,
                "sample_format": "unsigned_u8",
                "timing_evidence": SEPARATED_TIMING_EVIDENCE,
                "encoder": encoder,
                "samples": samples_metadata,
            }
            _write_json(bank_root / "resource.json", document)
            written.append(output / "sfx" / f"FDOTHER_{resource:03d}" / "resource.json")
        os.replace(staging, output)
    except Exception:
        shutil.rmtree(staging, ignore_errors=True)
        raise
    return written


def export_container(data: bytes, name_prefix: str, label: str):
    try:
        entries = parse_directory(data)
    except NotAContainer as e:
        print(f"  [{label}] 不是合法的 LLLLLL 容器,略過: {e}")
        return []

    os.makedirs(OUT_DIR, exist_ok=True)
    print(f"[{label}] {len(data)} bytes, {len(entries)} 個子樣本")

    written = []
    for i, (off, ln) in enumerate(entries):
        pcm = data[off:off + ln]
        if len(pcm) == 0:
            print(f"  {name_prefix}{i:02d}: 0 bytes(目錄結尾哨兵),略過")
            continue
        out_path = os.path.join(OUT_DIR, f"{name_prefix}{i:02d}.wav")
        with wave.open(out_path, "wb") as w:
            w.setnchannels(1)
            w.setsampwidth(1)  # 8-bit unsigned
            w.setframerate(SAMPLE_RATE)
            w.writeframes(pcm)
        dur_ms = len(pcm) / SAMPLE_RATE * 1000
        print(f"  {name_prefix}{i:02d}: {len(pcm):6d} bytes -> {dur_ms:7.1f} ms  ({out_path})")
        written.append(out_path)
    return written


def main(argv=None):
    argv = sys.argv[1:] if argv is None else argv

    # Canonical first-batch importer. Keep this branch before legacy switches so
    # ``--source`` is never accidentally interpreted as an old raw-resource path.
    if "--separated-pack" in argv:
        parser = argparse.ArgumentParser(description="Export first-batch battle SFX as a separated OGG pack")
        parser.add_argument("--separated-pack", required=True, type=Path, metavar="OUT")
        parser.add_argument("--source", required=True, type=Path, metavar="FDOTHER.DAT")
        args = parser.parse_args(argv)
        try:
            written = export_separated_pack(args.source, args.separated_pack)
        except (OSError, RuntimeError, ValueError) as exc:
            print(f"錯誤: {exc}", file=sys.stderr)
            return 1
        print(f"完成: {len(written) - len(SEPARATED_RESOURCES)} 個 OGG、{len(SEPARATED_RESOURCES)} 份 metadata 已原子寫入 {args.separated_pack}")
        return 0

    if "--battle" in argv or "--res" in argv:
        dat = open(FDOTHER_DAT, "rb").read()
        try:
            top_entries = parse_directory(dat)
        except NotAContainer as e:
            print(f"錯誤: {FDOTHER_DAT} 不是合法的 LLLLLL 容器: {e}")
            return 1

        if "--res" in argv:
            idx = int(argv[argv.index("--res") + 1])
            indices = [idx]
        else:
            indices = BATTLE_EXPORT_INDICES

        written = []
        for idx in indices:
            off, ln = top_entries[idx]
            chunk = dat[off:off + ln]
            written += export_container(chunk, f"battle_{idx:02d}_", f"FDOTHER.DAT #{idx}")

        print(f"\n完成: {len(written)} 個 WAV 已寫入 {OUT_DIR}")
        return 0

    data = open(SRC, "rb").read()
    written = export_container(data, "sfx_", os.path.basename(SRC))
    print(f"\n完成: {len(written)} 個 WAV 已寫入 {OUT_DIR}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
