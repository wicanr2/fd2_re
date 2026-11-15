#!/usr/bin/env python3
"""R5 階段 2 r5c 組（docs/goal/121）局部黃金的比對與逐字複本工具。

用法：
  rust_r5c_layers_compare.py --verbatim <remake/cmd/fd2 目錄>
      核對 rust_golden_r5c_layers_test.go 末尾的逐字複本：拿掉 `// r5c-insert` 行、把換過的名稱換回後，與原檔的
      函式本體逐行相同。
  rust_r5c_layers_compare.py --emit-copies <remake/cmd/fd2 目錄>
      從原檔重新產生逐字複本區段（印到 stdout），供更新 Go 測試檔時使用。
  rust_r5c_layers_compare.py --go <go/r5c-layers.json> --rust <rust/r5c-layers.json> [--stage0 <frames.json>]
      逐擷取點比對兩邊的比對鍵；--stage0 另把 pump 擷取裡有畫的圖層與階段 0 整張黃金同一擷取點對照（旁證）。

結束碼：比對或核對失敗為 1，參數錯誤為 2。
"""
import argparse
import json
import os
import sys

GO_TEST = "rust_golden_r5c_layers_test.go"
COPIES_MARK = "// ---- 逐字複本"
INSERT_MARK = "// r5c-insert"

# (名稱, 原檔, 原簽名, 複本簽名, [(原文字, 複本文字)], [插入規則])
# 插入規則：("after_block", 錨點子字串, 擷取點)：錨點所在行之後第一個縮排少一層的 "}" 之後插入；
#           ("after_line", 去掉縮排後的行首, 擷取點)：每個符合的行之後、同縮排插入。
COPIES = [
    ("TestChapterParityReplay", "chapter_parity_replay_test.go",
     "func TestChapterParityReplay(t *testing.T) {",
     "func (rec *r5cLayerRecorder) chapterParityReplay(t *testing.T) {",
     [("r.mark(action)", "rec.mark(r, action)"), ("r.townProbe(action)", "rec.townProbe(r, action)"),
      ("r.townSave(action)", "rec.townSave(r, action)")],
     []),
    ("ensureTown", "chapter_parity_replay_test.go",
     "func (r *parityReplay) ensureTown() {",
     "func (rec *r5cLayerRecorder) ensureTown(r *parityReplay) {",
     [("r.ensurePostbattleNode(r.townAfter)", "rec.ensurePostbattleNode(r, r.townAfter)")],
     []),
    ("ensurePostbattleNode", "chapter_parity_replay_test.go",
     "func (r *parityReplay) ensurePostbattleNode(target string) {",
     "func (rec *r5cLayerRecorder) ensurePostbattleNode(r *parityReplay, target string) {",
     [("driveStory(t, g, j, ", "rec.driveStory(t, g, j, "), ("r.settleTown()", "rec.settleTown(r)")],
     [("after_block", 't.Fatalf("Update：%v", err)', "postbattle")]),
    ("driveStory", "native_title_to_town_test.go",
     "func driveStory(t *testing.T, g *Game, j *journeyTrace, stop func() bool) {",
     "func (rec *r5cLayerRecorder) driveStory(t *testing.T, g *Game, j *journeyTrace, stop func() bool) {",
     [],
     [("after_block", 't.Fatalf("故事節點 %s 執行期錯誤', "story")]),
    ("settleTown", "chapter_parity_replay_test.go",
     "func (r *parityReplay) settleTown() {",
     "func (rec *r5cLayerRecorder) settleTown(r *parityReplay) {",
     [],
     [("after_block", 'r.t.Fatalf("Update：%v", err)', "settle")]),
    ("townProbe", "chapter_parity_replay_test.go",
     "func (r *parityReplay) townProbe(action parityAction) {",
     "func (rec *r5cLayerRecorder) townProbe(r *parityReplay, action parityAction) {",
     [("r.ensureTown()", "rec.ensureTown(r)")],
     [("after_block", "城鎮選擇無法移動", "town_move")]),
    ("townSave", "chapter_parity_replay_test.go",
     "func (r *parityReplay) townSave(action parityAction) {",
     "func (rec *r5cLayerRecorder) townSave(r *parityReplay, action parityAction) {",
     [("r.ensureTown()", "rec.ensureTown(r)")],
     [("after_block", "城鎮選擇移不到酒店", "town_move"),
      ("after_line", "g.handleNativeHotelInput(", "hotel_input")]),
]


