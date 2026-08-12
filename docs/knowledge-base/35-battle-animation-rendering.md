# 35 — 全螢幕戰鬥演出繪圖機制(FIGANI 攻守動畫)

> 反組譯《炎龍騎士團2》FD2.EXE（DOS4GW LE；本文位址均為 IDA／Capstone 線性位址，
> raw bytes 必須依 LE object／page 映射回檔案偏移）的**全螢幕戰鬥演出**:
> 攻擊發生時切到的那張大圖畫面 —— 守方 / 攻方全身 FIGANI、戰場背景、狀態欄、斬擊與閃紅。
> 所有結論附反組譯位址佐證;runtime 才決定的值標「待確認」。
> 相關:doc 06(FIGANI 格式)· doc 31(FDICON 地圖小人,另一套)· doc 13(戰鬥選單)· doc 25(戰鬥事件)。

> **範圍勘誤（2026-08-12）**：本檔標題保留歷史名稱，但 `0x28a6c` 是接受兩個
> runtime record index 的共用 renderer。`0x1561f`／`0x18fc6` 的正式戰鬥 caller
> 才可稱攻守演出；終局 `0x2c2a6` 固定傳 `(0,1)`，且在每輪前寫入 raw record
> bytes。後者不可套用本檔正式戰鬥的攻守、命中、動畫階段或逐幀斷言。

## 2026-08-09：幀延遲呈現橋接（E1，非完整演出閉合）

原版 `0x2b9a1` 的 descriptor `+6` 延遲現在由
`remake/assets/figani/delays.json` 保存，並由 `internal/figani.DisplayScheduler`
供戰鬥攻擊呈現使用。`cmd/fd2` 只在 PNG 幀數與固定版本原始封存檔逐幀對齊時建立
演出；缺少延遲資料不會以猜測的 15 幀或固定停留時間補上。

這項改動只負責幀順序與停留時間，`FD2_BATTLE_FPT` 是明示的顯示倍率。命中幀、
傷害提交、音效、閃紅、台座與原版一般玩家畫面仍是獨立證據項目；不能因延遲橋接
通過就宣稱全螢幕攻擊演出或逐像素一致已完成。

## 0. 兩個演出函式(別搞混)

| 函式 | 入口 | 參數 | 用途 |
|---|---|---|---|
| 單圖演出 | **0x28784** | 1 個 unit index | 顯示**單一**單位全身圖(施法/單體演出) |
| 雙 record 演出 | **0x28a6c** | **2 個** runtime record index | 正式戰鬥 caller 已證實用於對打全螢幕演出；其他 caller 仍須各別判讀 |

兩者 prologue 都是 Watcom 風格 `push <frameSize>; call 0x36cd7`(stack-probe):
- 0x28784:`push 0x54; call 0x36cd7`(0x28789),func 範圍 0x28784–0x28a6b `ret`。
- 0x28a6c:`push 0x64; call 0x36cd7`(0x28a71),func 範圍 0x28a6c–0x29116 `ret`。

> ⚠ 既有錨點把 0x287b5(`movzx esi,[ebx+7]`)當成「攻守演出」入口,實際它在**單圖** 0x28784 內;
> 真正的攻守雙圖演出是 **0x28a6c**(0x28ad6 `movzx eax,[ebx+7]` 攻方組、0x28ade `movzx eax,[esi+7]` 守方組)。

### 呼叫鏈(誰觸發演出)

`calls 0x28a6c`(相對 call 來源):

| caller linear | 傳參 | 意義 |
|---|---|---|
| **0x1561f** | `push [0x53c4b]; push ebx; call` → `0x28a6c(ebx, [0x53c4b])` | 正式戰鬥執行流：arg0=攻方、arg1=目標 runtime index（已證實） |
| 0x18fc6 | `push ebx; push ebp; call` | action UI 的一般攻擊路徑；選定 target 後進入同一 renderer（已證實） |
| 0x2c2a6 | `push 1; push 0; call` | 終局尾段固定 record 0/1；該輪前先寫兩筆 record 的 `+6/+7` 與 `[0x540ff]`（已證實，非一般攻守角色斷言） |
| 0x35435 | `push 0; push 0x11; call` | 事件專用 caller；其畫面／戰役語意未知（已證實 caller 形狀） |

`calls 0x28784` → 唯一 caller 0x15195(單圖路徑,與 0x1561f 同一攻擊執行區 0x15xxx,符合「攻擊執行 0x15xxx」推測)。

**[0x540ff] = renderer 分支輸入（強推論）**：函式在 `0x28ae6`、`0x28c15`、
`0x28cd9`、`0x28ef1` 讀寫它；正式戰鬥 caller 也有外部 writer。值為 0 時會額外建立
status／standoff 路徑，非 0 時略過其一部分並在尾端寫回 1（已證實的控制流）。
但是終局尾段每輪直接寫入 20 個非零值，因此不能再把它縮成「首幀 0、其後 1」或
直接命名為通用 phase；各 caller 的高階演出語意仍各自失敗即關閉。

---

## 1. FIGANI 載入(組 × 3)+ buffer

- 攻方組 = `unit_attacker[+7]`(0x28ad6 → `[esp+0x10]`);守方組 = `unit_defender[+7]`(0x28ade → `[esp+0xc]`)。
  這是 battle FIGANI selector；map FDICON `0x127e0` 另讀 `unit+2`，不得稱為同一欄。
- **FIGANI 動畫 index = 組 × 3**:0x28c57 `mov edx,[esp+0x10]; shl eax,2; sub eax,edx`(= 組×4−組 = 組×3),
  再 0x28c78 `組×3`、0x28c99 `inc`(組×3+1)。**每組 3 個動畫**(待機 / 出招 / 受擊,+0/+1/+2;對映待確認),
  確認了既有結論「FIGANI = sprite組 × 3」。
- 載入經 **0x111ba**(資源解碼器,見 §6),descriptor = **0x52388**(FIGANI 動畫容器表):
  `0x111ba(0x52388, prevSlot, 組×3+k)` → 回傳該動畫的 frame 描述子 buffer。
- 解出的動畫描述子存:**[0x54117]=攻方、[0x5411b]=守方**(0x28e4a / 0x28e5b,經 `0x2bc9a` 後處理)。
  另 [0x53a49] / [0x53a5d] 為單圖路徑(0x28784)用的 FIGANI buffer(對映既有錨點)。
- 龍騎兵 / 飛行特例:0x28b72 檢查 `unit[+0x20]==0x13`(職業 0x13=龍騎士)或 `unit[+0x1f] in {4,5}` 且 `unit[+7]==0x1c` → 走特殊組路徑(`call 0x12e38` 換組)。

> 動畫描述子格式(0x2939d / 0x2935b 讀法):`byte[ebp]` = frame 數;`[ebp + i*4 + 8]` = 第 i 幀相對 offset;
> `byte[ebp+1]` = 類型旗標(0=靜態單幀走 BG 路徑,非 0=多幀動畫)。

---

## 2. 守 / 攻 blit 座標 + 縮放(最關鍵)

### 2.1 blit 原語 0x4e63d(原生尺寸,無縮放)

`0x4e63d(src, X, Y, dst, stride, transp)`(由 `ebp+8..ebp+0x1c` 取參,0x4e643 起):
```
esi = src                         ; 來源(自帶尺寸)
word[src+0]→[0x627b4] = 寬          ; 0x4e646 lodsw
word[src+2]→[0x627b6] = 高          ; 0x4e64e lodsw
ecx = X (ebp+0xc)
eax = Y (ebp+0x10);  edx = stride (ebp+0x18)
edi = dst (ebp+0x14);  edi += Y*stride + X      ; 0x4e663 mul / 0x4e665-667
transp = ebp+0x1c   (-1 = 用 RLE 透明跳過;否則色鍵)
```
**關鍵結論:`dst 位址 = dst + Y*stride + X`,圖以 src header 自帶的寬高原生繪製。整條 blit 路徑沒有任何 `imul`/`fild`/`fmul` 縮放運算。**
→ **守方較小、攻方較大不是 runtime 縮放,而是 FIGANI 美術本身就畫成不同尺寸**(景深感燒進素材)。
remake 對不準的根因即在此:該照各 frame header 的寬高 + 下面的座標貼,不要自己 scale。

