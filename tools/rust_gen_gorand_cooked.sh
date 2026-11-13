#!/bin/bash
# 從 Go 1.22.12 的 $GOROOT/src/math/rand/rng.go 機械轉出 rngCooked 表（docs/goal/119、
# remake-rs/crates/fd2-rules/PORTING.md「開工前的決定」第 1 點）。不手抄表格。
# 輸出：remake-rs/crates/fd2-rules/src/battle/rng/cooked.rs（檔頭保留 Go 的 BSD 授權聲明與來源雜湊）。
# 用法：tools/rust_gen_gorand_cooked.sh
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out_dir="$root/remake-rs/crates/fd2-rules/src/battle/rng"
mkdir -p "$out_dir"
tmp="$(mktemp "$out_dir/.cooked.XXXXXX")"
trap 'rm -f "$tmp"' EXIT
timeout 120s docker run --rm --network none --memory 512m --cpus 1 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 --user "$(id -u):$(id -g)" \
  fd2-build-mingw:1.22.12-r1 bash -euo pipefail -c '
    goroot="$(go env GOROOT)"
    src="$goroot/src/math/rand/rng.go"
    ver="$(go version)"
    sum="$(sha256sum "$src" | cut -d" " -f1)"
    echo "// 由 tools/rust_gen_gorand_cooked.sh 自動產生，不要手改。"
    echo "//"
    echo "// 來源：$ver 的 \$GOROOT/src/math/rand/rng.go"
    echo "// 來源 SHA-256：$sum"
    echo "//"
    echo "// 以下是 rng.go 的檔頭與 Go 的 LICENSE 全文；rngCooked 表依 BSD 授權條款轉載。"
    echo "//"
    sed -n "1,3p" "$src"
    echo "//"
    sed "s/^/\/\/ /; s/ *$//" "$goroot/LICENSE"
    echo
    echo "/// Go \`rngCooked\`：\`rngSource.Seed\` 與種子序列逐項 XOR 的 607 項常數表。"
    echo "pub(super) const RNG_COOKED: [i64; 607] = ["
    awk "/rngCooked \\[rngLen\\]int64 = \\[\\.\\.\\.\\]int64\\{/ {on=1; next} on && /^\\t\\}/ {exit} on {print}" "$src" \
      | tr -s " \t," "\n" | grep -E "^-?[0-9]+$" | sed "s/^/    /; s/$/,/"
    echo "];"
  ' >"$tmp"
count="$(grep -cE '^    -?[0-9]+,$' "$tmp")"
test "$count" = 607 || { echo "rngCooked 項數 $count，預期 607" >&2; exit 1; }
mv "$tmp" "$out_dir/cooked.rs"
trap - EXIT
echo "已寫入 $out_dir/cooked.rs（607 項）"
