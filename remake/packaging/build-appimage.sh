#!/bin/bash
# build-appimage.sh — 產出 FD2-x86_64.AppImage(Linux 桌面散布版)。
#
# 全程 docker(fd2-build image:golang:1.22-bookworm + ebiten X11/ALSA 開發headers),
# 不污染系統;linuxdeploy/appimagetool 用 --appimage-extract-and-run 執行(不需 host FUSE)。
#
# 打包內容(見 docs/knowledge-base/41-packaging.md「版權資產分離」):
#   AppDir/assets/ 只放已入庫的原創內容 —— scenarios/、story/、spells.json(remake/.gitignore 例外清單)。
#   maps/sprites/music/portraits/tileset 等 ROM 衍生素材是版權物,不打包進散布物;
#   玩家自備原版跑 tools/export_engine_assets.py 等,把產出解到 ~/.local/share/fd2_re/assets/
#   (assetPath 三層查找的 XDG 覆蓋層,見 cmd/fd2/assets.go)。
set -euo pipefail
cd "$(dirname "$(readlink -f "$0")")/.."   # remake/

DIST=packaging/dist
APPDIR="$DIST/AppDir"
# docker 內以 root 寫入的檔案(linuxdeploy/appimagetool 產物)歸 root 所有,host 使用者刪不掉;
# 用同一個 image 以 root 身分先要回擁有權,再讓 host 使用者的 rm -rf 正常運作。
[ -d "$DIST" ] && docker run --rm -v "$PWD":/src -w /src fd2-build:latest \
  chown -R "$(id -u)":"$(id -g)" "$DIST" || true
rm -rf "$APPDIR"
mkdir -p "$APPDIR/usr/bin" "$APPDIR/usr/share/applications" \
         "$APPDIR/usr/share/icons/hicolor/256x256/apps" \
         "$APPDIR/assets/scenarios" "$APPDIR/assets/story"

echo "== 1/4 編譯 Linux binary(docker fd2-build)=="
docker run --rm -v "$PWD":/src -w /src -e GOCACHE=/src/.gocache fd2-build:latest bash -c '
  CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o packaging/dist/AppDir/usr/bin/fd2 ./cmd/fd2
'

echo "== 2/4 組 AppDir(唯讀基底:只放已入庫的原創資產)=="
cp packaging/AppRun "$APPDIR/AppRun"; chmod +x "$APPDIR/AppRun"
cp packaging/fd2.desktop "$APPDIR/fd2.desktop"
cp packaging/fd2.desktop "$APPDIR/usr/share/applications/fd2.desktop"
cp packaging/fd2.png "$APPDIR/fd2.png"
cp packaging/fd2.png "$APPDIR/usr/share/icons/hicolor/256x256/apps/fd2.png"
cp -r assets/scenarios/. "$APPDIR/assets/scenarios/"
cp -r assets/story/.    "$APPDIR/assets/story/"
cp assets/spells.json   "$APPDIR/assets/spells.json"

echo "== 3/4 linuxdeploy 補齊動態函式庫(libX11/libasound 等)=="
docker run --rm -v "$PWD":/src -w /src fd2-build:latest bash -c '
  set -e
  command -v file >/dev/null || (apt-get update -qq && apt-get install -y -qq file)  # appimagetool 依賴
  cd packaging/dist
  [ -f linuxdeploy-x86_64.AppImage ] || curl -sL -o linuxdeploy-x86_64.AppImage \
    https://github.com/linuxdeploy/linuxdeploy/releases/download/continuous/linuxdeploy-x86_64.AppImage
  [ -f appimagetool-x86_64.AppImage ] || curl -sL -o appimagetool-x86_64.AppImage \
    https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
  chmod +x linuxdeploy-x86_64.AppImage appimagetool-x86_64.AppImage
  ./linuxdeploy-x86_64.AppImage --appimage-extract-and-run \
    --appdir AppDir \
    --executable AppDir/usr/bin/fd2 \
    --desktop-file AppDir/fd2.desktop \
    --icon-file AppDir/fd2.png

  echo "== 4/4 封裝 AppImage =="
  ARCH=x86_64 ./appimagetool-x86_64.AppImage --appimage-extract-and-run AppDir FD2-x86_64.AppImage
'

docker run --rm -v "$PWD":/src -w /src fd2-build:latest chown -R "$(id -u)":"$(id -g)" "$DIST"  # 要回擁有權

echo "完成:$DIST/FD2-x86_64.AppImage"
ls -la "$DIST"/*.AppImage
