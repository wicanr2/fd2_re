#!/usr/bin/env python3
"""fd2_worklist.py verify|render — 讀未完成項的快照 docs/data/fd2-worklist.json。

markdown 的 `- [ ]` 清單會長出過期斷言：東西做好了而沒有人回頭改那一條，然後
有人照它去重做一遍、或拿它當「還剩多少」的依據。這支工具讓每一條未完成項都
掛一個會自己開口的檢查。

`verify` 跑起來為真＝這一條仍然未完成；為假就是東西做好了而條目沒改。

用法：
  tools/fd2_worklist.py verify            逐條檢查，發現可能已完成就以 exit 1 收場
  tools/fd2_worklist.py render            重寫 91-worklist.md 的產生區塊

未完成項的權威是 GitHub Issues（標籤 worklist）。這份 JSON 是
tools/fd2_worklist_issues.py pull 拉下來的快照，讓這裡的檢查能離線跑、git diff 看得到
條目何時被改；不要手改它，改 issue。

檢查只看產品程式碼：`*_test.*` 與 `test_*` 一律跳過。測試本來就會提到還沒接上
的東西（為了釘住將來的行為，或為了測資料結構本身），把它們算進來，`absent`
會因為測試裡有一行呼叫就判成「已經做了」，真缺口就這樣被蓋掉。
"""

import json
import os
import re
import sys
from pathlib import Path

# 根目錄可由環境變數覆寫，讓測試能在暫存樹上做正反對照：先確認訊號在時報「仍未
# 完成」，再把訊號拿掉、確認它真的開口。只驗「它印出仍未完成」證明不了機制有在看。
ROOT = Path(os.environ.get("FD2_WORKLIST_ROOT") or Path(__file__).resolve().parent.parent)
DATA_PATH = Path(os.environ.get("FD2_WORKLIST_DATA") or ROOT / "docs/data/fd2-worklist.json")
RENDER_PATH = ROOT / "docs/knowledge-base/91-worklist.md"
ISSUE_URL = "https://github.com/wicanr2/fd2_re/issues/"
BEGIN = "<!-- BEGIN fd2_worklist.py render；不要手改這一段 -->"
END = "<!-- END fd2_worklist.py render -->"
SCANNED_SUFFIXES = {".go", ".py", ".sh", ".md", ".json", ".jsonl", ".yml", ".yaml", ".html"}


def scannable(path):
    """產品程式碼才算數：測試檔會提到還沒接上的東西，掃進來會蓋掉真缺口。"""
    if not path.is_file() or path.suffix not in SCANNED_SUFFIXES:
        return False
    if "_test." in path.name or path.name.startswith("test_"):
        return False
    return True


def first_hit(verify):
    expression = re.compile(verify["pattern"])
    for target in verify["paths"]:
        candidate = ROOT / target
        files = sorted(candidate.rglob("*")) if candidate.is_dir() else [candidate]
        for path in files:
            if not scannable(path):
                continue
            if expression.search(path.read_text(encoding="utf-8", errors="ignore")):
                return str(path.relative_to(ROOT))
    return None


def still_open(item):
    """回傳（這一條是否仍然未完成, 依據）。"""
    verify = item["verify"]
    kind = verify["kind"]
    if kind == "manual":
        return True, "要人判"
    if kind == "json_len":
        document = json.loads((ROOT / verify["path"]).read_text(encoding="utf-8"))
        field = document[verify["field"]]
        return len(field) <= verify["max"], f'{verify["path"]} 的 {verify["field"]} 有 {len(field)} 項'
    where = first_hit(verify)
    if kind == "present":
        if where:
            return True, f"自承還在 {where}"
        return False, f'找不到 {verify["pattern"]}'
    if kind == "absent":
        if where:
            return False, f"已經出現在 {where}"
        return True, "還沒出現"
    raise SystemExit(f'未知的 verify 種類：{kind}（條目 {item["id"]}）')


