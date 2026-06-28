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
- `tools/export_units.py`:`portrait → 角色組` 對應(`PORTRAIT_TO_GROUP`),寫進 units.json 的 `fig` 欄位。

## 5. remake 接法

- 引擎 `loadSprites()` 載 `fig_<grp>_f*.png` 分組;`drawUnitSprite()` 用 `(g.frame/12)%3` 循環待機幀,**24×24 直貼格**(略上移讓單位站在格上),陣營色腳標 + HP bar,已行動套灰(對映原版 byte[+5] bit7,doc 27)。
- 原版實機截圖(`real_pic/`)當 oracle:序章我方紅帽/藍帽/紅髮、左上機器人砲台、下方海盜兵、右側綠敵 —— 用來校正 `portrait → 組` 對應。

## 6. ★ 重大突破:sprite index 公式 + 統一角色編號(反組譯確認)

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

→ **FDICON sprite index = 組 × 12 + 方向 × 3 + 幀**,組 = `unit[+2]`(角色 id)。
配合視覺 11/11(索爾/哈諾/鐵諾/士兵/盜賊/豹人/騎士/暗戰士全對),確認:

> **統一角色編號:角色 id N → DATO_N(頭像/face)+ FDICON 組 N(地圖 sprite,12 幀)。**
> 漢堂用**一個 id 同時綁頭像與地圖 sprite**(省管理)。故 remake 對應 = **`fig = portrait`(恆等)**,不需對應表。

| 欄位 | 意義 | 來源 |
|---|---|---|
| 組(角色 id) | FDICON 組 = DATO portrait | `unit[+2]`(= roster 肖像 id) |
| 方向 | 0=下 1=左 2=上 3=右 | `unit[+3]` |
| 幀 | 待機/走 3 幀(手擺) | `unit[+0x26]` 動作 |
| index | `組×12 + 方向×3 + 幀` | 繪製碼 0x1291e |

## 7. sprite & face 統一角色系統(remake 加新人的基礎)

既然原版「id → face + sprite」統一,remake 把角色做成**單一資料表**,加新人只要加一筆:

```jsonc
// characters.json — 角色 id → 頭像 + 地圖 sprite + 數值
{
  "0":   { "name":"索爾",  "face":"dato/000.png", "sprite":"fdicon/grp000", "stats":{...} },
  "3":   { "name":"哈瓦特", "face":"dato/003.png", "sprite":"fdicon/grp003" },
  "68":  { "name":"一般士兵","face":"dato/068.png", "sprite":"fdicon/grp068" },
  // …原版 id 0–136 = 原版角色(DATO 頭像 137 / FDICON 組 140)
  "200": { "name":"自創英雄","face":"custom/hero.png","sprite":"custom/hero_grp", "new":true }
}
```
- **face**:對話頭像(DATO_N,4 嘴型;或自繪)
- **sprite**:地圖 12 幀(FDICON 組 N=4方向×3幀;或自繪同規格)
- **加新人**:分配未用 id(≥137)、給 face PNG + 12 幀 sprite + 數值 → 引擎自動吃,事件/招募(doc 26/28 `roster_has`)直接用該 id。
- 工具:`decode_fdicon.py`(導原版組)、`decode_dato.py`(導原版頭像)、`export_units.py`(`fig=portrait`)已就緒;未來補 `gen_characters.py` 從 DATO+FDICON 自動生成 characters.json。

→ 這把「炎龍 remake」從「複刻」升級成**可擴角色的平台**:配合可擴展事件系統(doc 29),能做原版沒有的角色 + 劇情 + 戰役。

## 8. 受阻 / 待校

- **[已解]** ~~portrait→組對應~~ → 反組譯確認 **fig = portrait**(§6),組==DATO==FDICON 統一編號。
- 方向:目前只導「面向下」待機;4 方向(走動/面敵)待加。
- 戰鬥演出切 FIGANI 大圖:M1 戰鬥動畫階段再接。

> 相關:doc 10(sprite 繪製/陣營著色)· doc 06(FIGANI 動畫)· doc 27(byte[+5] 狀態旗標)· doc 30(工作拆解)。工具:`tools/decode_fdicon.py`、`export_sprites.py`。素材:`extracted/fdicon/`(本機,1680 PNG)。
