#!/usr/bin/env python3
"""正式三平台引擎包：共用清單、權利邊界、原生自檢收據與雜湊。"""
from __future__ import annotations

import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import plistlib
import re
import shutil
import stat
import struct
import subprocess
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[2]
PLATFORMS = ("linux-x86_64", "windows-x86_64", "macos-universal")


def sha(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def version():
    value = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"v\.\d+\.\d+\.\d+-\d{8}", value):
        raise ValueError("VERSION 必須是 v.主版.次版.修訂版-YYYYMMDD")
    datetime.datetime.strptime(value[-8:], "%Y%m%d")
    ref = os.environ.get("GITHUB_REF", "")
    if ref.startswith("refs/tags/") and ref != "refs/tags/" + value:
        raise ValueError("tag 與 VERSION 不同")
    return value


def commit():
    return subprocess.check_output(
        ["git", "-C", str(ROOT), "rev-parse", "HEAD"], text=True).strip()


def write_json(path, data):
    Path(path).write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n",
                          encoding="utf-8")


def module_licenses(destination, platform):
    """保留 Go、程式實際引用模組及其內嵌元件的原始授權文字。"""
    architectures = ("arm64", "amd64") if platform == "macos-universal" else (None,)
    parts = []
    for architecture in architectures:
        env = dict(os.environ)
        if architecture:
            env["GOARCH"] = architecture
            env["CGO_ENABLED"] = "1"
            native_arch = "x86_64" if architecture == "amd64" else architecture
            env["CGO_CFLAGS"] = f"-arch {native_arch}"
            env["CGO_LDFLAGS"] = f"-arch {native_arch}"
        parts.append(subprocess.check_output(
            ["go", "list", "-deps", "-json=Module", "./cmd/fd2"],
            cwd=ROOT / "remake", text=True, env=env))
    raw = "\n".join(parts)
    decoder = json.JSONDecoder()
    modules = []
    seen = set()
    while raw.strip():
        item, end = decoder.raw_decode(raw.lstrip())
        raw = raw.lstrip()[end:]
        item = item.get("Module")
        if not item or item.get("Main") or item["Path"] in seen:
            continue
        seen.add(item["Path"])
        name = item["Path"]
        module_dir = Path(item["Dir"])
        license_files = []
        for source in sorted(module_dir.rglob("*")):
            if source.is_file() and re.match(
                    r"^(licen[cs]e|copying|notice)([._-]|$)", source.name, re.I):
                relative = source.relative_to(module_dir)
                target = destination / name / relative
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copyfile(source, target)
                license_files.append(relative.as_posix())
        if not license_files:
            raise ValueError(f"模組缺少授權文字：{name}")
        modules.append({"module": name, "version": item["Version"],
                        "license_files": license_files})
    goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], text=True).strip())
    (destination / "Go").mkdir(parents=True, exist_ok=True)
    shutil.copyfile(goroot / "LICENSE", destination / "Go/LICENSE")
    write_json(destination / "modules.json", modules)


