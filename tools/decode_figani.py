#!/usr/bin/env python3
"""炎龍騎士團2 — FIGANI 戰鬥動畫逐幀解碼器。

從 FD2.EXE 反組譯出的 sprite RLE 解碼器(參數化版 0x4F43D)逐指令還原。
能把格式有效的 FIGANI 動畫逐幀解出透明 sprite；高階演出語意仍須依 caller
逐項證實，不以工具名稱或成功解圖冒稱完整 renderer。

== 容器(FIGANI.DAT)==
LLLLLL 容器(見 unpack_dat.py)→ 每個資源 = 一段動畫。
動畫:+0 uint8 frameCount、+1 raw prelude flag、+2 raw prelude frame count、
+4 raw header byte（特定 caller 消費）；u32[frameCount] 幀 offset 從 +8 開始。

== 每幀 ==
13-byte 標頭:
  +0  int16 dx       320×200 畫布中的絕對螢幕 X
  +2  int16 dy       320×200 畫布中的絕對螢幕 Y
  +4  u8 raw4       caller-specific marker
  +5  u8 raw5       caller-specific sample marker
  +6  u8 delay      caller-specific repeat count；直接索引效果可為 0
  +7  u8 raw7       caller-specific flag
  +8  u8 reserved
  +9  u16 W          點陣解碼寬(realW)
  +11 u16 H          點陣解碼高(realH)
  +13 …  RLE 像素資料(解碼到 W×H)

== RLE(4 模式;控制 byte 高 2 bit = 模式,低 6 bit → count=(c&0x3F)+1)==
  00  色彩 run       讀 1 像素,重複 count 次
  01  dither/陰影    讀 1 像素,輸出 [透明,值]×count(隔位寫,佔 2×count 寬)
  10  literal        讀 count 個像素原樣
  11  透明 skip      跳過 count(留底=透明)

調色盤:FDOTHER 資源 #0(見 decode_image.py)。透明色預設 index 0。

用法:
    python3 decode_figani.py frames <FIGANI_NNN.bin> <palette.bin> <out目錄>
    python3 decode_figani.py gif    <FIGANI_NNN.bin> <palette.bin> <out.gif>
    python3 decode_figani.py info   <FIGANI_NNN.bin>
"""
import sys
import os
import struct
import json


def load_palette(path):
    with open(path, "rb") as stream:
        raw = stream.read(768)
    pal = []
    for i in range(256):
        r, g, b = raw[i*3], raw[i*3+1], raw[i*3+2]
        pal += [(r << 2) | (r >> 4), (g << 2) | (g >> 4), (b << 2) | (b >> 4)]
    return pal


def decode_rle_layers(body, w, h, trans=0):
    total = w * h
    out = bytearray()
    mask = bytearray()
    i = 0
    n = len(body)
    while len(out) < total and i < n:
        c = body[i]; i += 1
        mode = c >> 6
        cnt = (c & 0x3F) + 1
        if mode == 0:                       # 色彩 run
            v = body[i]; i += 1
            out += bytes([v]) * cnt
            mask += b"\xff" * cnt
        elif mode == 1:                     # dither / 陰影
            v = body[i]; i += 1
            out += bytes([trans, v]) * cnt
            mask += bytes([0, 255]) * cnt
        elif mode == 2:                     # literal
            out += body[i:i + cnt]; i += cnt
            mask += b"\xff" * cnt
        else:                               # 透明 skip
            out += bytes([trans]) * cnt
            mask += b"\x00" * cnt
    return (
        bytes(out[:total]).ljust(total, bytes([trans])),
        bytes(mask[:total]).ljust(total, b"\x00"),
    )


def decode_rle(body, w, h, trans=0):
    """相容舊呼叫端；正式匯出另以 mask 保留透明與不透明索引零。"""
    return decode_rle_layers(body, w, h, trans)[0]


def parse_anim(d):
    """回傳 [(x,y,w,h,raw4,raw5,delay,raw7,body_bytes), ...]。"""
    if len(d) < 12:
        return []
    nf = d[0]
    if nf == 0 or 8 + 4 * nf > len(d):
        return []
    offs = [struct.unpack_from("<I", d, 8 + 4 * i)[0] for i in range(nf)]
    frames = []
    for fi in range(nf):
        o = offs[fi]
        end = offs[fi + 1] if fi + 1 < nf else len(d)
        if o + 13 > len(d):
            continue
        x, y = struct.unpack_from("<hh", d, o)
        raw4, raw5, delay, raw7 = d[o + 4:o + 8]
        w = struct.unpack_from("<H", d, o + 9)[0]
        h = struct.unpack_from("<H", d, o + 11)[0]
        if not (0 < w <= 1024 and 0 < h <= 1024):
            continue
        frames.append((x, y, w, h, raw4, raw5, delay, raw7, d[o + 13:end]))
    return frames


