# 35 — 全螢幕戰鬥演出繪圖機制(FIGANI 攻守動畫)

> 反組譯《炎龍騎士團2》FD2.EXE(DOS4GW LE,obj1 linear=file offset)的**全螢幕戰鬥演出**:
> 攻擊發生時切到的那張大圖畫面 —— 守方 / 攻方全身 FIGANI、戰場背景、狀態欄、斬擊與閃紅。
> 所有結論附反組譯位址佐證;runtime 才決定的值標「待確認」。
> 相關:doc 06(FIGANI 格式)· doc 31(FDICON 地圖小人,另一套)· doc 13(戰鬥選單)· doc 25(戰鬥事件)。

## 0. 兩個演出函式(別搞混)

| 函式 | 入口 | 參數 | 用途 |
|---|---|---|---|
| 單圖演出 | **0x28784** | 1 個 unit index | 顯示**單一**單位全身圖(施法/單體演出) |
| 攻守演出 | **0x28a6c** | **2 個** unit index(攻方、守方) | 攻擊的**對打**全螢幕演出(本篇重點) |

兩者 prologue 都是 Watcom 風格 `push <frameSize>; call 0x36cd7`(stack-probe):
- 0x28784:`push 0x54; call 0x36cd7`(0x28789),func 範圍 0x28784–0x28a6b `ret`。
- 0x28a6c:`push 0x64; call 0x36cd7`(0x28a71),func 範圍 0x28a6c–0x29116 `ret`。

> ⚠ 既有錨點把 0x287b5(`movzx esi,[ebx+7]`)當成「攻守演出」入口,實際它在**單圖** 0x28784 內;
> 真正的攻守雙圖演出是 **0x28a6c**(0x28ad6 `movzx eax,[ebx+7]` 攻方組、0x28ade `movzx eax,[esi+7]` 守方組)。

### 呼叫鏈(誰觸發演出)

`calls 0x28a6c`(相對 call 來源):

| caller linear | 傳參 | 意義 |
|---|---|---|
| **0x1561f** | `push [0x53c4b]; push ebx; call` → `0x28a6c(ebx, [0x53c4b])` | 攻擊執行流:**arg0=ebx=攻方 idx、arg1=[0x53c4b]=守方 idx** |
| 0x18fc6 | — | 另一觸發點(待確認) |
| 0x2c2aa | `mov [0x540ff],ecx; push 1; push 0; call` → `0x28a6c(0, 1)` | 先設演出 phase 旗標再呼叫 |
| 0x35435 | — | 另一觸發點(待確認) |

`calls 0x28784` → 唯一 caller 0x15195(單圖路徑,與 0x1561f 同一攻擊執行區 0x15xxx,符合「攻擊執行 0x15xxx」推測)。

**[0x540ff] = 演出 phase / 重繪旗標**(`refs 540ff` 找到 writer):函式外 0x25ac0、0x25b6f 設定;函式內 0x28ae8/0x28c15/0x28cdb/…/0x28ef1(設 1)讀寫。
語意:`0` = 第一次進場(載入資源 + 全畫面合成),非 0 = 後續增量幀(只重畫變動層)。動畫靠**重複呼叫 0x28a6c 推進**,phase 旗標決定這一幀做哪些事。

---

## 1. FIGANI 載入(組 × 3)+ buffer

- 攻方組 = `unit_attacker[+7]`(0x28ad6 → `[esp+0x10]`);守方組 = `unit_defender[+7]`(0x28ade → `[esp+0xc]`)。
  與 doc 31/06 一致:`unit[+7]` = FDICON/FIGANI 組號。
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

- **螢幕錨點常數 (X=0xa4=164, Y=0x9d=157)**:出現在 0x28f55(`0x4e63d(src, 0xa4, 0x9d, edi, 0x280, -1)`)、
  0x29164 狀態欄(0x291b0/0x29268/0x29295)。這是演出區/單位圖的**固定錨點**(160×100 半屏中央偏下)。
- **figure 的螢幕貼圖錨點是固定常數 (X=0xa4=164, Y=0x9d=157)**,不是 `word[unit+0x40]`:
  - ⚠ 修正既有斷言:`0x29582 push [esp+0x50]` 因前面已 `push -1; push 0x280`(esp 下移 8),實際讀的是 frame local `+0x48`(=**dst work buffer**),**不是** `word[unit+0x40]`。`word[unit+0x40]` 在 `0x294ad` 讀進 `[esp+0x50]` 後是**餵給 `0x29f72` 的算式**,不是直接當 blit X。
  - 最終 figure blit 的螢幕座標寫死在合成處:`0x28f67`(主合成)、`0x29ded:0x29ea2`(==0 路徑):
    `0x4e63d(figureSrc, 0xa4, 0x9d, dst, 0x140, -1)` → **(164,157) 是 figure 的螢幕左上錨**(全 figure 共用此常數)。
