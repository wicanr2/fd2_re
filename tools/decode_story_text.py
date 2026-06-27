#!/usr/bin/env python3
"""用 glyph_map.json 把 FDTXT 章節解成 UTF-8 文字(說話者 + 對白)。
未對映字模顯示為 〈idx〉。控制碼:0xFFFF 結束;其餘 0xFF00+ 視為換行/段落。"""
import sys,json,struct,os
sys.path.insert(0,os.path.dirname(__file__))
from decode_text import parse_strings
PORT={0:"索爾",1:"哈諾",2:"鐵諾",3:"哈瓦特",4:"亞雷斯",5:"洛娜",6:"萊汀",7:"蘭斯洛特",8:"希莉亞",9:"悠妮",0xA:"瑪琳",0xB:"索菲亞",0xC:"凱麗",0xD:"貝克威",0xE:"珊",0xF:"賽可邦勒",0x10:"凱拉斯",0x11:"米亞斯多德",0x12:"蜜蒂",0x13:"羅德曼",0x14:"莎拉",0x15:"約拿",0x16:"卡里斯",0x17:"羅蘭",0x18:"希爾法",0x19:"謝多",0x1A:"聖寇拉斯",0x1B:"巴拿羅西亞",0x1C:"達克賽",0x1D:"亞奇梅吉",0x1E:"蓋亞",0x1F:"渥德"}
def main(a):
    m=json.load(open("docs/data/glyph_map.json",encoding="utf-8"))
    gm={int(k):v for k,v in m.items() if k!="_comment"}
    ss=parse_strings(a[1])
    miss={}
    for si,s in enumerate(ss):
        out=[]
        for c in s:
            if c==0xFFFF: break
            if c>=0xFF00: out.append("\n  ")  # 換行
            elif c in gm: out.append(gm[c])
            else: out.append(f"〈{c}〉"); miss[c]=miss.get(c,0)+1
        print(f"[{si}] "+"".join(out))
    if miss:
        tot=sum(miss.values())
        print(f"\n# 未對映 {len(miss)} 種 glyph,共 {tot} 次:",dict(sorted(miss.items())[:40]))
if __name__=="__main__": main(sys.argv)
