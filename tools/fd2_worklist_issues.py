#!/usr/bin/env python3
"""fd2_worklist_issues.py — 未完成項的權威是 GitHub Issues（標籤 worklist）。

每一條未完成項是一個開著的 issue：

- 標題就是條目標題；標籤 `分層:<layer>`、`類別:<category>` 決定分層與類別。
- 內文上半是說明文字，下半是一個 ```fd2-worklist 程式碼區塊，放 id、怎樣算做完、
  證據與機器檢查（verify）。這個區塊是給工具讀的，改的時候保持 JSON 格式。
- issue 開著代表未完成；做完就關掉（留言寫是哪個提交）。

docs/data/fd2-worklist.json 是 `pull` 拉下來的快照，不要手改：它讓
`tools/fd2_worklist.py verify|render` 能在沒有網路的容器裡跑，也讓 git diff 看得到
條目何時被改。

用法：
  tools/fd2_worklist_issues.py pull                 拉下開著的條目，重寫快照
  tools/fd2_worklist_issues.py new <spec.json>      依規格開一個新條目
  tools/fd2_worklist_issues.py close <id> <留言>    關掉一個條目
  tools/fd2_worklist_issues.py labels [--apply]     依快照的 verify 結果更新狀態標籤
  tools/fd2_worklist_issues.py migrate [--apply]    把舊的鏡像內文改寫成新格式（一次性）

需要 `gh` 已登入，要在有網路與 gh 認證的地方執行（主機）。解析與格式化是純函式，
測試不需要網路。
"""

import json
import re
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import fd2_worklist  # noqa: E402

REPO_SLUG = "wicanr2/fd2_re"
BLOB = f"https://github.com/{REPO_SLUG}/blob/main/"
MANAGED_LABEL = "worklist"
STALE_LABEL = "verify:可能已完成"
MANUAL_LABEL = "verify:要人判"
BLOCK = re.compile(r"```fd2-worklist\s*\n(.*?)\n```", re.S)
OLD_MARKER = re.compile(r"<!-- fd2-worklist: ([A-Za-z0-9_.-]+) -->")
BLOCK_FIELDS = ("id", "acceptance", "evidence", "blocked_by", "verify")
BLOCK_NOTE = "_下面這個區塊給 `tools/fd2_worklist_issues.py` 讀，改的時候保持 JSON 格式。_"

LABEL_COLORS = {MANAGED_LABEL: "5319e7", STALE_LABEL: "d93f0b", MANUAL_LABEL: "fbca04"}
LAYER_COLOR = "0e8a16"
CATEGORY_COLOR = "1d76db"


def layer_label(layer):
    return f"分層:{layer}"


def category_label(category):
    return f"類別:{category}"


# ---- 內文格式 ----

def format_body(item):
    """條目 → issue 內文：說明文字 + fd2-worklist 區塊。"""
    block = {k: item[k] for k in BLOCK_FIELDS if item.get(k) is not None}
    parts = []
    if item.get("body"):
        parts.append(item["body"].strip())
    parts.append(BLOCK_NOTE)
    parts.append("```fd2-worklist\n" + json.dumps(block, ensure_ascii=False, indent=2) + "\n```")
    return "\n\n".join(parts) + "\n"


def parse_issue(issue, data):
    """issue → 條目。缺區塊、缺標籤或區塊不是 JSON 都回錯誤字串。"""
    number = issue["number"]
    text = issue.get("body") or ""
    match = BLOCK.search(text)
    if not match:
        return None, f"#{number} 沒有 fd2-worklist 區塊"
    try:
        block = json.loads(match.group(1))
    except json.JSONDecodeError as error:
        return None, f"#{number} 的 fd2-worklist 區塊不是 JSON：{error}"
    labels = {label["name"] for label in issue.get("labels", [])}
    layers = [name[len("分層:"):] for name in labels if name.startswith("分層:")]
    categories = [name[len("類別:"):] for name in labels if name.startswith("類別:")]
    if len(layers) != 1 or len(categories) != 1:
        return None, f"#{number} 要恰好一個分層標籤與一個類別標籤（現在 {sorted(labels)}）"
    body = text[:match.start()].replace(BLOCK_NOTE, "").strip()
    item = {"id": block.get("id"), "layer": layers[0], "category": categories[0],
            "title": issue["title"]}
    if body:
        item["body"] = body
    for key in ("acceptance", "evidence", "blocked_by"):
        if block.get(key):
            item[key] = block[key]
    item["verify"] = block.get("verify")
    item["issue"] = number
    problem = validate_item(item, data)
    if problem:
        return None, f"#{number}：{problem}"
    return item, None