def load():
    data = json.loads(DATA_PATH.read_text(encoding="utf-8"))
    seen = set()
    for item in data["items"]:
        for field in ("id", "layer", "title", "verify"):
            if field not in item:
                raise SystemExit(f'條目缺少 {field}：{item.get("id", item)}')
        if item["id"] in seen:
            raise SystemExit(f'條目 id 重複：{item["id"]}')
        seen.add(item["id"])
        if item["layer"] not in data["layers"]:
            raise SystemExit(f'條目 {item["id"]} 的分層 {item["layer"]} 沒有定義')
        if item["verify"]["kind"] not in data["verify_kinds"]:
            raise SystemExit(f'條目 {item["id"]} 的 verify 種類沒有定義')
        if "category" in item and item["category"] not in data.get("categories", {}):
            raise SystemExit(f'條目 {item["id"]} 的類別 {item["category"]} 沒有定義')
        if "issue" in item and (not isinstance(item["issue"], int) or item["issue"] <= 0):
            raise SystemExit(f'條目 {item["id"]} 的 issue 編號不是正整數')
    return data


def verify(argv):
    data = load()
    stale = []
    manual = []
    for item in data["items"]:
        open_, why = still_open(item)
        if not open_:
            stale.append(item["id"])
        if item["verify"]["kind"] == "manual":
            manual.append(item["id"])
        state = "仍未完成" if open_ else "**可能已完成**"
        print(f'{item["id"]:<34} {state:<16} {why}')
    print()
    print(f"共 {len(data['items'])} 條；要人判 {len(manual)} 條；可能已完成 {len(stale)} 條")
    if manual:
        print("要人判（沉默不等於通過）：" + "、".join(manual))
    if stale:
        print("下列條目的訊號已經消失，回去確認是不是做完了：" + "、".join(stale))
        return 1
    return 0


def render_block(data):
    lines = [BEGIN, ""]
    lines.append(f"共 {len(data['items'])} 條未完成項。權威是 GitHub Issues（標籤 `worklist`），"
                 "[`docs/data/fd2-worklist.json`](../data/fd2-worklist.json) 是拉下來的快照，"
                 "本節由 [`tools/fd2_worklist.py`](../../tools/fd2_worklist.py) 產生。")
    lines.append("")
    lines.append("`要人判` 的條目沒有機器訊號，每一輪都會被列出來——沉默不等於通過。")
    lines.append("新增、修改、關閉條目都在 GitHub 上做（[`tools/fd2_worklist_issues.py`]"
                 "(../../tools/fd2_worklist_issues.py) 的 `new`／`close`），之後 `pull` 更新快照。")
    lines.append("")
    for key, description in data["layers"].items():
        items = [i for i in data["items"] if i["layer"] == key]
        if not items:
            continue
        lines.append(f"## {key} — {description}")
        lines.append("")
        for item in items:
            open_, why = still_open(item)
            state = "仍未完成" if open_ else "**可能已完成，回去確認**"
            lines.append(f'### {item["title"]}')
            lines.append("")
            meta = [f'`{item["id"]}`']
            if item.get("category"):
                meta.append(item["category"])
            if item.get("issue"):
                meta.append(f'[#{item["issue"]}]({ISSUE_URL}{item["issue"]})')
            lines.append(" · ".join(meta + [state, why]))
            lines.append("")
            if item.get("body"):
                lines.append(item["body"])
                lines.append("")
            if item.get("blocked_by"):
                lines.append(f'卡在：{item["blocked_by"]}')
                lines.append("")
            if item.get("acceptance"):
                lines.append(f'怎樣算做完：{item["acceptance"]}')
                lines.append("")
            if item.get("evidence"):
                lines.append(f'證據：`{item["evidence"]}`')
                lines.append("")
    lines.append(END)
    return "\n".join(lines)


def render(argv):
    data = load()
    text = RENDER_PATH.read_text(encoding="utf-8")
    if BEGIN not in text or END not in text:
        raise SystemExit(f"{RENDER_PATH} 找不到產生區塊的標記")
    head, rest = text.split(BEGIN, 1)
    _, tail = rest.split(END, 1)
    RENDER_PATH.write_text(head + render_block(data) + tail, encoding="utf-8")
    print(f"已重寫 {RENDER_PATH.relative_to(ROOT)} 的產生區塊（{len(data['items'])} 條）")
    return 0


def main(argv):
    if len(argv) != 2 or argv[1] not in ("verify", "render"):
        raise SystemExit(__doc__)
    return verify(argv) if argv[1] == "verify" else render(argv)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
