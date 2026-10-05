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
FD2_ORACLE_HEAP_PROFILE=${FD2_ORACLE_HEAP_PROFILE:-adapter}
case "$FD2_ORACLE_HEAP_PROFILE" in adapter|native) ;; *) echo "未知近堆模式" >&2; exit 2 ;; esac
export FD2_ORACLE_HEAP_PROFILE
FD2_ORACLE_FRAMES=${FD2_ORACLE_FRAMES:-}
FD2_ORACLE_FRAME_STRIDE=${FD2_ORACLE_FRAME_STRIDE:-20000}
FD2_ORACLE_FRAME_SETTLE=${FD2_ORACLE_FRAME_SETTLE:-0}
FD2_ORACLE_FRAME_MAX=${FD2_ORACLE_FRAME_MAX:-4000}
FD2_ORACLE_FRAME_EIP=${FD2_ORACLE_FRAME_EIP:-}
FD2_ORACLE_FRAME_UNITS=${FD2_ORACLE_FRAME_UNITS:-}
FD2_ORACLE_MAP_STATE=${FD2_ORACLE_MAP_STATE:-}
FD2_ORACLE_FRAME_FROM=${FD2_ORACLE_FRAME_FROM:-0}
FD2_ORACLE_FRAME_TO=${FD2_ORACLE_FRAME_TO:-0}
FD2_ORACLE_EIP_WATCH=${FD2_ORACLE_EIP_WATCH:-}
FD2_ORACLE_EIP_TRACE=${FD2_ORACLE_EIP_TRACE:-}
FD2_ORACLE_EIP_TRACE_FROM=${FD2_ORACLE_EIP_TRACE_FROM:-0}
FD2_ORACLE_EIP_TRACE_TO=${FD2_ORACLE_EIP_TRACE_TO:-0}
FD2_ORACLE_EIP_TRACE_MAX=${FD2_ORACLE_EIP_TRACE_MAX:-200000}
FD2_ORACLE_LOCK_ALLY_HP=${FD2_ORACLE_LOCK_ALLY_HP:-}
FD2_ORACLE_STATE=${FD2_ORACLE_STATE:-}
if [ -n "$FD2_ORACLE_FRAME_UNITS" ] && [ "$FD2_ORACLE_FRAME_UNITS" != "1" ]; then
  echo "FRAME_UNITS只接受空值或1" >&2; exit 2
fi
if [ "$FD2_ORACLE_FRAME_UNITS" = "1" ] && [ -z "$FD2_ORACLE_FRAMES" ]; then
  echo "FRAME_UNITS需要FRAMES" >&2; exit 2
fi
if [ -n "$FD2_ORACLE_MAP_STATE" ] && [ "$FD2_ORACLE_MAP_STATE" != "1" ]; then
  echo "MAP_STATE只接受空值或1" >&2; exit 2
fi
if [ "$FD2_ORACLE_MAP_STATE" = "1" ] && [ -z "$FD2_ORACLE_FRAMES" ] && [ -z "$FD2_ORACLE_EIP_TRACE" ]; then
  echo "MAP_STATE需要FRAMES或EIP_TRACE" >&2; exit 2
fi
export FD2_ORACLE_MAP_STATE
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