def validate_item(item, data):
    if not isinstance(item.get("id"), str) or not re.fullmatch(r"[A-Za-z0-9_.-]+", item["id"]):
        return "id 缺少或含不允許的字元"
    if item["layer"] not in data["layers"]:
        return f"分層 {item['layer']} 沒有定義"
    if item["category"] not in data.get("categories", {}):
        return f"類別 {item['category']} 沒有定義"
    verify = item.get("verify")
    if not isinstance(verify, dict) or verify.get("kind") not in data["verify_kinds"]:
        return "verify 缺少或種類沒有定義"
    return None


def build_snapshot(data, issues):
    """開著的 worklist issue → 快照。任何一個 issue 讀不懂就整批失敗。"""
    items, problems, seen = [], [], {}
    for issue in sorted(issues, key=lambda i: i["number"]):
        if issue.get("state", "OPEN").upper() != "OPEN":
            continue
        item, problem = parse_issue(issue, data)
        if problem:
            problems.append(problem)
            continue
        if item["id"] in seen:
            problems.append(f"#{seen[item['id']]} 與 #{item['issue']} 用了同一個 id {item['id']}")
            continue
        seen[item["id"]] = item["issue"]
        items.append(item)
    if problems:
        raise SystemExit("有 issue 讀不懂，快照沒有寫：\n" + "\n".join(problems))
    snapshot = {k: v for k, v in data.items() if k != "items"}
    snapshot["items"] = items
    return snapshot


def labels_for(item, open_, manual):
    labels = {MANAGED_LABEL, layer_label(item["layer"]), category_label(item["category"])}
    if not open_:
        labels.add(STALE_LABEL)
    if manual:
        labels.add(MANUAL_LABEL)
    return labels


def labels_needed(data):
    labels = {
        MANAGED_LABEL: (LABEL_COLORS[MANAGED_LABEL], "fd2 未完成項（權威）"),
        STALE_LABEL: (LABEL_COLORS[STALE_LABEL], "機器檢查的訊號消失了，要回頭確認是否已完成"),
        MANUAL_LABEL: (LABEL_COLORS[MANUAL_LABEL], "沒有機器訊號，要人判斷"),
    }
    for key, description in data["layers"].items():
        labels[layer_label(key)] = (LAYER_COLOR, description)
    for key, description in data.get("categories", {}).items():
        labels[category_label(key)] = (CATEGORY_COLOR, description)
    return labels


def plan_label_updates(data, issues):
    """依快照的 verify 結果，算出每個 issue 要加／拿掉的狀態標籤。"""
    by_id = {}
    for issue in issues:
        item, problem = parse_issue(issue, data)
        if item:
            by_id[item["id"]] = (issue, item)
    updates = []
    for item in data["items"]:
        if item["id"] not in by_id:
            continue
        issue, _ = by_id[item["id"]]
        open_, _ = fd2_worklist.still_open(item)
        want = labels_for(item, open_, item["verify"]["kind"] == "manual")
        have = {label["name"] for label in issue.get("labels", [])}
        managed = {name for name in have if name in (STALE_LABEL, MANUAL_LABEL)}
        add = sorted(({STALE_LABEL, MANUAL_LABEL} & want) - have)
        remove = sorted(managed - want)
        if add or remove:
            updates.append({"number": issue["number"], "id": item["id"], "add": add, "remove": remove})
    return updates


def plan_migration(data, issues):
    """舊鏡像（只有 <!-- fd2-worklist: id --> 標記）→ 新格式內文。內容取自目前的快照。"""
    by_id = {item["id"]: item for item in data["items"]}
    updates = []
    for issue in issues:
        text = issue.get("body") or ""
        if BLOCK.search(text):
            continue
        match = OLD_MARKER.search(text)
        if not match:
            continue
        item = by_id.get(match.group(1))
        if item is None:
            continue
        updates.append({"number": issue["number"], "id": item["id"], "body": format_body(item)})
    return updates


# ---- 會碰 GitHub 的部分 ----

