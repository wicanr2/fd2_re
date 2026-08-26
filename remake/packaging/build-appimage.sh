#!/bin/bash
# build-appimage.sh — 產出 FD2-x86_64.AppImage(Linux 桌面散布版)。
#
# 全程使用 fd2-build-appimage Docker image；linuxdeploy/appimagetool 在 image build
# 階段下載並驗證固定 SHA-256，正式封包容器關閉網路且不需要 host FUSE。
#
# 打包內容(見 docs/knowledge-base/41-packaging.md「版權資產分離」):
#   AppDir/assets/ 只放已入庫的原創內容 —— scenarios/、story/、spells.json(remake/.gitignore 例外清單)。
#   maps/sprites/music/portraits/tileset 等 ROM 衍生素材是版權物,不打包進散布物;
#   玩家自備原版跑 tools/export_engine_assets.py 等,把產出解到 ~/.local/share/fd2_re/assets/
#   (assetPath 三層查找的 XDG 覆蓋層,見 cmd/fd2/assets.go)。
set -euo pipefail
REMAKE_ROOT="$(cd "$(dirname "$0")/.." && pwd -P)"

# 主機只負責啟動 Docker；檔案清理、編譯、AppDir 組裝、依賴收集、封裝與雜湊
# 全部在一次性容器內完成。packaging/dist 是唯一可寫輸出。
docker run --rm --network none \
  --memory 3g --cpus 2 --pids-limit 384 \
  --user "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/tmp/go-cache -e ARCH=x86_64 \
  -v "$REMAKE_ROOT":/src -w /src \
  fd2-build-appimage:latest bash -euo pipefail -c '
    dist=/src/packaging/dist
    appdir="$dist/AppDir"
    rm -rf "$appdir" "$dist/FD2-x86_64.AppImage" \
      "$dist/FD2-x86_64.AppImage.sha256" "$dist/FD2-x86_64.AppImage.file.txt"
    mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" \
      "$appdir/usr/share/icons/hicolor/256x256/apps" \
      "$appdir/assets/scenarios" "$appdir/assets/story" /tmp/home /tmp/go-cache /tmp/appimage-work

    CGO_ENABLED=1 go build -trimpath -buildvcs=false -ldflags="-s -w" \
      -o "$appdir/usr/bin/fd2" ./cmd/fd2
    install -m 0755 packaging/AppRun "$appdir/AppRun"
    install -m 0644 packaging/fd2.desktop "$appdir/fd2.desktop"
    install -m 0644 packaging/fd2.desktop "$appdir/usr/share/applications/fd2.desktop"
    install -m 0644 packaging/fd2.png "$appdir/fd2.png"
    install -m 0644 packaging/fd2.png "$appdir/usr/share/icons/hicolor/256x256/apps/fd2.png"
    cp -R assets/scenarios/. "$appdir/assets/scenarios/"
    cp -R assets/story/. "$appdir/assets/story/"
    cp assets/spells.json "$appdir/assets/spells.json"

    cd /tmp/appimage-work
    /opt/appimage-tools/linuxdeploy.AppImage --appimage-extract-and-run \
      --appdir "$appdir" --executable "$appdir/usr/bin/fd2" \
      --desktop-file "$appdir/fd2.desktop" --icon-file "$appdir/fd2.png"
    /opt/appimage-tools/appimagetool.AppImage --appimage-extract-and-run \
      --runtime-file /opt/appimage-tools/runtime-x86_64 \
      "$appdir" "$dist/FD2-x86_64.AppImage"
    file "$dist/FD2-x86_64.AppImage" | tee "$dist/FD2-x86_64.AppImage.file.txt"
    sha256sum "$dist/FD2-x86_64.AppImage" | tee "$dist/FD2-x86_64.AppImage.sha256"
  '

echo "完成：$REMAKE_ROOT/packaging/dist/FD2-x86_64.AppImage"
