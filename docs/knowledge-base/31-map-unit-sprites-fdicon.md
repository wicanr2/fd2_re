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
- `tools/export_units.py`:寫 `fig` 欄位(sprite組)。**fig = 角色 id(恆等,§6);敵方/我方皆同**。

## 5. remake 接法

- 引擎 `loadSprites()` 載 `fig_<grp>_f*.png` 分組;`drawUnitSprite()` 用 `(g.frame/12)%3` 循環待機幀,**24×24 直貼格**(略上移讓單位站在格上),陣營色腳標 + HP bar,已行動套灰(對映原版動作狀態 AA `+0x0D`=0x80 行動完畢,§6 / doc 27)。
- 原版實機截圖(`real_pic/`)+ DATO face 當 oracle 佐證:角色 id = sprite組 恆等(0→0;敵方 id 68士兵/76援軍/96盜賊/97頭目/103獸人)。

## 6. sprite index 公式(已驗證)+ 角色 id = 肖像 = sprite組(memory.md 權威,恆等)

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

> **mapping = 角色身份恆等(青衫 memory.md 權威證實 2026-06-28)**:
> 我方 32 角色:**角色 index = 肖像(FA)= sprite組(Z1)= 角色名,基本態三者恆等**。
> 依據:青衫 memory.md「肖像編號 0x00–0x1F」對上角色表 `[0x55BA1]` 的 index 0–31,**32 角色職業全部吻合**
> (索爾0劍士 … 蘭斯洛特7聖騎士 希莉亞8弓兵 悠妮9法師 瑪琳10/索菲亞11僧侶 凱麗12武者 珊14法師
> 凱拉斯16/米亞斯多德17龍劍士 蜜蒂18/羅德曼19劍聖 約拿21聖者 蓋亞30/渥德31機兵)。
> **敵方/通用** id>31:肖像/sprite組另排但仍恆等(士兵68、盜賊96、頭目97)。
> **轉職**:角色換成轉職態肖像編號(memory.md 0x20–0x41,如索爾→劍聖0x20→英雄0x32、亞雷斯→聖騎士0x24→龍騎士0x36);sprite組是否隨之切到另一組待確認。
>
> ⚠ **前一版作廢的錯誤斷言**:「龍人系打破恆等」「凱拉斯=portrait67 / sprite組17 / 轉職組49」「轉職當機」── 全部來自**我誤判 DATO_067 是凱拉斯 + 用 index17 循環論證**。memory.md 證明凱拉斯是 **id16**(肖像16=sprite組16=icon_192,放大確認龍人戰士),三者恆等,無跳號。
>
> remake:**fig = 角色 id(恆等)**;只有上戰場角色有 sprite 組。

| 欄位(memory.md 80B 單位結構) | 意義 | offset |
|---|---|---|
| Z1 = sprite組 | FDICON 組(= 角色 id,恆等) | `+0x0A` |
| Z2 = 方向 | 0=下 1=左 2=上 3=右 | `+0x0B` |
| Z3 = 跑步動作幀 | 待機/走 3 幀(手擺) | `+0x0C` |
| AA = 動作狀態 | 00=未行動 01=死亡 80=行動完畢 | `+0x0D` |
| BB = 陣營 | 00=敵 01=友 02=己 | `+0x0E` |
| FA = 肖像 | DATO 對話頭像 id(= 角色 id,恆等) | `+0x0F` |
| index | `組×12 + 方向×3 + 幀` | 繪製碼 0x1291e |

> 反組譯一致性:繪製碼 0x12831 `movzx edi,byte[eax+2]`(eax = 單位 base+8)即讀 `+0x0A` 的 Z1,與 memory.md 對齊。

## 7. sprite & face 統一角色系統(remake 加新人的基礎)

remake 把角色做成**單一資料表**(同一角色 id 對應 face 與 sprite,基本態恆等),加新人加一筆:

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

- **[已定論] 角色 id = 肖像(FA)= sprite組(Z1)= 角色名,基本態恆等(青衫 memory.md 權威 + 反組譯交叉驗證)**:
  - **memory.md(青衫攻略)** 給出三張權威表:① 單位 80B 結構(Z1 圖形 `+0x0A`、FA 肖像 `+0x0F` 等,見 §6 表)② 肖像編號 0x00–0x1F = 32 我方角色名(+ 0x20–0x41 轉職態)③ 職業編號 00–19。
  - **角色表 `[0x55BA1]`**(32 槽 × 24B:byte0=RA 種族、1=CL 職業、2=LV、3=baseHP;`0x300`=32×24 整除)的 index 0–31,**CL 職業逐一對上 memory.md 角色名**(32/32 吻合)→ 確認 **index = 肖像 = 角色**。
  - sprite組(Z1)在前 9(0–9 索爾→悠妮)、龍人系(16 凱拉斯/17 米亞斯多德,放大 icon_192/204 皆龍人戰士)均 = 角色 id → **sprite組 = 角色 id(恆等)**。
  - **敵方/通用**:id>31,肖像/sprite組另排但仍恆等(士兵68、盜賊96、頭目97)。
  - **轉職**:角色換轉職態肖像(memory.md 0x20–0x41);sprite組是否隨之切組待確認。
  - 產物:`docs/data/exe_tables/characters.json`(32 角色 id/sprite_group/face_portrait/race/cls/lv/baseHP + 全名),三者恆等,remake 加新人即用此表。
  - **職業編號表(memory.md,hex)**:00龍 01劍士 02戰士 03騎士 04弓兵 05法師 06僧侶 07盜賊 08武者 09劍聖 0A聖戰士 0B聖騎士 0C狙擊手 0D大法師 0E祭師 0F龍劍士 10鬥士 11英雄 12魔戰士 13龍騎士 14神射手 15召喚師 16聖者 17忍者 18武聖 19機兵。
  - ⚠ **作廢的錯誤鏈(compact 防呆)**:曾依「`unit[+7]` 非 portrait 公式」「sprite組 = 角色定義 index 而非 portrait」「龍人打破恆等」「凱拉斯 portrait67 / sprite組17 / 轉職組49 / 轉職當機」推演 → **全錯**,根因是我把 DATO_067 誤當凱拉斯 + 用 index17 循環論證。**正解:凱拉斯 = id16,三者恆等,無跳號、無「組49」、攻略無當機明文。**
  - 注:roster 26B(modify2 §7)無獨立 sprite組欄,正因 sprite組 = 角色 id 本身,不需額外欄位。
- **[線索] 廢案人物**:FDICON 有些組**沒畫滿 12 格**(未採用角色,僅部分方向/幀);因 sprite 用「組×12 + 方向×3 + 幀」定位,廢案組仍佔 12 格 stride(部分空/重複)。未來可挖廢案角色來用(加新人素材庫)。
- **[M2 待做]** 對話框**嘴型動畫**:DATO_N 的 m0~m3 對話時播放(嘴開合 + 眨眼)。哪幀=閉嘴/開嘴/眨眼、播放節奏(隨文字推進?固定循環?)待反組譯文字渲染器(0x16D00 區,doc 14)確認;M2 對話層實作。
- 方向:目前只導「面向下」待機;4 方向(走動/面敵)待加。
- 戰鬥演出切 FIGANI 大圖:M1 戰鬥動畫階段再接。

> 相關:doc 10(sprite 繪製/陣營著色)· doc 06(FIGANI 動畫)· doc 27(byte[+5] 狀態旗標)· doc 30(工作拆解)。工具:`tools/decode_fdicon.py`、`export_sprites.py`。素材:`extracted/fdicon/`(本機,1680 PNG)。
