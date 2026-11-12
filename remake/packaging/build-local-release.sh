#!/bin/bash
# build-local-release.sh — 在本機 Docker 建置三平台正式引擎包，輸出與 release.yml 相同的
# dist-all/<版號>/{patch,smoke,SHA256SUMS.json}。
#
# 與原生 CI 的對照（改一邊就改另一邊）：
#   Linux   build-appimage.sh，與 .github/workflows/build-linux.yml 同一支腳本。
#   Windows Dockerfile.mingw 交叉建置；自檢在 Wine 執行，對照 build-windows.yml。
#   macOS   Dockerfile.osxcross 每弧各編一次再 lipo，對照 build-macos.yml。Linux 上無法執行
#           Mach-O，收據標為 static-verified，不冒充原生自檢。
# 收據的 source_commit 取 HEAD，所以 remake/、VERSION 與本工具必須已提交。
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd -P)"
version="$(tr -d '\r\n' < "$root/VERSION")"
work="$root/work/local-release-$version"
delivery="$root/dist-all/$version"
mingw_image=fd2-build-mingw:1.22.12-r1
osxcross_image=fd2-build-osxcross:15.5-go1.22.12-r1

git -C "$root" diff --quiet HEAD -- remake VERSION tools/copy_tracked_remake_assets.py \
  || { echo "remake／VERSION 有未提交變更，收據的來源提交會不實" >&2; exit 1; }
test -z "$(git -C "$root" ls-files --others --exclude-standard -- remake/assets)" \
  || { echo "remake/assets 有未追蹤檔案" >&2; exit 1; }
test ! -e "$delivery" || { echo "已有 $delivery，不覆寫" >&2; exit 1; }
for platform in linux windows macos; do
  test ! -e "$root/work/release-$version/$platform" || { echo "已有 work/release-$version/$platform" >&2; exit 1; }
done
mkdir -p "$work"

common=(--rm --init --network none --memory 4g --cpus 4 --pids-limit 512
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)"
  --mount "type=bind,source=$root,target=/repo"
  -e HOME=/tmp/fd2-home -e GOCACHE=/tmp/fd2-go-cache -e GOPROXY=off
  -e FD2_RELEASE_BUILDER=local-docker -w /repo/remake)

echo "== Linux"
FD2_BUILD_OUTPUT_ROOT="/repo/work/local-release-$version/linux" FD2_RELEASE_BUILDER=local-docker \
  timeout 1800s bash "$root/remake/packaging/build-appimage.sh"

echo "== Windows"
timeout 1800s docker run "${common[@]}" "$mingw_image" bash -euo pipefail -c '
  version=$(cat /repo/VERSION)
  out="/repo/work/local-release-$version/windows/patch"
  smoke="/repo/work/local-release-$version/windows/smoke"
  stage="/repo/work/release-$version/windows/FD2-$version-windows-x86_64"
  mkdir -p "$out" "$smoke" /tmp/fd2-home /tmp/fd2-go-cache
  export GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++
  python3 packaging/release.py stage "$stage" --platform windows-x86_64
  go build -trimpath -buildvcs=false -ldflags="-s -w -H=windowsgui -X main.buildVersion=$version" \
    -o "$stage/fd2.exe" ./cmd/fd2
  python3 packaging/release.py verify "$stage" --platform windows-x86_64
  mkdir -p /tmp/fd2-empty
  cd /tmp/fd2-empty
  # Ebiten 在 init 就建立視窗，Wine 需要 X 顯示；WINEPREFIX 先建好，Wine 才不檢查 /tmp 擁有者。
  mkdir -p /tmp/fd2-wine
  Xvfb :97 -screen 0 1280x800x24 -nolisten tcp >"$smoke/windows-xvfb.log" 2>&1 &
  xvfb_pid=$!
  trap "wineserver -k 2>/dev/null || true; kill $xvfb_pid 2>/dev/null || true" EXIT
  for attempt in $(seq 1 100); do test ! -S /tmp/.X11-unix/X97 || break; sleep 0.1; done
  test -S /tmp/.X11-unix/X97
  DISPLAY=:97 WINEPREFIX=/tmp/fd2-wine WINEDEBUG=-all FD2_PACKAGE_SELF_CHECK=1 XDG_DATA_HOME=/tmp/fd2-xdg \
    timeout 240s wine "$stage/fd2.exe" >"$smoke/windows-self-check.log" 2>"$smoke/windows-stderr.log" \
    || { cat "$smoke/windows-self-check.log" "$smoke/windows-stderr.log"; exit 1; }
  wineserver -k 2>/dev/null || true
  cat "$smoke/windows-self-check.log"
  cd /repo/remake
  python3 packaging/release.py zip "$stage" --output "$out/FD2-$version-windows-x86_64.zip"
  python3 packaging/release.py attest "$stage" --platform windows-x86_64 \
    --output "$out/FD2-$version-windows-x86_64.zip" --log "$smoke/windows-self-check.log"
'

