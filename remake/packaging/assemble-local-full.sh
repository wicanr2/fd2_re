#!/usr/bin/env bash
# 組裝只供本機／授權交付使用的 FD2 完整版；主機只負責啟動 Docker。
set -euo pipefail

if [ "$#" -ne 3 ]; then
  echo "用法：$0 PAYLOAD_DIR ASSET_PACK_DIR OUTPUT_NAME" >&2
  exit 2
fi

payload=$(readlink -f "$1")
pack=$(readlink -f "$2")
name=$3
root=$(cd "$(dirname "$0")/../.." && pwd -P)
dist="$root/remake/packaging/dist"

case "$name" in
  ""|*[!A-Za-z0-9._-]*)
    echo "OUTPUT_NAME 只能包含英數、點、底線與連字號" >&2
    exit 2
    ;;
esac
if [ ! -d "$payload" ] || [ ! -d "$pack" ] || [ ! -f "$pack/manifest.json" ]; then
  echo "payload、素材包或 manifest.json 不存在" >&2
  exit 2
fi

head=$(git -C "$root" rev-parse HEAD)

docker run --rm --network none \
  --memory 3g --cpus 3 --pids-limit 384 \
  --user "$(id -u):$(id -g)" \
  -e FD2_ENGINE_HEAD="$head" -e FD2_OUTPUT_NAME="$name" \
  -v "$root":/repo:ro -v "$payload":/payload:ro -v "$pack":/pack:ro \
  -v "$dist":/out -w /repo \
  --entrypoint /bin/sh fd2-assets-local:20260829-sfx -c '
    set -eu
    PYTHONPATH=/repo/tools python3 /repo/tools/validate_separated_asset_pack.py \
      /pack/manifest.json --runtime-assets /repo/remake/assets \
      --coverage-summary /repo/docs/data/fd2-source-resource-coverage-summary.json

    target="/out/local-full/$FD2_OUTPUT_NAME"
    rm -rf "$target"
    mkdir -p "$target/assets"
    cp -a /payload/. "$target/"
    cp -a /repo/remake/assets/. "$target/assets/"
    cp -a /pack/. "$target/assets/"

    cat >"$target/run-fd2-local-full.sh" <<'"'"'EOF'"'"'
#!/bin/sh
set -eu
here=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
export FD2_ASSET_PACK="$here/assets"
if [ -x "$here/fd2" ]; then exec "$here/fd2" "$@"; fi
appimage=$(find "$here" -maxdepth 1 -type f -name "*.AppImage" | head -n 1)
if [ -n "$appimage" ]; then chmod +x "$appimage"; exec "$appimage" "$@"; fi
echo "找不到 fd2 或 AppImage" >&2
exit 1
EOF
    chmod 0755 "$target/run-fd2-local-full.sh"
    cat >"$target/run-fd2-local-full.bat" <<'"'"'EOF'"'"'
@echo off
set "FD2_ASSET_PACK=%~dp0assets"
"%~dp0fd2.exe" %*
EOF

    python3 - "$target" <<'"'"'PY'"'"'
from __future__ import annotations
import hashlib, json, os, pathlib, sys

root = pathlib.Path(sys.argv[1])
manifest = root / "assets" / "manifest.json"
manifest_sha = hashlib.sha256(manifest.read_bytes()).hexdigest()
metadata = {
    "schema_version": 1,
    "kind": "fd2_local_full_bundle",
    "engine_head": os.environ["FD2_ENGINE_HEAD"],
    "asset_pack": "fd2-original-b97caf22",
    "asset_manifest_sha256": manifest_sha,
    "public_distribution": False,
}
(root / "FD2-LOCAL-FULL.json").write_text(
    json.dumps(metadata, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
)
rows = []
for path in sorted(p for p in root.rglob("*") if p.is_file() and p.name != "SHA256SUMS"):
    rows.append(f"{hashlib.sha256(path.read_bytes()).hexdigest()}  {path.relative_to(root).as_posix()}")
(root / "SHA256SUMS").write_text("\n".join(rows) + "\n", encoding="utf-8")
print(f"本機完整版已組裝：{root}（{len(rows)} 個檔案）")
PY
  '

echo "完成：$dist/local-full/$name"
