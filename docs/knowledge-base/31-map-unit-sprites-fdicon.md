# 31 — 地圖單位 Sprite:FDICON Q 版小人 + 待機動畫

> 戰場地圖上的單位(原版那種 Q 版大頭小人)= **`FDICON.B24`** —— 1680 個 24×24 sprite。
> 這跟 `FIGANI`(戰鬥演出的全身大圖)是兩套東西:**地圖走 FDICON,戰鬥動畫走 FIGANI**。
> 本篇記錄 FDICON 格式、分組、解碼、與 remake 接法。用**原版實機截圖當 oracle**(rulebook 64)驗證。

## 0. 一個差點殘留的誤判(教訓)

`FDICON.B24`(624010 bytes,無 `LLLLLL` 外殼)早先被誤用「不透明 bg-RLE」模型解 → 全是橫條亂圖,於是一度想把「1680 個 24×24」這個斷言改成「待確認」。
**但斷言是對的,錯的是解碼方法**:FDICON 與 FDSHAP 都使用 native four-mode RLE ABI；FDICON 是含透明的單位 sprite，FDSHAP 也可含 mode-3 span，但原版 renderer 對兩者的 raw/LUT 分支不同。換對 four-mode 解碼器立刻解出 Q 版小人。
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
header 與 FDSHAP tileset 同骨架(尺寸+count+offset 表)，且兩者都可用同一 four-mode RLE ABI 解讀；差別是資產用途與 renderer branch：FDICON 走 unit raw/palette-band，FDSHAP 依 FDFIELD entry 可走 raw 或 destination-LUT compositor。

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
- `tools/export_units.py`:保留 legacy `fig` compatibility 欄位。它不是已閉合的 FDICON selector source；真正的
  runtime `unit+2` producer 仍在追查。

## 5. remake 接法

- 引擎 `loadSprites()` 載 `fig_<grp>_f*.png` 分組;`drawUnitSprite()` 用 `(g.frame/12)%3` 循環待機幀,**24×24 直貼格**(略上移讓單位站在格上),陣營色腳標 + HP bar,已行動套灰（對映原版已驗證 record `+5 bit7=0x80`，不是舊 AA `+0x0D` 標記；§6 / doc 27）。
- 原版實機截圖與 DATO face 可做單一角色素材的 oracle，但不能單獨證明 runtime map selector。battle FIGANI
  走另一條已閉合的 `unit+7×3` 路徑（doc06）。

## 6. sprite index 公式（已驗證）；raw key 來源已勘誤

不靠猜測,**反組譯戰場單位繪製碼(0x128e0–0x12932)鎖死了公式**:

```asm
0x12823  mov  eax,[0x53a45]        ; 單位陣列基底
0x12831  movzx edi, byte[eax+2]    ; raw map selector
0x12835  movzx esi, byte[eax+3]    ; 方向(0..3)
0x1291e  imul edi, edi, 0xc        ; 組 × 12
0x12921  mov eax,esi; shl 2; sub esi   ; 方向 × 3
0x12928  add eax, edi              ; 組×12 + 方向×3
0x1292a  add eax, edx              ; + cycle（由全域 idle/moving phase 選出）
0x1292c  mov edx,[0x53a61]         ; FDICON sprite 指標表
0x12932  mov eax,[edx + eax*4]     ; sprite[index]
```

→ **FDICON sprite index = slot × 12 + 方向 × 3 + cycle**（公式已驗證）；slot 為 `unit[+2]`。第三項不是 runtime `+4` 或 `+0x26`：`+4` 是沿方向的次格 placement offset；`+4==0` 選 global idle phase `0x3c0b`、非零選 moving phase `0x3c07`，phase 3 會正規化為 1，而 `+0x26!=0` 只強制 cycle 0 並加上已證實的全域繪製偏移。IDA Pro 9.4 的完整 constructor trace 現已證實 `0x10c50→0x11019` 以 FDFIELD **b1** 查全域 raw-key table；只有新 key 才用 caller archive pointer 建十二指標 block，回傳 cache slot 寫入 `unit+2`。它不是直接 copy 的角色／肖像 byte。

> **撤回全域 identity assertion（2026-07-26；raw key 來源於 2026-08-10 勘誤）**：角色表、DATO、FDICON 素材與若干玩家 roster 的數值相同，
> 只能作為素材觀察，不能證明 `unit+2 = character id = portrait`。完整 constructor trace 已證實 scripted
> FDFIELD **b1 同時作為** `0x11019` raw key（回傳 slot→`unit+2`）及 `unit+7/+8`；FDFIELD b0 則獨立寫入
> native `+6`（敵0／友1／己2），不是角色／portrait byte。玩家 persistent roster 則有獨立的 `+7`
> source path；兩者共享 cache ABI，不能 alias。
> 因此敵方、玩家及轉職都不得由「恆等」推導 map group，`fig` 僅保留 compatibility approximation。`unit+7`
> 與 `unit+2` 仍是不同 runtime 欄位，不能把欄位或 cache slot 混成角色身分；但 scripted FDFIELD 的 b1 同源
> 是已證實的。先前關於特定轉職 group、龍人例外與 DATO_067 的敘述皆
> 不再作為 renderer/exporter 的證據。

