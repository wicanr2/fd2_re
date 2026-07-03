#!/bin/bash
# build-windows.sh — CGO 跨編 Windows fd2.exe(mingw-w64,docker fd2-build-mingw image)。
# Ebiten desktop 後端在 Windows 走 win32/DirectX,CGO_ENABLED=1 是硬需求(cgo glfw binding)。
#
# 產物只有 binary + 已入庫資產(scenarios/story/spells.json);其餘版權資產由玩家自跑
# tools/export_engine_assets.py 產生後,放到 exe 旁的 assets/ 資料夾(Windows 無 XDG 概念,
# 桌面版走「cwd 相對 assets/」這條既有 fallback,見 cmd/fd2/assets.go assetPath 第 3 層)。
set -euo pipefail
cd "$(dirname "$(readlink -f "$0")")/.."   # remake/

DIST=packaging/dist/windows
rm -rf "$DIST"
mkdir -p "$DIST/assets/scenarios" "$DIST/assets/story"

echo "== 編譯 Windows amd64(CGO + mingw-w64)=="
docker run --rm -v "$PWD":/src -w /src -e GOCACHE=/src/.gocache fd2-build-mingw:latest bash -c '
  CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
    go build -trimpath -ldflags="-s -w -H=windowsgui" -o packaging/dist/windows/fd2.exe ./cmd/fd2
'
docker run --rm -v "$PWD":/src -w /src fd2-build-mingw:latest chown -R "$(id -u)":"$(id -g)" "$DIST"

cp -r assets/scenarios/. "$DIST/assets/scenarios/"
cp -r assets/story/.    "$DIST/assets/story/"
cp assets/spells.json   "$DIST/assets/spells.json"

(cd packaging/dist/windows && zip -qr ../fd2-windows-x86_64.zip .)

echo "完成:packaging/dist/fd2-windows-x86_64.zip"
file packaging/dist/windows/fd2.exe
ls -la packaging/dist/fd2-windows-x86_64.zip
