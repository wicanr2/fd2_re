#!/usr/bin/env python3
"""fd2_lessons.py brief|list|check — 踩過的坑，由工具強制每次讀到。

教訓寫在 docs/data/fd2-lessons.json，`brief` 由 .claude/settings.json 的
SessionStart hook 在每個工作階段開始時印出來，不靠任何人記得去讀——「記得去讀」
正是這些坑一開始被踩到的原因。

每一條的 `rule` 是現在式、可重用的規則，不是事件敘述：事件會過去，規則不會。
`why` 才放當時發生了什麼。有些條目另外掛 `guard`，`check` 會驗那個訊號還在不在
（例如工具鏈文件裡對應的那一節有沒有被刪掉）。

用法：
  tools/fd2_lessons.py brief    一行一條，給 hook 用
  tools/fd2_lessons.py list     完整內容（含 why 與出處）
  tools/fd2_lessons.py check    驗有 guard 的條目，訊號消失就 exit 1
"""

import json
import os
import re
import sys
from pathlib import Path

ROOT = Path(os.environ.get("FD2_LESSONS_ROOT") or Path(__file__).resolve().parent.parent)
DATA_PATH = Path(os.environ.get("FD2_LESSONS_DATA") or ROOT / "docs/data/fd2-lessons.json")
SCANNED_SUFFIXES = {".go", ".py", ".sh", ".md", ".json", ".yml", ".yaml"}


def load():
    data = json.loads(DATA_PATH.read_text(encoding="utf-8"))
    seen = set()
    for lesson in data["lessons"]:
        for field in ("id", "rule", "why"):
            if not lesson.get(field):
                raise SystemExit(f'教訓缺少 {field}：{lesson.get("id", lesson)}')
        if lesson["id"] in seen:
            raise SystemExit(f'教訓 id 重複：{lesson["id"]}')
        seen.add(lesson["id"])
    return data


def scannable(path):
    if not path.is_file() or path.suffix not in SCANNED_SUFFIXES:
        return False
    return "_test." not in path.name and not path.name.startswith("test_")


def first_hit(guard):
    expression = re.compile(guard["pattern"])
    for target in guard["paths"]:
        candidate = ROOT / target
        files = sorted(candidate.rglob("*")) if candidate.is_dir() else [candidate]
        for path in files:
            if scannable(path) and expression.search(
                    path.read_text(encoding="utf-8", errors="ignore")):
                return str(path.relative_to(ROOT))
    return None


def guard_holds(guard):
    """回傳（訊號還在嗎, 依據）。"""
    where = first_hit(guard)
    if guard["kind"] == "present":
        return bool(where), f"還在 {where}" if where else f'找不到 {guard["pattern"]}'
    if guard["kind"] == "absent":
        return not where, f"出現在 {where}" if where else "還沒出現"
    raise SystemExit(f'未知的 guard 種類：{guard["kind"]}')


def brief(data):
    print(f"FD2 踩過的坑（{len(data['lessons'])} 條，出處 docs/data/fd2-lessons.json，"
          f"詳情跑 tools/fd2_lessons.py list）")
    for lesson in data["lessons"]:
        print(f"  [{lesson.get('area', '通則')}] {lesson['rule']}")
    return 0


def full(data):
    for lesson in data["lessons"]:
        print(f"## {lesson['id']}（{lesson.get('area', '通則')}，{lesson.get('date', '')}）")
        print(f"規則：{lesson['rule']}")
        print(f"起因：{lesson['why']}")
        if lesson.get("evidence"):
            print(f"出處：{lesson['evidence']}")
        print()
    return 0


def check(data):
    guarded = [l for l in data["lessons"] if l.get("guard")]
    broken = []
    for lesson in guarded:
        holds, why = guard_holds(lesson["guard"])
        state = "訊號還在" if holds else "**訊號不見了**"
        print(f'{lesson["id"]:<34} {state:<16} {why}')
        if not holds:
            broken.append(lesson["id"])
    print()
    print(f"共 {len(data['lessons'])} 條教訓，其中 {len(guarded)} 條掛了 guard")
    if broken:
        print("下列教訓綁的訊號消失了，回去確認那一節是不是被刪掉或改寫："
              + "、".join(broken))
        return 1
    return 0


def main(argv):
    if len(argv) != 2 or argv[1] not in ("brief", "list", "check"):
        raise SystemExit(__doc__)
    data = load()
    return {"brief": brief, "list": full, "check": check}[argv[1]](data)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
