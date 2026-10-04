#!/usr/bin/env bash
# FD2_ORACLE_DIRECT_ENTRY_V1
# tools/dosgolem_oracle.sh 與直接 docker run 共用的容器入口。
# 只接受原版唯讀掛載 /orig、dosgolem 唯讀 /dos、控制器 /drive.py、
# 明確可寫 /out 與選用 /state；不得在主機執行。
set -euo pipefail
test -f /.dockerenv || { echo "oracle entry 必須在 Docker 容器執行" >&2; exit 2; }
test -f /orig/FD2.EXE
test -d /dos/apps/fd2/cmd/oracle
test -f /drive.py
test -d /out
FD2_ORACLE_BUDGET=${FD2_ORACLE_BUDGET:-20000000000}
FD2_ORACLE_FRAMES=${FD2_ORACLE_FRAMES:-}
FD2_ORACLE_FRAME_STRIDE=${FD2_ORACLE_FRAME_STRIDE:-20000}
FD2_ORACLE_FRAME_SETTLE=${FD2_ORACLE_FRAME_SETTLE:-0}
FD2_ORACLE_FRAME_MAX=${FD2_ORACLE_FRAME_MAX:-4000}
FD2_ORACLE_FRAME_EIP=${FD2_ORACLE_FRAME_EIP:-}
FD2_ORACLE_FRAME_FROM=${FD2_ORACLE_FRAME_FROM:-0}
FD2_ORACLE_FRAME_TO=${FD2_ORACLE_FRAME_TO:-0}
FD2_ORACLE_EIP_WATCH=${FD2_ORACLE_EIP_WATCH:-}
FD2_ORACLE_EIP_TRACE=${FD2_ORACLE_EIP_TRACE:-}
FD2_ORACLE_EIP_TRACE_FROM=${FD2_ORACLE_EIP_TRACE_FROM:-0}
FD2_ORACLE_EIP_TRACE_TO=${FD2_ORACLE_EIP_TRACE_TO:-0}
FD2_ORACLE_EIP_TRACE_MAX=${FD2_ORACLE_EIP_TRACE_MAX:-200000}
FD2_ORACLE_LOCK_ALLY_HP=${FD2_ORACLE_LOCK_ALLY_HP:-}
FD2_ORACLE_STATE=${FD2_ORACLE_STATE:-}
export FD2_ORACLE_EIP_TRACE_FROM FD2_ORACLE_EIP_TRACE_TO FD2_ORACLE_EIP_TRACE_MAX
for trace_bound in "$FD2_ORACLE_EIP_TRACE_FROM" "$FD2_ORACLE_EIP_TRACE_TO"; do
  [[ "$trace_bound" =~ ^(0|[1-9][0-9]{0,11})$ ]] || { echo "EIP 追蹤指令範圍格式無效" >&2; exit 2; }
done
[[ "$FD2_ORACLE_EIP_TRACE_MAX" =~ ^[1-9][0-9]{0,5}$ ]] && [ "$FD2_ORACLE_EIP_TRACE_MAX" -le 200000 ] || { echo "EIP 追蹤筆數越界" >&2; exit 2; }
[ "$FD2_ORACLE_EIP_TRACE_TO" -eq 0 ] || [ "$FD2_ORACLE_EIP_TRACE_TO" -ge "$FD2_ORACLE_EIP_TRACE_FROM" ] || { echo "EIP 追蹤終點早於起點" >&2; exit 2; }
# runner.json 也在容器產生；來源 Git 狀態由必要的主機 Git 檢查傳入。
python3 - <<'META'
import datetime, hashlib, json, os, re
from pathlib import Path
env = os.environ
required = ["FD2_ORACLE_SOURCE_COMMIT", "FD2_ORACLE_SOURCE_BRANCH",
            "FD2_ORACLE_SOURCE_DIRTY", "FD2_ORACLE_SOURCE_UNTRACKED",
            "FD2_ORACLE_SOURCE_ROOT", "FD2_ORACLE_ORIGINAL_ROOT"]
if any(not env.get(key) for key in required):
    raise SystemExit("oracle entry 缺少來源 Git／原版定位資料")
if not re.fullmatch(r"[0-9a-f]{40}", env["FD2_ORACLE_SOURCE_COMMIT"]):
    raise SystemExit("oracle entry 來源 commit 無效")
raw = Path("/orig/FD2.EXE").read_bytes()
sha = hashlib.sha256(raw).hexdigest()
if len(raw) != 357074 or sha != "222b7d067ad4450eb9c5f6e6bce1797d54bb050417ba39ced6067f8039f28c4f":
    raise SystemExit("oracle entry 原版 EXE 身分不符")
runner = {
    "schema_version": 1, "kind": "fd2_oracle_runner_provenance",
    "runner": "dosgolem apps/fd2/cmd/oracle",
    "container_entry": "tools/dosgolem_oracle_container.sh",
    "dosgolem_root": env["FD2_ORACLE_SOURCE_ROOT"],
    "dosgolem_commit": env["FD2_ORACLE_SOURCE_COMMIT"],
    "dosgolem_branch": env["FD2_ORACLE_SOURCE_BRANCH"],
    "dosgolem_tracked_dirty_files": int(env["FD2_ORACLE_SOURCE_DIRTY"]),
    "dosgolem_untracked_files": int(env["FD2_ORACLE_SOURCE_UNTRACKED"]),
    "original_root": env["FD2_ORACLE_ORIGINAL_ROOT"],
    "generated_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
    "control_plan": env.get("FD2_ORACLE_CONTROL_PLAN", ""),
    "eip_trace_window": {"from_step": int(env["FD2_ORACLE_EIP_TRACE_FROM"]),
                         "to_step": int(env["FD2_ORACLE_EIP_TRACE_TO"]),
                         "max_entries": int(env["FD2_ORACLE_EIP_TRACE_MAX"])},
    "lock_ally_hp": bool(env.get("FD2_ORACLE_LOCK_ALLY_HP")),
    "force_enemy_clear_declared": bool(env.get("FD2_ORACLE_FORCE_ENEMY_CLEAR")),
    "original_fd2_exe_sha256": sha,
    "original_reference_manifest": "docs/data/fd2-reference-files.json",
    "state_directory": env.get("FD2_ORACLE_SOURCE_STATE", ""),
    "state_injections": [
        "lock_ally_hp 為 true 時，camp 2 record +0x40 HP 會定期壓回該 identity 歷史最高值",
        "force_enemy_clear_declared 為 true 時，控制序列可要求 oracle 依單位陣列將 camp 0 record +0x40 HP 寫為 0；實際次數與筆數見 checkpoint"],
    "evidence_note": "任何上述注入啟用時皆為修改路徑，只可驗證節點、畫面、介面與存檔閉環；不得用於傷害、存活、戰鬥結果或一般玩家路徑（PLAYER-E2）宣稱"
}
Path("/out/runner.json").write_text(json.dumps(runner, ensure_ascii=False, indent=2) + "\n")
META
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
if [ -n "$FD2_ORACLE_EIP_TRACE" ]; then
  frameargs+=(-eip-trace "$FD2_ORACLE_EIP_TRACE"
             -eip-trace-from "$FD2_ORACLE_EIP_TRACE_FROM"
             -eip-trace-to "$FD2_ORACLE_EIP_TRACE_TO"
             -eip-trace-max "$FD2_ORACLE_EIP_TRACE_MAX")
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
python3 /drive.py | tee /out/driver.log
