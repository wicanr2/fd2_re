#!/usr/bin/env python3
"""gen_icon.py — 產生 fd2.png(AppImage/.desktop 圖示,256x256)。

原創幾何圖形(PIL 畫圖,無任何 ROM 抽取素材)——原版 title/sprite 素材受著作權保護、
本 repo 規範不入庫(見 remake/.gitignore),桌面圖示這種會被打包/入庫的檔案不能用抽取素材。
造型:深藍底 + 紅色劍形 + 金色邊框,呼應「炎龍騎士團」配色,純意象不臨摹原圖。
"""
from PIL import Image, ImageDraw

SIZE = 256
im = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
d = ImageDraw.Draw(im)

# 圓角深藍底
bg = (18, 28, 64, 255)
gold = (212, 168, 64, 255)
red = (176, 32, 40, 255)
d.rounded_rectangle([8, 8, SIZE - 8, SIZE - 8], radius=36, fill=bg, outline=gold, width=8)

# 劍身(直向,置中偏上)+ 劍柄(橫向,置中偏下)+ 劍尖
cx = SIZE // 2
d.polygon([(cx, 40), (cx + 16, 70), (cx + 16, 160), (cx - 16, 160), (cx - 16, 70)], fill=(230, 230, 236, 255))
d.polygon([(cx, 40), (cx + 16, 70), (cx - 16, 70)], fill=(230, 230, 236, 255))  # 劍尖收窄(視覺上與劍身同色即可)
d.rectangle([cx - 44, 160, cx + 44, 178], fill=gold)  # 護手
d.rectangle([cx - 10, 178, cx + 10, 214], fill=(90, 56, 30, 255))  # 握柄
d.ellipse([cx - 16, 210, cx + 16, 226], fill=gold)  # 柄頭

# 焰紋(劍身兩側簡化火焰三角,呼應「炎龍」)
d.polygon([(cx - 30, 130), (cx - 16, 100), (cx - 16, 150)], fill=red)
d.polygon([(cx + 30, 130), (cx + 16, 100), (cx + 16, 150)], fill=red)

im.save("fd2.png")
print("wrote fd2.png", im.size)
