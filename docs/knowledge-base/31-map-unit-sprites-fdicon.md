# 31 — 地圖單位 Sprite:FDICON Q 版小人 + 待機動畫

> 戰場地圖上的單位(原版那種 Q 版大頭小人)= **`FDICON.B24`** —— 1680 個 24×24 sprite。
> 這跟 `FIGANI`(戰鬥演出的全身大圖)是兩套東西:**地圖走 FDICON,戰鬥動畫走 FIGANI**。
> 本篇記錄 FDICON 格式、分組、解碼、與 remake 接法。用**原版實機截圖當 oracle**(rulebook 64)驗證。

## 0. 一個差點殘留的誤判(教訓)

`FDICON.B24`(624010 bytes,無 `LLLLLL` 外殼)早先用 FDSHAP 的 **bg-RLE** 解 → 全是橫條亂圖,於是一度想把「1680 個 24×24」這個斷言改成「待確認」。
**但斷言是對的,錯的是解碼方法**:FDICON 的 tile 是**含透明的 sprite,要用 sprite 4-mode RLE**(FIGANI 那套),不是 bg-RLE。換對解碼器立刻解出 Q 版小人。
→ 教訓:**解碼失敗 ≠ 斷言錯,先換解碼器/方法再質疑事實**(rulebook 62/63)。

## 1. 格式

```
+0  u16 tileW   = 0x18 (24)
+2  u16 tileH   = 0x18 (24)
+4  u16 count   = 0x0690 (1680)
+6  u32[count]  offset 表(相對檔頭)
各 tile:sprite 4-mode RLE(高 2 bit=模式:色run/dither/literal/透明;低 6 bit=count−1)
        透明 = index 0
```
header 與 FDSHAP tileset 同骨架(尺寸+count+offset 表),**差別在 tile 的 RLE**:FDSHAP 地形用 bg-RLE(不透明),FDICON 單位用 sprite-RLE(有透明背景)。

## 2. 分組:每角色 12 sprite = 4 方向 × 3 待機幀 [驗]

實測組 0(index 0–11):
```
 0  1  2   面向【下】3 幀(站 / 抬左手 / 抬右手)
 3  4  5   面向【左】3 幀
 6  7  8   面向【上】3 幀(背面)
 9 10 11   面向【右】3 幀
```
**3 幀循環 = 待機時手腳微擺的動感**(使用者指出的「手會左右移動」)。
角色組 = `index // 12`。已辨識:組 0=紅帽主角、1=藍帽、2=灰甲機器人、9(108–119)=紅髮主角、8(96–107)=綠衣盜賊…(共約 140 組,涵蓋全角色 + 敵兵 + 怪物 + 機器人)。

## 3. FDICON(地圖) vs FIGANI(戰鬥)

| | FDICON.B24 | FIGANI.DAT |
|---|---|---|
| 用途 | **地圖上的單位小人** | 戰鬥演出(攻擊/受擊)全身動畫 |
| 尺寸 | 24×24(正好一格) | 80–175(大圖) |
| 風格 | Q 版大頭 | 寫實全身 |
| 數量 | 1680(≈140 組×12) | 264 動畫 / 2118 幀 |
| codec | sprite 4-mode RLE | 同 codec(參數化 0x4F43D) |

> doc 10 提的「24×24 場景單位解碼器 0x4EB52」即對應 FDICON;FIGANI 用 0x4F43D。地圖顯示 FDICON 小人,選單位/進入戰鬥才切 FIGANI 大圖。

## 4. 工具

- `tools/decode_fdicon.py`:解全 1680 sprite(sprite-RLE,index 0 透明)→ 透明 PNG;`--overview` 出標 index 的總覽(看分組)。
- `tools/export_sprites.py`:對指定**角色組**導出「面向下」3 待機幀 → `remake/assets/sprites/fig_<grp>_f<0..2>.png`。
- `tools/export_units.py`:寫 `fig` 欄位(sprite組)。**注意:fig≠portrait,需 portrait→組 映射表(§6,反組譯中);目前暫填 portrait 為佔位,待映射鎖定後修正**。

## 5. remake 接法

