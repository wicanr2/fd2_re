#!/usr/bin/env bash
# build_wasm.sh 產生網頁版的可重現產物：`fd2.wasm`、`wasm_exec.js` 與 `index.html`。
#
# 產物一律寫到指定的輸出目錄，不寫回 `remake/web/`——那裡的 `wasm_exec.js` 是
# 早期以 root 執行的容器留下的，現在的使用者改不動它；而且產物本來就不該進版控。
#
# ⚠ 這支只保證「編得出來」。網頁版目前**還載不到資產**：`assetGlob` 走
# `filepath.Glob`，在 js/wasm 底下沒有檔案系統可掃，所以遊戲會在缺資產時失敗即
# 關閉。要能玩需要一層 `fs.FS` 抽象加一份資產清單（HTTP 上沒有「列目錄」），
# 見工作清單 wasm-web-release。
#
# 用法：tools/build_wasm.sh [輸出目錄]
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
out=${1:-$repo_root/remake/web/dist}
image=${FD2_GO_TEST_IMAGE:-fd2-go-test-local:latest}
cache=${FD2_GO_TEST_CACHE:-$(mktemp -d -t fd2-wasm-cache-XXXXXX)}
mkdir -p "$out" "$cache"

docker run --rm --network none --memory 4g --cpus "${FD2_WASM_CPUS:-2}" \
    --pids-limit 256 --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
    -e GOOS=js -e GOARCH=wasm \
    -v "$repo_root:/src:ro" -v "$cache:/gocache" -v "$out:/out" \
    -w /src/remake "$image" \
    sh -c 'go build -trimpath -o /out/fd2.wasm ./cmd/fd2 && \
           cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm_exec.js 2>/dev/null || \
           cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" /out/wasm_exec.js'

cp "$repo_root/remake/web/index.html" "$out/index.html"

# 產物存在且非空才算成功——建置的 exit code 不足以證明檔案寫出來了。
for name in fd2.wasm wasm_exec.js index.html; do
    [ -s "$out/$name" ] || { echo "缺少產物：$out/$name" >&2; exit 3; }
done
echo "已產出 $out："
ls -lh "$out" | tail -n +2 | awk '{printf "  %-16s %s\n", $9, $5}'
