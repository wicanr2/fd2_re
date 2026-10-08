#!/usr/bin/env python3
"""在既有 FD2 Docker 映像內重建本機 Watcom 10.0a 與 OMF 轉換工具。

下載檔與原廠工具只保存在 work/，不加入 Git。來源、版本與 SHA-256 固定。
"""

import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tarfile
import urllib.request


SOURCES = {
    "objconv_2.54+ds.orig.tar.xz": (
        "https://deb.debian.org/debian/pool/main/o/objconv/objconv_2.54+ds.orig.tar.xz",
        "bd2ed51061f6755e94e54012a49a0e17d3ed93da29c0d56f1dab4b16722fab59"),
    "p7zip-full_16.02+dfsg-8_amd64.deb": (
        "https://deb.debian.org/debian/pool/main/p/p7zip/p7zip-full_16.02+dfsg-8_amd64.deb",
        "874c425cf9852356fdae688586c31dfc657d082094208b5a8cc6721e20738cc5"),
    "watcom10a.7z": (
        "https://winworldpc.com/download/46c3bfc2-ade2-809c-18c3-9a11c3a4efbf/from/c39ac2af-c381-c2bf-1b25-11c3a4e284a2",
        "37f2ac13b414e441fe2a76df372263af433b142cf1de72ec431d4a7a2622568a"),
}
EXPECTED_TOOLS = {
    "WCC386.EXE": "c3666de94f6fa6800f452dae8acf45505ecdb62f0ade2cc27cc86c2d9e8e2b6b",
    "W32RUN.EXE": "d9028582ce7d2d7b0990d677e1947c10e0b0deaefb32d78319fbd2cea603cfcf",
    "DOS4GW.EXE": "dd9f4f342533f99570475b62e53231a468de2e3d83e4fc31e80c27d3a7d6b49c",
}


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def run(command, timeout=120):
    result = subprocess.run(command, capture_output=True, text=True, timeout=timeout)
    if result.returncode:
        raise ValueError(f"工具建置失敗：{command!r}\n{result.stdout}\n{result.stderr}")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    if not Path("/.dockerenv").exists():
        raise SystemExit("本工具只在 Docker 執行")
    root = args.output
    root.mkdir(parents=True, exist_ok=True)
    if root.stat().st_uid != os.getuid():
        raise ValueError("輸出目錄擁有權不符")
    for name, (url, expected) in SOURCES.items():
        path = root / name
        if not path.exists():
            partial = root / (name + ".partial")
            with urllib.request.urlopen(url, timeout=45) as response, partial.open("wb") as output:
                while data := response.read(1024 * 1024):
                    output.write(data)
            if sha(partial) != expected:
                raise ValueError(f"下載來源 {name} 與固定版本不符")
            partial.rename(path)
        if sha(path) != expected:
            raise ValueError(f"既有來源 {name} 與固定版本不符")
    source = root / "objconv-source"
    source.mkdir(exist_ok=True)
    with tarfile.open(root / "objconv_2.54+ds.orig.tar.xz") as archive:
        archive.extractall(source, filter="data")
    run(["g++", "-O2", "-o", str(root / "objconv")] +
        [str(p) for p in sorted((source / "objconv-2.54").glob("*.cpp"))])
    run(["dpkg-deb", "-x", str(root / "p7zip-full_16.02+dfsg-8_amd64.deb"), str(root / "p7zip-root")])
    seven = root / "p7zip-root/usr/lib/p7zip/7z"
    iso_root = root / "wc10a-iso"
    iso_root.mkdir(exist_ok=True)
    iso = iso_root / "WATCOM_C10A.ISO"
    if not iso.exists():
        run([str(seven), "e", "-y", "-o" + str(iso_root), str(root / "watcom10a.7z"),
             "Watcom CPP 10.0a/WATCOM_C10A.ISO"])
    if sha(iso) != "f51ed9358e7eaf6a0d292268bbbc15832530489dd8239a98f6938c244f1736fc":
        raise ValueError("工具 ISO 雜湊不符")
    inputs = root / "wc10a-input"
    inputs.mkdir(exist_ok=True)
    run([str(seven), "e", "-y", "-o" + str(inputs), str(iso), "WATCOM/BINB/WCC386.EXE",
         "WATCOM/BIN/W32RUN.EXE", "WATCOM/BIN/DOS4GW.EXE"])
    for name, expected in EXPECTED_TOOLS.items():
        if sha(inputs / name) != expected:
            raise ValueError(f"compiler 組件 {name} 雜湊不符")
    manifest = {"schema_version": 1, "sources": [{"file": name, "url": url, "sha256": digest}
                for name, (url, digest) in SOURCES.items()], "compiler_inputs": EXPECTED_TOOLS,
                "converter_sha256": sha(root / "objconv"), "driver_sha256": sha(Path(__file__)),
                "rights": "原廠工具不公開，僅於本機 Docker 內研究使用"}
    (root / "prepared-tools.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(manifest, ensure_ascii=False), flush=True)


if __name__ == "__main__":
    main()
