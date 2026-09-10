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
#   FD2_ORACLE_STATE      可寫檔案覆蓋目錄（接 oracle 的 -state）。原版目錄仍是
#                         唯讀掛載，遊戲的寫入（FD2.SAV／FD2.TMP）落在這個目錄。
#                         用它建立續跑點：打完一關在城鎮存檔，之後從標題 LOAD
#                         直接接上，不必每次從頭跑幾十億指令。
#   FD2_ORACLE_LOCK_ALLY_HP=1
#                         作弊：把我方 HP 壓回歷史最高值，讓長關卡跑得完。
#                         **這是修改路徑**：收據的 state_injections 會寫明注入了
#                         什麼、寫了幾次，runner.json 也會記一筆。這種收據不得
#                         當成一般玩家路徑（PLAYER-E2）證據，也不能用來談傷害、
#                         存活或任何與我方 HP 有關的結論。
#
# 逐幀擷取（判斷畫面時比狀態可靠，狀態層看不出「多畫了什麼」）：
#   FD2_ORACLE_FRAMES=1   啟用，輸出到 <輸出目錄>/frames/
#   FD2_ORACLE_FRAME_STRIDE  取樣間隔指令數（預設 20000，約 20 虛擬毫秒）
#   FD2_ORACLE_FRAME_SETTLE  內容連續相同幾次才寫出，用來濾掉畫到一半的畫面
#   FD2_ORACLE_FRAME_MAX     張數上限（預設 4000）
#   FD2_ORACLE_FRAME_EIP     改以遊戲自己的繪圖進入點為邊界，如 0x11CAC
#   FD2_ORACLE_FRAME_FROM／FD2_ORACLE_FRAME_TO
#                            只在這段指令區間取樣，用來把輸出限在要看的那一段
#   FD2_ORACLE_EIP_WATCH     逗號分隔的十六進位位址（最多 16 個），每一幀記錄
#                            各自的累計進入次數；用來回答「這一段是誰畫的」
set -euo pipefail

repo=$(cd "$(dirname "$0")/.." && pwd)
out=${1:?需要輸出目錄}
plan=${2:-}
# 預設用 FD2 對拍的專屬基底，不用共用的 ~/cht/dosgolem。共用那份隨時可能有別的
# 工作階段改到一半（2026-09-10 就遇到 machine.go 編不過），而收據要能由 commit
# 重現。專屬基底是一個 git worktree，固定在驗證過能重現既有收據的 commit：
#
#   git -C ~/cht/dosgolem worktree add --detach ~/cht/dosgolem-fd2-oracle <commit>
#
# 要跟上 dosgolem 的新功能，就把 worktree checkout 到新 commit，**然後先跑一次
# 重現對照**（同一份控制序列、比對逐幀 indexed_sha256），確認指令流沒變再繼續。
# 設 FD2_DOSGOLEM_ROOT 可以指回共用那份或任何別的路徑。
dos=${FD2_DOSGOLEM_ROOT:-$HOME/cht/dosgolem-fd2-oracle}
if [ ! -d "$dos/apps/fd2/cmd/oracle" ] && [ -d "$HOME/cht/dosgolem/apps/fd2/cmd/oracle" ]; then
  echo "⚠ 找不到專屬基底 $dos，改用共用的 ~/cht/dosgolem（那份可能有人正在改）" >&2
  dos=$HOME/cht/dosgolem
fi
orig=${FD2_ORIG_ROOT:-$repo/org_game/炎龍騎士團/FLAME2}
cpus=${FD2_ORACLE_CPUS:-2}
budget=${FD2_ORACLE_STEPS:-20000000000}
state_dir=${FD2_ORACLE_STATE:-}
lock_ally_hp=${FD2_ORACLE_LOCK_ALLY_HP:-}
if [ -n "$lock_ally_hp" ]; then lock_ally_hp_json=true; else lock_ally_hp_json=false; fi
frames=${FD2_ORACLE_FRAMES:-}
frame_stride=${FD2_ORACLE_FRAME_STRIDE:-20000}
frame_settle=${FD2_ORACLE_FRAME_SETTLE:-0}
frame_max=${FD2_ORACLE_FRAME_MAX:-4000}
frame_eip=${FD2_ORACLE_FRAME_EIP:-}
frame_from=${FD2_ORACLE_FRAME_FROM:-0}
frame_to=${FD2_ORACLE_FRAME_TO:-0}
eip_watch=${FD2_ORACLE_EIP_WATCH:-}

test -d "$dos/apps/fd2/cmd/oracle" || { echo "找不到 dosgolem oracle：$dos" >&2; exit 2; }
test -f "$orig/FD2.EXE" || { echo "找不到固定版本 FD2.EXE：$orig" >&2; exit 2; }
mkdir -p "$out"
out=$(cd "$out" && pwd)