> **玩家初始 record 的狹窄例外（2026-07-26）**：`JOIN` 的 `0x112a5(join_id)` 建立 persistent
> 0x50-byte record 時，直接將同一 `join_id` 寫入 `+7` 與 `+8`；`0x33499` 已證實 `+8` 是
> roster character-ID lookup。之後 `0x10a77` 讀 copied persistent `+7` 當 `0x11019` key。因此新加入
> 的玩家 record **初始** map key 等於 character ID。這是特定 writer 的資料流，不是 FDFIELD/NPC 的
> identity rule。它也不是 immutable：class-change flow `0x314a7..0x3157a` 以 selected roster slot
> 定位 live `0x53a45+slot×0x50`，最後把 UI-selected raw byte 寫回 `+7`（同時重算 `+0x20`）。因此
> 「join-id 相等」只適用於 fresh record；target byte 的高階名稱仍須另追。其 persistence ABI 已知：戰後
> `0x11506` 以 `+8` 配對後完整 copy 0x50 bytes runtime→persistent，故只要 flow 呼叫 `sync_party`，這個
> `+7` mutation 就會被保存；class-change flow 是否在同一 town interaction 立刻進該 post handler 仍不可假定。

| runtime byte | 已證實的狹窄意義 | evidence |
|---|---|---|
| `unit+2` | `0x11019` 回傳的 FDICON cache slot；不可命名為角色／肖像／素材組 | `0x10a92..0x10aa2`、`0x10c50→0x11019`、`0x12831` |
| `unit+3` | map pose/direction selector (`0..3`) | `0x12835`、`pose×3` |
| `unit+4` | movement placement offset；不是 animation frame | `0x127e0` placement branch |
| `unit+6` | native camp raw byte（FDFIELD 路徑的 `b0`） | `0x10ef1`、target predicates |
| `unit+7` | battle FIGANI selector（FDFIELD 路徑的 `b1`） | `0x10ef8`、`0x287b5..0x2884c` |
| index | `slot×12 + pose×3 + cycle` | `0x1291e` |

> **刪除舊表的錯誤 offset/identity 對應**：它把另一份結構標記混入 runtime 0x50-byte
> battle roster，並把觀察到的相等值升格為欄位 alias。今後僅以以上直接讀寫為準；未閉合欄位不另命名。

## 7. sprite & face 統一 authoring 系統（remake 擴充設計，非原版 ABI）

remake 可把自創角色做成**單一資料表**，明確配置 face 與 sprite；這是便利的
authoring schema，不表示原版 raw id、DATO、FDICON cache slot 數值恆等。加新人加一筆:

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

> 新引擎可自行定義角色 asset id；這是 remake extension schema，**不是**原版 runtime map-selector
> provenance。原版 `unit+2` 與 DATO/FIGANI field 的關係仍須由 constructor/resource trace 決定。
- **加新人**:分配未用 id(≥137)、給 face PNG + 12 幀 sprite + 數值 → 引擎自動吃,事件/招募(doc 26/28 `roster_has`)直接用該 id。
- **角色總覽**:`tools/char_summary.py` → 本機 character_summary.png(140 組 sprite+face 並排,統一編號全圖佐證、加新人看缺號)。
- 工具:`decode_fdicon.py`(導原版組)、`decode_dato.py`(導原版頭像)、`export_units.py`（legacy `fig` compatibility）已就緒；
  不得由此自動生成原版 selector mapping。

→ 這把「炎龍 remake」從「複刻」升級成**可擴角色的平台**:配合可擴展事件系統(doc 29),能做原版沒有的角色 + 劇情 + 戰役。

## 8. 受阻 / 待校

- **[待閉合] original map-selector presentation adapter**：`0x127e0` 的公式和 `unit+2` read 已證實；
  `0x11019` 已證實為全域 raw-key cache，且 player/scripted loaders 都開啟 `FDICON.B24`。但 indexed
  framebuffer 的 layer/palette schedule 尚未接到 GUI，故不能由角色表、DATO id、FDICON 檔案序號或轉職表直接
  定 legacy `Fig`。保留玩家/怪物素材的個別對照資料，但不把它當 exporter 或 runtime mapping。
- **[線索] 廢案人物**:FDICON 有些組**沒畫滿 12 格**(未採用角色,僅部分方向/幀);因 sprite 用「組×12 + 方向×3 + 幀」定位,廢案組仍佔 12 格 stride(部分空/重複)。未來可挖廢案角色來用(加新人素材庫)。
- **[M2 待做]** 對話框**嘴型動畫**:DATO_N 的 m0~m3 對話時播放(嘴開合 + 眨眼)。哪幀=閉嘴/開嘴/眨眼、播放節奏(隨文字推進?固定循環?)待反組譯文字渲染器(0x16D00 區,doc 14)確認;M2 對話層實作。
- 方向:目前只導「面向下」待機;4 方向(走動/面敵)待加。
- 戰鬥演出切 FIGANI 大圖:M1 戰鬥動畫階段再接。

> 相關:doc 10(sprite 繪製/陣營著色)· doc 06(FIGANI 動畫)· doc 27(byte[+5] 狀態旗標)· doc 30(工作拆解)。工具:`tools/decode_fdicon.py`、`export_sprites.py`。素材:`extracted/fdicon/`(本機,1680 PNG)。
