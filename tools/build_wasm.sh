#!/usr/bin/env bash
# build_wasm.sh 產生網頁版的可重現產物：`fd2.wasm`、`wasm_exec.js` 與 `index.html`。
#
# 產物一律寫到指定的輸出目錄，不寫回 `remake/web/`——那裡的 `wasm_exec.js` 是
# 早期以 root 執行的容器留下的，現在的使用者改不動它；而且產物本來就不該進版控。
#
# 產物包含資產索引（`assets-manifest.json`）與一個指向 `remake/assets` 的符號
# 連結，因為索引裡的路徑是 `assets/…`，而 81 MB 的資產不該複製一份進產物目錄。
# 要在本機開來看的話，把 HTTP 伺服器的根目錄指到輸出目錄即可。
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
python3 "$repo_root/tools/build_asset_manifest.py" "$out/assets-manifest.json"

# 索引裡的路徑是 assets/…，所以資產要在輸出目錄底下看得到。用符號連結而不是
# 複製：那是 81 MB，而且複製一份就會有兩份可能不同步。
rm -f "$out/assets"
ln -s "$repo_root/remake/assets" "$out/assets"

# 原版衍生的分離素材走另一個前綴。兩邊各有 fonts／maps／music／portraits／sfx／
# sprites／ui 七個同名目錄，混在同一個路徑底下會互相蓋掉；桌面版是用
# FD2_ASSET_PACK 分開的，網頁版用 pack/。素材不在公開庫，所以只有指定來源時才建
# 連結——沒有它遊戲會在缺素材時失敗即關閉，那是正確行為。
rm -f "$out/pack"
if [ -n "${FD2_ASSET_PACK:-}" ]; then
    [ -d "$FD2_ASSET_PACK" ] || { echo "FD2_ASSET_PACK 不是目錄：$FD2_ASSET_PACK" >&2; exit 2; }
    ln -s "$(cd -- "$FD2_ASSET_PACK" && pwd)" "$out/pack"
    echo "分離素材連結：$out/pack -> $FD2_ASSET_PACK"
else
    echo "沒有設 FD2_ASSET_PACK，產物不含分離素材；網頁版會停在「缺原版素材」畫面。"
fi

# 資產打成一個檔。逐檔取用在瀏覽器上不可行：Go 的 fs 介面是同步回呼，只能用
# 同步 XMLHttpRequest 去補，而 Chrome 對主執行緒的同步請求節流到每秒兩個左右——
# 這個遊戲光是啟動就要讀六千多個檔案。細節見 index.html 的說明。
pak_args=("assets=$repo_root/remake/assets")
if [ -n "${FD2_ASSET_PACK:-}" ]; then
    pak_args+=("pack=$(cd -- "$FD2_ASSET_PACK" && pwd)")
fi
python3 "$repo_root/tools/build_asset_pak.py" "$out/assets.pak" "${pak_args[@]}"

# 產物存在且非空才算成功——建置的 exit code 不足以證明檔案寫出來了。
for name in fd2.wasm wasm_exec.js index.html assets-manifest.json assets.pak; do
    [ -s "$out/$name" ] || { echo "缺少產物：$out/$name" >&2; exit 3; }
done
[ -e "$out/assets/sprites" ] || { echo "資產連結沒有指到東西：$out/assets" >&2; exit 3; }
echo "已產出 $out："
ls -lh "$out" | tail -n +2 | awk '{printf "  %-16s %s\n", $9, $5}'
