#!/usr/bin/env bash
# ida.sh 是本專案 IDA Pro 9.4／Hex-Rays 的受版控入口。
#
# 為什麼要包成腳本：原版放在中文目錄（org_game/炎龍騎士團/…），IDA 9.4 的批次
# 腳本會以 ASCII codec 開輸入路徑而爆掉。修法不是複製或改名原檔，而是把它唯讀
# bind mount 成容器內的短 ASCII 路徑；資料庫寫到工作目錄，原檔一個位元都不動。
#
# 用法：
#   tools/ida.sh <IDAPython 腳本> <輸出檔> [位址…]
#
# 位址以空白分隔的十六進位傳給腳本（環境變數 FD2_IDA_ADDRESSES），輸出路徑走
# FD2_IDA_OUTPUT。兩者都是既有 tools/ida_*.py 的介面。
#
# 成功的判準是**輸出檔存在且內容正確**。IDA 的 exit code、stdout 與 `.i64` 的
# 存在都不足以證明匯出完成——IDAPython 失敗時可以完全沒有輸出也沒有錯誤訊息。
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
script=${1:?需要 IDAPython 腳本（tools/ 底下的檔名）}
output=${2:?需要輸出檔路徑}
shift 2

image=${FD2_IDA_IMAGE:-fd2-ida-authorized-local:latest}
original=${FD2_ORIG_EXE:-$repo_root/org_game/炎龍騎士團/FLAME2/FD2.EXE}
[ -f "$original" ] || { echo "找不到原版執行檔：$original" >&2; exit 2; }

work=$(mktemp -d -t fd2-ida-XXXXXX)
trap 'rm -rf "$work"' EXIT
# 資料庫要可寫，原檔唯讀：把原檔的目錄唯讀掛進去，另給一個可寫的工作目錄。
cp "$original" "$work/FD2.EXE"   # IDA 在輸入旁建 .i64，所以輸入放可寫目錄
chmod u+w "$work/FD2.EXE"

docker run --rm --network none --memory 8g --cpus "${FD2_IDA_CPUS:-2}" \
    --pids-limit 512 --log-opt max-size=10m --log-opt max-file=3 \
    -u "$(id -u):$(id -g)" \
    -v "$work:/work" -v "$repo_root/tools:/work/tools:ro" \
    -e HOME=/work -e FD2_IDA_ADDRESSES="$*" -e FD2_IDA_OUTPUT=/work/out.txt \
    -w /work "$image" \
    idat -A "-S/work/tools/$(basename "$script")" /work/FD2.EXE >"$work/ida.log" 2>&1 || true

if [ ! -s "$work/out.txt" ]; then
    echo "IDA 沒有產生輸出（$work/ida.log 的最後幾行）：" >&2
    tail -20 "$work/ida.log" >&2 || true
    exit 3
fi
# 原檔一個位元都不能動：比對複本與原檔。
cmp -s "$work/FD2.EXE" "$original" || { echo "複本與原檔不同，中止" >&2; exit 4; }
mkdir -p "$(dirname "$output")"
cp "$work/out.txt" "$output"
echo "已輸出 $output（$(wc -l <"$output") 行）"