# 收據要自己帶出處。掛進容器的是 dosgolem 的**工作區**，所以真正決定結果的是
# 那個目錄當下 checkout 的內容，不是誰記得自己切在哪一個分支。這裡把 commit、
# 分支與「已追蹤檔案有沒有被改過」寫進 runner.json；未追蹤檔案（別的工作留下
# 的產物）不影響建置結果，分開記。
dos_head=$(git -C "$dos" rev-parse HEAD 2>/dev/null || echo unknown)
dos_branch=$(git -C "$dos" rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
dos_dirty=$(git -C "$dos" status --porcelain --untracked-files=no 2>/dev/null | wc -l)
dos_untracked=$(git -C "$dos" status --porcelain --untracked-files=all 2>/dev/null | grep -c '^??' || true)
cat > "$out/runner.json" <<JSON
{
  "schema_version": 1,
  "kind": "fd2_oracle_runner_provenance",
  "runner": "dosgolem apps/fd2/cmd/oracle",
  "dosgolem_root": "$dos",
  "dosgolem_commit": "$dos_head",
  "dosgolem_branch": "$dos_branch",
  "dosgolem_tracked_dirty_files": $dos_dirty,
  "dosgolem_untracked_files": $dos_untracked,
  "original_root": "$orig",
  "generated_at": "$(date -Iseconds)",
  "lock_ally_hp": $lock_ally_hp_json,
  "state_directory": "${state_dir}",
  "evidence_note": "lock_ally_hp 為 true 時本輪是修改路徑，不得作為一般玩家路徑（PLAYER-E2）證據"
}
JSON
if [ "$dos_dirty" -ne 0 ]; then
  echo "⚠ dosgolem 有 $dos_dirty 個已追蹤檔案被改過，這一輪的收據無法由 commit 重現" >&2
fi
cache=$dos/workplace
mkdir -p "$cache/gocache" "$cache/gomodcache"

if [ -n "$state_dir" ]; then
  mkdir -p "$state_dir"
  state_dir=$(cd "$state_dir" && pwd)
fi
mounts=(-v "$dos:/dos:ro" -v "$orig:/orig:ro" -v "$out:/out:rw"
        -v "$repo/tools/dosgolem_oracle_drive.py:/drive.py:ro"
        -v "$cache/gocache:/gocache" -v "$cache/gomodcache:/gomodcache")
if [ -n "$state_dir" ]; then
  mounts+=(-v "$state_dir:/state:rw")
fi
if [ -n "$plan" ]; then
  test -f "$plan" || { echo "找不到控制序列檔：$plan" >&2; exit 2; }
  mounts+=(-v "$(cd "$(dirname "$plan")" && pwd)/$(basename "$plan"):/plan.jsonl:ro")
fi

docker run --rm --network none --memory 4g --cpus "$cpus" --pids-limit 256 \
  --log-opt max-size=10m --log-opt max-file=3 \
  -u "$(id -u):$(id -g)" "${mounts[@]}" \
  -e GOCACHE=/gocache -e GOMODCACHE=/gomodcache -e HOME=/tmp \
  -e FD2_ORACLE_BUDGET="$budget" \
  -e FD2_ORACLE_FRAMES="$frames" \
  -e FD2_ORACLE_FRAME_STRIDE="$frame_stride" \
  -e FD2_ORACLE_FRAME_SETTLE="$frame_settle" \
  -e FD2_ORACLE_FRAME_MAX="$frame_max" \
  -e FD2_ORACLE_FRAME_EIP="$frame_eip" \
  -e FD2_ORACLE_FRAME_FROM="$frame_from" \
  -e FD2_ORACLE_FRAME_TO="$frame_to" \
  -e FD2_ORACLE_EIP_WATCH="$eip_watch" \
  -e FD2_ORACLE_LOCK_ALLY_HP="$lock_ally_hp" \
  -e FD2_ORACLE_STATE="${state_dir:+/state}" \
  -w /dos "${FD2_ORACLE_IMAGE:-golang:1.24-bookworm}" \
  bash -c '
set -euo pipefail
frameargs=()
if [ -n "$FD2_ORACLE_FRAMES" ]; then
  frameargs=(-frame-dir /out/frames
             -frame-stride "$FD2_ORACLE_FRAME_STRIDE"
             -frame-settle "$FD2_ORACLE_FRAME_SETTLE"
             -frame-max "$FD2_ORACLE_FRAME_MAX"
             -frame-from "$FD2_ORACLE_FRAME_FROM"
             -frame-to "$FD2_ORACLE_FRAME_TO")
  if [ -n "$FD2_ORACLE_FRAME_EIP" ]; then
    frameargs+=(-frame-eip "$FD2_ORACLE_FRAME_EIP")
  fi
  if [ -n "$FD2_ORACLE_EIP_WATCH" ]; then
    frameargs+=(-eip-watch "$FD2_ORACLE_EIP_WATCH")
  fi
fi
cheatargs=()
if [ -n "$FD2_ORACLE_LOCK_ALLY_HP" ]; then
  cheatargs=(-lock-ally-hp)
fi
if [ -n "$FD2_ORACLE_STATE" ]; then
  cheatargs+=(-state "$FD2_ORACLE_STATE")
fi
go run ./apps/fd2/cmd/oracle \
  -exe /orig/FD2.EXE -root /orig -run-dir /out \
  "${cheatargs[@]+"${cheatargs[@]}"}" \
  "${frameargs[@]+"${frameargs[@]}"}" \
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
