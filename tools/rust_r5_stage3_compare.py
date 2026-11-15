#!/usr/bin/env python3
"""R5 階段 3（docs/goal/121，代號 r5s）：整張畫格比對，逐張核對 Go 與 Rust 的差異是否全部落在同一擷取點 TTF 繪字的墨跡外框內。

用法：
  rust_r5_stage3_compare.py chapter --go <階段0 chapter/frames.json> --rust <rust/frames.json>
      --ttf-go <go/ttf-layers.json> --ttf-rust <rust/ttf-layers.json> --out <目錄> [--jobs N]
      Go 像素取階段 0 的 PNG（frames.json 旁的 frames/*.png）；Rust 像素取 rust/frames/<sha>.rgba
      （tools/rust_r5_chapter_frames.sh 以 FD2_R5_DUMP_DIFF=1 寫出）；TTF 繪字鍵、墨跡外框與圖層裁切取 r5f 的
      tools/rust_r5f_ttf_layers.sh 輸出（同一批 pump 擷取點）。
  rust_r5_stage3_compare.py facility --go <go/r5s-frames.json> --rust <rust/r5s-frames.json> [--stage0 <frames.json>]
      --out <目錄> [--jobs N]
      城鎮與設施的整張畫格（r5c 同一批 pump 與 gap 擷取點，每次整個 Draw）。Go 像素取 go/frames/<sha>.rgba.z（zlib），
      沒有時以 --stage0 同一 pump 擷取點的 PNG 補；Rust 像素取 rust/frames/<sha>.rgba。TTF 鍵與墨跡外框在 rows[].key。

規則（兩種模式相同；不遮蔽任何像素）：
- 兩邊依 rows 的 capture／count 展開成每次擷取的 RGBA SHA-256，逐次比；雜湊相同即 640×400×4 個位元組相同。
- 雜湊不同的擷取：讀兩邊的像素（先核對讀到的位元組雜湊等於 rows 記的雜湊），列出不同的像素。該擷取點兩邊的 TTF 鍵
  必須相同且有繪字，兩邊的墨跡外框（TTF 圖層非透明像素的外框）也要相同；每個不同的像素都要落在 Go 的墨跡外框內，另記
  落在 BoundString 外框外的個數。
- chapter 模式另以 r5f 的圖層裁切核對「不同的像素處兩邊 TTF 圖層也不同」，並記可由「同一底色上分別疊兩邊 TTF 圖層」
  逐通道重現的像素數（source-over：min(255, s + mul255(d*(255-a)))；TTF 之後還有別層蓋上時不成立，只是旁證）。
- 任何一張的差異落在墨跡外框外、TTF 鍵不同、不同像素處 TTF 圖層相同或缺像素，結束碼 1。
- 同一對（Go 雜湊、Rust 雜湊）的像素差只算一次（位元組相同，差異相同）；外框核對仍逐張做。

輸出：<目錄>/<模式>-captures.jsonl（每張不同的擷取一行）、<模式>-summary.json、<模式>-summary.txt。
"""
import argparse
import base64
import hashlib
import json
import multiprocessing
import os
import re
import struct
import sys
import zlib

