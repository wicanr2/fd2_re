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
+0   int16 LE   dx        該幀「絕對螢幕 X」(320×200 系統)
+2   int16 LE   dy        該幀「絕對螢幕 Y」
+4   uint16 LE  = 0
+6   uint16 LE  = 2
+8   uint8      = 0
+9   uint16 LE  W         點陣解碼寬(realW)
+11  uint16 LE  H         點陣解碼高(realH)
+13  …          RLE 像素(解碼到 W×H)
```

> 解碼器(`FD2.EXE` `0x4F43D`)的呼叫端傳入 **frame+9**,故它 `lodsw` 讀到的正是 realW / realH,
> 再從 +13 解 RLE。
> ⚠ **修正舊錯誤標註「+0/+2 = boundW/boundH 外框寬高」**:實為**每幀絕對螢幕座標 (dx,dy)**
> (0x2935b 讀 word[+0]/[+2] 當幀位移,doc35;實證:FIGANI_013 f01=(141,3)、FIGANI_288 f00=(16,41)
> 與 orig 截圖模板匹配的 sprite 落點**完全一致**)。**戰鬥演出的走位/伸擊/突刺全靠逐幀 (dx,dy) 變化**
> (如 013:f11 劈擊 dx=89 伸向左、f12-14 突刺 dy 38→63),引擎只要每幀貼在 (dx,dy),不需任何錨點/位移計算。
> remake 資產管線須把 (dx,dy) 一併導出(`remake/assets/figani/meta.json`),別只導像素。

### 2026-08-09：戰鬥呈現開始消費原始幀延遲（E1）

`remake/assets/figani/delays.json` 是從固定版本的 `FIGANI.DAT`（雜湊見
[`fd2-reference-files.json`](../data/fd2-reference-files.json)）逐幀擷取的 descriptor
`+6` 值，現有 22 個已匯出 PNG 動畫逐幀對齊。`cmd/fd2` 的回歸會在玩家提供原始
封存檔時逐筆比對延遲與幀數；缺檔只跳過該項原版資產比對，不把別的版本資料當成
相同位址的事實。匯出工具是版控中的 `tools/decode_figani.py`（Python `struct`，
以每個資源相對位址的 descriptor `+6` 讀取）；這是資源內偏移，不是 `FD2.EXE`
線性位址，也不取代官方 IDA 對 `0x2b9a1` 的控制流程證據。

`internal/figani.DisplayScheduler` 只把原始延遲轉成顯示時脈，並保留明示的
`FD2_BATTLE_FPT` 速度倍率。全螢幕攻擊演出現在要求 PNG 幀數與延遲表一一配對；
缺配對即不建立演出，不再以固定 15 幀補猜。這一刀只關閉幀選擇與停留時間的
呈現邊界，不證明命中幀、傷害、音效、台座或整個原版畫面已達 E2。

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
- `unit[+7]` 是這條**戰鬥 FIGANI** selector 的 raw input；它不可和地圖 FDICON selector 混稱。後者已由
  `0x127e0` 直接證實使用 `unit[+2]×12`。已盤點 roster 的兩個 visual id 可相同，但這不使 ABI 欄位等價。
  - 索爾的已驗證 battle value 0→FIGANI0、亞雷斯4→12、**盜賊96→FIGANI288**(實測 FIGANI_288/289=盜賊,對上 orig_05 守方)。
- **remake**:`battle_fig` 現保存已驗證的 FDFIELD roster `b1→unit+7` input，`figaniIndex` 只對它做 `×3`；
  舊 JSON 缺欄才顯式 fallback legacy `fig`。現有 `fig` 仍只是 map `unit+2` compatibility approximation；兩欄
  可不同，不能用「地圖組=FIGANI/3」作全域對應表。

> 教訓:做動畫前先 RE 組成機制,別 codec 解碼完就視覺猜 index(FIGANI_289=96×3+1 印證就是組×3)。

## 全螢幕戰鬥演出 → 已驗證邊界見 [doc 35](35-battle-animation-rendering.md)

原版攻擊是全螢幕戰鬥演出(320×200,非地圖小格)。已釘住的部分包含攻守雙圖演出主函式 `0x28a6c`、four-mode blit `0x4e63d`、phase `[0x540ff]`、戰場→BG 原始 selector 與狀態欄座標器 `0x2a289→0x18c6d`。**撤回**「繪圖機制已完整反組譯」與「狀態欄是 `0x29164`」：`0x29164` 是 figure + TAI.DAT 台座淡入；command presentation、部分 BG layer schedule、SFX、多段命中仍待，見 doc35。
- **關鍵**:blit **無 runtime 縮放**——守方小/攻方大是 **FIGANI 美術本身畫成不同尺寸**(景深燒進素材),不是 scale。`0x2935b` 已證實每幀位移來自 descriptor header 的 `(dx,dy)`；(164,157) 是特定 caller 的固定 anchor。**不可再把 `unit+0x40` 當 figure X**，該 word 是 current HP。
- figure／台座淡入會使用 **VGA DAC 色盤操作**；戰鬥命中分支另受 raw frame
  flag、傷害步進與 `0x29f72` 輸出欄位條件控制，不能簡化成無條件全畫面紅罩。
  (`0x11d40`，ports `0x3c8/0x3c9`)；HP／MP 條則由
  `0x18795→0x17d6f` 依目前值逐欄繪製，不可再稱為色盤抽乾。
