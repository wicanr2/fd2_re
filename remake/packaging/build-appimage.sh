#!/bin/bash
# 正式 Linux 引擎包；編譯、組裝及驗證均在一次性容器執行。
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd -P)"
test -d "$root"
original_mount=()
if test -d "$root/org_game"; then original_mount=(--mount type=bind,source="$root/org_game",target=/repo/org_game,readonly); fi
docker run --rm --init --network none --memory 3g --cpus 2 --pids-limit 384 \
  "${original_mount[@]}" --user "$(id -u):$(id -g)" --mount type=bind,source="$root",target=/repo \
  -e FD2_BUILD_OUTPUT_ROOT -e HOME=/tmp/fd2-home -e GOCACHE=/tmp/fd2-go-cache -w /repo/remake \
  fd2-build-appimage:latest bash -euo pipefail -c '
    version=$(cat /repo/VERSION)
    stage="/repo/work/release-$version/linux/AppDir"
    delivery="${FD2_BUILD_OUTPUT_ROOT:-/repo/dist-all}"
    out="$delivery/$version/patch"
    smoke="$delivery/$version/smoke"
    mkdir -p "$out" "$smoke" /tmp/fd2-home /tmp/fd2-go-cache
    test ! -e "$stage"
    python3 packaging/release.py stage "$stage" --platform linux-x86_64
    mkdir -p "$stage/usr/bin" "$stage/usr/share/metainfo"
    CGO_ENABLED=1 go build -trimpath -buildvcs=false \
      -ldflags="-s -w -X main.buildVersion=$version" -o "$stage/usr/bin/fd2" ./cmd/fd2
    install -m 0755 packaging/AppRun "$stage/AppRun"
    install -m 0644 packaging/fd2.desktop "$stage/fd2.desktop"
    install -m 0644 packaging/fd2.png "$stage/fd2.png"
    install -m 0644 packaging/fd2.appdata.xml "$stage/usr/share/metainfo/fd2.appdata.xml"
    mkdir -p /tmp/fd2-appimage
    cd /tmp/fd2-appimage
    ARCH=x86_64 /opt/appimage-tools/linuxdeploy.AppImage --appimage-extract-and-run \
      --appdir "$stage" --executable "$stage/usr/bin/fd2" \
      --desktop-file "$stage/fd2.desktop" --icon-file "$stage/fd2.png"
    mkdir -p "$stage/third-party/system"
    for name in libxau6 libxdmcp6 libbsd0 libmd0; do
      test -f "/usr/share/doc/$name/copyright"
      cp "/usr/share/doc/$name/copyright" "$stage/third-party/system/$name-copyright"
    done
    archive="$out/FD2-$version-linux-x86_64.AppImage"
    test ! -e "$archive"
    ARCH=x86_64 /opt/appimage-tools/appimagetool.AppImage --appimage-extract-and-run \
      --runtime-file /opt/appimage-tools/runtime-x86_64 "$stage" "$archive"
    python3 /repo/remake/packaging/release.py verify "$stage" --platform linux-x86_64
    Xvfb :99 -screen 0 1280x800x24 -nolisten tcp >"$smoke/linux-xvfb.log" 2>&1 &
    xvfb_pid=$!
    trap "kill $xvfb_pid 2>/dev/null || true; wait $xvfb_pid 2>/dev/null || true" EXIT
    for attempt in {1..100}; do
      test ! -S /tmp/.X11-unix/X99 || break
      sleep 0.1
    done
    test -S /tmp/.X11-unix/X99
    mkdir -p /tmp/fd2-empty
    cd /tmp/fd2-empty
    DISPLAY=:99 FD2_PACKAGE_SELF_CHECK=1 XDG_DATA_HOME=/tmp/fd2-xdg \
      timeout 90s "$archive" --appimage-extract-and-run >"$smoke/linux-self-check.log" 2>&1
    cat "$smoke/linux-self-check.log"
    python3 /repo/remake/packaging/release.py attest "$stage" --platform linux-x86_64 \
      --output "$archive" --log "$smoke/linux-self-check.log"
    chmod 0755 "$archive"
  '