W, H = 640, 400
ROW = W * 4


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def load_json(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def expand(rows, field="rgba_sha256"):
    """rows（capture／count）展開成每次擷取的 (值, row)。"""
    out = []
    for row in rows:
        if row["capture"] != len(out):
            raise SystemExit(f"rows 不連續：capture={row['capture']}，預期 {len(out)}")
        out.extend([(row[field], row)] * row["count"])
    return out


# ---- PNG（Go image/png 寫的 8 位元 RGB／RGBA，非交錯）----
def png_rgba(path):
    with open(path, "rb") as f:
        data = f.read()
    if data[:8] != b"\x89PNG\r\n\x1a\n":
        raise SystemExit(f"{path} 不是 PNG")
    pos, idat, hdr = 8, [], None
    while pos < len(data):
        n, kind = struct.unpack(">I4s", data[pos:pos + 8])
        body = data[pos + 8:pos + 8 + n]
        if kind == b"IHDR":
            hdr = struct.unpack(">IIBBBBB", body)
        elif kind == b"IDAT":
            idat.append(body)
        pos += 12 + n
    w, h, depth, ctype, _, _, interlace = hdr
    if (w, h) != (W, H) or depth != 8 or ctype not in (2, 6) or interlace:
        raise SystemExit(f"{path}：不支援的 PNG 格式 {hdr}")
    bpp = 3 if ctype == 2 else 4
    stride = w * bpp
    raw = zlib.decompress(b"".join(idat))
    prev = bytearray(stride)
    out = bytearray(W * H * 4)
    for y in range(h):
        ft = raw[y * (stride + 1)]
        line = bytearray(raw[y * (stride + 1) + 1:(y + 1) * (stride + 1)])
        if ft == 1:
            for i in range(bpp, stride):
                line[i] = (line[i] + line[i - bpp]) & 0xff
        elif ft == 2:
            for i in range(stride):
                line[i] = (line[i] + prev[i]) & 0xff
        elif ft == 3:
            for i in range(stride):
                left = line[i - bpp] if i >= bpp else 0
                line[i] = (line[i] + ((left + prev[i]) >> 1)) & 0xff
        elif ft == 4:
            for i in range(stride):
                a = line[i - bpp] if i >= bpp else 0
                b = prev[i]
                c = prev[i - bpp] if i >= bpp else 0
                p = a + b - c
                pa, pb, pc = abs(p - a), abs(p - b), abs(p - c)
                pred = a if pa <= pb and pa <= pc else (b if pb <= pc else c)
                line[i] = (line[i] + pred) & 0xff
        elif ft != 0:
            raise SystemExit(f"{path}：未知的濾波 {ft}")
        if bpp == 4:
            out[y * ROW:(y + 1) * ROW] = line
        else:
            o = y * ROW
            for x in range(w):
                out[o + x * 4:o + x * 4 + 3] = line[x * 3:x * 3 + 3]
                out[o + x * 4 + 3] = 0xff
        prev = line
    # Go 的 png.Encode 對 *image.RGBA：不透明時寫 RGB，否則轉成非預乘 NRGBA 再寫；後者讀回不等於預乘值，
    # 由呼叫端的雜湊核對擋下。
    return bytes(out)


def load_pixels(spec):
    """spec = (種類, 路徑)：png、zlib、raw。"""
    kind, path = spec
    if kind == "png":
        return png_rgba(path)
    with open(path, "rb") as f:
        raw = f.read()
    return zlib.decompress(raw) if kind == "zlib" else raw


def pair_diff(job):
    """一對（Go、Rust）畫格的差異：回傳 (Go 雜湊, Rust 雜湊, 錯誤, [(x, y, Go 像素, Rust 像素)], 最大通道差)。"""
    g_sha, g_spec, r_sha, r_spec = job
    a, b = load_pixels(g_spec), load_pixels(r_spec)
    if sha(a) != g_sha:
        return g_sha, r_sha, f"Go 像素 {g_spec[1]} 讀回的雜湊 {sha(a)} 不等於 {g_sha}", [], 0
    if sha(b) != r_sha:
        return g_sha, r_sha, f"Rust 像素 {r_spec[1]} 讀回的雜湊 {sha(b)} 不等於 {r_sha}", [], 0
    pts, maxd = [], 0
    for y in range(H):
        o = y * ROW
        if a[o:o + ROW] == b[o:o + ROW]:
            continue
        for x in range(W):
            i = o + x * 4
            pa, pb = a[i:i + 4], b[i:i + 4]
            if pa != pb:
                pts.append((x, y, pa, pb))
                maxd = max(maxd, max(abs(pa[k] - pb[k]) for k in range(4)))
    return g_sha, r_sha, None, pts, maxd


def inside(x, y, box):
    return box[0] <= x < box[2] and box[1] <= y < box[3]


def mul255(x):
    return (x + (x >> 8) + 0x80) >> 8


def over(s, a, d):
    return min(255, s + mul255(d * (255 - a)))


_UNDER = {}


def under(s, a, out):
    """所有使 over(s, a, d) == out 的底色 d（快取；TTF 圖層的 (s, a) 種類很少）。"""
    k = (s, a, out)
    if k not in _UNDER:
        _UNDER[k] = frozenset(d for d in range(256) if over(s, a, d) == out)
    return _UNDER[k]


BOUND_RE = re.compile(r"bound=(-?\d+),(-?\d+),(-?\d+),(-?\d+)")
INK_RE = re.compile(r"\|ink=(none|-?\d+,-?\d+,-?\d+,-?\d+)")
TEXT_RE = re.compile(r'text="((?:[^"\\]|\\.)*)"')


def ttf_from_key(key):
    texts = [json.loads('"' + m + '"') for m in TEXT_RE.findall(key)]
    bounds = [[int(v) for v in m] for m in BOUND_RE.findall(key)]
    return texts, bounds


class Checker:
    def __init__(self, mode, out, jobs):
        self.mode, self.out, self.jobs = mode, out, jobs
        self.pending = []  # (index, meta, g_sha, g_spec, r_sha, r_spec, ttf_ok, inks, bounds, layers, texts)
        self.pairs = {}
        self.rows = []
        self.violations = []

    def add(self, *item):
        self.pending.append(item)

    def run(self):
        jobs = {}
        for _, _, g_sha, g_spec, r_sha, r_spec, *_ in self.pending:
            jobs.setdefault((g_sha, r_sha), (g_sha, g_spec, r_sha, r_spec))
        work = list(jobs.values())
        if self.jobs > 1 and len(work) > 1:
            with multiprocessing.Pool(self.jobs) as pool:
                results = pool.map(pair_diff, work, chunksize=4)
        else:
            results = [pair_diff(j) for j in work]
        for g_sha, r_sha, err, pts, maxd in results:
            if err:
                raise SystemExit(err)
            self.pairs[(g_sha, r_sha)] = (pts, maxd)
        for item in self.pending:
            self.analyze(*item)

    def analyze(self, index, meta, g_sha, _gs, r_sha, _rs, ttf_ok, ink_boxes, bound_boxes, layers, texts):
        pts, maxd = self.pairs[(g_sha, r_sha)]
        outside_ink = sum(1 for x, y, _, _ in pts if not any(inside(x, y, b) for b in ink_boxes))
        outside_bound = sum(1 for x, y, _, _ in pts if not any(inside(x, y, b) for b in bound_boxes))
        bbox = None
        if pts:
            xs, ys = [p[0] for p in pts], [p[1] for p in pts]
            bbox = [min(xs), min(ys), max(xs) + 1, max(ys) + 1]
        row = dict(meta, capture=index, go_sha256=g_sha, rust_sha256=r_sha, diff_pixels=len(pts),
                   max_channel_diff=maxd, diff_bbox=bbox, ttf_texts=list(texts), ink_boxes=ink_boxes,
                   bound_boxes=bound_boxes, outside_ink=outside_ink, outside_bound=outside_bound, ttf_key_ok=ttf_ok)
        if layers is not None:
            row.update(self.layer_check(pts, layers))
        ok = ttf_ok and bool(ink_boxes) and outside_ink == 0 and row.get("layer_same_at_diff", 0) == 0
        row["inside"] = ok
        if not ok:
            self.violations.append(row)
        self.rows.append(row)

    @staticmethod
    def layer_check(pts, layers):
        """layers = (crop, go_crop_rgba, rust_crop_rgba)：r5f 的 TTF 圖層裁切。"""
        crop, gl, rl = layers
        cw = crop[2] - crop[0]
        same_at_diff = outside_crop = explained = 0
        for x, y, gf, rf in pts:
            if not inside(x, y, crop):
                outside_crop += 1
                continue
            j = ((y - crop[1]) * cw + (x - crop[0])) * 4
            gs, rs = gl[j:j + 4], rl[j:j + 4]
            if gs == rs:
                same_at_diff += 1
                continue
            if all(under(gs[c], gs[3], gf[c]) & under(rs[c], rs[3], rf[c]) for c in range(4)):
                explained += 1
        return {"layer_outside_crop": outside_crop, "layer_same_at_diff": same_at_diff,
                "composite_explained": explained}

    def missing(self, index, meta):
        self.violations.append(dict(meta, capture=index, inside=False))

    def write(self, summary):
        rows = self.rows
        summary["different"] = len(rows) + sum(1 for v in self.violations if "diff_pixels" not in v)
        summary["pairs"] = len(self.pairs)
        summary["diff_pixels_total"] = sum(r["diff_pixels"] for r in rows)
        summary["max_channel_diff"] = max([r["max_channel_diff"] for r in rows] or [0])
        summary["outside_ink_total"] = sum(r["outside_ink"] for r in rows)
        summary["outside_bound_total"] = sum(r["outside_bound"] for r in rows)
        if rows and "layer_same_at_diff" in rows[0]:
            summary["layer_same_at_diff_total"] = sum(r.get("layer_same_at_diff", 0) for r in rows)
            summary["composite_explained_total"] = sum(r.get("composite_explained", 0) for r in rows)
        by_text = {}
        for r in rows:
            t = " + ".join(r["ttf_texts"]) or "(沒有 TTF)"
            e = by_text.setdefault(t, {"captures": 0, "inside": 0, "pairs": set(), "diff_pixels_min": None,
                                       "diff_pixels_max": 0, "max_channel_diff": 0, "outside_bound": 0})
            e["captures"] += 1
            e["inside"] += r["inside"]
            e["pairs"].add((r["go_sha256"], r["rust_sha256"]))
            e["diff_pixels_max"] = max(e["diff_pixels_max"], r["diff_pixels"])
            e["diff_pixels_min"] = r["diff_pixels"] if e["diff_pixels_min"] is None else min(e["diff_pixels_min"], r["diff_pixels"])
            e["max_channel_diff"] = max(e["max_channel_diff"], r["max_channel_diff"])
            e["outside_bound"] += r["outside_bound"]
        for e in by_text.values():
            e["pairs"] = len(e["pairs"])
        summary["by_text"] = by_text
        if rows:
            summary["first_difference"] = {k: rows[0][k] for k in ("capture", "diff_pixels", "ttf_texts", "diff_bbox")}
        summary["violations"] = self.violations[:20]
        summary["violation_count"] = len(self.violations)
        summary["ok"] = summary.get("ok", True) and not self.violations
        with open(os.path.join(self.out, f"{self.mode}-captures.jsonl"), "w", encoding="utf-8") as f:
            for row in rows:
                f.write(json.dumps(row, ensure_ascii=False, sort_keys=True) + "\n")
        with open(os.path.join(self.out, f"{self.mode}-summary.json"), "w", encoding="utf-8") as f:
            json.dump(summary, f, ensure_ascii=False, indent=1, sort_keys=True)
            f.write("\n")
        lines = [f"模式：{self.mode}"]
        for k in ("captures", "equal", "different", "pairs", "diff_pixels_total", "max_channel_diff",
                  "outside_ink_total", "outside_bound_total", "layer_same_at_diff_total", "composite_explained_total",
                  "violation_count", "ok"):
            if k in summary:
                lines.append(f"{k}: {summary[k]}")
        for k in ("sequence_sha256", "pump_captures", "by_text", "by_point", "by_draw_path", "stage0",
                  "first_difference", "special"):
            if k in summary:
                lines.append(f"{k}: {json.dumps(summary[k], ensure_ascii=False, sort_keys=True)}")
        for v in self.violations[:5]:
            lines.append("違規：" + json.dumps(v, ensure_ascii=False, sort_keys=True))
        text = "\n".join(lines) + "\n"
        with open(os.path.join(self.out, f"{self.mode}-summary.txt"), "w", encoding="utf-8") as f:
            f.write(text)
        print(text, end="")


def chapter(args):
    go, rust = load_json(args.go), load_json(args.rust)
    tg, tr = load_json(args.ttf_go), load_json(args.ttf_rust)
    go_dir, rust_dir = os.path.dirname(os.path.abspath(args.go)), os.path.dirname(os.path.abspath(args.rust))
    gh, rh = expand(go["rows"]), expand(rust["rows"])
    tgk, trk = expand(tg["rows"], "key"), expand(tr["rows"], "key")
    summary = {"captures": len(gh), "rust_captures": len(rh), "rust_stopped": rust.get("stopped"),
               "ttf_layer_captures": [len(tgk), len(trk)], "go_frames_json_sha256": sha(open(args.go, "rb").read()),
               "sequence_sha256": [go["sequence_sha256"], rust["sequence_sha256"]]}
    ok = len(gh) == len(rh) == len(tgk) == len(trk) and not rust.get("stopped")
    go_file = {}
    for row in go["rows"]:
        go_file.setdefault(row["rgba_sha256"], row["file"])
    checker = Checker("chapter", args.out, args.jobs)
    equal = 0
    by_path = {}
    ttf_captures = sum(1 for k, _ in tgk if k.startswith("tail|ttf=") and not k.startswith("tail|ttf=0"))
    ttf_equal = 0
    for i in range(min(len(gh), len(rh))):
        (g, grow), (r, _) = gh[i], rh[i]
        gk, rk = tgk[i][0], trk[i][0]
        has_ttf = gk.startswith("tail|ttf=") and not gk.startswith("tail|ttf=0")
        path = grow.get("draw_path", "")
        e = by_path.setdefault(path, [0, 0])
        e[0] += 1
        if g == r:
            equal += 1
            e[1] += 1
            ttf_equal += has_ttf
            continue
        render_g, render_r = tg["renders"].get(gk), tr["renders"].get(rk)
        ttf_ok = bool(gk == rk and has_ttf and render_g and render_r)
        texts, bounds = ttf_from_key(gk) if ttf_ok else ([], [])
        inks, layers = [], None
        if ttf_ok:
            # 允許範圍只用 Go（黃金）的墨跡外框；Rust 的墨跡外框不同（文字位置不同）也算不通過。
            inks = [render_g["ink_box"]]
            ttf_ok = render_r["ink_box"] == render_g["ink_box"]
            if render_g["crop"] == render_r["crop"]:
                layers = (render_g["crop"], base64.b64decode(render_g["crop_rgba_base64"]),
                          base64.b64decode(render_r["crop_rgba_base64"]))
        meta = {"update_frame": grow["update_frame"], "node": grow.get("node"), "draw_path": path,
                "ttf_key_go": gk, "ttf_key_rust": rk}
        rpath = os.path.join(rust_dir, "frames", r + ".rgba")
        if not os.path.exists(rpath) or g not in go_file:
            checker.missing(i, dict(meta, go_sha256=g, rust_sha256=r, missing_pixels=True))
            continue
        checker.add(i, meta, g, ("png", os.path.join(go_dir, go_file[g])), r, ("raw", rpath), ttf_ok, inks, bounds,
                    layers, texts)
    checker.run()
    summary["equal"] = equal
    summary["ttf_captures"] = {"captures": ttf_captures, "equal": ttf_equal}
    summary["by_draw_path"] = {k: {"captures": v[0], "equal": v[1]} for k, v in sorted(by_path.items())}
    special = [r for r in checker.rows if r["capture"] == 13717]
    if special:
        summary["special"] = {k: special[0][k] for k in ("capture", "update_frame", "node", "draw_path", "ttf_texts",
                                                          "diff_pixels", "inside", "ink_boxes", "bound_boxes")}
    summary["ok"] = ok
    checker.write(summary)
    return summary["ok"]


def facility(args):
    go, rust = load_json(args.go), load_json(args.rust)
    go_dir, rust_dir = os.path.dirname(os.path.abspath(args.go)), os.path.dirname(os.path.abspath(args.rust))
    gk, rk = expand(go["rows"], "key"), expand(rust["rows"], "key")
    summary = {"captures": len(gk), "rust_captures": len(rk), "pump_captures": [go["pump_captures"], rust["pump_captures"]],
               "sequence_sha256": [go["sequence_sha256"], rust["sequence_sha256"]],
               "branches": [go["branches"], rust["branches"]]}
    ok = len(gk) == len(rk) and go["pump_captures"] == rust["pump_captures"]
    stage0 = None
    if args.stage0:
        s0 = load_json(args.stage0)
        stage0 = (expand(s0["rows"]), os.path.dirname(os.path.abspath(args.stage0)))
    checker = Checker("facility", args.out, args.jobs)
    equal = 0
    by_point = {}
    s0_total = s0_equal = 0
    for i in range(min(len(gk), len(rk))):
        (ga, grow), (ra, _) = gk[i], rk[i]
        pump = grow["pump_capture"] + (i - grow["capture"]) if grow["pump_capture"] >= 0 else -1
        point = ga.split("|", 1)[0]
        node = (grow.get("node") or "-").split("_")[0]
        cat = f"{point}:{node}"
        e = by_point.setdefault(cat, [0, 0])
        e[0] += 1
        g_sha, r_sha = ga.rsplit("rgba=", 1)[1], ra.rsplit("rgba=", 1)[1]
        if stage0 and pump >= 0:
            s0_total += 1
            s0_equal += pump < len(stage0[0]) and stage0[0][pump][0] == g_sha
        if ga == ra:
            equal += 1
            e[1] += 1
            continue
        g_state, r_state = ga.rsplit("|rgba=", 1)[0], ra.rsplit("|rgba=", 1)[0]
        ttf_ok = g_state == r_state and "|ttf=-" not in g_state and "|ttf=0" not in g_state and "|ttf=" in g_state
        inks, texts, bounds = [], [], []
        if ttf_ok:
            m = INK_RE.search(g_state)
            if m and m.group(1) != "none":
                inks = [[int(v) for v in m.group(1).split(",")]]
            texts, bounds = ttf_from_key(g_state)
        meta = {"point": point, "pump_capture": pump, "update_frame": grow["update_frame"], "node": grow.get("node"),
                "state_go": g_state, "state_rust": r_state}
        g_spec = None
        zpath = os.path.join(go_dir, "frames", g_sha + ".rgba.z")
        if os.path.exists(zpath):
            g_spec = ("zlib", zpath)
        elif stage0 and 0 <= pump < len(stage0[0]) and stage0[0][pump][0] == g_sha:
            g_spec = ("png", os.path.join(stage0[1], stage0[0][pump][1]["file"]))
        rpath = os.path.join(rust_dir, "frames", r_sha + ".rgba")
        if g_spec is None or not os.path.exists(rpath):
            checker.missing(i, dict(meta, go_sha256=g_sha, rust_sha256=r_sha,
                                    missing_pixels={"go": g_spec is None, "rust": not os.path.exists(rpath)}))
            continue
        checker.add(i, meta, g_sha, g_spec, r_sha, ("raw", rpath), ttf_ok, inks, bounds, None, texts)
    checker.run()
    summary["equal"] = equal
    summary["by_point"] = {k: {"captures": v[0], "equal": v[1]} for k, v in sorted(by_point.items())}
    if stage0:
        summary["stage0"] = {"pump_captures": s0_total, "go_equal_to_stage0": s0_equal}
    summary["ok"] = ok
    checker.write(summary)
    return summary["ok"]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("mode", choices=["chapter", "facility"])
    ap.add_argument("--go", required=True)
    ap.add_argument("--rust", required=True)
    ap.add_argument("--ttf-go")
    ap.add_argument("--ttf-rust")
    ap.add_argument("--stage0")
    ap.add_argument("--out", required=True)
    ap.add_argument("--jobs", type=int, default=1)
    args = ap.parse_args()
    os.makedirs(args.out, exist_ok=True)
    if args.mode == "chapter":
        if not (args.ttf_go and args.ttf_rust):
            ap.error("chapter 模式需要 --ttf-go 與 --ttf-rust")
        ok = chapter(args)
    else:
        ok = facility(args)
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