- **`word[unit+0x40]` = 該單位「當前戰場格 X」(動畫中會變)**,不是直接螢幕 X:
  - 配對欄位:**`+0x40`=current X、`+0x42`=home X、`+0x44`=current Y、`+0x46`=home Y**(布陣時 current=home,一起寫)。
  - **寫入點**:布陣 `0x10FE9`(`+0x40=+0x42=valX`、`+0x44=+0x46=valY`)、`0x1142A`(同款,另一陣營 / 召喚);
    戰鬥演出中 `0x2975A` 把 `0x29f72` 內插出的新 current X 寫回 `word[unit+0x40]`(lunge 前衝);
    演出結束 `0x250B1` 迴圈 `+0x42→+0x40`、`+0x46→+0x44` 把全單位 current 復位回 home。
  - **螢幕投影座標另存 `+0x48/+0x4a/+0x4c/+0x4e`**(unit icon 的螢幕 bounding box):由 `0x114E4` 累加 sprite 各幀錨點
    (`0x4e56c` 取 descriptor → `word[+1/+3/+5/+7]` 累加)算出;另有 `0x1B821` 變體對 `+0x4a/+0x4e` 乘 **`[0x5018d]=1.15`**
    浮點縮放(條件 `byte[unit+0x23]!=0`)。**remake 位置偏高的疑點之一即這顆 1.15 與錨點 157 沒對齊**。
- **`0x29f72(攻方idx, 守方idx, &out)` = lunge / 接近內插器**(figure 最終位移來源):
  - `ebp=&unit[arg0]`、`edi=&unit[arg1]`(各 `idx*80`,base `[0x53a45]`);讀雙方 `+0x40/+0x42`(格 X)與 `+0x48/+0x4a/+0x4c/+0x4e`(螢幕投影)。
  - 用**動畫進度** `0x4e893` → `idiv 100`(百分比)把位移內插;再套**方向 / pose 微調表**
    `[dir*4 + 0x51a12]`(X,值 `[5,0,-5,-5,-5,…]` %)、`[dir*4 + 0x51a2a]`(Y,值 `[0,0,10,10,-5,…]` %)。
  - 輸出 struct(`esi[0]/4/8/0x10`=各種旗標、`esi[0x14]`=內插位移量)+ 寫全域 `[0x53ec8]`(=守方+0x21 × spriteW ÷ 攻方+0x21,縮放後的 X 用量)。
  - → **figure 前衝幅度 = 雙方格距 × 動畫% × 方向微調**,疊在固定錨點 (164,157) 上。
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
  - [0x54107](0x28cd4,index = 章節表算出的變動值)、[0x54103](0x28daa, idx 0)、[0x5410b](0x28dc4, idx 0)、[0x5410f](0x28dde, idx 1)、[0x54113](0x28df8, idx 2)。
  - → BG.DAT 至少 **3–5 個 entry**(idx 0/1/2 + 章節索引層),遠景 / 近景 / 土台分層。
