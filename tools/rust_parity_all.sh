#!/bin/bash
# R8：以目前程式逐章重跑 Rust 受控重播，產出 30 章總表。
# 輸入表是 docs/data/rust-parity-chapters.json（第 2～30 章）；第 1 章走 tools/rust_parity_ch01.sh。
#
# 用法：tools/rust_parity_all.sh <輸出目錄> [章號…]
# 輸出：<輸出目錄>/chNN/（各章 rust_parity_chapter.sh 的輸出）、<輸出目錄>/summary.tsv。
# 每章先核對槽 SHA-256；原版追蹤只有 eip-trace.jsonl.gz 時，在 <輸出目錄>/oracle-chNN 建硬連結副本並解壓追蹤，
# 原版目錄本身不改。某章失敗不中斷，最後以失敗章數為結束碼。
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd -P)"
out="$(mkdir -p "${1:?輸出目錄}" && cd "$1" && pwd -P)"
shift
table="$root/docs/data/rust-parity-chapters.json"
chapters=("$@")
if [ ${#chapters[@]} -eq 0 ]; then
  chapters=(1 $(python3 -c 'import json,sys; print(" ".join(str(c["chapter"]) for c in json.load(open(sys.argv[1]))["chapters"]))' "$table"))
fi
summary="$out/summary.tsv"
printf 'chapter\tstatus\tcheckpoints\tseconds\trust_save_sha256\n' > "$summary"
failed=0
for ch in "${chapters[@]}"; do
  dir="$out/ch$(printf %02d "$ch")"
  start=$(date +%s)
  if [ "$ch" = 1 ]; then
    bash "$root/tools/rust_parity_ch01.sh" "$dir" > "$dir.log" 2>&1
    status=$?
  else
    row="$(python3 -c '
import json,sys
for c in json.load(open(sys.argv[1]))["chapters"]:
    if c["chapter"] == int(sys.argv[2]):
        print(c["slot"], c["slot_sha256"], c["oracle"])
' "$table" "$ch")"
    if [ -z "$row" ]; then echo "第 $ch 章不在 $table" >&2; failed=$((failed + 1)); continue; fi
    read -r slot sha oracle <<< "$row"
    slot="$root/$slot"; oracle="$root/$oracle"
    if [ "$(sha256sum "$slot" | cut -c1-64)" != "$sha" ]; then
      echo "第 $ch 章槽雜湊不符：$slot" >&2
      printf '%s\tslot-mismatch\t-\t-\t-\n' "$ch" >> "$summary"; failed=$((failed + 1)); continue
    fi
    if [ ! -f "$oracle/eip-trace.jsonl" ] && [ -f "$oracle/eip-trace.jsonl.gz" ]; then
      copy="$out/oracle-ch$(printf %02d "$ch")"
      rm -rf "$copy"; mkdir -p "$copy"
      # 硬連結副本：容器只掛這個目錄，符號連結指向的主機路徑在容器裡解析不到。
      cp -al "$oracle/." "$copy/"
      rm "$copy/eip-trace.jsonl.gz"
      gzip -dc "$oracle/eip-trace.jsonl.gz" > "$copy/eip-trace.jsonl"
      oracle="$copy"
    fi
    bash "$root/tools/rust_parity_chapter.sh" "$ch" "$slot" "$oracle" "$dir" > "$dir.log" 2>&1
    status=$?
  fi
  seconds=$(( $(date +%s) - start ))
  points="$( (wc -l < "$dir/rust/checkpoints.jsonl") 2>/dev/null || echo -)"
  save="$(sha256sum "$dir/rust/FD2.SAV" 2>/dev/null | cut -c1-64)"
  if [ "$status" = 0 ]; then result=passed; else result="failed($status)"; failed=$((failed + 1)); fi
  printf '%s\t%s\t%s\t%s\t%s\n' "$ch" "$result" "$points" "$seconds" "${save:--}" >> "$summary"
  echo "第 $ch 章 $result，$points 筆，${seconds} 秒"
done
echo "失敗 $failed 章；總表 $summary"
exit "$failed"
