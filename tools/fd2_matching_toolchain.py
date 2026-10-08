#!/usr/bin/env python3
"""在 Docker 內取得 matching 試驗使用的鎖版 Open Watcom 工具。

來源為官方 2026-10-01-Build 安裝包。範圍下載只驗證 ZIP CRC 與個別
工具雜湊，不宣稱已驗證整個 129 MB 安裝包的 SHA-256。
證據入口：docs/data/ida/fd2_matching_pilot_20261008.json。
"""

import argparse
import hashlib
import io
import json
from pathlib import Path
import urllib.request
import zipfile


URL = "https://github.com/open-watcom/open-watcom-v2/releases/download/2026-10-01-Build/open-watcom-2_0-c-linux-x64"
ARCHIVE_SIZE = 129081693
ARCHIVE_SHA256 = "a961f3e02ce27bcd88428345dc57483457a623b6e718b6cee52c48d24ab8065f"
TOOLS = ("wcc386", "wdis", "wlink", "wlib")
TOOL_SHA256 = {
    "wcc386": "86eeb5103035c1bfcaed3a1c93f87d82d74e7a31d9dc43fd8ade8251025ceb11",
    "wdis": "c6d20aa8aeb550e780c4a72a367f4446225ad4ab4b7d4e8f18203c4271805015",
    "wlink": "97ff1fd068c48de2745d25b0f85057e485b3eee4df0fcf6570f1839e5d315fb5",
    "wlib": "67bfcd5fd28fb035d0eebf66a0767af45ebc276bf8f8619e1ec23c0580ce1899",
}


class RemoteZip(io.RawIOBase):
    def __init__(self):
        self.position = 0

    def seekable(self):
        return True

    def readable(self):
        return True

    def tell(self):
        return self.position

    def seek(self, offset, whence=0):
        self.position = offset if whence == 0 else self.position + offset if whence == 1 else ARCHIVE_SIZE + offset
        if not 0 <= self.position <= ARCHIVE_SIZE:
            raise ValueError("ZIP seek 超出官方資產大小")
        return self.position

    def read(self, size=-1):
        size = ARCHIVE_SIZE - self.position if size < 0 else min(size, ARCHIVE_SIZE - self.position)
        if size == 0:
            return b""
        end = self.position + size - 1
        request = urllib.request.Request(URL, headers={
            "Range": f"bytes={self.position}-{end}",
            "User-Agent": "fd2-matching-pilot", "Accept-Encoding": "identity",
        })
        with urllib.request.urlopen(request, timeout=45) as response:
            expected = f"bytes {self.position}-{end}/{ARCHIVE_SIZE}"
            if response.status != 206 or response.headers.get("Content-Range") != expected:
                raise RuntimeError("官方來源未回傳所要求的精確位元組範圍")
            data = response.read()
        if len(data) != size:
            raise RuntimeError("範圍下載長度不符")
        self.position += size
        return data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    manifest = {
        "release": "2026-10-01-Build", "source_url": URL,
        "archive_size": ARCHIVE_SIZE,
        "official_archive_sha256": ARCHIVE_SHA256,
        "archive_sha256_verified": False,
        "method": "HTTPS range extraction; ZIP CRC checked; per-tool SHA-256 recorded",
        "tools": [],
    }
    with zipfile.ZipFile(RemoteZip()) as archive:
        for tool in TOOLS:
            name = f"binl64/{tool}"
            data = archive.read(name)
            if hashlib.sha256(data).hexdigest() != TOOL_SHA256[tool]:
                raise RuntimeError(f"{name} 與已鎖定工具 SHA-256 不符")
            target = args.output / name
            target.parent.mkdir(exist_ok=True)
            target.write_bytes(data)
            target.chmod(0o755)
            record = {"file": name, "size": len(data), "sha256": hashlib.sha256(data).hexdigest()}
            manifest["tools"].append(record)
            print(json.dumps(record), flush=True)
    (args.output / "source-manifest.json").write_text(
        json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
    )


if __name__ == "__main__":
    main()