### 2.2 座標來源

- **螢幕錨點常數 (X=0xa4=164, Y=0x9d=157)**:出現在 0x28f55(`0x4e63d(src, 0xa4, 0x9d, edi, 0x280, -1)`)與
  0x29164 的 figure/台座路徑(0x291b0/0x29268/0x29295)，**不是**狀態欄。這是演出區/單位圖的
  固定錨點(160×100 半屏中央偏下)。
- **figure 的螢幕貼圖錨點是固定常數 (X=0xa4=164, Y=0x9d=157)**,不是 `word[unit+0x40]`:
  - ⚠ 修正既有斷言:`0x29582 push [esp+0x50]` 因前面已 `push -1; push 0x280`(esp 下移 8),實際讀的是 frame local `+0x48`(=**dst work buffer**),**不是** `word[unit+0x40]`。`word[unit+0x40]` 在 `0x294ad` 讀進 `[esp+0x50]` 後是**餵給 `0x29f72` 的算式**,不是直接當 blit X。
  - 最終 figure blit 的螢幕座標寫死在合成處:`0x28f67`(主合成)、`0x29ded:0x29ea2`(==0 路徑):
    `0x4e63d(figureSrc, 0xa4, 0x9d, dst, 0x140, -1)` → **(164,157) 是 figure 的螢幕左上錨**(全 figure 共用此常數)。
- **⚠⚠ 重大修正(第一性原理追到底,2026-06-29):`word[unit+0x40]` = 當前 HP,不是「戰場格 X」!**
  舊斷言(把 +0x40 當座標)**錯誤,已推翻**。決定性兩證:
  - **血條鐵證**:`0x18c98 movsx ebp, word[esi+0x40]`(esi=[0x53a45]+idx×80)→ push 給 `0x18795`,
    算 `len = word[+0x40]×101 / word[+0x42]`(0x187ad imul 0x65 + idiv)。座標×101÷座標無意義,
    只有 **當前HP×101÷最大HP = 血條%長度** 才合理 → **+0x40=當前HP、+0x42=最大HP**。
  - **figure displacement 更正**：先前把 `0x29f72` stack locals 反推為 runtime
    `unit+0x48/+0x4a` 的螢幕投影座標，已被 command 17/18 的 direct writers 推翻：它們以
    `+0x22/+0x23` 暫時旗標對這兩個 word 套 15% 增幅，而 class synthesis 亦寫
    `+0x48/+0x4a/+0x4c/+0x4e` 作 derived AP/DP/HIT/EV，不能再用這些 runtime offsets 當座標。
    `0x2935b` 已直接閉合真正來源：descriptor 的 `frameIndex*4+8` relative pointer 指向單幀 header，
    其 `u16 +0/+2` 是每幀 X/Y，payload `+9` 交 `0x4e63d`。
  - **正確欄位**:**`+0x40`=當前HP、`+0x42`=最大HP、`+0x44`=當前MP、`+0x46`=最大MP**(spawn 0x10fe9 設 cur=max=滿血滿魔);
    figure displacement 是 frame metadata 的逐幀 X/Y；HP/MP 演出中被攻擊/施法改寫,狀態欄即時反映。
  - **寫入點**:spawn `0x10FE9`(`+0x40=+0x42=`HP、`+0x44=+0x46=`MP,值從 caller 參數來=該角色滿 HP/MP)、`0x1142A`(同款);
    戰鬥演出中 `0x2975A` 每幀寫 `word[unit+0x40]`——**= 被攻擊時 HP 抽乾的逐幀內插**(舊誤標「lunge 前衝 current X」);狀態欄血條即時跟著縮。
  - `0x114E4/0x1B821` 與 figure placement 的 exact ABI 需重新追查；先前把它們寫成
    `+0x48..+0x4e` screen bounding box 的斷言已撤回。
- **`0x29f72(攻方idx, 守方idx, &out)` = 戰鬥結果 resolver，非 lunge**:
  - `ebp`/`edi` = 雙方 unit(各 `idx*80`,base `[0x53a45]`)；直接讀攻方 `+0x48/+0x4c`、守方
    `+0x4a/+0x4e`（derived AP/DP/HIT/EV）、守方 current/max HP `+0x40/+0x42`，以及 item record。
  - 在 `0x1f183` gate 未通時，分別以 `0x12e38` control byte+1 索引 `0x51a12` AP 與
    `0x51a2a` DP 百分比，加入攻／守 derived stats；再以 `0x4e893` RNG 建立 hit/crit/damage 路徑。
    這也證實兩張 table 不是 figure direction/pose 微調表。
  - 輸出 struct 的 `+0/+4/+8/+0x10` 為結果旗標，`+0x14` 為最後 damage；`[0x53ec8]` 是由結果／
    unit fields 算出的 presentation 用量。它不輸出或讀取已證實的 figure screen coordinate。
  - → 撤回「figure 前衝幅度 = 雙方格距 × 動畫% × 方向微調」。figure 位移已由
    `0x2935b` frame metadata 的 X/Y header 閉合，不可借 `0x29f72` 命名。
- **frame 自帶 (dx,dy)**:figure 繪製 wrapper **0x2935b** 解單幀:
  ```
  eax = frameIdx*4 + descriptor          ; 0x2936a
  edx = descriptor + [eax+8]             ; 該幀資料 ptr
  word[edx+0]=幀X偏移, word[edx+2]=幀Y偏移 ; 0x2937a/0x2937d
  src 像素 = edx+9                        ; 0x2938f add eax,9
  → 0x4e63d(edx+9, 幀X, 幀Y, dst, stride, transp)
  ```
  → 每幀內嵌自己的 (dx,dy),**斬擊弧 / 出招前傾 / 受擊後仰就是逐幀換 (dx,dy)**。

### 2.3 翻轉 / 左右:**沒有 runtime 水平翻轉**,靠 `byte[unit+6]` 選合成路徑(已確認)

- **全 blit 家族不做水平鏡像**:`0x4e63d` 的 RLE 解碼只有前向 `stosb`/`movsb`/`rep movsb`(`0x4e6a7`/`0x4e6bd`/`0x4e6d3`),
  全檔唯一 `std`(反向)在 `0x373EB`(memcpy 輔助,與 blit 無關)。→ **攻 / 守 figure 不是同一張圖 runtime 翻轉,而是 FIGANI 美術各自畫好朝向**
  (與 §2.1「大小燒進素材」同理:朝向也燒進素材)。**remake 守方原圖已面右就別再翻**。
- **`byte[unit+6]` 分支(0x29536)決定走哪條合成路徑**(figure 本體 blit 兩路都用同一個 `0x2935b`,不翻):
  - `byte[unit+6]!=0` → `jne 0x295c3` 迴圈 → 收尾 `0x295f8` `call **0x29c90**`(BG 貼 (0,50)、figure 進 buffer 走 frame 內嵌 (dx,dy)、往一方向 slide-in)。
  - `byte[unit+6]==0` → `jmp 0x2969f` 迴圈 → 收尾 `0x296d4` `call **0x29ded**`(BG 貼 (0,50)、**figure 貼固定錨 (164,157)** `0x29ea2`、反方向 slide-in)。
  - → 兩路差在 **slide 進場方向** + **figure 錨點**(==0 用 (164,157),!=0 用 frame 內嵌 (dx,dy))→ 這正是**攻 / 守腳底 Y 不同**的程式來源
    (remake 量攻方腳 y≈175 / 守方 y≈150:一方錨在 157、一方走 frame dy;非統一 Y)。確切「哪邊是攻、哪邊 frame-dy 落在 150」需 runtime `byte[unit+6]` 對照,**機制已定、配對待確認**。