def stage(destination, platform):
    if destination.exists() and any(destination.iterdir()):
        raise ValueError("組包目錄必須是全新空目錄")
    destination.mkdir(parents=True, exist_ok=True)
    subprocess.run([sys.executable, str(ROOT / "tools/copy_tracked_remake_assets.py"),
                    "--repo", str(ROOT), "--destination", str(destination / "assets")],
                   check=True)
    shutil.copyfile(ROOT / "LICENSE", destination / "LICENSE")
    notices = destination / "third-party"
    notices.mkdir()
    module_licenses(notices, platform)
    if platform == "linux-x86_64":
        license_path = ROOT / "remake/packaging/appimage-runtime-LICENSE"
        if sha(license_path) != "aa154fc9070614bbe7921f89db11efd1dba7a1f3a41685958110e2230f9c0ca1":
            raise ValueError("AppImage runtime 授權來源雜湊不符")
        shutil.copyfile(license_path, notices / "appimage-runtime-LICENSE")
        write_json(notices / "appimage-runtime-source.json", {
            "source": "https://github.com/AppImage/type2-runtime/tree/20251108",
            "runtime_sha256": "2fca8b443c92510f1483a883f60061ad09b46b978b2631c807cd873a47ec260d",
            "license_sha256": sha(license_path)})
    readme = f"""炎龍騎士團 2 重製 {version()}
Wang Chun-Yu / wicanr2
https://github.com/wicanr2/fd2_re

這是第一版正式引擎發行。30章受控流程已通過逐章驗收。
各章使用的建構槽、清敵及保存差異見儲存庫 docs/REMAKE-STATUS.md。
HD、所有分支、自然難度與人耳音訊驗收仍未完成。

本包不含原版遊戲、美術、音樂、字型或私人現代素材。
首次遊玩前，請從自己合法持有的來源建立完整分離素材包。
現有 extract_all.py 與 fd2-asset-import 各自只涵蓋部分輸出，
不能把單一工具的輸出當作完整可玩素材包。
完整素材與來源契約：
https://github.com/wicanr2/fd2_re/blob/{version()}/docs/knowledge-base/60-editor-separated-assets-spec.md

將完整分離素材放到使用者資料目錄的 assets 子目錄，
或以 FD2_ASSET_PACK 指定完整包目錄。保留本包的正式劇本與 canonical 資料。
預設資料、存檔及設定位置：
Linux / macOS：~/.local/share/fd2_re/
Windows：%USERPROFILE%\\.local\\share\\fd2_re\\
設定 XDG_DATA_HOME 時，改用該目錄下的 fd2_re/。
不要覆蓋或刪除自己原有的存檔，升級前請先備份此目錄。

Windows：解壓後執行 fd2.exe 或 start-fd2.bat。
macOS：解壓後開啟 FD2.app，同時支援 Intel 與 Apple Silicon。
Linux：chmod +x FD2-*.AppImage 後執行。沒有 FUSE 時可使用 --appimage-extract-and-run。
Linux 基於 Debian 12，需相容的 glibc、X11/OpenGL 與 ALSA 執行環境。

封包通過原生程序資料自檢；這不等於實體桌面輸入、音訊或長時間遊玩驗收。
Windows / macOS 未簽署；macOS 尚未公證，系統可能要求使用者另行允許執行。
缺少素材時會阻擋啟動，不能直接以原版 DAT 取代分離素材。

程式及原創內容依隨包 LICENSE 的 RRSAL-1.0 提供。
第三方元件的授權文字在 third-party/。
"""
    if platform == "windows-x86_64":
        (destination / "README.txt").write_bytes(b"\xef\xbb\xbf" + readme.replace(
            "\n", "\r\n").encode("utf-8"))
        (destination / "start-fd2.bat").write_bytes(
            b'@echo off\r\ncd /d "%~dp0"\r\nstart "" "%~dp0fd2.exe" %*\r\n')
    else:
        (destination / "README.txt").write_text(readme, encoding="utf-8")
    assets = {p.relative_to(destination / "assets").as_posix():
              {"size": p.stat().st_size, "sha256": sha(p)}
              for p in sorted((destination / "assets").rglob("*")) if p.is_file()}
    write_json(destination / "FD2-RELEASE.json", {
        "schema_version": 1, "version": version(), "source_commit": commit(),
        "platform": platform, "distribution": "engine-only",
        "original_assets_included": False, "modern_private_assets_included": False,
        "gameplay_scope": "111受控流程30章；各章限制見正式台帳，HD未完成",
        "assets": assets})


def verify(destination, platform):
    doc = json.loads((destination / "FD2-RELEASE.json").read_text(encoding="utf-8"))
    if (doc["version"], doc["source_commit"], doc["platform"], doc["distribution"]) != (
            version(), commit(), platform, "engine-only"):
        raise ValueError("封包版本／提交／平台／權利範圍不符")
    expected = doc["assets"]
    found = {p.relative_to(destination / "assets").as_posix(): p
             for p in (destination / "assets").rglob("*") if p.is_file()}
    if set(found) != set(expected):
        raise ValueError("封包素材與公開清單不符")
    tracked = subprocess.check_output(
        ["git", "-C", str(ROOT), "ls-files", "-z", "--", "remake/assets"])
    allow = {x.decode()[len("remake/assets/"):] for x in tracked.split(b"\0") if x}
    if set(found) != allow:
        raise ValueError("封包混入未追蹤素材或遺漏公開資料")
    for relative, path in found.items():
        if sha(path) != expected[relative]["sha256"]:
            raise ValueError(f"素材雜湊不符：{relative}")
        if path.suffix.lower() in {".dat", ".exe", ".sav", ".ogg", ".wav", ".mid", ".b24"}:
            raise ValueError(f"拒絕原版或轉檔素材：{relative}")
    if (destination / "LICENSE").read_bytes() != (ROOT / "LICENSE").read_bytes():
        raise ValueError("LICENSE 不符")
    if not (destination / "third-party/modules.json").is_file():
        raise ValueError("缺少第三方授權清冊")
    if platform == "windows-x86_64":
        raw = (destination / "README.txt").read_bytes()
        if not raw.startswith(b"\xef\xbb\xbf") or b"\n" in raw.replace(b"\r\n", b""):
            raise ValueError("Windows README 必須為 UTF-8 BOM／CRLF")
        data = (destination / "fd2.exe").read_bytes()
        pe = struct.unpack_from("<I", data, 0x3c)[0]
        if data[pe:pe + 4] != b"PE\0\0" or struct.unpack_from("<H", data, pe + 4)[0] != 0x8664:
            raise ValueError("不是 Windows x86_64 PE")
        if struct.unpack_from("<H", data, pe + 24 + 68)[0] != 2:
            raise ValueError("Windows 正式程式必須是 GUI 子系統")
    elif platform == "linux-x86_64":
        data = (destination / "usr/bin/fd2").read_bytes()
        if data[:6] != b"\x7fELF\x02\x01" or struct.unpack_from("<H", data, 18)[0] != 62:
            raise ValueError("不是 Linux x86_64 ELF")
    else:
        app = destination.parent
        data = (app / "MacOS/fd2").read_bytes()
        if data[:4] != b"\xca\xfe\xba\xbe":
            raise ValueError("不是 macOS universal Mach-O")
        count = struct.unpack_from(">I", data, 4)[0]
        cpus = {struct.unpack_from(">I", data, 8 + index * 20)[0] for index in range(count)}
        if cpus != {0x1000007, 0x100000c}:
            raise ValueError("macOS 必須同時包含 x86_64／arm64")
        info = plistlib.loads((app / "Info.plist").read_bytes())
        if info.get("FD2ReleaseVersion") != version():
            raise ValueError("macOS bundle 完整版號不符")
    return doc


