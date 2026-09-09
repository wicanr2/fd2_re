#!/usr/bin/env bash
# FD2 原版側對拍執行器（dosgolem `apps/fd2/cmd/oracle`）的受版控驅動。
#
# 原版畫面、輸入、虛擬時間與狀態收據一律由這支產生。DOSBox 只在 dosgolem
# 尚無該能力時作輔助診斷，不得登錄為對拍收據；見 AGENTS.md 的
# 「原版側對拍執行器」。
#
# 用法：
#   tools/dosgolem_oracle.sh <輸出目錄> [控制序列.jsonl]
#
# 控制序列每行一個 JSON 物件：{"key": "enter", "steps": 3000000}
# 省略控制序列時只跑到第一個 BIOS 等待邊界，輸出 checkpoint-0000 供檢視。
#
# 環境變數：
#   FD2_DOSGOLEM_ROOT     dosgolem 儲存庫路徑（預設 ~/cht/dosgolem）
#   FD2_ORIG_ROOT         原版資料目錄（預設本儲存庫的 org_game/…/FLAME2）
#   FD2_ORACLE_CPUS       容器 CPU 上限（預設 2）
#   FD2_ORACLE_STEPS      指令預算上限（預設 20000000000）
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
out=${1:?需要輸出目錄}
plan=${2:-}
dos=${FD2_DOSGOLEM_ROOT:-$HOME/cht/dosgolem}
orig=${FD2_ORIG_ROOT:-$repo/org_game/炎龍騎士團/FLAME2}
cpus=${FD2_ORACLE_CPUS:-2}
budget=${FD2_ORACLE_STEPS:-20000000000}

test -d "$dos/apps/fd2/cmd/oracle" || { echo "找不到 dosgolem oracle：$dos" >&2; exit 2; }
test -f "$orig/FD2.EXE" || { echo "找不到固定版本 FD2.EXE：$orig" >&2; exit 2; }
mkdir -p "$out"
out=$(cd "$out" && pwd)
cache=$dos/workplace
mkdir -p "$cache/gocache" "$cache/gomodcache"

mounts=(-v "$dos:/dos:ro" -v "$orig:/orig:ro" -v "$out:/out:rw"
        -v "$repo/tools/dosgolem_oracle_drive.py:/drive.py:ro"
        -v "$cache/gocache:/gocache" -v "$cache/gomodcache:/gomodcache")
if [ -n "$plan" ]; then
  test -f "$plan" || { echo "找不到控制序列檔：$plan" >&2; exit 2; }
  mounts+=(-v "$(cd "$(dirname "$plan")" && pwd)/$(basename "$plan"):/plan.jsonl:ro")
fi

docker run --rm --network none --memory 4g --cpus "$cpus" --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" "${mounts[@]}" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomodcache -e HOME=/tmp \
  -e FD2_ORACLE_BUDGET="$budget" \
  -w /dos "${FD2_ORACLE_IMAGE:-golang:1.24-bookworm}" \
  bash -c '
set -euo pipefail
go run ./apps/fd2/cmd/oracle \
  -exe /orig/FD2.EXE -root /orig -run-dir /out \
  -steps "$FD2_ORACLE_BUDGET" -heap-mib 32 >/out/oracle.log 2>&1 &
oracle=$!
cleanup() { kill "$oracle" 2>/dev/null || true; wait "$oracle" 2>/dev/null || true; }
trap cleanup EXIT
for _ in $(seq 1 900); do
  test -f /out/current.json && break
  kill -0 "$oracle" 2>/dev/null || { echo "oracle 已結束" >&2; tail -20 /out/oracle.log >&2; exit 3; }
  sleep 1
done
test -f /out/current.json || { echo "等待第一個控制邊界逾時" >&2; exit 4; }
python3 /drive.py
'
echo "$out"
