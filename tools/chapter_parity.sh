#!/usr/bin/env bash
# 章對拍的重製側重播＋四 gate 判定（111 章工作單元的第 4、5 步）。
#
# 用法：
#   tools/chapter_parity.sh <章號> <建槽目錄> <dosgolem 輸出目錄> <重製側輸出目錄> [收據路徑]
#
#   <建槽目錄>       含 tools/fd2_chapter_slot.py build 產生的 FD2.SAV 與 manifest.json
#   <dosgolem 輸出>  tools/dosgolem_oracle.sh 的輸出（actions.jsonl、checkpoint-*.{json,png}、runner.json）
#   收據路徑預設 docs/data/ui-traces/parity-chNN.json
#
# 環境變數：
#   FD2_PARITY_PLAN        控制計畫（預設 docs/data/parity-plans/chNN-sample.jsonl）
#   FD2_PARITY_SAVE_ISSUE  重製側尚未寫原版槽 bytes 時引用的 issue 編號（交易 gate 標 blocked）
#   FD2_GO_TEST_IMAGE      預設 fd2-go-test-local:latest
#   FD2_ASSETS_IMAGE       預設 fd2-assets-local:20260829-sfx（有 Pillow）
#   FD2_PARITY_TRACE_KEYS  非空時重播端把每個游標鍵之後的游標／鏡頭寫進 replay.log（找鏡頭分歧用）
#   FD2_PARITY_CPUS        重播與判定容器的 --cpus（預設 4；原版側 oracle 同時在跑時調小）
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
chapter=${1:?章號}
slot_dir=$(cd -- "${2:?建槽目錄}" && pwd)
oracle_run=$(cd -- "${3:?dosgolem 輸出目錄}" && pwd)
out_dir=${4:?重製側輸出目錄}
mkdir -p "$out_dir"
out_dir=$(cd -- "$out_dir" && pwd)
padded=$(printf '%02d' "$chapter")
receipt=${5:-$repo_root/docs/data/ui-traces/parity-ch$padded.json}
plan=${FD2_PARITY_PLAN:-$repo_root/docs/data/parity-plans/ch$padded-sample.jsonl}
go_image=${FD2_GO_TEST_IMAGE:-fd2-go-test-local:latest}
assets_image=${FD2_ASSETS_IMAGE:-fd2-assets-local:20260829-sfx}
cache=${FD2_GO_TEST_CACHE:-$repo_root/work/gocache}
cpus=${FD2_PARITY_CPUS:-4}
mkdir -p "$cache"

for f in "$slot_dir/FD2.SAV" "$slot_dir/manifest.json" "$oracle_run/actions.jsonl" "$oracle_run/runner.json" "$plan"; do
  test -f "$f" || { echo "缺少 $f" >&2; exit 2; }
done

# 完整分離素材根（remake_go_test.sh 的作法：符號連結根，目標寫成容器內路徑）。
pack=$(mktemp -d -t fd2-parity-pack-XXXXXX)
trap 'rm -rf "$pack"' EXIT
for e in "$repo_root"/remake/generated-assets/fd2-original-b97caf22/*; do
  ln -s "/src/remake/generated-assets/fd2-original-b97caf22/$(basename "$e")" "$pack/$(basename "$e")"
done
ln -s /src/remake/assets/locales "$pack/locales"

echo "== 重製側重播（第 $chapter 章）"
docker run --rm --network none --memory 8g --cpus "$cpus" --pids-limit 512 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
  -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod -e FD2_ASSET_PACK=/pack \
  -e FD2_PARITY_CHAPTER="$chapter" -e FD2_PARITY_SLOT=/slot/FD2.SAV \
  -e FD2_PARITY_ORACLE_RUN=/oracle -e FD2_PARITY_OUT=/parity-out \
  -e FD2_PARITY_TRACE_KEYS="${FD2_PARITY_TRACE_KEYS:-}" -e FD2_FOCUS_TRACE_OUT="${FD2_FOCUS_TRACE_OUT:-}" \
  -v "$repo_root:/src" -v "$pack:/pack:ro" -v "$cache:/gocache" \
  -v "$slot_dir:/slot:ro" -v "$oracle_run:/oracle:ro" -v "$out_dir:/parity-out" \
  -w /src/remake "$go_image" \
  with-xvfb go test ./cmd/fd2 -run '^TestChapterParityReplay$' -count=1 -v 2>&1 | tee "$out_dir/replay.log" | grep -v '^XGB\|cutscene\]\|AI walk' | tail -30 || replay_status=$?
# 重播失敗（重製端執行期錯誤、分岔後 Fatalf）也要判 gate：收據要寫出停在哪裡。
echo "重播結束（status=${replay_status:-0}）"

echo "== 四 gate 判定"
docker run --rm --network none --memory 2g --cpus 1 --pids-limit 128 \
  --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" -e HOME=/tmp \
  -v "$repo_root:/repo" -v "$slot_dir:/slot:ro" -v "$oracle_run:/oracle:ro" -v "$out_dir:/parity-out:ro" \
  -w /repo "$assets_image" tools/verify_chapter_parity.py \
  --chapter "$chapter" --oracle /oracle --remake /parity-out \
  --slot-manifest /slot/manifest.json --plan "${plan#$repo_root/}" \
  --out "${receipt#$repo_root/}" --save-blocked-issue "${FD2_PARITY_SAVE_ISSUE:-}" || status=$?
echo "收據：$receipt"
exit "${status:-0}"
