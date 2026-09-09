#!/usr/bin/env bash
# 本機操作驗收完整版；沿用鎖版 AppImage image，原版分離包絕不進公開包。
set -euo pipefail
if [[ ${1:-} != --inside ]]; then
  [[ $# == 1 ]] || { echo "用法：$0 完整執行期素材目錄" >&2; exit 2; }
  root=$(cd "$(dirname "$0")/../.." && pwd -P)
  pack=$(readlink -f "$1")
  timeout 600s docker run --rm --network none --memory 4g --cpus 2 --pids-limit 384 \
    --user "$(id -u):$(id -g)" -e HOME=/tmp/home -e GOCACHE=/tmp/go-cache \
    -v "$root":/repo:ro -v "$pack":/pack:ro -v "$root/dist-all":/out \
    -w /repo/remake fd2-build-appimage \
    bash /repo/remake/packaging/build-local-input-review.sh --inside
  exit
fi
[[ -f /.dockerenv ]] || { echo "建置只能在 Docker 內執行" >&2; exit 2; }
version=$(tr -d '\r\n' </repo/VERSION)
[[ $version =~ ^v\.[0-9]+\.[0-9]+\.[0-9]+-[0-9]{8}$ ]] || exit 2
[[ $(stat -c %u /out) == "$(id -u)" ]] || { echo "交付目錄擁有權不符" >&2; exit 2; }
target="/out/$version"
[[ ! -e $target ]] || { echo "拒絕覆寫既有交付：$target" >&2; exit 2; }
for required in manifest.json surfaces palette animations; do
  [[ -e /pack/$required ]] || { echo "素材不完整：$required" >&2; exit 2; }
done
PYTHONPATH=/repo/tools python3 /repo/tools/validate_separated_asset_pack.py \
  /pack/manifest.json --runtime-assets /repo/remake/assets \
  --coverage-summary /repo/docs/data/fd2-source-resource-coverage-summary.json
appdir=/tmp/AppDir
mkdir -p "$appdir/usr/bin" "$appdir/usr/share/applications" \
  "$appdir/usr/share/icons/hicolor/256x256/apps" /tmp/home /tmp/go-cache
CGO_ENABLED=1 go build -trimpath -buildvcs=false \
  -ldflags="-s -w -X main.buildVersion=$version" -o "$appdir/usr/bin/fd2" ./cmd/fd2
install -m 0755 packaging/AppRun "$appdir/AppRun"
install -m 0644 packaging/fd2.desktop "$appdir/fd2.desktop"
install -m 0644 packaging/fd2.desktop "$appdir/usr/share/applications/fd2.desktop"
install -m 0644 packaging/fd2.png "$appdir/fd2.png"
install -m 0644 packaging/fd2.png "$appdir/usr/share/icons/hicolor/256x256/apps/fd2.png"
install -m 0644 /repo/LICENSE "$appdir/LICENSE"
mkdir -p "$appdir/assets"
cp -a /pack/. "$appdir/assets/"
python3 /repo/tools/copy_tracked_remake_assets.py --repo /repo --destination "$appdir/assets"
python3 - "$appdir/assets/editor-canonical" <<'PY'
import hashlib, json, pathlib, sys
root = pathlib.Path(sys.argv[1])
summary = json.loads((root/'bundle-summary.json').read_text())
for entry in summary['documents']:
    path = root/entry['output']
    raw = path.read_bytes()
    document = json.loads(raw)
    if hashlib.sha256(raw).hexdigest() != entry['sha256'] or document.get('document_id') != entry['document_id']:
        raise SystemExit('拒絕封裝：canonical 文件或清冊不一致：'+entry['output'])
print('已驗證封裝內 canonical 文件與清冊雜湊')
PY
cd /tmp
/opt/appimage-tools/linuxdeploy.AppImage --appimage-extract-and-run \
  --appdir "$appdir" --executable "$appdir/usr/bin/fd2" \
  --desktop-file "$appdir/fd2.desktop" --icon-file "$appdir/fd2.png"
name="FD2-$version-linux-x86_64-full.AppImage"
ARCH=x86_64 /opt/appimage-tools/appimagetool.AppImage --appimage-extract-and-run \
  --runtime-file /opt/appimage-tools/runtime-x86_64 "$appdir" "/tmp/$name"
mkdir -p "$target/full" "$target/release" "$target/promo"
mv "/tmp/$name" "$target/full/$name"
python3 - "$target" "$version" "$name" <<'PY'
import hashlib, json, pathlib, subprocess, sys
root, version, name = pathlib.Path(sys.argv[1]), sys.argv[2], sys.argv[3]
sha = lambda p: hashlib.sha256(p.read_bytes()).hexdigest()
source = []
for folder in ('cmd', 'internal'):
    for p in sorted((pathlib.Path('/repo/remake')/folder).rglob('*.go')):
        source.append([str(p.relative_to('/repo')), sha(p)])
app = root/'full'/name
manifest = dict(version=version, kind='local_input_audit_candidate', public_distribution=False,
    platform='linux', architecture='x86_64', sha256=sha(app),
    engine_head=subprocess.check_output(['git','-C','/repo','rev-parse','HEAD']).decode().strip(),
    source_files_sha256=hashlib.sha256(json.dumps(source).encode()).hexdigest(),
    source_files=source, asset_manifest_sha256=sha(pathlib.Path('/pack/manifest.json')),
    build_script='remake/packaging/build-local-input-review.sh',
    status='待實際 AppImage 操作驗收；不是穩定版發布',
    rights='包含原版衍生素材，僅限本機或明確授權的私人交付')
(root/'full'/'manifest.json').write_text(json.dumps(manifest,ensure_ascii=False,indent=2)+'\n')
(root/'full'/'SHA256SUMS').write_text(f'{sha(app)}  {name}\n')
(root/'release'/'STATUS.md').write_text(f'{version} 只作本機操作驗收，尚未公開發布。\n')
(root/'promo'/'STATUS.md').write_text('本次操作修正沒有新增宣傳素材。\n')
print(app, sha(app))
PY
stat -c '%u:%g %n' "$target/full/$name" "$target/full/manifest.json"
