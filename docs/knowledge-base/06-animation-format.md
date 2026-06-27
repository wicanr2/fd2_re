# 06 — 動畫機制(AFM)格式紀錄

> 《炎龍騎士團2》的戰鬥 / 特效動畫系統。本專案第 2–3 輪逆向工程整理。
> 與圖像壓縮(`05-…`)併為一份台灣 1990 年代 DOS 遊戲技術的保存紀錄。

## 一個珍貴的署名:AFM by Lo Yuan Tsung (1993)

`ANI.DAT` 容器資源 #0 開頭即原作者自製動畫工具的版權橫幅:

```
AFM - Animation File Manager Version 1.00 Copyright (C) 1993 Lo Yuan Tsung 09/29
```

**AFM(Animation File Manager)v1.00**，作者 **Lo Yuan Tsung**(羅元聰)，1993 年 9 月 29 日。
這是漢堂團隊為《炎龍騎士團》系列自製的動畫管理系統 —— 把當年程式設計師的名字與工具一併留下。

## 兩種動畫容器

| 容器 | 資源數 | 用途 | 格式 |
|---|---|---|---|
| `ANI.DAT` | 10 | 過場 / 片頭 / 大型動畫 | 完整 **AFM 檔**(資源 #0 帶 AFM 橫幅) |
| `FIGANI.DAT` | 409 | 戰鬥招式 / 法術特效動畫 | 精簡的「每動畫一資源」幀封裝(見下) |

`FIGANI.DAT` 是全專案最大檔(15.3 MB)，承載所有戰鬥動畫。

## FIGANI 每動畫結構 [已驗證]

每個 `FIGANI` 資源 = 一段動畫，自描述其幀數(與 `.DAT` 主容器同一手法):

```
+0   uint16 LE  frameCount        (幀數)
+2   uint16 LE  ?                  (常等於 frameCount;用途待定)
+4   uint16 LE  ?                  (0/2/5… 可能是播放參數)
+6   uint16 LE  ?
+8   uint32[frameCount] LE  各幀資料 offset(相對資源起點)
          frameCount = (offsets[0] - 8) / 4   ← 自洽驗證
```

每幀:

```
+0   uint16 LE  width
+2   uint16 LE  height
+4   …          壓縮像素(RLE,透明變體 — 見下「未解」)
```

**驗證實例**:
- `FIGANI_000`:frameCount=4，四幀皆約 167×75(同尺寸 → 連續動作幀)。
- `FIGANI_001`:frameCount=11，幀尺寸 167×75→206×45→… 多姿勢動畫。
- `(first_offset-8)/4` 與 `+0` 的 frameCount 在多個樣本一致 → 結構正確。

**3-byte 迷你資源**(如 `FIGANI_002` = `00 00 0A`):散落在動畫之間，推測為**群組分隔 / 索引標記**
(把 409 個資源切成「角色 → 招式 → 幀組」),非動畫本體。[假設]

## 幀像素 codec — 第 3 輪調查紀錄(尚未攻破)

逐幀拆解卡在「幀像素的壓縮 codec」。第 3 輪已排除多種假設，紀錄如下供後續接手:

**已知事實**
- 幀像素**不是**背景用的 RLE(`05-…`):用背景 RLE 解 `FIGANI_000` 幀0 只得 9203/12525 像素，
  渲染為水平亂條(逐列有結構但會 desync) → 確認是**另一套、逐列(per-row)** 的格式。
- 像素色值用滿 **0–255 全域**(含 0x80–0xFF) → **不能**用「高位元=命令」來區分命令與像素;必有專屬 escape。
- `0xFE` 是最頻繁的 byte(幀0 出現 1161 次,佔 22%),且**連續長度只有 1(875×)或 2(143×)**,
  其後接「正常像素色值」(分布與一般像素相同)。→ `0xFE` **不是** bulk 透明 run 的計數碼。
- 同一動畫不同 pose 的幀(`FIGANI_000` 幀0/幀1)**前 13 byte 完全相同**:
  `00 00 02 00 00 67 00 75 00 c0 02 fe ff` → 此前綴疑為**每幀子標頭 / 列結構**,非像素。

**已排除的假設**:① 背景 RLE 直接套用;② `0xFE n`=透明 run(過量 10×);
③ `0x80–0xFD`=run-of-next(過量 3×,因像素本就用高色值);④ `0xFF`/列尾(僅 3 次)。

**下一步(正解路徑)= 用反組譯當 oracle**
靜態猜 codec 已達合理上限,改從 `FD2.EXE` 反組譯**真正的 sprite 解碼迴圈**(Watcom C,
保護模式 LE)。第 3 輪反組譯進度(capstone in docker):
- `3C FE`(cmp al,0xFE)在 EXE 僅 1 處且為 `call` 位移之**假命中** → escape 檢查不走 `cmp al,0xFE`。
- `0x12cb0` 經反組譯確認為**純矩形列複製**(`for row: memcpy(dst,src,w); src+=src_stride; dst+=dst_stride`),
  **無透明、無 RLE** → frame 在進此 blit **之前**就已被另一個函式解壓成線性像素 buffer。
- 故 **decompressor 是獨立函式**,需從「誰填 `0x12cb0` 的 src 參數」回溯。

**已定位 sprite 解碼器家族(capstone 反組譯,第 3 輪)**
`rep stosb`/`rep movsb` 密集叢集落在 **`0x4E000`–`0x4F800`**(正是動畫播放器呼叫的函式群)。
其中 `0x4EB52` 為 **24×24 sprite RLE 解碼器**,已逐指令還原其文法:

```
edx = dst_stride - 24       ; 每列寫完跳 stride
bh  = 24 (每列剩餘寬), bl = 24 (列數)
迴圈讀控制 byte c:
  高 2 bit = 模式, 低 6 bit:count = (c & 0x3F) + 1
    00xxxxxx  色彩 run     : 讀 1 像素, rep stosb count 次
    01xxxxxx  dither/scaled: 讀 1 像素, 隔位寫(inc edi;stosb), 佔 2×count 寬
    10xxxxxx  literal      : 讀 count 個像素原樣寫
    11xxxxxx  透明 skip    : add edi,count(不寫,留底=透明)
  bh -= count;歸零換列(add edi,edx;dec bl)
像素經 [ebp+eax] 調色 remap 表轉換。
```

**狀態**:此 24×24 文法套用到 FIGANI(167 寬)能精確消耗位元組,但渲染仍為橫條(垂直相關峰值
偏離宣告寬度)→ FIGANI 戰鬥動畫用的是**同家族的另一個參數化變體**(非 24×24 那支),
其模式/位元配置需再反組譯 `0x4E000–0x4F800` 內對應函式確認。工具 `tools/decode_sprite.py`
已實作 24×24 文法,待對映正確變體後即可逐幀輸出 PNG。

> 重大進展:codec 從「完全未知」→「解碼器家族已定位、24×24 變體文法已逐指令還原」。
> 剩最後一哩:FIGANI 專屬變體的精確模式表。不偽造輸出,待對齊後再產圖。

## 其餘未解(後輪)
- **每幀前置欄位**:部分動畫(`FIGANI_013`)直接讀 W,H 得 H=0,顯示每幀資料前可能有繪製位移(x,y)/旗標。
- **標頭 +4 / +6 欄位**:疑為播放速度 / 迴圈 / 對齊基準。
- **ANI 完整 AFM 檔格式**:資源 #0 的 AFM 檔頭與 `FIGANI` 的關係。

> 完成定義:能把任一 `FIGANI` 動畫逐幀解出透明 sprite、依序輸出 PNG 序列 / GIF。
> 屆時上面「未攻破」段落改寫為「已驗證」,被推翻的假設一併刪除(誠實揭露,不偽造輸出)。