def near_heap_policy(root):
    # #167：綁定已審查配置器與呼叫端；不從檔名或 commit 猜政策。
    reviewed = {
        "internal/machine/watcom_runtime.go": "00c75b7a4172861c7c2d39187df3c962fb598db74a6e340cdfd61b9eb7f6907f",
        "apps/fd2/cmd/oracle/main.go": "d06565caef7252eeb7ad12c1c50e5c4d1961b9b119c33dac3c988ad1265e4bd3"}
    reviewed_main_revisions = {"d06565caef7252eeb7ad12c1c50e5c4d1961b9b119c33dac3c988ad1265e4bd3", "9509295de415e7c68a1144abd824794c66dfe00ef95e7cbc243a187242b4a234", 'dcea1824f58cff63c76f706ac9dc076eb2523919722b8c0412a323f64a625925'}
    native_reviewed = {'internal/machine/watcom_runtime.go': '00c75b7a4172861c7c2d39187df3c962fb598db74a6e340cdfd61b9eb7f6907f', 'apps/fd2/cmd/oracle/main.go': 'dcea1824f58cff63c76f706ac9dc076eb2523919722b8c0412a323f64a625925', 'internal/cpu386/cpu.go': '3059653f53fe37a44be270184cb9cb0dc4ee5ba4c309f79d79b77e70c8e11267', 'internal/machine/le_startup.go': 'bc80372d495a58778ad56b2f5d9e6783d49319e074f1a086a10e334080c1d206'}
    profile = os.environ["FD2_ORACLE_HEAP_PROFILE"]
    sources = []
    for name in reviewed:
        path = root / name
        sha = hashlib.sha256(path.read_bytes()).hexdigest() if path.is_file() else None
        sources.append({"path": name, "sha256": sha})
    known = all(source["sha256"] in reviewed_main_revisions
                if source["path"] == "apps/fd2/cmd/oracle/main.go"
                else source["sha256"] == reviewed[source["path"]] for source in sources)
    if profile == "native":
        native_sources = [{"path": name, "sha256": hashlib.sha256((root/name).read_bytes()).hexdigest()}
                          for name in native_reviewed]
        if any(source["sha256"] != native_reviewed[source["path"]] for source in native_sources):
            raise SystemExit("native原生近堆來源未通過已審查清冊")
        return {
            "kind": "original_instructions_with_existing_platform_adapters",
            "status": "reviewed", "profile": "native", "sources": native_sources,
            "original_entries": {"_nmalloc": "0x36D26", "_nfree": "0x37426", "__Init_Argv": "0x46114"},
            "preserved_hooks": ["memset 0x375C0", "int386 0x36D98", "LEVideo", "BIOS clock", "BIOS keyboard", "FD2StartupDOS/DPMI"],
            "capacity": "original instructions request growth from existing DPMI profile; adapter heap-mib is not a native capacity",
            "allocation_hook": None, "free_hook": None,
            "reused_allocation_contents": "determined by original instructions; no adapter clearing",
            "original_allocator_parity": "bounded original execution; all branches and real hardware unverified",
            "evidence_contract": "docs/data/ida/fd2_ch18_oracle_stosb_20261003.json#allocator_formal_native_profile",
            "evidence_restriction": "原始近堆指令仍搭配既有DOS/DPMI與周邊硬體近似；有界BOOT不代表完整章或全原版硬體一致。"}
    return {
        "kind": "dosgolem_runtime_approximation",
        "status": "reviewed" if known else "unknown",
        "sources": sources,
        "heap_mib": 32,
        "allocation_hook": "WatcomNearHeap.Handle / allocate",
        "free_hook": "WatcomNearHeap.handleFree / release",
        "reused_allocation_contents": "zeroed" if known else "unknown",
        "original_allocator_parity": "unverified",
        "evidence_contract": "docs/data/ida/fd2_terrain_mode3_review_20261001.json#oracle_heap_policy_correction",
        "evidence_restriction": "近堆替代實作的重用位址與初值未經原版配置器校準；受此影響的透明底色只能列工具政策結果，不能單獨證明原版 malloc 語意。"}

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
    "map_runtime_records": {"enabled": env.get("FD2_ORACLE_MAP_STATE") == "1",
                            "globals": 16, "palette_bytes": 768,
                            "scope": "same-instruction read-only map globals/view/units/palette; invalid source false"},
    "frame_unit_records": {"enabled": env.get("FD2_ORACLE_FRAME_UNITS") == "1",
                           "row_bytes": 80, "max_units": 128,
                           "scope": "accepted PNG same-instruction read-only units; invalid source marked false"},
    "eip_trace_window": {"from_step": int(env["FD2_ORACLE_EIP_TRACE_FROM"]),
                         "to_step": int(env["FD2_ORACLE_EIP_TRACE_TO"]),
                         "max_entries": int(env["FD2_ORACLE_EIP_TRACE_MAX"])},
    "lock_ally_hp": bool(env.get("FD2_ORACLE_LOCK_ALLY_HP")),
    "force_enemy_clear_declared": bool(env.get("FD2_ORACLE_FORCE_ENEMY_CLEAR")),
    "original_fd2_exe_sha256": sha,
    "original_reference_manifest": "docs/data/fd2-reference-files.json",
    "heap_profile": env["FD2_ORACLE_HEAP_PROFILE"],
    "near_heap_policy": near_heap_policy(Path("/dos")),
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
  if [ "$FD2_ORACLE_FRAME_UNITS" = "1" ]; then
    frameargs+=(-frame-units)
  fi
  if [ -n "$FD2_ORACLE_EIP_WATCH" ]; then
    frameargs+=(-eip-watch "$FD2_ORACLE_EIP_WATCH")
  fi
fi
if [ "$FD2_ORACLE_MAP_STATE" = "1" ]; then
  frameargs+=(-map-state)
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
heapargs=(-heap-profile "$FD2_ORACLE_HEAP_PROFILE")
if [ "$FD2_ORACLE_HEAP_PROFILE" = adapter ]; then heapargs+=(-heap-mib 32); fi
go run ./apps/fd2/cmd/oracle \
  "${heapargs[@]}" \
  -exe /orig/FD2.EXE -root /orig -run-dir /out \
  "${cheatargs[@]+"${cheatargs[@]}"}" \
  "${frameargs[@]+"${frameargs[@]}"}" \
  -steps "$FD2_ORACLE_BUDGET" >/out/oracle.log 2>&1 &
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
