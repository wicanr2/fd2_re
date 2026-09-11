#!/usr/bin/env python3
"""fd2_worklist_issues.py plan|apply — 把 docs/data/fd2-worklist.json 同步成 GitHub Issues。

權威仍然是 JSON。Issue 是鏡像：標題、內文與標籤全部由 JSON 產生，在 GitHub 上
改內文會在下一次同步被蓋掉；討論與進度回報可以留言，留言不受影響。

每一個 issue 的內文第一行是 `<!-- fd2-worklist: <id> -->`，工具靠它把 issue 對回
條目，不靠標題（標題會改）。建立之後把編號寫回 JSON 的 `issue` 欄位，讓
91-worklist.md 與提交訊息可以引用 `#N`。

同步規則：
  - JSON 有、GitHub 沒有  → 建立
  - 兩邊都有、內容不同    → 更新標題／內文／標籤
  - JSON 有、issue 已關閉 → 重開（條目還在就代表還沒做完）
  - issue 在、JSON 已移除 → 留言並關閉（條目做完才會從 JSON 移走）
  - verify 判「可能已完成」→ 掛 `verify:可能已完成` 標籤，不自動關閉；要人回頭確認、
    從 JSON 移走，下一次同步才關

用法：
  tools/fd2_worklist_issues.py plan     只列出要做的動作，不碰 GitHub 的寫入
  tools/fd2_worklist_issues.py apply    照計畫執行，並把新 issue 的編號寫回 JSON

需要 `gh` 已登入。讀寫都走 `gh`，所以要在有網路與 gh 認證的地方執行；計畫本身是
純函式（`plan_actions`），測試不需要網路。
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
MARKER = re.compile(r"<!-- fd2-worklist: ([A-Za-z0-9_.-]+) -->")

# 標籤的顏色只為了在清單上分得開；語意在描述裡。
LABEL_COLORS = {
    MANAGED_LABEL: "5319e7",
    STALE_LABEL: "d93f0b",
    MANUAL_LABEL: "fbca04",
}
LAYER_COLOR = "0e8a16"
CATEGORY_COLOR = "1d76db"


def layer_label(layer):
    return f"分層:{layer}"


def category_label(category):
    return f"類別:{category}"


def evidence_link(evidence):
    """`path §3` → 連到 main 上的檔案，§ 之後的段落說明保留成文字。"""
    path, _, section = evidence.partition(" ")
    text = f"[`{path}`]({BLOB}{path})"
    return f"{text} {section}".rstrip()


def desired_issue(item, data):
    """一個條目在 GitHub 上應該長什麼樣子。"""
    open_, why = fd2_worklist.still_open(item)
    verify = item["verify"]
    lines = [
        f'<!-- fd2-worklist: {item["id"]} -->',
        f'由 [`docs/data/fd2-worklist.json`]({BLOB}docs/data/fd2-worklist.json) 的 '
        f'`{item["id"]}` 同步產生；要改內容請改 JSON，這裡的內文會在下一次同步被覆寫。',
        "",
    ]
    if item.get("body"):
        lines += [item["body"], ""]
    if item.get("blocked_by"):
        lines += [f'**卡在**：{item["blocked_by"]}', ""]
    if item.get("acceptance"):
        lines += [f'**怎樣算做完**：{item["acceptance"]}', ""]
    if item.get("evidence"):
        lines += [f'**證據**：{evidence_link(item["evidence"])}', ""]
    kind = verify["kind"]
    lines.append(f'**機器檢查**：`{kind}`（{data["verify_kinds"][kind]}）')
    if verify.get("pattern"):
        paths = "、".join(f"`{p}`" for p in verify.get("paths", []))
        lines.append(f'pattern `{verify["pattern"]}`，範圍 {paths}')
    if verify.get("note"):
        lines.append(verify["note"])
    lines.append("")
    state = "仍未完成" if open_ else "**可能已完成，回去確認後從 JSON 移走**"
    lines.append(f"最近一次同步時：{state}（{why}）")

    labels = {MANAGED_LABEL, layer_label(item["layer"]),
              category_label(item.get("category", "工作"))}
    if not open_:
        labels.add(STALE_LABEL)
    if kind == "manual":
        labels.add(MANUAL_LABEL)
    return {
        "id": item["id"],
        "title": item["title"],
        "body": "\n".join(lines).rstrip() + "\n",
        "labels": labels,
    }


def issue_id(issue):
    match = MARKER.search(issue.get("body") or "")
    return match.group(1) if match else None


def plan_actions(data, issues, head=""):
    """比對 JSON 與現有 issue，回傳要執行的動作清單（依序執行）。

    issues 是 `gh issue list --json number,title,body,state,labels` 的結果，
    只含帶 `worklist` 標籤的那些。
    """
    by_id = {}
    for issue in issues:
        ident = issue_id(issue)
        if ident is None:
            continue
        if ident in by_id:
            raise SystemExit(f"兩個 issue 標記成同一個條目 {ident}："
                             f"#{by_id[ident]['number']}、#{issue['number']}")
        by_id[ident] = issue

    actions = []
    wanted = set()
    for item in data["items"]:
        want = desired_issue(item, data)
        wanted.add(item["id"])
        issue = by_id.get(item["id"])
        if issue is None:
            actions.append({"op": "create", **want})
            continue
        number = issue["number"]
        if issue["state"].upper() == "CLOSED":
            actions.append({"op": "reopen", "id": item["id"], "number": number,
                            "comment": "條目仍在 `docs/data/fd2-worklist.json`，重新開啟。"})
        have_labels = {label["name"] for label in issue.get("labels", [])}
        changes = {}
        if issue["title"] != want["title"]:
            changes["title"] = want["title"]
        if (issue.get("body") or "").rstrip() != want["body"].rstrip():
            changes["body"] = want["body"]
        add = sorted(want["labels"] - have_labels)
        # 只收掉工具自己管的標籤；人手加上的其他標籤保留。
        managed = {label for label in have_labels
                   if label in (MANAGED_LABEL, STALE_LABEL, MANUAL_LABEL)
                   or label.startswith("分層:") or label.startswith("類別:")}
        remove = sorted(managed - want["labels"])
        if add:
            changes["add_labels"] = add
        if remove:
            changes["remove_labels"] = remove
        if changes:
            actions.append({"op": "update", "id": item["id"], "number": number, **changes})
        if item.get("issue") != number:
            actions.append({"op": "record", "id": item["id"], "number": number})

    for ident, issue in sorted(by_id.items(), key=lambda pair: pair[1]["number"]):
        if ident in wanted or issue["state"].upper() == "CLOSED":
            continue
        where = f"（同步時的 HEAD：`{head}`）" if head else ""
        actions.append({
            "op": "close", "id": ident, "number": issue["number"],
            "comment": f"條目 `{ident}` 已從 `docs/data/fd2-worklist.json` 移除，視為完成{where}。",
        })
    return actions


def labels_needed(data):
    """同步會用到的全部標籤與描述。"""
    labels = {
        MANAGED_LABEL: (LABEL_COLORS[MANAGED_LABEL], "由 fd2-worklist.json 同步的條目"),
        STALE_LABEL: (LABEL_COLORS[STALE_LABEL], "機器檢查的訊號消失了，要回頭確認是否已完成"),
        MANUAL_LABEL: (LABEL_COLORS[MANUAL_LABEL], "沒有機器訊號，要人判斷"),
    }
    for key, description in data["layers"].items():
        labels[layer_label(key)] = (LAYER_COLOR, description)
    for key, description in data.get("categories", {}).items():
        labels[category_label(key)] = (CATEGORY_COLOR, description)
    return labels


def describe(action):
    op = action["op"]
    if op == "create":
        return f'建立  {action["id"]}：{action["title"]}  標籤 {sorted(action["labels"])}'
    if op == "update":
        parts = [key for key in ("title", "body") if key in action]
        if action.get("add_labels"):
            parts.append(f'+{action["add_labels"]}')
        if action.get("remove_labels"):
            parts.append(f'-{action["remove_labels"]}')
        return f'更新  #{action["number"]} {action["id"]}：{"、".join(parts)}'
    if op == "record":
        return f'記錄  {action["id"]} → #{action["number"]}（寫回 JSON）'
    return f'{ {"reopen": "重開", "close": "關閉"}[op]}  #{action["number"]} {action["id"]}'


# ---- 以下是會碰 GitHub 的部分 ----

def gh(args, stdin=None):
    result = subprocess.run(["gh", *args], input=stdin, capture_output=True, text=True)
    if result.returncode != 0:
        raise SystemExit(f"gh {' '.join(args)} 失敗：{result.stderr.strip()}")
    return result.stdout


def fetch_issues():
    out = gh(["issue", "list", "--repo", REPO_SLUG, "--label", MANAGED_LABEL,
              "--state", "all", "--limit", "1000",
              "--json", "number,title,body,state,labels"])
    return json.loads(out)


def ensure_labels(data):
    existing = {label["name"]: label for label in json.loads(gh(
        ["label", "list", "--repo", REPO_SLUG, "--limit", "1000",
         "--json", "name,color,description"]))}
    for name, (color, description) in labels_needed(data).items():
        have = existing.get(name)
        if have and have["color"].lower() == color and have["description"] == description:
            continue
        gh(["label", "create", name, "--repo", REPO_SLUG, "--color", color,
            "--description", description, "--force"])
        print(f"標籤  {name}")


def execute(action):
    op, number = action["op"], action.get("number")
    if op == "create":
        args = ["issue", "create", "--repo", REPO_SLUG, "--title", action["title"],
                "--body-file", "-"]
        for label in sorted(action["labels"]):
            args += ["--label", label]
        url = gh(args, stdin=action["body"]).strip()
        return int(url.rstrip("/").rsplit("/", 1)[1])
    if op == "update":
        args = ["issue", "edit", str(number), "--repo", REPO_SLUG]
        if "title" in action:
            args += ["--title", action["title"]]
        if "body" in action:
            args += ["--body-file", "-"]
        for label in action.get("add_labels", []):
            args += ["--add-label", label]
        for label in action.get("remove_labels", []):
            args += ["--remove-label", label]
        gh(args, stdin=action.get("body"))
    elif op == "reopen":
        gh(["issue", "reopen", str(number), "--repo", REPO_SLUG, "--comment", action["comment"]])
    elif op == "close":
        gh(["issue", "close", str(number), "--repo", REPO_SLUG, "--comment", action["comment"]])
    return number


def write_issue_numbers(numbers):
    """把 id → 編號寫回 JSON，保留原本的兩格縮排。"""
    path = fd2_worklist.DATA_PATH
    data = json.loads(path.read_text(encoding="utf-8"))
    changed = False
    for item in data["items"]:
        number = numbers.get(item["id"])
        if number is not None and item.get("issue") != number:
            item["issue"] = number
            changed = True
    if changed:
        path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    return changed


def head_commit():
    result = subprocess.run(["git", "-C", str(fd2_worklist.ROOT), "rev-parse", "--short", "HEAD"],
                            capture_output=True, text=True)
    return result.stdout.strip()


def main(argv):
    if len(argv) != 2 or argv[1] not in ("plan", "apply"):
        raise SystemExit(__doc__)
    data = fd2_worklist.load()
    actions = plan_actions(data, fetch_issues(), head_commit())
    for action in actions:
        print(describe(action))
    if not actions:
        print("GitHub 與 JSON 一致，沒有要做的事")
    if argv[1] == "plan":
        return 0
    ensure_labels(data)
    numbers = {}
    for action in actions:
        number = execute(action)
        if action["op"] in ("create", "record"):
            numbers[action["id"]] = number
    if write_issue_numbers(numbers):
        print(f"已把 issue 編號寫回 {fd2_worklist.DATA_PATH.relative_to(fd2_worklist.ROOT)}；"
              "記得重跑 render 並提交")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