def render_frame(w, h, body, pal, trans=0):
    from PIL import Image

    px, mask = decode_rle_layers(body, w, h, trans)
    rgba = bytearray(w * h * 4)
    for index, palette_index in enumerate(px):
        rgba[index * 4:index * 4 + 3] = bytes(pal[palette_index * 3:palette_index * 3 + 3])
        rgba[index * 4 + 3] = mask[index]
    return Image.frombytes("RGBA", (w, h), bytes(rgba))


def cmd_frames(src, palp, outdir):
    with open(src, "rb") as stream:
        d = stream.read()
    pal = load_palette(palp)
    os.makedirs(outdir, exist_ok=True)
    base = os.path.splitext(os.path.basename(src))[0]
    frames = parse_anim(d)
    document = {
        "schema_version": 1,
        "document_id": f"animation/{base.lower()}",
        "kind": "animation",
        "source": {"file": "FIGANI.DAT", "resource": int(base.rsplit("_", 1)[-1])},
        "animation_id": f"animation/{base.lower()}",
        "native_header": {"byte_1": d[1], "byte_2": d[2], "byte_4": d[4]},
        "frames": [],
        "extensions": {},
    }
    for fi, (x, y, w, h, raw4, raw5, delay, raw7, body) in enumerate(frames):
        from PIL import Image
        pixels, mask = decode_rle_layers(body, w, h)
        filename = f"{base}_f{fi:02d}.png"
        mask_filename = f"{base}_f{fi:02d}_mask.png"
        indexed = Image.frombytes("P", (w, h), pixels)
        indexed.putpalette(pal)
        indexed.save(os.path.join(outdir, filename))
        Image.frombytes("L", (w, h), mask).save(os.path.join(outdir, mask_filename))
        document["frames"].append({
            "frame_id": f"frame/{fi:03d}",
            "asset_id": f"battle_animation/animations/{base.lower()}/frame_{fi:03d}.png",
            "path": filename,
            "mask_asset_id": f"battle_animation/animations/{base.lower()}/frame_{fi:03d}_mask.png",
            "mask_path": mask_filename,
            "delay_native": delay,
            "x": x,
            "y": y,
            "width": w,
            "height": h,
            "raw_byte_4": raw4,
            "raw_byte_5": raw5,
            "raw_byte_7": raw7,
            "extensions": {},
        })
    with open(os.path.join(outdir, "animation.json"), "w", encoding="utf-8") as stream:
        json.dump(document, stream, ensure_ascii=False, indent=2)
        stream.write("\n")
    print(f"{base}: {len(frames)} 幀 -> {outdir}")


def cmd_gif(src, palp, out):
    from PIL import Image

    with open(src, "rb") as stream:
        d = stream.read()
    pal = load_palette(palp)
    frames = parse_anim(d)
    if not frames:
        print("無幀"); return
    W = max(w for _, _, w, _, _, _, _, _, _ in frames)
    H = max(h for _, _, _, h, _, _, _, _, _ in frames)
    ims = []
    for _, _, w, h, _, _, _, _, body in frames:
        canvas = Image.new("P", (W, H), 0)
        canvas.putpalette(pal)
        canvas.paste(render_frame(w, h, body, pal), (0, H - h))
        ims.append(canvas.convert("RGB"))
    ims[0].save(out, save_all=True, append_images=ims[1:], duration=120, loop=0)
    print(f"{len(ims)} 幀 -> {out}")


def cmd_info(src):
    with open(src, "rb") as stream:
        d = stream.read()
    frames = parse_anim(d)
    print(f"{os.path.basename(src)}: {len(frames)} 幀")
    for fi, (x, y, w, h, raw4, raw5, delay, raw7, body) in enumerate(frames):
        print(f"  幀{fi}: ({x},{y}) {w}x{h} raw4={raw4} raw5={raw5} delay={delay} raw7={raw7} 壓縮={len(body)}B")


def main(argv):
    if len(argv) < 3:
        print(__doc__); return 1
    cmd = argv[1]
    if cmd == "frames":
        cmd_frames(argv[2], argv[3], argv[4])
    elif cmd == "gif":
        cmd_gif(argv[2], argv[3], argv[4])
    elif cmd == "info":
        cmd_info(argv[2])
    else:
        print(__doc__); return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
