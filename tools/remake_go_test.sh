#!/usr/bin/env bash
# remake_go_test.sh 在 fd2-go-test-local 內跑重製端回歸，並把結果與
# docs/data/regression-baseline-20260909.json 做差異比對。
#
# 用法：
#   tools/remake_go_test.sh <素材根> [輸出 log] [套件…]
#
# <素材根> 是完整分離素材根（含 ui/、palette/、locales/…）。分離素材不在公開庫，
# 見 AGENTS.md 的私人素材保存庫。素材根缺件會讓失敗數大幅膨脹，而且失敗訊息
# 看起來像功能缺陷，所以這裡會先檢查幾個必要目錄。
#
# 沒有 locales/ 的素材根（例如 remake/generated-assets/fd2-original-b97caf22/）
# 要先疊上儲存庫的 remake/assets/locales。巢狀 bind mount 掛不進唯讀掛載點，
# 作法是組一個符號連結根，連結目標寫成容器內路徑：
#
#   root=$(mktemp -d)
#   for e in remake/generated-assets/fd2-original-b97caf22/*; do
#     ln -s "/src/${e}" "$root/$(basename "$e")"
#   done
#   ln -s /src/remake/assets/locales "$root/locales"
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
pack_root=${1:?需要完整分離素材根}
log=${2:-$(mktemp -t fd2-remake-go-test-XXXXXX.log)}
shift || true
shift || true
packages=("$@")
if [ ${#packages[@]} -eq 0 ]; then
    packages=("./...")
fi

image=${FD2_GO_TEST_IMAGE:-fd2-go-test-local:latest}
cache=${FD2_GO_TEST_CACHE:-$(mktemp -d -t fd2-gocache-XXXXXX)}
baseline=${FD2_REGRESSION_BASELINE:-$repo_root/docs/data/regression-baseline-20260909.json}

# 符號連結根的目標是容器內路徑（/src/…），在主機上解析不了，所以這裡接受
# dangling symlink：只要那一項存在就算數，真正的可讀性由容器內的測試判定。
for entry in ui palette locales; do
    if [ ! -e "$pack_root/$entry" ] && [ ! -L "$pack_root/$entry" ]; then
        echo "素材根缺少 $entry：$pack_root" >&2
        exit 2
    fi
done

set +e
docker run --rm --network none --memory 8g --cpus 4 --pids-limit 512 \
    --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod -e FD2_ASSET_PACK=/pack \
    -v "$repo_root:/src" -v "$pack_root:/pack:ro" -v "$cache:/gocache" \
    -w /src/remake "$image" \
    with-xvfb go test "${packages[@]}" -count=1 > "$log" 2>&1
status=$?
set -e

echo "log=$log exit=$status"
python3 - "$baseline" "$log" <<'PY'
import json
import re
import sys

baseline_path, log_path = sys.argv[1], sys.argv[2]
baseline = set(json.load(open(baseline_path, encoding="utf-8"))["known_failures"])
now = set()
for line in open(log_path, encoding="utf-8", errors="replace"):
    match = re.match(r"^--- FAIL: (\S+)", line.strip())
    if match:
        now.add(match.group(1))
print(f"baseline {len(baseline)} now {len(now)}")
print("新增:", ", ".join(sorted(now - baseline)) or "無")
print("修復:", ", ".join(sorted(baseline - now)) or "無")
PY
