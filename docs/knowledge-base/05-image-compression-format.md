# 05 — 圖像壓縮格式完整規格

> 《炎龍騎士團2》(漢堂國際, 1995) 的圖像壓縮格式，由本專案第 2 輪逆向工程還原並驗證。
> 這份文件刻意寫得完整、可重現，作為 1990 年代台灣 DOS 遊戲技術的一份保存紀錄。

## 背景

本作為 VGA mode 13h(320×200, 256 色)遊戲。所有圖像存於 `.DAT` 容器(見 `01-container-and-asset-formats.md`)。
全螢幕圖若不壓縮為 64000 byte;原版用一套極簡的 **位元組導向 RLE**(run-length encoding)把標題與
戰鬥背景壓到約 1/3。本格式無查表、無 Huffman、無字典——是當年「夠快又夠省」的務實設計。

## 像素容器標頭

```
偏移  型別        說明
+0    uint16 LE   width   (寬,像素)
+2    uint16 LE   height  (高,像素)
+4    …           像素資料(見下)
```

像素為 **8-bit 調色盤索引**。資料分兩型:

### 型 A — 未壓縮

當 `資源長度 - 4 == width × height`，`+4` 之後即原始逐列像素，無壓縮。
實例:`FDOTHER` 容器資源 #15、#55(皆 320×200，size = 64004)。

### 型 B — RLE 壓縮

當 `資源長度 - 4 < width × height`，像素資料為下列 RLE 串流。

## RLE 演算法

串流由一連串 **token** 組成。每個 token 先讀一個控制位元組 `c`：

```
c = next_byte()
if c >= 0x80:                      # 文字串 (literal)
    n = (c & 0x7F) + 1             #   接下來 n 個 byte 原樣輸出
    output( next_bytes(n) )
else:                              # 連續執行 (run)
    v = next_byte()               #   下一個 byte 是像素值
    output( v repeated (c + 1) times )
```

- literal 一次最多 `0x7F + 1 = 128` 個像素。
- run 一次最多 `0x7F + 1 = 128` 個相同像素。
- 解碼到輸出累計達 `width × height` 即結束。

**判定強約束(驗證用)**：正確的解碼器對任一圖必輸出**剛好 `width × height`** 個像素;
長度不符即代表演算法或起點錯誤。本演算法在 `TITLE_000`(320×200)、`BG_*`(320×100)、
`FDOTHER` 背景上全部精確命中，並渲染出可辨識的正確畫面。

## 逐步實例(TITLE_000 開頭)

標頭 `40 01 c8 00` → width=0x0140=320, height=0x00C8=200, target=64000。
像素串流開頭：

```
3F F0 3F F0 3F …
```

| token | c | 動作 | 產生像素 |
|---|---|---|---|
| 1 | `3F` | run，c<0x80，重複下一 byte `F0` 共 0x3F+1=64 次 | 64×0xF0 |
| 2 | `3F` | run，重複 `F0` 64 次 | 64×0xF0 |
| … | | (標題背景大片同色 → 連續多個 64 長 run) | |

(標題畫面四周為大片單色背景，故開頭是一連串長 run;進入 logo 區後才出現 literal。)

## 調色盤

256 色 VGA 調色盤存於 `FDOTHER` 容器資源 #0，768 byte = 256 × (R, G, B)，每分量 6-bit(0–63)。
轉 8-bit:`v8 = (v6 << 2) | (v6 >> 4)`。此調色盤對標題與多數戰鬥背景正確;不同場景可能切換調色盤，
其餘調色盤資源待後輪清點。

## 參考實作

`tools/decode_image.py`：

```bash
# 單張轉 PNG(需自備原版,先用 unpack_dat.py 解出資源)
python3 tools/decode_image.py 資源.bin 調色盤.bin 輸出.png
```

核心解碼函式 `decode_rle(body, target)` 即上述演算法。

## 已驗證成果

| 資源 | 尺寸 | 型 | 還原內容 |
|---|---|---|---|
| `TITLE_000` | 320×200 | RLE | 遊戲標題「FLAME DRAGON 2 — Legend of Golden Castle」 |
| `BG_003` | 320×100 | RLE | 戰鬥背景:連綿山脈 |
| `BG_010` | 320×100 | RLE | 戰鬥背景:村莊房舍 + 石牆 |
| `FDOTHER_015` | 320×200 | 未壓縮 | 熔岩材質 |
| `FDOTHER_055` | 320×200 | 未壓縮 | 藍色雲層 |

全幅背景 / 標題類約 125 張已可解。**sprite 類**(24×24 圖塊、戰鬥動畫格、人物立繪)
推測使用本 RLE 的**透明(skip)變體**以支援非矩形圖形，於 `06-animation-format.md` 接續處理。