def extract(lines, signature, where):
    """回傳 signature 那一行到第一個單獨 "}" 行（含）之間的行。"""
    starts = [i for i, line in enumerate(lines) if line == signature]
    if len(starts) != 1:
        raise SystemExit(f"{where}：找到 {len(starts)} 個 {signature!r}")
    start = starts[0]
    for end in range(start + 1, len(lines)):
        if lines[end] == "}":
            return lines[start:end + 1]
    raise SystemExit(f"{where}：{signature!r} 沒有結尾")


def indent_of(line):
    return line[:len(line) - len(line.lstrip("\t"))]


def emit(original, copy_sig, subs, rules):
    out = [copy_sig]
    body = original[1:]
    pending = []  # (縮排, 擷取點)：等待縮排少一層的 "}"
    used = set()
    for line in body:
        new = line
        for old, repl in subs:
            if old in new:
                new = new.replace(old, repl)
                used.add(old)
        out.append(new)
        for kind, anchor, point in rules:
            if kind == "after_block" and anchor in line:
                pending.append((indent_of(line)[:-1], point))
                used.add(anchor)
            elif kind == "after_line" and line.lstrip("\t").startswith(anchor):
                out.append(f'{indent_of(line)}rec.observeGap(g, "{point}") {INSERT_MARK}')
                used.add(anchor)
        if pending and line == pending[0][0] + "}":
            ind, point = pending.pop(0)
            out.append(f'{ind}rec.observeGap(g, "{point}") {INSERT_MARK}')
    missing = [x for x in [s[0] for s in subs] + [r[1] for r in rules] if x not in used]
    if missing or pending:
        raise SystemExit(f"{copy_sig}：替換或插入規則沒有用到：{missing} {pending}")
    return out


def read_lines(path):
    with open(path, encoding="utf-8") as f:
        return f.read().split("\n")


def emit_copies(gosrc):
    blocks = []
    for name, src, sig, copy_sig, subs, rules in COPIES:
        original = extract(read_lines(os.path.join(gosrc, src)), sig, src)
        blocks.append(f"// {name}：{src} 的逐字複本。")
        blocks.extend(emit(original, copy_sig, subs, rules))
        blocks.append("")
    return blocks


def verbatim(gosrc):
    test = read_lines(os.path.join(gosrc, GO_TEST))
    ok = True
    for name, src, sig, copy_sig, subs, rules in COPIES:
        original = extract(read_lines(os.path.join(gosrc, src)), sig, src)
        copy = extract(test, copy_sig, GO_TEST)
        inserted = [line for line in copy if line.endswith(INSERT_MARK)]
        restored = [original[0]]
        for line in copy[1:]:
            if line.endswith(INSERT_MARK):
                continue
            for old, repl in subs:
                line = line.replace(repl, old)
            restored.append(line)
        same = restored == original and copy == emit(original, copy_sig, subs, rules)
        ok &= same
        print(f"{name}：{'逐字相同' if same else '不同'}（原檔 {len(original)} 行、插入 {len(inserted)} 行、"
              f"換名 {len(subs)} 種）")
        if not same:
            for i, (a, b) in enumerate(zip(restored, original)):
                if a != b:
                    print(f"  第一個不同在第 {i + 1} 行：\n    複本 {a!r}\n    原檔 {b!r}")
                    break
            else:
                print(f"  行數不同：複本 {len(restored)}、原檔 {len(original)}")
    return ok


def expand(rows):
    out = []
    for row in rows:
        for k in range(row["count"]):
            out.append((row["key"], row["pump_capture"] + k if row["pump_capture"] >= 0 else -1, row["node"]))
    return out