def gh(args, stdin=None):
    result = subprocess.run(["gh", *args], input=stdin, capture_output=True, text=True)
    if result.returncode != 0:
        raise SystemExit(f"gh {' '.join(args)} 失敗：{result.stderr.strip()}")
    return result.stdout


def fetch_issues(state="open"):
    out = gh(["issue", "list", "--repo", REPO_SLUG, "--label", MANAGED_LABEL, "--state", state,
              "--limit", "1000", "--json", "number,title,body,state,labels"])
    return json.loads(out)


def ensure_labels(data):
    existing = {label["name"]: label for label in json.loads(gh(
        ["label", "list", "--repo", REPO_SLUG, "--limit", "1000", "--json", "name,color,description"]))}
    for name, (color, description) in labels_needed(data).items():
        have = existing.get(name)
        if have and have["color"].lower() == color and have["description"] == description:
            continue
        gh(["label", "create", name, "--repo", REPO_SLUG, "--color", color,
            "--description", description, "--force"])


def load_schema():
    return json.loads(fd2_worklist.DATA_PATH.read_text(encoding="utf-8"))


def write_snapshot(snapshot):
    fd2_worklist.DATA_PATH.write_text(json.dumps(snapshot, ensure_ascii=False, indent=2) + "\n",
                                      encoding="utf-8")


def cmd_pull():
    snapshot = build_snapshot(load_schema(), fetch_issues("open"))
    write_snapshot(snapshot)
    print(f"快照已更新：{len(snapshot['items'])} 條開著的條目；記得重跑 fd2_worklist.py render 並提交")


def cmd_new(spec_path):
    data = load_schema()
    spec = json.loads(Path(spec_path).read_text(encoding="utf-8"))
    problem = validate_item(spec, data)
    if problem or not spec.get("title"):
        raise SystemExit(f"規格不完整：{problem or '缺標題'}")
    if any(spec["id"] == item["id"] for item in data["items"]):
        raise SystemExit(f"id {spec['id']} 已經有條目（先 pull 確認）")
    ensure_labels(data)
    args = ["issue", "create", "--repo", REPO_SLUG, "--title", spec["title"], "--body-file", "-"]
    manual = spec["verify"]["kind"] == "manual"
    for label in sorted(labels_for(spec, True, manual)):
        args += ["--label", label]
    print(gh(args, stdin=format_body(spec)).strip())


def cmd_close(ident, comment):
    data = load_schema()
    for issue in fetch_issues("open"):
        item, _ = parse_issue(issue, data)
        if item and item["id"] == ident:
            gh(["issue", "close", str(issue["number"]), "--repo", REPO_SLUG, "--comment", comment])
            print(f"已關閉 #{issue['number']} {ident}")
            return
    raise SystemExit(f"找不到開著的條目 {ident}")


def cmd_labels(apply):
    data = load_schema()
    updates = plan_label_updates(data, fetch_issues("open"))
    for update in updates:
        print(f"#{update['number']} {update['id']}：+{update['add']} -{update['remove']}")
    if not updates:
        print("狀態標籤與快照一致")
    if apply:
        ensure_labels(data)
        for update in updates:
            args = ["issue", "edit", str(update["number"]), "--repo", REPO_SLUG]
            for label in update["add"]:
                args += ["--add-label", label]
            for label in update["remove"]:
                args += ["--remove-label", label]
            gh(args)


def cmd_migrate(apply):
    data = load_schema()
    updates = plan_migration(data, fetch_issues("all"))
    for update in updates:
        print(f"改寫 #{update['number']} {update['id']}")
    if not updates:
        print("沒有舊格式的內文")
    if apply:
        for update in updates:
            gh(["issue", "edit", str(update["number"]), "--repo", REPO_SLUG, "--body-file", "-"],
               stdin=update["body"])


def main(argv):
    if len(argv) >= 2 and argv[1] == "pull" and len(argv) == 2:
        cmd_pull()
    elif len(argv) == 3 and argv[1] == "new":
        cmd_new(argv[2])
    elif len(argv) == 4 and argv[1] == "close":
        cmd_close(argv[2], argv[3])
    elif len(argv) in (2, 3) and argv[1] == "labels":
        cmd_labels(argv[2:] == ["--apply"])
    elif len(argv) in (2, 3) and argv[1] == "migrate":
        cmd_migrate(argv[2:] == ["--apply"])
    else:
        raise SystemExit(__doc__)
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
