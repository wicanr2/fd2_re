#!/bin/bash
# build-windows.sh — CGO 跨編 Windows fd2.exe(mingw-w64,docker fd2-build-mingw image)。
# Ebiten desktop 後端在 Windows 走 win32/DirectX,CGO_ENABLED=1 是硬需求(cgo glfw binding)。
#
# 產物只有 binary + 已入庫資產(scenarios/story/locales/spells.json);其餘版權資產由玩家自跑
# tools/export_engine_assets.py 產生後,放到 exe 旁的 assets/ 資料夾(Windows 無 XDG 概念,
# 桌面版走「cwd 相對 assets/」這條既有 fallback,見 cmd/fd2/assets.go assetPath 第 3 層)。
set -euo pipefail
REMAKE_ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"

# 主機只負責啟動 Docker；清理、編譯、資產組裝、格式檢查與壓縮全部在一次性容器內。
# packaging/dist 是唯一可寫輸出，原始碼與資產在同一掛載內維持不變。
docker run --rm --network none \
  --memory 3g --cpus 2 --pids-limit 384 \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/tmp/go-cache \
  -v "$REMAKE_ROOT":/src -w /src \
  fd2-build-mingw:latest bash -euo pipefail -c '
    dist=packaging/dist/windows
    archive=packaging/dist/fd2-windows-x86_64.zip
    rm -rf "$dist" "$archive"
    mkdir -p "$dist/assets/scenarios" "$dist/assets/story" "$dist/assets/locales" /tmp/home /tmp/go-cache

    CGO_ENABLED=1 GOOS=windows GOARCH=amd64 \
      CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
      go build -trimpath -buildvcs=false -ldflags="-s -w -H=windowsgui" \
      -o "$dist/fd2.exe" ./cmd/fd2

    cp -R assets/scenarios/. "$dist/assets/scenarios/"
    cp -R assets/story/. "$dist/assets/story/"
    cp -R assets/locales/. "$dist/assets/locales/"
    cp assets/spells.json "$dist/assets/spells.json"
    file "$dist/fd2.exe" | tee packaging/dist/fd2-windows-x86_64.file.txt
    (cd "$dist" && zip -qr "../fd2-windows-x86_64.zip" .)
    sha256sum "$archive" | tee packaging/dist/fd2-windows-x86_64.sha256
  '

echo "完成：$REMAKE_ROOT/packaging/dist/fd2-windows-x86_64.zip"