- **左 / 右 buffer**:`0x28dfd` 同檢查 `byte[unit+6]`,`0x28e05-0x28e16` 視之交換 `[0x54107]↔[0x54103]`(決定誰進左 / 右 BG buffer)。

---

## 3. 背景 BG 繪製 + 戰場→BG 對應

### 3.1 BG 多層載入 + blit

- 演出進場前,BG.DAT 由 **0x22d1b** 載入(既有錨點;0x2866x 區三次 `0x22d1b` 以 index 對載地形圖,在前置函式內)。
- 演出函式內,BG 分**多層**經 0x111ba(descriptor = **0x52381**,該位址本身就是字串 `"BG.DAT\0"` → 0x111ba 是「開 .DAT 取 entry[index]」)解出:
  - `0x54107`（`0x28CD4`，index＝章節表算出的變動值）、`0x54103`
    （`0x28DAA`，index 0）、`0x5410B`（`0x28DC4`，index 0）、`0x5410F`
    （`0x28DDE`，index 1）、`0x54113`（`0x28DF8`，index 2）。
  - → BG.DAT 至少 **3–5 個 entry**(idx 0/1/2 + 章節索引層),遠景 / 近景 / 土台分層。
- **BG blit 座標**:`0x4e63d(BGsrc, X=0, Y=0x32=50, dst, stride, -1)`:
  - 0x28d42(`[0x54107]`)、0x28e27(`[0x54103]`),stride 0x140=320;slide 合成 `0x29c90`/`0x29ded` 再把 `[0x5410b/0f/13]`(idx 0/1/2)循環貼於 (0,50)。
  - → **背景與各層都貼在 X=0, Y=50,寬 320**(整屏寬;上方 50px 與下方留給狀態 UI）。

### 3.2.5 已驗 battle fixture：我方背影+台座／敵方正面（不可全域外推）

- 已驗收的亞雷斯／盜賊 capture 是從我方背後看戰場：我方背影並帶TAI台座，
  敵方正面。這證明該fixture的合成選擇，不足以證明所有角色、特殊戰鬥或
  `unit+6` raw分支永遠等同高階陣營。
- 在這個fixture中台座跟隨我方slice，而不是當回合攻／守；跨caller規則仍須
  由TAI selector與`unit+6` writer驗證。
- 故 orig_05:亞雷斯(藍衣**背影**、腳下大 dither 土台)= **我方**;盜賊(紅頭巾**正面**)= **敵方**。
  (先前 doc 用「攻方 / 守方」描述左右是**誤框架**,正名為「我方(背影,右,有土台)/ 敵方(正面,左)」。)
- remake 對映:我方單位用背影 FIGANI 幀 + 腳下貼 **TAI.DAT 台座**(見 §3.3);敵方用正面 FIGANI 幀、無台座。

### 3.3 我方腳下「土台」= **TAI.DAT 菱形台座素材**(反組譯確認),非 FIGANI 自帶、非 BG 層、非程式畫

- **反組譯鐵證**:figure 演出函式 `0x29164` 在 `0x28c46` 呼叫 `0x111ba("TAI.DAT"@0x52393, prevSlot, idx)`
  載入台座圖,**與 figure 一起淡入畫在腳下**(見 §4.0)。→ **土台是 TAI.DAT 獨立 sprite,不是 FIGANI 圖的一部分**。
- **TAI.DAT 格式 / 內容**:sprite-RLE(FIGANI codec 家族);**TAI_004=154×42、TAI_005=155×42 = 菱形透視台座**
  (worklist `91` 早記錄「TAI.DAT=WxH sprite-RLE,如 155×42」);台座 idx 由 `byte[unit+6]`/職業算(預設 3)。
- **靜態前提**:戰鬥合成只走 `0x4e63d`(blit)/`0x11eb0`(present)/`0x11d40`(色盤),無 fillrect/circle → 土台必是素材。
- **刪除的錯誤斷言**(本輪推翻,省 token):
  - ❌ **「土台 = FIGANI_013_f01 自帶 dither 橢圓」= 誤判**:FIGANI 圖底那圈 dither 是 sprite 自帶的**腳下小陰影**,
    **不是** orig 顯示的綠色大台座;真正的大台座是 `0x29164` 獨立載入 blit 的 **TAI.DAT**。我把小陰影誤當主土台,繞了一圈。
  - ❌ 「土台 = BG.DAT 前景層 `[0x5410b/0f/13]`」:該三層由 `0x111ba("BG.DAT", idx=0/1/2)` 載入 = `BG_000/1/2`
    = 垂直藍漸層(`3f4f→3f4e` 每列遞減,反組譯 + dump 雙證),是藍底 / 捲簾,非土台。
  - ❌ 「土台 = FIGANI_012 自帶小 dither」:用錯幀。
- **remake 對映**:抽 **TAI.DAT 台座 sprite**(`decode_figani`/`decode_sprite` codec)貼我方腳下,
  **不要 drawEllipse、也不要倚賴 FIGANI 自帶 dither**;台座顯示為綠是 TAI 素材本身的顏色(疊草地)。
  **TAI 確切 entry / 顏色 / 對齊位置待 decode 驗證**(worklist 第 6 輪步驟 1)。

### 3.4 攻擊白斬擊弧 = FIGANI 攻擊幀自帶,非程式畫(視覺確認)

- orig_05 那道大白色揮砍弧**燒在 `FIGANI_013` 攻擊幀的 sprite 像素裡**(連續幀畫出揮劍殘影),
  不是 runtime 用 vector / 畫線疊的。remake **不要自己用 `vector.StrokeLine` 補白弧**(舊版這樣做,已移除),
  畫對攻擊動作幀(組×3+1)弧就自然帶出。

### 3.2 戰場 → BG 參數對應

- **章節參數表 0x52363**:`[0x53c03]`(= 章節 byte,既有錨點)當 index 取 `byte[chapter + 0x52363]`:
  0x28b61 `mov edx,[0x53c03]; movzx edx, byte[edx+0x2363]` → `[esp+0x18]`。
  表值 = `[4, 9, 14, 18, 14, 0, 2, 4, 6, 8, 10, …]`(0x52363 起,前 6 個是章節用)。
- 此值參與 figure / BG 選擇分支(0x28b89 與龍騎兵特例聯動),**確切語意(選哪張 BG / 色盤)待確認**;
  但「**戰場→演出參數由章節 [0x53c03] 索引 0x52363 表**」這條對應已確認。
- 既有錨點 BG_004 森林 320×100:與本處「BG 寬 320、貼在 y=50」尺寸吻合。地形→BG index 的細部對應待續(追 0x22d1b 的 index 參數來源)。

---

## 4. 狀態欄(血條框)繪製 — 真正函式 0x18c6d(2026-06 嚴格 RE 修正)

> ⚠ **重大修正:0x29164 不是狀態欄函式**(舊 §4 標錯)。0x29164 畫的是 **figure 全身圖 + 腳下台座(TAI.DAT)** 的淡入演出(見 §4.0)。
> 真正畫「深藍底框 + 立體邊 + 名字 + LV + HP/MP 條 + 數值」的是 **0x18c6d**(見 §4.2)。
> 三元素拆解結論(各附位址):**框 = 素材 sprite、HP/MP 條 = 程式畫(寬度算出來)+ 逐欄 cell、文字 = 點陣字 / 數字 cell 素材**。

### 4.-1 [z-order] 狀態欄先畫、figure 後畫 → figure 蓋住狀態欄(已確認)

演出主函式 0x28a6c 內的繪製順序(反組譯 call 序):
- **狀態欄(0x2a289→0x18c6d)在 `0x28ce7`、`0x28d62` 先畫**;
- **figure(0x29164 全身圖+台座、0x2939d renderer)在 `0x28e76`、`0x28e9a`、`0x28ee0` 後畫**。
- → **figure z-order 高於狀態欄**:動畫格完整、狀態欄被 figure 蓋住一部分(orig_05 亞雷斯的劍穿過我方欄上緣即此)。
- remake 對映:`drawBattleScene` 順序必須 **BG → 狀態欄 → figure**(figure 最後畫蓋上),別反過來。
- 另:**我方欄上緣離畫面頂有間隔**(非貼頂)、敵方欄離 150 線有 ~3px@320 間隔(對照截圖量)。

### 4.0 0x29164 = figure + 台座(TAI.DAT)淡入,非狀態欄

`0x29164(unitIdx, stride, dst, …, platformFrame, figureSrc)`(攻守各一次,caller 0x28e76 / 0x288f3):

- prologue `push 0x2c; call 0x36cd7`;`esi=[0x53a45]+idx*80`;`movzx ebx, byte[unit+6]`(0x2918a)選兩條對稱路徑(左半 / 640-寬右半 off-screen buffer)。
- **figure 全身圖**:`0x4e63d(figureSrc, X=0xa4=164, Y=0x9d=157, …)`(0x291b0 / 0x29268 / 0x29295)— 與 §2.2 figure 錨點一致。
- **台座(platformFrame)**:`0x2935b(platformFrame, 0, dst, stride, -1)`(0x291cf / 0x29247)貼一張 frame。
  **platformFrame 來自 TAI.DAT**:0x28c46 `0x111ba("TAI.DAT"@0x52393, prevSlot, idx)`(idx 由 byte[unit+6]/職業算,預設 3)。
  TAI.DAT = LLLLLL 容器、**53 個 ~155×42 菱形台座 sprite**(7-byte 空槽佔位另計);解出來是**透視菱形格台座**(orig 截圖右下藍 figure 腳下那塊綠色台座)。
  → 舊 §4「name/LV/HP/MP 經 0x2935b 以幀方式貼」**錯**:0x2935b 在此貼的是台座,不是文字。
- **8 次迴圈 + `0x11d40(0, 0xff, esi*6)`**(0x291f9 esi=8→0)= **figure/台座的色盤淡入(brightness ramp 0→48)**,每圈 `0x11eb0` present 一次 → 進場時 figure 由暗轉亮。
  → 舊 §4 / §5「HP 條 = 色盤寫入 0x11d40」**錯**:這個 0x11d40 迴圈是 figure 淡入,跟 HP 條無關(HP 條在 0x18c6d 用程式畫,見 §4.2)。

> TAI = 台(台座 / dais),非「態(狀態)」。視覺與反組譯雙重確認(155×42 菱形 + 0x29164 貼在 figure 同函式)。
> 這也補充 §3.3:figure 腳下那塊 dais = TAI.DAT sprite(由 0x29164 經 0x2935b 貼),不是 BG 層、不是純 FIGANI dither(§3.3 的「圓圈/土台」歸屬需併此重看)。

### 4.1 視覺對照(orig_05 放大)

放大 orig_05 攻 / 守欄(`_orig_atkbar` / `_orig_defbar`):
- **底框**:深藍底 + **左上亮白、右下暗的 raised bevel 立體邊**(光源左上);半屏寬;敵欄右上、我欄左下。
- **內容**:名(白字,左上)、`LV‧NN`(白字右上,僅我方單位有)、`HP`/`MP` 淺藍標籤、HP **亮黃**長條 / MP **暗紅**長條、數值(白字右對齊)。

### 4.2 狀態欄繪製器 0x18c6d(三元素拆死)

**呼叫鏈**:演出函式 0x28a6c 在 0x288a9 / 0x28ce6 `call 0x2a289` →

- **0x2a289 = 算欄位螢幕座標 + 呼叫繪製器**:`movzx eax, byte[unit+6]`;`==0 → off=0xc080`(= work 偏移 0xc080,/0x140=row154 → **(x=0, y=154) 左下**,我方 盜賊 欄)、`!=0 → off=0x5ab`(/0x140 → **(x=171, y=4) 右上**,敵方 亞雷斯 欄);章節 0x18 + unit 0x11 特例強制 0xc080。`dst = bufBase + off`;`call 0x18c6d(dst, stride=0x140, unitIdx)`。
- **0x18c6d** body 0x18c6d–0x18d87:`esi=[0x53a45]+idx*80`;`ebp=word[unit+0x40]`、`[esp+8]=word[+0x42]`、`[esp+4]=word[+0x44]`、`[esp]=word[+0x46]`。

**① 框 / 底框 + 左上亮白立體邊 = 素材 sprite(blit)**
- `eax=[0x53a81]; eax+=[eax+0x5e]`(取 UI 容器內的面板底圖 sub-resource)→ `call 0x4e8af(dst, panelBgSprite, stride)`(0x18cbb-0x18cc1)。
- **0x4e8af = RLE 逐列 blit 原語**(`lodsw`寬、`lodsw`高、逐列 `call 0x4e916` 解 RLE `stosb`、`add edi,stride`)。姊妹 0x4e8e1 = 水平鏡像版(`dec edi`)。
- → **整塊面板背景(深藍底 + bevel 立體邊 + 可能含 "HP"/"MP"/"LV" 標籤字)是一張預渲染 sprite,blit 上去,不是程式畫線/填色**。來源 = 全域 UI 容器 `[0x53a81]` 的 +0x5e entry(`[0x53a81]` = FDOTHER.DAT 級 UI sprite 容器,loader 待確認;由 digit/bar/面板共用此容器佐證)。
- → **修正 §4.1 舊註「bevel 來源未確認 / remake 用程式 bevel 近似」**:bevel 是素材,remake 應改 blit 面板 sprite(FDOTHER 內),別再用 drawBattlePanel 程式畫。

**② HP / MP 血條 = 程式畫(長度算出來)+ 逐欄 cell blit,非色盤、非單張條 tile**
- HP 條:`0x18795(x, dst, 0x17, curHP=word[+0x40], maxHP=word[+0x42])`(0x18cc9);MP 條:`0x18795(x, dst, 0x1a, curMP=word[+0x44], maxMP=word[+0x46])`(0x18ce6)。`0x17/0x1a` = 兩條的 Y 列。
- **0x18795 算填充長度**:`if maxHP==0 return; if curHP==0 len=0; else len = curHP*0x65/maxHP + 1`(0x65=101)→ **條長 = 當前/最大 × 101 + 1 像素欄**;再 `call 0x17d6f` 畫。
- **0x17d6f 逐欄畫條**:`ebp=len`;前 `len` 欄 `call 0x1685c(x+i, y, [0x53a81], colorIdx)` 用「傳入的條色 cell」;`len..101` 欄改用**空槽 cell index 0x1d**(暗版),端點 0x1e / 0x5d 收尾。
- **0x1685c → 0x4e9bb**:`edx=[0x53a81]; edx += [edx + colorIdx*4 + 6]`(容器查 entry)→ `0x4e9bb` 逐列 `rep movsb` blit 該欄 cell。→ **條的像素是 [0x53a81] 容器裡的 1px-寬漸層欄 cell(素材),但「畫幾欄」由 HP 比例算(程式)**;空槽 = index 0x1d cell。
- **血條長度來源釘死**:`unit+0x40 = 當前 HP`、`+0x42 = 最大 HP`、`+0x44 = 當前 MP`、`+0x46 = 最大 MP`(由 0x18795 的 `cur*101/max` 推得;spawn 時 0x10fe9 把 +0x40=+0x42、+0x44=+0x46 設成同值=滿血滿魔)。
  > ✅ **衝突已釐清(第一性原理追到底)**:`+0x40/+0x42/+0x44/+0x46` = **HP/MaxHP/MP/MaxMP**(血條算式鐵證)。
  > 舊「戰場格 current/home X/Y」標法錯誤；其後把 figure lunge 位置改標為
  > `+0x48/+0x4a`（螢幕投影）的說法也已撤回，
  > `0x29f72` 雖讀 +0x40 但沒拿去算位置。詳見 §2.2 開頭「重大修正」框。
- → **修正 §4/§5 舊註「HP 條 = 色盤動畫 0x11d40」**:HP/MP 條是 0x18795/0x17d6f 程式畫(逐欄填),色盤 0x11d40 那段是 0x29164 的 figure 淡入,兩者無關。

**③ 名字 / LV / 數值 = 素材文字 cell(點陣字 + 數字 cell),非 TTF、非單張預渲染整條**
- **角色名(亞雷斯 / 盜賊)**:`0x15f84([0x53a7d] 字串表, nameIdx=byte[unit+8]+1, dstAddr, …, 0xcd, 0x4c)`(0x18d61-0x18d7f)。
  0x15f84 = 字串排版器:`esi=字串表; eax=index; movsx eax,word[esi+idx*2]; esi+=eax`(word-offset 字串表)→ 逐字 `call 0x16559`。
  **2026-07-27 official IDA correction**：一般字不是走 `0x16559/[0x53a85]`。
  `0x15f84` 對每個普通 word 呼叫
  `0x4ea2a([0x53a75],glyph,dst,stride,foreground,shadow,background)`；
  `[0x53a75]` 是 boot 載入的 **FDOTHER.DAT #4 16×16 1bpp font**。
  `0x16559(index)` 才會從目前 DATO `[0x53a85]` 的 offset table取 mouth
  frame並以 `0x4e8af/0x4e8e1` 重貼 portrait，只由對話控制／嘴型路徑呼叫。
  → **名字 = 16×16 1bpp glyph + 程式指定 foreground/shadow/background**，
  不是 TTF、不是 DATO sprite、不是整條預渲染。
  → **remake 名字偏小的修法**:名字用 16px-class 點陣字(每全形字 16px 寬),別用 TTF 縮放;名字寬 ≈ 字數 × 16(狀態欄路徑 0x15f84 的實際 advance 可能略窄,精確值待確認)。
- **數值(LV-NN / HP 028 / MP 000)**:`0x187d6` = 數字繪製器(`call 0x377d9` itoa → 逐位 `[0x53a81]` 容器取 **digit cell**,`imul eax,ebx,6` → **每位數字 cell 寬 6px**,blit 經 0x1685c/0x4e9bb)。
  - LV/狀態值:0x18d02 `0x187d6(…, byte[unit+0x21], 0x1f, mode2)`(mode2 上限 99);HP/MP 數值:`0x1875d`(0x18d25/0x18d42)→ 內部 `cur==max ? 色0x1f : 色0x2a` 再 `call 0x187d6`(滿血 / 非滿血換色)。
  - → **數字 = 6px-寬預渲染 digit cell 素材**([0x53a81] 容器),itoa 後逐位 blit;不是字型。
- **"HP" / "MP" / "LV" 標籤字**:內含在 ① 的面板底圖 sprite 裡(已視覺確認:框圖 #22 自帶 HP/MP/LV‧ 標籤,見 §4.2.5)。

### 4.2.5 [已破解] UI 容器 = FDOTHER#5 "LMI1" directory（codec 必須依 caller 決定）

`[0x53a81]` = `0x111ba("FDOTHER.DAT", [0x53a81], 5)` 載入的 **FDOTHER 第 5 個資源**,本身是 **"LMI1" 子容器**(doc 14 對話框框圖同源):
- **LMI1 結構**:`char[4]"LMI1"` + `uint16 N`(sub-resource 數,FDOTHER#5=138) + `uint32[N] offset` + 各 sub-resource(`uint16 w, uint16 h, codec 資料`)。
  - 2026-07-25 player-asset regression 更正：offset 是各 entry 的**開始**位址，不是壓縮資料的嚴格 end；
    `0x4e916` 依目的 `w×h` 停止，最後一段 repeat 可跨下一 entry 的 offset。解析器須以容器末端為
    唯一 source bound，不得以 `offset[i+1]` 截斷資料並誤判原版 #5 為 malformed。
- **`0x4e916` 像素 codec（僅適用其 caller 選到的 entries）** —— **本輪關鍵破解,跟 FIGANI/TAI 的 4-mode、doc05 image-RLE 都不同**:
  ```
  讀控制 byte c:
    c <= 0xC0 : c 本身就是一個像素值(literal 單 px)        ; 0x4e91e cmp 0xc0 / 0x4e922 xor ah,ah
    c >  0xC0 : run,長度 = c - 0xC0,後跟 1 個像素值,重複  ; 0x4e925 sub ah,0xc1 / 0x4e92a lodsb
  (透明 = palette index 0;run 跨行,線性解 w*h;純 literal 小圖等同 raw)
  ```
- **`0x4e916` blit 端**:`0x4e8af`(正向)/ `0x4e8e1`(水平鏡像 `dec edi`)逐列呼叫 `0x4e916` 取像素 `stosb`；不得據此推論整個 LMI1 directory 都走這條 codec。
- **⚠ LMI1 容器內混用兩種 codec**(對應兩條 blit 路徑,踩過):
  - 大圖(框 #20/21/22)= **0x4e916 codec**(上述),blit 走 `0x4e8af`;
  - 小 cell(血條欄 / 數字)= **FIGANI 4-mode sprite RLE**(doc06),blit 走 `0x1685c→0x4e9bb`。
  - 用錯 codec 解出來是彩色雜訊——先看該資源的 blit 端(0x4e8af vs 0x4e9bb)決定 codec。
- **狀態欄用到的 sub-resources(視覺+模板匹配驗證,err=0 像素全等)**:
  - **框 = #22(149×42)**:深藍底 + 左上亮白 raised bevel + 「LV‧」+「HP」「MP」標籤 + 血條紅槽,**全燒在這張素材**(`[0x53a81]+0x5e` → #22)。框內槽 native:HP y22–26、MP y31–35、槽 x21–123。
  - **血條 cell = #27–30(1×5)**:純色漸層欄(#27/28 黃=HP、#29/30 紅=MP/空槽);raw(值全 ≤0xC0)。
  - **數字 digit cell(6×8,「1」5 寬)三套色**:**#31–40 = 白/藍影 0–9(戰鬥狀態欄用)**、#42–51 = 白/綠、#119–128 = 白/黃橘(`0x1875d` 滿血/非滿血換色的素材面)。
  - **數字排版(模板匹配 orig 定位)**:advance 7px;首位數字 local 座標 LV(132,4)/HP(126,21)/MP(126,30)。
  - **框在畫面上是 149×42 原生 blit**(非拉伸):敵方欄 @320 (0,154)、我方欄 (171,4)(右緣貼齊 320)。
- **工具**:`tools/decode_lmi.py`(列 sub-resource / 解 PNG,index0 透明)。**remake**:抽 #22 框貼上(`remake/assets/ui/panel.png`),名字 / LV / 血條填充 / 數值疊在框上(槽座標見上)。
- ⚠ **palette 陷阱(踩過)**:FDOTHER#0 調色盤是 **VGA DAC 6-bit(0–63)**(doc05),任何新解碼工具都要 `(v<<2)|(v>>4)` 轉 8-bit,否則圖整體暗 4 倍(狀態欄底色 (14,21,38) vs 正確 (56,85,154))。

### 4.3 三題結論速查

| 元素 | 素材 or 程式畫 | 位址 / 來源 |
|---|---|---|
| 框 + 深藍底 + 立體 bevel | **素材 sprite(blit)** | 0x4e8af blit `[0x53a81]+0x5e`(UI 容器面板底圖);繪製器 0x18c6d,座標器 0x2a289(byte[+6]: 0→(0,154)、≠0→(171,4)) |
| HP / MP 血條 | **程式畫長度 + 逐欄 cell** | 0x18795(len=`cur*101/max+1`)→ 0x17d6f 逐欄 0x1685c blit `[0x53a81]` 漸層欄 cell;空槽 cell 0x1d;HP=unit+0x40/+0x42、MP=+0x44/+0x46 |
| 角色名 | **16×16 1bpp 點陣 glyph** | 0x15f84 排版 → 0x4ea2a blit `[0x53a75]` FDOTHER#4 font；字串表 `[0x53a7d]`,index byte[unit+8]+1；caller指定前景/陰影/背景色 |
| LV / HP / MP 數值 | **6px digit cell 素材** | 0x187d6 itoa→逐位 blit `[0x53a81]` digit cell(寬6);0x1875d 滿血換色 |
| 台座(figure 腳下) | **素材 sprite** | TAI.DAT(0x52393)entry,0x29164 經 0x2935b blit;155×42 菱形 dais |

### 4.1 視覺對照(orig_05 放大)+ remake 對映

放大 orig_05 攻 / 守欄(`_orig_atkbar` / `_orig_defbar`):
- **底框**:深藍底 + **左上亮白、右下暗的 raised bevel 立體邊**(光源左上);半屏寬(160×40@320,攻右上 / 守左下)。
- **內容**:名(白字深描邊,左上)、`LV‧NN`(白字右上)、`HP`/`MP`(淺藍標籤)、HP **亮黃**長條 / MP **暗紅**長條(幾乎佔欄寬)、數值(白字右對齊)。空槽 = 該條色的暗版(暗黃 / 暗紅),非統一黑。
- **勘誤（2026-08-10）**：底框與 bevel 已由 FDOTHER#5 LMI1 #22
  證實為預渲染素材（見 §4.2.5），重製端 `drawBattlePanel` 已貼上
  `assets/ui/panel.png`；HP／MP 仍依 `0x18795→0x17d6f` 逐欄繪製，不能把整個
  面板描述成程式畫線。姓名也已由現代 28 像素字型改接 FDOTHER#4 的 16×16
  點陣 glyph；未知 Unicode 對映才保留失敗即關閉的 TTF fallback。這只修正
  已證實的戰鬥狀態欄消費端，未提升全螢幕演出或一般玩家 E2 等級。

---

## 5. 正式戰鬥 caller 的動畫控制（windup → swing → impact → standoff）

> 本節只整理 `0x1561f`／`0x18fc6` 的正式戰鬥脈絡。`0x2c2a6` 的終局尾段 20 次
> 呼叫在每輪先寫入非零 `[0x540ff]`；沒有一般玩家擷取或完整 renderer input 前，
> 不得把它稱為同一段動畫、同一幀迴圈或同一套戰鬥結果。2026-08-12 的
> IDA／Capstone 追加證據已證明非零分支仍會載入 TAI／FIGANI／BG、經
> `0x29164→0x2939d` 合成並輸出 VGA；它略過的是 `[0x540ff]==0` 才呼叫的
> `0x29f72` 一般戰鬥結果解析器，不是整段視覺輸出。

- **renderer 分支輸入**：正式戰鬥上層可重複呼叫 `0x28a6c`；已證實的是
  `[0x540ff]==0` 時會走載入／全合成相關控制流（`0x28e3e` 計算
  `[0x54117]/[0x5411b]`），`0x28ef1` 隨後寫回 1。各 caller 的呼叫頻率與
  「一呼叫是否等於一可見幀」尚未由一般玩家 E2 證實。
- **終局非零分支（E0）**：`0x28c15` 起依 `[0x540ff]` 選 TAI／BG，並載入
  `3*record1[+7]`、`3*record1[+7]+1`、`3*record0[+7]`、
  `3*record0[+7]+1` 的 FIGANI 資源；
  `0x29164` 的 `+6==0` 與非零兩支都會完成九階段合成、`0x2935b`、VGA copy
  與 `0x11d40`。`0x2939d` 在 `[0x540ff]!=0` 時把一般戰鬥結果欄位歸零，
  不呼叫 `0x29f72`，但仍執行 frame loop、`0x2935b`、`0x11eb0`、等待與
  `0x29c90`／`0x29ded`。因此現況是「原版可見消費鏈已證實，重製轉接器與
  `0x2c2a6` 呼叫當下完整 records/globals 未閉合」，不能再寫成非零分支無畫面，
  也不能因 E0 控制流已閉合就直接接正式尾段。
- **進度百分比**:figure renderer **0x2939d** 內 0x2946a `call 0x4e893` → 0x2947b `idiv 100`(`mov ebx,0x64; idiv`),
  取餘數判斷階段(`cmp edx,3` < 3 時 `[esp+0x4c]=2`,多畫一層)→ **動畫進度以 0–99% 表示,百分比決定當前幀 / 疊層**。
- **幀迴圈**:0x2939d 以 `byte[ebp]`=幀數迴圈(0x29409-0x29424)；在
  `[0x540ff]==0` 的正式戰鬥 branch 會呼 `0x29f72`，再由 `0x2935b` 貼 frame。
  終局尾段的非零 branch 不據此取得 `0x29f72` 結果；不可把該 resolver 當位移輸出，
  也不可把它猜接到尾段。
  幀的 (dx,dy) 內嵌(§2.2)→ **swing 斬擊弧 = 逐幀位移 + 換幀**。
- **idle / fallback 描述子**:0x2939d 進場 `rep movsd` 從 **0x5255f**(6 dword)與 **0x52577**(6 dword)複製預設描述子到區域 frame
  (0x293cf / 0x293df)→ 沒有真實動畫時的**待機姿態 fallback**。
- **閃紅 / figure 淡入涉及色盤操作,但不是無條件的全畫面紅罩**:
  - **0x11d40** 是 VGA DAC 寫入迴圈:`push 0x3c8 / push 0x3c9; call 0x37795`(0x11d5c / 0x11d73)→
    out 到埠 **0x3c8(palette index)/0x3c9(palette data)**。0x37795 = DAC 埠寫入原語。
  - 同手法在 0x28784 / 0x286dd 的 fade-in 迴圈(用 0x53a65 色表插值),以及 0x29164 的 figure/台座淡入(`0x11d40(0,0xff,esi*6)`,brightness ramp,見 §4.0)。
  - `0x2939d` 的命中分支只有在 frame record `+4 == 1`、傷害步進完成且
    `0x29f72` 的兩個原始輸出欄位非零時，才寫入兩段短暫 DAC 序列；色值、
    20/40 毫秒等待與原始位址見 [`fd2_battle_impact_pulse_ida.txt`](../data/ida/fd2_battle_impact_pulse_ida.txt)。
    這證實「條件式色盤脈衝」，不證實可用 RGBA 紅罩取代它。
  - ⚠ **修正**:**HP 條變化不走色盤**。HP/MP 條是 0x18c6d 的程式畫(0x18795 算長度 `cur*101/max+1` → 0x17d6f 逐欄填),見 §4.2;舊註「HP 條亦走 0x11d40 色盤」已刪。
- **standoff**:演出結束 0x290xx 釋放所有 buffer(0x28fc1-0x2900e 連續 `0x37416` free)、復原色盤(0x290b8 `0x375c0`)、`0x11cac` 還原畫面。

---

## 6. 螢幕座標系(確認)

| 量 | 值 | 出處 |
|---|---|---|
| 螢幕寬 | 0x140 = 320 | 0x28f3c / 0x4e63d stride 參數 |
| 螢幕高 | 0xc8 = 200 | 0x28f37 / 0x11eb0 rows |
| VGA framebuffer | 0xa0000 | 0x28945 / 0x28fb4 |
| **work buffer stride** | **0x280 = 640** | 0x28f47 / 0x28f57 / 0x2935b |
| present 來源寬 | 0x140 = 320 | 0x11eb0 bytesPerRow |

- **work buffer 是 640 寬、但只 present 左半 320**:`0x11eb0` 每列 memcpy 320 byte、來源 stride 640、200 列 → VGA 320×200。
  雙倍寬 work buffer(`lea ebx,[edi+0x140]` 0x2929 系列存取右半)疑作**off-screen 預備區**(下一幀 / 滑入的 figure 先畫右半再捲入),
  具體用途待確認,但「**work stride 640、可視 320**」這條已確認 → remake 若用單寬 buffer 要注意座標換算。
- present 原語 **0x11eb0**(`rows, dstStride, src/dst, srcStride …`,逐列 `0x373c4` memcpy):BG→work、work→VGA 都走它。

---

## 7. 函式 / 位址速查

| 位址 | 角色 |
|---|---|
| 0x28784 | 單圖演出(1 unit) |
| **0x28a6c** | **雙 runtime record renderer**（正式戰鬥 caller 才是攻守演出） |
| 0x29164 | **figure 全身圖 + 台座(TAI.DAT)淡入**(非狀態欄;舊標錯,見 §4.0) |
| **0x2a289** | **狀態欄座標器**:byte[+6] → off 0xc080=(0,154)/0x5ab=(171,4);`call 0x18c6d` |
| **0x18c6d** | **狀態欄(血條框)繪製器**:框 sprite + HP/MP 條 + 名 + 數值(§4.2) |
| 0x4e8af / 0x4e8e1 | RLE 逐列 blit 原語(正向 / 水平鏡像);面板底圖 + glyph 都走它 |
| 0x18795 | 血條長度算 + 畫(`len=cur*101/max+1` → 0x17d6f) |
| 0x17d6f / 0x1685c / 0x4e9bb | 逐欄畫條(查 [0x53a81] cell → 逐列 rep movsb);空槽 cell 0x1d |
| 0x15f84 / 0x4ea2a | FDTXT 排版 / FDOTHER#4 16×16 1bpp glyph blit |
| 0x16559 | 從目前 DATO `[0x53a85]` 取 mouth frame重貼 portrait；不是一般 glyph renderer |
| 0x187d6 / 0x1875d / 0x377d9 | 數值繪製(itoa → 6px digit cell @ [0x53a81];0x1875d 滿血換色) |
| 0x2935b | 單幀 figure/台座 貼圖 wrapper(解 frame header dx/dy → 0x4e63d;dst/stride/transp 由 caller 傳穿) |
| 0x2939d | figure 動畫 renderer（幀迴圈 + 百分比進度；`[0x540ff]==0` 的正式戰鬥 branch 才會消費 `0x29f72`） |
| **0x29f72** | **戰鬥結果 resolver**：derived AP/DP/HIT/EV、HP、item record、terrain table、RNG → hit/crit/damage flags 與 presentation value；非 lunge coordinate helper |
| 0x29c90 | 合成路徑 A(byte[unit+6]≠0):BG (0,50) + figure 走 frame (dx,dy) + slide-in 方向 A |
| 0x29ded | 合成路徑 B(byte[unit+6]==0):BG (0,50) + **figure 固定錨 (164,157)**(0x29ea2)+ slide-in 方向 B |
| 0x114e4 / 0x1b821 | figure placement / derived-field interaction待重判；不得再命名 `+0x48..+0x4e` 為螢幕投影 |
| 0x10fe9 / 0x1142a / 0x250b1 | unit 格座標寫入 / 布陣 / 演出後復位(+0x42→+0x40) |
| 0x4e63d | blit 原語(原生尺寸 RLE,dst+Y*stride+X) |
| 0x11eb0 | 矩形 present(逐列 memcpy,work↔VGA) |
| 0x11d40 | VGA DAC 色盤寫（figure 淡入／fade，ports `0x3c8/0x3c9`；命中條件式脈衝見 `0x2939d`）；HP／MP 條不走此路徑 |
| 0x111ba | 資源解碼器:`(descriptor, prevSlot, index)` → 解 entry[index],釋放 prevSlot,回新 buffer |
| 0x22d1b | BG.DAT 載入(前置) |
| 0x4e893 | 動畫進度來源(被 idiv 100) |
| 0x37795 | VGA DAC 埠寫入原語 |

### 關鍵 descriptor / 變數
| 符號 | 意義 |
|---|---|
| 0x52381 | BG 多層 descriptor(0x111ba 用) |
| 0x52388 | FIGANI 動畫 descriptor(index = 組×3+k) |
| 0x52393 | **"TAI.DAT" 字串**(台座容器,53× ~155×42 菱形 dais sprite;0x29164 經 0x2935b 貼於 figure 腳下) |
| [0x53a81] | **FDOTHER.DAT resource #5** 的 LMI1 UI directory；boot `0x25c97` 明確呼叫 `0x111ba(...,5)`，不是待確認來源 |
| [0x53a7d] / [0x53a85] | `[0x53a7d]` boot 載入 FDTXT.DAT #0；`[0x53a85]` 是會被 caller 重載的工作指標（例如 `0x17eef` 依 unit `+7` 載 DATO portrait），不可跨 scene 固定命名成單一容器 |
| 0x52363 | 章節→演出參數表 `[4,9,14,18,14,0,…]`(`[0x53c03]` 索引) |
| 0x5255f / 0x52577 | idle / fallback 動畫描述子(各 6 dword) |
| [0x53a45] | 單位陣列基底(每單位 80 byte) |
| [0x540ff] | renderer 分支輸入（強推論；正式戰鬥 phase 解讀不得外推至終局尾段） |
| [0x54117] / [0x5411b] | 攻方 / 守方 FIGANI 動畫描述子 buffer |
| [0x54107]/[0x54103]/[0x5410b]/[0x5410f]/[0x54113] | BG 各層 buffer |
| [0x53c03] | 當前章節 |
| [0x53c4b] | 守方 unit idx(0x1561f 傳入) |
| unit[+7] | battle FIGANI resource selector（`×3`）；非 map FDICON `unit+2` selector |
| unit[+0x40]/[+0x42] (word) | **當前 HP / 最大 HP**(0x18c6d 血條 `cur×101/max` + 第一性原理釘死;spawn 0x10fe9 設 cur=max)。✅ 已推翻舊「戰場格 X」誤標 |
| unit[+0x44]/[+0x46] (word) | **當前 MP / 最大 MP** |
| unit[+0x48]/[+0x4a] (word) | **derived AP / DP**（class synthesis `0x1b750` 與 command 17/18 的 15% modifier writers）；非螢幕座標 |
| unit[+0x44]/[+0x46] (word) | **當前 MP / 最大 MP**(同上;舊「戰場格 Y」標法推翻) |
| unit[+8] | 角色名 / 職業 index(0x18d73 `byte[+8]+1` 查名字表 [0x53a7d]) |
| unit[+0x21] | 狀態欄顯示的等級 / 數值(0x18d06 餵 0x187d6,mode2 上限 99) |
| unit[+0x48]/[+0x4a]/[+0x4c]/[+0x4e] (word) | derived AP / DP / HIT / EV；先前 screen bounding-box 斷言已撤回 |
| unit[+6] | renderer 合成路徑 selector（0x29c90 vs 0x29ded）及左右 buffer 交換（0x28e05）；正式戰鬥 caller 可作攻守側別，終局 record 0/1 不可套用此名稱 |
| [0x5018d] | 1.15 double 常數；與 transient modifier / renderer 的精確關係待重判 |
| 0x51a12 / 0x51a2a | map HUD `0x1acf3` 的地形 AP / DP 百分比表，索引為 FDSHAP control byte+1；已驗證 0→(+5,0)、1/5→(0,0)、2/3→(-5,+10)、4→(-5,-5) |
| [0x53ec8] | `0x29f72`相關combat-result／presentation state；下游語意尚未閉合，不是已證實的縮放或figure X座標 |

---

## 8. 六項成果摘要 + 待確認

1. **入口 + 呼叫鏈** ✅:單圖 0x28784(caller 0x15195)、雙 record renderer 0x28a6c（`0x1561f` 傳正式戰鬥攻方 `ebx`、目標 `[0x53c4b]`；另有 0x18fc6/0x2c2a6/0x35435）。`[0x540ff]` 是 renderer 分支輸入，不能以單一 phase 名稱涵蓋所有 caller。
2. **figure 座標 / 翻轉 / 縮放** partial:blit 0x4e63d 原生尺寸 `dst+Y*stride+X`,**全程無縮放運算**;每幀 displacement=descriptor header `u16 X/Y`，固定 `(164,157)` 是某些 figure/台座 caller 的 anchor；`word[unit+0x40]`=當前 HP（非座標）。`+0x48/+0x4a` 是 derived AP/DP，`0x29f72` 是 combat-result resolver。**待確認**:byte[unit+6] 攻守配對、土台 entry、所有 caller 的 schedule。
3. **BG 繪製與TAI台座是兩條素材路徑**：BG.DAT多層走
   `[0x54107…54113]`並以`0x4e63d(X=0,Y=50,寬320)`合成；腳下大台座由
   `0x29164`另載TAI.DAT sprite，不是BG層或程式純色。TAI entry、raw selector
   與跨角色對齊仍待逐caller驗證。
4. **狀態欄(血條框)** ✅(本輪嚴格 RE 重做,§4):真函式 = **0x18c6d**(座標器 0x2a289,byte[+6]→ 我方(0,154)/敵方(171,4))。**0x29164 不是狀態欄,是 figure + 台座(TAI.DAT)淡入**(舊標錯已改)。三元素釘死:**① 框/深藍底/立體 bevel = 素材 sprite**(0x4e8af blit [0x53a81]+0x5e);**② HP/MP 條 = 程式畫**(0x18795 算 `len=cur*101/max+1` → 0x17d6f 逐欄 blit [0x53a81] 漸層欄 cell,空槽 0x1d;HP=unit+0x40/+0x42、MP=+0x44/+0x46);**③ 名 = `0x15f84→0x4ea2a` 以 `[0x53a75]` FDOTHER#4 font畫 16×16 glyph**、**數值 = 6px digit cell**([0x53a81],0x187d6)。`[0x53a81]` loader 已由 boot `0x25c97` 定案為 FDOTHER #5；`[0x53a85]` 是 DATO mouth-frame工作指標，不再誤稱字模。
5. **正式戰鬥動畫控制** ✅：`[0x540ff]` 的 0／非零 branch、0x2939d 幀迴圈與 `idiv 100` 百分比進度已固定；`[0x540ff]==0` 的正式戰鬥 branch 才消費 `0x29f72`。幀 (dx,dy) 的 swing 斬擊弧、**命中色盤脈衝的條件式 VGA DAC 0x3c8/0x3c9 寫入**（原始 frame `+4`、傷害步進與 `0x29f72` 欄位共同控制，證據見 §8）、HP 條非色盤及 idle fallback 0x5255f/0x52577 都不外推到終局尾段。**待確認**：原始輸出欄位的完整 producer/consumer、各 caller 的 schedule 與一般玩家路徑幀數。
6. **座標系** ✅:320×200、VGA 0xa0000、**work stride 640 但只 present 左半 320**(雙寬 off-screen 預備區,用途待確認)。

## 2026-08-10：命中色盤脈衝與戰場畫面修正（E0／E1）

以固定雜湊 `FD2.EXE` 重新執行合法 IDA Pro 9.4 與 Docker Capstone 5.0.3 後，
`0x2939d` 的實際條件已固定：frame record `+4 == 1`、傷害步進完成，並且
`0x29f72` 的兩個原始輸出欄位各自非零時，才會寫入兩段 VGA DAC 序列。第一段
是索引 0 的 `(1,0x20,0)`、等待 20 毫秒後清零；第二段是 `(0x3f,0x3f,0x3f)`、
等待 20 毫秒、清零再等待 40 毫秒。每個 FIGANI 子幀另以 `0x17aa9(1)` 等待，
不能化約為固定 `impactS+8` 的閃爍時間。逐位址證據見
[`fd2_battle_impact_pulse_ida.txt`](../data/ida/fd2_battle_impact_pulse_ida.txt)。

重製端目前沒有可追溯的原始輸出欄位，因此已移除原先無條件繪製的 RGBA 全畫面
紅罩；目前只保留原版 impact 參考圖支持的守方紅剪影，攻方維持原本 FIGANI
幀；真正 DAC 脈衝先保持失敗即關閉。這直接修正 GitHub 戰鬥演出圖中「整個
背景與狀態欄一起泛紅、攻方也被染紅」的可見偏差，但尚未閉合原版輸出欄位
轉接器、完整幀序列或一般玩家 E2。
修正後的代表性畫面保存為
[`battle-impact-no-global-tint.png`](../figures/battle-impact-no-global-tint.png)，
其擷取條件與雜湊見 [`battle-impact-no-global-tint.json`](../data/ui-traces/battle-impact-no-global-tint.json)。

### 8.1 2026-08-10：命中狀態欄邊界與正規化對照

重新對照 `orig_05_attack_03_impact.png` 與目前 `FD2_SHOT_SERIES` 的同一命中幀後，
確認原版在命中呈現開始時立即顯示扣血後的 HP；重製端原先以 8 個 tick 緩降，會
出現原版沒有的中間數值。現改為只在已觀測的 impact 邊界切換 `defHP0→defHP1`，
不推導未知的傷害寫入者或 DAC 時序。原版擷取的守方剪影主色為 RGB `(190,0,0)`，
重製端 E1 剪影近似也改用此色值；這不是原版色盤脈衝的轉接器。

新增正規化對照圖
[`battle-impact-compare-20260810.png`](../figures/battle-impact-compare-20260810.png)：
原版 DOSBox 640×480 以垂直 2.4 倍取樣正規化為 320×200，重製端 640×400 以 2 倍
最近鄰縮小，右欄是逐 RGB 差異遮罩。此固定命中 fixture 尚有 3933 個差異像素，
因此只關閉 HP 中間值與剪影色值兩項可見差異，未提升為完整戰鬥介面或一般玩家 E2。

同輪修正守方 FIGANI 待機排程：重製端原先以固定 `(prog/6)` 循環選幀，現改依
各守方資源的 descriptor `+6` 與 `FD2_BATTLE_FPT` 以純排程橋選幀；攻方與守方
任一組缺少完整延遲表時，`newAtkAnim` 失敗即關閉，不回退到猜測的呼吸週期。這
只修正資源消費管線，尚未證實命中分支的 raw presentation 欄位或一般玩家時序。