def point_of(key):
    return key.split("|", 1)[0]


def compare(go_path, rust_path, stage0_path):
    with open(go_path, encoding="utf-8") as f:
        go = json.load(f)
    with open(rust_path, encoding="utf-8") as f:
        rust = json.load(f)
    gk, rk = expand(go["rows"]), expand(rust["rows"])
    print(f"擷取數：Go {go['captures']}（pump {go['pump_captures']}）、Rust {rust['captures']}（pump {rust['pump_captures']}）")
    print(f"序列雜湊：Go {go['sequence_sha256']}、Rust {rust['sequence_sha256']}")
    ok = go["captures"] == rust["captures"] == len(gk) == len(rk) and go["sequence_sha256"] == rust["sequence_sha256"]
    first = None
    same = 0
    points, drawn, drawn_same = {}, {}, {}
    for i, ((a, pa, node), (b, _, _)) in enumerate(zip(gk, rk)):
        p = point_of(a)
        points[p] = points.get(p, 0) + 1
        hit = a == b
        same += hit
        if "rgba=" in a:
            kind = f"{p}:{node.split('_')[0] if node else '-'}"
            drawn[kind] = drawn.get(kind, 0) + 1
            drawn_same[kind] = drawn_same.get(kind, 0) + hit
        if not hit and first is None:
            first = (i, a, b)
    print(f"逐擷取點：{same}／{min(len(gk), len(rk))} 相同")
    print(f"擷取點分布（Go）：{json.dumps(points, ensure_ascii=False, sort_keys=True)}")
    print("有畫的圖層（擷取點:節點前綴 相同／總數）：")
    for kind in sorted(drawn):
        print(f"  {kind} {drawn_same[kind]}／{drawn[kind]}")
    if first is not None:
        ok = False
        i, a, b = first
        print(f"第一個不同在第 {i} 次擷取：\n  Go   {a}\n  Rust {b}")
    print(f"分支：Go {json.dumps(go['branches'], sort_keys=True)}、Rust {json.dumps(rust['branches'], sort_keys=True)}")
    print(f"統計（Go）：{json.dumps(go['stats'], ensure_ascii=False, sort_keys=True)}")
    print(f"統計（Rust）：{json.dumps(rust['stats'], ensure_ascii=False, sort_keys=True)}")
    if stage0_path:
        with open(stage0_path, encoding="utf-8") as f:
            stage0 = json.load(f)
        golden = []
        for row in stage0["rows"]:
            golden.extend([row["rgba_sha256"]] * row["count"])
        total = equal = 0
        by_node = {}
        for key, pump, node in gk:
            if pump < 0 or "rgba=" not in key:
                continue
            sha = key.split("rgba=", 1)[1]
            total += 1
            hit = pump < len(golden) and golden[pump] == sha
            equal += hit
            prefix = node.split("_")[0] if node else "-"
            t, e = by_node.get(prefix, (0, 0))
            by_node[prefix] = (t + 1, e + hit)
        detail = "、".join(f"{k} {e}／{t}" for k, (t, e) in sorted(by_node.items()))
        print(f"旁證：pump 擷取裡有畫的圖層 {total} 張，{equal} 張與階段 0 整張黃金同一擷取點相同（{detail}）")
    print("結果：" + ("相同" if ok else "不同"))
    return ok


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--verbatim")
    ap.add_argument("--emit-copies")
    ap.add_argument("--go")
    ap.add_argument("--rust")
    ap.add_argument("--stage0")
    args = ap.parse_args()
    if args.verbatim:
        sys.exit(0 if verbatim(args.verbatim) else 1)
    if args.emit_copies:
        print("\n".join(emit_copies(args.emit_copies)), end="")
        return
    if not (args.go and args.rust):
        ap.print_usage()
        sys.exit(2)
    sys.exit(0 if compare(args.go, args.rust, args.stage0) else 1)


if __name__ == "__main__":
    main()