echo "== macOS"
timeout 2400s docker run "${common[@]}" "$osxcross_image" bash -euo pipefail -c '
  version=$(cat /repo/VERSION)
  out="/repo/work/local-release-$version/macos/patch"
  smoke="/repo/work/local-release-$version/macos/smoke"
  build="/repo/work/release-$version/macos"
  package="$build/FD2-$version-macos-universal"
  app="$package/FD2.app"
  mkdir -p "$out" "$smoke" "$app/Contents/MacOS" /tmp/fd2-home /tmp/fd2-go-cache
  eval "$(osxcross-conf)"
  target="$OSXCROSS_TARGET"
  # Go 1.22 支援的最低 macOS 為 11；不沿用建置機 SDK 版本當最低需求。
  export GOOS=darwin CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=11.0
  python3 packaging/release.py stage "$app/Contents/Resources" --platform macos-universal
  for arch in arm64 x86_64; do
    goarch=$arch; test "$arch" = x86_64 && goarch=amd64
    CC="$arch-apple-$target-clang" CXX="$arch-apple-$target-clang++" GOARCH=$goarch \
      CGO_CFLAGS="-arch $arch" CGO_LDFLAGS="-arch $arch" \
      go build -trimpath -buildvcs=false -ldflags="-s -w -X main.buildVersion=$version" \
      -o "$build/fd2-$goarch" ./cmd/fd2
  done
  lipo="x86_64-apple-$target-lipo"
  otool="x86_64-apple-$target-otool"
  "$lipo" -create "$build/fd2-arm64" "$build/fd2-amd64" -output "$app/Contents/MacOS/fd2"
  python3 - "$app" "$version" <<"PY"
import pathlib, plistlib, sys
from PIL import Image
app = pathlib.Path(sys.argv[1])
version = sys.argv[2]
with (app / "Contents/Info.plist").open("wb") as out:
    plistlib.dump(dict(CFBundleExecutable="fd2", CFBundleIdentifier="tw.wicanr2.fd2re",
        CFBundleName="炎龍騎士團2 重製", CFBundlePackageType="APPL",
        CFBundleShortVersionString=version[2:].split("-")[0],
        CFBundleVersion=version[2:].split("-")[0], FD2ReleaseVersion=version,
        NSHighResolutionCapable=True, CFBundleIconFile="fd2.icns"), out)
icon = Image.open("packaging/fd2.png").convert("RGBA")
icon.save(app / "Contents/Resources/fd2.icns",
          sizes=[(s, s) for s in (16, 32, 64, 128, 256, 512, 1024)])
PY
  chmod +x "$app/Contents/MacOS/fd2"
  cp "$app/Contents/Resources/README.txt" "$package/README.txt"
  python3 packaging/release.py verify "$app/Contents/Resources" --platform macos-universal
  log="$smoke/macos-static-check.log"
  {
    echo "osxcross target $target; MACOSX_DEPLOYMENT_TARGET=$MACOSX_DEPLOYMENT_TARGET"
    "$lipo" -info "$app/Contents/MacOS/fd2"
    "$lipo" "$app/Contents/MacOS/fd2" -verify_arch arm64 x86_64
    for arch in arm64 x86_64; do
      thin="/tmp/fd2-$arch"
      "$lipo" -thin $arch "$app/Contents/MacOS/fd2" -output "$thin"
      "$otool" -l "$thin" | grep -A4 LC_BUILD_VERSION | grep -E "minos|sdk"
      if test $arch = arm64; then "$otool" -l "$thin" | grep -q LC_CODE_SIGNATURE; echo "arm64 LC_CODE_SIGNATURE present"; fi
      foreign=$("$otool" -L "$thin" | tail -n +2 | awk "{print \$1}" | grep -vE "^(/usr/lib/|/System/Library/)" || true)
      test -z "$foreign" || { echo "外部動態庫：$foreign"; exit 1; }
      echo "$arch dylibs system-only"
    done
    grep -c "戰場快照" "$app/Contents/MacOS/fd2" >/dev/null
    echo "battle snapshot strings present"
    file "$app/Contents/MacOS/fd2"
    printf "FD2 %s macOS 靜態檢查通過\n" "$version"
  } >"$log" 2>&1 || { cat "$log"; exit 1; }
  cat "$log"
  rm -f "$build/fd2-arm64" "$build/fd2-amd64"
  python3 packaging/release.py zip "$package" --output "$out/FD2-$version-macos-universal.zip"
  python3 packaging/release.py attest "$app/Contents/Resources" --platform macos-universal \
    --output "$out/FD2-$version-macos-universal.zip" --log "$log" --static
'

echo "== 彙整"
timeout 600s docker run "${common[@]}" -w /repo fd2-build-appimage:latest bash -euo pipefail -c '
  version=$(cat VERSION)
  python3 remake/packaging/release.py collect "work/local-release-$version" \
    --output "dist-all/$version/patch" --allow-static macos-universal
  mkdir -p "dist-all/$version/smoke"
  mv "dist-all/$version/patch/SHA256SUMS.json" "dist-all/$version/SHA256SUMS.json"
  mv "dist-all/$version/patch/VALIDATION.json" "dist-all/$version/smoke/VALIDATION.json"
  find "work/local-release-$version" -path "*/smoke/*.log" -exec cp {} "dist-all/$version/smoke/" \;
'
find "$root/work/local-release-$version" "$root/work/release-$version" "$delivery" -user root -print -quit | grep -q . \
  && { echo "發現 root 擁有的輸出" >&2; exit 1; }
echo "完成：$delivery"
