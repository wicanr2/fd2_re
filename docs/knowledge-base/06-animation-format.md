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

每幀 **13-byte 標頭** + RLE 像素(第 3 輪反組譯 + 視覺驗證,**已完整破解**):

```
+0   uint16 LE  boundW    顯示 / 外框寬(同一動畫內固定)
+2   uint16 LE  boundH    顯示 / 外框高(逐幀微調,用於對齊)
+4   uint16 LE  = 0
+6   uint16 LE  = 2
+8   uint8      = 0
+9   uint16 LE  W         點陣解碼寬(realW)
+11  uint16 LE  H         點陣解碼高(realH)
+13  …          RLE 像素(解碼到 W×H)
```

> 解碼器(`FD2.EXE` `0x4F43D`)的呼叫端傳入 **frame+9**,故它 `lodsw` 讀到的正是 realW / realH,
> 再從 +13 解 RLE。前 9 byte(boundW, boundH, 0, 2, 0)是呼叫端用於畫面定位的 metadata。

**3-byte 迷你資源**(如 `FIGANI_002` = `00 00 0A`):動畫之間的群組分隔 / 索引標記,非動畫本體。

## 幀像素 codec(已完整破解)

從 `FD2.EXE` 反組譯出的 **sprite RLE**。解碼器家族落在 `0x4E000`–`0x4F800`
(以 `rep stosb`/`rep movsb` 叢集定位):`0x4EB52` 等為固定 24×24 版(地圖單位 sprite),
**`0x4F43D` 為參數化版**(用 `[0x27B4]` 每列重設寬、`[0x27B6]` 為列數)——FIGANI 戰鬥動畫即用此版。

文法(控制 byte `c`:高 2 bit = 模式,低 6 bit → `count = (c & 0x3F) + 1`):

```
00xxxxxx  色彩 run    讀 1 像素, 重複 count 次
01xxxxxx  dither/陰影  讀 1 像素, 輸出 [透明,值]×count(隔位寫, 佔 2×count 寬)— 地面陰影即此
10xxxxxx  literal     讀 count 個像素原樣
11xxxxxx  透明 skip    跳過 count(留底 = 透明)
每列以 bx=W 遞減追蹤;歸零換列(寫到螢幕 buffer 時 += stride−W)。
```

調色盤:FDOTHER 資源 #0;透明色 = index 0。

**驗證(視覺)**:`FIGANI_000` 解出 4 幀皆為「持劍騎士(藍灰盔甲 + 紅披風)」連續動作,
`FIGANI_001` 解出 11 幀完整揮劍攻擊(含黃色斬擊特效),地面 dither 陰影正確。
**全 `FIGANI.DAT`:264 個動畫、合計 2118 幀,全部可解。** 工具 `tools/decode_figani.py`
(`frames` 出 PNG 序列 / `gif` 出動畫 / `info` 印幀資訊)。

## 破解歷程(供方法論參考)

此 codec 是本專案最硬的一關,歷程值得留存:
1. 純資料靜態猜測(~8 種 RLE 假設)全失敗 → 確認「byte 消耗對齊 ≠ 解碼正確」需視覺驗證。
2. capstone(docker)反組譯,以 `rep stosb`/`rep movsb` 叢集定位解碼器家族。
3. 還原 24×24 版文法 → 套 FIGANI 仍橫條 → 找到參數化版 `0x4F43D`(讀 `[0x27B4]` 寬)。
4. 垂直相關分析發現真實寬 ≈103 而非標頭首欄 167 → 回頭解出 **13-byte 幀標頭**(realW/H 在 +9/+11)。
5. 從 +13、用 realW 解 RLE → 騎士 sprite 完美還原。

> 已推翻的舊假設(誠實揭露):`0xFE 為透明 escape`(實為控制 byte 高 2 bit=11)、
> `首欄 167 為解碼寬`(實為外框寬,真實寬在 +9)。

## 其餘待辦(後輪)
- 把 264 動畫對應到遊戲招式 / 角色(命名)。
- `ANI.DAT` 完整 AFM 檔格式(過場動畫)與 `FIGANI` 的關係。
- 調色 remap 表(部分 24×24 變體用 `[ebp+eax]` 重新著色,推測為陣營 / 受傷閃色)。

## 戰鬥動畫組成機制(FIGANI index,反組譯確認 2026-06-29)

codec 解碼只還原「幀→圖」;**哪個單位用哪個 FIGANI** 反組譯如下:
- 戰鬥演出載入碼 `0x287b5`:`movzx esi,[ebx+7]`(讀 `unit[+7]`),
  `0x2884c` `mov eax,esi; shl eax,2; sub eax,esi`(= esi×4−esi = **esi×3**)→ 組 FIGANI 資源 index;`inc` 後再載 `×3+1`。
- **FIGANI index = `unit[+7]` × 3**(+0/+1 兩攻擊動作幀組)。
- **`unit[+7]` = FDICON 地圖組號(恆等,我方敵方統一)**:同一 `unit[+7]` 經 `0x11019` ×12 算地圖 sprite、經 `0x2884c` ×3 算 FIGANI。
  - 索爾組0→FIGANI0、亞雷斯組4→12、**盜賊組96→FIGANI288**(實測 FIGANI_288/289=盜賊,對上 orig_05 守方)。
- **remake**:`figaniIndex(fig)=fig×3`,不需任何對應表(地圖組=FIGANI/3)。

> 教訓:做動畫前先 RE 組成機制,別 codec 解碼完就視覺猜 index(FIGANI_289=96×3+1 印證就是組×3)。

## 全螢幕戰鬥畫面組成(對照原版 orig_05 + RE,2026-06-29)

原版攻擊是**全螢幕戰鬥演出**(非地圖小格),畫面 320×200。組成:
- **背景 = BG.DAT 資源**(by 戰場;map0 第1章海邊 = BG_004 森林,320×100)。載入碼 0x28687(`call 0x22d1b` 載 BG.DAT);
  「戰場→BG index」的精確對應待續追(疑 by mapno/地形)。
- **守方在左、攻方在右下站橢圓土台**(景深:守方較小較高、攻方較大較低)。雙方全身 = FIGANI(角色組×3,本檔上節)。
- **狀態欄**:攻方右上、守方左下。深藍框+淺藍立體邊;名(左上)+ `LV‧NN`(右上)+ HP 黃條+數字 + MP 紅條+數字。
- 演出階段:windup(舉武器)→ swing(大白斬擊弧右上掃左)→ impact(守方整體閃紅 + HP 條即時抽乾)→ standoff。

> remake `drawBattleScene`:BG 背景 + 守/攻 FIGANI(不同 sc/y 景深)+ drawBattlePanel 狀態欄。對照圖 orig_05_attack_01~05。