- 引擎 `loadSprites()` 載 `fig_<grp>_f*.png` 分組;`drawUnitSprite()` 用 `(g.frame/12)%3` 循環待機幀,**24×24 直貼格**(略上移讓單位站在格上),陣營色腳標 + HP bar,已行動套灰(對映原版 byte[+5] bit7,doc 27)。
- 原版實機截圖(`real_pic/`)+ DATO face 當 oracle 校 portrait→組 映射(已知 0→0、67→17;序章敵 portrait 76士兵援軍/96盜賊/97盜賊頭目/103獸人)。

## 6. sprite index 公式(已驗證)+ portrait→sprite組 映射(修正:非恆等)

不靠猜測,**反組譯戰場單位繪製碼(0x128e0–0x12932)鎖死了公式**:

```asm
0x12823  mov  eax,[0x53a45]        ; 單位陣列基底
0x12831  movzx edi, byte[eax+2]    ; 組 = 單位欄位 +2(= 角色 id)
0x12835  movzx esi, byte[eax+3]    ; 方向(0..3)
0x1291e  imul edi, edi, 0xc        ; 組 × 12
0x12921  mov eax,esi; shl 2; sub esi   ; 方向 × 3
0x12928  add eax, edi              ; 組×12 + 方向×3
0x1292a  add eax, edx              ; + 幀(unit+0x26 動作)
0x1292c  mov edx,[0x53a61]         ; FDICON sprite 指標表
0x12932  mov eax,[edx + eax*4]     ; sprite[index]
```

→ **FDICON sprite index = 組 × 12 + 方向 × 3 + 幀**(公式已驗證),組 = `unit[+2]`(經 `call 0x11019` 從 `unit[+7]` 決定)。

> **⚠ 修正(使用者抓到,2026-06-28):portrait(face)≠ sprite組,非恆等!**
> 早先「組==portrait」是前幾個巧合(portrait 0 索爾→組 0);**但 portrait 67(龍人戰士)→ 組 17**(icon_204–215),不是組 67。
> **face(FA)與 sprite組(Z1)是兩個獨立欄位**:face=`unit[+1]`(DATO 肖像 id)、sprite組=`unit[+2]`(經 0x11019 由 `unit[+7]` 算/載入)。
> **portrait→sprite組 是映射(非恆等),機制反組譯進行中**(0x11019 載入該組12幀;portrait→組 的查表/算法待鎖)。
> 故 remake 需 **portrait→組 對應表**;且**只有上戰場的角色有 sprite 組**,純劇情角色只有 face。已知點:portrait 0→組0、67→組17。

| 欄位 | 意義 | 來源 |
|---|---|---|
| 組(sprite組,Z1) | FDICON 組(**≠** portrait/face) | `unit[+2]`(經 0x11019 由 unit[+7]) |
| face/portrait(FA) | DATO 肖像 id | `unit[+1]` |
| 方向 | 0=下 1=左 2=上 3=右 | `unit[+3]` |
| 幀 | 待機/走 3 幀(手擺) | `unit[+0x26]` 動作 |
| index | `組×12 + 方向×3 + 幀` | 繪製碼 0x1291e |

## 7. sprite & face 統一角色系統(remake 加新人的基礎)

remake 把角色做成**單一資料表**(face 與 sprite 是兩個欄位,經 portrait→組 映射關聯),加新人加一筆:

```jsonc
// characters.json — 角色 id → 頭像 + 地圖 sprite + 數值
{
  "0":   { "name":"索爾",  "face":"dato/000_m*"   /* 4 嘴型幀 */, "sprite":"fdicon/grp000", "stats":{...} },
  "3":   { "name":"哈瓦特", "face":"dato/003_m*", "sprite":"fdicon/grp003" },
  "68":  { "name":"一般士兵","face":"dato/068_m*", "sprite":"fdicon/grp068" },
  // …原版 id 0–136 = 原版角色(DATO 頭像 137 / FDICON 組 140)
  "200": { "name":"自創英雄","face":"custom/hero_m*" /* 自繪也要 4 嘴型 */,"sprite":"custom/hero_grp", "new":true }
}
```
- **face**:對話頭像 = DATO_N 的 **4 嘴型幀**(`DATO_N_m0~m3`,80×80,本機 `extracted/portraits/`)。
  **對話時循環播放做嘴巴開合 + 眨眼,不是單張靜圖**(漢堂讓對話有生氣的手法)。characters.json 的 `face` 指向「一組 4 幀」。