- **BG blit 座標**:`0x4e63d(BGsrc, X=0, Y=0x32=50, dst, stride, -1)`:
  - 0x28d42(`[0x54107]`)、0x28e27(`[0x54103]`),stride 0x140=320;slide 合成 `0x29c90`/`0x29ded` 再把 `[0x5410b/0f/13]`(idx 0/1/2)循環貼於 (0,50)。
  - → **背景與各層都貼在 X=0, Y=50,寬 320**(整屏寬;上方 50px 與下方留給狀態 UI）。

### 3.3 攻方腳下「圓圈 / 土台」= FIGANI sprite 自帶的 dither 陰影,**不是 BG 層、不是程式畫純色**(視覺確認)

- **靜態前提**:整條戰鬥合成沒有任何 fillrect / 畫圓 / 畫線原語——畫面只走 `0x4e63d`(sprite RLE blit)、
  `0x11eb0`(逐列 memcpy present)、`0x11d40`(色盤寫)。→ 圓圈必是素材,非程式生成。
- **視覺確認(放大 orig_05 攻方腳下 + 比對 FIGANI_012)**:那圈不是綠色實心草地圈,是**半透明網點(dither)橢圓陰影**,
  且它**燒在 FIGANI 攻方圖本身**——`decode_figani` 解 FIGANI_012 出來腳下就帶這圈 dither 陰影。
  - 機制 = FIGANI 幀像素 codec 的 **`01xxxxxx` dither/陰影模式**(見 doc06:「讀 1 像素,輸出 `[透明,值]×count` 隔位寫」),
    解碼後就是「一半透明一半灰」的網點橢圓,正是地面投影陰影。攻 / 守 figure 各自的 FIGANI 都自帶。
- **修正先前推測**:本節舊版(agent 靜態推論)說圓圈是「BG.DAT 某前景層 `[0x5410b/0f/13]`」——**經視覺對照推翻**。
  BG.DAT 各 entry dump 出來是 320×100 的天空 / 遠山 / 森林草地 / 樹幹近景(`BG_004`=map0 戰場背景),**沒有獨立土台圈層**;
  圓圈在攻方腳下 y≈160–175,已超出 BG 貼圖範圍(y50–150),不可能是 BG 層。它就是 FIGANI 自帶 dither 陰影。
- **remake 對映**:**不要 drawCircle / drawEllipse 自畫**;正確解碼 FIGANI(保留 dither 透明),畫 figure 時陰影自動帶出。

### 3.2 戰場 → BG 參數對應

- **章節參數表 0x52363**:`[0x53c03]`(= 章節 byte,既有錨點)當 index 取 `byte[chapter + 0x52363]`:
  0x28b61 `mov edx,[0x53c03]; movzx edx, byte[edx+0x2363]` → `[esp+0x18]`。
  表值 = `[4, 9, 14, 18, 14, 0, 2, 4, 6, 8, 10, …]`(0x52363 起,前 6 個是章節用)。
- 此值參與 figure / BG 選擇分支(0x28b89 與龍騎兵特例聯動),**確切語意(選哪張 BG / 色盤)待確認**;
  但「**戰場→演出參數由章節 [0x53c03] 索引 0x52363 表**」這條對應已確認。
- 既有錨點 BG_004 森林 320×100:與本處「BG 寬 320、貼在 y=50」尺寸吻合。地形→BG index 的細部對應待續(追 0x22d1b 的 index 參數來源)。

---

## 4. 狀態欄繪製(0x29164)

`0x29164(...)` 為單一單位畫**全身圖 + 狀態條**(攻守各呼叫一次,0x28e76 / 0x288f3):

- prologue `push 0x2c; call 0x36cd7`;`esi = [0x53a45]+idx*80`(0x2917c-0x29184,**單位 80=0x50 byte stride**,`idx*5*16`),確認既有錨點。
- 0x2918a `movzx ebx, byte[unit + 6]` → 分支(+6 旗標,同 §2 守方判定):
- **全身圖**:0x4e63d 於 **(X=0xa4=164, Y=0x9d=157)** 貼圖(0x291b0 / 0x29268 / 0x29295)。
- **8 次迴圈**(`esi=8 → 0`,0x29197 / 0x2920d):每圈
  - `imul ebp, esi, 0xa`(×10):層 / 行間距;
  - **HP/狀態條 = 色盤寫入** `0x11d40(0, 0xff, esi*6)`(0x291f9 `imul ebx,esi,6`):**0x11d40 直接寫 VGA DAC 埠**(見 §5)。
  - 每圈 `0x11eb0` 把 work buffer 段 present 到 VGA 0xa0000(0x291ec)。
- **數字 / 名字 / LV**:經 0x2935b(0x291cf / 0x29247)以幀方式貼(name/LV/HP/MP 為預渲染圖塊)。
- 座標體系:work stride 0x280=640、螢幕寬 0x140=320、高 0xc8=200、VGA 0xa0000(全部出現,§6)。

> 攻方欄 / 守方欄的左右分屬:0x29164 被呼叫兩次,各傳不同 unit idx + 不同 buffer base;**各欄絕對 X 偏移待確認**
> (主錨點 164,157 是共用 figure 位;狀態條相對位由迴圈 `esi*10` / `esi*6` 決定)。顏色來源 = 色盤 index(0x11d40 寫 0x3c9)。

---

## 5. 動畫階段機制(windup → swing → impact → standoff)

- **驅動**:phase 旗標 **[0x540ff]** + **重複呼叫 0x28a6c**(每幀一次);非「一次畫完整段」。
  進場 phase=0(載資源 + 全合成,0x28e3e 區算 [0x54117]/[0x5411b]),之後 0x28ef1 設 [0x540ff]=1 走增量。
- **進度百分比**:figure renderer **0x2939d** 內 0x2946a `call 0x4e893` → 0x2947b `idiv 100`(`mov ebx,0x64; idiv`),
  取餘數判斷階段(`cmp edx,3` < 3 時 `[esp+0x4c]=2`,多畫一層)→ **動畫進度以 0–99% 表示,百分比決定當前幀 / 疊層**。
- **幀迴圈**:0x2939d 以 `byte[ebp]`=幀數迴圈(0x29409-0x29424),每幀先 0x29f72(算 lunge 位移 + 階段旗標)再 0x2935b 貼;
  幀的 (dx,dy) 內嵌(§2.2)→ **swing 斬擊弧 = 逐幀位移 + 換幀**。
- **idle / fallback 描述子**:0x2939d 進場 `rep movsd` 從 **0x5255f**(6 dword)與 **0x52577**(6 dword)複製預設描述子到區域 frame
  (0x293cf / 0x293df)→ 沒有真實動畫時的**待機姿態 fallback**。
- **閃紅 / 抽 HP(impact)= 色盤操作,不是重畫像素**:
  - **0x11d40** 是 VGA DAC 寫入迴圈:`push 0x3c8 / push 0x3c9; call 0x37795`(0x11d5c / 0x11d73)→
    out 到埠 **0x3c8(palette index)/0x3c9(palette data)**。0x37795 = DAC 埠寫入原語。
  - 同手法在 0x28784 / 0x286dd 的 fade-in 迴圈(用 0x53a65 色表插值)。
  - → **守方受擊閃紅 = 改色盤**(把該圖用色暫時拉紅再復原);HP 條變化亦走 0x11d40 色盤,效能極低成本。
    精確的「閃紅幀數 / 色值序列」待確認(需追 0x37795 的色值來源表)。
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
| **0x28a6c** | **攻守演出主函式(2 unit)** |
| 0x29164 | 單位全身圖 + 狀態條繪製 |
| 0x2935b | 單幀 figure 貼圖 wrapper(解 frame header dx/dy → 0x4e63d;dst/stride/transp 由 caller 傳穿) |
| 0x2939d | figure 動畫 renderer(幀迴圈 + 百分比進度;0x29536 依 byte[unit+6] 分兩合成路徑) |
| **0x29f72** | **lunge / 接近內插器**(讀雙方 +0x40/+0x48… + 動畫% + 方向微調表 → 內插位移;寫回 unit+0x40 @0x2975a、[0x53ec8]) |
| 0x29c90 | 合成路徑 A(byte[unit+6]≠0):BG (0,50) + figure 走 frame (dx,dy) + slide-in 方向 A |
| 0x29ded | 合成路徑 B(byte[unit+6]==0):BG (0,50) + **figure 固定錨 (164,157)**(0x29ea2)+ slide-in 方向 B |
| 0x114e4 / 0x1b821 | 單位螢幕投影:算 +0x48/+0x4a/+0x4c/+0x4e(後者對 +0x4a/+0x4e 乘 1.15) |
| 0x10fe9 / 0x1142a / 0x250b1 | unit 格座標寫入 / 布陣 / 演出後復位(+0x42→+0x40) |
| 0x4e63d | blit 原語(原生尺寸 RLE,dst+Y*stride+X) |
| 0x11eb0 | 矩形 present(逐列 memcpy,work↔VGA) |
| 0x11d40 | VGA DAC 色盤寫(閃紅 / HP 條 / fade,埠 0x3c8/0x3c9) |
| 0x111ba | 資源解碼器:`(descriptor, prevSlot, index)` → 解 entry[index],釋放 prevSlot,回新 buffer |
| 0x22d1b | BG.DAT 載入(前置) |
| 0x4e893 | 動畫進度來源(被 idiv 100) |
| 0x37795 | VGA DAC 埠寫入原語 |

### 關鍵 descriptor / 變數
| 符號 | 意義 |
|---|---|
| 0x52381 | BG 多層 descriptor(0x111ba 用) |
| 0x52388 | FIGANI 動畫 descriptor(index = 組×3+k) |
| 0x52393 | 演出輔助 descriptor(index 含章節值;語意待確認) |
| 0x52363 | 章節→演出參數表 `[4,9,14,18,14,0,…]`(`[0x53c03]` 索引) |
| 0x5255f / 0x52577 | idle / fallback 動畫描述子(各 6 dword) |
| [0x53a45] | 單位陣列基底(每單位 80 byte) |
| [0x540ff] | 演出 phase / 重繪旗標 |
| [0x54117] / [0x5411b] | 攻方 / 守方 FIGANI 動畫描述子 buffer |
| [0x54107]/[0x54103]/[0x5410b]/[0x5410f]/[0x54113] | BG 各層 buffer |
| [0x53c03] | 當前章節 |
| [0x53c4b] | 守方 unit idx(0x1561f 傳入) |
| unit[+7] | FIGANI/FDICON 組號 |
| unit[+0x40]/[+0x42] (word) | 戰場格 X:current / home(布陣 0x10fe9、復位 0x250b1) |
| unit[+0x44]/[+0x46] (word) | 戰場格 Y:current / home |
| unit[+0x48]/[+0x4a]/[+0x4c]/[+0x4e] (word) | 螢幕投影 bounding box(0x114e4 累加;0x1b821 ×1.15) |
| unit[+6] | 攻 / 守旗標:選合成路徑(0x29c90 vs 0x29ded)+ 左右 buffer 交換(0x28e05) |
| [0x5018d] | 投影縮放常數 = **1.15**(double) |
| 0x51a12 / 0x51a2a | 方向 / pose 微調表(X:[5,0,-5,-5,-5,…]%、Y:[0,0,10,10,-5,…]%) |
| [0x53ec8] | 0x29f72 輸出:縮放後 figure X 用量(被攻擊執行區 0x15/0x18/0x19xxx 廣泛讀取) |

---

## 8. 六項成果摘要 + 待確認

1. **入口 + 呼叫鏈** ✅:單圖 0x28784(caller 0x15195)、攻守 0x28a6c(caller 0x1561f 傳 `攻方ebx, 守方[0x53c4b]`,另 0x18fc6/0x2c2aa/0x35435)。phase = [0x540ff]。
2. **figure 座標 / 翻轉 / 縮放** ✅(重點):blit 0x4e63d 原生尺寸 `dst+Y*stride+X`,**全程無縮放運算**;螢幕錨點 = **固定常數 (164,157)**(非 word[unit+0x40]——舊斷言已修正);`word[unit+0x40]`=戰場格 current X,經 **0x29f72** 用動畫%(0x4e893/idiv 100)+ 方向微調表(0x51a12/0x51a2a)+ 投影縮放 1.15([0x5018d])內插出 lunge 位移;**無 runtime 水平翻轉**(blit 家族全前向,朝向燒進素材);`byte[unit+6]` 選合成路徑 0x29c90(≠0)/ 0x29ded(==0,用 (164,157))→ 即攻 / 守腳底 Y(175/150)不同之源。**待確認**:byte[unit+6] 攻守配對、土台 entry。
3. **BG 繪製 + 腳下圓圈** ✅:BG.DAT(0x52381 即 `"BG.DAT"` 字串)多層 → [0x54107…54113],全部 `0x4e63d(X=0,Y=50,寬320)`;戰場→章節 [0x53c03] 索引 0x52363。**腳下圓圈 / 土台 = BG 素材層(sprite blit),非程式畫純色**(戰鬥區無 rect/circle 原語,只有 0x4e63d/0x11eb0/0x11d40)。**待確認**:哪個 BG.DAT entry 是土台(需 dump 視覺對照)。
4. **狀態欄 0x29164** ✅:全身圖 (164,157)、8 次迴圈、HP/狀態條走色盤 0x11d40、present 0x11eb0。**待確認**:攻欄/守欄絕對 X、名字/數字相對排版細節。
5. **動畫階段** ✅:[0x540ff] phase + 重複呼叫驅動;0x2939d 幀迴圈 + `idiv 100` 百分比進度;幀 (dx,dy) = swing 斬擊弧;**閃紅/HP 抽乾 = VGA DAC 色盤 0x3c8/0x3c9(0x11d40)**;idle fallback 0x5255f/0x52577。**待確認**:閃紅色值序列、各階段確切幀數。
6. **座標系** ✅:320×200、VGA 0xa0000、**work stride 640 但只 present 左半 320**(雙寬 off-screen 預備區,用途待確認)。