def make_zip(source, archive):
    archive.parent.mkdir(parents=True, exist_ok=True)
    if archive.exists():
        raise ValueError("不覆寫既有封包")
    with zipfile.ZipFile(archive, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as output:
        timestamp = datetime.datetime.strptime(version()[-8:], "%Y%m%d").timetuple()[:6]
        for path in sorted(source.rglob("*")):
            if path.is_file():
                item = zipfile.ZipInfo(path.relative_to(source.parent).as_posix(), timestamp)
                item.create_system = 3
                item.external_attr = (stat.S_IFREG | stat.S_IMODE(path.stat().st_mode)) << 16
                item.compress_type = zipfile.ZIP_DEFLATED
                output.writestr(item, path.read_bytes(), compresslevel=9)
    with zipfile.ZipFile(archive) as zipped:
        if zipped.testzip() is not None:
            raise ValueError("ZIP CRC 錯誤")
        for item in zipped.infolist():
            if not item.filename.isascii() and not item.flag_bits & 0x800:
                raise ValueError("非ASCII ZIP名稱缺少UTF-8旗標")


def attest(destination, platform, archive, log):
    doc = verify(destination, platform)
    expected = f"FD2 {version()} 封裝自我檢查通過"
    if expected not in log.read_text(encoding="utf-8"):
        raise ValueError("缺少本版本原生封包自檢成功紀錄")
    report = {"version": version(), "source_commit": commit(), "platform": platform,
              "status": "passed", "distribution": "engine-only",
              "check": "native-package-self-check; not physical desktop acceptance",
              "log_sha256": sha(log), "package": archive.name,
              "size": archive.stat().st_size, "sha256": sha(archive),
              "public_asset_count": len(doc["assets"])}
    write_json(archive.parent / f"{platform}-smoke.json", report)
    archive.with_suffix(archive.suffix + ".sha256").write_text(
        f"{sha(archive)}  {archive.name}\n", encoding="ascii")


def collect(downloads, output):
    output.mkdir(parents=True, exist_ok=True)
    reports = []
    for platform in PLATFORMS:
        matches = list(downloads.rglob(f"{platform}-smoke.json"))
        if len(matches) != 1:
            raise ValueError(f"{platform} 收據缺少或重複")
        report = json.loads(matches[0].read_text(encoding="utf-8"))
        if (report["version"], report["source_commit"], report["status"],
                report["distribution"], report["platform"]) != (
                    version(), commit(), "passed", "engine-only", platform):
            raise ValueError(f"{platform} 收據來源或狀態不符")
        suffix = ".AppImage" if platform.startswith("linux") else ".zip"
        expected = f"FD2-{version()}-{platform}{suffix}"
        if report["package"] != expected:
            raise ValueError("封包名稱不符")
        package = matches[0].parent / expected
        if sha(package) != report["sha256"] or package.stat().st_size != report["size"]:
            raise ValueError("封包雜湊或大小不符")
        shutil.copyfile(package, output / expected)
        reports.append(report)
    write_json(output / "SHA256SUMS.json", {
        "version": version(), "source_commit": commit(),
        "files": [{"file": r["package"], "size": r["size"], "sha256": r["sha256"]}
                  for r in reports]})
    write_json(output / "VALIDATION.json", {
        "version": version(), "source_commit": commit(), "platforms": reports,
        "limitations": ["HD未完成", "各章受控流程與保存例外見正式台帳",
                        "實體桌面輸入／音訊未驗；Windows／macOS未簽署，macOS未公證",
                        "公開引擎包需玩家自備完整合法分離素材；現有單一匯入工具不產生全包"]})


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("stage", "verify", "zip", "attest", "collect"))
    parser.add_argument("source", type=Path)
    parser.add_argument("--platform", choices=PLATFORMS)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--log", type=Path)
    args = parser.parse_args()
    version()
    if args.command == "stage":
        stage(args.source, args.platform)
    elif args.command == "verify":
        verify(args.source, args.platform)
    elif args.command == "zip":
        make_zip(args.source, args.output)
    elif args.command == "attest":
        attest(args.source, args.platform, args.output, args.log)
    else:
        collect(args.source, args.output)


if __name__ == "__main__":
    main()
