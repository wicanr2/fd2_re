#!/usr/bin/env python3
"""R5 階段 4（docs/goal/121，r5w）：fd2-app 外殼與無畫面驅動端的比對。

用法：rust_r5_app_frames_compare.py --headless <frames.jsonl> --app <外殼輸出目錄> --out <compare.json>

1. 驅動一致：外殼 `frames.jsonl` 與無畫面驅動端的 `frames.jsonl` 逐行相同（每個顯示幀的 Update 次數、CPU 畫格
   SHA-256、`g.frame`、標題階段與節點）。
2. 外殼截圖：每張 `shots/<名稱>.png`（Bevy 截取的視窗畫面）以本檔的 PNG 解碼器讀回，與 `shots/<名稱>.rgba`
   （同一幀的 640×400 CPU 畫格，alpha 255）做整數倍最近鄰放大、置中、黑邊後的預期畫面逐像素比對；倍率與位置
   由本檔依視窗大小獨立計算（`max(1, min(w/640, h/400))`，餘下平分）。另核對 `.rgba` 的雜湊等於同一幀
   `frames.jsonl` 的 CPU 畫格雜湊（只差 alpha 改 255 時不會相同，所以改比外殼 `shots.jsonl` 記的 `cpu_rgba_sha256`）。
不遮蔽任何像素。結束碼：全部相同 0，否則 1。
"""
import argparse
import hashlib
import json
import os
import struct
import sys
import zlib

LW, LH = 640, 400


def png_rgba(path):
    """8 位元 RGB／RGBA、非交錯 PNG → (w, h, RGBA bytes)。"""
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
    if depth != 8 or ctype not in (2, 6) or interlace:
        raise SystemExit(f"{path}：不支援的 PNG 格式 {hdr}")
    bpp = 3 if ctype == 2 else 4
    stride = w * bpp
    raw = zlib.decompress(b"".join(idat))
    prev = bytearray(stride)
    out = bytearray(w * h * 4)
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
        o = y * w * 4
        if bpp == 4:
            out[o:o + w * 4] = line
        else:
            for x in range(w):
                out[o + x * 4:o + x * 4 + 3] = line[x * 3:x * 3 + 3]
                out[o + x * 4 + 3] = 0xff
        prev = line
    return w, h, bytes(out)


def expected(frame, w, h):
    k = max(1, min(w // LW, h // LH))
    ox, oy = max(0, w - LW * k) // 2, max(0, h - LH * k) // 2
    black = b"\x00\x00\x00\xff"
    rows = []
    for y in range(h):
        if not (oy <= y < oy + LH * k):
            rows.append(black * w)
            continue
        sy = (y - oy) // k
        src = frame[sy * LW * 4:(sy + 1) * LW * 4]
        mid = b"".join(src[x * 4:x * 4 + 4] * k for x in range(LW))
        right = max(0, w - ox - LW * k)
        rows.append(black * ox + mid[:(w - ox) * 4] + black * right)
    return k, ox, oy, b"".join(rows)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--headless", required=True)
    ap.add_argument("--app", required=True)
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    ok = True
    with open(a.headless, encoding="utf-8") as f:
        hl = f.read().splitlines()
    with open(os.path.join(a.app, "frames.jsonl"), encoding="utf-8") as f:
        al = f.read().splitlines()
    first = next((i for i, (x, y) in enumerate(zip(hl, al)) if x != y), None)
    frames = {"headless": len(hl), "app": len(al), "first_diff": first,
              "equal": first is None and len(hl) == len(al),
              "headless_sha256": hashlib.sha256("\n".join(hl).encode()).hexdigest(),
              "app_sha256": hashlib.sha256("\n".join(al).encode()).hexdigest(),
              "distinct_frames": len({json.loads(x)["rgba_sha256"] for x in al})}
    ok &= frames["equal"]
    shots = []
    with open(os.path.join(a.app, "shots.jsonl"), encoding="utf-8") as f:
        app_shots = [json.loads(x) for x in f.read().splitlines()]
    for rec in app_shots:
        name = rec["name"]
        w, h, shot = png_rgba(os.path.join(a.app, "shots", name + ".png"))
        with open(os.path.join(a.app, "shots", name + ".rgba"), "rb") as f:
            frame = f.read()
        row = json.loads(al[rec["frame"]])
        # .rgba 是同一幀 CPU 畫格改 alpha 255；原畫格雜湊記在 frames.jsonl，兩者只差 alpha。
        alpha_only = len(frame) == LW * LH * 4 and all(frame[i] == 255 for i in range(3, len(frame), 4))
        k, ox, oy, exp = expected(frame, w, h)
        diff = sum(1 for i in range(0, len(exp), 4) if shot[i:i + 4] != exp[i:i + 4]) if len(shot) == len(exp) else -1
        item = {"name": name, "frame": rec["frame"], "window": [w, h], "scale": k, "offset": [ox, oy],
                "png_sha256": hashlib.sha256(shot).hexdigest(), "expected_sha256": hashlib.sha256(exp).hexdigest(),
                "diff_pixels": diff, "equal": diff == 0 and alpha_only,
                "cpu_frame_sha256": row["rgba_sha256"], "app_reported_equal": rec["equal"],
                "app_cpu_sha256_matches_frames": rec["cpu_rgba_sha256"] == row["rgba_sha256"]}
        ok &= item["equal"] and item["app_reported_equal"] and item["app_cpu_sha256_matches_frames"]
        shots.append(item)
    result = {"schema": "fd2-r5-app-compare/1", "frames": frames, "shots": shots, "ok": bool(ok)}
    with open(a.out, "w", encoding="utf-8") as f:
        json.dump(result, f, ensure_ascii=False, indent=2)
    print(json.dumps({"frames": frames, "shots": [(s["name"], s["window"], s["scale"], s["diff_pixels"]) for s in shots],
                      "ok": bool(ok)}, ensure_ascii=False))
    return 0 if ok else 1


if __name__ == "__main__":
    sys.exit(main())
