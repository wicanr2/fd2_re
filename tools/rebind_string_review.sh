#!/usr/bin/env bash
# rebind_string_review.sh 在 Go 原始碼行號漂移之後重新綁定
# docs/data/fd2-string-review.json。
#
# 為什麼需要它：review 的 string_id 是「檔案＋行號＋欄位」座標，改任何一支被
# 盤點的 Go 檔（連加一行註解都算）都會讓後面的條目整批位移，
# TestReviewedGoCandidatesMatchCurrentInventory 就會失敗。失敗訊息長得像功能
# 壞掉（review binds sha=… want sha=…），實際上處置內容一項都沒變。
#
# **這一步要在最後一次改 Go 檔之後才做。** 先重綁再改一行註解，綁定立刻又失效，
# 而且下一次還會以為自己已經處理過了。
#
# 用法：tools/rebind_string_review.sh [工作目錄]
#   工作目錄預設是 mktemp -d；裡面會留下新舊兩份盤點供比對。
#
# 舊盤點預設從 git 取——review 上次被**提交**時的那一版原始碼。同一輪裡第二次
# 重綁時這個預設是錯的（工作區的 review 已經是上一次遷移的結果，git 裡卻還是
# 更早的版本），這時用 FD2_OLD_INVENTORY 指向上一次留下的 inv-new.json：
#
#   FD2_OLD_INVENTORY=$work/inv-new.json tools/rebind_string_review.sh $work2
set -euo pipefail

repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
work=${1:-$(mktemp -d -t fd2-rebind-XXXXXX)}
mkdir -p "$work/gocache"
image=${FD2_GO_TEST_IMAGE:-fd2-go-test-local:latest}
review=$repo_root/docs/data/fd2-string-review.json
old_inventory=${FD2_OLD_INVENTORY:-}

inventory() { # <掛載來源> <輸出檔名>
    docker run --rm --network none --memory 4g --cpus 2 --pids-limit 256 \
        --log-opt max-size=10m --log-opt max-file=3 -u "$(id -u):$(id -g)" \
        -e HOME=/tmp/home -e GOCACHE=/gocache -e GOFLAGS=-mod=mod \
        -v "$1:/tree:ro" -v "$work/gocache:/gocache" -v "$work:/out" \
        -w /tree/remake/cmd/fd2-string-inventory "$image" \
        go run . -repo ../../.. -output "/out/$2"
}

# repo 路徑寫法不影響雜湊，但這裡仍與測試用同一個相對路徑，少一個變數。
if [ -n "$old_inventory" ]; then
    cp "$old_inventory" "$work/inv-old.json"
    echo "舊盤點取自 $old_inventory"
else
    old_commit=$(git -C "$repo_root" log -1 --format=%H -- docs/data/fd2-string-review.json)
    old_tree=$work/oldtree
    rm -rf "$old_tree"
    git -C "$repo_root" worktree add --detach "$old_tree" "$old_commit" >/dev/null
    trap 'git -C "$repo_root" worktree remove --force "$old_tree" >/dev/null 2>&1 || true' EXIT
    echo "舊盤點取自 ${old_commit:0:8}（review 上次提交時的樹）"
    inventory "$old_tree" inv-old.json
fi
inventory "$repo_root" inv-new.json

python3 "$repo_root/tools/migrate_string_review.py" \
    --old-inventory "$work/inv-old.json" \
    --new-inventory "$work/inv-new.json" \
    --review "$review" --output "$work/review-new.json"

# 遷移只搬座標，不該動處置的內容。搬完的 id 集合必須正好等於目前的候選集合，
# 否則就是有條目被搬丟或搬到不存在的位置——那比行號漂移嚴重得多。
python3 - "$work/inv-new.json" "$work/review-new.json" "$review" <<'PY'
import json
import sys

inventory, migrated, original = (json.load(open(p, encoding="utf-8")) for p in sys.argv[1:4])
role = migrated["reviewed_role"]
candidates = {e["string_id"] for e in inventory["entries"] if e["role"] == role}
ids = {i for g in migrated["dispositions"].values() for i in g["string_ids"]}
if ids != candidates:
    raise SystemExit(
        f"遷移後的 id 集合與候選不符：多出 {sorted(ids - candidates)[:5]}、"
        f"少了 {sorted(candidates - ids)[:5]}")
for name, group in migrated["dispositions"].items():
    before = len(original["dispositions"][name]["string_ids"])
    if len(group["string_ids"]) != before:
        raise SystemExit(f"{name} 的條目數由 {before} 變成 {len(group['string_ids'])}")
print(f"遷移後 {len(ids)} 個 id 與目前候選一致，各處置的條目數不變")
PY

cp "$work/review-new.json" "$review"
echo "已重綁 $review（工作目錄 $work）"
echo "接著跑：tools/remake_go_test.sh <素材根> <log> ./cmd/fd2-string-inventory"