- **sprite**:地圖 12 幀(FDICON 組 N=4方向×3幀;或自繪同規格)

> 對映關係:**同一角色 id N → DATO_N(4 嘴型 face)+ FDICON 組 N(12 地圖 sprite)**。
> 加新人時兩者都要備齊同 id:4 張嘴型頭像 + 12 幀地圖 sprite。
- **加新人**:分配未用 id(≥137)、給 face PNG + 12 幀 sprite + 數值 → 引擎自動吃,事件/招募(doc 26/28 `roster_has`)直接用該 id。
- **角色總覽**:`tools/char_summary.py` → 本機 character_summary.png(140 組 sprite+face 並排,統一編號全圖佐證、加新人看缺號)。
- 工具:`decode_fdicon.py`(導原版組)、`decode_dato.py`(導原版頭像)、`export_units.py`(`fig=portrait`)已就緒;未來補 `gen_characters.py` 從 DATO+FDICON 自動生成 characters.json。

→ 這把「炎龍 remake」從「複刻」升級成**可擴角色的平台**:配合可擴展事件系統(doc 29),能做原版沒有的角色 + 劇情 + 戰役。

## 8. 受阻 / 待校

- **[進行中] portrait→sprite組 映射機制**,反組譯已縮小但未鎖定:
  - sprite組 = `unit[+2]` = `call 0x11019(unit[+7])` 的回傳;`unit[+7]` 從**我方名冊**(`[0x53bf7]`,存檔載入)memcpy,敵方從 roster。
  - `0x11019` = 「載入第 N 組的 12 幀」(`imul ×0xc`),**輸入本身就是 sprite組 id**,非 portrait→組 查表。
  - map0 roster 26B 確認**無 sprite組 byte**(全 portrait96/race1/cls1 + 物品法術填充)→ sprite組不在 roster。
  - 故 mapping 在更深的「`unit[+7]` 怎麼得到組 id」:我方從名冊 `[0x53bf7]`(存檔初始隊伍)、敵方從某處 by portrait/race-cls;**單位建立鏈需專門一輪靜追**(本輪繞太久,先停)。
  - 已知映射點:portrait 0–9 → 組 0–9(恆等)、**portrait 67(龍人)→ 組 17**(非恆等)。
  - **搜尋失敗**:EXE data 段找不到「byte 表 / struct 表」使 [0..9]=0..9 且 [67]=17 → mapping **不是單一連續表**,而是存在「角色/單位定義」(名冊/roster 的獨立欄位)。
  - **下一步**:反組譯 `unit[+7]` 的最終來源 ——(a) 我方:存檔/初始隊伍表怎麼填 sprite組;(b) 敵方:roster 26B 哪個 byte = sprite組。鎖定後即得完整 portrait↔組 對照。
- **[線索] 廢案人物**:FDICON 有些組**沒畫滿 12 格**(未採用角色,僅部分方向/幀);因 sprite 用「組×12 + 方向×3 + 幀」定位,廢案組仍佔 12 格 stride(部分空/重複)。未來可挖廢案角色來用(加新人素材庫)。
- **[M2 待做]** 對話框**嘴型動畫**:DATO_N 的 m0~m3 對話時播放(嘴開合 + 眨眼)。哪幀=閉嘴/開嘴/眨眼、播放節奏(隨文字推進?固定循環?)待反組譯文字渲染器(0x16D00 區,doc 14)確認;M2 對話層實作。
- 方向:目前只導「面向下」待機;4 方向(走動/面敵)待加。
- 戰鬥演出切 FIGANI 大圖:M1 戰鬥動畫階段再接。

> 相關:doc 10(sprite 繪製/陣營著色)· doc 06(FIGANI 動畫)· doc 27(byte[+5] 狀態旗標)· doc 30(工作拆解)。工具:`tools/decode_fdicon.py`、`export_sprites.py`。素材:`extracted/fdicon/`(本機,1680 PNG)。
