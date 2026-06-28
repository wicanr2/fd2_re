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

## 6. sprite index 公式(已驗證)+ portrait→sprite組(恆等為主,龍人/轉職系跳號)

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

> **mapping = 恆等為主 + 少數跳號(使用者校正 2026-06-28)**:
> 絕大多數 **portrait → 組 恆等**(已確認:0–9、68 士兵→組68、96 盜賊→組96、97 頭目→組97)。
> **少數「龍人系」跳號 + 轉職切組**:**portrait 67 = 凱拉斯(龍劍士):sprite 組17(轉職前,icon_204)→ 組49(轉職後,icon_588)**,兩組都龍人外型。即龍劍士不走恆等(67≠組67),且**轉職會切換 sprite 組**。
> face(FA,`unit[+1]`=DATO 肖像)與 sprite組(Z1,`unit[+2]`,經 0x11019 由 `unit[+7]`)是獨立欄位;恆等只是兩者編號大多同步,龍人系打破同步。
> remake:**fig=portrait 為主 + 少數例外覆蓋表**(67→49…);只有上戰場角色有 sprite 組。
>
> **轉職切換 sprite 組(反組譯實證)**:凱拉斯 組17→49 證明轉職會換 sprite 組。**推論**:若轉職後職業的組沒畫(FDICON 廢案/空組),可能讀不到 sprite。
> ⚠ **「轉職會當機」是使用者口述,青衫攻略查無明文**(grep 全攻略無「當機/死機/crash」),後果(當機 or 顯示異常)**待實機驗證**。remake 仍應為可轉職職業備齊 sprite。

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

- **[已鎖定] sprite組機制 = 單位結構欄位 `unit[+7]`(非 portrait 公式)**:
  - 反組譯單位建立(0x10a5b 迴圈)+ `0x11019`:`call 0x11019(FDICON_buf, unit[+7])`,內部 `esi=[esp+0x4c]=unit[+7]`,`imul esi,0xc`(組×12)載入該組 12 幀;回傳 `edx`→`unit[+2]`(繪製用組,0x12831)。
  - **sprite組 id 是每單位/角色預存的欄位 `unit[+7]`**,從出場模板 `[0x53bf7]` memcpy,**不是從 portrait 算**。
  - 故「恆等」(0–9/68/96/97)只是**角色定義時 sprite組 剛好=portrait** 的巧合;**龍人系不同步**:凱拉斯 portrait67 但 sprite組=17(轉職前)/49(轉職後)。
  - map0 roster 26B 無 sprite組 byte(它在出場模板/角色定義層,非 FDFIELD roster);搜不到「portrait→組」連續表也因此(根本沒這種表)。
  - **remake 對映**:每角色 characters.json 獨立存 `sprite_group`(= 原版 unit[+7]),與 portrait/face 分開;值從「視覺對照 + 已知點(0-9恆等、67→17/49)」或續追出場模板 `[0x53bf7]` 填值處建表。
  - **轉職**:改 `unit[+7]`(切 sprite 組);凱拉斯 17→49 即一例。
  - **青衫 modify 查證(2026-06-28)**:roster 26B(modify2 §7)**無 sprite組欄**;角色屬性表 `[0x55BA1]`(24B/人物,RA/CL/LV/HP/MP/MV/法術/裝備/AP/DP/DX)byte 0–4 也非 sprite組(值非 0,1,2…序列)。→ **sprite組 不在任何靜態表**,是出場模板 `[0x53bf7]` 建立時由程式邏輯(by 肖像/職業 + 龍人系特例 + 轉職)填的 runtime 值。
  - **實用結論**:精確 portrait↔sprite組 對照用「**視覺對照 + 已知點**」建表最快(0–9/68/96/97 恆等、67 凱拉斯→17/49…);完整反組譯填值邏輯(出場模板建立鏈)留待需要時。
  - **肖像→角色名補充**(modify1 §8):12=蜜蒂、13=羅德曼、22=劍聖鐵諾(轉職後肖像也變)。
- **[線索] 廢案人物**:FDICON 有些組**沒畫滿 12 格**(未採用角色,僅部分方向/幀);因 sprite 用「組×12 + 方向×3 + 幀」定位,廢案組仍佔 12 格 stride(部分空/重複)。未來可挖廢案角色來用(加新人素材庫)。
- **[M2 待做]** 對話框**嘴型動畫**:DATO_N 的 m0~m3 對話時播放(嘴開合 + 眨眼)。哪幀=閉嘴/開嘴/眨眼、播放節奏(隨文字推進?固定循環?)待反組譯文字渲染器(0x16D00 區,doc 14)確認;M2 對話層實作。
- 方向:目前只導「面向下」待機;4 方向(走動/面敵)待加。
- 戰鬥演出切 FIGANI 大圖:M1 戰鬥動畫階段再接。

> 相關:doc 10(sprite 繪製/陣營著色)· doc 06(FIGANI 動畫)· doc 27(byte[+5] 狀態旗標)· doc 30(工作拆解)。工具:`tools/decode_fdicon.py`、`export_sprites.py`。素材:`extracted/fdicon/`(本機,1680 PNG)。
